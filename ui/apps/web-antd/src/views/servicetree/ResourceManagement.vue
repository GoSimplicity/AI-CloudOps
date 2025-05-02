<template>
  <div class="resource-management">
    <a-page-header
      title="资源管理平台"
      sub-title="多云资源统一管理"
      class="page-header"
    >
      <template #extra>
        <a-button type="primary" @click="handleSyncResources">
          <sync-outlined /> 同步资源
        </a-button>
      </template>
    </a-page-header>

    <a-card class="filter-card">
      <a-form layout="inline" :model="filterForm">
        <a-form-item label="云厂商">
          <a-select
            v-model:value="filterForm.provider"
            style="width: 120px"
            placeholder="选择厂商"
            allow-clear
          >
            <a-select-option v-for="provider in cloudProviders" :key="provider.value" :value="provider.value">
              {{ provider.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="地区">
          <a-select
            v-model:value="filterForm.region"
            style="width: 150px"
            placeholder="选择地区"
            allow-clear
          >
            <a-select-option v-for="region in regions" :key="region.value" :value="region.value">
              {{ region.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="资源名称">
          <a-input
            v-model:value="filterForm.name"
            placeholder="输入资源名称"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="状态">
          <a-select
            v-model:value="filterForm.status"
            style="width: 120px"
            placeholder="选择状态"
            allow-clear
          >
            <a-select-option value="running">运行中</a-select-option>
            <a-select-option value="stopped">已停止</a-select-option>
            <a-select-option value="starting">启动中</a-select-option>
            <a-select-option value="stopping">停止中</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch">
            <search-outlined /> 搜索
          </a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">
            重置
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-tabs v-model:activeKey="activeTab" class="resource-tabs">
      <a-tab-pane key="ecs" tab="云服务器 ECS">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('ecs')">
              <plus-outlined /> 创建实例
            </a-button>
          </template>
          <a-table
            :columns="ecsColumns"
            :data-source="ecsData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-dropdown>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item key="detail" @click="handleViewDetail('ecs', record)">
                        <info-circle-outlined /> 详情
                      </a-menu-item>
                      <a-menu-item key="start" @click="handleEcsAction('start', record)" v-if="record.status === 'stopped'">
                        <play-circle-outlined /> 启动
                      </a-menu-item>
                      <a-menu-item key="stop" @click="handleEcsAction('stop', record)" v-if="record.status === 'running'">
                        <pause-circle-outlined /> 停止
                      </a-menu-item>
                      <a-menu-item key="restart" @click="handleEcsAction('restart', record)" v-if="record.status === 'running'">
                        <reload-outlined /> 重启
                      </a-menu-item>
                      <a-menu-item key="delete" @click="handleDeleteResource('ecs', record)">
                        <delete-outlined /> 删除
                      </a-menu-item>
                    </a-menu>
                  </template>
                  <a-button type="link">
                    操作 <down-outlined />
                  </a-button>
                </a-dropdown>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="vpc" tab="专有网络 VPC">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('vpc')">
              <plus-outlined /> 创建VPC
            </a-button>
          </template>
          <a-table
            :columns="vpcColumns"
            :data-source="vpcData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" @click="handleViewDetail('vpc', record)">详情</a-button>
                <a-button type="link" @click="handleDeleteResource('vpc', record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="sg" tab="安全组">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('sg')">
              <plus-outlined /> 创建安全组
            </a-button>
          </template>
          <a-table
            :columns="sgColumns"
            :data-source="sgData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" @click="handleViewDetail('sg', record)">详情</a-button>
                <a-button type="link" @click="handleDeleteResource('sg', record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="elb" tab="负载均衡 ELB">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('elb')">
              <plus-outlined /> 创建负载均衡
            </a-button>
          </template>
          <a-table
            :columns="elbColumns"
            :data-source="elbData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-button type="link" @click="handleViewDetail('elb', record)">详情</a-button>
                <a-button type="link" @click="handleDeleteResource('elb', record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="rds" tab="云数据库 RDS">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('rds')">
              <plus-outlined /> 创建数据库实例
            </a-button>
          </template>
          <a-table
            :columns="rdsColumns"
            :data-source="rdsData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-dropdown>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item key="detail" @click="handleViewDetail('rds', record)">
                        <info-circle-outlined /> 详情
                      </a-menu-item>
                      <a-menu-item key="start" @click="handleRdsAction('start', record)" v-if="record.status === 'stopped'">
                        <play-circle-outlined /> 启动
                      </a-menu-item>
                      <a-menu-item key="stop" @click="handleRdsAction('stop', record)" v-if="record.status === 'running'">
                        <pause-circle-outlined /> 停止
                      </a-menu-item>
                      <a-menu-item key="restart" @click="handleRdsAction('restart', record)" v-if="record.status === 'running'">
                        <reload-outlined /> 重启
                      </a-menu-item>
                      <a-menu-item key="delete" @click="handleDeleteResource('rds', record)">
                        <delete-outlined /> 删除
                      </a-menu-item>
                    </a-menu>
                  </template>
                  <a-button type="link">
                    操作 <down-outlined />
                  </a-button>
                </a-dropdown>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- ECS创建对话框 -->
    <a-modal
      v-model:visible="modals.ecs"
      title="创建云服务器实例"
      width="700px"
      @ok="handleCreateResource('ecs')"
      :confirmLoading="modalLoading"
    >
      <a-form :model="ecsForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="实例名称" name="instanceName" :rules="[{ required: true, message: '请输入实例名称' }]">
              <a-input v-model:value="ecsForm.instanceName" placeholder="请输入实例名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="云厂商" name="provider" :rules="[{ required: true, message: '请选择云厂商' }]">
              <a-select v-model:value="ecsForm.provider" placeholder="请选择云厂商">
                <a-select-option v-for="provider in cloudProviders" :key="provider.value" :value="provider.value">
                  {{ provider.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="地区" name="region" :rules="[{ required: true, message: '请选择地区' }]">
              <a-select v-model:value="ecsForm.region" placeholder="请选择地区">
                <a-select-option v-for="region in regions" :key="region.value" :value="region.value">
                  {{ region.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="可用区" name="zone" :rules="[{ required: true, message: '请选择可用区' }]">
              <a-select v-model:value="ecsForm.zone" placeholder="请选择可用区">
                <a-select-option v-for="zone in zones" :key="zone.value" :value="zone.value">
                  {{ zone.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="实例类型" name="instanceType" :rules="[{ required: true, message: '请选择实例类型' }]">
              <a-select v-model:value="ecsForm.instanceType" placeholder="请选择实例类型">
                <a-select-option v-for="type in instanceTypes" :key="type.value" :value="type.value">
                  {{ type.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="镜像" name="imageId" :rules="[{ required: true, message: '请选择镜像' }]">
              <a-select v-model:value="ecsForm.imageId" placeholder="请选择镜像">
                <a-select-option v-for="image in images" :key="image.value" :value="image.value">
                  {{ image.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="VPC" name="vpcId" :rules="[{ required: true, message: '请选择VPC' }]">
              <a-select v-model:value="ecsForm.vpcId" placeholder="请选择VPC">
                <a-select-option v-for="vpc in vpcs" :key="vpc.value" :value="vpc.value">
                  {{ vpc.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="安全组" name="securityGroupId" :rules="[{ required: true, message: '请选择安全组' }]">
              <a-select 
                v-model:value="ecsForm.securityGroupIds" 
                mode="multiple" 
                placeholder="请选择安全组"
              >
                <a-select-option v-for="sg in securityGroups" :key="sg.value" :value="sg.value">
                  {{ sg.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="付费类型" name="instanceChargeType" :rules="[{ required: true, message: '请选择付费类型' }]">
              <a-radio-group v-model:value="ecsForm.instanceChargeType">
                <a-radio value="PostPaid">按量付费</a-radio>
                <a-radio value="PrePaid">包年包月</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="密码设置" name="password" :rules="[{ required: true, message: '请输入密码' }]">
              <a-input-password v-model:value="ecsForm.password" placeholder="请输入密码" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="ecsForm.description" placeholder="请输入描述信息" :rows="2" />
        </a-form-item>
        
        <a-form-item label="标签" name="tags">
          <a-select 
            v-model:value="ecsForm.tags" 
            mode="tags" 
            placeholder="输入标签后按Enter确认"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 资源详情对话框 -->
    <a-modal
      v-model:visible="detailVisible"
      :title="`${resourceDetailTitle}详情`"
      width="800px"
      footer={null}
    >
      <a-descriptions bordered :column="2">
        <template v-if="resourceType === 'ecs'">
          <a-descriptions-item label="实例名称" span="2">{{ resourceDetail?.instanceName }}</a-descriptions-item>
          <a-descriptions-item label="实例ID">{{ resourceDetail?.instanceId }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="getStatusColor(resourceDetail?.status)">
              {{ getStatusText(resourceDetail?.status) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="云厂商">{{ getProviderName(resourceDetail?.provider) }}</a-descriptions-item>
          <a-descriptions-item label="地区/可用区">{{ resourceDetail?.regionId }}/{{ resourceDetail?.zoneId }}</a-descriptions-item>
          <a-descriptions-item label="实例类型">{{ resourceDetail?.instanceType }}</a-descriptions-item>
          <a-descriptions-item label="配置">{{ resourceDetail?.cpu }}核 {{ resourceDetail?.memory }}GB</a-descriptions-item>
          <a-descriptions-item label="私有IP" span="2">
            <template v-for="(ip, index) in resourceDetail?.privateIpAddress" :key="index">
              <a-tag>{{ ip }}</a-tag>
            </template>
          </a-descriptions-item>
          <a-descriptions-item label="公网IP" span="2">
            <template v-for="(ip, index) in resourceDetail?.publicIpAddress" :key="index">
              <a-tag>{{ ip }}</a-tag>
            </template>
          </a-descriptions-item>
          <a-descriptions-item label="VPC ID">{{ resourceDetail?.vpcId }}</a-descriptions-item>
          <a-descriptions-item label="安全组">
            <template v-for="(sg, index) in resourceDetail?.securityGroupIds" :key="index">
              <a-tag>{{ sg }}</a-tag>
            </template>
          </a-descriptions-item>
          <a-descriptions-item label="创建时间" span="2">{{ resourceDetail?.creationTime }}</a-descriptions-item>
        </template>
        
        <template v-else-if="resourceType === 'vpc'">
          <a-descriptions-item label="VPC名称" span="2">{{ resourceDetail?.instanceName }}</a-descriptions-item>
          <a-descriptions-item label="VPC ID">{{ resourceDetail?.instanceId }}</a-descriptions-item>
          <a-descriptions-item label="状态">{{ resourceDetail?.status }}</a-descriptions-item>
          <a-descriptions-item label="云厂商">{{ getProviderName(resourceDetail?.provider) }}</a-descriptions-item>
          <a-descriptions-item label="地区">{{ resourceDetail?.regionId }}</a-descriptions-item>
          <a-descriptions-item label="CIDR块" span="2">{{ resourceDetail?.cidrBlock }}</a-descriptions-item>
          <a-descriptions-item label="创建时间" span="2">{{ resourceDetail?.creationTime }}</a-descriptions-item>
        </template>
        
        <!-- 其他资源类型的详情可以根据需要添加 -->
        
        <a-descriptions-item label="标签" span="2">
          <template v-for="(tag, index) in resourceDetail?.tags" :key="index">
            <a-tag>{{ tag }}</a-tag>
          </template>
        </a-descriptions-item>
        <a-descriptions-item label="描述" span="2">{{ resourceDetail?.description }}</a-descriptions-item>
        <a-descriptions-item label="最后同步时间" span="2">{{ resourceDetail?.lastSyncTime }}</a-descriptions-item>
      </a-descriptions>
      
      <div style="margin-top: 24px; text-align: right;">
        <a-button @click="detailVisible = false">关闭</a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SyncOutlined,
  PlusOutlined,
  SearchOutlined,
  InfoCircleOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined,
  DeleteOutlined,
  DownOutlined
} from '@ant-design/icons-vue';

// 云厂商列表
const cloudProviders = [
  { label: '阿里云', value: 'ALIYUN' },
  { label: '腾讯云', value: 'TENCENT' },
  { label: '华为云', value: 'HUAWEI' },
  { label: 'AWS', value: 'AWS' }
];

// 区域列表
const regions = [
  { label: '华北1（青岛）', value: 'cn-qingdao' },
  { label: '华北2（北京）', value: 'cn-beijing' },
  { label: '华东1（杭州）', value: 'cn-hangzhou' },
  { label: '华东2（上海）', value: 'cn-shanghai' },
  { label: '华南1（深圳）', value: 'cn-shenzhen' }
];

// 可用区列表
const zones = [
  { label: '华北2可用区A', value: 'cn-beijing-a' },
  { label: '华北2可用区B', value: 'cn-beijing-b' },
  { label: '华北2可用区C', value: 'cn-beijing-c' },
  { label: '华东1可用区A', value: 'cn-hangzhou-a' },
  { label: '华东1可用区B', value: 'cn-hangzhou-b' }
];

// 实例类型列表
const instanceTypes = [
  { label: 'ecs.g6.large (2核8GB)', value: 'ecs.g6.large' },
  { label: 'ecs.g6.xlarge (4核16GB)', value: 'ecs.g6.xlarge' },
  { label: 'ecs.g6.2xlarge (8核32GB)', value: 'ecs.g6.2xlarge' },
  { label: 'ecs.c6.large (2核4GB)', value: 'ecs.c6.large' },
  { label: 'ecs.c6.xlarge (4核8GB)', value: 'ecs.c6.xlarge' }
];

// 镜像列表
const images = [
  { label: 'CentOS 7.9 64位', value: 'centos_7_9_x64' },
  { label: 'Ubuntu 20.04 64位', value: 'ubuntu_20_04_x64' },
  { label: 'Debian 10.9 64位', value: 'debian_10_9_x64' },
  { label: 'Windows Server 2019', value: 'win_server_2019' }
];

// VPC列表
const vpcs = [
  { label: 'vpc-default (172.16.0.0/16)', value: 'vpc-default' },
  { label: 'vpc-prod (10.0.0.0/16)', value: 'vpc-prod' },
  { label: 'vpc-test (192.168.0.0/16)', value: 'vpc-test' }
];

// 安全组列表
const securityGroups = [
  { label: 'sg-default (默认安全组)', value: 'sg-default' },
  { label: 'sg-web (Web服务安全组)', value: 'sg-web' },
  { label: 'sg-db (数据库安全组)', value: 'sg-db' }
];

// 活动标签页
const activeTab = ref('ecs');

// 加载状态
const loading = ref(false);

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

// 过滤条件
const filterForm = reactive({
  provider: undefined,
  region: undefined,
  name: '',
  status: undefined
});

// 模态框状态
const modals = reactive({
  ecs: false,
  vpc: false,
  sg: false,
  elb: false,
  rds: false
});

// 模态框加载状态
const modalLoading = ref(false);

// 详情模态框
const detailVisible = ref(false);
const resourceType = ref('');
const resourceDetail = ref<Record<string, any>>({});
const resourceDetailTitle = computed(() => {
  const typeMap: Record<string, string> = {
    'ecs': '云服务器',
    'vpc': '专有网络',
    'sg': '安全组',
    'elb': '负载均衡',
    'rds': '云数据库'
  };
  return typeMap[resourceType.value] || '资源';
});

// ECS表单数据
const ecsForm = reactive({
  instanceName: '',
  provider: 'ALIYUN',
  region: 'cn-hangzhou',
  zone: 'cn-hangzhou-a',
  instanceType: 'ecs.g6.large',
  imageId: 'centos_7_9_x64',
  vpcId: 'vpc-default',
  securityGroupIds: ['sg-default'],
  instanceChargeType: 'PostPaid',
  password: '',
  description: '',
  tags: []
});

// ECS表格列定义
const ecsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '实例ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
  { title: 'IP地址', dataIndex: 'primaryIp', key: 'primaryIp' },
  { title: '地区/可用区', dataIndex: 'regionAndZone', key: 'regionAndZone' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// VPC表格列定义
const vpcColumns = [
  { title: 'VPC名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: 'VPC ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: 'CIDR块', dataIndex: 'cidrBlock', key: 'cidrBlock' },
  { title: '地区', dataIndex: 'regionId', key: 'regionId' },
  { title: '云厂商', dataIndex: 'provider', key: 'provider' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 安全组表格列定义
const sgColumns = [
  { title: '安全组名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '安全组ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '云厂商', dataIndex: 'provider', key: 'provider' },
  { title: '地区', dataIndex: 'regionId', key: 'regionId' },
  { title: 'VPC ID', dataIndex: 'vpcId', key: 'vpcId' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// ELB表格列定义
const elbColumns = [
  { title: '负载均衡名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '负载均衡ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '地址类型', dataIndex: 'addressType', key: 'addressType' },
  { title: 'IP地址', dataIndex: 'address', key: 'address' },
  { title: '地区', dataIndex: 'regionId', key: 'regionId' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// RDS表格列定义
const rdsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '实例ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '数据库类型', dataIndex: 'dbType', key: 'dbType' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
  { title: '地区/可用区', dataIndex: 'regionAndZone', key: 'regionAndZone' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 模拟数据 - ECS
const ecsData = ref([
  {
    id: 1,
    instanceName: 'web-server-prod-01',
    instanceId: 'i-2ze4ljs82kismj45cxxx',
    status: 'running',
    instanceType: 'ecs.g6.large',
    primaryIp: '172.16.1.10',
    regionAndZone: '华东1(杭州)/可用区A',
    regionId: 'cn-hangzhou',
    zoneId: 'cn-hangzhou-a',
    cpu: 2,
    memory: 8,
    provider: 'ALIYUN',
    vpcId: 'vpc-default',
    securityGroupIds: ['sg-default', 'sg-web'],
    privateIpAddress: ['172.16.1.10'],
    publicIpAddress: ['47.98.123.456'],
    creationTime: '2023-10-01 12:34:56',
    description: '生产环境Web服务器',
    tags: ['env:prod', 'app:web'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 2,
    instanceName: 'app-server-prod-01',
    instanceId: 'i-2ze4ljs82kismj45cyyy',
    status: 'running',
    instanceType: 'ecs.g6.xlarge',
    primaryIp: '172.16.1.20',
    regionAndZone: '华东1(杭州)/可用区B',
    regionId: 'cn-hangzhou',
    zoneId: 'cn-hangzhou-b',
    cpu: 4,
    memory: 16,
    provider: 'ALIYUN',
    vpcId: 'vpc-default',
    securityGroupIds: ['sg-default'],
    privateIpAddress: ['172.16.1.20'],
    publicIpAddress: [],
    creationTime: '2023-10-02 15:24:36',
    description: '生产环境应用服务器',
    tags: ['env:prod', 'app:backend'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 3,
    instanceName: 'db-proxy-prod-01',
    instanceId: 'i-2ze4ljs82kismj45czzz',
    status: 'stopped',
    instanceType: 'ecs.c6.large',
    primaryIp: '172.16.1.30',
    regionAndZone: '华东1(杭州)/可用区B',
    regionId: 'cn-hangzhou',
    zoneId: 'cn-hangzhou-b',
    cpu: 2,
    memory: 4,
    provider: 'ALIYUN',
    vpcId: 'vpc-default',
    securityGroupIds: ['sg-default', 'sg-db'],
    privateIpAddress: ['172.16.1.30'],
    publicIpAddress: [],
    creationTime: '2023-10-03 08:12:45',
    description: '生产环境数据库代理',
    tags: ['env:prod', 'app:db-proxy'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 模拟数据 - VPC
const vpcData = ref([
  {
    id: 1,
    instanceName: 'vpc-default',
    instanceId: 'vpc-2zeisljxz9bmxhj2qyyy',
    cidrBlock: '172.16.0.0/16',
    regionId: 'cn-hangzhou',
    provider: 'ALIYUN',
    status: 'Available',
    creationTime: '2023-09-01 09:00:00',
    description: '默认VPC',
    tags: ['default'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 2,
    instanceName: 'vpc-prod',
    instanceId: 'vpc-2zeisljxz9bmxhj2qzzz',
    cidrBlock: '10.0.0.0/16',
    regionId: 'cn-hangzhou',
    provider: 'ALIYUN',
    status: 'Available',
    creationTime: '2023-09-05 14:30:00',
    description: '生产环境VPC',
    tags: ['env:prod'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 模拟数据 - 安全组
const sgData = ref([
  {
    id: 1,
    instanceName: 'sg-default',
    instanceId: 'sg-2ze8mmbpj96wr4i8xxxx',
    provider: 'ALIYUN',
    regionId: 'cn-hangzhou',
    vpcId: 'vpc-default',
    creationTime: '2023-09-01 09:10:00',
    description: '默认安全组',
    tags: ['default'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 2,
    instanceName: 'sg-web',
    instanceId: 'sg-2ze8mmbpj96wr4i8yyyy',
    provider: 'ALIYUN',
    regionId: 'cn-hangzhou',
    vpcId: 'vpc-default',
    creationTime: '2023-09-10 11:20:00',
    description: 'Web服务安全组',
    tags: ['service:web'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 3,
    instanceName: 'sg-db',
    instanceId: 'sg-2ze8mmbpj96wr4i8zzzz',
    provider: 'ALIYUN',
    regionId: 'cn-hangzhou',
    vpcId: 'vpc-default',
    creationTime: '2023-09-10 11:25:00',
    description: '数据库安全组',
    tags: ['service:db'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 模拟数据 - ELB
const elbData = ref([
  {
    id: 1,
    instanceName: 'web-lb-prod',
    instanceId: 'lb-2zejplm93vgl58s1xxxx',
    status: 'running',
    addressType: '公网',
    address: '47.98.234.567',
    regionId: 'cn-hangzhou',
    provider: 'ALIYUN',
    creationTime: '2023-10-05 16:40:00',
    description: '生产环境Web负载均衡器',
    tags: ['env:prod', 'service:web'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 2,
    instanceName: 'api-lb-prod',
    instanceId: 'lb-2zejplm93vgl58s1yyyy',
    status: 'running',
    addressType: '内网',
    address: '172.16.5.10',
    regionId: 'cn-hangzhou',
    provider: 'ALIYUN',
    creationTime: '2023-10-05 16:50:00',
    description: '生产环境API负载均衡器',
    tags: ['env:prod', 'service:api'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 模拟数据 - RDS
const rdsData = ref([
  {
    id: 1,
    instanceName: 'mysql-prod-master',
    instanceId: 'rm-2ze3o57f291q7xxxx',
    status: 'running',
    dbType: 'MySQL 5.7',
    instanceType: 'rds.mysql.s3.large',
    regionAndZone: '华东1(杭州)/可用区B',
    regionId: 'cn-hangzhou',
    zoneId: 'cn-hangzhou-b',
    provider: 'ALIYUN',
    vpcId: 'vpc-default',
    creationTime: '2023-10-10 10:00:00',
    description: '生产环境MySQL主库',
    tags: ['env:prod', 'db:mysql', 'role:master'],
    lastSyncTime: '2025-04-30 10:00:00'
  },
  {
    id: 2,
    instanceName: 'mysql-prod-slave',
    instanceId: 'rm-2ze3o57f291q7yyyy',
    status: 'running',
    dbType: 'MySQL 5.7',
    instanceType: 'rds.mysql.s3.large',
    regionAndZone: '华东1(杭州)/可用区C',
    regionId: 'cn-hangzhou',
    zoneId: 'cn-hangzhou-c',
    provider: 'ALIYUN',
    vpcId: 'vpc-default',
    creationTime: '2023-10-10 10:30:00',
    description: '生产环境MySQL从库',
    tags: ['env:prod', 'db:mysql', 'role:slave'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 获取状态颜色
const getStatusColor = (status: string) => {
  const statusColorMap: Record<string, string> = {
    'running': 'green',
    'stopped': 'red',
    'starting': 'blue',
    'stopping': 'orange',
    'creating': 'blue',
    'deleting': 'orange',
    'Available': 'green'
  };
  return statusColorMap[status] || 'default';
};

// 获取状态文本
const getStatusText = (status: string) => {
  const statusTextMap: Record<string, string> = {
    'running': '运行中',
    'stopped': '已停止',
    'starting': '启动中',
    'stopping': '停止中',
    'creating': '创建中',
    'deleting': '删除中',
    'Available': '可用'
  };
  return statusTextMap[status] || status;
};

// 获取云厂商名称
const getProviderName = (provider: string) => {
  const providerMap: Record<string, string> = {
    'ALIYUN': '阿里云',
    'TENCENT': '腾讯云',
    'HUAWEI': '华为云',
    'AWS': 'AWS'
  };
  return providerMap[provider] || provider;
};

// 组件挂载时执行
onMounted(() => {
  // 初始化页面数据
  // 实际项目中可能需要根据用户权限和设置加载不同的数据
  pagination.total = ecsData.value.length;
});

// 处理表格变化
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  // 实际应用中这里应该发起请求获取新的分页数据
};

// 处理搜索
const handleSearch = () => {
  loading.value = true;
  
  // 这里应该根据 filterForm 中的条件向后端发起请求
  // 模拟搜索请求
  setTimeout(() => {
    loading.value = false;
    message.success('搜索完成');
  }, 500);
};

// 重置过滤条件
const resetFilter = () => {
  Object.keys(filterForm).forEach(key => {
    (filterForm as Record<string, any>)[key] = undefined;
  });
  filterForm.name = '';
};

// 同步资源
const handleSyncResources = () => {
  if (!filterForm.provider || !filterForm.region) {
    message.warning('请选择需要同步的云厂商和地区');
    return;
  }
  
  loading.value = true;
  message.loading('正在同步资源，请稍候...', 2);
  
  // 模拟同步请求
  setTimeout(() => {
    loading.value = false;
    message.success('资源同步成功');
  }, 2000);
};

// 显示创建模态框
const showCreateModal = (type: string) => {
  (modals as Record<string, boolean>)[type] = true;
};

// 处理创建资源
const handleCreateResource = (type: string) => {
  modalLoading.value = true;
  
  // 模拟创建请求
  setTimeout(() => {
    modalLoading.value = false;
    (modals as Record<string, boolean>)[type] = false;
    message.success(`${type === 'ecs' ? '云服务器' : type === 'vpc' ? 'VPC' : type === 'sg' ? '安全组' : type === 'elb' ? '负载均衡' : '数据库'}创建请求已提交`);
    
    // 重置表单
    if (type === 'ecs') {
      Object.assign(ecsForm, {
        instanceName: '',
        provider: 'ALIYUN',
        region: 'cn-hangzhou',
        zone: 'cn-hangzhou-a',
        instanceType: 'ecs.g6.large',
        imageId: 'centos_7_9_x64',
        vpcId: 'vpc-default',
        securityGroupIds: ['sg-default'],
        instanceChargeType: 'PostPaid',
        password: '',
        description: '',
        tags: []
      });
    }
  }, 1500);
};

// 查看资源详情
const handleViewDetail = (type: string, record: any) => {
  resourceType.value = type;
  resourceDetail.value = record;
  detailVisible.value = true;
};

// 处理ECS操作(启动/停止/重启)
const handleEcsAction = (action: string, record: any) => {
  const actionMap: Record<string, string> = {
    'start': '启动',
    'stop': '停止',
    'restart': '重启'
  };
  
  message.loading(`正在${actionMap[action]}云服务器，请稍候...`, 1);
  
  // 模拟操作请求
  setTimeout(() => {
    // 更新本地数据状态
    if (action === 'start') {
      record.status = 'running';
    } else if (action === 'stop') {
      record.status = 'stopped';
    }
    
    message.success(`云服务器${actionMap[action]}操作已完成`);
  }, 1500);
};

// 处理RDS操作(启动/停止/重启)
const handleRdsAction = (action: string, record: any) => {
  const actionMap: Record<string, string> = {
    'start': '启动',
    'stop': '停止',
    'restart': '重启'
  };
  
  message.loading(`正在${actionMap[action]}数据库实例，请稍候...`, 1);
  
  // 模拟操作请求
  setTimeout(() => {
    // 更新本地数据状态
    if (action === 'start') {
      record.status = 'running';
    } else if (action === 'stop') {
      record.status = 'stopped';
    }
    
    message.success(`数据库实例${actionMap[action]}操作已完成`);
  }, 1500);
};

// 删除资源
const handleDeleteResource = (type: string, record: any) => {
  const typeMap: Record<string, string> = {
    'ecs': '云服务器',
    'vpc': 'VPC',
    'sg': '安全组',
    'elb': '负载均衡',
    'rds': '数据库'
  };
  
  Modal.confirm({
    title: `确定要删除${typeMap[type]}吗？`,
    content: `您正在删除${typeMap[type]}: ${record.instanceName}，该操作不可恢复。`,
    okText: '确认删除',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      message.loading(`正在删除${typeMap[type]}，请稍候...`, 1);
      
      // 模拟删除请求
      setTimeout(() => {
        // 从本地数据中移除
        if (type === 'ecs') {
          ecsData.value = ecsData.value.filter(item => item.id !== record.id);
        } else if (type === 'vpc') {
          vpcData.value = vpcData.value.filter(item => item.id !== record.id);
        } else if (type === 'sg') {
          sgData.value = sgData.value.filter(item => item.id !== record.id);
        } else if (type === 'elb') {
          elbData.value = elbData.value.filter(item => item.id !== record.id);
        } else if (type === 'rds') {
          rdsData.value = rdsData.value.filter(item => item.id !== record.id);
        }
        
        message.success(`${typeMap[type]}删除成功`);
      }, 1500);
    }
  });
};
</script>

<style scoped lang="scss">
.resource-management {
  padding: 0 16px;
  
  .page-header {
    margin-bottom: 16px;
    padding: 16px 0;
  }
  .filter-card {
    margin-bottom: 16px;
  }
  .resource-tabs {
    .resource-card {
      margin-top: 16px;
      
      :deep(.ant-card-body) {
        padding: 0;
      }
    }
  }
  
  .action-buttons {
    display: flex;
    gap: 8px;
  }
  
  :deep(.ant-table-pagination.ant-pagination) {
    margin: 16px;
  }
}
</style>
