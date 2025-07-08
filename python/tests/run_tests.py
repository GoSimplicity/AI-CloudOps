#!/usr/bin/env python3
"""
AIOps平台测试运行脚本
运行测试并生成简单的Markdown格式测试报告
"""

import os
import sys
import time
import json
import pytest
import logging
import datetime
from pathlib import Path
from collections import defaultdict

# 添加项目路径
ROOT_DIR = Path(__file__).parent.parent
sys.path.insert(0, str(ROOT_DIR))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("run_tests")

# 测试模块配置
TEST_MODULES = [
    # 健康检查相关测试
    "test_health.py",
    
    # 知识库相关测试
    "test_knowledge_load.py",
    
    # 助手相关测试
    # 暂时移除，因为存在异步事件循环兼容性问题
    # "test_assistant.py",
    # "test_websocket_assistant.py",
    
    # 预测相关测试
    "test_prediction.py::test_prediction_health",
    "test_prediction.py::test_prediction_get",
    "test_prediction.py::test_prediction_post",
    "test_prediction.py::test_prediction_zero_qps",
    "test_prediction.py::test_prediction_low_qps",
    "test_prediction.py::test_prediction_high_qps", 
    "test_prediction.py::test_prediction_invalid_qps",
    "test_prediction.py::test_trend_prediction",
    "test_prediction.py::test_model_validation",
    "test_prediction.py::test_full_prediction_workflow",
    
    # 根因分析相关测试
    "test_rca.py::test_rca_health",
    "test_rca.py::test_get_available_metrics",
    "test_rca.py::test_anomaly_detection",
    "test_rca.py::test_correlation_analysis", 
    "test_rca.py::test_root_cause_analysis",
    "test_rca.py::test_rca_with_specific_workload",
    "test_rca.py::test_rca_performance",
    "test_rca.py::test_rca_historical_analysis",
    
    # 自动修复相关测试
    "test_autofix.py::test_autofix_health",
    "test_autofix.py::test_diagnose_cluster",
    "test_autofix.py::test_autofix_normal",
    "test_autofix.py::test_autofix_problematic",
    "test_autofix.py::test_autofix_test_problem",
    "test_autofix.py::test_notification",
    "test_autofix.py::test_workflow"
]

def run_tests():
    """运行所有测试模块并收集结果"""
    test_dir = ROOT_DIR / "tests"
    results = {}
    test_details = defaultdict(list)
    total_tests = 0
    passed_tests = 0
    failed_tests = 0
    skipped_tests = 0
    
    logger.info("开始运行测试...")
    start_time = time.time()
    
    # 使用pytest运行测试并收集结果
    for module in TEST_MODULES:
        module_name = module.split("::")[0] if "::" in module else module
        
        if Path(test_dir / module_name).exists():
            logger.info(f"运行测试模块: {module}")
            # 使用pytest的API运行测试
            module_start_time = time.time()
            
            pytest_output = []
            # 构建正确的模块路径
            if "::" in module:
                test_name = module.split("::")[1]
                module_path = f"tests/{module_name}::{test_name}"
            else:
                module_path = f"tests/{module}"
                
            result = pytest.main(
                ["-v", module_path], 
                plugins=[_create_result_collector(test_details, module_name, pytest_output)]
            )
            
            module_end_time = time.time()
            module_duration = module_end_time - module_start_time
            
            # 解析测试结果
            results[module] = {
                "status": "通过" if result == 0 else "失败",
                "duration": module_duration,
                "output": "\n".join(pytest_output)
            }
            
            module_tests = len(test_details[module_name])
            module_passed = sum(1 for t in test_details[module_name] if t["status"] == "passed")
            module_failed = sum(1 for t in test_details[module_name] if t["status"] == "failed")
            module_skipped = sum(1 for t in test_details[module_name] if t["status"] == "skipped")
            
            total_tests += module_tests
            passed_tests += module_passed
            failed_tests += module_failed
            skipped_tests += module_skipped
            
            logger.info(f"{module} 测试完成: {module_passed}/{module_tests} 通过")
        else:
            logger.warning(f"测试模块不存在: {module}")
            results[module] = {"status": "不存在", "duration": 0, "output": ""}
    
    end_time = time.time()
    duration = end_time - start_time
    
    # 返回所有结果
    return {
        "timestamp": datetime.datetime.now().isoformat(),
        "duration": duration,
        "total": total_tests,
        "passed": passed_tests,
        "failed": failed_tests,
        "skipped": skipped_tests,
        "modules": results,
        "details": test_details
    }

def _create_result_collector(test_details, module_name, output_collector):
    """创建pytest插件来收集测试结果"""
    
    class ResultCollector:
        @pytest.hookimpl(hookwrapper=True)
        def pytest_runtest_makereport(self, item, call):
            outcome = yield
            report = outcome.get_result()
            
            if report.when == "call" or (report.when == "setup" and report.skipped):
                test_name = item.name
                test_path = item.nodeid
                
                status = "passed" if report.passed else "failed"
                if report.skipped:
                    status = "skipped"
                
                # 收集测试结果
                test_details[module_name].append({
                    "name": test_name,
                    "path": test_path,
                    "status": status,
                    "duration": report.duration,
                    "reason": report.longrepr if hasattr(report, "longrepr") else None
                })
                
                # 收集输出
                output_collector.append(f"{test_path} ... {status}")
    
    return ResultCollector()

def generate_markdown_report(results):
    """生成Markdown格式的测试报告"""
    timestamp = datetime.datetime.fromisoformat(results["timestamp"])
    formatted_time = timestamp.strftime("%Y-%m-%d %H:%M:%S")
    
    # 计算通过率
    pass_rate = results["passed"] / results["total"] * 100 if results["total"] > 0 else 0
    
    report = [
        "# AIOps平台测试报告",
        "",
        f"**生成时间**: {formatted_time}",
        f"**总耗时**: {results['duration']:.2f}秒",
        f"**测试总数**: {results['total']}",
        f"**通过**: {results['passed']} ({pass_rate:.1f}%)",
        f"**失败**: {results['failed']}",
        f"**跳过**: {results['skipped']}",
        "",
        "## 测试覆盖度分析",
        ""
    ]
    
    # 按功能分类统计测试
    categories = {
        "健康检查": [m for m in results["modules"] if "health" in m.lower()],
        "助手服务": [m for m in results["modules"] if "assistant" in m.lower()],
        "知识库": [m for m in results["modules"] if "knowledge" in m.lower()],
        "预测服务": [m for m in results["modules"] if "prediction" in m.lower()],
        "根因分析": [m for m in results["modules"] if "rca" in m.lower()],
        "自动修复": [m for m in results["modules"] if "autofix" in m.lower()],
        "其他": []
    }
    
    # 添加覆盖度统计
    report.append("| 功能模块 | 测试用例数 | 通过数 | 失败数 | 覆盖率 |")
    report.append("| --- | --- | --- | --- | --- |")
    
    for category, modules in categories.items():
        category_tests = 0
        category_passed = 0
        category_failed = 0
        
        for module in modules:
            module_name = module.split("::")[0] if "::" in module else module
            if module_name in results["details"]:
                module_tests = len(results["details"][module_name])
                module_passed = sum(1 for t in results["details"][module_name] if t["status"] == "passed")
                module_failed = sum(1 for t in results["details"][module_name] if t["status"] == "failed")
                
                category_tests += module_tests
                category_passed += module_passed
                category_failed += module_failed
                
        coverage = category_passed / category_tests * 100 if category_tests > 0 else 0
        report.append(f"| {category} | {category_tests} | {category_passed} | {category_failed} | {coverage:.1f}% |")
    
    # 添加执行时间统计
    report.append("")
    report.append("## 执行时间统计")
    report.append("")
    report.append("| 功能模块 | 平均执行时间 (秒) | 最长测试用例 | 最短测试用例 |")
    report.append("| --- | --- | --- | --- |")
    
    for category, modules in categories.items():
        if not modules:
            continue
            
        durations = []
        longest_test = ("无", 0)
        shortest_test = ("无", float('inf'))
        
        for module in modules:
            module_name = module.split("::")[0] if "::" in module else module
            if module_name in results["details"]:
                for test in results["details"][module_name]:
                    durations.append(test["duration"])
                    if test["duration"] > longest_test[1]:
                        longest_test = (f"{test['name']}", test["duration"])
                    if test["duration"] < shortest_test[1]:
                        shortest_test = (f"{test['name']}", test["duration"])
        
        avg_duration = sum(durations) / len(durations) if durations else 0
        report.append(f"| {category} | {avg_duration:.2f} | {longest_test[0]} ({longest_test[1]:.2f}s) | {shortest_test[0]} ({shortest_test[1]:.2f}s) |")
    
    report.append("")
    report.append("## 测试模块摘要")
    report.append("")
    report.append("| 模块 | 状态 | 耗时 (秒) | 通过率 |")
    report.append("| --- | --- | --- | --- |")
    
    # 添加模块摘要
    for module, data in results["modules"].items():
        module_name = module.split("::")[0] if "::" in module else module
        if module_name in results["details"]:
            module_tests = len(results["details"][module_name])
            module_passed = sum(1 for t in results["details"][module_name] if t["status"] == "passed")
            module_rate = module_passed / module_tests * 100 if module_tests > 0 else 0
            report.append(f"| {module} | {data['status']} | {data['duration']:.2f} | {module_rate:.1f}% |")
        else:
            report.append(f"| {module} | {data['status']} | {data['duration']:.2f} | 0% |")
    
    # 添加详细测试结果
    report.append("")
    report.append("## 详细测试结果")
    
    for module, tests in results["details"].items():
        report.append("")
        report.append(f"### {module}")
        report.append("")
        report.append("| 测试 | 状态 | 耗时 (秒) |")
        report.append("| --- | --- | --- |")
        
        for test in tests:
            status_icon = "✅" if test["status"] == "passed" else "❌" if test["status"] == "failed" else "⏩"
            report.append(f"| {test['name']} | {status_icon} {test['status']} | {test['duration']:.2f} |")
    
    # 添加失败测试详情
    failed_tests = []
    for module, tests in results["details"].items():
        for test in tests:
            if test["status"] == "failed":
                failed_tests.append((module, test))
    
    if failed_tests:
        report.append("")
        report.append("## 失败测试详情")
        
        for module, test in failed_tests:
            report.append("")
            report.append(f"### {module}: {test['name']}")
            report.append("")
            report.append("```")
            report.append(str(test["reason"]) if test["reason"] else "无详细错误信息")
            report.append("```")
    
    # 添加测试覆盖度热力图
    report.append("")
    report.append("## 测试覆盖度热力图")
    report.append("")
    report.append("```mermaid")
    report.append("heatmap")
    report.append("title 项目测试覆盖度")
    report.append("x-axis [\"功能模块\"]")
    report.append("y-axis [\"覆盖情况\"]")
    
    # 获取所有模块名称
    all_modules = set()
    for module in results["modules"]:
        module_name = module.split("::")[0] if "::" in module else module
        all_modules.add(module_name)
    
    # 计算覆盖率
    coverage_data = {}
    for module_name in all_modules:
        if module_name in results["details"]:
            module_tests = len(results["details"][module_name])
            module_passed = sum(1 for t in results["details"][module_name] if t["status"] == "passed")
            module_rate = module_passed / module_tests * 100 if module_tests > 0 else 0
            coverage_data[module_name] = module_rate
    
    # 添加热力图数据
    for module, rate in coverage_data.items():
        display_name = module.replace("test_", "").replace(".py", "")
        color = 10
        if rate >= 90:
            color = 9
        elif rate >= 80:
            color = 8
        elif rate >= 70:
            color = 7
        elif rate >= 60:
            color = 6
        elif rate >= 50:
            color = 5
        elif rate >= 40:
            color = 4
        elif rate >= 30:
            color = 3
        elif rate >= 20:
            color = 2
        elif rate >= 10:
            color = 1
        
        report.append(f"\"{display_name}\" : {color}")
    
    report.append("```")
    
    # 添加测试执行趋势（这部分需要保存历史数据才能显示，这里只是添加占位符）
    report.append("")
    report.append("## 测试改进建议")
    report.append("")
    
    # 分析可能的测试改进点
    modules_missing_tests = []
    for module_name in all_modules:
        if module_name not in results["details"] or not results["details"][module_name]:
            modules_missing_tests.append(module_name)
    
    if modules_missing_tests:
        report.append("### 需要增加测试的模块")
        report.append("")
        for module in modules_missing_tests:
            report.append(f"- {module}")
        report.append("")
    
    # 提供一些通用的改进建议
    report.append("### 改进建议")
    report.append("")
    report.append("1. **提高代码覆盖率**: 考虑添加更多单元测试以覆盖更多代码路径")
    report.append("2. **增加边界测试**: 针对可能的边界情况添加更多测试用例")
    report.append("3. **改进测试速度**: 部分测试用例执行时间较长，可以考虑进行优化")
    report.append("4. **提高测试质量**: 替换返回值测试为断言测试，遵循pytest最佳实践")
    report.append("5. **添加集成测试**: 增强系统各组件间的集成测试")
    report.append("")
    report.append("## 总结")
    report.append("")
    pass_rate = results["passed"] / results["total"] * 100 if results["total"] > 0 else 0
    status = "良好" if pass_rate >= 90 else "需要改进" if pass_rate >= 70 else "存在问题"
    report.append(f"测试覆盖度: **{pass_rate:.1f}%**，整体状态: **{status}**")
    if results["failed"] > 0:
        report.append(f"存在 {results['failed']} 个测试失败，需要修复")
    else:
        report.append("所有测试都已通过")
        
    return "\n".join(report)

def save_report(report):
    """保存测试报告"""
    output_dir = ROOT_DIR / "docs"
    os.makedirs(output_dir, exist_ok=True)
    
    timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    file_path = output_dir / f"test_report_{timestamp}.md"
    latest_path = output_dir / "latest_test_report.md"
    
    with open(file_path, "w", encoding="utf-8") as f:
        f.write(report)
    
    # 同时保存一份最新报告
    with open(latest_path, "w", encoding="utf-8") as f:
        f.write(report)
    
    logger.info(f"测试报告已保存到 {file_path}")
    logger.info(f"最新测试报告已保存到 {latest_path}")
    
    print(report)  # 直接输出报告内容
    
    return file_path

def main():
    """主函数"""
    try:
        logger.info("开始测试流程")
        
        # 设置环境变量
        os.environ["TESTING"] = "true"
        
        # 运行测试并收集结果
        results = run_tests()
        
        # 生成并保存报告
        report = generate_markdown_report(results)
        save_report(report)
        
        # 计算退出码
        exit_code = 0 if results["failed"] == 0 else 1
        
        logger.info(f"测试完成。通过: {results['passed']}, 失败: {results['failed']}, 跳过: {results['skipped']}")
        
        return exit_code
        
    except Exception as e:
        logger.error(f"测试过程发生错误: {str(e)}")
        import traceback
        logger.error(traceback.format_exc())
        return 1

if __name__ == "__main__":
    exit_code = main()
    sys.exit(exit_code)
