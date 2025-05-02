<template>
  <div class="ecs-management-container">
    <!-- 标题栏 -->
    <div class="header-section">
      <a-row align="middle">
        <a-col :span="12">
          <div class="page-title">云服务器 ECS 管理控制台</div>
        </a-col>
        <a-col :span="12" class="header-actions">
          <a-button type="primary" shape="round" @click="showCreateModal">
            <plus-outlined /> 创建实例
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
          <a-button type="primary" @click="fetchEcsList">
            <search-outlined /> 查询
          </a-button>
          <a-button class="reset-btn" @click="resetFilters">
            <reload-outlined /> 重置
          </a-button>
        </a-col>
      </a-row>
    </a-card>

    <!-- 列表区域 -->
    <a-card class="ecs-list-card" :bordered="false">
      <a-table :dataSource="ecsList" :columns="columns" :loading="tableLoading" :pagination="pagination"
        @change="handleTableChange" :row-key="(record: ResourceEcs) => record.instance_id" size="middle">
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
                    <a-menu-item v-if="record.status !== 'Running'" key="start" @click="startEcs(record)">
                      <play-circle-outlined /> 启动
                    </a-menu-item>
                    <a-menu-item v-if="record.status === 'Running'" key="stop" @click="stopEcs(record)">
                      <pause-circle-outlined /> 停止
                    </a-menu-item>
                    <a-menu-item v-if="record.status === 'Running'" key="restart" @click="restartEcs(record)">
                      <reload-outlined /> 重启
                    </a-menu-item>
                    <a-menu-divider />
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

    <!-- 创建实例弹窗 -->
    <a-modal v-model:visible="createModalVisible" title="创建 ECS 实例" width="800px" :footer="null" class="create-modal"
      :destroyOnClose="true">
      <a-steps :current="currentStep" size="small" class="create-steps">
        <a-step title="基础配置" />
        <a-step title="网络配置" />
        <a-step title="系统配置" />
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

          <a-form-item label="实例规格" name="instanceType" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.instanceType" placeholder="选择实例规格" @change="handleInstanceTypeChange"
              :disabled="!createForm.zoneId" show-search :filter-option="filterInstanceType" :options="instanceTypeOptions.map(type => ({
                value: type.instanceType,
                label: `${type.instanceType} (${type.cpu}核${type.memory}GB)`
              }))">
            </a-select>
          </a-form-item>

          <a-form-item label="镜像" name="imageId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.imageId" placeholder="选择镜像" @change="handleImageIdChange"
              :disabled="!createForm.instanceType" show-search :filter-option="filterImage" :options="imageOptions.map(image => ({
                value: image.imageId,
                label: `${image.osName} (${image.osType} - ${image.architecture})`
              }))" :virtual="false" :dropdown-style="{ maxHeight: '400px', overflow: 'auto' }">
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 2: 网络配置 -->
        <div v-if="currentStep === 1">
          <a-form-item label="实例数量" name="amount" :rules="[{ required: true }]">
            <a-input-number v-model:value="createForm.amount" :min="1" :max="100" style="width: 100%" />
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

          <a-form-item label="安全组" name="securityGroupIds" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.securityGroupIds" placeholder="选择安全组" mode="multiple"
              :loading="securityGroupLoading" :disabled="!createForm.vpcId">
              <a-select-option v-for="sg in securityGroupOptions" :key="sg.securityGroupId" :value="sg.securityGroupId">
                {{ sg.securityGroupName }} ({{ sg.description || '无描述' }})
              </a-select-option>
              <a-empty v-if="securityGroupOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
                description="暂无可用安全组" />
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 3: 系统配置 -->
        <div v-if="currentStep === 2">
          <a-form-item label="实例名称" name="instanceName" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.instanceName" placeholder="实例名称，如web-server-01" />
          </a-form-item>

          <a-form-item label="主机名" name="hostname" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.hostname" placeholder="主机名，如cloudops" />
          </a-form-item>

          <a-form-item label="登录密码" name="password" :rules="[{ required: true }]">
            <a-input-password v-model:value="createForm.password" placeholder="请输入登录密码" />
          </a-form-item>

          <a-form-item label="实例描述" name="description">
            <a-textarea v-model:value="createForm.description" placeholder="实例描述" :rows="2" />
          </a-form-item>

          <a-form-item label="系统盘类型" name="systemDiskCategory" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.systemDiskCategory" placeholder="选择系统盘类型"
              @change="handleSystemDiskCategoryChange">
              <a-select-option v-for="disk in systemDiskOptions" :key="disk.systemDiskCategory"
                :value="disk.systemDiskCategory">
                {{ disk.systemDiskCategory }}
              </a-select-option>
              <a-empty v-if="systemDiskOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
                description="暂无可用系统盘类型" />
            </a-select>
          </a-form-item>

          <a-form-item label="系统盘大小 (GB)" name="systemDiskSize" :rules="[{ required: true }]">
            <a-slider v-model:value="createForm.systemDiskSize" :min="20" :max="500" :step="10"
              :marks="{ 20: '20G', 100: '100G', 200: '200G', 500: '500G' }" />
          </a-form-item>

          <a-form-item label="数据盘类型" name="dataDiskCategory">
            <a-select v-model:value="createForm.dataDiskCategory" placeholder="选择数据盘类型"
              @change="handleDataDiskCategoryChange" :disabled="!createForm.systemDiskCategory">
              <a-select-option v-for="disk in dataDiskOptions" :key="disk.dataDiskCategory"
                :value="disk.dataDiskCategory">
                {{ disk.dataDiskCategory }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="数据盘大小 (GB)" name="dataDiskSize">
            <a-slider v-model:value="createForm.dataDiskSize" :min="20" :max="2000" :step="10"
              :marks="{ 20: '20G', 100: '100G', 500: '500G', 2000: '2TB' }" :disabled="!createForm.dataDiskCategory" />
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
            <a-descriptions-item label="实例规格">
              {{ getInstanceTypeById(createForm.instanceType)?.instanceType || createForm.instanceType }}
            </a-descriptions-item>
            <a-descriptions-item label="镜像">{{ createForm.imageId }}</a-descriptions-item>
            <a-descriptions-item label="实例数量">{{ createForm.amount }}</a-descriptions-item>
            <a-descriptions-item label="VPC">
              {{ getVpcById(createForm.vpcId)?.vpcName || createForm.vpcId }}
            </a-descriptions-item>
            <a-descriptions-item label="交换机">
              {{ getVSwitchById(createForm.vSwitchId)?.vSwitchName || createForm.vSwitchId }}
            </a-descriptions-item>
            <a-descriptions-item label="安全组">
              <template v-if="createForm.securityGroupIds && createForm.securityGroupIds.length > 0">
                <a-tag v-for="(sgId, idx) in createForm.securityGroupIds" :key="idx" color="blue">
                  {{ getSecurityGroupById(sgId)?.securityGroupName || sgId }}
                </a-tag>
              </template>
              <template v-else>
                <span>未选择安全组</span>
              </template>
            </a-descriptions-item>
            <a-descriptions-item label="实例名称">{{ createForm.instanceName }}</a-descriptions-item>
            <a-descriptions-item label="系统盘">
              {{ getSystemDiskById(createForm.systemDiskCategory)?.systemDiskCategory ||
                createForm.systemDiskCategory }} {{ createForm.systemDiskSize }}GB
            </a-descriptions-item>
            <a-descriptions-item label="数据盘" v-if="createForm.dataDiskCategory">
              {{ getDataDiskById(createForm.dataDiskCategory)?.dataDiskCategory ||
                createForm.dataDiskCategory }} {{ createForm.dataDiskSize }}GB
            </a-descriptions-item>
            <a-descriptions-item label="标签" v-if="tagsArray.length > 0">
              <a-tag v-for="(tag, index) in tagsArray" :key="index" color="blue">{{ tag }}</a-tag>
            </a-descriptions-item>
          </a-descriptions>

          <a-alert type="info" showIcon style="margin-top: 20px;">
            <template #message>
              <span>创建 ECS 服务器后，服务器将立即启动，实例费用将根据付费类型收取。</span>
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
    <a-drawer v-model:visible="detailDrawerVisible" title="ECS 实例详情" width="600" :destroyOnClose="true"
      class="detail-drawer">
      <a-skeleton :loading="detailLoading" active>
        <a-descriptions bordered :column="1">
          <a-descriptions-item label="实例 ID">{{ instanceDetail.instance_id }}</a-descriptions-item>
          <a-descriptions-item label="实例名称">{{ instanceDetail.instance_name }}</a-descriptions-item>
          <a-descriptions-item label="实例状态">
            <a-tag :color="getStatusColor(instanceDetail.status)">
              {{ getStatusText(instanceDetail.status) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="区域">
            {{ getRegionById(instanceDetail.region_id)?.region || instanceDetail.region_id }}
          </a-descriptions-item>
          <a-descriptions-item label="可用区">{{ instanceDetail.zone_id }}</a-descriptions-item>
          <a-descriptions-item label="实例规格">{{ instanceDetail.instanceType }}</a-descriptions-item>
          <a-descriptions-item label="CPU">{{ instanceDetail.cpu }} 核</a-descriptions-item>
          <a-descriptions-item label="内存">{{ instanceDetail.memory }} GB</a-descriptions-item>
          <a-descriptions-item label="操作系统">{{ instanceDetail.osName }}</a-descriptions-item>
          <a-descriptions-item label="IP 地址">
            <div>
              <div>内网: {{ instanceDetail.private_ip_address?.join(', ') }}</div>
              <div v-if="instanceDetail.public_ip_address && instanceDetail.public_ip_address.length > 0">
                公网: {{ instanceDetail.public_ip_address?.join(', ') }}
              </div>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ instanceDetail.creation_time }}</a-descriptions-item>
          <a-descriptions-item label="付费方式">
            {{ getPayTypeName(instanceDetail.instance_charge_type) }}
          </a-descriptions-item>
        </a-descriptions>

        <a-divider orientation="left">磁盘信息</a-divider>
        <a-table :dataSource="disks" :columns="diskColumns" :pagination="false" size="small"
          :row-key="(record: Disk) => record.diskId"></a-table>

        <a-divider orientation="left">标签</a-divider>
        <div class="tag-list">
          <a-tag v-for="(tag, index) in instanceDetail.tags" :key="index" color="blue">{{ tag }}</a-tag>
          <a-empty v-if="!instanceDetail.tags || instanceDetail.tags.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
            description="暂无标签" />
        </div>

        <div class="drawer-actions">
          <a-button-group>
            <a-button type="primary" :disabled="instanceDetail.status === 'Running'" @click="startEcs(instanceDetail)">
              <play-circle-outlined /> 启动
            </a-button>
            <a-button :disabled="instanceDetail.status !== 'Running'" @click="stopEcs(instanceDetail)">
              <pause-circle-outlined /> 停止
            </a-button>
            <a-button :disabled="instanceDetail.status !== 'Running'" @click="restartEcs(instanceDetail)">
              <reload-outlined /> 重启
            </a-button>
          </a-button-group>
          <a-button danger @click="confirmDelete(instanceDetail)">
            <delete-outlined /> 删除
          </a-button>
        </div>
      </a-skeleton>
    </a-drawer>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, watch } from 'vue';
import { message, Empty, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EyeOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  MoreOutlined,
} from '@ant-design/icons-vue';

import {
  getEcsResourceList,
  getEcsResourceDetail,
  createEcsResource,
  startEcsResource,
  stopEcsResource,
  restartEcsResource,
  deleteEcsResource,
  getInstanceOptions,
  getVpcResourceList,
  listSecurityGroups,
} from '#/api/core/tree';

// 接口定义
interface ResourceEcs {
  instance_id: string;
  instance_name: string;
  cloud_provider: string;
  region_id: string;
  zone_id: string;
  status: string;
  cpu: number;
  memory: number;
  instanceType: string;
  osName: string;
  private_ip_address?: string[];
  public_ip_address?: string[];
  creation_time: string;
  instance_charge_type: string;
  diskIds?: string[];
  tags?: string[];
}

interface Disk {
  diskId: string;
  diskName: string;
  type: string;
  category: string;
  size: number;
}

interface VpcOption {
  vpcId: string;
  vpcName: string;
  cidrBlock: string;
  description?: string;
}

interface VSwitchOption {
  vSwitchId: string;
  vSwitchName: string;
  cidrBlock: string;
  zoneId: string;
  vpcId: string;
}

interface SecurityGroupOption {
  securityGroupId: string;
  securityGroupName: string;
  description?: string;
  vpcId: string;
}

interface ListInstanceOptionsResp {
  region: string;
  zone: string;
  instanceType: string;
  cpu: number;
  memory: number;
  imageId: string;
  osName: string;
  osType: string;
  architecture: string;
  systemDiskCategory: string;
  dataDiskCategory: string;
  payType: string;
  valid: boolean;
}

// 状态和数据定义
const instanceDetail = ref<ResourceEcs>({} as ResourceEcs);
const disks = ref<Disk[]>([]);
const ecsList = ref<ResourceEcs[]>([]);
const tableLoading = ref(false);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const createModalVisible = ref(false);
const createFormRef = ref(null);
const createLoading = ref(false);
const currentStep = ref(0);
const tagsArray = ref<string[]>([]);
const tagInputValue = ref('');
const regionOptions = ref<ListInstanceOptionsResp[]>([]);
const zoneOptions = ref<ListInstanceOptionsResp[]>([]);
const instanceTypeOptions = ref<ListInstanceOptionsResp[]>([]);
const imageOptions = ref<ListInstanceOptionsResp[]>([]);
const systemDiskOptions = ref<ListInstanceOptionsResp[]>([]);
const dataDiskOptions = ref<ListInstanceOptionsResp[]>([]);
const vpcOptions = ref<VpcOption[]>([]);
const vSwitchOptions = ref<VSwitchOption[]>([]);
const securityGroupOptions = ref<SecurityGroupOption[]>([]);
const vpcLoading = ref(false);
const vSwitchLoading = ref(false);
const securityGroupLoading = ref(false);
// 添加一个标志来跟踪步骤之间的切换
const stepChanged = ref(false);

// 表格配置
const columns = [
  {
    title: '实例名称/ID',
    dataIndex: 'instance_name',
    key: 'instance_name',
    render: (text: string, record: ResourceEcs) => {
      return `${text}\n${record.instance_id}`;
    },
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
  },
  {
    title: '实例规格',
    dataIndex: 'instanceType',
    key: 'instanceType',
  },
  {
    title: 'IP地址',
    dataIndex: 'ipAddr',
    key: 'ipAddr',
    render: (text: string, record: ResourceEcs) => {
      let output = `内网: ${record.private_ip_address?.join(', ') || ''}`;
      if (record.public_ip_address && record.public_ip_address.length > 0) {
        output += `\n公网: ${record.public_ip_address.join(', ')}`;
      }
      return output;
    },
  },
  {
    title: '地域/可用区',
    dataIndex: 'region_id',
    key: 'region_id',
    render: (text: string, record: ResourceEcs) => {
      return `${getRegionById(record.region_id)?.region || record.region_id}\n${getZoneById(record.zone_id)?.zone || record.zone_id}`;
    },
  },
  {
    title: '创建时间',
    dataIndex: 'creation_time',
    key: 'creation_time',
  },
  {
    title: '操作',
    key: 'action',
    width: 160,
  },
];

const diskColumns = [
  { title: '磁盘名称', dataIndex: 'diskName', key: 'diskName' },
  { title: '磁盘ID', dataIndex: 'diskId', key: 'diskId' },
  { title: '类型', dataIndex: 'type', key: 'type' },
  { title: '类别', dataIndex: 'category', key: 'category' },
  { title: '大小(GB)', dataIndex: 'size', key: 'size' },
];

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条数据`,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50', '100'],
});

// 筛选参数
const filterParams = reactive({
  provider: 'aliyun',
  region: 'cn-hangzhou',
  pageNumber: 1,
  pageSize: 10,
});

// 创建表单数据
const createForm = reactive({
  provider: 'aliyun',
  region: '',
  imageId: '',
  instanceType: '',
  amount: 1,
  zoneId: '',
  vpcId: '',
  vSwitchId: '',
  securityGroupIds: [] as string[],
  hostname: '',
  password: '',
  instanceName: '',
  payType: '',
  instanceChargeType: '',
  spotStrategy: 'NoSpot',
  description: '',
  systemDiskCategory: '',
  systemDiskSize: 40,
  dataDiskCategory: '',
  dataDiskSize: 100,
  dryRun: false,
  tags: {} as Record<string, string>,
  periodUnit: 'Month',
  period: 1,
  autoRenew: false,
  spotDuration: 1,
});

// 添加表单步骤状态的监听，当从步骤3返回步骤2时，重新获取系统盘类型
watch(currentStep, async (newVal, oldVal) => {
  stepChanged.value = true;

  // 当从第三步返回第二步时，确保系统盘信息不丢失
  if (newVal === 2 && oldVal === 3) {
    if (createForm.imageId && createForm.instanceType && !createForm.systemDiskCategory) {
      await refreshSystemDiskOptions();
    }
  }

  // 当从第一步返回第零步时，确保实例类型和镜像兼容
  if (newVal === 0 && oldVal === 1) {
    if (createForm.imageId && createForm.instanceType) {
      await verifyInstanceTypeAndImageCompatibility();
    }
  }

  stepChanged.value = false;
});

// 新增验证实例类型和镜像兼容性的函数
const verifyInstanceTypeAndImageCompatibility = async () => {
  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: createForm.imageId
    };

    const response = await getInstanceOptions(req);

    // 如果没有返回数据，说明当前实例类型和镜像不兼容
    if (!response || response.length === 0) {
      message.warning('当前选择的实例类型与镜像架构不兼容，请重新选择');
      createForm.imageId = '';

      // 重新加载镜像列表
      await handleInstanceTypeChange(createForm.instanceType);
    }
  } catch (error) {
    console.error('验证实例类型和镜像兼容性失败:', error);
  }
};

// 新增刷新系统盘选项的函数
const refreshSystemDiskOptions = async () => {
  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: createForm.imageId
    };

    const response = await getInstanceOptions(req);
    systemDiskOptions.value = response || [];
  } catch (error) {
    console.error('刷新系统盘选项失败:', error);
    message.error('获取系统盘类型列表失败');
  }
};

// 工具函数
const getStatusColor = (status: string) => {
  const statusMap: Record<string, string> = {
    'Running': 'green',
    'Stopped': 'red',
    'Starting': 'blue',
    'Stopping': 'orange',
    'Creating': 'purple',
  };
  return statusMap[status] || 'gray';
};

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    'Running': '运行中',
    'Stopped': '已停止',
    'Starting': '启动中',
    'Stopping': '停止中',
    'Creating': '创建中',
  };
  return statusMap[status] || status;
};

const getProviderName = (provider: string): string => {
  const map: Record<string, string> = {
    'aliyun': '阿里云',
    'aws': 'AWS',
    'tencent': '腾讯云',
  };
  return map[provider] || provider;
};

const getPayTypeName = (payType: string): string => {
  const map: Record<string, string> = {
    'PostPaid': '按量付费',
    'PrePaid': '包年包月',
  };
  return map[payType] || payType;
};

const getRegionById = (regionId: string) => {
  return regionOptions.value.find(region => region.region === regionId);
};

const getZoneById = (zoneId: string) => {
  return zoneOptions.value.find(zone => zone.zone === zoneId);
};

const getInstanceTypeById = (instanceTypeId: string) => {
  return instanceTypeOptions.value.find(type => type.instanceType === instanceTypeId);
};

const getSystemDiskById = (diskId: string) => {
  return systemDiskOptions.value.find(disk => disk.systemDiskCategory === diskId);
};

const getDataDiskById = (diskId: string) => {
  return dataDiskOptions.value.find(disk => disk.dataDiskCategory === diskId);
};

const getVpcById = (vpcId: string) => {
  return vpcOptions.value.find(vpc => vpc.vpcId === vpcId);
};

const getVSwitchById = (vSwitchId: string) => {
  return vSwitchOptions.value.find(vSwitch => vSwitch.vSwitchId === vSwitchId);
};

const getSecurityGroupById = (securityGroupId: string) => {
  return securityGroupOptions.value.find(sg => sg.securityGroupId === securityGroupId);
};

const filterInstanceType = (input: string, option: any) => {
  const normalizedInput = input.toLowerCase().replace(/\s+/g, '');
  const normalizedLabel = option.label.toLowerCase().replace(/\s+/g, '');
  return normalizedLabel.indexOf(normalizedInput) >= 0;
};

const filterImage = (input: string, option: any) => {
  const normalizedInput = input.toLowerCase().replace(/\s+/g, '');
  const normalizedLabel = option.label.toLowerCase().replace(/\s+/g, '');
  return normalizedLabel.indexOf(normalizedInput) >= 0;
};

// 数据加载函数
onMounted(() => {
  fetchEcsList();
});

const fetchEcsList = async () => {
  tableLoading.value = true;
  try {
    const response = await getEcsResourceList(filterParams);
    ecsList.value = response.data || [];
    pagination.total = response.total || 0;
  } catch (error) {
    message.error('获取ECS实例列表失败');
    console.error('获取ECS实例列表失败:', error);
  } finally {
    tableLoading.value = false;
  }
};

const fetchVpcOptions = async () => {
  if (!createForm.provider || !createForm.region) return;

  vpcLoading.value = true;
  vpcOptions.value = [];
  createForm.vpcId = '';
  createForm.vSwitchId = '';

  try {
    const req = {
      pageNumber: 1,
      pageSize: 10,
      provider: createForm.provider,
      region: createForm.region,
    };

    const response = await getVpcResourceList(req);

    vpcOptions.value = response.data.map((vpc: any) => ({
      vpcId: vpc.instance_id || vpc.vpc_id || vpc.vpcId,
      vpcName: vpc.vpcName || vpc.instance_name || '',
      cidrBlock: vpc.cidrBlock || '',
      description: vpc.description || ''
    }));

    vSwitchLoading.value = true;
    const vSwitches: VSwitchOption[] = [];

    for (const vpc of response.data) {
      if (vpc.vSwitchIds && Array.isArray(vpc.vSwitchIds) && vpc.vSwitchIds.length > 0) {
        for (const vSwitchId of vpc.vSwitchIds) {
          vSwitches.push({
            vSwitchId: vSwitchId,
            vSwitchName: `交换机-${vSwitchId.substring(vSwitchId.length - 8)}`,
            cidrBlock: '未知',
            zoneId: '',
            vpcId: vpc.instance_id || vpc.vpc_id || vpc.vpcId
          });
        }
      }
    }

    vSwitchOptions.value = vSwitches;
  } catch (error) {
    message.error('获取VPC列表失败');
    console.error('获取VPC列表失败:', error);
  } finally {
    vpcLoading.value = false;
    vSwitchLoading.value = false;
  }
};

const fetchSecurityGroupOptions = async () => {
  if (!createForm.provider || !createForm.region) return;

  securityGroupLoading.value = true;
  securityGroupOptions.value = [];
  createForm.securityGroupIds = [];

  try {
    const req = {
      provider: createForm.provider,
      region: createForm.region,
      pageNumber: 1,
      pageSize: 100
    };

    const response = await listSecurityGroups(req);

    securityGroupOptions.value = response.data.map((sg: any) => ({
      securityGroupId: sg.instance_id || sg.security_group_id,
      securityGroupName: sg.securityGroupName || sg.instance_name,
      description: sg.description || '',
      vpcId: sg.vpcId || sg.vpc_id || ''
    }));
  } catch (error) {
    message.error('获取安全组列表失败');
    console.error('获取安全组列表失败:', error);
  } finally {
    securityGroupLoading.value = false;
  }
};

// 事件处理函数
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  filterParams.pageNumber = pag.current;
  filterParams.pageSize = pag.pageSize;
  fetchEcsList();
};

const resetFilters = () => {
  filterParams.provider = 'aliyun';
  filterParams.region = 'cn-hangzhou';
  filterParams.pageNumber = 1;
  filterParams.pageSize = 10;
  pagination.current = 1;
  fetchEcsList();
};

const showDetailDrawer = async (record: ResourceEcs) => {
  detailDrawerVisible.value = true;
  detailLoading.value = true;

  try {
    const req = {
      provider: record.cloud_provider,
      region: record.region_id,
      instanceId: record.instance_id
    };

    const response = await getEcsResourceDetail(req);
    instanceDetail.value = response.data;

    if (instanceDetail.value.diskIds && instanceDetail.value.diskIds.length > 0) {
      disks.value = instanceDetail.value.diskIds.map((diskId, index) => {
        return {
          diskId: diskId,
          diskName: index === 0 ? '系统盘' : `数据盘${index}`,
          type: index === 0 ? 'system' : 'data',
          category: 'cloud_essd',
          size: index === 0 ? 40 : 100
        };
      });
    } else {
      disks.value = [];
    }
  } catch (error) {
    message.error('获取ECS实例详情失败');
    console.error('获取ECS实例详情失败:', error);
  } finally {
    detailLoading.value = false;
  }
};

const startEcs = async (record: ResourceEcs) => {
  const hide = message.loading(`正在启动实例 ${record.instance_name}...`, 0);

  try {
    const req = {
      provider: record.cloud_provider,
      region: record.region_id,
      instanceId: record.instance_id
    };

    await startEcsResource(req);
    message.success(`实例 ${record.instance_name} 正在启动中`);

    record.status = 'Starting';
    if (instanceDetail.value && instanceDetail.value.instance_id === record.instance_id) {
      instanceDetail.value.status = 'Starting';
    }

    setTimeout(() => fetchEcsList(), 2000);
  } catch (error) {
    message.error(`启动实例 ${record.instance_name} 失败`);
    console.error('启动实例失败:', error);
  } finally {
    hide();
  }
};

const stopEcs = async (record: ResourceEcs) => {
  const hide = message.loading(`正在停止实例 ${record.instance_name}...`, 0);

  try {
    const req = {
      provider: record.cloud_provider,
      region: record.region_id,
      instanceId: record.instance_id
    };

    await stopEcsResource(req);
    message.success(`实例 ${record.instance_name} 正在停止中`);

    record.status = 'Stopping';
    if (instanceDetail.value && instanceDetail.value.instance_id === record.instance_id) {
      instanceDetail.value.status = 'Stopping';
    }

    setTimeout(() => fetchEcsList(), 2000);
  } catch (error) {
    message.error(`停止实例 ${record.instance_name} 失败`);
    console.error('停止实例失败:', error);
  } finally {
    hide();
  }
};

const restartEcs = async (record: ResourceEcs) => {
  const hide = message.loading(`正在重启实例 ${record.instance_name}...`, 0);

  try {
    const req = {
      provider: record.cloud_provider,
      region: record.region_id,
      instanceId: record.instance_id
    };

    await restartEcsResource(req);
    message.success(`实例 ${record.instance_name} 正在重启中`);

    record.status = 'Stopping';
    if (instanceDetail.value && instanceDetail.value.instance_id === record.instance_id) {
      instanceDetail.value.status = 'Stopping';
    }

    setTimeout(() => fetchEcsList(), 3000);
  } catch (error) {
    message.error(`重启实例 ${record.instance_name} 失败`);
    console.error('重启实例失败:', error);
  } finally {
    hide();
  }
};

const confirmDelete = (record: ResourceEcs) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除实例 "${record.instance_name}" 吗？此操作不可恢复。`,
    okText: '确认',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      deleteEcs(record);
    },
  });
};

const deleteEcs = async (record: ResourceEcs) => {
  const hide = message.loading(`正在删除实例 ${record.instance_name}...`, 0);

  try {
    const req = {
      provider: record.cloud_provider,
      region: record.region_id,
      instanceId: record.instance_id
    };

    await deleteEcsResource(req);
    message.success(`实例 ${record.instance_name} 已成功删除`);

    if (detailDrawerVisible.value && instanceDetail.value &&
      instanceDetail.value.instance_id === record.instance_id) {
      detailDrawerVisible.value = false;
    }

    fetchEcsList();
  } catch (error) {
    message.error(`删除实例 ${record.instance_name} 失败`);
    console.error('删除实例失败:', error);
  } finally {
    hide();
  }
};

// 创建实例相关函数
const showCreateModal = () => {
  createModalVisible.value = true;
  currentStep.value = 0;

  Object.assign(createForm, {
    provider: 'aliyun',
    region: '',
    imageId: '',
    instanceType: '',
    amount: 1,
    zoneId: '',
    vpcId: '',
    vSwitchId: '',
    securityGroupIds: [],
    hostname: '',
    password: '',
    instanceName: '',
    payType: '',
    instanceChargeType: '',
    spotStrategy: 'NoSpot',
    description: '',
    systemDiskCategory: '',
    systemDiskSize: 40,
    dataDiskCategory: '',
    dataDiskSize: 100,
    dryRun: false,
    tags: {},
    periodUnit: 'Month',
    period: 1,
    autoRenew: false,
    spotDuration: 1,
  });

  tagsArray.value = [];
  tagInputValue.value = '';
};

const nextStep = async () => {
  if (currentStep.value < 3) {
    if (currentStep.value === 0) {
      // 在进入网络配置前，先验证实例类型和镜像是否兼容
      if (createForm.imageId && createForm.instanceType) {
        await verifyInstanceTypeAndImageCompatibility();
      }

      await fetchVpcOptions();
      await fetchSecurityGroupOptions();
    } else if (currentStep.value === 1 && !stepChanged.value) {
      // 在进入系统配置前，确保系统盘类型已加载
      if (createForm.imageId && createForm.instanceType && (!systemDiskOptions.value.length || !createForm.systemDiskCategory)) {
        await refreshSystemDiskOptions();
      }
    }
    currentStep.value += 1;
  }
};

const prevStep = async () => {
  if (currentStep.value > 0) {
    currentStep.value -= 1;

    // 如果从第三步返回第二步，确保系统盘信息不丢失
    if (currentStep.value === 2 && !stepChanged.value) {
      if (createForm.imageId && createForm.instanceType && !createForm.systemDiskCategory) {
        await refreshSystemDiskOptions();
      }
    }

    // 如果从第一步返回第零步，需要确保实例类型和镜像兼容
    if (currentStep.value === 0 && !stepChanged.value) {
      if (createForm.imageId && createForm.instanceType) {
        await verifyInstanceTypeAndImageCompatibility();
      }
    }
  }
};

const addTag = () => {
  if (tagInputValue.value && tagInputValue.value.includes('=')) {
    tagsArray.value.push(tagInputValue.value);

    const parts = tagInputValue.value.split('=');
    if (parts.length === 2 && createForm.tags) {
      const key = parts[0]?.trim();
      const value = parts[1]?.trim();

      if (key && value) {
        createForm.tags[key] = value;
      } else {
        message.warning('标签格式不正确，请确保包含 key=value 格式');
      }
    }

    tagInputValue.value = '';
  } else {
    message.warning('标签格式应为 key=value');
  }
};

const removeTag = (index: number) => {
  if (index >= 0 && index < tagsArray.value.length) {
    const tag = tagsArray.value[index];
    if (tag) {
      const parts = tag.split('=');
      if (parts.length === 2) {
        const key = parts[0]?.trim();
        if (key && createForm.tags && key in createForm.tags) {
          delete createForm.tags[key];
        }
      }

      tagsArray.value.splice(index, 1);
    }
  }
};

const handleCreateSubmit = async () => {
  createLoading.value = true;

  // 再次验证实例类型与镜像的兼容性
  if (createForm.imageId && createForm.instanceType) {
    await verifyInstanceTypeAndImageCompatibility();

    // 确保系统盘类型已设置
    if (!createForm.systemDiskCategory) {
      await refreshSystemDiskOptions();
    }

    // 如果验证后镜像被清空，说明不兼容
    if (!createForm.imageId) {
      message.error('实例类型与镜像架构不兼容，请返回修改');
      createLoading.value = false;
      return;
    }
  }

  createForm.instanceChargeType = createForm.payType;

  try {
    const createParams = {
      provider: createForm.provider,
      periodUnit: createForm.periodUnit,
      period: createForm.period,
      region: createForm.region,
      zoneId: createForm.zoneId,
      autoRenew: createForm.autoRenew,
      instanceChargeType: createForm.instanceChargeType,
      spotStrategy: createForm.spotStrategy,
      spotDuration: createForm.spotDuration,
      systemDiskSize: createForm.systemDiskSize,
      systemDiskCategory: createForm.systemDiskCategory, // 确保包含系统盘类型
      dataDiskSize: createForm.dataDiskSize,
      dataDiskCategory: createForm.dataDiskCategory,
      dryRun: createForm.dryRun,
      tags: createForm.tags,
      imageId: createForm.imageId,
      instanceType: createForm.instanceType,
      amount: createForm.amount || 1,
      vpcId: createForm.vpcId,
      vSwitchId: createForm.vSwitchId,
      securityGroupIds: createForm.securityGroupIds,
      hostname: createForm.hostname,
      password: createForm.password,
      instanceName: createForm.instanceName,
      payType: createForm.payType,
      description: createForm.description
    };

    await createEcsResource(createParams);
    message.success('ECS实例创建成功');
    createModalVisible.value = false;
    setTimeout(() => fetchEcsList(), 5000);
  } catch (error) {
    message.error('创建ECS实例失败');
    console.error('创建ECS实例失败:', error);
  } finally {
    createLoading.value = false;
  }
};

// 表单联动处理函数
const handleProviderChange = async (value: string) => {
  createForm.payType = '';
  createForm.region = '';
  createForm.zoneId = '';
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  regionOptions.value = [];
  zoneOptions.value = [];
  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];
  vpcOptions.value = [];
  vSwitchOptions.value = [];
  securityGroupOptions.value = [];

  try {
    const req = { provider: value };
    const response = await getInstanceOptions(req);
    regionOptions.value = response.data;
  } catch (error) {
    message.error('获取地域列表失败');
  }
};

const handlePayTypeChange = async (value: string) => {
  createForm.region = '';
  createForm.zoneId = '';
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  zoneOptions.value = [];
  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];
  vpcOptions.value = [];
  vSwitchOptions.value = [];
  securityGroupOptions.value = [];

  regionOptions.value = [
    { region: 'cn-beijing', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
    { region: 'cn-hangzhou', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
    { region: 'cn-shanghai', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
    { region: 'cn-shenzhen', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
    { region: 'cn-hongkong', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' }
  ];
};

const handleRegionChange = async (value: string) => {
  createForm.zoneId = '';
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  zoneOptions.value = [];
  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];
  vpcOptions.value = [];
  vSwitchOptions.value = [];
  securityGroupOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: value
    };
    const response = await getInstanceOptions(req);
    zoneOptions.value = response;
  } catch (error) {
    console.error('获取可用区列表失败:', error);
    message.error('获取可用区列表失败');
  }
};

const handleZoneChange = async (value: string) => {
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';

  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: value
    };
    const response = await getInstanceOptions(req);
    instanceTypeOptions.value = response;
  } catch (error) {
    console.error('获取实例规格列表失败:', error);
    message.error('获取实例规格列表失败');
  }
};

const handleInstanceTypeChange = async (value: string) => {
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';

  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: value,
      pageNumber: 1,
      pageSize: 10
    };
    const response = await getInstanceOptions(req);
    imageOptions.value = response || [];

    if (imageOptions.value.length === 0) {
      message.warning('当前配置下没有可用的镜像选项');
    }
  } catch (error) {
    console.error('获取镜像列表失败:', error);
    message.error('获取镜像列表失败');
  }
};

const handleImageIdChange = async (value: string) => {
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';

  systemDiskOptions.value = [];
  dataDiskOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: value
    };
    const response = await getInstanceOptions(req);

    // 如果响应为空，可能是实例类型和镜像不兼容
    if (!response || response.length === 0) {
      message.warning('选择的镜像与实例规格不兼容，请重新选择');
      createForm.imageId = '';
      return;
    }

    systemDiskOptions.value = response || [];
  } catch (error) {
    console.error('获取系统盘类型列表失败:', error);
    message.error('获取系统盘类型列表失败');
  }
};

const handleSystemDiskCategoryChange = async (value: string) => {
  createForm.dataDiskCategory = '';
  dataDiskOptions.value = [];

  createForm.systemDiskCategory = value;

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: createForm.imageId,
      systemDiskCategory: value
    };
    const response = await getInstanceOptions(req);
    dataDiskOptions.value = response || [];
  } catch (error) {
    console.error('获取数据盘类型列表失败:', error);
    message.error('获取数据盘类型列表失败');
  }
};

const handleVpcChange = (vpcId: string) => {
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  const filteredVSwitches = vSwitchOptions.value.filter(vSwitch => vSwitch.vpcId === vpcId);
  const zoneVSwitch = filteredVSwitches.find(vSwitch => vSwitch.zoneId === createForm.zoneId);

  if (zoneVSwitch) {
    createForm.vSwitchId = zoneVSwitch.vSwitchId;
  } else if (filteredVSwitches.length > 0) {
    createForm.vSwitchId = filteredVSwitches[0]?.vSwitchId || '';
  }

  const filteredSecurityGroups = securityGroupOptions.value.filter(sg => sg.vpcId === vpcId);
  if (filteredSecurityGroups.length > 0) {
    createForm.securityGroupIds = [filteredSecurityGroups[0]?.securityGroupId || ''];
  }
};

const handleDataDiskCategoryChange = () => {
  // 数据盘类型选择变更时的处理逻辑
};
</script>

<style scoped lang="scss">
.ecs-management-container {
  padding: 24px;
  height: 100%;
  min-height: 100vh;
}

.header-section {
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: bold;
  margin: 0;
}

.header-actions {
  display: flex;
  justify-content: flex-end;
}

.filter-card {
  margin-bottom: 24px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.search-buttons {
  display: flex;
  justify-content: flex-end;
  align-items: flex-end;
}

.reset-btn {
  margin-left: 8px;
}

.ecs-list-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.action-buttons {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
}

.detail-btn {
  color: var(--ant-primary-color, #1890ff);
}

.create-steps {
  margin-bottom: 24px;
}

.create-form {
  max-height: 500px;
  overflow-y: auto;
  padding: 0 12px;
}

.steps-action {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 20px;
}

.drawer-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 24px;
}

.tag-input-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.tag-item {
  margin-bottom: 4px;
}

:deep(.ant-form-item) {
  margin-bottom: 20px;
}

:deep(.ant-tag) {
  margin-right: 0;
}
</style>
