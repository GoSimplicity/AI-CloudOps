"""
MIT License

Copyright (c) 2024 Bamboo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

description: FastAPI应用
"""

from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
import time
import uvicorn
import os
import sys

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from utils.logger import get_logger
from utils.config import get_config
from utils.metrics import APIMetrics

# 导入路由
from api.routes.anomaly_routes import router as anomaly_router
from api.routes.prediction_routes import router as prediction_router
from api.routes.assistant_routes import router as assistant_router

logger = get_logger("api")

# 创建FastAPI应用
app = FastAPI(
    title=get_config("service.name", "AIOps"),
    description="AIOps平台API",
    version=get_config("service.version", "0.1.0"),
    docs_url="/docs",
    redoc_url="/redoc"
)

# CORS配置
app.add_middleware(
    CORSMiddleware,
    allow_origins=get_config("api.cors_origins", ["*"]),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 请求计时中间件
@app.middleware("http")
async def add_process_time_header(request: Request, call_next):
    start_time = time.time()
    response = await call_next(request)
    process_time = time.time() - start_time
    response.headers["X-Process-Time"] = str(process_time)
    
    # 记录API指标
    APIMetrics.record_api_call(
        endpoint=request.url.path,
        status_code=response.status_code,
        duration=process_time
    )
    
    return response

# 健康检查路由
@app.get("/health")
async def health_check():
    return {"status": "healthy"}

# 指标路由
@app.get("/metrics")
async def metrics():
    return {
        "api_metrics": APIMetrics.get_api_metrics()
    }

# 挂载路由
app.include_router(anomaly_router, prefix="/api/anomaly", tags=["anomaly"])
app.include_router(prediction_router, prefix="/api/prediction", tags=["prediction"])
app.include_router(assistant_router, prefix="/api/assistant", tags=["assistant"])

def start_server():
    """启动API服务器"""
    host = get_config("service.host", "0.0.0.0")
    port = get_config("service.port", 8000)
    debug = get_config("service.debug", False)
    
    logger.info(f"Starting AIOps API server at http://{host}:{port}")
    uvicorn.run("api.fastapi_app:app", host=host, port=port, reload=debug)

if __name__ == "__main__":
    start_server()