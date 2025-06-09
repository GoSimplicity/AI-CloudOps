import logging
import sys
import io
import contextlib
from typing import Dict, Any, List, Optional
from langchain_core.tools import tool
from langchain_experimental.tools import PythonAstREPLTool
from langchain_openai import ChatOpenAI
import pandas as pd
import numpy as np
import json
from app.config.settings import config

logger = logging.getLogger("aiops.coder")

class CoderAgent:
    def __init__(self):
        self.llm = ChatOpenAI(
            model=config.llm.model,
            api_key=config.llm.api_key,
            base_url=config.llm.base_url
        )
        self.python_tool = PythonAstREPLTool()
        self.execution_context = {}
        logger.info("Coder Agent初始化完成")
    
    @tool
    async def analyze_metrics_data(self, metrics_data: str) -> str:
        """分析指标数据并提供洞察"""
        try:
            # 解析指标数据
            if isinstance(metrics_data, str):
                try:
                    data = json.loads(metrics_data)
                except json.JSONDecodeError:
                    return "无法解析指标数据，请提供有效的JSON格式数据"
            else:
                data = metrics_data
            
            # 执行数据分析
            analysis_code = f"""
import pandas as pd
import numpy as np
import json
from datetime import datetime

# 解析数据
data = {json.dumps(data)}

analysis_results = []

for metric_name, metric_info in data.items():
    if isinstance(metric_info, dict) and 'count' in metric_info:
        result = {{
            'metric': metric_name,
            'anomaly_count': metric_info.get('count', 0),
            'max_score': metric_info.get('max_score', 0),
            'avg_score': metric_info.get('avg_score', 0),
            'first_occurrence': metric_info.get('first_occurrence', 'N/A')
        }}
        analysis_results.append(result)

# 排序并分析
analysis_results.sort(key=lambda x: x['max_score'], reverse=True)

# 生成分析报告
report = "**指标异常分析报告:**\\n\\n"

if analysis_results:
    report += f"总共分析了 {{len(analysis_results)}} 个异常指标\\n\\n"
    
    report += "**TOP 3 异常指标:**\\n"
    for i, result in enumerate(analysis_results[:3], 1):
        report += f"{{i}}. {{result['metric']}}\\n"
        report += f"   - 异常次数: {{result['anomaly_count']}}\\n"
        report += f"   - 最高异常分数: {{result['max_score']:.3f}}\\n"
        report += f"   - 平均异常分数: {{result['avg_score']:.3f}}\\n"
        report += f"   - 首次发现: {{result['first_occurrence']}}\\n\\n"
    
    # 统计分析
    total_anomalies = sum(r['anomaly_count'] for r in analysis_results)
    avg_max_score = np.mean([r['max_score'] for r in analysis_results])
    
    report += f"**统计摘要:**\\n"
    report += f"- 总异常次数: {{total_anomalies}}\\n"
    report += f"- 平均最高异常分数: {{avg_max_score:.3f}}\\n"
    report += f"- 异常指标占比: {{len(analysis_results)}} / {{len(data)}}\\n"
    
    # 严重程度分析
    high_severity = sum(1 for r in analysis_results if r['max_score'] > 0.8)
    medium_severity = sum(1 for r in analysis_results if 0.5 < r['max_score'] <= 0.8)
    low_severity = sum(1 for r in analysis_results if r['max_score'] <= 0.5)
    
    report += f"\\n**严重程度分布:**\\n"
    report += f"- 高危 (>0.8): {{high_severity}} 个\\n"
    report += f"- 中危 (0.5-0.8): {{medium_severity}} 个\\n"
    report += f"- 低危 (≤0.5): {{low_severity}} 个\\n"
    
else:
    report += "未发现异常指标"

print(report)
"""
            
            # 执行分析代码
            result = self._execute_python_code(analysis_code)
            return result
            
        except Exception as e:
            logger.error(f"分析指标数据失败: {str(e)}")
            return f"分析失败: {str(e)}"
    
    @tool
    async def calculate_correlation_insights(self, correlation_data: str) -> str:
        """计算相关性洞察"""
        try:
            # 解析相关性数据
            if isinstance(correlation_data, str):
                try:
                    data = json.loads(correlation_data)
                except json.JSONDecodeError:
                    return "无法解析相关性数据"
            else:
                data = correlation_data
            
            analysis_code = f"""
import json
import numpy as np

# 解析相关性数据
correlation_data = {json.dumps(data)}

report = "**指标相关性分析报告:**\\n\\n"

if correlation_data:
    total_correlations = sum(len(corrs) for corrs in correlation_data.values())
    
    report += f"总共发现 {{total_correlations}} 组相关性\\n\\n"
    
    # 分析每个指标的相关性
    report += "**指标相关性详情:**\\n"
    
    for metric, correlations in correlation_data.items():
        if correlations:
            report += f"\\n**{{metric}}:**\\n"
            
            # 按相关性强度排序
            sorted_corrs = sorted(correlations, key=lambda x: abs(x[1]), reverse=True)
            
            for related_metric, corr_value in sorted_corrs[:3]:  # 只显示前3个
                strength = "强" if abs(corr_value) > 0.8 else "中" if abs(corr_value) > 0.6 else "弱"
                direction = "正" if corr_value > 0 else "负"
                
                report += f"  - 与 {{related_metric}}: {{corr_value:.3f}} ({{direction}}{{strength}}相关)\\n"
    
    # 相关性强度统计
    all_correlations = []
    for correlations in correlation_data.values():
        for _, corr_value in correlations:
            all_correlations.append(abs(corr_value))
    
    if all_correlations:
        avg_correlation = np.mean(all_correlations)
        max_correlation = np.max(all_correlations)
        
        strong_correlations = sum(1 for c in all_correlations if c > 0.8)
        medium_correlations = sum(1 for c in all_correlations if 0.6 < c <= 0.8)
        weak_correlations = sum(1 for c in all_correlations if c <= 0.6)
        
        report += f"\\n**相关性统计:**\\n"
        report += f"- 平均相关性强度: {{avg_correlation:.3f}}\\n"
        report += f"- 最高相关性: {{max_correlation:.3f}}\\n"
        report += f"- 强相关 (>0.8): {{strong_correlations}} 组\\n"
        report += f"- 中等相关 (0.6-0.8): {{medium_correlations}} 组\\n"
        report += f"- 弱相关 (≤0.6): {{weak_correlations}} 组\\n"
        
        # 关键发现
        report += f"\\n**关键发现:**\\n"
        if strong_correlations > 0:
            report += f"- 发现 {{strong_correlations}} 组强相关指标，可能存在共同的根因\\n"
        if avg_correlation > 0.7:
            report += f"- 整体相关性较强（{{avg_correlation:.3f}}），系统组件间耦合度高\\n"
        else:
            report += f"- 整体相关性适中（{{avg_correlation:.3f}}），问题可能相对独立\\n"

else:
    report += "未发现指标间显著相关性"

print(report)
"""
            
            result = self._execute_python_code(analysis_code)
            return result
            
        except Exception as e:
            logger.error(f"计算相关性洞察失败: {str(e)}")
            return f"计算失败: {str(e)}"
    
    @tool
    async def generate_prediction_analysis(self, prediction_data: str) -> str:
        """生成预测分析报告"""
        try:
            if isinstance(prediction_data, str):
                try:
                    data = json.loads(prediction_data)
                except json.JSONDecodeError:
                    return "无法解析预测数据"
            else:
                data = prediction_data
            
            analysis_code = f"""
import json
from datetime import datetime

# 解析预测数据
pred_data = {json.dumps(data)}

report = "**负载预测分析报告:**\\n\\n"

current_instances = pred_data.get('instances', 0)
current_qps = pred_data.get('current_qps', 0)
confidence = pred_data.get('confidence', 0)
features = pred_data.get('features', {{}})

report += f"**当前状态:**\\n"
report += f"- 建议实例数: {{current_instances}}\\n"
report += f"- 当前QPS: {{current_qps:.2f}}\\n"
report += f"- 预测置信度: {{confidence:.2f}}\\n"

if features:
    hour = features.get('hour', 0)
    is_business_hour = features.get('is_business_hour', False)
    is_weekend = features.get('is_weekend', False)
    
    report += f"\\n**时间特征分析:**\\n"
    report += f"- 当前时间: {{hour}}:00\\n"
    report += f"- 工作时间: {{'是' if is_business_hour else '否'}}\\n"
    report += f"- 周末: {{'是' if is_weekend else '否'}}\\n"
    
    # 基于时间特征的建议
    report += f"\\n**基于时间的分析:**\\n"
    if is_business_hour and not is_weekend:
        report += "- 当前为工作日工作时间，负载通常较高\\n"
        report += "- 建议保持较高的实例数以应对流量高峰\\n"
    elif is_weekend:
        report += "- 当前为周末，负载通常较低\\n"
        report += "- 可以适当降低实例数以节省成本\\n"
    elif hour < 6 or hour > 22:
        report += "- 当前为深夜时间，负载通常很低\\n"
        report += "- 建议使用最少的实例数\\n"
    else:
        report += "- 当前为非工作时间，负载适中\\n"
        report += "- 建议使用中等数量的实例\\n"

# 置信度分析
report += f"\\n**置信度分析:**\\n"
if confidence > 0.8:
    report += f"- 预测置信度很高（{{confidence:.2f}}），建议采纳预测结果\\n"
elif confidence > 0.6:
    report += f"- 预测置信度中等（{{confidence:.2f}}），建议结合人工判断\\n"
else:
    report += f"- 预测置信度较低（{{confidence:.2f}}），建议谨慎采纳\\n"

# QPS分析
report += f"\\n**QPS分析:**\\n"
if current_qps > 1000:
    report += f"- 当前QPS较高（{{current_qps:.2f}}），系统负载较重\\n"
elif current_qps > 100:
    report += f"- 当前QPS适中（{{current_qps:.2f}}），系统运行正常\\n"
else:
    report += f"- 当前QPS较低（{{current_qps:.2f}}），系统负载轻松\\n"

# 实例数建议
report += f"\\n**实例数建议:**\\n"
qps_per_instance = current_qps / max(current_instances, 1)
report += f"- 当前平均每实例QPS: {{qps_per_instance:.2f}}\\n"

if qps_per_instance > 100:
    report += "- 每实例负载较高，建议增加实例数\\n"
elif qps_per_instance < 10:
    report += "- 每实例负载较低，可以考虑减少实例数\\n"
else:
    report += "- 每实例负载适中，当前配置合理\\n"

print(report)
"""
            
            result = self._execute_python_code(analysis_code)
            return result
            
        except Exception as e:
            logger.error(f"生成预测分析失败: {str(e)}")
            return f"分析失败: {str(e)}"
    
    def _execute_python_code(self, code: str) -> str:
        """安全执行Python代码"""
        try:
            # 捕获输出
            old_stdout = sys.stdout
            sys.stdout = captured_output = io.StringIO()
            
            # 创建安全的执行环境
            safe_globals = {
                '__builtins__': {
                    'print': print,
                    'len': len,
                    'sum': sum,
                    'max': max,
                    'min': min,
                    'abs': abs,
                    'round': round,
                    'enumerate': enumerate,
                    'range': range,
                    'isinstance': isinstance,
                },
                'pd': pd,
                'np': np,
                'json': json,
                'datetime': __import__('datetime')
            }
            
            # 执行代码
            exec(code, safe_globals)
            
            # 恢复输出
            sys.stdout = old_stdout
            
            # 获取结果
            output = captured_output.getvalue()
            return output if output else "代码执行完成，但无输出"
            
        except Exception as e:
            # 恢复输出
            sys.stdout = old_stdout
            logger.error(f"Python代码执行失败: {str(e)}")
            return f"代码执行失败: {str(e)}"
    
    @tool
    async def create_data_visualization(self, data: str, chart_type: str = "summary") -> str:
        """创建数据可视化（文本版）"""
        try:
            # 由于无法生成真实图表，创建ASCII艺术风格的数据展示
            if isinstance(data, str):
                try:
                    parsed_data = json.loads(data)
                except json.JSONDecodeError:
                    return "无法解析数据进行可视化"
            else:
                parsed_data = data
            
            visualization_code = f"""
import json

data = {json.dumps(parsed_data)}

def create_ascii_chart(values, labels, title="数据图表"):
    chart = f"{{title}}\\n" + "="*len(title) + "\\n\\n"
    
    if not values:
        return chart + "无数据可显示"
    
    max_val = max(values)
    if max_val == 0:
        return chart + "所有值为0"
    
    # 创建水平条形图
    for i, (label, value) in enumerate(zip(labels, values)):
        bar_length = int((value / max_val) * 30)  # 最大30个字符
        bar = "█" * bar_length
        chart += f"{{label:15}} |{{bar:30}} {{value:.3f}}\\n"
    
    return chart

# 根据数据类型创建不同的可视化
if isinstance(data, dict) and any('score' in str(v) for v in data.values()):
    # 异常分数可视化
    labels = []
    scores = []
    
    for metric, info in data.items():
        if isinstance(info, dict) and 'max_score' in info:
            labels.append(metric[:15] + "..." if len(metric) > 15 else metric)
            scores.append(info['max_score'])
    
    chart = create_ascii_chart(scores, labels, "异常分数分布图")
    
elif isinstance(data, dict) and any('count' in str(v) for v in data.values()):
    # 异常次数可视化
    labels = []
    counts = []
    
    for metric, info in data.items():
        if isinstance(info, dict) and 'count' in info:
            labels.append(metric[:15] + "..." if len(metric) > 15 else metric)
            counts.append(info['count'])
    
    chart = create_ascii_chart(counts, labels, "异常次数分布图")
    
else:
    chart = "数据格式不支持可视化\\n可视化数据格式:\\n" + json.dumps(data, indent=2)[:200] + "..."

print(chart)
"""
            
            result = self._execute_python_code(visualization_code)
            return result
            
        except Exception as e:
            logger.error(f"创建数据可视化失败: {str(e)}")
            return f"可视化失败: {str(e)}"
    
    def get_available_tools(self) -> List[str]:
        """获取可用的编程工具"""
        return [
            "analyze_metrics_data",
            "calculate_correlation_insights", 
            "generate_prediction_analysis",
            "create_data_visualization",
            "python_code_execution"
        ]