<template>
  <div class="cloud-provider-container">
    <a-page-header title="云提供商管理" subtitle="管理您的云账户和资源配置" class="page-header">
      <template #extra>
        <a-button type="primary" @click="showAddAccountModal">
          <template #icon>
            <plus-outlined />
          </template>
          添加云账户
        </a-button>
        <a-button @click="refreshData">
          <template #icon>
            <reload-outlined />
          </template>
          刷新
        </a-button>
      </template>
    </a-page-header>

    <div class="content-layout">
      <!-- 资源统计概览卡片 -->
      <a-row :gutter="16" class="dashboard-cards">
        <a-col :xs="24" :sm="12" :md="8" :lg="8" :xl="6">
          <a-card class="stats-card">
            <div class="stats-card-content">
              <div class="stats-info">
                <div class="stats-title">实例总数</div>
                <div class="stats-value">{{ cloudStats.totalEcsCount }}</div>
                <div class="stats-desc">
                  <span class="stats-highlight"><check-circle-outlined /> {{ cloudStats.runningEcsCount }} 运行中</span>
                  <span class="stats-muted"><pause-circle-outlined /> {{ cloudStats.stoppedEcsCount }} 已停止</span>
                </div>
              </div>
              <div class="stats-icon">
                <cloud-server-outlined />
              </div>
            </div>
          </a-card>
        </a-col>
        <a-col :xs="24" :sm="12" :md="8" :lg="8" :xl="6">
          <a-card class="stats-card">
            <div class="stats-card-content">
              <div class="stats-info">
                <div class="stats-title">VPC网络</div>
                <div class="stats-value">{{ cloudStats.totalVpcCount }}</div>
                <div class="stats-desc">
                  <span class="stats-highlight">多区域互联网络</span>
                </div>
              </div>
              <div class="stats-icon">
                <global-outlined />
              </div>
            </div>
          </a-card>
        </a-col>
        <a-col :xs="24" :sm="12" :md="8" :lg="8" :xl="6">
          <a-card class="stats-card">
            <div class="stats-card-content">
              <div class="stats-info">
                <div class="stats-title">安全组</div>
                <div class="stats-value">{{ cloudStats.totalSecurityGroupCount }}</div>
                <div class="stats-desc">
                  <span class="stats-highlight">网络访问控制</span>
                </div>
              </div>
              <div class="stats-icon">
                <safety-outlined />
              </div>
            </div>
          </a-card>
        </a-col>
        <a-col :xs="24" :sm="12" :md="8" :lg="8" :xl="6">
          <a-card class="stats-card">
            <div class="stats-card-content">
              <div class="stats-info">
                <div class="stats-title">月度费用</div>
                <div class="stats-value">¥{{ cloudStats.totalMonthlyCost.toFixed(2) }}</div>
                <div class="stats-desc">
                  <span class="stats-highlight">更新于: {{ formatDate(cloudStats.updateTime) }}</span>
                </div>
              </div>
              <div class="stats-icon">
                <account-book-outlined />
              </div>
            </div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 图表展示区域 -->
      <a-row :gutter="16" class="chart-row">
        <a-col :xs="24" :lg="12">
          <a-card title="实例状态分布" class="chart-card">
            <div ref="instanceStatusChart" class="chart-container"></div>
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card title="区域资源分布" class="chart-card">
            <div ref="regionDistributionChart" class="chart-container"></div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 云账户列表 -->
      <a-card title="云账户列表" class="account-card">
        <template #extra>
          <a-space>
            <a-select v-model:value="providerFilter" placeholder="云提供商" style="width: 130px" allowClear>
              <a-select-option v-for="provider in providers" :key="provider.provider" :value="provider.provider">
                {{ provider.localName }}
              </a-select-option>
            </a-select>
            <a-select v-model:value="statusFilter" placeholder="状态" style="width: 100px" allowClear>
              <a-select-option value="enabled">已启用</a-select-option>
              <a-select-option value="disabled">已禁用</a-select-option>
            </a-select>
            <a-input-search v-model:value="searchValue" placeholder="搜索账户" style="width: 220px" @search="onSearch" />
          </a-space>
        </template>

        <a-table :dataSource="filteredAccounts" :columns="columns" :loading="loading" :pagination="{ pageSize: 10 }"
          rowKey="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'provider'">
              <a-tag :color="getProviderColor(record.provider)">
                {{ getProviderName(record.provider) }}
              </a-tag>
            </template>
            <template v-if="column.key === 'isEnabled'">
              <a-badge :status="record.isEnabled ? 'success' : 'default'" :text="record.isEnabled ? '已启用' : '已禁用'" />
            </template>
            <template v-if="column.key === 'regions'">
              <a-tooltip v-if="record.regions && record.regions.length > 0">
                <template #title>
                  <div v-for="region in record.regions" :key="region">{{ region }}</div>
                </template>
                <a-tag>{{ record.regions.length }} 个区域</a-tag>
              </a-tooltip>
              <span v-else>-</span>
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button type="link" size="small" @click="viewAccountDetails(record)">
                  <template #icon><eye-outlined /></template>
                  详情
                </a-button>
                <a-button type="link" size="small" @click="editAccount(record)">
                  <template #icon><edit-outlined /></template>
                  编辑
                </a-button>
                <a-button type="link" size="small" @click="syncAccount(record)" :loading="record.syncing">
                  <template #icon><sync-outlined /></template>
                  同步
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item @click="toggleAccountStatus(record)">
                        {{ record.isEnabled ? '禁用账户' : '启用账户' }}
                      </a-menu-item>
                      <a-menu-item danger @click="confirmDeleteAccount(record)">删除账户</a-menu-item>
                    </a-menu>
                  </template>
                  <a-button type="link" size="small">
                    <more-outlined />
                  </a-button>
                </a-dropdown>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- 添加/编辑云账户对话框 -->
    <a-modal v-model:visible="accountModalVisible" :title="isEditing ? '编辑云账户' : '添加云账户'" @ok="handleAccountSubmit"
      :confirmLoading="submitLoading" width="600px">
      <a-form :model="accountForm" :rules="accountRules" ref="accountFormRef" :label-col="{ span: 6 }"
        :wrapper-col="{ span: 16 }">
        <a-form-item label="账户名称" name="name">
          <a-input v-model:value="accountForm.name" placeholder="请输入账户名称" />
        </a-form-item>
        <a-form-item label="云提供商" name="provider">
          <a-select v-model:value="accountForm.provider" placeholder="请选择云提供商" @change="handleProviderChange">
            <a-select-option v-for="provider in providers" :key="provider.provider" :value="provider.provider">
              {{ provider.localName }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="账户ID" name="accountId">
          <a-input v-model:value="accountForm.accountId" placeholder="请输入账户ID" />
        </a-form-item>
        <a-form-item label="访问密钥ID" name="accessKey">
          <a-input v-model:value="accountForm.accessKey" placeholder="请输入访问密钥ID" />
        </a-form-item>
        <a-form-item label="访问密钥" name="secretKey">
          <a-input-password v-model:value="accountForm.secretKey" placeholder="请输入访问密钥" />
        </a-form-item>
        <a-form-item label="可用区域" name="regions">
          <a-select v-model:value="accountForm.regions" mode="multiple" placeholder="请选择可用区域" :options="regionOptions"
            :loading="regionsLoading"></a-select>
        </a-form-item>
        <a-form-item label="是否启用" name="isEnabled">
          <a-switch v-model:checked="accountForm.isEnabled" />
        </a-form-item>
        <a-form-item label="账户描述" name="description">
          <a-textarea v-model:value="accountForm.description" placeholder="请输入账户描述" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 账户详情抽屉 -->
    <a-drawer v-model:visible="detailDrawerVisible" title="云账户详情" width="600" :destroyOnClose="true">
      <a-descriptions bordered :column="1">
        <a-descriptions-item label="账户名称">{{ selectedAccount.name }}</a-descriptions-item>
        <a-descriptions-item label="云提供商">
          <a-tag :color="getProviderColor(selectedAccount.provider)">
            {{ getProviderName(selectedAccount.provider) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="账户ID">{{ selectedAccount.accountId }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-badge :status="selectedAccount.isEnabled ? 'success' : 'default'"
            :text="selectedAccount.isEnabled ? '已启用' : '已禁用'" />
        </a-descriptions-item>
        <a-descriptions-item label="最后同步时间">{{ selectedAccount.lastSyncTime }}</a-descriptions-item>
        <a-descriptions-item label="账户描述">{{ selectedAccount.description || '-' }}</a-descriptions-item>
      </a-descriptions>

      <a-divider orientation="left">可用区域</a-divider>
      <a-list :dataSource="selectedAccount.regions || []" :bordered="true">
        <template #renderItem="{ item }">
          <a-list-item>{{ item }}</a-list-item>
        </template>
        <template #emptyText>
          <a-empty description="暂无可用区域" />
        </template>
      </a-list>

      <a-divider orientation="left">资源概览</a-divider>
      <div ref="accountResourceChart" class="account-resource-chart"></div>

      <div class="drawer-actions">
        <a-button type="primary" @click="editAccount(selectedAccount)">
          <template #icon><edit-outlined /></template>
          编辑账户
        </a-button>
        <a-button @click="syncAccount(selectedAccount)">
          <template #icon><sync-outlined /></template>
          同步资源
        </a-button>
        <a-button danger @click="confirmDeleteAccount(selectedAccount)">
          <template #icon><delete-outlined /></template>
          删除账户
        </a-button>
      </div>
    </a-drawer>

    <!-- 删除确认对话框 -->
    <a-modal v-model:visible="deleteModalVisible" title="删除确认" @ok="deleteAccount" :okButtonProps="{ danger: true }"
      okText="删除" cancelText="取消">
      <p>确定要删除云账户 "{{ selectedAccount.name }}" 吗？此操作不可恢复。</p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch, nextTick } from 'vue';
import { message } from 'ant-design-vue';
import {
  PlusOutlined,
  ReloadOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  SyncOutlined,
  MoreOutlined,
  CloudServerOutlined,
  GlobalOutlined,
  SafetyOutlined,
  CheckCircleOutlined,
  PauseCircleOutlined,
  AccountBookOutlined
} from '@ant-design/icons-vue';
import * as echarts from 'echarts';

// 类型定义
interface CloudAccount {
  id: number;
  name: string;
  provider: string;
  accountId: string;
  accessKey: string;
  regions: string[];
  isEnabled: boolean;
  lastSyncTime: string;
  description: string;
  syncing?: boolean;
}

interface Provider {
  provider: string;
  localName: string;
}

interface Region {
  regionId: string;
  localName: string;
  regionEndpoint: string;
}

interface CloudStatistics {
  regionDistribution: number;
  totalEcsCount: number;
  runningEcsCount: number;
  stoppedEcsCount: number;
  totalVpcCount: number;
  totalSecurityGroupCount: number;
  totalMonthlyCost: number;
  updateTime: number;
}

// 表格列定义
const columns = [
  {
    title: '账户名称',
    dataIndex: 'name',
    key: 'name',
    sorter: (a: CloudAccount, b: CloudAccount) => a.name.localeCompare(b.name),
  },
  {
    title: '云提供商',
    dataIndex: 'provider',
    key: 'provider',
    filters: [
      { text: '阿里云', value: 'aliyun' },
      { text: '腾讯云', value: 'tencent' },
      { text: '华为云', value: 'huawei' },
      { text: 'AWS', value: 'aws' },
      { text: 'Azure', value: 'azure' },
      { text: 'Google Cloud', value: 'gcp' },
    ],
    onFilter: (value: string, record: CloudAccount) => record.provider === value,
  },
  {
    title: '账户ID',
    dataIndex: 'accountId',
    key: 'accountId',
  },
  {
    title: '状态',
    dataIndex: 'isEnabled',
    key: 'isEnabled',
    filters: [
      { text: '已启用', value: true },
      { text: '已禁用', value: false },
    ],
    onFilter: (value: boolean, record: CloudAccount) => record.isEnabled === value,
  },
  {
    title: '可用区域',
    dataIndex: 'regions',
    key: 'regions',
  },
  {
    title: '最后同步时间',
    dataIndex: 'lastSyncTime',
    key: 'lastSyncTime',
    sorter: (a: CloudAccount, b: CloudAccount) => new Date(a.lastSyncTime).getTime() - new Date(b.lastSyncTime).getTime(),
  },
  {
    title: '操作',
    key: 'action',
    fixed: 'right',
    width: 200,
  },
];

// 状态变量
const loading = ref(false);
const searchValue = ref('');
const providerFilter = ref('');
const statusFilter = ref('');
const cloudAccounts = ref<CloudAccount[]>([]);
const providers = ref<Provider[]>([]);
const regions = ref<Region[]>([]);
const regionsLoading = ref(false);
const accountModalVisible = ref(false);
const detailDrawerVisible = ref(false);
const deleteModalVisible = ref(false);
const submitLoading = ref(false);
const isEditing = ref(false);
const selectedAccount = ref<CloudAccount>({} as CloudAccount);
const accountFormRef = ref();

// 图表引用
const instanceStatusChart = ref();
const regionDistributionChart = ref();
const accountResourceChart = ref();

// 云资源统计数据
const cloudStats = ref<CloudStatistics>({
  regionDistribution: 12,
  totalEcsCount: 1,
  runningEcsCount: 1,
  stoppedEcsCount: 0,
  totalVpcCount: 1,
  totalSecurityGroupCount: 1,
  totalMonthlyCost: 10.94,
  updateTime: Date.now(),
});

// 表单数据
const accountForm = reactive({
  id: 0,
  name: '',
  provider: '',
  accountId: '',
  accessKey: '',
  secretKey: '',
  regions: [] as string[],
  isEnabled: true,
  description: '',
});

// 表单验证规则
const accountRules = {
  name: [{ required: true, message: '请输入账户名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择云提供商', trigger: 'change' }],
  accountId: [{ required: true, message: '请输入账户ID', trigger: 'blur' }],
  accessKey: [{ required: true, message: '请输入访问密钥ID', trigger: 'blur' }],
  secretKey: [{ required: true, message: '请输入访问密钥', trigger: 'blur' }],
};

// 计算属性
const regionOptions = computed(() => {
  return regions.value.map(region => ({
    label: `${region.localName} (${region.regionId})`,
    value: region.regionId,
  }));
});

const filteredAccounts = computed(() => {
  let result = [...cloudAccounts.value];
  
  // 根据搜索值过滤
  if (searchValue.value) {
    const searchText = searchValue.value.toLowerCase();
    result = result.filter(account => 
      account.name.toLowerCase().includes(searchText) || 
      account.accountId.toLowerCase().includes(searchText) ||
      account.description?.toLowerCase().includes(searchText)
    );
  }
  
  // 根据云提供商过滤
  if (providerFilter.value) {
    result = result.filter(account => account.provider === providerFilter.value);
  }
  
  // 根据状态过滤
  if (statusFilter.value) {
    const isEnabled = statusFilter.value === 'enabled';
    result = result.filter(account => account.isEnabled === isEnabled);
  }
  
  return result;
});

// 生命周期钩子
onMounted(() => {
  fetchProviders();
  fetchCloudAccounts();
  
  // 初始化图表
  nextTick(() => {
    initInstanceStatusChart();
    initRegionDistributionChart();
  });
});

// 监听抽屉显示状态
watch(detailDrawerVisible, (newVal) => {
  if (newVal) {
    nextTick(() => {
      initAccountResourceChart();
    });
  }
});

// 格式化日期
const formatDate = (timestamp: any) => {
  const date = new Date(timestamp);
  return date.toLocaleString();
};

// 方法
const fetchProviders = () => {
  // 模拟获取云提供商列表
  providers.value = [
    { provider: 'aliyun', localName: '阿里云' },
    { provider: 'huawei', localName: '华为云' },
    { provider: 'tencent', localName: '腾讯云' },
    { provider: 'aws', localName: 'AWS' },
    { provider: 'azure', localName: 'Azure' },
    { provider: 'gcp', localName: 'Google Cloud' },
    { provider: 'local', localName: '本地环境' },
  ];
};

const fetchCloudAccounts = () => {
  loading.value = true;
  // 模拟获取云账户列表
  setTimeout(() => {
    cloudAccounts.value = [
      {
        id: 1754247775246957,
        name: '生产环境-阿里云',
        provider: 'aliyun',
        accountId: 'aliyun123456',
        accessKey: 'LTAI4*********',
        regions: ['cn-beijing', 'cn-shanghai', 'cn-hangzhou'],
        isEnabled: true,
        lastSyncTime: '2025-05-15 14:30:22',
        description: '阿里云生产环境账户',
        syncing: false,
      }
    ];
    loading.value = false;
  }, 500);
};

const fetchRegions = (provider: string) => {
  regionsLoading.value = true;
  // 模拟获取区域列表
  setTimeout(() => {
    if (provider === 'aliyun') {
      regions.value = [
        { regionId: 'cn-beijing', localName: '华北2（北京）', regionEndpoint: 'ecs.cn-beijing.aliyuncs.com' },
        { regionId: 'cn-shanghai', localName: '华东2（上海）', regionEndpoint: 'ecs.cn-shanghai.aliyuncs.com' },
        { regionId: 'cn-hangzhou', localName: '华东1（杭州）', regionEndpoint: 'ecs.cn-hangzhou.aliyuncs.com' },
        { regionId: 'cn-shenzhen', localName: '华南1（深圳）', regionEndpoint: 'ecs.cn-shenzhen.aliyuncs.com' },
      ];
    } else if (provider === 'tencent') {
      regions.value = [
        { regionId: 'ap-beijing', localName: '华北地区(北京)', regionEndpoint: 'cvm.ap-beijing.tencentcloudapi.com' },
        { regionId: 'ap-shanghai', localName: '华东地区(上海)', regionEndpoint: 'cvm.ap-shanghai.tencentcloudapi.com' },
        { regionId: 'ap-guangzhou', localName: '华南地区(广州)', regionEndpoint: 'cvm.ap-guangzhou.tencentcloudapi.com' },
      ];
    } else if (provider === 'huawei') {
      regions.value = [
        { regionId: 'cn-north-4', localName: '华北-北京四', regionEndpoint: 'ecs.cn-north-4.myhuaweicloud.com' },
        { regionId: 'cn-east-3', localName: '华东-上海一', regionEndpoint: 'ecs.cn-east-3.myhuaweicloud.com' },
        { regionId: 'cn-south-1', localName: '华南-广州', regionEndpoint: 'ecs.cn-south-1.myhuaweicloud.com' },
      ];
    } else if (provider === 'aws') {
      regions.value = [
        { regionId: 'us-east-1', localName: '美国东部（弗吉尼亚北部）', regionEndpoint: 'ec2.us-east-1.amazonaws.com' },
        { regionId: 'us-west-2', localName: '美国西部（俄勒冈）', regionEndpoint: 'ec2.us-west-2.amazonaws.com' },
        { regionId: 'ap-northeast-1', localName: '亚太地区（东京）', regionEndpoint: 'ec2.ap-northeast-1.amazonaws.com' },
      ];
    } else if (provider === 'azure') {
      regions.value = [
        { regionId: 'eastus', localName: '美国东部', regionEndpoint: 'management.azure.com' },
        { regionId: 'westeurope', localName: '西欧', regionEndpoint: 'management.azure.com' },
      ];
    } else if (provider === 'gcp') {
      regions.value = [
        { regionId: 'us-central1', localName: '美国中部（爱荷华）', regionEndpoint: 'compute.googleapis.com' },
        { regionId: 'asia-east1', localName: '亚洲东部（台湾）', regionEndpoint: 'compute.googleapis.com' },
      ];
    } else {
      regions.value = [];
    }
    regionsLoading.value = false;
  }, 300);
};

// 实例状态图表初始化
const initInstanceStatusChart = () => {
  if (!instanceStatusChart.value) return;
  
  const chart = echarts.init(instanceStatusChart.value);
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '实例状态',
        type: 'pie',
        radius: ['50%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
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
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: cloudStats.value.runningEcsCount, name: '运行中', itemStyle: { color: '#52c41a' } },
          { value: cloudStats.value.stoppedEcsCount, name: '已停止', itemStyle: { color: '#d9d9d9' } }
        ]
      }
    ]
  };
  
  chart.setOption(option);
  
  // 窗口大小变化时，重新调整图表大小
  window.addEventListener('resize', () => {
    chart.resize();
  });
};

// 区域分布图表初始化
const initRegionDistributionChart = () => {
  if (!regionDistributionChart.value) return;
  
  const chart = echarts.init(regionDistributionChart.value);
  
  // 模拟区域分布数据
  const regionData = [
    { name: '华北地区', value: 1 },
    { name: '华东地区', value: 0 },
    { name: '华南地区', value: 0 },
    { name: '西南地区', value: 0 },
    { name: '海外地区', value: 0 }
  ];
  
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
      data: regionData.map(item => item.name),
      axisLabel: {
        interval: 0,
        rotate: 0
      }
    },
    series: [
      {
        name: '资源数量',
        type: 'bar',
        data: regionData.map(item => item.value),
        itemStyle: {
          color: function(params: any) {
            const colorList = ['#1890ff', '#13c2c2', '#52c41a', '#faad14', '#722ed1'];
            return colorList[params.dataIndex % colorList.length];
          }
        }
      }
    ]
  };
  
  chart.setOption(option);
  
  window.addEventListener('resize', () => {
    chart.resize();
  });
};

// 账户资源图表初始化
const initAccountResourceChart = () => {
  if (!accountResourceChart.value) return;
  
  const chart = echarts.init(accountResourceChart.value);
  
  // 模拟账户资源数据
  const resourceData = [
    { name: 'ECS实例', value: 32 },
    { name: 'VPC网络', value: 8 },
    { name: '安全组', value: 12 },
    { name: '弹性IP', value: 18 },
    { name: '负载均衡', value: 5 }
  ];
  
  const option = {
    tooltip: {
      trigger: 'item'
    },
    radar: {
      indicator: resourceData.map(item => ({ name: item.name, max: Math.max(...resourceData.map(d => d.value)) * 1.2 }))
    },
    series: [
      {
        name: '资源分布',
        type: 'radar',
        data: [
          {
            value: resourceData.map(item => item.value),
            name: '资源数量',
            areaStyle: {
              color: 'rgba(0, 112, 192, 0.4)'
            },
            lineStyle: {
              color: 'rgba(0, 112, 192, 0.8)'
            }
          }
        ]
      }
    ]
  };
  
  chart.setOption(option);
  chart.resize();
};

const getProviderName = (provider: string) => {
  const found = providers.value.find(p => p.provider === provider);
  return found ? found.localName : provider;
};

const getProviderColor = (provider: string) => {
  const colorMap: Record<string, string> = {
    aliyun: 'orange',
    tencent: 'blue',
    huawei: 'red',
    aws: 'purple',
    azure: 'cyan',
    gcp: 'green',
    local: 'default',
  };
  return colorMap[provider] || 'default';
};

const refreshData = () => {
  fetchCloudAccounts();
  
  // 更新统计数据
  cloudStats.value = {
    regionDistribution: Math.floor(Math.random() * 5) + 10,
    totalEcsCount: Math.floor(Math.random() * 50) + 100,
    runningEcsCount: Math.floor(Math.random() * 30) + 80,
    stoppedEcsCount: Math.floor(Math.random() * 20) + 20,
    totalVpcCount: Math.floor(Math.random() * 10) + 20,
    totalSecurityGroupCount: Math.floor(Math.random() * 15) + 30,
    totalMonthlyCost: Math.floor(Math.random() * 5000) + 10000,
    updateTime: Date.now(),
  };
  
  // 更新图表
  nextTick(() => {
    initInstanceStatusChart();
    initRegionDistributionChart();
  });
  
  message.success('数据已刷新');
};

const onSearch = (value: string) => {
  searchValue.value = value;
};

const showAddAccountModal = () => {
  isEditing.value = false;
  resetAccountForm();
  accountModalVisible.value = true;
};

const resetAccountForm = () => {
  accountForm.id = 0;
  accountForm.name = '';
  accountForm.provider = '';
  accountForm.accountId = '';
  accountForm.accessKey = '';
  accountForm.secretKey = '';
  accountForm.regions = [];
  accountForm.isEnabled = true;
  accountForm.description = '';
};

const handleProviderChange = (value: string) => {
  accountForm.regions = [];
  fetchRegions(value);
};

const editAccount = (account: CloudAccount) => {
  isEditing.value = true;
  selectedAccount.value = account;
  accountForm.id = account.id;
  accountForm.name = account.name;
  accountForm.provider = account.provider;
  accountForm.accountId = account.accountId;
  accountForm.accessKey = account.accessKey;
  accountForm.secretKey = ''; 
  accountForm.regions = [...account.regions];
  accountForm.isEnabled = account.isEnabled;
  accountForm.description = account.description;

  fetchRegions(account.provider);
  accountModalVisible.value = true;
  detailDrawerVisible.value = false;
};

const handleAccountSubmit = () => {
  accountFormRef.value.validate().then(() => {
    submitLoading.value = true;
    // 模拟提交
    setTimeout(() => {
      if (isEditing.value) {
        // 更新现有账户
        const index = cloudAccounts.value.findIndex(item => item.id === accountForm.id);
        if (index !== -1) {
          cloudAccounts.value[index] = {
            ...cloudAccounts.value[index],
            name: accountForm.name,
            provider: accountForm.provider,
            accountId: accountForm.accountId,
            accessKey: accountForm.accessKey,
            regions: [...accountForm.regions],
            isEnabled: accountForm.isEnabled,
            description: accountForm.description,
            lastSyncTime: cloudAccounts.value[index]?.lastSyncTime || new Date().toLocaleString(),
            id: accountForm.id as number,
          };
        }
        message.success('云账户更新成功');
      } else {
        // 添加新账户
        const newAccount: CloudAccount = {
          id: Date.now(),
          name: accountForm.name,
          provider: accountForm.provider,
          accountId: accountForm.accountId,
          accessKey: accountForm.accessKey,
          regions: [...accountForm.regions],
          isEnabled: accountForm.isEnabled,
          lastSyncTime: new Date().toLocaleString(),
          description: accountForm.description,
          syncing: false,
        };
        cloudAccounts.value.unshift(newAccount);
        message.success('云账户添加成功');
      }
      submitLoading.value = false;
      accountModalVisible.value = false;
    }, 500);
  }).catch((error: any) => {
    console.log('验证失败', error);
  });
};

const viewAccountDetails = (account: CloudAccount) => {
  selectedAccount.value = account;
  detailDrawerVisible.value = true;
};

const syncAccount = (account: CloudAccount) => {
  // 找到对应账户并设置同步状态
  const index = cloudAccounts.value.findIndex(item => item.id === account.id);
  if (index !== -1 && cloudAccounts.value[index]) {
    cloudAccounts.value[index].syncing = true;
  }
  
  message.loading({ content: `正在同步 ${account.name} 的资源数据...`, key: 'sync' });
  
  // 模拟同步操作
  setTimeout(() => {
    if (index !== -1 && cloudAccounts.value[index]) {
      cloudAccounts.value[index].syncing = false;
      cloudAccounts.value[index].lastSyncTime = new Date().toLocaleString();
    }
    
    message.success({ content: `${account.name} 资源数据同步完成`, key: 'sync' });
    
    // 如果详情抽屉打开着，更新资源图表
    if (detailDrawerVisible.value && selectedAccount.value.id === account.id) {
      nextTick(() => {
        initAccountResourceChart();
      });
    }
  }, 1500);
};

const toggleAccountStatus = (account: CloudAccount) => {
  const index = cloudAccounts.value.findIndex(item => item.id === account.id);
  if (index !== -1 && cloudAccounts.value[index]) {
    cloudAccounts.value[index].isEnabled = !cloudAccounts.value[index].isEnabled;
    const isEnabled = cloudAccounts.value[index].isEnabled;
    message.success(`账户 ${account.name} 已${isEnabled ? '启用' : '禁用'}`);
  }
};

const confirmDeleteAccount = (account: CloudAccount) => {
  selectedAccount.value = account;
  deleteModalVisible.value = true;
};

const deleteAccount = () => {
  // 模拟删除操作
  const index = cloudAccounts.value.findIndex(item => item.id === selectedAccount.value.id);
  if (index !== -1) {
    cloudAccounts.value.splice(index, 1);
    message.success(`账户 ${selectedAccount.value.name} 已删除`);
  }
  deleteModalVisible.value = false;
  detailDrawerVisible.value = false;
};
</script>

<style scoped lang="scss">
.cloud-provider-container {
  padding: 0 16px;

  .page-header {
    margin-bottom: 16px;
  }

  .content-layout {
    padding: 16px;
    border-radius: 4px;

    .dashboard-cards {
      margin-bottom: 16px;
    }

    .stats-card {
      height: 100%;
      border-radius: 8px;
      overflow: hidden;
      transition: all 0.3s;

      &:hover {
        transform: translateY(-3px);
      }

      .stats-card-content {
        display: flex;
        justify-content: space-between;
        align-items: center;

        .stats-info {
          flex: 1;

          .stats-title {
            font-size: 14px;
            margin-bottom: 8px;
          }

          .stats-value {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 8px;
          }

          .stats-desc {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            font-size: 12px;

            .stats-highlight {
              display: flex;
              align-items: center;
              gap: 4px;
            }

            .stats-muted {
              display: flex;
              align-items: center;
              gap: 4px;
            }
          }
        }

        .stats-icon {
          font-size: 32px;
          color: #1890ff;
          padding: 0 16px;
          opacity: 0.6;
        }
      }
    }

    .chart-row {
      margin-bottom: 16px;
    }

    .chart-card {
      margin-bottom: 16px;
      height: 360px;
    }

    .chart-container {
      height: 300px;
      width: 100%;
    }

    .account-card {
      margin-bottom: 16px;
    }
  }

  .account-resource-chart {
    height: 300px;
    width: 100%;
    margin-bottom: 75px;
  }

  .drawer-actions {
    position: absolute;
    bottom: 24px;
    width: calc(100% - 48px);
    display: flex;
    justify-content: space-between;
  }
}

@media (max-width: 768px) {
  .cloud-provider-container {
    .content-layout {
      .stats-card {
        margin-bottom: 16px;
      }
      
      .chart-card {
        height: 300px;
      }
      
      .chart-container {
        height: 240px;
      }
    }
    
    .drawer-actions {
      flex-direction: column;
      gap: 8px;
      
      button {
        width: 100%;
      }
    }
  }
}
</style>