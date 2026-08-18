<template>
  <div>
    <NuxtLink :to="to" :href="href" class="nuxt-link" :target="target">
      <var-link underline="none" class="popup-menu-item">
        <Icon :name="iconName" size="24" style="margin: 16px" />
        <div style="font-size: large">{{ text }}</div>
      </var-link>
    </NuxtLink>
  </div>
</template>

<script lang="ts" setup>
defineProps<{
  iconName: string
  text: string
  to?: string
  href?: string
  target?: string
}>()
</script>

<style scoped>
.nuxt-link {
  text-decoration: none;
}

.popup-menu-item {
  border-radius: 0;
  height: 56px;
  display: flex;
  justify-content: flex-start;
}

.popup-menu-item:hover {
  background-color: rgba(0, 0, 0, 0.2);
}

.popup-menu-item {
  transition: background-color 0.2s;
}
</style>
