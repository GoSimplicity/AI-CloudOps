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
        if not config.llm.api_key or config.llm.api_key.startswith("sk-") is False:
            logger.warning("API密钥未正确设置，请检查环境变量")
        
        try:
            if self.llm_provider == 'openai':
                logger.info("使用OpenAI嵌入模型")
                self.embedding = OpenAIEmbeddings(
                    model=config.rag.openai_embedding_model,
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url
                )
            else:
                logger.info("使用Ollama嵌入模型")
                self.embedding = OllamaEmbeddings(
                    model=config.rag.ollama_embedding_model,
                    base_url=config.llm.ollama_base_url
                )
                
            # 测试嵌入模型
            try:
                # 简单测试嵌入功能
                test_embedding = self.embedding.embed_query("测试嵌入功能")
                if test_embedding and len(test_embedding) > 0:
                    logger.info(f"嵌入模型测试成功: 维度={len(test_embedding)}")
                else:
                    logger.warning("嵌入模型测试结果异常")
            except Exception as test_error:
                logger.warning(f"嵌入模型测试失败: {test_error}")
                
        except Exception as e:
            logger.error(f"初始化嵌入模型失败: {e}，尝试备用方法...")
            
            # 尝试备用方法
            if self.llm_provider == 'openai':
                # 如果OpenAI失败，尝试Ollama
                try:
                    self.embedding = OllamaEmbeddings(
                        model=config.rag.ollama_embedding_model,
                        base_url=config.llm.ollama_base_url
                    )
                except Exception as e2:
                    logger.error(f"备用嵌入方法也失败: {e2}")
                    # 创建一个简单的嵌入模型，返回随机向量
                    raise ValueError("无法初始化任何可用的嵌入模型")
                    return
            else:
                # 如果Ollama失败，尝试OpenAI
                try:
                    self.embedding = OpenAIEmbeddings(
                        model=config.rag.openai_embedding_model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url
                    )
                except Exception as e2:
                    logger.error(f"备用嵌入方法也失败: {e2}")
                    # 创建一个简单的嵌入模型，返回随机向量
                    raise ValueError("无法初始化任何可用的嵌入模型")
                    return
    
    
    def _init_llm(self) -> None:
        """初始化语言模型"""
        try:
            if self.llm_provider == 'openai':
                # 使用deepseek-ai/DeepSeek-R1-Distill-Qwen-14B作为推理模型
                inference_model = "deepseek-ai/DeepSeek-R1-Distill-Qwen-14B"
                # 使用Qwen/Qwen3-14B作为对话模型
                chat_model = config.llm.model
                
                # 根据任务不同使用不同的模型
                self.llm = ChatOpenAI(
                    model=chat_model,
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url,
                    temperature=config.rag.temperature,
                    timeout=30
                )
            else:
                self.llm = ChatOllama(
                    model=config.llm.ollama_model,
                    base_url=config.llm.ollama_base_url,
                    temperature=config.rag.temperature,
                    timeout=30
                )
                
            # 测试LLM
            try:
                test_response = self.llm.invoke("测试")
                if test_response:
                    logger.info("LLM测试成功")
            except Exception as test_error:
                logger.warning(f"LLM测试失败: {test_error}")
            
            # 初始化结构化输出的LLM - 使用JsonOutputParser替代structured_output
            self.json_parser = JsonOutputParser()
            
        except Exception as e:
            logger.error(f"初始化语言模型失败: {e}，尝试备用方法...")
            
            # 尝试备用方法
            if self.llm_provider == 'openai':
                # 如果OpenAI失败，尝试Ollama
                try:
                    self.llm = ChatOllama(
                        model=config.llm.ollama_model,
                        base_url=config.llm.ollama_base_url,
                        temperature=config.rag.temperature,
                        timeout=30
                    )
                except Exception as e2:
                    logger.error(f"备用语言模型也失败: {e2}")
                    raise ValueError("无法初始化任何可用的语言模型")
                    return
            else:
                # 如果Ollama失败，尝试OpenAI
                try:
                    self.llm = ChatOpenAI(
                        model=config.llm.model,
                        api_key=config.llm.api_key,
                        base_url=config.llm.base_url,
                        temperature=config.rag.temperature,
                        timeout=30
                    )
                except Exception as e2:
                    logger.error(f"备用语言模型也失败: {e2}")
                    raise ValueError("无法初始化任何可用的语言模型")
                    return
            
            # 初始化JSON解析器
            self.json_parser = JsonOutputParser()
            
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
            return documents
            
        all_files = list(kb_path.glob("**/*"))
        logger.info(f"知识库目录下的文件: {[str(f) for f in all_files]}")
        
        try:
            logger.info("使用基础文档加载实现")
            
            # 加载TXT文件
            txt_files = list(kb_path.glob("**/*.txt"))
            logger.info(f"找到的TXT文件: {len(txt_files)}个")
            for file_path in txt_files:
                try:
                    with open(file_path, "r", encoding="utf-8") as f:
                        content = f.read()
                        documents.append(Document(
                            page_content=content,
                            metadata={"source": str(file_path)}
                        ))
                        logger.info(f"成功加载TXT文件: {file_path}")
                except Exception as file_error:
                    logger.error(f"加载文件 {file_path} 失败: {file_error}")
            
            # 加载MD文件
            md_files = list(kb_path.glob("**/*.md"))
            logger.info(f"找到的MD文件: {len(md_files)}个")
            for file_path in md_files:
                try:
                    with open(file_path, "r", encoding="utf-8") as f:
                        content = f.read()
                        documents.append(Document(
                            page_content=content,
                            metadata={"source": str(file_path)}
                        ))
                        logger.info(f"成功加载MD文件: {file_path}")
                except Exception as file_error:
                    logger.error(f"加载文件 {file_path} 失败: {file_error}")
            
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
            
            # 检查知识库路径
            kb_path = Path(self.knowledge_base_path)
            if not kb_path.exists():
                os.makedirs(kb_path, exist_ok=True)
                logger.info(f"创建知识库目录: {kb_path}")
            
            # 检查向量数据库目录权限
            use_in_memory = in_test_environment
            
            db_path = Path(self.vector_db_path)
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
            
            # 如果使用内存模式
            if use_in_memory:
                logger.info("使用内存模式刷新知识库")
                self._create_vector_store(use_in_memory=True)
            else:
                # 正常环境 - 清理现有文件并创建新数据库
                try:
                    # 清理现有的向量数据库文件
                    import shutil
                    try:
                        for item in db_path.glob("*"):
                            if item.is_file():
                                item.unlink()
                            elif item.is_dir() and item.name != "cache":  # 保留缓存目录
                                shutil.rmtree(item)
                        logger.info("已清理现有向量数据库文件")
                    except Exception as clean_error:
                        logger.warning(f"清理向量数据库文件失败: {clean_error}")
                    
                    # 创建新的向量数据库
                    self._create_vector_store(use_in_memory=False)
                except Exception as store_error:
                    logger.warning(f"创建向量数据库失败: {store_error}，尝试使用内存模式")
                    self._create_vector_store(use_in_memory=True)
            
            # 测试检索器是否可用
            doc_count = 0
            if self.retriever:
                try:
                    test_result = self.retriever.invoke("test")
                    doc_count = len(test_result)
                    logger.info(f"检索器测试成功: 返回 {doc_count} 个结果")
                except Exception as test_error:
                    logger.warning(f"检索器测试失败: {test_error}")
            else:
                logger.warning("刷新后检索器仍不可用")
            
            # 保存更新的空缓存
            self._save_cache()
            logger.info("已清理响应缓存")
            
            return {"success": True, "documents_count": doc_count}
        except Exception as e:
            logger.error(f"刷新知识库失败: {e}")
            # 最后尝试使用内存模式
            try:
                logger.info("尝试使用内存模式作为最后的备用方案")
                self._create_vector_store(use_in_memory=True)
                if self.retriever:
                    test_result = self.retriever.invoke("test")
                    doc_count = len(test_result)
                    logger.info(f"内存模式检索器测试成功: 返回 {doc_count} 个结果")
                    return {"success": True, "documents_count": doc_count}
            except:
                pass
            
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
                    logger.warning("检索器未初始化")
                    return []
                
                docs = self.retriever.invoke(question)
                return docs
            except Exception as e:
                logger.error(f"检索尝试 {attempt+1}/{max_retries} 失败: {e}")
                if attempt == max_retries - 1:
                    logger.error("无法检索文档")
                    return []
        return []
    
    async def _filter_relevant_docs(self, question: str, docs: List[Document]) -> List[Document]:
        """过滤与问题相关的文档，提高检索准确率"""
        if not docs:
            return []
        
        # 提高相关度评估的精确提示模板
        system = """您是一名文档相关性评估专家，负责评估检索到的文档与用户问题的相关性。
        
评估标准:
1. 文档必须包含与问题直接相关的信息，而不仅仅是同一主题
2. 如果文档只是泛泛地涉及主题但不能回答具体问题，则视为不相关
3. 如果文档包含用户问题所需的关键信息，则视为相关
4. 如果文档部分相关，但提供了有价值的背景信息，可以视为相关
5. 对于询问联系方式的问题，如果文档中包含该人的联系方式信息，应评为"yes"
6. 对于人员负责或维护服务的问题，只要文档中提到了相关人员的信息，就应评为相关

您的评估应更严格，目标是过滤掉不相关的文档，提高检索质量。
以JSON格式输出，键为"binary_score"，值为"yes"或"no"，表示文档是否与问题相关。"""

        grade_prompt = ChatPromptTemplate.from_messages([
            ("system", system),
            ("human", "文档内容: \n\n {document} \n\n 用户问题: {question}\n\n此文档与问题相关吗？请用JSON格式回答，例如：{\"binary_score\": \"yes\"} 或 {\"binary_score\": \"no\"}"),
        ])
        
        relevant_docs = []
        filtered_docs_count = 0
        
        # 创建检索评分链
        retrieval_grader = grade_prompt | self.llm | self.json_parser
        
        for idx, doc in enumerate(docs):
            try:
                # 截取文档内容，避免过长
                doc_content = doc.page_content[:2500] if len(doc.page_content) > 2500 else doc.page_content
                
                # 评估文档相关性
                result = retrieval_grader.invoke({
                    "question": question, 
                    "document": doc_content
                })
                
                if result and isinstance(result, dict) and result.get("binary_score", "").lower() == "yes":
                    relevant_docs.append(doc)
                else:
                    filtered_docs_count += 1
                    logger.debug(f"过滤掉不相关文档: {doc.metadata.get('source', '未知')}")
            
            except Exception as e:
                logger.error(f"评估文档相关性时出错 (文档 {idx+1}/{len(docs)}): {e}")
                # 出错时默认保留文档
                relevant_docs.append(doc)
        
        # 如果过滤后没有相关文档，但原来有文档，则保留原始的前2个文档
        if not relevant_docs and docs:
            logger.warning("过滤后没有相关文档，保留原始的前2个文档")
            relevant_docs = docs[:min(2, len(docs))]
        
        # 计算文档召回率
        total_docs = len(docs)
        relevant_count = len(relevant_docs)
        recall_rate = relevant_count / total_docs if total_docs > 0 else 0
        
        logger.info(f"文档过滤结果: 原始 {len(docs)} 个，过滤后 {len(relevant_docs)} 个，过滤掉 {filtered_docs_count} 个")
        logger.info(f"文档召回率: {recall_rate:.2f} ({relevant_count}/{total_docs})")
        
        # 将文档召回率添加到第一个文档的元数据中，如果有文档
        if relevant_docs:
            for doc in relevant_docs:
                if not doc.metadata:
                    doc.metadata = {}
                doc.metadata["recall_rate"] = recall_rate
        
        return relevant_docs
    
    async def _rewrite_question(self, question: str, max_retries: int = 3) -> str:
        """重写问题以提高检索效果"""
        # 对于某些类型的问题，可以选择不重写
        if "联系方式" in question.lower():
            return question  # 保持原始问题
            
        # 提示模板
        system = """您有一个问题重写器，可将输入问题转换为针对vectorstore检索进行了优化的更好版本。
查看输入并尝试推断底层语义意图/含义，使用用户语言回复。
对于询问联系方式的问题，保持原始形式或仅进行细微调整以匹配可能的格式。
对于查询"谁负责/维护的服务最多"这类问题，请重写为查询具体的人员-服务分配关系。"""

        re_write_prompt = ChatPromptTemplate.from_messages([
            ("system", system),
            ("human", "Here is the initial question: \n\n {question} \n Formulate an improved question."),
        ])
        
        for attempt in range(max_retries):
            try:
                # 创建问题重写链
                question_rewriter = re_write_prompt | self.llm | StrOutputParser()
                return question_rewriter.invoke({"question": question})
            except Exception as e:
                logger.error(f"问题重写尝试 {attempt+1}/{max_retries} 失败: {e}")
                if attempt == max_retries - 1:
                    return question  # 返回原始问题
        
        return question
    
    async def _generate_from_docs(
        self, 
        question: str, 
        docs: List[Document], 
        max_retries: int = 3,
        prevent_hallucinations: bool = False,
        context_with_history: str = None
    ) -> str:
        """基于文档生成回答
        
        参数:
            question: 用户问题
            docs: 相关文档列表
            max_retries: 最大重试次数
            prevent_hallucinations: 是否启用额外的防幻觉机制
            context_with_history: 对话历史上下文
        """
        # 没有相关文档，直接返回"我不知道"
        if not docs:
            return "我不知道，因为没有找到相关文档。"
            
        # 格式化文档内容
        formatted_docs = self._format_docs(docs)
            
        # 尝试直接匹配简单的联系方式问题
        if "联系方式" in question.lower():
            # 在文档中查找联系方式
            matches = re.findall(r"联系方式：(\d+)", formatted_docs)
            if matches:
                # 尝试找出是谁的联系方式
                person_match = re.search(r"([\w]+).*?联系方式", question)
                if person_match:
                    person = person_match.group(1)
                    person_contact = re.search(fr"{person}.*?联系方式：(\d+)", formatted_docs)
                    if person_contact:
                        return f"{person}的联系方式是：{person_contact.group(1)}"
                return f"找到的联系方式是：{matches[0]}"
        
        # 准备系统提示
        system_prompt = """你是一个专业的助手，负责基于给定的文档回答用户问题。

使用提供的文档中明确存在的信息来回答用户问题。
如果是询问联系方式或简单事实的问题，请直接提供答案，无需详细解释。
如果文档中没有包含回答问题所需的信息，请明确回答"我不知道"或"提供的文档中没有这些信息"。
"""

        # 根据条件增强系统提示
        if prevent_hallucinations:
            system_prompt += """
绝对不要编造信息。你的回答必须100%基于提供的文档。
对于每个信息点，必须确认它明确存在于文档中。
当用户提问的内容不在文档中时，请明确指出"这个信息不在提供的文档中"。
"""

        # 准备人类提示
        human_template = "文档信息:\n\n{context}\n\n"
        
        # 添加对话历史上下文(如果有)
        if context_with_history:
            human_template += f"{context_with_history}\n"
            
        human_template += "用户问题: {question}"
            
        # 构建提示模板
        custom_prompt = ChatPromptTemplate.from_messages([
            ("system", system_prompt),
            ("human", human_template)
        ])
            
        # 创建改进的RAG链
        rag_chain = custom_prompt | self.llm | StrOutputParser()
            
        for attempt in range(max_retries):
            try:
                response = rag_chain.invoke({
                    "context": formatted_docs, 
                    "question": question
                })
                return response
            except Exception as e:
                logger.error(f"生成尝试 {attempt+1}/{max_retries} 失败: {e}")
                if attempt == max_retries - 1:
                    return "抱歉，我无法基于提供的信息回答您的问题。"
                
        return "抱歉，我无法基于提供的信息回答您的问题。"
    
    def _format_docs(self, docs: List[Document]) -> str:
        """格式化文档内容"""
        return "\n\n".join(doc.page_content for doc in docs)
    
    async def _check_hallucination(
        self, 
        question: str, 
        answer: str, 
        docs: List[Document],
        default: str = "yes"
    ) -> bool:
        """检查回答是否存在幻觉"""
        if not docs:
            return False  # 如果没有文档，则肯定是幻觉
            
        # 格式化文档内容为字符串
        doc_content = "\n".join(d.page_content for d in docs)
            
        try:
            # 改进的幻觉评估提示模板
            system = """您是一名评分员，负责评估生成的回答是否基于提供的文档。
如果生成的回答包含任何文档中没有明确提及的重要信息或事实，请评为'no'。
特别是对于人员负责或维护服务的陈述，必须在文档中有明确支持才能评为'yes'。
对于联系方式类的简单事实性问题，如果回答准确反映了文档中的信息，应评为'yes'。
以JSON格式输出，键为"binary_score"，值为"yes"或"no"，表示回答是否符合文档中的事实。"""

            hallucination_prompt = ChatPromptTemplate.from_messages([
                ("system", system),
                ("human", "文档内容: \n\n {documents} \n\n 生成的回答: {generation} \n\n 用户问题: {question}\n\n请用JSON格式回答，例如：{\"binary_score\": \"yes\"} 或 {\"binary_score\": \"no\"}"),
            ])
                
            # 创建幻觉评分链
            hallucination_grader = hallucination_prompt | self.llm | self.json_parser
                
            result = hallucination_grader.invoke({
                "documents": doc_content,
                "generation": answer,
                "question": question
            })
                
            if result and isinstance(result, dict) and "binary_score" in result:
                return result["binary_score"] == "yes"
            return default == "yes"
        except Exception as e:
            logger.error(f"幻觉评分失败: {e}")
            return default == "yes"
    
    async def _generate_follow_up_questions(
        self, 
        question: str, 
        answer: str, 
        max_questions: int = 3
    ) -> List[str]:
        """生成后续问题建议"""
        try:
            # 提示模板
            prompt = ChatPromptTemplate.from_messages([
                ("system", f"""基于用户的原始问题和提供的回答，生成 {max_questions} 个相关的后续问题。
这些问题应该是用户可能接下来想问的，并且与原始话题紧密相关。
直接返回问题列表，每行一个问题，不要有编号或其他标记。"""),
                ("human", "原始问题: {question}\n\n回答: {answer}\n\n生成 {max_questions} 个后续问题:"),
            ])
            
            # 创建生成链
            chain = prompt | self.llm | StrOutputParser()
            
            result = chain.invoke({
                "question": question,
                "answer": answer,
                "max_questions": max_questions
            })
            
            # 处理结果，分割成列表
            questions = [q.strip() for q in result.split('\n') if q.strip()]
            
            # 限制数量
            return questions[:max_questions]
            
        except Exception as e:
            logger.error(f"生成后续问题失败: {e}")
            return []
