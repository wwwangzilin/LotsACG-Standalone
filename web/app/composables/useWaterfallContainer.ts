import { ref, onMounted, onUnmounted } from 'vue'

// headerHeight 页面顶部的高度偏移量，默认为 64px
const useWaterfallContainer = (headerHeight: number = 64) => {
  const containerRef = ref<HTMLElement | null>(null)

  const updateContainerHeight = () => {
    if (containerRef.value && typeof window !== 'undefined') {
      containerRef.value.style.height = `${window.innerHeight - headerHeight}px`
    }
  }

  const handleResize = () => {
    updateContainerHeight()
  }

  onMounted(() => {
    updateContainerHeight()
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
  })
  return {
    containerRef
  }
}

export default useWaterfallContainer
