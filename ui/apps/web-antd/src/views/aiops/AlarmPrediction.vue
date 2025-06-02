<template>
  <div class="alarm-prediction-container">
    <div class="header">
      <h1 class="title">智能副本数预测与自动伸缩系统</h1>
      <div class="actions">
        <a-select v-model:value="predictionTimeRange" style="width: 150px" class="time-selector">
          <a-select-option value="1h">预测1小时内</a-select-option>
          <a-select-option value="6h">预测6小时内</a-select-option>
          <a-select-option value="24h">预测24小时内</a-select-option>
          <a-select-option value="7d">预测7天内</a-select-option>
        </a-select>
        <a-button type="primary" class="refresh-btn" @click="refreshData" :loading="loading">
          <template #icon><sync-outlined /></template>
          刷新
        </a-button>
      </div>
    </div>

    <div class="dashboard">
      <div class="stats-cards">
        <a-card class="stat-card prediction-card">
          <template #title>
            <cloud-server-outlined /> 当前副本数
          </template>
          <div class="stat-value">{{ deploymentStats.currentReplicas }}</div>
          <div class="stat-trend" :class="deploymentStats.replicasTrend > 0 ? 'up' : 'down'">
            <template v-if="deploymentStats.replicasTrend > 0">
              <arrow-up-outlined /> +{{ deploymentStats.replicasTrend }}
            </template>
            <template v-else-if="deploymentStats.replicasTrend < 0">
              <arrow-down-outlined /> {{ deploymentStats.replicasTrend }}
            </template>
            <template v-else>
              <minus-outlined /> 无变化
            </template>
          </div>
        </a-card>
        <a-card class="stat-card prediction-card">
          <template #title>
            <rocket-outlined /> 推荐副本数
          </template>
          <div class="stat-value">{{ deploymentStats.recommendedReplicas }}</div>
          <div class="stat-trend" :class="getRecommendationClass()">
            <template v-if="deploymentStats.recommendedReplicas > deploymentStats.currentReplicas">
              <arrow-up-outlined /> 建议扩容
            </template>
            <template v-else-if="deploymentStats.recommendedReplicas < deploymentStats.currentReplicas">
              <arrow-down-outlined /> 建议缩容
            </template>
            <template v-else>
              <check-outlined /> 副本数合适
            </template>
          </div>
        </a-card>
        <a-card class="stat-card prediction-card">
          <template #title>
            <line-chart-outlined /> 当前QPS
          </template>
          <div class="stat-value">{{ deploymentStats.currentQPS }}</div>
          <div class="stat-trend neutral">
            <clock-circle-outlined /> {{ deploymentStats.lastUpdateTime }}
          </div>
        </a-card>
        <a-card class="stat-card prediction-card">
          <template #title>
            <clock-circle-outlined /> 更新时间
          </template>
          <div class="stat-value">{{ deploymentStats.nextUpdateTime }}</div>
          <div class="stat-trend neutral">
            <reload-outlined /> 每 {{ deploymentStats.updateInterval }}s 更新
          </div>
        </a-card>
      </div>

      <div class="charts-container">
        <a-card class="chart-card">
          <template #title>
            <area-chart-outlined /> 负载与副本数历史趋势
          </template>
          <div class="chart" ref="loadChartRef"></div>
        </a-card>

        <a-card class="chart-card">
          <template #title>
            <pie-chart-outlined /> 预测准确性分析
          </template>
          <div class="chart" ref="resourceChartRef"></div>
        </a-card>
      </div>

      <a-card class="prediction-table-card">
        <template #title>
          <table-outlined /> 副本数调整历史
          <a-tag color="warning" v-if="hasAutoScaleEvents" class="blink-tag">
            <warning-outlined /> 自动伸缩事件
          </a-tag>
        </template>
        <a-table :columns="columns" :data-source="scaleHistory" :pagination="{ pageSize: 5 }" :loading="loading">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'scaleType'">
              <a-tag :color="getScaleTypeColor(record.scaleType)">
                {{ record.scaleType }}
              </a-tag>
            </template>
            <template v-if="column.key === 'confidence'">
              <a-progress :percent="record.confidence" :stroke-color="getConfidenceColor(record.confidence)"
                size="small" />
            </template>
            <template v-if="column.key === 'action'">
              <a-button type="primary" size="small" @click="showDetails(record)">详情</a-button>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- API状态提示 -->
    <a-alert 
      v-if="apiError" 
      :message="apiError" 
      type="error" 
      closable 
      @close="apiError = ''"
      style="position: fixed; top: 20px; right: 20px; z-index: 1000; max-width: 300px;"
    />

    <a-modal v-model:visible="detailModalVisible" title="伸缩事件详情" width="700px" :footer="null"
      class="prediction-detail-modal">
      <template v-if="selectedScaleEvent">
        <div class="prediction-detail-header">
          <div class="prediction-id">
            <span class="label">事件ID:</span>
            <span class="value">{{ selectedScaleEvent.id }}</span>
          </div>
          <a-tag :color="getScaleTypeColor(selectedScaleEvent.scaleType)" class="severity-tag">
            {{ selectedScaleEvent.scaleType }}
          </a-tag>
        </div>

        <div class="prediction-detail-content">
          <div class="detail-item">
            <span class="label">操作资源:</span>
            <span class="value">{{ selectedScaleEvent.resource }}</span>
          </div>
          <div class="detail-item">
            <span class="label">事件时间:</span>
            <span class="value">{{ selectedScaleEvent.timestamp }}</span>
          </div>
          <div class="detail-item">
            <span class="label">副本变化:</span>
            <span class="value">{{ selectedScaleEvent.oldReplicas }} → {{ selectedScaleEvent.newReplicas }}</span>
          </div>
          <div class="detail-item">
            <span class="label">预测置信度:</span>
            <a-progress :percent="selectedScaleEvent.confidence"
              :stroke-color="getConfidenceColor(selectedScaleEvent.confidence)" size="small" />
          </div>
          <div class="detail-item">
            <span class="label">触发原因:</span>
            <span class="value">{{ selectedScaleEvent.reason }}</span>
          </div>
          <div class="detail-item">
            <span class="label">模型特征:</span>
            <div class="features-list">
              <div v-for="(feature, index) in selectedScaleEvent.features" :key="index" class="feature-item">
                <check-circle-outlined /> {{ feature.name }}: {{ feature.value }}
              </div>
            </div>
          </div>
        </div>

        <div class="prediction-detail-chart">
          <div class="chart-title">相关指标趋势</div>
          <div class="metric-chart" ref="metricChartRef"></div>
        </div>

        <div class="prediction-actions">
          <a-button type="primary" @click="handleAction('apply')">
            <template #icon><check-circle-outlined /></template>
            确认应用
          </a-button>
          <a-button @click="handleAction('export')">
            <template #icon><export-outlined /></template>
            导出数据
          </a-button>
          <a-button type="dashed" @click="handleAction('manual')">
            <template #icon><edit-outlined /></template>
            手动设置
          </a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, nextTick, onUnmounted } from 'vue';
import * as echarts from 'echarts';
import axios from 'axios';
import {
  SyncOutlined,
  CloudServerOutlined,
  RocketOutlined,
  WarningOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  MinusOutlined,
  CheckOutlined,
  LineChartOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  AreaChartOutlined,
  PieChartOutlined,
  TableOutlined,
  CheckCircleOutlined,
  ExportOutlined,
  EditOutlined
} from '@ant-design/icons-vue';
import { message } from 'ant-design-vue';

// 状态定义
const predictionTimeRange = ref('24h');
const loading = ref(false);
const loadChartRef = ref(null);
const resourceChartRef = ref(null);
const metricChartRef = ref(null);
const detailModalVisible = ref(false);
const selectedScaleEvent = ref<any>(null);
const apiError = ref('');
let updateTimer: any = null;

// API配置 - 根据你的实际部署情况修改
const API_BASE_URL = 'http://localhost:8080'; // 修改为你的Flask服务地址

// 模拟数据 - 部署统计数据
const deploymentStats = ref({
  currentReplicas: 4,
  recommendedReplicas: 4,
  replicasTrend: 0,
  currentQPS: 0,
  lastUpdateTime: '',
  updateInterval: 30,
  nextUpdateTime: "15:30:25"
});

// 存储历史预测数据用于图表展示
const predictionHistory = ref<any[]>([]);

// 表格列定义
const columns = [
  {
    title: '事件ID',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '资源名称',
    dataIndex: 'resource',
    key: 'resource',
  },
  {
    title: '操作类型',
    dataIndex: 'scaleType',
    key: 'scaleType',
  },
  {
    title: '时间',
    dataIndex: 'timestamp',
    key: 'timestamp',
  },
  {
    title: '副本变化',
    dataIndex: 'replicaChange',
    key: 'replicaChange',
  },
  {
    title: '预测置信度',
    dataIndex: 'confidence',
    key: 'confidence',
  },
  {
    title: '操作',
    key: 'action',
  },
];

// 副本数调整历史数据
const scaleHistory = ref([
  {
    id: 'SCALE-2025-0517-001',
    resource: 'frontend-service',
    scaleType: '扩容',
    timestamp: '2025-05-17 14:00:25',
    oldReplicas: 2,
    newReplicas: 4,
    replicaChange: '2 → 4',
    confidence: 95,
    reason: '预测到流量高峰，CPU利用率预期超过80%',
    features: [
      { name: 'CPU利用率', value: '78%' },
      { name: '内存利用率', value: '65%' },
      { name: '请求量/分钟', value: '2450' },
      { name: '响应时间', value: '180ms' }
    ]
  }
]);

// 计算属性
const hasAutoScaleEvents = computed(() => {
  return scaleHistory.value.some(item => item.confidence > 90);
});

// 调用预测API
const fetchPrediction = async () => {
  try {
    const response = await axios.get(`${API_BASE_URL}/predict`, {
      timeout: 10000 // 10秒超时
    });
    
    if (response.data) {
      const { instances, current_qps, timestamp } = response.data;
      
      // 更新统计数据
      const oldRecommendedReplicas = deploymentStats.value.recommendedReplicas;
      deploymentStats.value.recommendedReplicas = instances;
      deploymentStats.value.currentQPS = Math.round(current_qps * 100) / 100; // 保留2位小数
      deploymentStats.value.lastUpdateTime = new Date(timestamp).toLocaleTimeString();
      deploymentStats.value.replicasTrend = instances - deploymentStats.value.currentReplicas;
      
      // 添加到历史数据
      predictionHistory.value.push({
        timestamp: new Date(timestamp),
        instances,
        qps: current_qps,
        currentReplicas: deploymentStats.value.currentReplicas
      });
      
      // 保持历史数据不超过100条
      if (predictionHistory.value.length > 100) {
        predictionHistory.value = predictionHistory.value.slice(-100);
      }
      
      // 如果推荐副本数发生变化，模拟自动伸缩
      if (oldRecommendedReplicas !== instances) {
        await simulateAutoScaling(instances);
      }
      
      // 清除错误信息
      apiError.value = '';
      
      console.log('预测数据更新:', response.data);
      
    } else {
      throw new Error('API返回数据格式错误');
    }
    
  } catch (error: any) {
    console.error('获取预测数据失败:', error);
    
    let errorMessage = '获取预测数据失败';
    if (error.code === 'ECONNREFUSED') {
      errorMessage = '无法连接到预测服务，请检查服务是否正常运行';
    } else if (error.response) {
      errorMessage = `预测服务错误: ${error.response.status} ${error.response.data?.error || error.response.statusText}`;
    } else if (error.request) {
      errorMessage = '请求超时，请检查网络连接';
    } else {
      errorMessage = `请求错误: ${error.message}`;
    }
    
    apiError.value = errorMessage;
    message.error(errorMessage);
  }
};

// 模拟自动伸缩操作
const simulateAutoScaling = async (newReplicas: number) => {
  const now = new Date();
  const formattedTime = now.toLocaleString();
  
  const scaleType = newReplicas > deploymentStats.value.currentReplicas ? '扩容' : '缩容';
  const reason = scaleType === '扩容' 
    ? `基于ML模型预测，当前QPS=${deploymentStats.value.currentQPS}，预计需要扩容以保证服务质量` 
    : `基于ML模型预测，当前QPS=${deploymentStats.value.currentQPS}，可以缩容以节约资源成本`;
  
  const eventId = `SCALE-${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}-${String(scaleHistory.value.length + 1).padStart(3, '0')}`;
  
  // 生成基于真实QPS的特征数据
  const qps = deploymentStats.value.currentQPS;
  const cpuUsage = scaleType === '扩容' ? `${Math.min(85, Math.max(60, Math.floor(qps * 10 + 50)))}%` : `${Math.max(15, Math.min(40, Math.floor(qps * 5 + 20)))}%`;
  const memUsage = scaleType === '扩容' ? `${Math.min(75, Math.max(50, Math.floor(qps * 8 + 45)))}%` : `${Math.max(25, Math.min(50, Math.floor(qps * 6 + 30)))}%`;
  const reqPerMin = `${Math.floor(qps * 60)}`;
  const respTime = scaleType === '扩容' ? `${Math.max(120, Math.floor(200 - qps * 20))}ms` : `${Math.min(100, Math.max(80, Math.floor(90 + qps * 5)))}ms`;
  
  const newEvent = {
    id: eventId,
    resource: 'frontend-service',
    scaleType: scaleType,
    timestamp: formattedTime,
    oldReplicas: deploymentStats.value.currentReplicas,
    newReplicas: newReplicas,
    replicaChange: `${deploymentStats.value.currentReplicas} → ${newReplicas}`,
    confidence: Math.floor(Math.random() * 10) + 85, // 85-95之间的随机值
    reason: reason,
    features: [
      { name: 'CPU利用率', value: cpuUsage },
      { name: '内存利用率', value: memUsage },
      { name: '请求量/分钟', value: reqPerMin },
      { name: '响应时间', value: respTime },
      { name: '当前QPS', value: qps.toString() }
    ]
  };
  
  // 将新事件添加到历史列表的开头
  scaleHistory.value.unshift(newEvent);
  
  // 更新当前副本数为推荐副本数
  deploymentStats.value.currentReplicas = newReplicas;
  deploymentStats.value.replicasTrend = 0; // 重置趋势
  
  message.success(`已自动${scaleType}至${newReplicas}个副本 (基于QPS=${qps}的预测)`);
};

// 方法
const refreshData = async () => {
  loading.value = true;
  
  try {
    await fetchPrediction();
    
    // 更新下次更新时间
    const now = new Date();
    const nextUpdate = new Date(now.getTime() + deploymentStats.value.updateInterval * 1000);
    const hours = String(nextUpdate.getHours()).padStart(2, '0');
    const minutes = String(nextUpdate.getMinutes()).padStart(2, '0');
    const seconds = String(nextUpdate.getSeconds()).padStart(2, '0');
    deploymentStats.value.nextUpdateTime = `${hours}:${minutes}:${seconds}`;

    // 重新初始化图表
    initCharts();
    
    message.success('数据已从预测服务刷新');
    
  } catch (error) {
    console.error('刷新数据失败:', error);
  } finally {
    loading.value = false;
  }
};

const getRecommendationClass = () => {
  if (deploymentStats.value.recommendedReplicas > deploymentStats.value.currentReplicas) return 'up';
  if (deploymentStats.value.recommendedReplicas < deploymentStats.value.currentReplicas) return 'down';
  return 'neutral';
};

const getScaleTypeColor = (scaleType: string): string => {
  switch (scaleType) {
    case '扩容': return 'green';
    case '缩容': return 'blue';
    case '无变化': return 'gray';
    default: return 'orange';
  }
};

const getConfidenceColor = (confidence: number): string => {
  if (confidence >= 90) return '#52c41a';
  if (confidence >= 70) return '#faad14';
  return '#ff4d4f';
};

const showDetails = (record: any) => {
  selectedScaleEvent.value = record;
  detailModalVisible.value = true;

  // 在模态框显示后初始化指标图表
  setTimeout(() => {
    initMetricChart();
  }, 100);
};

// 处理按钮点击事件
const handleAction = (action: string) => {
  switch (action) {
    case 'apply':
      message.success('已手动确认应用副本数调整');
      break;
    case 'export':
      // 导出预测历史数据
      const dataToExport = {
        currentStats: deploymentStats.value,
        predictionHistory: predictionHistory.value,
        scaleHistory: scaleHistory.value
      };
      const blob = new Blob([JSON.stringify(dataToExport, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `prediction-data-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
      URL.revokeObjectURL(url);
      message.success('预测数据已导出');
      break;
    case 'manual':
      message.success('已打开手动设置副本数对话框');
      break;
  }
  
  // 延迟关闭模态框
  setTimeout(() => {
    detailModalVisible.value = false;
  }, 1000);
};

// 初始化负载与副本数历史趋势图表
const initLoadChart = () => {
  if (!loadChartRef.value) return;

  const chart = echarts.init(loadChartRef.value);

  // 使用真实的预测历史数据
  let hours: string[] = [];
  let replicaData: number[] = [];
  let qpsData: number[] = [];
  let predictedReplicaData: number[] = [];

  if (predictionHistory.value.length > 0) {
    // 使用真实数据
    const recentData = predictionHistory.value.slice(-12); // 最近12个数据点
    hours = recentData.map(item => {
      const time = new Date(item.timestamp);
      return `${String(time.getHours()).padStart(2, '0')}:${String(time.getMinutes()).padStart(2, '0')}`;
    });
    replicaData = recentData.map(item => item.currentReplicas);
    qpsData = recentData.map(item => item.qps);
    predictedReplicaData = recentData.map(item => item.instances);
  } else {
    // 生成过去24小时的时间点（模拟数据）
    const now = new Date();
    hours = Array.from({ length: 12 }, (_, i) => {
      const pastTime = new Date(now.getTime() - (11 - i) * 2 * 60 * 60 * 1000);
      return `${String(pastTime.getHours()).padStart(2, '0')}:00`;
    });
    replicaData = [2, 2, 3, 3, 4, 4, 5, 6, 6, 4, 4, 4];
    qpsData = [0.8, 0.95, 1.5, 2.2, 2.8, 3.1, 3.4, 3.5, 2.8, 2.0, 1.6, 1.3];
    predictedReplicaData = [2, 3, 3, 4, 4, 5, 5, 6, 6, 4, 4, 4];
  }

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['当前副本数', '预测副本数', 'QPS'],
      textStyle: {
        color: '#333333'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLine: {
        lineStyle: {
          color: '#333333'
        }
      },
      axisLabel: {
        color: '#333333'
      }
    },
    yAxis: [
      {
        type: 'value',
        name: '副本数',
        nameTextStyle: {
          color: '#333333'
        },
        axisLine: {
          lineStyle: {
            color: '#333333'
          }
        },
        axisLabel: {
          color: '#333333'
        },
        splitLine: {
          lineStyle: {
            color: 'rgba(0, 0, 0, 0.1)'
          }
        }
      },
      {
        type: 'value',
        name: 'QPS',
        nameTextStyle: {
          color: '#333333'
        },
        axisLine: {
          lineStyle: {
            color: '#333333'
          }
        },
        axisLabel: {
          color: '#333333'
        },
        splitLine: {
          show: false
        }
      }
    ],
    series: [
      {
        name: '当前副本数',
        type: 'line',
        yAxisIndex: 0,
        data: replicaData,
        lineStyle: {
          width: 4,
          type: 'solid'
        },
        itemStyle: {
          color: '#722ed1'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: false
      },
      {
        name: '预测副本数',
        type: 'line',
        yAxisIndex: 0,
        data: predictedReplicaData,
        lineStyle: {
          width: 2,
          type: 'dashed'
        },
        itemStyle: {
          color: '#ff4d4f'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: true
      },
      {
        name: 'QPS',
        type: 'line',
        yAxisIndex: 1,
        data: qpsData,
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(24, 144, 255, 0.8)' },
            { offset: 1, color: 'rgba(24, 144, 255, 0.1)' }
          ])
        },
        lineStyle: {
          width: 2
        },
        itemStyle: {
          color: '#1890ff'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: true
      }
    ]
  };

  chart.setOption(option);

  // 响应窗口大小变化
  window.addEventListener('resize', () => {
    chart.resize();
  });
};

// 初始化预测准确性分析图表
const initResourceChart = () => {
  if (!resourceChartRef.value) return;

  const chart = echarts.init(resourceChartRef.value);

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c}%'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: ['高准确度预测', '中等准确度预测', '低准确度预测', '预测偏差', '模型优化空间'],
      textStyle: {
        color: '#333333'
      }
    },
    series: [
      {
        name: '预测准确性',
        type: 'pie',
        radius: ['50%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#ffffff',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: '18',
            fontWeight: 'bold',
            color: '#333333'
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: 68, name: '高准确度预测', itemStyle: { color: '#52c41a' } },
          { value: 22, name: '中等准确度预测', itemStyle: { color: '#faad14' } },
          { value: 6, name: '低准确度预测', itemStyle: { color: '#ff4d4f' } },
          { value: 3, name: '预测偏差', itemStyle: { color: '#f759ab' } },
          { value: 1, name: '模型优化空间', itemStyle: { color: '#d9d9d9' } }
        ]
      }
    ]
  };

  chart.setOption(option);

  // 响应窗口大小变化
  window.addEventListener('resize', () => {
    chart.resize();
  });
};

// 初始化指标图表（详情模态框中）
const initMetricChart = () => {
  if (!metricChartRef.value) return;

  const chart = echarts.init(metricChartRef.value);

  // 生成时间轴
  const hours = ['09:00', '10:00', '11:00', '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00'];

  // 根据当前选中的事件生成相应的指标数据
  let cpuData, memoryData, requestsData, responseTimeData;
  
  if (selectedScaleEvent.value.scaleType === '扩容') {
    // 扩容场景
    cpuData = [50, 55, 62, 70, 75, 82, 85, 80, 75, 72];
    memoryData = [45, 48, 52, 58, 65, 70, 68, 65, 62, 60];
    requestsData = [15, 18, 22, 26, 30, 31, 28, 26, 24, 22];
    responseTimeData = [120, 130, 150, 165, 180, 190, 170, 160, 150, 140];
  } else {
    // 缩容场景
    cpuData = [60, 55, 48, 40, 35, 28, 22, 18, 15, 12];
    memoryData = [55, 50, 45, 40, 35, 30, 28, 25, 22, 20];
    requestsData = [20, 18, 15, 12, 10, 8, 7, 6, 5, 4];
    responseTimeData = [150, 140, 130, 120, 110, 100, 95, 90, 85, 80];
  }

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'line',
        lineStyle: {
          color: '#333333',
          width: 1,
          type: 'dashed'
        }
      }
    },
    legend: {
      data: ['CPU利用率', '内存利用率', 'QPS', '响应时间(ms)'],
      textStyle: {
        color: '#333333'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLine: {
        lineStyle: {
          color: '#333333'
        }
      },
      axisLabel: {
        color: '#333333'
      }
    },
    yAxis: {
      type: 'value',
      name: '数值',
      nameTextStyle: {
        color: '#333333'
      },
      axisLine: {
        lineStyle: {
          color: '#333333'
        }
      },
      axisLabel: {
        color: '#333333'
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(0, 0, 0, 0.1)'
        }
      }
    },
    series: [
      {
        name: 'CPU利用率',
        type: 'line',
        data: cpuData,
        markArea: {
          itemStyle: {
            color: 'rgba(255, 77, 79, 0.1)'
          },
          data: [
            [
              { xAxis: selectedScaleEvent.value.scaleType === '扩容' ? '4' : '2' },
              { xAxis: selectedScaleEvent.value.scaleType === '扩容' ? '7' : '5' }
            ]
          ]
        },
        markPoint: {
          data: [
            { type: selectedScaleEvent.value.scaleType === '扩容' ? 'max' : 'min', name: selectedScaleEvent.value.scaleType === '扩容' ? '最大值' : '最小值' }
          ]
        },
        markLine: {
          data: [
            {
              yAxis: selectedScaleEvent.value.scaleType === '扩容' ? 75 : 25,
              lineStyle: {
                color: selectedScaleEvent.value.scaleType === '扩容' ? '#ff4d4f' : '#52c41a'
              },
              label: {
                formatter: selectedScaleEvent.value.scaleType === '扩容' ? '扩容阈值' : '缩容阈值',
                position: 'end'
              }
            }
          ]
        },
        lineStyle: {
          width: 3
        },
        itemStyle: {
          color: '#ff4d4f'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: true
      },
      {
        name: '内存利用率',
        type: 'line',
        data: memoryData,
        lineStyle: {
          width: 3
        },
        itemStyle: {
          color: '#faad14'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: true
      },
      {
        name: 'QPS',
        type: 'line',
        data: requestsData,
        lineStyle: {
          width: 3
        },
        itemStyle: {
          color: '#1890ff'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: true
      },
      {
        name: '响应时间(ms)',
        type: 'line',
        data: responseTimeData,
        lineStyle: {
          width: 3
        },
        itemStyle: {
          color: '#52c41a'
        },
        emphasis: {
          focus: 'series'
        },
        smooth: true
      }
    ]
  };

  chart.setOption(option);

  // 响应窗口大小变化
  window.addEventListener('resize', () => {
    chart.resize();
  });
};

// 初始化所有图表
const initCharts = () => {
  nextTick(() => {
    initLoadChart();
    initResourceChart();
  });
};

// 自动更新推荐副本数的定时器
const startAutoUpdate = () => {
  updateTimer = setInterval(() => {
    console.log('自动从预测服务获取推荐副本数...');
    fetchPrediction();
  }, deploymentStats.value.updateInterval * 1000);
};

// 生命周期钩子
onMounted(() => {
  refreshData();
  startAutoUpdate();
});

onUnmounted(() => {
  if (updateTimer) clearInterval(updateTimer);
});
</script>

<style scoped>
.alarm-prediction-container {
  padding: 20px;
  min-height: 100vh;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.title {
  font-size: 24px;
  font-weight: bold;
  margin: 0;
  background: linear-gradient(90deg, #1890ff, #52c41a);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 0 10px rgba(24, 144, 255, 0.3);
}

.actions {
  display: flex;
  gap: 12px;
}

.dashboard {
  margin-top: 20px;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  border-radius: 8px;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
}

.stat-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.prediction-card {
  position: relative;
  overflow: hidden;
}

.prediction-card::before {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  z-index: -1;
  border-radius: 10px;
  animation: glowing 10s linear infinite;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  margin: 10px 0;
  color: var(--ant-heading-color);
}

.stat-trend {
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.up {
  color: #52c41a;
}

.down {
  color: #ff4d4f;
}

.neutral {
  color: #1890ff;
}

.charts-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.chart-card {
  border-radius: 8px;
  padding: 16px;
  height: 350px;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
}

.chart-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.chart {
  height: 280px;
}

.prediction-detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.prediction-detail-content {
  margin-bottom: 24px;
}

.detail-item {
  margin-bottom: 12px;
}

.detail-item .label {
  font-weight: bold;
  margin-right: 8px;
  color: #666;
  min-width: 100px;
  display: inline-block;
}

.features-list {
  margin-top: 8px;
}

.feature-item {
  margin-bottom: 6px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.prediction-detail-chart {
  margin-bottom: 24px;
}

.chart-title {
  font-weight: bold;
  margin-bottom: 12px;
  color: #333;
}

.metric-chart {
  height: 300px;
}

.prediction-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.blink-tag {
  animation: blink 1.5s linear infinite;
}

@keyframes blink {
  0% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
  100% {
    opacity: 1;
  }
}

@keyframes glowing {
  0% {
    background-position: 0 0;
  }

  50% {
    background-position: 400% 0;
  }

  100% {
    background-position: 0 0;
  }
}

/* 响应式调整 */
@media (max-width: 1200px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .charts-container {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .stats-cards {
    grid-template-columns: 1fr;
  }

  .header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .actions {
    width: 100%;
    display: flex;
    justify-content: space-between;
  }

  .time-selector {
    width: 48% !important;
  }

  .refresh-btn {
    width: 48%;
  }
}
</style>