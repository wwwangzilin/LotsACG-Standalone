<template>
  <div>
    <var-paper ripple class="card" @click.left="handleCardClick(item)" @click.right.prevent="handleRightClick"
      @mouseenter="hovered = true" @mouseleave="hovered = false">
      <div :data-id="item.id" class="card-content" underline="none" rel="prefetch">
        <div class="cover" :style="{
          aspectRatio:
            firstPic?.width && firstPic?.height
              ? `${firstPic.width} / ${firstPic.height}`
              : '1 / 1'
        }">
          <Transition>
            <img v-if="thumbHashDataURL && !loaded" :src="thumbHashDataURL" :alt="item.detail.title" class="img"
              loading="lazy" />
          </Transition>
          <Transition>
            <img v-if="loaded" :src="firstPic?.thumbnail" :alt="item.detail.title" class="img" loading="lazy"
              ref="cardImage" />
          </Transition>

          <!-- hover 操作层: 收藏 / 预览 / 下载 -->
          <div class="card-actions" v-show="hovered || isSmall" @click.stop>
            <var-button size="small" round :type="isFavorited ? 'danger' : 'primary'" @click="toggleFavorite"
              :title="isFavorited ? '取消收藏' : '收藏'">
              <var-icon :name="isFavorited ? 'heart' : 'heart-outline'" size="16" />
            </var-button>
            <var-button size="small" round type="info" @click="handlePreview" title="预览">
              <var-icon name="eye-outline" size="16" />
            </var-button>
            <var-button size="small" round type="warning" @click="handleDownload" title="下载原图">
              <var-icon name="download-outline" size="16" />
            </var-button>
          </div>
        </div>

        <div class="overlay" v-if="!onlyImage">
          <div class="card-body">
            <h3>{{ item.detail.title }}</h3>
            <div class="author">
              {{ item.detail.artist.name }}
            </div>
          </div>
        </div>
      </div>
    </var-paper>
    <var-image-preview v-model:show="showViewer" :images="item.detail.pictures.map((pic) => pic.regular)"
      close-on-key-escape closeable />
  </div>
</template>

<script setup lang="ts">
import type { WaterfallItem } from '~/types/artwork'
import { thumbHashToDataURL } from 'thumbhash'
import { Snackbar } from '@varlet/ui'
import filesaver from 'file-saver'

const props = withDefaults(
  defineProps<{
    item: WaterfallItem
    onlyImage?: boolean
  }>(),
  {
    onlyImage: false
  }
)

const hovered = ref(false)
const isSmall = useSmallWindow()

const imageLoadCache = useState<Map<string, boolean>>('imageLoadCache', () => new Map())
const MAX_CACHE_SIZE = 1000

const addToCache = (id: string, loaded: boolean) => {
  if (imageLoadCache.value.size >= MAX_CACHE_SIZE) {
    const firstKey = imageLoadCache.value.keys().next().value
    if (firstKey) {
      imageLoadCache.value.delete(firstKey)
    }
  }
  imageLoadCache.value.set(id, loaded)
}

const firstPic = computed(() => props.item.detail.pictures?.[0])

const thumbHashDataURL = computed(() => {
  const hash = firstPic.value?.thumb_hash
  if (!hash) return ''
  try {
    const binary = atob(hash)
    const buffer = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) {
      buffer[i] = binary.charCodeAt(i)
    }
    return thumbHashToDataURL(buffer)
  } catch (e) {
    console.error('thumbHash decode error:', e)
    return ''
  }
})

const loaded = ref(false)

onBeforeMount(() => {
  if (!firstPic.value?.thumbnail) {
    loaded.value = false
    return
  }
  const cachedStatus = imageLoadCache.value.get(firstPic.value.id)
  if (cachedStatus !== undefined) {
    loaded.value = cachedStatus
    return
  }
  const image = new Image()
  image.onload = () => {
    loaded.value = true
    addToCache(firstPic.value!.id, true)
  }
  image.onerror = (error) => {
    console.error(error)
    loaded.value = true
    addToCache(firstPic.value!.id, false)
  }
  image.src = firstPic.value.thumbnail
})

const showViewer = ref(false)

const handleCardClick = (item: WaterfallItem) => {
  useArtworkStore().addArtwork(item.detail)
  navigateTo({
    path: `/artwork/${item.id}`
  })
}

const handleRightClick = () => {
  showViewer.value = true
}

const handlePreview = () => {
  showViewer.value = true
}

// 收藏
const favoritesStore = useFavoritesStore()
const isFavorited = computed(() => favoritesStore.isFavorite(props.item.id))

const toggleFavorite = () => {
  const added = favoritesStore.toggle(props.item.detail)
  Snackbar({
    content: added ? '已收藏 ♥' : '已取消收藏',
    position: 'bottom',
    type: added ? 'success' : 'warning'
  })
}

// 下载第一张原图
const handleDownload = async () => {
  const pic = firstPic.value
  if (!pic) return
  try {
    const resp = await $acgapi<Blob>(`/picture/file/${pic.id}`)
    if (resp) {
      saveAs(resp, pic.file_name)
      Snackbar({ content: '下载成功', position: 'bottom', type: 'success' })
    }
  } catch (e: any) {
    console.error(e)
    Snackbar({ content: '下载失败', position: 'bottom', type: 'error' })
  }
}
</script>

<style scoped>
.card {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: var(--card-radius, 14px);
  cursor: pointer;
  box-shadow: var(--card-shadow, none);
  transition: box-shadow 0.3s ease, transform 0.3s ease;
  background: var(--color-surface-container);
}

.card:hover {
  box-shadow: var(--card-shadow-hover, none);
  transform: translateY(-3px);
}

.card-content {
  width: 100%;
  height: 100%;
  display: block;
}

.cover {
  width: 100%;
  height: auto;
  position: relative;
}

.img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.4s ease;
}

.card-actions {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 2;
  display: flex;
  gap: 6px;
  opacity: 0;
  transform: translateY(-6px);
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.card:hover .card-actions,
.card-actions.is-visible {
  opacity: 1;
  transform: translateY(0);
}

/* 小屏常显操作层 (无 hover) */
@media (hover: none) {
  .card-actions {
    opacity: 1;
    transform: none;
  }
}

.card:hover .img {
  transform: scale(1.08);
}

.card:hover .overlay {
  transform: translateY(0);
}

.overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.75));
  padding: 10px;
  transform: translateY(100%);
  transition: transform 0.3s ease;
}

.card-body {
  color: white;

  h3 {
    margin: 0 0 3px;
    font-size: 14px;
    font-weight: bold;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .author {
    font-size: 12px;
    opacity: 0.85;
  }
}

@media (max-width: 768px) {
  .card {
    border-radius: 10px;
  }

  .card:hover .img {
    transform: none;
  }
}

.v-enter-active,
.v-leave-active {
  transition: opacity 0.4s linear;
}

.v-enter-from,
.v-leave-to {
  opacity: 0;
}
</style>
