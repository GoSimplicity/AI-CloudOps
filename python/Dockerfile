FROM python:3.11-slim

# 设置维护者信息
LABEL maintainer="AIOps Team"
LABEL description="AIOps Platform - Root Cause Analysis and Auto-fixing System"

# 设置工作目录
WORKDIR /app

# 设置环境变量
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1
ENV FLASK_APP=app.main:app

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    curl \
    procps \
    && rm -rf /var/lib/apt/lists/*

# 复制依赖文件
COPY requirements.txt .

# 安装Python依赖
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# 创建必要的目录
RUN mkdir -p data/models data/sample logs

# 复制应用代码
COPY . .

# 创建非root用户
RUN useradd --create-home --shell /bin/bash aiops && \
    chown -R aiops:aiops /app

USER aiops

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# 启动应用
CMD ["python", "app/main.py"]