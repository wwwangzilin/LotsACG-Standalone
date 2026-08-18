
import yaml
from pathlib import Path
import logging

logger = logging.getLogger(__name__)

CONFIG_PATH = Path("config.yaml")

def load_config(path: Path = CONFIG_PATH) -> dict:
    """加载配置文件"""
    if not path.exists():
        # Fallback to example if exists? No, just log error
        logger.error(f"配置文件未找到: {path}")
        return {}
    
    try:
        with open(path, "r", encoding="utf-8") as f:
            return yaml.safe_load(f) or {}
    except Exception as e:
        logger.error(f"加载配置文件失败: {e}")
        return {}
