// Copyright (C) 2024 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <v-form ref="form" class="pa-6" @submit.prevent="emit('submit')">
        <v-row>
            <v-col cols="12">
                <v-text-field
                    v-model="name"
                    label="Access Name"
                    placeholder="Enter access name"
                    variant="outlined"
                    autofocus
                    :hide-details="false"
                    :rules="nameRules"
                    maxlength="100"
                    class="mb-n5 mt-2"
                    required
                />
            </v-col>
            <v-col>
                <p>Access type</p>
                <v-chip-group
                    v-model="accessType"
                    class="mb-3"
                    selected-class="font-weight-bold"
                    color="info"
                    mandatory
                    column
                >
                    <v-chip
                        :key="AccessType.S3"
                        :value="AccessType.S3"
                        color="info"
                        variant="outlined"
                        filter
                    >
                        S3 Credentials
                    </v-chip>

                    <v-chip
                        :key="AccessType.AccessGrant"
                        :value="AccessType.AccessGrant"
                        color="info"
                        variant="outlined"
                        filter
                    >
                        Access Grant
                    </v-chip>

                    <v-chip
                        v-if="!hasManagedPassphrase"
                        :key="AccessType.APIKey"
                        :value="AccessType.APIKey"
                        color="info"
                        variant="outlined"
                        filter
                    >
                        API Key
                    </v-chip>
                </v-chip-group>

                <v-alert v-if="accessType === AccessType.S3" variant="tonal" width="auto">
                    <p class="text-subtitle-2">Gives access through S3 compatible applications. <a href="https://docs.storj.io/dcs/access#create-s3-credentials" target="_blank" rel="noopener noreferrer" class="link">Learn more in the documentation.</a></p>
                </v-alert>

                <v-alert v-else-if="accessType === AccessType.AccessGrant" variant="tonal" width="auto">
                    <p class="text-subtitle-2">Gives access through native clients such as uplink. <a href="https://docs.storj.io/learn/concepts/access/access-grants" target="_blank" rel="noopener noreferrer" class="link">Learn more in the documentation.</a></p>
                </v-alert>

                <v-alert v-else-if="accessType === AccessType.APIKey" variant="tonal" width="auto">
                    <p class="text-subtitle-2">Use it for generating access keys programatically. <a href="https://docs.storj.io/learn/concepts/access/access-grants/api-key" target="_blank" rel="noopener noreferrer" class="link">Learn more in the documentation.</a></p>
                </v-alert>
            </v-col>
        </v-row>
    </v-form>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { VAlert, VChip, VChipGroup, VCol, VForm, VRow, VTextField } from 'vuetify/components';

import { AccessType } from '@/types/setupAccess';
import { IDialogFlowStep, RequiredRule, ValidationRule } from '@/types/common';
import { useAccessGrantsStore } from '@/store/modules/accessGrantsStore';
import { useProjectsStore } from '@/store/modules/projectsStore';

const agStore = useAccessGrantsStore();
const projectsStore = useProjectsStore();

const emit = defineEmits<{
    'nameChanged': [name: string];
    'typeChanged': [type: AccessType];
    'submit': [];
}>();

const hasManagedPassphrase = computed(() => !!projectsStore.state.selectedProjectConfig.passphrase);

const form = ref<VForm | null>(null);
const name = ref<string>('');
const accessType = ref<AccessType>(AccessType.S3);

const nameRules: ValidationRule<string>[] = [
    RequiredRule,
    v => !agStore.state.allAGNames.includes(v) || 'This name is already in use',
];

watch(name, value => emit('nameChanged', value));
watch(accessType, value => emit('typeChanged', value));

defineExpose<IDialogFlowStep>({
    validate: () => {
        form.value?.validate();
        return !!form.value?.isValid;
    },
});
</script>
