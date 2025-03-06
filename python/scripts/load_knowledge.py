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

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from rag.knowledge_base import KnowledgeBase

# 配置日志
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

logger = logging.getLogger(__name__)


def main():
    parser = argparse.ArgumentParser(description="加载知识库文档")
    parser.add_argument(
        "--docs-dir", type=str, default="./knowledge_docs", help="知识文档目录"
    )
    parser.add_argument(
        "--persist-dir",
        type=str,
        default="./data/storage/vector_store",
        help="向量存储持久化目录",
    )
    parser.add_argument(
        "--embedding-model",
        type=str,
        default="nomic-embed-text:latest",
        help="嵌入向量模型名称",
    )
    parser.add_argument(
        "--embedding-provider",
        type=str,
        default="ollama",
        choices=["openai", "ollama"],
        help="嵌入向量提供者",
    )
    parser.add_argument("--reload", action="store_true", help="强制重新加载文档")
    args = parser.parse_args()

    try:
        # 检查Ollama服务是否可用
        if args.embedding_provider == "ollama":
            try:
                from ollama import Client

                client = Client(host=os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434"))
                try:
                    models = client.list()
                    logger.info("Ollama服务可用")

                    # 检查模型是否存在
                    model_exists = False
                    for model in models.models:
                        if args.embedding_model in model["model"]:
                            model_exists = True
                            break

                    if not model_exists:
                        logger.warning(
                            f"模型 {args.embedding_model} 未找到，尝试下载..."
                        )
                        client.pull(args.embedding_model)

                except Exception as e:
                    logger.error(f"无法连接Ollama服务: {e}")
                    sys.exit(1)
            except ImportError:
                logger.error("未安装ollama客户端库，请运行 'pip install ollama'")
                sys.exit(1)

        # 确保向量存储路径存在
        vector_store_path = os.path.abspath(args.persist_dir)
        os.makedirs(vector_store_path, exist_ok=True)

        # 确保文档目录存在
        docs_dir_path = os.path.abspath(args.docs_dir)
        os.makedirs(docs_dir_path, exist_ok=True)

        # 创建知识库
        kb = KnowledgeBase(
            docs_dir=docs_dir_path,
            persist_directory=vector_store_path,
            embedding_model=args.embedding_model,
        )

        # 检查文档目录是否存在
        if not os.path.exists(kb.docs_dir):
            logger.error(f"文档目录不存在: {kb.docs_dir}")
            sys.exit(1)

        # 如果指定了重新加载，则强制重新加载文档
        if args.reload:
            logger.info("强制重新加载文档...")
            kb.load_documents(force_reload=True)
        else:
            # 加载文档
            kb.load_documents()

        # 打印持久化目录
        logger.info(f"知识库已加载并存储在: {kb.persist_directory}")
        logger.info(
            f"文档数量: {kb.get_document_count() if hasattr(kb, 'get_document_count') else '未知'}"
        )

    except Exception as e:
        logger.error(f"程序运行出错: {e}", exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
