/**
 * Telegram Bot API Cloudflare Worker 反向代理
 *
 * 部署：https://dash.cloudflare.com → Workers → Create Worker → 粘贴本文件 → Deploy
 * 部署后得到 https://<your-name>.workers.dev
 *
 * LotsACG 配置：
 *   [telegram]
 *   api_url = "https://<your-name>.workers.dev"
 *
 * 参考项目：https://github.com/4n0nymou3/Telegram-API-Proxy
 */

const TELEGRAM_API = 'https://api.telegram.org'

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url)

    // 只代理 /bot<token>/... 路径
    if (!url.pathname.startsWith('/bot')) {
      return new Response('Telegram Bot API proxy: use /bot<token>/<method>', { status: 404 })
    }

    // 构造目标 URL
    const target = TELEGRAM_API + url.pathname + url.search

    // 透传请求体（Telegram API 用 POST multipart/json）
    const headers = new Headers(request.headers)
    headers.delete('host')
    headers.set('host', new URL(TELEGRAM_API).host)

    const init = {
      method: request.method,
      headers,
      redirect: 'follow',
    }

    // 有 body 时透传（如 sendPhoto 上传）
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      init.body = request.body
      // 若没有 Content-Type，Telegram 默认按表单处理，这里不强制
    }

    const resp = await fetch(target, init)

    // 原样返回 Telegram 响应（状态码 + body + content-type）
    const responseHeaders = new Headers(resp.headers)
    return new Response(resp.body, {
      status: resp.status,
      headers: responseHeaders,
    })
  },
}
