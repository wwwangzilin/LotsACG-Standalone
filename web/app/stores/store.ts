import { defineStore } from 'pinia'
import type { Artwork } from '~/types/artwork'

export interface WaterfallSettings {
  itemMinWidth: number
  minColumnCount: number
  maxColumnCount: number
  preloadScreenCount: number
  bottomDistance: number
  onlyImage: boolean
}

const defaultWaterfallSettings = (): WaterfallSettings => ({
  itemMinWidth: 320,
  minColumnCount: 2,
  maxColumnCount: 8,
  preloadScreenCount: 1,
  bottomDistance: 1000,
  onlyImage: false
})

export const usePiniaStore = defineStore('ManyACG', {
  state: () => ({
    preferLight: false,
    r18: false,
    waterfall: defaultWaterfallSettings() as WaterfallSettings
  }),
  actions: {
    setpreferLight(preferLight: boolean) {
      this.preferLight = preferLight
    },
    setR18(r18: boolean) {
      this.r18 = r18
    },
    setWaterfall(partial: Partial<WaterfallSettings>) {
      this.waterfall = { ...this.waterfall, ...partial }
    },
    resetWaterfall() {
      this.waterfall = defaultWaterfallSettings()
    }
  },
  persist: true
})

export const useArtworkStore = defineStore('Artwork', {
  state: () => ({
    artworks: {} as Record<string, Artwork>,
    maxCacheSize: 50
  }),
  actions: {
    addArtwork(artwork: Artwork) {
      const id = artwork.id
      this.artworks[id] = artwork
    },
    getArtwork(id: string) {
      const artwork = this.artworks[id]
      if (Object.keys(this.artworks).length > this.maxCacheSize) {
        this.clearArtworks()
      }
      return artwork || null
    },
    clearArtworks() {
      this.artworks = {}
    },
    removeArtwork(id: string) {
      if (!this.artworks[id]) return
      delete this.artworks[id]
    }
  }
})

// 收藏夹 store：localStorage 持久化收藏的作品 ID 与元数据
export const useFavoritesStore = defineStore('Favorites', {
  state: () => ({
    // id -> 收藏时的元数据快照（用于收藏页展示，避免依赖网络）
    favorites: {} as Record<string, {
      id: string
      title: string
      thumbnail: string
      artist_name: string
      source_url: string
      source_type: string
      r18: boolean
      added_at: number
    }>
  }),
  getters: {
    count: (state) => Object.keys(state.favorites).length,
    list: (state) => Object.values(state.favorites).sort((a, b) => b.added_at - a.added_at),
    isFavorite: (state) => (id: string) => !!state.favorites[id]
  },
  actions: {
    toggle(artwork: Artwork) {
      if (this.favorites[artwork.id]) {
        delete this.favorites[artwork.id]
        return false
      }
      this.favorites[artwork.id] = {
        id: artwork.id,
        title: artwork.title || '',
        thumbnail: artwork.pictures?.[0]?.thumbnail || '',
        artist_name: artwork.artist?.name || '',
        source_url: artwork.source_url || '',
        source_type: artwork.source_type || '',
        r18: !!artwork.r18,
        added_at: Date.now()
      }
      return true
    },
    remove(id: string) {
      if (this.favorites[id]) delete this.favorites[id]
    },
    clear() {
      this.favorites = {}
    }
  },
  persist: true
})
