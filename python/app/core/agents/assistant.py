import os
import sys
import uuid
import logging
import re
import time
import json
import asyncio
import hashlib
import shutil
from datetime import datetime
from typing import List, Dict, Any, Optional, Union, Tuple
from pathlib import Path
from dataclasses import dataclass
from contextlib import asynccontextmanager
import threading
from concurrent.futures import ThreadPoolExecutor

# 核心依赖
from langchain_chroma import Chroma
from langchain_openai import OpenAIEmbeddings, ChatOpenAI
from langchain_ollama import OllamaEmbeddings, ChatOllama
from langchain.text_splitter import RecursiveCharacterTextSplitter, MarkdownHeaderTextSplitter
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.pydantic_v1 import BaseModel, Field
from langchain_core.output_parsers import StrOutputParser, JsonOutputParser
from langchain_core.documents import Document
from langchain_core.messages import HumanMessage, AIMessage, SystemMessage
from langchain_core.embeddings import Embeddings
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.outputs import ChatGeneration, ChatResult

# 高级加载器（可选）
try:
    from langchain_community.document_loaders import (
        TextLoader, PyPDFLoader, DirectoryLoader,
        UnstructuredMarkdownLoader, CSVLoader, JSONLoader, BSHTMLLoader
    )
    from langchain_community.utilities import TavilySearchAPIWrapper
    ADVANCED_LOADERS_AVAILABLE = True
    WEB_SEARCH_AVAILABLE = True
except ImportError:
    ADVANCED_LOADERS_AVAILABLE = False
    WEB_SEARCH_AVAILABLE = False

import numpy as np
from chromadb.config import Settings

from app.config.settings import config

logger = logging.getLogger("aiops.assistant")

# ==================== 数据类和模型定义 ====================

@dataclass
class DocumentMetadata:
    """文档元数据"""
    source: str
    filename: str
    filetype: str
    modified_time: float
    is_web_result: bool = False
    relevance_score: float = 0.0
    recall_rate: float = 0.0

@dataclass
class CacheEntry:
    """缓存条目"""
    timestamp: float
    data: Dict[str, Any]
    
    def is_expired(self, expiry_seconds: int) -> bool:
        return time.time() - self.timestamp > expiry_seconds

@dataclass
class SessionData:
    """会话数据"""
    session_id: str
    created_at: str
    history: List[Dict[str, Any]]
    metadata: Dict[str, Any]

class GradeDocuments(BaseModel):
    """文档相关性评估模型"""
    binary_score: str = Field(description="文档是否与问题相关，'yes'或'no'")

class GradeHallucinations(BaseModel):
    """幻觉检测模型"""
    binary_score: str = Field(description="回答是否基于事实，'yes'或'no'")

# ==================== 备用实现类 ====================

class FallbackEmbeddings(Embeddings):
    """备用嵌入实现，使用简单的哈希和随机生成"""
    
    def __init__(self, dimensions: int = 384):
        self.dimensions = dimensions
        
    def embed_documents(self, texts: List[str]) -> List[List[float]]:
        return [self.embed_query(text) for text in texts]
        
    def embed_query(self, text: str) -> List[float]:
        # 使用文本哈希生成确定性向量
        text_hash = hash(text) % (2**32)
        np.random.seed(text_hash)
        return list(np.random.rand(self.dimensions))

class FallbackChatModel(BaseChatModel):
    """备用聊天模型，提供基础响应"""
    
    @property
    def _llm_type(self) -> str:
        return "fallback_chat_model"
    
    def _generate(self, messages, stop=None, run_manager=None, **kwargs):
        last_message = messages[-1].content if messages else "无输入"
        response = f"我是备用助手。您的问题是：'{last_message}'。由于主要模型暂时不可用，功能受限。请稍后重试。"
        message = AIMessage(content=response)
        generation = ChatGeneration(message=message)
        return ChatResult(generations=[generation])

# ==================== 向量存储管理器 ====================

class VectorStoreManager:
    """向量存储管理器，负责向量数据库的创建、维护和查询"""
    
    def __init__(self, vector_db_path: str, collection_name: str, embedding_model):
        self.vector_db_path = vector_db_path
        self.collection_name = collection_name
        self.embedding_model = embedding_model
        self.db = None
        self.retriever = None
        self._lock = threading.Lock()
        
        # 确保目录存在
        os.makedirs(vector_db_path, exist_ok=True)
        
    def _get_client_settings(self, persistent: bool = True) -> Settings:
        """获取ChromaDB客户端设置"""
        return Settings(
            anonymized_telemetry=False,
            allow_reset=True,
            is_persistent=persistent,
            chroma_db_impl="duckdb+parquet" if not persistent else None
        )
    
    def _cleanup_temp_files(self):
        """清理临时文件，避免数据库锁定问题"""
        temp_files = [
            os.path.join(self.vector_db_path, ".lock"),
            os.path.join(self.vector_db_path, ".uuid"),
            os.path.join(self.vector_db_path, "chroma.sqlite3-shm"),
            os.path.join(self.vector_db_path, "chroma.sqlite3-wal")
        ]
        
        for file_path in temp_files:
            try:
                if os.path.exists(file_path):
                    os.remove(file_path)
                    logger.debug(f"清理临时文件: {file_path}")
            except Exception as e:
                logger.warning(f"清理临时文件失败 {file_path}: {e}")
    
    def load_existing_db(self) -> bool:
        """加载现有数据库"""
        db_file = os.path.join(self.vector_db_path, "chroma.sqlite3")
        
        if not os.path.exists(db_file):
            return False
            
        try:
            with self._lock:
                logger.info(f"加载现有向量数据库: {db_file}")
                self.db = Chroma(
                    persist_directory=self.vector_db_path,
                    embedding_function=self.embedding_model,
                    collection_name=self.collection_name,
                    client_settings=self._get_client_settings(persistent=True)
                )
                
                self.retriever = self.db.as_retriever(
                    search_kwargs={"k": config.rag.top_k}
                )
                
                # 测试数据库
                test_results = self.retriever.invoke("测试查询")
                logger.info(f"数据库加载成功，测试查询返回 {len(test_results)} 个结果")
                return True
                
        except Exception as e:
            logger.error(f"加载现有数据库失败: {e}")
            self._backup_corrupted_db(db_file)
            return False
    
    def _backup_corrupted_db(self, db_file: str):
        """备份损坏的数据库"""
        try:
            backup_dir = os.path.join(self.vector_db_path, f"backup_corrupt_{int(time.time())}")
            os.makedirs(backup_dir, exist_ok=True)
            shutil.copy2(db_file, backup_dir)
            os.remove(db_file)
            logger.info(f"已备份并删除损坏的数据库文件: {backup_dir}")
        except Exception as e:
            logger.error(f"备份损坏数据库失败: {e}")
    
    def create_vector_store(self, documents: List[Document], use_memory: bool = False) -> bool:
        """创建向量存储"""
        if not documents:
            logger.warning("没有文档可供创建向量存储")
            documents = [Document(
                page_content="这是一个系统自动创建的示例文档。请添加更多文档到知识库中。",
                metadata={"source": "system", "filename": "example.txt", "filetype": "text"}
            )]
        
        # 文档分割
        text_splitter = RecursiveCharacterTextSplitter(
            chunk_size=config.rag.chunk_size,
            chunk_overlap=config.rag.chunk_overlap,
            separators=["\n\n", "\n", "。", "！", "？", ".", "!", "?", " ", ""]
        )
        
        splits = text_splitter.split_documents(documents)
        logger.info(f"文档分割完成: {len(splits)} 个块")
        
        try:
            with self._lock:
                self._cleanup_temp_files()
                
                if use_memory:
                    logger.info("使用内存模式创建向量存储")
                    client_settings = self._get_client_settings(persistent=False)
                else:
                    logger.info("使用持久化模式创建向量存储")
                    client_settings = self._get_client_settings(persistent=True)
                
                self.db = Chroma.from_documents(
                    documents=splits,
                    embedding=self.embedding_model,
                    persist_directory=None if use_memory else self.vector_db_path,
                    collection_name=self.collection_name,
                    client_settings=client_settings
                )
                
                self.retriever = self.db.as_retriever(
                    search_kwargs={"k": config.rag.top_k}
                )
                
                # 测试创建的数据库
                test_results = self.retriever.invoke("测试查询")
                logger.info(f"向量存储创建成功，包含 {len(splits)} 个文档块")
                return True
                
        except Exception as e:
            logger.error(f"创建向量存储失败: {e}")
            if not use_memory:
                logger.info("尝试回退到内存模式")
                return self.create_vector_store(documents, use_memory=True)
            return False
    
    def get_retriever(self):
        """获取检索器"""
        return self.retriever
    
    def search_documents(self, query: str, max_retries: int = 3) -> List[Document]:
        """搜索文档"""
        if not self.retriever:
            logger.warning("检索器未初始化")
            return []
        
        for attempt in range(max_retries):
            try:
                docs = self.retriever.invoke(query)
                if docs:
                    logger.debug(f"搜索到 {len(docs)} 个文档")
                    return docs
                else:
                    logger.warning(f"搜索返回空结果 (尝试 {attempt+1}/{max_retries})")
                    
            except Exception as e:
                logger.error(f"文档搜索失败 (尝试 {attempt+1}/{max_retries}): {e}")
                if attempt < max_retries - 1:
                    time.sleep(1)
        
        return []

# ==================== 缓存管理器 ====================

class CacheManager:
    """缓存管理器，负责响应缓存的管理"""
    
    def __init__(self, cache_dir: str, expiry_seconds: int = 3600):
        self.cache_dir = cache_dir
        self.expiry_seconds = expiry_seconds
        self.cache: Dict[str, CacheEntry] = {}
        self.cache_file = os.path.join(cache_dir, "response_cache.json")
        self._lock = threading.Lock()
        
        os.makedirs(cache_dir, exist_ok=True)
        self._load_cache()
    
    def _generate_cache_key(self, question: str, session_id: str = None, history: List = None) -> str:
        """生成缓存键"""
        cache_input = question
        
        if session_id and history:
            # 使用最近的对话历史生成缓存键
            recent_history = history[-3:] if len(history) >= 3 else history
            if recent_history:
                history_str = json.dumps([
                    {"role": h["role"], "content": h["content"][:50]}
                    for h in recent_history
                ], ensure_ascii=False)
                cache_input = f"{question}|{history_str}"
        
        return hashlib.sha256(cache_input.encode('utf-8')).hexdigest()
    
    def _load_cache(self):
        """加载缓存文件"""
        try:
            if os.path.exists(self.cache_file):
                with open(self.cache_file, 'r', encoding='utf-8') as f:
                    cache_data = json.load(f)
                    
                # 过滤过期缓存
                valid_cache = {}
                for k, v in cache_data.items():
                    entry = CacheEntry(timestamp=v["timestamp"], data=v["data"])
                    if not entry.is_expired(self.expiry_seconds):
                        valid_cache[k] = entry
                
                self.cache = valid_cache
                logger.info(f"加载了 {len(self.cache)} 条有效缓存")
        except Exception as e:
            logger.warning(f"加载缓存失败: {e}")
            self.cache = {}
    
    def _save_cache(self):
        """保存缓存到文件"""
        try:
            with self._lock:
                # 清理过期缓存
                valid_cache = {
                    k: {"timestamp": v.timestamp, "data": v.data}
                    for k, v in self.cache.items()
                    if not v.is_expired(self.expiry_seconds)
                }
                
                with open(self.cache_file, 'w', encoding='utf-8') as f:
                    json.dump(valid_cache, f, ensure_ascii=False, indent=2)
                
                self.cache = {
                    k: CacheEntry(timestamp=v["timestamp"], data=v["data"])
                    for k, v in valid_cache.items()
                }
                
                logger.debug(f"保存了 {len(valid_cache)} 条缓存")
        except Exception as e:
            logger.warning(f"保存缓存失败: {e}")
    
    def get(self, question: str, session_id: str = None, history: List = None) -> Optional[Dict[str, Any]]:
        """获取缓存"""
        cache_key = self._generate_cache_key(question, session_id, history)
        
        if cache_key in self.cache:
            entry = self.cache[cache_key]
            if not entry.is_expired(self.expiry_seconds):
                logger.debug(f"缓存命中: {cache_key[:8]}...")
                return entry.data
            else:
                # 删除过期缓存
                del self.cache[cache_key]
        
        return None
    
    def set(self, question: str, response_data: Dict[str, Any], session_id: str = None, history: List = None):
        """设置缓存"""
        cache_key = self._generate_cache_key(question, session_id, history)
        entry = CacheEntry(timestamp=time.time(), data=response_data)
        
        with self._lock:
            self.cache[cache_key] = entry
    
    async def save_async(self):
        """异步保存缓存"""
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self._save_cache)

# ==================== 文档加载器 ====================

class DocumentLoader:
    """文档加载器，负责加载各种格式的文档"""
    
    def __init__(self, knowledge_base_path: str):
        self.knowledge_base_path = Path(knowledge_base_path)
        self.executor = ThreadPoolExecutor(max_workers=4)
        
        # 支持的文件扩展名
        self.supported_extensions = {
            '.txt': self._load_text_file,
            '.md': self._load_markdown_file,
            '.markdown': self._load_markdown_file,
        }
        
        # 如果高级加载器可用，添加更多支持
        if ADVANCED_LOADERS_AVAILABLE:
            self.supported_extensions.update({
                '.pdf': self._load_pdf_file,
                '.html': self._load_html_file,
                '.htm': self._load_html_file,
                '.csv': self._load_csv_file,
                '.json': self._load_json_file,
            })
    
    def load_documents(self) -> List[Document]:
        """加载所有支持的文档"""
        if not self.knowledge_base_path.exists():
            logger.warning(f"知识库路径不存在: {self.knowledge_base_path}")
            self.knowledge_base_path.mkdir(parents=True, exist_ok=True)
            return []
        
        documents = []
        all_files = list(self.knowledge_base_path.rglob("*"))
        supported_files = [
            f for f in all_files 
            if f.is_file() and f.suffix.lower() in self.supported_extensions
        ]
        
        # 按修改时间排序
        supported_files.sort(key=lambda x: x.stat().st_mtime, reverse=True)
        
        logger.info(f"发现 {len(supported_files)} 个支持的文件")
        
        # 并行加载文件
        for file_path in supported_files:
            try:
                loader_func = self.supported_extensions[file_path.suffix.lower()]
                file_docs = loader_func(file_path)
                documents.extend(file_docs)
                logger.debug(f"加载文件: {file_path.name}, 生成 {len(file_docs)} 个文档")
            except Exception as e:
                logger.error(f"加载文件失败 {file_path}: {e}")
        
        logger.info(f"总共加载 {len(documents)} 个文档")
        return documents
    
    def _load_text_file(self, file_path: Path) -> List[Document]:
        """加载文本文件"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read().strip()
                
            if not content:
                return []
            
            return [Document(
                page_content=content,
                metadata={
                    "source": str(file_path),
                    "filename": file_path.name,
                    "filetype": "text",
                    "modified_time": file_path.stat().st_mtime
                }
            )]
        except Exception as e:
            logger.error(f"加载文本文件失败 {file_path}: {e}")
            return []
    
    def _load_markdown_file(self, file_path: Path) -> List[Document]:
        """加载Markdown文件，使用标题分割"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read().strip()
                
            if not content:
                return []
            
            # 尝试使用Markdown标题分割
            try:
                headers_to_split_on = [
                    ("#", "Header 1"),
                    ("##", "Header 2"),
                    ("###", "Header 3"),
                ]
                
                markdown_splitter = MarkdownHeaderTextSplitter(
                    headers_to_split_on=headers_to_split_on
                )
                md_docs = markdown_splitter.split_text(content)
                
                # 添加元数据
                for doc in md_docs:
                    doc.metadata.update({
                        "source": str(file_path),
                        "filename": file_path.name,
                        "filetype": "markdown",
                        "modified_time": file_path.stat().st_mtime
                    })
                
                return md_docs
                
            except Exception:
                # 如果分割失败，作为单个文档处理
                return [Document(
                    page_content=content,
                    metadata={
                        "source": str(file_path),
                        "filename": file_path.name,
                        "filetype": "markdown",
                        "modified_time": file_path.stat().st_mtime
                    }
                )]
                
        except Exception as e:
            logger.error(f"加载Markdown文件失败 {file_path}: {e}")
            return []
    
    def _load_pdf_file(self, file_path: Path) -> List[Document]:
        """加载PDF文件"""
        if not ADVANCED_LOADERS_AVAILABLE:
            return []
            
        try:
            loader = PyPDFLoader(str(file_path))
            pdf_docs = loader.load()
            
            for doc in pdf_docs:
                doc.metadata.update({
                    "filename": file_path.name,
                    "filetype": "pdf",
                    "modified_time": file_path.stat().st_mtime
                })
            
            return pdf_docs
            
        except Exception as e:
            logger.error(f"加载PDF文件失败 {file_path}: {e}")
            return []
    
    def _load_html_file(self, file_path: Path) -> List[Document]:
        """加载HTML文件"""
        if not ADVANCED_LOADERS_AVAILABLE:
            return []
            
        try:
            loader = BSHTMLLoader(str(file_path))
            html_docs = loader.load()
            
            for doc in html_docs:
                doc.metadata.update({
                    "filename": file_path.name,
                    "filetype": "html",
                    "modified_time": file_path.stat().st_mtime
                })
            
            return html_docs
            
        except Exception as e:
            logger.error(f"加载HTML文件失败 {file_path}: {e}")
            return []
    
    def _load_csv_file(self, file_path: Path) -> List[Document]:
        """加载CSV文件"""
        if not ADVANCED_LOADERS_AVAILABLE:
            return []
            
        try:
            loader = CSVLoader(str(file_path))
            csv_docs = loader.load()
            
            for doc in csv_docs:
                doc.metadata.update({
                    "filename": file_path.name,
                    "filetype": "csv",
                    "modified_time": file_path.stat().st_mtime
                })
            
            return csv_docs
            
        except Exception as e:
            logger.error(f"加载CSV文件失败 {file_path}: {e}")
            return []
    
    def _load_json_file(self, file_path: Path) -> List[Document]:
        """加载JSON文件"""
        if not ADVANCED_LOADERS_AVAILABLE:
            return []
            
        try:
            loader = JSONLoader(str(file_path), jq_schema='.', text_content=False)
            json_docs = loader.load()
            
            for doc in json_docs:
                doc.metadata.update({
                    "filename": file_path.name,
                    "filetype": "json",
                    "modified_time": file_path.stat().st_mtime
                })
            
            return json_docs
            
        except Exception as e:
            logger.error(f"加载JSON文件失败 {file_path}: {e}")
            return []

# ==================== 主要的智能助手类 ====================

class AssistantAgent:
    """智能小助手代理 - 优化版"""
    
    def __init__(self):
        """初始化助手代理"""
        self.llm_provider = config.llm.provider.lower()
        
        # 路径设置
        base_dir = Path(__file__).parent.parent.parent.parent
        self.vector_db_path = base_dir / config.rag.vector_db_path
        self.knowledge_base_path = base_dir / config.rag.knowledge_base_path
        self.collection_name = config.rag.collection_name
        
        # 创建必要目录
        self.vector_db_path.mkdir(parents=True, exist_ok=True)
        self.knowledge_base_path.mkdir(parents=True, exist_ok=True)
        
        # 初始化组件
        self.embedding = None
        self.llm = None
        self.task_llm = None
        self.web_search = None
        
        # 管理器
        self.vector_store_manager = None
        self.cache_manager = CacheManager(str(self.vector_db_path / "cache"))
        self.document_loader = DocumentLoader(str(self.knowledge_base_path))
        
        # 会话管理
        self.sessions: Dict[str, SessionData] = {}
        self._session_lock = threading.Lock()
        
        # 线程池
        self.executor = ThreadPoolExecutor(max_workers=4)
        
        # 初始化所有组件
        self._initialize_components()
        
        logger.info(f"智能小助手初始化完成，提供商: {self.llm_provider}")
    
    def _initialize_components(self):
        """初始化所有组件"""
        try:
            # 1. 初始化嵌入模型
            self._init_embedding()
            
            # 2. 初始化语言模型
            self._init_llm()
            
            # 3. 初始化向量存储
            self._init_vector_store()
            
            # 4. 初始化网络搜索
            self._init_web_search()
            
        except Exception as e:
            logger.error(f"组件初始化失败: {e}")
            raise
    
    def _init_embedding(self):
        """初始化嵌入模型，带有重试和回退机制"""
        max_retries = 3
        
        for attempt in range(max_retries):
            try:
                if self.llm_provider == 'openai':
                    logger.info(f"初始化OpenAI嵌入模型 (尝试 {attempt+1})")
                    self.embedding = OpenAIEmbeddings(
                        model=config.rag.openai_embedding_model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url,
                        timeout=10,
                        max_retries=2
                    )
                else:
                    logger.info(f"初始化Ollama嵌入模型 (尝试 {attempt+1})")
                    self.embedding = OllamaEmbeddings(
                        model=config.rag.ollama_embedding_model,
                        base_url=config.llm.ollama_base_url,
                        timeout=10
                    )
                
                # 测试嵌入
                test_embedding = self.embedding.embed_query("测试")
                if test_embedding and len(test_embedding) > 0:
                    logger.info(f"嵌入模型初始化成功，维度: {len(test_embedding)}")
                    return
                    
            except Exception as e:
                logger.error(f"嵌入模型初始化失败 (尝试 {attempt+1}): {e}")
                if attempt < max_retries - 1:
                    # 切换提供商重试
                    self.llm_provider = 'ollama' if self.llm_provider == 'openai' else 'openai'
                    time.sleep(1)
        
        # 使用备用嵌入
        logger.warning("使用备用嵌入模型")
        self.embedding = FallbackEmbeddings()
    
    def _init_llm(self):
        """初始化语言模型"""
        max_retries = 3
        
        for attempt in range(max_retries):
            try:
                if self.llm_provider == 'openai':
                    logger.info(f"初始化OpenAI语言模型 (尝试 {attempt+1})")
                    
                    # 主聊天模型
                    self.llm = ChatOpenAI(
                        model=config.llm.model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url,
                        temperature=config.rag.temperature,
                        timeout=30,
                        max_retries=2
                    )
                    
                    # 任务模型
                    task_model = getattr(config.llm, 'task_model', config.llm.model)
                    self.task_llm = ChatOpenAI(
                        model=task_model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url,
                        temperature=0.1,
                        timeout=15,
                        max_retries=2
                    )
                else:
                    logger.info(f"初始化Ollama语言模型 (尝试 {attempt+1})")
                    self.llm = ChatOllama(
                        model=config.llm.ollama_model,
                        base_url=config.llm.ollama_base_url,
                        temperature=config.rag.temperature,
                        timeout=30
                    )
                    self.task_llm = self.llm
                
                # 测试模型
                test_response = self.llm.invoke("返回'OK'")
                if test_response and test_response.content:
                    logger.info("语言模型初始化成功")
                    return
                    
            except Exception as e:
                logger.error(f"语言模型初始化失败 (尝试 {attempt+1}): {e}")
                if attempt < max_retries - 1:
                    self.llm_provider = 'ollama' if self.llm_provider == 'openai' else 'openai'
                    time.sleep(1)
        
        # 使用备用模型
        logger.warning("使用备用语言模型")
        self.llm = FallbackChatModel()
        self.task_llm = self.llm
    
    def _init_vector_store(self):
        """初始化向量存储"""
        self.vector_store_manager = VectorStoreManager(
            str(self.vector_db_path),
            self.collection_name,
            self.embedding
        )
        
        # 尝试加载现有数据库
        if not self.vector_store_manager.load_existing_db():
            # 如果没有现有数据库，创建新的
            logger.info("创建新的向量数据库")
            documents = self.document_loader.load_documents()
            
            # 检查是否在测试环境
            use_memory = 'pytest' in sys.modules
            success = self.vector_store_manager.create_vector_store(documents, use_memory)
            
            if not success:
                logger.error("向量存储初始化失败")
                raise RuntimeError("无法初始化向量存储")
        
        logger.info("向量存储初始化完成")
    
    def _init_web_search(self):
        """初始化网络搜索"""
        if WEB_SEARCH_AVAILABLE and config.tavily.api_key:
            try:
                self.web_search = TavilySearchAPIWrapper(
                    api_key=config.tavily.api_key,
                    max_results=config.tavily.max_results
                )
                logger.info("网络搜索工具初始化成功")
            except Exception as e:
                logger.warning(f"网络搜索工具初始化失败: {e}")
                self.web_search = None
        else:
            self.web_search = None
            logger.info("网络搜索功能未启用")
    
    # ==================== 会话管理 ====================
    
    def create_session(self) -> str:
        """创建新会话"""
        session_id = str(uuid.uuid4())
        session_data = SessionData(
            session_id=session_id,
            created_at=datetime.now().isoformat(),
            history=[],
            metadata={}
        )
        
        with self._session_lock:
            self.sessions[session_id] = session_data
        
        return session_id
    
    def get_session(self, session_id: str) -> Optional[SessionData]:
        """获取会话数据"""
        return self.sessions.get(session_id)
    
    def add_message_to_history(self, session_id: str, role: str, content: str) -> str:
        """添加消息到会话历史"""
        if session_id not in self.sessions:
            session_id = self.create_session()
        
        with self._session_lock:
            session = self.sessions[session_id]
            session.history.append({
                "role": role,
                "content": content,
                "timestamp": datetime.now().isoformat()
            })
            
            # 限制历史长度
            max_history = 20
            if len(session.history) > max_history:
                session.history = session.history[-max_history:]
        
        return session_id
    
    def clear_session_history(self, session_id: str) -> bool:
        """清空会话历史"""
        if session_id in self.sessions:
            with self._session_lock:
                self.sessions[session_id].history = []
            return True
        return False
    
    # ==================== 知识库管理 ====================
    
    async def refresh_knowledge_base(self) -> Dict[str, Any]:
        """刷新知识库"""
        try:
            logger.info("开始刷新知识库...")
            
            # 清理缓存
            self.cache_manager = CacheManager(str(self.vector_db_path / "cache"))
            
            # 加载文档
            documents = await asyncio.get_event_loop().run_in_executor(
                self.executor, self.document_loader.load_documents
            )
            
            # 重新创建向量存储
            use_memory = 'pytest' in sys.modules
            success = await asyncio.get_event_loop().run_in_executor(
                self.executor,
                self.vector_store_manager.create_vector_store,
                documents,
                use_memory
            )
            
            if success:
                doc_count = len(documents)
                logger.info(f"知识库刷新成功，包含 {doc_count} 个文档")
                return {"success": True, "documents_count": doc_count}
            else:
                return {"success": False, "documents_count": 0, "error": "向量存储创建失败"}
                
        except Exception as e:
            logger.error(f"刷新知识库失败: {e}")
            return {"success": False, "documents_count": 0, "error": str(e)}
    
    def add_document(self, content: str, metadata: Dict[str, Any] = None) -> bool:
        """添加文档到知识库"""
        try:
            if not content.strip():
                return False
            
            # 生成文件名
            doc_id = str(uuid.uuid4())
            filename = metadata.get('filename', f"{doc_id}.txt") if metadata else f"{doc_id}.txt"
            file_path = self.knowledge_base_path / filename
            
            # 写入文件
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            
            # 清理缓存
            self.cache_manager = CacheManager(str(self.vector_db_path / "cache"))
            
            logger.info(f"文档已添加: {filename}")
            return True
            
        except Exception as e:
            logger.error(f"添加文档失败: {e}")
            return False
    
    # ==================== 网络搜索 ====================
    
    async def search_web(self, query: str, max_results: int = None) -> List[Dict]:
        """网络搜索"""
        if not self.web_search:
            return []
        
        try:
            max_results = max_results or config.tavily.max_results
            results = await asyncio.get_event_loop().run_in_executor(
                self.executor,
                self.web_search.results,
                query,
                max_results
            )
            return results
        except Exception as e:
            logger.error(f"网络搜索失败: {e}")
            return []
    
    # ==================== 核心问答逻辑 ====================
    
    async def get_answer(
        self,
        question: str,
        session_id: str = None,
        use_web_search: bool = False,
        max_context_docs: int = 4
    ) -> Dict[str, Any]:
        """获取问题答案 - 核心方法"""
        
        try:
            # 获取会话历史
            session = self.get_session(session_id) if session_id else None
            history = session.history if session else []
            
            # 检查缓存
            cached_response = self.cache_manager.get(question, session_id, history)
            if cached_response:
                # 添加到会话历史
                if session_id:
                    self.add_message_to_history(session_id, "user", question)
                    self.add_message_to_history(session_id, "assistant", cached_response["answer"])
                return cached_response
            
            # 添加用户消息到历史
            if session_id:
                self.add_message_to_history(session_id, "user", question)
            
            # 网络搜索（如果启用）
            web_results = []
            if use_web_search:
                web_results = await self.search_web(question)
                if web_results:
                    logger.info(f"网络搜索返回 {len(web_results)} 个结果")
            
            # 检索相关文档
            relevant_docs = await self._retrieve_relevant_docs(question, max_context_docs)
            
            # 合并网络搜索结果
            if web_results:
                web_docs = self._convert_web_results_to_docs(web_results)
                relevant_docs.extend(web_docs)
            
            # 如果没有相关文档，尝试重写问题
            if not relevant_docs:
                rewritten_question = await self._rewrite_question(question)
                if rewritten_question != question:
                    relevant_docs = await self._retrieve_relevant_docs(rewritten_question, max_context_docs)
            
            # 生成回答
            if relevant_docs:
                context_with_history = self._build_context_with_history(session)
                answer = await self._generate_answer(question, relevant_docs, context_with_history)
            else:
                answer = self._generate_fallback_answer()
            
            # 检查幻觉
            hallucination_free = await self._check_hallucination(question, answer, relevant_docs) if relevant_docs else False
            
            # 生成后续问题
            follow_up_questions = await self._generate_follow_up_questions(question, answer)
            
            # 格式化源文档
            source_docs = self._format_source_documents(relevant_docs)
            
            # 构建响应
            result = {
                "answer": answer,
                "source_documents": source_docs,
                "relevance_score": 1.0 if hallucination_free else 0.5,
                "recall_rate": len(relevant_docs) / max_context_docs if relevant_docs else 0.0,
                "follow_up_questions": follow_up_questions
            }
            
            # 添加助手回复到历史
            if session_id:
                self.add_message_to_history(session_id, "assistant", answer)
            
            # 缓存结果
            self.cache_manager.set(question, result, session_id, history)
            asyncio.create_task(self.cache_manager.save_async())
            
            return result
            
        except Exception as e:
            logger.error(f"获取回答失败: {e}")
            error_answer = "抱歉，处理您的问题时出现了错误，请稍后重试。"
            
            if session_id:
                self.add_message_to_history(session_id, "assistant", error_answer)
            
            return {
                "answer": error_answer,
                "source_documents": [],
                "relevance_score": 0.0,
                "recall_rate": 0.0,
                "follow_up_questions": ["AIOps平台有哪些核心功能？", "如何部署AIOps系统？"]
            }
    
    async def _retrieve_relevant_docs(self, question: str, max_docs: int) -> List[Document]:
        """检索相关文档"""
        try:
            # 检索文档
            docs = await asyncio.get_event_loop().run_in_executor(
                self.executor,
                self.vector_store_manager.search_documents,
                question
            )
            
            if not docs:
                return []
            
            # 过滤相关文档
            relevant_docs = await self._filter_relevant_docs(question, docs[:max_docs])
            
            return relevant_docs
            
        except Exception as e:
            logger.error(f"检索文档失败: {e}")
            return []
    
    async def _filter_relevant_docs(self, question: str, docs: List[Document]) -> List[Document]:
        """过滤相关文档"""
        if not docs or len(docs) <= 2:
            return docs
        
        try:
            relevant_docs = []
            
            for doc in docs:
                is_relevant, score = await self._evaluate_doc_relevance(question, doc)
                
                if is_relevant:
                    doc.metadata = doc.metadata or {}
                    doc.metadata["relevance_score"] = score
                    relevant_docs.append(doc)
            
            # 如果没有相关文档，返回前几个
            if not relevant_docs:
                return docs[:3]
            
            # 按相关性排序
            relevant_docs.sort(
                key=lambda x: x.metadata.get("relevance_score", 0),
                reverse=True
            )
            
            return relevant_docs
            
        except Exception as e:
            logger.error(f"过滤文档失败: {e}")
            return docs[:3]
    
    async def _evaluate_doc_relevance(self, question: str, doc: Document) -> Tuple[bool, float]:
        """评估文档相关性"""
        try:
            # 简单的关键词匹配
            question_words = set(question.lower().split())
            doc_words = set(doc.page_content.lower().split())
            
            # 计算重叠度
            overlap = len(question_words & doc_words)
            total = len(question_words | doc_words)
            similarity = overlap / total if total > 0 else 0
            
            # 基于相似度判断相关性
            is_relevant = similarity > 0.1
            score = min(similarity * 2, 1.0)  # 归一化到[0,1]
            
            return is_relevant, score
            
        except Exception as e:
            logger.error(f"评估文档相关性失败: {e}")
            return True, 0.5  # 默认相关
    
    def _convert_web_results_to_docs(self, web_results: List[Dict]) -> List[Document]:
        """将网络搜索结果转换为文档"""
        docs = []
        for result in web_results:
            content = f"标题: {result.get('title', '未知')}\n"
            content += f"来源: {result.get('url', '未知')}\n"
            content += f"内容: {result.get('content', '无内容')}"
            
            doc = Document(
                page_content=content,
                metadata={
                    "source": result.get('url', '网络搜索'),
                    "title": result.get('title', '网络搜索结果'),
                    "is_web_result": True,
                    "filetype": "web"
                }
            )
            docs.append(doc)
        
        return docs
    
    def _build_context_with_history(self, session: Optional[SessionData]) -> Optional[str]:
        """构建包含历史的上下文"""
        if not session or not session.history:
            return None
        
        # 获取最近的对话
        recent_history = session.history[-6:]  # 最近3轮对话
        if len(recent_history) < 2:
            return None
        
        context = "以下是之前的对话历史:\n"
        for msg in recent_history:
            role = "用户" if msg["role"] == "user" else "助手"
            context += f"{role}: {msg['content']}\n"
        
        return context + "\n"
    
    async def _rewrite_question(self, question: str) -> str:
        """重写问题以提高检索效果"""
        try:
            if len(question) < 10:
                return question
            
            system_prompt = """重写用户问题，使其更适合搜索。保持问题本意，只返回重写后的问题。"""
            
            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=f"问题: {question}")
            ]
            
            response = await asyncio.wait_for(
                self.task_llm.ainvoke(messages),
                timeout=5
            )
            
            rewritten = response.content.strip()
            return rewritten if rewritten != question else question
            
        except Exception as e:
            logger.warning(f"重写问题失败: {e}")
            return question
    
    async def _generate_answer(
        self,
        question: str,
        docs: List[Document],
        context_with_history: Optional[str] = None
    ) -> str:
        """生成回答"""
        try:
            # 构建文档内容
            docs_content = ""
            for i, doc in enumerate(docs):
                source = doc.metadata.get("source", "未知") if doc.metadata else "未知"
                filename = doc.metadata.get("filename", "未知文件") if doc.metadata else "未知文件"
                
                # 更详细的文档标识
                docs_content += f"\n\n文档[{i+1}] (文件: {filename}, 来源: {source}):\n{doc.page_content}"
            
            # 限制长度
            max_length = getattr(config.rag, 'max_context_length', 4000)
            if len(docs_content) > max_length:
                docs_content = docs_content[:max_length] + "...(内容已截断)"
            
            # 构建提示
            system_prompt = """您是专业的AI助手。请基于提供的文档内容回答用户问题。
            
规则:
1. 仅基于文档内容回答，不要编造信息
2. 回答要简洁清晰，直接解决问题
3. 如果文档信息不足，明确说明
4. 使用专业友好的语气
5. 语言与用户问题保持一致"""
            
            user_prompt = f"{context_with_history}\n\n" if context_with_history else ""
            user_prompt += f"问题: {question}\n\n文档内容:\n{docs_content}\n\n请回答问题："
            
            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=user_prompt)
            ]
            
            response = await asyncio.wait_for(
                self.llm.ainvoke(messages),
                timeout=30
            )
            
            return response.content.strip()
            
        except Exception as e:
            logger.error(f"生成回答失败: {e}")
            return "抱歉，生成回答时遇到问题，请稍后重试。"
    
    def _generate_fallback_answer(self) -> str:
        """生成后备回答"""
        fallback_answers = [
            "抱歉，我在知识库中没有找到相关信息。",
            "您的问题超出了我的知识范围，请尝试其他问题。",
            "我无法在当前知识库中找到相关内容。"
        ]
        import random
        return random.choice(fallback_answers)
    
    async def _check_hallucination(self, question: str, answer: str, docs: List[Document]) -> bool:
        """检查回答是否存在幻觉"""
        try:
            if len(answer) < 80 or not docs:
                return True
            
            # 简单检查 - 基于关键词匹配
            answer_words = set(answer.lower().split())
            doc_words = set()
            
            for doc in docs:
                doc_words.update(doc.page_content.lower().split())
            
            # 计算回答中有多少词汇来自文档
            common_words = answer_words & doc_words
            coverage = len(common_words) / len(answer_words) if answer_words else 0
            
            # 如果覆盖率较高，认为没有幻觉
            return coverage > 0.3
            
        except Exception as e:
            logger.error(f"幻觉检查失败: {e}")
            return True  # 默认通过
    
    async def _generate_follow_up_questions(self, question: str, answer: str) -> List[str]:
        """生成后续问题"""
        default_questions = [
            "AIOps平台有哪些核心功能？",
            "如何部署和配置AIOps系统？",
            "AIOps如何帮助解决运维问题？"
        ]
        
        try:
            if len(answer) < 100:
                return default_questions[:3]
            
            system_prompt = """生成3个与当前话题相关的后续问题，每行一个，以问号结尾。"""
            
            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=f"原问题: {question}\n回答: {answer[:300]}")
            ]
            
            response = await asyncio.wait_for(
                self.task_llm.ainvoke(messages),
                timeout=5
            )
            
            # 解析问题
            questions = []
            for line in response.content.strip().split("\n"):
                line = re.sub(r"^\d+[\.\)、]\s*", "", line.strip())
                if line and (line.endswith("?") or line.endswith("？")):
                    questions.append(line)
                elif len(line) > 10:
                    questions.append(line + "?")
            
            return questions[:3] if len(questions) >= 2 else default_questions[:3]
            
        except Exception as e:
            logger.error(f"生成后续问题失败: {e}")
            return default_questions[:3]
    
    def _format_source_documents(self, docs: List[Document]) -> List[Dict[str, Any]]:
        """格式化源文档"""
        source_docs = []
        
        for doc in docs:
            metadata = doc.metadata or {}
            content = doc.page_content
            
            # 截断长内容
            if len(content) > 200:
                content = content[:200] + "..."
            
            source_docs.append({
                "content": content,
                "source": metadata.get("source", "未知来源"),
                "is_web_result": metadata.get("is_web_result", False),
                "metadata": metadata
            })
        
        return source_docs
    
    def __del__(self):
        """清理资源"""
        try:
            if hasattr(self, 'executor'):
                self.executor.shutdown(wait=False)
        except:
            pass