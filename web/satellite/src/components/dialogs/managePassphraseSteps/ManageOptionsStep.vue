// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <div class="pa-4">
        <v-sheet v-for="item in items" :key="item.next" class="ma-2">
            <v-list-item class="pa-2 mb-3" border link @click="emit('optionClick', item.next)">
                <template #prepend>
                    <component :is="item.icon" width="20" height="20" class="ml-3 mr-4" />
                </template>
                <v-list-item-title>
                    <p class="font-weight-bold">{{ item.title }}</p>
                </v-list-item-title>
                <v-list-item-subtitle>
                    <p class="text-caption text-wrap text-break">{{ item.subtitle }}</p>
                </v-list-item-subtitle>
                <template #append>
                    <v-icon size="24" :icon="ChevronRight" color="default" />
                </template>
            </v-list-item>
        </v-sheet>
    </div>
</template>

<script setup lang="ts">
import { Component } from 'vue';
import { VSheet, VListItem, VIcon, VListItemTitle, VListItemSubtitle } from 'vuetify/components';
import { ChevronRight } from 'lucide-vue-next';

import { ManageProjectPassphraseStep } from '@/types/managePassphrase';
import { DialogStepComponent } from '@/types/common';

import IconCirclePlus from '@/components/icons/IconCirclePlus.vue';
import IconSwitch from '@/components/icons/IconSwitch.vue';
import IconLock from '@/components/icons/IconLock.vue';

type Item = {
    icon: Component;
    title: string;
    subtitle: string;
    next: ManageProjectPassphraseStep;
};

const items: Item[] = [
    {
        icon: IconCirclePlus,
        title: 'Create a new passphrase',
        subtitle: 'Allows you to upload data with a different passphrase.',
        next: ManageProjectPassphraseStep.Create,
    }, {
        icon: IconSwitch,
        title: 'Switch active passphrase',
        subtitle: 'View and upload data using another passphrase.',
        next: ManageProjectPassphraseStep.Switch,
    }, {
        icon: IconLock,
        title: 'Clear saved passphrase',
        subtitle: 'Lock your data and clear passphrase from this session.',
        next: ManageProjectPassphraseStep.Clear,
    },
];

const emit = defineEmits<{
    'optionClick': [option: ManageProjectPassphraseStep];
}>();

defineExpose<DialogStepComponent>({
    title: 'Manage Passphrase',
});
</script>
