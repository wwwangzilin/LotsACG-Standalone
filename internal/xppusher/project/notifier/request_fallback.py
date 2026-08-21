"""
多策略降级 Telegram 请求器: 直连 → Cloudflare Worker → 本地代理。

用于免梯子场景: 直连 api.telegram.org 失败时自动切换到 CF Worker 反代,
两者都失败时才使用配置的本地代理 (proxy_url)。

cf_worker_url 支持两种写法:
  - 含 {url} 占位符: "https://worker.example.com/proxy?url={url}" (完整 URL 替换)
  - 不含占位符: "https://worker.example.com" (把 /botxxx/xxx 路径拼接到反代域名后)
"""
import logging
import urllib.parse

from telegram.request import HTTPXRequest
from telegram.request._httpxrequest import RequestData

logger = logging.getLogger(__name__)


class FallbackTelegramRequest(HTTPXRequest):
    """按 直连 → CF Worker → 本地代理 顺序降级的 Telegram API 请求器。"""

    def __init__(self, proxy_url=None, cf_worker_url=None, **kwargs):
        super().__init__(**kwargs)
        self.proxy_url = proxy_url
        self.cf_worker_url = cf_worker_url
        # 降级客户端列表: (标签, 客户端, 是否改写 URL)
        self._fallbacks = []
        if cf_worker_url:
            self._fallbacks.append(("cf_worker", HTTPXRequest(**kwargs), True))
        if proxy_url:
            self._fallbacks.append(("proxy", HTTPXRequest(proxy=proxy_url, **kwargs), False))

    async def do_request(self, request_data: RequestData, url: str):
        # 策略 0: 直连 (自身即默认客户端)
        last_err = None
        try:
            return await super().do_request(request_data, url)
        except Exception as e:
            last_err = e
            logger.warning("Telegram API 直连失败, 尝试降级: %s", e)
        # 降级策略
        for label, client, rewrite in self._fallbacks:
            try:
                target = self._rewrite_url(url) if rewrite else url
                return await client.do_request(request_data, target)
            except Exception as e:
                last_err = e
                logger.warning("Telegram API 经 %s 失败: %s", label, e)
        if last_err is not None:
            raise last_err
        raise RuntimeError("no Telegram request strategy available")

    def _rewrite_url(self, url: str) -> str:
        """把 api.telegram.org 的完整 URL 改写为 CF Worker 反代地址。"""
        base = (self.cf_worker_url or "").strip()
        if not base:
            return url
        if "{url}" in base:
            return base.replace("{url}", url)
        parsed = urllib.parse.urlsplit(url)
        path = parsed.path or "/"
        if parsed.query:
            return base.rstrip("/") + path + "?" + parsed.query
        return base.rstrip("/") + path
