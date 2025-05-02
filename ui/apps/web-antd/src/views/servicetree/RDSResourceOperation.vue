<template>
  <div class="rds-management-container">
    <!-- 标题栏 -->
    <div class="header-section">
      <a-row align="middle">
        <a-col :span="12">
          <div class="page-title">云数据库 RDS 管理控制台</div>
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
          <a-button type="primary" @click="fetchRdsList">
            <search-outlined /> 查询
          </a-button>
          <a-button class="reset-btn" @click="resetFilters">
            <reload-outlined /> 重置
          </a-button>
        </a-col>
      </a-row>
    </a-card>

    <!-- 列表区域 -->
    <a-card class="rds-list-card" :bordered="false">
      <a-table :dataSource="rdsList" :columns="columns" :loading="tableLoading" :pagination="pagination"
        @change="handleTableChange" :row-key="(record: any) => record.instance_id" size="middle">
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
                    <a-menu-item v-if="record.status !== 'Running'" key="start" @click="startRds(record)">
                      <play-circle-outlined /> 启动
                    </a-menu-item>
                    <a-menu-item v-if="record.status === 'Running'" key="stop" @click="stopRds(record)">
                      <pause-circle-outlined /> 停止
                    </a-menu-item>
                    <a-menu-item v-if="record.status === 'Running'" key="restart" @click="restartRds(record)">
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
    <a-modal v-model:visible="createModalVisible" title="创建 RDS 实例" width="800px" :footer="null" class="create-modal"
      :destroyOnClose="true">
      <a-steps :current="currentStep" size="small" class="create-steps">
        <a-step title="基础配置" />
        <a-step title="网络配置" />
        <a-step title="数据库配置" />
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

          <a-form-item label="数据库引擎" name="engine" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.engine" placeholder="选择数据库引擎" @change="handleEngineChange"
              :disabled="!createForm.zoneId">
              <a-select-option value="MySQL">MySQL</a-select-option>
              <a-select-option value="PostgreSQL">PostgreSQL</a-select-option>
              <a-select-option value="SQLServer">SQL Server</a-select-option>
              <a-select-option value="MariaDB">MariaDB</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="引擎版本" name="engineVersion" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.engineVersion" placeholder="选择引擎版本" @change="handleEngineVersionChange"
              :disabled="!createForm.engine">
              <a-select-option v-for="version in engineVersionOptions" :key="version" :value="version">
                {{ version }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="实例规格" name="dbInstanceClass" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.dbInstanceClass" placeholder="选择实例规格"
              :disabled="!createForm.engineVersion" show-search :filter-option="filterInstanceClass">
              <a-select-option v-for="spec in instanceClassOptions" :key="spec.value" :value="spec.value">
                {{ spec.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 2: 网络配置 -->
        <div v-if="currentStep === 1">
          <a-form-item label="存储类型" name="storageType" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.storageType" placeholder="选择存储类型">
              <a-select-option value="local_ssd">本地SSD盘</a-select-option>
              <a-select-option value="cloud_ssd">SSD云盘</a-select-option>
              <a-select-option value="cloud_essd">ESSD云盘</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="存储空间" name="storageSize" :rules="[{ required: true }]">
            <a-slider v-model:value="createForm.storageSize" :min="5" :max="2000" :step="5"
              :marks="{ 5: '5GB', 100: '100GB', 500: '500GB', 2000: '2000GB' }" />
          </a-form-item>

          <a-form-item label="VPC" name="vpcId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.vpcId" placeholder="选择VPC" @change="handleVpcChange"
              :loading="vpcLoading">
              <a-select-option v-for="vpc in vpcOptions" :key="vpc.vpcId" :value="vpc.vpcId">
                {{ vpc.vpcName }} ({{ vpc.cidrBlock }})
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="交换机" name="vSwitchId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.vSwitchId" placeholder="选择交换机" :loading="vSwitchLoading"
              :disabled="!createForm.vpcId">
              <a-select-option v-for="vSwitch in vSwitchOptions" :key="vSwitch.vSwitchId" :value="vSwitch.vSwitchId">
                {{ vSwitch.vSwitchName }} ({{ vSwitch.cidrBlock }})
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="网络类型" name="networkType" :rules="[{ required: true }]">
            <a-radio-group v-model:value="createForm.networkType">
              <a-radio value="VPC">专有网络</a-radio>
              <a-radio value="Classic">经典网络</a-radio>
            </a-radio-group>
          </a-form-item>

          <a-form-item label="公网访问" name="publicAccess">
            <a-switch v-model:checked="createForm.publicAccess" />
          </a-form-item>
        </div>

        <!-- 步骤 3: 数据库配置 -->
        <div v-if="currentStep === 2">
          <a-form-item label="实例名称" name="dbInstanceName" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.dbInstanceName" placeholder="实例名称，如rds-mysql-01" />
          </a-form-item>

          <a-form-item label="数据库账号" name="accountName" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.accountName" placeholder="数据库账号名称" />
          </a-form-item>

          <a-form-item label="账号密码" name="accountPassword" :rules="[{ required: true }]">
            <a-input-password v-model:value="createForm.accountPassword" placeholder="请输入账号密码" />
          </a-form-item>

          <a-form-item label="数据库名称" name="dbName">
            <a-input v-model:value="createForm.dbName" placeholder="数据库名称，如mydb" />
          </a-form-item>

          <a-form-item label="字符集" name="characterSetName" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.characterSetName" placeholder="选择字符集">
              <a-select-option value="utf8">UTF-8</a-select-option>
              <a-select-option value="gbk">GBK</a-select-option>
              <a-select-option value="latin1">Latin1</a-select-option>
              <a-select-option value="utf8mb4">UTF-8 MB4</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="实例描述" name="description">
            <a-textarea v-model:value="createForm.description" placeholder="实例描述" :rows="2" />
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
            <a-descriptions-item label="云服务商">{{ getProviderName(createForm.provider || '') }}</a-descriptions-item>
            <a-descriptions-item label="付费类型">{{ getPayTypeName(createForm.payType || '') }}</a-descriptions-item>
            <a-descriptions-item label="地域">{{ createForm.region }}</a-descriptions-item>
            <a-descriptions-item label="可用区">{{ createForm.zoneId }}</a-descriptions-item>
            <a-descriptions-item label="数据库引擎">{{ createForm.engine }} {{ createForm.engineVersion
            }}</a-descriptions-item>
            <a-descriptions-item label="实例规格">{{ createForm.dbInstanceClass }}</a-descriptions-item>
            <a-descriptions-item label="存储类型">{{ getStorageTypeName(createForm.storageType) }}</a-descriptions-item>
            <a-descriptions-item label="存储空间">{{ createForm.storageSize }}GB</a-descriptions-item>
            <a-descriptions-item label="网络类型">{{ createForm.networkType === 'VPC' ? '专有网络' : '经典网络'
            }}</a-descriptions-item>
            <a-descriptions-item label="VPC">
              {{ getVpcById(createForm.vpcId || '')?.vpcName || createForm.vpcId }}
            </a-descriptions-item>
            <a-descriptions-item label="交换机">
              {{ getVSwitchById(createForm.vSwitchId || '')?.vSwitchName || createForm.vSwitchId }}
            </a-descriptions-item>
            <a-descriptions-item label="公网访问">{{ createForm.publicAccess ? '开启' : '关闭' }}</a-descriptions-item>
            <a-descriptions-item label="实例名称">{{ createForm.dbInstanceName }}</a-descriptions-item>
            <a-descriptions-item label="数据库账号">{{ createForm.accountName }}</a-descriptions-item>
            <a-descriptions-item label="数据库名称" v-if="createForm.dbName">{{ createForm.dbName }}</a-descriptions-item>
            <a-descriptions-item label="字符集">{{ createForm.characterSetName }}</a-descriptions-item>
            <a-descriptions-item label="标签" v-if="tagsArray.length > 0">
              <a-tag v-for="(tag, index) in tagsArray" :key="index" color="blue">{{ tag }}</a-tag>
            </a-descriptions-item>
          </a-descriptions>

          <a-alert type="info" showIcon style="margin-top: 20px;">
            <template #message>
              <span>创建 RDS 实例后，实例将立即启动，费用将根据付费类型收取。</span>
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
    <a-drawer v-model:visible="detailDrawerVisible" title="RDS 实例详情" width="600" :destroyOnClose="true"
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
          <a-descriptions-item label="区域">{{ instanceDetail.region_id }}</a-descriptions-item>
          <a-descriptions-item label="可用区">{{ instanceDetail.zone_id }}</a-descriptions-item>
          <a-descriptions-item label="数据库引擎">{{ instanceDetail.engine }} {{ instanceDetail.engine_version
          }}</a-descriptions-item>
          <a-descriptions-item label="实例规格">{{ instanceDetail.db_instance_class }}</a-descriptions-item>
          <a-descriptions-item label="存储空间">{{ instanceDetail.storage_size }}GB</a-descriptions-item>
          <a-descriptions-item label="连接地址">
            <div>
              <div v-if="instanceDetail.connection_string">内网: {{ instanceDetail.connection_string }}</div>
              <div v-if="instanceDetail.public_connection_string">公网: {{ instanceDetail.public_connection_string }}
              </div>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="端口号">{{ instanceDetail.port }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ instanceDetail.creation_time }}</a-descriptions-item>
          <a-descriptions-item label="付费方式">
            {{ getPayTypeName(instanceDetail.instance_charge_type) }}
          </a-descriptions-item>
        </a-descriptions>

        <a-divider orientation="left">数据库信息</a-divider>
        <a-table :dataSource="databases" :columns="databaseColumns" :pagination="false" size="small"
          :row-key="(record: any) => record.dbName"></a-table>

        <a-divider orientation="left">标签</a-divider>
        <div class="tag-list">
          <a-tag v-for="(tag, index) in instanceDetail.tags" :key="index" color="blue">{{ tag }}</a-tag>
          <a-empty v-if="!instanceDetail.tags || instanceDetail.tags.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
            description="暂无标签" />
        </div>

        <div class="drawer-actions">
          <a-button-group>
            <a-button type="primary" :disabled="instanceDetail.status === 'Running'" @click="startRds(instanceDetail)">
              <play-circle-outlined /> 启动
            </a-button>
            <a-button :disabled="instanceDetail.status !== 'Running'" @click="stopRds(instanceDetail)">
              <pause-circle-outlined /> 停止
            </a-button>
            <a-button :disabled="instanceDetail.status !== 'Running'" @click="restartRds(instanceDetail)">
              <reload-outlined /> 重启
            </a-button>
          </a-button-group>
          <a-button type="primary" danger @click="confirmDelete(instanceDetail)">
            <delete-outlined /> 删除实例
          </a-button>
        </div>
      </a-skeleton>
    </a-drawer>

    <!-- 删除确认对话框 -->
    <a-modal v-model:visible="deleteModalVisible" title="删除确认" @ok="deleteRds" :okButtonProps="{ danger: true }"
      okText="删除" cancelText="取消">
      <p>确定要删除实例 "{{ selectedInstance?.instance_name || selectedInstance?.instance_id }}" 吗？此操作不可恢复。</p>
    </a-modal>
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
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  MoreOutlined
} from '@ant-design/icons-vue';
import type { TablePaginationConfig } from 'ant-design-vue';

// 表格列定义
const columns = [
  {
    title: '实例ID',
    dataIndex: 'instance_id',
    key: 'instance_id',
  },
  {
    title: '实例名称',
    dataIndex: 'instance_name',
    key: 'instance_name',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
  },
  {
    title: '区域',
    dataIndex: 'region_id',
    key: 'region_id',
  },
  {
    title: '数据库类型',
    dataIndex: 'engine',
    key: 'engine',
  },
  {
    title: '创建时间',
    dataIndex: 'creation_time',
    key: 'creation_time',
  },
  {
    title: '操作',
    key: 'action',
  },
];

// 数据库表格列定义
const databaseColumns = [
  {
    title: '数据库名称',
    dataIndex: 'dbName',
    key: 'dbName',
  },
  {
    title: '字符集',
    dataIndex: 'characterSetName',
    key: 'characterSetName',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
  },
];

// 状态相关
const getStatusColor = (status: string) => {
  const statusMap: Record<string, string> = {
    'Running': 'green',
    'Creating': 'blue',
    'Stopped': 'orange',
    'Deleting': 'red',
    'Error': 'red',
  };
  return statusMap[status] || 'default';
};

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    'Running': '运行中',
    'Creating': '创建中',
    'Stopped': '已停止',
    'Deleting': '删除中',
    'Error': '错误',
  };
  return statusMap[status] || status;
};

// 付费方式名称
const getPayTypeName = (payType: string) => {
  const payTypeMap: Record<string, string> = {
    'PostPaid': '按量付费',
    'PrePaid': '包年包月',
  };
  return payTypeMap[payType] || payType;
};

// 云服务商名称
const getProviderName = (provider: string) => {
  const providerMap: Record<string, string> = {
    'aliyun': '阿里云',
    'aws': 'AWS',
    'tencent': '腾讯云',
  };
  return providerMap[provider] || provider;
};

// 存储类型名称
const getStorageTypeName = (storageType: string) => {
  const storageTypeMap: Record<string, string> = {
    'local_ssd': '本地SSD盘',
    'cloud_ssd': 'SSD云盘',
    'cloud_essd': 'ESSD云盘',
  };
  return storageTypeMap[storageType] || storageType;
};

// 筛选参数
const filterParams = reactive({
  provider: undefined,
  region: undefined,
});

// 表格数据 - 使用假数据
const rdsList = ref<any[]>([
  {
    instance_id: 'rm-bp1234567890',
    instance_name: 'mysql-prod-01',
    status: 'Running',
    region_id: 'cn-hangzhou',
    zone_id: 'cn-hangzhou-b',
    engine: 'MySQL',
    engine_version: '8.0',
    db_instance_class: 'mysql.x4.large.2c',
    storage_size: 200,
    creation_time: '2023-01-15 08:30:45',
    instance_charge_type: 'PostPaid',
    connection_string: 'rm-bp1234567890.mysql.rds.aliyuncs.com',
    public_connection_string: 'rm-bp1234567890-public.mysql.rds.aliyuncs.com',
    port: 3306,
    tags: ['env=prod', 'project=erp']
  },
  {
    instance_id: 'rm-bp9876543210',
    instance_name: 'mysql-test-01',
    status: 'Stopped',
    region_id: 'cn-beijing',
    zone_id: 'cn-beijing-c',
    engine: 'MySQL',
    engine_version: '5.7',
    db_instance_class: 'mysql.x2.medium.1c',
    storage_size: 100,
    creation_time: '2023-02-20 14:15:30',
    instance_charge_type: 'PrePaid',
    connection_string: 'rm-bp9876543210.mysql.rds.aliyuncs.com',
    port: 3306,
    tags: ['env=test']
  },
  {
    instance_id: 'pgm-bp1357924680',
    instance_name: 'postgres-dev-01',
    status: 'Running',
    region_id: 'cn-shanghai',
    zone_id: 'cn-shanghai-a',
    engine: 'PostgreSQL',
    engine_version: '14.0',
    db_instance_class: 'pg.x4.large.2c',
    storage_size: 150,
    creation_time: '2023-03-10 09:45:20',
    instance_charge_type: 'PostPaid',
    connection_string: 'pgm-bp1357924680.pg.rds.aliyuncs.com',
    public_connection_string: 'pgm-bp1357924680-public.pg.rds.aliyuncs.com',
    port: 5432,
    tags: ['env=dev', 'team=data']
  }
]);

const tableLoading = ref(false);
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 3,
});

// 详情抽屉
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const instanceDetail = ref<any>({});
const databases = ref<any[]>([
  { dbName: 'erp_db', characterSetName: 'utf8mb4', status: 'Running' },
  { dbName: 'crm_db', characterSetName: 'utf8mb4', status: 'Running' },
  { dbName: 'analytics', characterSetName: 'utf8mb4', status: 'Running' }
]);

// 创建实例相关
const createModalVisible = ref(false);
const currentStep = ref(0);
const createLoading = ref(false);
const createFormRef = ref();
const createForm = reactive({
  provider: undefined,
  payType: undefined,
  region: undefined,
  zoneId: undefined,
  engine: undefined,
  engineVersion: undefined,
  dbInstanceClass: undefined,
  storageType: 'cloud_ssd',
  storageSize: 50,
  vpcId: undefined,
  vSwitchId: undefined,
  networkType: 'VPC',
  publicAccess: false,
  dbInstanceName: '',
  accountName: '',
  accountPassword: '',
  dbName: '',
  characterSetName: 'utf8mb4',
  description: '',
});

// 标签相关
const tagsArray = ref<string[]>([]);
const tagInputValue = ref('');

// 删除确认
const deleteModalVisible = ref(false);
const selectedInstance = ref<any>(null);

// 选项数据 - 使用假数据
const regionOptions = ref<any[]>([
  { region: 'cn-hangzhou', name: '华东 1 (杭州)' },
  { region: 'cn-beijing', name: '华北 2 (北京)' },
  { region: 'cn-shanghai', name: '华东 2 (上海)' }
]);

const zoneOptions = ref<any[]>([
  { zone: 'cn-hangzhou-a', name: '杭州 可用区A' },
  { zone: 'cn-hangzhou-b', name: '杭州 可用区B' },
  { zone: 'cn-hangzhou-c', name: '杭州 可用区C' }
]);

const engineVersionOptions = ref<string[]>(['5.7', '8.0']);
const instanceClassOptions = ref<any[]>([
  { value: 'mysql.x2.small.1c', label: '通用型 1核2G' },
  { value: 'mysql.x4.medium.1c', label: '通用型 1核4G' },
  { value: 'mysql.x8.large.2c', label: '通用型 2核8G' },
  { value: 'mysql.x16.xlarge.4c', label: '通用型 4核16G' }
]);

const vpcOptions = ref<any[]>([
  { vpcId: 'vpc-bp1', vpcName: '默认VPC', cidrBlock: '172.16.0.0/16' },
  { vpcId: 'vpc-bp2', vpcName: '生产环境VPC', cidrBlock: '10.0.0.0/16' },
  { vpcId: 'vpc-bp3', vpcName: '测试环境VPC', cidrBlock: '192.168.0.0/16' }
]);

const vSwitchOptions = ref<any[]>([
  { vSwitchId: 'vsw-bp1', vSwitchName: '默认交换机', cidrBlock: '172.16.1.0/24', vpcId: 'vpc-bp1' },
  { vSwitchId: 'vsw-bp2', vSwitchName: '生产交换机A', cidrBlock: '10.0.1.0/24', vpcId: 'vpc-bp2' },
  { vSwitchId: 'vsw-bp3', vSwitchName: '生产交换机B', cidrBlock: '10.0.2.0/24', vpcId: 'vpc-bp2' },
  { vSwitchId: 'vsw-bp4', vSwitchName: '测试交换机', cidrBlock: '192.168.1.0/24', vpcId: 'vpc-bp3' }
]);

const vpcLoading = ref(false);
const vSwitchLoading = ref(false);

// 生命周期钩子
onMounted(() => {
  fetchRdsList();
});

// 方法
const fetchRdsList = () => {
  tableLoading.value = true;
  // 模拟API请求延迟
  setTimeout(() => {
    // 这里使用假数据，实际应该调用API
    tableLoading.value = false;
  }, 500);
};

const resetFilters = () => {
  filterParams.provider = undefined;
  filterParams.region = undefined;
  fetchRdsList();
};

const handleTableChange = (pag: TablePaginationConfig) => {
  pagination.current = pag.current || 1;
  pagination.pageSize = pag.pageSize || 10;
  fetchRdsList();
};

const showCreateModal = () => {
  createModalVisible.value = true;
  currentStep.value = 0;
  // 重置表单
  Object.keys(createForm).forEach(key => {
    // @ts-ignore
    createForm[key] = key === 'storageSize' ? 50 :
      key === 'storageType' ? 'cloud_ssd' :
        key === 'networkType' ? 'VPC' :
          key === 'publicAccess' ? false :
            key === 'characterSetName' ? 'utf8mb4' : undefined;
  });
  tagsArray.value = [];
  tagInputValue.value = '';
};

const nextStep = () => {
  currentStep.value += 1;
};

const prevStep = () => {
  currentStep.value -= 1;
};

const handleCreateSubmit = () => {
  createLoading.value = true;
  // 模拟API请求
  setTimeout(() => {
    createLoading.value = false;
    createModalVisible.value = false;
    message.success('RDS实例创建请求已提交，请稍后刷新查看');
    fetchRdsList();
  }, 1000);
};

const showDetailDrawer = (record: any) => {
  selectedInstance.value = record;
  detailDrawerVisible.value = true;
  detailLoading.value = true;

  // 模拟API请求
  setTimeout(() => {
    instanceDetail.value = record;
    detailLoading.value = false;
  }, 500);
};

const confirmDelete = (record: any) => {
  selectedInstance.value = record;
  deleteModalVisible.value = true;
};

const deleteRds = () => {
  if (!selectedInstance.value) return;

  // 模拟API请求
  message.loading({ content: '正在删除...', key: 'deleteRds' });
  setTimeout(() => {
    message.success({ content: '删除成功', key: 'deleteRds' });
    deleteModalVisible.value = false;

    // 从列表中移除
    rdsList.value = rdsList.value.filter(item => item.instance_id !== selectedInstance.value.instance_id);

    // 如果详情抽屉打开且显示的是被删除的实例，则关闭抽屉
    if (detailDrawerVisible.value && instanceDetail.value.instance_id === selectedInstance.value.instance_id) {
      detailDrawerVisible.value = false;
    }
  }, 1000);
};

const startRds = (record: any) => {
  message.loading({ content: '正在启动实例...', key: 'startRds' });
  setTimeout(() => {
    // 更新状态
    const index = rdsList.value.findIndex(item => item.instance_id === record.instance_id);
    if (index !== -1) {
      rdsList.value[index].status = 'Running';
      if (instanceDetail.value.instance_id === record.instance_id) {
        instanceDetail.value.status = 'Running';
      }
    }
    message.success({ content: '启动成功', key: 'startRds' });
  }, 1000);
};

const stopRds = (record: any) => {
  message.loading({ content: '正在停止实例...', key: 'stopRds' });
  setTimeout(() => {
    // 更新状态
    const index = rdsList.value.findIndex(item => item.instance_id === record.instance_id);
    if (index !== -1) {
      rdsList.value[index].status = 'Stopped';
      if (instanceDetail.value.instance_id === record.instance_id) {
        instanceDetail.value.status = 'Stopped';
      }
    }
    message.success({ content: '停止成功', key: 'stopRds' });
  }, 1000);
};

const restartRds = (record: any) => {
  message.loading({ content: '正在重启实例...', key: 'restartRds' });
  setTimeout(() => {
    message.success({ content: '重启成功', key: 'restartRds' });
  }, 1500);
};

// 表单相关处理方法
const handleProviderChange = () => {
  // 重置相关字段
  createForm.payType = undefined;
  createForm.region = undefined;
  createForm.zoneId = undefined;
  createForm.engine = undefined;
  createForm.engineVersion = undefined;
  createForm.dbInstanceClass = undefined;
};

const handlePayTypeChange = () => {
  // 重置相关字段
  createForm.region = undefined;
  createForm.zoneId = undefined;
  createForm.engine = undefined;
  createForm.engineVersion = undefined;
  createForm.dbInstanceClass = undefined;
};

const handleRegionChange = () => {
  // 重置相关字段并加载可用区
  createForm.zoneId = undefined;
  createForm.engine = undefined;
  createForm.engineVersion = undefined;
  createForm.dbInstanceClass = undefined;

  // 根据选择的region筛选可用区
  if (createForm.region === 'cn-hangzhou') {
    zoneOptions.value = [
      { zone: 'cn-hangzhou-a', name: '杭州 可用区A' },
      { zone: 'cn-hangzhou-b', name: '杭州 可用区B' },
      { zone: 'cn-hangzhou-c', name: '杭州 可用区C' }
    ];
  } else if (createForm.region === 'cn-beijing') {
    zoneOptions.value = [
      { zone: 'cn-beijing-a', name: '北京 可用区A' },
      { zone: 'cn-beijing-b', name: '北京 可用区B' }
    ];
  } else if (createForm.region === 'cn-shanghai') {
    zoneOptions.value = [
      { zone: 'cn-shanghai-a', name: '上海 可用区A' },
      { zone: 'cn-shanghai-b', name: '上海 可用区B' }
    ];
  }
};

const handleZoneChange = () => {
  // 重置相关字段
  createForm.engine = undefined;
  createForm.engineVersion = undefined;
  createForm.dbInstanceClass = undefined;
};

const handleEngineChange = () => {
  // 重置相关字段并加载引擎版本
  createForm.engineVersion = undefined;
  createForm.dbInstanceClass = undefined;

  // 根据选择的引擎加载版本
  if (createForm.engine === 'MySQL') {
    engineVersionOptions.value = ['5.7', '8.0'];
  } else if (createForm.engine === 'PostgreSQL') {
    engineVersionOptions.value = ['12.0', '13.0', '14.0'];
  } else if (createForm.engine === 'SQLServer') {
    engineVersionOptions.value = ['2019', '2017', '2016'];
  } else if (createForm.engine === 'MariaDB') {
    engineVersionOptions.value = ['10.3', '10.4'];
  }
};

const handleEngineVersionChange = () => {
  // 重置实例规格
  createForm.dbInstanceClass = undefined;

  // 根据引擎和版本加载实例规格
  if (createForm.engine === 'MySQL') {
    instanceClassOptions.value = [
      { value: 'mysql.x2.small.1c', label: '通用型 1核2G' },
      { value: 'mysql.x4.medium.1c', label: '通用型 1核4G' },
      { value: 'mysql.x8.large.2c', label: '通用型 2核8G' },
      { value: 'mysql.x16.xlarge.4c', label: '通用型 4核16G' }
    ];
  } else if (createForm.engine === 'PostgreSQL') {
    instanceClassOptions.value = [
      { value: 'pg.x2.small.1c', label: '通用型 1核2G' },
      { value: 'pg.x4.medium.1c', label: '通用型 1核4G' },
      { value: 'pg.x8.large.2c', label: '通用型 2核8G' }
    ];
  } else {
    instanceClassOptions.value = [
      { value: 'x2.small.1c', label: '通用型 1核2G' },
      { value: 'x4.medium.1c', label: '通用型 1核4G' }
    ];
  }
};

const handleVpcChange = () => {
  // 重置交换机并加载对应VPC的交换机
  createForm.vSwitchId = undefined;
  vSwitchLoading.value = true;

  // 模拟加载交换机
  setTimeout(() => {
    // 根据VPC ID筛选交换机
    vSwitchOptions.value = vSwitchOptions.value.filter(item => item.vpcId === createForm.vpcId);
    vSwitchLoading.value = false;
  }, 500);
};

const addTag = () => {
  if (tagInputValue.value && !tagsArray.value.includes(tagInputValue.value)) {
    tagsArray.value.push(tagInputValue.value);
    tagInputValue.value = '';
  }
};

const removeTag = (index: number) => {
  tagsArray.value.splice(index, 1);
};

const filterInstanceClass = (input: string, option: any) => {
  return option.label.toLowerCase().indexOf(input.toLowerCase()) >= 0;
};

const getVpcById = (vpcId: string) => {
  return vpcOptions.value.find(vpc => vpc.vpcId === vpcId);
};

const getVSwitchById = (vSwitchId: string) => {
  return vSwitchOptions.value.find(vSwitch => vSwitch.vSwitchId === vSwitchId);
};
</script>

<style scoped lang="scss">
.rds-management-container {
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

  .rds-list-card {
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
    }

    .confirmation-step {
      margin-bottom: 24px;
    }
  }

  .detail-drawer {
    .drawer-actions {
      margin-top: 24px;
      display: flex;
      justify-content: space-between;
    }

    .tag-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
  }
}
</style>
