<template>
  <div class="favorites-page">
    <div class="favorites-header">
      <h1 class="favorites-title">我的收藏</h1>
      <span class="favorites-count" v-if="favoritesStore.count > 0">
        {{ favoritesStore.count }} 件作品
      </span>
      <var-button size="small" text-color="#f44336" v-if="favoritesStore.count > 0" @click="clearAll">
        <var-icon name="delete-outline" />
        清空收藏
      </var-button>
    </div>

    <div v-if="favoritesStore.count === 0" class="empty-state">
      <var-icon name="heart-outline" size="64" color="#b0bec5" />
      <p class="empty-text">还没有收藏任何作品</p>
      <var-link underline="none" :to="'/'">
        <var-button type="primary" round>去逛逛</var-button>
      </var-link>
    </div>

    <div v-else class="favorites-grid">
      <div class="favorite-card var-elevation--2" v-for="item in favoritesStore.list" :key="item.id">
        <var-link underline="none" :to="`/artwork/${item.id}`" class="card-link">
          <div class="card-thumbnail">
            <img :src="item.thumbnail" :alt="item.title" loading="lazy" />
            <span class="card-r18-badge" v-if="item.r18">R18</span>
          </div>
          <div class="card-info">
            <div class="card-title" :title="item.title">{{ item.title || '(无标题)' }}</div>
            <div class="card-artist">{{ item.artist_name }}</div>
            <div class="card-meta">
              <span class="card-source">{{ item.source_type }}</span>
              <span class="card-date">{{ formatDate(item.added_at) }}</span>
            </div>
          </div>
        </var-link>
        <var-button size="small" text-color="#f44336" class="card-remove" @click="removeOne(item.id)">
          <var-icon name="close" />
        </var-button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Snackbar } from '@varlet/ui'
import { useFavoritesStore } from '~/stores/store'

const favoritesStore = useFavoritesStore()

const formatDate = (ts: number) => {
  const d = new Date(ts)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const removeOne = (id: string) => {
  favoritesStore.remove(id)
  Snackbar({ content: '已移除收藏', position: 'bottom', type: 'success' })
}

const clearAll = () => {
  favoritesStore.clear()
  Snackbar({ content: '已清空收藏', position: 'bottom', type: 'success' })
}
</script>

<style scoped>
.favorites-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px 16px;
  min-height: 70vh;
}

.favorites-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.favorites-title {
  font-size: 26px;
  font-weight: 700;
  margin: 0;
}

.favorites-count {
  color: hsla(var(--hsl-text), 0.6);
  font-size: 14px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 80px 0;
}

.empty-text {
  color: hsla(var(--hsl-text), 0.6);
  font-size: 16px;
  margin: 0;
}

.favorites-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}

.favorite-card {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(192, 238, 240, 0.08);
  transition: transform 0.2s ease;
}

.favorite-card:hover {
  transform: translateY(-3px);
}

.card-link {
  display: block;
  color: inherit;
  text-decoration: none;
}

.card-thumbnail {
  position: relative;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.1);
}

.card-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.card-r18-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  background: #f44336;
  color: #fff;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 600;
}

.card-info {
  padding: 10px 12px 12px;
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.card-artist {
  font-size: 12px;
  color: hsla(var(--hsl-text), 0.6);
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: hsla(var(--hsl-text), 0.5);
}

.card-source {
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.card-remove {
  position: absolute;
  top: 8px;
  left: 8px;
  background: rgba(0, 0, 0, 0.45);
  border-radius: 50%;
  min-width: 28px;
  width: 28px;
  height: 28px;
  padding: 0;
}

@media (max-width: 600px) {
  .favorites-grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }

  .favorites-title {
    font-size: 22px;
  }
}
</style>
