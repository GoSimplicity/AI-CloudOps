<template>
  <div class="k8s-aiops-container">
    <div class="header">
      <h1 class="title">Kubernetes 智能运维平台</h1>
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
      <!-- 集群状态卡片 -->
      <div class="stats-cards">
        <a-card class="stat-card k8s-card">
          <template #title>
            <cluster-outlined /> 集群健康度
          </template>
          <div class="stat-value">{{ clusterStats.health }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ clusterStats.healthImprovement }}%
          </div>
        </a-card>
        <a-card class="stat-card k8s-card">
          <template #title>
            <api-outlined /> Pod 可用率
          </template>
          <div class="stat-value">{{ clusterStats.podAvailability }}%</div>
          <div class="stat-trend" :class="clusterStats.podTrend > 0 ? 'up' : 'down'">
            <template v-if="clusterStats.podTrend > 0">
              <arrow-up-outlined /> +{{ clusterStats.podTrend }}%
            </template>
            <template v-else>
              <arrow-down-outlined /> {{ clusterStats.podTrend }}%
            </template>
          </div>
        </a-card>
        <a-card class="stat-card k8s-card">
          <template #title>
            <alert-outlined /> 异常节点数
          </template>
          <div class="stat-value">{{ clusterStats.abnormalNodes }}</div>
          <div class="stat-trend" :class="clusterStats.nodeTrend > 0 ? 'down' : 'up'">
            <template v-if="clusterStats.nodeTrend > 0">
              <arrow-up-outlined /> +{{ clusterStats.nodeTrend }}%
            </template>
            <template v-else>
              <arrow-down-outlined /> {{ clusterStats.nodeTrend }}%
            </template>
          </div>
        </a-card>
        <a-card class="stat-card k8s-card">
          <template #title>
            <dashboard-outlined /> 资源利用率
          </template>
          <div class="stat-value">{{ clusterStats.resourceUtilization }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ clusterStats.utilizationImprovement }}%
          </div>
        </a-card>
      </div>

      <!-- 图表区域 -->
      <div class="chart-cards">
        <a-card class="chart-card" title="集群资源使用趋势">
          <div class="chart" ref="resourceChart"></div>
        </a-card>
        <a-card class="chart-card" title="Pod 状态分布">
          <div class="chart" ref="podStatusChart"></div>
        </a-card>
      </div>

      <div class="chart-cards">
        <a-card class="chart-card" title="节点负载热力图">
          <div class="chart" ref="nodeHeatmapChart"></div>
        </a-card>
        <a-card class="chart-card" title="异常事件趋势">
          <div class="chart" ref="eventTrendChart"></div>
        </a-card>
      </div>

      <!-- 智能优化建议 -->
      <a-card class="optimization-card" title="智能优化建议">
        <a-list :data-source="optimizationSuggestions" :pagination="false">
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar :style="{ backgroundColor: item.color }">
                    <template v-if="item.type === 'resource'">
                      <control-outlined />
                    </template>
                    <template v-else-if="item.type === 'performance'">
                      <thunderbolt-outlined />
                    </template>
                    <template v-else-if="item.type === 'security'">
                      <safety-outlined />
                    </template>
                    <template v-else>
                      <bulb-outlined />
                    </template>
                  </a-avatar>
                </template>
                <template #title>
                  <span class="suggestion-title">{{ item.title }}</span>
                </template>
                <template #description>
                  <div class="suggestion-description">{{ item.description }}</div>
                  <div class="suggestion-metrics">
                    <span class="metric">
                      <arrow-up-outlined v-if="item.impact > 0" class="up" />
                      <arrow-down-outlined v-else class="down" />
                      预期影响: {{ Math.abs(item.impact) }}%
                    </span>
                    <span class="metric">
                      <clock-circle-outlined />
                      实施难度: {{ item.difficulty }}
                    </span>
                  </div>
                </template>
              </a-list-item-meta>
              <template #extra>
                <a-button type="primary" ghost @click="handleOptimizationApply(item)">应用</a-button>
              </template>
            </a-list-item>
          </template>
        </a-list>
      </a-card>

      <!-- 实时监控面板 -->
      <a-card class="monitoring-card" title="实时监控面板">
        <a-tabs default-active-key="1">
          <a-tab-pane key="1" tab="Pod 状态">
            <a-table :columns="podColumns" :data-source="pods" :pagination="{ pageSize: 5 }" :scroll="{ x: 1000 }">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'status'">
                  <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
                </template>
                <template v-if="column.key === 'cpu'">
                  <a-progress :percent="record.cpu" size="small" :status="getResourceStatus(record.cpu)" />
                </template>
                <template v-if="column.key === 'memory'">
                  <a-progress :percent="record.memory" size="small" :status="getResourceStatus(record.memory)" />
                </template>
                <template v-if="column.key === 'action'">
                  <a-space>
                    <a-button type="link" size="small" @click="handlePodDetail(record)">详情</a-button>
                    <a-button type="link" size="small" @click="handlePodRestart(record)">重启</a-button>
                  </a-space>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
          <a-tab-pane key="2" tab="节点状态">
            <a-table :columns="nodeColumns" :data-source="nodes" :pagination="{ pageSize: 5 }" :scroll="{ x: 1000 }">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'status'">
                  <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
                </template>
                <template v-if="column.key === 'cpu'">
                  <a-progress :percent="record.cpu" size="small" :status="getResourceStatus(record.cpu)" />
                </template>
                <template v-if="column.key === 'memory'">
                  <a-progress :percent="record.memory" size="small" :status="getResourceStatus(record.memory)" />
                </template>
                <template v-if="column.key === 'disk'">
                  <a-progress :percent="record.disk" size="small" :status="getResourceStatus(record.disk)" />
                </template>
                <template v-if="column.key === 'action'">
                  <a-space>
                    <a-button type="link" size="small" @click="handleNodeDetail(record)">详情</a-button>
                    <a-button type="link" size="small" @click="handleNodeMaintenance(record)">维护</a-button>
                  </a-space>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
          <a-tab-pane key="3" tab="事件日志">
            <a-timeline mode="alternate">
              <a-timeline-item v-for="(event, index) in events" :key="index" :color="getEventColor(event.level)">
                <template #dot>
                  <template v-if="event.level === 'warning'">
                    <warning-outlined style="font-size: 16px;" />
                  </template>
                  <template v-else-if="event.level === 'error'">
                    <close-circle-outlined style="font-size: 16px;" />
                  </template>
                  <template v-else-if="event.level === 'info'">
                    <info-circle-outlined style="font-size: 16px;" />
                  </template>
                  <template v-else>
                    <check-circle-outlined style="font-size: 16px;" />
                  </template>
                </template>
                <div class="event-item">
                  <div class="event-time">{{ event.time }}</div>
                  <div class="event-content">{{ event.content }}</div>
                  <div class="event-source">{{ event.source }}</div>
                </div>
              </a-timeline-item>
            </a-timeline>
          </a-tab-pane>
        </a-tabs>
      </a-card>
    </div>
    <!-- 添加通知组件 -->
    <a-modal v-model:visible="messageVisible" :title="messageTitle" @ok="closeMessage">
      <p>{{ messageContent }}</p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue';
import * as echarts from 'echarts';
import {
  SyncOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  ClusterOutlined,
  ApiOutlined,
  AlertOutlined,
  DashboardOutlined,
  ControlOutlined,
  ThunderboltOutlined,
  SafetyOutlined,
  BulbOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  CheckCircleOutlined
} from '@ant-design/icons-vue';

// 消息通知状态
const messageVisible = ref(false);
const messageTitle = ref('');
const messageContent = ref('');

// 显示消息通知
const showMessage = (title: string, content: string) => {
  messageTitle.value = title;
  messageContent.value = content;
  messageVisible.value = true;
};

// 关闭消息通知
const closeMessage = () => {
  messageVisible.value = false;
};

// 时间范围选择
const timeRange = ref('24h');

// 集群统计数据
const clusterStats = reactive({
  health: 97.2,
  healthImprovement: 1.8,
  podAvailability: 99.5,
  podTrend: 0.3,
  abnormalNodes: 1,
  nodeTrend: -50,
  resourceUtilization: 63.5,
  utilizationImprovement: 2.1
});

// 图表引用
const resourceChart = ref(null);
const podStatusChart = ref(null);
const nodeHeatmapChart = ref(null);
const eventTrendChart = ref(null);

// 优化建议数据 - 只保留两条更真实的建议
const optimizationSuggestions = ref([
  {
    type: 'resource',
    title: '优化 frontend-service 资源配置',
    description: '检测到 frontend-service 内存请求过高(2Gi)，而实际使用率平均只有56%。建议将内存请求调整至1.2Gi，可节省约40%的内存资源。',
    impact: 40,
    difficulty: '低',
    color: '#1890ff'
  },
  {
    type: 'performance',
    title: '增加 data-processor 副本数',
    description: 'data-processor 服务在过去48小时内CPU使用率持续超过85%，造成处理延迟增加了32%。建议将副本数从2扩展到3，缓解负载压力。',
    impact: 35,
    difficulty: '中',
    color: '#722ed1'
  }
]);

// Pod 表格列定义
const podColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', fixed: 'left' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '就绪', dataIndex: 'ready', key: 'ready' },
  { title: '重启次数', dataIndex: 'restarts', key: 'restarts' },
  { title: 'CPU 使用率', dataIndex: 'cpu', key: 'cpu' },
  { title: '内存使用率', dataIndex: 'memory', key: 'memory' },
  { title: '创建时间', dataIndex: 'age', key: 'age' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 节点表格列定义
const nodeColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', fixed: 'left' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '角色', dataIndex: 'roles', key: 'roles' },
  { title: 'CPU 使用率', dataIndex: 'cpu', key: 'cpu' },
  { title: '内存使用率', dataIndex: 'memory', key: 'memory' },
  { title: '磁盘使用率', dataIndex: 'disk', key: 'disk' },
  { title: 'Pod 数量', dataIndex: 'pods', key: 'pods' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 定义类型接口
interface Pod {
  key: number;
  name: string;
  namespace: string;
  status: string;
  ready: string;
  restarts: number;
  cpu: number;
  memory: number;
  age: string;
}

interface Node {
  key: number;
  name: string;
  status: string;
  roles: string;
  cpu: number;
  memory: number;
  disk: number;
  pods: string;
}

interface Event {
  level: string;
  content: string;
  source: string;
  time: string;
}

// 计算过去几天的日期
const getDateFromDaysAgo = (daysAgo: number): string => {
  const date = new Date();
  date.setDate(date.getDate() - daysAgo);
  return `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')}`;
};

// 真实 Pod 数据 - 只包含两条真实样例
const generatePods = (): Pod[] => {
  return [
    {
      key: 1,
      name: 'frontend-service-7d9f4b9876-2x8ht',
      namespace: 'production',
      status: 'Running',
      ready: '1/1',
      restarts: 0,
      cpu: 38,
      memory: 56,
      age: `${getDateFromDaysAgo(3)} 08:42:15`
    },
    {
      key: 2,
      name: 'data-processor-6b8c7d45f9-j4kl7',
      namespace: 'data-services',
      status: 'Running',
      ready: '1/1',
      restarts: 2,
      cpu: 87,
      memory: 72,
      age: `${getDateFromDaysAgo(5)} 14:37:22`
    }
  ];
};

// 真实节点数据 - 只包含两条真实样例
const generateNodes = (): Node[] => {
  return [
    {
      key: 1,
      name: 'node-worker-east1-01',
      status: 'Ready',
      roles: 'worker',
      cpu: 63,
      memory: 72,
      disk: 48,
      pods: '18/110'
    },
    {
      key: 2,
      name: 'node-master-east1-01',
      status: 'Ready',
      roles: 'master',
      cpu: 32,
      memory: 45,
      disk: 37,
      pods: '12/110'
    }
  ];
};

// 真实事件数据 - 只包含两条真实样例
const generateEvents = (): Event[] => {
  return [
    {
      level: 'warning',
      content: 'data-processor-6b8c7d45f9-j4kl7 Pod 内存使用率达到阈值 (72%)',
      source: 'kubelet',
      time: `${getDateFromDaysAgo(3)} 15:32:47`
    },
    {
      level: 'info',
      content: 'HorizontalPodAutoscaler 调整 frontend-service 副本数 2->3',
      source: 'kube-controller-manager',
      time: `${getDateFromDaysAgo(4)} 09:15:23`
    }
  ];
};

// 状态颜色映射
const getStatusColor = (status: string): string => {
  const colorMap: Record<string, string> = {
    'Running': 'success',
    'Ready': 'success',
    'Pending': 'processing',
    'Failed': 'error',
    'NotReady': 'error',
    'SchedulingDisabled': 'warning',
    'Succeeded': 'success',
    'Unknown': 'default'
  };
  return colorMap[status] || 'default';
};

// 资源状态判断
const getResourceStatus = (value: number): string => {
  if (value >= 90) return 'exception';
  if (value >= 75) return 'warning';
  return 'normal';
};

// 事件颜色映射
const getEventColor = (level: string): string => {
  const colorMap: Record<string, string> = {
    'info': 'blue',
    'warning': 'orange',
    'error': 'red',
    'success': 'green'
  };
  return colorMap[level] || 'blue';
};
// 处理优化应用按钮点击
const handleOptimizationApply = (item: { title: string }) => {
  showMessage('操作成功', `已成功应用优化: ${item.title}`);
};

// 处理Pod详情按钮点击
const handlePodDetail = (record: { name: string }) => {
  showMessage('Pod详情', `正在查看Pod "${record.name}" 的详细信息`);
};

// 处理Pod重启按钮点击
const handlePodRestart = (record: { name: string }) => {
  showMessage('操作成功', `Pod "${record.name}" 重启命令已发送，正在执行中...`);
};

// 处理节点详情按钮点击
const handleNodeDetail = (record: { name: string }) => {
  showMessage('节点详情', `正在查看节点 "${record.name}" 的详细信息`);
};

// 处理节点维护按钮点击
const handleNodeMaintenance = (record: { name: string }) => {
  showMessage('操作成功', `节点 "${record.name}" 已进入维护模式，workload正在迁移...`);
};

// 数据引用
const pods = ref<Pod[]>([]);
const nodes = ref<Node[]>([]);
const events = ref<Event[]>([]);

// 初始化资源使用趋势图表
const initResourceChart = () => {
  const chart = echarts.init(resourceChart.value);

  // 使用过去5天的日期作为x轴数据
  const days = [];
  for (let i = 5; i >= 0; i--) {
    days.push(getDateFromDaysAgo(i));
  }

  // 真实的资源使用率数据
  const cpuData = [52, 58, 61, 65, 63, 59];
  const memoryData = [64, 67, 70, 72, 68, 65];
  const diskData = [42, 43, 45, 45, 48, 48];

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985'
        }
      }
    },
    legend: {
      data: ['CPU', '内存', '磁盘'],
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      },
      padding: 10,
      borderRadius: 4
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: days,
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(127, 127, 127, 0.2)'
        }
      },
      axisLabel: {
        formatter: '{value}%',
        color: 'inherit'
      }
    },
    series: [
      {
        name: 'CPU',
        type: 'line',
        stack: 'Total',
        areaStyle: {
          opacity: 0.3
        },
        emphasis: {
          focus: 'series'
        },
        data: cpuData,
        smooth: true,
        lineStyle: {
          width: 2
        },
        itemStyle: {
          color: '#1890ff'
        }
      },
      {
        name: '内存',
        type: 'line',
        stack: 'Total',
        areaStyle: {
          opacity: 0.3
        },
        emphasis: {
          focus: 'series'
        },
        data: memoryData,
        smooth: true,
        lineStyle: {
          width: 2
        },
        itemStyle: {
          color: '#52c41a'
        }
      },
      {
        name: '磁盘',
        type: 'line',
        stack: 'Total',
        areaStyle: {
          opacity: 0.3
        },
        emphasis: {
          focus: 'series'
        },
        data: diskData,
        smooth: true,
        lineStyle: {
          width: 2
        },
        itemStyle: {
          color: '#722ed1'
        }
      }
    ]
  };

  chart.setOption(option);
  window.addEventListener('resize', () => chart.resize());
};

// 初始化 Pod 状态分布图表
const initPodStatusChart = () => {
  const chart = echarts.init(podStatusChart.value);

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item'
    },
    legend: {
      top: '5%',
      left: 'center',
      textStyle: {
        color: 'inherit'
      }
    },
    series: [
      {
        name: 'Pod 状态',
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
            color: 'inherit'
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: 42, name: 'Running', itemStyle: { color: '#52c41a' } },
          { value: 3, name: 'Pending', itemStyle: { color: '#1890ff' } },
          { value: 1, name: 'Failed', itemStyle: { color: '#f5222d' } },
          { value: 0, name: 'Succeeded', itemStyle: { color: '#13c2c2' } },
          { value: 0, name: 'Unknown', itemStyle: { color: '#faad14' } }
        ]
      }
    ]
  };

  chart.setOption(option);
  window.addEventListener('resize', () => chart.resize());
};

// 初始化节点负载热力图
const initNodeHeatmapChart = () => {
  const chart = echarts.init(nodeHeatmapChart.value);

  // 使用过去5天的日期作为x轴数据
  const days: string[] = [];
  for (let i = 5; i >= 0; i--) {
    days.push(getDateFromDaysAgo(i));
  }

  const nodes = ['worker-east1-01', 'worker-east1-02', 'worker-east1-03', 'master-east1-01'];

  // 创建更真实的数据
  const data = [
    [0, 0, 65], [0, 1, 42], [0, 2, 38], [0, 3, 30],
    [1, 0, 68], [1, 1, 45], [1, 2, 40], [1, 3, 33],
    [2, 0, 72], [2, 1, 48], [2, 2, 42], [2, 3, 35],
    [3, 0, 70], [3, 1, 50], [3, 2, 45], [3, 3, 32],
    [4, 0, 63], [4, 1, 52], [4, 2, 48], [4, 3, 30],
    [5, 0, 58], [5, 1, 48], [5, 2, 45], [5, 3, 28]
  ];

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      position: 'top',
      formatter: function (params: any) {
        return `${nodes[params.value[1]]} 在 ${days[params.value[0]]} 的负载: ${params.value[2]}%`;
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
      data: days,
      splitArea: {
        show: true
      },
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
      type: 'category',
      data: nodes,
      splitArea: {
        show: true
      },
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      },
      axisLabel: {
        color: 'inherit'
      }
    },
    visualMap: {
      min: 0,
      max: 100,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: '0%',
      textStyle: {
        color: 'inherit'
      },
      inRange: {
        color: ['#313695', '#4575b4', '#74add1', '#abd9e9', '#e0f3f8', '#ffffbf', '#fee090', '#fdae61', '#f46d43', '#d73027', '#a50026']
      }
    },
    series: [
      {
        name: '节点负载',
        type: 'heatmap',
        data: data,
        label: {
          show: false
        },
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  };

  chart.setOption(option);
  window.addEventListener('resize', () => chart.resize());
};

// 初始化事件趋势图表
const initEventTrendChart = () => {
  const chart = echarts.init(eventTrendChart.value);

  // 使用过去5天的日期作为x轴数据
  const days = [];
  for (let i = 5; i >= 0; i--) {
    days.push(getDateFromDaysAgo(i));
  }

  // 真实的事件趋势数据
  const infoData = [3, 5, 4, 7, 6, 4];
  const warningData = [1, 0, 2, 3, 1, 0];
  const errorData = [0, 0, 0, 1, 0, 0];

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['信息', '警告', '错误'],
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
      data: days,
      axisLine: {
        lineStyle: {
          color: 'inherit'
        }
      },
      axisLabel: {
        color: 'inherit',
        fontSize: 12
      },
      axisTick: {
        alignWithLabel: true
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
        color: 'inherit',
        fontSize: 12
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(127, 127, 127, 0.2)'
        }
      }
    },
    series: [
      {
        name: '信息',
        type: 'bar',
        stack: 'total',
        emphasis: {
          focus: 'series'
        },
        data: infoData,
        itemStyle: {
          color: '#1890ff'
        }
      },
      {
        name: '警告',
        type: 'bar',
        stack: 'total',
        emphasis: {
          focus: 'series'
        },
        data: warningData,
        itemStyle: {
          color: '#faad14'
        }
      },
      {
        name: '错误',
        type: 'bar',
        stack: 'total',
        emphasis: {
          focus: 'series'
        },
        data: errorData,
        itemStyle: {
          color: '#f5222d'
        }
      }
    ]
  };

  chart.setOption(option);
  window.addEventListener('resize', () => chart.resize());
};

// 初始化数据和图表
onMounted(() => {
  pods.value = generatePods();
  nodes.value = generateNodes();
  events.value = generateEvents();

  initResourceChart();
  initPodStatusChart();
  initNodeHeatmapChart();
  initEventTrendChart();
});

// 刷新数据
const refreshData = () => {
  showMessage('数据刷新', '数据已更新至最新状态');
  
  // 刷新图表
  initResourceChart();
  initPodStatusChart();
  initNodeHeatmapChart();
  initEventTrendChart();
};
</script>

<style scoped>
.k8s-aiops-container {
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
  color: #f5222d;
}

.chart-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
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

.optimization-card {
  border-radius: 8px;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
}

.suggestion-title {
  font-weight: bold;
  color: var(--ant-heading-color);
}

.suggestion-description {
  margin-bottom: 8px;
  color: var(--ant-text-color);
}

.suggestion-metrics {
  display: flex;
  gap: 10px;
  font-size: 12px;
}

.metric {
  display: flex;
  align-items: center;
  gap: 5px;
}

.metric .up {
  color: #52c41a;
}

.metric .down {
  color: #f5222d;
}

.monitoring-card {
  border-radius: 8px;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
}

.event-item {
  color: var(--ant-text-color);
}

.event-time {
  font-size: 12px;
  color: var(--ant-text-color-secondary);
}

.event-content {
  font-weight: bold;
  color: var(--ant-heading-color);
}

.event-source {
  font-size: 12px;
  color: var(--ant-text-color-secondary);
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