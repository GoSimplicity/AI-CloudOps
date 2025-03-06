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
from typing import List, Dict, Any, Optional
from langchain_community.vectorstores import Chroma, FAISS
from langchain_community.embeddings import OpenAIEmbeddings, HuggingFaceEmbeddings
from langchain_core.documents import Document

class VectorStore:
    """向量存储模块，支持多种底层向量数据库"""

    def __init__(
            self,
            persist_directory: str = "./rag/vector_store",
            embedding_model: str = None,
            embedding_provider: str = None,
            collection_name: str = "aiops_docs"
        ):
            """
            初始化向量存储

            Args:
                persist_directory: 持久化目录
                embedding_model: 嵌入模型名称
                embedding_provider: 嵌入提供者 ("openai" 或 "huggingface" 或 "ollama")
                collection_name: 集合名称
            """
            # 禁用Chroma遥测功能
            os.environ["ANONYMIZED_TELEMETRY"] = "False"

            self.persist_directory = persist_directory
            self.embedding_model = embedding_model or os.getenv("EMBEDDING_MODEL", "text-embedding-ada-002")
            self.embedding_provider = embedding_provider or os.getenv("LLM_PROVIDER", "openai").lower()
            self.collection_name = collection_name
            self.logger = logging.getLogger(__name__)

            # 创建存储目录
            os.makedirs(self.persist_directory, exist_ok=True)

            # 初始化嵌入模型
            try:
                if self.embedding_provider == "openai":
                    self.logger.info(f"Using OpenAI embeddings with model: {self.embedding_model}")
                    api_key = os.getenv("OPENAI_API_KEY")
                    if not api_key:
                        self.logger.warning("OPENAI_API_KEY not set, embeddings may fail")
                    self.embeddings = OpenAIEmbeddings(model=self.embedding_model)

                    # 测试嵌入功能
                    try:
                        test_embedding = self.embeddings.embed_query("test")
                        self.logger.info("OpenAI embeddings initialized successfully")
                    except Exception as e:
                        self.logger.error(f"OpenAI embedding test failed: {e}")
                        raise

                elif self.embedding_provider == "huggingface":
                    self.logger.info(f"Using HuggingFace embeddings with model: {self.embedding_model}")

                    # 检查sentence-transformers是否已安装
                    try:
                        import sentence_transformers
                    except ImportError:
                        self.logger.error("sentence-transformers package not installed")
                        raise ImportError("请安装sentence-transformers: pip install sentence-transformers")

                    self.embeddings = HuggingFaceEmbeddings(model_name=self.embedding_model)

                elif self.embedding_provider == "ollama":
                    # 修改嵌入模型为nomic-embed-text:latest
                    embedding_model = os.getenv("EMBEDDING_MODEL", "nomic-embed-text:latest")
                    self.logger.info(f"Using embeddings with model: {embedding_model}")

                    # 检查ollama包是否已安装
                    try:
                        from langchain_community.embeddings import OllamaEmbeddings
                    except ImportError:
                        self.logger.error("langchain_community package not installed or outdated")
                        raise ImportError("请安装最新版本的langchain和langchain_community")

                    ollama_host = os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434").rstrip("/")
                    self.logger.info(f"Connecting to Ollama at: {ollama_host}")

                    self.embeddings = OllamaEmbeddings(
                        model=embedding_model,
                        base_url=ollama_host
                    )

                    try:
                        test_embedding = self.embeddings.embed_query("test")
                        self.logger.info("Ollama embeddings initialized successfully")
                    except Exception as e:
                        self.logger.error(f"Ollama embedding test failed: {e}")
                        self.logger.error("请确保Ollama服务已启动且可访问")
                        raise
                else:
                    error_msg = f"Unsupported embedding provider: {self.embedding_provider}"
                    self.logger.error(error_msg)
                    raise ValueError(error_msg)
            except Exception as e:
                self.logger.error(f"Error initializing embeddings: {e}")
                raise

            # 初始化向量数据库
            self.db = self._initialize_vector_db()

    def _initialize_vector_db(self):
        """初始化向量数据库"""
        try:
            db_path = os.path.join(self.persist_directory, "chroma.sqlite3")
            if os.path.exists(db_path):
                self.logger.info(f"Loading existing vector database from {self.persist_directory}")
                return Chroma(
                    persist_directory=self.persist_directory,
                    embedding_function=self.embeddings,
                    collection_name=self.collection_name
                )
            else:
                self.logger.info(f"Creating new vector database at {self.persist_directory}")
                return Chroma(
                    persist_directory=self.persist_directory,
                    embedding_function=self.embeddings,
                    collection_name=self.collection_name
                )
        except Exception as e:
            self.logger.error(f"Error initializing vector database: {e}")
            raise

    def add_documents(self, documents: List[Document]) -> None:
        """添加文档到向量存储"""
        self.logger.info(f"Adding {len(documents)} documents to vector store")
        try:
            self.db.add_documents(documents)
            # 不再需要显式调用persist()，因为Chroma会自动持久化
            self.logger.info("Documents added and persisted successfully")
        except Exception as e:
            self.logger.error(f"Error adding documents to vector store: {e}")
            raise

    def persist(self) -> None:
        """
        持久化向量存储（为了兼容性保留此方法，但实际上Chroma已经自动持久化）
        """
        self.logger.info("Vector store persistence is handled automatically by Chroma")
        pass

    def search(
        self,
        query: str,
        k: int = 5,
        filters: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """
        搜索相关文档

        Args:
            query: 查询文本
            k: 返回结果数量
            filters: 元数据过滤条件

        Returns:
            相关文档列表
        """
        try:
            self.logger.info(f"Searching for: '{query}' with k={k}")
            results = self.db.similarity_search_with_relevance_scores(
                query,
                k=k,
                filter=filters
            )

            self.logger.info(f"Found {len(results)} results")

            # 修复相关性分数：将负分数转换为0-1范围
            normalized_results = []
            for doc, score in results:
                # 处理负分数，将其映射到0-1范围
                normalized_score = 1.0 / (1.0 + abs(score)) if score < 0 else score
                normalized_results.append((doc, normalized_score))

            return [
                {
                    "content": doc.page_content,
                    "metadata": doc.metadata,
                    "score": score
                }
                for doc, score in normalized_results
            ]
        except Exception as e:
            self.logger.error(f"Error searching documents: {e}")
            return []

    def delete(self, filter: Dict[str, Any]) -> None:
        """
        删除匹配条件的文档

        Args:
            filter: 元数据过滤条件
        """
        try:
            self.logger.info(f"Deleting documents with filter: {filter}")
            self.db.delete(filter=filter)
            self.logger.info("Documents deleted successfully")
        except Exception as e:
            self.logger.error(f"Error deleting documents: {e}")
            raise

    def get_collection_stats(self) -> Dict[str, Any]:
        """
        获取集合统计信息

        Returns:
            统计信息
        """
        try:
            collection = self.db._collection
            count = collection.count()
            self.logger.info(f"Collection '{self.collection_name}' has {count} documents")
            return {
                "count": count,
                "name": self.collection_name,
                "embedding_model": self.embedding_model,
                "embedding_provider": self.embedding_provider
            }
        except Exception as e:
            self.logger.error(f"Error getting collection stats: {e}")
            return {
                "error": str(e),
                "name": self.collection_name
            }

    def export_to_faiss(self, export_path: str) -> None:
        """
        导出向量存储到FAISS格式

        Args:
            export_path: 导出路径
        """
        try:
            self.logger.info(f"Exporting vector store to FAISS at {export_path}")

            # 获取所有文档和嵌入
            docs = self.db.get()

            if not docs or not docs.get("documents"):
                self.logger.warning("No documents to export")
                return

            texts = docs["documents"]
            embeddings = docs["embeddings"]
            metadatas = docs["metadatas"]

            # 创建FAISS向量存储
            faiss_db = FAISS.from_embeddings(
                text_embeddings=list(zip(texts, embeddings)),
                embedding=self.embeddings,
                metadatas=metadatas
            )

            # 保存
            os.makedirs(os.path.dirname(export_path), exist_ok=True)
            faiss_db.save_local(export_path)
            self.logger.info(f"Successfully exported to {export_path}")
        except Exception as e:
            self.logger.error(f"Error exporting to FAISS: {e}")
            raise
