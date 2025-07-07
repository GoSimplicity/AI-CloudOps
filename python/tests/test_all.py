#!/usr/bin/env python
"""
AIOps平台全量测试脚本
运行所有测试模块，包括:
1. 健康检查测试
2. 根因分析测试
3. 负载预测测试
4. 自动修复测试
5. 智能小助手测试
"""

import os
import sys
import pytest
import logging
from pathlib import Path

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("test_all")

def run_all_tests():
    """运行所有测试模块"""
    logger.info("开始运行所有测试...")
    
    # 获取测试目录
    test_dir = Path(__file__).parent
    
    # 定义测试模块顺序
    test_modules = [
        "test_rca.py",
        "test_prediction.py",
        "test_autofix.py",
        "test_assistant.py"
    ]
    
    # 收集结果
    results = {}
    
    # 运行测试
    for module in test_modules:
        module_path = test_dir / module
        if module_path.exists():
            logger.info(f"运行测试模块: {module}")
            result = pytest.main(["-xvs", str(module_path)])
            results[module] = "通过" if result == 0 else "失败"
        else:
            logger.warning(f"测试模块不存在: {module}")
            results[module] = "不存在"
    
    # 输出结果摘要
    logger.info("\n" + "=" * 50)
    logger.info("测试结果摘要")
    logger.info("=" * 50)
    
    all_passed = True
    for module, status in results.items():
        logger.info(f"{module}: {status}")
        if status != "通过":
            all_passed = False
    
    return all_passed

if __name__ == "__main__":
    try:
        success = run_all_tests()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        logger.info("测试被用户中断")
        sys.exit(130)
    except Exception as e:
        logger.error(f"测试过程出现未处理异常: {str(e)}")
        import traceback
        logger.error(traceback.format_exc())
        sys.exit(1)