import logging
import requests
import json
from typing import Dict, Any, Optional, List
from datetime import datetime
from app.config.settings import config

logger = logging.getLogger("aiops.notification")

class NotificationService:
    def __init__(self):
        self.feishu_webhook = config.notification.feishu_webhook
        self.enabled = config.notification.enabled
        logger.info(f"通知服务初始化完成, 启用状态: {self.enabled}")
    
    async def send_feishu_message(
        self, 
        message: str, 
        title: str = "AIOps通知",
        color: str = "blue"
    ) -> bool:
        """发送飞书消息"""
        if not self.enabled or not self.feishu_webhook:
            logger.warning("通知服务未启用或未配置Webhook")
            return False
        
        try:
            headers = {"Content-Type": "application/json"}
            
            # 构建卡片消息
            card_data = {
                "msg_type": "interactive",
                "card": {
                    "config": {
                        "wide_screen_mode": True
                    },
                    "elements": [
                        {
                            "tag": "div",
                            "text": {
                                "content": message,
                                "tag": "lark_md"
                            }
                        },
                        {
                            "tag": "hr"
                        },
                        {
                            "tag": "div",
                            "text": {
                                "content": f"**发送时间：** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
                                "tag": "lark_md"
                            }
                        }
                    ],
                    "header": {
                        "title": {
                            "content": title,
                            "tag": "plain_text"
                        },
                        "template": color
                    }
                }
            }
            
            logger.debug(f"发送飞书消息: {title}")
            response = requests.post(
                self.feishu_webhook,
                headers=headers,
                data=json.dumps(card_data),
                timeout=10
            )
            
            if response.status_code == 200:
                response_data = response.json()
                if response_data.get("code") == 0:
                    logger.info("飞书消息发送成功")
                    return True
                else:
                    logger.error(f"飞书消息发送失败: {response_data}")
                    return False
            else:
                logger.error(f"飞书消息发送失败，状态码：{response.status_code}")
                return False
                
        except Exception as e:
            logger.error(f"发送飞书消息失败：{str(e)}")
            return False
    
    async def send_rca_alert(
        self, 
        root_causes: List[Dict[str, Any]],
        time_range: Dict[str, str],
        metrics_count: int
    ) -> bool:
        """发送根因分析告警"""
        try:
            if not root_causes:
                return True  # 没有根因不需要发送
            
            message = f"""
🚨 **根因分析告警**

**分析时间范围：**
- 开始时间: {time_range.get('start', 'N/A')}
- 结束时间: {time_range.get('end', 'N/A')}
- 分析指标数: {metrics_count}

**发现的根因：**
"""
            
            for i, cause in enumerate(root_causes[:3], 1):
                confidence = cause.get('confidence', 0)
                confidence_emoji = "🔴" if confidence > 0.8 else "🟡" if confidence > 0.5 else "🟢"
                
                message += f"""
{i}. {confidence_emoji} **{cause.get('metric', 'Unknown')}**
   - 置信度: {confidence:.2f}
   - 异常次数: {cause.get('anomaly_count', 0)}
   - 首次发现: {cause.get('first_occurrence', 'N/A')}
"""
                
                if cause.get('description'):
                    message += f"   - 描述: {cause['description']}\n"
            
            message += f"""
**建议操作：**
- 检查相关服务状态
- 查看详细监控数据
- 考虑扩容或重启服务

[查看详细分析结果](#)
"""
            
            return await self.send_feishu_message(message, "根因分析告警", "red")
            
        except Exception as e:
            logger.error(f"发送根因分析告警失败: {str(e)}")
            return False
    
    async def send_autofix_notification(
        self, 
        deployment: str, 
        namespace: str, 
        status: str, 
        actions: List[str],
        error_message: Optional[str] = None
    ) -> bool:
        """发送自动修复通知"""
        try:
            success = status == "success"
            status_emoji = "✅" if success else "❌"
            color = "green" if success else "red"
            
            message = f"""
{status_emoji} **自动修复通知**

**部署信息：**
- Deployment: `{deployment}`
- Namespace: `{namespace}`
- 修复状态: {status}

**执行的操作：**
"""
            
            for action in actions:
                message += f"- {action}\n"
            
            if error_message:
                message += f"""
**错误信息：**
{error_message}

"""
            
            if success:
                message += "\n**结果：** 自动修复成功完成 🎉"
            else:
                message += "\n**结果：** 自动修复失败，需要人工介入 ⚠️"
            
            return await self.send_feishu_message(message, "自动修复通知", color)
            
        except Exception as e:
            logger.error(f"发送自动修复通知失败: {str(e)}")
            return False
    
    async def send_prediction_alert(
        self, 
        current_instances: int, 
        predicted_instances: int, 
        current_qps: float,
        confidence: float
    ) -> bool:
        """发送负载预测告警"""
        try:
            if abs(predicted_instances - current_instances) <= 1:
                return True  # 变化不大，不需要告警
            
            trend = "增加" if predicted_instances > current_instances else "减少"
            trend_emoji = "📈" if predicted_instances > current_instances else "📉"
            
            confidence_level = "高" if confidence > 0.8 else "中" if confidence > 0.6 else "低"
            
            message = f"""
{trend_emoji} **负载预测告警**

**当前状态：**
- 当前实例数: {current_instances}
- 当前QPS: {current_qps:.2f}

**预测结果：**
- 建议实例数: {predicted_instances}
- 变化趋势: {trend}
- 预测置信度: {confidence:.2f} ({confidence_level})

**建议操作：**
- 检查当前负载情况
- 考虑手动调整实例数
- 监控后续变化趋势
"""
            
            color = "orange" if abs(predicted_instances - current_instances) > 3 else "blue"
            
            return await self.send_feishu_message(message, "负载预测告警", color)
            
        except Exception as e:
            logger.error(f"发送负载预测告警失败: {str(e)}")
            return False
    
    async def send_system_health_alert(
        self, 
        unhealthy_components: List[str],
        healthy_components: List[str]
    ) -> bool:
        """发送系统健康告警"""
        try:
            if not unhealthy_components:
                return True  # 系统健康，不需要告警
            
            message = f"""
🚨 **系统健康告警**

**异常组件：**
"""
            for component in unhealthy_components:
                message += f"- ❌ {component}\n"
            
            if healthy_components:
                message += f"""
**正常组件：**
"""
                for component in healthy_components:
                    message += f"- ✅ {component}\n"
            
            message += """
**建议操作：**
- 检查异常组件状态
- 查看相关日志
- 联系相关负责人
"""
            
            return await self.send_feishu_message(message, "系统健康告警", "red")
            
        except Exception as e:
            logger.error(f"发送系统健康告警失败: {str(e)}")
            return False
    
    def is_healthy(self) -> bool:
        """检查通知服务健康状态"""
        if not self.enabled:
            return True  # 服务未启用视为健康
        
        return bool(self.feishu_webhook)