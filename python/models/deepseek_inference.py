import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
import os

class DeepSeekQAModel:
    def __init__(
        self,
        base_model_path="./models/finetuned-model/run_20250303_181040/final_model",
        adapter_path="./models/finetuned-model/run_20250303_181040/inference_model",
        device="cuda" if torch.cuda.is_available() else "cpu",
        max_length=2048,
        temperature=0.7,
        top_p=0.9,
    ):
        self.base_model_path = base_model_path
        self.adapter_path = adapter_path
        self.device = device
        self.max_length = max_length
        self.temperature = temperature
        self.top_p = top_p
        
        self.load_model()
        
    def load_model(self):
        """加载模型和分词器"""
        print(f"Loading base model from {self.base_model_path}...")
        
        # 加载分词器
        self.tokenizer = AutoTokenizer.from_pretrained(self.base_model_path)
        
        # 加载模型
        model_kwargs = {
            "torch_dtype": torch.bfloat16 if torch.cuda.is_available() else torch.float32,
            "device_map": "auto" if torch.cuda.is_available() else None,
        }
        self.model = AutoModelForCausalLM.from_pretrained(
            self.base_model_path,
            **model_kwargs
        )
        
        # 加载微调适配器
        if os.path.exists(self.adapter_path):
            print(f"Loading adapters from {self.adapter_path}...")
            self.model = PeftModel.from_pretrained(self.model, self.adapter_path)
            
        # 配置tokenizer
        if not self.tokenizer.pad_token_id:
            self.tokenizer.pad_token_id = self.tokenizer.eos_token_id
            
        self.model.eval()
        print("Model loaded successfully")
    
    def generate_response(self, instruction, input_text=""):
        """生成回答"""
        # 构建提示
        if input_text:
            prompt = f"Human: {instruction}\n\n{input_text}\n\nAssistant:"
        else:
            prompt = f"Human: {instruction}\n\nAssistant:"
            
        # 编码输入
        inputs = self.tokenizer(prompt, return_tensors="pt").to(self.device)
        
        # 生成输出
        with torch.no_grad():
            outputs = self.model.generate(
                **inputs,
                max_length=self.max_length,
                do_sample=True,
                temperature=self.temperature,
                top_p=self.top_p,
                pad_token_id=self.tokenizer.eos_token_id
            )
            
        # 解码输出
        response = self.tokenizer.decode(outputs[0], skip_special_tokens=True)
        
        # 提取回答部分
        response = response.split("Assistant:")[-1].strip()
        
        return response


if __name__ == "__main__":
    # 测试模型
    qa_model = DeepSeekQAModel()
    
    # 示例查询
    instruction = "请问AI+CloudOps的作者是谁？"
    response = qa_model.generate_response(instruction)
    
    print(f"Q: {instruction}")
    print(f"A: {response}")