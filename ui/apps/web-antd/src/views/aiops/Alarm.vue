<template>
  <div class="alarm-container">
    <div class="header">
      <h1 class="title">智能运维告警分析平台</h1>
      <div class="actions">
        <a-select v-model:value="timeRange" style="width: 150px" class="time-selector" @change="refreshData">
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
        <a-card class="stat-card">
          <template #title>
            <alert-outlined /> 告警总数
          </template>
          <div class="stat-value">{{ alarmStats.total }}</div>
          <div class="stat-trend" :class="{ up: alarmStats.totalTrend > 0, down: alarmStats.totalTrend < 0 }">
            <arrow-up-outlined v-if="alarmStats.totalTrend > 0" />
            <arrow-down-outlined v-else />
            {{ Math.abs(alarmStats.totalTrend) }}%
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <warning-outlined /> 严重告警
          </template>
          <div class="stat-value">{{ alarmStats.critical }}</div>
          <div class="stat-trend" :class="{ up: alarmStats.criticalTrend > 0, down: alarmStats.criticalTrend < 0 }">
            <arrow-up-outlined v-if="alarmStats.criticalTrend > 0" />
            <arrow-down-outlined v-else />
            {{ Math.abs(alarmStats.criticalTrend) }}%
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <check-circle-outlined /> 已解决
          </template>
          <div class="stat-value">{{ alarmStats.resolved }}</div>
          <div class="stat-trend" :class="{ up: alarmStats.resolvedTrend > 0, down: alarmStats.resolvedTrend < 0 }">
            <arrow-up-outlined v-if="alarmStats.resolvedTrend > 0" />
            <arrow-down-outlined v-else />
            {{ Math.abs(alarmStats.resolvedTrend) }}%
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <clock-circle-outlined /> 平均解决时间
          </template>
          <div class="stat-value">{{ alarmStats.avgResolveTime }}</div>
          <div class="stat-trend" :class="{ up: alarmStats.avgTimeTrend > 0, down: alarmStats.avgTimeTrend < 0 }">
            <arrow-up-outlined v-if="alarmStats.avgTimeTrend > 0" />
            <arrow-down-outlined v-else />
            {{ Math.abs(alarmStats.avgTimeTrend) }}%
          </div>
        </a-card>
      </div>

      <!-- 图表区域 -->
      <div class="charts-container">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-card class="chart-card" title="告警趋势分析">
              <div ref="trendChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card class="chart-card" title="告警类型分布">
              <div ref="typeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
        <a-row :gutter="16" style="margin-top: 16px;">
          <a-col :span="12">
            <a-card class="chart-card" title="告警来源分布">
              <div ref="sourceChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card class="chart-card" title="告警解决时间分布">
              <div ref="timeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
      </div>

      <!-- 告警列表 -->
      <a-card class="alarm-list-card" title="最近告警列表">
        <a-table :dataSource="alarmList" :columns="columns" :pagination="{ pageSize: 5 }" :loading="loading">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'level'">
              <a-tag :color="getAlarmLevelColor(record.level)">{{ record.level }}</a-tag>
            </template>
            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === '已解决' ? 'success' : record.status === '处理中' ? 'processing' : 'error'">
                {{ record.status }}
              </a-tag>
            </template>
            <template v-if="column.key === 'action'">
              <a-button type="link" @click="viewAlarmDetail(record)">查看详情</a-button>
              <a-button type="link" @click="handleAlarm(record)" v-if="record.status !== '已解决'">处理</a-button>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- 告警详情弹窗 -->
    <a-modal v-model:visible="detailModalVisible" title="告警详情" width="700px" :footer="null">
      <div v-if="selectedAlarm" class="alarm-detail">
        <div class="detail-header">
          <h2>{{ selectedAlarm.title }}</h2>
          <a-tag :color="getAlarmLevelColor(selectedAlarm.level)">{{ selectedAlarm.level }}</a-tag>
        </div>
        <a-descriptions bordered>
          <a-descriptions-item label="告警ID" span="3">{{ selectedAlarm.id }}</a-descriptions-item>
          <a-descriptions-item label="告警源" span="3">{{ selectedAlarm.source }}</a-descriptions-item>
          <a-descriptions-item label="发生时间" span="3">{{ selectedAlarm.time }}</a-descriptions-item>
          <a-descriptions-item label="状态" span="3">
            <a-tag
              :color="selectedAlarm.status === '已解决' ? 'success' : selectedAlarm.status === '处理中' ? 'processing' : 'error'">
              {{ selectedAlarm.status }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="告警内容" span="3">{{ selectedAlarm.content }}</a-descriptions-item>
          <a-descriptions-item label="可能原因" span="3">{{ selectedAlarm.possibleCause }}</a-descriptions-item>
          <a-descriptions-item label="建议解决方案" span="3">{{ selectedAlarm.solution }}</a-descriptions-item>
        </a-descriptions>
        <div class="detail-actions" v-if="selectedAlarm.status !== '已解决'">
          <a-button type="primary" @click="handleAlarm(selectedAlarm)">处理告警</a-button>
        </div>
      </div>
    </a-modal>

    <!-- 操作结果反馈 -->
    <a-message></a-message>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, nextTick } from 'vue';
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
import { message } from 'ant-design-vue';

// 定义告警记录类型
interface AlarmRecord {
  id: string;
  title: string;
  level: string;
  source: string;
  time: string;
  status: string;
  content: string;
  possibleCause: string;
  solution: string;
}

// 时间范围选择
const timeRange = ref('24h');

// 获取当前日期和时间
const now = new Date();
const formatDate = (date: Date): string => {
  return date.toISOString().slice(0, 19).replace('T', ' ');
};

// 生成过去N小时的时间
const getTimeAgo = (hours: number): string => {
  const date = new Date(now.getTime() - hours * 60 * 60 * 1000);
  return formatDate(date);
};

// 统计数据 - 使用更真实的数据
const alarmStats = reactive({
  total: 37,
  totalTrend: -5,
  critical: 8,
  criticalTrend: 3,
  resolved: 24,
  resolvedTrend: 7,
  avgResolveTime: '1.5小时',
  avgTimeTrend: -12
});

// 图表引用
const trendChartRef = ref<HTMLElement | null>(null);
const typeChartRef = ref<HTMLElement | null>(null);
const sourceChartRef = ref<HTMLElement | null>(null);
const timeChartRef = ref<HTMLElement | null>(null);

// 生成告警ID
const generateAlarmId = (index: number): string => {
  const dateStr = now.toISOString().slice(2, 10).replace(/-/g, '');
  return `ALM-${dateStr}-${String(index).padStart(3, '0')}`;
};

// 告警列表数据 - 更真实、更少的数据
const loading = ref(false);
const alarmList = ref<AlarmRecord[]>([
  {
    id: "ALM-230517-001",
    title: 'MySQL数据库连接异常',
    level: '严重',
    source: '核心数据库服务器-DB01',
    time: '2025-05-17 08:23:45',
    status: '处理中',
    content: 'MySQL数据库连接池耗尽，应用无法获取新连接，导致服务响应缓慢',
    possibleCause: '数据库连接未正确释放或并发请求量突增导致连接池耗尽',
    solution: '1. 检查应用代码是否存在连接泄漏  2. 适当增加连接池容量  3. 优化数据库访问逻辑'
  },
  {
    id: "ALM-230517-002",
    title: 'Nginx 5xx错误率升高',
    level: '严重',
    source: '负载均衡器-LB02',
    time: '2025-05-17 09:15:22',
    status: '未处理',
    content: 'Nginx错误日志中5xx状态码数量显著增加，当前错误率6.7%，超出阈值(1%)',
    possibleCause: '后端应用服务响应超时或宕机',
    solution: '1. 检查后端应用服务健康状况  2. 分析错误日志定位具体原因  3. 临时扩容应用服务器'
  },
  {
    id: "ALM-230517-003",
    title: 'Redis缓存命中率下降',
    level: '警告',
    source: '缓存服务器-CACHE01',
    time: '2025-05-17 10:42:37',
    status: '已解决',
    content: 'Redis缓存命中率从95%下降到78%，导致数据库负载增加',
    possibleCause: '缓存过期策略不合理或缓存容量不足',
    solution: '增加缓存容量并优化缓存键的过期策略，为热点数据设置更长的过期时间'
  },
  {
    id: "ALM-230516-005",
    title: 'API响应时间异常',
    level: '一般',
    source: '微服务-OrderService',
    time: '2025-05-16 23:05:18',
    status: '已解决',
    content: '订单服务API平均响应时间增加到780ms，超出正常水平(200ms)',
    possibleCause: '数据库查询效率低下或服务资源不足',
    solution: '优化SQL查询语句，添加合适的索引，并考虑增加服务实例数量'
  },
  {
    id: "ALM-230516-004",
    title: '磁盘空间不足',
    level: '警告',
    source: '日志服务器-LOG01',
    time: '2025-05-16 15:37:42',
    status: '处理中',
    content: '日志服务器剩余磁盘空间低于10%，当前可用空间8.2GB',
    possibleCause: '日志文件累积未及时清理或日志记录过于冗长',
    solution: '清理过期日志，配置日志轮转策略，考虑增加磁盘容量'
  }
]);

// 表格列定义
const columns = [
  {
    title: '告警ID',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '告警标题',
    dataIndex: 'title',
    key: 'title',
  },
  {
    title: '级别',
    dataIndex: 'level',
    key: 'level',
  },
  {
    title: '来源',
    dataIndex: 'source',
    key: 'source',
  },
  {
    title: '时间',
    dataIndex: 'time',
    key: 'time',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
  },
  {
    title: '操作',
    key: 'action',
  }
];

// 告警详情弹窗
const detailModalVisible = ref(false);
const selectedAlarm = ref<AlarmRecord | null>(null);

// 查看告警详情
const viewAlarmDetail = (record: AlarmRecord): void => {
  selectedAlarm.value = record;
  detailModalVisible.value = true;
};

// 处理告警
const handleAlarm = (record: AlarmRecord): void => {
  // 显示处理中状态
  loading.value = true;
  
  // 模拟API请求延迟
  setTimeout(() => {
    // 更新状态
    if (record.status === '未处理') {
      record.status = '处理中';
      message.success('告警已开始处理');
    } else if (record.status === '处理中') {
      record.status = '已解决';
      message.success('告警已解决');
      
      // 更新统计数据
      alarmStats.resolved++;
      alarmStats.resolvedTrend = 5;
      
      // 如果是严重告警，更新严重告警数
      if (record.level === '严重') {
        alarmStats.critical--;
        alarmStats.criticalTrend = -5;
      }
    }
    
    loading.value = false;
    
    // 关闭弹窗
    if (detailModalVisible.value) {
      detailModalVisible.value = false;
    }
    
    // 重新初始化图表以反映变化
    initCharts();
  }, 800);
};

// 获取告警级别对应的颜色
const getAlarmLevelColor = (level: string): string => {
  switch (level) {
    case '严重':
      return 'red';
    case '警告':
      return 'orange';
    case '一般':
      return 'blue';
    default:
      return 'default';
  }
};

// 刷新数据
const refreshData = () => {
  loading.value = true;
  message.loading('正在加载数据...', 1);
  
  // 模拟API请求延迟
  setTimeout(() => {
    // 根据时间范围更新统计数据
    switch (timeRange.value) {
      case '1h':
        alarmStats.total = 12;
        alarmStats.critical = 3;
        alarmStats.resolved = 7;
        alarmStats.totalTrend = -2;
        alarmStats.criticalTrend = 0;
        alarmStats.resolvedTrend = 4;
        alarmStats.avgResolveTime = '0.8小时';
        alarmStats.avgTimeTrend = -5;
        break;
      case '6h':
        alarmStats.total = 25;
        alarmStats.critical = 6;
        alarmStats.resolved = 15;
        alarmStats.totalTrend = 3;
        alarmStats.criticalTrend = -2;
        alarmStats.resolvedTrend = 8;
        alarmStats.avgResolveTime = '1.2小时';
        alarmStats.avgTimeTrend = -8;
        break;
      case '24h':
        alarmStats.total = 37;
        alarmStats.critical = 8;
        alarmStats.resolved = 24;
        alarmStats.totalTrend = -5;
        alarmStats.criticalTrend = 3;
        alarmStats.resolvedTrend = 7;
        alarmStats.avgResolveTime = '1.5小时';
        alarmStats.avgTimeTrend = -12;
        break;
      case '7d':
        alarmStats.total = 82;
        alarmStats.critical = 15;
        alarmStats.resolved = 61;
        alarmStats.totalTrend = 6;
        alarmStats.criticalTrend = -8;
        alarmStats.resolvedTrend = 12;
        alarmStats.avgResolveTime = '2.3小时';
        alarmStats.avgTimeTrend = -15;
        break;
    }

    // 更新图表
    initCharts();
    loading.value = false;
    message.success('数据刷新成功');
  }, 1000);
};

// 初始化图表 - 使用更真实的数据
const initCharts = () => {
  nextTick(() => {
    // 告警趋势图
    if (trendChartRef.value) {
      const trendChart = echarts.init(trendChartRef.value);
      
      // 根据选择的时间范围生成对应的时间轴
      let times = [];
      let criticalData = [];
      let warningData = [];
      let normalData = [];
      
      if (timeRange.value === '1h') {
        // 最近1小时，每5分钟一个数据点
        for (let i = 11; i >= 0; i--) {
          const time = new Date(now.getTime() - i * 5 * 60 * 1000);
          times.push(`${time.getHours()}:${time.getMinutes().toString().padStart(2, '0')}`);
          
          // 真实的告警通常是有规律的，而非完全随机
          if (i === 8) {
            criticalData.push(2); // 突然出现的严重告警
            warningData.push(1);
            normalData.push(0);
          } else if (i === 7) {
            criticalData.push(1);
            warningData.push(2);
            normalData.push(1);
          } else if (i === 6) {
            criticalData.push(0);
            warningData.push(1);
            normalData.push(0);
          } else if (i <= 2) {
            criticalData.push(0);
            warningData.push(0);
            normalData.push(i === 0 ? 1 : 0); // 最近时间点出现一个一般告警
          } else {
            criticalData.push(0);
            warningData.push(i === 3 ? 1 : 0);
            normalData.push(0);
          }
        }
      } else if (timeRange.value === '6h') {
        // 最近6小时，每30分钟一个数据点
        for (let i = 11; i >= 0; i--) {
          const time = new Date(now.getTime() - i * 30 * 60 * 1000);
          times.push(`${time.getHours()}:${time.getMinutes().toString().padStart(2, '0')}`);
          
          if (i === 10) {
            criticalData.push(1);
            warningData.push(2);
            normalData.push(0);
          } else if (i === 8) {
            criticalData.push(2);
            warningData.push(1);
            normalData.push(1);
          } else if (i === 6) {
            criticalData.push(1);
            warningData.push(0);
            normalData.push(2);
          } else if (i === 4) {
            criticalData.push(0);
            warningData.push(3);
            normalData.push(1);
          } else if (i === 2) {
            criticalData.push(2);
            warningData.push(1);
            normalData.push(0);
          } else if (i === 0) {
            criticalData.push(0);
            warningData.push(1);
            normalData.push(2);
          } else {
            criticalData.push(0);
            warningData.push(i % 3 === 0 ? 1 : 0);
            normalData.push(i % 4 === 0 ? 1 : 0);
          }
        }
      } else if (timeRange.value === '24h') {
        // 最近24小时，每2小时一个数据点
        for (let i = 11; i >= 0; i--) {
          const time = new Date(now.getTime() - i * 2 * 60 * 60 * 1000);
          times.push(`${time.getHours()}:00`);
          
          // 白天和夜间告警数量通常不同
          const hour = time.getHours();
          const isDaytime = hour >= 8 && hour <= 20;
          
          if (i === 10) { // 生产高峰期
            criticalData.push(2);
            warningData.push(3);
            normalData.push(1);
          } else if (i === 7) { // 业务低谷
            criticalData.push(0);
            warningData.push(1);
            normalData.push(0);
          } else if (i === 5) { // 系统维护
            criticalData.push(3);
            warningData.push(2);
            normalData.push(2);
          } else if (i === 3) { // 业务高峰
            criticalData.push(1);
            warningData.push(4);
            normalData.push(2);
          } else if (i === 1) { // 最近出现的问题
            criticalData.push(2);
            warningData.push(1);
            normalData.push(0);
          } else {
            criticalData.push(isDaytime ? Math.floor(Math.random() * 2) : 0);
            warningData.push(isDaytime ? Math.floor(Math.random() * 3) : Math.floor(Math.random() * 1));
            normalData.push(Math.floor(Math.random() * 2));
          }
        }
      } else { // 7d
        // 最近7天，每天一个数据点
        for (let i = 6; i >= 0; i--) {
          const time = new Date(now.getTime() - i * 24 * 60 * 60 * 1000);
          times.push(`${time.getMonth() + 1}/${time.getDate()}`);
          
          // 模拟工作日和周末的区别
          const day = time.getDay();
          const isWeekend = day === 0 || day === 6;
          
          if (i === 6) { // 一周前
            criticalData.push(isWeekend ? 1 : 3);
            warningData.push(isWeekend ? 2 : 5);
            normalData.push(isWeekend ? 1 : 3);
          } else if (i === 4) { // 系统升级日
            criticalData.push(4);
            warningData.push(6);
            normalData.push(2);
          } else if (i === 2) { // 正常工作日
            criticalData.push(isWeekend ? 0 : 2);
            warningData.push(isWeekend ? 1 : 4);
            normalData.push(isWeekend ? 1 : 3);
          } else if (i === 0) { // 今天
            criticalData.push(2);
            warningData.push(3);
            normalData.push(1);
          } else {
            criticalData.push(isWeekend ? Math.floor(Math.random() * 2) : Math.floor(Math.random() * 3 + 1));
            warningData.push(isWeekend ? Math.floor(Math.random() * 3) : Math.floor(Math.random() * 4 + 2));
            normalData.push(Math.floor(Math.random() * 3 + 1));
          }
        }
      }

      trendChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'axis',
          axisPointer: {
            type: 'shadow'
          }
        },
        legend: {
          data: ['严重', '警告', '一般'],
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
          data: times,
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
            name: '严重',
            type: 'line',
            stack: 'Total',
            data: criticalData,
            lineStyle: {
              width: 2
            },
            itemStyle: {
              color: '#f5222d'
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: 'rgba(245, 34, 45, 0.5)' },
                { offset: 1, color: 'rgba(245, 34, 45, 0.1)' }
              ])
            }
          },
          {
            name: '警告',
            type: 'line',
            stack: 'Total',
            data: warningData,
            lineStyle: {
              width: 2
            },
            itemStyle: {
              color: '#faad14'
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: 'rgba(250, 173, 20, 0.5)' },
                { offset: 1, color: 'rgba(250, 173, 20, 0.1)' }
              ])
            }
          },
          {
            name: '一般',
            type: 'line',
            stack: 'Total',
            data: normalData,
            lineStyle: {
              width: 2
            },
            itemStyle: {
              color: '#1890ff'
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: 'rgba(24, 144, 255, 0.5)' },
                { offset: 1, color: 'rgba(24, 144, 255, 0.1)' }
              ])
            }
          }
        ]
      });
    }

    // 告警类型分布图 - 更真实的数据
    if (typeChartRef.value) {
      const typeChart = echarts.init(typeChartRef.value);
      typeChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'item',
          formatter: '{a} <br/>{b}: {c} ({d}%)'
        },
        legend: {
          orient: 'vertical',
          left: 'left',
          textStyle: {
            color: '#333333'
          }
        },
        series: [
          {
            name: '告警类型',
            type: 'pie',
            radius: '70%',
            center: ['50%', '50%'],
            data: [
              { value: 12, name: '资源利用' },
              { value: 8, name: '连接异常' },
              { value: 6, name: '响应超时' },
              { value: 5, name: '服务不可用' },
              { value: 4, name: '安全事件' },
              { value: 2, name: '其他' }
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
              borderColor: '#ffffff',
              borderWidth: 2
            },
            label: {
              color: '#333333'
            }
          }
        ]
      });
    }

    // 告警来源分布图 - 更真实的数据
    if (sourceChartRef.value) {
      const sourceChart = echarts.init(sourceChartRef.value);
      sourceChart.setOption({
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
          type: 'value',
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
        yAxis: {
          type: 'category',
          data: ['数据库服务', '应用服务', '负载均衡器', '存储设备', '网络设备', '安全设备'],
          axisLine: {
            lineStyle: {
              color: '#333333'
            }
          },
          axisLabel: {
            color: '#333333'
          }
        },
        series: [
          {
            name: '告警数量',
            type: 'bar',
            data: [11, 9, 6, 5, 4, 2],
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#1890ff' },
                { offset: 1, color: '#36cfc9' }
              ])
            }
          }
        ]
      });
    }

    // 告警解决时间分布图 - 更真实的数据
    if (timeChartRef.value) {
      const timeChart = echarts.init(timeChartRef.value);
      timeChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'item',
          formatter: '{a} <br/>{b}: {c} ({d}%)'
        },
        legend: {
          bottom: '5%',
          left: 'center',
          textStyle: {
            color: '#333333'
          }
        },
        series: [
          {
            name: '解决时间',
            type: 'pie',
            radius: ['40%', '70%'],
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
              { value: 15, name: '<30分钟' },
              { value: 9, name: '30分钟-1小时' },
              { value: 7, name: '1-2小时' },
              { value: 5, name: '2-4小时' },
              { value: 1, name: '>4小时' }
            ]
          }
        ]
      });
    }
  });
};

// 页面加载时初始化
onMounted(() => {
  refreshData();

  // 监听窗口大小变化，重绘图表
  window.addEventListener('resize', () => {
    if (trendChartRef.value) {
      echarts.getInstanceByDom(trendChartRef.value)?.resize();
    }
    if (typeChartRef.value) {
      echarts.getInstanceByDom(typeChartRef.value)?.resize();
    }
    if (sourceChartRef.value) {
      echarts.getInstanceByDom(sourceChartRef.value)?.resize();
    }
    if (timeChartRef.value) {
      echarts.getInstanceByDom(timeChartRef.value)?.resize();
    }
  });
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
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  -webkit-background-clip: text;
  color: transparent;
  text-shadow: 0 0 10px rgba(24, 144, 255, 0.5);
}

.actions {
  display: flex;
  gap: 12px;
}

.time-selector {
  border-color: #303030;
}

.refresh-btn {
  display: flex;
  align-items: center;
}

.dashboard {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s ease;
  position: relative;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  z-index: -1;
  border-radius: 10px;
  background: linear-gradient(45deg, #1890ff, #36cfc9, #1890ff);
  background-size: 200% 200%;
  animation: glowing 10s linear infinite;
}

.stat-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  margin: 10px 0;
}

.stat-trend {
  display: flex;
  align-items: center;
  font-size: 14px;
}

.stat-trend.up {
  color: #52c41a;
}

.stat-trend.down {
  color: #f5222d;
}

.charts-container {
  margin-top: 16px;
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

.alarm-list-card {
  border-radius: 8px;
  overflow: hidden;
  margin-top: 16px;
}

.alarm-detail {
  padding: 16px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.detail-actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
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

  .time-selector {
    width: 48% !important;
  }

  .refresh-btn {
    width: 48%;
  }

  .chart-cards {
    grid-template-columns: 1fr;
  }

  .chart-card {
    height: 300px;
  }

  .chart {
    height: 230px;
  }
}
</style>