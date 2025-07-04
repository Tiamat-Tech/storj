// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <v-container class="pb-15">
        <minimum-charge-banner v-if="billingEnabled" />

        <trial-expiration-banner v-if="isTrialExpirationBanner && isUserProjectOwner" :expired="isExpired" />

        <card-expire-banner />

        <next-steps-container />

        <low-token-balance-banner
            v-if="isLowBalance && billingEnabled"
            cta-label="Go to billing"
            @click="redirectToBilling"
        />
        <limit-warning-banners v-if="billingEnabled" />

        <v-row align="center" justify="space-between">
            <v-col cols="12" md="auto">
                <PageTitleComponent
                    title="Project Dashboard"
                    extra-info="Project usage statistics are not real-time. Recent uploads, downloads, or other actions may not be immediately reflected."
                />
                <PageSubtitleComponent
                    subtitle="View your project statistics, check daily usage, and set project limits."
                    link="https://docs.storj.io/support/projects"
                />
            </v-col>
            <v-col cols="auto" class="pt-0 mt-0 pt-md-5">
                <v-btn v-if="isUserProjectOwner && !isPaidTier && billingEnabled" variant="outlined" color="default" :prepend-icon="CircleArrowUp" @click="appStore.toggleUpgradeFlow(true)">
                    Upgrade
                </v-btn>
            </v-col>
        </v-row>

        <team-passphrase-banner v-if="isTeamPassphraseBanner" />

        <v-row class="d-flex align-center mt-2">
            <v-col cols="6" md="4" lg="2">
                <CardStatsComponent
                    title="Objects"
                    subtitle="Project total"
                    :data="limits.objectCount.toLocaleString()"
                    :to="ROUTES.Buckets.path"
                    color="info"
                    extra-info="Project usage statistics are not real-time. Recent uploads, downloads, or other actions may not be immediately reflected."
                />
            </v-col>
            <v-col v-if="!emissionImpactViewEnabled" cols="6" md="4" lg="2">
                <CardStatsComponent title="Segments" color="info" subtitle="All object pieces" :data="limits.segmentCount.toLocaleString()" :to="ROUTES.Buckets.path" />
            </v-col>
            <v-col cols="6" md="4" lg="2">
                <CardStatsComponent title="Buckets" color="info" subtitle="In this project" :data="bucketsCount.toLocaleString()" :to="ROUTES.Buckets.path" />
            </v-col>
            <v-col cols="6" md="4" lg="2">
                <CardStatsComponent title="Access Keys" color="info" subtitle="Total keys" :data="accessGrantsCount.toLocaleString()" :to="ROUTES.Access.path" />
            </v-col>
            <v-col cols="6" md="4" lg="2">
                <CardStatsComponent title="Team" color="info" subtitle="Project members" :data="teamSize.toLocaleString()" :to="ROUTES.Team.path" />
            </v-col>
            <template v-if="emissionImpactViewEnabled">
                <v-col cols="12" sm="6" md="4" lg="2">
                    <emissions-dialog />
                    <v-tooltip
                        activator="parent"
                        location="top"
                        offset="-20"
                        opacity="80"
                    >
                        Click to learn more
                    </v-tooltip>
                    <CardStatsComponent title="CO₂ Estimated" subtitle="For this project" color="info" :data="co2Estimated" link />
                </v-col>
                <v-col cols="12" sm="6" md="4" lg="2">
                    <emissions-dialog />
                    <v-tooltip
                        activator="parent"
                        location="top"
                        offset="-20"
                        opacity="80"
                    >
                        Click to learn more
                    </v-tooltip>
                    <CardStatsComponent title="CO₂ Avoided" subtitle="By using Storj" :data="co2Saved" color="success" link />
                </v-col>
            </template>
            <v-col v-if="billingEnabled && !emissionImpactViewEnabled" cols="6" md="4" lg="2">
                <CardStatsComponent title="Billing" :subtitle="`${paidTierString} account`" :data="paidTierString" :to="ROUTES.Account.with(ROUTES.Billing).path" />
            </v-col>
        </v-row>

        <v-row class="d-flex align-center justify-center mb-5">
            <v-col cols="12" md="6" xl="3">
                <UsageProgressComponent
                    icon="storage"
                    title="Storage"
                    :progress="storageUsedPercent"
                    :used="`${usedLimitFormatted(limits.storageUsed)} Used`"
                    :limit="storageLimitTxt"
                    :available="storageAvailableTxt"
                    :cta="storageCTA"
                    :no-limit="noLimitsUiEnabled && ownerHasPaidPrivileges && !limits.userSetStorageLimit"
                    extra-info="Project usage statistics are not real-time. Recent uploads, downloads, or other actions may not be immediately reflected."
                    @cta-click="onNeedMoreClicked(LimitToChange.Storage)"
                />
            </v-col>
            <v-col cols="12" md="6" xl="3">
                <UsageProgressComponent
                    icon="download"
                    title="Download"
                    :progress="egressUsedPercent"
                    :used="`${usedLimitFormatted(limits.bandwidthUsed)} Used`"
                    :limit="bandwidthLimitTxt"
                    :available="bandwidthAvailableTxt"
                    :cta="bandwidthCTA"
                    :no-limit="noLimitsUiEnabled && ownerHasPaidPrivileges && !limits.userSetBandwidthLimit"
                    extra-info="The download bandwidth usage is only for the current billing period of one month."
                    @cta-click="onNeedMoreClicked(LimitToChange.Bandwidth)"
                />
            </v-col>
            <v-col cols="12" md="6" xl="3">
                <UsageProgressComponent
                    icon="segments"
                    title="Segments"
                    :progress="segmentUsedPercent"
                    :used="`${limits.segmentUsed.toLocaleString()} Used`"
                    :limit="`Limit: ${limits.segmentLimit.toLocaleString()}`"
                    :available="`${availableSegment.toLocaleString()} Available`"
                    :cta="getCTALabel(segmentUsedPercent, true)"
                    @cta-click="onSegmentsCTAClicked"
                >
                    <template #extraInfo>
                        <p>
                            Segments are the encrypted parts of an uploaded object.
                            <a
                                class="link"
                                href="https://docs.storj.io/dcs/pricing#per-segment-fee"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Learn more
                            </a>
                        </p>
                    </template>
                </UsageProgressComponent>
            </v-col>
            <v-col cols="12" md="6" xl="3">
                <UsageProgressComponent
                    v-if="isCouponCard"
                    icon="coupon"
                    :title="isFreeTierCoupon ? 'Free Usage' : 'Coupon'"
                    :progress="couponProgress"
                    :used="`${couponProgress}% Used`"
                    :limit="`Included free usage: ${couponValue}`"
                    :available="`${couponRemainingPercent}% Available`"
                    :hide-cta="!isUserProjectOwner"
                    :cta="isFreeTierCoupon ? 'Learn more' : 'View Coupons'"
                    @cta-click="onCouponCTAClicked"
                />
                <UsageProgressComponent
                    v-else
                    icon="bucket"
                    title="Buckets"
                    :progress="bucketsUsedPercent"
                    :used="`${limits.bucketsUsed.toLocaleString()} Used`"
                    :limit="`Limit: ${limits.bucketsLimit.toLocaleString()}`"
                    :available="`${availableBuckets.toLocaleString()} Available`"
                    cta="Need more?"
                    @cta-click="onBucketsCTAClicked"
                />
            </v-col>
        </v-row>

        <v-row align="center" justify="space-between">
            <v-col cols="12" md="auto">
                <v-card-title class="font-weight-bold pl-0">Daily Usage</v-card-title>
                <p class="text-medium-emphasis">
                    Select date range to view daily usage statistics.
                </p>
            </v-col>
            <v-col cols="auto" class="pt-0 mt-2 mt-md-0 pt-md-7">
                <v-date-input
                    v-model="chartDateRange"
                    :allowed-dates="allowDate"
                    label="Select Date Range"
                    min-width="260px"
                    multiple="range"
                    prepend-icon=""
                    density="comfortable"
                    variant="outlined"
                    :loading="isLoading"
                    class="bg-surface"
                    show-adjacent-months
                    hide-details
                >
                    <v-icon class="mr-2" size="20" icon="$calendar" />
                </v-date-input>
            </v-col>
        </v-row>

        <v-row class="d-flex align-center justify-center mt-2 mb-5">
            <v-col cols="12" md="6">
                <v-card ref="chartContainer" class="pa-1 pb-3">
                    <template #title>
                        <v-card-title class="d-flex align-center">
                            <v-icon :icon="Cloud" size="small" color="primary" class="mr-2" />
                            Storage
                        </v-card-title>
                    </template>
                    <v-card-item class="pt-1">
                        <v-card class="dot-background" rounded="md">
                            <StorageChart
                                :width="chartWidth"
                                :height="240"
                                :data="storageUsage"
                                :since="chartsSinceDate"
                                :before="chartsBeforeDate"
                            />
                        </v-card>
                    </v-card-item>
                </v-card>
            </v-col>
            <v-col cols="12" md="6">
                <v-card class="pa-1 pb-3">
                    <template #title>
                        <v-card-title class="d-flex align-center justify-space-between">
                            <v-row class="ma-0 align-center">
                                <v-icon :icon="CloudDownload" size="small" color="primary" class="mr-2" />
                                Download
                                <v-tooltip width="240" location="bottom">
                                    <template #activator="{ props }">
                                        <v-icon v-bind="props" size="12" :icon="Info" class="ml-2 text-medium-emphasis" />
                                    </template>
                                    <template #default>
                                        <p>
                                            Download bandwidth appears here after downloads complete or cancel within 48 hours.
                                        </p>
                                    </template>
                                </v-tooltip>
                            </v-row>
                        </v-card-title>
                    </template>
                    <v-card-item class="pt-1">
                        <v-card class="dot-background" rounded="md">
                            <BandwidthChart
                                :width="chartWidth"
                                :height="240"
                                :data="settledBandwidthUsage"
                                :since="chartsSinceDate"
                                :before="chartsBeforeDate"
                            />
                        </v-card>
                    </v-card-item>
                </v-card>
            </v-col>
        </v-row>

        <v-row align="center" justify="space-between">
            <v-col cols="12" md="auto">
                <v-card-title class="font-weight-bold pl-0">
                    Storage Buckets
                    <v-tooltip width="240" location="bottom">
                        <template #activator="activator">
                            <v-icon v-bind="activator.props" size="14" :icon="Info" color="info" class="ml-1" />
                        </template>
                        <template #default>
                            <p>Project usage statistics are not real-time. Recent uploads, downloads, or other actions may not be immediately reflected.</p>
                        </template>
                    </v-tooltip>
                </v-card-title>
                <p class="text-medium-emphasis">
                    Buckets are where you upload and organize your data.
                </p>
            </v-col>
            <v-col cols="auto" class="pt-0 mt-0 pt-md-5">
                <v-btn
                    variant="outlined"
                    color="default"
                    :prepend-icon="CirclePlus"
                    @click="onCreateBucket"
                >
                    New Bucket
                </v-btn>
            </v-col>
        </v-row>

        <v-row>
            <v-col>
                <buckets-data-table />
            </v-col>
        </v-row>
    </v-container>

    <edit-project-limit-dialog v-model="isEditLimitDialogShown" :limit-type="limitToChange" />
    <create-bucket-dialog v-model="isCreateBucketDialogShown" />
    <CreateBucketDialog v-model="isCreateBucketDialogOpen" />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch , ComponentPublicInstance } from 'vue';
import {
    VBtn,
    VCard,
    VCardTitle,
    VCardItem,
    VCol,
    VContainer,
    VRow,
    VIcon,
    VTooltip,
} from 'vuetify/components';
import { VDateInput } from 'vuetify/labs/components';
import { useRouter } from 'vue-router';
import { Info, CirclePlus, CircleArrowUp, Cloud, CloudDownload } from 'lucide-vue-next';

import { useUsersStore } from '@/store/modules/usersStore';
import { useProjectsStore } from '@/store/modules/projectsStore';
import { useProjectMembersStore } from '@/store/modules/projectMembersStore';
import { useAccessGrantsStore } from '@/store/modules/accessGrantsStore';
import { useBillingStore } from '@/store/modules/billingStore';
import { useBucketsStore } from '@/store/modules/bucketsStore';
import { DataStamp, Emission, LimitToChange, Project, ProjectLimits } from '@/types/projects';
import { Dimensions, Size } from '@/utils/bytesSize';
import { ChartUtils } from '@/utils/chart';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';
import { useNotify } from '@/composables/useNotify';
import { useAppStore } from '@/store/modules/appStore';
import { ProjectMembersPage, ProjectRole } from '@/types/projectMembers';
import { AccessGrantsPage } from '@/types/accessGrants';
import { useConfigStore } from '@/store/modules/configStore';
import { useLowTokenBalance } from '@/composables/useLowTokenBalance';
import { ROUTES } from '@/router';
import { AccountBalance, CreditCard } from '@/types/payments';
import { useLoading } from '@/composables/useLoading';
import { usePreCheck } from '@/composables/usePreCheck';

import PageTitleComponent from '@/components/PageTitleComponent.vue';
import PageSubtitleComponent from '@/components/PageSubtitleComponent.vue';
import CardStatsComponent from '@/components/CardStatsComponent.vue';
import UsageProgressComponent from '@/components/UsageProgressComponent.vue';
import BandwidthChart from '@/components/BandwidthChart.vue';
import StorageChart from '@/components/StorageChart.vue';
import BucketsDataTable from '@/components/BucketsDataTable.vue';
import EditProjectLimitDialog from '@/components/dialogs/EditProjectLimitDialog.vue';
import CreateBucketDialog from '@/components/dialogs/CreateBucketDialog.vue';
import LimitWarningBanners from '@/components/LimitWarningBanners.vue';
import LowTokenBalanceBanner from '@/components/LowTokenBalanceBanner.vue';
import NextStepsContainer from '@/components/onboarding/NextStepsContainer.vue';
import TeamPassphraseBanner from '@/components/TeamPassphraseBanner.vue';
import EmissionsDialog from '@/components/dialogs/EmissionsDialog.vue';
import TrialExpirationBanner from '@/components/TrialExpirationBanner.vue';
import CardExpireBanner from '@/components/CardExpireBanner.vue';
import MinimumChargeBanner from '@/components/MinimumChargeBanner.vue';

type ValueUnit = {
    value: number
    unit: string
};

const appStore = useAppStore();
const usersStore = useUsersStore();
const projectsStore = useProjectsStore();
const pmStore = useProjectMembersStore();
const agStore = useAccessGrantsStore();
const billingStore = useBillingStore();
const bucketsStore = useBucketsStore();
const configStore = useConfigStore();

const notify = useNotify();
const router = useRouter();
const isLowBalance = useLowTokenBalance();
const { isLoading, withLoading } = useLoading();
const { isTrialExpirationBanner, isUserProjectOwner, isExpired, withTrialCheck, withManagedPassphraseCheck } = usePreCheck();

const chartWidth = ref<number>(0);
const chartContainer = ref<ComponentPublicInstance>();
const isEditLimitDialogShown = ref<boolean>(false);
const limitToChange = ref<LimitToChange>(LimitToChange.Storage);
const isCreateBucketDialogShown = ref<boolean>(false);
const isCreateBucketDialogOpen = ref<boolean>(false);
const datePickerModel = ref<Date[]>([]);

/**
 * Returns formatted CO2 estimated info.
 */
const co2Estimated = computed<string>(() => {
    const formatted = getValueAndUnit(Math.round(emission.value.storjImpact));

    return `${formatted.value.toLocaleString()} ${formatted.unit} CO₂e`;
});

/**
 * Returns formatted CO2 save info.
 */
const co2Saved = computed<string>(() => {
    let value = Math.round(emission.value.hyperscalerImpact) - Math.round(emission.value.storjImpact);
    if (value < 0) value = 0;

    const formatted = getValueAndUnit(value);

    return `${formatted.value.toLocaleString()} ${formatted.unit} CO₂e`;
});

/**
 * Indicates if billing coupon card should be shown.
 */
const isCouponCard = computed<boolean>(() => {
    return billingStore.state.coupon !== null &&
        billingEnabled.value &&
        !isPaidTier.value &&
        selectedProject.value.ownerId === usersStore.state.user.id;
});

const productBasedInvoicingEnabled = computed<boolean>(() => configStore.state.config.productBasedInvoicingEnabled);

/**
 * Indicates if billing features are enabled.
 */
const billingEnabled = computed<boolean>(() => configStore.getBillingEnabled(usersStore.state.user));

/**
 * Returns percent of coupon used.
 */
const couponProgress = computed((): number => {
    if (!billingStore.state.coupon) {
        return 0;
    }

    let charges;
    if (productBasedInvoicingEnabled.value) {
        charges = billingStore.state.productCharges.getPrice();
    } else {
        charges = billingStore.state.projectCharges.getPrice();
    }
    const couponValue = billingStore.state.coupon.amountOff;
    if (charges > couponValue) {
        return 100;
    }
    return Math.round(charges / couponValue * 100);
});

/**
 * Indicates if active coupon is free tier coupon.
 */
const isFreeTierCoupon = computed((): boolean => {
    if (!billingStore.state.coupon) {
        return true;
    }

    const freeTierCouponName = 'Free Tier';

    return billingStore.state.coupon.name.includes(freeTierCouponName);
});

/**
 * Returns coupon value.
 */
const couponValue = computed((): string => {
    return billingStore.state.coupon?.amountOff ? '$' + (billingStore.state.coupon.amountOff * 0.01).toLocaleString() : billingStore.state.coupon?.percentOff.toLocaleString() + '%';
});

/**
 * Returns percent of coupon value remaining.
 */
const couponRemainingPercent = computed((): number => {
    return 100 - couponProgress.value;
});

/**
 * Whether the new no-limits UI is enabled.
 */
const noLimitsUiEnabled = computed((): boolean => {
    return configStore.state.config.noLimitsUiEnabled;
});

/**
 * Whether the user is in paid tier.
 */
const isPaidTier = computed((): boolean => {
    return usersStore.state.user.isPaid;
});

/**
 * Whether project members passphrase banner should be shown.
 */
const isTeamPassphraseBanner = computed<boolean>(() => {
    return !usersStore.state.settings.noticeDismissal.projectMembersPassphrase && teamSize.value > 1;
});

/**
 * Returns user account tier string.
 */
const paidTierString = computed((): string => {
    return isPaidTier.value ? 'Pro' : 'Free';
});

/**
 * Returns current limits from store.
 */
const limits = computed((): ProjectLimits => {
    return projectsStore.state.currentLimits;
});

/**
 * Returns remaining segments available.
 */
const availableSegment = computed((): number => {
    const diff = limits.value.segmentLimit - limits.value.segmentUsed;
    return diff < 0 ? 0 : diff;
});

/**
 * Returns percentage of segment limit used.
 */
const segmentUsedPercent = computed((): number => {
    return limits.value.segmentUsed / limits.value.segmentLimit * 100;
});

/**
 * Returns whether this project is owned by a paid tier user.
 */
const isProjectOwnerPaidTier = computed(() => projectsStore.selectedProjectConfig.isOwnerPaidTier);
/**
 * Returns whether the owner of this project has paid privileges
 */
const ownerHasPaidPrivileges = computed(() => projectsStore.selectedProjectConfig.hasPaidPrivileges);

/**
 * Returns whether this project is owned by the current user
 * or whether they're an admin.
 */
const isProjectOwnerOrAdmin = computed(() => {
    const isAdmin = projectsStore.selectedProjectConfig.role === ProjectRole.Admin;
    return isUserProjectOwner.value || isAdmin;
});

/**
 * Returns remaining egress available.
 */
const availableEgress = computed((): number => {
    let diff = (limits.value.userSetBandwidthLimit || limits.value.bandwidthLimit) - limits.value.bandwidthUsed;
    if (ownerHasPaidPrivileges.value && noLimitsUiEnabled.value && !limits.value.userSetBandwidthLimit) {
        diff = Number.MAX_SAFE_INTEGER;
    } else if (!noLimitsUiEnabled.value) {
        diff = limits.value.bandwidthLimit - limits.value.bandwidthUsed;
    }
    return diff < 0 ? 0 : diff;
});

/**
 * Returns percentage of egress limit used.
 */
const egressUsedPercent = computed((): number => {
    return limits.value.bandwidthUsed / (limits.value.userSetBandwidthLimit || limits.value.bandwidthLimit) * 100;
});

/**
 * Returns the CTA text on the bandwidth usage card.
 */
const bandwidthCTA = computed((): string => {
    if (!ownerHasPaidPrivileges.value) {
        return getCTALabel(egressUsedPercent.value);
    }
    if (limits.value.userSetBandwidthLimit) {
        return 'Edit / Remove Limit';
    } else {
        return 'Set Download Limit';
    }
});

/**
 * Returns the used bandwidth text for the storage usage card.
 */
const bandwidthLimitTxt = computed((): string => {
    if (ownerHasPaidPrivileges.value && noLimitsUiEnabled.value && !limits.value.userSetBandwidthLimit) {
        return 'This Month';
    }
    return `Limit: ${usedLimitFormatted(limits.value.userSetBandwidthLimit || limits.value.bandwidthLimit)}`;
});

/**
 * Returns the available bandwidth text for the storage usage card.
 */
const bandwidthAvailableTxt = computed((): string => {
    if (availableEgress.value === Number.MAX_SAFE_INTEGER) {
        return `∞ No Limit`;
    }
    return `${usedLimitFormatted(availableEgress.value)} Available`;
});

/**
 * Returns remaining storage available.
 */
const availableStorage = computed((): number => {
    let diff = (limits.value.userSetStorageLimit || limits.value.storageLimit) - limits.value.storageUsed;
    if (ownerHasPaidPrivileges.value && noLimitsUiEnabled.value && !limits.value.userSetStorageLimit) {
        diff = Number.MAX_SAFE_INTEGER;
    } else if (!noLimitsUiEnabled.value) {
        diff = limits.value.storageLimit - limits.value.storageUsed;
    }
    return diff < 0 ? 0 : diff;
});

/**
 * Returns percentage of storage limit used.
 */
const storageUsedPercent = computed((): number => {
    return limits.value.storageUsed / (limits.value.userSetStorageLimit || limits.value.storageLimit) * 100;
});

/**
 * Returns the CTA text on the storage usage card.
 */
const storageCTA = computed((): string => {
    if (!ownerHasPaidPrivileges.value) {
        return getCTALabel(storageUsedPercent.value);
    }
    if (limits.value.userSetStorageLimit) {
        return 'Edit / Remove Limit';
    } else {
        return 'Set Storage Limit';
    }
});

/**
 * Returns the used storage text for the storage usage card.
 */
const storageLimitTxt = computed((): string => {
    if (ownerHasPaidPrivileges.value && noLimitsUiEnabled.value && !limits.value.userSetStorageLimit) {
        return 'Total';
    }
    return `Limit: ${usedLimitFormatted(limits.value.userSetStorageLimit || limits.value.storageLimit)}`;
});

/**
 * Returns the available storage text for the storage usage card.
 */
const storageAvailableTxt = computed((): string => {
    if (availableStorage.value === Number.MAX_SAFE_INTEGER) {
        return `∞ No Limit`;
    }
    return `${usedLimitFormatted(availableStorage.value)} Available`;
});

/**
 * Returns percentage of buckets limit used.
 */
const bucketsUsedPercent = computed((): number => {
    return limits.value.bucketsUsed / limits.value.bucketsLimit * 100;
});

/**
 * Returns remaining buckets available.
 */
const availableBuckets = computed((): number => {
    const diff = limits.value.bucketsLimit - limits.value.bucketsUsed;
    return diff < 0 ? 0 : diff;
});

/**
 * Get selected project from store.
 */
const selectedProject = computed((): Project => {
    return projectsStore.state.selectedProject;
});

/**
 * Returns current team size from store.
 */
const teamSize = computed((): number => {
    return pmStore.state.page.totalCount;
});

/**
 * Returns access grants count from store.
 */
const accessGrantsCount = computed((): number => {
    return agStore.state.page.totalCount;
});

/**
 * Returns access grants count from store.
 */
const bucketsCount = computed((): number => {
    return bucketsStore.state.page.totalCount;
});

/**
 * Returns charts since date from store.
 */
const chartsSinceDate = computed((): Date => {
    return projectsStore.state.chartDataSince;
});

/**
 * Returns charts before date from store.
 */
const chartsBeforeDate = computed((): Date => {
    return projectsStore.state.chartDataBefore;
});

/**
 * Return a new 7 days range if datePickerModel is empty.
 */
const chartDateRange = computed<Date[]>({
    get: () => {
        const dates: Date[] = [...datePickerModel.value];
        if (!dates.length) {
            for (let i = 6; i >= 0; i--) {
                const d = new Date();
                d.setDate(d.getDate() - i);
                dates.push(d);
            }
        }
        return dates;
    },
    set: newValue => {
        const newRange = [...newValue];
        if (newRange.length === 0) {
            return;
        }
        if (newRange.length < 2) {
            const d = new Date();
            d.setDate(newRange[0].getDate() + 1);
            newRange.push(d);
        }
        datePickerModel.value = newRange;
    },
});

/**
 * Returns storage chart data from store.
 */
const storageUsage = computed((): DataStamp[] => {
    return ChartUtils.populateEmptyUsage(
        projectsStore.state.storageChartData, chartsSinceDate.value, chartsBeforeDate.value,
    );
});

/**
 * Returns allocated bandwidth chart data from store.
 */
const settledBandwidthUsage = computed((): DataStamp[] => {
    return ChartUtils.populateEmptyUsage(
        projectsStore.state.settledBandwidthChartData, chartsSinceDate.value, chartsBeforeDate.value,
    );
});

/**
 * Indicates if emission impact view should be shown.
 */
const emissionImpactViewEnabled = computed<boolean>(() => {
    return configStore.state.config.emissionImpactViewEnabled;
});

/**
 * Returns project's emission impact.
 */
const emission = computed<Emission>(()  => {
    return projectsStore.state.emission;
});

function allowDate(date: unknown): boolean {
    if (!date) return false;
    const d = new Date(date as string);
    if (isNaN(d.getTime())) return false;

    d.setHours(0, 0, 0, 0);
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    return d <= today;
}

/**
 * Returns adjusted value and unit.
 */
function getValueAndUnit(value: number): ValueUnit {
    const unitUpgradeThreshold = 999999;
    const [newValue, unit] = value > unitUpgradeThreshold ? [value / 1000, 't'] : [value, 'kg'];

    return { value: newValue, unit };
}

/**
 * Starts create bucket flow if user's free trial is not expired.
 */
function onCreateBucket(): void {
    withTrialCheck(() => { withManagedPassphraseCheck(() => {
        isCreateBucketDialogOpen.value = true;
    });});
}

/**
 * Returns formatted amount.
 */
function usedLimitFormatted(value: number): string {
    return formattedValue(new Size(value, 2));
}

/**
 * Formats value to needed form and returns it.
 */
function formattedValue(value: Size): string {
    switch (value.label) {
    case Dimensions.Bytes:
        return '0';
    default:
        return `${value.formattedBytes.replace(/\.0+$/, '')}${value.label}`;
    }
}

/**
 * Used container size recalculation for charts resizing.
 */
function recalculateChartWidth(): void {
    chartWidth.value = chartContainer.value?.$el.getBoundingClientRect().width - 16 || 0;
}

/**
 * Conditionally opens the upgrade dialog
 * or the edit limit dialog.
 */
function onNeedMoreClicked(source: LimitToChange): void {
    if (isUserProjectOwner.value && !isPaidTier.value && billingEnabled.value) {
        appStore.toggleUpgradeFlow(true);
        return;
    }
    if (!ownerHasPaidPrivileges.value) {
        notify.notify('Contact project owner to upgrade to edit limits');
        return;
    }
    if (!isProjectOwnerOrAdmin.value) {
        notify.notify('Contact project owner or admin to edit limits');
        return;
    }
    limitToChange.value = source;
    isEditLimitDialogShown.value = true;
}

/**
 * Returns CTA label based on paid tier status and current usage.
 */
function getCTALabel(usage: number, isSegment = false): string {
    if (isUserProjectOwner.value && !isPaidTier.value && billingEnabled.value) {
        if (usage >= 100) {
            return 'Upgrade now';
        }
        if (usage >= 80) {
            return 'Upgrade';
        }
        return 'Need more?';
    }

    if (isSegment) return 'Learn more';

    if (usage >= 80) {
        return 'Increase limits';
    }
    return 'Edit Limit';
}

/**
 * Conditionally opens the upgrade dialog or docs link.
 */
function onSegmentsCTAClicked(): void {
    if (isUserProjectOwner.value && !isPaidTier.value && billingEnabled.value) {
        appStore.toggleUpgradeFlow(true);
        return;
    }

    window.open('https://storj.dev/support/usage-limit-increases#segment-limit', '_blank', 'noreferrer');
}

/**
 * Conditionally opens docs link or navigates to billing overview.
 */
function onCouponCTAClicked(): void {
    if (isFreeTierCoupon.value) {
        window.open('https://docs.storj.io/dcs/pricing#free-tier', '_blank', 'noreferrer');
        return;
    }

    redirectToBilling();
}

/**
 * Opens limit increase request link in a new tab.
 */
function onBucketsCTAClicked(): void {
    if (isUserProjectOwner.value && !isPaidTier.value && billingEnabled.value) {
        appStore.toggleUpgradeFlow(true);
        return;
    }
    if (!ownerHasPaidPrivileges.value) {
        notify.notify('Contact project owner to upgrade to edit limits');
        return;
    }
    if (!isProjectOwnerOrAdmin.value) {
        notify.notify('Contact project owner or admin to edit limits');
        return;
    }

    window.open(configStore.state.config.projectLimitsIncreaseRequestURL, '_blank', 'noreferrer');
}

/**
 * Redirects to Billing Page tab.
 */
function redirectToBilling(): void {
    router.push({ name: ROUTES.Billing.name });
}

/**
 * Lifecycle hook after initial render.
 * Fetches project limits.
 */
onMounted(async (): Promise<void> => {
    const projectID = selectedProject.value.id;
    const FIRST_PAGE = 1;

    window.addEventListener('resize', recalculateChartWidth);
    recalculateChartWidth();

    const promises: Promise<void | ProjectMembersPage | AccessGrantsPage | AccountBalance | CreditCard[]>[] = [
        projectsStore.getDailyProjectData({ since: chartDateRange.value[0], before: chartDateRange.value[chartDateRange.value.length - 1] }),
        pmStore.getProjectMembers(FIRST_PAGE, projectID),
        agStore.getAccessGrants(FIRST_PAGE, projectID),
        bucketsStore.getBuckets(FIRST_PAGE, projectID),
    ];

    if (emissionImpactViewEnabled.value) {
        promises.push(projectsStore.getEmissionImpact(projectID));
    }

    if (billingEnabled.value) {
        promises.push(
            billingStore.getBalance(),
            billingStore.getCreditCards(),
            billingStore.getCoupon(),
        );

        if (productBasedInvoicingEnabled.value) {
            promises.push(billingStore.getProductUsageAndChargesCurrentRollup());
        } else {
            promises.push(billingStore.getProjectUsageAndChargesCurrentRollup());
        }
    }

    if (configStore.state.config.nativeTokenPaymentsEnabled && billingEnabled.value) {
        promises.push(billingStore.getNativePaymentsHistory());
    }

    try {
        await Promise.all(promises);
    } catch (error) {
        notify.notifyError(error, AnalyticsErrorEventSource.PROJECT_DASHBOARD_PAGE);
    }
});

/**
 * Lifecycle hook before component destruction.
 * Removes event listener on window resizing.
 */
onBeforeUnmount((): void => {
    window.removeEventListener('resize', recalculateChartWidth);
    appStore.toggleHasJustLoggedIn(false);
});

watch(datePickerModel, async (newRange) => {
    if (newRange.length < 2) return;

    await withLoading(async () => {
        let startDate = newRange[0];
        let endDate = newRange[newRange.length - 1];
        if (startDate.getTime() > endDate.getTime()) {
            [startDate, endDate] = [endDate, startDate];
        }

        const since = new Date(startDate);
        const before = new Date(endDate);
        before.setHours(23, 59, 59, 999);

        try {
            await projectsStore.getDailyProjectData({ since, before });
        } catch (error) {
            notify.notifyError(error, AnalyticsErrorEventSource.PROJECT_DASHBOARD_PAGE);
        }
    });
});
</script>
<style scoped lang="scss">
:deep(.v-field__input) {
    cursor: pointer;

    input {
        cursor: pointer;
    }
}

.dot-background {
    background-image: radial-gradient(circle, rgb(var(--v-theme-on-surface),0.04) 1px, transparent 1px);
    background-size: 12px 12px;
    background-color: rgb(var(--v-theme-surface));;
}
</style>
