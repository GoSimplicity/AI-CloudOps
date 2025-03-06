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

import os
import logging
from typing import List, Dict, Any
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain_community.document_loaders import DirectoryLoader, TextLoader
from langchain.schema import Document
from rag.vector_store import VectorStore
from langchain_ollama import OllamaEmbeddings
from langchain_community.document_loaders import UnstructuredMarkdownLoader, PyPDFLoader


class KnowledgeBase:
  """知识库，集成文档处理和向量存储功能"""

  def __init__(
    self,
    docs_dir: str = "docs",
    embedding_model: str = None,
    persist_directory: str = "./data/chroma_db",
    chunk_size: int = 1000,
    chunk_overlap: int = 200
  ):
    """
    初始化知识库

    Args:
        docs_dir: 文档目录
        embedding_model: embedding模型名称
        persist_directory: 向量存储持久化目录
        chunk_size: 文本分块大小
        chunk_overlap: 文本分块重叠大小
    """
    self.docs_dir = docs_dir
    self.persist_directory = persist_directory
    self.embedding_model = embedding_model or os.getenv("EMBEDDING_MODEL", "llama2")
    self.chunk_size = chunk_size
    self.chunk_overlap = chunk_overlap
    self.logger = logging.getLogger(__name__)

    # 初始化嵌入模型
    self.embeddings = OllamaEmbeddings(
      model=self.embedding_model,
      base_url=os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434")
    )

    # 初始化向量存储
    self.vector_store = VectorStore(
      persist_directory=persist_directory,
      embedding_model=self.embedding_model,
      embedding_provider="ollama"
    )

    # 创建存储目录
    os.makedirs(self.persist_directory, exist_ok=True)
    os.makedirs(self.docs_dir, exist_ok=True)

  def load_documents(self, file_types: List[str] = None) -> List[Document]:
    """加载文档并创建向量存储"""
    try:
      # 处理文档
      if not os.path.exists(self.docs_dir):
        self.logger.warning(f"文档目录不存在: {self.docs_dir}")
        os.makedirs(self.docs_dir, exist_ok=True)
        self._add_sample_documents()

      # 加载和处理文档
      documents = self.load_from_directory(file_types)
      if not documents:
        self.logger.warning("没有找到文档，将添加示例文档")
        self._add_sample_documents()
        documents = self.load_from_directory(file_types)

      # 将文档添加到向量存储
      self.logger.info("将文档添加到向量存储...")
      self.vector_store.add_documents(documents)

      return documents
    except Exception as e:
      self.logger.error(f"加载文档时出错: {e}")
      raise

  def load_from_directory(self, file_types: List[str] = None) -> List[Document]:
    """
    从指定目录加载文档

    Args:
        file_types: 要加载的文件类型列表，例如 ['.txt', '.md', '.pdf']

    Returns:
        加载的文档列表
    """
    if file_types is None:
      file_types = ['.txt', '.md', '.pdf']

    documents = []
    self.logger.info(f"从目录 {self.docs_dir} 加载文件类型: {file_types}")

    try:
      for file_type in file_types:
        self.logger.info(f"加载 {file_type} 文件...")
        if file_type.lower() == '.txt':
          loader = DirectoryLoader(
            self.docs_dir,
            glob=f"**/*{file_type}",
            loader_cls=TextLoader
          )
          docs = loader.load() if os.path.exists(self.docs_dir) else []
          documents.extend(docs)
        elif file_type.lower() == '.md':
          for root, _, files in os.walk(self.docs_dir):
            for file in files:
              if file.endswith('.md'):
                file_path = os.path.join(root, file)
                try:
                  loader = UnstructuredMarkdownLoader(file_path)
                  documents.extend(loader.load())
                except Exception as e:
                  self.logger.error(f"加载Markdown文件 {file_path} 时出错: {e}")
        elif file_type.lower() == '.pdf':
          for root, _, files in os.walk(self.docs_dir):
            for file in files:
              if file.endswith('.pdf'):
                file_path = os.path.join(root, file)
                try:
                  loader = PyPDFLoader(file_path)
                  documents.extend(loader.load())
                except Exception as e:
                  self.logger.error(f"加载PDF文件 {file_path} 时出错: {e}")

      # 分割文档
      self.logger.info(f"加载了 {len(documents)} 个文档，正在进行文本分割...")
      text_splitter = RecursiveCharacterTextSplitter(
        chunk_size=self.chunk_size,
        chunk_overlap=self.chunk_overlap
      )
      split_docs = text_splitter.split_documents(documents)
      self.logger.info(f"文档分割完成，共 {len(split_docs)} 个块")

      return split_docs
    except Exception as e:
      self.logger.error(f"从目录加载文档时出错: {e}")
      raise

  def _add_sample_documents(self):
    """添加示例文档以确保知识库不为空"""
    self.logger.info("添加示例文档到知识库...")
    sample_dir = os.path.join(self.docs_dir, "samples")
    os.makedirs(sample_dir, exist_ok=True)

    sample_content = (
      "# 示例文档\n\n"
      "这是一个示例文档，用于测试知识库功能。\n\n"
      "## 基本信息\n\n"
      "- 项目名称: 智能RAG系统\n"
      "- 版本: 1.0.0\n"
      "- 创建日期: 2025-03-06\n\n"
      "## 功能\n\n"
      "该系统提供以下功能:\n"
      "1. 文档加载和处理\n"
      "2. 向量存储和检索\n"
      "3. 与大语言模型集成\n\n"
    )

    with open(os.path.join(sample_dir, "sample.md"), "w", encoding="utf-8") as f:
      f.write(sample_content)

    self.logger.info("示例文档添加完成")

  def search(self, query: str, k: int = 3) -> List[Dict[str, Any]]:
    """
    搜索相关文档

    Args:
        query: 查询文本
        k: 返回结果数量

    Returns:
        相关文档列表
    """
    if not self.vector_store:
      self.logger.error("知识库未初始化，请先调用 load_documents()")
      raise ValueError("知识库未初始化，请先调用 load_documents()")

    try:
      results = self.vector_store.search(query, k=k)

      formatted_results = [
        {
          "content": doc.page_content,
          "metadata": doc.metadata,
          "score": float(score)
        }
        for doc, score in results
      ]

      self.logger.info(f"查询 '{query}' 找到 {len(formatted_results)} 个相关文档")
      return formatted_results
    except Exception as e:
      self.logger.error(f"搜索知识库时出错: {e}")
      raise

  def add_documents(self, documents: List[Dict[str, Any]]) -> None:
    """
    向知识库添加文档

    Args:
        documents: 文档列表
    """
    if not self.vector_store:
      self.logger.error("知识库未初始化，请先调用 load_documents()")
      raise ValueError("知识库未初始化，请先调用 load_documents()")

    try:
      langchain_docs = [
        Document(page_content=doc["content"], metadata=doc.get("metadata", {}))
        for doc in documents
      ]

      self.vector_store.add_documents(langchain_docs)
      self.vector_store.persist()
      self.logger.info(f"向知识库添加了 {len(documents)} 个文档")
    except Exception as e:
      self.logger.error(f"向知识库添加文档时出错: {e}")
      raise
