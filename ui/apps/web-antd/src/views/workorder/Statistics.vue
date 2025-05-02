<template>
  <div class="statistics-container">
    <div class="page-header">
      <div class="header-actions">
        <a-range-picker v-model:value="dateRange" @change="handleDateRangeChange" :ranges="dateRanges"
          style="width: 300px" />
        <a-select v-model:value="processFilter" placeholder="选择流程" style="width: 200px" @change="handleProcessChange">
          <a-select-option :value="null">全部流程</a-select-option>
          <a-select-option v-for="process in processes" :key="process.id" :value="process.id">
            {{ process.name }}
          </a-select-option>
        </a-select>
        <a-button @click="refreshData">
          <template #icon>
            <ReloadOutlined />
          </template>
          刷新数据
        </a-button>
      </div>
    </div>

    <div class="dashboard-container">
      <!-- 概览统计卡片 -->
      <a-row :gutter="16" class="stats-row">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="总工单数" :value="overviewStats.total" :value-style="{ color: '#40a9ff' }">
              <template #prefix>
                <FileDoneOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.totalChange >= 0 ? 'increase' : 'decrease'">
                <ArrowUpOutlined v-if="overviewStats.totalChange >= 0" />
                <ArrowDownOutlined v-else />
                {{ Math.abs(overviewStats.totalChange) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已完成" :value="overviewStats.completed" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.completedChange >= 0 ? 'increase' : 'decrease'">
                <ArrowUpOutlined v-if="overviewStats.completedChange >= 0" />
                <ArrowDownOutlined v-else />
                {{ Math.abs(overviewStats.completedChange) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="处理中" :value="overviewStats.inProgress" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <SyncOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.inProgressChange >= 0 ? 'increase' : 'decrease'">
                <ArrowUpOutlined v-if="overviewStats.inProgressChange >= 0" />
                <ArrowDownOutlined v-else />
                {{ Math.abs(overviewStats.inProgressChange) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="平均处理时间" :value="overviewStats.avgProcessTime" suffix="小时"
              :value-style="{ color: '#722ed1' }">
              <template #prefix>
                <ClockCircleOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.avgProcessTimeChange <= 0 ? 'increase' : 'decrease'">
                <ArrowDownOutlined v-if="overviewStats.avgProcessTimeChange <= 0" />
                <ArrowUpOutlined v-else />
                {{ Math.abs(overviewStats.avgProcessTimeChange) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 工单趋势图和状态分布 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="16">
          <a-card title="工单趋势" class="chart-card">
            <div class="trend-chart-container" ref="trendChartRef"></div>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card title="工单状态分布" class="chart-card">
            <div class="status-chart-container" ref="statusChartRef"></div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 流程使用统计和部门分布 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="12">
          <a-card title="流程使用统计" class="chart-card">
            <div class="process-chart-container" ref="processChartRef"></div>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="部门工单分布" class="chart-card">
            <div class="department-chart-container" ref="departmentChartRef"></div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 审批效率分析 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="24">
          <a-card title="审批效率分析" class="chart-card">
            <div class="efficiency-chart-container" ref="efficiencyChartRef"></div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 处理人排行榜 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="12">
          <a-card title="处理人排行榜 - 处理工单数" class="chart-card">
            <a-table :dataSource="handlerRankingByCount" :pagination="false" :columns="rankColumns" size="small">
              <template #bodyCell="{ column, record, index }">
                <template v-if="column.key === 'rank'">
                  <div class="rank-cell">
                    <a-tag :color="getRankColor(index)">第 {{ index + 1 }} 名</a-tag>
                  </div>
                </template>
                <template v-if="column.key === 'user'">
                  <div class="user-cell">
                    <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.name) }">
                      {{ getInitials(record.name) }}
                    </a-avatar>
                    <span>{{ record.name }}</span>
                  </div>
                </template>
                <template v-if="column.key === 'count'">
                  <a-progress :percent="getPercentage(record.count, handlerRankingByCount[0]?.count || 0)"
                    :show-info="false" status="active" :stroke-color="getProgressColor(index)" />
                  <span class="count-value">{{ record.count }}</span>
                </template>
              </template>
            </a-table>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="处理人排行榜 - 平均处理时间" class="chart-card">
            <a-table :dataSource="handlerRankingByTime" :pagination="false" :columns="timeRankColumns" size="small">
              <template #bodyCell="{ column, record, index }">
                <template v-if="column.key === 'rank'">
                  <div class="rank-cell">
                    <a-tag :color="getRankColor(index, true)">第 {{ index + 1 }} 名</a-tag>
                  </div>
                </template>
                <template v-if="column.key === 'user'">
                  <div class="user-cell">
                    <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.name) }">
                      {{ getInitials(record.name) }}
                    </a-avatar>
                    <span>{{ record.name }}</span>
                  </div>
                </template>
                <template v-if="column.key === 'time'">
                  <a-progress
                    :percent="getPercentage(handlerRankingByTime[handlerRankingByTime.length - 1]?.time || 0, record.time)"
                    :show-info="false" status="active" :stroke-color="getProgressColor(index, true)" />
                  <span class="time-value">{{ record.time }}小时</span>
                </template>
              </template>
            </a-table>
          </a-card>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick, onBeforeUnmount, watch } from 'vue';
import { message } from 'ant-design-vue';
import * as echarts from 'echarts';
import {
  FileDoneOutlined,
  CheckCircleOutlined,
  SyncOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined
} from '@ant-design/icons-vue';
import dayjs from 'dayjs';

// 图表引用
const trendChartRef = ref<HTMLElement | null>(null);
const statusChartRef = ref<HTMLElement | null>(null);
const processChartRef = ref<HTMLElement | null>(null);
const departmentChartRef = ref<HTMLElement | null>(null);
const efficiencyChartRef = ref<HTMLElement | null>(null);

// 图表实例
let trendChart: echarts.ECharts | null = null;
let statusChart: echarts.ECharts | null = null;
let processChart: echarts.ECharts | null = null;
let departmentChart: echarts.ECharts | null = null;
let efficiencyChart: echarts.ECharts | null = null;

// 数据和过滤相关
const loading = ref(false);
const processFilter = ref(null);
const dateRange = ref([dayjs().subtract(30, 'days'), dayjs()]);
const dateRanges = {
  '最近7天': [dayjs().subtract(7, 'days'), dayjs()],
  '最近30天': [dayjs().subtract(30, 'days'), dayjs()],
  '最近90天': [dayjs().subtract(90, 'days'), dayjs()],
  '今年': [dayjs().startOf('year'), dayjs()]
};

// 模拟流程数据
const processes = ref([
  { id: 1, name: '员工入职审批流程' },
  { id: 2, name: '休假申请流程' },
  { id: 3, name: 'IT设备申请流程' },
  { id: 4, name: '报销审批流程' },
  { id: 5, name: '项目立项流程' }
]);

// 概览统计数据
const overviewStats = reactive({
  total: 458,
  totalChange: 12.5,
  completed: 347,
  completedChange: 15.2,
  inProgress: 111,
  inProgressChange: -8.3,
  avgProcessTime: 36.4,
  avgProcessTimeChange: -5.7
});

// 排行榜数据
const handlerRankingByCount = ref([
  { name: '张三', department: '人力资源部', count: 86 },
  { name: '李四', department: '财务部', count: 73 },
  { name: '王五', department: 'IT部门', count: 65 },
  { name: '赵六', department: '营销部', count: 54 },
  { name: '钱七', department: '人力资源部', count: 47 }
]);

const handlerRankingByTime = ref([
  { name: '刘八', department: 'IT部门', time: 12.5 },
  { name: '孙九', department: '财务部', time: 15.3 },
  { name: '周十', department: '人力资源部', time: 18.7 },
  { name: '吴十一', department: '行政部', time: 22.4 },
  { name: '郑十二', department: '销售部', time: 24.8 }
]);

// 表格列定义
const rankColumns = [
  { title: '排名', key: 'rank', width: '15%' },
  { title: '处理人', key: 'user', dataIndex: 'name', width: '25%' },
  { title: '部门', dataIndex: 'department', width: '25%' },
  { title: '处理工单数', key: 'count', width: '35%' }
];

const timeRankColumns = [
  { title: '排名', key: 'rank', width: '15%' },
  { title: '处理人', key: 'user', dataIndex: 'name', width: '25%' },
  { title: '部门', dataIndex: 'department', width: '25%' },
  { title: '平均处理时间', key: 'time', width: '35%' }
];

// 模拟工单趋势数据
const trendData = {
  dates: [
    '2025-03-01', '2025-03-05', '2025-03-10', '2025-03-15',
    '2025-03-20', '2025-03-25', '2025-03-31', '2025-04-05'
  ],
  created: [42, 65, 53, 78, 62, 58, 70, 65],
  completed: [35, 52, 48, 64, 56, 42, 65, 60]
};

// 模拟工单状态分布数据
const statusData = [
  { value: 347, name: '已完成' },
  { value: 62, name: '处理中' },
  { value: 28, name: '等待审批' },
  { value: 15, name: '已退回' },
  { value: 6, name: '已取消' }
];

// 模拟流程使用统计数据
const processUsageData = [
  { name: '员工入职审批流程', value: 125 },
  { name: '休假申请流程', value: 98 },
  { name: 'IT设备申请流程', value: 67 },
  { name: '报销审批流程', value: 108 },
  { name: '项目立项流程', value: 60 }
];

// 模拟部门分布数据
const departmentData = [
  { name: '人力资源部', value: 120 },
  { name: '财务部', value: 92 },
  { name: 'IT部门', value: 85 },
  { name: '市场部', value: 78 },
  { name: '销售部', value: 66 },
  { name: '行政部', value: 45 }
];

// 模拟审批效率数据
const efficiencyData = {
  processes: ['员工入职审批流程', '休假申请流程', 'IT设备申请流程', '报销审批流程', '项目立项流程'],
  avgTimes: [42, 24, 36, 48, 56] // 平均完成时间（小时）
};

// 方法
const handleDateRangeChange = (dates: any) => {
  if (dates && dates.length === 2) {
    // 这里可以处理日期范围变更
    refreshData();
  }
};

const handleProcessChange = () => {
  refreshData();
};

const refreshData = () => {
  loading.value = true;
  // 模拟API请求
  setTimeout(() => {
    initCharts();
    loading.value = false;
    message.success('数据已刷新');
  }, 800);
};

// 初始化所有图表
const initCharts = () => {
  nextTick(() => {
    initTrendChart();
    initStatusChart();
    initProcessChart();
    initDepartmentChart();
    initEfficiencyChart();
  });
};

// 工单趋势图表
const initTrendChart = () => {
  if (!trendChartRef.value) return;

  if (trendChart) {
    trendChart.dispose();
  }

  trendChart = echarts.init(trendChartRef.value);
  const option = {
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: ['创建工单', '完成工单'],
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '10%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: trendData.dates
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '创建工单',
        type: 'line',
        data: trendData.created,
        smooth: true,
        lineStyle: {
          width: 3,
          color: '#40a9ff'
        },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64, 169, 255, 0.5)' },
            { offset: 1, color: 'rgba(64, 169, 255, 0.1)' }
          ])
        }
      },
      {
        name: '完成工单',
        type: 'line',
        data: trendData.completed,
        smooth: true,
        lineStyle: {
          width: 3,
          color: '#52c41a'
        },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(82, 196, 26, 0.5)' },
            { offset: 1, color: 'rgba(82, 196, 26, 0.1)' }
          ])
        }
      }
    ]
  };

  trendChart.setOption(option);
};

// 工单状态分布图表
const initStatusChart = () => {
  if (!statusChartRef.value) return;

  if (statusChart) {
    statusChart.dispose();
  }

  statusChart = echarts.init(statusChartRef.value);
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: statusData.map(item => item.name)
    },
    series: [
      {
        name: '工单状态',
        type: 'pie',
        radius: ['50%', '70%'],
        center: ['40%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false
        },
        emphasis: {
          label: {
            show: true,
            fontSize: '16',
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: statusData,
        color: ['#52c41a', '#faad14', '#1890ff', '#f5222d', '#bfbfbf']
      }
    ]
  };

  statusChart.setOption(option);
};

// 流程使用统计图表
const initProcessChart = () => {
  if (!processChartRef.value) return;

  if (processChart) {
    processChart.dispose();
  }

  processChart = echarts.init(processChartRef.value);
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value',
      boundaryGap: [0, 0.01]
    },
    yAxis: {
      type: 'category',
      data: processUsageData.map(item => item.name),
      inverse: true
    },
    series: [
      {
        name: '工单数',
        type: 'bar',
        data: processUsageData.map(item => ({
          value: item.value,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: '#1890ff' },
              { offset: 1, color: '#69c0ff' }
            ])
          }
        })),
        showBackground: true,
        backgroundStyle: {
          color: 'rgba(180, 180, 180, 0.1)'
        }
      }
    ]
  };

  processChart.setOption(option);
};

// 部门工单分布图表
const initDepartmentChart = () => {
  if (!departmentChartRef.value) return;

  if (departmentChart) {
    departmentChart.dispose();
  }

  departmentChart = echarts.init(departmentChartRef.value);
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      type: 'scroll',
      orient: 'horizontal',
      bottom: 0,
      data: departmentData.map(item => item.name)
    },
    series: [
      {
        type: 'pie',
        radius: ['35%', '60%'],
        center: ['50%', '45%'],
        avoidLabelOverlap: false,
        label: {
          show: true,
          formatter: '{b}: {c}'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: '16',
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: true
        },
        data: departmentData,
        color: ['#1890ff', '#52c41a', '#faad14', '#722ed1', '#13c2c2', '#eb2f96']
      }
    ]
  };

  departmentChart.setOption(option);
};

// 审批效率分析图表
const initEfficiencyChart = () => {
  if (!efficiencyChartRef.value) return;

  if (efficiencyChart) {
    efficiencyChart.dispose();
  }

  efficiencyChart = echarts.init(efficiencyChartRef.value);
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      formatter: '{b}: {c} 小时'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: efficiencyData.processes,
      axisLabel: {
        interval: 0,
        rotate: 30
      }
    },
    yAxis: {
      type: 'value',
      name: '平均处理时间（小时）'
    },
    series: [
      {
        name: '平均处理时间',
        type: 'bar',
        barWidth: '40%',
        data: efficiencyData.avgTimes.map((value, index) => ({
          value,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#1890ff' },
              { offset: 1, color: '#69c0ff' }
            ])
          }
        })),
        label: {
          show: true,
          position: 'top',
          formatter: '{c} 小时'
        }
      }
    ]
  };

  efficiencyChart.setOption(option);
};

// 辅助方法
const getInitials = (name: string) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getAvatarColor = (name: string) => {
  // 根据名称生成一致的颜色
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];

  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length];
};

const getRankColor = (index: number, isTime: boolean = false) => {
  if (isTime) {
    // 对于处理时间，索引越小表示效率越高（时间越短）
    const colors = ['#52c41a', '#85ce61', '#b3e19d', '#e6a23c', '#f56c6c'];
    return colors[Math.min(index, colors.length - 1)];
  } else {
    // 对于处理数量，索引越小表示数量越多
    const colors = ['#f56c6c', '#e6a23c', '#85ce61', '#52c41a', '#409eff'];
    return colors[Math.min(index, colors.length - 1)];
  }
};

const getProgressColor = (index: number, isTime: boolean = false) => {
  if (isTime) {
    // 处理时间短的颜色更好看
    const colors = ['#52c41a', '#7ec050', '#b9de7c', '#faad14', '#f56c6c'];
    return colors[Math.min(index, colors.length - 1)];
  } else {
    // 处理数量多的颜色更好看
    const colors = ['#1890ff', '#40a9ff', '#69c0ff', '#91d5ff', '#bae7ff'];
    return colors[Math.min(index, colors.length - 1)];
  }
};

const getPercentage = (value: number, max: number) => {
  return Math.round((value / max) * 100);
};

// 响应窗口大小变化
const handleResize = () => {
  trendChart?.resize();
  statusChart?.resize();
  processChart?.resize();
  departmentChart?.resize();
  efficiencyChart?.resize();
};

// 生命周期钩子
onMounted(() => {
  refreshData();
  window.addEventListener('resize', handleResize);
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize);
  trendChart?.dispose();
  statusChart?.dispose();
  processChart?.dispose();
  departmentChart?.dispose();
  efficiencyChart?.dispose();
});

// 监听过滤条件变化
watch([processFilter], () => {
  refreshData();
});
</script>

<style scoped>
.statistics-container {
  padding: 24px;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  color: #1f2937;
  margin: 0;
  background: linear-gradient(90deg, #1890ff 0%, #52c41a 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  font-weight: 700;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.dashboard-container {
  padding: 16px 0;
}

.stats-row {
  margin-bottom: 24px;
}

.stats-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  height: 100%;
  position: relative;
}

.stat-change {
  position: absolute;
  bottom: 16px;
  right: 24px;
  font-size: 12px;
  color: #8c8c8c;
  display: flex;
  align-items: center;
  gap: 4px;
}

.increase {
  color: #52c41a;
  display: flex;
  align-items: center;
}

.decrease {
  color: #f5222d;
  display: flex;
  align-items: center;
}

.chart-row {
  margin-bottom: 24px;
}

.chart-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  height: 100%;
}

.trend-chart-container,
.status-chart-container,
.process-chart-container,
.department-chart-container,
.efficiency-chart-container {
  width: 100%;
  height: 350px;
}

.rank-cell {
  text-align: center;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.count-value,
.time-value {
  margin-left: 8px;
  font-weight: bold;
}
</style>
