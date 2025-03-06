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

import logging
from typing import List, Dict, Any, Optional

class Generator:
    """生成器，基于检索结果生成回答"""

    def __init__(
        self,
        llm_provider,
        system_prompt: str = None,
        max_tokens: int = 2000,
        temperature: float = 0.3
    ):
        """
        初始化生成器

        Args:
            llm_provider: 语言模型提供商实例
            system_prompt: 系统提示语
            max_tokens: 最大生成token数
            temperature: 生成温度
        """
        self.llm_provider = llm_provider
        self.max_tokens = max_tokens
        self.temperature = temperature
        self.logger = logging.getLogger(__name__)

        # 默认系统提示语
        self.system_prompt = system_prompt or """
        你是一个AIOps系统的智能助手，专注于帮助用户解决IT运维和系统监控相关问题。
        使用以下检索到的文档内容回答用户问题。如果检索内容不足以回答问题，请清晰地说明。
        不要编造信息，保持回答的准确性和专业性。
        """

    def format_context(self, documents: List[Dict[str, Any]]) -> str:
        """
        格式化检索文档作为上下文

        Args:
            documents: 检索到的文档列表

        Returns:
            格式化后的上下文字符串
        """
        if not documents:
            return "未找到相关文档。"

        context_parts = []
        for i, doc in enumerate(documents, 1):
            metadata = doc.get("metadata", {})
            source = metadata.get("source", "未知来源")
            content = doc.get("content", "").strip()

            # 添加块ID和置信度分数
            chunk_id = metadata.get("chunk_id", "?")
            score = doc.get("score", 0)

            context_parts.append(f"[文档 {i}] (来源: {source}, ID: {chunk_id}, 相关度: {score:.2f})\n{content}\n")

        return "\n".join(context_parts)

    def generate_simple_answer(
        self,
        query: str,
        retrieved_docs: List[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """
        生成简单回答
        Args:
            query: 用户问题
            retrieved_docs: 检索到的文档
        Returns:
            生成的简单回答
        """
        if not retrieved_docs:
            return {
                "answer": "抱歉，我没有找到相关信息。",
                "sources": []
            }
        
        # 构建简单提示
        context = self.format_context(retrieved_docs)
        
        prompt = f"""
        根据以下参考信息，直接回答问题。如果找到明确的答案，请使用简洁的格式回答，例如"答案是：XXX"。
        如果没有找到明确答案，请回答"抱歉，我无法在提供的信息中找到答案"。
        
        ### 参考信息:
        {context}
        
        ### 问题:
        {query}
        """
        
        try:
            # 使用更简单的系统提示
            simple_system_prompt = """
            你是一个精确的问答助手。请直接从提供的参考信息中提取答案，不要添加额外解释。
            如果参考信息中包含明确的答案，请以"[答案]：XXX"的格式回答。
            如果参考信息中没有明确的答案，请回答"未找到相关信息"。
            """
            
            # 生成回答
            response = self.llm_provider.generate(prompt=prompt)
            
            result = {
                "answer": response.strip(),
                "sources": [doc.get("metadata", {}).get("source", "未知来源") for doc in retrieved_docs]
            }
            
            return result
        except Exception as e:
            self.logger.error(f"生成简单回答时出错: {e}")
            return {
                "answer": "抱歉，生成回答时发生错误。",
                "sources": [doc.get("metadata", {}).get("source", "未知来源") for doc in retrieved_docs]
            }
    def generate_answer(
        self,
        query: str,
        retrieved_docs: List[Dict[str, Any]],
        chat_history: Optional[List[Dict[str, str]]] = None
    ) -> Dict[str, Any]:
        """
        生成回答

        Args:
            query: 用户问题
            retrieved_docs: 检索到的文档
            chat_history: 对话历史

        Returns:
            生成的回答和相关信息
        """
        # 格式化上下文
        context = self.format_context(retrieved_docs)

        # 构建提示
        if chat_history and len(chat_history) > 0:
            # 历史对话格式化
            history_text = "\n".join([
                f"用户: {msg['user']}\n助手: {msg['assistant']}"
                for msg in chat_history[-5:]  # 保留最近5轮对话
            ])

            prompt = f"""
            ### 对话历史:
            {history_text}

            ### 参考信息:
            {context}

            ### 当前问题:
            {query}
            """
        else:
            prompt = f"""
            ### 参考信息:
            {context}

            ### 问题:
            {query}
            """

        self.logger.debug(f"Generated prompt with {len(retrieved_docs)} documents")

        # 生成回答
        try:
            response = self.llm_provider.generate_response(
                prompt=prompt,
                system_prompt=self.system_prompt,
                max_tokens=self.max_tokens,
                temperature=self.temperature
            )

            # 构造返回结果
            result = {
                "query": query,
                "answer": response,
                "retrieved_documents": retrieved_docs,
                "sources": [doc.get("metadata", {}).get("source") for doc in retrieved_docs if "metadata" in doc]
            }

            return result
        except Exception as e:
            self.logger.error(f"Error generating response: {e}")
            return {
                "query": query,
                "answer": "抱歉，生成回答时发生错误。",
                "error": str(e),
                "retrieved_documents": retrieved_docs
            }

    def generate_structured_answer(
        self,
        query: str,
        retrieved_docs: List[Dict[str, Any]],
        output_format: str = "json"
    ) -> Dict[str, Any]:
        """
        生成结构化回答

        Args:
            query: 用户问题
            retrieved_docs: 检索到的文档
            output_format: 输出格式 ("json" 或 "markdown")

        Returns:
            结构化回答
        """
        # 格式化上下文
        context = self.format_context(retrieved_docs)

        # 构建结构化输出的提示语
        format_instruction = """
        以JSON格式输出回答，包含以下字段:
        1. "answer": 你的回答
        2. "reasoning": 推理过程
        3. "sources": 使用的参考来源
        4. "confidence": 答案的置信度 (0-1)
        """

        if output_format == "markdown":
            format_instruction = """
            以Markdown格式输出回答，包含以下部分:
            1. ## 回答
               你的详细回答

            2. ## 推理过程
               你的推理过程

            3. ## 参考来源
               使用的参考来源列表
            """

        prompt = f"""
        ### 参考信息:
        {context}

        ### 问题:
        {query}

        ### 输出格式要求:
        {format_instruction}
        """

        try:
            # 生成回答
            response = self.llm_provider.generate_response(
                prompt=prompt,
                system_prompt=self.system_prompt,
                max_tokens=self.max_tokens,
                temperature=self.temperature
            )

            # 构造返回结果
            result = {
                "query": query,
                "structured_answer": response,
                "format": output_format,
                "retrieved_documents": retrieved_docs
            }

            return result
        except Exception as e:
            self.logger.error(f"Error generating structured response: {e}")
            return {
                "query": query,
                "structured_answer": "抱歉，生成结构化回答时发生错误。",
                "error": str(e),
                "format": output_format,
                "retrieved_documents": retrieved_docs
            }
