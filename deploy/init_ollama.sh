#!/bin/bash

# 终止脚本执行并返回错误状态码，如果有任何命令失败
set -euo pipefail

echo_info() {
    echo -e "\033[1;34m[INFO]\033[0m $1"
}

echo_error() {
    echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
    exit 1
}

# 检查是否以root用户运行
if [[ "$EUID" -ne 0 ]]; then
    echo_error "请以root用户或使用sudo运行此脚本。"
fi

# 更新包列表
echo_info "更新包列表..."
apt-get update

# 安装必要的依赖包
echo_info "安装必要的依赖包..."
apt-get install -y curl gnupg lsb-release software-properties-common

# 添加NVIDIA的GPG密钥
echo_info "添加NVIDIA的GPG密钥..."
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor | tee /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg > /dev/null

# 添加NVIDIA的APT仓库
echo_info "添加NVIDIA的APT仓库..."
curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    tee /etc/apt/sources.list.d/nvidia-container-toolkit.list > /dev/null

# 再次更新包列表以包含NVIDIA的仓库
echo_info "再次更新包列表以包含NVIDIA的仓库..."
apt-get update

# 安装NVIDIA容器工具包
echo_info "安装NVIDIA容器工具包..."
apt-get install -y nvidia-container-toolkit

# 重启Docker守护进程以应用更改
echo_info "重启Docker守护进程以应用更改..."
systemctl restart docker

# 检查并运行ollama容器
if ! docker ps -q -f name=ollama > /dev/null; then
    if docker ps -aq -f status=exited -f name=ollama > /dev/null; then
        echo_info "移除已停止的ollama容器..."
        docker rm ollama
    fi
    echo_info "运行ollama容器..."
    docker run --gpus all -d \
        -v /opt/ai/ollama:/root/.ollama \
        -p 11434:11434 \
        --name ollama \
        ollama/ollama
else
    echo_info "ollama容器已在运行。"
fi

# 在ollama容器中执行命令
echo_info "在ollama容器中执行'ollama run llama3.1'命令..."
docker exec -it ollama ollama run llama3.1

# 检查并运行open-webui容器
if ! docker ps -q -f name=open-webui > /dev/null; then
    if docker ps -aq -f status=exited -f name=open-webui > /dev/null; then
        echo_info "移除已停止的open-webui容器..."
        docker rm open-webui
    fi
    echo_info "运行open-webui容器..."
    docker run -d \
        -p 3000:8080 \
        --add-host=host.docker.internal:host-gateway \
        -v open-webui:/app/backend/data \
        --name open-webui \
        --restart always \
        ghcr.io/open-webui/open-webui:main
else
    echo_info "open-webui容器已在运行。"
fi

echo_info "所有服务已成功启动。"