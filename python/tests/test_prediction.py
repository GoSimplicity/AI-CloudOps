#!/usr/bin/env python3
"""
Kubernetes预测服务功能测试脚本
该脚本用于测试AI驱动的Kubernetes预测服务功能，包括:
1. 健康检查API测试
2. 预测API测试（GET和POST）
3. 零QPS预测测试
4. 低QPS预测测试
5. 高QPS预测测试
6. 趋势预测API测试
7. 模型验证测试
8. 完整工作流API测试
"""

import requests
import json
import sys
import time
import os
import subprocess
from datetime import datetime, timedelta
import logging
from pathlib import Path
import numpy as np

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("prediction_test")

# 测试配置
API_BASE_URL = "http://localhost:8080/api/v1"
MAX_RETRIES = 5
RETRY_DELAY = 3
REQUEST_TIMEOUT = 60
TEST_RESULT_FILE = "prediction_test_results.json"

def print_header(message):
    """打印测试标题"""
    print("\n" + "=" * 80)
    print(f" {message}")
    print("=" * 80)

def make_request(method, url, json_data=None, max_retries=MAX_RETRIES):
    """发送请求，包含重试逻辑"""
    for attempt in range(max_retries):
        try:
            logger.info(f"请求 {method.upper()} {url} (尝试 {attempt+1}/{max_retries})")
            if method.lower() == 'get':
                response = requests.get(url, timeout=REQUEST_TIMEOUT)
            elif method.lower() == 'post':
                logger.info(f"发送数据: {json.dumps(json_data, ensure_ascii=False)}")
                response = requests.post(url, json=json_data, timeout=REQUEST_TIMEOUT)
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
    """准备测试环境"""
    print_header("准备测试环境")
    
    try:
        # 检查服务是否运行
        health_url = f"{API_BASE_URL}/health"
        response = make_request('get', health_url, max_retries=3)
        
        if response and response.status_code == 200:
            logger.info("预测服务正在运行")
            return True
        else:
            logger.warning("预测服务可能未运行，但继续测试")
            return True
            
    except Exception as e:
        logger.error(f"设置测试环境失败: {str(e)}")
        return False

def test_health():
    """测试健康检查API"""
    print_header("测试健康检查API")
    
    url = f"{API_BASE_URL}/health"
    response = make_request('get', url)
    
    if not response:
        logger.error("健康检查API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_health():
    """测试预测服务健康API"""
    print_header("测试预测服务健康API")
    
    url = f"{API_BASE_URL}/predict/health"
    response = make_request('get', url)
    
    if not response:
        logger.error("预测服务健康检查API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_get():
    """测试GET预测接口"""
    print_header("测试GET预测接口")
    
    url = f"{API_BASE_URL}/predict"
    response = make_request('get', url)
    
    if not response:
        logger.error("GET预测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证响应数据结构
        if response.status_code == 200 and 'data' in result:
            data = result['data']
            required_fields = ['instances', 'current_qps', 'timestamp']
            missing_fields = [field for field in required_fields if field not in data]
            
            if missing_fields:
                logger.warning(f"响应缺少字段: {missing_fields}")
            else:
                logger.info("响应数据结构验证通过")
                
                # 验证实例数范围
                instances = data.get('instances', 0)
                if 1 <= instances <= 20:
                    logger.info(f"实例数在有效范围内: {instances}")
                else:
                    logger.warning(f"实例数可能超出范围: {instances}")
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_post():
    """测试POST预测接口"""
    print_header("测试POST预测接口")
    
    url = f"{API_BASE_URL}/predict"
    data = {
        "current_qps": 150.5,
        "timestamp": datetime.utcnow().isoformat()
    }
    
    response = make_request('post', url, data)
    
    if not response:
        logger.error("POST预测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证响应数据
        if response.status_code == 200 and 'data' in result:
            response_data = result['data']
            
            # 验证QPS是否正确返回
            if response_data.get('current_qps') == 150.5:
                logger.info("QPS值正确返回")
            else:
                logger.warning(f"QPS值不匹配: 期望150.5, 实际{response_data.get('current_qps')}")
            
            # 验证实例数
            instances = response_data.get('instances', 0)
            if 1 <= instances <= 20:
                logger.info(f"预测实例数: {instances}")
            else:
                logger.warning(f"实例数可能超出范围: {instances}")
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_zero_qps():
    """测试零QPS预测"""
    print_header("测试零QPS预测")
    
    url = f"{API_BASE_URL}/predict"
    data = {
        "current_qps": 0.0,
        "timestamp": datetime.utcnow().isoformat()
    }
    
    response = make_request('post', url, data)
    
    if not response:
        logger.error("零QPS预测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证零QPS场景
        if response.status_code == 200 and 'data' in result:
            response_data = result['data']
            
            # 验证QPS为0
            if response_data.get('current_qps') == 0.0:
                logger.info("零QPS正确处理")
            
            # 验证返回最小实例数
            instances = response_data.get('instances', 0)
            if instances == 1:
                logger.info("零QPS时正确返回最小实例数")
            else:
                logger.warning(f"零QPS时实例数异常: {instances}")
            
            # 验证预测类型
            prediction_type = response_data.get('prediction_type')
            if prediction_type == 'threshold_based':
                logger.info("零QPS时使用基于阈值的预测")
            
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_low_qps():
    """测试低QPS预测"""
    print_header("测试低QPS预测")
    
    url = f"{API_BASE_URL}/predict"
    data = {
        "current_qps": 3.5,  # 低于阈值的QPS
        "timestamp": datetime.utcnow().isoformat()
    }
    
    response = make_request('post', url, data)
    
    if not response:
        logger.error("低QPS预测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证低QPS场景
        if response.status_code == 200 and 'data' in result:
            response_data = result['data']
            
            # 验证QPS值
            if response_data.get('current_qps') == 3.5:
                logger.info("低QPS值正确处理")
            
            # 验证实例数（低QPS应该返回最小实例数）
            instances = response_data.get('instances', 0)
            if instances == 1:
                logger.info("低QPS时正确返回最小实例数")
            else:
                logger.info(f"低QPS预测实例数: {instances}")
                
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_high_qps():
    """测试高QPS预测"""
    print_header("测试高QPS预测")
    
    url = f"{API_BASE_URL}/predict"
    data = {
        "current_qps": 1000.0,  # 高QPS
        "timestamp": datetime.utcnow().isoformat()
    }
    
    response = make_request('post', url, data)
    
    if not response:
        logger.error("高QPS预测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证高QPS场景
        if response.status_code == 200 and 'data' in result:
            response_data = result['data']
            
            # 验证QPS值
            if response_data.get('current_qps') == 1000.0:
                logger.info("高QPS值正确处理")
            
            # 验证实例数（高QPS应该返回较多实例）
            instances = response_data.get('instances', 0)
            if instances > 5:
                logger.info(f"高QPS预测合理的实例数: {instances}")
            else:
                logger.warning(f"高QPS预测实例数可能偏低: {instances}")
                
            # 验证置信度
            confidence = response_data.get('confidence', 0)
            logger.info(f"预测置信度: {confidence}")
                
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_prediction_invalid_qps():
    """测试无效QPS参数"""
    print_header("测试无效QPS参数")
    
    url = f"{API_BASE_URL}/predict"
    data = {
        "current_qps": -10.0,  # 无效的负数QPS
        "timestamp": datetime.utcnow().isoformat()
    }
    
    response = make_request('post', url, data)
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证错误处理
        if response.status_code == 400:
            logger.info("正确识别并拒绝无效QPS参数")
        else:
            logger.warning("接受了无效QPS参数，可能需要加强验证")
            
    except Exception as e:
        logger.error(f"解析响应时出错: {str(e)}")
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 400,  # 期望返回400错误状态码
        "status_code": response.status_code,
        "response": result
    }

def test_trend_prediction():
    """测试趋势预测API"""
    print_header("测试趋势预测API")
    
    url = f"{API_BASE_URL}/predict/trend"
    data = {
        "current_qps": 100.0,
        "hours_ahead": 6,
        "timestamp": datetime.utcnow().isoformat()
    }
    
    response = make_request('post', url, data)
    
    if not response:
        logger.error("趋势预测API请求失败")
        return {"success": False, "message": "API请求失败"}
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证趋势预测响应
        if response.status_code == 200 and 'data' in result:
            response_data = result['data']
            
            # 验证预测数据
            forecast = response_data.get('forecast', [])
            if isinstance(forecast, list) and len(forecast) > 0:
                logger.info(f"趋势预测返回 {len(forecast)} 个预测点")
                
                # 验证预测点数据结构
                first_point = forecast[0] if forecast else {}
                required_fields = ['timestamp', 'qps', 'instances']
                missing_fields = [field for field in required_fields if field not in first_point]
                
                if not missing_fields:
                    logger.info("趋势预测数据结构验证通过")
                else:
                    logger.warning(f"趋势预测数据缺少字段: {missing_fields}")
            else:
                logger.warning("趋势预测返回的预测数据为空")
                
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_model_validation():
    """测试模型验证API"""
    print_header("测试模型验证API")
    
    url = f"{API_BASE_URL}/predict/model/validate"
    response = make_request('get', url)
    
    print(f"状态码: {response.status_code}")
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # 验证模型状态
        if response.status_code == 200 and 'data' in result:
            response_data = result['data']
            
            # 检查模型加载状态
            model_loaded = response_data.get('model_loaded', False)
            scaler_loaded = response_data.get('scaler_loaded', False)
            
            if model_loaded and scaler_loaded:
                logger.info("模型和标准化器都已正确加载")
            else:
                logger.warning(f"模型加载状态: 模型={model_loaded}, 标准化器={scaler_loaded}")
                
            # 检查模型版本
            model_version = response_data.get('model_version')
            if model_version:
                logger.info(f"模型版本: {model_version}")
                
    except:
        result = {"raw_response": response.text}
        print(f"响应内容: {response.text}")
    
    return {
        "success": response.status_code == 200,
        "status_code": response.status_code,
        "response": result
    }

def test_full_prediction_workflow():
    """测试完整预测工作流"""
    print_header("测试完整预测工作流")
    
    workflow_results = {}
    
    try:
        # 1. 检查服务健康状态
        logger.info("步骤1: 检查预测服务健康状态")
        health_result = test_prediction_health()
        workflow_results['health_check'] = health_result
        
        if not health_result['success']:
            logger.warning("预测服务健康检查失败，但继续测试")
        
        # 2. 验证模型状态
        logger.info("步骤2: 验证模型状态")
        model_result = test_model_validation()
        workflow_results['model_validation'] = model_result
        
        # 3. 执行不同场景的预测
        logger.info("步骤3: 执行多场景预测测试")
        
        # 正常QPS预测
        normal_prediction = test_prediction_post()
        workflow_results['normal_prediction'] = normal_prediction
        
        # 等待1秒避免请求过快
        time.sleep(1)
        
        # 零QPS预测
        zero_prediction = test_prediction_zero_qps()
        workflow_results['zero_qps_prediction'] = zero_prediction
        
        time.sleep(1)
        
        # 高QPS预测
        high_prediction = test_prediction_high_qps()
        workflow_results['high_qps_prediction'] = high_prediction
        
        time.sleep(1)
        
        # 4. 趋势预测
        logger.info("步骤4: 执行趋势预测")
        trend_result = test_trend_prediction()
        workflow_results['trend_prediction'] = trend_result
        
        # 5. 分析工作流结果
        logger.info("步骤5: 分析工作流结果")
        
        successful_steps = sum(1 for result in workflow_results.values() if result.get('success'))
        total_steps = len(workflow_results)
        
        logger.info(f"工作流完成: {successful_steps}/{total_steps} 步骤成功")
        
        return {
            "success": successful_steps >= total_steps * 0.7,  # 70%成功率认为工作流成功
            "successful_steps": successful_steps,
            "total_steps": total_steps,
            "workflow_results": workflow_results
        }
        
    except Exception as e:
        logger.error(f"工作流执行失败: {str(e)}")
        return {
            "success": False,
            "error": str(e),
            "workflow_results": workflow_results
        }

def save_test_results(results):
    """保存测试结果到JSON文件"""
    with open(TEST_RESULT_FILE, "w", encoding="utf-8") as f:
        json.dump(results, f, ensure_ascii=False, indent=2)
    logger.info(f"测试结果已保存到 {TEST_RESULT_FILE}")

def main():
    """主测试流程"""
    start_time = time.time()
    
    # 记录测试开始
    logger.info("=" * 50)
    logger.info("开始Kubernetes预测服务功能测试")
    logger.info("=" * 50)
    
    # 初始化测试结果
    results = {
        "timestamp": datetime.utcnow().isoformat(),
        "results": {},
        "environment_setup": False
    }
    
    try:
        # 设置测试环境
        setup_success = setup_test_environment()
        results["environment_setup"] = setup_success
        
        # 执行测试
        logger.info("开始执行API测试...")
        
        # 基础健康检查
        health_result = test_health()
        results["results"]["health"] = health_result
        
        prediction_health_result = test_prediction_health()
        results["results"]["prediction_health"] = prediction_health_result
        
        # 预测API测试
        prediction_get_result = test_prediction_get()
        results["results"]["prediction_get"] = prediction_get_result
        
        prediction_post_result = test_prediction_post()
        results["results"]["prediction_post"] = prediction_post_result
        
        # 特殊场景测试
        zero_qps_result = test_prediction_zero_qps()
        results["results"]["zero_qps_prediction"] = zero_qps_result
        
        low_qps_result = test_prediction_low_qps()
        results["results"]["low_qps_prediction"] = low_qps_result
        
        high_qps_result = test_prediction_high_qps()
        results["results"]["high_qps_prediction"] = high_qps_result
        
        invalid_qps_result = test_prediction_invalid_qps()
        results["results"]["invalid_qps_prediction"] = invalid_qps_result
        
        # 趋势预测测试
        trend_result = test_trend_prediction()
        results["results"]["trend_prediction"] = trend_result
        
        # 模型验证测试
        model_validation_result = test_model_validation()
        results["results"]["model_validation"] = model_validation_result
        
        # 完整工作流测试
        workflow_result = test_full_prediction_workflow()
        results["results"]["full_workflow"] = workflow_result
        
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
            "duration_seconds": duration
        }
        
        # 打印摘要
        print_header("测试摘要")
        print(f"总测试数: {total_tests}")
        print(f"通过测试数: {passed_tests}")
        print(f"成功率: {success_rate:.2f}%")
        print(f"测试持续时间: {duration:.2f} 秒")
        
        # 打印详细结果
        print("\n详细测试结果:")
        for test_name, test_result in results["results"].items():
            status = "✅ 通过" if test_result.get("success") else "❌ 失败"
            status_code = test_result.get("status_code", "N/A")
            print(f"  {test_name}: {status} (状态码: {status_code})")
        
    except Exception as e:
        logger.error(f"测试过程中发生错误: {str(e)}")
        results["error"] = str(e)
    finally:
        # 保存测试结果
        save_test_results(results)
        
        logger.info("=" * 50)
        logger.info("Kubernetes预测服务功能测试完成")
        logger.info("=" * 50)
        
        return results

if __name__ == "__main__":
    main()