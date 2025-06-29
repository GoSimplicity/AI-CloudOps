<template>
  <div class="welcome-page dark">
    <!-- 顶部欢迎语 -->
    <div class="welcome-header">
      <h1 class="title rainbow-text">AI-CloudOps 智能运维平台</h1>
      <p class="subtitle">让云运维更智能、更高效</p>
      <div class="time-info">
        <a-space>
          <a-tag color="blue">{{ currentDate }}</a-tag>
          <a-tag color="green">当前时间: {{ currentTime }}</a-tag>
          <a-tag color="green">在线时长: {{ onlineTime }}</a-tag>
        </a-space>
      </div>
    </div>

    <!-- 核心指标卡片 -->
    <div class="statistics-cards">
      <a-row :gutter="[16, 16]">
        <a-col :span="6" :xs="24" :sm="12" :md="6">
          <a-card class="stat-card" :bordered="false" hoverable>
            <div class="stat-header">
              <span class="stat-title">AI 预测准确率</span>
              <a-tag color="success">同比上升{{ accuracyIncrease }}%</a-tag>
            </div>
            <div class="stat-body">
              <span class="stat-number">{{ accuracyRate }}%</span>
              <div
                ref="accuracyChart"
                style="width: 100px; height: 100px"
              ></div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="6" :xs="24" :sm="12" :md="6">
          <a-card class="stat-card" :bordered="false" hoverable>
            <div class="stat-header">
              <span class="stat-title">云资源使用率</span>
              <a-tag :color="getResourceTagColor">{{ resourceStatus }}</a-tag>
            </div>
            <div class="stat-body">
              <span class="stat-number">{{ resourceRate }}%</span>
              <div
                ref="resourceChart"
                style="width: 100px; height: 100px"
              ></div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="6" :xs="24" :sm="12" :md="6">
          <a-card class="stat-card" :bordered="false" hoverable>
            <div class="stat-header">
              <span class="stat-title">系统健康度</span>
              <a-tag :color="healthStatus.color">{{ healthStatus.text }}</a-tag>
            </div>
            <div class="stat-body">
              <span class="stat-number">{{ healthRate }}%</span>
              <div ref="healthChart" style="width: 100px; height: 100px"></div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="6" :xs="24" :sm="12" :md="6">
          <a-card class="stat-card" :bordered="false" hoverable>
            <div class="stat-header">
              <span class="stat-title">智能告警处理</span>
              <a-tag :color="alertStatus.color">{{ alertStatus.text }}</a-tag>
            </div>
            <div class="stat-body">
              <span class="stat-number">{{ alertRate }}%</span>
              <div ref="alertChart" style="width: 100px; height: 100px"></div>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <!-- 运维概览 -->
    <div class="overview-section">
      <a-row :gutter="[16, 16]">
        <a-col :span="16" :xs="24" :md="16">
          <a-card title="AI 智能运维分析" :bordered="false" hoverable>
            <template #extra>
              <a-space>
                <a-radio-group
                  v-model:value="timeRange"
                  size="small"
                  button-style="solid"
                  @change="handleTimeRangeChange"
                >
                  <a-radio-button value="day">今日</a-radio-button>
                  <a-radio-button value="week">本周</a-radio-button>
                  <a-radio-button value="month">本月</a-radio-button>
                </a-radio-group>
              </a-space>
            </template>
            <div ref="analysisChart" style="height: 300px; width: 100%"></div>
          </a-card>
        </a-col>
        <a-col :span="8" :xs="24" :md="8">
          <a-card title="实时监控动态" :bordered="false" class="monitor-card" hoverable>
            <a-list size="small" class="monitor-list">
              <a-list-item class="monitor-item monitor-item-warning">
                <div class="monitor-content">
                  <span class="monitor-time">08:15:32  </span>
                  <span class="monitor-event">CPU使用率峰值达到87.3%，已触发自动扩容</span>
                </div>
                <a-tag color="orange" class="monitor-tag pulse-animation">处理中</a-tag>
              </a-list-item>
              <a-list-item class="monitor-item monitor-item-error">
                <div class="monitor-content">
                  <span class="monitor-time">08:03:17  </span>
                  <span class="monitor-event">数据库连接池使用率达到92%，建议优化查询</span>
                </div>
                <a-tag color="red" class="monitor-tag blink-animation">紧急</a-tag>
              </a-list-item>
              <a-list-item class="monitor-item monitor-item-info">
                <div class="monitor-content">
                  <span class="monitor-time">07:58:45  </span>
                  <span class="monitor-event">检测到网络延迟增加，平均响应时间215ms</span>
                </div>
                <a-tag color="blue" class="monitor-tag">信息</a-tag>
              </a-list-item>
              <a-list-item class="monitor-item monitor-item-success fade-in">
                <div class="monitor-content">
                  <span class="monitor-time">07:42:09  </span>
                  <span class="monitor-event">实例自动扩展：2 → 4，负载均衡已完成</span>
                </div>
                <a-tag color="green" class="monitor-tag">已完成</a-tag>
              </a-list-item>
              <a-list-item class="monitor-item monitor-item-normal slide-in">
                <div class="monitor-content">
                  <span class="monitor-time">07:30:22  </span>
                  <span class="monitor-event">系统QPS达到峰值：93，运行正常</span>
                </div>
                <a-tag color="default" class="monitor-tag">正常</a-tag>
              </a-list-item>
            </a-list>
            <div class="monitor-refresh-indicator">
              <a-spin size="small" />
              <span class="refresh-text">实时更新中</span>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue';
import * as echarts from 'echarts';

const currentDate = ref('');
const currentTime = ref('');
const onlineTime = ref('');
const startTime = Date.now();

// 动态数据
const accuracyRate = ref(97.2);
const accuracyIncrease = ref(6.3);
const resourceRate = ref(68.4);
const resourceStatus = ref('优化建议');
const healthRate = ref(94.5);
const healthStatus = ref({ color: 'success', text: '良好' });
const alertRate = ref(92.1);
const alertStatus = ref({ color: 'warning', text: '2个待处理' });

// 根据资源使用率设置不同的标签颜色
const getResourceTagColor = computed(() => {
  if (resourceRate.value > 85) return 'error';
  if (resourceRate.value > 70) return 'warning';
  return 'processing';
});

// 更新日期和时间
const updateDateTime = () => {
  const now = new Date();
  
  // 更真实的日期格式
  currentDate.value = now.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  });

  // 更真实的时间格式
  currentTime.value = now.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
  
  // 计算在线时长
  const elapsedMs = Date.now() - startTime;
  const hours = Math.floor(elapsedMs / 3600000);
  const minutes = Math.floor((elapsedMs % 3600000) / 60000);
  const seconds = Math.floor((elapsedMs % 60000) / 1000);
  
  if (hours > 0) {
    onlineTime.value = `${hours}小时${minutes}分钟`;
  } else if (minutes > 0) {
    onlineTime.value = `${minutes}分钟${seconds}秒`;
  } else {
    onlineTime.value = `${seconds}秒`;
  }
};

// 更新动态数据 - 使用更真实的变化模式
const updateDynamicData = () => {
  // 生成真实的小幅度波动数据，避免大幅波动
  const generateSmallChange = (base: number, maxChange = 0.5) => {
    return +(base + (Math.random() * 2 - 1) * maxChange).toFixed(1);
  };
  
  // 小幅度波动准确率 (95-99%)
  accuracyRate.value = Math.max(95, Math.min(99, generateSmallChange(accuracyRate.value, 0.3)));
  
  // 同比增长小幅波动 (5-8%)
  accuracyIncrease.value = Math.max(5, Math.min(8, generateSmallChange(accuracyIncrease.value, 0.2)));
  
  // 资源使用率波动 (60-85%)，有小的波动趋势
  const resourceTrend = Math.sin(Date.now() / 10000000) * 5; // 缓慢的波动趋势
  resourceRate.value = Math.max(60, Math.min(85, generateSmallChange(resourceRate.value + resourceTrend * 0.1, 0.6)));
  
  // 根据资源使用率更新状态
  if (resourceRate.value > 80) {
    resourceStatus.value = '需要优化';
  } else if (resourceRate.value > 70) {
    resourceStatus.value = '优化建议';
  } else {
    resourceStatus.value = '运行良好';
  }
  
  // 系统健康度 (90-99%)
  healthRate.value = Math.max(90, Math.min(99, generateSmallChange(healthRate.value, 0.2)));
  
  // 根据健康度更新状态文本
  if (healthRate.value > 95) {
    healthStatus.value = { color: 'success', text: '优良' };
  } else if (healthRate.value > 90) {
    healthStatus.value = { color: 'success', text: '良好' };
  } else {
    healthStatus.value = { color: 'warning', text: '一般' };
  }
  
  // 智能告警处理率 (85-98%)
  alertRate.value = Math.max(85, Math.min(98, generateSmallChange(alertRate.value, 0.4)));
  
  // 模拟不同的告警数量和状态
  const pendingAlerts = Math.floor(Math.random() * 5);
  if (pendingAlerts === 0) {
    alertStatus.value = { color: 'success', text: '全部处理' };
  } else if (pendingAlerts === 1) {
    alertStatus.value = { color: 'processing', text: '1个待处理' };
  } else {
    alertStatus.value = { color: 'warning', text: `${pendingAlerts}个待处理` };
  }
  
  // 更新图表
  updateCharts();
};

// 定义响应式变量
const dataLoaded = ref(true);
const timelineItems = ref<Array<{
  color: string;
  content: string;
  time: string;
}>>([]);

// 定期更新时间轴数据
const updateTimeline = () => {
  // 有5%的概率显示"暂未获取监控数据"状态
  if (Math.random() < 0.05 && dataLoaded.value) {
    dataLoaded.value = false;
    timelineItems.value = [];
    
    // 5秒后恢复数据
    setTimeout(() => {
      dataLoaded.value = true;
      generateTimeline();
    }, 5000);
  } else if (!dataLoaded.value) {
    // 已经在无数据状态，不做处理
  } else {
    generateTimeline();
  }
};

// 生成新的时间轴数据
const generateTimeline = () => {
  // 事件类型
  const eventTypes = [
    { color: 'green', content: 'AI预测：系统负载将在{n}小时后达到峰值', probability: 0.2 },
    { color: 'blue', content: '自动扩容：已添加{n}个新的服务节点', probability: 0.15 },
    { color: 'orange', content: '检测到网络延迟略有波动，已自动优化', probability: 0.1 },
    { color: 'red', content: '发现异常访问模式，已启动安全防护', probability: 0.05 },
    { color: 'gray', content: '系统定时备份完成，数据完整性校验通过', probability: 0.1 },
    { color: 'purple', content: 'AI模型更新完成，预测准确率提升{n}%', probability: 0.1 },
    { color: 'cyan', content: '智能调度：已优化{n}个容器资源分配', probability: 0.15 },
    { color: 'green', content: '流量分析：用户访问高峰期预计在{n}分钟后到来', probability: 0.15 }
  ];

  // 随机生成一个新事件
  if (Math.random() < 0.3 && timelineItems.value.length > 0) {  // 30%概率添加新事件
    // 基于概率选择事件类型
    let randomValue = Math.random();
    let cumulativeProbability = 0;
    let selectedEvent = eventTypes[0];
    
    for (const eventType of eventTypes) {
      cumulativeProbability += eventType.probability;
      if (randomValue <= cumulativeProbability) {
        selectedEvent = eventType;
        break;
      }
    }
    
    // 替换占位符
    if (!selectedEvent) {
      return;
    }
    
    let content = selectedEvent.content;
    if (content.includes('{n}')) {
      const num = Math.floor(Math.random() * 5) + 1;
      content = content.replace('{n}', num.toString());
    }
    
    // 创建新事件并添加到顶部
    const newEvent = {
      color: selectedEvent.color,
      content: content,
      time: '刚刚'
    };
    
    timelineItems.value.unshift(newEvent);
    
    // 更新旧事件的时间
    updateEventTimes();
    
    // 保持最多显示5条
    if (timelineItems.value.length > 5) {
      timelineItems.value.pop();
    }
  } else {
    // 仅更新事件时间
    updateEventTimes();
  }
};

// 更新事件时间
const updateEventTimes = () => {
  type TimeTerm = '刚刚' | '1分钟前' | '3分钟前' | '7分钟前' | '12分钟前' | '18分钟前' | 
                 '25分钟前' | '32分钟前' | '47分钟前' | '1小时前' | '1小时23分钟前' | 
                 '1小时45分钟前' | '2小时15分钟前' | '3小时前';
                 
  const timeTerms: TimeTerm[] = ['刚刚', '1分钟前', '3分钟前', '7分钟前', '12分钟前', '18分钟前', 
                    '25分钟前', '32分钟前', '47分钟前', '1小时前', '1小时23分钟前', 
                    '1小时45分钟前', '2小时15分钟前', '3小时前'];
  
  timelineItems.value.forEach((item: { time: string }, index: number) => {
    if (item.time === '刚刚' && index > 0) {
      item.time = '1分钟前';
    } else if (item.time !== '刚刚') {
      // 找到当前时间在数组中的位置
      const currentIndex = timeTerms.indexOf(item.time as TimeTerm);
      if (currentIndex !== -1 && currentIndex < timeTerms.length - 1) {
        // 随机决定是否更新时间（70%概率）
        if (Math.random() < 0.7) {
          const nextTime = timeTerms[currentIndex + 1];
          if (nextTime) {
            item.time = nextTime;
          }
        }
      }
    }
  });
};

// 初始更新并设置定时器
updateDateTime();
setInterval(updateDateTime, 600000);
setInterval(updateDynamicData, 600000);
setInterval(updateTimeline, 600000);

const timeRange = ref('day');

const accuracyChart = ref();
const resourceChart = ref();
const healthChart = ref();
const alertChart = ref();
const analysisChart = ref();

// 处理时间范围变化
const handleTimeRangeChange = () => {
  if (analysisChart.value) {
    updateAnalysisChart();
  }
};

// 更新图表数据
const updateCharts = () => {
  if (accuracyChart.value) {
    const chart = echarts.getInstanceByDom(accuracyChart.value);
    chart?.setOption({
      series: [
        {
          data: [
            {
              value: accuracyRate.value,
            },
          ],
        },
      ],
    });
  }

  if (resourceChart.value) {
    const chart = echarts.getInstanceByDom(resourceChart.value);
    chart?.setOption({
      series: [
        {
          data: [
            {
              value: resourceRate.value,
            },
          ],
        },
      ],
    });
  }

  if (healthChart.value) {
    const chart = echarts.getInstanceByDom(healthChart.value);
    chart?.setOption({
      series: [
        {
          data: [
            {
              value: healthRate.value,
            },
          ],
        },
      ],
    });
  }

  if (alertChart.value) {
    const chart = echarts.getInstanceByDom(alertChart.value);
    chart?.setOption({
      series: [
        {
          data: [
            {
              value: alertRate.value,
            },
          ],
        },
      ],
    });
  }
};

// 生成更真实的分析图表数据
const generateAnalysisData = (range: string) => {
  const now = new Date();
  
  let xAxisData: string[] = [];
  let dataPoints = 0;
  
  // 根据选择的时间范围设置X轴数据
  if (range === 'day') {
    xAxisData = ['00:00', '03:00', '06:00', '09:00', '12:00', '15:00', '18:00', '21:00', '23:59'];
    dataPoints = 9;
  } else if (range === 'week') {
    xAxisData = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
    dataPoints = 7;
  } else if (range === 'month') {
    xAxisData = ['1日', '5日', '10日', '15日', '20日', '25日', '30日'];
    dataPoints = 7;
  }
  
  // 模拟系统负载 - 工作时间较高的模式
  const loadData = Array(dataPoints).fill(0).map((_, i) => {
    if (range === 'day') {
      // 日负载: 9-18点较高
      const timeHour = Math.floor(i * 24 / (dataPoints - 1));
      return timeHour >= 9 && timeHour <= 18 
        ? 50 + Math.random() * 30 
        : 20 + Math.random() * 20;
    } else if (range === 'week') {
      // 周负载: 工作日较高
      return i < 5 ? 60 + Math.random() * 20 : 30 + Math.random() * 15;
    } else {
      // 月负载: 工作日较高
      return 40 + Math.random() * 30;
    }
  });
  
  // 模拟资源使用 - 与负载相关但略高
  const resourceData = loadData.map(load => Math.min(95, load * 1.2 + Math.random() * 10));
  
  // 模拟告警数量 - 负载高的时候告警可能更多
  const alertData = loadData.map(load => Math.max(0, Math.floor((load - 30) / 20 + Math.random() * 3)));
  
  return {
    xAxisData,
    loadData,
    resourceData,
    alertData
  };
};

// 更新分析图表
const updateAnalysisChart = () => {
  if (!analysisChart.value) return;
  
  const chart = echarts.getInstanceByDom(analysisChart.value);
  if (!chart) return; // 添加空值检查
  
  const data = generateAnalysisData(timeRange.value);
  
  chart.setOption({
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      data: ['系统负载', '资源使用', '告警数量'],
      textStyle: {
        color: '#a6a6a6',
      },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data.xAxisData,
      axisLine: {
        lineStyle: {
          color: '#a6a6a6',
        },
      },
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: '#a6a6a6',
        },
      },
      splitLine: {
        lineStyle: {
          color: '#303030',
        },
      },
    },
    series: [
      {
        name: '系统负载',
        type: 'line',
        smooth: true,
        data: data.loadData,
      },
      {
        name: '资源使用',
        type: 'line',
        smooth: true,
        data: data.resourceData,
      },
      {
        name: '告警数量',
        type: 'line',
        smooth: true,
        data: data.alertData,
      },
    ],
  });
};

onMounted(() => {
  // 初始化圆环图表
  const initGaugeChart = (el: HTMLElement | null | undefined, value: number, color: string) => {
    const chart = echarts.init(el);
    chart.setOption({
      series: [
        {
          type: 'gauge',
          startAngle: 90,
          endAngle: -270,
          pointer: {
            show: false,
          },
          progress: {
            show: true,
            overlap: false,
            roundCap: true,
            clip: false,
            itemStyle: {
              color,
            },
          },
          axisLine: {
            lineStyle: {
              width: 10,
            },
          },
          splitLine: {
            show: false,
          },
          axisTick: {
            show: false,
          },
          axisLabel: {
            show: false,
          },
          data: [
            {
              value,
              name: '',
              detail: {
                show: false,
              },
            },
          ],
          detail: {
            show: false,
          },
        },
      ],
    });
    return chart;
  };

  // 初始化各图表
  initGaugeChart(accuracyChart.value, accuracyRate.value, '#87d068');
  initGaugeChart(resourceChart.value, resourceRate.value, '#1890ff');
  initGaugeChart(healthChart.value, healthRate.value, '#87d068');
  initGaugeChart(alertChart.value, alertRate.value, '#ffc53d');
  
  // 设置分析图表
  if (analysisChart.value) {
    const chart = echarts.init(analysisChart.value);
    const data = generateAnalysisData(timeRange.value);
    
    chart.setOption({
      tooltip: {
        trigger: 'axis',
      },
      legend: {
        data: ['系统负载', '资源使用', '告警数量'],
        textStyle: {
          color: '#a6a6a6',
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: data.xAxisData,
        axisLine: {
          lineStyle: {
            color: '#a6a6a6',
          },
        },
      },
      yAxis: {
        type: 'value',
        axisLine: {
          lineStyle: {
            color: '#a6a6a6',
          },
        },
        splitLine: {
          lineStyle: {
            color: '#303030',
          },
        },
      },
      series: [
        {
          name: '系统负载',
          type: 'line',
          smooth: true,
          data: data.loadData,
        },
        {
          name: '资源使用',
          type: 'line',
          smooth: true,
          data: data.resourceData,
        },
        {
          name: '告警数量',
          type: 'line',
          smooth: true,
          data: data.alertData,
        },
      ],
    });
    
    // 定时更新分析图表数据（每10分钟）
    setInterval(() => {
      updateAnalysisChart();
    }, 600000);
  }
  
  // 立即触发一次时间轴更新以初始化数据
  updateTimeline();
});
</script>

<style scoped>
.welcome-page {
  padding: 24px;
  min-height: 100vh;
}

.welcome-header {
  text-align: center;
  margin-bottom: 32px;
  padding: 24px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
}

.title {
  font-size: 36px;
  font-weight: bold;
  margin-bottom: 12px;
  color: var(--ant-heading-color);
}

.rainbow-text {
  background: linear-gradient(
    124deg,
    #ff2400,
    #e81d1d,
    #e8b71d,
    #e3e81d,
    #1de840,
    #1ddde8,
    #2b1de8,
    #dd00f3,
    #dd00f3
  );
  background-size: 1800% 1800%;
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  animation: rainbow 18s ease infinite;
  text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
  letter-spacing: 1px;
}

@keyframes rainbow {
  0% {
    background-position: 0% 82%;
  }
  50% {
    background-position: 100% 19%;
  }
  100% {
    background-position: 0% 82%;
  }
}

.subtitle {
  font-size: 18px;
  color: var(--ant-text-color-secondary);
  margin-bottom: 16px;
}

.time-info {
  margin-top: 16px;
  color: var(--ant-text-color);
}

.stat-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 5px 15px rgba(0, 0, 0, 0.4);
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.stat-title {
  font-size: 16px;
  color: var(--ant-text-color-secondary);
}

.stat-body {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-number {
  font-size: 28px;
  font-weight: bold;
  color: var(--ant-primary-color);
  transition: color 0.3s ease;
}

.overview-section {
  margin-top: 24px;
}

.timeline-time {
  font-size: 12px;
  color: var(--ant-text-color-secondary);
  margin-top: 4px;
}

.no-data-message {
  text-align: center;
  padding: 20px;
  color: #ff4d4f;
  font-size: 14px;
  background: rgba(255, 77, 79, 0.1);
  border-radius: 4px;
  margin: 10px 0;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    opacity: 0.7;
  }
  50% {
    opacity: 1;
  }
  100% {
    opacity: 0.7;
  }
}

:deep(.ant-card) {
  border: none;
  color: var(--ant-text-color);
  transition: all 0.3s ease;
}

:deep(.ant-card-head) {
  color: var(--ant-heading-color);
  border-bottom: 1px solid var(--ant-border-color-split);
}

:deep(.ant-timeline-item-content) {
  color: var(--ant-text-color);
}

@media (max-width: 768px) {
  .welcome-header {
    padding: 16px;
  }
  
  .title {
    font-size: 28px;
  }
  
  .subtitle {
    font-size: 16px;
  }
}
</style>