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
import subprocess
import argparse
import logging
import platform

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

def check_and_install_dependencies():
    """检查并安装必要的依赖"""
    required_packages = [
        "langchain",
        "chromadb",
        "tiktoken",
        "unstructured",
        "pypdf",
        "pandas",
        "python-dotenv",
        "ollama"
    ]

    optional_packages = {
        "sentence-transformers": "使用本地嵌入模型",
        "faiss-cpu": "使用FAISS向量存储",
        "langchain-community": "使用LangChain社区组件"
    }

    logger.info("检查必要依赖...")
    for package in required_packages:
        try:
            __import__(package.replace("-", "_"))
            logger.info(f"✓ {package} 已安装")
        except ImportError:
            logger.warning(f"! {package} 未安装，正在安装...")
            subprocess.check_call([sys.executable, "-m", "pip", "install", package])
            logger.info(f"✓ {package} 安装完成")

    logger.info("\n检查可选依赖...")
    for package, description in optional_packages.items():
        try:
            __import__(package.replace("-", "_"))
            logger.info(f"✓ {package} 已安装 ({description})")
        except ImportError:
            logger.warning(f"! {package} 未安装 ({description})")
            install = input(f"是否安装 {package}? (y/n): ").lower() == 'y'
            if install:
                try:
                    subprocess.check_call([sys.executable, "-m", "pip", "install", package])
                    logger.info(f"✓ {package} 安装完成")
                except subprocess.CalledProcessError:
                    logger.error(f"× 安装 {package} 失败")
                    logger.warning(f"请稍后手动安装 {package}")
            else:
                logger.info(f"× 跳过安装 {package}")

def setup_environment():
    """设置环境变量"""
    env_vars = {
        "LLM_PROVIDER": "使用的LLM提供者 (openai 或 ollama)",
        "OPENAI_API_KEY": "OpenAI API密钥 (如果使用OpenAI)",
        "LLM_MODEL": "使用的模型名称",
        "OLLAMA_HOST": "Ollama服务地址 (如果使用Ollama，默认: http://127.0.0.1:11434)",
        "EMBEDDING_MODEL": "嵌入模型名称",
        "VECTOR_STORE_TYPE": "向量存储类型 (chroma 或 faiss)",
        "VECTOR_STORE_PATH": "向量存储路径"
    }

    logger.info("\n设置环境变量...")

    # 检查是否已有环境变量配置文件
    env_file = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), ".env")
    existing_vars = {}

    if os.path.exists(env_file):
        logger.info(f"发现现有环境变量配置文件: {env_file}")
        try:
            with open(env_file, 'r') as f:
                for line in f:
                    if '=' in line and not line.startswith('#'):
                        key, value = line.strip().split('=', 1)
                        existing_vars[key] = value
        except Exception as e:
            logger.warning(f"读取环境变量文件时出错: {e}")

    # 设置默认值
    default_values = {
        "LLM_PROVIDER": existing_vars.get("LLM_PROVIDER", "openai"),
        "LLM_MODEL": existing_vars.get("LLM_MODEL", "gpt-3.5-turbo"),
        "OLLAMA_HOST": existing_vars.get("OLLAMA_HOST", "http://127.0.0.1:11434"),
        "EMBEDDING_MODEL": existing_vars.get("EMBEDDING_MODEL", "text-embedding-ada-002"),
        "VECTOR_STORE_TYPE": existing_vars.get("VECTOR_STORE_TYPE", "chroma"),
        "VECTOR_STORE_PATH": existing_vars.get("VECTOR_STORE_PATH", "./data/storage/vector_store")
    }

    # 询问用户设置环境变量
    new_vars = {}
    for var, description in env_vars.items():
        current_value = existing_vars.get(var, os.environ.get(var, default_values.get(var, '')))
        print(f"\n{var}: {description}")
        print(f"当前值: {current_value or '未设置'}")

        new_value = input(f"输入新值 (直接回车保持当前值): ")
        if new_value:
            new_vars[var] = new_value
        elif current_value:
            new_vars[var] = current_value

    # 保存环境变量
    try:
        with open(env_file, 'w') as f:
            for var, value in new_vars.items():
                f.write(f"{var}={value}\n")
        logger.info(f"环境变量已保存到: {env_file}")
    except Exception as e:
        logger.error(f"保存环境变量文件时出错: {e}")
        sys.exit(1)

    # 显示如何加载环境变量的提示
    if platform.system() == "Windows":
        logger.info("请确保在运行前加载这些环境变量，例如:")
        logger.info(f"$env:OPENAI_API_KEY=\"your-key-here\"")
    else:
        logger.info("请确保在运行前加载这些环境变量，例如:")
        logger.info(f"source {env_file} 或 export $(cat {env_file} | xargs)")

def create_directories():
    """创建必要的目录"""
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    dirs = [
        os.path.join(project_root, "knowledge_docs"),
        os.path.join(project_root, "data/storage/vector_store"),
        os.path.join(project_root, "data/processed"),
        os.path.join(project_root, "logs")
    ]

    logger.info("\n创建必要目录...")
    for d in dirs:
        try:
            os.makedirs(d, exist_ok=True)
            logger.info(f"✓ 目录已创建: {d}")
        except Exception as e:
            logger.error(f"× 创建目录失败 {d}: {e}")
            sys.exit(1)
    
    # 创建示例文档
    example_doc_path = os.path.join(project_root, "knowledge_docs/example.txt")
    if not os.path.exists(example_doc_path):
        try:
            with open(example_doc_path, 'w') as f:
                f.write("这是一个示例文档，用于测试RAG系统。\n")
                f.write("您可以将您的知识文档放在knowledge_docs目录下。\n")
                f.write("支持的文件格式包括：txt, pdf, docx等。\n")
            logger.info(f"✓ 示例文档已创建: {example_doc_path}")
        except Exception as e:
            logger.warning(f"× 创建示例文档失败: {e}")

def check_system_requirements():
    """检查系统要求"""
    logger.info("\n检查系统要求...")
    
    # 检查Python版本
    python_version = sys.version_info
    if python_version.major < 3 or (python_version.major == 3 and python_version.minor < 8):
        logger.warning(f"× Python版本过低: {python_version.major}.{python_version.minor}")
        logger.warning("推荐使用Python 3.8或更高版本")
    else:
        logger.info(f"✓ Python版本: {python_version.major}.{python_version.minor}")
    
    # 检查内存
    try:
        import psutil
        memory_gb = psutil.virtual_memory().total / (1024**3)
        if memory_gb < 4:
            logger.warning(f"× 系统内存可能不足: {memory_gb:.1f} GB")
            logger.warning("推荐至少4GB内存用于RAG系统")
        else:
            logger.info(f"✓ 系统内存: {memory_gb:.1f} GB")
    except ImportError:
        logger.info("无法检查系统内存 (psutil未安装)")

def main():
    parser = argparse.ArgumentParser(description='设置RAG系统环境')
    parser.add_argument('--skip-deps', action='store_true', help='跳过依赖检查')
    parser.add_argument('--skip-env', action='store_true', help='跳过环境变量设置')
    parser.add_argument('--skip-dirs', action='store_true', help='跳过目录创建')
    parser.add_argument('--skip-checks', action='store_true', help='跳过系统检查')
    args = parser.parse_args()

    print("="*50)
    print("RAG系统环境设置")
    print("="*50)

    if not args.skip_checks:
        check_system_requirements()

    if not args.skip_deps:
        check_and_install_dependencies()

    if not args.skip_dirs:
        create_directories()

    if not args.skip_env:
        setup_environment()

    print("\n"+"="*50)
    print("设置完成！现在您可以运行以下命令来测试RAG系统:")
    print("python scripts/rag_demo.py")
    print("="*50)

if __name__ == "__main__":
    main()