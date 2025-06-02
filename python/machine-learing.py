import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error
from sklearn.preprocessing import StandardScaler
import joblib


# 加载 csv 数据
df = pd.read_csv("data.csv")

# 把 timestamp 转化成分钟数
df["minutes"] = (
    pd.to_datetime(df["timestamp"], format="%H:%M:%S").dt.hour * 60 
    + pd.to_datetime(df["timestamp"], format="%H:%M:%S").dt.minute
)

df["sin_time"] = np.sin(2 * np.pi * df["minutes"] / 1440)
df["cos_time"] = np.cos(2 * np.pi * df["minutes"] / 1440)

# print(df.to_string())

# 输入特征和目标变量
x = df[["QPS", "sin_time", "cos_time"]]
y = df["instances"]

# 分割数据集： 训练集和测试集
x_train, x_test, y_train, y_test = train_test_split(x,y, test_size=0.2, random_state=0)

# 标准化特征
scaler = StandardScaler()
x_train_scaled =  scaler.fit_transform(x_train)
x_test_scaled = scaler.transform(x_test)

# 训练模型
model = LinearRegression()
model.fit(x_train_scaled, y_train)

# 模型评估
y_pred = model.coef_ * x_test_scaled + model.intercept_
mse = mean_squared_error(y_test , y_pred.sum(axis=1))

print("Mean Squared Error:", mse)

# 保存模型
joblib.dump(model, "time_qps_auto_scaling_model.pkl")

# 保存标准化器
joblib.dump(scaler, "time_qps_auto_scaling_scaler.pkl")