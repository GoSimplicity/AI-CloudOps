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
import importlib
import subprocess
import platform
import requests
import json
from typing import Dict, Any

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# 配置日志
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

logger = logging.getLogger(__name__)


def check_system_info() -> Dict[str, Any]:
    """检查系统信息"""
    info = {
        "platform": platform.platform(),
        "python_version": platform.python_version(),
        "processor": platform.processor(),
        "memory": "Unknown",
    }

    # 尝试获取内存信息
    try:
        if platform.system() == "Darwin":  # macOS
            mem_cmd = "sysctl -n hw.memsize"
            mem_bytes = int(subprocess.check_output(mem_cmd, shell=True).strip())
            info["memory"] = f"{mem_bytes / (1024**3):.2f} GB"
        elif platform.system() == "Linux":
            with open("/proc/meminfo") as f:
                for line in f:
                    if "MemTotal" in line:
                        mem_kb = int(line.split()[1])
                        info["memory"] = f"{mem_kb / (1024**2):.2f} GB"
                        break
    except:
        pass

    return info


def check_dependencies() -> Dict[str, bool]:
    """检查依赖包是否已安装"""
    dependencies = {
        "langchain": False,
        "openai": False,
        "chromadb": False,
        "tiktoken": False,
        "unstructured": False,
        "pypdf": False,
        "ollama": False,
        "sentence_transformers": False,
    }

    for package in dependencies:
        try:
            importlib.import_module(package)
            dependencies[package] = True
        except ImportError:
            pass

    return dependencies


def check_environment_variables() -> Dict[str, str]:
    """检查环境变量"""
    env_vars = {
        "LLM_PROVIDER": os.getenv("LLM_PROVIDER", "未设置"),
        "LLM_MODEL": os.getenv("LLM_MODEL", "未设置"),
        "OPENAI_API_KEY": "已设置" if os.getenv("OPENAI_API_KEY") else "未设置",
        "OLLAMA_HOST": os.getenv("OLLAMA_HOST", "未设置"),
        "EMBEDDING_MODEL": os.getenv("EMBEDDING_MODEL", "未设置"),
    }

    return env_vars


def check_ollama_service() -> Dict[str, Any]:
    """检查Ollama服务状态"""
    result = {"status": "未检查", "available_models": [], "error": None}

    if os.getenv("LLM_PROVIDER", "").lower() != "ollama":
        result["status"] = "跳过 (未使用Ollama)"
        return result

    ollama_host = os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434").rstrip("/")

    try:
        # 检查Ollama服务是否可访问
        response = requests.get(f"{ollama_host}/api/tags", timeout=5)
        if response.status_code == 200:
            result["status"] = "可访问"
            data = response.json()
            result["available_models"] = [
                model["name"] for model in data.get("models", [])
            ]
        else:
            result["status"] = "服务返回错误"
            result["error"] = f"HTTP状态码: {response.status_code}"
    except requests.exceptions.RequestException as e:
        result["status"] = "无法连接"
        result["error"] = str(e)

    return result

def check_vector_store() -> Dict[str, Any]:
    """检查向量存储状态"""
    result = {"status": "未检查", "error": None, "documents_count": 0}

    persist_dir = os.getenv("VECTOR_STORE_DIR", "./data/storage/vector_store")

    if not os.path.exists(persist_dir):
        result["status"] = "存储目录不存在"
        return result

    try:
        # 尝试导入必要的模块
        from rag.vector_store import VectorStore

        # 初始化向量存储
        vector_store = VectorStore(persist_directory=persist_dir)

        # 获取文档数量
        collection = vector_store.db._collection
        result["documents_count"] = collection.count()
        result["status"] = "正常" if result["documents_count"] > 0 else "空"
    except Exception as e:
        result["status"] = "初始化失败"
        result["error"] = str(e)

    return result


def check_knowledge_base() -> Dict[str, Any]:
    """检查知识库状态"""
    result = {
        "status": "未检查",
        "error": None,
        "docs_dir_exists": False,
        "docs_count": 0,
    }

    docs_dir = os.getenv("DOCS_DIR", "./knowledge_docs")

    if not os.path.exists(docs_dir):
        result["status"] = "文档目录不存在"
        return result

    result["docs_dir_exists"] = True

    # 统计文档数量
    supported_extensions = [".txt", ".md", ".pdf", ".csv"]
    docs_count = 0

    for root, _, files in os.walk(docs_dir):
        for file in files:
            if any(file.lower().endswith(ext) for ext in supported_extensions):
                docs_count += 1

    result["docs_count"] = docs_count
    result["status"] = "正常" if docs_count > 0 else "无文档"

    return result


def run_simple_test() -> Dict[str, Any]:
    """运行简单的测试查询"""
    result = {"status": "未执行", "error": None, "response": None, "execution_time": 0}

    try:
        import time
        from rag.llm_providers import LLMProvider

        # 创建LLM提供者
        llm_provider = LLMProvider()

        # 简单测试查询
        test_prompt = "你好，请用一句话介绍自己。"

        start_time = time.time()
        response = llm_provider.generate(prompt=test_prompt)
        end_time = time.time()

        result["status"] = "成功"
        result["response"] = response
        result["execution_time"] = round(end_time - start_time, 2)
    except Exception as e:
        result["status"] = "失败"
        result["error"] = str(e)

    return result


def print_diagnosis_report(data: Dict[str, Any]):
    """打印诊断报告"""
    print("\n" + "=" * 60)
    print("RAG系统诊断报告")
    print("=" * 60)

    # 系统信息
    print("\n[系统信息]")
    for key, value in data["system_info"].items():
        print(f"  {key}: {value}")

    # 依赖检查
    print("\n[依赖包状态]")
    for package, installed in data["dependencies"].items():
        status = "✓ 已安装" if installed else "✗ 未安装"
        print(f"  {package}: {status}")

    # 环境变量
    print("\n[环境变量]")
    for var, value in data["environment_variables"].items():
        print(f"  {var}: {value}")

    # Ollama服务
    print("\n[Ollama服务]")
    print(f"  状态: {data['ollama_service']['status']}")
    if data["ollama_service"]["error"]:
        print(f"  错误: {data['ollama_service']['error']}")
    if data["ollama_service"]["available_models"]:
        print(f"  可用模型: {', '.join(data['ollama_service']['available_models'])}")

    # OpenAI API
    print("\n[OpenAI API]")
    print(f"  状态: {data['openai_api']['status']}")
    if data["openai_api"]["error"]:
        print(f"  错误: {data['openai_api']['error']}")

    # 向量存储
    print("\n[向量存储]")
    print(f"  状态: {data['vector_store']['status']}")
    print(f"  文档数量: {data['vector_store']['documents_count']}")
    if data["vector_store"]["error"]:
        print(f"  错误: {data['vector_store']['error']}")

    # 知识库
    print("\n[知识库]")
    print(f"  状态: {data['knowledge_base']['status']}")
    print(f"  文档数量: {data['knowledge_base']['docs_count']}")
    if data["knowledge_base"]["error"]:
        print(f"  错误: {data['knowledge_base']['error']}")

    # 测试结果
    print("\n[简单测试]")
    print(f"  状态: {data['test_result']['status']}")
    print(f"  执行时间: {data['test_result']['execution_time']}秒")
    if data["test_result"]["response"]:
        print(f"  响应: {data['test_result']['response'][:100]}...")
    if data["test_result"]["error"]:
        print(f"  错误: {data['test_result']['error']}")

    # 诊断结论
    print("\n[诊断结论]")

    # 检查是否有错误
    errors = []
    if not all(data["dependencies"].values()):
        missing = [
            pkg for pkg, installed in data["dependencies"].items() if not installed
        ]
        errors.append(f"缺少依赖包: {', '.join(missing)}")

    if (
        data["ollama_service"]["status"] == "无法连接"
        and data["environment_variables"]["LLM_PROVIDER"] == "ollama"
    ):
        errors.append("Ollama服务无法连接")

    if (
        data["openai_api"]["status"] == "连接失败"
        and data["environment_variables"]["LLM_PROVIDER"] == "openai"
    ):
        errors.append("OpenAI API连接失败")

    if data["vector_store"]["status"] in ["存储目录不存在", "初始化失败"]:
        errors.append("向量存储初始化失败")

    if data["knowledge_base"]["status"] == "文档目录不存在":
        errors.append("知识库文档目录不存在")

    if data["test_result"]["status"] == "失败":
        errors.append("LLM测试失败")

    if errors:
        print("  发现以下问题:")
        for error in errors:
            print(f"  - {error}")
        print("\n  建议修复步骤:")

        if any("依赖包" in e for e in errors):
            print("  1. 运行 'python scripts/setup_rag.py' 安装缺失的依赖")

        if any("Ollama" in e for e in errors):
            print("  2. 检查Ollama服务是否已启动，或切换到OpenAI提供者")
            print("     - 启动Ollama: 'ollama serve'")
            print("     - 或设置环境变量: 'export LLM_PROVIDER=openai'")

        if any("OpenAI" in e for e in errors):
            print("  3. 检查OpenAI API密钥是否正确，或切换到Ollama提供者")
            print("     - 设置API密钥: 'export OPENAI_API_KEY=your_key'")
            print("     - 或设置环境变量: 'export LLM_PROVIDER=ollama'")

        if any("向量存储" in e for e in errors) or any("知识库" in e for e in errors):
            print("  4. 确保知识库目录和向量存储目录存在")
            print(
                "     - 创建目录: 'mkdir -p ./knowledge_docs ./data/storage/vector_store'"
            )
    else:
        print("  未发现明显问题，RAG系统应该可以正常工作")

    print("\n" + "=" * 60)


def main():
    """主函数"""
    print("正在诊断RAG系统...")

    # 收集诊断数据
    diagnosis_data = {
        "system_info": check_system_info(),
        "dependencies": check_dependencies(),
        "environment_variables": check_environment_variables(),
        "ollama_service": check_ollama_service(),
        "vector_store": check_vector_store(),
        "knowledge_base": check_knowledge_base(),
        "test_result": run_simple_test(),
    }

    # 打印诊断报告
    print_diagnosis_report(diagnosis_data)

    # 保存诊断报告
    try:
        with open("rag_diagnosis_report.json", "w") as f:
            json.dump(diagnosis_data, f, indent=2, ensure_ascii=False)
        print(f"诊断报告已保存到: {os.path.abspath('rag_diagnosis_report.json')}")
    except Exception as e:
        print(f"保存诊断报告时出错: {e}")


if __name__ == "__main__":
    main()
