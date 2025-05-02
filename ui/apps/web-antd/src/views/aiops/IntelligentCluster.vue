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
                <a-button type="primary" ghost>应用</a-button>
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
                    <a-button type="link" size="small">详情</a-button>
                    <a-button type="link" size="small">重启</a-button>
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
                    <a-button type="link" size="small">详情</a-button>
                    <a-button type="link" size="small">维护</a-button>
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

// 时间范围选择
const timeRange = ref('24h');

// 集群统计数据
const clusterStats = reactive({
  health: 96,
  healthImprovement: 2.5,
  podAvailability: 99.2,
  podTrend: 0.8,
  abnormalNodes: 2,
  nodeTrend: -50,
  resourceUtilization: 78,
  utilizationImprovement: 5.3
});

// 图表引用
const resourceChart = ref(null);
const podStatusChart = ref(null);
const nodeHeatmapChart = ref(null);
const eventTrendChart = ref(null);

// 优化建议数据
const optimizationSuggestions = ref([
  {
    type: 'resource',
    title: '优化 Deployment 资源配置',
    description: '检测到 frontend-app 部署组的资源请求过高，建议根据历史使用情况调整 CPU 和内存限制，可节省约 25% 的资源消耗。',
    impact: 25,
    difficulty: '低',
    color: '#1890ff'
  },
  {
    type: 'performance',
    title: '水平扩展数据处理服务',
    description: '数据处理服务 data-processor 负载持续超过 85%，建议增加副本数量从 3 扩展到 5，以提高处理能力和响应速度。',
    impact: 40,
    difficulty: '中',
    color: '#722ed1'
  },
  {
    type: 'security',
    title: '更新过期的 Secret 凭证',
    description: '发现 database-credentials Secret 已超过 90 天未更新，存在安全风险，建议立即轮换更新相关凭证。',
    impact: 15,
    difficulty: '中',
    color: '#f5222d'
  },
  {
    type: 'stability',
    title: '调整 Pod 反亲和性策略',
    description: '核心服务 api-gateway 的所有实例集中在同一节点，建议配置 podAntiAffinity 确保实例分散在不同节点，提高可用性。',
    impact: 30,
    difficulty: '低',
    color: '#52c41a'
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

// 生成随机 Pod 数据
const generatePods = (): Pod[] => {
  const statuses = ['Running', 'Pending', 'Failed', 'Succeeded', 'Unknown'];
  const namespaces = ['default', 'kube-system', 'monitoring', 'app', 'database'];
  const pods: Pod[] = [];

  for (let i = 1; i <= 20; i++) {
    const status = statuses[Math.floor(Math.random() * 5)];
    const cpu = Math.floor(Math.random() * 100);
    const memory = Math.floor(Math.random() * 100);

    pods.push({
      key: i,
      name: `pod-${i < 10 ? '0' + i : i}`,
      namespace: namespaces[Math.floor(Math.random() * namespaces.length)] || 'default',
      status: status || 'Unknown',
      ready: `${Math.floor(Math.random() * 3) + 1}/${Math.floor(Math.random() * 3) + 1}`,
      restarts: Math.floor(Math.random() * 5),
      cpu,
      memory,
      age: `${Math.floor(Math.random() * 30) + 1}d`
    });
  }

  return pods;
};

// 生成随机节点数据
const generateNodes = (): Node[] => {
  const statuses = ['Ready', 'NotReady', 'SchedulingDisabled'];
  const roles = ['master', 'worker', 'master,worker'];
  const nodes: Node[] = [];

  for (let i = 1; i <= 10; i++) {
    const status = statuses[Math.floor(Math.random() * 3)];
    const cpu = Math.floor(Math.random() * 100);
    const memory = Math.floor(Math.random() * 100);
    const disk = Math.floor(Math.random() * 100);

    nodes.push({
      key: i,
      name: `node-${i < 10 ? '0' + i : i}`,
      status: status || 'Unknown',
      roles: roles[Math.floor(Math.random() * roles.length)] || 'Unknown',
      cpu,
      memory,
      disk,
      pods: `${Math.floor(Math.random() * 50) + 10}/${Math.floor(Math.random() * 50) + 60}`
    });
  }

  return nodes;
};

// 生成随机事件数据
const generateEvents = (): Event[] => {
  const levels = ['info', 'warning', 'error', 'success'];
  const contents = [
    'Pod 启动成功',
    '节点 CPU 使用率超过阈值',
    '容器 OOM 被杀死',
    '服务自动扩展成功',
    '节点不可达',
    'ConfigMap 更新完成',
    'Secret 创建成功',
    'PVC 绑定成功',
    'Deployment 滚动更新完成',
    'Service Endpoint 变更'
  ];
  const sources = ['kubelet', 'scheduler', 'controller-manager', 'api-server', 'kube-proxy'];
  const events: Event[] = [];

  const now = new Date();

  for (let i = 0; i < 10; i++) {
    const level = levels[Math.floor(Math.random() * levels.length)];
    const eventTime = new Date(now.getTime() - Math.floor(Math.random() * 3600000));

    events.push({
      level: level || 'info',
      content: contents[Math.floor(Math.random() * contents.length)] || 'Pod 启动成功',
      source: sources[Math.floor(Math.random() * sources.length)] || 'kubelet',
      time: eventTime.toLocaleTimeString()
    });
  }

  return events.sort((a, b) => new Date(a.time).getTime() - new Date(b.time).getTime());
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

// 数据引用
const pods = ref<Pod[]>([]);
const nodes = ref<Node[]>([]);
const events = ref<Event[]>([]);

// 初始化资源使用趋势图表
const initResourceChart = () => {
  const chart = echarts.init(resourceChart.value);

  const hours: string[] = [];
  const now = new Date();
  for (let i = 24; i >= 0; i--) {
    const time = new Date(now.getTime() - i * 3600 * 1000);
    hours.push(time.getHours() + ':00');
  }

  const cpuData = [];
  const memoryData = [];
  const diskData = [];

  for (let i = 0; i < 25; i++) {
    cpuData.push((Math.random() * 30 + 50).toFixed(1));
    memoryData.push((Math.random() * 20 + 60).toFixed(1));
    diskData.push((Math.random() * 10 + 70).toFixed(1));
  }

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
          color: 'inherit'  // 使用继承的颜色，适应主题
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
      data: hours,
      axisLine: {
        lineStyle: {
          color: 'inherit'  // 使用继承的颜色，适应主题
        }
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: 'inherit'  // 使用继承的颜色，适应主题
        }
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(127, 127, 127, 0.2)'  // 使用半透明颜色，适应主题
        }
      },
      axisLabel: {
        formatter: '{value}%',
        color: 'inherit'  // 使用继承的颜色，适应主题
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
        color: 'inherit'  // 使用继承的颜色，适应主题
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
            color: 'inherit'  // 使用继承的颜色，适应主题
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: 735, name: 'Running', itemStyle: { color: '#52c41a' } },
          { value: 58, name: 'Pending', itemStyle: { color: '#1890ff' } },
          { value: 12, name: 'Failed', itemStyle: { color: '#f5222d' } },
          { value: 34, name: 'Succeeded', itemStyle: { color: '#13c2c2' } },
          { value: 8, name: 'Unknown', itemStyle: { color: '#faad14' } }
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

  const hours = ['00:00', '01:00', '02:00', '03:00', '04:00', '05:00', '06:00', '07:00', '08:00', '09:00', '10:00', '11:00',
    '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00', '19:00', '20:00', '21:00', '22:00', '23:00'];

  const nodes = ['node-01', 'node-02', 'node-03', 'node-04', 'node-05', 'node-06', 'node-07', 'node-08'];

  const data = [];
  for (let i = 0; i < nodes.length; i++) {
    for (let j = 0; j < hours.length; j++) {
      data.push([j, i, Math.round(Math.random() * 100)]);
    }
  }

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      position: 'top',
      formatter: function (params: any) {
        return `${nodes[params.value[1]]} 在 ${hours[params.value[0]]} 的负载: ${params.value[2]}%`;
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
      splitArea: {
        show: true
      },
      axisLine: {
        lineStyle: {
          color: 'inherit'  // 使用继承的颜色，适应主题
        }
      },
      axisLabel: {
        interval: 3,
        color: 'inherit'  // 使用继承的颜色，适应主题
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
          color: 'inherit'  // 使用继承的颜色，适应主题
        }
      },
      axisLabel: {
        color: 'inherit'  // 使用继承的颜色，适应主题
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
        color: 'inherit'  // 使用继承的颜色，适应主题
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

  const hours = [];
  const now = new Date();
  for (let i = 24; i >= 0; i--) {
    const time = new Date(now.getTime() - i * 3600 * 1000);
    hours.push(time.getHours() + ':00');
  }

  const infoData = [];
  const warningData = [];
  const errorData = [];

  for (let i = 0; i < 25; i++) {
    infoData.push(Math.floor(Math.random() * 10));
    warningData.push(Math.floor(Math.random() * 5));
    errorData.push(Math.floor(Math.random() * 3));
  }

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
        color: 'inherit'  // 使用继承的颜色，适应主题
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
          color: 'inherit'  // 使用继承的颜色，适应主题
        }
      },
      axisLabel: {
        color: 'inherit',  // 使用继承的颜色，适应主题
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
          color: 'inherit'  // 使用继承的颜色，适应主题
        }
      },
      axisLabel: {
        color: 'inherit',  // 使用继承的颜色，适应主题
        fontSize: 12
      },
      splitLine: {
        lineStyle: {
          color: 'rgba(127, 127, 127, 0.2)'  // 使用半透明颜色，适应主题
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
  pods.value = generatePods();
  nodes.value = generateNodes();
  events.value = generateEvents();
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
