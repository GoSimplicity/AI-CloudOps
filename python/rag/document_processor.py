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
import re
from typing import List, Dict, Any, Optional
from langchain.text_splitter import RecursiveCharacterTextSplitter, MarkdownHeaderTextSplitter
from langchain_community.document_loaders import (
  DirectoryLoader,
  TextLoader,
  PyPDFLoader,
  CSVLoader,
  UnstructuredMarkdownLoader
)
import logging


class DocumentProcessor:
  def __init__(
    self,
    chunk_size: int = 1000,
    chunk_overlap: int = 200,
    encoding_name: str = "utf-8"
  ):
    """
    初始化文档处理器

    Args:
        chunk_size: 文本分块大小
        chunk_overlap: 文本分块重叠大小
        encoding_name: 文本编码
    """
    self.chunk_size = chunk_size
    self.chunk_overlap = chunk_overlap
    self.encoding_name = encoding_name

    # 设置日志
    self.logger = logging.getLogger(__name__)

    # 设置Markdown专用分割器
    self.md_header_splitter = MarkdownHeaderTextSplitter(
      headers_to_split_on=[
        ("#", "header_1"),
        ("##", "header_2"),
        ("###", "header_3"),
      ]
    )

    # 优化的文本分割器（增加了更多语义化的分隔符）
    self.text_splitter = RecursiveCharacterTextSplitter(
      chunk_size=self.chunk_size,
      chunk_overlap=self.chunk_overlap,
      length_function=len,
      separators=[
        # 优先级从高到低
        "\n# ", "\n## ", "\n### ",  # Markdown标题
        "\n\n**", "\n**",  # 粗体标记（通常用于关键词）
        "\n\n", "\n",  # 段落和行
        ". ", "? ", "! ",  # 句子
        ": ", "; ",  # 分句符号
        ", ",  # 短语
        " ", ""  # 单词和字符
      ]
    )

  def load_directory(self, directory_path: str, glob_pattern: str = "**/*.*") -> List[
    Dict[str, Any]]:
    """
    加载目录中的文档

    Args:
        directory_path: 目录路径
        glob_pattern: 文件匹配模式

    Returns:
        文档列表
    """
    if not os.path.exists(directory_path):
      self.logger.error(f"Directory {directory_path} does not exist")
      raise ValueError(f"Directory {directory_path} does not exist")

    loaders = {
      ".txt": TextLoader,
      ".md": UnstructuredMarkdownLoader,
      ".pdf": PyPDFLoader,
      ".csv": CSVLoader
    }

    documents = []

    # 对每种文件类型使用适当的加载器
    for ext, loader_cls in loaders.items():
      try:
        loader = DirectoryLoader(
          directory_path,
          glob=f"**/*{ext}",
          loader_cls=loader_cls,
          loader_kwargs={"encoding": self.encoding_name} if ext in [".txt", ".md"] else {}
        )
        docs = loader.load()
        self.logger.info(f"Loaded {len(docs)} documents with extension {ext}")
        documents.extend(docs)
      except Exception as e:
        self.logger.error(f"Error loading {ext} files: {e}")

    # 在循环后添加文件类型检查
    loaded_extensions = {os.path.splitext(doc.metadata['source'])[1].lower() for doc in documents if
                         'source' in doc.metadata}
    self.logger.info(f"Loaded documents with extensions: {loaded_extensions}")

    # 处理文档前添加空值检查
    valid_documents = [doc for doc in documents if doc.page_content.strip()]
    if len(valid_documents) < len(documents):
      self.logger.warning(f"Filtered {len(documents) - len(valid_documents)} empty documents")

    processed_docs = self.process_documents(valid_documents)
    self.logger.info(f"Processed {len(processed_docs)} document chunks")

    return processed_docs

  def load_file(self, file_path: str) -> List[Dict[str, Any]]:
    """
    加载单个文件

    Args:
        file_path: 文件路径

    Returns:
        文档列表
    """
    if not os.path.exists(file_path):
      self.logger.error(f"File {file_path} does not exist")
      raise ValueError(f"File {file_path} does not exist")

    # 确定文件类型并使用适当的加载器
    file_ext = os.path.splitext(file_path)[1].lower()

    loader = None
    if file_ext == ".txt":
      loader = TextLoader(file_path, encoding=self.encoding_name)
    elif file_ext == ".md":
      loader = UnstructuredMarkdownLoader(file_path, encoding=self.encoding_name)
    elif file_ext == ".pdf":
      loader = PyPDFLoader(file_path)
    elif file_ext == ".csv":
      loader = CSVLoader(file_path)
    else:
      self.logger.error(f"Unsupported file type: {file_ext}")
      raise ValueError(f"Unsupported file type: {file_ext}")

    try:
      documents = loader.load()
      self.logger.info(f"Loaded document: {file_path}")
      processed_docs = self.process_documents(documents)
      self.logger.info(f"Processed {len(processed_docs)} document chunks")
      return processed_docs
    except Exception as e:
      self.logger.error(f"Error loading file {file_path}: {e}")
      raise

  def process_documents(self, documents: List) -> List[Dict[str, Any]]:
    """
    处理文档，分割成块

    Args:
        documents: 原始文档列表

    Returns:
        处理后的文档块列表
    """
    if not documents:
      return []

    processed_docs = []
    for doc in documents:
      # 根据文件类型选择不同的处理方法
      file_path = doc.metadata.get('source', '')
      file_ext = os.path.splitext(file_path)[1].lower() if file_path else ''

      if file_ext == '.md':
        # 对Markdown文件使用特殊的处理
        chunks = self._process_markdown_document(doc)
      else:
        # 对其他文件使用通用处理
        chunks = self._process_general_document(doc)

      processed_docs.extend(chunks)

    return processed_docs

  def _process_markdown_document(self, document):
    """
    特别处理Markdown文档，保留标题结构

    Args:
        document: Markdown文档

    Returns:
        处理后的文档块列表
    """
    # 先通过标题分割
    try:
      md_splits = self.md_header_splitter.split_text(document.page_content)

      # 再对每个标题部分进行进一步分割（如果内容较长）
      doc_chunks = []
      for md_split in md_splits:
        # 提取标题信息保存到metadata
        header_metadata = {k: v for k, v in md_split.metadata.items() if k.startswith('header_')}

        # 如果内容较长，进一步分割
        if len(md_split.page_content) > self.chunk_size:
          sub_chunks = self.text_splitter.split_text(md_split.page_content)
          for i, sub_chunk in enumerate(sub_chunks):
            # 复制原始元数据并添加新信息
            new_metadata = document.metadata.copy()
            new_metadata.update(header_metadata)
            new_metadata[
              'chunk_id'] = f"{document.metadata.get('source', 'unknown')}_chunk_{len(doc_chunks) + i}"
            new_metadata['is_subsection'] = True

            doc_chunks.append({
              "content": sub_chunk,
              "metadata": new_metadata
            })
        else:
          # 如果内容不长，直接使用
          new_metadata = document.metadata.copy()
          new_metadata.update(header_metadata)
          new_metadata[
            'chunk_id'] = f"{document.metadata.get('source', 'unknown')}_chunk_{len(doc_chunks)}"

          doc_chunks.append({
            "content": md_split.page_content,
            "metadata": new_metadata
          })
    except Exception as e:
      # 如果标题分割失败，回退到通用处理
      self.logger.warning(
        f"Markdown header splitting failed, falling back to general processing: {e}")
      doc_chunks = self._process_general_document(document)

    return doc_chunks

  def _process_general_document(self, document):
    """
    处理通用文档

    Args:
        document: 文档对象

    Returns:
        处理后的文档块列表
    """
    # 分割文档
    chunks = self.text_splitter.split_text(document.page_content)

    # 转换为标准格式
    processed_docs = []
    for i, chunk in enumerate(chunks):
      # 复制原始元数据
      new_metadata = document.metadata.copy()
      new_metadata['chunk_id'] = f"{document.metadata.get('source', 'unknown')}_chunk_{i}"

      # 提取可能的标题作为额外元数据
      title_match = re.search(r'^#+\s+(.+)$', chunk, re.MULTILINE)
      if title_match:
        new_metadata['extracted_title'] = title_match.group(1).strip()

      processed_docs.append({
        "content": chunk,
        "metadata": new_metadata
      })

    return processed_docs

  def process_text(self, text: str, metadata: Optional[Dict[str, Any]] = None) -> List[
    Dict[str, Any]]:
    """
    处理文本字符串

    Args:
        text: 文本内容
        metadata: 元数据

    Returns:
        处理后的文档块列表
    """
    if not text.strip():
      return []

    # 检查文本是否符合Markdown格式
    has_markdown_headers = bool(re.search(r'^#+\s+.+$', text, re.MULTILINE))

    if has_markdown_headers:
      # 使用Markdown处理
      try:
        md_splits = self.md_header_splitter.split_text(text)

        # 处理每个Markdown部分
        processed_docs = []
        for i, md_split in enumerate(md_splits):
          # 提取标题信息保存到metadata
          header_metadata = {k: v for k, v in md_split.metadata.items() if k.startswith('header_')}

          # 复制用户提供的元数据并添加标题信息
          doc_metadata = metadata.copy() if metadata else {}
          doc_metadata.update(header_metadata)
          doc_metadata["chunk_id"] = i

          # 如果每个部分仍然太长，进一步分割
          if len(md_split.page_content) > self.chunk_size:
            sub_chunks = self.text_splitter.split_text(md_split.page_content)
            for j, sub_chunk in enumerate(sub_chunks):
              sub_metadata = doc_metadata.copy()
              sub_metadata["sub_chunk_id"] = j

              processed_docs.append({
                "content": sub_chunk,
                "metadata": sub_metadata
              })
          else:
            processed_docs.append({
              "content": md_split.page_content,
              "metadata": doc_metadata
            })

        return processed_docs

      except Exception as e:
        # 如果Markdown处理失败，回退到通用处理
        self.logger.warning(f"Markdown processing failed, using general text splitting: {e}")

    # 通用文本处理
    chunks = self.text_splitter.split_text(text)

    # 转换为标准格式
    processed_docs = []
    for i, chunk in enumerate(chunks):
      doc_metadata = metadata.copy() if metadata else {}
      doc_metadata["chunk_id"] = i

      processed_docs.append({
        "content": chunk,
        "metadata": doc_metadata
      })

    return processed_docs
