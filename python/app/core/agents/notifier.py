import logging
from typing import Dict, Any, List, Optional
from langchain_core.tools import tool
from app.services.notification import NotificationService
from app.config.settings import config

logger = logging.getLogger("aiops.notifier")

class NotifierAgent:
    def __init__(self):
        self.notification_service = NotificationService()
        logger.info("Notifier Agent初始化完成")
    
    @tool
    async def send_human_help_request(self, problem_description: str, urgency: str = "medium") -> str:
        """发送人工帮助请求"""
        try:
            urgency_emoji = {
                "low": "🔵",
                "medium": "🟡", 
                "high": "🔴",
                "critical": "🚨"
            }.get(urgency.lower(), "🟡")
            
            message = f"""
{urgency_emoji} **需要人工协助处理问题**

**紧急程度:** {urgency.upper()}

**问题描述:**
{problem_description}

**建议操作:**
- 检查系统状态和日志
- 评估问题影响范围
- 制定应急处理方案
- 联系相关技术人员

**自动化处理状态:** 已尝试自动修复但需要人工介入

请及时处理此问题。
"""
            
            success = await self.notification_service.send_feishu_message(
                message, 
                f"人工协助请求 - {urgency.upper()}", 
                "red" if urgency in ["high", "critical"] else "orange"
            )
            
            if success:
                logger.info(f"成功发送人工帮助请求: {urgency}")
                return f"✅ 已发送{urgency}级别的人工帮助请求，相关人员将收到通知"
            else:
                logger.error("发送人工帮助请求失败")
                return "❌ 发送人工帮助请求失败，请检查通知配置"
                
        except Exception as e:
            logger.error(f"发送人工帮助请求异常: {str(e)}")
            return f"❌ 发送人工帮助请求异常: {str(e)}"
    
    @tool
    async def send_incident_alert(
        self, 
        incident_summary: str, 
        affected_services: List[str], 
        severity: str = "medium"
    ) -> str:
        """发送事件告警"""
        try:
            severity_config = {
                "low": {"emoji": "🟢", "color": "green"},
                "medium": {"emoji": "🟡", "color": "orange"},
                "high": {"emoji": "🔴", "color": "red"},
                "critical": {"emoji": "🚨", "color": "red"}
            }
            
            config_info = severity_config.get(severity.lower(), severity_config["medium"])
            
            services_list = "\n".join([f"- {service}" for service in affected_services])
            
            message = f"""
{config_info['emoji']} **系统事件告警**

**严重程度:** {severity.upper()}

**事件摘要:**
{incident_summary}

**受影响的服务:**
{services_list}

**处理状态:** 自动化系统正在处理

**建议操作:**
- 监控事件处理进展
- 准备应急处理方案
- 检查相关系统状态
"""
            
            success = await self.notification_service.send_feishu_message(
                message,
                f"系统事件告警 - {severity.upper()}",
                config_info['color']
            )
            
            if success:
                logger.info(f"成功发送事件告警: {severity}")
                return f"✅ 已发送{severity}级别的事件告警通知"
            else:
                logger.error("发送事件告警失败")
                return "❌ 发送事件告警失败，请检查通知配置"
                
        except Exception as e:
            logger.error(f"发送事件告警异常: {str(e)}")
            return f"❌ 发送事件告警异常: {str(e)}"
    
    @tool
    async def send_resolution_notification(
        self, 
        problem_description: str, 
        solution_summary: str, 
        actions_taken: List[str]
    ) -> str:
        """发送问题解决通知"""
        try:
            actions_list = "\n".join([f"- {action}" for action in actions_taken])
            
            message = f"""
✅ **问题解决通知**

**原始问题:**
{problem_description}

**解决方案:**
{solution_summary}

**执行的操作:**
{actions_list}

**处理结果:** 问题已通过自动化修复解决

**后续建议:**
- 监控系统稳定性
- 检查修复效果
- 更新运维文档
"""
            
            success = await self.notification_service.send_feishu_message(
                message,
                "问题解决通知",
                "green"
            )
            
            if success:
                logger.info("成功发送问题解决通知")
                return "✅ 已发送问题解决通知"
            else:
                logger.error("发送问题解决通知失败")
                return "❌ 发送问题解决通知失败"
                
        except Exception as e:
            logger.error(f"发送问题解决通知异常: {str(e)}")
            return f"❌ 发送问题解决通知异常: {str(e)}"
    
    @tool
    async def send_system_health_report(self, health_data: Dict[str, Any]) -> str:
        """发送系统健康报告"""
        try:
            healthy_components = [k for k, v in health_data.get('components', {}).items() if v]
            unhealthy_components = [k for k, v in health_data.get('components', {}).items() if not v]
            
            overall_status = "健康" if not unhealthy_components else "异常"
            status_emoji = "✅" if not unhealthy_components else "⚠️"
            
            message = f"""
{status_emoji} **系统健康状态报告**

**整体状态:** {overall_status}
**检查时间:** {health_data.get('timestamp', 'N/A')}
**系统版本:** {health_data.get('version', 'N/A')}
"""
            
            if unhealthy_components:
                message += f"""
**异常组件:**
{chr(10).join([f"- ❌ {comp}" for comp in unhealthy_components])}
"""
            
            if healthy_components:
                message += f"""
**正常组件:**
{chr(10).join([f"- ✅ {comp}" for comp in healthy_components])}
"""
            
            if health_data.get('uptime'):
                message += f"\n**系统运行时间:** {health_data['uptime']:.1f} 秒"
            
            color = "green" if overall_status == "健康" else "orange"
            
            success = await self.notification_service.send_feishu_message(
                message,
                "系统健康状态报告",
                color
            )
            
            if success:
                logger.info("成功发送系统健康报告")
                return "✅ 已发送系统健康状态报告"
            else:
                logger.error("发送系统健康报告失败")
                return "❌ 发送系统健康报告失败"
                
        except Exception as e:
            logger.error(f"发送系统健康报告异常: {str(e)}")
            return f"❌ 发送系统健康报告异常: {str(e)}"
    
    @tool
    async def send_maintenance_notification(
        self, 
        maintenance_type: str, 
        scheduled_time: str, 
        estimated_duration: str,
        affected_services: List[str]
    ) -> str:
        """发送维护通知"""
        try:
            services_list = "\n".join([f"- {service}" for service in affected_services])
            
            message = f"""
🔧 **系统维护通知**

**维护类型:** {maintenance_type}
**计划时间:** {scheduled_time}
**预计持续时间:** {estimated_duration}

**受影响的服务:**
{services_list}

**注意事项:**
- 维护期间可能出现服务中断
- 请提前做好业务准备
- 如有紧急情况请联系运维团队

**联系方式:** 运维团队值班电话
"""
            
            success = await self.notification_service.send_feishu_message(
                message,
                "系统维护通知",
                "blue"
            )
            
            if success:
                logger.info("成功发送维护通知")
                return "✅ 已发送系统维护通知"
            else:
                logger.error("发送维护通知失败")
                return "❌ 发送维护通知失败"
                
        except Exception as e:
            logger.error(f"发送维护通知异常: {str(e)}")
            return f"❌ 发送维护通知异常: {str(e)}"
    
    async def check_notification_health(self) -> Dict[str, Any]:
        """检查通知服务健康状态"""
        try:
            is_healthy = self.notification_service.is_healthy()
            
            health_info = {
                "healthy": is_healthy,
                "enabled": self.notification_service.enabled,
                "webhook_configured": bool(self.notification_service.feishu_webhook),
                "service_type": "feishu"
            }
            
            return health_info
            
        except Exception as e:
            logger.error(f"检查通知服务健康状态失败: {str(e)}")
            return {
                "healthy": False,
                "error": str(e)
            }
    
    def get_available_tools(self) -> List[str]:
        """获取可用的通知工具"""
        tools = [
            "send_human_help_request",
            "send_incident_alert",
            "send_resolution_notification",
            "send_system_health_report",
            "send_maintenance_notification"
        ]
        
        if self.notification_service.enabled:
            tools.append("notifications_enabled")
        else:
            tools.append("notifications_disabled")
        
        return tools