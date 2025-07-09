#!/bin/bash

# AIOps平台环境设置脚本

set -e

echo "🚀 开始设置AIOps平台环境..."

# 检查Python版本
check_python() {
    echo "🐍 检查Python版本..."
    if command -v python3 &> /dev/null; then
        PYTHON_VERSION=$(python3 -c "import sys; print('.'.join(map(str, sys.version_info[:2])))")
        echo "Python版本: $PYTHON_VERSION"
        
        # 检查是否为3.11+
        if python3 -c "import sys; exit(0 if sys.version_info >= (3, 11) else 1)"; then
            echo "✅ Python版本满足要求"
        else
            echo "❌ Python版本需要3.11或更高版本"
            exit 1
        fi
    else
        echo "❌ 未找到Python3，请先安装Python 3.11+"
        exit 1
    fi
}

# 检查Docker
check_docker() {
    echo "🐳 检查Docker..."
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version)
        echo "Docker版本: $DOCKER_VERSION"
        echo "✅ Docker已安装"
    else
        echo "❌ 未找到Docker，请先安装Docker"
        exit 1
    fi
    
    if command -v docker-compose &> /dev/null; then
        COMPOSE_VERSION=$(docker-compose --version)
        echo "Docker Compose版本: $COMPOSE_VERSION"
        echo "✅ Docker Compose已安装"
    else
        echo "❌ 未找到Docker Compose，请先安装Docker Compose"
        exit 1
    fi
}

# 创建必要的目录
create_directories() {
    echo "📁 创建必要的目录..."
    mkdir -p data/models
    mkdir -p data/sample
    mkdir -p logs
    mkdir -p config
    mkdir -p deploy/kubernetes
    mkdir -p deploy/grafana/dashboards
    mkdir -p deploy/grafana/datasources
    mkdir -p deploy/prometheus
    echo "✅ 目录创建完成"
}

# 设置配置文件
setup_config() {
    echo "⚙️  设置配置文件..."
    
    # 环境变量文件 (仅包含敏感数据)
    if [ ! -f .env ]; then
        cp env.example .env
        echo "✅ 已创建 .env 文件，请根据需要修改API密钥和敏感数据"
    else
        echo "⚠️  .env 文件已存在，跳过创建"
    fi
    
    # 创建开发环境YAML配置
    if [ ! -f config/config.yaml ]; then
        cat > config/config.yaml << 'EOF'
# ==============================================
# AIOps平台配置文件
# ==============================================

# 应用基础配置
app:
  debug: true
  host: 0.0.0.0
  port: 8080
  log_level: INFO

# Prometheus配置
prometheus:
  host: 127.0.0.1:9090
  timeout: 30

# LLM模型配置
llm:
  provider: openai  # 可选值: openai, ollama - 设置主要的LLM提供商
  model: Qwen/Qwen3-14B
  temperature: 0.7
  max_tokens: 2048
  # 备用Ollama模型配置
  ollama_model: qwen2.5:3b
  ollama_base_url: http://127.0.0.1:11434/v1

# 测试配置
testing:
  skip_llm_tests: false

# Kubernetes配置
kubernetes:
  in_cluster: false
  config_path: ./deploy/kubernetes/config
  namespace: default

# 根因分析配置
rca:
  default_time_range: 30
  max_time_range: 1440
  anomaly_threshold: 0.65
  correlation_threshold: 0.7
  default_metrics:
    - container_cpu_usage_seconds_total
    - container_memory_working_set_bytes
    - kube_pod_container_status_restarts_total
    - kube_pod_status_phase
    - node_cpu_seconds_total
    - node_memory_MemFree_bytes
    - kubelet_http_requests_duration_seconds_count
    - kubelet_http_requests_duration_seconds_sum

# 预测配置
prediction:
  model_path: data/models/time_qps_auto_scaling_model.pkl
  scaler_path: data/models/time_qps_auto_scaling_scaler.pkl
  max_instances: 20
  min_instances: 1
  prometheus_query: 'rate(nginx_ingress_controller_nginx_process_requests_total{service="ingress-nginx-controller-metrics"}[10m])'

# 通知配置
notification:
  enabled: true

# Tavily搜索配置
tavily:
  max_results: 5

# 小助手配置
rag:
  vector_db_path: data/vector_db
  collection_name: aiops-assistant
  knowledge_base_path: data/knowledge_base
  chunk_size: 1000
  chunk_overlap: 200
  top_k: 4
  similarity_threshold: 0.7
  openai_embedding_model: Pro/BAAI/bge-m3
  ollama_embedding_model: nomic-embed-text
  max_context_length: 4000
  temperature: 0.1
EOF
        echo "✅ 已创建开发环境配置文件 config/config.yaml"
    else
        echo "⚠️  config/config.yaml 文件已存在，跳过创建"
    fi
    
    # 创建生产环境YAML配置
    if [ ! -f config/config.production.yaml ]; then
        cat > config/config.production.yaml << 'EOF'
# ==============================================
# AIOps平台生产环境配置文件
# ==============================================

# 应用基础配置
app:
  debug: false
  host: 0.0.0.0
  port: 8080
  log_level: INFO

# Prometheus配置
prometheus:
  host: prometheus-server:9090
  timeout: 30

# LLM模型配置
llm:
  provider: openai
  model: Qwen/Qwen3-14B
  temperature: 0.3
  max_tokens: 4096
  # 备用Ollama模型配置
  ollama_model: qwen2.5:3b
  ollama_base_url: http://ollama-service:11434/v1

# 测试配置
testing:
  skip_llm_tests: false

# Kubernetes配置
kubernetes:
  in_cluster: true
  namespace: default

# 根因分析配置
rca:
  default_time_range: 30
  max_time_range: 1440
  anomaly_threshold: 0.7
  correlation_threshold: 0.75
  default_metrics:
    - container_cpu_usage_seconds_total
    - container_memory_working_set_bytes
    - kube_pod_container_status_restarts_total
    - kube_pod_status_phase
    - node_cpu_seconds_total
    - node_memory_MemFree_bytes
    - kubelet_http_requests_duration_seconds_count
    - kubelet_http_requests_duration_seconds_sum

# 预测配置
prediction:
  model_path: /app/data/models/time_qps_auto_scaling_model.pkl
  scaler_path: /app/data/models/time_qps_auto_scaling_scaler.pkl
  max_instances: 20
  min_instances: 1
  prometheus_query: 'rate(nginx_ingress_controller_nginx_process_requests_total{service="ingress-nginx-controller-metrics"}[10m])'

# 通知配置
notification:
  enabled: true

# Tavily搜索配置
tavily:
  max_results: 5

# 小助手配置
rag:
  vector_db_path: /app/data/vector_db
  collection_name: aiops-assistant-prod
  knowledge_base_path: /app/data/knowledge_base
  chunk_size: 1000
  chunk_overlap: 200
  top_k: 5
  similarity_threshold: 0.75
  openai_embedding_model: Pro/BAAI/bge-m3
  ollama_embedding_model: nomic-embed-text
  max_context_length: 6000
  temperature: 0.1
EOF
        echo "✅ 已创建生产环境配置文件 config/config.production.yaml"
    else
        echo "⚠️  config/config.production.yaml 文件已存在，跳过创建"
    fi
    
    # 创建配置说明文件
    if [ ! -f config/README.md ]; then
        cat > config/README.md << 'EOF'
# AIOps 平台配置指南

## 配置文件说明

AIOps 平台使用两种配置机制：YAML 配置文件和环境变量。这种方式分离了普通配置和敏感数据，提高了系统的安全性和可维护性。

### 配置优先级

系统加载配置的优先级顺序为：

1. 环境变量（最高优先级）
2. 环境特定 YAML 配置文件（如`config.production.yaml`）
3. 默认 YAML 配置文件（`config.yaml`）
4. 代码中的默认值（最低优先级）

### 配置文件

- `config.yaml`：默认配置文件，包含开发环境的所有非敏感配置
- `config.production.yaml`：生产环境配置文件，包含生产环境的非敏感配置
- 可以根据需要创建其他环境配置文件，如`config.test.yaml`、`config.staging.yaml`等

### 环境变量文件

- `env.example`：示例环境变量文件，仅包含敏感数据和 API 密钥
- `env.production`：生产环境的环境变量文件，包含生产环境的敏感数据和 API 密钥

## 使用方法

### 切换环境

通过设置`ENV`环境变量来切换不同环境的配置：

```bash
# 开发环境（默认）
export ENV=development

# 生产环境
export ENV=production

# 测试环境
export ENV=test
```

### 增加新的配置项

1. 在相应的 YAML 配置文件中添加新的配置项
2. 在`app/config/settings.py`中更新相应的配置类

### 配置敏感数据

敏感数据（如 API 密钥、密码等）应始终通过环境变量或`.env`文件配置，而不是直接写入 YAML 配置文件。
EOF
        echo "✅ 已创建配置说明文件 config/README.md"
    else
        echo "⚠️  config/README.md 文件已存在，跳过创建"
    fi
}

# 安装Python依赖
install_python_deps() {
    echo "📦 安装Python依赖..."
    
    # 检查是否在虚拟环境中
    if [[ "$VIRTUAL_ENV" != "" ]]; then
        echo "✅ 检测到虚拟环境: $VIRTUAL_ENV"
    else
        echo "⚠️  建议在虚拟环境中安装依赖"
        read -p "是否继续在系统环境中安装？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "请先创建并激活虚拟环境："
            echo "  python3 -m venv venv"
            echo "  source venv/bin/activate"
            exit 1
        fi
    fi
    
    pip install --upgrade pip
    pip install -r requirements.txt
    echo "✅ Python依赖安装完成"
}

# 创建示例配置文件
create_sample_configs() {
    echo "📝 创建示例配置文件..."
    
    # Prometheus配置
    cat > deploy/prometheus/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rules/*.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
  
  - job_name: 'aiops-platform'
    static_configs:
      - targets: ['aiops-platform:8080']
    metrics_path: '/api/v1/health/metrics'
    scrape_interval: 30s

  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
EOF

    # Grafana数据源配置
    mkdir -p deploy/grafana/datasources
    cat > deploy/grafana/datasources/prometheus.yml << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
EOF

    # 创建Kubernetes配置示例文件
    mkdir -p deploy/kubernetes
    cat > deploy/kubernetes/config.example << 'EOF'
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.default.svc
    certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
  name: default
contexts:
- context:
    cluster: default
    namespace: default
    user: default
  name: default
current-context: default
users:
- name: default
  user:
    tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
EOF

    echo "✅ 示例配置文件创建完成"
}

# 下载示例模型文件（如果需要）
download_sample_models() {
    echo "🤖 检查模型文件..."
    
    if [ ! -f "data/models/time_qps_auto_scaling_model.pkl" ]; then
        echo "⚠️  未找到预测模型文件"
        echo "请将训练好的模型文件放置在 data/models/ 目录下："
        echo "  - time_qps_auto_scaling_model.pkl"
        echo "  - time_qps_auto_scaling_scaler.pkl"
        echo "或者运行训练脚本生成模型"
    else
        echo "✅ 模型文件已存在"
    fi
}

# 验证安装
verify_installation() {
    echo "🔍 验证安装..."
    
    # 检查Python导入
    python3 -c "
import flask
import pandas
import numpy
import sklearn
import yaml
import requests
print('✅ 主要Python包导入成功')
"
    
    # 检查应用能否启动（语法检查）
    python3 -c "
import sys
sys.path.append('.')
try:
    from app.main import create_app
    app = create_app()
    import flask
    print('✅ 应用代码语法检查通过')
except Exception as e:
    print(f'❌ 应用代码检查失败: {str(e)}')
    sys.exit(1)
"
    
    echo "✅ 安装验证完成"
}

# 配置Kubernetes
setup_kubernetes() {
    echo "☸️  配置Kubernetes环境..."
    
    # 检查是否存在kubeconfig
    if [ -f "$HOME/.kube/config" ]; then
        echo "✅ 检测到Kubernetes配置文件"
        # 复制到项目目录
        mkdir -p deploy/kubernetes
        cp "$HOME/.kube/config" deploy/kubernetes/config
        echo "✅ 已复制Kubernetes配置到项目目录"
    else
        echo "⚠️  未找到Kubernetes配置文件"
        echo "请确保您有权限访问Kubernetes集群，并将配置文件放置在以下位置之一："
        echo "  - $HOME/.kube/config"
        echo "  - deploy/kubernetes/config"
        
        # 创建示例配置
        echo "已创建示例配置文件，请根据实际情况修改："
        echo "  - deploy/kubernetes/config.example"
    fi
}

# 显示下一步操作
show_next_steps() {
    echo ""
    echo "🎉 AIOps平台环境设置完成！"
    echo ""
    echo "下一步操作："
    echo "1. 配置文件："
    echo "   - 编辑 config/config.yaml 文件配置应用参数"
    echo "   - 编辑 .env 文件配置API密钥和敏感数据"
    echo "2. 确保Kubernetes配置正确（如果使用K8s功能）"
    echo "   - 检查 deploy/kubernetes/config 文件"
    echo "3. 启动服务："
    echo "   # 使用Docker Compose（推荐）"
    echo "   docker-compose up -d"
    echo ""
    echo "   # 或本地开发模式"
    echo "   ENV=development ./scripts/start.sh"
    echo ""
    echo "   # 或生产环境"
    echo "   ENV=production ./scripts/start_production.sh"
    echo ""
    echo "4. 访问服务："
    echo "   - AIOps API: http://localhost:8080"
    echo "   - Prometheus: http://localhost:9090"
    echo "   - Grafana: http://localhost:3000 (admin/admin123)"
    echo ""
    echo "5. 健康检查："
    echo "   curl http://localhost:8080/api/v1/health"
    echo ""
}

# 主函数
main() {
    echo "AIOps平台环境设置脚本"
    echo "========================"
    
    check_python
    check_docker
    create_directories
    setup_config
    install_python_deps
    create_sample_configs
    setup_kubernetes
    download_sample_models
    verify_installation
    show_next_steps
}

# 处理中断信号
trap 'echo "❌ 设置被中断"; exit 1' INT

# 运行主函数
main "$@"