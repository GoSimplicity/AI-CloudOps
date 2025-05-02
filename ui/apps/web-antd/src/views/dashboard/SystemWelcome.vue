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
        <a-col :span="6">
          <a-card class="stat-card" :bordered="false">
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
        <a-col :span="6">
          <a-card class="stat-card" :bordered="false">
            <div class="stat-header">
              <span class="stat-title">云资源使用率</span>
              <a-tag color="processing">{{ resourceStatus }}</a-tag>
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
        <a-col :span="6">
          <a-card class="stat-card" :bordered="false">
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
        <a-col :span="6">
          <a-card class="stat-card" :bordered="false">
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
        <a-col :span="16">
          <a-card title="AI 智能运维分析" :bordered="false">
            <template #extra>
              <a-space>
                <a-radio-group
                  v-model:value="timeRange"
                  size="small"
                  button-style="solid"
                >
                  <a-radio-button value="day">今日</a-radio-button>
                  <a-radio-button value="week">本周</a-radio-button>
                  <a-radio-button value="month">本月</a-radio-button>
                </a-radio-group>
              </a-space>
            </template>
            <div ref="analysisChart" style="height: 300px"></div>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card title="实时监控动态" :bordered="false">
            <a-timeline>
              <a-timeline-item
                v-for="(item, index) in timelineItems"
                :key="index"
                :color="item.color"
              >
                {{ item.content }}
                <div class="timeline-time">{{ item.time }}</div>
              </a-timeline-item>
            </a-timeline>
          </a-card>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import * as echarts from 'echarts';

const currentDate = ref('');
const currentTime = ref('');
const onlineTime = ref('1分钟内');

// 动态数据
const accuracyRate = ref(98.7);
const accuracyIncrease = ref(8);
const resourceRate = ref(78.3);
const resourceStatus = ref('优化建议');
const healthRate = ref(95.2);
const healthStatus = ref({ color: 'success', text: '良好' });
const alertRate = ref(89.5);
const alertStatus = ref({ color: 'warning', text: '3个待处理' });

// 时间轴数据
const timelineItems = ref([
  {
    color: 'green',
    content: 'AI预测：系统负载将在2小时后达到峰值',
    time: '10分钟前',
  },
  {
    color: 'blue',
    content: '自动扩容：已添加2个新的服务节点',
    time: '30分钟前',
  },
  {
    color: 'red',
    content: '检测到异常流量，已自动启动防护',
    time: '1小时前',
  },
  {
    color: 'gray',
    content: '日常巡检完成，系统运行正常',
    time: '2小时前',
  },
]);

// 更新日期和时间
const updateDateTime = () => {
  const now = new Date();
  currentDate.value = now.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  });

  currentTime.value = now.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
};

// 更新动态数据
const updateDynamicData = () => {
  // 随机波动数据
  accuracyRate.value = +(95 + Math.random() * 4).toFixed(1);
  accuracyIncrease.value = +(5 + Math.random() * 5).toFixed(1);
  resourceRate.value = +(70 + Math.random() * 15).toFixed(1);
  healthRate.value = +(90 + Math.random() * 8).toFixed(1);
  alertRate.value = +(85 + Math.random() * 10).toFixed(1);

  // 更新在线时长
  const minutes = Math.floor(Date.now() / 60000) % 60;
  onlineTime.value = `${minutes}分钟内`;

  // 更新图表
  updateCharts();
};

// 初始更新并设置定时器
updateDateTime();
setInterval(updateDateTime, 1000);
setInterval(updateDynamicData, 5000);

const timeRange = ref('day');

const accuracyChart = ref();
const resourceChart = ref();
const healthChart = ref();
const alertChart = ref();
const analysisChart = ref();

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

onMounted(() => {
  // 初始化圆环图表
  const initGaugeChart = (el: HTMLElement, value: number, color: string) => {
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
  };

  // 初始化趋势图表
  const initAnalysisChart = () => {
    const chart = echarts.init(analysisChart.value);
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
        data: [
          '00:00',
          '03:00',
          '06:00',
          '09:00',
          '12:00',
          '15:00',
          '18:00',
          '21:00',
        ],
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
          data: [30, 40, 35, 50, 45, 65, 55, 40],
        },
        {
          name: '资源使用',
          type: 'line',
          smooth: true,
          data: [45, 50, 40, 60, 55, 75, 65, 50],
        },
        {
          name: '告警数量',
          type: 'line',
          smooth: true,
          data: [5, 3, 4, 8, 6, 4, 3, 2],
        },
      ],
    });

    // 定时更新趋势图数据
    setInterval(() => {
      chart.setOption({
        series: [
          {
            data: Array(8)
              .fill(0)
              .map(() => Math.floor(30 + Math.random() * 40)),
          },
          {
            data: Array(8)
              .fill(0)
              .map(() => Math.floor(40 + Math.random() * 40)),
          },
          {
            data: Array(8)
              .fill(0)
              .map(() => Math.floor(2 + Math.random() * 8)),
          },
        ],
      });
    }, 50000);
  };

  initGaugeChart(accuracyChart.value, accuracyRate.value, '#87d068');
  initGaugeChart(resourceChart.value, resourceRate.value, '#1890ff');
  initGaugeChart(healthChart.value, healthRate.value, '#87d068');
  initGaugeChart(alertChart.value, alertRate.value, '#ffc53d');
  initAnalysisChart();
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
}

.overview-section {
  margin-top: 24px;
}

.timeline-time {
  font-size: 12px;
  color: var(--ant-text-color-secondary);
  margin-top: 4px;
}

:deep(.ant-card) {
  border: none;
  color: var(--ant-text-color);
}

:deep(.ant-card-head) {
  color: var(--ant-heading-color);
  border-bottom: 1px solid var(--ant-border-color-split);
}

:deep(.ant-timeline-item-content) {
  color: var(--ant-text-color);
}
</style>
