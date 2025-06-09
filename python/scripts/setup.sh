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
    mkdir -p deploy/kubernetes
    mkdir -p deploy/grafana/dashboards
    mkdir -p deploy/grafana/datasources
    mkdir -p deploy/prometheus
    echo "✅ 目录创建完成"
}

# 复制环境变量文件
setup_env() {
    echo "⚙️  设置环境变量..."
    if [ ! -f .env ]; then
        cp .env.example .env
        echo "✅ 已创建.env文件，请根据需要修改配置"
    else
        echo "⚠️  .env文件已存在，跳过创建"
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
    
    # 更新.env文件中的K8s配置路径
    if [ -f ".env" ]; then
        if grep -q "K8S_CONFIG_PATH" .env; then
            sed -i.bak 's|K8S_CONFIG_PATH=.*|K8S_CONFIG_PATH=./deploy/kubernetes/config|g' .env
        else
            echo "K8S_CONFIG_PATH=./deploy/kubernetes/config" >> .env
        fi
        echo "✅ 已更新.env文件中的Kubernetes配置路径"
    fi
}

# 显示下一步操作
show_next_steps() {
    echo ""
    echo "🎉 AIOps平台环境设置完成！"
    echo ""
    echo "下一步操作："
    echo "1. 编辑 .env 文件配置参数"
    echo "2. 确保Kubernetes配置正确（如果使用K8s功能）"
    echo "   - 检查 deploy/kubernetes/config 文件"
    echo "3. 启动服务："
    echo "   # 使用Docker Compose（推荐）"
    echo "   docker-compose up -d"
    echo ""
    echo "   # 或本地开发"
    echo "   python app/main.py"
    echo ""
    echo "3. 访问服务："
    echo "   - AIOps API: http://localhost:8080"
    echo "   - Prometheus: http://localhost:9090"
    echo "   - Grafana: http://localhost:3000 (admin/admin123)"
    echo ""
    echo "4. 健康检查："
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
    setup_env
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