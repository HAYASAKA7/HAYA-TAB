<script setup lang="ts">
defineProps<{
  name: string
}>()
</script>

<template>
  <svg class="menu-icon" viewBox="0 0 24 24" fill="currentColor">
    <!-- Pin -->
    <path v-if="name === 'pin'"
      d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />

    <!-- Unpin -->
    <g v-else-if="name === 'unpin'">
      <path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />
      <path d="M2.27 3.27l19.46 19.46 1.41-1.41L3.68 1.86 2.27 3.27z" />
    </g>

    <!-- Close -->
    <path v-else-if="name === 'close'"
      d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12 19 6.41z" />

    <!-- Open -->
    <path v-else-if="name === 'open'"
      d="M19 19H5V5h7V3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7h-2v7zM14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7z" />

    <!-- Rename / Edit -->
    <path v-else-if="name === 'rename' || name === 'edit'"
      d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1 1 0 0 0 0-1.41l-2.34-2.34a1 1 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z" />

    <!-- Delete -->
    <path v-else-if="name === 'delete'"
      d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM8 9h8v10H8V9zm7.5-5l-1-1h-5l-1 1H5v2h14V4h-3.5z" />

    <!-- System App -->
    <path v-else-if="name === 'system'"
      d="M21 2H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h7l-2 3v1h8v-1l-2-3h7c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H3V4h18v12z" />

    <!-- Inner Viewer -->
    <path v-else-if="name === 'viewer'"
      d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z" />

    <!-- Folder / Category -->
    <path v-else-if="name === 'folder' || name === 'category'"
      d="M10 4H4a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-8l-2-2z" />

    <!-- Add to Category -->
    <path v-else-if="name === 'add-folder'"
      d="M20 6h-8l-2-2H4c-1.11 0-1.99.89-1.99 2L2 18c0 1.11.89 2 2 2h16c1.11 0 2-.89 2-2V8c0-1.11-.89-2-2-2zm-1 8h-3v3h-2v-3h-3v-2h3v-3h2v3h3v2z" />

    <!-- Remove from Category -->
    <path v-else-if="name === 'remove-folder'"
      d="M20 6h-8l-2-2H4c-1.11 0-1.99.89-1.99 2L2 18c0 1.11.89 2 2 2h16c1.11 0 2-.89 2-2V8c0-1.11-.89-2-2-2zm-5 8H9v-2h6v2z" />

    <!-- Export -->
    <path v-else-if="name === 'export'"
      d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z" />

    <!-- Upload / Cloud Upload -->
    <path v-else-if="name === 'upload' || name === 'cloud-upload'"
      d="M9 16h6v-6h4l-7-7-7 7h4v6zm-4 2h14v2H5v-2z" />

    <!-- Download / Cloud Download -->
    <path v-else-if="name === 'download' || name === 'cloud-download'"
      d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z" />

    <!-- Unlink -->
    <path v-else-if="name === 'unlink'"
      d="M17 7h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1 0 1.43-.98 2.63-2.31 2.98l1.46 1.46C20.88 15.61 22 13.95 22 12c0-2.76-2.24-5-5-5zm-1 4h-2.19l2 2H16v-2zM2 4.27l3.11 3.11C3.29 8.12 2 9.91 2 12c0 2.76 2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1 0-1.59 1.21-2.9 2.76-3.07L8.73 11H8v2h2.73L17.59 20 19 18.59 3.41 3 2 4.27z" />

    <!-- Remove -->
    <path v-else-if="name === 'remove'"
      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z" />

    <!-- Info / Metadata -->
    <path v-else-if="name === 'info'"
      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z" />
  </svg>
</template>

<style scoped>
.menu-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: currentColor;
}
</style>
