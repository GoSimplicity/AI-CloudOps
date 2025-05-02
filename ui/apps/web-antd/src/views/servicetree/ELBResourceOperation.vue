<template>
  <div class="elb-management-container">
    <!-- 标题栏 -->
    <div class="header-section">
      <a-row align="middle">
        <a-col :span="12">
          <div class="page-title">负载均衡 ELB 管理控制台</div>
        </a-col>
        <a-col :span="12" class="header-actions">
          <a-button type="primary" shape="round" @click="showCreateModal">
            <plus-outlined /> 创建负载均衡
          </a-button>
        </a-col>
      </a-row>
    </div>

    <!-- 筛选区域 -->
    <a-card class="filter-card" :bordered="false">
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item label="云服务商">
            <a-select v-model:value="filterParams.provider" placeholder="选择云服务商" allowClear>
              <a-select-option value="aliyun">阿里云</a-select-option>
              <a-select-option value="aws">AWS</a-select-option>
              <a-select-option value="tencent">腾讯云</a-select-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="区域">
            <a-select v-model:value="filterParams.region" placeholder="选择区域" allowClear>
              <a-select-option value="cn-hangzhou">华东 1 (杭州)</a-select-option>
              <a-select-option value="cn-beijing">华北 2 (北京)</a-select-option>
              <a-select-option value="cn-shanghai">华东 2 (上海)</a-select-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="8" class="search-buttons">
          <a-button type="primary" @click="fetchElbList">
            <search-outlined /> 查询
          </a-button>
          <a-button class="reset-btn" @click="resetFilters">
            <reload-outlined /> 重置
          </a-button>
        </a-col>
      </a-row>
    </a-card>

    <!-- 列表区域 -->
    <a-card class="elb-list-card" :bordered="false">
      <a-table :dataSource="elbList" :columns="columns" :loading="tableLoading" :pagination="pagination"
        @change="handleTableChange" :row-key="(record: ResourceElb) => record.load_balancer_id" size="middle">
        <template #bodyCell="{ column, record }">
          <!-- 状态列 -->
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>

          <!-- 操作列 -->
          <template v-if="column.key === 'action'">
            <div class="action-buttons">
              <a-button type="link" size="small" @click="showDetailDrawer(record)" class="detail-btn">
                <eye-outlined /> 详情
              </a-button>
              <a-dropdown>
                <template #overlay>
                  <a-menu>
                    <a-menu-item key="delete" danger @click="confirmDelete(record)">
                      <delete-outlined /> 删除
                    </a-menu-item>
                  </a-menu>
                </template>
                <a-button type="text" size="small">
                  <more-outlined />
                </a-button>
              </a-dropdown>
            </div>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建负载均衡弹窗 -->
    <a-modal v-model:visible="createModalVisible" title="创建负载均衡实例" width="800px" :footer="null" class="create-modal"
      :destroyOnClose="true">
      <a-steps :current="currentStep" size="small" class="create-steps">
        <a-step title="基础配置" />
        <a-step title="网络配置" />
        <a-step title="监听配置" />
        <a-step title="确认信息" />
      </a-steps>

      <a-form :model="createForm" layout="vertical" ref="createFormRef" class="create-form">
        <!-- 步骤 1: 基础配置 -->
        <div v-if="currentStep === 0">
          <a-form-item label="云服务商" name="provider" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.provider" placeholder="选择云服务商" @change="handleProviderChange">
              <a-select-option value="aliyun">阿里云</a-select-option>
              <a-select-option value="aws">AWS</a-select-option>
              <a-select-option value="tencent">腾讯云</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="付费类型" name="payType" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.payType" placeholder="选择付费类型" @change="handlePayTypeChange"
              :disabled="!createForm.provider">
              <a-select-option value="PostPaid">按量付费</a-select-option>
              <a-select-option value="PrePaid">包年包月</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="地域" name="region" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.region" placeholder="选择地域" @change="handleRegionChange"
              :disabled="!createForm.payType">
              <a-select-option v-for="data in regionOptions" :key="data.region" :value="data.region">
                {{ data.region }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="可用区" name="zoneId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.zoneId" placeholder="选择可用区" @change="handleZoneChange"
              :disabled="!createForm.region">
              <a-select-option v-for="zone in zoneOptions" :key="zone.zone" :value="zone.zone">
                {{ zone.zone }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="实例名称" name="loadBalancerName" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.loadBalancerName" placeholder="负载均衡实例名称" />
          </a-form-item>

          <a-form-item label="实例规格" name="loadBalancerSpec" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.loadBalancerSpec" placeholder="选择实例规格" :disabled="!createForm.zoneId">
              <a-select-option value="slb.s1.small">共享型</a-select-option>
              <a-select-option value="slb.s2.small">标准型</a-select-option>
              <a-select-option value="slb.s3.medium">高阶型</a-select-option>
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 2: 网络配置 -->
        <div v-if="currentStep === 1">
          <a-form-item label="网络类型" name="addressType" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.addressType" placeholder="选择网络类型">
              <a-select-option value="internet">公网</a-select-option>
              <a-select-option value="intranet">私网</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="IP版本" name="addressIpVersion" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.addressIpVersion" placeholder="选择IP版本">
              <a-select-option value="ipv4">IPv4</a-select-option>
              <a-select-option value="ipv6">IPv6</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="VPC" name="vpcId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.vpcId" placeholder="选择VPC" @change="handleVpcChange"
              :loading="vpcLoading">
              <a-select-option v-for="vpc in vpcOptions" :key="vpc.vpcId" :value="vpc.vpcId">
                {{ vpc.vpcName }} ({{ vpc.cidrBlock }})
              </a-select-option>
              <a-empty v-if="vpcOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无VPC资源" />
            </a-select>
          </a-form-item>

          <a-form-item label="交换机" name="vSwitchId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.vSwitchId" placeholder="选择交换机" :loading="vSwitchLoading"
              :disabled="!createForm.vpcId">
              <a-select-option v-for="vSwitch in vSwitchOptions" :key="vSwitch.vSwitchId" :value="vSwitch.vSwitchId">
                {{ vSwitch.vSwitchName }} ({{ vSwitch.cidrBlock }})
              </a-select-option>
              <a-empty v-if="vSwitchOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无可用交换机" />
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 3: 监听配置 -->
        <div v-if="currentStep === 2">
          <a-form-item label="监听协议" name="listenerProtocol" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.listenerProtocol" placeholder="选择监听协议">
              <a-select-option value="TCP">TCP</a-select-option>
              <a-select-option value="UDP">UDP</a-select-option>
              <a-select-option value="HTTP">HTTP</a-select-option>
              <a-select-option value="HTTPS">HTTPS</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="监听端口" name="listenerPort" :rules="[{ required: true }]">
            <a-input-number v-model:value="createForm.listenerPort" :min="1" :max="65535" style="width: 100%" />
          </a-form-item>

          <a-form-item label="后端服务器组" name="backendServerGroup" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.backendServerGroup" placeholder="选择后端服务器组">
              <a-select-option value="default">默认服务器组</a-select-option>
              <a-select-option value="vserver">虚拟服务器组</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="健康检查" name="healthCheck" :rules="[{ required: true }]">
            <a-switch v-model:checked="createForm.healthCheck" />
          </a-form-item>

          <a-form-item label="调度算法" name="scheduler" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.scheduler" placeholder="选择调度算法">
              <a-select-option value="wrr">加权轮询</a-select-option>
              <a-select-option value="wlc">加权最小连接数</a-select-option>
              <a-select-option value="rr">轮询</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="标签" name="tags">
            <div class="tag-input-container">
              <div v-for="(tag, index) in tagsArray" :key="index" class="tag-item">
                <a-tag closable @close="removeTag(index)">{{ tag }}</a-tag>
              </div>
              <a-input v-model:value="tagInputValue" placeholder="输入标签，格式为key=value，按回车添加" @pressEnter="addTag"
                style="width: 200px" />
            </div>
          </a-form-item>
        </div>

        <!-- 步骤 4: 确认信息 -->
        <div v-if="currentStep === 3" class="confirmation-step">
          <a-descriptions bordered :column="1" size="small">
            <a-descriptions-item label="云服务商">{{ getProviderName(createForm.provider) }}</a-descriptions-item>
            <a-descriptions-item label="付费类型">{{ getPayTypeName(createForm.payType) }}</a-descriptions-item>
            <a-descriptions-item label="地域">
              {{ getRegionById(createForm.region)?.region || createForm.region }}
            </a-descriptions-item>
            <a-descriptions-item label="可用区">
              {{ getZoneById(createForm.zoneId)?.zone || createForm.zoneId }}
            </a-descriptions-item>
            <a-descriptions-item label="实例名称">{{ createForm.loadBalancerName }}</a-descriptions-item>
            <a-descriptions-item label="实例规格">{{ createForm.loadBalancerSpec }}</a-descriptions-item>
            <a-descriptions-item label="网络类型">{{ getAddressTypeName(createForm.addressType) }}</a-descriptions-item>
            <a-descriptions-item label="IP版本">{{ getAddressIpVersionName(createForm.addressIpVersion)
              }}</a-descriptions-item>
            <a-descriptions-item label="VPC">
              {{ getVpcById(createForm.vpcId)?.vpcName || createForm.vpcId }}
            </a-descriptions-item>
            <a-descriptions-item label="交换机">
              {{ getVSwitchById(createForm.vSwitchId)?.vSwitchName || createForm.vSwitchId }}
            </a-descriptions-item>
            <a-descriptions-item label="监听协议">{{ createForm.listenerProtocol }}</a-descriptions-item>
            <a-descriptions-item label="监听端口">{{ createForm.listenerPort }}</a-descriptions-item>
            <a-descriptions-item label="后端服务器组">{{ getBackendServerGroupName(createForm.backendServerGroup)
              }}</a-descriptions-item>
            <a-descriptions-item label="健康检查">{{ createForm.healthCheck ? '开启' : '关闭' }}</a-descriptions-item>
            <a-descriptions-item label="调度算法">{{ getSchedulerName(createForm.scheduler) }}</a-descriptions-item>
            <a-descriptions-item label="标签" v-if="tagsArray.length > 0">
              <a-tag v-for="(tag, index) in tagsArray" :key="index" color="blue">{{ tag }}</a-tag>
            </a-descriptions-item>
          </a-descriptions>

          <a-alert type="info" showIcon style="margin-top: 20px;">
            <template #message>
              <span>创建负载均衡实例后，实例将立即启动，费用将根据付费类型收取。</span>
            </template>
          </a-alert>
        </div>

        <div class="steps-action">
          <a-button v-if="currentStep > 0" style="margin-right: 8px" @click="prevStep">
            上一步
          </a-button>
          <a-button v-if="currentStep < 3" type="primary" @click="nextStep">
            下一步
          </a-button>
          <a-button v-if="currentStep === 3" type="primary" @click="handleCreateSubmit" :loading="createLoading">
            创建实例
          </a-button>
        </div>
      </a-form>
    </a-modal>

    <!-- 详情抽屉 -->
    <a-drawer v-model:visible="detailDrawerVisible" title="ELB 实例详情" width="600" :destroyOnClose="true"
      class="detail-drawer">
      <a-skeleton :loading="detailLoading" active>
        <a-descriptions bordered :column="1">
          <a-descriptions-item label="实例 ID">{{ elbDetail.load_balancer_id }}</a-descriptions-item>
          <a-descriptions-item label="实例名称">{{ elbDetail.load_balancer_name }}</a-descriptions-item>
          <a-descriptions-item label="实例状态">
            <a-tag :color="getStatusColor(elbDetail.status)">
              {{ getStatusText(elbDetail.status) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="区域">
            {{ getRegionById(elbDetail.region_id)?.region || elbDetail.region_id }}
          </a-descriptions-item>
          <a-descriptions-item label="可用区">{{ elbDetail.zone_id }}</a-descriptions-item>
          <a-descriptions-item label="实例规格">{{ elbDetail.load_balancer_spec }}</a-descriptions-item>
          <a-descriptions-item label="网络类型">{{ getAddressTypeName(elbDetail.address_type) }}</a-descriptions-item>
          <a-descriptions-item label="IP地址">{{ elbDetail.address }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ elbDetail.creation_time }}</a-descriptions-item>
          <a-descriptions-item label="付费方式">
            {{ getPayTypeName(elbDetail.instance_charge_type) }}
          </a-descriptions-item>
        </a-descriptions>

        <a-divider orientation="left">监听器信息</a-divider>
        <a-table :dataSource="listeners" :columns="listenerColumns" :pagination="false" size="small"
          :row-key="(record: Listener) => record.listenerId"></a-table>

        <a-divider orientation="left">标签</a-divider>
        <div class="tag-list">
          <a-tag v-for="(tag, index) in elbDetail.tags" :key="index" color="blue">{{ tag }}</a-tag>
          <a-empty v-if="!elbDetail.tags || elbDetail.tags.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
            description="暂无标签" />
        </div>
      </a-skeleton>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { message, Empty } from 'ant-design-vue';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EyeOutlined,
  DeleteOutlined,
  MoreOutlined
} from '@ant-design/icons-vue';

// 定义类型
interface ResourceElb {
  load_balancer_id: string;
  load_balancer_name: string;
  status: string;
  region_id: string;
  zone_id: string;
  address_type: string;
  address: string;
  load_balancer_spec: string;
  instance_charge_type: string;
  creation_time: string;
  tags?: string[];
}

interface Listener {
  listenerId: string;
  protocol: string;
  port: number;
  status: string;
  scheduler: string;
  healthCheck: boolean;
}

interface RegionOption {
  region: string;
  valid: boolean;
}

interface ZoneOption {
  zone: string;
}

interface VpcOption {
  vpcId: string;
  vpcName: string;
  cidrBlock: string;
}

interface VSwitchOption {
  vSwitchId: string;
  vSwitchName: string;
  cidrBlock: string;
  vpcId: string;
  zoneId: string;
}

// 表格数据
const elbList = ref<ResourceElb[]>([]);
const tableLoading = ref(false);
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
});

// 筛选参数
const filterParams = reactive({
  provider: '',
  region: '',
});

// 表格列定义
const columns = [
  {
    title: '实例ID',
    dataIndex: 'load_balancer_id',
    key: 'load_balancer_id',
    width: 180,
  },
  {
    title: '实例名称',
    dataIndex: 'load_balancer_name',
    key: 'load_balancer_name',
    width: 180,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
  },
  {
    title: '区域',
    dataIndex: 'region_id',
    key: 'region_id',
    width: 120,
  },
  {
    title: '可用区',
    dataIndex: 'zone_id',
    key: 'zone_id',
    width: 120,
  },
  {
    title: '网络类型',
    dataIndex: 'address_type',
    key: 'address_type',
    width: 100,
  },
  {
    title: 'IP地址',
    dataIndex: 'address',
    key: 'address',
    width: 150,
  },
  {
    title: '创建时间',
    dataIndex: 'creation_time',
    key: 'creation_time',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    fixed: 'right',
    width: 120,
  },
];

// 监听器表格列定义
const listenerColumns = [
  {
    title: '监听器ID',
    dataIndex: 'listenerId',
    key: 'listenerId',
  },
  {
    title: '协议',
    dataIndex: 'protocol',
    key: 'protocol',
  },
  {
    title: '端口',
    dataIndex: 'port',
    key: 'port',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
  },
  {
    title: '调度算法',
    dataIndex: 'scheduler',
    key: 'scheduler',
  },
  {
    title: '健康检查',
    dataIndex: 'healthCheck',
    key: 'healthCheck',
    customRender: ({ text }: { text: boolean }) => (text ? '开启' : '关闭'),
  },
];

// 详情抽屉
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const elbDetail = ref<ResourceElb>({} as ResourceElb);
const listeners = ref<Listener[]>([]);

// 创建弹窗
const createModalVisible = ref(false);
const createLoading = ref(false);
const currentStep = ref(0);
const createFormRef = ref();

// 创建表单
const createForm = reactive({
  provider: '',
  payType: '',
  region: '',
  zoneId: '',
  loadBalancerName: '',
  loadBalancerSpec: '',
  addressType: 'internet',
  addressIpVersion: 'ipv4',
  vpcId: '',
  vSwitchId: '',
  listenerProtocol: 'TCP',
  listenerPort: 80,
  backendServerGroup: 'default',
  healthCheck: true,
  scheduler: 'wrr',
  tags: {},
});

// 选项数据
const regionOptions = ref<RegionOption[]>([]);
const zoneOptions = ref<ZoneOption[]>([]);
const vpcOptions = ref<VpcOption[]>([]);
const vSwitchOptions = ref<VSwitchOption[]>([]);
const vpcLoading = ref(false);
const vSwitchLoading = ref(false);

// 标签相关
const tagsArray = ref<string[]>([]);
const tagInputValue = ref('');

// 生命周期钩子
onMounted(() => {
  fetchElbList();
});

// 获取ELB列表
const fetchElbList = async () => {
  tableLoading.value = true;
  try {
    // 模拟API调用
    setTimeout(() => {
      elbList.value = [
        {
          load_balancer_id: 'lb-123456',
          load_balancer_name: '测试负载均衡-1',
          status: 'active',
          region_id: 'cn-beijing',
          zone_id: 'cn-beijing-a',
          address_type: 'internet',
          address: '123.123.123.123',
          load_balancer_spec: 'slb.s2.small',
          instance_charge_type: 'PostPaid',
          creation_time: '2023-01-01 12:00:00',
          tags: ['env=prod', 'app=web'],
        },
        {
          load_balancer_id: 'lb-234567',
          load_balancer_name: '测试负载均衡-2',
          status: 'inactive',
          region_id: 'cn-hangzhou',
          zone_id: 'cn-hangzhou-b',
          address_type: 'intranet',
          address: '10.0.0.1',
          load_balancer_spec: 'slb.s1.small',
          instance_charge_type: 'PrePaid',
          creation_time: '2023-02-01 12:00:00',
          tags: ['env=test'],
        },
      ];
      pagination.total = 2;
      tableLoading.value = false;
    }, 1000);
  } catch (error) {
    message.error('获取负载均衡列表失败');
    tableLoading.value = false;
  }
};

// 表格变化处理
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchElbList();
};

// 处理付费类型变更
const handlePayTypeChange = (value: string) => {
  createForm.payType = value;
};

// 重置筛选条件
const resetFilters = () => {
  Object.keys(filterParams).forEach((key) => {
    filterParams[key as keyof typeof filterParams] = '';
  });
  fetchElbList();
};

// 显示详情抽屉
const showDetailDrawer = (record: ResourceElb) => {
  detailDrawerVisible.value = true;
  detailLoading.value = true;

  // 模拟API调用获取详情
  setTimeout(() => {
    elbDetail.value = record;
    listeners.value = [
      {
        listenerId: 'lsn-123',
        protocol: 'TCP',
        port: 80,
        status: 'running',
        scheduler: 'wrr',
        healthCheck: true,
      },
      {
        listenerId: 'lsn-456',
        protocol: 'HTTP',
        port: 443,
        status: 'running',
        scheduler: 'rr',
        healthCheck: true,
      },
    ];
    detailLoading.value = false;
  }, 1000);
};

// 确认删除
const confirmDelete = (record: ResourceElb) => {
  // 实现删除逻辑
};

// 显示创建弹窗
const showCreateModal = () => {
  createModalVisible.value = true;
  currentStep.value = 0;

  // 重置表单
  Object.keys(createForm).forEach((key) => {
    const formKey = key as keyof typeof createForm;
    if (typeof createForm[formKey] === 'string') {
      (createForm[formKey] as string) = '';
    } else if (typeof createForm[formKey] === 'boolean') {
      (createForm[formKey] as boolean) = true; // 或者根据需要设置为 false
    } else if (typeof createForm[formKey] === 'number') {
      (createForm[formKey] as number) = 0;
    } else if (Array.isArray(createForm[formKey])) {
      (createForm[formKey] as any[]) = [];
    } else if (typeof createForm[formKey] === 'object' && createForm[formKey] !== null) {
      // 对于对象类型，可能需要更复杂的重置逻辑，这里暂时置为空对象
      // 或者根据具体字段进行重置
      if (formKey === 'tags') {
        (createForm[formKey] as Record<string, string>) = {};
      } else {
        // 其他对象类型根据需要处理
        (createForm[formKey] as object) = {};
      }
    }
  });

  createForm.listenerPort = 80;
  createForm.addressType = 'internet';
  createForm.addressIpVersion = 'ipv4';
  createForm.backendServerGroup = 'default';
  createForm.scheduler = 'wrr';

  tagsArray.value = [];
  tagInputValue.value = '';
};

// 步骤控制
const nextStep = () => {
  currentStep.value++;
};

const prevStep = () => {
  currentStep.value--;
};

// 创建提交
const handleCreateSubmit = async () => {
  createLoading.value = true;

  try {
    // 模拟API调用
    setTimeout(() => {
      message.success('负载均衡实例创建成功');
      createModalVisible.value = false;
      fetchElbList();
      createLoading.value = false;
    }, 2000);
  } catch (error) {
    message.error('创建负载均衡实例失败');
    createLoading.value = false;
  }
};

// 处理云服务商变更
const handleProviderChange = (value: string) => {
  // 根据云服务商加载区域选项
  loadRegionOptions(value);
  createForm.region = '';
  createForm.zoneId = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
};

// 加载区域选项
const loadRegionOptions = (provider: string) => {
  // 模拟API调用
  setTimeout(() => {
    if (provider === 'aliyun') {
      regionOptions.value = [
        { region: '华东 1 (杭州)', valid: true },
        { region: '华北 2 (北京)', valid: true },
        { region: '华东 2 (上海)', valid: true },
      ];
    } else if (provider === 'aws') {
      regionOptions.value = [
        { region: '美国东部 (弗吉尼亚)', valid: true },
        { region: '美国西部 (俄勒冈)', valid: true },
        { region: '亚太地区 (东京)', valid: true },
      ];
    } else if (provider === 'tencent') {
      regionOptions.value = [
        { region: '华南地区 (广州)', valid: true },
        { region: '华东地区 (上海)', valid: true },
        { region: '华北地区 (北京)', valid: true },
      ];
    } else {
      regionOptions.value = [];
    }
  }, 500);
};

// 处理区域变更
const handleRegionChange = (value: string) => {
  // 根据区域加载可用区选项
  loadZoneOptions(createForm.provider, value);
  createForm.zoneId = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
};

// 加载可用区选项
const loadZoneOptions = (provider: string, region: string) => {
  // 模拟API调用
  setTimeout(() => {
    if (provider === 'aliyun') {
      if (region.includes('杭州')) {
        zoneOptions.value = [
          { zone: '杭州可用区A' },
          { zone: '杭州可用区B' },
          { zone: '杭州可用区C' },
        ];
      } else if (region.includes('北京')) {
        zoneOptions.value = [
          { zone: '北京可用区A' },
          { zone: '北京可用区B' },
        ];
      } else {
        zoneOptions.value = [
          { zone: '上海可用区A' },
          { zone: '上海可用区B' },
        ];
      }
    } else {
      zoneOptions.value = [
        { zone: '可用区1' },
        { zone: '可用区2' },
      ];
    }
  }, 500);
};

// 处理可用区变更
const handleZoneChange = (value: string) => {
  // 加载VPC选项
  loadVpcOptions(createForm.provider, createForm.region);
  createForm.vpcId = '';
  createForm.vSwitchId = '';
};

// 加载VPC选项
const loadVpcOptions = (provider: string, region: string) => {
  vpcLoading.value = true;
  // 模拟API调用
  setTimeout(() => {
    vpcOptions.value = [
      { vpcId: 'vpc-123', vpcName: 'VPC-测试-1', cidrBlock: '10.0.0.0/16' },
      { vpcId: 'vpc-456', vpcName: 'VPC-测试-2', cidrBlock: '172.16.0.0/16' },
    ];
    vpcLoading.value = false;
  }, 800);
};

// 处理VPC变更
const handleVpcChange = (value: string) => {
  // 加载交换机选项
  loadVSwitchOptions(value, createForm.zoneId);
  createForm.vSwitchId = '';
};

// 加载交换机选项
const loadVSwitchOptions = (vpcId: string, zoneId: string) => {
  vSwitchLoading.value = true;
  // 模拟API调用
  setTimeout(() => {
    vSwitchOptions.value = [
      { vSwitchId: 'vsw-123', vSwitchName: '交换机-1', cidrBlock: '10.0.1.0/24', vpcId: 'vpc-123', zoneId: zoneId },
      { vSwitchId: 'vsw-456', vSwitchName: '交换机-2', cidrBlock: '10.0.2.0/24', vpcId: 'vpc-123', zoneId: zoneId },
    ];
    vSwitchLoading.value = false;
  }, 800);
};

// 添加标签
const addTag = () => {
  if (tagInputValue.value && tagInputValue.value.includes('=')) {
    tagsArray.value.push(tagInputValue.value);
    tagInputValue.value = '';
  } else {
    message.warning('标签格式不正确，请使用key=value格式');
  }
};

// 移除标签
const removeTag = (index: number) => {
  tagsArray.value.splice(index, 1);
};

// 获取状态颜色
const getStatusColor = (status: string): string => {
  switch (status) {
    case 'active':
    case 'running':
      return 'green';
    case 'inactive':
      return 'orange';
    case 'error':
      return 'red';
    default:
      return 'blue';
  }
};

// 获取状态文本
const getStatusText = (status: string): string => {
  switch (status) {
    case 'active':
      return '运行中';
    case 'inactive':
      return '已停止';
    case 'error':
      return '异常';
    default:
      return status;
  }
};

// 获取云服务商名称
const getProviderName = (provider: string): string => {
  switch (provider) {
    case 'aliyun':
      return '阿里云';
    case 'aws':
      return 'AWS';
    case 'tencent':
      return '腾讯云';
    default:
      return provider;
  }
};

// 获取付费类型名称
const getPayTypeName = (payType: string): string => {
  switch (payType) {
    case 'PrePaid':
      return '包年包月';
    case 'PostPaid':
      return '按量付费';
    default:
      return payType;
  }
};

// 获取网络类型名称
const getAddressTypeName = (addressType: string): string => {
  switch (addressType) {
    case 'internet':
      return '公网';
    case 'intranet':
      return '私网';
    default:
      return addressType;
  }
};

// 获取IP版本名称
const getAddressIpVersionName = (ipVersion: string): string => {
  switch (ipVersion) {
    case 'ipv4':
      return 'IPv4';
    case 'ipv6':
      return 'IPv6';
    default:
      return ipVersion;
  }
};

// 获取后端服务器组名称
const getBackendServerGroupName = (group: string): string => {
  switch (group) {
    case 'default':
      return '默认服务器组';
    case 'vserver':
      return '虚拟服务器组';
    default:
      return group;
  }
};

// 获取调度算法名称
const getSchedulerName = (scheduler: string): string => {
  switch (scheduler) {
    case 'wrr':
      return '加权轮询';
    case 'wlc':
      return '加权最小连接数';
    case 'rr':
      return '轮询';
    default:
      return scheduler;
  }
};

// 根据ID获取区域信息
const getRegionById = (regionId: string) => {
  // 模拟数据
  const regionMap: Record<string, RegionOption> = {
    'cn-hangzhou': { region: '华东 1 (杭州)', valid: true },
    'cn-beijing': { region: '华北 2 (北京)', valid: true },
    'cn-shanghai': { region: '华东 2 (上海)', valid: true },
  };
  return regionMap[regionId];
};

// 根据ID获取可用区信息
const getZoneById = (zoneId: string) => {
  // 模拟数据
  const zoneMap: Record<string, ZoneOption> = {
    'cn-hangzhou-a': { zone: '杭州可用区A' },
    'cn-beijing-a': { zone: '北京可用区A' },
    'cn-shanghai-b': { zone: '上海可用区B' },
  };
  return zoneMap[zoneId];
};

// 根据ID获取VPC信息
const getVpcById = (vpcId: string) => {
  // 模拟数据
  const vpcMap: Record<string, VpcOption> = {
    'vpc-123': { vpcId: 'vpc-123', vpcName: 'VPC-测试-1', cidrBlock: '10.0.0.0/16' },
    'vpc-456': { vpcId: 'vpc-456', vpcName: 'VPC-测试-2', cidrBlock: '172.16.0.0/16' },
  };
  return vpcMap[vpcId];
};

// 根据ID获取交换机信息
const getVSwitchById = (vSwitchId: string) => {
  // 模拟数据
  const vSwitchMap: Record<string, VSwitchOption> = {
    'vsw-123': { vSwitchId: 'vsw-123', vSwitchName: '交换机-1', cidrBlock: '10.0.1.0/24', vpcId: 'vpc-123', zoneId: 'cn-hangzhou-a' },
    'vsw-456': { vSwitchId: 'vsw-456', vSwitchName: '交换机-2', cidrBlock: '10.0.2.0/24', vpcId: 'vpc-123', zoneId: 'cn-beijing-a' },
  };
  return vSwitchMap[vSwitchId];
};
</script>

<style scoped lang="scss">
.elb-management-container {
  padding: 20px;

  .header-section {
    margin-bottom: 20px;

    .page-title {
      font-size: 20px;
      font-weight: 500;
    }

    .header-actions {
      text-align: right;
    }
  }

  .filter-card {
    margin-bottom: 20px;

    .search-buttons {
      display: flex;
      align-items: flex-end;

      .reset-btn {
        margin-left: 8px;
      }
    }
  }

  .elb-list-card {
    .action-buttons {
      display: flex;
      align-items: center;

      .detail-btn {
        margin-right: 8px;
      }
    }
  }

  .create-modal {
    .create-steps {
      margin-bottom: 24px;
    }

    .create-form {
      max-height: 400px;
      overflow-y: auto;
      padding: 0 10px;
    }

    .steps-action {
      margin-top: 24px;
      text-align: right;
    }

    .tag-input-container {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      align-items: center;
    }

    .confirmation-step {
      max-height: 400px;
      overflow-y: auto;
    }
  }

  .detail-drawer {
    .tag-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
  }
}
</style>
