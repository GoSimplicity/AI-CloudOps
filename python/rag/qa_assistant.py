import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from models.deepseek_inference import DeepSeekQAModel
from rag.retriever import Retriever
from rag.vector_store import VectorStore
from typing import List, Dict, Any, Optional

class QAAssistant:
    def __init__(
        self,
        model_path: str = "./models/ollama/deepseek-r1-8b",
        adapter_path: str = "./models/finetuned-deepseek-qa",
        vector_store_path: str = "./rag/vector_store",
        retrieval_top_k: int = 5,
        use_retrieval: bool = True
    ):
        self.model = DeepSeekQAModel(
            base_model_path=model_path,
            adapter_path=adapter_path
        )
        
        self.use_retrieval = use_retrieval
        if use_retrieval:
            self.vector_store = VectorStore(vector_store_path)
            self.retriever = Retriever(self.vector_store, top_k=retrieval_top_k)
        
    def answer_question(
        self, 
        question: str,
        context: Optional[str] = None,
        retrieval_filters: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        回答用户的问题
        
        Args:
            question: 用户问题
            context: 可选的附加上下文
            retrieval_filters: 检索过滤条件
            
        Returns:
            包含答案和检索信息的字典
        """
        retrieved_docs = []
        
        # 检索相关文档（如果启用）
        if self.use_retrieval:
            retrieved_docs = self.retriever.retrieve(
                query=question,
                filters=retrieval_filters
            )
            
            # 准备检索内容作为上下文
            if retrieved_docs:
                if context:
                    context += "\n\n参考信息:\n" + "\n".join([doc["content"] for doc in retrieved_docs])
                else:
                    context = "参考信息:\n" + "\n".join([doc["content"] for doc in retrieved_docs])
        
        # 生成回答
        response = self.model.generate_response(question, context or "")
        
        # 返回结果
        result = {
            "question": question,
            "answer": response,
            "retrieved_documents": retrieved_docs if self.use_retrieval else []
        }
        
        return result
    
    def train_on_feedback(
        self,
        question: str,
        correct_answer: str,
        feedback_data_path: str = "./models/data/qa_dataset/feedback.json"
    ) -> None:
        """
        基于用户反馈添加训练数据，用于未来的模型更新
        
        Args:
            question: 用户问题
            correct_answer: 正确答案
            feedback_data_path: 反馈数据存储路径
        """
        import json
        
        # 确保目录存在
        os.makedirs(os.path.dirname(feedback_data_path), exist_ok=True)
        
        # 准备新的训练样本
        new_sample = {
            "instruction": question,
            "input": "",
            "output": correct_answer
        }
        
        # 添加到现有数据或创建新文件
        if os.path.exists(feedback_data_path):
            try:
                with open(feedback_data_path, 'r', encoding='utf-8') as f:
                    existing_data = json.load(f)
            except json.JSONDecodeError:
                existing_data = []
        else:
            existing_data = []
            
        existing_data.append(new_sample)
        
        # 保存更新后的数据
        with open(feedback_data_path, 'w', encoding='utf-8') as f:
            json.dump(existing_data, f, ensure_ascii=False, indent=2)
            
        print(f"Feedback saved to {feedback_data_path}. Use this data for future model updates.")