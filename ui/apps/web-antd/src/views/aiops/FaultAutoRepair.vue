<template>
  <div class="fault-repair-container">
    <div class="header">
      <h1 class="title">智能运维故障自动修复系统</h1>
      <div class="actions">
        <!-- 手动故障修复触发 -->
        <a-button type="primary" @click="showManualRepairModal" class="manual-repair-btn">
          <template #icon><tool-outlined /></template>
          手动故障修复
        </a-button>
        <a-select v-model:value="timeRange" style="width: 150px" class="time-selector" @change="refreshData">
          <a-select-option value="1h">最近1小时</a-select-option>
          <a-select-option value="6h">最近6小时</a-select-option>
          <a-select-option value="24h">最近24小时</a-select-option>
          <a-select-option value="7d">最近7天</a-select-option>
        </a-select>
        <a-button type="primary" class="refresh-btn" @click="refreshData" :loading="loading">
          <template #icon><sync-outlined /></template>
          刷新
        </a-button>
      </div>
    </div>

    <!-- API健康状态 -->
    <a-alert 
      :message="apiHealthStatus.message" 
      :type="apiHealthStatus.type" 
      :show-icon="true" 
      :closable="false"
      style="margin-bottom: 20px"
    />

    <div class="dashboard">
      <!-- 统计卡片 -->
      <div class="stats-cards">
        <a-card class="stat-card repair-card">
          <template #title>
            <tool-outlined /> 自动修复总数
          </template>
          <div class="stat-value">{{ repairStats.total }}</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ repairStats.totalIncrease }}%
          </div>
        </a-card>
        <a-card class="stat-card repair-card">
          <template #title>
            <check-circle-outlined /> 修复成功率
          </template>
          <div class="stat-value">{{ repairStats.successRate }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ repairStats.successRateIncrease }}%
          </div>
        </a-card>
        <a-card class="stat-card repair-card">
          <template #title>
            <clock-circle-outlined /> 平均修复时间
          </template>
          <div class="stat-value">{{ repairStats.avgTime }}</div>
          <div class="stat-trend down">
            <arrow-down-outlined /> {{ repairStats.avgTimeDecrease }}%
          </div>
        </a-card>
        <a-card class="stat-card repair-card">
          <template #title>
            <thunderbolt-outlined /> 自动化程度
          </template>
          <div class="stat-value">{{ repairStats.automationRate }}%</div>
          <div class="stat-trend up">
            <arrow-up-outlined /> {{ repairStats.automationIncrease }}%
          </div>
        </a-card>
      </div>

      <!-- Agent处理流程图 -->
      <a-card class="workflow-card" title="Agent智能分流处理流程">
        <div class="workflow-container">
          <div class="workflow-node supervisor">
            <robot-outlined />
            <div class="node-title">Supervisor</div>
            <div class="node-desc">故障接收与初步分析</div>
          </div>
          <div class="workflow-arrow">
            <arrow-right-outlined />
          </div>
          <div class="workflow-node agent">
            <api-outlined />
            <div class="node-title">Agent</div>
            <div class="node-desc">故障分流与自动修复</div>
          </div>
          <div class="workflow-decision">
            <question-circle-outlined />
            <div class="decision-title">问题解决?</div>
          </div>
          <div class="workflow-branches">
            <div class="branch yes">
              <check-outlined />
              <span>是</span>
              <arrow-down-outlined />
              <div class="branch-node success">
                <check-circle-outlined />
                <div>修复完成</div>
              </div>
            </div>
            <div class="branch no">
              <close-outlined />
              <span>否</span>
              <arrow-right-outlined />
              <div class="loop-arrow">
                <redo-outlined />
                <div>重新分流</div>
              </div>
              <div class="branch-node human">
                <user-outlined />
                <div>人工介入</div>
                <div class="human-notify">飞书通知</div>
              </div>
            </div>
          </div>
        </div>
      </a-card>

      <!-- 实时修复状态 -->
      <a-card class="real-time-status" title="实时修复状态" v-if="ongoingRepairs.length > 0">
        <div class="ongoing-repairs">
          <div v-for="repair in ongoingRepairs" :key="repair.id" class="ongoing-item">
            <a-badge status="processing" />
            <span class="repair-name">{{ repair.deployment }}</span>
            <span class="repair-namespace">{{ repair.namespace }}</span>
            <a-spin size="small" />
            <span class="repair-status">{{ repair.statusText }}</span>
          </div>
        </div>
      </a-card>

      <!-- 图表区域 -->
      <div class="charts-container">
        <a-row :gutter="16">
          <a-col :span="24" :lg="12">
            <a-card class="chart-card" title="故障修复趋势">
              <div ref="trendChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="24" :lg="12">
            <a-card class="chart-card" title="故障类型分布">
              <div ref="typeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
        <a-row :gutter="16" style="margin-top: 16px;">
          <a-col :span="24" :lg="12">
            <a-card class="chart-card" title="修复方法分布">
              <div ref="methodChartRef" class="chart"></div>
            </a-card>
          </a-col>
          <a-col :span="24" :lg="12">
            <a-card class="chart-card" title="修复时间分布">
              <div ref="timeChartRef" class="chart"></div>
            </a-card>
          </a-col>
        </a-row>
      </div>

      <!-- 最近修复记录 -->
      <a-card class="recent-repairs" title="最近修复记录">
        <a-table :dataSource="repairList" :columns="columns" :loading="loading" :pagination="{ pageSize: 5 }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
            </template>
            <template v-if="column.key === 'flowStatus'">
              <a-popover title="Agent流转状态" trigger="hover">
                <template #content>
                  <div class="agent-flow-info">
                    <div><b>处理Agent:</b> {{ record.agentInfo?.name || 'N/A' }}</div>
                    <div><b>分流次数:</b> {{ record.agentInfo?.flowCount || 0 }}</div>
                    <div><b>人工介入:</b> {{ record.agentInfo?.humanIntervention ? '是' : '否' }}</div>
                  </div>
                </template>
                <a-tag :color="getAgentStatusColor(record)">
                  {{ getAgentStatusText(record) }}
                </a-tag>
              </a-popover>
            </template>
            <template v-if="column.key === 'action'">
              <a-button type="link" @click="showRepairDetail(record)">详情</a-button>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- 手动故障修复弹窗 -->
    <a-modal 
      v-model:visible="manualRepairVisible" 
      title="手动触发故障修复" 
      width="600px"
      :footer="null"
    >
      <a-form
        ref="formRef"
        :model="manualRepairForm"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
        :rules="formRules"
        @finish="handleManualRepair"
      >
        <a-form-item
          label="Deployment名称"
          name="deployment"
        >
          <a-input v-model:value="manualRepairForm.deployment" placeholder="例如：nginx-deployment" />
        </a-form-item>
        
        <a-form-item
          label="命名空间"
          name="namespace"
        >
          <a-input v-model:value="manualRepairForm.namespace" placeholder="例如：default" />
        </a-form-item>
        
        <a-form-item
          label="故障事件描述"
          name="event"
        >
          <a-textarea 
            v-model:value="manualRepairForm.event" 
            :rows="4"
            placeholder="请详细描述故障现象，例如：容器频繁重启、CPU使用率过高、内存不足等"
          />
        </a-form-item>
        
        <a-form-item :wrapper-col="{ offset: 6, span: 18 }">
          <a-space>
            <a-button type="primary" html-type="submit" :loading="repairSubmitting">
              <template #icon><thunderbolt-outlined /></template>
              开始修复
            </a-button>
            <a-button @click="manualRepairVisible = false">取消</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 修复详情弹窗 -->
    <a-modal v-model:visible="detailVisible" title="修复详情" width="800px" :footer="null">
      <div v-if="selectedRepair" class="repair-detail">
        <div class="detail-header">
          <h2>{{ selectedRepair.faultName }}</h2>
          <a-tag :color="getStatusColor(selectedRepair.status)">{{ selectedRepair.status }}</a-tag>
        </div>
        
        <!-- Agent处理信息 -->
        <a-descriptions bordered>
          <a-descriptions-item label="故障ID" :span="3">{{ selectedRepair.id }}</a-descriptions-item>
          <a-descriptions-item label="故障源" :span="3">{{ selectedRepair.source }}</a-descriptions-item>
          <a-descriptions-item label="发生时间" :span="3">{{ selectedRepair.faultTime }}</a-descriptions-item>
          <a-descriptions-item label="修复时间" :span="3">{{ selectedRepair.repairTime }}</a-descriptions-item>
          <a-descriptions-item label="修复方法" :span="3">{{ selectedRepair.method }}</a-descriptions-item>
          <a-descriptions-item label="故障描述" :span="3">{{ selectedRepair.description }}</a-descriptions-item>
          
          <!-- Agent处理流程 -->
          <a-descriptions-item label="Agent处理流程" :span="3">
            <div class="agent-flow-timeline">
              <a-timeline>
                <a-timeline-item color="blue">
                  <template #dot><robot-outlined /></template>
                  <div class="flow-item">
                    <div class="flow-title">Supervisor接收故障</div>
                    <div class="flow-time">{{ selectedRepair.agentTimeline?.receiveTime }}</div>
                    <div class="flow-desc">{{ selectedRepair.agentTimeline?.receiveDesc }}</div>
                  </div>
                </a-timeline-item>
                <a-timeline-item v-for="(flow, index) in selectedRepair.agentTimeline?.flows" :key="index" 
                                :color="flow.success ? 'green' : 'orange'">
                  <template #dot><api-outlined /></template>
                  <div class="flow-item">
                    <div class="flow-title">Agent处理 #{{ index + 1 }} - {{ flow.agentName }}</div>
                    <div class="flow-time">{{ flow.time }}</div>
                    <div class="flow-desc">{{ flow.description }}</div>
                    <div class="flow-result" :class="flow.success ? 'success' : 'pending'">
                      {{ flow.success ? '处理成功' : '问题未解决, 继续分流' }}
                    </div>
                  </div>
                </a-timeline-item>
                <a-timeline-item v-if="selectedRepair.agentTimeline?.humanIntervention" color="red">
                  <template #dot><user-outlined /></template>
                  <div class="flow-item">
                    <div class="flow-title">人工介入</div>
                    <div class="flow-time">{{ selectedRepair.agentTimeline?.humanTime }}</div>
                    <div class="flow-desc">
                      {{ selectedRepair.agentTimeline?.humanDesc }}
                      <a-tag color="red" v-if="selectedRepair.agentTimeline?.notifyMethod">
                        通过{{ selectedRepair.agentTimeline?.notifyMethod }}通知
                      </a-tag>
                    </div>
                  </div>
                </a-timeline-item>
                <a-timeline-item v-if="selectedRepair.status === '修复成功'" color="green">
                  <template #dot><check-circle-outlined /></template>
                  <div class="flow-item">
                    <div class="flow-title">问题解决</div>
                    <div class="flow-time">{{ selectedRepair.agentTimeline?.resolveTime }}</div>
                    <div class="flow-desc">{{ selectedRepair.agentTimeline?.resolveDesc }}</div>
                  </div>
                </a-timeline-item>
              </a-timeline>
            </div>
          </a-descriptions-item>
          
          <a-descriptions-item label="修复步骤" :span="3">
            <div class="repair-steps">
              <div v-for="(step, index) in selectedRepair.steps" :key="index" class="repair-step">
                <div class="step-number">{{ index + 1 }}</div>
                <div class="step-content">
                  <div class="step-title">{{ step.title }}</div>
                  <div class="step-desc">{{ step.description }}</div>
                  <div class="step-result" :class="step.success ? 'success' : 'failed'">
                    {{ step.success ? '成功' : '失败' }}
                  </div>
                </div>
              </div>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="修复结果" :span="3">{{ selectedRepair.result }}</a-descriptions-item>
        </a-descriptions>
        
        <div class="detail-actions">
          <a-button type="primary" @click="handleRepairAction('runDiagnostic')">
            <template #icon><search-outlined /></template>
            运行诊断
          </a-button>
          <a-button @click="handleRepairAction('exportReport')">
            <template #icon><export-outlined /></template>
            导出报告
          </a-button>
          <a-button type="dashed" @click="handleRepairAction('addToKnowledgeBase')" v-if="selectedRepair.status === '修复成功'">
            <template #icon><book-outlined /></template>
            添加到知识库
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, nextTick, onUnmounted } from 'vue';
import { message } from 'ant-design-vue';
import {
  SyncOutlined,
  ToolOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  SearchOutlined,
  ExportOutlined,
  BookOutlined,
  RobotOutlined,
  ApiOutlined,
  UserOutlined,
  QuestionCircleOutlined,
  CheckOutlined,
  CloseOutlined,
  ArrowRightOutlined,
  RedoOutlined
} from '@ant-design/icons-vue';
import * as echarts from 'echarts';

// 接口定义
interface OngoingRepair {
  id: string;
  deployment: string;
  namespace: string;
  statusText: string;
}

interface AgentFlow {
  agentName: string;
  time: string;
  description: string;
  success: boolean;
}

interface AgentTimeline {
  receiveTime: string;
  receiveDesc: string;
  flows: AgentFlow[];
  humanIntervention?: boolean;
  humanTime?: string;
  humanDesc?: string;
  notifyMethod?: string;
  resolveTime?: string;
  resolveDesc?: string;
}

interface RepairStep {
  title: string;
  description: string;
  success: boolean;
}

interface AgentInfo {
  name: string;
  flowCount: number;
  humanIntervention: boolean;
}

interface RepairRecord {
  id: string;
  faultName: string;
  source: string;
  method: string;
  faultTime: string;
  repairTime: string;
  status: string;
  description: string;
  steps: RepairStep[];
  result: string;
  agentInfo: AgentInfo;
  agentTimeline: AgentTimeline;
}

interface ApiHealthStatus {
  type: 'info' | 'success' | 'warning' | 'error';
  message: string;
}

interface RepairStats {
  total: number;
  totalIncrease: number;
  successRate: number;
  successRateIncrease: number;
  avgTime: string;
  avgTimeDecrease: number;
  automationRate: number;
  automationIncrease: number;
}

interface ManualRepairForm {
  deployment: string;
  namespace: string;
  event: string;
}

// API配置
const API_BASE_URL = 'http://localhost:8080';

// 响应式数据
const timeRange = ref('24h');
const loading = ref(false);
const manualRepairVisible = ref(false);
const repairSubmitting = ref(false);
const detailVisible = ref(false);
const selectedRepair = ref<RepairRecord | null>(null);
const ongoingRepairs = ref<OngoingRepair[]>([]);
const formRef = ref();

// API健康状态
const apiHealthStatus = reactive<ApiHealthStatus>({
  type: 'info',
  message: '正在检查API状态...'
});

// 手动修复表单
const manualRepairForm = reactive<ManualRepairForm>({
  deployment: '',
  namespace: 'default',
  event: ''
});

// 表单验证规则
const formRules = {
  deployment: [
    { required: true, message: '请输入Deployment名称' },
    { pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, message: '请输入有效的Deployment名称' }
  ],
  namespace: [
    { required: true, message: '请输入命名空间' },
    { pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, message: '请输入有效的命名空间' }
  ],
  event: [
    { required: true, message: '请输入故障事件描述' },
    { min: 10, message: '故障描述至少需要10个字符' }
  ]
};

// 统计数据
const repairStats = reactive<RepairStats>({
  total: 17,
  totalIncrease: 5,
  successRate: 92.3,
  successRateIncrease: 1.2,
  avgTime: '1.8分钟',
  avgTimeDecrease: 7,
  automationRate: 88,
  automationIncrease: 2
});

// 图表引用
const trendChartRef = ref<HTMLElement>();
const typeChartRef = ref<HTMLElement>();
const methodChartRef = ref<HTMLElement>();
const timeChartRef = ref<HTMLElement>();

// 图表实例管理
const chartInstances = ref<echarts.ECharts[]>([]);

// 表格列定义
const columns = [
  { title: '故障ID', dataIndex: 'id', key: 'id' },
  { title: '故障名称', dataIndex: 'faultName', key: 'faultName' },
  { title: '故障源', dataIndex: 'source', key: 'source' },
  { title: '修复方法', dataIndex: 'method', key: 'method' },
  { title: '修复时间', dataIndex: 'repairTime', key: 'repairTime' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'Agent流转', dataIndex: 'flowStatus', key: 'flowStatus' },
  { title: '操作', key: 'action' }
];

// 修复记录列表
const repairList = ref<RepairRecord[]>([
  {
    id: 'FLT-20250513-001',
    faultName: 'MySQL主库连接超限',
    source: '数据库服务器-DB01',
    method: '动态调整连接池参数',
    faultTime: '2025-05-13 14:27:35',
    repairTime: '2025-05-13 14:29:12',
    status: '修复成功',
    description: '数据库连接数突增至最大值(max_connections=300)，导致新连接被拒绝，应用程序报错"Too many connections"',
    steps: [
      { title: '连接池监控告警', description: '监测到连接数达到阈值95%(285/300)', success: true },
      { title: '分析连接状态', description: '确认137个idle连接超过5分钟未释放', success: true },
      { title: '识别异常连接来源', description: '定位到订单服务(order-service)未正确关闭连接', success: true },
      { title: '主动回收空闲连接', description: '清理超过2分钟的空闲连接', success: true },
      { title: '临时调整参数', description: '将wait_timeout从600秒调整为300秒', success: true }
    ],
    result: '空闲连接成功回收，连接数从285降至148，数据库服务恢复正常，应用错误率从15%降至0%',
    agentInfo: {
      name: 'DB-Agent',
      flowCount: 1,
      humanIntervention: false
    },
    agentTimeline: {
      receiveTime: '2025-05-13 14:27:35',
      receiveDesc: 'Supervisor接收到数据库连接告警',
      flows: [
        {
          agentName: 'DB-Agent',
          time: '2025-05-13 14:27:40',
          description: '分析连接池状态并执行自动修复',
          success: true
        }
      ],
      resolveTime: '2025-05-13 14:29:12',
      resolveDesc: '问题完全解决，数据库连接恢复正常'
    }
  }
]);

// API调用函数
const apiRequest = async (url: string, options: any = {}) => {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10000);

  try {
    const response = await fetch(`${API_BASE_URL}${url}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      },
      signal: controller.signal,
      ...options
    });
    
    clearTimeout(timeoutId);
    
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`HTTP ${response.status}: ${errorText || response.statusText}`);
    }
    
    return await response.json();
  } catch (error: any) {
    clearTimeout(timeoutId);
    if (error.name === 'AbortError') {
      throw new Error('请求超时，请稍后重试');
    }
    console.error('API请求失败:', error);
    throw error;
  }
};

// 检查API健康状态
const checkApiHealth = async () => {
  try {
    const response = await apiRequest('/health');
    
    if (response.status === 'healthy') {
      apiHealthStatus.type = 'success';
      apiHealthStatus.message = `API服务正常 - 模型已加载: ${response.model_loaded ? '是' : '否'}, 工作流已加载: ${response.workflow_loaded ? '是' : '否'}`;
    } else {
      apiHealthStatus.type = 'warning';
      apiHealthStatus.message = 'API服务异常，请检查后端服务';
    }
  } catch (error: any) {
    apiHealthStatus.type = 'error';
    apiHealthStatus.message = `无法连接到API服务 (${API_BASE_URL}) - ${error.message}`;
  }
};

// 获取预测数据
const getPredictionData = async () => {
  try {
    const response = await apiRequest('/predict');
    console.log('预测数据:', response);
    
    if (response.instances) {
      message.info(`当前QPS: ${response.current_qps.toFixed(2)}, 建议实例数: ${response.instances}`);
    }
    
    return response;
  } catch (error: any) {
    console.error('获取预测数据失败:', error);
    message.error('获取预测数据失败');
  }
};

// 手动触发故障修复
const handleManualRepair = async () => {
  try {
    repairSubmitting.value = true;
    
    const repairData = {
      deployment: manualRepairForm.deployment,
      namespace: manualRepairForm.namespace,
      event: manualRepairForm.event
    };
    
    console.log('发送修复请求:', repairData);
    
    // 添加到正在修复列表
    const ongoingRepair: OngoingRepair = {
      id: `repair-${Date.now()}`,
      deployment: repairData.deployment,
      namespace: repairData.namespace,
      statusText: '正在分析故障...'
    };
    ongoingRepairs.value.push(ongoingRepair);
    
    message.loading('正在提交故障修复请求...', 1);
    
    const response = await apiRequest('/autofix', {
      method: 'POST',
      body: JSON.stringify(repairData),
      headers: {
        'Content-Type': 'application/json'
      }
    });
    
    console.log('修复响应:', response);
    
    // 从正在修复列表中移除
    const index = ongoingRepairs.value.findIndex(item => item.id === ongoingRepair.id);
    if (index !== -1) {
      ongoingRepairs.value.splice(index, 1);
    }
    
    if (response.status === '成功') {
      message.success('故障修复请求已成功处理');
      
      // 添加到修复记录列表
      const newRepair: RepairRecord = {
        id: `FLT-${new Date().toISOString().slice(0, 10).replace(/-/g, '')}-${String(Math.floor(Math.random() * 1000)).padStart(3, '0')}`,
        faultName: `故障修复 - ${repairData.deployment}`,
        source: `${repairData.deployment} (${repairData.namespace})`,
        method: response.result.includes('成功') ? '自动修复' : '人工介入',
        faultTime: new Date().toLocaleString(),
        repairTime: new Date().toLocaleString(),
        status: response.result.includes('成功') ? '修复成功' : '修复中',
        description: repairData.event,
        steps: [
          { title: '接收故障请求', description: '用户手动提交故障修复请求', success: true },
          { title: 'Agent分析处理', description: response.result, success: response.result.includes('成功') }
        ],
        result: response.result,
        agentInfo: {
          name: 'AutoFixK8s',
          flowCount: 1,
          humanIntervention: response.result.includes('人工') || response.result.includes('失败')
        },
        agentTimeline: {
          receiveTime: new Date().toLocaleString(),
          receiveDesc: '用户手动触发故障修复',
          flows: [
            {
              agentName: 'AutoFixK8s',
              time: new Date().toLocaleString(),
              description: response.result,
              success: response.result.includes('成功')
            }
          ],
          humanIntervention: response.result.includes('人工') || response.result.includes('失败'),
          resolveTime: response.result.includes('成功') ? new Date().toLocaleString() : '',
          resolveDesc: response.result.includes('成功') ? response.result : ''
        }
      };
      
      repairList.value.unshift(newRepair);
      
      // 关闭弹窗并重置表单
      manualRepairVisible.value = false;
      Object.assign(manualRepairForm, {
        deployment: '',
        namespace: 'default',
        event: ''
      });
      
      // 更新统计数据
      repairStats.total += 1;
      if (response.result.includes('成功')) {
        const totalSuccessful = repairList.value.filter(item => item.status === '修复成功').length;
        repairStats.successRate = Number(((totalSuccessful / repairList.value.length) * 100).toFixed(1));
      }
      
    } else {
      message.error(`故障修复失败: ${response.error || '未知错误'}`);
    }
    
  } catch (error: any) {
    console.error('故障修复请求失败:', error);
    message.error(`故障修复请求失败: ${error.message}`);
    
    // 从正在修复列表中移除
    const ongoingRepair = ongoingRepairs.value.find((item: OngoingRepair) => 
      item.deployment === manualRepairForm.deployment && 
      item.namespace === manualRepairForm.namespace
    );
    if (ongoingRepair) {
      const index = ongoingRepairs.value.findIndex((item: OngoingRepair) => item.id === ongoingRepair.id);
      if (index !== -1) {
        ongoingRepairs.value.splice(index, 1);
      }
    }
  } finally {
    repairSubmitting.value = false;
  }
};

// 显示手动修复弹窗
const showManualRepairModal = () => {
  manualRepairVisible.value = true;
};

// 显示修复详情
const showRepairDetail = (repair: RepairRecord) => {
  selectedRepair.value = repair;
  detailVisible.value = true;
};

// 处理详情页面按钮操作
const handleRepairAction = (action: string) => {
  switch(action) {
    case 'runDiagnostic':
      message.loading('正在执行故障诊断...', 1.5).then(() => {
        getPredictionData().then(() => {
          message.success('诊断完成，系统运行正常');
        });
      });
      break;
    case 'exportReport':
      message.loading('正在生成故障修复报告...', 1.5).then(() => {
        message.success('报告已生成并发送至相关运维人员');
      });
      break;
    case 'addToKnowledgeBase':
      message.loading('正在添加到知识库...', 1.5).then(() => {
        message.success('故障修复方案已添加到知识库，未来类似问题将自动处理');
      });
      break;
  }
};

// 获取状态颜色
const getStatusColor = (status: string) => {
  switch (status) {
    case '修复成功':
      return 'success';
    case '修复中':
      return 'processing';
    case '修复失败':
      return 'error';
    default:
      return 'default';
  }
};

// 获取Agent流转状态颜色
const getAgentStatusColor = (record: RepairRecord) => {
  if (!record.agentInfo) return 'default';
  
  if (record.agentInfo.humanIntervention) {
    return 'red';
  } else if (record.agentInfo.flowCount > 2) {
    return 'orange';
  } else if (record.agentInfo.flowCount > 1) {
    return 'blue';
  } else {
    return 'green';
  }
};

// 获取Agent流转状态文本
const getAgentStatusText = (record: RepairRecord) => {
  if (!record.agentInfo) return '未知';
  
  if (record.agentInfo.humanIntervention) {
    return '人工介入';
  } else if (record.agentInfo.flowCount > 2) {
    return `多次分流(${record.agentInfo.flowCount})`;
  } else if (record.agentInfo.flowCount > 1) {
    return `重新分流(${record.agentInfo.flowCount})`;
  } else {
    return '直接解决';
  }
};

// 获取最近7天的日期列表
const getRecentDays = () => {
  const days = [];
  const currentDate = new Date();
  for (let i = 6; i >= 0; i--) {
    const date = new Date(currentDate);
    date.setDate(date.getDate() - i);
    days.push(`${date.getMonth() + 1}/${date.getDate()}`);
  }
  return days;
};

// 窗口大小变化处理
const handleResize = () => {
  chartInstances.value.forEach(chart => {
    if (chart && !chart.isDisposed()) {
      chart.resize();
    }
  });
};

// 清理函数
const cleanup = () => {
  chartInstances.value.forEach(chart => {
    if (chart && !chart.isDisposed()) {
      chart.dispose();
    }
  });
  chartInstances.value = [];
  window.removeEventListener('resize', handleResize);
};

// 初始化图表
const initCharts = () => {
  nextTick(() => {
    if (!trendChartRef.value || !typeChartRef.value || !methodChartRef.value || !timeChartRef.value) {
      return;
    }

    // 清理旧的图表实例
    cleanup();

    // 故障修复趋势图
    const trendChart = echarts.init(trendChartRef.value);
    chartInstances.value.push(trendChart);
    
    const days = getRecentDays();
    
    // 根据选择的时间范围调整数据
    let faultData = [];
    let repairData = [];
    let successRateData = [];
    
    if (timeRange.value === '1h') {
      faultData = [0, 0, 0, 0, 0, 1, 1];
      repairData = [0, 0, 0, 0, 0, 1, 1];
      successRateData = [0, 0, 0, 0, 0, 100, 100];
    } else if (timeRange.value === '6h') {
      faultData = [0, 0, 0, 0, 1, 1, 2];
      repairData = [0, 0, 0, 0, 1, 1, 2];
      successRateData = [0, 0, 0, 0, 100, 100, 100];
    } else if (timeRange.value === '24h') {
      faultData = [0, 0, 0, 1, 2, 2, 2];
      repairData = [0, 0, 0, 1, 2, 2, 2];
      successRateData = [0, 0, 0, 100, 100, 85.7, 100];
    } else { // 7d
      faultData = [2, 3, 1, 2, 3, 3, 3];
      repairData = [2, 3, 1, 2, 3, 2, 3];
      successRateData = [100, 90, 100, 100, 93.3, 91.7, 100];
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
        data: ['故障数', '自动修复数', '修复成功率'],
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
        data: days,
        axisLine: {
          lineStyle: {
            color: '#333333'
          }
        }
      },
      yAxis: [
        {
          type: 'value',
          name: '数量',
          axisLine: {
            lineStyle: {
              color: '#333333'
            }
          },
          splitLine: {
            lineStyle: {
              color: 'rgba(0, 0, 0, 0.1)'
            }
          }
        },
        {
          type: 'value',
          name: '成功率',
          min: 0,
          max: 100,
          interval: 20,
          axisLabel: {
            formatter: '{value}%',
            color: '#333333'
          },
          axisLine: {
            lineStyle: {
              color: '#333333'
            }
          },
          splitLine: {
            show: false
          }
        }
      ],
      series: [
        {
          name: '故障数',
          type: 'bar',
          data: faultData,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#ff9a9e' },
              { offset: 1, color: '#fad0c4' }
            ])
          }
        },
        {
          name: '自动修复数',
          type: 'bar',
          data: repairData,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#a1c4fd' },
              { offset: 1, color: '#c2e9fb' }
            ])
          }
        },
        {
          name: '修复成功率',
          type: 'line',
          yAxisIndex: 1,
          data: successRateData,
          lineStyle: {
            width: 3,
            color: '#00f2fe'
          },
          symbol: 'circle',
          symbolSize: 8,
          itemStyle: {
            color: '#00f2fe'
          }
        }
      ]
    });

    // 故障类型分布图
    const typeChart = echarts.init(typeChartRef.value);
    chartInstances.value.push(typeChart);
    
    let typeData = [];
    
    if (timeRange.value === '1h' || timeRange.value === '6h') {
      typeData = [
        { value: 1, name: '数据库故障', itemStyle: { color: '#ff9a9e' } },
        { value: 1, name: '资源不足(OOM)', itemStyle: { color: '#d4fc79' } },
        { value: 1, name: '镜像版本错误', itemStyle: { color: '#a1c4fd' } },
        { value: 1, name: '网络故障', itemStyle: { color: '#fbc2eb' } }
      ];
    } else if (timeRange.value === '24h') {
      typeData = [
        { value: 2, name: '数据库故障', itemStyle: { color: '#ff9a9e' } },
        { value: 2, name: '资源不足(OOM)', itemStyle: { color: '#d4fc79' } },
        { value: 1, name: '镜像版本错误', itemStyle: { color: '#a1c4fd' } },
        { value: 1, name: '网络故障', itemStyle: { color: '#fbc2eb' } },
        { value: 1, name: '存储故障', itemStyle: { color: '#84fab0' } }
      ];
    } else { // 7d
      typeData = [
        { value: 4, name: '数据库故障', itemStyle: { color: '#ff9a9e' } },
        { value: 4, name: '资源不足(OOM)', itemStyle: { color: '#d4fc79' } },
        { value: 3, name: '镜像版本错误', itemStyle: { color: '#a1c4fd' } },
        { value: 3, name: '网络故障', itemStyle: { color: '#fbc2eb' } },
        { value: 2, name: '系统故障', itemStyle: { color: '#ffd1ff' } },
        { value: 1, name: '存储故障', itemStyle: { color: '#84fab0' } }
      ];
    }
    
    typeChart.setOption({
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
        formatter: '{a} <br/>{b}: {c} ({d}%)'
      },
      legend: {
        orient: 'vertical',
        right: 10,
        top: 'center',
        data: typeData.map(item => item.name),
        textStyle: {
          color: '#333333'
        }
      },
      series: [
        {
          name: '故障类型',
          type: 'pie',
          radius: ['50%', '70%'],
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
          data: typeData
        }
      ]
    });

    // 修复方法分布图
    const methodChart = echarts.init(methodChartRef.value);
    chartInstances.value.push(methodChart);
    
    let methodData = [];
    
    if (timeRange.value === '1h' || timeRange.value === '6h') {
      methodData = [
        { value: 2, name: '配置调整', itemStyle: { color: '#0ba360' } },
        { value: 1, name: '自动资源扩容', itemStyle: { color: '#00f2fe' } },
        { value: 1, name: '版本回滚', itemStyle: { color: '#4facfe' } }
      ];
    } else if (timeRange.value === '24h') {
      methodData = [
        { value: 2, name: '配置调整', itemStyle: { color: '#0ba360' } },
        { value: 2, name: '自动资源扩容', itemStyle: { color: '#00f2fe' } },
        { value: 1, name: '版本回滚', itemStyle: { color: '#4facfe' } },
        { value: 1, name: '清理操作', itemStyle: { color: '#f093fb' } },
        { value: 1, name: '人工介入', itemStyle: { color: '#fa709a' } }
      ];
    } else { // 7d
      methodData = [
        { value: 4, name: '配置调整', itemStyle: { color: '#0ba360' } },
        { value: 4, name: '自动资源扩容', itemStyle: { color: '#00f2fe' } },
        { value: 3, name: '版本回滚', itemStyle: { color: '#4facfe' } },
        { value: 3, name: '清理操作', itemStyle: { color: '#f093fb' } },
        { value: 3, name: '人工介入', itemStyle: { color: '#fa709a' } }
      ];
    }
    
    methodChart.setOption({
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
        formatter: '{a} <br/>{b}: {c} ({d}%)'
      },
      legend: {
        orient: 'vertical',
        right: 10,
        top: 'center',
        data: methodData.map(item => item.name),
        textStyle: {
          color: '#333333'
        }
      },
      series: [
        {
          name: '修复方法',
          type: 'pie',
          radius: ['50%', '70%'],
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
          data: methodData
        }
      ]
    });

    // 修复时间分布图
    const timeChart = echarts.init(timeChartRef.value);
    chartInstances.value.push(timeChart);
    
    let timeData = [];
    
    if (timeRange.value === '1h' || timeRange.value === '6h') {
      timeData = [2, 1, 1, 0, 0];
    } else if (timeRange.value === '24h') {
      timeData = [3, 2, 1, 1, 0];
    } else { // 7d
      timeData = [8, 5, 2, 1, 1];
    }
    
    timeChart.setOption({
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
        type: 'category',
        data: ['<1分钟', '1-2分钟', '2-5分钟', '5-10分钟', '>10分钟'],
        axisLine: {
          lineStyle: {
            color: '#333333'
          }
        }
      },
      yAxis: {
        type: 'value',
        axisLine: {
          lineStyle: {
            color: '#333333'
          }
        },
        splitLine: {
          lineStyle: {
            color: 'rgba(0, 0, 0, 0.1)'
          }
        }
      },
      series: [
        {
          name: '修复时间分布',
          type: 'bar',
          data: timeData,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#43e97b' },
              { offset: 1, color: '#38f9d7' }
            ])
          }
        }
      ]
    });

    // 添加窗口大小变化监听
    window.addEventListener('resize', handleResize);
  });
};

// 刷新数据
const refreshData = async () => {
  loading.value = true;
  message.loading('正在加载修复数据...', 1);
  
  try {
    // 检查API健康状态
    await checkApiHealth();
    
    // 获取预测数据
    await getPredictionData();
    
    // 根据选择的时间范围更新统计数据
    switch(timeRange.value) {
      case '1h':
        Object.assign(repairStats, {
          total: 2,
          totalIncrease: 0,
          successRate: 100,
          successRateIncrease: 0,
          avgTime: '1.5分钟',
          avgTimeDecrease: 5,
          automationRate: 100,
          automationIncrease: 0
        });
        break;
      case '6h':
        Object.assign(repairStats, {
          total: 4,
          totalIncrease: 0,
          successRate: 100,
          successRateIncrease: 0,
          avgTime: '1.7分钟',
          avgTimeDecrease: 3,
          automationRate: 100,
          automationIncrease: 0
        });
        break;
      case '24h':
        Object.assign(repairStats, {
          total: 7,
          totalIncrease: 2,
          successRate: 93.5,
          successRateIncrease: 0.8,
          avgTime: '1.8分钟',
          avgTimeDecrease: 7,
          automationRate: 91,
          automationIncrease: 1
        });
        break;
      case '7d':
        Object.assign(repairStats, {
          total: 17,
          totalIncrease: 5,
          successRate: 92.3,
          successRateIncrease: 1.2,
          avgTime: '1.8分钟',
          avgTimeDecrease: 7,
          automationRate: 88,
          automationIncrease: 2
        });
        break;
    }
    
    // 重新初始化图表
    initCharts();
    message.success('数据已刷新');
    
  } catch (error) {
    console.error('刷新数据失败:', error);
    message.error('刷新数据失败，请检查网络连接');
  } finally {
    loading.value = false;
  }
};

// 页面加载完成后初始化
onMounted(() => {
  refreshData();
});

// 组件卸载时清理
onUnmounted(() => {
  cleanup();
});
</script>

<style scoped>
.fault-repair-container {
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
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
  gap: 10px;
}

.manual-repair-btn {
  background: linear-gradient(45deg, #ff6b6b, #ffa726);
  border: none;
  color: white;
  box-shadow: 0 4px 8px rgba(255, 107, 107, 0.3);
  transition: all 0.3s ease;
}

.manual-repair-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 12px rgba(255, 107, 107, 0.4);
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
}

.stat-card {
  border-radius: 10px;
  overflow: hidden;
  transition: all 0.3s ease;
  border: 1px solid var(--ant-border-color-split);
}

.repair-card {
  position: relative;
  overflow: hidden;
}

.repair-card::before {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  z-index: -1;
  border-radius: 12px;
  background: linear-gradient(45deg, #1890ff, #52c41a, #1890ff);
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
  margin-bottom: 10px;
  color: var(--ant-heading-color);
}

.stat-trend {
  display: flex;
  align-items: center;
  font-size: 14px;
  gap: 5px;
}

.up {
  color: var(--ant-success-color);
}

.down {
  color: var(--ant-error-color);
}

/* 实时修复状态样式 */
.real-time-status {
  border-radius: 10px;
  border: 2px solid #1890ff;
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.05), rgba(82, 196, 26, 0.05));
}

.ongoing-repairs {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ongoing-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: rgba(24, 144, 255, 0.1);
  border-radius: 8px;
  border: 1px solid rgba(24, 144, 255, 0.2);
}

.repair-name {
  font-weight: bold;
  color: #1890ff;
}

.repair-namespace {
  background: rgba(24, 144, 255, 0.2);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.repair-status {
  color: #666;
  font-style: italic;
}

/* 工作流程图样式 */
.workflow-card {
  margin-bottom: 20px;
  border-radius: 10px;
  border: 1px solid var(--ant-border-color-split);
  transition: all 0.3s ease;
}

.workflow-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.15);
}

.workflow-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.workflow-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 120px;
  height: 120px;
  border-radius: 10px;
  color: white;
  font-weight: bold;
  text-align: center;
  position: relative;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
  transition: all 0.3s ease;
}

.workflow-node:hover {
  transform: scale(1.05);
}

.workflow-node.supervisor {
  background: linear-gradient(135deg, #667eea, #764ba2);
}

.workflow-node.agent {
  background: linear-gradient(135deg, #2af598, #009efd);
}

.node-title {
  margin-top: 8px;
  font-size: 14px;
}

.node-desc {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.9;
}

.workflow-arrow {
  margin: 15px 0;
  color: #1890ff;
  font-size: 24px;
}

.workflow-decision {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin: 15px 0;
  padding: 15px;
  background: #fffbe6;
  border: 1px dashed #faad14;
  border-radius: 8px;
  color: #d46b08;
  font-weight: bold;
}

.decision-title {
  margin-top: 8px;
}

.workflow-branches {
  display: flex;
  justify-content: center;
  gap: 80px;
  width: 100%;
  margin-top: 15px;
}

.branch {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
}

.branch span {
  margin: 8px 0;
  font-weight: bold;
}

.branch-node {
  margin-top: 15px;
  padding: 12px 20px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  color: white;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.branch-node.success {
  background: linear-gradient(135deg, #52c41a, #b7eb8f);
}

.branch-node.human {
  background: linear-gradient(135deg, #f5222d, #ff7875);
  margin-top: 40px;
}

.human-notify {
  margin-top: 5px;
  background: rgba(255, 255, 255, 0.3);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.loop-arrow {
  position: absolute;
  top: 60px;
  right: -40px;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-size: 12px;
  color: #1890ff;
}

/* Agent流程时间线样式 */
.agent-flow-timeline {
  padding: 10px;
}

.flow-item {
  padding: 10px;
  border-radius: 6px;
  background-color: rgba(24, 144, 255, 0.05);
}

.flow-title {
  font-weight: bold;
  margin-bottom: 5px;
}

.flow-time {
  font-size: 12px;
  color: #999;
  margin-bottom: 5px;
}

.flow-desc {
  margin-bottom: 5px;
}

.flow-result {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  margin-top: 5px;
}

.flow-result.success {
  background-color: #f6ffed;
  border: 1px solid #b7eb8f;
  color: #52c41a;
}

.flow-result.pending {
  background-color: #e6f7ff;
  border: 1px solid #91d5ff;
  color: #1890ff;
}

.flow-result.failed {
  background-color: #fff2f0;
  border: 1px solid #ffccc7;
  color: #ff4d4f;
}

.agent-flow-info {
  padding: 8px;
}

.charts-container {
  margin-top: 20px;
}

.chart-card {
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--ant-border-color-split);
  height: 350px;
  transition: all 0.3s ease;
  margin-bottom: 16px;
}

.chart-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.chart {
  height: 300px;
}

.recent-repairs {
  margin-top: 20px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--ant-border-color-split);
  transition: all 0.3s ease;
}

.recent-repairs:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.repair-detail {
  color: var(--ant-text-color);
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.repair-steps {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.repair-step {
  background-color: var(--ant-background-color-base);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid var(--ant-border-color-split);
  transition: all 0.3s ease;
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.repair-step:hover {
  transform: translateX(5px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.step-number {
  width: 24px;
  height: 24px;
  background: linear-gradient(45deg, #1890ff, #52c41a);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  flex-shrink: 0;
}

.step-content {
  flex: 1;
}

.step-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--ant-heading-color);
  margin-bottom: 8px;
}

.step-desc {
  color: var(--ant-text-color);
  margin-bottom: 8px;
}

.step-result {
  padding: 2px 8px;
  border-radius: 4px;
  display: inline-block;
  font-size: 12px;
}

.step-result.success {
  background-color: #f6ffed;
  border: 1px solid #b7eb8f;
  color: #52c41a;
}

.step-result.failed {
  background-color: #fff2f0;
  border: 1px solid #ffccc7;
  color: #ff4d4f;
}

.detail-actions {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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
  
  .workflow-branches {
    flex-direction: column;
    align-items: center;
    gap: 40px;
  }
  
  .branch.no .loop-arrow {
    position: static;
    margin: 20px 0;
  }
  
  .branch-node.human {
    margin-top: 15px;
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
    flex-wrap: wrap;
    gap: 8px;
  }

  .manual-repair-btn {
    width: 100%;
    margin-bottom: 8px;
  }

  .time-selector {
    width: calc(50% - 4px) !important;
  }

  .refresh-btn {
    width: calc(50% - 4px);
  }
  
  .workflow-node {
    width: 100px;
    height: 100px;
  }
}

@media (max-width: 576px) {
  .workflow-container {
    padding: 10px;
  }
  
  .workflow-node {
    width: 80px;
    height: 80px;
    font-size: 12px;
  }
  
  .node-title {
    font-size: 10px;
  }
  
  .node-desc {
    font-size: 8px;
  }
}
</style>