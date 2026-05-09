<template>
  <router-view />
</template>

<script setup>
</script>

<style>
*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  font-size: 14px;
  color: #1a1a1a;
  background: #f5f5f3;
}

:root {
  --color-teal: #1D9E75;
  --color-teal-light: #E1F5EE;
  --color-text: #1a1a1a;
  --color-text-muted: #888;
  --color-border: #e5e5e3;
  --color-surface: #ffffff;
  --color-bg: #f5f5f3;
  --sidebar-width: 220px;
  --list-width: 320px;
}
</style>
