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

"""
支持多模型提供商的LLM接口
"""
import logging
import os
import ollama
import requests
import time


class LLMProvider:
  def __init__(
    self,
    ollama_host: str = None,
    default_model: str = None
  ):
    self.provider = "ollama"
    self.default_model = default_model or os.getenv("LLM_MODEL", "deepseek-r1:8b")
    self.logger = logging.getLogger(__name__)
    self.client = None

    # 初始化Ollama提供者
    try:
      self._initialize_ollama(ollama_host)
      self.logger.info("Successfully initialized Ollama as LLM provider")
    except Exception as e:
      self.logger.error(f"Failed to initialize Ollama: {e}")
      raise ValueError(f"无法初始化Ollama提供者: {e}")

  def _initialize_ollama(self, ollama_host):
    """初始化Ollama提供者"""
    ollama_host = ollama_host or os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434")
    ollama_host = ollama_host.rstrip('/')

    # 尝试使用HTTP请求检查Ollama服务是否可用
    max_retries = 3
    retry_delay = 2
    connected = False

    for attempt in range(max_retries):
      try:
        response = requests.get(f"{ollama_host}/api/tags", timeout=5)
        if response.status_code == 200:
          self.logger.info(f"Successfully connected to Ollama API at {ollama_host}")
          connected = True
          break
        else:
          self.logger.warning(
            f"Ollama API returned unexpected status code: {response.status_code}")
      except requests.exceptions.RequestException as e:
        self.logger.warning(f"Connection attempt {attempt + 1}/{max_retries} failed: {e}")
        if attempt < max_retries - 1:
          time.sleep(retry_delay)

    if not connected:
      raise ConnectionError(f"无法连接到Ollama服务，请确保服务已启动并且可访问")

    # 如果连接成功，初始化客户端
    try:
      self.client = ollama.Client(host=ollama_host)
    except Exception as e:
      self.logger.error(f"Failed to initialize Ollama client: {e}")
      raise ConnectionError(f"无法初始化Ollama客户端: {e}")

  def generate(self, prompt: str, **kwargs) -> str:
    """
    生成文本

    Args:
        prompt: 提示文本
        **kwargs: 其他参数

    Returns:
        生成的文本
    """
    return self._ollama_generate(prompt=prompt, **kwargs)

  def _ollama_generate(self, **kwargs) -> str:
    try:
      options = {
        "num_predict": kwargs.get('max_tokens', 2000),
        "temperature": kwargs.get('temperature', 0.7),
        "seed": 42  # 增加确定性
      }

      messages = []
      if kwargs.get('system_prompt'):
        messages.append({"role": "system", "content": kwargs['system_prompt']})
      messages.append({"role": "user", "content": kwargs['prompt']})

      # 添加重试机制
      max_retries = 3
      retry_delay = 2
      last_error = None

      for attempt in range(max_retries):
        try:
          response = self.client.chat(
            model=kwargs.get('model', self.default_model),
            messages=messages,
            options=options
          )
          return response['message']['content']
        except Exception as e:
          last_error = e
          self.logger.warning(f"Ollama API error (attempt {attempt + 1}/{max_retries}): {e}")
          if attempt < max_retries - 1:
            self.logger.info(f"Retrying in {retry_delay} seconds...")
            time.sleep(retry_delay)

      # 如果所有重试都失败，则返回友好错误信息
      self.logger.error(f"All attempts failed: {last_error}")
      return "抱歉，模型生成回答时遇到了问题。请稍后再试或尝试使用不同的问题。"
    except Exception as e:
      self.logger.error(f"Ollama API error: {e}")
      return "抱歉，与AI模型的连接出现了问题。请检查Ollama服务是否正常运行。"

  def generate_response(self, prompt: str, context: str = None, **kwargs) -> str:
    """
    生成回答

    Args:
        prompt: 提示文本
        context: 上下文信息
        **kwargs: 其他参数

    Returns:
        生成的回答
    """
    system_prompt = kwargs.get('system_prompt', "你是一个智能助手，请根据提供的上下文信息回答问题。")

    if context:
      system_prompt += f"\n\n上下文信息:\n{context}"

    # 调用generate方法生成回答
    return self.generate(
      prompt=prompt,
      system_prompt=system_prompt,
      model=kwargs.get('model', self.default_model),
      max_tokens=kwargs.get('max_tokens', 2000),
      temperature=kwargs.get('temperature', 0.7)
    )
