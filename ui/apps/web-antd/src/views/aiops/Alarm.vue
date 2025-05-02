<template>
  <div class="alarm-container">
    <div class="header">
      <h1 class="title">智能运维告警分析平台</h1>
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

// 统计数据
const alarmStats = reactive({
  total: 143,
  totalTrend: 8,
  critical: 27,
  criticalTrend: -5,
  resolved: 92,
  resolvedTrend: 12,
  avgResolveTime: '2.8小时',
  avgTimeTrend: -3
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

// 告警列表数据
const loading = ref(false);
const alarmList = ref<AlarmRecord[]>([
  {
    id: generateAlarmId(1),
    title: 'CPU使用率过高',
    level: '严重',
    source: '生产服务器-Web01',
    time: getTimeAgo(3),
    status: '已解决',
    content: '服务器CPU使用率持续超过95%达10分钟',
    possibleCause: '应用程序内存泄漏或高并发请求导致',
    solution: '检查应用程序日志，重启应用或增加资源配置'
  },
  {
    id: generateAlarmId(2),
    title: '数据库连接异常',
    level: '严重',
    source: '数据库服务器-DB01',
    time: getTimeAgo(2),
    status: '处理中',
    content: '数据库连接池耗尽，新连接请求被拒绝',
    possibleCause: '连接未正确关闭或连接池配置不合理',
    solution: '检查应用程序是否正确关闭连接，调整连接池大小'
  },
  {
    id: generateAlarmId(3),
    title: '磁盘空间不足',
    level: '警告',
    source: '存储服务器-STO02',
    time: getTimeAgo(5),
    status: '未处理',
    content: '磁盘使用率达到85%，接近警戒线',
    possibleCause: '日志文件过大或临时文件未清理',
    solution: '清理日志文件，删除临时文件，考虑扩容'
  },
  {
    id: generateAlarmId(4),
    title: '网络延迟异常',
    level: '一般',
    source: '网络设备-SW01',
    time: getTimeAgo(8),
    status: '已解决',
    content: '网络延迟超过200ms，影响用户体验',
    possibleCause: '网络拥塞或路由配置问题',
    solution: '检查网络设备负载，优化路由配置'
  },
  {
    id: generateAlarmId(5),
    title: '应用响应超时',
    level: '严重',
    source: '应用服务器-APP03',
    time: getTimeAgo(1),
    status: '未处理',
    content: '应用响应时间超过5秒，用户反馈系统卡顿',
    possibleCause: '数据库查询效率低或应用代码性能问题',
    solution: '优化SQL查询，检查应用代码性能瓶颈'
  },
  {
    id: generateAlarmId(6),
    title: '内存使用率过高',
    level: '警告',
    source: '生产服务器-Web02',
    time: getTimeAgo(4),
    status: '处理中',
    content: '服务器内存使用率达到90%',
    possibleCause: '应用程序内存泄漏或配置不合理',
    solution: '检查应用程序内存使用情况，调整JVM参数'
  },
  {
    id: generateAlarmId(7),
    title: '安全漏洞检测',
    level: '严重',
    source: '安全网关-SEC01',
    time: getTimeAgo(6),
    status: '未处理',
    content: '检测到潜在的SQL注入攻击尝试',
    possibleCause: '应用程序未对用户输入进行充分验证',
    solution: '更新WAF规则，修复应用程序输入验证逻辑'
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
  // 实际应用中这里会调用API处理告警
  record.status = '处理中';
  if (detailModalVisible.value) {
    detailModalVisible.value = false;
  }
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
  // 模拟API请求延迟
  setTimeout(() => {
    // 更新统计数据
    alarmStats.total = Math.floor(Math.random() * 50) + 100;
    alarmStats.totalTrend = Math.floor(Math.random() * 30) - 15;
    alarmStats.critical = Math.floor(Math.random() * 20) + 10;
    alarmStats.criticalTrend = Math.floor(Math.random() * 30) - 15;
    alarmStats.resolved = Math.floor(Math.random() * 40) + 60;
    alarmStats.resolvedTrend = Math.floor(Math.random() * 30) - 15;

    // 更新图表
    initCharts();
    loading.value = false;
  }, 1000);
};

// 初始化图表
const initCharts = () => {
  nextTick(() => {
    // 告警趋势图
    if (trendChartRef.value) {
      const trendChart = echarts.init(trendChartRef.value);
      const times = [];
      const criticalData = [];
      const warningData = [];
      const normalData = [];

      for (let i = 23; i >= 0; i--) {
        const time = new Date(now.getTime() - i * 3600 * 1000);
        times.push(`${time.getHours()}:00`);
        criticalData.push(Math.floor(Math.random() * 10));
        warningData.push(Math.floor(Math.random() * 15));
        normalData.push(Math.floor(Math.random() * 8));
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
          data: times,
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

    // 告警类型分布图
    if (typeChartRef.value) {
      const typeChart = echarts.init(typeChartRef.value);
      typeChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'item'
        },
        legend: {
          orient: 'vertical',
          left: 'left',
          textStyle: {
            color: '#333333'  // 修改为深色字体
          }
        },
        series: [
          {
            name: '告警类型',
            type: 'pie',
            radius: '70%',
            center: ['50%', '50%'],
            data: [
              { value: 38, name: '资源使用率' },
              { value: 27, name: '应用异常' },
              { value: 16, name: '网络问题' },
              { value: 12, name: '安全事件' },
              { value: 7, name: '其他' }
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
              borderColor: '#ffffff',  // 修改为白色边框
              borderWidth: 2
            },
            label: {
              color: '#333333'  // 修改为深色字体
            }
          }
        ]
      });
    }

    // 告警来源分布图
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
        yAxis: {
          type: 'category',
          data: ['应用服务器', '数据库服务器', '网络设备', '存储设备', '安全设备'],
          axisLine: {
            lineStyle: {
              color: '#333333'  // 修改为深色
            }
          },
          axisLabel: {
            color: '#333333'  // 修改为深色字体
          }
        },
        series: [
          {
            name: '告警数量',
            type: 'bar',
            data: [45, 26, 18, 14, 9],
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

    // 告警解决时间分布图
    if (timeChartRef.value) {
      const timeChart = echarts.init(timeChartRef.value);
      timeChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'item'
        },
        legend: {
          bottom: '5%',
          left: 'center',
          textStyle: {
            color: '#333333'  // 修改为深色字体
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
              borderColor: '#ffffff',  // 修改为白色边框
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
              { value: 42, name: '<1小时' },
              { value: 35, name: '1-3小时' },
              { value: 15, name: '3-6小时' },
              { value: 8, name: '>6小时' }
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
