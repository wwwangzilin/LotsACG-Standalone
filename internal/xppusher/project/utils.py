"""
工具函数模块
- 日志配置
- 异步限流器
- 重试装饰器
- 图片URL处理
- 标签自动扩展
"""
import asyncio
import logging
import random
import time
from functools import wraps
from logging.handlers import RotatingFileHandler
from logging.handlers import RotatingFileHandler
from pathlib import Path
import io
import zipfile
import tempfile
import os

import aiohttp

TAG_TRANSLATIONS = {
    # Visual Traits
    "white_hair": "白髪 OR 銀髪 OR white_hair",
    "silver_hair": "銀髪 OR 白髪",
    "grey_hair": "灰髪",
    "black_hair": "黒髪",
    "blonde_hair": "金髪",
    "red_hair": "赤髪",
    "blue_hair": "青髪",
    "pink_hair": "ピンク髪",
    "green_hair": "緑髪",
    "purple_hair": "紫髪",
    "brown_hair": "茶髪",
    "long_hair": "ロングヘア OR 長髪",
    "short_hair": "ショートヘア OR 短髪",
    "twintails": "ツインテール",
    "ponytail": "ポニーテール",
    
    # Body & Cloth
    "large_breasts": "巨乳",
    "flat_chest": "貧乳",
    "maid": "メイド",
    "swimsuit": "水着",
    "school_uniform": "セーラー服 OR 制服 OR ブレザー",
    "pantyhose": "パンスト OR ストッキング",
    "thighhighs": "ニーソ OR ニーソックス",
    "glasses": "眼鏡 OR メガネ",
    "kimono": "着物 OR 浴衣",
    "bunny_suit": "バニー OR バニーガール",
    "cat_ears": "猫耳 OR ネコミミ",
    
    # Popular IP
    "genshin_impact": "原神 OR GenshinImpact",
    "原神": "原神 OR GenshinImpact",
    
    "blue_archive": "ブルーアーカイブ OR BlueArchive OR 碧蓝档案",
    "ブルーアーカイブ": "ブルーアーカイブ OR BlueArchive OR 碧蓝档案",
    "ブルアカ": "ブルーアーカイブ OR BlueArchive OR 碧蓝档案",
    
    "arknights": "アークナイツ OR Arknights OR 明日方舟",
    "アークナイツ": "アークナイツ OR Arknights OR 明日方舟",
    "明日方舟": "アークナイツ OR Arknights OR 明日方舟",
    
    "fate_grand_order": "FGO OR Fate/GrandOrder",
    "azur_lane": "アズールレーン",
    "hololive": "ホロライブ",
    
    # Elements
    "scenery": "風景",
    "cyberpunk": "サイバーパンク",
    "steampunk": "スチームパンク",
    "fantasy": "ファンタジー",
}

def expand_search_query(tag: str) -> str:
    """
    将标准化的英文标签扩展为适合 Pixiv 搜索的查询字符串 (包含日文别名)
    并且自动加上括号以防逻辑错误: (A OR B) AND C
    """
    expanded = TAG_TRANSLATIONS.get(tag, tag)
    if " OR " in expanded:
        return f"({expanded})"
    return expanded


def setup_logging(log_dir: Path = Path("logs")):
    """配置日志（分级、控制台输出）"""
    log_dir.mkdir(exist_ok=True)
    
    # 强制 stdout/stderr 使用 UTF-8, 避免重定向到文件时按 GBK 编码导致乱码
    import sys
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            try:
                stream.reconfigure(encoding="utf-8", errors="replace")
            except Exception:
                pass
    
    formatter = logging.Formatter(
        "[%(asctime)s] %(levelname)s %(name)s: %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S"
    )
    
    # 日志统一由 LotsACG 管理器重定向 stdout/stderr 到控制台/日志文件 (带 [xppusher] 前缀),
    # 不再单独写文件, 避免产生多个日志文件。
    
    # 控制台Handler
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(formatter)
    console_handler.setLevel(logging.INFO)
    
    # 根Logger
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.INFO)
    root_logger.addHandler(console_handler)
    
    # 屏蔽第三方库的废话
    logging.getLogger("aiosqlite").setLevel(logging.WARNING)
    logging.getLogger("urllib3").setLevel(logging.WARNING)
    logging.getLogger("asyncio").setLevel(logging.WARNING)
    logging.getLogger("httpx").setLevel(logging.WARNING)
    logging.getLogger("httpcore").setLevel(logging.WARNING)
    logging.getLogger("pixivpy_async").setLevel(logging.ERROR)  # 屏蔽重试警告
    
    return root_logger


class AsyncRateLimiter:
    """
    令牌桶限流器
    """
    
    def __init__(self, requests_per_minute: int = 60, random_delay: tuple[float, float] = (1.0, 3.0)):
        self.rate = requests_per_minute / 60.0  # 每秒令牌数
        self.max_tokens = requests_per_minute
        self.tokens = self.max_tokens
        self.last_update = time.monotonic()
        self.lock = asyncio.Lock()
        self.random_delay = random_delay
    
    async def acquire(self):
        """获取令牌"""
        async with self.lock:
            now = time.monotonic()
            # 补充令牌
            elapsed = now - self.last_update
            self.tokens = min(self.max_tokens, self.tokens + elapsed * self.rate)
            self.last_update = now
            
            if self.tokens < 1:
                # 等待令牌恢复
                wait_time = (1 - self.tokens) / self.rate
                await asyncio.sleep(wait_time)
                self.tokens = 0
            else:
                self.tokens -= 1
            
            # 随机抖动
            if self.random_delay:
                delay = random.uniform(*self.random_delay)
                await asyncio.sleep(delay)
    
    async def __aenter__(self):
        await self.acquire()
        return self
    
    async def __aexit__(self, *args):
        pass


def retry_async(max_retries: int = 3, delay: float = 1.0, backoff: float = 2.0):
    """
    异步重试装饰器
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            last_exception = None
            current_delay = delay
            
            for attempt in range(max_retries + 1):
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    last_exception = e
                    if attempt < max_retries:
                        logging.warning(
                            f"{func.__name__} 失败 (尝试 {attempt + 1}/{max_retries + 1}): {e}"
                        )
                        await asyncio.sleep(current_delay)
                        current_delay *= backoff
            
            raise last_exception
        return wrapper
    return decorator


def get_pixiv_cat_url(illust_id: int, page: int = 0) -> str:
    """
    获取 pixiv.cat 反代图片URL
    """
    if page == 0:
        return f"https://pixiv.cat/{illust_id}.jpg"
    else:
        return f"https://pixiv.cat/{illust_id}-{page + 1}.jpg"


async def download_image_with_referer(
    session: aiohttp.ClientSession,
    url: str,
    semaphore: asyncio.Semaphore | None = None,
    proxy: str | None = None
) -> bytes:
    """
    带Referer下载Pixiv图片
    """
    headers = {
        "Referer": "https://www.pixiv.net/",
        "User-Agent": "PixivIOSApp/7.13.3 (iOS 14.6; iPhone13,2)"
    }
    
    async def _download():
        async with session.get(url, headers=headers, proxy=proxy) as resp:
            resp.raise_for_status()
            return await resp.read()
    
    if semaphore:
        async with semaphore:
            return await _download()
    return await _download()


# --- Tag Normalization Utilities ---

import re

# 常见别名映射表 (与 profiler.py 保持一致)
TAG_ALIASES = {
    "白髪": "white_hair",
    "silver hair": "white_hair",
    "白髮": "white_hair",
    "猫耳": "cat_ears",
    "cat ears": "cat_ears",
    "nekomimi": "cat_ears",
    "ロリ": "loli",
    "巨乳": "large_breasts",
    "おっぱい": "breasts",
    "黒髪": "black_hair",
    "金髪": "blonde_hair",
    "ツインテール": "twintails",
    "twin tails": "twintails",
    "maid": "maid",
    "メイド": "maid",
    "水着": "swimsuit",
    "制服": "uniform",
    "ストッキング": "stockings",
    "ニーソ": "thighhighs",
    "眼鏡": "glasses",
}

_pattern_users = re.compile(r"^(.*?)\d+users入り$", re.IGNORECASE)

def normalize_tag(tag: str) -> str:
    """
    Tag归一化 (Shared logic)
    1. 去除 xxxusers入り 后缀
    2. 统一转小写
    3. 去除空格
    4. 别名映射
    """
    tag = tag.strip()
    
    # 去除 users入り 后缀
    match = _pattern_users.match(tag)
    if match:
        prefix = match.group(1)
        if prefix:  # 如果前缀非空，使用前缀
            tag = prefix
    
    tag = tag.lower()
    
    # 检查别名映射
    for alias, canonical in TAG_ALIASES.items():
        if tag == alias.lower():
            return canonical
    
    # 替换空格为下划线
    tag = tag.replace(" ", "_")
    
    return tag


def convert_ugoira_to_mp4(zip_data: bytes, frames: list[dict]) -> bytes:
    """将 Ugoira ZIP 转换为 MP4 (依赖 imageio[ffmpeg])"""
    try:
        import imageio
    except ImportError:
        logger.warning("未安装 imageio，无法本地转换动图。请 pip install imageio[ffmpeg]")
        return None

    try:
        # 解压 zip
        with zipfile.ZipFile(io.BytesIO(zip_data)) as zf:
            images = []
            
            # 按 frames 顺序读取
            for frame in frames:
                fname = frame['file']
                with zf.open(fname) as f:
                    # imageio.imread 支持读取 bytes
                    img = imageio.imread(f.read())
                    images.append(img)
            
            # 计算 fps
            total_delay = sum(f['delay'] for f in frames)
            if not total_delay: return None
            
            avg_delay = total_delay / len(frames)
            fps = 1000 / avg_delay
            
            # 写入 MP4 到 BytesIO 需要 imageio 支持 ffmpeg plugin
            # 由于 BytesIO 写入比较复杂（需要 format='mp4' 且 ffmpeg 支持 pipe），
            # 使用临时文件更稳妥
            
            with tempfile.NamedTemporaryFile(suffix=".mp4", delete=False) as tmp_file:
                tmp_path = tmp_file.name
            
            try:
                # fix: libx264 要求宽高必须是偶数，添加 pad filter
                # scale=trunc(iw/2)*2:trunc(ih/2)*2 也可以，但 pad 不会变形
                imageio.mimwrite(
                    tmp_path, 
                    images, 
                    fps=fps, 
                    codec='libx264', 
                    pixelformat='yuv420p', 
                    macro_block_size=None,
                    ffmpeg_params=['-vf', 'pad=ceil(iw/2)*2:ceil(ih/2)*2']
                )
                
                with open(tmp_path, "rb") as f:
                    mp4_bytes = f.read()
            finally:
                if os.path.exists(tmp_path):
                    os.remove(tmp_path)
            
            return mp4_bytes
            
    except Exception as e:
        logger.error(f"动图转换失败: {e}")
        return None


def convert_ugoira_to_gif(zip_data: bytes, frames: list[dict], max_width: int = 720) -> bytes:
    """将 Ugoira ZIP 转换为 GIF (使用 Pillow，不依赖 ffmpeg)"""
    from PIL import Image
    import io
    import zipfile

    try:
        with zipfile.ZipFile(io.BytesIO(zip_data)) as zf:
            images = []
            durations = []
            
            for frame in frames:
                fname = frame['file']
                with zf.open(fname) as f:
                    img = Image.open(f)
                    img.load() # 确保加载到内存
                    
                    # 缩放以控制体积
                    if img.width > max_width:
                        h = int(img.height * (max_width / img.width))
                        img = img.resize((max_width, h), Image.Resampling.LANCZOS)
                    
                    images.append(img)
                    durations.append(frame['delay'])
            
            if not images:
                return None
            
            output = io.BytesIO()
            # 保存为 GIF
            images[0].save(
                output,
                format='GIF',
                save_all=True,
                append_images=images[1:],
                duration=durations,
                loop=0,
                optimize=True
            )
            return output.getvalue()
            
    except Exception as e:
        logging.error(f"GIF 转换失败: {e}")
        return None
