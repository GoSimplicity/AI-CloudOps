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

## K8s 故障自动修复模块

K8s 故障自动修复模块是一个用于自动诊断和修复 Kubernetes 集群中常见问题的子系统。该模块可以检测和修复以下常见问题：

- Pod 健康检查（readinessProbe 和 livenessProbe）配置不当
- 资源请求和限制设置不合理
- 探针路径和探针频率配置错误
- CrashLoopBackOff 故障的分析和修复
- 集群中部署的其他常见错误

### 架构

自动修复模块由以下组件组成：

1. **K8sFixerAgent**: 核心修复逻辑，负责分析和修复 Kubernetes 相关问题
2. **SupervisorAgent**: 工作流协调器，管理修复过程中的决策链和修复步骤
3. **NotifierAgent**: 在关键修复事件中发送通知
4. **KubernetesService**: K8s API 的抽象层，提供集群操作功能

### 使用方法

#### 自动修复 API

```bash
# 修复特定部署
curl -X POST http://localhost:8080/api/v1/autofix \
  -H "Content-Type: application/json" \
  -d '{
    "deployment": "nginx-problematic",
    "namespace": "default",
    "event": "Pod启动失败，原因是readinessProbe探针路径(/health)不存在，探针频率过高，且资源请求过高",
    "force": true
  }'
```

#### 诊断集群健康状态

```bash
# 诊断特定命名空间
curl -X POST http://localhost:8080/api/v1/autofix/diagnose \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default"
  }'
```

### 典型使用场景

1. **自动修复健康检查问题**：

   - 检测并修复错误的探针路径
   - 调整不合理的探针频率和失败阈值
   - 添加缺失的 readinessProbe 或修复 livenessProbe

2. **资源优化**：

   - 识别并修复资源请求过高的配置
   - 平衡 CPU 和内存限制与请求

3. **多因素问题修复**：
   - 处理由多个因素引起的 CrashLoopBackOff 问题
   - 组合应用多项修复

### 样例修复流程

1. 发现 Pod 无法启动，处于 CrashLoopBackOff 状态
2. 调用自动修复 API
3. 系统分析 Pod 状态、事件和配置
4. 识别出 livenessProbe 配置不当：路径错误、频率过高
5. 自动应用修复配置
6. 验证 Pod 状态恢复正常

### 配置和调试

K8s 自动修复模块需要正确配置以下项：

- Kubernetes 配置文件路径: `K8S_CONFIG_PATH`环境变量
- 集群连接方式: `K8S_IN_CLUSTER`环境变量

#### API 端点

自动修复模块提供以下 API 端点：

- `POST /api/v1/autofix`: 自动修复指定的 deployment

  ```json
  {
    "deployment": "nginx-deployment",
    "namespace": "default",
    "event": "Pod启动失败，原因是健康检查配置不当",
    "force": true
  }
  ```

- `GET /api/v1/autofix/health`: 检查自动修复服务健康状态

- `POST /api/v1/autofix/diagnose`: 诊断集群健康状态

  ```json
  {
    "namespace": "default"
  }
  ```

- `POST /api/v1/autofix/workflow`: 执行完整的自动修复工作流

  ```json
  {
    "problem_description": "Kubernetes集群中的nginx-deployment出现了Pod无法启动的问题"
  }
  ```

- `POST /api/v1/autofix/notify`: 发送通知
  ```json
  {
    "type": "human_help",
    "message": "需要人工协助处理Kubernetes集群中的问题",
    "urgency": "medium"
  }
  ```

### 测试

项目包含一个测试脚本`tests/test-autofix.py`，可以用于测试自动修复功能。测试脚本会检查健康状态、执行诊断、尝试修复正常和异常部署，并验证结果。

#### 前提条件

1. 有一个可用的 Kubernetes 集群
2. 集群中已部署示例应用（正常的 nginx-deployment 和有问题的 nginx-problematic）
3. K8s 配置文件位于`deploy/kubernetes/config`

#### 运行测试

```bash
# 设置环境变量
export KUBECONFIG=deploy/kubernetes/config
export PYTHONPATH=$(pwd)

# 启动应用
python app/main.py

# 在另一个终端运行测试
python tests/test-autofix.py
```

### 常见问题排查

#### 无法连接到 Kubernetes

确保：

- K8s 配置文件正确且包含访问权限
- 设置了正确的环境变量：`KUBECONFIG`和`K8S_CONFIG_PATH`

#### 修复不生效

可能的原因：

- 集群资源不足，无法满足新 Pod 的请求
- 权限问题，无法修改 deployment
- 网络问题，无法访问 K8s API 服务器

可以尝试减小资源请求：

```bash
kubectl patch deployment <problematic-deployment> -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container-name>","resources":{"requests":{"memory":"32Mi","cpu":"50m"},"limits":{"memory":"64Mi","cpu":"100m"}},"readinessProbe":{"httpGet":{"path":"/"},"periodSeconds":10,"failureThreshold":3}}]}}}}'
```