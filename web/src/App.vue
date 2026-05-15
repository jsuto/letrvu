<template>
  <router-view />
</template>

<script setup>
import { useDarkMode } from './composables/useDarkMode'
useDarkMode() // applies data-theme attribute on mount
</script>

<style>
input, textarea, select {
  background: var(--color-surface);
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  font-size: 14px;
  color: var(--color-text);
  background: var(--color-bg);
}

:root {
  --color-teal: #1D9E75;
  --color-teal-light: #E1F5EE;
  --color-text: #1a1a1a;
  --color-text-muted: #888;
  --color-border: #e5e5e3;
  --color-surface: #ffffff;
  --color-bg: #f5f5f3;
  --sidebar-width: 260px;
  --list-width: 360px;
}

[data-theme="dark"] {
  --color-teal-light: #1a3329;
  --color-text: #e8e8e6;
  --color-text-muted: #777;
  --color-border: #2e2e2c;
  --color-surface: #1e1e1c;
  --color-bg: #141412;
}
</style>
