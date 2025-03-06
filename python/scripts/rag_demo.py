#!/usr/bin/env python3
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
import sys
import logging
import argparse
import shutil

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from rag.knowledge_base import KnowledgeBase
from rag.qa_assistant import QaAssistant
from rag.llm_providers import LLMProvider
from rag.vector_store import VectorStore
from rag.generator import Generator

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

def main():
    # 解析命令行参数 (目前仅支持ollama本地模型)
    parser = argparse.ArgumentParser(description='RAG系统演示')
    parser.add_argument('--provider', type=str, choices=['openai', 'ollama'],
                        help='LLM提供者 (openai 或 ollama)')
    parser.add_argument('--model', type=str, help='LLM模型名称')
    parser.add_argument('--docs-dir', type=str, default='./knowledge_docs',
                        help='知识文档目录')
    parser.add_argument('--persist-dir', type=str, default='./data/storage/vector_store',
                        help='向量存储持久化目录')
    parser.add_argument('--question', type=str,
                        help='要提问的问题')
    parser.add_argument('--reload', action='store_true', help='强制重新加载文档')
    parser.add_argument('--embedding-provider', type=str, default='ollama',
                        choices=['openai', 'ollama'], help='嵌入向量提供者')
    parser.add_argument('--embedding-model', type=str, default='nomic-embed-text:latest',
                        help='嵌入向量模型名称')
    args = parser.parse_args()

    try:
        # 设置环境变量
        if args.provider:
            os.environ["LLM_PROVIDER"] = args.provider
        if args.model:
            os.environ["LLM_MODEL"] = args.model

        # 初始化LLM提供者
        logger.info("正在初始化LLM提供者...")
        llm_provider = LLMProvider()

        # 确保向量存储路径存在
        vector_store_path = os.path.abspath(args.persist_dir)
        os.makedirs(vector_store_path, exist_ok=True)

        # 如果指定了重新加载，则清空向量存储目录
        if args.reload:
            logger.info("检测到--reload参数，正在清空向量存储...")
            # 清空向量存储目录中的所有文件，但保留目录本身
            for item in os.listdir(vector_store_path):
                item_path = os.path.join(vector_store_path, item)
                if os.path.isfile(item_path):
                    os.remove(item_path)
                elif os.path.isdir(item_path):
                    shutil.rmtree(item_path)
            logger.info("向量存储已清空，将重新加载所有文档")

        # 初始化向量存储
        logger.info(f"正在初始化向量存储，使用 {args.embedding_provider} 提供者和 {args.embedding_model} 模型...")
        vector_store = VectorStore(
            persist_directory=vector_store_path,
            embedding_provider=args.embedding_provider,
            embedding_model=args.embedding_model
        )

        # 创建知识库
        logger.info(f"正在创建知识库，文档目录: {args.docs_dir}")
        kb = KnowledgeBase(
            docs_dir=args.docs_dir,
            persist_directory=vector_store_path,
            embedding_model=args.embedding_model
        )

        # 加载文档
        logger.info("正在加载文档...")
        if args.reload:
            try:
                # 强制重新加载文档
                kb.load_documents()
                logger.info("文档已重新加载完成")
            except Exception as e:
                logger.error(f"加载文档时出错: {e}")
                if "没有找到文档" in str(e):
                    logger.info("尝试添加示例文档...")
                    kb._add_sample_documents()
                    kb.load_documents()  # 再次尝试加载
        else:
            # 检查向量存储是否为空
            doc_count = vector_store.get_collection_stats().get('count', 0)
            if doc_count == 0:
                kb.load_documents()
                logger.info("文档已加载完成")
            else:
                logger.info(f"向量存储已包含 {doc_count} 个文档，跳过加载")

        # 初始化生成器和QA助手
        logger.info("正在初始化生成器和QA助手...")
        generator = Generator(llm_provider=llm_provider)
        qa = QaAssistant(knowledge_base=kb, generator=generator)

        # 处理问答
        question = args.question or "v小姐的电话号码是多少？"
        logger.info(f"正在处理问题: {question}")
        result = qa.answer_question(question)

        # 输出结果
        print("\n" + "="*50)
        print(f"问题: {question}")
        print("-"*50)
        print(f"回答: {result['answer']}")
        print("-"*50)
        print(f"来源: {result.get('sources', [])}")
        print("="*50)

    except Exception as e:
        logger.error(f"运行过程中发生错误: {e}", exc_info=True)
        print(f"\n错误: {e}")
        print("请检查配置并确保所有依赖已正确安装。")
        sys.exit(1)

if __name__ == "__main__":
    main()
