// Copyright (C) 2025 Storj Labs, Inc.
// See LICENSE for copying information.

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/zeebo/clingy"
	"golang.org/x/sync/errgroup"

	"storj.io/storj/shared/modular"
	"storj.io/storj/shared/modular/config"
	"storj.io/storj/shared/mud"
)

// MudCommand is a command that initializes and runs modular components.
type MudCommand struct {
	ball        *mud.Ball
	selector    mud.ComponentSelector // selector for components to initialize and run
	runSelector mud.ComponentSelector // optional selector for components to run. Used for config list, where everything is used to initialize, but only the subcommand is executed.
	cfg         *ConfigSupport
}

// Setup implements clingy setup phase.
func (m *MudCommand) Setup(params clingy.Parameters) {
	ctx := context.Background()

	selectorStr := params.Flag("components", "Modular component selection. If empty, all default components will be running", "").(string)

	if m.selector == nil {
		m.selector = modular.CreateSelectorFromString(m.ball, selectorStr)
	} else if selectorStr != "" {
		m.selector = mud.Or(m.selector, modular.CreateSelectorFromString(m.ball, selectorStr))
	}

	// create all the config structs
	err := mud.ForEachDependency(m.ball, m.selector, func(component *mud.Component) error {
		return component.Init(ctx)
	}, mud.Tagged[config.Config]())
	if err != nil {
		panic(err)
	}

	// register config structs as clingy parameters
	err = mud.ForEachDependency(m.ball, m.selector, func(component *mud.Component) error {

		tag, found := mud.GetTagOf[config.Config](component)
		if !found {
			return nil
		}

		bindConfig(params, tag.Prefix, reflect.ValueOf(component.Instance()), m.cfg)
		return nil
	}, mud.Tagged[config.Config]())
	if err != nil {
		panic(err)
	}
}

// Execute is the clingy entry point.
func (m *MudCommand) Execute(ctx context.Context) error {
	if m.runSelector == nil {
		m.runSelector = m.selector
	}
	err := mud.ForEachDependency(m.ball, m.runSelector, func(component *mud.Component) error {
		return component.Init(ctx)
	}, mud.All)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		shutdownTimeout := 15 * time.Second
		if timeoutStr := os.Getenv("STORJ_SHUTDOWN_TIMEOUT"); timeoutStr != "" {
			if timeoutSecs, parseErr := strconv.Atoi(timeoutStr); parseErr == nil && timeoutSecs > 0 {
				shutdownTimeout = time.Duration(timeoutSecs) * time.Second
			}
		}

		closeCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- mud.ForEachDependencyReverse(m.ball, m.runSelector, func(component *mud.Component) error {
				return component.Close(closeCtx)
			}, mud.All)
		}()

		select {
		case err = <-done:
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		case <-time.After(shutdownTimeout):
			if debugPath := os.Getenv("STORJ_SHUTDOWN_DEBUG_PATH"); debugPath != "" {
				pid := os.Getpid()
				timestamp := time.Now().Unix()
				filename := fmt.Sprintf("%d-%d.goroutines", pid, timestamp)
				fullPath := filepath.Join(debugPath, filename)

				buf := make([]byte, 1<<20) // 1MB buffer
				stackSize := runtime.Stack(buf, true)
				err := os.WriteFile(fullPath, buf[:stackSize], 0644)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
				}
			}
			cancel()
			err = <-done
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		}
	}()

	eg := &errgroup.Group{}
	err = mud.ForEachDependency(m.ball, m.runSelector, func(component *mud.Component) error {
		return component.Run(pprof.WithLabels(ctx, pprof.Labels("component", component.Name())), eg)
	}, mud.All)
	if err != nil {
		return errors.WithStack(err)
	}

	return eg.Wait()
}
