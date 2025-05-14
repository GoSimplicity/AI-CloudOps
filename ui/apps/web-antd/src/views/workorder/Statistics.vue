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
        <a-button @click="refreshData" :loading="loading">
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
            <a-statistic title="总工单数" :value="overviewStats.total_count" :value-style="{ color: '#40a9ff' }">
              <template #prefix>
                <FileDoneOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.totalChange >= 0 ? 'increase' : 'decrease'">
                <ArrowUpOutlined v-if="overviewStats.totalChange >= 0" />
                <ArrowDownOutlined v-else />
                {{ Math.abs(overviewStats.totalChange || 0) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已完成" :value="overviewStats.completed_count" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.completedChange >= 0 ? 'increase' : 'decrease'">
                <ArrowUpOutlined v-if="overviewStats.completedChange >= 0" />
                <ArrowDownOutlined v-else />
                {{ Math.abs(overviewStats.completedChange || 0) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="处理中" :value="overviewStats.processing_count" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <SyncOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.processingChange >= 0 ? 'increase' : 'decrease'">
                <ArrowUpOutlined v-if="overviewStats.processingChange >= 0" />
                <ArrowDownOutlined v-else />
                {{ Math.abs(overviewStats.processingChange || 0) }}%
              </span>
              比上期
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="平均处理时间" :value="overviewStats.avg_process_time" suffix="小时"
              :value-style="{ color: '#722ed1' }">
              <template #prefix>
                <ClockCircleOutlined />
              </template>
            </a-statistic>
            <div class="stat-change">
              <span :class="overviewStats.avgProcessTimeChange <= 0 ? 'increase' : 'decrease'">
                <ArrowDownOutlined v-if="overviewStats.avgProcessTimeChange <= 0" />
                <ArrowUpOutlined v-else />
                {{ Math.abs(overviewStats.avgProcessTimeChange || 0) }}%
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
            <a-spin :spinning="chartLoading.trend">
              <div class="trend-chart-container" ref="trendChartRef"></div>
            </a-spin>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card title="工单状态分布" class="chart-card">
            <a-spin :spinning="chartLoading.status">
              <div class="status-chart-container" ref="statusChartRef"></div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>

      <!-- 流程使用统计和部门分布 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="12">
          <a-card title="流程使用统计" class="chart-card">
            <a-spin :spinning="chartLoading.process">
              <div class="process-chart-container" ref="processChartRef"></div>
            </a-spin>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="部门工单分布" class="chart-card">
            <a-spin :spinning="chartLoading.department">
              <div class="department-chart-container" ref="departmentChartRef"></div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>

      <!-- 审批效率分析 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="24">
          <a-card title="审批效率分析" class="chart-card">
            <a-spin :spinning="chartLoading.efficiency">
              <div class="efficiency-chart-container" ref="efficiencyChartRef"></div>
            </a-spin>
          </a-card>
        </a-col>
      </a-row>

      <!-- 处理人排行榜 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :span="12">
          <a-card title="处理人排行榜 - 处理工单数" class="chart-card">
            <a-spin :spinning="chartLoading.userRanking">
              <a-table :dataSource="handlerRankingByCount" :pagination="false" :columns="rankColumns" size="small">
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
                    <a-progress :percent="getPercentage(record.completed_count, handlerRankingByCount[0]?.completed_count || 0)"
                      :show-info="false" status="active" :stroke-color="getProgressColor(index)" />
                    <span class="count-value">{{ record.completed_count }}</span>
                  </template>
                </template>
              </a-table>
            </a-spin>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="处理人排行榜 - 平均处理时间" class="chart-card">
            <a-spin :spinning="chartLoading.userRanking">
              <a-table :dataSource="handlerRankingByTime" :pagination="false" :columns="timeRankColumns" size="small">
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
                      :percent="getPercentage(handlerRankingByTime[handlerRankingByTime.length - 1]?.avg_processing_time || 0, record.avg_processing_time)"
                      :show-info="false" status="active" :stroke-color="getProgressColor(index, true)" />
                    <span class="time-value">{{ record.avg_processing_time }}小时</span>
                  </template>
                </template>
              </a-table>
            </a-spin>
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
// import { 
//   getStatisticsOverview, 
//   getStatisticsTrend, 
//   getStatisticsCategory, 
//   getStatisticsPerformance,
//   getStatisticsUser, 
//   listProcess 
// } from '#/api/core/workorder';
// import type { Process, WorkOrderStatistics, UserPerformance } from '#/api/core/workorder';
import type { Process, UserPerformance } from '#/api/core/workorder';

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
const chartLoading = reactive({
  trend: false,
  status: false,
  process: false,
  department: false,
  efficiency: false,
  userRanking: false
});

const processFilter = ref(null);
const dateRange = ref([dayjs().subtract(30, 'days'), dayjs()]);
const dateRanges = {
  '最近7天': [dayjs().subtract(7, 'days'), dayjs()],
  '最近30天': [dayjs().subtract(30, 'days'), dayjs()],
  '最近90天': [dayjs().subtract(90, 'days'), dayjs()],
  '今年': [dayjs().startOf('year'), dayjs()]
};

// 流程数据
const processes = ref<Process[]>([
  { id: 1, name: '员工入职流程', status: 1, description: '', form_design_id: 1, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 2, name: '请假申请流程', status: 1, description: '', form_design_id: 2, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 3, name: '报销申请流程', status: 1, description: '', form_design_id: 3, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 4, name: '采购申请流程', status: 1, description: '', form_design_id: 4, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 5, name: '合同审批流程', status: 1, description: '', form_design_id: 5, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 6, name: '招聘需求流程', status: 1, description: '', form_design_id: 6, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 7, name: '员工离职流程', status: 1, description: '', form_design_id: 7, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' },
  { id: 8, name: 'IT资源申请流程', status: 1, description: '', form_design_id: 8, definition: "", version: 1, created_at: '', updated_at: '', creator_id: 1, creator_name: '系统管理员' }
]);

// 概览统计数据
const overviewStats = reactive({
  total_count: 1856,
  totalChange: 8.3,
  completed_count: 1423,
  completedChange: 10.5,
  processing_count: 324,
  processingChange: -5.8,
  canceled_count: 67,
  rejected_count: 42,
  avg_process_time: 18.6,
  avgProcessTimeChange: -3.9
});

// 统计趋势数据
const trendData = reactive({
  dates: [
    '2025-04-13', '2025-04-14', '2025-04-15', '2025-04-16', '2025-04-17', '2025-04-18', '2025-04-19',
    '2025-04-20', '2025-04-21', '2025-04-22', '2025-04-23', '2025-04-24', '2025-04-25', '2025-04-26',
    '2025-04-27', '2025-04-28', '2025-04-29', '2025-04-30', '2025-05-01', '2025-05-02', '2025-05-03',
    '2025-05-04', '2025-05-05', '2025-05-06', '2025-05-07', '2025-05-08', '2025-05-09', '2025-05-10',
    '2025-05-11', '2025-05-12', '2025-05-13'
  ],
  created: [
    62, 58, 64, 72, 68, 45, 38,
    42, 69, 73, 68, 65, 57, 35,
    39, 75, 78, 69, 32, 29, 36,
    40, 80, 76, 68, 64, 69, 42,
    37, 64, 58
  ],
  completed: [
    57, 54, 60, 65, 62, 40, 35,
    38, 63, 68, 65, 61, 52, 32,
    35, 68, 72, 65, 28, 26, 30,
    36, 73, 70, 63, 58, 64, 38,
    32, 59, 54
  ]
});

// 排行榜数据
interface ExtendedUserPerformance extends UserPerformance {
  department: string;
}

const handlerRankingByCount = ref<UserPerformance[]>([
  { id: 1, user_id: 101, user_name: '张明辉', department: '人力资源部', date: '2023-05-01', assigned_count: 200, completed_count: 187, avg_response_time: 2.5, avg_processing_time: 16.8, satisfaction_score: 4.8, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 2, user_id: 102, user_name: '李婷', department: '行政部', date: '2023-05-01', assigned_count: 180, completed_count: 165, avg_response_time: 3.1, avg_processing_time: 17.5, satisfaction_score: 4.6, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 3, user_id: 103, user_name: '王浩', department: '财务部', date: '2023-05-01', assigned_count: 160, completed_count: 149, avg_response_time: 2.8, avg_processing_time: 19.3, satisfaction_score: 4.5, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 4, user_id: 104, user_name: '陈静', department: '人力资源部', date: '2023-05-01', assigned_count: 140, completed_count: 128, avg_response_time: 2.2, avg_processing_time: 15.2, satisfaction_score: 4.9, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 5, user_id: 105, user_name: '赵鑫', department: 'IT部', date: '2023-05-01', assigned_count: 120, completed_count: 112, avg_response_time: 3.5, avg_processing_time: 20.1, satisfaction_score: 4.3, created_at: '2023-05-01', updated_at: '2023-05-01' }
]);

const handlerRankingByTime = ref<UserPerformance[]>([
  { id: 4, user_id: 104, user_name: '陈静', department: '人力资源部', date: '2023-05-01', assigned_count: 140, completed_count: 128, avg_response_time: 2.2, avg_processing_time: 15.2, satisfaction_score: 4.9, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 6, user_id: 106, user_name: '刘伟', department: '市场部', date: '2023-05-01', assigned_count: 110, completed_count: 98, avg_response_time: 2.4, avg_processing_time: 16.1, satisfaction_score: 4.7, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 1, user_id: 101, user_name: '张明辉', department: '人力资源部', date: '2023-05-01', assigned_count: 200, completed_count: 187, avg_response_time: 2.5, avg_processing_time: 16.8, satisfaction_score: 4.8, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 2, user_id: 102, user_name: '李婷', department: '行政部', date: '2023-05-01', assigned_count: 180, completed_count: 165, avg_response_time: 3.1, avg_processing_time: 17.5, satisfaction_score: 4.6, created_at: '2023-05-01', updated_at: '2023-05-01' },
  { id: 7, user_id: 107, user_name: '徐文', department: '法务部', date: '2023-05-01', assigned_count: 95, completed_count: 86, avg_response_time: 2.9, avg_processing_time: 18.9, satisfaction_score: 4.4, created_at: '2023-05-01', updated_at: '2023-05-01' }
]);

// 状态分布数据
const statusDistribution = ref<{ name: string, value: number }[]>([
  { name: '已完成', value: 1423 },
  { name: '处理中', value: 324 },
  { name: '已取消', value: 67 },
  { name: '已拒绝', value: 42 }
]);

// 流程使用统计数据
const processUsageData = ref<{ name: string, value: number }[]>([
  { name: '请假申请流程', value: 432 },
  { name: '报销申请流程', value: 368 },
  { name: 'IT资源申请流程', value: 286 },
  { name: '员工入职流程', value: 217 },
  { name: '采购申请流程', value: 198 },
  { name: '合同审批流程', value: 156 },
  { name: '员工离职流程', value: 124 },
  { name: '招聘需求流程', value: 75 }
]);

// 部门分布数据
const departmentData = ref<{ name: string, value: number }[]>([
  { name: '销售部', value: 346 },
  { name: '研发部', value: 312 },
  { name: '市场部', value: 287 },
  { name: '人力资源部', value: 268 },
  { name: '财务部', value: 231 },
  { name: '行政部', value: 186 },
  { name: 'IT部', value: 157 },
  { name: '法务部', value: 69 }
]);

// 审批效率数据
const efficiencyData = reactive({
  processes: [
    '请假申请流程', 
    'IT资源申请流程', 
    '报销申请流程', 
    '员工入职流程', 
    '合同审批流程', 
    '采购申请流程', 
    '员工离职流程', 
    '招聘需求流程'
  ],
  avgTimes: [12.4, 15.8, 17.3, 21.6, 24.2, 25.7, 28.9, 32.6]
});

// 表格列定义
const rankColumns = [
  { title: '排名', key: 'rank', width: '15%' },
  { title: '处理人', key: 'user', dataIndex: 'user_name', width: '25%' },
  { title: '部门', dataIndex: 'department', width: '25%' },
  { title: '处理工单数', key: 'count', width: '35%' }
];

const timeRankColumns = [
  { title: '排名', key: 'rank', width: '15%' },
  { title: '处理人', key: 'user', dataIndex: 'user_name', width: '25%' },
  { title: '部门', dataIndex: 'department', width: '25%' },
  { title: '平均处理时间', key: 'time', width: '35%' }
];

// API请求相关方法
const fetchProcesses = async () => {
  try {
    // Mock implementation - in real app this would call the API
    // processes.value already set with mock data above
  } catch (error) {
    console.error('获取流程列表失败:', error);
    message.error('获取流程列表失败');
  }
};

const fetchOverviewStats = async () => {
  try {
    // Mock implementation - in real app this would call the API
    // overviewStats already set with mock data above
  } catch (error) {
    console.error('获取概览统计失败:', error);
    message.error('获取概览统计失败');
  }
};

const fetchTrendData = async () => {
  chartLoading.trend = true;
  try {
    // Mock implementation - in real app this would call the API
    // trendData already set with mock data above
    
    // 更新趋势图
    nextTick(() => {
      initTrendChart();
    });
  } catch (error) {
    console.error('获取趋势数据失败:', error);
    message.error('获取趋势数据失败');
  } finally {
    chartLoading.trend = false;
  }
};

const fetchCategoryStats = async () => {
  chartLoading.status = true;
  chartLoading.process = true;
  try {
    // Mock implementation - in real app this would call the API
    // statusDistribution and processUsageData already set with mock data above
    
    // 更新图表
    nextTick(() => {
      initStatusChart();
      initProcessChart();
    });
  } catch (error) {
    console.error('获取分类统计失败:', error);
    message.error('获取分类统计失败');
  } finally {
    chartLoading.status = false;
    chartLoading.process = false;
  }
};

const fetchDepartmentData = async () => {
  chartLoading.department = true;
  try {
    // Mock implementation - in real app this would call the API
    // departmentData already set with mock data above
    
    // 更新图表
    nextTick(() => {
      initDepartmentChart();
    });
  } catch (error) {
    console.error('获取部门分布数据失败:', error);
    message.error('获取部门分布数据失败');
  } finally {
    chartLoading.department = false;
  }
};

const fetchEfficiencyData = async () => {
  chartLoading.efficiency = true;
  try {
    // Mock implementation - in real app this would call the API
    // efficiencyData already set with mock data above
    
    // 更新图表
    nextTick(() => {
      initEfficiencyChart();
    });
  } catch (error) {
    console.error('获取审批效率数据失败:', error);
    message.error('获取审批效率数据失败');
  } finally {
    chartLoading.efficiency = false;
  }
};

const fetchUserPerformance = async () => {
  chartLoading.userRanking = true;
  try {
    // Mock implementation - in real app this would call the API
    // handlerRankingByCount and handlerRankingByTime already set with mock data above
  } catch (error) {
    console.error('获取用户绩效数据失败:', error);
    message.error('获取用户绩效数据失败');
  } finally {
    chartLoading.userRanking = false;
  }
};

// 方法
const handleDateRangeChange = (dates: any) => {
  if (dates && dates.length === 2) {
    // 根据日期范围变更获取数据
    refreshData();
  }
};

const handleProcessChange = () => {
  refreshData();
};

const refreshData = async () => {
  loading.value = true;
  
  try {
    // 并行请求所有数据
    await Promise.all([
      fetchProcesses(),
      fetchOverviewStats(),
      fetchTrendData(),
      fetchCategoryStats(),
      fetchDepartmentData(),
      fetchEfficiencyData(),
      fetchUserPerformance()
    ]);
  } catch (error) {
    console.error('刷新数据失败:', error);
    message.error('刷新数据失败');
  } finally {
    loading.value = false;
  }
};

// 图表初始化方法
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
      data: statusDistribution.value.map(item => item.name)
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
        data: statusDistribution.value,
        color: ['#52c41a', '#faad14', '#bfbfbf', '#f5222d']
      }
    ]
  };

  statusChart.setOption(option);
};

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
      data: processUsageData.value.map(item => item.name),
      inverse: true
    },
    series: [
      {
        name: '工单数',
        type: 'bar',
        data: processUsageData.value.map(item => ({
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
      data: departmentData.value.map(item => item.name)
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
        data: departmentData.value,
        color: ['#1890ff', '#52c41a', '#faad14', '#722ed1', '#13c2c2', '#eb2f96']
      }
    ]
  };

  departmentChart.setOption(option);
};

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
  if (!max) return 0;
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