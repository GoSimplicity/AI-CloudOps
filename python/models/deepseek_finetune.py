import os
import json
import torch
import logging
from transformers import (
    AutoModelForCausalLM,
    AutoTokenizer,
    TrainingArguments,
    Trainer,
    DataCollatorForLanguageModeling,
    set_seed,
    BitsAndBytesConfig,
)
from datasets import load_dataset, Dataset
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training
import numpy as np
from typing import Dict, List, Optional, Union, Any
from datetime import datetime
import gc
import argparse

# 配置日志记录
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler("training.log")
    ]
)
logger = logging.getLogger(__name__)

class DeepSeekFineTuner:
    """
    DeepSeek模型微调器
    
    使用LoRA方法对DeepSeek等大型语言模型进行高效微调，支持多种配置选项和训练模式。
    """
    
    def __init__(
        self, 
        # 基本配置
        model_name: str = "./models/qa-model",                         # 预训练模型路径或名称
        output_dir: str = "./models/finetuned-deepseek-qa",            # 输出目录
        dataset_path: str = "./models/qa_dataset",                     # 数据集路径
        
        # 模型加载配置
        load_in_8bit: bool = False,                                    # 是否以8bit精度加载模型
        load_in_4bit: bool = True,                                    # 是否以4bit精度加载模型
        use_gpu: bool = True,                                          # 是否使用GPU
        gpu_ids: Optional[List[int]] = None,                           # 指定使用的GPU ID列表
        
        # LoRA配置
        lora_r: int = 8,                                               # LoRA秩
        lora_alpha: int = 16,                                          # LoRA缩放参数
        lora_dropout: float = 0.1,                                     # LoRA dropout概率
        target_modules: Optional[List[str]] = None,                    # 目标模块列表
        
        # 训练配置
        learning_rate: float = 3e-5,                                   # 学习率
        num_train_epochs: int = 10,                                     # 训练轮数
        per_device_train_batch_size: int = 1,                          # 每设备训练批大小
        gradient_accumulation_steps: int = 16,                         # 梯度累积步数
        save_steps: int = 100,                                         # 保存检查点步数
        logging_steps: int = 10,                                       # 日志记录步数
        max_seq_length: int = 1024,                                    # 最大序列长度
        warmup_ratio: float = 0.1,                                     # 预热比例
        weight_decay: float = 0.01,                                    # 权重衰减
        
        # 高级配置
        seed: int = 42,                                                # 随机种子
        fp16: bool = False,                                            # 是否使用混合精度训练
        bf16: bool = False,                                            # 是否使用bfloat16精度
        gradient_checkpointing: bool = True,                          # 是否使用梯度检查点
        prompt_template: Optional[str] = None,                         # 自定义提示模板
        save_total_limit: int = 3,                                     # 保存的检查点总数限制
        report_to: Union[List[str], str] = "tensorboard",              # 报告工具
        eval_strategy: Optional[str] = None,                           # 评估策略
        eval_steps: Optional[int] = None,                              # 评估步数
        max_grad_norm: float = 1.0,                                    # 最大梯度范数
        early_stopping_patience: Optional[int] = None,                 # 早停耐心值
        
        # 推理配置
        inference_ready: bool = True,                                  # 是否准备推理就绪模型
        merge_weights: bool = False                                    # 是否合并权重
    ):
        """
        初始化DeepSeek模型微调器
        
        Args:
            model_name: 预训练模型的路径或Hugging Face模型ID
            output_dir: 保存微调模型的目录
            dataset_path: 训练数据集路径，可以是目录或单个JSON文件
            load_in_8bit: 是否以8bit精度加载模型以节省内存
            load_in_4bit: 是否以4bit精度加载模型以节省更多内存
            use_gpu: 是否使用GPU训练
            gpu_ids: 要使用的特定GPU ID列表，如[0,1]
            lora_r: LoRA适配器的秩
            lora_alpha: LoRA缩放参数
            lora_dropout: LoRA适配器的dropout率
            target_modules: 要应用LoRA的目标模块列表
            learning_rate: 学习率
            num_train_epochs: 训练轮数
            per_device_train_batch_size: 每个设备的训练批大小
            gradient_accumulation_steps: 梯度累积步数
            save_steps: 每多少步保存一次模型
            logging_steps: 每多少步记录一次日志
            max_seq_length: 最大序列长度
            warmup_ratio: 学习率预热比例
            weight_decay: 权重衰减
            seed: 随机种子
            fp16: 是否启用半精度训练
            bf16: 是否启用bfloat16训练
            gradient_checkpointing: 是否启用梯度检查点以节省内存
            prompt_template: 自定义提示模板
            save_total_limit: 保存的检查点总数限制
            report_to: 报告工具，如"tensorboard"
            eval_strategy: 评估策略
            eval_steps: 评估步数
            max_grad_norm: 最大梯度范数
            early_stopping_patience: 早停耐心值
            inference_ready: 训练后是否准备推理就绪模型
            merge_weights: 是否合并LoRA权重到基础模型
        """
        # 设置随机种子以确保可复现性
        set_seed(seed)
        self.seed = seed
        
        # 基本配置
        self.model_name = model_name
        self.output_dir = output_dir
        self.dataset_path = dataset_path
        
        # 模型加载配置
        self.load_in_8bit = load_in_8bit
        self.load_in_4bit = load_in_4bit
        self.use_gpu = use_gpu and torch.cuda.is_available()
        self.gpu_ids = gpu_ids if gpu_ids else list(range(torch.cuda.device_count()))
        
        # LoRA配置
        self.lora_r = lora_r
        self.lora_alpha = lora_alpha
        self.lora_dropout = lora_dropout
        # self.target_modules = target_modules or ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]
        self.target_modules = target_modules or ["q_proj", "v_proj"]
        
        # 训练配置
        self.learning_rate = learning_rate
        self.num_train_epochs = num_train_epochs
        self.per_device_train_batch_size = per_device_train_batch_size
        self.gradient_accumulation_steps = gradient_accumulation_steps
        self.save_steps = save_steps
        self.logging_steps = logging_steps
        self.max_seq_length = max_seq_length
        self.warmup_ratio = warmup_ratio
        self.weight_decay = weight_decay
        
        # 高级配置
        self.fp16 = fp16 and self.use_gpu
        self.bf16 = bf16 and self.use_gpu and torch.cuda.is_bf16_supported()
        self.gradient_checkpointing = gradient_checkpointing
        self.prompt_template = prompt_template
        self.save_total_limit = save_total_limit
        self.report_to = report_to
        self.eval_strategy = eval_strategy
        self.eval_steps = eval_steps
        self.max_grad_norm = max_grad_norm
        self.early_stopping_patience = early_stopping_patience
        
        # 推理配置
        self.inference_ready = inference_ready
        self.merge_weights = merge_weights
        
        # 实例变量初始化
        self.tokenizer = None
        self.model = None
        self.dataset = None
        self.processed_dataset = None
        self.training_args = None
        self.trainer = None
        
        # 创建输出目录
        os.makedirs(output_dir, exist_ok=True)
        
        # 创建以时间戳命名的运行目录
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.run_dir = os.path.join(output_dir, f"run_{timestamp}")
        os.makedirs(self.run_dir, exist_ok=True)
        
        # 保存配置
        self._save_config()
        
        # 记录初始化信息
        logger.info(f"DeepSeek微调器已初始化，使用模型: {model_name}")
        logger.info(f"输出目录: {self.run_dir}")
        if self.use_gpu:
            logger.info(f"使用GPU: {self.gpu_ids}")
        else:
            logger.info("使用CPU进行训练")
    
    def _save_config(self) -> None:
        """保存当前配置到JSON文件"""
        config = {
            "model_name": self.model_name,
            "output_dir": self.output_dir,
            "dataset_path": self.dataset_path,
            "load_in_8bit": self.load_in_8bit,
            "load_in_4bit": self.load_in_4bit,
            "use_gpu": self.use_gpu,
            "gpu_ids": self.gpu_ids,
            "lora_r": self.lora_r,
            "lora_alpha": self.lora_alpha,
            "lora_dropout": self.lora_dropout,
            "target_modules": self.target_modules,
            "learning_rate": self.learning_rate,
            "num_train_epochs": self.num_train_epochs,
            "per_device_train_batch_size": self.per_device_train_batch_size,
            "gradient_accumulation_steps": self.gradient_accumulation_steps,
            "save_steps": self.save_steps,
            "logging_steps": self.logging_steps,
            "max_seq_length": self.max_seq_length,
            "warmup_ratio": self.warmup_ratio,
            "weight_decay": self.weight_decay,
            "seed": self.seed,
            "fp16": self.fp16,
            "bf16": self.bf16,
            "gradient_checkpointing": self.gradient_checkpointing,
            "save_total_limit": self.save_total_limit,
            "report_to": self.report_to,
            "eval_strategy": self.eval_strategy,
            "eval_steps": self.eval_steps,
            "max_grad_norm": self.max_grad_norm,
            "early_stopping_patience": self.early_stopping_patience,
            "inference_ready": self.inference_ready,
            "merge_weights": self.merge_weights,
            "timestamp": datetime.now().isoformat()
        }
        
        config_path = os.path.join(self.run_dir, "config.json")
        with open(config_path, 'w') as f:
            json.dump(config, f, indent=2)
        logger.info(f"配置已保存到 {config_path}")
    
    def load_model_and_tokenizer(self) -> None:
        """
        加载模型和分词器
        
        根据配置加载预训练模型和对应的分词器，支持不同精度和设备配置。
        """
        logger.info(f"正在从 {self.model_name} 加载模型...")
        
        # 释放内存
        gc.collect()
        torch.cuda.empty_cache()
        
        # 设置模型加载参数
        model_kwargs = {}
        
        # 设置设备映射
        if self.use_gpu:
            if len(self.gpu_ids) > 1:
                device_map = "auto"  # 自动分配到多个GPU
            else:
                device_map = f"cuda:{self.gpu_ids[0]}"
        else:
            device_map = "cpu"
        
        model_kwargs["device_map"] = device_map
        
        # 设置量化参数
        if self.load_in_8bit:
            model_kwargs["load_in_8bit"] = True
        elif self.load_in_4bit:
            model_kwargs["load_in_4bit"] = True
            model_kwargs["quantization_config"] = BitsAndBytesConfig(
                load_in_4bit=True,
                bnb_4bit_compute_dtype=torch.float16,
                bnb_4bit_use_double_quant=True,
                bnb_4bit_quant_type="nf4"
            )
            # 移除冗余的load_in_4bit参数
            model_kwargs.pop("load_in_4bit")
        else:
            # 设置数据类型
            if self.bf16:
                model_kwargs["torch_dtype"] = torch.bfloat16
            elif self.fp16:
                model_kwargs["torch_dtype"] = torch.float16
            else:
                model_kwargs["torch_dtype"] = torch.float16  # 默认使用半精度而非全精度
        
        # 加载分词器
        try:
            self.tokenizer = AutoTokenizer.from_pretrained(self.model_name)
            logger.info("分词器加载成功")
        except Exception as e:
            logger.error(f"加载分词器时出错: {e}")
            raise
        
        # 配置分词器
        if not self.tokenizer.pad_token_id:
            self.tokenizer.pad_token_id = self.tokenizer.eos_token_id
            logger.info("已将pad_token_id设置为eos_token_id")
        
        if is_flash_attn_2_available():
            model_kwargs["use_flash_attention_2"] = True
            logger.info("Flash Attention 2 已启用")
        else:
            logger.warning("当前环境不支持 Flash Attention 2，将回退到普通注意力机制")
        
        # 加载模型，分批加载以减少内存峰值
        try:
            # 确保使用低内存配置
            model_kwargs["low_cpu_mem_usage"] = True
            
            # 动态调整显存分配
            if self.use_gpu:
                max_memory = {}
                for gpu_id in self.gpu_ids:
                    total_memory = torch.cuda.get_device_properties(gpu_id).total_memory
                    reserved_memory = int(total_memory * 0.8)  # 预留 20% 显存
                    max_memory[f"cuda:{gpu_id}"] = f"{reserved_memory // (1024 ** 3)}GiB"
                max_memory["cpu"] = "20GB"
                model_kwargs["max_memory"] = max_memory
                logger.info(f"显存分配: {max_memory}")
            
            self.model = AutoModelForCausalLM.from_pretrained(
                self.model_name, 
                **model_kwargs
            )
            logger.info(f"模型加载成功，配置: {model_kwargs}")
        except Exception as e:
            logger.error(f"加载模型时出错: {e}")
            raise
        
        # 为量化模型准备LoRA训练
        if self.load_in_8bit or self.load_in_4bit:
            self.model = prepare_model_for_kbit_training(self.model)
            logger.info("模型已准备好进行量化训练")
        
        # 启用梯度检查点以节省内存
        if self.gradient_checkpointing:
            self.model.gradient_checkpointing_enable()
            logger.info("已启用梯度检查点")
            
    def prepare_lora_config(self) -> None:
        """
        配置LoRA参数
        
        设置LoRA适配器配置并将其应用到模型，使其可训练。
        """
        logger.info("正在配置LoRA参数...")
        
        # 定义LoRA配置
        lora_config = LoraConfig(
            r=self.lora_r,
            lora_alpha=self.lora_alpha,
            lora_dropout=self.lora_dropout,
            bias="none",
            task_type="CAUSAL_LM",
            target_modules=self.target_modules
        )
        
        # 应用LoRA适配器到模型
        self.model = get_peft_model(self.model, lora_config)
        
        # 打印可训练参数信息
        trainable_params, all_params = self.model.get_nb_trainable_parameters()
        logger.info(f"可训练参数: {trainable_params:,} ({trainable_params/all_params:.2%}，总参数: {all_params:,})")
    def prepare_dataset(self) -> None:
        """
        准备训练数据集
        
        加载、处理和格式化数据集以适应模型训练。
        """
        logger.info(f"正在从 {self.dataset_path} 准备数据集...")

        # 加载数据集
        try:
            if os.path.isdir(self.dataset_path):
                # 从目录加载多个JSON文件
                data = []
                json_files = [f for f in os.listdir(self.dataset_path) if f.endswith('.json')]
                logger.info(f"在数据集目录中找到 {len(json_files)} 个JSON文件")
                
                for filename in json_files:
                    file_path = os.path.join(self.dataset_path, filename)
                    try:
                        with open(file_path, 'r', encoding='utf-8') as f:
                            file_data = json.load(f)
                            # 处理不同格式的JSON文件
                            if isinstance(file_data, list):
                                data.extend(file_data)
                            elif isinstance(file_data, dict) and 'data' in file_data:
                                data.extend(file_data['data'])
                        logger.info(f"从 {filename} 加载了 {len(data)} 个样本")
                    except Exception as e:
                        logger.warning(f"加载 {filename} 时出错: {e}")
                
                if not data:
                    raise ValueError("在数据集目录中未找到有效数据")
                
                self.dataset = Dataset.from_dict({
                    "instruction": [d.get("instruction", "") for d in data], 
                    "input": [d.get("input", "") for d in data],
                    "output": [d.get("output", "") for d in data],
                })
            else:
                # 如果是一个单独的JSON文件
                with open(self.dataset_path, 'r', encoding='utf-8') as f:
                    file_data = json.load(f)
                    if isinstance(file_data, list):
                        data = file_data
                    elif isinstance(file_data, dict) and 'data' in file_data:
                        data = file_data['data']
                    else:
                        data = [file_data]
                
                self.dataset = Dataset.from_dict({
                    "instruction": [d.get("instruction", "") for d in data], 
                    "input": [d.get("input", "") for d in data],
                    "output": [d.get("output", "") for d in data],
                })
                logger.info(f"从单个文件加载数据集: {self.dataset_path}")
        except Exception as e:
            logger.error(f"加载数据集时出错: {e}")
            raise
        
        logger.info(f"数据集加载完成，共 {len(self.dataset)} 个样本")
        
        # 检查数据集格式
        if not all(col in self.dataset.column_names for col in ["instruction", "output"]):
            logger.warning("数据集缺少必要的列。需要'instruction'和'output'列")
            
        # 数据预处理 - 分批处理以减少内存占用
        try:
            # 设置较小的批处理大小
            batch_size = 32  # 小批量处理以减少内存使用
            
            def process_batch(examples):
                return self.preprocess_function(examples)
            
            self.processed_dataset = self.dataset.map(
                process_batch,
                batched=True,
                batch_size=batch_size,
                remove_columns=self.dataset.column_names,
                desc="处理数据集"
            )
            logger.info(f"数据集处理成功，共 {len(self.processed_dataset)} 个样本")
        except Exception as e:
            logger.error(f"处理数据集时出错: {e}")
            raise
        
        # 清理内存
        gc.collect()
        torch.cuda.empty_cache()
        
        # 输出一些样本数据以便检查
        logger.info("处理后数据示例:")
        sample_idx = min(0, len(self.processed_dataset) - 1)
        sample = {k: v[:100] if isinstance(v, str) else v for k, v in self.dataset[sample_idx].items()}
        logger.info(f"原始数据: {sample}")
        
    def preprocess_function(self, examples: Dict[str, List]) -> Dict[str, List]:
        """
        将数据转换为适合模型训练的格式
        
        Args:
            examples: 包含多个样本的字典
            
        Returns:
            处理后的样本字典，包含输入ID和标签
        """
        instructions = examples.get("instruction", [""] * len(examples.get("output", [])))
        inputs = examples.get("input", [""] * len(instructions))
        outputs = examples.get("output", [""] * len(instructions))
        
        processed_texts = []
        
        # 使用自定义或默认模板格式化输入
        for instruction, inp, output in zip(instructions, inputs, outputs):
            if self.prompt_template:
                # 使用自定义模板
                text = self.prompt_template.format(
                    instruction=instruction,
                    input=inp,
                    output=output
                )
            else:
                # 使用默认模板
                if inp:
                    prompt = f"人类: {instruction}\n\n{inp}\n\nAI:"
                else:
                    prompt = f"人类: {instruction}\n\nAI:"
                    
                text = f"{prompt} {output}</s>"
            
            processed_texts.append(text)
            
        # 使用tokenizer处理文本，使用较短的序列长度以减少内存使用
        try:
            tokenized_inputs = self.tokenizer(
                processed_texts,
                padding="max_length",
                truncation=True,
                max_length=min(self.max_seq_length, 2048),  # 限制序列长度
                return_tensors="pt"
            )
        except Exception as e:
            logger.error(f"分词过程中出错: {e}")
            raise
        
        # 准备标签（与输入相同，因为是自回归任务）
        tokenized_inputs["labels"] = tokenized_inputs["input_ids"].clone()
        
        # 将padding token的标签设为-100，在计算损失时被忽略
        tokenized_inputs["labels"][tokenized_inputs["input_ids"] == self.tokenizer.pad_token_id] = -100
        
        return tokenized_inputs
    
    def configure_training_args(self) -> TrainingArguments:
        """
        配置训练参数
        
        Returns:
            训练参数对象
        """
        logger.info("正在配置训练参数...")
        
        # 设置评估策略
        evaluation_strategy = "no"
        if self.eval_strategy:
            evaluation_strategy = self.eval_strategy
        
        # 定义训练参数
        training_args = TrainingArguments(
            output_dir=self.run_dir,
            learning_rate=self.learning_rate,
            num_train_epochs=self.num_train_epochs,
            per_device_train_batch_size=self.per_device_train_batch_size,
            gradient_accumulation_steps=self.gradient_accumulation_steps,
            save_steps=self.save_steps,
            logging_steps=self.logging_steps,
            save_total_limit=self.save_total_limit,
            remove_unused_columns=False,
            push_to_hub=False,
            report_to=self.report_to,
            fp16=self.fp16,
            bf16=self.bf16,
            evaluation_strategy=evaluation_strategy,
            eval_steps=self.eval_steps,
            load_best_model_at_end=self.eval_strategy is not None,
            warmup_ratio=self.warmup_ratio,
            weight_decay=self.weight_decay,
            max_grad_norm=self.max_grad_norm,
            dataloader_num_workers=2,  # 减少worker数量以降低内存使用
            group_by_length=True,  # 对相似长度序列分组以提高效率
            seed=self.seed,
            optim="adamw_torch",  # 使用torch版本的AdamW优化器更节省内存
            ddp_find_unused_parameters=False,  # 提高分布式训练效率
            gradient_checkpointing=self.gradient_checkpointing,  # 确保梯度检查点在训练参数中也启用
            deepspeed=None,  # 如需进一步优化，可以配置DeepSpeed
        )
        
        self.training_args = training_args
        logger.info(f"训练参数配置完成: {training_args}")
        return training_args
        
    def train(self) -> None:
        """
        训练模型
        
        执行完整的微调流程：加载模型和分词器，准备数据集，配置训练参数并启动训练。
        """
        try:
            # 加载模型和分词器
            self.load_model_and_tokenizer()
            
            # 准备LoRA配置
            self.prepare_lora_config()
            
            # 准备数据集
            self.prepare_dataset()
            
            # 配置训练参数
            self.configure_training_args()
            
            # 数据整理器
            data_collator = DataCollatorForLanguageModeling(
                tokenizer=self.tokenizer, 
                mlm=False
            )
            
            # 创建Trainer
            self.trainer = Trainer(
                model=self.model,
                args=self.training_args,
                train_dataset=self.processed_dataset,
                data_collator=data_collator,
                tokenizer=self.tokenizer,
            )
            
            # 开始训练
            logger.info("开始训练...")
            self.trainer.train()
            
            # 保存最终模型
            final_model_path = os.path.join(self.run_dir, "final_model")
            os.makedirs(final_model_path, exist_ok=True)
            
            # 保存训练结果
            self.model.save_pretrained(final_model_path)
            self.tokenizer.save_pretrained(final_model_path)
            logger.info(f"模型和分词器已保存到 {final_model_path}")
            
            # 如果需要，创建推理就绪模型
            if self.inference_ready:
                self._prepare_inference_model()
                
            logger.info("训练成功完成")
            
            # 最终清理内存
            del self.model
            del self.trainer
            gc.collect()
            torch.cuda.empty_cache()
            
        except Exception as e:
            logger.error(f"训练过程中出错: {e}", exc_info=True)
            # 尝试清理内存
            if hasattr(self, 'model') and self.model is not None:
                del self.model
            if hasattr(self, 'trainer') and self.trainer is not None:
                del self.trainer
            gc.collect()
            torch.cuda.empty_cache()
            raise
    
    def _prepare_inference_model(self) -> None:
        """准备推理就绪的模型"""
        logger.info("正在准备推理就绪模型...")
        
        inference_path = os.path.join(self.run_dir, "inference_model")
        os.makedirs(inference_path, exist_ok=True)
        
        # 如果需要合并权重
        if self.merge_weights:
            logger.info("正在将LoRA权重与基础模型合并...")
            # 解放内存
            gc.collect()
            torch.cuda.empty_cache()
            
            try:
                # 合并LoRA权重
                merged_model = self.model.merge_and_unload()
                
                # 保存合并后的模型
                merged_model.save_pretrained(inference_path)
                self.tokenizer.save_pretrained(inference_path)
                logger.info(f"合并后的模型已保存到 {inference_path}")
                
                # 清理内存
                del merged_model
                gc.collect()
                torch.cuda.empty_cache()
            except Exception as e:
                logger.error(f"合并权重时出错: {e}")
                logger.info("回退到仅保存适配器权重")
                self.model.save_pretrained(inference_path)
                self.tokenizer.save_pretrained(inference_path)
        else:
            # 只保存适配器权重
            self.model.save_pretrained(inference_path)
            self.tokenizer.save_pretrained(inference_path)
            logger.info(f"适配器权重已保存到 {inference_path}")
            
        # 保存inference.py示例脚本
        self._create_inference_script(inference_path)
            
    def _create_inference_script(self, path: str) -> None:
        """创建推理脚本示例"""
        script = """
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel, PeftConfig

# 加载模型和分词器
def load_model(model_path, load_in_8bit=False, load_in_4bit=False, device="auto"):
    # 确定设备
    if device == "auto":
        device = "cuda" if torch.cuda.is_available() else "cpu"
    
    # 加载配置和分词器
    tokenizer = AutoTokenizer.from_pretrained(model_path)
    
    # 设置模型加载选项
    model_kwargs = {"device_map": device}
    if load_in_8bit:
        model_kwargs["load_in_8bit"] = True
    elif load_in_4bit:
        model_kwargs["load_in_4bit"] = True
    else:
        model_kwargs["torch_dtype"] = torch.float16 if device == "cuda" else torch.float32
    
    try:
        # 尝试直接加载模型（如果是合并的模型）
        model = AutoModelForCausalLM.from_pretrained(model_path, **model_kwargs)
        print("已加载完整模型")
    except:
        # 如果失败，尝试作为LoRA适配器加载
        print("正在加载LoRA适配器")
        config = PeftConfig.from_pretrained(model_path)
        model = AutoModelForCausalLM.from_pretrained(
            config.base_model_name_or_path, **model_kwargs
        )
        model = PeftModel.from_pretrained(model, model_path)
    
    return model, tokenizer

# 生成回复
def generate_response(model, tokenizer, instruction, input_text="", max_length=1024, temperature=0.7, top_p=0.9):
    # 格式化提示
    if input_text:
        prompt = f"人类: {instruction}\\n\\n{input_text}\\n\\nAI:"
    else:
        prompt = f"人类: {instruction}\\n\\nAI:"
    
    # 生成参数
    gen_kwargs = {
        "max_length": max_length,
        "temperature": temperature,
        "top_p": top_p,
        "do_sample": temperature > 0,
        "pad_token_id": tokenizer.pad_token_id or tokenizer.eos_token_id
    }
    
    # 编码输入
    inputs = tokenizer(prompt, return_tensors="pt")
    inputs = {k: v.to(model.device) for k, v in inputs.items()}
    
    # 生成回复
    with torch.no_grad():
        outputs = model.generate(**inputs, **gen_kwargs)
    
    # 解码输出
    full_response = tokenizer.decode(outputs[0], skip_special_tokens=True)
    
    # 提取AI回复部分
    response = full_response.split("AI:")[-1].strip()
    return response

# 示例用法
if __name__ == "__main__":
    model_path = "."
    
    # 加载模型
    model, tokenizer = load_model(model_path)
    
    # 示例查询
    instruction = "请问AI+CloudOps的发起人是谁"
    response = generate_response(model, tokenizer, instruction)
    print(f"Query: {instruction}")
    print(f"Response: {response}")
    
    # 交互式模式
    print("\\nEnter 'quit' to exit")
    while True:
        user_input = input("\\nYour question: ")
        if user_input.lower() in ["quit", "exit", "q"]:
            break
        
        response = generate_response(model, tokenizer, user_input)
        print(f"Response: {response}")
"""
        
        with open(os.path.join(path, "inference.py"), 'w', encoding='utf-8') as f:
            f.write(script.strip())
        
        logger.info(f"Inference script created at {os.path.join(path, 'inference.py')}")

def parse_args():
    """解析命令行参数"""
    parser = argparse.ArgumentParser(description="Fine-tune DeepSeek models with LoRA")
    
    # 基本配置
    parser.add_argument("--model_name", type=str, default="./models/qa-model", 
                        help="Path to pre-trained model or model name")
    parser.add_argument("--output_dir", type=str, default="./models/finetuned-deepseek-qa", 
                        help="Directory to save the fine-tuned model")
    parser.add_argument("--dataset_path", type=str, default="./data", 
                        help="Path to dataset directory or file")
    
    # 模型加载配置
    parser.add_argument("--load_in_8bit", action="store_true", 
                        help="Load model in 8-bit precision")
    parser.add_argument("--load_in_4bit", action="store_true", 
                        help="Load model in 4-bit precision")
    parser.add_argument("--no_gpu", action="store_true", 
                        help="Disable GPU usage")
    parser.add_argument("--gpu_ids", type=int, nargs="+", default=None, 
                        help="Specific GPU IDs to use")
    
    # LoRA配置
    parser.add_argument("--lora_r", type=int, default=32, 
                        help="LoRA attention dimension")
    parser.add_argument("--lora_alpha", type=int, default=64, 
                        help="LoRA alpha parameter")
    parser.add_argument("--lora_dropout", type=float, default=0.1, 
                        help="LoRA dropout probability")
    
    # 训练配置
    parser.add_argument("--learning_rate", type=float, default=3e-5, 
                        help="Learning rate")
    parser.add_argument("--num_train_epochs", type=int, default=10, 
                        help="Number of training epochs")
    parser.add_argument("--per_device_train_batch_size", type=int, default=2, 
                        help="Training batch size per device")
    parser.add_argument("--gradient_accumulation_steps", type=int, default=8, 
                        help="Number of gradient accumulation steps")
    parser.add_argument("--max_seq_length", type=int, default=1024, 
                        help="Maximum sequence length")
    
    # 高级配置
    parser.add_argument("--seed", type=int, default=42, 
                        help="Random seed")
    parser.add_argument("--fp16", action="store_true", 
                        help="Enable mixed precision training with fp16")
    parser.add_argument("--bf16", action="store_true", 
                        help="Enable mixed precision training with bf16")
    parser.add_argument("--gradient_checkpointing", action="store_true", 
                        help="Enable gradient checkpointing")
    
    args = parser.parse_args()
    return args

if __name__ == "__main__":
    # 确保最大限度释放内存
    gc.collect()
    torch.cuda.empty_cache()
    
    # 解析命令行参数
    args = parse_args()
    
    # 创建微调器实例
    finetuner = DeepSeekFineTuner(
        model_name=args.model_name,
        output_dir=args.output_dir,
        dataset_path=args.dataset_path,
        load_in_8bit=args.load_in_8bit,
        load_in_4bit=args.load_in_4bit or True,  # 默认使用4bit量化
        use_gpu=not args.no_gpu,
        gpu_ids=args.gpu_ids,
        lora_r=args.lora_r,
        lora_alpha=args.lora_alpha,
        lora_dropout=args.lora_dropout,
        learning_rate=args.learning_rate,
        num_train_epochs=args.num_train_epochs,
        per_device_train_batch_size=args.per_device_train_batch_size or 1,  # 默认批大小为1
        gradient_accumulation_steps=args.gradient_accumulation_steps or 16, # 默认累积16步
        max_seq_length=args.max_seq_length,
        seed=args.seed,
        fp16=args.fp16,
        bf16=args.bf16,
        gradient_checkpointing=args.gradient_checkpointing or True  # 默认启用梯度检查点
    )
    
    # 开始训练
    finetuner.train()

