// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

package metabase

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"storj.io/storj/shared/dbutil/spannerutil"
	"storj.io/storj/shared/tagsql"
)

// BucketTally contains information about aggregate data stored in a bucket.
type BucketTally struct {
	BucketLocation

	ObjectCount        int64
	PendingObjectCount int64

	TotalSegments int64
	TotalBytes    int64

	MetadataSize int64
}

// CollectBucketTallies contains arguments necessary for looping through objects in metabase.
type CollectBucketTallies struct {
	From               BucketLocation
	To                 BucketLocation
	AsOfSystemTime     time.Time
	AsOfSystemInterval time.Duration
	Now                time.Time

	UsePartitionQuery bool
}

// Verify verifies CollectBucketTallies request fields.
func (opts *CollectBucketTallies) Verify() error {
	if opts.To.ProjectID.Less(opts.From.ProjectID) {
		return ErrInvalidRequest.New("project ID To is before project ID From")
	}
	if opts.To.ProjectID == opts.From.ProjectID && opts.To.BucketName < opts.From.BucketName {
		return ErrInvalidRequest.New("bucket name To is before bucket name From")
	}
	return nil
}

// CollectBucketTallies collect limited bucket tallies from given bucket locations.
func (db *DB) CollectBucketTallies(ctx context.Context, opts CollectBucketTallies) (result []BucketTally, err error) {
	defer mon.Task()(&ctx)(&err)

	if err := opts.Verify(); err != nil {
		return []BucketTally{}, err
	}

	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	for _, adapter := range db.adapters {
		adapterResult, err := adapter.CollectBucketTallies(ctx, opts)
		if err != nil {
			return nil, err
		}
		result = append(result, adapterResult...)
	}

	// only a merge sort should be strictly required here, but this is much easier to implement for now
	slices.SortFunc(result, func(a, b BucketTally) int {
		return a.BucketLocation.Compare(b.BucketLocation)
	})

	return result, nil
}

// CollectBucketTallies collect limited bucket tallies from given bucket locations.
func (p *PostgresAdapter) CollectBucketTallies(ctx context.Context, opts CollectBucketTallies) (result []BucketTally, err error) {
	err = withRows(p.db.QueryContext(ctx, `
			SELECT
				project_id, bucket_name,
				SUM(total_encrypted_size), SUM(segment_count),
				COALESCE(SUM(length(encrypted_metadata)),0)+COALESCE(SUM(length(encrypted_etag)), 0),
				count(*), count(*) FILTER (WHERE status = `+statusPending+`)
			FROM objects
			`+LimitedAsOfSystemTime(p.impl, time.Now(), opts.AsOfSystemTime, opts.AsOfSystemInterval)+`
			WHERE (project_id, bucket_name) BETWEEN ($1, $2) AND ($3, $4) AND
			(expires_at IS NULL OR expires_at > $5)
			GROUP BY (project_id, bucket_name)
			ORDER BY (project_id, bucket_name) ASC
		`, opts.From.ProjectID, opts.From.BucketName, opts.To.ProjectID, opts.To.BucketName, opts.Now))(func(rows tagsql.Rows) error {
		for rows.Next() {
			var bucketTally BucketTally

			if err = rows.Scan(
				&bucketTally.ProjectID, &bucketTally.BucketName,
				&bucketTally.TotalBytes, &bucketTally.TotalSegments,
				&bucketTally.MetadataSize, &bucketTally.ObjectCount,
				&bucketTally.PendingObjectCount,
			); err != nil {
				return Error.New("unable to query bucket tally: %w", err)
			}

			result = append(result, bucketTally)
		}

		return nil
	})
	if err != nil {
		return []BucketTally{}, err
	}

	return result, nil
}

// CollectBucketTallies collect limited bucket tallies from given bucket locations.
func (s *SpannerAdapter) CollectBucketTallies(ctx context.Context, opts CollectBucketTallies) (result []BucketTally, err error) {
	defer mon.Task()(&ctx)(&err)

	if opts.UsePartitionQuery {
		return s.collectBucketTalliesWithPartitionedQuery(ctx, opts)
	}

	fromTuple, err := spannerutil.TupleGreaterThanSQL([]string{"project_id", "bucket_name"}, []string{"@from_project_id", "@from_bucket_name"}, true)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	toTuple, err := spannerutil.TupleGreaterThanSQL([]string{"@to_project_id", "@to_bucket_name"}, []string{"project_id", "bucket_name"}, true)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	txn := s.client.Single().WithTimestampBound(spannerutil.MaxStalenessFromAOSI(opts.AsOfSystemInterval))
	return spannerutil.CollectRows(txn.QueryWithOptions(ctx, spanner.Statement{
		SQL: `
			SELECT
				project_id, bucket_name,
				SUM(total_encrypted_size), SUM(segment_count),
				COALESCE(SUM(length(encrypted_metadata)),0)+COALESCE(SUM(length(encrypted_etag)), 0),
				count(*) AS total_objects_count, COUNTIF(status = ` + statusPending + `) AS pending_objects_count
			FROM objects
			WHERE ` + fromTuple + `
				AND ` + toTuple + `
				AND (expires_at IS NULL OR expires_at > @when)
			GROUP BY project_id, bucket_name
			ORDER BY project_id ASC, bucket_name ASC
		`,
		Params: map[string]any{
			"from_project_id":  opts.From.ProjectID,
			"from_bucket_name": opts.From.BucketName,
			"to_project_id":    opts.To.ProjectID,
			"to_bucket_name":   opts.To.BucketName,
			"when":             opts.Now,
		},
	}, spanner.QueryOptions{
		Priority: spannerpb.RequestOptions_PRIORITY_LOW,
	}), func(row *spanner.Row, bucketTally *BucketTally) error {
		return row.Columns(
			&bucketTally.ProjectID, &bucketTally.BucketName,
			&bucketTally.TotalBytes, &bucketTally.TotalSegments,
			&bucketTally.MetadataSize, &bucketTally.ObjectCount,
			&bucketTally.PendingObjectCount,
		)
	})
}

func (s *SpannerAdapter) collectBucketTalliesWithPartitionedQuery(ctx context.Context, opts CollectBucketTallies) (result []BucketTally, err error) {
	tb := spanner.StrongRead()
	if !opts.AsOfSystemTime.IsZero() {
		tb = spanner.ReadTimestamp(opts.AsOfSystemTime)
	}
	txn, err := s.client.BatchReadOnlyTransaction(ctx, tb)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	defer txn.Close()

	fromTuple, err := spannerutil.TupleGreaterThanSQL([]string{"project_id", "bucket_name"}, []string{"@from_project_id", "@from_bucket_name"}, true)
	if err != nil {
		return nil, Error.Wrap(err)
	}
	toTuple, err := spannerutil.TupleGreaterThanSQL([]string{"@to_project_id", "@to_bucket_name"}, []string{"project_id", "bucket_name"}, true)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	stmt := spanner.Statement{
		SQL: `
			SELECT
				project_id, bucket_name, total_encrypted_size, segment_count,
				COALESCE(length(encrypted_metadata), 0)+COALESCE(length(encrypted_etag), 0),
				status
			FROM objects
			WHERE ` + fromTuple + `
				AND ` + toTuple + `
				AND (expires_at IS NULL OR expires_at > @when)
		`,
		Params: map[string]any{
			"from_project_id":  opts.From.ProjectID,
			"from_bucket_name": opts.From.BucketName,
			"to_project_id":    opts.To.ProjectID,
			"to_bucket_name":   opts.To.BucketName,
			"when":             opts.Now,
		},
	}

	partitions, err := txn.PartitionQueryWithOptions(ctx, stmt, spanner.PartitionOptions{
		PartitionBytes: 0,
		MaxPartitions:  0,
	}, spanner.QueryOptions{
		Priority: spannerpb.RequestOptions_PRIORITY_LOW,
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}

	bucketTallies := map[BucketLocation]BucketTally{}
	for _, partition := range partitions {
		iter := txn.Execute(ctx, partition)
		err := iter.Do(func(r *spanner.Row) error {
			var bucketLocation BucketLocation
			var totalEncryptedSize int64
			var segmentCount int64
			var encryptedMetadataSize int64
			var status ObjectStatus
			if err := r.Columns(&bucketLocation.ProjectID, &bucketLocation.BucketName, &totalEncryptedSize, &segmentCount, &encryptedMetadataSize, &status); err != nil {
				return Error.Wrap(err)
			}

			bucketTally, ok := bucketTallies[bucketLocation]
			if !ok {
				bucketTally = BucketTally{
					BucketLocation: bucketLocation,
				}
			}
			bucketTally.TotalBytes += totalEncryptedSize
			bucketTally.TotalSegments += segmentCount
			bucketTally.MetadataSize += encryptedMetadataSize
			bucketTally.ObjectCount++
			if status == Pending {
				bucketTally.PendingObjectCount++
			}
			bucketTallies[bucketLocation] = bucketTally
			return nil
		})
		if err != nil {
			return nil, Error.Wrap(err)
		}
	}
	return maps.Values(bucketTallies), nil
}
