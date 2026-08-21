<template>
  <nav v-if="isSmall" class="mobile-tabbar glass-panel">
    <button v-for="t in tabs" :key="t.name" class="tab-item" :class="{ active: activeName === t.name }"
      @click="handleChange(t)" :title="t.label">
      <Icon :name="t.icon" size="22" />
      <span class="tab-label">{{ t.label }}</span>
    </button>
  </nav>
</template>

<script setup lang="ts">
const isSmall = useSmallWindow()
const route = useRoute()

const tabs = [
  { name: 'home', icon: 'i-mdi:home-outline', label: '首页', to: '/' },
  { name: 'random', icon: 'i-mdi:shuffle', label: '随机', to: '/random' },
  { name: 'favorites', icon: 'i-mdi:heart-outline', label: '收藏', to: '/favorites' },
  { name: 'settings', icon: 'i-mdi:cog-outline', label: '设置', to: '/settings' }
]

const activeName = computed(() => {
  const path = route.path
  if (path === '/') return 'home'
  if (path.startsWith('/random')) return 'random'
  if (path.startsWith('/favorites')) return 'favorites'
  if (path.startsWith('/settings')) return 'settings'
  return ''
})

const handleChange = (tab: (typeof tabs)[number]) => {
  if (tab.to !== route.path) {
    navigateTo(tab.to)
  }
}
</script>

<style scoped>
.mobile-tabbar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  display: flex;
  justify-content: space-around;
  align-items: center;
  padding: 6px 0 calc(6px + env(safe-area-inset-bottom));
  background: var(--glass-bg);
  backdrop-filter: blur(var(--glass-blur));
  -webkit-backdrop-filter: blur(var(--glass-blur));
  border-top: 1px solid var(--glass-border);
}

.tab-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 14px;
  border: none;
  background: transparent;
  color: var(--color-on-surface-variant);
  cursor: pointer;
  border-radius: 12px;
  transition: color 0.2s ease, background 0.2s ease;
}

.tab-item.active {
  color: var(--color-primary);
  background: var(--gradient-soft);
}

.tab-item:active {
  transform: scale(0.94);
}

.tab-label {
  font-size: 11px;
  line-height: 1;
}
</style>
