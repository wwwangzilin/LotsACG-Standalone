import logging
import sys
from datetime import timedelta

from loguru import logger

from kmua.config import app_config


class InterceptHandler(logging.Handler):
    def emit(self, record):
        try:
            level = logger.level(record.levelname).name
        except ValueError:
            level = record.levelno
        logger.opt(depth=6, exception=record.exc_info).log(level, record.getMessage())


logger.remove()

# 日志统一由 LotsACG 管理器重定向 stdout 到控制台/日志文件 (带 [kmua] 前缀),
# 不再单独写文件, 避免产生多个日志文件。
logger.add(sys.stdout, level=app_config.log_level)

logging.basicConfig(
    handlers=[InterceptHandler()],
    level=logging.NOTSET,
    force=True,
)

__all__ = ["logger"]
