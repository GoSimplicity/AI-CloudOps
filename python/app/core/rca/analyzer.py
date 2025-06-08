import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional
import pandas as pd
from app.core.rca.detector import AnomalyDetector
from app.core.rca.correlator import CorrelationAnalyzer
from app.services.prometheus import PrometheusService
from app.services.llm import LLMService
from app.models.response_models import RCAResponse, RootCauseCandidate, AnomalyInfo
from app.config.settings import config

logger = logging.getLogger("aiops.rca")

class RCAAnalyzer:
    def __init__(self):
        self.prometheus = PrometheusService()
        self.detector = AnomalyDetector(config.rca.anomaly_threshold)
        self.correlator = CorrelationAnalyzer(config.rca.correlation_threshold)
        self.llm = LLMService()
        logger.info("根因分析器初始化完成")
    
    async def analyze(
        self, 
        start_time: datetime, 
        end_time: datetime,
        metrics: Optional[List[str]] = None
    ) -> Dict:
        """执行根因分析"""
        try:
            logger.info(f"开始根因分析: {start_time} - {end_time}")
            
            # 使用默认指标如果未提供
            if not metrics:
                metrics = config.rca.default_metrics
            
            # 收集指标数据
            metrics_data = await self._collect_metrics_data(
                start_time, end_time, metrics
            )
            
            if not metrics_data:
                return {"error": "未获取到有效的监控数据"}
            
            logger.info(f"收集到 {len(metrics_data)} 个指标的数据")
            
            # 异常检测
            anomalies = await self.detector.detect_anomalies(metrics_data)
            logger.info(f"检测到 {len(anomalies)} 个指标存在异常")
            
            # 相关性分析
            correlations = await self.correlator.analyze_correlations(metrics_data)
            logger.info(f"分析了 {len(correlations)} 个指标的相关性")
            
            # 生成根因候选
            root_causes = self._generate_root_cause_candidates(anomalies, correlations)
            
            # 生成LLM摘要（可选）
            summary = await self._generate_summary(anomalies, correlations, root_causes)
            
            # 构建响应
            response = {
                "status": "success",
                "anomalies": {
                    metric: AnomalyInfo(**info).__dict__ 
                    for metric, info in anomalies.items()
                },
                "correlations": correlations,
                "root_cause_candidates": [
                    RootCauseCandidate(**candidate).__dict__
                    for candidate in root_causes
                ],
                "analysis_time": datetime.utcnow().isoformat(),
                "time_range": {
                    "start": start_time.isoformat(),
                    "end": end_time.isoformat()
                },
                "metrics_analyzed": list(metrics_data.keys()),
                "summary": summary,
                "statistics": {
                    "total_metrics": len(metrics_data),
                    "anomalous_metrics": len(anomalies),
                    "correlation_pairs": sum(len(corrs) for corrs in correlations.values()),
                    "analysis_duration": (datetime.utcnow() - start_time).total_seconds()
                }
            }
            
            logger.info("根因分析完成")
            return response
            
        except Exception as e:
            logger.error(f"根因分析失败: {str(e)}")
            return {"error": f"分析失败: {str(e)}"}
    
    async def _collect_metrics_data(
        self, 
        start_time: datetime, 
        end_time: datetime,
        metrics: List[str]
    ) -> Dict[str, pd.DataFrame]:
        """收集指标数据"""
        metrics_data = {}
        
        for metric in metrics:
            try:
                data = await self.prometheus.query_range(
                    metric, start_time, end_time, "1m"
                )
                
                if data is not None and not data.empty:
                    # 处理多个时间序列
                    if len(data) > 0:
                        # 如果有多个系列，按标签分组
                        if 'label_pod' in data.columns:
                            # 按pod分组
                            grouped_data = {}
                            for pod in data['label_pod'].unique():
                                if pd.notna(pod):
                                    pod_data = data[data['label_pod'] == pod]
                                    if not pod_data.empty:
                                        metric_name = f"{metric}|pod:{pod}"
                                        grouped_data[metric_name] = pod_data[['value']]
                            metrics_data.update(grouped_data)
                        elif 'label_container' in data.columns:
                            # 按容器分组
                            grouped_data = {}
                            for container in data['label_container'].unique():
                                if pd.notna(container):
                                    container_data = data[data['label_container'] == container]
                                    if not container_data.empty:
                                        metric_name = f"{metric}|container:{container}"
                                        grouped_data[metric_name] = container_data[['value']]
                            metrics_data.update(grouped_data)
                        else:
                            # 单个序列或聚合数据
                            metrics_data[metric] = data[['value']]
                
            except Exception as e:
                logger.warning(f"获取指标 {metric} 失败: {str(e)}")
                continue
        
        # 过滤掉空数据
        metrics_data = {k: v for k, v in metrics_data.items() if not v.empty}
        
        logger.info(f"成功收集 {len(metrics_data)} 个时间序列数据")
        return metrics_data
    
    def _generate_root_cause_candidates(
        self, 
        anomalies: Dict, 
        correlations: Dict
    ) -> List[Dict]:
        """生成根因候选列表"""
        candidates = []
        
        try:
            # 基于异常分数生成候选
            for metric, anomaly_info in anomalies.items():
                if anomaly_info.get('count', 0) > 0:
                    # 计算置信度
                    confidence = self._calculate_confidence(
                        anomaly_info, correlations.get(metric, [])
                    )
                    
                    # 生成描述
                    description = self._generate_description(metric, anomaly_info)
                    
                    candidate = {
                        "metric": metric,
                        "confidence": confidence,
                        "first_occurrence": anomaly_info.get('first_occurrence'),
                        "anomaly_count": anomaly_info.get('count'),
                        "related_metrics": correlations.get(metric, []),
                        "description": description
                    }
                    candidates.append(candidate)
            
            # 按置信度排序
            candidates.sort(key=lambda x: x['confidence'], reverse=True)
            
            # 返回前5个候选
            return candidates[:5]
            
        except Exception as e:
            logger.error(f"生成根因候选失败: {str(e)}")
            return []
    
    def _calculate_confidence(self, anomaly_info: Dict, related_metrics: List) -> float:
        """计算根因置信度"""
        try:
            # 基础置信度来自异常分数
            base_confidence = min(anomaly_info.get('max_score', 0), 1.0)
            
            # 异常持续性加权
            count = anomaly_info.get('count', 0)
            count_factor = min(count / 20, 0.3)  # 最多加0.3分
            
            # 相关性加权
            correlation_factor = min(len(related_metrics) * 0.05, 0.2)  # 最多加0.2分
            
            # 检测方法一致性加权
            detection_methods = anomaly_info.get('detection_methods', {})
            method_consistency = sum(1 for v in detection_methods.values() if isinstance(v, (int, float)) and v > 0)
            consistency_factor = min(method_consistency * 0.05, 0.15)  # 最多加0.15分
            
            # 综合置信度
            confidence = base_confidence + count_factor + correlation_factor + consistency_factor
            
            return min(confidence, 1.0)
            
        except Exception:
            return 0.0
    
    def _generate_description(self, metric: str, anomaly_info: Dict) -> str:
        """生成根因描述"""
        try:
            count = anomaly_info.get('count', 0)
            max_score = anomaly_info.get('max_score', 0)
            avg_score = anomaly_info.get('avg_score', 0)
            
            # 基于指标名称生成描述
            metric_lower = metric.lower()
            
            if 'cpu' in metric_lower:
                return f"CPU使用率异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            elif 'memory' in metric_lower:
                return f"内存使用异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            elif 'restart' in metric_lower:
                return f"容器重启异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            elif any(keyword in metric_lower for keyword in ['network', 'http', 'request']):
                return f"网络/HTTP请求异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            elif 'disk' in metric_lower or 'storage' in metric_lower:
                return f"磁盘/存储异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            elif 'node' in metric_lower:
                return f"节点指标异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            elif 'pod' in metric_lower:
                return f"Pod状态异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
            else:
                return f"指标 {metric} 异常，检测到 {count} 个异常点，最高异常分数 {max_score:.2f}，平均异常分数 {avg_score:.2f}"
                
        except Exception:
            return f"指标 {metric} 存在异常"
    
    async def _generate_summary(
        self, 
        anomalies: Dict, 
        correlations: Dict, 
        candidates: List[Dict]
    ) -> Optional[str]:
        """生成AI摘要"""
        try:
            if not candidates:
                return "未发现明显的异常模式，系统运行正常。"
            
            # 调用LLM生成摘要
            summary = await self.llm.generate_rca_summary(
                anomalies, correlations, candidates
            )
            
            return summary or "无法生成分析摘要，但检测到异常模式。"
            
        except Exception as e:
            logger.error(f"生成摘要失败: {str(e)}")
            return None
    
    async def analyze_specific_incident(
        self,
        start_time: datetime,
        end_time: datetime,
        affected_services: List[str],
        symptoms: List[str]
    ) -> Dict:
        """分析特定事件的根因"""
        try:
            logger.info(f"分析特定事件: 服务={affected_services}, 症状={symptoms}")
            
            # 基于受影响的服务和症状，选择相关指标
            relevant_metrics = self._select_relevant_metrics(affected_services, symptoms)
            
            # 执行针对性分析
            result = await self.analyze(start_time, end_time, relevant_metrics)
            
            if 'error' not in result:
                # 添加事件特定的分析结果
                result['incident_analysis'] = {
                    'affected_services': affected_services,
                    'reported_symptoms': symptoms,
                    'relevant_metrics': relevant_metrics,
                    'recommendation': self._generate_incident_recommendation(
                        result.get('root_cause_candidates', []),
                        affected_services,
                        symptoms
                    )
                }
            
            return result
            
        except Exception as e:
            logger.error(f"特定事件分析失败: {str(e)}")
            return {"error": f"事件分析失败: {str(e)}"}
    
    def _select_relevant_metrics(
        self,
        services: List[str],
        symptoms: List[str]
    ) -> List[str]:
        """基于服务和症状选择相关指标"""
        relevant_metrics = set(config.rca.default_metrics)
        
        # 基于症状添加特定指标
        for symptom in symptoms:
            symptom_lower = symptom.lower()
            if 'slow' in symptom_lower or 'latency' in symptom_lower:
                relevant_metrics.update([
                    'kubelet_http_requests_duration_seconds_sum',
                    'kubelet_http_requests_duration_seconds_count'
                ])
            elif 'error' in symptom_lower or 'fail' in symptom_lower:
                relevant_metrics.update([
                    'kube_pod_container_status_restarts_total'
                ])
            elif 'cpu' in symptom_lower:
                relevant_metrics.update([
                    'container_cpu_usage_seconds_total',
                    'node_cpu_seconds_total'
                ])
            elif 'memory' in symptom_lower:
                relevant_metrics.update([
                    'container_memory_working_set_bytes',
                    'node_memory_MemFree_bytes'
                ])
        
        return list(relevant_metrics)
    
    def _generate_incident_recommendation(
        self,
        root_causes: List[Dict],
        services: List[str],
        symptoms: List[str]
    ) -> str:
        """生成事件处理建议"""
        if not root_causes:
            return "建议检查服务配置和资源分配，监控系统负载变化。"
        
        top_cause = root_causes[0]
        metric = top_cause.get('metric', '')
        confidence = top_cause.get('confidence', 0)
        
        recommendations = []
        
        if 'cpu' in metric.lower():
            recommendations.append("检查CPU使用率，考虑扩容或优化应用性能")
        elif 'memory' in metric.lower():
            recommendations.append("检查内存使用情况，可能需要增加内存限制或优化内存使用")
        elif 'restart' in metric.lower():
            recommendations.append("检查容器重启原因，查看相关日志和健康检查配置")
        elif 'network' in metric.lower() or 'http' in metric.lower():
            recommendations.append("检查网络连接和服务间通信，查看负载均衡配置")
        
        if confidence > 0.8:
            recommendations.append(f"根因分析置信度较高({confidence:.2f})，建议优先处理该问题")
        elif confidence < 0.5:
            recommendations.append("根因分析置信度较低，建议进行更详细的调查")
        
        return "; ".join(recommendations) if recommendations else "建议进行详细的系统检查和日志分析。"