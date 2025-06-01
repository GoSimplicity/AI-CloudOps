<template>
  <div class="statistics-container">
    <div class="page-header">
      <div class="header-actions">
        <a-range-picker 
          v-model:value="dateRange" 
          @change="handleDateRangeChange" 
          :ranges="dateRanges"
          style="width: 300px" 
        />
        <a-select 
          v-model:value="categoryFilter" 
          placeholder="选择分类" 
          style="width: 200px" 
          @change="handleFilterChange"
          allow-clear
        >
          <a-select-option :value="undefined">全部分类</a-select-option>
          <a-select-option v-for="category in categories" :key="category.id" :value="category.id">
            {{ category.name }}
          </a-select-option>
        </a-select>
        <a-select 
          v-model:value="userFilter" 
          placeholder="选择用户" 
          style="width: 200px" 
          @change="handleFilterChange"
          allow-clear
        >
          <a-select-option :value="undefined">全部用户</a-select-option>
          <a-select-option v-for="user in users" :key="user.id" :value="user.id">
            {{ user.name }}
          </a-select-option>
        </a-select>
        <a-button @click="refreshData" :loading="loading">
          <template #icon>
            <ReloadOutlined />
          </template>
          刷新数据
        </a-button>
      </div>
    </div>

    <div v-if="isMounted" class="dashboard-container">
      <!-- 概览统计卡片 -->
      <a-row :gutter="16" class="stats-row">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic 
              title="总工单数" 
              :value="overviewStats.total_count" 
              :value-style="{ color: '#40a9ff' }"
            >
              <template #prefix>
                <FileDoneOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic 
              title="已完成" 
              :value="overviewStats.completed_count" 
              :value-style="{ color: '#52c41a' }"
            >
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
            <div class="completion-rate">
              完成率: {{ (overviewStats.completion_rate * 100).toFixed(1) }}%
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic 
              title="处理中" 
              :value="overviewStats.processing_count" 
              :value-style="{ color: '#faad14' }"
            >
              <template #prefix>
                <SyncOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic 
              title="平均处理时间" 
              :value="overviewStats.avg_process_time" 
              suffix="小时"
              :value-style="{ color: '#722ed1' }"
            >
              <template #prefix>
                <ClockCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>

      <!-- 工单趋势图和状态分布 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="16">
          <a-card title="工单趋势" class="chart-card">
            <div class="trend-controls">
              <a-radio-group v-model:value="trendDimension" @change="fetchTrendData">
                <a-radio-button value="day">按天</a-radio-button>
                <a-radio-button value="week">按周</a-radio-button>
                <a-radio-button value="month">按月</a-radio-button>
              </a-radio-group>
            </div>
            <a-spin :spinning="chartLoading.trend">
              <div v-if="hasData(trendData.dates)" class="trend-chart-container" ref="trendChartRef"></div>
              <div v-else class="empty-chart">
                <a-empty description="暂无趋势数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card title="工单状态分布" class="chart-card">
            <a-spin :spinning="chartLoading.status">
              <div v-if="hasData(statusDistributionData)" class="status-chart-container" ref="statusChartRef"></div>
              <div v-else class="empty-chart">
                <a-empty description="暂无状态分布数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>

      <!-- 分类统计和优先级分布 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="12">
          <a-card title="分类统计" class="chart-card">
            <a-spin :spinning="chartLoading.category">
              <div v-if="hasData(categoryStatsData)" class="category-chart-container" ref="categoryChartRef"></div>
              <div v-else class="empty-chart">
                <a-empty description="暂无分类统计数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="优先级分布" class="chart-card">
            <a-spin :spinning="chartLoading.priority">
              <div v-if="hasData(priorityDistributionData)" class="priority-chart-container" ref="priorityChartRef"></div>
              <div v-else class="empty-chart">
                <a-empty description="暂无优先级分布数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>

      <!-- 模板使用统计 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="24">
          <a-card title="模板使用统计" class="chart-card">
            <a-spin :spinning="chartLoading.template">
              <div v-if="hasData(templateStatsData)" class="template-chart-container" ref="templateChartRef"></div>
              <div v-else class="empty-chart">
                <a-empty description="暂无模板使用数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>

      <!-- 用户排行榜 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="12">
          <a-card title="用户处理数量排行" class="chart-card">
            <a-spin :spinning="chartLoading.user">
              <div v-if="hasData(userStatsData)">
                <a-table 
                  :dataSource="userStatsData" 
                  :pagination="false" 
                  :columns="userCountColumns" 
                  size="small"
                  :rowKey="(record: any) => `user-count-${record.user_id || record.user_name}`"
                >
                  <template #bodyCell="{ column, record, index }">
                    <template v-if="column.key === 'rank'">
                      <div class="rank-cell">
                        <a-tag :color="getRankColor(index)">第 {{ index + 1 }} 名</a-tag>
                      </div>
                    </template>
                    <template v-if="column.key === 'user'">
                      <div class="user-cell">
                        <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.user_name) }">
                          {{ getInitials(record.user_name) }}
                        </a-avatar>
                        <span>{{ record.user_name }}</span>
                      </div>
                    </template>
                    <template v-if="column.key === 'count'">
                      <a-progress 
                        :percent="getPercentage(record.completed_count, userStatsData[0]?.completed_count || 0)"
                        :show-info="false" 
                        status="active" 
                        :stroke-color="getProgressColor(index)" 
                      />
                      <span class="count-value">{{ record.completed_count }}</span>
                    </template>
                  </template>
                </a-table>
              </div>
              <div v-else class="empty-chart">
                <a-empty description="暂无用户处理数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="用户处理时间排行" class="chart-card">
            <a-spin :spinning="chartLoading.user">
              <div v-if="hasData(userTimeRanking)">
                <a-table 
                  :dataSource="userTimeRanking" 
                  :pagination="false" 
                  :columns="userTimeColumns" 
                  size="small"
                  :rowKey="(record: any) => `user-time-${record.user_id || record.user_name}`"
                >
                  <template #bodyCell="{ column, record, index }">
                    <template v-if="column.key === 'rank'">
                      <div class="rank-cell">
                        <a-tag :color="getRankColor(index, true)">第 {{ index + 1 }} 名</a-tag>
                      </div>
                    </template>
                    <template v-if="column.key === 'user'">
                      <div class="user-cell">
                        <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.user_name) }">
                          {{ getInitials(record.user_name) }}
                        </a-avatar>
                        <span>{{ record.user_name }}</span>
                      </div>
                    </template>
                    <template v-if="column.key === 'time'">
                      <a-progress 
                        :percent="getTimePercentage(record.avg_processing_time, userTimeRanking)"
                        :show-info="false" 
                        status="active" 
                        :stroke-color="getProgressColor(index, true)" 
                      />
                      <span class="time-value">{{ record.avg_processing_time }}小时</span>
                    </template>
                  </template>
                </a-table>
              </div>
              <div v-else class="empty-chart">
                <a-empty description="暂无用户处理时间数据" />
              </div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick, onBeforeUnmount, computed, getCurrentInstance } from 'vue';
import { message } from 'ant-design-vue';
import * as echarts from 'echarts';
import {
  FileDoneOutlined,
  CheckCircleOutlined,
  SyncOutlined,
  ClockCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons-vue';
import dayjs from 'dayjs';
import {
  getWorkorderOverview,
  getWorkorderTrend,
  getWorkorderCategoryStats,
  getWorkorderUserStats,
  getWorkorderTemplateStats,
  getWorkorderStatusDistribution,
  getWorkorderPriorityDistribution,
  type StatsReq,
  type OverviewStats,
  type TrendStats,
  type CategoryStats,
  type UserStats,
  type TemplateStats,
  type StatusDistribution,
  type PriorityDistribution
} from '#/api/core/workorder_statistic';

// 获取当前实例，用于生命周期检查
const instance = getCurrentInstance();

// 组件挂载状态
const isMounted = ref(false);

// 图表引用
const trendChartRef = ref<HTMLElement | null>(null);
const statusChartRef = ref<HTMLElement | null>(null);
const categoryChartRef = ref<HTMLElement | null>(null);
const priorityChartRef = ref<HTMLElement | null>(null);
const templateChartRef = ref<HTMLElement | null>(null);

// 图表实例
let trendChart: echarts.ECharts | null = null;
let statusChart: echarts.ECharts | null = null;
let categoryChart: echarts.ECharts | null = null;
let priorityChart: echarts.ECharts | null = null;
let templateChart: echarts.ECharts | null = null;

// 数据和过滤相关
const loading = ref(false);
const chartLoading = reactive({
  trend: false,
  status: false,
  category: false,
  priority: false,
  template: false,
  user: false
});

// 过滤条件
const categoryFilter = ref<number | undefined>(undefined);
const userFilter = ref<number | undefined>(undefined);
const trendDimension = ref<'day' | 'week' | 'month'>('day');
const dateRange = ref([dayjs().subtract(30, 'days'), dayjs()]);

const dateRanges = {
  '最近7天': [dayjs().subtract(7, 'days'), dayjs()],
  '最近30天': [dayjs().subtract(30, 'days'), dayjs()],
  '最近90天': [dayjs().subtract(90, 'days'), dayjs()],
  '今年': [dayjs().startOf('year'), dayjs()]
};

// 基础数据
const categories = ref([
  { id: 1, name: '技术支持' },
  { id: 2, name: '系统故障' },
  { id: 3, name: '账户问题' },
  { id: 4, name: '功能请求' }
]);

const users = ref([
  { id: 1, name: '张三' },
  { id: 2, name: '李四' },
  { id: 3, name: '王五' },
  { id: 4, name: '赵六' }
]);

// 统计数据
const overviewStats = ref<OverviewStats>({
  total_count: 0,
  completed_count: 0,
  processing_count: 0,
  pending_count: 0,
  overdue_count: 0,
  completion_rate: 0,
  avg_process_time: 0,
  avg_response_time: 0,
  today_created: 0,
  today_completed: 0
});

const trendData = ref<TrendStats>({
  dates: [],
  created_counts: [],
  completed_counts: [],
  completion_rates: [],
  avg_process_times: []
});

const categoryStatsData = ref<CategoryStats[]>([]);
const userStatsData = ref<UserStats[]>([]);
const templateStatsData = ref<TemplateStats[]>([]);
const statusDistributionData = ref<StatusDistribution[]>([]);
const priorityDistributionData = ref<PriorityDistribution[]>([]);

// 计算属性 - 添加安全检查
const userTimeRanking = computed(() => {
  if (!isMounted.value || !Array.isArray(userStatsData.value) || userStatsData.value.length === 0) {
    return [];
  }
  
  try {
    return [...userStatsData.value]
      .filter(item => item && typeof item.avg_processing_time === 'number')
      .sort((a, b) => a.avg_processing_time - b.avg_processing_time)
      .slice(0, 10);
  } catch (error) {
    console.error('计算用户时间排行失败:', error);
    return [];
  }
});

// 表格列定义
const userCountColumns = [
  { title: '排名', key: 'rank', width: '15%' },
  { title: '用户', key: 'user', dataIndex: 'user_name', width: '30%' },
  { title: '完成数量', key: 'count', width: '55%' }
];

const userTimeColumns = [
  { title: '排名', key: 'rank', width: '15%' },
  { title: '用户', key: 'user', dataIndex: 'user_name', width: '30%' },
  { title: '处理时间', key: 'time', width: '55%' }
];

// 检查组件是否已卸载
const isUnmounted = () => !isMounted.value || !instance;

// 检查数据是否存在的辅助函数
const hasData = (data: any) => {
  if (!isMounted.value) return false;
  if (Array.isArray(data)) {
    return data.length > 0;
  }
  return false;
};

// 构建请求参数
const buildStatsParams = (): StatsReq => {
  const params: StatsReq = {};
  
  if (dateRange.value && dateRange.value.length === 2 && dateRange.value[0] && dateRange.value[1]) {
    params.start_date = dateRange.value[0].startOf('day').format('YYYY-MM-DDTHH:mm:ssZ');
    params.end_date = dateRange.value[1].endOf('day').format('YYYY-MM-DDTHH:mm:ssZ');
  }
  
  if (categoryFilter.value) {
    params.category_id = categoryFilter.value;
  }
  
  if (userFilter.value) {
    params.user_id = userFilter.value;
  }
  
  return params;
};

// 安全地清理图表
const safeDisposeChart = (chart: echarts.ECharts | null) => {
  if (chart && !chart.isDisposed()) {
    try {
      chart.dispose();
    } catch (error) {
      console.warn('图表销毁时出错:', error);
    }
  }
};

// API 调用方法
const fetchOverviewStats = async () => {
  if (isUnmounted()) return;
  
  try {
    const params = buildStatsParams();
    const response = await getWorkorderOverview(params);
    if (!isUnmounted()) {
      overviewStats.value = response;
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取概览统计失败:', error);
      message.error('获取概览统计失败');
    }
  }
};

const fetchTrendData = async () => {
  if (isUnmounted()) return;
  
  chartLoading.trend = true;
  try {
    const params = buildStatsParams();
    params.dimension = trendDimension.value;
    
    const response = await getWorkorderTrend(params);
    if (!isUnmounted()) {
      trendData.value = response;
      
      await nextTick();
      if (!isUnmounted() && hasData(trendData.value.dates)) {
        initTrendChart();
      }
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取趋势数据失败:', error);
      message.error('获取趋势数据失败');
    }
  } finally {
    if (!isUnmounted()) {
      chartLoading.trend = false;
    }
  }
};

const fetchCategoryStats = async () => {
  if (isUnmounted()) return;
  
  chartLoading.category = true;
  try {
    const params = buildStatsParams();
    const response = await getWorkorderCategoryStats(params);
    if (!isUnmounted()) {
      categoryStatsData.value = response;
      
      await nextTick();
      if (!isUnmounted() && hasData(categoryStatsData.value)) {
        initCategoryChart();
      }
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取分类统计失败:', error);
      message.error('获取分类统计失败');
    }
  } finally {
    if (!isUnmounted()) {
      chartLoading.category = false;
    }
  }
};

const fetchUserStats = async () => {
  if (isUnmounted()) return;
  
  chartLoading.user = true;
  try {
    const params = buildStatsParams();
    params.sort_by = 'count';
    params.top = 10;
    
    const response = await getWorkorderUserStats(params);
    if (!isUnmounted()) {
      // 确保返回的是数组
      userStatsData.value = Array.isArray(response) ? response : [];
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取用户统计失败:', error);
      message.error('获取用户统计失败');
      userStatsData.value = [];
    }
  } finally {
    if (!isUnmounted()) {
      chartLoading.user = false;
    }
  }
};

const fetchTemplateStats = async () => {
  if (isUnmounted()) return;
  
  chartLoading.template = true;
  try {
    const params = buildStatsParams();
    const response = await getWorkorderTemplateStats(params);
    if (!isUnmounted()) {
      templateStatsData.value = response;
      
      await nextTick();
      if (!isUnmounted() && hasData(templateStatsData.value)) {
        initTemplateChart();
      }
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取模板统计失败:', error);
      message.error('获取模板统计失败');
    }
  } finally {
    if (!isUnmounted()) {
      chartLoading.template = false;
    }
  }
};

const fetchStatusDistribution = async () => {
  if (isUnmounted()) return;
  
  chartLoading.status = true;
  try {
    const params = buildStatsParams();
    const response = await getWorkorderStatusDistribution(params);
    if (!isUnmounted()) {
      statusDistributionData.value = response;
      
      await nextTick();
      if (!isUnmounted() && hasData(statusDistributionData.value)) {
        initStatusChart();
      }
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取状态分布失败:', error);
      message.error('获取状态分布失败');
    }
  } finally {
    if (!isUnmounted()) {
      chartLoading.status = false;
    }
  }
};

const fetchPriorityDistribution = async () => {
  if (isUnmounted()) return;
  
  chartLoading.priority = true;
  try {
    const params = buildStatsParams();
    const response = await getWorkorderPriorityDistribution(params);
    if (!isUnmounted()) {
      priorityDistributionData.value = response;
      
      await nextTick();
      if (!isUnmounted() && hasData(priorityDistributionData.value)) {
        initPriorityChart();
      }
    }
  } catch (error) {
    if (!isUnmounted()) {
      console.error('获取优先级分布失败:', error);
      message.error('获取优先级分布失败');
    }
  } finally {
    if (!isUnmounted()) {
      chartLoading.priority = false;
    }
  }
};

// 事件处理
const handleDateRangeChange = () => {
  if (!isUnmounted()) {
    refreshData();
  }
};

const handleFilterChange = () => {
  if (!isUnmounted()) {
    refreshData();
  }
};

const refreshData = async () => {
  if (isUnmounted()) return;
  
  loading.value = true;
  
  try {
    await Promise.all([
      fetchOverviewStats(),
      fetchTrendData(),
      fetchCategoryStats(),
      fetchUserStats(),
      fetchTemplateStats(),
      fetchStatusDistribution(),
      fetchPriorityDistribution()
    ]);
  } catch (error) {
    if (!isUnmounted()) {
      console.error('刷新数据失败:', error);
      message.error('刷新数据失败');
    }
  } finally {
    if (!isUnmounted()) {
      loading.value = false;
    }
  }
};

// 图表初始化方法（保持原有的实现，但添加更多安全检查）
const initTrendChart = () => {
  if (isUnmounted() || !trendChartRef.value || !hasData(trendData.value.dates)) return;

  safeDisposeChart(trendChart);
  trendChart = null;

  try {
    trendChart = echarts.init(trendChartRef.value);
    const option = {
      tooltip: {
        trigger: 'axis'
      },
      legend: {
        data: ['创建工单', '完成工单', '完成率'],
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
        data: trendData.value.dates
      },
      yAxis: [
        {
          type: 'value',
          name: '工单数量'
        },
        {
          type: 'value',
          name: '完成率(%)',
          min: 0,
          max: 100
        }
      ],
      series: [
        {
          name: '创建工单',
          type: 'line',
          yAxisIndex: 0,
          data: trendData.value.created_counts,
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
          yAxisIndex: 0,
          data: trendData.value.completed_counts,
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
        },
        {
          name: '完成率',
          type: 'line',
          yAxisIndex: 1,
          data: trendData.value.completion_rates.map((rate: number) => (rate * 100).toFixed(1)),
          smooth: true,
          lineStyle: {
            width: 2,
            color: '#722ed1'
          }
        }
      ]
    };

    if (!isUnmounted() && trendChart) {
      trendChart.setOption(option);
    }
  } catch (error) {
    console.error('初始化趋势图表失败:', error);
  }
};

const initStatusChart = () => {
  if (isUnmounted() || !statusChartRef.value || !hasData(statusDistributionData.value)) return;

  safeDisposeChart(statusChart);
  statusChart = null;

  try {
    statusChart = echarts.init(statusChartRef.value);
    const option = {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)'
      },
      legend: {
        orient: 'vertical',
        right: 10,
        top: 'center'
      },
      series: [
        {
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
          data: statusDistributionData.value.map((item: { status: string; count: number }) => ({
            name: item.status,
            value: item.count
          })),
          color: ['#52c41a', '#faad14', '#bfbfbf', '#f5222d']
        }
      ]
    };

    if (!isUnmounted() && statusChart) {
      statusChart.setOption(option);
    }
  } catch (error) {
    console.error('初始化状态图表失败:', error);
  }
};

const initCategoryChart = () => {
  if (isUnmounted() || !categoryChartRef.value || !hasData(categoryStatsData.value)) return;

  safeDisposeChart(categoryChart);
  categoryChart = null;

  try {
    categoryChart = echarts.init(categoryChartRef.value);
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
        containLabel: true
      },
      xAxis: {
        type: 'value'
      },
      yAxis: {
        type: 'category',
        data: categoryStatsData.value.map((item: { category_name: string }) => item.category_name)
      },
      series: [
        {
          name: '工单数量',
          type: 'bar',
          data: categoryStatsData.value.map((item: { count: number }) => ({
            value: item.count,
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#1890ff' },
                { offset: 1, color: '#69c0ff' }
              ])
            }
          }))
        }
      ]
    };

    if (!isUnmounted() && categoryChart) {
      categoryChart.setOption(option);
    }
  } catch (error) {
    console.error('初始化分类图表失败:', error);
  }
};

const initPriorityChart = () => {
  if (isUnmounted() || !priorityChartRef.value || !hasData(priorityDistributionData.value)) return;

  safeDisposeChart(priorityChart);
  priorityChart = null;

  try {
    priorityChart = echarts.init(priorityChartRef.value);
    const option = {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)'
      },
      legend: {
        bottom: 0
      },
      series: [
        {
          type: 'pie',
          radius: '65%',
          center: ['50%', '45%'],
          data: priorityDistributionData.value.map((item: { priority: string; count: number }) => ({
            name: item.priority,
            value: item.count
          })),
          color: ['#f5222d', '#faad14', '#52c41a', '#1890ff']
        }
      ]
    };

    if (!isUnmounted() && priorityChart) {
      priorityChart.setOption(option);
    }
  } catch (error) {
    console.error('初始化优先级图表失败:', error);
  }
};

const initTemplateChart = () => {
  if (isUnmounted() || !templateChartRef.value || !hasData(templateStatsData.value)) return;

  safeDisposeChart(templateChart);
  templateChart = null;

  try {
    templateChart = echarts.init(templateChartRef.value);
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
        bottom: '15%',
        containLabel: true
      },
      xAxis: {
        type: 'category',
        data: templateStatsData.value.map((item: { template_name: string }) => item.template_name),
        axisLabel: {
          interval: 0,
          rotate: 30
        }
      },
      yAxis: {
        type: 'value'
      },
      series: [
        {
          name: '使用次数',
          type: 'bar',
          data: templateStatsData.value.map((item: { count: number }) => ({
            value: item.count,
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: '#1890ff' },
                { offset: 1, color: '#69c0ff' }
              ])
            }
          }))
        }
      ]
    };

    if (!isUnmounted() && templateChart) {
      templateChart.setOption(option);
    }
  } catch (error) {
    console.error('初始化模板图表失败:', error);
  }
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
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];

  let hash = 0;
  for (let i = 0; i < (name || '').length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length];
};

const getRankColor = (index: number, isTime: boolean = false) => {
  if (isTime) {
    const colors = ['#52c41a', '#85ce61', '#b3e19d', '#e6a23c', '#f56c6c'];
    return colors[Math.min(index, colors.length - 1)];
  } else {
    const colors = ['#f56c6c', '#e6a23c', '#85ce61', '#52c41a', '#409eff'];
    return colors[Math.min(index, colors.length - 1)];
  }
};

const getProgressColor = (index: number, isTime: boolean = false) => {
  if (isTime) {
    const colors = ['#52c41a', '#7ec050', '#b9de7c', '#faad14', '#f56c6c'];
    return colors[Math.min(index, colors.length - 1)];
  } else {
    const colors = ['#1890ff', '#40a9ff', '#69c0ff', '#91d5ff', '#bae7ff'];
    return colors[Math.min(index, colors.length - 1)];
  }
};

const getPercentage = (value: number, max: number) => {
  if (!max) return 0;
  return Math.round((value / max) * 100);
};

const getTimePercentage = (value: number, data: UserStats[]) => {
  if (!data.length) return 0;
  const max = Math.max(...data.map(item => item.avg_processing_time));
  return Math.round(((max - value) / max) * 100);
};

// 响应窗口大小变化
const handleResize = () => {
  if (isUnmounted()) return;
  
  try {
    trendChart?.resize();
    statusChart?.resize();
    categoryChart?.resize();
    priorityChart?.resize();
    templateChart?.resize();
  } catch (error) {
    console.error('图表大小调整失败:', error);
  }
};

// 生命周期钩子
onMounted(() => {
  isMounted.value = true;
  refreshData();
  window.addEventListener('resize', handleResize);
});

onBeforeUnmount(() => {
  isMounted.value = false;
  
  // 移除事件监听器
  window.removeEventListener('resize', handleResize);
  
  // 安全地销毁所有图表实例
  safeDisposeChart(trendChart);
  safeDisposeChart(statusChart);
  safeDisposeChart(categoryChart);
  safeDisposeChart(priorityChart);
  safeDisposeChart(templateChart);
  
  // 清空图表引用
  trendChart = null;
  statusChart = null;
  categoryChart = null;
  priorityChart = null;
  templateChart = null;
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

.completion-rate {
  position: absolute;
  bottom: 16px;
  right: 24px;
  font-size: 12px;
  color: #8c8c8c;
}

.chart-row {
  margin-bottom: 24px;
}

.chart-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  height: 100%;
}

.trend-controls {
  margin-bottom: 16px;
  text-align: center;
}

.trend-chart-container,
.status-chart-container,
.category-chart-container,
.priority-chart-container,
.template-chart-container {
  width: 100%;
  height: 350px;
}

.empty-chart {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 350px;
  background-color: #fafafa;
  border-radius: 6px;
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