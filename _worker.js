/**
 * 通用 API 反向代理 — Cloudflare Pages 单文件模式（_worker.js）
 *
 * 支持任意目标域名转发，当前内置：
 *   /bot*            → Telegram Bot API (api.telegram.org)
 *   /px/*            → pixiv 网页 ajax (www.pixiv.net) —— 用于画师主页/作品页抓取
 *
 * 使用方法：重命名为 _worker.js 上传到 Cloudflare Pages 根目录。
 *
 * LotsACG 配置：
 *   [telegram]
 *   api_url = "https://11lotsacg.pages.dev"
 *
 *   [source.pixiv] 相关 API 若需走本代理，由 LotsACG 请求前缀控制（见 docs）
 */

const TELEGRAM_API = 'https://api.telegram.org'
const PIXIV_WEB = 'https://www.pixiv.net'

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url)
    let target

    // Telegram: /bot<token>/<method>
    if (url.pathname.startsWith('/bot')) {
      target = TELEGRAM_API + url.pathname + url.search
    }
    // Pixiv 网页 ajax: /px/ajax/... -> https://www.pixiv.net/ajax/...
    else if (url.pathname.startsWith('/px/')) {
      const pxPath = url.pathname.slice('/px'.length) // 去掉 /px
      target = PIXIV_WEB + pxPath + url.search
    }
    // 其他
    else {
      return new Response('Proxy: use /bot<token>/<method> or /px/ajax/...', { status: 404 })
    }

    const headers = new Headers(request.headers)
    headers.delete('host')
    headers.set('host', new URL(target).host)

    // pixiv 需要 Referer，强制设置
    if (target.startsWith(PIXIV_WEB)) {
      headers.set('referer', 'https://www.pixiv.net/')
      headers.set('accept', 'application/json')
      headers.set('accept-language', 'zh-CN,zh;q=0.9')
    }

    const init = {
      method: request.method,
      headers,
      redirect: 'follow',
    }
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      init.body = request.body
    }

    const resp = await fetch(target, init)
    return new Response(resp.body, {
      status: resp.status,
      headers: resp.headers,
    })
  },
}
