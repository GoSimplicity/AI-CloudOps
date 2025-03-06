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
import argparse
import logging
import json

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from models.deepseek_finetune import DeepSeekFineTuner
from rag.qa_assistant import QaAssistant

# 配置日志
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def train_model(args):
    """训练问答模型"""
    logger.info("初始化微调过程...")

    # 确保输出目录存在
    os.makedirs(args.output_dir, exist_ok=True)

    # 确保数据集目录存在
    if not os.path.exists(args.dataset):
        logger.error(f"数据集目录不存在: {args.dataset}")
        if args.create_sample:
            logger.info("创建示例数据集...")
            os.makedirs(os.path.dirname(args.dataset), exist_ok=True)
            create_sample_dataset(args.dataset)
        else:
            logger.error("请提供有效的数据集路径或使用 --create-sample 创建示例数据集")
            sys.exit(1)

    finetuner = DeepSeekFineTuner(
        model_name=args.base_model,
        output_dir=args.output_dir,
        dataset_path=args.dataset,
        num_train_epochs=args.epochs,
        learning_rate=args.learning_rate,
        per_device_train_batch_size=args.batch_size,
        lora_r=args.lora_r,
        max_seq_length=args.max_length,
    )

    logger.info("开始训练...")
    finetuner.train()
    logger.info(f"训练完成。模型已保存到 {args.output_dir}")


def test_model(args):
    """测试微调后的模型"""
    logger.info("加载QA助手进行测试...")

    if not os.path.exists(args.output_dir):
        logger.error(f"模型目录不存在: {args.output_dir}")
        sys.exit(1)

    try:
        assistant = QaAssistant(
            model_path=args.base_model,
            adapter_path=args.output_dir,
            use_retrieval=False,
        )

        # 测试问题
        test_questions = [
            "什么是异常检测?",
            "根因分析如何帮助解决问题?",
            "解释下知识图谱是如何工作的",
        ]

        if args.test_questions:
            try:
                with open(args.test_questions, "r", encoding="utf-8") as f:
                    custom_questions = json.load(f)
                    if isinstance(custom_questions, list) and len(custom_questions) > 0:
                        test_questions = custom_questions
                    else:
                        logger.warning("测试问题文件格式不正确，使用默认问题")
            except Exception as e:
                logger.warning(f"无法加载测试问题文件: {e}")

        for question in test_questions:
            logger.info(f"测试问题: {question}")
            response = assistant.answer_question(question)
            logger.info(f"回答: {response['answer']}")
            logger.info("-" * 50)
    except Exception as e:
        logger.error(f"测试模型时出错: {e}", exc_info=True)
        sys.exit(1)


def create_sample_dataset(dataset_path):
    """创建示例数据集"""
    sample_data = [
        {
            "instruction": "回答以下关于异常检测的问题",
            "input": "什么是异常检测？它在IT运维中有什么应用？",
            "output": "异常检测是一种识别数据中不符合预期模式的数据点或观测值的技术。在IT运维中，异常检测可以用于监控系统性能、识别网络入侵、检测硬件故障、预测系统崩溃等。通过分析系统日志、网络流量、资源使用率等数据，异常检测算法可以及时发现潜在问题，帮助运维人员在问题扩大前采取措施。",
        },
        {
            "instruction": "解释以下概念",
            "input": "什么是根因分析？",
            "output": "根因分析是一种问题解决方法，旨在识别问题的根本原因，而不仅仅是表面现象。在IT运维中，当系统出现故障时，根因分析帮助团队找出导致问题的真正原因，而不是简单地处理症状。这种方法通常涉及系统地分析故障链，使用技术如'5个为什么'、鱼骨图或故障树分析，以确保问题不会再次发生。有效的根因分析可以减少系统停机时间，提高服务可靠性。",
        },
        {
            "instruction": "回答关于知识图谱的问题",
            "input": "知识图谱是什么？它如何帮助IT运维？",
            "output": "知识图谱是一种表示知识的结构化数据形式，它以图的形式存储实体之间的关系。在知识图谱中，节点代表实体（如服务器、应用程序、网络设备等），边代表实体之间的关系（如'部署在'、'依赖于'等）。在IT运维中，知识图谱可以帮助理解复杂系统的组件关系，支持根因分析，加速故障排除，实现智能告警关联，并为自动化决策提供基础。通过将分散的IT知识整合到统一的图谱中，运维团队可以更快地解决问题并预防潜在故障。",
        },
    ]

    # 确保目录存在
    os.makedirs(os.path.dirname(dataset_path), exist_ok=True)

    # 写入示例数据
    with open(dataset_path, "w", encoding="utf-8") as f:
        json.dump(sample_data, f, ensure_ascii=False, indent=2)

    logger.info(f"示例数据集已创建: {dataset_path}")


def main():
    parser = argparse.ArgumentParser(description="训练和测试QA助手")
    parser.add_argument(
        "--action",
        type=str,
        choices=["train", "test", "both"],
        default="both",
        help="执行操作: train(训练), test(测试), both(两者)",
    )
    parser.add_argument("--base_model", type=str, required=True, help="基础模型路径")
    parser.add_argument(
        "--output_dir",
        type=str,
        default="./models/finetuned-deepseek-qa",
        help="微调模型输出目录",
    )
    parser.add_argument(
        "--dataset",
        type=str,
        default="./models/data/qa_dataset.json",
        help="数据集路径",
    )
    parser.add_argument(
        "--create-sample", action="store_true", help="如果数据集不存在，创建示例数据集"
    )
    parser.add_argument("--test_questions", type=str, help="测试问题JSON文件路径")
    parser.add_argument("--epochs", type=int, default=3, help="训练轮数")
    parser.add_argument("--learning_rate", type=float, default=5e-5, help="学习率")
    parser.add_argument("--batch_size", type=int, default=4, help="训练批次大小")
    parser.add_argument("--lora_r", type=int, default=8, help="LoRA r参数")
    parser.add_argument("--max_length", type=int, default=1024, help="最大序列长度")

    args = parser.parse_args()

    # 检查基础模型是否存在
    if not os.path.exists(args.base_model):
        logger.error(f"基础模型路径不存在: {args.base_model}")
        sys.exit(1)

    if args.action in ["train", "both"]:
        train_model(args)

    if args.action in ["test", "both"]:
        test_model(args)


if __name__ == "__main__":
    main()
