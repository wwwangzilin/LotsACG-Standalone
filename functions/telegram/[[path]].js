/**
 * Telegram Bot API 反向代理 — Cloudflare Pages Functions 版
 *
 * 部署方式（Pages + GitHub 集成）：
 * 1. Cloudflare Pages → Create project → Connect to Git → 选 LotsACG-Standalone
 * 2. Build command 留空，Output directory 留空（纯 Functions，无静态站）
 * 3. 本文件位于 functions/telegram/[[path]].js，会自动成为路由 /telegram/*
 *
 * LotsACG 配置：
 *   [telegram]
 *   api_url = "https://lotsacg.pages.dev/telegram"
 *
 * 说明：Pages Functions 每个文件对应一个路由，[[path]].js 捕获所有子路径。
 * 访问 https://lotsacg.pages.dev/telegram/bot<token>/<method> 会被转发到
 * https://api.telegram.org/bot<token>/<method>
 */

const TELEGRAM_API = 'https://api.telegram.org'

/**
 * 处理 /telegram/* 的代理请求
 * @param {Request} context.request
 */
export async function onRequest(context) {
  const { request } = context
  const url = new URL(request.url)

  // url.pathname 形如 /telegram/bot123:xxx/getMe
  // 去掉 /telegram 前缀后即 /bot<token>/<method>
  const tgPath = url.pathname.replace(/^\/telegram/, '')

  // 只代理 /bot<token>/... 路径
  if (!tgPath.startsWith('/bot')) {
    return new Response('Telegram Bot API proxy: use /telegram/bot<token>/<method>', { status: 404 })
  }

  // 构造目标 URL
  const target = TELEGRAM_API + tgPath + url.search

  // 透传请求（Telegram API 用 POST multipart/json）
  const headers = new Headers(request.headers)
  headers.delete('host')
  headers.set('host', new URL(TELEGRAM_API).host)

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
}
