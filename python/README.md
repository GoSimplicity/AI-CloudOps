# ai部分目录结构说明

本模块是一个基于 Python 的 AIOps 系统，包含核心功能模块、AI 模型、RAG 系统、数据处理、自动化 MLOps、知识融合、智能代理等模块，以下是目录结构和模块的详细说明

---

## 目录结构

```plaintext
python/
│
├── core/                       # 核心功能模块
│   ├── anomaly_detection/      # 异常检测模块
│   ├── root_cause/             # 根因分析模块
│   ├── prediction/             # 预测模块
│   └── optimization/           # 优化模块
│
├── models/                     # AI 模型
│
├── rag/                        # RAG 系统（AI 小助手）
│
├── data/                       # 数据处理
│   ├── collectors/             # 数据收集器
│   ├── preprocessors/          # 数据预处理
│   └── storage/                # 数据存储
│
├── auto_mlops/                 # 自动化 MLOps
│
├── knowledge_fusion/           # 知识融合
│
├── agents/                     # 智能代理
│
├── interfaces/                 # 与 Go 底座交互接口
│
├── api/                        # API 接口
│   ├── routes/                 # 路由
│   └── schemas/                # 数据模型
│
├── tests/                      # 测试
│   ├── unit/                   # 单元测试
│   └── integration/            # 集成测试
│
├── utils/                      # 工具函数
│
├── config/                     # 配置文件
│
├── scripts/                    # 脚本
│
├── requirements.txt            # 依赖包
└── README.md                   # 文档
```

---

## 模块说明

### 1. **核心功能模块 (`core/`)**
- **`anomaly_detection/`**：异常检测模块，用于检测系统中的异常行为
  - `metric_anomaly.py`：指标异常检测
  - `log_anomaly.py`：日志异常检测
  - `trace_anomaly.py`：链路追踪异常检测
- **`root_cause/`**：根因分析模块，用于定位系统故障的根本原因
  - `causal_graph.py`：因果图分析
  - `fault_localization.py`：故障定位
- **`prediction/`**：预测模块，用于预测系统未来的状态
  - `resource_prediction.py`：资源预测
  - `failure_prediction.py`：故障预测
- **`optimization/`**：优化模块，用于优化系统性能
  - `auto_scaling.py`：自动伸缩优化
  - `resource_allocation.py`：资源分配优化

---

### 2. **AI 模型 (`models/`)**
- **`base_model.py`**：模型基类，定义通用的模型接口和方法
- **`transformer_models.py`**：基于 Transformer 的模型
- **`time_series_models.py`**：时间序列模型
- **`graph_models.py`**：图模型

---

### 3. **RAG 系统 (`rag/`)**
- **`document_processor.py`**：文档处理模块，用于处理输入的文档数据
- **`vector_store.py`**：向量存储模块，用于存储和检索向量数据
- **`retriever.py`**：检索器模块，用于从知识库中检索相关信息
- **`generator.py`**：生成器模块，用于生成回答或建议
- **`knowledge_base.py`**：知识库管理模块，用于管理知识库数据

---

### 4. **数据处理 (`data/`)**
- **`collectors/`**：数据收集器模块，用于从不同来源收集数据
  - `prometheus_collector.py`：从 Prometheus 收集数据
  - `k8s_collector.py`：从 Kubernetes 收集数据
  - `log_collector.py`：从日志系统收集数据
- **`preprocessors/`**：数据预处理模块，用于清洗和转换数据
  - `normalization.py`：数据标准化
  - `feature_extraction.py`：特征提取
- **`storage/`**：数据存储模块，用于存储处理后的数据
  - `time_series_db.py`：时间序列数据存储
  - `vector_db.py`：向量数据存储

---

### 5. **自动化 MLOps (`auto_mlops/`)**
- **`model_registry.py`**：模型注册模块，用于管理模型的版本和元数据
- **`model_training.py`**：模型训练模块，用于自动化训练模型
- **`model_deployment.py`**：模型部署模块，用于自动化部署模型
- **`model_monitoring.py`**：模型监控模块，用于监控模型性能

---

### 6. **知识融合 (`knowledge_fusion/`)**
- **`domain_knowledge.py`**：领域知识模块，用于管理领域相关的知识
- **`experience_knowledge.py`**：经验知识模块，用于管理系统运行中的经验数据
- **`knowledge_graph.py`**：知识图谱模块，用于构建和管理知识图谱

---

### 7. **智能代理 (`agents/`)**
- **`healing_agent.py`**：自愈代理模块，用于自动修复系统故障
- **`optimization_agent.py`**：优化代理模块，用于自动优化系统性能
- **`decision_agent.py`**：决策代理模块，用于辅助系统决策

---

### 8. **接口 (`interfaces/`)**
- **`service_tree_interface.py`**：服务树接口模块，用于与服务树系统交互
- **`prometheus_interface.py`**：Prometheus 接口模块，用于与 Prometheus 交互
- **`k8s_interface.py`**：Kubernetes 接口模块，用于与 Kubernetes 交互

---

### 9. **API 接口 (`api/`)**
- **`fastapi_app.py`**：FastAPI 应用模块，用于启动 API 服务
- **`routes/`**：路由模块，用于定义 API 路由
  - `anomaly_routes.py`：异常检测相关路由
  - `prediction_routes.py`：预测相关路由
  - `assistant_routes.py`：助手相关路由
- **`schemas/`**：数据模型模块，用于定义 API 请求和响应模型
  - `request_models.py`：请求模型定义

---

### 10. **测试 (`tests/`)**
- **`unit/`**：单元测试模块，用于测试各个功能模块
- **`integration/`**：集成测试模块，用于测试模块之间的集成

---

### 11. **工具函数 (`utils/`)**
- **`logger.py`**：日志工具模块，用于记录系统日志
- **`config.py`**：配置工具模块，用于加载和管理配置文件
- **`metrics.py`**：指标工具模块，用于定义和收集系统指标

---

### 12. **配置文件 (`config/`)**
- **`config.yaml`**：主配置文件，用于定义系统全局配置
- **`models_config.yaml`**：模型配置文件，用于定义模型相关配置

---

### 13. **脚本 (`scripts/`)**
- **`setup.py`**：安装脚本，用于安装项目依赖
- **`start_services.py`**：启动服务脚本，用于启动系统服务

---

## 如何运行
推荐python版本：Python 3.11.11
1. 安装依赖：
   ```bash
   pip install -r requirements.txt
   ```

2. 启动服务：
   ```bash
   python scripts/start_services.py
   ```

3. 访问 API：
   - 默认地址：`http://localhost:8000`
