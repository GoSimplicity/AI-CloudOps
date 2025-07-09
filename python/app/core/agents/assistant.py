"""
智能小助手模块 - 基于RAG的知识检索与问答
"""

import os
import sys
import uuid
import logging
import re
import time
import json
import asyncio
from datetime import datetime
from typing import List, Dict, Any, Optional, Union, Tuple
from pathlib import Path
import hashlib

# 从langchain_chroma导入Chroma，替代langchain_community中的版本
from langchain_chroma import Chroma
from langchain_openai import OpenAIEmbeddings
from langchain_ollama import OllamaEmbeddings
from langchain.text_splitter import RecursiveCharacterTextSplitter, MarkdownHeaderTextSplitter
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.pydantic_v1 import BaseModel, Field
from langchain_openai import ChatOpenAI
from langchain_ollama import ChatOllama
from langchain_core.output_parsers import StrOutputParser, JsonOutputParser
from langchain_core.documents import Document
from langchain_core.messages import HumanMessage, AIMessage, SystemMessage

try:
    from langchain_community.document_loaders import (
        TextLoader, 
        PyPDFLoader,
        DirectoryLoader,
        UnstructuredMarkdownLoader,
        CSVLoader,
        JSONLoader,
        BSHTMLLoader
    )
    # 尝试导入网络搜索工具
    try:
        from langchain_community.utilities import TavilySearchAPIWrapper
        WEB_SEARCH_AVAILABLE = True
    except ImportError:
        WEB_SEARCH_AVAILABLE = False
        logger = logging.getLogger("aiops.assistant")
        logger.warning("网络搜索功能不可用")
        
    ADVANCED_LOADERS_AVAILABLE = True
except ImportError:
    ADVANCED_LOADERS_AVAILABLE = False
    WEB_SEARCH_AVAILABLE = False
    logger = logging.getLogger("aiops.assistant")
    logger.warning("高级文档加载器不可用，将使用基础实现")

from app.config.settings import config

logger = logging.getLogger("aiops.assistant")

class GradeDocuments(BaseModel):
    """评价检索到的文档是否与用户问题相关的二元评分"""
    binary_score: str = Field(description="文档是否与问题相关，'yes'或'no'")

class GradeHallucinations(BaseModel):
    """评估生成回答中是否存在幻觉的二元评分"""
    binary_score: str = Field(description="回答是否基于事实，'yes'或'no'")
    

class AssistantAgent:
    """智能小助手代理，基于RAG实现知识库检索与问答"""
    
    def __init__(self):
        """初始化助手代理"""
        self.llm_provider = config.llm.provider.lower()
        
        # 使用绝对路径
        base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
        self.vector_db_path = os.path.join(base_dir, config.rag.vector_db_path)
        self.collection_name = config.rag.collection_name
        self.knowledge_base_path = os.path.join(base_dir, config.rag.knowledge_base_path)
        
        logger.info(f"向量数据库路径: {self.vector_db_path}")
        logger.info(f"知识库路径: {self.knowledge_base_path}")
        
        self.retriever = None
        self.llm = None
        self.structured_llm_docs = None
        self.structured_llm_hall = None
        self.embedding = None
        
        # 缓存设置
        self.cache_dir = os.path.join(self.vector_db_path, "cache")
        self.response_cache = {}
        self.cache_expiry = 3600  # 缓存过期时间(秒)
        os.makedirs(self.cache_dir, exist_ok=True)
        
        # 会话管理
        self.sessions = {}  # session_id -> 会话数据
        
        # 初始化向量存储路径
        os.makedirs(self.vector_db_path, exist_ok=True)
        os.makedirs(self.knowledge_base_path, exist_ok=True)
        
        # 初始化组件
        self._init_embedding()
        self._init_llm()
        self._init_retriever()
        self._init_web_search()
        
        # 加载缓存
        self._load_cache()
        
        logger.info(f"智能小助手初始化完成，使用LLM提供商: {self.llm_provider}")
        
    def _init_embedding(self) -> None:
        """初始化嵌入模型"""
        # 检查环境变量和配置
        if not config.llm.api_key:
            logger.warning("API密钥未设置，请检查环境变量")
        
        # 最多尝试3次
        max_retries = 3
        for attempt in range(max_retries):
            try:
                if self.llm_provider == 'openai':
                    logger.info(f"尝试初始化OpenAI嵌入模型 (尝试 {attempt+1}/{max_retries})")
                    
                    # 添加超时设置，防止长时间阻塞
                    timeout_kwargs = {"timeout": 10}
                    
                    self.embedding = OpenAIEmbeddings(
                        model=config.rag.openai_embedding_model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url,
                        **timeout_kwargs
                    )
                else:
                    logger.info(f"尝试初始化Ollama嵌入模型 (尝试 {attempt+1}/{max_retries})")
                    self.embedding = OllamaEmbeddings(
                        model=config.rag.ollama_embedding_model,
                        base_url=config.llm.ollama_base_url
                    )
                    
                # 测试嵌入模型
                test_embedding = self.embedding.embed_query("测试嵌入功能")
                if test_embedding and len(test_embedding) > 0:
                    logger.info(f"嵌入模型测试成功: 维度={len(test_embedding)}")
                    return  # 成功初始化，直接返回
                else:
                    logger.warning("嵌入模型测试结果异常，将尝试备用方法")
                    continue
                    
            except Exception as e:
                logger.error(f"嵌入模型初始化尝试 {attempt+1}/{max_retries} 失败: {e}")
                # 如果不是最后一次尝试，则切换提供商并重试
                if attempt < max_retries - 1:
                    self.llm_provider = 'ollama' if self.llm_provider == 'openai' else 'openai'
                    logger.info(f"切换到 {self.llm_provider} 提供商并重试")
                    # 短暂延迟后重试
                    import time
                    time.sleep(1)
        
        # 如果所有尝试都失败，创建一个简单的备用嵌入模型
        logger.critical("无法初始化任何嵌入模型，使用备用简单嵌入")
        
        # 创建一个简单的备用嵌入实现
        from langchain_core.embeddings import Embeddings
        import numpy as np
        
        class FallbackEmbeddings(Embeddings):
            """备用嵌入实现，返回简单的随机向量，仅用于应急"""
            
            def embed_documents(self, texts):
                return [self.embed_query(text) for text in texts]
                
            def embed_query(self, text):
                # 使用文本长度和内容生成确定性向量
                np.random.seed(hash(text) % 2**32)
                return list(np.random.rand(384))  # 使用384维向量
        
        self.embedding = FallbackEmbeddings()
        logger.warning("使用备用嵌入实现，搜索精度将受到影响")
    
    
    def _init_llm(self) -> None:
        """初始化语言模型"""
        # 最多尝试3次
        max_retries = 3
        for attempt in range(max_retries):
            try:
                if self.llm_provider == 'openai':
                    # 对话模型
                    chat_model = config.llm.model
                    logger.info(f"尝试初始化OpenAI语言模型 {chat_model} (尝试 {attempt+1}/{max_retries})")
                    
                    # 添加更多容错参数
                    self.llm = ChatOpenAI(
                        model=chat_model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url,
                        temperature=config.rag.temperature,
                        timeout=30,
                        max_retries=2,  # API级别的重试
                        request_timeout=30  # 请求超时
                    )
                else:
                    logger.info(f"尝试初始化Ollama语言模型 {config.llm.ollama_model} (尝试 {attempt+1}/{max_retries})")
                    self.llm = ChatOllama(
                        model=config.llm.ollama_model,
                        base_url=config.llm.ollama_base_url,
                        temperature=config.rag.temperature,
                        timeout=30
                    )
                
                # 测试LLM
                logger.info("测试语言模型...")
                test_response = self.llm.invoke("返回'测试成功'两个字")
                test_content = test_response.content if hasattr(test_response, 'content') else str(test_response)
                
                if test_content and len(test_content) > 0:
                    logger.info(f"LLM测试成功: {test_content[:20]}...")
                    
                    # 初始化JSON解析器
                    self.json_parser = JsonOutputParser()
                    return  # 成功初始化，直接返回
                else:
                    logger.warning("LLM测试结果异常，将尝试备用方法")
                
            except Exception as e:
                logger.error(f"语言模型初始化尝试 {attempt+1}/{max_retries} 失败: {e}")
                
                # 如果不是最后一次尝试，则切换提供商并重试
                if attempt < max_retries - 1:
                    self.llm_provider = 'ollama' if self.llm_provider == 'openai' else 'openai'
                    logger.info(f"切换到 {self.llm_provider} 提供商并重试")
                    # 短暂延迟后重试
                    import time
                    time.sleep(1)
            
        # 所有尝试都失败，使用备用模型策略
        logger.critical("所有语言模型初始化尝试均失败，使用备用策略")
        
        # 创建一个简单的备用语言模型
        from langchain_core.language_models.chat_models import BaseChatModel
        from langchain_core.outputs import ChatGeneration, ChatResult
        from langchain_core.messages import AIMessage
        
        class FallbackChatModel(BaseChatModel):
            """备用聊天模型，返回预定义的回复"""
            
            @property
            def _llm_type(self) -> str:
                """返回模型类型"""
                return "fallback_chat_model"
            
            def _generate(self, messages, stop=None, run_manager=None, **kwargs):
                # 创建一个简单的回复
                message = messages[-1].content if messages else "无输入"
                response = f"我是备用助手模型。您的问题是：'{message}'。由于语言模型暂时不可用，我只能提供有限的帮助。请稍后再试或联系系统管理员。"
                message = AIMessage(content=response)
                generation = ChatGeneration(message=message)
                return ChatResult(generations=[generation])
        
        self.llm = FallbackChatModel()
        self.json_parser = JsonOutputParser()
        logger.warning("使用备用语言模型，功能将受到限制")
            
    # 此方法已移除，不再使用模拟LLM
        
    def _generate_cache_key(self, question: str, session_id: str = None) -> str:
        """生成缓存键"""
        # 组合问题和会话ID (如果有)
        cache_input = question
        if session_id and session_id in self.sessions:
            # 获取最近的3条历史记录用于缓存键计算
            history = self.sessions[session_id]["history"][-3:] if self.sessions[session_id]["history"] else []
            if history:
                history_str = json.dumps([{
                    "role": h["role"],
                    "content": h["content"][:50]  # 只使用前50个字符减小缓存键大小
                } for h in history], ensure_ascii=False)
                cache_input = f"{question}|{history_str}"
        
        # 使用SHA256哈希作为缓存键
        return hashlib.sha256(cache_input.encode('utf-8')).hexdigest()
    
    def _load_cache(self) -> None:
        """加载缓存文件"""
        cache_file = os.path.join(self.cache_dir, "response_cache.json")
        try:
            if os.path.exists(cache_file):
                with open(cache_file, 'r', encoding='utf-8') as f:
                    cache_data = json.load(f)
                    # 过滤过期缓存
                    now = time.time()
                    self.response_cache = {
                        k: v for k, v in cache_data.items() 
                        if now - v.get("timestamp", 0) < self.cache_expiry
                    }
                logger.info(f"已加载 {len(self.response_cache)} 条有效缓存")
        except Exception as e:
            logger.warning(f"加载缓存失败: {e}")
            self.response_cache = {}
    
    def _save_cache(self) -> None:
        """保存缓存到文件"""
        cache_file = os.path.join(self.cache_dir, "response_cache.json")
        try:
            # 过滤过期缓存
            now = time.time()
            valid_cache = {
                k: v for k, v in self.response_cache.items() 
                if now - v.get("timestamp", 0) < self.cache_expiry
            }
            
            with open(cache_file, 'w', encoding='utf-8') as f:
                json.dump(valid_cache, f, ensure_ascii=False)
            
            logger.info(f"已保存 {len(valid_cache)} 条缓存")
        except Exception as e:
            logger.warning(f"保存缓存失败: {e}")
            
    def create_session(self) -> str:
        """创建新会话"""
        session_id = str(uuid.uuid4())
        self.sessions[session_id] = {
            "created_at": datetime.now().isoformat(),
            "history": [],
            "metadata": {}
        }
        return session_id
    
    def get_session(self, session_id: str) -> Dict:
        """获取会话信息"""
        if session_id not in self.sessions:
            return None
        return self.sessions[session_id]
    
    def add_message_to_history(self, session_id: str, role: str, content: str) -> str:
        """添加消息到会话历史"""
        if session_id not in self.sessions:
            session_id = self.create_session()
            
        self.sessions[session_id]["history"].append({
            "role": role,
            "content": content,
            "timestamp": datetime.now().isoformat()
        })
        
        # 限制历史记录长度
        max_history = 20
        if len(self.sessions[session_id]["history"]) > max_history:
            self.sessions[session_id]["history"] = self.sessions[session_id]["history"][-max_history:]
        
        return session_id
        
    def clear_session_history(self, session_id: str) -> bool:
        """清除会话历史"""
        if session_id in self.sessions:
            self.sessions[session_id]["history"] = []
            return True
        return False
        
    async def search_web(self, query: str, max_results: int = None) -> List[Dict]:
        """从网络搜索相关信息"""
        if not hasattr(self, 'web_search') or self.web_search is None:
            logger.warning("网络搜索功能未启用")
            return []
            
        try:
            max_results = max_results or config.tavily.max_results
            results = self.web_search.results(query, max_results=max_results)
            return results
        except Exception as e:
            logger.error(f"网络搜索失败: {e}")
            return []
    
    def _init_web_search(self) -> None:
        """初始化网络搜索工具"""
        self.web_search = None
        if 'WEB_SEARCH_AVAILABLE' in globals() and WEB_SEARCH_AVAILABLE and config.tavily.api_key:
            try:
                self.web_search = TavilySearchAPIWrapper(
                    api_key=config.tavily.api_key,
                    max_results=config.tavily.max_results
                )
                logger.info("网络搜索工具初始化成功")
            except Exception as e:
                logger.warning(f"初始化网络搜索工具失败: {e}")
        else:
            logger.info("网络搜索功能未启用")
    
    def _init_retriever(self) -> None:
        """初始化检索器"""
        try:
            # 检查是否已存在向量数据库
            if os.path.exists(os.path.join(self.vector_db_path, "chroma.sqlite3")):
                logger.info("检测到现有向量数据库，正在加载...")
                db = Chroma(
                    persist_directory=self.vector_db_path,
                    embedding_function=self.embedding,
                    collection_name=self.collection_name
                )
                self.retriever = db.as_retriever(
                    search_kwargs={"k": config.rag.top_k}
                )
                logger.info("向量数据库加载完成")
            else:
                logger.info("未检测到向量数据库，正在扫描知识库文档...")
                self._create_vector_store()
        except Exception as e:
            logger.error(f"初始化检索器失败: {e}")
            raise RuntimeError(f"无法初始化检索器: {e}")
    
    def _create_vector_store(self, use_in_memory: bool = False) -> None:
        """创建向量数据库
        
        参数:
            use_in_memory: 是否使用内存模式（适用于测试环境）
        """
        try:
            # 确保向量数据库目录存在且具有写入权限
            os.makedirs(self.vector_db_path, exist_ok=True)
            
            # 检查写入权限
            if not os.access(self.vector_db_path, os.W_OK):
                try:
                    # 尝试修复权限
                    import stat
                    os.chmod(self.vector_db_path, stat.S_IRWXU | stat.S_IRWXG | stat.S_IRWXO)
                    logger.info(f"已修复向量数据库目录权限: {self.vector_db_path}")
                except Exception as perm_error:
                    logger.warning(f"无法修复向量数据库目录权限: {perm_error}")
                    # 使用内存模式作为备用方案
                    logger.info("由于权限问题，切换到内存模式")
                    use_in_memory = True
            
            # 加载知识库文档
            documents = self._load_documents()
            if not documents:
                logger.warning("没有找到知识库文档，无法创建向量数据库")
                return
            
            # 分割文档
            text_splitter = RecursiveCharacterTextSplitter(
                chunk_size=config.rag.chunk_size,
                chunk_overlap=config.rag.chunk_overlap
            )
            splits = text_splitter.split_documents(documents)
            
            # 根据模式创建不同的向量存储
            if use_in_memory:
                # 内存模式 - 不持久化到磁盘
                db = Chroma.from_documents(
                    documents=splits,
                    embedding=self.embedding,
                    collection_name=self.collection_name
                )
            else:
                # 持久化模式 - 将向量存储到磁盘
                try:
                    db = Chroma.from_documents(
                        documents=splits,
                        embedding=self.embedding,
                        persist_directory=self.vector_db_path,
                        collection_name=self.collection_name
                    )
                except Exception as db_error:
                    if "readonly database" in str(db_error).lower():
                        logger.warning("数据库为只读状态，切换到内存模式")
                        db = Chroma.from_documents(
                            documents=splits,
                            embedding=self.embedding,
                            collection_name=self.collection_name
                        )
                    else:
                        raise
            
            # 创建检索器
            self.retriever = db.as_retriever(
                search_kwargs={"k": config.rag.top_k}
            )
            
            logger.info(f"向量数据库创建完成，包含 {len(splits)} 个文档块")
        
        except Exception as e:
            logger.error(f"创建向量数据库失败: {e}")
            raise RuntimeError(f"无法创建向量数据库: {e}")
    
    def _load_documents(self) -> List[Document]:
        """加载知识库文档"""
        documents = []
        
        # 检查知识库目录是否存在
        kb_path = Path(self.knowledge_base_path)
        if not kb_path.exists():
            logger.warning(f"知识库路径 {kb_path} 不存在")
            os.makedirs(kb_path, exist_ok=True)
            logger.info(f"已创建知识库目录: {kb_path}")
            return documents
            
        # 获取所有文件并按修改时间排序，优先处理新文件
        all_files = list(kb_path.glob("**/*"))
        all_files = [f for f in all_files if f.is_file()]
        # 按修改时间排序
        all_files.sort(key=lambda x: os.path.getmtime(x), reverse=True)
        
        logger.info(f"知识库目录下的文件: {len(all_files)}个")
        
        try:
            # 1. 加载TXT文件
            txt_files = [f for f in all_files if f.suffix.lower() == '.txt']
            logger.info(f"找到的TXT文件: {len(txt_files)}个")
            for file_path in txt_files:
                try:
                    with open(file_path, "r", encoding="utf-8") as f:
                        content = f.read()
                        # 检查文件内容是否为空
                        if not content.strip():
                            logger.warning(f"跳过空文件: {file_path}")
                            continue
                        
                        documents.append(Document(
                            page_content=content,
                            metadata={
                                "source": str(file_path),
                                "filename": file_path.name,
                                "filetype": "text",
                                "modified_time": os.path.getmtime(file_path)
                            }
                        ))
                        logger.info(f"成功加载TXT文件: {file_path}")
                except Exception as file_error:
                    logger.error(f"加载文件 {file_path} 失败: {file_error}")
            
            # 2. 加载MD文件 - 使用Markdown分割器优化分块
            md_files = [f for f in all_files if f.suffix.lower() in ['.md', '.markdown']]
            logger.info(f"找到的MD文件: {len(md_files)}个")
            
            # 使用Markdown特定的分割器
            headers_to_split_on = [
                ("#", "Header 1"),
                ("##", "Header 2"),
                ("###", "Header 3"),
            ]
            
            for file_path in md_files:
                try:
                    with open(file_path, "r", encoding="utf-8") as f:
                        content = f.read()
                        # 检查文件内容是否为空
                        if not content.strip():
                            logger.warning(f"跳过空文件: {file_path}")
                            continue
                        
                        # 尝试使用Markdown标题分割文本
                        try:
                            markdown_splitter = MarkdownHeaderTextSplitter(headers_to_split_on=headers_to_split_on)
                            md_docs = markdown_splitter.split_text(content)
                            
                            # 添加元数据
                            for doc in md_docs:
                                doc.metadata.update({
                                    "source": str(file_path),
                                    "filename": file_path.name,
                                    "filetype": "markdown",
                                    "modified_time": os.path.getmtime(file_path)
                                })
                            
                            documents.extend(md_docs)
                            logger.info(f"成功使用Markdown标题分割文件: {file_path}, 生成了{len(md_docs)}个文档块")
                        except Exception as split_error:
                            # 如果分割失败，作为单个文档加载
                            logger.warning(f"Markdown分割失败，以整个文件加载: {file_path}, 错误: {split_error}")
                            documents.append(Document(
                                page_content=content,
                                metadata={
                                    "source": str(file_path),
                                    "filename": file_path.name,
                                    "filetype": "markdown",
                                    "modified_time": os.path.getmtime(file_path)
                                }
                            ))
                except Exception as file_error:
                    logger.error(f"加载文件 {file_path} 失败: {file_error}")
            
            # 3. 使用可能的高级加载器处理其他文件
            if 'ADVANCED_LOADERS_AVAILABLE' in globals() and ADVANCED_LOADERS_AVAILABLE:
                try:
                    # 尝试加载PDF文件
                    pdf_files = [f for f in all_files if f.suffix.lower() == '.pdf']
                    if pdf_files:
                        logger.info(f"找到的PDF文件: {len(pdf_files)}个")
                        for pdf_file in pdf_files:
                            try:
                                loader = PyPDFLoader(str(pdf_file))
                                pdf_docs = loader.load()
                                
                                # 添加元数据
                                for doc in pdf_docs:
                                    doc.metadata.update({
                                        "filename": pdf_file.name,
                                        "filetype": "pdf",
                                        "modified_time": os.path.getmtime(pdf_file)
                                    })
                                    
                                documents.extend(pdf_docs)
                                logger.info(f"成功加载PDF文件: {pdf_file}, 生成了{len(pdf_docs)}个文档块")
                            except Exception as e:
                                logger.error(f"加载PDF文件失败: {pdf_file}, 错误: {e}")
                    
                    # 尝试加载HTML文件
                    html_files = [f for f in all_files if f.suffix.lower() in ['.html', '.htm']]
                    if html_files:
                        logger.info(f"找到的HTML文件: {len(html_files)}个")
                        for html_file in html_files:
                            try:
                                loader = BSHTMLLoader(str(html_file))
                                html_docs = loader.load()
                                
                                # 添加元数据
                                for doc in html_docs:
                                    doc.metadata.update({
                                        "filename": html_file.name,
                                        "filetype": "html",
                                        "modified_time": os.path.getmtime(html_file)
                                    })
                                    
                                documents.extend(html_docs)
                                logger.info(f"成功加载HTML文件: {html_file}, 生成了{len(html_docs)}个文档块")
                            except Exception as e:
                                logger.error(f"加载HTML文件失败: {html_file}, 错误: {e}")
                except Exception as e:
                    logger.warning(f"尝试使用高级加载器时出错: {e}")
            
            logger.info(f"成功加载 {len(documents)} 个文档")
            return documents
            
        except Exception as e:
            logger.error(f"加载文档失败: {e}")
            return documents
    
    async def refresh_knowledge_base(self) -> Dict[str, Any]:
        """刷新知识库，返回包含文档数量的字典"""
        try:
            logger.info("正在刷新知识库...")
            
            # 清理缓存以确保新添加的文档能被检索到
            self.response_cache = {}
            
            # 检查是否在测试环境中运行
            in_test_environment = 'pytest' in sys.modules
            
            # 确保知识库路径存在
            kb_path = Path(self.knowledge_base_path)
            if not kb_path.exists():
                os.makedirs(kb_path, exist_ok=True)
                logger.info(f"创建知识库目录: {kb_path}")
                # 创建一个简单的测试文档，确保知识库不为空
                test_doc_path = kb_path / "welcome.md"
                if not test_doc_path.exists():
                    with open(test_doc_path, "w", encoding="utf-8") as f:
                        f.write("# 欢迎使用AIOps智能小助手\n\n这是一个自动创建的初始文档，您可以添加更多文档到知识库中。")
                    logger.info("已创建欢迎文档")
            
            # 检查向量数据库目录
            db_path = Path(self.vector_db_path)
            
            # 检查向量数据库目录权限和状态
            use_in_memory = in_test_environment
            if not db_path.exists():
                try:
                    os.makedirs(db_path, exist_ok=True)
                except Exception as e:
                    logger.warning(f"创建向量数据库目录失败: {e}，将使用内存模式")
                    use_in_memory = True
            
            # 检查写入权限
            if not use_in_memory and not os.access(str(db_path), os.W_OK):
                logger.warning(f"向量数据库目录没有写入权限: {db_path}，将使用内存模式")
                use_in_memory = True
            
            # 准备刷新向量库
            success = False
            doc_count = 0
            error_msg = None
            
            # 最多尝试3次
            max_attempts = 3
            for attempt in range(max_attempts):
                try:
                    # 如果使用内存模式
                    if use_in_memory:
                        logger.info(f"使用内存模式刷新知识库 (尝试 {attempt+1}/{max_attempts})")
                        self._create_vector_store(use_in_memory=True)
                    else:
                        # 正常环境 - 清理现有文件并创建新数据库
                        logger.info(f"使用持久化模式刷新知识库 (尝试 {attempt+1}/{max_attempts})")
                        # 先创建备份目录
                        backup_dir = os.path.join(self.vector_db_path, f"backup_{int(time.time())}")
                        try:
                            os.makedirs(backup_dir, exist_ok=True)
                            logger.info(f"已创建备份目录: {backup_dir}")
                            
                            # 备份重要文件
                            import shutil
                            for item in db_path.glob("*"):
                                if item.is_file() and item.name not in ["chroma.sqlite3-shm", "chroma.sqlite3-wal"]:
                                    shutil.copy2(item, backup_dir)
                                    logger.info(f"已备份文件: {item.name}")
                            
                            # 清理现有的向量数据库文件
                            for item in db_path.glob("*"):
                                if item.is_file():
                                    # 临时文件保留，避免冲突
                                    if item.name in ["chroma.sqlite3-shm", "chroma.sqlite3-wal"]:
                                        continue
                                    item.unlink()
                                elif item.is_dir() and item.name not in ["cache", "backup_*"]:  # 保留缓存和备份目录
                                    shutil.rmtree(item)
                            logger.info("已清理现有向量数据库文件")
                            
                            # 创建新的向量数据库
                            self._create_vector_store(use_in_memory=False)
                            
                        except Exception as clean_error:
                            logger.warning(f"清理或创建向量数据库失败: {clean_error}，尝试使用内存模式")
                            self._create_vector_store(use_in_memory=True)
                    
                    # 测试检索器是否可用
                    if self.retriever:
                        try:
                            test_result = self.retriever.invoke("测试查询")
                            doc_count = len(test_result)
                            logger.info(f"检索器测试成功: 返回 {doc_count} 个结果")
                            success = True
                            break  # 成功，跳出尝试循环
                        except Exception as test_error:
                            logger.warning(f"检索器测试失败 (尝试 {attempt+1}/{max_attempts}): {test_error}")
                            error_msg = str(test_error)
                            if attempt < max_attempts - 1:
                                logger.info(f"等待1秒后重试...")
                                await asyncio.sleep(1)
                    else:
                        logger.warning(f"检索器未初始化 (尝试 {attempt+1}/{max_attempts})")
                        if attempt < max_attempts - 1:
                            logger.info(f"尝试重新初始化...")
                            try:
                                self._init_retriever()
                            except:
                                pass
                
                except Exception as e:
                    logger.error(f"刷新知识库尝试 {attempt+1}/{max_attempts} 失败: {e}")
                    error_msg = str(e)
                    if attempt < max_attempts - 1:
                        logger.info(f"等待1秒后重试...")
                        await asyncio.sleep(1)
            
            # 保存更新的空缓存
            self._save_cache()
            logger.info("已清理响应缓存")
            
            if success:
                return {"success": True, "documents_count": doc_count}
            else:
                return {"success": False, "documents_count": 0, "error": error_msg or "未知错误"}
                
        except Exception as e:
            logger.error(f"刷新知识库失败: {e}")
            return {"success": False, "documents_count": 0, "error": str(e)}
    
    def add_document(self, content: str, metadata: Dict[str, Any] = None) -> bool:
        """向知识库添加文档"""
        try:
            if not content.strip():
                return False
            
            # 生成唯一文件名
            doc_id = str(uuid.uuid4())
            file_path = os.path.join(self.knowledge_base_path, f"{doc_id}.txt")
            
            # 写入文件
            with open(file_path, "w", encoding="utf-8") as f:
                f.write(content)
            
            # 清理缓存以确保新添加的文档能被检索到
            self.response_cache = {}
            logger.info("已清理响应缓存以支持新添加的文档")
            
            # 刷新知识库操作移至API层异步执行，这里仅返回成功
            return True
        
        except Exception as e:
            logger.error(f"添加文档失败: {e}")
            return False
    
    async def get_answer(self, question: str, session_id: str = None, 
                     use_web_search: bool = False, max_context_docs: int = 4,
                     max_retries: int = 3) -> Dict[str, Any]:
        """获取问题的回答
        
        参数:
            question: 用户问题
            session_id: 会话ID，用于保持对话上下文
            use_web_search: 是否使用网络搜索增强回答
            max_context_docs: 最大上下文文档数量
            max_retries: 最大重试次数
        """
        try:
            # 检查缓存
            if session_id is None:
                cache_key = self._generate_cache_key(question)
            else:
                cache_key = self._generate_cache_key(question, session_id)
                
            # 尝试从缓存获取
            if cache_key in self.response_cache:
                cache_data = self.response_cache[cache_key]
                # 检查缓存是否过期
                if time.time() - cache_data.get("timestamp", 0) < self.cache_expiry:
                    logger.info(f"从缓存中获取回答: {cache_key[:8]}...")
                    # 添加到会话历史(如果有)
                    if session_id:
                        self.add_message_to_history(session_id, "user", question)
                        self.add_message_to_history(session_id, "assistant", cache_data["data"]["answer"])
                    return cache_data["data"]
            
            # 添加到会话历史
            if session_id:
                self.add_message_to_history(session_id, "user", question)
            
            # 如果启用了网络搜索
            web_results = []
            if use_web_search and hasattr(self, 'web_search') and self.web_search:
                logger.info(f"为问题 '{question}' 执行网络搜索")
                web_results = await self.search_web(question)
                if web_results:
                    logger.info(f"网络搜索返回了 {len(web_results)} 个结果")
            
            # 1. 检索相关文档
            docs = self._get_relevant_docs(question)
            
            # 合并网络搜索结果
            if web_results:
                for result in web_results:
                    content = f"标题: {result.get('title', '未知标题')}\n"
                    content += f"来源: {result.get('url', '未知来源')}\n"
                    content += f"内容:\n{result.get('content', '无内容')}"
                    
                    web_doc = Document(
                        page_content=content,
                        metadata={
                            "source": result.get('url', '网络搜索'),
                            "title": result.get('title', '网络搜索结果'),
                            "is_web_result": True
                        }
                    )
                    docs.append(web_doc)
            
            # 2. 评估文档相关性
            relevant_docs = await self._filter_relevant_docs(question, docs)
            
            # 3. 如果没有相关文档，尝试重写问题
            if not relevant_docs:
                logger.info("没有找到相关文档，尝试重写问题...")
                try:
                    rewritten_question = await self._rewrite_question(question)
                    if rewritten_question and rewritten_question != question:
                                            docs = self._get_relevant_docs(rewritten_question)
                    relevant_docs = await self._filter_relevant_docs(rewritten_question, docs)
                except Exception as rewrite_error:
                    logger.error(f"重写问题时出错: {rewrite_error}")
            
            # 限制上下文文档数量
            if relevant_docs and len(relevant_docs) > max_context_docs:
                logger.info(f"限制文档数量从 {len(relevant_docs)} 到 {max_context_docs}")
                relevant_docs = relevant_docs[:max_context_docs]
            
            # 使用会话历史增强上下文
            context_with_history = ""
            if session_id and session_id in self.sessions:
                history = self.sessions[session_id]["history"]
                if history and len(history) >= 2:  # 至少有一轮对话
                    # 选择最近的2-3轮对话
                    recent_history = history[-min(6, len(history)):]
                    context_with_history = "以下是之前的对话历史:\n"
                    for h in recent_history:
                        role = "用户" if h["role"] == "user" else "助手"
                        context_with_history += f"{role}: {h['content']}\n"
                    context_with_history += "\n"
            
            # 4. 生成回答
            if relevant_docs:
                try:
                    answer = await self._generate_from_docs(
                        question, 
                        relevant_docs,
                        context_with_history=context_with_history if context_with_history else None
                    )
                except Exception as generate_error:
                    logger.error(f"生成回答时出错: {generate_error}")
                    answer = "抱歉，生成回答时出现了错误。请稍后再试。"
            else:
                # 返回通用回答
                possible_answers = [
                    "抱歉，我无法在知识库中找到与您问题相关的信息。",
                    "我在知识库中没有找到相关内容，请尝试其他问题。",
                    "您的问题超出了我的知识范围，我无法提供准确答案。"
                ]
                import random
                answer = random.choice(possible_answers)
            
            # 5. 评估回答是否存在幻觉
            hallucination_free = False
            if relevant_docs:
                try:
                    hallucination_free = await self._check_hallucination(
                        question, answer, relevant_docs
                    )
                except Exception as hall_error:
                    logger.error(f"检查回答幻觉时出错: {hall_error}")
            
            # 6. 格式化源文档
            source_docs = []
            for doc in relevant_docs:
                metadata = doc.metadata.copy() if doc.metadata else {}
                # 检查是否为网络搜索结果
                is_web_result = metadata.get("is_web_result", False)
                
                source_docs.append({
                    "content": doc.page_content[:200] + "..." if len(doc.page_content) > 200 else doc.page_content,
                    "source": metadata.get("source", "未知来源"),
                    "is_web_result": is_web_result,
                    "metadata": metadata
                })
            
            # 7. 生成后续问题
            follow_up_questions = []
            try:
                follow_up_questions = await self._generate_follow_up_questions(question, answer)
            except Exception as follow_up_error:
                logger.error(f"生成后续问题时出错: {follow_up_error}")
                # 提供一些默认的后续问题
                follow_up_questions = [
                    "AIOps平台有哪些核心功能？",
                    "如何启动AIOps平台？",
                    "AIOps平台的部署要求是什么？"
                ]
            
            # 从文档元数据中提取召回率
            recall_rate = None
            for doc in relevant_docs:
                if doc.metadata and "recall_rate" in doc.metadata:
                    recall_rate = doc.metadata.get("recall_rate")
                    break
            
            result = {
                "answer": answer,
                "source_documents": source_docs,
                "relevance_score": 1.0 if hallucination_free else 0.5,
                "recall_rate": recall_rate if recall_rate is not None else 0.0,
                "follow_up_questions": follow_up_questions
            }
            
            # 添加到会话历史
            if session_id:
                self.add_message_to_history(session_id, "assistant", answer)
                
            # 添加到缓存
            self.response_cache[cache_key] = {
                "timestamp": time.time(),
                "data": result
            }
            
            # 异步保存缓存
            asyncio.create_task(self._async_save_cache())
                
            return result
        
        except Exception as e:
            logger.error(f"获取回答失败: {e}")
            error_message = "抱歉，处理您的问题时出现了错误。请检查API密钥和网络连接后重试。"
            
            # 添加到会话历史(如果有)
            if session_id:
                self.add_message_to_history(session_id, "assistant", error_message)
                
            return {
                "answer": error_message,
                "source_documents": [],
                "relevance_score": 0.0,
                "recall_rate": 0.0,
                "follow_up_questions": [
                    "AIOps平台有哪些核心功能？",
                    "如何启动AIOps平台？",
                    "AIOps平台的部署要求是什么？"
                ]
            }
            
    async def _async_save_cache(self):
        """异步保存缓存，避免阻塞主线程"""
        try:
            self._save_cache()
        except Exception as e:
            logger.error(f"异步保存缓存失败: {e}")
    
    def _get_relevant_docs(self, question: str, max_retries: int = 3) -> List[Document]:
        """从检索器获取相关文档"""
        for attempt in range(max_retries):
            try:
                if not self.retriever:
                    logger.warning("检索器未初始化，尝试重新初始化...")
                    try:
                        self._init_retriever()
                        if not self.retriever:
                            logger.error("无法初始化检索器")
                            return []
                    except Exception as init_error:
                        logger.error(f"重新初始化检索器失败: {init_error}")
                        return []
                
                # 尝试检索文档
                docs = self.retriever.invoke(question)
                
                # 验证结果
                if docs and isinstance(docs, list) and len(docs) > 0:
                    logger.info(f"成功检索到 {len(docs)} 个文档")
                    return docs
                else:
                    logger.warning(f"检索器返回了空或无效结果 (尝试 {attempt+1}/{max_retries})")
                    # 如果是最后一次尝试，返回空列表
                    if attempt == max_retries - 1:
                        return []
                    # 否则短暂延迟后重试
                    time.sleep(1)
                    
            except Exception as e:
                logger.error(f"检索尝试 {attempt+1}/{max_retries} 失败: {e}")
                # 如果是最后一次尝试，返回空列表
                if attempt == max_retries - 1:
                    logger.error("所有检索尝试都失败")
                    return []
                # 否则短暂延迟后重试
                time.sleep(1)
        
        # 如果执行到这里，说明所有尝试都失败了
        return []
    
    async def _filter_relevant_docs(self, question: str, docs: List[Document]) -> List[Document]:
        """过滤不相关的文档"""
        if not docs:
            return []
            
        # 对于少量文档，直接全部返回
        if len(docs) <= 2:
            return docs
        
        # 对于较多文档，进行相关性过滤
        try:
            # 初始化评分器
            if not hasattr(self, 'structured_llm_docs') or self.structured_llm_docs is None:
                # 使用更低的温度增强评分稳定性
                docs_llm = ChatOpenAI(
                    model=config.llm.model,
                    temperature=0.1,  # 低温度，减少随机性
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url,
                ) if self.llm_provider == "openai" else self.llm
                
                # 创建结构化输出链
                self.structured_llm_docs = docs_llm
            
            # 准备系统提示和用户提示
            # 对每个文档进行评分
            relevant_docs = []
            scores = []  # 存储评分结果
            
            # 更宽松的相关性评分标准，提高召回率
            for doc in docs:
                is_relevant, score = await self._evaluate_doc_relevance(
                    self.structured_llm_docs, question, doc.page_content, doc
                )
                
                if is_relevant:
                    doc.metadata["relevance_score"] = score
                    relevant_docs.append(doc)
                    scores.append(score)
                # 即使评为不相关，如果是少量文档场景，也考虑纳入
                elif len(docs) <= 4:
                    doc.metadata["relevance_score"] = 0.4  # 给定一个较低的相关性分数
                    relevant_docs.append(doc)
                    scores.append(0.4)
            
            # 如果没有找到任何相关文档，则返回原始的top k文档
            if not relevant_docs and docs:
                logger.warning("没有找到相关文档，直接返回前 %d 个原始文档", min(3, len(docs)))
                return docs[:min(3, len(docs))]
                
            # 根据相关性评分排序
            if scores:
                sorted_pairs = sorted(zip(relevant_docs, scores), key=lambda x: x[1], reverse=True)
                relevant_docs = [doc for doc, _ in sorted_pairs]
            
            avg_score = sum(scores) / len(scores) if scores else 0
            logger.info(f"文档过滤完成: 保留 {len(relevant_docs)}/{len(docs)} 个文档，平均相关性: {avg_score:.2f}")
            
            # 记录召回率
            recall_rate = len(relevant_docs) / len(docs) if docs else 0
            for doc in relevant_docs:
                doc.metadata["recall_rate"] = recall_rate
                
            return relevant_docs
            
        except Exception as e:
            logger.error(f"文档相关性评估失败: {str(e)}")
            # 如果评估失败，返回原始文档列表，但限制数量
            return docs[:min(4, len(docs))]
    
    async def _evaluate_doc_relevance(self, grader, question, doc_content, doc) -> Tuple[bool, float]:
        """评估文档与问题的相关性"""
        try:
            # 限制文档内容长度，避免超过上下文限制
            max_doc_len = 2000
            if len(doc_content) > max_doc_len:
                doc_content = doc_content[:max_doc_len] + "..."
                
            # 创建提示
            system_prompt = """您是一名文档相关性评估专家，负责判断文档与问题的相关程度。
请根据以下标准评估文档相关性：
1. 完全相关: 文档直接回答问题或包含问题答案的所有要素
2. 部分相关: 文档包含部分相关信息，能间接帮助回答问题
3. 稍微相关: 文档提及相关概念，但不直接解答问题
4. 不相关: 文档内容与问题完全无关

请使用宽松的评价标准，宁可错误地将文档判为相关，也不要错过有用信息。
仅输出二元判断，回答"yes"或"no"。"""

            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=f"问题: {question}\n\n文档: {doc_content}\n\n这个文档与问题相关吗？请只回答'yes'或'no'。")
            ]
            
            # 调用LLM评估
            max_retries = 2
            for attempt in range(max_retries):
                try:
                    response = await asyncio.wait_for(
                        asyncio.create_task(grader.ainvoke(messages)), 
                        timeout=10
                    )
                    result = response.content.lower().strip()
                    
                    # 解析结果
                    is_relevant = "yes" in result or "relevant" in result
                    
                    # 计算相关性分数 (0.7-1.0为相关，更倾向于保留文档)
                    if is_relevant:
                        # 根据回答内容粗略评估相关程度
                        if "highly" in result or "完全" in result or "非常" in result:
                            score = 1.0
                        elif "部分" in result or "somewhat" in result:
                            score = 0.85
                        else:
                            score = 0.75
                    else:
                        score = 0.3
                        
                    return is_relevant, score
                    
                except Exception as e:
                    if attempt < max_retries - 1:
                        logger.warning(f"评估文档相关性尝试 {attempt+1} 失败: {e}，重试中...")
                        await asyncio.sleep(1)
                    else:
                        logger.error(f"评估文档相关性失败: {e}")
                        # 评估失败时，默认认为文档相关
                        return True, 0.7
        except Exception as e:
            logger.error(f"评估文档相关性时出现异常: {e}")
            # 出错时默认相关
            return True, 0.7
    
    async def _rewrite_question(self, question: str, max_retries: int = 3) -> str:
        """重写问题以提高检索质量"""
        try:
            # 如果问题很短或者已经很清晰，直接返回原问题
            if len(question) < 10:
                return question
                
            system_prompt = """您是一位问题重写专家。您的任务是将用户的原始问题重写为更清晰、更具体、更易于检索相关文档的形式，同时保持问题的原始意图。不要添加新的问题或改变问题的范围。"""
            
            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=f"原始问题: {question}\n\n请重写这个问题，使其更清晰、更容易搜索到相关答案。只需返回重写后的问题，不要包含任何解释。")
            ]
            
            # 调用LLM重写问题
            for attempt in range(max_retries):
                try:
                    response = await asyncio.wait_for(
                        asyncio.create_task(self.llm.ainvoke(messages)),
                        timeout=10
                    )
                    
                    rewritten_question = response.content.strip()
                    
                    # 验证重写的问题不是空的且不完全相同
                    if rewritten_question and rewritten_question != question:
                        logger.info(f"问题重写: '{question}' -> '{rewritten_question}'")
                        return rewritten_question
                    else:
                        return question
                        
                except Exception as e:
                    if attempt < max_retries - 1:
                        logger.warning(f"重写问题尝试 {attempt+1} 失败: {e}，重试中...")
                        await asyncio.sleep(1)
                    else:
                        logger.error(f"重写问题失败: {e}")
                        return question
        
        except Exception as e:
            logger.error(f"重写问题时出现异常: {e}")
            return question
    
    async def _generate_from_docs(
        self, 
        question: str, 
        docs: List[Document], 
        max_retries: int = 3,
        prevent_hallucinations: bool = False,
        context_with_history: str = None
    ) -> str:
        """基于检索到的文档生成回答
        
        参数:
            question: 用户问题
            docs: 相关文档列表
            max_retries: 最大重试次数
            prevent_hallucinations: 是否开启防幻觉检查
            context_with_history: 对话历史上下文
        """
        if not docs:
            return "抱歉，我无法在知识库中找到与您问题相关的信息。"
            
        # 格式化文档内容作为上下文
        docs_content = ""
        for i, doc in enumerate(docs):
            # 提取元数据中的来源
            source = doc.metadata.get("source", "未知来源") if doc.metadata else "未知来源"
            # 添加文档内容，每个文档由分隔线和来源标记
            docs_content += f"\n\n文档 [{i+1}] (来源: {source}):\n{doc.page_content}"
            
        # 限制上下文长度，避免超出模型限制
        max_context_length = config.rag.max_context_length or 4000
        if len(docs_content) > max_context_length:
            docs_content = docs_content[:max_context_length] + "...(内容已截断)"
            
        # 构建系统提示
        system_prompt = """您是一个专业的AI助手。请仔细阅读以下文档内容，然后回答用户的问题。
        
遵循以下规则:
1. 回答必须基于提供的文档内容，不要编造信息
2. 如果文档内容不足以完全回答问题，请明确说明，不要猜测
3. 回答要简洁、清晰，直接解决用户问题
4. 使用专业、友好的语气，但不要过于口语化
5. 不要在回答中提及"根据文档"、"根据提供的信息"等词语
6. 禁止直接复制大段文档内容，请用自己的话组织回答

回答语言必须与用户问题的语言保持一致。"""

        # 添加历史上下文（如果有）
        user_prompt = f"{context_with_history}\n\n" if context_with_history else ""
        user_prompt += f"问题: {question}\n\n以下是相关文档内容:\n{docs_content}\n\n请基于以上文档内容回答问题。"
        
        # 构建消息列表
        messages = [
            SystemMessage(content=system_prompt),
            HumanMessage(content=user_prompt)
        ]
        
        # 尝试生成回答，最多重试max_retries次
        for attempt in range(max_retries):
            try:
                # 设置超时，避免无限等待
                response = await asyncio.wait_for(
                    asyncio.create_task(self.llm.ainvoke(messages)),
                    timeout=30  # 30秒超时
                )
                
                answer = response.content.strip()
                
                # 如果启用了防幻觉检查
                if prevent_hallucinations:
                    is_factual = await self._check_hallucination(question, answer, docs)
                    if not is_factual:
                        # 如果检测到幻觉，添加警告并重试
                        if attempt < max_retries - 1:
                            logger.warning(f"检测到回答存在幻觉，尝试重新生成 (尝试 {attempt+1}/{max_retries})")
                            # 添加更严格的提示
                            messages = [
                                SystemMessage(content=system_prompt + "\n\n请特别注意：只能基于文档中明确提供的信息回答，不要添加任何未在文档中明确提及的信息。"),
                                HumanMessage(content=user_prompt)
                            ]
                            continue
                
                return answer
                
            except Exception as e:
                logger.error(f"生成回答时出错 (尝试 {attempt+1}/{max_retries}): {e}")
                if attempt < max_retries - 1:
                    # 短暂延迟后重试
                    await asyncio.sleep(1)
                else:
                    # 所有重试失败后，返回错误信息
                    return "抱歉，在处理您的问题时遇到了技术问题。请稍后再试。"
        
        # 如果执行到这里，说明所有尝试都失败了
        return "抱歉，我无法生成对您问题的回答。请尝试用不同方式提问。"
    
    def _format_docs(self, docs: List[Document]) -> str:
        """将文档格式化为字符串"""
        if not docs:
            return ""
            
        formatted = ""
        for i, doc in enumerate(docs):
            source = doc.metadata.get("source", "未知来源") if doc.metadata else "未知来源"
            formatted += f"\n\n文档[{i+1}] (来源: {source}):\n{doc.page_content}"
            
        return formatted
    
    async def _check_hallucination(
        self, 
        question: str, 
        answer: str, 
        docs: List[Document],
        default: str = "yes"
    ) -> bool:
        """检查回答是否存在幻觉（是否基于事实）
        
        返回:
            bool: 如果回答基于事实返回True，存在幻觉返回False
        """
        try:
            # 对于较短的问答，可能不需要严格检查
            if len(answer) < 30:
                return True
                
            # 准备文档内容
            docs_content = ""
            for i, doc in enumerate(docs):
                docs_content += f"\n\n文档[{i+1}]:\n{doc.page_content[:1000]}"
                
            # 限制文档长度
            if len(docs_content) > 4000:
                docs_content = docs_content[:4000] + "...(内容已截断)"
                
            # 创建提示
            system_prompt = """您是一名真实性检查专家。您的任务是严格评估AI回答是否完全基于提供的文档内容。

请按照以下标准进行评估:
1. 严格检查: 回答中的每一个事实或断言都必须在文档中明确或合理推断出
2. 允许合理组织: 回答可以重组文档中的信息，但不能添加新信息
3. 允许概括和简化: 可以对复杂信息进行简化，但不能改变信息的本质
4. 禁止知识混合: 不允许将文档中没有的外部知识混入回答

评估结果只需回答'yes'或'no'。yes表示回答完全基于文档，无幻觉；no表示存在幻觉内容。"""
            
            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=f"问题: {question}\n\n文档内容: {docs_content}\n\nAI回答: {answer}\n\n这个回答是否完全基于文档内容，不存在幻觉？请只回答'yes'或'no'。")
            ]
            
            # 初始化评分器
            if not hasattr(self, 'structured_llm_hall') or self.structured_llm_hall is None:
                # 使用更低的温度增强稳定性
                hall_llm = ChatOpenAI(
                    model=config.llm.model,
                    temperature=0.1,
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url,
                ) if self.llm_provider == "openai" else self.llm
                
                self.structured_llm_hall = hall_llm
            
            # 调用LLM
            try:
                response = await asyncio.wait_for(
                    asyncio.create_task(self.structured_llm_hall.ainvoke(messages)),
                    timeout=15
                )
                
                result = response.content.strip().lower()
                
                # 解析结果
                factual = "yes" in result and "no" not in result
                logger.info(f"幻觉检查结果: {'无幻觉' if factual else '存在幻觉'}")
                return factual
                
            except Exception as e:
                logger.error(f"幻觉检查失败: {e}")
                # 出错时假设回答基于事实
                return default == "yes"
                
        except Exception as e:
            logger.error(f"幻觉检查过程中出现异常: {e}")
            return default == "yes"
    
    async def _generate_follow_up_questions(
        self, 
        question: str, 
        answer: str, 
        max_questions: int = 3
    ) -> List[str]:
        """生成后续问题建议"""
        try:
            # 创建提示
            system_prompt = """您是一个AI助手，负责生成相关的后续问题，帮助用户继续探索相关内容。

后续问题应该:
1. 与原问题和回答的主题直接相关
2. 鼓励用户进一步探索相关领域
3. 涵盖不同但相关的角度
4. 简短明了，易于理解
5. 表述自然且流畅

请生成3个后续问题，每个问题应该是完整的句子并以问号结尾。
直接返回问题列表，无需其他格式或解释。"""
            
            messages = [
                SystemMessage(content=system_prompt),
                HumanMessage(content=f"原始问题: {question}\n\n回答: {answer}\n\n请生成{max_questions}个后续问题。")
            ]
            
            # 调用LLM
            response = await asyncio.wait_for(
                asyncio.create_task(self.llm.ainvoke(messages)),
                timeout=10
            )
            
            # 解析回答，提取问题列表
            content = response.content.strip()
            
            # 提取问题
            questions = []
            for line in content.split("\n"):
                line = line.strip()
                # 删除前面的数字、点和空格
                line = re.sub(r"^\d+[\.\)、]\s*", "", line)
                
                if line and (line.endswith("?") or line.endswith("？")):
                    questions.append(line)
                elif len(line) > 10:  # 如果是较长的行，可能是问题但忘了加问号
                    questions.append(line + "?")
                    
            # 如果没有解析到足够的问题，提供默认问题
            if len(questions) < max_questions:
                default_questions = [
                    "AIOps平台有哪些核心功能?",
                    "如何部署和配置AIOps系统?",
                    "AIOps如何帮助解决常见的运维问题?"
                ]
                while len(questions) < max_questions and default_questions:
                    questions.append(default_questions.pop(0))
                    
            return questions[:max_questions]
            
        except Exception as e:
            logger.error(f"生成后续问题失败: {e}")
            # 返回默认问题
            return [
                "AIOps平台有哪些核心功能?",
                "如何部署和配置AIOps系统?",
                "AIOps如何帮助解决常见的运维问题?"
            ]
