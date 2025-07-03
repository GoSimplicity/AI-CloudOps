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
MODEL_PATH = 'models/time_qps_auto_scaling_model.pkl'
SCALER_PATH = 'models/time_qps_auto_scaling_scaler.pkl'
METADATA_PATH = 'models/time_qps_auto_scaling_model_metadata.json'
CSV_PATH = 'data.csv'

def load_real_data():
    """加载真实数据"""
    print("正在加载真实数据...")
    
    try:
        # 尝试加载CSV文件
        if os.path.exists(CSV_PATH):
            df = pd.read_csv(CSV_PATH)
            
            # 确保时间戳列存在
            if 'timestamp' not in df.columns:
                print(f"错误: 数据集缺少timestamp列")
                return None
                
            # 确保QPS和实例数列存在
            if 'QPS' not in df.columns or 'instances' not in df.columns:
                print(f"错误: 数据集缺少QPS或instances列")
                return None
                
            # 将时间戳转换为datetime格式
            if not pd.api.types.is_datetime64_any_dtype(df['timestamp']):
                try:
                    df['timestamp'] = pd.to_datetime(df['timestamp'], errors='coerce')
                except:
                    print("警告: 无法解析时间戳字段，尝试使用默认格式...")
            
            # 检查数据是否有空值
            if df['QPS'].isnull().any() or df['instances'].isnull().any():
                print("警告: 数据包含空值，正在清理...")
                df = df.dropna(subset=['QPS', 'instances'])
            
            # 基本数据验证
            if len(df) < 100:
                print(f"警告: 数据量较少 ({len(df)} 条)，可能导致模型性能不佳")
            
            # 检查并处理异常值
            qps_mean = df['QPS'].mean()
            qps_std = df['QPS'].std()
            outlier_mask = (df['QPS'] > qps_mean + 5 * qps_std) | (df['QPS'] < 0)
            if outlier_mask.any():
                print(f"警告: 检测到 {outlier_mask.sum()} 个QPS异常值，将被限制在合理范围内")
                df.loc[df['QPS'] < 0, 'QPS'] = 0
                df.loc[df['QPS'] > qps_mean + 5 * qps_std, 'QPS'] = qps_mean + 5 * qps_std
            
            # 确保实例数是正整数
            if (df['instances'] < 1).any():
                print("警告: 发现实例数小于1，将设置为最小值1")
                df.loc[df['instances'] < 1, 'instances'] = 1
                
            # 将实例数转为整数
            df['instances'] = df['instances'].round().astype(int)
            
            print(f"成功加载了 {len(df)} 条数据")
            return df
        else:
            print(f"错误: 数据文件 {CSV_PATH} 不存在")
            return None
    except Exception as e:
        print(f"加载数据时出错: {str(e)}")
        return None

def extract_features(df):
    """从数据集提取训练特征"""
    if df is None or len(df) == 0:
        print("错误: 无法从空数据集提取特征")
        return None, None
    
    try:
        print("正在提取特征...")
        
        # 提取时间特征
        df['hour'] = df['timestamp'].dt.hour
        df['day_of_week'] = df['timestamp'].dt.dayofweek  # 0是周一，6是周日
        df['month'] = df['timestamp'].dt.month
        df['day'] = df['timestamp'].dt.day
        df['is_weekend'] = df['day_of_week'].isin([5, 6]).astype(int)
        df['is_business_hour'] = ((df['hour'] >= 9) & (df['hour'] <= 17) & 
                                (df['is_weekend'] == 0)).astype(int)
        
        # 创建周期性特征
        df['sin_time'] = np.sin(2 * np.pi * df['hour'] / 24)
        df['cos_time'] = np.cos(2 * np.pi * df['hour'] / 24)
        df['sin_day'] = np.sin(2 * np.pi * df['day_of_week'] / 7)
        df['cos_day'] = np.cos(2 * np.pi * df['day_of_week'] / 7)
        
        # 为每个时间点添加历史QPS数据
        df = df.sort_values('timestamp')
        
        # 添加滞后特征（前一小时，前一天，前一周）
        # 注意: 实际使用时，确保数据已按时间排序
        df['QPS_1h_ago'] = df['QPS'].shift(1)  # 假设数据点间隔为1小时
        df['QPS_1d_ago'] = df['QPS'].shift(24)  # 24小时前
        df['QPS_1w_ago'] = df['QPS'].shift(24*7)  # 一周前
        
        # 计算变化率
        df['QPS_change'] = df['QPS'].pct_change().fillna(0)
        
        # 计算移动平均
        df['QPS_avg_6h'] = df['QPS'].rolling(window=6).mean().fillna(df['QPS'])
        
        # 删除包含NaN的行
        df = df.dropna()
        
        # 选择特征和目标变量
        features = df[[
            'QPS', 'sin_time', 'cos_time', 'sin_day', 'cos_day', 
            'is_business_hour', 'is_weekend', 'QPS_1h_ago', 
            'QPS_1d_ago', 'QPS_1w_ago', 'QPS_change', 'QPS_avg_6h'
        ]]
        
        target = df['instances']
        
        print(f"提取了 {len(features)} 条训练数据，包含 {len(features.columns)} 个特征")
        return features, target
    except Exception as e:
        print(f"提取特征时出错: {str(e)}")
        return None, None

def train_model():
    """训练和评估预测模型"""
    print("开始训练模型...")
    
    # 加载真实数据
    df = load_real_data()
    if df is None:
        print("错误: 无法加载数据，模型训练终止")
        return False
    
    # 提取特征
    features, target = extract_features(df)
    if features is None or target is None:
        print("错误: 特征提取失败，模型训练终止")
        return False
    
    # 划分训练集和测试集
    X_train, X_test, y_train, y_test = train_test_split(
        features, target, test_size=0.2, random_state=42
    )
    
    print(f"训练集: {X_train.shape[0]} 样本, 测试集: {X_test.shape[0]} 样本")
    
    # 标准化特征
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_test_scaled = scaler.transform(X_test)
    
    # 定义模型列表
    models = {
        "Ridge回归": Ridge(),
        "随机森林回归": RandomForestRegressor(random_state=42),
        "梯度提升回归": GradientBoostingRegressor(random_state=42)
    }
    
    # 定义参数网格
    param_grids = {
        "Ridge回归": {
            'alpha': [0.1, 1.0, 10.0]
        },
        "随机森林回归": {
            'n_estimators': [50, 100],
            'max_depth': [None, 10, 20],
            'min_samples_split': [2, 5]
        },
        "梯度提升回归": {
            'n_estimators': [50, 100],
            'learning_rate': [0.01, 0.1],
            'max_depth': [3, 5]
        }
    }
    
    best_model = None
    best_score = float('-inf')
    best_name = None
    
    # 模型训练与评估
    for name, model in models.items():
        print(f"\n训练模型: {name}")
        
        # 网格搜索最佳参数
        grid_search = GridSearchCV(
            model, param_grids[name], cv=5, 
            scoring='neg_mean_squared_error', n_jobs=-1
        )
        grid_search.fit(X_train_scaled, y_train)
        
        # 获取最佳模型
        best_params = grid_search.best_params_
        model = grid_search.best_estimator_
        print(f"最佳参数: {best_params}")
        
        # 在测试集上评估
        y_pred = model.predict(X_test_scaled)
        
        # 计算性能指标
        mse = mean_squared_error(y_test, y_pred)
        rmse = np.sqrt(mse)
        mae = mean_absolute_error(y_test, y_pred)
        r2 = r2_score(y_test, y_pred)
        
        print(f"性能指标:")
        print(f"  MSE: {mse:.4f}")
        print(f"  RMSE: {rmse:.4f}")
        print(f"  MAE: {mae:.4f}")
        print(f"  R²: {r2:.4f}")
        
        # 更新最佳模型
        if r2 > best_score:
            best_model = model
            best_score = r2
            best_name = name
    
    if best_model is not None:
        print(f"\n选择最佳模型: {best_name} (R² = {best_score:.4f})")
        
        # 保存模型和标准化器
        joblib.dump(best_model, MODEL_PATH)
        joblib.dump(scaler, SCALER_PATH)
        print(f"模型已保存到 {MODEL_PATH}")
        print(f"标准化器已保存到 {SCALER_PATH}")
        
        # 保存模型元数据
        model_metadata = {
            "version": "2.0",
            "created_at": datetime.now().isoformat(),
            "features": list(features.columns),
            "target": "instances",
            "algorithm": best_name,
            "performance": {
                "r2": best_score,
                "rmse": rmse,
                "mae": mae
            },
            "parameters": str(best_model.get_params()),
            "data_stats": {
                "n_samples": len(df),
                "mean_qps": float(df['QPS'].mean()),
                "std_qps": float(df['QPS'].std()),
                "min_qps": float(df['QPS'].min()),
                "max_qps": float(df['QPS'].max()),
                "mean_instances": float(df['instances'].mean()),
                "min_instances": int(df['instances'].min()),
                "max_instances": int(df['instances'].max())
            }
        }
        
        with open(METADATA_PATH, 'w') as f:
            json.dump(model_metadata, f, indent=2)
        print(f"模型元数据已保存到 {METADATA_PATH}")
        
        # 可视化实际值与预测值的对比
        y_pred_train = best_model.predict(X_train_scaled)
        y_pred_test = best_model.predict(X_test_scaled)
        
        plt.figure(figsize=(15, 10))
        
        # 训练集上的实际值与预测值对比
        plt.subplot(2, 2, 1)
        plt.scatter(y_train, y_pred_train, alpha=0.5)
        plt.plot([y_train.min(), y_train.max()], [y_train.min(), y_train.max()], 'r--')
        plt.xlabel('实际实例数')
        plt.ylabel('预测实例数')
        plt.title('训练集: 实际值 vs 预测值')
        
        # 测试集上的实际值与预测值对比
        plt.subplot(2, 2, 2)
        plt.scatter(y_test, y_pred_test, alpha=0.5)
        plt.plot([y_test.min(), y_test.max()], [y_test.min(), y_test.max()], 'r--')
        plt.xlabel('实际实例数')
        plt.ylabel('预测实例数')
        plt.title('测试集: 实际值 vs 预测值')
        
        # QPS与实例数的关系
        plt.subplot(2, 2, 3)
        plt.scatter(df['QPS'], df['instances'], alpha=0.5)
        plt.xlabel('QPS')
        plt.ylabel('实例数')
        plt.title('QPS与实例数关系')
        plt.grid(True)
        
        # 误差分布
        plt.subplot(2, 2, 4)
        errors = y_test - y_pred_test
        plt.hist(errors, bins=20)
        plt.xlabel('预测误差')
        plt.ylabel('频率')
        plt.title('预测误差分布')
        
        plt.tight_layout()
        plt.savefig('data/models/prediction_results.png')
        print("模型评估结果已保存为图像")
        
        # 额外创建QPS和实例数的可视化
        plt.figure(figsize=(12, 6))
        
        # 选择一部分时间段的数据进行可视化
        time_series_df = df.sort_values('timestamp').reset_index(drop=True)
        sample_size = min(1000, len(time_series_df))
        sample_df = time_series_df.iloc[:sample_size]
        
        plt.plot(sample_df.index, sample_df['QPS'], 'b-', label='QPS')
        plt.plot(sample_df.index, sample_df['instances'] * 10, 'r-', label='实例数 x 10')
        plt.xlabel('时间索引')
        plt.ylabel('值')
        plt.title('QPS与实例数随时间的变化')
        plt.legend()
        plt.grid(True)
        
        plt.savefig('data/models/qps_instances_visualization.png')
        print("QPS与实例数可视化已保存")
        
        return True
    else:
        print("错误: 未能找到合适的模型")
        return False

def test_model(model, scaler):
    """测试模型在不同场景下的表现"""
    if model is None or scaler is None:
        print("错误: 模型或标准化器未提供")
        return
    
    print("\n模型场景测试:")
    
    # 测试场景
    test_cases = [
        {"name": "低QPS场景", "qps": 5.0, "hour": 12, "day_of_week": 2, "is_weekend": 0},
        {"name": "中等QPS场景", "qps": 50.0, "hour": 14, "day_of_week": 3, "is_weekend": 0},
        {"name": "高QPS场景", "qps": 500.0, "hour": 10, "day_of_week": 4, "is_weekend": 0},
        {"name": "工作时间峰值", "qps": 300.0, "hour": 11, "day_of_week": 1, "is_weekend": 0},
        {"name": "夜间低流量", "qps": 20.0, "hour": 2, "day_of_week": 2, "is_weekend": 0},
        {"name": "周末场景", "qps": 100.0, "hour": 15, "day_of_week": 6, "is_weekend": 1},
        {"name": "零流量场景", "qps": 0.0, "hour": 3, "day_of_week": 2, "is_weekend": 0}
    ]
    
    for case in test_cases:
        test_prediction(model, scaler, case)
    
def test_prediction(model, scaler, case):
    """测试单个预测场景"""
    # 创建特征字典
    features_dict = {
        "QPS": [case["qps"]],
        "sin_time": [np.sin(2 * np.pi * case["hour"] / 24)],
        "cos_time": [np.cos(2 * np.pi * case["hour"] / 24)],
        "sin_day": [np.sin(2 * np.pi * case["day_of_week"] / 7)],
        "cos_day": [np.cos(2 * np.pi * case["day_of_week"] / 7)],
        "is_business_hour": [1 if 9 <= case["hour"] <= 17 and case["is_weekend"] == 0 else 0],
        "is_weekend": [case["is_weekend"]]
    }
    
    # 添加历史QPS特征（模拟值）
    features_dict["QPS_1h_ago"] = [case["qps"] * 0.9]
    features_dict["QPS_1d_ago"] = [case["qps"] * 1.1]
    features_dict["QPS_1w_ago"] = [case["qps"] * 1.0]
    features_dict["QPS_change"] = [0.1]
    features_dict["QPS_avg_6h"] = [case["qps"] * 0.95]
    
    # 构建特征向量
    features_df = pd.DataFrame(features_dict)
    
    # 标准化特征
    try:
        features_scaled = scaler.transform(features_df)
    except:
        # 如果特征列不匹配，可能需要调整
        print(f"警告: 特征不匹配，尝试调整...")
        # 获取标准化器的特征列表
        scaler_features = getattr(scaler, "feature_names_in_", None)
        if scaler_features is None:
            print("错误: 无法确定标准化器的特征列")
            return
            
        # 根据标准化器要求的特征调整
        adjusted_features = {}
        for i, feature in enumerate(scaler_features):
            if feature in features_dict:
                adjusted_features[feature] = features_dict[feature]
            else:
                print(f"警告: 缺少特征 '{feature}'，使用0.0代替")
                adjusted_features[feature] = [0.0]
        
        features_df = pd.DataFrame(adjusted_features)
        features_scaled = scaler.transform(features_df)
    
    # 执行预测
    try:
        prediction = model.predict(features_scaled)[0]
        
        # 限制实例数范围并四舍五入（实例数应为整数）
        instances = max(1, int(round(prediction)))
        
        print(f"{case['name']}: QPS={case['qps']:.1f}, 预测实例数={instances}")
    except Exception as e:
        print(f"预测失败: {str(e)}")
    

if __name__ == '__main__':
    # 训练模型
    success = train_model()
    
    if success:
        # 加载已训练的模型和标准化器
        try:
            model = joblib.load(MODEL_PATH)
            scaler = joblib.load(SCALER_PATH)
            
            # 测试模型
            test_model(model, scaler)
        except Exception as e:
            print(f"加载模型失败: {str(e)}")
    else:
        print("模型训练失败，无法执行测试")