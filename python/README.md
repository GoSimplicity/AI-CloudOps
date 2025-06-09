# AI-CloudOps AI Module - 智能运维根因分析与自动修复系统

[![Python Version](https://img.shields.io/badge/python-3.11+-blue.svg)](https://python.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-enabled-blue.svg)](Dockerfile)

## 📖 项目简介

AI-CloudOps AI 模块是一个基于人工智能的智能运维系统，专注于 Kubernetes 环境的根因分析和自动化修复。系统集成了多种 AI 技术，包括异常检测、相关性分析、大语言模型和多 Agent 协作，为运维团队提供智能化的问题诊断和解决方案。

## ✨ 核心功能

### 🔍 智能根因分析

- **多维度异常检测**: 基于 Z-Score、IQR、孤立森林、DBSCAN 等多种算法
- **相关性分析**: 发现指标间的关联关系，识别潜在的因果链
- **时间序列分析**: 支持时间窗口分析和趋势检测
- **AI 智能总结**: 基于 LLM 生成人类可读的分析报告

### 🤖 多 Agent 自动修复

- **Supervisor Agent**: 协调整体修复流程
- **K8s Fixer Agent**: 专门处理 Kubernetes 问题
- **Researcher Agent**: 搜索解决方案和最佳实践
- **Coder Agent**: 执行数据分析和计算任务
- **Notifier Agent**: 发送通知和告警

### 📊 负载预测

- **基于机器学习的预测模型**: 预测未来实例需求
- **时间特征工程**: 考虑时间周期性和业务模式
- **置信度评估**: 提供预测结果的可信度

### 🔔 智能通知

- **飞书集成**: 支持富文本卡片消息
- **分级告警**: 根据严重程度发送不同级别的通知
- **人工干预**: 自动识别需要人工介入的场景

## 🏗 系统架构

```bash
┌─────────────────────────────────────────────────────────────┐
│ API Gateway │
├─────────────────────────────────────────────────────────────┤
│ Health API │ Predict API │ RCA API │ AutoFix API │
├─────────────────────────────────────────────────────────────┤
│ Core Business Logic │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
│ │ RCA │ │ Prediction │ │ Multi-Agent │ │
│ │ Engine │ │ Service │ │ System │ │
│ └─────────────┘ └─────────────┘ └─────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ Service Layer │
│ Prometheus │ Kubernetes │ LLM │ Notification │
├─────────────────────────────────────────────────────────────┤
│ Infrastructure │
│ Prometheus │ Grafana │ Ollama │ Redis │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求

- Python 3.11+
- Docker & Docker Compose
- Kubernetes 集群（可选）
- 至少 4GB RAM

### 1. 克隆项目

```bash
git clone <repository-url>
cd aiops-platform
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，配置相关参数
```

### 3. 使用 Docker Compose 启动

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f aiops-platform
```

### 4. 验证服务

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 查看API文档
curl http://localhost:8080/
```

## 📋 API 文档

### 健康检查

```bash
GET /api/v1/health
```

### 负载预测

```bash
# 获取当前预测
GET /api/v1/predict

# 自定义预测
POST /api/v1/predict
{
  "current_qps": 100.5,
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 根因分析

```bash
POST /api/v1/rca
{
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-01T11:00:00Z",
  "metrics": ["container_cpu_usage_seconds_total"]
}
```

### 自动修复

```bash
POST /api/v1/autofix
{
  "deployment": "my-app",
  "namespace": "default",
  "event": "Pod启动失败"
}
```

## 🛠 开发指南

### 本地开发环境

```bash
# 创建虚拟环境
python -m venv venv
source venv/bin/activate  # Linux/Mac
# 或
venv\Scripts\activate  # Windows

# 安装依赖
pip install -r requirements.txt

# 启动开发服务器
python app/main.py
```

### 项目结构

```bash
python/
├── app/                    # 应用核心代码
│   ├── api/               # API路由和中间件
│   ├── config/            # 配置管理
│   ├── core/              # 核心业务逻辑
│   ├── models/            # 数据模型
│   ├── services/          # 外部服务集成
│   ├── utils/             # 工具函数
│   └── main.py            # 应用入口
├── data/                  # 数据文件
├── deploy/                # 部署配置
├── tests/                 # 测试代码
├── scripts/               # 脚本文件
└── docker-compose.yml     # Docker编排文件
```

### 运行测试

```bash
# 运行所有测试
pytest

# 运行特定测试
pytest tests/test_rca.py

# 生成覆盖率报告
pytest --cov=app tests/
```

## 🔧 配置说明

### 主要配置项

| 配置项                   | 说明                | 默认值         |
| ------------------------ | ------------------- | -------------- |
| PROMETHEUS_HOST          | Prometheus 服务地址 | 127.0.0.1:9090 |
| LLM_MODEL                | 使用的 LLM 模型     | qwen2.5:3b     |
| RCA_ANOMALY_THRESHOLD    | 异常检测阈值        | 0.65           |
| PREDICTION_MAX_INSTANCES | 最大实例数          | 20             |
| NOTIFICATION_ENABLED     | 是否启用通知        | true           |

### LLM 模型配置

支持的模型：

- Ollama 本地模型（推荐）
- OpenAI GPT 系列
- 其他兼容 OpenAI API 的模型

## 📞 联系我们

- **项目负责人**: Bamboo
- **邮箱**: [13664854532@163.com](mailto:13664854532@163.com)
- **项目主页**: [https://github.com/GoSimplicity/AI-CloudOps]

## 📄 许可证

本项目采用 MIT 许可证 - 详见 LICENSE 文件
