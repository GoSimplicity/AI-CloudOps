#!/usr/bin/env python3
"""
AIOps 机器学习模型训练脚本
用于构建QPS-实例数预测模型
"""

import os
import json
import pandas as pd
import numpy as np
from sklearn.linear_model import Ridge
from sklearn.ensemble import RandomForestRegressor, GradientBoostingRegressor
from sklearn.preprocessing import StandardScaler
from sklearn.model_selection import train_test_split, GridSearchCV
from sklearn.metrics import mean_squared_error, r2_score, mean_absolute_error
import joblib
from datetime import datetime, timedelta
import matplotlib.pyplot as plt

# 创建数据目录
os.makedirs('data/models', exist_ok=True)

# 配置
MODEL_PATH = 'data/models/time_qps_auto_scaling_model.pkl'
SCALER_PATH = 'data/models/time_qps_auto_scaling_scaler.pkl'
METADATA_PATH = 'data/models/time_qps_auto_scaling_model_metadata.json'
CSV_PATH = 'data.csv'

def generate_synthetic_data():
    """生成更贴近真实场景的合成数据用于训练模型"""
    print("正在生成企业级合成训练数据...")
    
    # 创建时间序列
    start_date = datetime(2023, 1, 1)
    end_date = datetime(2023, 12, 31)
    dates = [start_date + timedelta(hours=i) for i in range(0, 24*365, 1)]
    
    # 创建DataFrame
    df = pd.DataFrame({
        'timestamp': dates
    })
    
    # 提取时间特征
    df['hour'] = df['timestamp'].dt.hour
    df['day_of_week'] = df['timestamp'].dt.dayofweek  # 0是周一，6是周日
    df['month'] = df['timestamp'].dt.month
    df['day'] = df['timestamp'].dt.day
    df['is_weekend'] = df['day_of_week'].isin([5, 6]).astype(int)
    df['is_business_hour'] = ((df['hour'] >= 9) & (df['hour'] <= 17) & 
                              (df['is_weekend'] == 0)).astype(int)
    df['is_holiday'] = 0  # 默认不是假期
    
    # 添加主要节假日 (简化版，实际系统应该使用完整的假期数据)
    holidays = [
        # 元旦
        (1, 1), (1, 2), (1, 3),
        # 春节 (2023年简化版)
        (1, 21), (1, 22), (1, 23), (1, 24), (1, 25), (1, 26), (1, 27),
        # 劳动节
        (5, 1), (5, 2), (5, 3),
        # 国庆节
        (10, 1), (10, 2), (10, 3), (10, 4), (10, 5), (10, 6), (10, 7)
    ]
    
    for month, day in holidays:
        holiday_mask = (df['month'] == month) & (df['day'] == day)
        df.loc[holiday_mask, 'is_holiday'] = 1
    
    # 创建周期性特征
    df['sin_time'] = np.sin(2 * np.pi * df['hour'] / 24)
    df['cos_time'] = np.cos(2 * np.pi * df['hour'] / 24)
    df['sin_day'] = np.sin(2 * np.pi * df['day_of_week'] / 7)
    df['cos_day'] = np.cos(2 * np.pi * df['day_of_week'] / 7)
    
    # 模拟真实企业的QPS模式
    # 1. 基础负载 - 总是存在的低水平流量
    base_load = 20 + np.random.normal(0, 5, size=len(df))
    
    # 2. 日间模式 - 工作时间的流量峰值
    # 早晨上升，中午小幅下降，下午再次上升，晚上下降
    hour_factors = {
        0: 0.1, 1: 0.05, 2: 0.05, 3: 0.05, 4: 0.05, 5: 0.1,
        6: 0.2, 7: 0.4, 8: 0.6, 9: 0.8, 10: 0.9, 11: 0.85,
        12: 0.7, 13: 0.8, 14: 0.9, 15: 0.95, 16: 1.0, 17: 0.9,
        18: 0.7, 19: 0.5, 20: 0.4, 21: 0.3, 22: 0.2, 23: 0.1
    }
    
    daily_pattern = df['hour'].map(hour_factors).values * 100
    
    # 3. 周间模式 - 工作日比周末流量高
    weekday_factors = {
        0: 1.0,  # 周一
        1: 1.1,  # 周二
        2: 1.2,  # 周三
        3: 1.15, # 周四
        4: 0.9,  # 周五
        5: 0.6,  # 周六
        6: 0.5   # 周日
    }
    
    weekly_pattern = df['day_of_week'].map(weekday_factors).values * 80
    
    # 4. 月度模式 - 某些月份可能有业务高峰
    monthly_factors = {
        1: 0.7,   # 元月
        2: 0.6,   # 二月
        3: 0.8,   # 三月
        4: 0.9,   # 四月
        5: 1.0,   # 五月
        6: 1.1,   # 六月
        7: 1.0,   # 七月
        8: 0.9,   # 八月
        9: 1.1,   # 九月
        10: 1.2,  # 十月
        11: 1.3,  # 十一月（双十一）
        12: 1.4   # 十二月（年终）
    }
    
    monthly_pattern = df['month'].map(monthly_factors).values * 30
    
    # 5. 特殊事件 - 模拟促销活动、发布等突发流量
    special_events = []
    # 添加10个随机特殊事件
    for _ in range(10):
        event_day = np.random.randint(0, 364)
        event_duration = np.random.randint(4, 24)  # 持续4-24小时
        event_intensity = np.random.uniform(1.5, 3.0)  # 流量放大1.5-3倍
        special_events.append((event_day, event_duration, event_intensity))
    
    # 初始化特殊事件影响
    special_event_impact = np.zeros(len(df))
    
    for event_day, duration, intensity in special_events:
        start_idx = event_day * 24
        for i in range(duration):
            if start_idx + i < len(special_event_impact):
                # 模拟事件期间流量逐渐上升再下降的模式
                if i < duration / 2:
                    factor = i / (duration / 2) * intensity
                else:
                    factor = (duration - i) / (duration / 2) * intensity
                special_event_impact[start_idx + i] = factor * 100
    
    # 6. 节假日影响 - 某些业务在节假日流量下降，有些则上升
    holiday_impact = df['is_holiday'] * np.random.choice([-50, 100], size=len(df), p=[0.3, 0.7])
    
    # 7. 随机波动 - 模拟不可预测的流量变化
    noise = np.random.normal(0, 15, size=len(df))
    
    # 8. 长期趋势 - 业务逐渐增长
    days_since_start = (df['timestamp'] - df['timestamp'].min()).dt.days
    trend = days_since_start / 365 * 50  # 一年增长50 QPS
    
    # 组合所有因素
    df['QPS'] = (base_load + daily_pattern + weekly_pattern + monthly_pattern + 
                special_event_impact + holiday_impact + noise + trend)
    df['QPS'] = df['QPS'].clip(lower=5)  # 确保QPS至少为5
    
    # 实例数模拟 - 基于现实的弹性伸缩策略
    # 基本规则:
    # 1. 基础实例数: 每100 QPS需要1个实例
    # 2. 最小实例数: 2个（保证高可用）
    # 3. 考虑时间因素（高峰期提前扩容）
    # 4. 添加业务规则和一些人为决策因素
    
    # 基本实例数计算
    df['base_instances'] = 2 + (df['QPS'] / 100).astype(int)
    
    # 高峰期提前扩容（早9点到下午5点额外增加实例）
    df['instances'] = df['base_instances'].copy()
    peak_hours_mask = (df['hour'] >= 8) & (df['hour'] <= 18) & (df['is_weekend'] == 0)
    df.loc[peak_hours_mask, 'instances'] = df.loc[peak_hours_mask, 'base_instances'] + 1
    
    # 特殊事件期间可能会额外增加实例
    for event_day, duration, intensity in special_events:
        start_idx = event_day * 24
        for i in range(duration):
            if start_idx + i < len(df):
                # 根据事件强度增加实例
                df.iloc[start_idx + i, df.columns.get_loc('instances')] += int(intensity)
    
    # 节假日调整
    df.loc[df['is_holiday'] == 1, 'instances'] = df.loc[df['is_holiday'] == 1, 'instances'].apply(
        lambda x: max(2, x - 1) if np.random.random() < 0.7 else x + 1
    )
    
    # 深夜时间减少实例（但保持至少2个实例）
    night_hours_mask = (df['hour'] >= 0) & (df['hour'] <= 5)
    df.loc[night_hours_mask, 'instances'] = df.loc[night_hours_mask, 'instances'].apply(
        lambda x: max(2, x - 1)
    )
    
    # 周末可能减少实例数（但仍保持最小实例数）
    weekend_mask = df['is_weekend'] == 1
    df.loc[weekend_mask, 'instances'] = df.loc[weekend_mask, 'instances'].apply(
        lambda x: max(2, int(x * 0.8)) if np.random.random() < 0.7 else x
    )
    
    # 确保实例数在合理范围内
    df['instances'] = df['instances'].clip(lower=1, upper=20)
    
    # 保存数据
    df.to_csv(CSV_PATH, index=False)
    print(f"生成了 {len(df)} 条训练数据并保存到 {CSV_PATH}")
    
    # 绘制数据可视化
    try:
        plt.figure(figsize=(15, 10))
        
        # 绘制一周的QPS曲线
        week_data = df[df['timestamp'] < df['timestamp'].min() + pd.Timedelta(days=7)]
        plt.subplot(2, 1, 1)
        plt.plot(week_data['timestamp'], week_data['QPS'], 'b-')
        plt.title('一周内的QPS变化')
        plt.xlabel('时间')
        plt.ylabel('QPS')
        plt.grid(True)
        
        # 绘制QPS和实例数的关系
        plt.subplot(2, 1, 2)
        plt.scatter(df['QPS'], df['instances'], alpha=0.5)
        plt.title('QPS与实例数关系')
        plt.xlabel('QPS')
        plt.ylabel('实例数')
        plt.grid(True)
        
        plt.tight_layout()
        plt.savefig('data/models/qps_instances_visualization.png')
        plt.close()
        print("已生成数据可视化图表保存到 data/models/qps_instances_visualization.png")
    except Exception as e:
        print(f"生成可视化图表失败: {str(e)}")
    
    return df

def load_or_generate_data():
    """加载数据或生成合成数据"""
    if os.path.exists(CSV_PATH):
        print(f"加载现有数据: {CSV_PATH}")
        df = pd.read_csv(CSV_PATH)
        if 'timestamp' in df.columns:
            df['timestamp'] = pd.to_datetime(df['timestamp'])
        return df
    else:
        return generate_synthetic_data()

def extract_features(df):
    """提取特征，使用更多的特征以提高预测准确性"""
    print("提取特征...")
    
    # 基础特征
    features = df[['QPS', 'sin_time', 'cos_time', 'sin_day', 'cos_day', 
                  'is_business_hour', 'is_weekend', 'is_holiday']].copy()
    
    # 添加一小时前的QPS作为特征（如果可用）
    df['QPS_1h_ago'] = df['QPS'].shift(1).fillna(df['QPS'])
    features['QPS_1h_ago'] = df['QPS_1h_ago']
    
    # 添加一天前同一时间的QPS作为特征
    df['QPS_1d_ago'] = df['QPS'].shift(24).fillna(df['QPS'])
    features['QPS_1d_ago'] = df['QPS_1d_ago']
    
    # 添加一周前同一时间的QPS作为特征
    df['QPS_1w_ago'] = df['QPS'].shift(24*7).fillna(df['QPS'])
    features['QPS_1w_ago'] = df['QPS_1w_ago']
    
    # 计算近期QPS变化率
    df['QPS_change'] = (df['QPS'] - df['QPS_1h_ago']) / (df['QPS_1h_ago'] + 1) # 避免除零
    features['QPS_change'] = df['QPS_change']
    
    # 计算最近6小时的平均QPS
    df['QPS_avg_6h'] = df['QPS'].rolling(6).mean().fillna(df['QPS'])
    features['QPS_avg_6h'] = df['QPS_avg_6h']
    
    # 目标变量
    target = df['instances'].copy()
    
    return features, target

def train_model():
    """训练模型，测试多种算法并选择最佳模型"""
    print("开始训练模型...")
    
    # 加载数据
    df = load_or_generate_data()
    
    # 提取特征
    features, target = extract_features(df)
    
    # 划分训练集和测试集
    X_train, X_test, y_train, y_test = train_test_split(
        features, target, test_size=0.2, random_state=42
    )
    
    # 特征标准化
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_test_scaled = scaler.transform(X_test)
    
    print("训练数据形状:", X_train.shape)
    print("特征列:", features.columns.tolist())
    
    # 选择算法并训练
    models = {
        'Ridge回归': Ridge(alpha=1.0),
        '随机森林': RandomForestRegressor(n_estimators=100, random_state=42),
        '梯度提升树': GradientBoostingRegressor(n_estimators=100, 
                                        learning_rate=0.1, 
                                        max_depth=5, 
                                        random_state=42)
    }
    
    best_model = None
    best_score = float('inf')
    best_model_name = ""
    results = {}
    
    for name, model in models.items():
        print(f"\n训练模型: {name}")
        model.fit(X_train_scaled, y_train)
        
        # 评估
        y_pred = model.predict(X_test_scaled)
        mse = mean_squared_error(y_test, y_pred)
        rmse = np.sqrt(mse)
        mae = mean_absolute_error(y_test, y_pred)
        r2 = r2_score(y_test, y_pred)
        
        # 计算精确匹配率(实例数是整数，我们需要检查舍入后的预测有多少是精确匹配的)
        y_pred_rounded = np.round(y_pred)
        exact_matches = np.sum(y_pred_rounded == y_test) / len(y_test)
        off_by_one = np.sum(np.abs(y_pred_rounded - y_test) <= 1) / len(y_test)
        
        results[name] = {
            'mse': mse,
            'rmse': rmse,
            'mae': mae,
            'r2': r2,
            'exact_match': exact_matches,
            'off_by_one': off_by_one
        }
        
        print(f"  均方误差(MSE): {mse:.4f}")
        print(f"  均方根误差(RMSE): {rmse:.4f}")
        print(f"  平均绝对误差(MAE): {mae:.4f}")
        print(f"  决定系数(R²): {r2:.4f}")
        print(f"  精确匹配率: {exact_matches:.2%}")
        print(f"  误差≤1的比例: {off_by_one:.2%}")
        
        # 特征重要性（如果模型支持）
        if hasattr(model, 'feature_importances_'):
            importances = model.feature_importances_
            indices = np.argsort(importances)[::-1]
            print("  特征重要性:")
            for i in range(min(10, len(features.columns))):
                idx = indices[i]
                print(f"    {features.columns[idx]}: {importances[idx]:.4f}")
        
        if mse < best_score:
            best_score = mse
            best_model = model
            best_model_name = name
    
    print(f"\n最佳模型: {best_model_name}")
    
    # 保存最佳模型
    joblib.dump(best_model, MODEL_PATH)
    print(f"模型已保存到 {MODEL_PATH}")
    
    # 保存标准化器
    joblib.dump(scaler, SCALER_PATH)
    print(f"标准化器已保存到 {SCALER_PATH}")
    
    # 保存模型元数据
    metadata = {
        "version": "2.0",
        "created_at": datetime.now().isoformat(),
        "algorithm": best_model_name,
        "features": features.columns.tolist(),
        "mse": float(best_score),
        "rmse": float(np.sqrt(best_score)),
        "mae": float(results[best_model_name]['mae']),
        "r2": float(results[best_model_name]['r2']),
        "exact_match": float(results[best_model_name]['exact_match']),
        "off_by_one": float(results[best_model_name]['off_by_one']),
        "samples": len(df),
        "description": "基于QPS和时间特征的企业级自动扩缩容预测模型"
    }
    
    with open(METADATA_PATH, 'w') as f:
        json.dump(metadata, f, indent=2)
    
    print(f"模型元数据已保存到 {METADATA_PATH}")
    
    # 绘制预测结果散点图
    try:
        plt.figure(figsize=(10, 6))
        plt.scatter(y_test, y_pred, alpha=0.5)
        plt.plot([y_test.min(), y_test.max()], [y_test.min(), y_test.max()], 'r--')
        plt.xlabel('实际实例数')
        plt.ylabel('预测实例数')
        plt.title(f'实例数预测结果 ({best_model_name})')
        plt.grid(True)
        plt.savefig('data/models/prediction_results.png')
        plt.close()
        print("已生成预测结果可视化图表保存到 data/models/prediction_results.png")
    except Exception as e:
        print(f"生成预测结果可视化失败: {str(e)}")
    
    return best_model, scaler, best_score

def test_model(model, scaler):
    """测试模型在各种场景下的表现"""
    print("\n测试模型...")
    
    # 创建测试样例，模拟各种企业场景
    test_cases = [
        {"QPS": 50, "hour": 10, "is_weekend": 0, "desc": "工作日上午，低负载"},
        {"QPS": 150, "hour": 13, "is_weekend": 0, "desc": "工作日中午，中等负载"},
        {"QPS": 300, "hour": 16, "is_weekend": 0, "desc": "工作日下午高峰，高负载"},
        {"QPS": 30, "hour": 2, "is_weekend": 0, "desc": "工作日深夜，极低负载"},
        {"QPS": 250, "hour": 20, "is_weekend": 0, "desc": "工作日晚上，高负载"},
        {"QPS": 180, "hour": 15, "is_weekend": 1, "desc": "周末下午，中等负载"},
        {"QPS": 400, "hour": 14, "is_weekend": 0, "is_holiday": 1, "desc": "节假日，超高负载"}
    ]
    
    for case in test_cases:
        qps = case["QPS"]
        hour = case["hour"]
        is_weekend = case.get("is_weekend", 0)
        is_holiday = case.get("is_holiday", 0)
        is_business_hour = 1 if 9 <= hour <= 17 and not is_weekend else 0
        
        # 计算周期性特征
        sin_time = np.sin(2 * np.pi * hour / 24)
        cos_time = np.cos(2 * np.pi * hour / 24)
        
        # 一天中的时间（假设是周三）
        day_of_week = 2 if not is_weekend else 6
        sin_day = np.sin(2 * np.pi * day_of_week / 7)
        cos_day = np.cos(2 * np.pi * day_of_week / 7)
        
        # 创建特征向量
        features_dict = {
            "QPS": [qps],
            "sin_time": [sin_time],
            "cos_time": [cos_time],
            "sin_day": [sin_day],
            "cos_day": [cos_day],
            "is_business_hour": [is_business_hour],
            "is_weekend": [is_weekend],
            "is_holiday": [is_holiday],
            "QPS_1h_ago": [qps * 0.9],  # 假设前一小时的QPS略低
            "QPS_1d_ago": [qps * 1.1],  # 假设昨天同一时间的QPS略高
            "QPS_1w_ago": [qps * 0.95],  # 假设上周同一时间的QPS略低
            "QPS_change": [0.1],  # 假设QPS在增长
            "QPS_avg_6h": [qps * 0.95]  # 假设6小时平均QPS略低
        }
        
        features = pd.DataFrame(features_dict)
        
        # 标准化特征
        features_scaled = scaler.transform(features)
        
        # 预测
        prediction = model.predict(features_scaled)[0]
        instances = int(np.clip(np.round(prediction), 1, 20))
        
        print(f"{case['desc']}: QPS={qps}, 预测实例数={instances}, 原始预测值={prediction:.2f}")

if __name__ == "__main__":
    print("=" * 50)
    print("企业级自动扩缩容预测模型训练")
    print("=" * 50)
    
    model, scaler, score = train_model()
    test_model(model, scaler)
    
    print("\n训练完成!")
    print("=" * 50)