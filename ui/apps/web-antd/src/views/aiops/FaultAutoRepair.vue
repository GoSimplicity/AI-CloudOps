<template>
  <div class="fault-repair-container">
    <div class="header">
      <h1 class="title">智能运维故障自动修复系统</h1>
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
      <!-- 统计卡片 -->
      <div class="stats-cards">
        <a-card class="stat-card repair-card">
          <template #title>
            <tool-outlined /> 自动修复总数
          </template>
          <div class="stat-value">{{ repairStats.total }}</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ repairStats.totalIncrease }}%
          </div>
        </a-card>
        <a-card class="stat-card repair-card">
          <template #title>
            <check-circle-outlined /> 修复成功率
          </template>
          <div class="stat-value">{{ repairStats.successRate }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ repairStats.successRateIncrease }}%
          </div>
        </a-card>
        <a-card class="stat-card repair-card">
          <template #title>
            <clock-circle-outlined /> 平均修复时间
          </template>
          <div class="stat-value">{{ repairStats.avgTime }}</div>
          <div class="stat-trend down">
            <arrow-down-outlined /> {{ repairStats.avgTimeDecrease }}%
          </div>
        </a-card>
        <a-card class="stat-card repair-card">
          <template #title>
            <thunderbolt-outlined /> 自动化程度
          </template>
          <div class="stat-value">{{ repairStats.automationRate }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ repairStats.automationIncrease }}%
          </div>
        </a-card>
      </div>

      <!-- 图表区域 -->
      <div class="charts-container">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-card class="chart-card" title="故障修复趋势">
              <div ref="trendChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card class="chart-card" title="故障类型分布">
              <div ref="typeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
        <a-row :gutter="16" style="margin-top: 16px;">
          <a-col :span="12">
            <a-card class="chart-card" title="修复方法分布">
              <div ref="methodChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card class="chart-card" title="修复时间分布">
              <div ref="timeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
      </div>

      <!-- 最近修复记录 -->
      <a-card class="recent-repairs" title="最近修复记录">
        <a-table :dataSource="repairList" :columns="columns" :loading="loading" :pagination="{ pageSize: 5 }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
            </template>
            <template v-if="column.key === 'action'">
              <a-button type="link" @click="showRepairDetail(record)">详情</a-button>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- 修复详情弹窗 -->
    <a-modal v-model:visible="detailVisible" title="修复详情" width="800px" :footer="null">
      <div v-if="selectedRepair" class="repair-detail">
        <div class="detail-header">
          <h2>{{ selectedRepair.faultName }}</h2>
          <a-tag :color="getStatusColor(selectedRepair.status)">{{ selectedRepair.status }}</a-tag>
        </div>
        <a-descriptions bordered>
          <a-descriptions-item label="故障ID" span="3">{{ selectedRepair.id }}</a-descriptions-item>
          <a-descriptions-item label="故障源" span="3">{{ selectedRepair.source }}</a-descriptions-item>
          <a-descriptions-item label="发生时间" span="3">{{ selectedRepair.faultTime }}</a-descriptions-item>
          <a-descriptions-item label="修复时间" span="3">{{ selectedRepair.repairTime }}</a-descriptions-item>
          <a-descriptions-item label="修复方法" span="3">{{ selectedRepair.method }}</a-descriptions-item>
          <a-descriptions-item label="故障描述" span="3">{{ selectedRepair.description }}</a-descriptions-item>
          <a-descriptions-item label="修复步骤" span="3">
            <div class="repair-steps">
              <div v-for="(step, index) in selectedRepair.steps" :key="index" class="repair-step">
                <div class="step-number">{{ index + 1 }}</div>
                <div class="step-content">
                  <div class="step-title">{{ step.title }}</div>
                  <div class="step-desc">{{ step.description }}</div>
                  <div class="step-result" :class="step.success ? 'success' : 'failed'">
                    {{ step.success ? '成功' : '失败' }}
                  </div>
                </div>
              </div>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="修复结果" span="3">{{ selectedRepair.result }}</a-descriptions-item>
        </a-descriptions>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, nextTick } from 'vue';
import {
  SyncOutlined,
  ToolOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined
} from '@ant-design/icons-vue';
import * as echarts from 'echarts';

// 时间范围选择
const timeRange = ref('24h');

// 获取当前日期
const currentDate = new Date();
const formatDate = (date: Date) => {
  return date.toISOString().split('T')[0];
};
const formatDateTime = (date: Date) => {
  return `${formatDate(date)} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
};

// 统计数据
const repairStats = reactive({
  total: 143,
  totalIncrease: 8,
  successRate: 94.2,
  successRateIncrease: 2.1,
  avgTime: '2.3分钟',
  avgTimeDecrease: 12,
  automationRate: 89,
  automationIncrease: 4
});

// 图表引用
const trendChartRef = ref(null);
const typeChartRef = ref(null);
const methodChartRef = ref(null);
const timeChartRef = ref(null);

// 表格列定义
const columns = [
  { title: '故障ID', dataIndex: 'id', key: 'id' },
  { title: '故障名称', dataIndex: 'faultName', key: 'faultName' },
  { title: '故障源', dataIndex: 'source', key: 'source' },
  { title: '修复方法', dataIndex: 'method', key: 'method' },
  { title: '修复时间', dataIndex: 'repairTime', key: 'repairTime' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '操作', key: 'action' }
];

// 生成过去N小时的时间
const getTimeAgo = (hoursAgo: number) => {
  const date = new Date();
  date.setHours(date.getHours() - hoursAgo);
  return date;
};

// 修复记录列表
const loading = ref(false);
const repairList = ref([
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-001`,
    faultName: '数据库连接池耗尽',
    source: '数据库服务器-DB01',
    method: '自动扩容连接池',
    faultTime: formatDateTime(getTimeAgo(2)),
    repairTime: formatDateTime(getTimeAgo(1.95)),
    status: '修复成功',
    description: '数据库连接池达到最大值，新连接请求被拒绝',
    steps: [
      { title: '检测连接池状态', description: '监控到连接池使用率达到95%', success: true },
      { title: '分析连接使用情况', description: '识别到异常连接请求模式', success: true },
      { title: '动态调整连接池大小', description: '将最大连接数从100增加到150', success: true },
      { title: '优化连接超时设置', description: '将空闲连接超时时间从30分钟调整为10分钟', success: true }
    ],
    result: '连接池扩容成功，系统恢复正常运行，响应时间从2.5秒降低到0.8秒'
  },
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-002`,
    faultName: 'CPU使用率过高',
    source: '应用服务器-APP03',
    method: '自动识别并终止异常进程',
    faultTime: formatDateTime(getTimeAgo(5)),
    repairTime: formatDateTime(getTimeAgo(4.9)),
    status: '修复成功',
    description: '服务器CPU使用率持续超过95%达10分钟',
    steps: [
      { title: '检测系统资源', description: '监控到CPU使用率达到98%', success: true },
      { title: '分析进程占用', description: '识别到异常进程ID 12345占用CPU 80%', success: true },
      { title: '分析进程行为', description: '确认为陷入死循环的后台任务', success: true },
      { title: '终止异常进程', description: '安全终止进程ID 12345', success: true },
      { title: '重启相关服务', description: '以正常模式重启后台任务服务', success: true }
    ],
    result: 'CPU使用率恢复正常，从98%降低到35%，系统响应时间恢复正常'
  },
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-003`,
    faultName: '磁盘空间不足',
    source: '存储服务器-STO02',
    method: '自动清理临时文件',
    faultTime: formatDateTime(getTimeAgo(8)),
    repairTime: formatDateTime(getTimeAgo(7.9)),
    status: '修复成功',
    description: '磁盘使用率达到95%，接近警戒线',
    steps: [
      { title: '检测磁盘空间', description: '监控到磁盘使用率达到95%', success: true },
      { title: '分析磁盘占用', description: '识别到/tmp目录占用异常', success: true },
      { title: '清理临时文件', description: '清理超过7天的临时文件', success: true },
      { title: '压缩日志文件', description: '压缩超过30天的日志文件', success: true }
    ],
    result: '磁盘使用率从95%降低到65%，释放了30GB空间'
  },
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-004`,
    faultName: '网络连接超时',
    source: '网络设备-NET01',
    method: '自动重置网络连接',
    faultTime: formatDateTime(getTimeAgo(12)),
    repairTime: formatDateTime(getTimeAgo(11.95)),
    status: '修复成功',
    description: '网络连接超时率超过10%',
    steps: [
      { title: '检测网络状态', description: '监控到网络连接超时率达到15%', success: true },
      { title: '分析网络流量', description: '识别到异常流量模式', success: true },
      { title: '重置网络连接', description: '重置所有空闲连接', success: true },
      { title: '优化路由表', description: '更新路由表配置', success: true }
    ],
    result: '网络连接超时率从15%降低到0.5%，网络响应时间从200ms降低到50ms'
  },
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-005`,
    faultName: '内存泄漏',
    source: '应用服务器-APP01',
    method: '自动重启应用实例',
    faultTime: formatDateTime(getTimeAgo(15)),
    repairTime: formatDateTime(getTimeAgo(14.9)),
    status: '修复成功',
    description: '应用内存使用持续增长不释放',
    steps: [
      { title: '检测内存使用', description: '监控到内存使用率持续增长', success: true },
      { title: '分析内存占用', description: '确认为应用实例内存泄漏', success: true },
      { title: '创建新应用实例', description: '启动新的应用实例', success: true },
      { title: '切换流量', description: '将流量切换到新实例', success: true },
      { title: '关闭异常实例', description: '安全关闭存在内存泄漏的实例', success: true }
    ],
    result: '应用内存使用恢复正常，从92%降低到45%，系统响应时间从3秒降低到0.5秒'
  },
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-006`,
    faultName: 'API响应超时',
    source: '微服务-SVC03',
    method: '自动扩容服务实例',
    faultTime: formatDateTime(getTimeAgo(18)),
    repairTime: formatDateTime(getTimeAgo(17.95)),
    status: '修复中',
    description: 'API平均响应时间超过2秒',
    steps: [
      { title: '检测API响应时间', description: '监控到API响应时间达到2.5秒', success: true },
      { title: '分析服务负载', description: '确认为服务实例负载过高', success: true },
      { title: '自动扩容服务', description: '增加3个服务实例', success: true },
      { title: '优化负载均衡', description: '更新负载均衡策略', success: false }
    ],
    result: '服务扩容完成，负载均衡配置更新中'
  },
  {
    id: `FLT-${(formatDate(currentDate) || '').replace(/-/g, '')}-007`,
    faultName: '缓存服务异常',
    source: '缓存服务器-CACHE01',
    method: '自动重建缓存索引',
    faultTime: formatDateTime(getTimeAgo(22)),
    repairTime: formatDateTime(getTimeAgo(21.9)),
    status: '修复成功',
    description: '缓存命中率下降到30%以下',
    steps: [
      { title: '检测缓存性能', description: '监控到缓存命中率降至25%', success: true },
      { title: '分析缓存状态', description: '确认为缓存索引损坏', success: true },
      { title: '备份当前数据', description: '创建缓存数据快照', success: true },
      { title: '重建缓存索引', description: '重新构建缓存索引结构', success: true },
      { title: '验证缓存性能', description: '测试缓存响应时间和命中率', success: true }
    ],
    result: '缓存命中率从25%提升到95%，平均响应时间从150ms降低到15ms'
  }
]);

// 详情弹窗
const detailVisible = ref(false);
const selectedRepair = ref<any>(null);

// 显示修复详情
const showRepairDetail = (repair: any) => {
  selectedRepair.value = repair;
  detailVisible.value = true;
};

// 获取状态颜色
const getStatusColor = (status: string) => {
  switch (status) {
    case '修复成功':
      return 'success';
    case '修复中':
      return 'processing';
    case '修复失败':
      return 'error';
    default:
      return 'default';
  }
};

// 刷新数据
const refreshData = () => {
  loading.value = true;
  setTimeout(() => {
    loading.value = false;
  }, 1000);
};

// 获取最近6个月的名称
const getRecentMonths = () => {
  const months = [];
  const currentMonth = currentDate.getMonth();

  for (let i = 5; i >= 0; i--) {
    const month = (currentMonth - i + 12) % 12;
    months.push(`${month + 1}月`);
  }

  return months;
};

// 初始化图表
const initCharts = () => {
  // 故障修复趋势图
  const trendChart = echarts.init(trendChartRef.value);
  const months = getRecentMonths();

  trendChart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['故障数', '自动修复数', '修复成功率'],
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
      data: months,
      axisLine: {
        lineStyle: {
          color: '#333333'  // 修改为深色
        }
      }
    },
    yAxis: [
      {
        type: 'value',
        name: '数量',
        axisLine: {
          lineStyle: {
            color: '#333333'  // 修改为深色
          }
        },
        splitLine: {
          lineStyle: {
            color: 'rgba(0, 0, 0, 0.1)'  // 修改为深色
          }
        }
      },
      {
        type: 'value',
        name: '成功率',
        min: 0,
        max: 100,
        interval: 20,
        axisLabel: {
          formatter: '{value}%',
          color: '#333333'  // 修改为深色
        },
        axisLine: {
          lineStyle: {
            color: '#333333'  // 修改为深色
          }
        },
        splitLine: {
          show: false
        }
      }
    ],
    series: [
      {
        name: '故障数',
        type: 'bar',
        data: [32, 38, 45, 51, 57, 63],
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#ff9a9e' },
            { offset: 1, color: '#fad0c4' }
          ])
        }
      },
      {
        name: '自动修复数',
        type: 'bar',
        data: [25, 32, 39, 45, 50, 56],
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#a1c4fd' },
            { offset: 1, color: '#c2e9fb' }
          ])
        }
      },
      {
        name: '修复成功率',
        type: 'line',
        yAxisIndex: 1,
        data: [78.1, 84.2, 86.7, 88.2, 87.7, 94.2],
        lineStyle: {
          width: 3,
          color: '#00f2fe'
        },
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          color: '#00f2fe'
        }
      }
    ]
  });

  // 故障类型分布图
  const typeChart = echarts.init(typeChartRef.value);
  typeChart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: ['数据库故障', '网络故障', '应用故障', '系统故障', '存储故障'],
      textStyle: {
        color: '#333333'  // 修改为深色字体
      }
    },
    series: [
      {
        name: '故障类型',
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
          { value: 32, name: '数据库故障', itemStyle: { color: '#ff9a9e' } },
          { value: 25, name: '网络故障', itemStyle: { color: '#a1c4fd' } },
          { value: 38, name: '应用故障', itemStyle: { color: '#d4fc79' } },
          { value: 19, name: '系统故障', itemStyle: { color: '#fbc2eb' } },
          { value: 14, name: '存储故障', itemStyle: { color: '#84fab0' } }
        ]
      }
    ]
  });

  // 修复方法分布图
  const methodChart = echarts.init(methodChartRef.value);
  methodChart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: ['自动重启', '资源扩容', '配置调整', '清理操作', '其他方法'],
      textStyle: {
        color: '#333333'  // 修改为深色字体
      }
    },
    series: [
      {
        name: '修复方法',
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
          { value: 41, name: '自动重启', itemStyle: { color: '#00f2fe' } },
          { value: 29, name: '资源扩容', itemStyle: { color: '#4facfe' } },
          { value: 25, name: '配置调整', itemStyle: { color: '#0ba360' } },
          { value: 18, name: '清理操作', itemStyle: { color: '#f093fb' } },
          { value: 12, name: '其他方法', itemStyle: { color: '#f6d365' } }
        ]
      }
    ]
  });

  // 修复时间分布图
  const timeChart = echarts.init(timeChartRef.value);
  timeChart.setOption({
    backgroundColor: 'transparent',
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
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: ['<1分钟', '1-2分钟', '2-5分钟', '5-10分钟', '>10分钟'],
      axisLine: {
        lineStyle: {
          color: '#333333'  // 修改为深色
        }
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: '#333333'  // 修改为深色
        }
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(0, 0, 0, 0.1)'  // 修改为深色
        }
      }
    },
    series: [
      {
        name: '修复时间分布',
        type: 'bar',
        data: [38, 59, 27, 14, 5],
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#43e97b' },
            { offset: 1, color: '#38f9d7' }
          ])
        }
      }
    ]
  });

  // 窗口大小变化时重绘图表
  window.addEventListener('resize', () => {
    trendChart.resize();
    typeChart.resize();
    methodChart.resize();
    timeChart.resize();
  });
};

onMounted(() => {
  nextTick(() => {
    initCharts();
  });
});
</script>

<style scoped>
.fault-repair-container {
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
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
  gap: 10px;
}

.time-selector {
  border-color: var(--ant-border-color-base);
}

.refresh-btn {
  display: flex;
  align-items: center;
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
}

.stat-card {
  border-radius: 10px;
  overflow: hidden;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  margin-bottom: 10px;
  color: var(--ant-heading-color);
}

.stat-trend {
  display: flex;
  align-items: center;
  font-size: 14px;
  gap: 5px;
}

.up {
  color: var(--ant-success-color);
}

.down {
  color: var(--ant-error-color);
}

.charts-container {
  margin-top: 20px;
}

.chart-card {
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--ant-border-color-split);
}

.chart {
  height: 300px;
}

.recent-repairs {
  margin-top: 20px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--ant-border-color-split);
}

.repair-detail {
  color: var(--ant-text-color);
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.repair-steps {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.repair-step {
  background-color: var(--ant-background-color-base);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid var(--ant-border-color-split);
  transition: all 0.3s ease;
}

.repair-step:hover {
  transform: translateX(5px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.step-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--ant-heading-color);
}

.step-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 14px;
}

.step-status.success {
  background-color: rgba(var(--ant-success-color-rgb), 0.1);
  color: var(--ant-success-color);
}

.step-status.failed {
  background-color: rgba(var(--ant-error-color-rgb), 0.1);
  color: var(--ant-error-color);
}

.step-status.running {
  background-color: rgba(var(--ant-primary-color-rgb), 0.1);
  color: var(--ant-primary-color);
}

.step-content {
  color: var(--ant-text-color-secondary);
  font-size: 14px;
  line-height: 1.5;
}

.step-time {
  color: var(--ant-text-color-quaternary);
  font-size: 12px;
  margin-top: 8px;
}
</style>
