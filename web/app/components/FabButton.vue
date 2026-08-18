<template>
  <ClientOnly>
    <var-fab drag position="right-bottom" type="primary" title="操作菜单">
      <var-button @click="toggleAutoScroll" title="自动滚动" type="primary">
        <Icon name="i-line-md:chevron-double-down" v-if="!isAutoScrolling" />
        <Icon name="i-line-md:pause" v-else />
      </var-button>
      <var-button @click="startSlideShow" title="播放幻灯片" type="primary">
        <Icon name="i-line-md:play" />
      </var-button>
    </var-fab>

    <var-overlay v-model:show="showingSlide">
      <div class="slide-show-content">
        <img v-show="showingSlide" :src="currentSlideImage" height="100%" />
      </div>
    </var-overlay>
  </ClientOnly>
</template>

<script setup lang="ts">
import type { ArtworkListResponse } from '~/types/artwork'

const isAutoScrolling = ref(false)
let scrollInterval: NodeJS.Timeout
const scrollStep = 2
const scrollSpeed = 20
const router = useRouter()

const startAutoScroll = () => {
  window.scrollBy({ top: scrollStep, behavior: 'smooth' })
  scrollInterval = setInterval(() => {
    window.scrollBy({ top: scrollStep, behavior: 'smooth' })
  }, scrollSpeed)
}

const stopAutoScroll = () => {
  clearInterval(scrollInterval)
}

const toggleAutoScroll = () => {
  isAutoScrolling.value = !isAutoScrolling.value
  if (isAutoScrolling.value) {
    startAutoScroll()
  } else {
    stopAutoScroll()
  }
}

const currentSlideImage = ref()
const showingSlide = ref(false)
let slideTimer: NodeJS.Timeout
const startSlideShow = async () => {
  if (isAutoScrolling.value) {
    stopAutoScroll()
    isAutoScrolling.value = false
  }
  showingSlide.value = true
  const currentRoute = router.currentRoute.value
  const currentPage = currentRoute.name
  const apiEndpoint = currentPage?.toString() === 'random' ? '/artwork/random' : '/artwork/list'
  const reqParams = reactive({
    page: 1,
    page_size: 30,
    r18: usePiniaStore().r18 ? 2 : 0,
    limit: 30,
    simple: true,
    tag: currentRoute.path.startsWith('/tag/') ? currentRoute.path.replace('/tag/', '') : '',
    artist_id: currentRoute.path.startsWith('/artist/')
      ? currentRoute.path.replace('/artist/', '')
      : '',
    keyword: currentRoute.query.q?.toString() || '',
    hybrid: currentRoute.query.hybrid || false,
    similar_target: currentRoute.path.startsWith('/artwork/')
      ? currentRoute.path.replace('/artwork/', '')
      : currentRoute.query.similar_target
  })
  const cachedImages: string[] = []
  let page = 0
  const updateImages = async () => {
    page++
    reqParams.page = page
    const resp = await $acgapi<ArtworkListResponse>(apiEndpoint, {
      method: 'GET',
      query: reqParams
    })
    cachedImages.length = 0
    resp.data.forEach((item) => {
      if (item.pictures && item.pictures.length > 0) {
        cachedImages.push(item.pictures[0]!.regular)
      }
    })
  }
  try {
    await updateImages()
  } catch (error) {
    console.error('获取图片列表失败:', error)
    return
  }
  if (cachedImages.length === 0) {
    console.warn('没有可用的图片')
    return
  }
  let currentIndex = 0
  const showNextImage = async () => {
    if (currentIndex >= cachedImages.length) {
      currentIndex = 0
      try {
        await updateImages()
      } catch (error) {
        console.error('获取更多图片失败:', error)
      }
    }
    currentSlideImage.value = cachedImages[currentIndex]
    currentIndex++
  }
  showNextImage()
  slideTimer = setInterval(() => {
    showNextImage()
  }, 3000)
}

watch(showingSlide, (newValue) => {
  if (!newValue && slideTimer) {
    clearInterval(slideTimer)
    currentSlideImage.value = null
  }
})

const handleScroll = () => {
  if (isAutoScrolling.value) {
    stopAutoScroll()
    isAutoScrolling.value = false
  }
}

onActivated(() => {
  window.addEventListener('scroll', handleScroll)
})

onDeactivated(() => {
  stopAutoScroll()
  window.removeEventListener('scroll', handleScroll)
})

router.beforeEach(() => {
  stopAutoScroll()
  isAutoScrolling.value = false
  showingSlide.value = false
  currentSlideImage.value = null
})
</script>

<style scoped>
.slide-show-content {
  height: 95vh;
}

@media (max-width: 800px) {
  .slide-show-content {
    height: 80vh;
  }
}
</style>
