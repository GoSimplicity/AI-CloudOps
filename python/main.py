import datetime
import os
import logging
from flask import Flask, jsonify, request
import numpy as np
import pandas as pd
import joblib
import requests
import getpass
import time
import yaml
import json
from typing import Literal, Sequence, Annotated
import operator
from typing_extensions import TypedDict
import functools
import traceback
from flask_cors import CORS

from langchain_community.tools.tavily_search import TavilySearchResults
from langchain_experimental.tools import PythonAstREPLTool
from langchain_core.messages import HumanMessage, BaseMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_openai import ChatOpenAI
from pydantic import BaseModel
from langchain_core.tools import tool
from openai import OpenAI
from kubernetes import client, config, watch
from langgraph.graph import END, StateGraph, START
from langgraph.prebuilt import create_react_agent

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
# 添加CORS支持，允许所有来源的请求
CORS(app, resources={r"/*": {"origins": "*"}}, supports_credentials=True)

# 配置类
class Config:
    # Prometheus配置
    PROMETHEUS_HOST = os.getenv(
        "PROMETHEUS_HOST",
        "127.0.0.1:9090"
    )
    PROMETHEUS_QUERY = 'rate(nginx_ingress_controller_nginx_process_requests_total{service="ingress-nginx-controller-metrics"}[10m])'
    
    # 模型配置
    MODEL_PATH = "time_qps_auto_scaling_model.pkl"
    SCALER_PATH = "time_qps_auto_scaling_scaler.pkl"
    
    # LLM配置
    LLM_MODEL = os.getenv("LLM_MODEL", "qwen2.5:3b")
    LLM_API_KEY = os.getenv("LLM_API_KEY", "ollama")
    LLM_BASE_URL = os.getenv("LLM_BASE_URL", "http://127.0.0.1:11434/v1")
    
    # 飞书Webhook
    FEISHU_WEBHOOK = os.getenv("FEISHU_WEBHOOK", "https://open.feishu.cn/open-apis/bot/v2/hook/d219b128-1520-40b3-b7ce-8df8c01422d2")
    
    # 实例限制
    MAX_INSTANCES = 20
    MIN_INSTANCES = 1

# 初始化配置
config_obj = Config()

# 加载模型和标准化器
try:
    model = joblib.load(config_obj.MODEL_PATH)
    scaler = joblib.load(config_obj.SCALER_PATH)
    logger.info("成功加载模型和标准化器")
except Exception as e:
    logger.error(f"加载模型失败: {str(e)}")
    model = None
    scaler = None

# # 设置环境变量
# def set_if_undefined(var: str):
#     if not os.environ.get(var):
#         os.environ[var] = getpass.getpass(f"请提供您的 {var}:")

# # 设置必要的环境变量
# set_if_undefined("TAVILY_API_KEY")

# 从Prometheus获取QPS
def get_qps_from_prometheus():
    try:
        url = f"http://{config_obj.PROMETHEUS_HOST}/api/v1/query"
        response = requests.get(url, params={"query": config_obj.PROMETHEUS_QUERY}, timeout=10)
        response.raise_for_status()
        
        data = response.json()
        if data["status"] != "success" or not data["data"]["result"]:
            logger.warning("Prometheus查询无结果")
            return 0.0
            
        qps = float(data["data"]["result"][0]["value"][1])
        logger.info(f"当前QPS: {qps}")
        return qps
        
    except Exception as e:
        logger.error(f"获取QPS失败: {str(e)}")
        return 0.0

# 预测接口
@app.route("/predict", methods=["GET"])
def predict():
    try:
        if model is None or scaler is None:
            return jsonify({"error": "模型未加载"}), 500
            
        # 获取当前QPS
        qps = get_qps_from_prometheus()
        
        # 获取当前时间
        current_time = datetime.datetime.now()
        minutes = current_time.hour * 60 + current_time.minute
        
        # 计算时间特征
        sin_time = np.sin(2 * np.pi * minutes / 1440)
        cos_time = np.cos(2 * np.pi * minutes / 1440)
        
        # 构建特征向量
        features = pd.DataFrame({
            "QPS": [qps],
            "sin_time": [sin_time],
            "cos_time": [cos_time]
        })
        
        features_scaled = scaler.transform(features)
        
        # 预测
        prediction = model.predict(features_scaled)[0]
        logger.info(f"预测实例数: {prediction}")
        
        # 限制实例数范围
        instances = int(np.clip(prediction, config_obj.MIN_INSTANCES, config_obj.MAX_INSTANCES))
        
        return jsonify({
            "instances": instances,
            "current_qps": qps,
            "timestamp": current_time.isoformat()
        })
        
    except Exception as e:
        logger.error(f"预测失败: {str(e)}")
        return jsonify({"error": str(e)}), 500

# 多Agent自动故障修复相关代码
class AgentState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], operator.add]
    next: str

# 定义消息节点
def agent_node(state, agent, name):
    result = agent.invoke(state)
    if isinstance(result, dict) and "messages" in result:
        return {"messages": [HumanMessage(content=result["messages"][-1].content, name=name)]}
    else:
        return {"messages": [HumanMessage(content=str(result), name=name)]}

# 定义成员Agent
members = ["Researcher", "Coder", "AutoFixK8s", "HumanHelp"]
system_prompt = (
    "你是一个主管，负责管理以下工作人员之间的对话："
    "{members}。根据以下用户请求，"
    "回复下一个应该行动的工作人员。每个工作人员将执行一项"
    "任务并回复他们的结果和状态。完成后，"
    "回复 FINISH。"
    "补充信息：Researcher负责网络搜索；Coder负责代码执行；AutoFixK8s负责修复Kubernetes问题；HumanHelp寻求人工服务"
)

options = ["FINISH"] + members

# 定义supervisor的响应类
class RouteResponse(BaseModel):
    next: Literal[*options] # type: ignore

# 创建提示语模板
prompt = ChatPromptTemplate.from_messages(
    [
        ("system", system_prompt),
        MessagesPlaceholder(variable_name="messages"),
        (
            "system",
            "根据上面的对话，下一步谁应该行动？"
            "或者我们应该结束(FINISH)？从以下选项中选择一个：{options}"
        ),
    ]
).partial(options=str(options), members=", ".join(members))

# 初始化LLM
llm = ChatOpenAI(
    model=config_obj.LLM_MODEL,
    api_key=config_obj.LLM_API_KEY,
    base_url=config_obj.LLM_BASE_URL
)

# supervisor agent函数
def supervisor_agent(state):
    supervisor_chain = prompt | llm.with_structured_output(RouteResponse)
    result = supervisor_chain.invoke(state)
    return {"next": result.next}

# 加载K8s配置
def load_k8s_config():
    try:
        # 先尝试集群内配置
        config.load_incluster_config()
        logger.info("使用集群内K8s配置")
    except:
        try:
            # 失败则尝试本地配置
            config.load_kube_config()
            logger.info("使用本地K8s配置")
        except Exception as e:
            logger.error(f"无法加载K8s配置: {str(e)}")
            raise

# 定义K8s自动修复工具
@tool
def auto_fix_k8s(deployment_name: str, namespace: str, event: str) -> str:
    """自动修复K8s问题
    
    Args:
        deployment_name: Deployment名称
        namespace: 命名空间
        event: 错误事件描述
    """
    try:
        load_k8s_config()
        k8s_apps_v1 = client.AppsV1Api()
        
        # 获取Deployment
        deployment = k8s_apps_v1.read_namespaced_deployment(
            name=deployment_name,
            namespace=namespace
        )
        
        deployment_dict = deployment.to_dict()
        # 清理不需要的字段
        deployment_dict.pop("status", None)
        if "metadata" in deployment_dict:
            deployment_dict["metadata"].pop("managed_fields", None)
            deployment_dict["metadata"].pop("resource_version", None)
            deployment_dict["metadata"].pop("uid", None)
            deployment_dict["metadata"].pop("self_link", None)
        
        deployment_yaml = yaml.dump(deployment_dict)
        
        # 使用OpenAI生成修复方案
        openai_client = OpenAI(
            api_key=config_obj.LLM_API_KEY,
            base_url=config_obj.LLM_BASE_URL
        )
        
        # 第一次生成patch
        response = openai_client.chat.completions.create(
            model=config_obj.LLM_MODEL,
            response_format={"type": "json_object"},
            messages=[
                {
                    "role": "system",
                    "content": "你是一个Kubernetes专家，只输出JSON格式的patch内容"
                },
                {
                    "role": "user",
                    "content": f"""根据K8s错误信息生成kubectl patch所需的JSON：
错误信息：{event}
当前Deployment YAML：
{deployment_yaml}

请直接返回可用于kubectl patch的JSON内容，不要包含其他说明。"""
                }
            ]
        )
        
        # 验证并优化patch
        response_validation = openai_client.chat.completions.create(
            model=config_obj.LLM_MODEL,
            response_format={"type": "json_object"},
            messages=[
                {
                    "role": "system",
                    "content": "你是一个Kubernetes专家，验证并优化patch JSON"
                },
                {
                    "role": "user",
                    "content": f"""验证以下Kubernetes patch JSON是否正确，如有问题请修正：
Patch JSON：{response.choices[0].message.content}
原始Deployment YAML：
{deployment_yaml}

直接返回优化后的JSON。"""
                }
            ]
        )
        
        patch_json = json.loads(response_validation.choices[0].message.content)
        logger.info(f"生成的patch: {json.dumps(patch_json, indent=2)}")
        
        # 应用patch
        k8s_apps_v1.patch_namespaced_deployment(
            name=deployment_name,
            namespace=namespace,
            body=patch_json
        )
        
        return f"成功修复Deployment {deployment_name}，已应用patch"
        
    except Exception as e:
        error_msg = f"修复失败：{str(e)}\n{traceback.format_exc()}"
        logger.error(error_msg)
        return error_msg

# 定义人工帮助工具
@tool
def human_help(event_message: str) -> str:
    """发送飞书通知寻求人工帮助
    
    Args:
        event_message: 需要人工处理的事件消息
    """
    try:
        headers = {"Content-Type": "application/json"}
        data = {
            "msg_type": "text",
            "content": {
                "text": f"K8s自动修复失败，需要人工介入：\n{event_message}"
            }
        }
        
        response = requests.post(
            config_obj.FEISHU_WEBHOOK,
            headers=headers,
            data=json.dumps(data),
            timeout=10
        )
        
        if response.status_code == 200:
            return "已发送飞书通知，等待人工处理"
        else:
            return f"发送通知失败，状态码：{response.status_code}"
            
    except Exception as e:
        return f"发送通知失败：{str(e)}"

# 初始化工具
try:
    tavily_tool = TavilySearchResults(max_results=5)
except:
    logger.warning("TavilySearchResults初始化失败")
    tavily_tool = None

python_repl_tool = PythonAstREPLTool()

# 创建Agent
def create_agents():
    agents = {}
    
    if tavily_tool:
        research_agent = create_react_agent(llm, tools=[tavily_tool])
        agents["research_node"] = functools.partial(agent_node, agent=research_agent, name="Researcher")
    
    code_agent = create_react_agent(llm, tools=[python_repl_tool])
    agents["code_node"] = functools.partial(agent_node, agent=code_agent, name="Coder")
    
    auto_fix_agent = create_react_agent(llm, tools=[auto_fix_k8s])
    agents["auto_fix_node"] = functools.partial(agent_node, agent=auto_fix_agent, name="AutoFixK8s")
    
    human_help_agent = create_react_agent(llm, tools=[human_help])
    agents["human_help_node"] = functools.partial(agent_node, agent=human_help_agent, name="HumanHelp")
    
    return agents

# 创建Graph
def create_workflow():
    agents = create_agents()
    workflow = StateGraph(AgentState)
    
    # 添加supervisor节点
    workflow.add_node("supervisor", supervisor_agent)
    
    # 添加agent节点
    if "research_node" in agents:
        workflow.add_node("Researcher", agents["research_node"])
        workflow.add_edge("Researcher", "supervisor")
    
    workflow.add_node("Coder", agents["code_node"])
    workflow.add_node("AutoFixK8s", agents["auto_fix_node"])
    workflow.add_node("HumanHelp", agents["human_help_node"])
    
    # 添加边
    for member in ["Coder", "AutoFixK8s", "HumanHelp"]:
        workflow.add_edge(member, "supervisor")
    
    # 条件路由
    available_members = ["Coder", "AutoFixK8s", "HumanHelp"]
    if "research_node" in agents:
        available_members.append("Researcher")
    
    conditional_map = {k: k for k in available_members}
    conditional_map["FINISH"] = END
    workflow.add_conditional_edges("supervisor", lambda x: x["next"], conditional_map)
    
    workflow.add_edge(START, "supervisor")
    
    return workflow.compile()

# 编译Graph
try:
    graph = create_workflow()
    logger.info("成功创建工作流")
except Exception as e:
    logger.error(f"创建工作流失败: {str(e)}")
    graph = None

# 自动修复K8s问题的API接口
@app.route("/autofix", methods=["POST"])
def autofix_k8s():
    try:
        if graph is None:
            return jsonify({"error": "工作流未初始化", "status": "失败"}), 500
            
        data = request.json
        deployment_name = data.get("deployment")
        namespace = data.get("namespace", "default")
        event = data.get("event")
        
        if not deployment_name or not event:
            return jsonify({
                "error": "缺少必要参数：deployment和event",
                "status": "失败"
            }), 400
        
        logger.info(f"开始处理自动修复请求：deployment={deployment_name}, namespace={namespace}")
        
        # 使用多Agent系统处理问题
        initial_message = f"请修复Kubernetes问题：部署名称={deployment_name}，命名空间={namespace}，错误事件={event}"
        
        result = graph.invoke(
            {
                "messages": [HumanMessage(content=initial_message)]
            },
            config={"recursion_limit": 10}
        )
        
        # 提取结果
        messages = result.get("messages", [])
        if messages:
            final_message = messages[-1].content
        else:
            final_message = "处理完成但没有返回消息"
        
        return jsonify({
            "result": final_message,
            "status": "成功",
            "deployment": deployment_name,
            "namespace": namespace
        })
        
    except Exception as e:
        error_msg = f"处理失败：{str(e)}"
        logger.error(f"{error_msg}\n{traceback.format_exc()}")
        return jsonify({
            "error": error_msg,
            "status": "失败"
        }), 500

# 健康检查接口
@app.route("/health", methods=["GET"])
def health():
    health_status = {
        "status": "healthy",
        "model_loaded": model is not None,
        "scaler_loaded": scaler is not None,
        "workflow_loaded": graph is not None,
        "timestamp": datetime.datetime.now().isoformat()
    }
    return jsonify(health_status)

# 错误处理器
@app.errorhandler(Exception)
def handle_exception(e):
    logger.error(f"未处理的异常: {str(e)}\n{traceback.format_exc()}")
    return jsonify({
        "error": "内部服务器错误",
        "message": str(e)
    }), 500

# 添加CORS预检请求处理
@app.after_request
def after_request(response):
    response.headers.add('Access-Control-Allow-Headers', 'Content-Type,Authorization,X-Requested-With')
    response.headers.add('Access-Control-Allow-Methods', 'GET,PUT,POST,DELETE,OPTIONS')
    response.headers.add('Access-Control-Allow-Credentials', 'true')
    return response

# 启动服务
if __name__ == "__main__":
    logger.info("启动Flask应用")
    app.run(host="0.0.0.0", port=8080, debug=False)