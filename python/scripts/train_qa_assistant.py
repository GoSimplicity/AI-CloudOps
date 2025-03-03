import os
import sys
import argparse
import logging

sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from models.deepseek_finetune import DeepSeekFineTuner
from rag.qa_assistant import QAAssistant

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def train_model(args):
    """训练问答模型"""
    logger.info("Initializing fine-tuning process...")
    
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
    
    logger.info("Starting training...")
    finetuner.train()
    logger.info(f"Training completed. Model saved to {args.output_dir}")

def test_model(args):
    """测试微调后的模型"""
    logger.info("Loading QA Assistant for testing...")
    
    assistant = QAAssistant(
        model_path=args.base_model,
        adapter_path=args.output_dir,
        use_retrieval=False
    )
    
    # 测试问题
    test_questions = [
        "什么是异常检测?",
        "根因分析如何帮助解决问题?",
        "解释下知识图谱是如何工作的",
    ]
    
    for question in test_questions:
        logger.info(f"Testing question: {question}")
        response = assistant.answer_question(question)
        logger.info(f"Answer: {response['answer']}")
        logger.info("-" * 50)

def main():
    parser = argparse.ArgumentParser(description="Train and test QA assistant")
    parser.add_argument("--action", type=str, choices=["train", "test", "both"], default="both",
                       help="Action to perform: train, test, or both")
    parser.add_argument("--base_model", type=str, required=True,
                       help="Path to base model")
    parser.add_argument("--output_dir", type=str, default="./models/finetuned-deepseek-qa",
                       help="Output directory for fine-tuned model")
    parser.add_argument("--dataset", type=str, default="./models/data/qa_dataset",
                       help="Path to dataset")
    parser.add_argument("--epochs", type=int, default=3,
                       help="Number of training epochs")
    parser.add_argument("--learning_rate", type=float, default=5e-5,
                       help="Learning rate")
    parser.add_argument("--batch_size", type=int, default=4,
                       help="Training batch size")
    parser.add_argument("--lora_r", type=int, default=8,
                       help="LoRA r parameter")
    parser.add_argument("--max_length", type=int, default=1024,
                       help="Maximum sequence length")
    
    args = parser.parse_args()
    
    if args.action in ["train", "both"]:
        train_model(args)
    
    if args.action in ["test", "both"]:
        test_model(args)

if __name__ == "__main__":
    main()