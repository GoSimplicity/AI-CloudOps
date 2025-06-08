from flask import Blueprint, request, jsonify
from datetime import datetime, timedelta
import asyncio
import logging
from app.core.rca.analyzer import RCAAnalyzer
from app.models.request_models import RCARequest
from app.utils.validators import validate_time_range, validate_metric_list
from app.config.settings import config

logger = logging.getLogger("aiops.rca")

rca_bp = Blueprint('rca', __name__)

# 初始化分析器
rca_analyzer = RCAAnalyzer()

@rca_bp.route('/rca', methods=['POST'])
def root_cause_analysis():
    """根因分析接口"""
    try:
        # 解析请求参数
        data = request.get_json() or {}
        
        # 验证请求
        try:
            rca_request = RCARequest(**data)
        except Exception as e:
            logger.warning(f"RCA请求参数错误: {str(e)}")
            return jsonify({"error": f"请求参数错误: {str(e)}"}), 400
        
        # 验证时间范围
        if not validate_time_range(rca_request.start_time, rca_request.end_time):
            return jsonify({"error": "无效的时间范围"}), 400
        
        # 检查时间范围限制
        time_diff = (rca_request.end_time - rca_request.start_time).total_seconds() / 60
        if time_diff > config.rca.max_time_range:
            return jsonify({
                "error": f"时间范围超过最大限制 {config.rca.max_time_range} 分钟，当前: {time_diff:.1f} 分钟"
            }), 400
        
        # 验证指标列表
        if rca_request.metrics and not validate_metric_list(rca_request.metrics):
            return jsonify({"error": "指标名称格式错误"}), 400
        
        logger.info(f"执行根因分析: {rca_request.start_time} - {rca_request.end_time}, 指标数: {len(rca_request.metrics)}")
        
        # 执行根因分析
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            result = loop.run_until_complete(
                rca_analyzer.analyze(
                    rca_request.start_time,
                    rca_request.end_time,
                    rca_request.metrics
                )
            )
        finally:
            loop.close()
        
        # 记录分析结果
        if 'error' not in result:
            anomaly_count = len(result.get('anomalies', {}))
            candidate_count = len(result.get('root_cause_candidates', []))
            logger.info(f"根因分析完成: 异常指标={anomaly_count}, 根因候选={candidate_count}")
        else:
            logger.error(f"根因分析失败: {result['error']}")
        
        return jsonify(result)
        
    except Exception as e:
        logger.error(f"根因分析请求失败: {str(e)}")
        return jsonify({"error": f"处理请求失败: {str(e)}"}), 500

@rca_bp.route('/rca/incident', methods=['POST'])
def analyze_incident():
    """分析特定事件"""
    try:
        data = request.get_json() or {}
        
        # 解析事件分析参数
        start_time_str = data.get('start_time')
        end_time_str = data.get('end_time')
        affected_services = data.get('affected_services', [])
        symptoms = data.get('symptoms', [])
        
        # 验证必要参数
        if not affected_services:
            return jsonify({"error": "必须指定受影响的服务"}), 400
        
        if not symptoms:
            return jsonify({"error": "必须描述症状"}), 400
        
        # 解析时间
        try:
            if start_time_str:
                start_time = datetime.fromisoformat(start_time_str.replace('Z', '+00:00'))
            else:
                start_time = datetime.utcnow() - timedelta(minutes=30)
            
            if end_time_str:
                end_time = datetime.fromisoformat(end_time_str.replace('Z', '+00:00'))
            else:
                end_time = datetime.utcnow()
        except ValueError as e:
            return jsonify({"error": f"时间格式错误: {str(e)}"}), 400
        
        logger.info(f"分析特定事件: 服务={affected_services}, 症状={symptoms}")
        
        # 执行事件分析
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            result = loop.run_until_complete(
                rca_analyzer.analyze_specific_incident(
                    start_time, end_time, affected_services, symptoms
                )
            )
        finally:
            loop.close()
        
        return jsonify(result)
        
    except Exception as e:
        logger.error(f"事件分析失败: {str(e)}")
        return jsonify({"error": f"事件分析失败: {str(e)}"}), 500

@rca_bp.route('/rca/metrics', methods=['GET'])
def get_available_metrics():
    """获取可用的监控指标"""
    try:
        # 返回默认指标列表和分类
        metrics_info = {
            "default_metrics": config.rca.default_metrics,
            "categories": {
                "CPU": [
                    "container_cpu_usage_seconds_total",
                    "node_cpu_seconds_total"
                ],
                "Memory": [
                    "container_memory_working_set_bytes",
                    "node_memory_MemFree_bytes",
                    "container_memory_usage_bytes"
                ],
                "Network": [
                    "container_network_receive_bytes_total",
                    "container_network_transmit_bytes_total"
                ],
                "Kubernetes": [
                    "kube_pod_container_status_restarts_total",
                    "kube_pod_status_phase",
                    "kube_deployment_status_replicas"
                ],
                "HTTP": [
                    "kubelet_http_requests_duration_seconds_count",
                    "kubelet_http_requests_duration_seconds_sum"
                ]
            },
            "config": {
                "default_time_range": config.rca.default_time_range,
                "max_time_range": config.rca.max_time_range,
                "anomaly_threshold": config.rca.anomaly_threshold,
                "correlation_threshold": config.rca.correlation_threshold
            }
        }
        
        # 尝试从Prometheus获取可用指标（可选）
        try:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                available_metrics = loop.run_until_complete(
                    rca_analyzer.prometheus.get_available_metrics()
                )
                if available_metrics:
                    metrics_info["available_from_prometheus"] = available_metrics[:50]  # 限制返回数量
            finally:
                loop.close()
        except Exception as e:
            logger.warning(f"获取Prometheus指标失败: {str(e)}")
            metrics_info["prometheus_error"] = str(e)
        
        return jsonify(metrics_info)
        
    except Exception as e:
        logger.error(f"获取指标列表失败: {str(e)}")
        return jsonify({"error": str(e)}), 500

@rca_bp.route('/rca/health', methods=['GET'])
def rca_health():
    """根因分析服务健康检查"""
    try:
        # 检查各组件状态
        prometheus_healthy = rca_analyzer.prometheus.is_healthy()
        llm_healthy = rca_analyzer.llm.is_healthy()
        
        # 检查检测器和相关性分析器（它们是纯计算模块，通常健康）
        detector_healthy = True
        correlator_healthy = True
        
        health_status = {
            "status": "healthy" if prometheus_healthy else "degraded",
            "components": {
                "prometheus": prometheus_healthy,
                "llm": llm_healthy,
                "detector": detector_healthy,
                "correlator": correlator_healthy
            },
            "timestamp": datetime.utcnow().isoformat(),
            "config": {
                "anomaly_threshold": rca_analyzer.detector.anomaly_threshold,
                "correlation_threshold": rca_analyzer.correlator.correlation_threshold
            }
        }
        
        # 如果Prometheus不健康，服务仍可用但功能受限
        status_code = 200 if prometheus_healthy else 200  # 保持200，因为服务仍可用
        
        return jsonify(health_status), status_code
        
    except Exception as e:
        logger.error(f"RCA健康检查失败: {str(e)}")
        return jsonify({
            "status": "error",
            "error": str(e),
            "timestamp": datetime.utcnow().isoformat()
        }), 500

@rca_bp.route('/rca/config', methods=['GET'])
def get_rca_config():
    """获取根因分析配置"""
    try:
        rca_config = {
            "anomaly_detection": {
                "threshold": config.rca.anomaly_threshold,
                "methods": ["zscore", "iqr", "isolation_forest", "dbscan", "moving_average"]
            },
            "correlation_analysis": {
                "threshold": config.rca.correlation_threshold,
                "methods": ["pearson", "spearman"]
            },
            "time_range": {
                "default_minutes": config.rca.default_time_range,
                "max_minutes": config.rca.max_time_range
            },
            "metrics": {
                "default_count": len(config.rca.default_metrics),
                "default_metrics": config.rca.default_metrics
            }
        }
        
        return jsonify(rca_config)
        
    except Exception as e:
        logger.error(f"获取RCA配置失败: {str(e)}")
        return jsonify({"error": str(e)}), 500

@rca_bp.route('/rca/config', methods=['PUT'])
def update_rca_config():
    """更新根因分析配置"""
    try:
        data = request.get_json() or {}
        
        updated_fields = []
        
        # 更新异常检测阈值
        if 'anomaly_threshold' in data:
            new_threshold = float(data['anomaly_threshold'])
            if 0 < new_threshold <= 1:
                rca_analyzer.detector.update_threshold(new_threshold)
                updated_fields.append(f"anomaly_threshold: {new_threshold}")
            else:
                return jsonify({"error": "异常检测阈值必须在0-1之间"}), 400
        
        # 更新相关性阈值
        if 'correlation_threshold' in data:
            new_threshold = float(data['correlation_threshold'])
            if 0 < new_threshold <= 1:
                rca_analyzer.correlator.correlation_threshold = new_threshold
                updated_fields.append(f"correlation_threshold: {new_threshold}")
            else:
                return jsonify({"error": "相关性阈值必须在0-1之间"}), 400
        
        if updated_fields:
            logger.info(f"RCA配置已更新: {', '.join(updated_fields)}")
            return jsonify({
                "message": "配置更新成功",
                "updated_fields": updated_fields,
                "timestamp": datetime.utcnow().isoformat()
            })
        else:
            return jsonify({"message": "没有可更新的配置项"}), 400
        
    except Exception as e:
        logger.error(f"更新RCA配置失败: {str(e)}")
        return jsonify({"error": str(e)}), 500