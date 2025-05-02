<template>
  <div class="alarm-prediction-container">
    <div class="header">
      <h1 class="title">智能运维告警预测系统</h1>
      <div class="actions">
        <a-select v-model:value="timeRange" style="width: 150px" class="time-selector">
          <a-select-option value="1h">未来1小时</a-select-option>
          <a-select-option value="6h">未来6小时</a-select-option>
          <a-select-option value="24h">未来24小时</a-select-option>
          <a-select-option value="7d">未来7天</a-select-option>
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
            <thunderbolt-outlined /> 预测告警总数
          </template>
          <div class="stat-value">{{ predictionStats.total }}</div>
          <div class="stat-trend" :class="predictionStats.trend > 0 ? 'up' : 'down'">
            <template v-if="predictionStats.trend > 0">
              <arrow-up-outlined /> +{{ predictionStats.trend }}%
            </template>
            <template v-else>
              <arrow-down-outlined /> {{ predictionStats.trend }}%
            </template>
          </div>
        </a-card>
        <a-card class="stat-card prediction-card">
          <template #title>
            <warning-outlined /> 高危告警预测
          </template>
          <div class="stat-value">{{ predictionStats.critical }}</div>
          <div class="stat-trend" :class="predictionStats.criticalTrend > 0 ? 'up' : 'down'">
            <template v-if="predictionStats.criticalTrend > 0">
              <arrow-up-outlined /> +{{ predictionStats.criticalTrend }}%
            </template>
            <template v-else>
              <arrow-down-outlined /> {{ predictionStats.criticalTrend }}%
            </template>
          </div>
        </a-card>
        <a-card class="stat-card prediction-card">
          <template #title>
            <line-chart-outlined /> 预测准确率
          </template>
          <div class="stat-value">{{ predictionStats.accuracy }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> +{{ predictionStats.accuracyImprovement }}%
          </div>
        </a-card>
        <a-card class="stat-card prediction-card">
          <template #title>
            <clock-circle-outlined /> 预测提前时间
          </template>
          <div class="stat-value">{{ predictionStats.leadTime }}分钟</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> +{{ predictionStats.leadTimeImprovement }}%
          </div>
        </a-card>
      </div>

      <div class="charts-container">
        <a-card class="chart-card">
          <template #title>
            <area-chart-outlined /> 未来告警趋势预测
          </template>
          <div class="chart" ref="trendChartRef"></div>
        </a-card>

        <a-card class="chart-card">
          <template #title>
            <pie-chart-outlined /> 告警类型分布预测
          </template>
          <div class="chart" ref="typeChartRef"></div>
        </a-card>
      </div>

      <a-card class="prediction-table-card">
        <template #title>
          <table-outlined /> 告警预测详情
          <a-tag color="warning" v-if="hasHighRiskPredictions" class="blink-tag">
            <warning-outlined /> 高风险预警
          </a-tag>
        </template>
        <a-table :columns="columns" :data-source="predictionList" :pagination="{ pageSize: 5 }" :loading="loading">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'severity'">
              <a-tag :color="getSeverityColor(record.severity)">
                {{ record.severity }}
              </a-tag>
            </template>
            <template v-if="column.key === 'probability'">
              <a-progress :percent="record.probability" :stroke-color="getProbabilityColor(record.probability)"
                size="small" />
            </template>
            <template v-if="column.key === 'action'">
              <a-button type="primary" size="small" @click="showDetails(record)">详情</a-button>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <a-modal v-model:visible="detailModalVisible" title="告警预测详情" width="700px" :footer="null"
      class="prediction-detail-modal">
      <template v-if="selectedPrediction">
        <div class="prediction-detail-header">
          <div class="prediction-id">
            <span class="label">预测ID:</span>
            <span class="value">{{ selectedPrediction.id }}</span>
          </div>
          <a-tag :color="getSeverityColor(selectedPrediction.severity)" class="severity-tag">
            {{ selectedPrediction.severity }}
          </a-tag>
        </div>

        <div class="prediction-detail-content">
          <div class="detail-item">
            <span class="label">预测资源:</span>
            <span class="value">{{ selectedPrediction.resource }}</span>
          </div>
          <div class="detail-item">
            <span class="label">预测时间:</span>
            <span class="value">{{ selectedPrediction.predictedTime }}</span>
          </div>
          <div class="detail-item">
            <span class="label">告警类型:</span>
            <span class="value">{{ selectedPrediction.type }}</span>
          </div>
          <div class="detail-item">
            <span class="label">发生概率:</span>
            <a-progress :percent="selectedPrediction.probability"
              :stroke-color="getProbabilityColor(selectedPrediction.probability)" size="small" />
          </div>
          <div class="detail-item">
            <span class="label">可能原因:</span>
            <span class="value">{{ selectedPrediction.possibleCause }}</span>
          </div>
          <div class="detail-item">
            <span class="label">建议操作:</span>
            <div class="recommendation-list">
              <div v-for="(rec, index) in selectedPrediction.recommendations" :key="index" class="recommendation-item">
                <check-circle-outlined /> {{ rec }}
              </div>
            </div>
          </div>
        </div>

        <div class="prediction-detail-chart">
          <div class="chart-title">相关指标趋势</div>
          <div class="metric-chart" ref="metricChartRef"></div>
        </div>

        <div class="prediction-actions">
          <a-button type="primary">
            <template #icon><notification-outlined /></template>
            设置提醒
          </a-button>
          <a-button>
            <template #icon><export-outlined /></template>
            导出报告
          </a-button>
          <a-button type="dashed">
            <template #icon><solution-outlined /></template>
            自动修复
          </a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue';
import * as echarts from 'echarts';
import {
  SyncOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  LineChartOutlined,
  ClockCircleOutlined,
  AreaChartOutlined,
  PieChartOutlined,
  TableOutlined,
  CheckCircleOutlined,
  NotificationOutlined,
  ExportOutlined,
  SolutionOutlined
} from '@ant-design/icons-vue';

// 状态定义
const timeRange = ref('24h');
const loading = ref(false);
const trendChartRef = ref(null);
const typeChartRef = ref(null);
const metricChartRef = ref(null);
const detailModalVisible = ref(false);
const selectedPrediction = ref<any>(null);

// 模拟数据
const predictionStats = ref({
  total: 52,
  trend: 8,
  critical: 11,
  criticalTrend: -3,
  accuracy: 96,
  accuracyImprovement: "3.2",
  leadTime: 42,
  leadTimeImprovement: 12
});

const columns = [
  {
    title: '预测ID',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '资源',
    dataIndex: 'resource',
    key: 'resource',
  },
  {
    title: '告警类型',
    dataIndex: 'type',
    key: 'type',
  },
  {
    title: '预测时间',
    dataIndex: 'predictedTime',
    key: 'predictedTime',
  },
  {
    title: '严重程度',
    dataIndex: 'severity',
    key: 'severity',
  },
  {
    title: '发生概率',
    dataIndex: 'probability',
    key: 'probability',
  },
  {
    title: '操作',
    key: 'action',
  },
];

// 生成当前时间后的随机时间
const generateFutureTime = (maxHours = 24) => {
  const now = new Date();
  const futureDate = new Date(now.getTime() + Math.random() * maxHours * 60 * 60 * 1000);
  return `${futureDate.getFullYear()}-${String(futureDate.getMonth() + 1).padStart(2, '0')}-${String(futureDate.getDate()).padStart(2, '0')} ${String(futureDate.getHours()).padStart(2, '0')}:${String(futureDate.getMinutes()).padStart(2, '0')}`;
};

// 获取当前年份
const currentYear = new Date().getFullYear();

// 模拟预测列表数据
const predictionList = ref([
  {
    id: `PRED-${currentYear}-0127`,
    resource: 'API网关服务 (api-gateway-prod)',
    type: '响应时间异常',
    predictedTime: generateFutureTime(2),
    severity: '严重',
    probability: 94,
    possibleCause: '后端服务连接池耗尽，数据库查询性能下降导致API响应延迟',
    recommendations: [
      '扩展后端服务实例数量',
      '优化关键路径SQL查询',
      '增加连接池容量并调整超时参数'
    ]
  },
  {
    id: `PRED-${currentYear}-0128`,
    resource: '主数据库集群 (postgres-main-01)',
    type: '连接数接近上限',
    predictedTime: generateFutureTime(5),
    severity: '警告',
    probability: 82,
    possibleCause: '应用服务未正确释放数据库连接，连接泄漏导致可用连接数减少',
    recommendations: [
      '检查应用代码中的连接关闭逻辑',
      '实施连接池监控告警',
      '临时增加最大连接数限制'
    ]
  },
  {
    id: `PRED-${currentYear}-0129`,
    resource: '对象存储服务 (oss-bucket-media)',
    type: '存储容量告警',
    predictedTime: generateFutureTime(10),
    severity: '一般',
    probability: 68,
    possibleCause: '媒体文件上传量激增，清理策略未及时执行',
    recommendations: [
      '启动紧急清理过期媒体文件任务',
      '调整自动清理策略频率',
      '评估扩展存储容量需求'
    ]
  },
  {
    id: `PRED-${currentYear}-0130`,
    resource: '微服务集群 (payment-service)',
    type: '服务熔断风险',
    predictedTime: generateFutureTime(1),
    severity: '严重',
    probability: 91,
    possibleCause: '第三方支付网关响应缓慢，导致服务调用超时增加',
    recommendations: [
      '临时增加熔断器超时阈值',
      '启用支付请求本地缓存',
      '切换至备用支付网关'
    ]
  },
  {
    id: `PRED-${currentYear}-0131`,
    resource: '消息队列集群 (rabbitmq-prod-02)',
    type: '队列积压',
    predictedTime: generateFutureTime(4),
    severity: '警告',
    probability: 76,
    possibleCause: '消费者处理能力下降，新版本代码引入性能回退',
    recommendations: [
      '回滚至上一版本消费者代码',
      '增加消费者实例数量',
      '优化消息处理逻辑'
    ]
  },
  {
    id: `PRED-${currentYear}-0132`,
    resource: 'CDN边缘节点 (cdn-edge-east)',
    type: '缓存命中率下降',
    predictedTime: generateFutureTime(8),
    severity: '一般',
    probability: 63,
    possibleCause: '新内容发布导致缓存失效请求增加，源站负载上升',
    recommendations: [
      '预热热门内容缓存',
      '调整缓存策略',
      '临时增加源站容量'
    ]
  },
  {
    id: `PRED-${currentYear}-0133`,
    resource: '容器编排平台 (k8s-prod-cluster)',
    type: '节点资源不足',
    predictedTime: generateFutureTime(3),
    severity: '严重',
    probability: 89,
    possibleCause: '自动扩缩容策略参数设置不合理，新应用部署资源请求过高',
    recommendations: [
      '调整HPA配置参数',
      '优化容器资源请求设置',
      '紧急添加新的工作节点'
    ]
  }
]);

// 计算属性
const hasHighRiskPredictions = computed(() => {
  return predictionList.value.some(item => item.severity === '严重' && item.probability > 80);
});

// 方法
const refreshData = () => {
  loading.value = true;
  setTimeout(() => {
    // 模拟数据刷新
    predictionStats.value = {
      total: Math.floor(Math.random() * 30) + 40,
      trend: Math.floor(Math.random() * 20) - 5,
      critical: Math.floor(Math.random() * 10) + 8,
      criticalTrend: Math.floor(Math.random() * 20) - 10,
      accuracy: Math.floor(Math.random() * 5) + 93,
      accuracyImprovement: (Math.random() * 3 + 1).toFixed(1),
      leadTime: Math.floor(Math.random() * 20) + 30,
      leadTimeImprovement: Math.floor(Math.random() * 10) + 8
    };

    // 更新预测列表
    predictionList.value.forEach(item => {
      item.predictedTime = generateFutureTime(24);
      item.probability = Math.floor(Math.random() * 30) + 65;
    });

    // 重新初始化图表
    initCharts();
    loading.value = false;
  }, 1000);
};

const getSeverityColor = (severity: string): string => {
  switch (severity) {
    case '严重': return 'red';
    case '警告': return 'orange';
    case '一般': return 'blue';
    default: return 'green';
  }
};

const getProbabilityColor = (probability: number): string => {
  if (probability >= 80) return '#ff4d4f';
  if (probability >= 60) return '#faad14';
  return '#52c41a';
};

const showDetails = (record: any) => {
  selectedPrediction.value = record;
  detailModalVisible.value = true;

  // 在模态框显示后初始化指标图表
  setTimeout(() => {
    initMetricChart();
  }, 100);
};

// 初始化趋势预测图表
const initTrendChart = () => {
  if (!trendChartRef.value) return;

  const chart = echarts.init(trendChartRef.value);

  // 生成未来24小时的时间点
  const now = new Date();
  const hours = Array.from({ length: 24 }, (_, i) => {
    const futureTime = new Date(now.getTime() + i * 60 * 60 * 1000);
    return `${String(futureTime.getHours()).padStart(2, '0')}:00`;
  });

  // 生成模拟数据
  const generateData = (baseValue: number, variance: number) => {
    return Array.from({ length: 24 }, (_, i) => {
      // 模拟工作时间段(9:00-18:00)告警增多的情况
      const hour = (now.getHours() + i) % 24;
      const isWorkHour = hour >= 9 && hour <= 18;
      const workHourFactor = isWorkHour ? 1.5 : 0.7;
      return Math.floor(Math.random() * variance * workHourFactor) + baseValue * workHourFactor;
    });
  };

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['严重告警', '警告告警', '一般告警'],
      textStyle: {
        color: '#333333'  // 修改为深色字体
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
          color: '#333333'  // 修改为深色
        }
      },
      axisLabel: {
        color: '#333333'  // 修改为深色字体
      }
    },
    yAxis: {
      type: 'value',
      name: '预测告警数',
      nameTextStyle: {
        color: '#333333'  // 修改为深色字体
      },
      axisLine: {
        lineStyle: {
          color: '#333333'  // 修改为深色
        }
      },
      axisLabel: {
        color: '#333333'  // 修改为深色字体
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(0, 0, 0, 0.1)'  // 修改为浅灰色
        }
      }
    },
    series: [
      {
        name: '严重告警',
        type: 'line',
        stack: 'Total',
        data: generateData(3, 3),
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(255, 77, 79, 0.8)' },
            { offset: 1, color: 'rgba(255, 77, 79, 0.1)' }
          ])
        },
        lineStyle: {
          width: 2
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
        name: '警告告警',
        type: 'line',
        stack: 'Total',
        data: generateData(5, 4),
        areaStyle: {
          opacity: 0.3,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(250, 173, 20, 0.8)' },
            { offset: 1, color: 'rgba(250, 173, 20, 0.1)' }
          ])
        },
        lineStyle: {
          width: 2
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
        name: '一般告警',
        type: 'line',
        stack: 'Total',
        data: generateData(7, 5),
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

// 初始化告警类型分布图表
const initTypeChart = () => {
  if (!typeChartRef.value) return;

  const chart = echarts.init(typeChartRef.value);

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: ['性能相关', '资源容量', '连接问题', '服务可用性', '安全风险'],
      textStyle: {
        color: '#333333'  // 修改为深色字体
      }
    },
    series: [
      {
        name: '告警类型',
        type: 'pie',
        radius: ['50%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#ffffff',  // 修改为白色背景
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
            color: '#333333'  // 修改为深色字体
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: 32, name: '性能相关', itemStyle: { color: '#ff4d4f' } },
          { value: 26, name: '资源容量', itemStyle: { color: '#faad14' } },
          { value: 19, name: '连接问题', itemStyle: { color: '#1890ff' } },
          { value: 14, name: '服务可用性', itemStyle: { color: '#52c41a' } },
          { value: 9, name: '安全风险', itemStyle: { color: '#722ed1' } }
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

  // 生成过去6小时到未来6小时的时间点
  const now = new Date();
  const hours = Array.from({ length: 13 }, (_, i) => {
    const time = new Date(now.getTime() + (i - 6) * 60 * 60 * 1000);
    return `${String(time.getHours()).padStart(2, '0')}:${String(time.getMinutes()).padStart(2, '0')}`;
  });

  // 生成模拟数据，在预测时间点附近有明显变化
  const generateMetricData = (baseValue: number, anomalyFactor: number) => {
    const data = [];
    for (let i = 0; i < 13; i++) {
      if (i < 6) {
        // 过去数据，相对平稳但有小波动
        data.push(Math.floor(Math.random() * 15) + baseValue);
      } else if (i === 6) {
        // 当前时间点，开始有轻微变化
        data.push(Math.floor(Math.random() * 15) + baseValue + 5);
      } else {
        // 未来数据，呈现明显趋势
        const trend = (i - 6) * anomalyFactor;
        data.push(Math.floor(Math.random() * 10) + baseValue + trend);
      }
    }
    return data;
  };

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
      data: ['CPU使用率', '内存使用率', '响应时间'],
      textStyle: {
        color: '#333333'  // 修改为深色字体
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
          color: '#333333'  // 修改为深色
        }
      },
      axisLabel: {
        color: '#333333',  // 修改为深色字体
        formatter: function(value: string) {
          return value;
        }
      },
      axisPointer: {
        label: {
          formatter: function (params: any) {
            return '时间: ' + params.value;
          }
        }
      }
    },
    yAxis: {
      type: 'value',
      name: '使用率/响应时间',
      nameTextStyle: {
        color: '#333333'  // 修改为深色字体
      },
      axisLine: {
        lineStyle: {
          color: '#333333'  // 修改为深色
        }
      },
      axisLabel: {
        color: '#333333'  // 修改为深色字体
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(0, 0, 0, 0.1)'  // 修改为浅灰色
        }
      }
    },
    series: [
      {
        name: 'CPU使用率',
        type: 'line',
        data: generateMetricData(45, 8),
        markArea: {
          itemStyle: {
            color: 'rgba(255, 77, 79, 0.1)'
          },
          data: [
            [
              { xAxis: '6' },
              { xAxis: '12' }
            ]
          ]
        },
        markPoint: {
          data: [
            { type: 'max', name: '最大值' },
            { type: 'min', name: '最小值' }
          ]
        },
        markLine: {
          data: [
            { type: 'average', name: '平均值' },
            {
              yAxis: 85,
              lineStyle: {
                color: '#ff4d4f'
              },
              label: {
                formatter: '告警阈值',
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
        name: '内存使用率',
        type: 'line',
        data: generateMetricData(60, 5),
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
        name: '响应时间',
        type: 'line',
        data: generateMetricData(30, 12),
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
    initTrendChart();
    initTypeChart();
  });
};

// 生命周期钩子
onMounted(() => {
  refreshData();
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

.chart-cards {
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

  .chart-cards {
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
