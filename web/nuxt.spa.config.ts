// SPA 构建配置: 继承主配置但关闭 SSR。
// 用于静态托管 (由 LotsACG REST 服务器直接提供 .output/public)。
import base from './nuxt.config'

export default defineNuxtConfig({
  ...base,
  ssr: false,
})
