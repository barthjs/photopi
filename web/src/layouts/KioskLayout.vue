<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { useEventListener } from '@vueuse/core';

// The context menu is blocked on the guest pages only, the admin pages keep it
useEventListener(document, 'contextmenu', (event: Event) => {
  event.preventDefault();
});

onMounted(() => {
  document.documentElement.classList.add('kiosk-mode');
});

onUnmounted(() => {
  document.documentElement.classList.remove('kiosk-mode');
});
</script>

<template>
  <!-- touch-manipulation blocks zooming by double tap and pinch, but keeps panning and scrolling. -->
  <div class="bg-default flex h-dvh w-full touch-manipulation flex-col overflow-hidden select-none">
    <RouterView />
  </div>
</template>
