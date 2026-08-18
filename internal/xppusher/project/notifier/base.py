"""
推送器抽象基类
"""
from abc import ABC, abstractmethod
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from pixiv_client import Illust


class BaseNotifier(ABC):
    """推送器抽象基类"""
    
    @abstractmethod
    async def send(self, illusts: list["Illust"]) -> list[int]:
        """
        发送推送
        
        Args:
            illusts: 作品列表
            
        Returns:
            是否成功
        """
        pass
    
    @abstractmethod
    def format_message(self, illust: "Illust") -> str:
        """
        格式化单条消息
        
        Args:
            illust: 作品对象
            
        Returns:
            格式化后的消息文本
        """
        pass
    
    @abstractmethod
    def handle_feedback(self, illust_id: int, action: str) -> bool:
        """
        处理用户反馈
        
        Args:
            illust_id: 作品ID
            action: 'like' | 'dislike'
            
        Returns:
            是否处理成功
        """
        pass

    async def send_text(self, text: str, buttons: list[tuple[str, str]] | None = None) -> bool:
        """
        发送纯文本消息（可选带按钮）
        
        Args:
            text: 消息文本
            buttons: 按钮列表 [(标签, callback_data), ...]
        """
        # 默认实现不发送或仅打印
        return True
