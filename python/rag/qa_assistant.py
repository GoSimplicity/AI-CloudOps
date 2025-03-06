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
import os
from typing import Dict, Any, Optional

from .knowledge_base import KnowledgeBase
from .retriever import Retriever
from .generator import Generator
from .llm_providers import LLMProvider


class QaAssistant:
  def __init__(
    self,
    knowledge_base: Optional[KnowledgeBase] = None,
    generator: Optional[Generator] = None,
    retriever: Optional[Retriever] = None,
    system_prompt: str = None,
    llm_model: str = None,
    ollama_host: str = None
  ):
    """
    初始化问答助手

    Args:
        knowledge_base: 知识库实例
        generator: 生成器实例
        retriever: 检索器实例
        system_prompt: 系统提示语
        llm_model: Ollama模型名称
        ollama_host: Ollama服务器地址
    """
    self.logger = logging.getLogger(__name__)

    # 使用提供的组件或创建新的
    self.knowledge_base = knowledge_base

    # 如果没有提供生成器，创建一个新的
    if generator is None:
      try:
        # 仅使用Ollama作为LLM提供者
        model = llm_model or os.getenv("LLM_MODEL", "deepseek-r1:8b")

        llm = LLMProvider(
          ollama_host=ollama_host or os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434"),
          default_model=model
        )
        self.generator = Generator(llm_provider=llm, system_prompt=system_prompt)
      except ImportError as e:
        self.logger.error(f"无法初始化Ollama提供者: {e}")
        raise
    else:
      self.generator = generator

    # 如果没有提供检索器但有知识库，创建检索器
    if retriever is None and knowledge_base is not None:
      vector_store = knowledge_base.vector_store
      self.retriever = Retriever(vector_store=vector_store)
    else:
      self.retriever = retriever

  def answer_question(self, question: str) -> Dict[str, Any]:
    """
    回答问题

    Args:
        question: 问题文本

    Returns:
        包含答案和来源的字典
    """
    try:
      # 检索相关文档
      docs = self.retriever.retrieve(query=question)
      self.logger.info(f"Retrieved {len(docs)} documents")

      # 如果没有找到相关文档
      if not docs:
        return {
          "answer": "抱歉，我没有找到与您问题相关的信息。",
          "sources": []
        }

      # 生成答案 详细版
      # answer = self.generator.generate_answer(question, docs)
      # 简单版
      answer = self.generator.generate_simple_answer(question, docs)

      # 如果生成答案失败
      if not answer or "抱歉，我暂时无法回答您的问题" in answer:
        return {
          "answer": "抱歉，我无法根据现有信息回答您的问题。请尝试重新表述或提供更多细节。",
          "sources": [doc.get("metadata", {}).get("source", "未知来源") for doc in docs]
        }

      return {
        "answer": answer,
        "sources": [doc.get("metadata", {}).get("source", "未知来源") for doc in docs]
      }
    except Exception as e:
      self.logger.error(f"回答问题时出错: {e}")
      return {
        "answer": "抱歉，处理您的问题时发生了错误。请稍后再试。",
        "sources": []
      }

  def answer_with_structured_output(
    self,
    question: str,
    output_format: str = "json",
    filters: Optional[Dict[str, Any]] = None
  ) -> Dict[str, Any]:
    """
    以结构化格式回答问题

    Args:
        question: 用户问题
        output_format: 输出格式 ("json" 或 "markdown")
        filters: 元数据过滤条件

    Returns:
        结构化回答结果
    """
    try:
      self.logger.info(f"Generating structured answer for: {question}")

      # 检索相关文档
      retrieved_docs = self.retriever.retrieve(
        query=question,
        filters=filters
      )

      # 生成结构化回答
      result = self.generator.generate_structured_answer(
        query=question,
        retrieved_docs=retrieved_docs,
        output_format=output_format
      )

      return result
    except Exception as e:
      self.logger.error(f"Error generating structured answer: {e}")
      return {
        "query": question,
        "structured_answer": "抱歉，生成结构化回答时发生错误。",
        "error": str(e),
        "format": output_format
      }
