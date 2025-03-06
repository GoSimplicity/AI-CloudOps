"""
MIT License

Copyright (c) 2024 Bamboo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
"""

import logging
from typing import List, Dict, Any, Optional, Union
from .vector_store import VectorStore


class Retriever:
  """检索器，用于从向量存储中检索相关文档"""

  def __init__(
    self,
    vector_store: VectorStore,
    top_k: int = 5,
    min_relevance_score: float = -1000.0  # 修改默认值，允许负数分数
  ):
    """
    初始化检索器

    Args:
        vector_store: 向量存储实例
        top_k: 检索结果数量
        min_relevance_score: 最小相关性分数
    """
    self.vector_store = vector_store
    self.top_k = top_k
    self.min_relevance_score = min_relevance_score
    self.logger = logging.getLogger(__name__)

  def retrieve(
    self,
    query: str,
    filters: Optional[Dict[str, Any]] = None,
    reranking: bool = False
  ) -> List[Dict[str, Any]]:
    """
    检索相关文档

    Args:
        query: 查询文本
        filters: 元数据过滤条件
        reranking: 是否进行重排序

    Returns:
        相关文档列表
    """
    try:
      # 检索相关文档
      search_k = self.top_k * 2 if reranking else self.top_k
      self.logger.info(f"Retrieving documents for query: '{query}' with k={search_k}")

      results = self.vector_store.search(
        query=query,
        k=search_k,
        filters=filters
      )

      self.logger.info(f"Retrieved {len(results)} documents")

      # 如果没有检索到文档，直接返回空列表
      if not results:
        return []

      # 对所有文档进行归一化处理
      for doc in results:
        # 将任何分数转换为0-1范围的正分数
        score = doc["score"]
        # 使用sigmoid函数将任何分数映射到0-1范围
        normalized_score = 1.0 / (1.0 + max(1.0, abs(score) * 0.01))
        doc["score"] = normalized_score  # 更新分数为归一化后的值

      self.logger.info(f"Normalized scores for {len(results)} documents")

      # 如果启用了重排序，对结果进行重排序
      if reranking and results:
        results = self._rerank_results(query, results)
        self.logger.info("Results reranked")

      # 限制返回结果数量
      return results[:self.top_k]

    except Exception as e:
      self.logger.error(f"Error retrieving documents: {e}")
      return []

  def _rerank_results(
    self,
    query: str,
    results: List[Dict[str, Any]]
  ) -> List[Dict[str, Any]]:
    """
    对检索结果进行重排序

    Args:
        query: 查询文本
        results: 检索结果

    Returns:
        重排序后的结果
    """
    # 计算查询词的重要性
    query_words = [word.lower() for word in query.split() if len(word) > 3]

    for doc in results:
      content = doc["content"].lower()

      # 基础分数是向量相似度
      base_score = doc["score"]

      # 计算关键词匹配得分
      keyword_matches = sum(1 for word in query_words if word in content)
      keyword_score = keyword_matches / max(1, len(query_words))

      # 计算精确短语匹配得分
      phrase_score = 1.0 if query.lower() in content else 0.0

      # 计算标题匹配得分
      title = doc.get("metadata", {}).get("title", "").lower()
      title_score = 0.0
      if title:
        title_word_matches = sum(1 for word in query_words if word in title)
        title_score = title_word_matches / max(1, len(query_words))

      # 计算最终组合分数 (可以调整权重)
      doc["combined_score"] = (
        0.6 * base_score +  # 向量相似度
        0.2 * keyword_score +  # 关键词匹配
        0.1 * phrase_score +  # 精确短语匹配
        0.1 * title_score  # 标题匹配
      )

    # 根据组合分数排序
    return sorted(results, key=lambda x: x.get("combined_score", 0), reverse=True)

  def hybrid_retrieve(
    self,
    query: str,
    filters: Optional[Dict[str, Any]] = None,
    bm25_weight: float = 0.3
  ) -> List[Dict[str, Any]]:
    """
    混合检索 (向量检索 + 关键词检索)

    Args:
        query: 查询文本
        filters: 元数据过滤条件
        bm25_weight: BM25权重

    Returns:
        相关文档列表
    """
    try:
      # 向量检索结果
      vector_results = self.vector_store.search(
        query=query,
        k=self.top_k,
        filters=filters
      )

      # 可以在此添加基于BM25的关键词检索
      # 为了示例，这里使用简单的关键词匹配作为替代
      # 在实际实现中，应该使用完整的BM25算法

      # 对每个向量结果计算更新后的分数
      for doc in vector_results:
        content = doc["content"].lower()
        query_terms = query.lower().split()

        # 计算简单的词频统计作为示例
        term_matches = sum(1 for term in query_terms if term in content)
        term_score = term_matches / max(1, len(query_terms))

        # 组合分数 (bm25_weight 比例的关键词分数 + 剩余比例的向量分数)
        doc["score"] = (1 - bm25_weight) * doc["score"] + bm25_weight * term_score

      # 重新排序
      sorted_results = sorted(vector_results, key=lambda x: x["score"], reverse=True)

      self.logger.info(f"Hybrid retrieval found {len(sorted_results)} documents")
      return sorted_results
    except Exception as e:
      self.logger.error(f"Error in hybrid retrieval: {e}")
      return []
