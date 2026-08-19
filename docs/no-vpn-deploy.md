# 免梯子部署：Cloudflare Worker 代理 Telegram Bot API

> 在 Telegram API 被封锁的地区（或需要稳定访问的场景），可以通过自建 Cloudflare Worker 反向代理
> 访问 `api.telegram.org`，LotsACG 只需把 `[telegram] api_url` 指向代理地址即可，**代码无需改动**。

## 原理

LotsACG 的 Telegram Bot 通过 `api_url` 配置 API 基地址（`internal/infra/config/runtimecfg/telegram.go` 的 `APIURL` 字段，`telego.WithAPIServer` 使用）。默认是官方 `https://api.telegram.org`。填入自建 Worker 代理地址后，Bot 的所有请求都走代理转发到官方 API。

```
LotsACG  →  https://your-name.workers.dev  →  api.telegram.org
```

## 方案选型（GitHub 开源项目）

| 项目 | 说明 | 推荐度 |
| --- | --- | --- |
| [4n0nymou3/Telegram-API-Proxy](https://github.com/4n0nymou3/Telegram-API-Proxy) | Cloudflare 方案，受限地区免 VPN 访问 Telegram API，部署简单 | ⭐⭐⭐ 推荐 |
| [ndneighbor/telegram-cloudflare-proxy](https://github.com/ndneighbor/telegram-cloudflare-proxy) | Cloudflare Worker 代理，绕过云服务商 IP 封锁 | ⭐⭐ |
| [Andromeda-Collective/andro-cfw](https://github.com/Andromeda-Collective/andro-cfw) | 受限地区（如伊朗）免 VPN 运行 Telegram bot 的 Worker 反向代理 | ⭐⭐ |

## 部署步骤（以 4n0nymou3/Telegram-API-Proxy 为例）

### 1. 注册 Cloudflare 账号

前往 [dash.cloudflare.com](https://dash.cloudflare.com) 注册（免费版即可，Worker 免费额度足够个人 Bot 使用）。

### 2. 部署 Worker

**方式 A：Wrangler CLI（推荐）**

```sh
# 安装 wrangler
npm install -g wrangler

# 克隆项目
git clone https://github.com/4n0nymou3/Telegram-API-Proxy
cd Telegram-API-Proxy

# 登录并部署
wrangler login
wrangler deploy
```

**方式 B：Cloudflare Dashboard 网页**

1. 打开 [Cloudflare Workers 控制台](https://dash.cloudflare.com/?to=/:account/workers)
2. 点 **Create Worker** → **Deploy with code**
3. 把 `Telegram-API-Proxy` 项目的 `worker.js` 内容粘贴进去
4. 保存并部署

### 3. 获取代理地址

部署完成后得到一个地址：

```
https://<你的子域名>.workers.dev
```

> 也可以绑定自定义域名（Workers 设置 → Triggers → Custom Domain），如 `https://tg.example.com`

### 4. 验证代理可用

```sh
curl https://<你的子域名>.workers.dev/getMe
```

正常返回 `{"ok":true,"result":{...}}` 即部署成功。

### 5. 配置 LotsACG

在 `config.toml` 中填入代理地址：

```toml
[telegram]
bot_token = "123456:ABCDEF"
# 免梯子：自建 Cloudflare Worker 代理地址（不填则用官方 api.telegram.org）
api_url = "https://<你的子域名>.workers.dev"
```

重启 LotsACG 即可生效，Bot 的 API 请求全部走代理，无需梯子。

## 注意事项

- **代理稳定性**：Worker 免费额度（每天 10 万请求）对个人 Bot 足够；若 Bot 高频运行可绑定自定义域名或升级付费版
- **数据安全**：请求会经过你的 Worker 转发，建议只在自己的 Worker 上运行，不要使用第三方公开代理
- **与 `[source] proxy` 的区别**：`[telegram] api_url` 只影响 Telegram API 访问；抓取图源（pixiv 等）的代理由 `[source] proxy` 控制，两者独立
- **国内访问 Cloudflare**：`*.workers.dev` 在国内偶尔不稳定，若遇到问题可绑定自定义域名（CF 域名在国内也需可达性测试）

## 仓库自带模板（推荐）

本仓库提供开箱即用的 Worker 模板：[deploy/telegram-proxy-worker.js](../deploy/telegram-proxy-worker.js)

1. Cloudflare Dashboard → Workers → **Create Worker**
2. 点 **Edit Code**，删除默认内容，粘贴 `telegram-proxy-worker.js` 的全部代码
3. **Save and Deploy**
4. 得到地址 `https://<your-name>.workers.dev`，填入 `[telegram] api_url`

模板特性：仅代理 `/bot<token>/<method>` 路径，完整透传请求体（支持 sendPhoto 等上传），原样返回 Telegram 响应。
