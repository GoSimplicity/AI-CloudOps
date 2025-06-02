## 数据处理与模型训练流程

### 1. 核心库安装与功能说明

#### 1.1 Pandas 数据处理库

```bash
pip install pandas
```

**主要功能**：

- 数据清洗：处理缺失值、异常值
- 数据转换：类型转换、特征工程
- 数据聚合：分组统计、数据透视
- 数据筛选：条件过滤、列选择

**常用操作**：
| 方法 | 功能说明 | 示例 |
|------|----------|------|
| `read_csv()` | 从 CSV 文件读取数据 | `df = pd.read_csv("data.csv")` |
| `dropna()` | 删除包含缺失值的行 | `df_clean = df.dropna()` |
| `groupby()` | 数据分组聚合操作 | `df_group = df.groupby("category").mean()` |
| `merge()` | 合并多个数据集 | `merged_df = pd.merge(df1, df2, on="key")` |

---

#### 1.2 Scikit-learn 机器学习库

```bash
pip install scikit-learn
```

**核心功能**：

- **数据预处理**：
  - `StandardScaler`：数据标准化（均值为 0，方差为 1）
  - `LabelEncoder`：分类标签编码
- **模型训练**：
  - `LinearRegression`：线性回归模型
  - `RandomForestClassifier`：随机森林分类器
- **模型评估**：
  - `cross_val_score`：交叉验证评估
  - `accuracy_score`：分类准确率评估
  - `mean_squared_error`：回归均方误差评估

---

### 2. 数据加载与预处理

#### 2.1 数据载入

```python
import pandas as pd

# 从CSV文件加载数据
df = pd.read_csv("metrics_data.csv")

# 数据概览
print(f"数据集形状: {df.shape}")
print(f"列名: {df.columns.tolist()}")
print(df.head(3))  # 显示前3行
```

**数据操作示例**：

```python
# 读取特定列
qps_column = df["QPS"]

# 读取第二行数据
second_row = df.iloc[1]

# 读取第二行的QPS值
second_row_qps = df.loc[1, "QPS"]

# 筛选QPS > 10的行
high_qps_df = df[df["QPS"] > 10]
```

---

#### 2.2 时间特征工程

**时间格式转换**：

```python
# 将时间戳转换为分钟数（从午夜开始计算）
df["timestamp"] = pd.to_datetime(df["timestamp"], format="%H:%M:%S")
df["minutes"] = df["timestamp"].dt.hour * 60 + df["timestamp"].dt.minute
```

**周期性时间特征**：

```python
import numpy as np

# 创建周期性时间特征（正弦/余弦转换）
df["sin_time"] = np.sin(2 * np.pi * df["minutes"] / 1440)  # 1440 = 24*60
df["cos_time"] = np.cos(2 * np.pi * df["minutes"] / 1440)

# 原理：将线性时间转换为周期性特征，帮助模型理解时间的循环特性
```

---

### 3. 模型训练与评估

#### 3.1 特征与目标变量定义

```python
# 特征矩阵 (QPS + 时间特征)
X = df[["QPS", "sin_time", "cos_time"]]

# 目标变量 (需要预测的实例数)
y = df["instances"]
```

#### 3.2 数据集分割

```python
from sklearn.model_selection import train_test_split

# 划分训练集(80%)和测试集(20%)
X_train, X_test, y_train, y_test = train_test_split(
    X, y,
    test_size=0.2,
    random_state=42  # 固定随机种子确保可复现性
)
```

#### 3.3 数据标准化

```python
from sklearn.preprocessing import StandardScaler

# 初始化标准化器
scaler = StandardScaler()

# 训练集拟合并转换
X_train_scaled = scaler.fit_transform(X_train)

# 测试集转换（使用训练集的参数）
X_test_scaled = scaler.transform(X_test)

# 注意：测试集必须使用与训练集相同的缩放参数
```

#### 3.4 模型训练

```python
from sklearn.linear_model import LinearRegression

# 初始化并训练线性回归模型
model = LinearRegression()
model.fit(X_train_scaled, y_train)
```

#### 3.5 模型评估

```python
from sklearn.metrics import mean_squared_error

# 测试集预测
y_pred = model.predict(X_test_scaled)

# 计算均方误差(MSE)
mse = mean_squared_error(y_test, y_pred)
print(f"模型均方误差(MSE): {mse:.2f}")

# 模型系数分析
print("特征系数:")
print(f"QPS: {model.coef_[0]:.4f}")
print(f"sin_time: {model.coef_[1]:.4f}")
print(f"cos_time: {model.coef_[2]:.4f}")
print(f"截距: {model.intercept_:.4f}")
```

---

### 4. 模型部署与推理

#### 4.1 模型持久化

```python
import joblib

# 保存训练好的模型
joblib.dump(model, "time_qps_auto_scaling_model.pkl")

# 保存标准化器
joblib.dump(scaler, "time_qps_auto_scaling_scaler.pkl")

# 注意：模型和标准化器必须成对使用
```

#### 4.2 模型推理服务

```python
from flask import Flask, request, jsonify
import pandas as pd
import numpy as np
import joblib

app = Flask(__name__)

# 加载模型和标准化器
model = joblib.load("time_qps_auto_scaling_model.pkl")
scaler = joblib.load("time_qps_auto_scaling_scaler.pkl")

@app.route('/predict', methods=['POST'])
def predict():
    # 获取请求数据
    data = request.json

    # 创建特征DataFrame
    features = pd.DataFrame({
        "QPS": [data["qps"]],
        "sin_time": [np.sin(2 * np.pi * data["minutes"] / 1440)],
        "cos_time": [np.cos(2 * np.pi * data["minutes"] / 1440)]
    })

    # 标准化特征
    scaled_features = scaler.transform(features)

    # 预测实例数
    prediction = model.predict(scaled_features)[0]

    # 返回预测结果
    return jsonify({
        "predicted_instances": round(prediction),
        "features": features.to_dict()
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

**API 请求示例**：

```bash
curl -X POST http://localhost:8080/predict \
  -H "Content-Type: application/json" \
  -d '{"qps": 150, "minutes": 630}'  # 630分钟 = 10:30 AM
```

```json
{
  "predicted_instances": 8,
  "features": { "QPS": 150, "cos_time": -0.707, "sin_time": 0.707 }
}
```

---

### 5. Kubernetes HPA Operator 开发

#### 5.1 初始化 Operator 项目

```bash
# 创建项目目录
mkdir hpa-operator && cd hpa-operator

# 初始化Go模块
go mod init github.com/lostar01/hpa-operator

# 初始化Kubebuilder项目
kubebuilder init --domain=aiops.com

# 创建API和控制器
kubebuilder create api \
  --group hpa \
  --version v1 \
  --kind PredictHPA
```

#### 5.2 CRD 结构设计 (api/v1/predicthpa_types.go)

```go
type PredictHPASpec struct {
    // 目标部署名称
    TargetDeployment string `json:"targetDeployment"`

    // 预测服务端点
    PredictorEndpoint string `json:"predictorEndpoint"`

    // 最小实例数
    MinReplicas int32 `json:"minReplicas,omitempty"`

    // 最大实例数
    MaxReplicas int32 `json:"maxReplicas,omitempty"`

    // 指标采集间隔（秒）
    MetricsInterval int32 `json:"metricsInterval,omitempty"`
}

type PredictHPAStatus struct {
    // 当前副本数
    CurrentReplicas int32 `json:"currentReplicas"`

    // 最后预测时间
    LastPredictedTime metav1.Time `json:"lastPredictedTime"`
}
```

#### 5.3 控制器核心逻辑 (controllers/predicthpa_controller.go)

```go
func (r *PredictHPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. 获取PredictHPA实例
    var phpa hpaaiopsv1.PredictHPA
    if err := r.Get(ctx, req.NamespacedName, &phpa); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. 获取目标Deployment
    deployment := &appsv1.Deployment{}
    if err := r.Get(ctx, types.NamespacedName{
        Name:      phpa.Spec.TargetDeployment,
        Namespace: req.Namespace,
    }, deployment); err != nil {
        return ctrl.Result{}, err
    }

    // 3. 从监控系统获取QPS指标
    qps, err := r.metricsClient.GetQPS(phpa.Spec.TargetDeployment)
    if err != nil {
        return ctrl.Result{}, err
    }

    // 4. 计算当前时间特征
    now := time.Now()
    minutes := now.Hour()*60 + now.Minute()
    sinTime := math.Sin(2 * math.Pi * float64(minutes) / 1440)
    cosTime := math.Cos(2 * math.Pi * float64(minutes) / 1440)

    // 5. 调用预测服务
    prediction, err := r.predictor.Predict(qps, sinTime, cosTime)
    if err != nil {
        return ctrl.Result{}, err
    }

    // 6. 更新Deployment副本数
    if *deployment.Spec.Replicas != prediction {
        *deployment.Spec.Replicas = prediction
        if err := r.Update(ctx, deployment); err != nil {
            return ctrl.Result{}, err
        }
    }

    // 7. 更新状态
    phpa.Status.CurrentReplicas = prediction
    phpa.Status.LastPredictedTime = metav1.Now()
    if err := r.Status().Update(ctx, &phpa); err != nil {
        return ctrl.Result{}, err
    }

    // 8. 设置下次调和时间
    return ctrl.Result{
        RequeueAfter: time.Duration(phpa.Spec.MetricsInterval) * time.Second,
    }, nil
}
```

#### 5.4 部署 Operator

```bash
# 生成CRD manifests
make manifests

# 安装CRD
make install

# 构建并推送Operator镜像
make docker-build docker-push IMG=your-registry/hpa-operator:v1

# 部署Operator到集群
make deploy IMG=your-registry/hpa-operator:v1
```

#### 5.5 创建 PredictHPA 实例

```yaml
apiVersion: hpa.aiops.com/v1
kind: PredictHPA
metadata:
  name: webapp-predict-hpa
  namespace: production
spec:
  targetDeployment: webapp-frontend
  predictorEndpoint: "http://predict-service.production.svc:8080/predict"
  minReplicas: 2
  maxReplicas: 20
  metricsInterval: 30 # 每30秒预测一次
```

---

### 架构总结

1. **数据流**：
   监控系统 -> Operator -> 预测服务 -> Kubernetes Deployment
2. **核心组件**：
   - 时间序列预测模型（Python）
   - 预测服务（REST API）
   - 自定义 HPA Operator（Go）
3. **自动扩缩流程**：
   ```mermaid
   graph LR
   A[监控指标] --> B[Operator]
   B --> C[预测服务]
   C --> D[预测结果]
   D --> E[调整副本数]
   E --> F[应用部署]
   ```