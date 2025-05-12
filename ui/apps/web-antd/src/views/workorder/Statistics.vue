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
import { ref, reactive, onMounted, nextTick, onBeforeUnmount, watch, computed } from 'vue';
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
import { 
  getStatisticsOverview, 
  getStatisticsTrend, 
  getStatisticsCategory, 
  getStatisticsPerformance,
  getStatisticsUser, 
  listProcess 
} from '#/api/core/workorder';
import type { Process, WorkOrderStatistics, UserPerformance } from '#/api/core/workorder';

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
const processes = ref<Process[]>([]);

// 概览统计数据
const overviewStats = reactive({
  total_count: 0,
  totalChange: 0,
  completed_count: 0,
  completedChange: 0,
  processing_count: 0,
  processingChange: 0,
  canceled_count: 0,
  rejected_count: 0,
  avg_process_time: 0,
  avgProcessTimeChange: 0
});

// 统计趋势数据
const trendData = reactive({
  dates: [] as string[],
  created: [] as number[],
  completed: [] as number[]
});

// 排行榜数据
const handlerRankingByCount = ref<UserPerformance[]>([]);
const handlerRankingByTime = ref<UserPerformance[]>([]);

// 状态分布数据
const statusDistribution = ref<{ name: string, value: number }[]>([]);

// 流程使用统计数据
const processUsageData = ref<{ name: string, value: number }[]>([]);

// 部门分布数据
const departmentData = ref<{ name: string, value: number }[]>([]);

// 审批效率数据
const efficiencyData = reactive({
  processes: [] as string[],
  avgTimes: [] as number[]
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
    const res = await listProcess({
      page: 1,
      size: 100,
      status: 1 // 只获取已发布的流程
    });
    processes.value = res.list || [];
  } catch (error) {
    console.error('获取流程列表失败:', error);
    message.error('获取流程列表失败');
  }
};

const fetchOverviewStats = async () => {
  try {
    const res = await getStatisticsOverview();
    const data = res;
    
    if (data) {
      // 更新概览数据
      overviewStats.total_count = data.total_count;
      overviewStats.completed_count = data.completed_count;
      overviewStats.processing_count = data.processing_count;
      overviewStats.canceled_count = data.canceled_count;
      overviewStats.rejected_count = data.rejected_count;
      overviewStats.avg_process_time = data.avg_process_time;
      
      // 计算同比变化（这里假设后端没有直接返回变化率）
      // 实际应用中可能需要获取前一个时间段的数据来计算
      overviewStats.totalChange = 8.5; // 示例值，实际应从API获取或计算
      overviewStats.completedChange = 10.2;
      overviewStats.processingChange = -5.3;
      overviewStats.avgProcessTimeChange = -3.7;
    }
  } catch (error) {
    console.error('获取概览统计失败:', error);
    message.error('获取概览统计失败');
  }
};

const fetchTrendData = async () => {
  chartLoading.trend = true;
  try {
    const res = await getStatisticsTrend();
    const data = res || [];
    
    // 提取日期和创建/完成数量
    trendData.dates = data.map((item: any) => item.date);
    trendData.created = data.map((item: any) => item.total_count);
    trendData.completed = data.map((item: any) => item.completed_count);
    
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
    const res = await getStatisticsCategory();
    const data = res;
    
    if (data) {
      // 解析工单状态分布
      const categoryStats = typeof data.category_stats === 'string' 
        ? JSON.parse(data.category_stats) 
        : data.category_stats;
      
      statusDistribution.value = [
        { name: '已完成', value: data.completed_count },
        { name: '处理中', value: data.processing_count },
        { name: '已取消', value: data.canceled_count },
        { name: '已拒绝', value: data.rejected_count }
      ];
      
      // 解析流程使用统计 (假设category_stats包含流程使用信息)
      if (categoryStats && Array.isArray(categoryStats)) {
        processUsageData.value = categoryStats.map((item: any) => ({
          name: item.name || '未知流程',
          value: item.count || 0
        }));
      }
      
      // 更新图表
      nextTick(() => {
        initStatusChart();
        initProcessChart();
      });
    }
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
    const res = await getStatisticsUser();
    const data = res;
    
    if (data) {
      // 解析用户/部门统计数据
      const userStats = typeof data.user_stats === 'string' 
        ? JSON.parse(data.user_stats) 
        : data.user_stats;
      
      if (userStats && Array.isArray(userStats)) {
        // 按部门聚合数据
        const deptMap = new Map<string, number>();
        
        userStats.forEach((item: any) => {
          const dept = item.department || '未知部门';
          const count = item.count || 0;
          
          if (deptMap.has(dept)) {
            deptMap.set(dept, deptMap.get(dept)! + count);
          } else {
            deptMap.set(dept, count);
          }
        });
        
        // 转换为图表所需格式
        departmentData.value = Array.from(deptMap.entries()).map(([name, value]) => ({
          name,
          value
        }));
      }
      
      // 更新图表
      nextTick(() => {
        initDepartmentChart();
      });
    }
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
    // 假设我们需要组合流程数据和处理时间
    const res = await getStatisticsPerformance();
    const data = res || [];
    
    if (data.length > 0) {
      // 按流程分组，计算平均处理时间
      const processMap = new Map<string, { count: number, totalTime: number }>();
      
      processes.value.forEach((process: Process) => {
        // 初始化所有流程条目
        processMap.set(process.name, { count: 0, totalTime: 0 });
      });
      
      // 累计每个流程的时间
      data.forEach((item: any) => {
        if (item.process_name) {
          if (processMap.has(item.process_name)) {
            const current = processMap.get(item.process_name)!;
            current.count += 1;
            current.totalTime += item.avg_processing_time || 0;
          }
        }
      });
      
      // 计算平均值并转换为图表所需格式
      const processEfficiency = Array.from(processMap.entries())
        .filter(([_, stats]) => stats.count > 0)
        .map(([name, stats]) => ({
          name,
          avgTime: stats.totalTime / stats.count
        }))
        .sort((a, b) => b.avgTime - a.avgTime) // 从高到低排序
        .slice(0, 10); // 取前10个
      
      efficiencyData.processes = processEfficiency.map(item => item.name);
      efficiencyData.avgTimes = processEfficiency.map(item => item.avgTime);
      
      // 更新图表
      nextTick(() => {
        initEfficiencyChart();
      });
    }
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
    const res = await getStatisticsPerformance();
    const data = res || [];
    
    if (data.length > 0) {
      // 按照完成数量排序
      const sortedByCount = [...data]
        .sort((a, b) => b.completed_count - a.completed_count)
        .slice(0, 5); // 取前5名
      
      // 按照平均处理时间排序（从小到大）
      const sortedByTime = [...data]
        .sort((a, b) => a.avg_processing_time - b.avg_processing_time)
        .slice(0, 5); // 取前5名
      
      handlerRankingByCount.value = sortedByCount;
      handlerRankingByTime.value = sortedByTime;
    }
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
    
    message.success('数据已刷新');
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