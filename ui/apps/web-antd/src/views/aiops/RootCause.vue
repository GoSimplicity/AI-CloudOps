<template>
  <div class="alarm-container">
    <div class="header">
      <h1 class="title">智能运维告警根因分析</h1>
      <div class="actions">
        <a-select v-model:value="timeRange" style="width: 150px" class="time-selector">
          <a-select-option value="1h">最近1小时</a-select-option>
          <a-select-option value="6h">最近6小时</a-select-option>
          <a-select-option value="24h">最近24小时</a-select-option>
          <a-select-option value="7d">最近7天</a-select-option>
        </a-select>
        <a-button type="primary" class="refresh-btn" @click="refreshData">
          <template #icon><sync-outlined /></template>
          刷新
        </a-button>
      </div>
    </div>

    <div class="dashboard">
      <div class="stats-cards">
        <a-card class="stat-card">
          <template #title>
            <alert-outlined /> 告警总数
          </template>
          <div class="stat-value">{{ alarmStats.total }}</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ alarmStats.totalIncrease }}%
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <warning-outlined /> 严重告警
          </template>
          <div class="stat-value">{{ alarmStats.critical }}</div>
          <div class="stat-trend down">
            <arrow-down-outlined /> {{ alarmStats.criticalDecrease }}%
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <check-circle-outlined /> 已解决
          </template>
          <div class="stat-value">{{ alarmStats.resolved }}</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ alarmStats.resolvedIncrease }}%
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <clock-circle-outlined /> 平均解决时间
          </template>
          <div class="stat-value">{{ alarmStats.avgResolveTime }}</div>
          <div class="stat-trend down">
            <arrow-down-outlined /> {{ alarmStats.timeDecrease }}%
          </div>
        </a-card>
      </div>

      <div class="charts-section">
        <a-row :gutter="16">
          <a-col :span="16">
            <a-card title="告警趋势分析" class="chart-card">
              <div ref="trendChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card title="告警类型分布" class="chart-card">
              <div ref="typeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
      </div>

      <a-card title="根因分析结果" class="root-cause-card">
        <a-tabs v-model:activeKey="activeTab">
          <a-tab-pane key="1" tab="实时告警">
            <a-table :dataSource="alarmList" :columns="columns" :pagination="{ pageSize: 5 }" class="alarm-table">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'status'">
                  <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
                </template>
                <template v-if="column.key === 'severity'">
                  <a-tag :color="getSeverityColor(record.severity)">{{ record.severity }}</a-tag>
                </template>
                <template v-if="column.key === 'action'">
                  <a-button type="link" @click="showRootCauseAnalysis(record)">查看根因</a-button>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
          <a-tab-pane key="2" tab="历史告警">
            <a-table :dataSource="historyAlarmList" :columns="columns" :pagination="{ pageSize: 5 }"
              class="alarm-table">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'status'">
                  <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
                </template>
                <template v-if="column.key === 'severity'">
                  <a-tag :color="getSeverityColor(record.severity)">{{ record.severity }}</a-tag>
                </template>
                <template v-if="column.key === 'action'">
                  <a-button type="link" @click="showRootCauseAnalysis(record)">查看根因</a-button>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
        </a-tabs>
      </a-card>
    </div>

    <a-modal v-model:visible="rootCauseModalVisible" title="根因分析详情" width="800px" class="root-cause-modal">
      <div class="root-cause-content">
        <div class="root-cause-header">
          <div class="alarm-info">
            <h3>{{ selectedAlarm?.name }}</h3>
            <p>发生时间: {{ selectedAlarm?.time }}</p>
            <a-tag :color="getSeverityColor(selectedAlarm?.severity || '')">{{ selectedAlarm?.severity }}</a-tag>
          </div>
        </div>

        <a-divider />

        <div class="root-cause-graph">
          <h3>故障传播路径</h3>
          <div ref="rootCauseGraphRef" class="graph-container"></div>
        </div>

        <a-divider />

        <div class="root-cause-analysis">
          <h3>根因分析结果</h3>
          <a-timeline>
            <a-timeline-item v-for="(item, index) in rootCauseSteps" :key="index" :color="item.color">
              <div class="timeline-content">
                <h4>{{ item.title }}</h4>
                <p>{{ item.description }}</p>
                <a-tag v-if="item.confidence">置信度: {{ item.confidence }}%</a-tag>
              </div>
            </a-timeline-item>
          </a-timeline>
        </div>

        <div class="recommendation">
          <h3>修复建议</h3>
          <a-alert type="info" show-icon>
            <template #message>{{ selectedAlarm?.recommendation }}</template>
          </a-alert>
        </div>
      </div>
      <template #footer>
        <a-button @click="rootCauseModalVisible = false">关闭</a-button>
        <a-button type="primary" @click="handleResolveAlarm">标记为已解决</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive } from 'vue';
import {
  SyncOutlined,
  AlertOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined
} from '@ant-design/icons-vue';
import * as echarts from 'echarts';

// 定义告警记录类型
interface AlarmRecord {
  id: string;
  name: string;
  time: string;
  severity: string;
  status: string;
  service: string;
  recommendation: string;
}

// 状态数据
const timeRange = ref('24h');
const activeTab = ref('1');
const rootCauseModalVisible = ref(false);
const selectedAlarm = ref<AlarmRecord | null>(null);

// 图表引用
const trendChartRef = ref(null);
const typeChartRef = ref(null);
const rootCauseGraphRef = ref(null);

// 获取当前日期时间
const getCurrentDateTime = () => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  const hours = String(now.getHours()).padStart(2, '0');
  const minutes = String(now.getMinutes()).padStart(2, '0');
  const seconds = String(now.getSeconds()).padStart(2, '0');

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
};

// 获取过去的时间
const getPastDateTime = (hoursAgo: number) => {
  const date = new Date();
  date.setHours(date.getHours() - hoursAgo);

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  const seconds = String(date.getSeconds()).padStart(2, '0');

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
};

// 告警统计数据
const alarmStats = reactive({
  total: 156,
  totalIncrease: 18,
  critical: 32,
  criticalDecrease: 5,
  resolved: 94,
  resolvedIncrease: 21,
  avgResolveTime: '38分钟',
  timeDecrease: 7
});

// 表格列定义
const columns = [
  { title: '告警ID', dataIndex: 'id', key: 'id' },
  { title: '告警名称', dataIndex: 'name', key: 'name' },
  { title: '告警时间', dataIndex: 'time', key: 'time' },
  { title: '告警级别', dataIndex: 'severity', key: 'severity' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '影响服务', dataIndex: 'service', key: 'service' },
  { title: '操作', key: 'action' }
];

// 告警列表数据
const alarmList = ref<AlarmRecord[]>([
  { id: 'ALM-2023-0057', name: '内存泄漏检测', time: getCurrentDateTime(), severity: '严重', status: '未解决', service: '支付网关', recommendation: '检查最近部署的代码中是否存在内存未释放的情况，重启服务并监控内存使用趋势' },
  { id: 'ALM-2023-0056', name: '数据库连接池耗尽', time: getPastDateTime(2), severity: '严重', status: '处理中', service: '用户中心', recommendation: '增加连接池容量，检查是否存在连接未释放的代码' },
  { id: 'ALM-2023-0055', name: 'API响应延迟增加', time: getPastDateTime(4), severity: '警告', status: '未解决', service: '商品服务', recommendation: '检查数据库索引，优化查询语句，考虑增加缓存层' },
  { id: 'ALM-2023-0054', name: '消息队列积压', time: getPastDateTime(6), severity: '一般', status: '处理中', service: '订单服务', recommendation: '增加消费者数量，检查消息处理逻辑是否存在性能瓶颈' },
  { id: 'ALM-2023-0053', name: '磁盘空间不足', time: getPastDateTime(8), severity: '警告', status: '未解决', service: '日志服务', recommendation: '清理过期日志，考虑扩容或实施日志轮转策略' }
]);

// 历史告警数据
const historyAlarmList = ref<AlarmRecord[]>([
  { id: 'ALM-2023-0052', name: '服务实例崩溃', time: getPastDateTime(20), severity: '严重', status: '已解决', service: '认证服务', recommendation: '检查服务依赖项，增加健康检查和自动恢复机制' },
  { id: 'ALM-2023-0051', name: '缓存命中率下降', time: getPastDateTime(24), severity: '一般', status: '已解决', service: '推荐系统', recommendation: '优化缓存策略，调整TTL，预热热点数据' },
  { id: 'ALM-2023-0050', name: '网络延迟波动', time: getPastDateTime(36), severity: '警告', status: '已解决', service: '网关服务', recommendation: '检查网络设备，优化路由配置，考虑使用CDN' },
  { id: 'ALM-2023-0049', name: 'SSL证书即将过期', time: getPastDateTime(40), severity: '警告', status: '已解决', service: '安全服务', recommendation: '更新SSL证书，设置自动续期提醒' },
  { id: 'ALM-2023-0048', name: '数据库慢查询', time: getPastDateTime(48), severity: '一般', status: '已解决', service: '搜索服务', recommendation: '优化SQL语句，添加适当索引，考虑分表分库' }
]);

// 根因分析步骤
const rootCauseSteps = ref([
  { title: '告警触发', description: '系统检测到内存使用率持续上升，超过阈值90%，触发告警', color: 'red', confidence: 100 },
  { title: '服务定位', description: '支付网关服务的多个实例均出现内存占用异常增长', color: 'orange', confidence: 98 },
  { title: '日志分析', description: '日志显示大量对象创建但未释放，垃圾回收频繁触发', color: 'orange', confidence: 92 },
  { title: '代码审查', description: '最近部署的支付验证模块存在资源未释放问题', color: 'green', confidence: 88 },
  { title: '根因确认', description: '支付验证模块中的HTTP连接未正确关闭，导致连接资源泄漏', color: 'green', confidence: 95 }
]);

// 获取状态颜色
const getStatusColor = (status: string): string => {
  const colorMap: Record<string, string> = {
    '未解决': 'red',
    '处理中': 'orange',
    '已解决': 'green'
  };
  return colorMap[status] || 'blue';
};

// 获取严重程度颜色
const getSeverityColor = (severity: string): string => {
  const colorMap: Record<string, string> = {
    '严重': 'red',
    '警告': 'orange',
    '一般': 'blue'
  };
  return colorMap[severity] || 'blue';
};

// 显示根因分析
const showRootCauseAnalysis = (record: AlarmRecord): void => {
  selectedAlarm.value = record;
  rootCauseModalVisible.value = true;

  // 在模态框显示后初始化根因分析图
  setTimeout(() => {
    initRootCauseGraph();
  }, 100);
};

// 处理解决告警
// 处理解决告警
const handleResolveAlarm = (): void => {
  if (selectedAlarm.value) {
    selectedAlarm.value.status = '已解决';
    rootCauseModalVisible.value = false;

    // 更新列表中的状态
    const index = alarmList.value.findIndex(item => item.id === selectedAlarm.value?.id);
    if (index !== -1 && alarmList.value[index]) { // 验证索引有效且元素存在
      alarmList.value[index].status = '已解决';
    }
  }
};

// 刷新数据
const refreshData = () => {
  // 更新告警统计数据
  alarmStats.total = Math.floor(Math.random() * 50) + 120;
  alarmStats.totalIncrease = Math.floor(Math.random() * 20) + 10;
  alarmStats.critical = Math.floor(Math.random() * 30) + 15;
  alarmStats.criticalDecrease = Math.floor(Math.random() * 15) + 3;
  alarmStats.resolved = Math.floor(Math.random() * 40) + 80;
  alarmStats.resolvedIncrease = Math.floor(Math.random() * 25) + 10;
  alarmStats.avgResolveTime = `${Math.floor(Math.random() * 60) + 20}分钟`;
  alarmStats.timeDecrease = Math.floor(Math.random() * 10) + 2;

  // 更新图表
  initTrendChart();
  initTypeChart();
};

// 初始化趋势图表
const initTrendChart = () => {
  const chartDom = trendChartRef.value;
  if (!chartDom) return;

  const myChart = echarts.init(chartDom);

  // 获取当前小时作为最后一个时间点
  const now = new Date();
  const currentHour = now.getHours();
  const timePoints = [];

  for (let i = 0; i < 8; i++) {
    const hour = (currentHour - 7 + i + 24) % 24; // 确保小时值为正数
    timePoints.push(`${String(hour).padStart(2, '0')}:00`);
  }

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['严重', '警告', '一般'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: timePoints,
      axisLine: {
        lineStyle: {
          color: 'inherit' // 使用继承的颜色，适应主题
        }
      },
      axisLabel: {
        color: 'inherit' // 使用继承的颜色，适应主题
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: 'inherit' // 使用继承的颜色，适应主题
        }
      },
      axisLabel: {
        color: 'inherit' // 使用继承的颜色，适应主题
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(127, 127, 127, 0.2)' // 使用半透明颜色，适应主题
        }
      }
    },
    series: [
      {
        name: '严重',
        type: 'line',
        stack: 'Total',
        data: Array.from({ length: 8 }, () => Math.floor(Math.random() * 10) + 5),
        lineStyle: {
          width: 3
        },
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          color: '#ff4d4f'
        }
      },
      {
        name: '警告',
        type: 'line',
        stack: 'Total',
        data: Array.from({ length: 8 }, () => Math.floor(Math.random() * 12) + 10),
        lineStyle: {
          width: 3
        },
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          color: '#faad14'
        }
      },
      {
        name: '一般',
        type: 'line',
        stack: 'Total',
        data: Array.from({ length: 8 }, () => Math.floor(Math.random() * 15) + 12),
        lineStyle: {
          width: 3
        },
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          color: '#1890ff'
        }
      }
    ]
  };

  myChart.setOption(option);

  // 响应窗口大小变化
  window.addEventListener('resize', () => {
    myChart.resize();
  });
};

// 初始化类型分布图表
const initTypeChart = () => {
  const chartDom = typeChartRef.value;
  if (!chartDom) return;

  const myChart = echarts.init(chartDom);

  // 生成随机数据
  const resourceValue = Math.floor(Math.random() * 20) + 30;
  const serviceValue = Math.floor(Math.random() * 15) + 25;
  const performanceValue = Math.floor(Math.random() * 10) + 20;
  const networkValue = Math.floor(Math.random() * 10) + 15;
  const otherValue = Math.floor(Math.random() * 5) + 10;

  const option = {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      textStyle: {
        color: 'inherit' // 使用继承的颜色，适应主题
      }
    },
    series: [
      {
        name: '告警类型',
        type: 'pie',
        radius: '70%',
        data: [
          { value: resourceValue, name: '资源异常' },
          { value: serviceValue, name: '服务不可用' },
          { value: performanceValue, name: '性能下降' },
          { value: networkValue, name: '网络异常' },
          { value: otherValue, name: '其他' }
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        },
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          formatter: '{b}: {c} ({d}%)',
          color: 'inherit' // 使用继承的颜色，适应主题
        }
      }
    ]
  };

  myChart.setOption(option);

  // 响应窗口大小变化
  window.addEventListener('resize', () => {
    myChart.resize();
  });
};

// 初始化根因分析图
const initRootCauseGraph = () => {
  const chartDom = rootCauseGraphRef.value;
  if (!chartDom) return;

  const myChart = echarts.init(chartDom);

  const option = {
    tooltip: {},
    legend: [
      {
        data: ['服务', '组件', '告警点'],
        textStyle: {
          color: 'inherit' // 使用继承的颜色，适应主题
        }
      }
    ],
    series: [
      {
        name: '根因分析',
        type: 'graph',
        layout: 'force',
        data: [
          { name: '支付网关', category: 0, symbolSize: 50, value: 20, itemStyle: { color: '#ff4d4f' } },
          { name: '订单服务', category: 0, symbolSize: 40, value: 15, itemStyle: { color: '#1890ff' } },
          { name: '用户中心', category: 0, symbolSize: 40, value: 15, itemStyle: { color: '#1890ff' } },
          { name: '数据库', category: 1, symbolSize: 30, value: 10, itemStyle: { color: '#52c41a' } },
          { name: '缓存服务', category: 1, symbolSize: 30, value: 10, itemStyle: { color: '#52c41a' } },
          { name: '消息队列', category: 1, symbolSize: 30, value: 10, itemStyle: { color: '#52c41a' } },
          { name: '内存泄漏检测', category: 2, symbolSize: 40, value: 15, itemStyle: { color: '#faad14' } }
        ],
        links: [
          { source: '内存泄漏检测', target: '支付网关', lineStyle: { color: '#ff4d4f', width: 3 } },
          { source: '支付网关', target: '订单服务', lineStyle: { color: '#1890ff', width: 2 } },
          { source: '支付网关', target: '用户中心', lineStyle: { color: '#1890ff', width: 2 } },
          { source: '支付网关', target: '数据库', lineStyle: { color: '#52c41a', width: 2 } },
          { source: '订单服务', target: '缓存服务', lineStyle: { color: '#52c41a', width: 1 } },
          { source: '用户中心', target: '消息队列', lineStyle: { color: '#52c41a', width: 1 } }
        ],
        categories: [
          { name: '服务' },
          { name: '组件' },
          { name: '告警点' }
        ],
        roam: true,
        label: {
          show: true,
          position: 'right',
          formatter: '{b}',
          color: 'inherit' // 使用继承的颜色，适应主题
        },
        force: {
          repulsion: 200,
          edgeLength: 120
        },
        emphasis: {
          focus: 'adjacency',
          lineStyle: {
            width: 5
          }
        }
      }
    ]
  };

  myChart.setOption(option);

  // 响应窗口大小变化
  window.addEventListener('resize', () => {
    myChart.resize();
  });
};

// 组件挂载后初始化图表
onMounted(() => {
  initTrendChart();
  initTypeChart();
});
</script>

<style scoped>
.alarm-container {
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
  display: flex;
  flex-direction: column;
  gap: 20px;
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

.stat-value {
  font-size: 28px;
  font-weight: bold;
  margin: 10px 0;
  color: var(--ant-heading-color);
}

.stat-trend {
  display: flex;
  align-items: center;
  font-size: 14px;
  gap: 5px;
}

.stat-trend.up {
  color: #52c41a;
}

.stat-trend.down {
  color: #ff4d4f;
}

.charts-section {
  margin-bottom: 20px;
}

.chart-card {
  border-radius: 8px;
  overflow: hidden;
  height: 350px;
  transition: all 0.3s ease;
}

.chart-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.chart {
  height: 280px;
}

.root-cause-card {
  border-radius: 8px;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
  position: relative;
  overflow: hidden;
}

.root-cause-card:hover {
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.alarm-table {
  margin-top: 10px;
}

.root-cause-modal {
  color: var(--ant-text-color);
}

.root-cause-content {
  padding: 10px;
}

.root-cause-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.alarm-info h3 {
  margin-bottom: 5px;
  color: var(--ant-heading-color);
}

.graph-container {
  height: 300px;
  border: 1px solid var(--ant-border-color-split);
  border-radius: 4px;
  margin: 10px 0;
}

.timeline-content {
  padding: 10px;
  border-radius: 4px;
  margin-bottom: 10px;
  background-color: var(--ant-background-color-base);
}

.timeline-content h4 {
  margin: 0 0 5px 0;
  color: var(--ant-heading-color);
}

.recommendation {
  margin-top: 20px;
}

/* 添加科技感的发光边框效果 */
.root-cause-card::before {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  z-index: -1;
  border-radius: 10px;
  background: linear-gradient(45deg, #1890ff, #52c41a, #1890ff);
  background-size: 200% 200%;
  animation: glowing 10s linear infinite;
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
}
</style>
