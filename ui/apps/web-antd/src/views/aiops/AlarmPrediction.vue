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
        <a-button type="primary" class="refresh-btn" @click="refreshData">
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
            <line-chart-outlined /> 预测准确率
          </template>
          <div class="stat-value">{{ deploymentStats.modelAccuracy }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> +{{ deploymentStats.accuracyImprovement }}%
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
            <pie-chart-outlined /> 资源利用率分布
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
let updateTimer: any = null;

// 模拟数据 - 部署统计数据
const deploymentStats = ref({
  currentReplicas: 4,
  recommendedReplicas: 6,
  replicasTrend: 2,
  modelAccuracy: 92.5,
  accuracyImprovement: 3.2,
  updateInterval: 30,
  nextUpdateTime: "15:30:25"
});

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
  },
  {
    id: 'SCALE-2025-0516-008',
    resource: 'api-gateway',
    scaleType: '缩容',
    timestamp: '2025-05-16 23:15:10',
    oldReplicas: 6,
    newReplicas: 3,
    replicaChange: '6 → 3',
    confidence: 88,
    reason: '夜间流量下降，资源利用率低',
    features: [
      { name: 'CPU利用率', value: '22%' },
      { name: '内存利用率', value: '35%' },
      { name: '请求量/分钟', value: '320' },
      { name: '响应时间', value: '90ms' }
    ]
  }
]);

// 计算属性
const hasAutoScaleEvents = computed(() => {
  return scaleHistory.value.some(item => item.confidence > 90);
});

// 方法
const refreshData = () => {
  loading.value = true;
  setTimeout(() => {
    // 模拟从预测服务获取数据
    message.success('数据已从预测服务刷新');
    
    // 更新推荐副本数和统计数据
    const now = new Date();
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const seconds = String(now.getSeconds()).padStart(2, '0');
    
    // 模拟从预测服务返回的推荐副本数
    const newRecommendedReplicas = Math.floor(Math.random() * 4) + 3; // 随机3-6之间
    
    deploymentStats.value = {
      currentReplicas: deploymentStats.value.currentReplicas,
      recommendedReplicas: newRecommendedReplicas,
      replicasTrend: newRecommendedReplicas - deploymentStats.value.currentReplicas,
      modelAccuracy: 92.8,
      accuracyImprovement: 3.5,
      updateInterval: 30,
      nextUpdateTime: `${hours}:${minutes}:${seconds}`
    };

    // 模拟自动调整副本数
    applyRecommendedReplicas();

    // 重新初始化图表
    initCharts();
    loading.value = false;
  }, 800);
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

// 应用推荐的副本数
const applyRecommendedReplicas = () => {
  // 模拟通过修改 deployment.Spec.Replicas 字段调整副本数
  if (deploymentStats.value.currentReplicas !== deploymentStats.value.recommendedReplicas) {
    // 添加新的伸缩事件记录
    const now = new Date();
    const formattedTime = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`;
    
    const scaleType = deploymentStats.value.recommendedReplicas > deploymentStats.value.currentReplicas ? '扩容' : '缩容';
    const reason = scaleType === '扩容' 
      ? '预测到流量增加，提前扩容以保证服务质量' 
      : '预测到流量减少，缩容以节约资源成本';
    
    const eventId = `SCALE-${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}-${String(scaleHistory.value.length + 1).padStart(3, '0')}`;
    
    // 生成模拟的特征数据
    const cpuUsage = scaleType === '扩容' ? `${Math.floor(Math.random() * 15) + 70}%` : `${Math.floor(Math.random() * 20) + 15}%`;
    const memUsage = scaleType === '扩容' ? `${Math.floor(Math.random() * 15) + 60}%` : `${Math.floor(Math.random() * 25) + 30}%`;
    const reqPerMin = scaleType === '扩容' ? `${Math.floor(Math.random() * 1000) + 2000}` : `${Math.floor(Math.random() * 500) + 200}`;
    const respTime = scaleType === '扩容' ? `${Math.floor(Math.random() * 100) + 150}ms` : `${Math.floor(Math.random() * 50) + 80}ms`;
    
    const newEvent = {
      id: eventId,
      resource: 'frontend-service',
      scaleType: scaleType,
      timestamp: formattedTime,
      oldReplicas: deploymentStats.value.currentReplicas,
      newReplicas: deploymentStats.value.recommendedReplicas,
      replicaChange: `${deploymentStats.value.currentReplicas} → ${deploymentStats.value.recommendedReplicas}`,
      confidence: Math.floor(Math.random() * 10) + 85, // 85-95之间的随机值
      reason: reason,
      features: [
        { name: 'CPU利用率', value: cpuUsage },
        { name: '内存利用率', value: memUsage },
        { name: '请求量/分钟', value: reqPerMin },
        { name: '响应时间', value: respTime }
      ]
    };
    
    // 将新事件添加到历史列表的开头
    scaleHistory.value.unshift(newEvent);
    
    // 更新当前副本数为推荐副本数
    deploymentStats.value.currentReplicas = deploymentStats.value.recommendedReplicas;
    
    // message.success(`已自动${scaleType}至${deploymentStats.value.currentReplicas}个副本`);
  }
};

// 处理按钮点击事件
const handleAction = (action: string) => {
  switch (action) {
    case 'apply':
      message.success('已手动确认应用副本数调整');
      break;
    case 'export':
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

  // 生成过去24小时的时间点
  const now = new Date();
  const hours = Array.from({ length: 12 }, (_, i) => {
    const pastTime = new Date(now.getTime() - (11 - i) * 2 * 60 * 60 * 1000);
    return `${String(pastTime.getHours()).padStart(2, '0')}:00`;
  });

  // 生成副本数和负载数据
  const replicaData = [2, 2, 3, 3, 4, 4, 5, 6, 6, 4, 4, 4];
  const cpuLoadData = [30, 35, 55, 65, 75, 82, 88, 90, 75, 60, 50, 45];
  const requestsData = [800, 950, 1500, 2200, 2800, 3100, 3400, 3500, 2800, 2000, 1600, 1300];

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['副本数', 'CPU负载(%)', '请求数/分钟(/100)'],
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
        name: '副本数',
        type: 'line',
        data: replicaData,
        lineStyle: {
          width: 4,
          type: 'dashed'
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
        name: 'CPU负载(%)',
        type: 'line',
        data: cpuLoadData,
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
      },
      {
        name: '请求数/分钟(/100)',
        type: 'line',
        data: requestsData.map(val => val / 100),
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(82, 196, 26, 0.8)' },
            { offset: 1, color: 'rgba(82, 196, 26, 0.1)' }
          ])
        },
        lineStyle: {
          width: 2
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

// 初始化资源利用率分布图表
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
      data: ['CPU使用率', '内存使用率', '网络I/O', '磁盘I/O', '空闲资源'],
      textStyle: {
        color: '#333333'
      }
    },
    series: [
      {
        name: '资源利用率',
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
          { value: 35, name: 'CPU使用率', itemStyle: { color: '#ff4d4f' } },
          { value: 28, name: '内存使用率', itemStyle: { color: '#faad14' } },
          { value: 18, name: '网络I/O', itemStyle: { color: '#1890ff' } },
          { value: 12, name: '磁盘I/O', itemStyle: { color: '#52c41a' } },
          { value: 7, name: '空闲资源', itemStyle: { color: '#d9d9d9' } }
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
      data: ['CPU利用率', '内存利用率', '请求数(百/分钟)', '响应时间(ms)'],
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
        name: '请求数(百/分钟)',
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
    refreshData();
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