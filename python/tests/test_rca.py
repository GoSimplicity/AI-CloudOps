#!/usr/bin/env python3
"""
Kubernetes根因分析(RCA)功能测试脚本
该脚本用于测试AI驱动的Kubernetes根因分析功能，包括:
1. RCA API健康检查测试
2. 异常检测API测试
3. 相关性分析API测试
4. 根因分析API测试
5. 指标查询API测试
6. 完整RCA工作流测试
7. 性能压力测试
"""

import requests
import json
import sys
import time
import os
import yaml
import subprocess
from datetime import datetime, timedelta
import logging
from pathlib import Path
import numpy as np
import pandas as pd
from concurrent.futures import ThreadPoolExecutor
import threading

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("rca_test")

# 测试配置
API_BASE_URL = "http://localhost:8080/api/v1"
MAX_RETRIES = 5
RETRY_DELAY = 3
REQUEST_TIMEOUT = 120  # RCA分析可能需要更长时间
SAMPLE_DIR = Path(__file__).parent.parent / "data" / "sample"
TEST_RESULT_FILE = "rca_test_results.json"

# RCA测试配置
DEFAULT_METRICS = [
    "container_cpu_usage_seconds_total",
    "container_memory_working_set_bytes",
    "container_network_receive_bytes_total",
    "container_network_transmit_bytes_total",
    "kube_pod_container_status_restarts_total",
    "kube_deployment_status_replicas_unavailable"
]

def print_header(message):
    """打印测试标题"""
    print("\n" + "=" * 80)
    print(f" {message}")
    print("=" * 80)

def make_request(method, url, json_data=None, max_retries=MAX_RETRIES, timeout=REQUEST_TIMEOUT):
    """发送请求，包含重试逻辑"""
    for attempt in range(max_retries):
        try:
            logger.info(f"请求 {method.upper()} {url} (尝试 {attempt+1}/{max_retries})")
            if method.lower() == 'get':
                response = requests.get(url, timeout=timeout)
            elif method.lower() == 'post':
                logger.info(f"发送数据: {json.dumps(json_data, ensure_ascii=False)}")
                response = requests.post(url, json=json_data, timeout=timeout)
            else:
                logger.error(f"不支持的HTTP方法: {method}")
                return None
            
            logger.info(f"响应状态码: {response.status_code}")
            return response
        except requests.exceptions.RequestException as e:
            logger.warning(f"请求失败 (尝试 {attempt+1}/{max_retries}): {str(e)}")
            if attempt < max_retries - 1:
                logger.info(f"等待 {RETRY_DELAY} 秒后重试...")
                time.sleep(RETRY_DELAY)
            else:
                logger.error(f"请求最终失败: {str(e)}")
                return None

def setup_test_environment():
    """准备测试环境，部署包含问题的测试资源"""
    print_header("准备RCA测试环境")
    
    try:
        # 检查kubectl是否可用
        result = subprocess.run(["kubectl", "version", "--client"], 
                              capture_output=True, text=True, check=False)
        if result.returncode != 0:
            logger.error("kubectl未安装或无法正常工作")
            return False
            
        # 检查是否可以访问Kubernetes集群
        result = subprocess.run(["kubectl", "get", "nodes"], 
                               capture_output=True, text=True, check=False)
        if result.returncode != 0:
            logger.error("无法连接到Kubernetes集群")
            return False
            
        # 创建测试命名空间
        logger.info("创建测试命名空间...")
        subprocess.run(["kubectl", "create", "namespace", "rca-test"], 
                      capture_output=True, text=True, check=False)
        
        # 部署有问题的应用以触发异常
        logger.info("部署测试应用...")
        test_yamls = [
            create_cpu_stress_deployment(),
            create_memory_leak_deployment(),
            create_failing_deployment(),
            create_network_heavy_deployment()
        ]
        
        for i, yaml_content in enumerate(test_yamls):
            yaml_file = f"/tmp/rca_test_{i}.yaml"
            with open(yaml_file, 'w') as f:
                f.write(yaml_content)
            
            result = subprocess.run(["kubectl", "apply", "-f", yaml_file], 
                                   capture_output=True, text=True, check=False)
            if result.returncode != 0:
                logger.warning(f"部署测试应用失败: {result.stderr}")
            else:
                logger.info(f"成功部署测试应用 {i}")
        
        # 等待应用部署并开始产生指标
        logger.info("等待应用部署并产生指标 (30秒)...")
        time.sleep(30)
        
        return True
    except Exception as e:
        logger.error(f"设置测试环境失败: {str(e)}")
        return False

def create_cpu_stress_deployment():
    """创建CPU压力测试部署"""
    return """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cpu-stress-test
  namespace: rca-test
  labels:
    app: cpu-stress-test
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cpu-stress-test
  template:
    metadata:
      labels:
        app: cpu-stress-test
    spec:
      containers:
      - name: cpu-stress
        image: busybox:1.35
        command: ["sh", "-c"]
        args: ["while true; do dd if=/dev/zero of=/dev/null; done"]
        resources:
          limits:
            cpu: "200m"
            memory: "128Mi"
          requests:
            cpu: "100m"
            memory: "64Mi"
"""

def create_memory_leak_deployment():
    """创建内存泄漏测试部署"""
    return """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memory-leak-test
  namespace: rca-test
  labels:
    app: memory-leak-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: memory-leak-test
  template:
    metadata:
      labels:
        app: memory-leak-test
    spec:
      containers:
      - name: memory-leak
        image: busybox:1.35
        command: ["sh", "-c"]
        args: ["while true; do cat /dev/zero | head -c 50m > /tmp/mem_$RANDOM; sleep 5; done"]
        resources:
          limits:
            cpu: "100m"
            memory: "256Mi"
          requests:
            cpu: "50m"
            memory: "128Mi"
"""

def create_failing_deployment():
    """创建会失败的部署"""
    return """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: failing-app-test
  namespace: rca-test
  labels:
    app: failing-app-test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: failing-app-test
  template:
    metadata:
      labels:
        app: failing-app-test
    spec:
      containers:
      - name: failing-app
        image: busybox:1.35
        command: ["sh", "-c"]
        args: ["sleep 10; exit 1"]
        resources:
          limits:
            cpu: "100m"
            memory: "128Mi"
          requests:
            cpu: "50m"
            memory: "64Mi"
"""

def create_network_heavy_deployment():
    """创建网络密集型部署"""
    return """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: network-heavy-test
  namespace: rca-test
  labels:
    app: network-heavy-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: network-heavy-test
  template:
    metadata:
      labels:
        app: network-heavy-test
    spec:
      containers:
      - name: network-heavy
        image: busybox:1.35
        command: ["sh", "-c"]
        args: ["while true; do wget -q -O - http://httpbin.org/delay/1 > /dev/null 2>&1; done"]
        resources:
          limits:
            cpu: "100m"
            memory: "128Mi"
          requests:
            cpu: "50m"
            memory: "64Mi"
"""

def cleanup_test_environment():
    """清理测试环境"""
    print_header("清理RCA测试环境")
    
    try:
        # 删除测试命名空间及其所有资源
        logger.info("删除测试命名空间...")
        result = subprocess.run(["kubectl", "delete", "namespace", "rca-test", "--ignore-not-found"], 
                               capture_output=True, text=True, check=False)
        if result.returncode == 0:
            logger.info("成功删除测试命名空间")
        else:
            logger.warning(f"删除测试命名空间失败: {result.stderr}")
        
        # 删除临时文件
        for i in range(4):
            try:
                os.remove(f"/tmp/rca_test_{i}.yaml")
            except FileNotFoundError:
                pass
        
        return True
    except Exception as e:
        logger.error(f"清理测试环境失败: {str(e)}")
        return False

def test_rca_health():
    """测试RCA健康检查API"""
    print_header("测试RCA健康检查API")
    
    url = f"{API_BASE_URL}/rca/health"
    response = make_request('get', url)
    
    if not response:
        logger.error("RCA健康检查API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_get_available_metrics():
    """测试获取可用指标API"""
    print_header("测试获取可用指标API")
    
    url = f"{API_BASE_URL}/rca/metrics"
    response = make_request('get', url)
    
    if not response:
        logger.error("获取可用指标API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    success = (
        response.status_code == 200 and
        "data" in result and
        "default_metrics" in result["data"]
    )
    
    return {
        "success": success,
        "status_code": response.status_code,
        "response": result
    }

def test_anomaly_detection():
    """测试异常检测API"""
    print_header("测试异常检测API")
    
    # 使用最近1小时的时间范围
    end_time = datetime.utcnow()
    start_time = end_time - timedelta(hours=1)
    
    url = f"{API_BASE_URL}/rca/anomalies"
    data = {
        "start_time": start_time.isoformat() + "Z",
        "end_time": end_time.isoformat() + "Z",
        "metrics": DEFAULT_METRICS[:3],  # 只使用前3个指标以加快测试
        "namespace": "rca-test",
        "threshold": 0.7
    }
    
    response = make_request('post', url, data, timeout=180)  # 异常检测可能需要更长时间
    
    if not response:
        logger.error("异常检测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    success = (
        response.status_code == 200 and
        "data" in result and
        "anomalies" in result["data"]
    )
    
    return {
        "success": success,
        "status_code": response.status_code,
        "response": result
    }

def test_correlation_analysis():
    """测试相关性分析API"""
    print_header("测试相关性分析API")
    
    # 使用最近1小时的时间范围
    end_time = datetime.utcnow()
    start_time = end_time - timedelta(hours=1)
    
    url = f"{API_BASE_URL}/rca/correlations"
    data = {
        "start_time": start_time.isoformat() + "Z",
        "end_time": end_time.isoformat() + "Z",
        "metrics": DEFAULT_METRICS[:4],  # 使用前4个指标
        "namespace": "rca-test",
        "min_correlation": 0.5
    }
    
    response = make_request('post', url, data, timeout=180)
    
    if not response:
        logger.error("相关性分析API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    success = (
        response.status_code == 200 and
        "data" in result and
        "correlations" in result["data"]
    )
    
    return {
        "success": success,
        "status_code": response.status_code,
        "response": result
    }

def test_root_cause_analysis():
    """测试根因分析API"""
    print_header("测试根因分析API")
    
    # 使用最近2小时的时间范围以获得更多数据
    end_time = datetime.utcnow()
    start_time = end_time - timedelta(hours=2)
    
    url = f"{API_BASE_URL}/rca"
    data = {
        "start_time": start_time.isoformat() + "Z",
        "end_time": end_time.isoformat() + "Z",
        "metrics": DEFAULT_METRICS,
        "namespace": "rca-test",
        "problem_description": "测试环境中的应用出现性能问题，包括CPU使用率过高、内存泄漏和容器重启",
        "include_logs": True,
        "include_events": True
    }
    
    response = make_request('post', url, data, timeout=300)  # RCA分析需要更长时间
    
    if not response:
        logger.error("根因分析API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    success = (
        response.status_code == 200 and
        "data" in result and
        "status" in result["data"] and
        "root_cause_candidates" in result["data"]
    )
    
    return {
        "success": success,
        "status_code": response.status_code,
        "response": result
    }

def test_rca_with_specific_workload():
    """测试针对特定工作负载的RCA"""
    print_header("测试针对特定工作负载的RCA")
    
    end_time = datetime.utcnow()
    start_time = end_time - timedelta(hours=1)
    
    url = f"{API_BASE_URL}/rca"
    data = {
        "start_time": start_time.isoformat() + "Z",
        "end_time": end_time.isoformat() + "Z",
        "metrics": DEFAULT_METRICS,
        "namespace": "rca-test",
        "workload_type": "deployment",
        "workload_name": "cpu-stress-test",
        "problem_description": "CPU密集型应用性能分析",
        "analysis_depth": "deep"
    }
    
    response = make_request('post', url, data, timeout=300)
    
    if not response:
        logger.error("特定工作负载RCA API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    success = (
        response.status_code == 200 and
        "data" in result
    )
    
    return {
        "success": success,
        "status_code": response.status_code,
        "response": result
    }

def test_rca_performance():
    """测试RCA性能（并发请求）"""
    print_header("测试RCA性能（并发请求）")
    
    def single_rca_request(thread_id):
        """单个RCA请求"""
        end_time = datetime.utcnow()
        start_time = end_time - timedelta(minutes=30)  # 使用较短时间范围以提高性能
        
        url = f"{API_BASE_URL}/rca"
        data = {
            "start_time": start_time.isoformat() + "Z",
            "end_time": end_time.isoformat() + "Z",
            "metrics": DEFAULT_METRICS[:2],  # 只使用2个指标
            "namespace": "rca-test",
            "problem_description": f"并发测试请求 {thread_id}",
            "analysis_depth": "quick"
        }
        
        start_request_time = time.time()
        response = make_request('post', url, data, timeout=180, max_retries=2)
        request_duration = time.time() - start_request_time
        
        success = bool(response and response.status_code == 200)
        status_code = response.status_code if response else None
        
        return {
            "thread_id": thread_id,
            "success": success,
            "duration": request_duration,
            "status_code": status_code
        }
    
    # 执行并发测试
    concurrent_requests = 3  # 适度并发以避免过载
    results = []
    
    logger.info(f"开始 {concurrent_requests} 个并发RCA请求...")
    start_time = time.time()
    
    with ThreadPoolExecutor(max_workers=concurrent_requests) as executor:
        futures = [executor.submit(single_rca_request, i) for i in range(concurrent_requests)]
        for future in futures:
            try:
                result = future.result(timeout=300)  # 每个请求最多等待5分钟
                results.append(result)
                logger.info(f"线程 {result['thread_id']} 完成: 成功={result['success']}, 耗时={result['duration']:.2f}秒")
            except Exception as e:
                logger.error(f"并发请求失败: {str(e)}")
                results.append({"success": False, "error": str(e)})
    
    total_duration = time.time() - start_time
    successful_requests = sum(1 for r in results if r.get("success"))
    average_duration = sum(r.get("duration", 0) for r in results if r.get("duration")) / len(results)
    
    performance_summary = {
        "total_requests": concurrent_requests,
        "successful_requests": successful_requests,
        "success_rate": (successful_requests / concurrent_requests) * 100,
        "total_duration": total_duration,
        "average_request_duration": average_duration,
        "results": results
    }
    
    logger.info(f"并发测试完成: 成功率={performance_summary['success_rate']:.1f}%, 平均耗时={average_duration:.2f}秒")
    
    return {
        "success": successful_requests > 0,
        "status_code": 200,
        "response": performance_summary
    }

def test_rca_historical_analysis():
    """测试历史数据RCA分析"""
    print_header("测试历史数据RCA分析")
    
    # 使用过去24小时的数据
    end_time = datetime.utcnow() - timedelta(hours=1)  # 1小时前结束
    start_time = end_time - timedelta(hours=6)  # 向前推6小时
    
    url = f"{API_BASE_URL}/rca"
    data = {
        "start_time": start_time.isoformat() + "Z",
        "end_time": end_time.isoformat() + "Z",
        "metrics": DEFAULT_METRICS,
        "problem_description": "历史数据根因分析测试",
        "analysis_type": "historical",
        "include_trends": True
    }
    
    response = make_request('post', url, data, timeout=300)
    
    if not response:
        logger.error("历史数据RCA API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    result = response.json()
    print(json.dumps(result, indent=2, ensure_ascii=False))
    
    success = response.status_code == 200
    
    return {
        "success": success,
        "status_code": response.status_code,
        "response": result
    }

def verify_test_environment():
    """验证测试环境状态"""
    logger.info("验证测试环境状态...")
    
    try:
        # 检查Pod状态
        result = subprocess.run(
            ["kubectl", "get", "pods", "-n", "rca-test", "-o", "json"],
            capture_output=True, text=True, check=False
        )
        
        if result.returncode != 0:
            logger.error(f"获取Pod状态失败: {result.stderr}")
            return {"success": False, "message": "无法获取Pod状态"}
        
        pods_data = json.loads(result.stdout)
        pods = pods_data.get("items", [])
        
        if not pods:
            logger.warning("测试命名空间中没有找到Pod")
            return {"success": False, "message": "没有找到测试Pod"}
        
        pod_status = {}
        for pod in pods:
            name = pod.get("metadata", {}).get("name", "未知")
            phase = pod.get("status", {}).get("phase", "未知")
            pod_status[name] = phase
            logger.info(f"Pod {name}: {phase}")
        
        return {
            "success": True,
            "pod_count": len(pods),
            "pod_status": pod_status
        }
        
    except Exception as e:
        logger.error(f"验证测试环境失败: {str(e)}")
        return {"success": False, "message": str(e)}

def save_test_results(results):
    """保存测试结果到JSON文件"""
    # 确保所有结果都可以被JSON序列化
    def sanitize_for_json(obj):
        """处理不可序列化的对象"""
        if isinstance(obj, dict):
            return {k: sanitize_for_json(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [sanitize_for_json(item) for item in obj]
        elif hasattr(obj, '__dict__'):
            return str(obj)
        else:
            return obj
            
    sanitized_results = sanitize_for_json(results)
    
    with open(TEST_RESULT_FILE, "w", encoding="utf-8") as f:
        json.dump(sanitized_results, f, ensure_ascii=False, indent=2)
    logger.info(f"测试结果已保存到 {TEST_RESULT_FILE}")

def main():
    """主测试流程"""
    start_time = time.time()
    
    # 记录测试开始
    logger.info("=" * 50)
    logger.info("开始Kubernetes根因分析(RCA)功能测试")
    logger.info("=" * 50)
    
    # 初始化测试结果
    results = {
        "timestamp": datetime.utcnow().isoformat(),
        "test_type": "RCA功能测试",
        "results": {},
        "environment_setup": False,
        "environment_verification": {}
    }
    
    try:
        # 设置测试环境
        setup_success = setup_test_environment()
        results["environment_setup"] = setup_success
        
        if setup_success:
            # 验证环境状态
            env_verification = verify_test_environment()
            results["environment_verification"] = env_verification
            
            # 等待指标数据生成
            logger.info("等待60秒以生成足够的指标数据...")
            time.sleep(60)
        
        # 基础API测试（不依赖环境）
        health_result = test_rca_health()
        results["results"]["rca_health"] = health_result
        
        metrics_result = test_get_available_metrics()
        results["results"]["available_metrics"] = metrics_result
        
        if setup_success:
            # 需要测试环境的测试
            anomaly_result = test_anomaly_detection()
            results["results"]["anomaly_detection"] = anomaly_result
            
            correlation_result = test_correlation_analysis()
            results["results"]["correlation_analysis"] = correlation_result
            
            rca_result = test_root_cause_analysis()
            results["results"]["root_cause_analysis"] = rca_result
            
            workload_rca_result = test_rca_with_specific_workload()
            results["results"]["workload_specific_rca"] = workload_rca_result
            
            historical_result = test_rca_historical_analysis()
            results["results"]["historical_analysis"] = historical_result
            
            # 性能测试
            performance_result = test_rca_performance()
            results["results"]["performance_test"] = performance_result
        else:
            logger.warning("测试环境设置失败，跳过需要环境的测试")
            for test_name in ["anomaly_detection", "correlation_analysis", "root_cause_analysis", 
                             "workload_specific_rca", "historical_analysis", "performance_test"]:
                results["results"][test_name] = {
                    "success": False,
                    "message": "测试环境设置失败，跳过测试"
                }
        
        # 计算测试持续时间
        duration = time.time() - start_time
        
        # 计算成功率
        total_tests = len(results["results"])
        passed_tests = sum(1 for test_result in results["results"].values() if test_result.get("success"))
        success_rate = (passed_tests / total_tests) * 100 if total_tests > 0 else 0
        
        # 添加摘要
        results["summary"] = {
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "success_rate": f"{success_rate:.2f}%",
            "duration_seconds": duration,
            "environment_setup_success": setup_success
        }
        
        # 打印摘要
        print_header("RCA测试摘要")
        print(f"总测试数: {total_tests}")
        print(f"通过测试数: {passed_tests}")
        print(f"成功率: {success_rate:.2f}%")
        print(f"测试持续时间: {duration:.2f} 秒")
        print(f"环境设置: {'成功' if setup_success else '失败'}")
        
        # 详细结果
        print("\n详细测试结果:")
        for test_name, test_result in results["results"].items():
            status = "✓ 通过" if test_result.get("success") else "✗ 失败"
            print(f"  {test_name}: {status}")
        
    except Exception as e:
        logger.error(f"测试过程中发生错误: {str(e)}")
        results["error"] = str(e)
    finally:
        # 保存测试结果
        save_test_results(results)
        
        # 清理测试环境
        if results.get("environment_setup"):
            cleanup_success = cleanup_test_environment()
            results["environment_cleanup"] = cleanup_success
        
        logger.info("=" * 50)
        logger.info("Kubernetes根因分析(RCA)功能测试完成")
        logger.info("=" * 50)
        
        return results

if __name__ == "__main__":
    main()