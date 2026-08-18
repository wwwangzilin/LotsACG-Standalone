import { reactive, watchEffect, onActivated, onDeactivated } from 'vue'
import { Snackbar } from '@varlet/ui'
import { useDebounceFn } from '@vueuse/core'
import type { ArtworkListResponse, WaterfallItem } from '~/types/artwork'

const useWaterfall = ({
  artistId,
  tag,
  mode = 'index',
  keyword,
  hybrid,
  similarTarget
}: {
  artistId?: string
  tag?: string
  mode?: 'index' | 'random'
  keyword?: string
  hybrid?: boolean
  similarTarget?: string
}) => {
  const piniaStore = usePiniaStore()
  const waterfallOption = reactive({
    loading: false,
    bottomDistance: piniaStore.waterfall.bottomDistance,
    onlyImage: piniaStore.waterfall.onlyImage,
    preloadScreenCount: [
      piniaStore.waterfall.preloadScreenCount,
      piniaStore.waterfall.preloadScreenCount
    ] as [number, number],
    virtual: true,
    gap: useWaterfallGap(),
    enableCache: true,
    itemMinWidth: piniaStore.waterfall.itemMinWidth,
    minColumnCount: piniaStore.waterfall.minColumnCount,
    maxColumnCount: piniaStore.waterfall.maxColumnCount
  })

  const calcItemHeight = (item: WaterfallItem, itemWidth: number) => {
    const picture = item.detail.pictures?.[0]
    if (!picture) {
      return 0
    }
    return picture.height * (itemWidth / picture.width)
  }

  const fetchParams = reactive({
    page: 1,
    page_size: 20,
    artist_id: artistId,
    tag: tag,
    r18: usePiniaStore().r18 ? 2 : 0,
    limit: 20,
    // simple: true,
    keyword: keyword,
    hybrid: hybrid,
    similar_target: similarTarget
  })

  const result = reactive({
    list: [] as WaterfallItem[],
    end: false,
    errorMessage: '',
    statusCode: 200
  })

  const apiEndpoint = mode === 'random' ? '/artwork/random' : '/artwork/list'

  const { data, status, error } = useAcgapiData<ArtworkListResponse>(apiEndpoint, {
    method: 'GET',
    query: fetchParams,
    // 仅在客户端请求: 避免静态预渲染 (nuxt generate) 时相对路径请求失败, 把错误状态写入 payload 导致页面崩
    server: false,
    onResponse({ response }) {
      result.statusCode = response.status
    }
  })

  watchEffect(() => {
    if (error.value) {
      if (error.value.statusCode === 404 && fetchParams.page !== 1) {
        result.end = true
      } else {
        result.errorMessage = error.value.message
        if (error.value.statusCode === 401) {
          Snackbar.error({
            content: '请登录哦'
          })
        }
      }
    }
  })

  watchEffect(() => {
    if (data.value && data.value.data) {
      for (const artwork of data.value.data) {
        const item: WaterfallItem = {
          id: artwork.id,
          detail: artwork
        }
        result.list.push(item)
      }
      result.end = false
    } else {
      result.end = true
    }
  })

  const checkScrollPosition = () => {
    if (waterfallOption.loading || status.value === 'pending' || result.end) {
      return
    }

    const scrollHeight = document.documentElement.scrollHeight
    const scrollTop = document.documentElement.scrollTop
    const clientHeight = window.innerHeight

    const distanceFromBottom = scrollHeight - scrollTop - clientHeight
    if (distanceFromBottom <= waterfallOption.bottomDistance) {
      waterfallOption.loading = true
      fetchParams.page += 1
      waterfallOption.loading = false
    }
  }

  const scrollHandler = useDebounceFn(checkScrollPosition, 125)

  onActivated(() => {
    window.addEventListener('scroll', scrollHandler)
  })

  onDeactivated(() => {
    window.removeEventListener('scroll', scrollHandler)
  })

  return {
    waterfallOption,
    result,
    calcItemHeight
  }
}

export default useWaterfall
