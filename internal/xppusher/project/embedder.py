"""
AI Embedding 模块

使用第三方 API 计算文本向量，支持语义匹配。
借鉴 X 算法 Two-Tower Model 的理念。

支持的 Provider:
- openai: OpenAI / DeepSeek / 兼容 API
- local: 本地 sentence-transformers 模型 (需要额外安装)
"""
import logging
import asyncio
import json
import numpy as np
from typing import Optional
from dataclasses import dataclass

logger = logging.getLogger(__name__)

# 尝试导入 OpenAI 客户端
try:
    from openai import AsyncOpenAI
    HAS_OPENAI = True
except ImportError:
    HAS_OPENAI = False
    logger.warning("openai 未安装，Embedding 功能将不可用")

# 尝试导入本地模型
try:
    from sentence_transformers import SentenceTransformer
    HAS_LOCAL_MODEL = True
except ImportError:
    HAS_LOCAL_MODEL = False


@dataclass
class EmbeddingConfig:
    """Embedding 配置"""
    enabled: bool = False
    provider: str = "openai"  # openai | local
    model: str = "text-embedding-3-small"  # OpenAI 模型或本地模型名
    api_key: str = ""
    base_url: str = ""
    dimensions: int = 256  # 向量维度 (OpenAI 支持 256/512/1024/1536)
    cache_ttl_days: int = 30  # 缓存天数
    semantic_weight: float = 0.3  # 语义匹配在最终分数中的权重


class Embedder:
    """
    Embedding 计算器
    
    支持:
    1. 用户画像 Embedding (基于 Top XP Tags)
    2. 作品 Embedding (基于作品 Tags)
    3. 相似度计算 (余弦相似度)
    """
    
    def __init__(self, config: dict):
        """
        初始化 Embedder
        
        Args:
            config: ai.embedding 配置块
        """
        self.enabled = config.get("enabled", False)
        self.provider = config.get("provider", "openai")
        self.model = config.get("model", "text-embedding-3-small")
        self.dimensions = config.get("dimensions", 256)
        self.semantic_weight = config.get("semantic_weight", 0.3)
        self.cache_ttl_days = config.get("cache_ttl_days", 30)
        
        self._client = None
        self._local_model = None
        
        if not self.enabled:
            return
        
        if self.provider == "openai":
            if not HAS_OPENAI:
                logger.error("openai 库未安装，无法使用 OpenAI Embedding")
                self.enabled = False
                return
            
            api_key = config.get("api_key", "")
            base_url = config.get("base_url", "")
            
            if not api_key:
                logger.warning("未配置 Embedding API Key，功能已禁用")
                self.enabled = False
                return
            
            self._client = AsyncOpenAI(
                api_key=api_key,
                base_url=base_url or None
            )
            logger.info(f"Embedder 已初始化 (provider={self.provider}, model={self.model})")
            
        elif self.provider == "local":
            if not HAS_LOCAL_MODEL:
                logger.error("sentence-transformers 未安装，无法使用本地模型")
                self.enabled = False
                return
            
            try:
                self._local_model = SentenceTransformer(self.model)
                logger.info(f"本地 Embedding 模型已加载: {self.model}")
            except Exception as e:
                logger.error(f"加载本地模型失败: {e}")
                self.enabled = False
        else:
            logger.error(f"不支持的 Embedding Provider: {self.provider}")
            self.enabled = False
    
    async def embed_text(self, text: str) -> Optional[list[float]]:
        """
        计算单个文本的 Embedding
        
        Args:
            text: 输入文本
            
        Returns:
            向量列表，失败返回 None
        """
        if not self.enabled or not text:
            return None
        
        try:
            if self.provider == "openai":
                response = await self._client.embeddings.create(
                    model=self.model,
                    input=text,
                    dimensions=self.dimensions
                )
                return response.data[0].embedding
                
            elif self.provider == "local":
                # sentence-transformers 是同步的，用 run_in_executor 包装
                loop = asyncio.get_event_loop()
                embedding = await loop.run_in_executor(
                    None, 
                    lambda: self._local_model.encode(text, normalize_embeddings=True).tolist()
                )
                return embedding
                
        except Exception as e:
            logger.error(f"Embedding 计算失败: {e}")
            return None
    
    async def embed_tags(self, tags: list[str], weights: Optional[list[float]] = None) -> Optional[list[float]]:
        """
        将标签列表转换为 Embedding
        
        Args:
            tags: 标签列表
            weights: 可选的权重列表 (用于 XP Profile)
            
        Returns:
            向量列表
        """
        if not tags:
            return None
        
        # 构建文本表示
        if weights:
            # 带权重的标签：重复高权重标签以增加其影响
            # 简化处理：取 Top 10 并拼接
            tagged = sorted(zip(tags, weights), key=lambda x: x[1], reverse=True)[:10]
            text = ", ".join(t[0] for t in tagged)
        else:
            # 普通标签：直接拼接 (取前 10 个)
            text = ", ".join(tags[:10])
        
        return await self.embed_text(text)
    
    async def embed_batch(self, texts: list[str]) -> list[Optional[list[float]]]:
        """
        批量计算 Embedding
        
        Args:
            texts: 文本列表
            
        Returns:
            Embedding 列表 (失败的位置为 None)
        """
        if not self.enabled or not texts:
            return [None] * len(texts)
        
        try:
            if self.provider == "openai":
                response = await self._client.embeddings.create(
                    model=self.model,
                    input=texts,
                    dimensions=self.dimensions
                )
                # 按 index 排序确保顺序正确
                sorted_data = sorted(response.data, key=lambda x: x.index)
                return [item.embedding for item in sorted_data]
                
            elif self.provider == "local":
                loop = asyncio.get_event_loop()
                embeddings = await loop.run_in_executor(
                    None,
                    lambda: self._local_model.encode(texts, normalize_embeddings=True).tolist()
                )
                return embeddings
                
        except Exception as e:
            logger.error(f"批量 Embedding 失败: {e}")
            return [None] * len(texts)
    
    @staticmethod
    def cosine_similarity(vec1: list[float], vec2: list[float]) -> float:
        """
        计算两个向量的余弦相似度
        
        Args:
            vec1, vec2: 两个向量
            
        Returns:
            相似度 (-1 到 1)
        """
        if not vec1 or not vec2:
            return 0.0
        
        a = np.array(vec1)
        b = np.array(vec2)
        
        # 余弦相似度
        dot_product = np.dot(a, b)
        norm_a = np.linalg.norm(a)
        norm_b = np.linalg.norm(b)
        
        if norm_a == 0 or norm_b == 0:
            return 0.0
        
        return float(dot_product / (norm_a * norm_b))
    
    def normalize_similarity(self, similarity: float) -> float:
        """
        将余弦相似度归一化到 0-1 范围
        
        Args:
            similarity: 余弦相似度 (-1 到 1)
            
        Returns:
            归一化分数 (0 到 1)
        """
        # 余弦相似度范围是 -1 到 1，我们映射到 0 到 1
        return (similarity + 1) / 2
