/**
 * Telegram Bot API 反向代理 — Cloudflare Pages 单文件模式（_worker.js）
 *
 * 使用方法：
 * 1. 本文件重命名为 _worker.js（必须这个名字）
 * 2. 上传到 Cloudflare Pages 项目的根目录（和之前的 telegram-proxy-worker.js 同级位置，但覆盖式替换）
 * 3. Pages 会自动把它作为整个 Worker 运行（不需要 functions/ 目录）
 *
 * 代理路径：任何 /bot<token>/<method> 请求都会被转发到 api.telegram.org
 * LotsACG 配置：
 *   [telegram]
 *   api_url = "https://11lotsacg.pages.dev"
 */

const TELEGRAM_API = 'https://api.telegram.org'

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url)

    // 只代理 /bot<token>/... 路径（api_url 直接填 https://11lotsacg.pages.dev）
    if (!url.pathname.startsWith('/bot')) {
      return new Response('Telegram Bot API proxy: use /bot<token>/<method>', { status: 404 })
    }

    // 构造目标 URL
    const target = TELEGRAM_API + url.pathname + url.search

    // 透传请求（Telegram API 用 POST multipart/json）
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
    }

    const resp = await fetch(target, init)
    return new Response(resp.body, {
      status: resp.status,
      headers: resp.headers,
    })
  },
}
