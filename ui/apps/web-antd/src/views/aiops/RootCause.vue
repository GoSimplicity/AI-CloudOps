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
          <div class="stat-trend" :class="alarmStats.totalIncrease > 0 ? 'up' : 'down'">
            <template v-if="alarmStats.totalIncrease > 0">
              <arrow-up-outlined /> +{{ alarmStats.totalIncrease }}%
            </template>
            <template v-else>
              <arrow-down-outlined /> {{ alarmStats.totalIncrease }}%
            </template>
          </div>
        </a-card>
        <a-card class="stat-card">
          <template #title>
            <warning-outlined /> 严重告警
          </template>
          <div class="stat-value">{{ alarmStats.critical }}</div>
          <div class="stat-trend" :class="alarmStats.criticalIncrease > 0 ? 'up' : 'down'">
            <template v-if="alarmStats.criticalIncrease > 0">
              <arrow-up-outlined /> +{{ alarmStats.criticalIncrease }}%
            </template>
            <template v-else>
              <arrow-down-outlined /> {{ alarmStats.criticalIncrease }}%
            </template>
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
        <a-button @click="closeRootCauseModal">关闭</a-button>
        <a-button type="primary" @click="handleResolveAlarm">标记为已解决</a-button>
      </template>
    </a-modal>

    <!-- 添加操作反馈弹窗 -->
    <a-modal v-model:visible="feedbackModalVisible" :title="feedbackTitle" @ok="closeFeedbackModal">
      <p>{{ feedbackMessage }}</p>
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

// 定义根因分析步骤类型
interface RootCauseStep {
  title: string;
  description: string;
  color: string;
  confidence: number;
}

// 状态数据
const timeRange = ref('24h');
const activeTab = ref('1');
const rootCauseModalVisible = ref(false);
const selectedAlarm = ref<AlarmRecord | null>(null);

// 操作反馈状态
const feedbackModalVisible = ref(false);
const feedbackTitle = ref('');
const feedbackMessage = ref('');

// 图表引用
const trendChartRef = ref(null);
const typeChartRef = ref(null);
const rootCauseGraphRef = ref(null);

// 显示操作反馈
const showFeedback = (title: string, message: string) => {
  feedbackTitle.value = title;
  feedbackMessage.value = message;
  feedbackModalVisible.value = true;
};

// 关闭操作反馈弹窗
const closeFeedbackModal = () => {
  feedbackModalVisible.value = false;
};

// 获取当前日期的前几天日期
const getDateDaysAgo = (daysAgo: number) => {
  const date = new Date();
  date.setDate(date.getDate() - daysAgo);
  
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');

  return `${year}-${month}-${day} ${hours}:${minutes}`;
};

// 告警统计数据
const alarmStats = reactive({
  total: 37,
  totalIncrease: -5,
  critical: 3,
  criticalIncrease: -12,
  resolved: 24,
  resolvedIncrease: 8,
  avgResolveTime: '27分钟',
  timeDecrease: 14
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

// 告警列表数据 - 只保留两条真实数据
const alarmList = ref<AlarmRecord[]>([
  { 
    id: 'ALM-2025-0142', 
    name: 'API响应延迟超阈值', 
    time: getDateDaysAgo(3), 
    severity: '严重', 
    status: '未解决', 
    service: '订单服务', 
    recommendation: '检查数据库索引是否失效，优化查询SQL语句，检查数据库连接池配置，考虑增加读库分担负载。' 
  },
  { 
    id: 'ALM-2025-0141', 
    name: '容器内存使用率超阈值', 
    time: getDateDaysAgo(4), 
    severity: '警告', 
    status: '处理中', 
    service: '商品服务', 
    recommendation: '检查内存泄漏可能，增加容器内存配额，优化大查询内存占用，调整JVM参数配置。' 
  }
]);

// 历史告警数据 - 只保留两条真实数据
const historyAlarmList = ref<AlarmRecord[]>([
  { 
    id: 'ALM-2025-0140', 
    name: '数据库连接池耗尽', 
    time: getDateDaysAgo(5), 
    severity: '严重', 
    status: '已解决', 
    service: '用户中心', 
    recommendation: '扩大连接池配置，优化慢查询，修复连接未释放问题，增加超时配置。' 
  },
  { 
    id: 'ALM-2025-0139', 
    name: 'Redis缓存命中率下降', 
    time: getDateDaysAgo(5), 
    severity: '一般', 
    status: '已解决', 
    service: '搜索服务', 
    recommendation: '检查缓存过期策略，预热热点数据，增加缓存容量，优化缓存key设计。' 
  }
]);

// 初始化根因分析步骤
const initRootCauseSteps = (alarmType: string): RootCauseStep[] => {
  // 根据不同的告警类型返回不同的分析步骤
  if (alarmType === 'API响应延迟超阈值') {
    return [
      { title: '告警触发', description: '系统监测到订单服务API平均响应时间超过500ms，持续15分钟', color: 'red', confidence: 100 },
      { title: '关联分析', description: '订单服务依赖的数据库查询响应时间同步增加', color: 'orange', confidence: 95 },
      { title: '根因定位', description: '数据库监控发现ORDER_ITEMS表上的索引未被使用，导致全表扫描', color: 'blue', confidence: 92 },
      { title: '问题确认', description: '最近发布的订单查询接口未按索引字段查询，导致查询效率下降', color: 'green', confidence: 97 }
    ];
  } else if (alarmType === '容器内存使用率超阈值') {
    return [
      { title: '告警触发', description: '商品服务容器内存使用率达到87%，超过预设阈值85%', color: 'red', confidence: 100 },
      { title: '表现分析', description: 'GC日志显示频繁Full GC但释放内存效果不明显', color: 'orange', confidence: 94 },
      { title: '根因定位', description: '发现商品图片处理模块未正确释放临时文件句柄', color: 'blue', confidence: 90 },
      { title: '问题确认', description: '5月12日部署的新版本中，文件处理逻辑缺少finally语句块中的资源释放代码', color: 'green', confidence: 96 }
    ];
  } else if (alarmType === '数据库连接池耗尽') {
    return [
      { title: '告警触发', description: '用户中心数据库连接池活跃连接数达到最大值50，新请求被拒绝', color: 'red', confidence: 100 },
      { title: '表现分析', description: '连接平均生命周期异常延长，从正常的2秒增加到15秒', color: 'orange', confidence: 93 },
      { title: '根因定位', description: '用户登录接口未正确关闭数据库连接，导致连接泄露', color: 'blue', confidence: 91 },
      { title: '问题确认', description: '5月11日功能更新中，try-with-resources语句被错误修改，导致连接未自动关闭', color: 'green', confidence: 98 }
    ];
  } else {
    return [
      { title: '告警触发', description: '搜索服务Redis缓存命中率从95%下降到68%', color: 'red', confidence: 100 },
      { title: '表现分析', description: '缓存请求量正常，但命中次数明显减少', color: 'orange', confidence: 92 },
      { title: '根因定位', description: '缓存TTL设置过短（10分钟），热门商品数据频繁过期', color: 'blue', confidence: 89 },
      { title: '问题确认', description: '5月12日配置变更将缓存时间从30分钟调整为10分钟，但未考虑访问频率', color: 'green', confidence: 95 }
    ];
  }
};

// 当前选中告警的根因分析步骤
const rootCauseSteps = ref<RootCauseStep[]>([]);

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
  rootCauseSteps.value = initRootCauseSteps(record.name);
  rootCauseModalVisible.value = true;

  // 在模态框显示后初始化根因分析图
  setTimeout(() => {
    initRootCauseGraph(record.name);
  }, 100);
};

// 关闭根因分析模态框
const closeRootCauseModal = (): void => {
  rootCauseModalVisible.value = false;
  showFeedback('操作成功', '已关闭当前根因分析详情');
};

// 处理解决告警
const handleResolveAlarm = (): void => {
  if (selectedAlarm.value) {
    // 构建反馈信息
    const alarmName = selectedAlarm.value.name;
    const alarmId = selectedAlarm.value.id;
    
    // 更新状态
    selectedAlarm.value.status = '已解决';
    
    // 更新列表
    const currentList = activeTab.value === '1' ? alarmList : historyAlarmList;
    if (currentList.value) {
      const index = currentList.value.findIndex(item => item.id === alarmId);
      if (index !== -1) {
        console.log('已解决')
      }
    }
    
    rootCauseModalVisible.value = false;
    
    // 显示操作反馈
    showFeedback('告警已解决', `已成功将告警 "${alarmName}" (${alarmId}) 标记为已解决状态。系统将记录相关处理信息用于后续分析。`);
    
    // 更新统计数据
    alarmStats.resolved++;
    if (selectedAlarm.value.severity === '严重') {
      alarmStats.critical--;
    }
  }
};

// 刷新数据
const refreshData = () => {
  // 显示操作反馈
  showFeedback('数据刷新成功', '已更新告警数据和分析结果');
  
  // 更新图表
  initTrendChart();
  initTypeChart();
};

// 初始化趋势图表
const initTrendChart = () => {
  const chartDom = trendChartRef.value;
  if (!chartDom) return;

  const myChart = echarts.init(chartDom);

  // 获取过去7天的日期作为X轴
  const dates = [];
  for (let i = 7; i >= 0; i--) {
    const date = new Date();
    date.setDate(date.getDate() - i);
    const month = date.getMonth() + 1;
    const day = date.getDate();
    dates.push(`${month}/${day}`);
  }

  // 生成更真实的告警数据
  const criticalData = [2, 3, 1, 2, 4, 3, 3, 2];
  const warningData = [5, 6, 4, 5, 7, 5, 6, 4];
  const infoData = [8, 9, 7, 7, 10, 8, 7, 6];

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['严重', '警告', '一般'],
      textStyle: {
        color: 'inherit'
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
      data: dates,
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      },
      axisLabel: {
        color: 'inherit'
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      },
      axisLabel: {
        color: 'inherit'
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(127, 127, 127, 0.2)'
        }
      }
    },
    series: [
      {
        name: '严重',
        type: 'line',
        data: criticalData,
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
        data: warningData,
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
        data: infoData,
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

  const option = {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      textStyle: {
        color: 'inherit'
      }
    },
    series: [
      {
        name: '告警类型',
        type: 'pie',
        radius: '70%',
        data: [
          { value: 12, name: '性能异常' },
          { value: 9, name: '资源耗尽' },
          { value: 7, name: '连接失败' },
          { value: 5, name: '服务不可用' },
          { value: 4, name: '其他' }
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
          color: 'inherit'
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

// 初始化根因分析图 - 根据不同告警类型生成不同图表
const initRootCauseGraph = (alarmType: string) => {
  const chartDom = rootCauseGraphRef.value;
  if (!chartDom) return;

  const myChart = echarts.init(chartDom);
  
  let option;
  
  if (alarmType === 'API响应延迟超阈值') {
    option = {
      tooltip: {},
      legend: [
        {
          data: ['服务', '组件', '告警点'],
          textStyle: {
            color: 'inherit'
          }
        }
      ],
      series: [
        {
          name: '根因分析',
          type: 'graph',
          layout: 'force',
          data: [
            { name: '订单服务', category: 0, symbolSize: 50, value: 20, itemStyle: { color: '#ff4d4f' } },
            { name: '订单数据库', category: 1, symbolSize: 40, value: 15, itemStyle: { color: '#52c41a' } },
            { name: 'ORDER_ITEMS表', category: 1, symbolSize: 35, value: 15, itemStyle: { color: '#faad14' } },
            { name: '缺失索引', category: 2, symbolSize: 30, value: 10, itemStyle: { color: '#1890ff' } },
            { name: '查询接口', category: 0, symbolSize: 35, value: 10, itemStyle: { color: '#722ed1' } }
          ],
          links: [
            { source: '订单服务', target: '订单数据库', lineStyle: { color: '#1890ff', width: 3 } },
            { source: '订单数据库', target: 'ORDER_ITEMS表', lineStyle: { color: '#faad14', width: 3 } },
            { source: 'ORDER_ITEMS表', target: '缺失索引', lineStyle: { color: '#ff4d4f', width: 4 } },
            { source: '订单服务', target: '查询接口', lineStyle: { color: '#722ed1', width: 2 } },
            { source: '查询接口', target: 'ORDER_ITEMS表', lineStyle: { color: '#52c41a', width: 2 } }
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
            color: 'inherit'
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
  } else if (alarmType === '容器内存使用率超阈值') {
    option = {
      tooltip: {},
      legend: [
        {
          data: ['服务', '组件', '告警点'],
          textStyle: {
            color: 'inherit'
          }
        }
      ],
      series: [
        {
          name: '根因分析',
          type: 'graph',
          layout: 'force',
          data: [
            { name: '商品服务', category: 0, symbolSize: 50, value: 20, itemStyle: { color: '#ff4d4f' } },
            { name: '图片处理模块', category: 1, symbolSize: 40, value: 15, itemStyle: { color: '#faad14' } },
            { name: '文件句柄泄漏', category: 2, symbolSize: 45, value: 15, itemStyle: { color: '#1890ff' } },
            { name: 'JVM内存', category: 1, symbolSize: 35, value: 10, itemStyle: { color: '#52c41a' } },
            { name: '临时文件存储', category: 1, symbolSize: 30, value: 10, itemStyle: { color: '#722ed1' } }
          ],
          links: [
            { source: '商品服务', target: '图片处理模块', lineStyle: { color: '#1890ff', width: 3 } },
            { source: '图片处理模块', target: '文件句柄泄漏', lineStyle: { color: '#ff4d4f', width: 4 } },
            { source: '文件句柄泄漏', target: 'JVM内存', lineStyle: { color: '#faad14', width: 3 } },
            { source: '图片处理模块', target: '临时文件存储', lineStyle: { color: '#52c41a', width: 2 } },
            { source: '临时文件存储', target: '文件句柄泄漏', lineStyle: { color: '#722ed1', width: 2 } }
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
            color: 'inherit'
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
  } else {
    // 默认图表
    option = {
      tooltip: {},
      legend: [
        {
          data: ['服务', '组件', '告警点'],
          textStyle: {
            color: 'inherit'
          }
        }
      ],
      series: [
        {
          name: '根因分析',
          type: 'graph',
          layout: 'force',
          data: [
            { name: selectedAlarm.value?.service, category: 0, symbolSize: 50, value: 20, itemStyle: { color: '#ff4d4f' } },
            { name: '相关组件', category: 1, symbolSize: 40, value: 15, itemStyle: { color: '#52c41a' } },
            { name: '依赖服务', category: 0, symbolSize: 40, value: 15, itemStyle: { color: '#1890ff' } },
            { name: '数据存储', category: 1, symbolSize: 30, value: 10, itemStyle: { color: '#722ed1' } },
            { name: selectedAlarm.value?.name, category: 2, symbolSize: 40, value: 15, itemStyle: { color: '#faad14' } }
          ],
          links: [
            { source: selectedAlarm.value?.name, target: selectedAlarm.value?.service, lineStyle: { color: '#ff4d4f', width: 3 } },
            { source: selectedAlarm.value?.service, target: '相关组件', lineStyle: { color: '#1890ff', width: 2 } },
            { source: selectedAlarm.value?.service, target: '依赖服务', lineStyle: { color: '#52c41a', width: 2 } },
            { source: '相关组件', target: '数据存储', lineStyle: { color: '#722ed1', width: 2 } },
            { source: '依赖服务', target: '数据存储', lineStyle: { color: '#faad14', width: 1 } }
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
            color: 'inherit'
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
  }

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