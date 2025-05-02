<template>
  <div class="security-group-container">
    <!-- 标题栏 -->
    <div class="header-section">
      <a-row align="middle">
        <a-col :span="12">
          <div class="page-title">安全组管理控制台</div>
        </a-col>
        <a-col :span="12" class="header-actions">
          <a-button type="primary" shape="round" @click="showCreateModal">
            <plus-outlined /> 创建安全组
          </a-button>
        </a-col>
      </a-row>
    </div>

    <!-- 筛选区域 -->
    <a-card class="filter-card" :bordered="false">
      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item label="云厂商">
            <a-select v-model:value="filterParams.provider" placeholder="请选择云厂商" allowClear>
              <a-select-option v-for="provider in cloudProviders" :key="provider.value" :value="provider.value">
                {{ provider.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="地区">
            <a-select v-model:value="filterParams.region" placeholder="请选择地区" allowClear>
              <a-select-option v-for="region in regions" :key="region.value" :value="region.value">
                {{ region.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="8" class="search-buttons">
          <a-button type="primary" @click="fetchSecurityGroups">
            <search-outlined /> 查询
          </a-button>
          <a-button class="reset-btn" @click="resetFilters">
            <reload-outlined /> 重置
          </a-button>
        </a-col>
      </a-row>
    </a-card>

    <!-- 安全组列表 -->
    <a-card class="sg-list-card" :bordered="false">
      <a-table :columns="columns" :data-source="securityGroupList" :loading="tableLoading" :pagination="pagination"
        @change="handleTableChange" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 'Available' ? 'success' : 'processing'">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'provider'">
            {{ getProviderName(record.provider) }}
          </template>
          <template v-if="column.key === 'action'">
            <div class="action-buttons">
              <a-button type="link" size="small" @click="showDetailDrawer(record)" class="detail-btn">
                <eye-outlined /> 详情
              </a-button>
              <a-popconfirm title="确定要删除此安全组吗？" ok-text="确定" cancel-text="取消" @confirm="deleteSecurityGroup(record)">
                <a-button type="link" size="small" danger>
                  <delete-outlined /> 删除
                </a-button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建安全组对话框 -->
    <a-modal v-model:visible="createModalVisible" title="创建安全组" width="700px" :confirm-loading="modalLoading"
      @ok="handleCreateSecurityGroup" :destroyOnClose="true" class="create-modal">
      <a-form :model="createForm" layout="vertical" ref="createFormRef" class="create-form">
        <a-form-item label="云厂商" name="provider" :rules="[{ required: true, message: '请选择云厂商' }]">
          <a-select v-model:value="createForm.provider" placeholder="请选择云厂商">
            <a-select-option v-for="provider in cloudProviders" :key="provider.value" :value="provider.value">
              {{ provider.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="地区" name="region" :rules="[{ required: true, message: '请选择地区' }]">
          <a-select v-model:value="createForm.region" placeholder="请选择地区">
            <a-select-option v-for="region in regions" :key="region.value" :value="region.value">
              {{ region.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="安全组名称" name="securityGroupName" :rules="[{ required: true, message: '请输入安全组名称' }]">
          <a-input v-model:value="createForm.securityGroupName" placeholder="请输入安全组名称" />
        </a-form-item>
        <a-form-item label="VPC" name="vpcId" :rules="[{ required: true, message: '请选择VPC' }]">
          <a-select v-model:value="createForm.vpcId" placeholder="请选择VPC">
            <a-select-option v-for="vpc in vpcOptions" :key="vpc.value" :value="vpc.value">
              {{ vpc.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="安全组类型" name="securityGroupType">
          <a-select v-model:value="createForm.securityGroupType" placeholder="请选择安全组类型">
            <a-select-option value="normal">普通安全组</a-select-option>
            <a-select-option value="enterprise">企业安全组</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="createForm.description" placeholder="请输入描述信息" :rows="2" />
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
      </a-form>
    </a-modal>

    <!-- 安全组详情抽屉 -->
    <a-drawer v-model:visible="detailDrawerVisible" title="安全组详情" width="700" placement="right" :destroyOnClose="true"
      class="detail-drawer">
      <a-skeleton :loading="detailLoading" active>
        <a-descriptions bordered :column="1" size="small">
          <a-descriptions-item label="安全组ID">{{ securityGroupDetail.id }}</a-descriptions-item>
          <a-descriptions-item label="安全组名称">{{ securityGroupDetail.securityGroupName }}</a-descriptions-item>
          <a-descriptions-item label="云厂商">{{ getProviderName(securityGroupDetail.provider) }}</a-descriptions-item>
          <a-descriptions-item label="地区">{{ securityGroupDetail.region }}</a-descriptions-item>
          <a-descriptions-item label="VPC ID">{{ securityGroupDetail.vpcId }}</a-descriptions-item>
          <a-descriptions-item label="安全组类型">{{ securityGroupDetail.securityGroupType === 'normal' ? '普通安全组' : '企业安全组'
          }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ securityGroupDetail.creationTime }}</a-descriptions-item>
          <a-descriptions-item label="描述">{{ securityGroupDetail.description || '无' }}</a-descriptions-item>
        </a-descriptions>

        <a-divider orientation="left">安全组规则</a-divider>

        <a-tabs default-active-key="ingress">
          <a-tab-pane key="ingress" tab="入方向规则">
            <a-button type="primary" size="small" style="margin-bottom: 16px" @click="showAddRuleModal('ingress')">
              <plus-outlined /> 添加入方向规则
            </a-button>
            <a-table :columns="ingressRuleColumns" :data-source="ingressRules" :pagination="false" size="small">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'action'">
                  <a-popconfirm title="确定要删除此规则吗？" ok-text="确定" cancel-text="取消" @confirm="deleteRule(record)">
                    <a-button type="link" size="small" danger>
                      <delete-outlined /> 删除
                    </a-button>
                  </a-popconfirm>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
          <a-tab-pane key="egress" tab="出方向规则">
            <a-button type="primary" size="small" style="margin-bottom: 16px" @click="showAddRuleModal('egress')">
              <plus-outlined /> 添加出方向规则
            </a-button>
            <a-table :columns="egressRuleColumns" :data-source="egressRules" :pagination="false" size="small">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'action'">
                  <a-popconfirm title="确定要删除此规则吗？" ok-text="确定" cancel-text="取消" @confirm="deleteRule(record)">
                    <a-button type="link" size="small" danger>
                      <delete-outlined /> 删除
                    </a-button>
                  </a-popconfirm>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
        </a-tabs>

        <a-divider orientation="left">标签</a-divider>
        <div class="tag-list">
          <a-tag v-for="(tag, index) in securityGroupDetail.tags" :key="index" color="blue">{{ tag }}</a-tag>
          <a-empty v-if="!securityGroupDetail.tags || securityGroupDetail.tags.length === 0"
            :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无标签" />
        </div>
      </a-skeleton>
    </a-drawer>

    <!-- 添加规则对话框 -->
    <a-modal v-model:visible="ruleModalVisible" :title="`添加${ruleDirection === 'ingress' ? '入' : '出'}方向规则`"
      @ok="handleAddRule" :confirm-loading="ruleModalLoading" :destroyOnClose="true">
      <a-form :model="ruleForm" layout="vertical">
        <a-form-item label="协议类型" name="ipProtocol" :rules="[{ required: true, message: '请选择协议类型' }]">
          <a-select v-model:value="ruleForm.ipProtocol" placeholder="请选择协议类型">
            <a-select-option value="tcp">TCP</a-select-option>
            <a-select-option value="udp">UDP</a-select-option>
            <a-select-option value="icmp">ICMP</a-select-option>
            <a-select-option value="all">ALL</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="端口范围" name="portRange" :rules="[{ required: true, message: '请输入端口范围' }]">
          <a-input v-model:value="ruleForm.portRange" placeholder="例如：1/200 或 80/80" />
        </a-form-item>
        <a-form-item label="授权策略" name="policy">
          <a-radio-group v-model:value="ruleForm.policy">
            <a-radio value="accept">接受</a-radio>
            <a-radio value="drop">拒绝</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="优先级" name="priority">
          <a-input-number v-model:value="ruleForm.priority" :min="1" :max="100" style="width: 100%" />
        </a-form-item>
        <a-form-item :label="ruleDirection === 'ingress' ? '源IP地址段' : '目标IP地址段'" name="cidrIp">
          <a-input :value="ruleDirection === 'ingress' ? ruleForm.sourceCidrIp : ruleForm.destCidrIp"
            @update:value="(val: string) => ruleDirection === 'ingress' ? ruleForm.sourceCidrIp = val : ruleForm.destCidrIp = val"
            placeholder="例如：0.0.0.0/0" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="ruleForm.description" placeholder="请输入描述信息" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { message, Empty } from 'ant-design-vue';
import {
  SearchOutlined,
  PlusOutlined,
  ReloadOutlined,
  EyeOutlined,
  DeleteOutlined,
} from '@ant-design/icons-vue';

// 类型定义
interface SecurityGroup {
  id: string;
  securityGroupName: string;
  provider: string;
  region: string;
  status: string;
  vpcId: string;
  securityGroupType: string;
  description: string;
  creationTime: string;
  tags: string[];
  securityGroupRules: SecurityGroupRule[];
}

interface SecurityGroupRule {
  id: number;
  securityGroupId: string;
  ipProtocol: string;
  portRange: string;
  direction: 'ingress' | 'egress';
  policy: 'accept' | 'drop';
  priority: number;
  sourceCidrIp: string;
  destCidrIp: string;
  sourceGroupId: string;
  destGroupId: string;
  description: string;
}

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

// VPC选项 (模拟数据，实际应从API获取)
const vpcOptions = [
  { label: 'vpc-default (默认VPC)', value: 'vpc-default' },
  { label: 'vpc-app (应用VPC)', value: 'vpc-app' },
  { label: 'vpc-db (数据库VPC)', value: 'vpc-db' }
];

// 表格列定义
const columns = [
  { title: '安全组ID', dataIndex: 'id', key: 'id', width: 150 },
  { title: '安全组名称', dataIndex: 'securityGroupName', key: 'securityGroupName' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '云厂商', dataIndex: 'provider', key: 'provider', width: 100 },
  { title: '地区', dataIndex: 'region', key: 'region', width: 150 },
  { title: 'VPC ID', dataIndex: 'vpcId', key: 'vpcId', width: 150 },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime', width: 180 },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 入方向规则表格列
const ingressRuleColumns = [
  { title: '协议类型', dataIndex: 'ipProtocol', key: 'ipProtocol', width: 80 },
  { title: '端口范围', dataIndex: 'portRange', key: 'portRange', width: 100 },
  {
    title: '授权策略', dataIndex: 'policy', key: 'policy', width: 80,
    customRender: ({ text }: { text: string }) => text === 'accept' ? '接受' : '拒绝'
  },
  { title: '优先级', dataIndex: 'priority', key: 'priority', width: 80 },
  { title: '源IP地址段', dataIndex: 'sourceCidrIp', key: 'sourceCidrIp' },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '操作', key: 'action', width: 80 }
];

// 出方向规则表格列
const egressRuleColumns = [
  { title: '协议类型', dataIndex: 'ipProtocol', key: 'ipProtocol', width: 80 },
  { title: '端口范围', dataIndex: 'portRange', key: 'portRange', width: 100 },
  {
    title: '授权策略', dataIndex: 'policy', key: 'policy', width: 80,
    customRender: ({ text }: { text: string }) => text === 'accept' ? '接受' : '拒绝'
  },
  { title: '优先级', dataIndex: 'priority', key: 'priority', width: 80 },
  { title: '目标IP地址段', dataIndex: 'destCidrIp', key: 'destCidrIp' },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '操作', key: 'action', width: 80 }
];

// 状态
const tableLoading = ref(false);
const securityGroupList = ref<SecurityGroup[]>([]);
const createModalVisible = ref(false);
const modalLoading = ref(false);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const securityGroupDetail = ref<SecurityGroup>({} as SecurityGroup);
const ruleModalVisible = ref(false);
const ruleModalLoading = ref(false);
const ruleDirection = ref<'ingress' | 'egress'>('ingress');
const createFormRef = ref(); // 创建表单引用

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

// 过滤参数
const filterParams = reactive({
  provider: undefined,
  region: undefined,
});

// 创建表单
const createForm = reactive({
  provider: 'ALIYUN',
  region: 'cn-hangzhou',
  securityGroupName: '',
  vpcId: 'vpc-default',
  securityGroupType: 'normal',
  description: '',
  tags: {} as Record<string, string>,
});

// 规则表单
const ruleForm = reactive({
  ipProtocol: 'tcp',
  portRange: '1/65535',
  direction: 'ingress',
  policy: 'accept',
  priority: 1,
  sourceCidrIp: '0.0.0.0/0',
  destCidrIp: '0.0.0.0/0',
  description: '',
});

// 标签相关
const tagsArray = ref<string[]>([]);
const tagInputValue = ref('');

// 计算属性
const ingressRules = computed(() => {
  return securityGroupDetail.value.securityGroupRules?.filter(rule => rule.direction === 'ingress') || [];
});

const egressRules = computed(() => {
  return securityGroupDetail.value.securityGroupRules?.filter(rule => rule.direction === 'egress') || [];
});

// 生命周期钩子
onMounted(() => {
  fetchSecurityGroups();
});

// 获取安全组列表
const fetchSecurityGroups = async () => {
  tableLoading.value = true;
  try {
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));
    const mockData = [
      {
        id: 'sg-123456',
        securityGroupName: '默认安全组',
        provider: 'ALIYUN',
        region: 'cn-hangzhou',
        status: 'Available',
        vpcId: 'vpc-default',
        securityGroupType: 'normal',
        description: '默认安全组，允许所有出站流量',
        creationTime: '2023-01-01 12:00:00',
        tags: ['env=prod', 'app=web'],
        securityGroupRules: [
          {
            id: 1,
            securityGroupId: 'sg-123456',
            ipProtocol: 'tcp',
            portRange: '22/22',
            direction: 'ingress',
            policy: 'accept',
            priority: 1,
            sourceCidrIp: '0.0.0.0/0',
            destCidrIp: '',
            sourceGroupId: '',
            destGroupId: '',
            description: 'SSH访问'
          },
          {
            id: 2,
            securityGroupId: 'sg-123456',
            ipProtocol: 'tcp',
            portRange: '80/80',
            direction: 'ingress', // ✅ 修改
            policy: 'accept',
            priority: 1,
            sourceCidrIp: '0.0.0.0/0',
            destCidrIp: '',
            sourceGroupId: '',
            destGroupId: '',
            description: 'HTTP访问'
          },
          {
            id: 3,
            securityGroupId: 'sg-123456',
            ipProtocol: 'all',
            portRange: '-1/-1',
            direction: 'egress',
            policy: 'accept',
            priority: 1,
            sourceCidrIp: '',
            destCidrIp: '0.0.0.0/0',
            sourceGroupId: '',
            destGroupId: '',
            description: '允许所有出站流量'
          }
        ]
      },
      {
        id: 'sg-234567',
        securityGroupName: 'Web服务安全组',
        provider: 'TENCENT',
        region: 'cn-beijing',
        status: 'Available',
        vpcId: 'vpc-app',
        securityGroupType: 'normal',
        description: 'Web服务器安全组',
        creationTime: '2023-02-01 12:00:00',
        tags: ['env=test'],
        securityGroupRules: []
      }
    ];
    securityGroupList.value = mockData.filter(sg => {
      const providerMatch = !filterParams.provider || sg.provider === filterParams.provider;
      const regionMatch = !filterParams.region || sg.region === filterParams.region;
      return providerMatch && regionMatch;
    }) as SecurityGroup[];
    pagination.total = securityGroupList.value.length;

  } catch (error) {
    message.error('获取安全组列表失败');
    console.error('获取安全组列表失败:', error);
  } finally {
    tableLoading.value = false;
  }
};

// 表格变化处理
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchSecurityGroups();
};

// 重置筛选条件
const resetFilters = () => {
  filterParams.provider = undefined;
  filterParams.region = undefined;
  pagination.current = 1; // 重置到第一页
  fetchSecurityGroups();
};

// 显示创建模态框
const showCreateModal = () => {
  // 重置表单和标签
  Object.assign(createForm, {
    provider: 'ALIYUN',
    region: 'cn-hangzhou',
    securityGroupName: '',
    vpcId: 'vpc-default',
    securityGroupType: 'normal',
    description: '',
    tags: {},
  });
  tagsArray.value = [];
  tagInputValue.value = '';
  createModalVisible.value = true;
};

// 处理创建安全组
const handleCreateSecurityGroup = async () => {
  try {
    // await createFormRef.value?.validate(); // 触发表单验证
    if (!createForm.securityGroupName) { // 简单验证
      message.warning('请输入安全组名称');
      return;
    }

    modalLoading.value = true;

    // 处理标签
    const tags: Record<string, string> = {};
    tagsArray.value.forEach(tag => {
      const [key, value] = tag.split('=');
      if (key && value) {
        tags[key.trim()] = value.trim();
      }
    });
    createForm.tags = tags;

    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));
    console.log('创建安全组参数:', createForm);
    message.success('安全组创建成功');
    createModalVisible.value = false;
    fetchSecurityGroups(); // 刷新列表
  } catch (errorInfo) {
    console.log('表单验证失败:', errorInfo);
    message.error('请检查表单输入');
  } finally {
    modalLoading.value = false;
  }
};

// 显示详情抽屉
const showDetailDrawer = (record: SecurityGroup) => {
  detailDrawerVisible.value = true;
  detailLoading.value = true;

  // 模拟API调用获取详情
  setTimeout(() => {
    // 深拷贝记录以防意外修改列表数据
    securityGroupDetail.value = JSON.parse(JSON.stringify(record));
    detailLoading.value = false;
  }, 500); // 模拟加载时间
};

// 删除安全组
const deleteSecurityGroup = async (record: SecurityGroup) => {
  tableLoading.value = true;
  try {
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));
    console.log('删除安全组:', record.id);
    message.success('安全组删除成功');
    fetchSecurityGroups(); // 刷新列表
  } catch (error) {
    message.error('删除安全组失败');
    console.error('删除安全组失败:', error);
    tableLoading.value = false; // 仅在失败时保留loading
  }
};

// 显示添加规则模态框
const showAddRuleModal = (direction: 'ingress' | 'egress') => {
  ruleDirection.value = direction;
  // 重置规则表单
  Object.assign(ruleForm, {
    ipProtocol: 'tcp',
    portRange: '1/65535',
    direction: direction,
    policy: 'accept',
    priority: 1,
    sourceCidrIp: direction === 'ingress' ? '0.0.0.0/0' : '',
    destCidrIp: direction === 'egress' ? '0.0.0.0/0' : '',
    description: '',
  });
  ruleModalVisible.value = true;
};

// 处理添加规则
const handleAddRule = async () => {
  ruleModalLoading.value = true;
  try {
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));

    const newRule: SecurityGroupRule = {
      id: Math.floor(Math.random() * 1000) + 10, // 模拟生成ID
      securityGroupId: securityGroupDetail.value.id,
      ipProtocol: ruleForm.ipProtocol,
      portRange: ruleForm.portRange,
      direction: ruleDirection.value,
      policy: ruleForm.policy as 'accept' | 'drop',
      priority: ruleForm.priority,
      sourceCidrIp: ruleDirection.value === 'ingress' ? ruleForm.sourceCidrIp : '',
      destCidrIp: ruleDirection.value === 'egress' ? ruleForm.destCidrIp : '',
      sourceGroupId: '', // 模拟数据
      destGroupId: '', // 模拟数据
      description: ruleForm.description
    };

    console.log('添加规则:', newRule);

    if (!securityGroupDetail.value.securityGroupRules) {
      securityGroupDetail.value.securityGroupRules = [];
    }
    securityGroupDetail.value.securityGroupRules.push(newRule); // 直接更新详情里的规则列表

    message.success('安全组规则添加成功');
    ruleModalVisible.value = false;
  } catch (error) {
    message.error('添加规则失败');
    console.error('添加规则失败:', error);
  } finally {
    ruleModalLoading.value = false;
  }
};

// 删除规则
const deleteRule = async (record: SecurityGroupRule) => {
  if (!securityGroupDetail.value.securityGroupRules) return;

  detailLoading.value = true; // 显示加载状态
  try {
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 500));
    console.log('删除规则:', record.id);

    securityGroupDetail.value.securityGroupRules = securityGroupDetail.value.securityGroupRules.filter(
      rule => rule.id !== record.id
    );
    message.success('安全组规则删除成功');
  } catch (error) {
    message.error('删除规则失败');
    console.error('删除规则失败:', error);
  } finally {
    detailLoading.value = false;
  }
};

// 添加标签
const addTag = () => {
  if (tagInputValue.value) {
    if (tagInputValue.value.includes('=')) {
      tagsArray.value.push(tagInputValue.value);
      tagInputValue.value = '';
    } else {
      message.warning('标签格式应为 key=value');
    }
  }
};

// 移除标签
const removeTag = (index: number) => {
  tagsArray.value.splice(index, 1);
};

// 获取状态文本
const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    'Available': '可用',
    'Creating': '创建中',
    'Deleting': '删除中'
  };
  return statusMap[status] || status;
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
</script>

<style scoped lang="scss">
.security-group-container {
  padding: 24px;
  height: 100%;
  min-height: 100vh;
}

.header-section {
  margin-bottom: 24px;
  padding: 16px 24px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.09);
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
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.09);
}

.search-buttons {
  display: flex;
  justify-content: flex-end;
  align-items: flex-end;
  height: 100%;
}

.reset-btn {
  margin-left: 8px;
}

.sg-list-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.09);
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 0px;
}

.detail-btn {
  padding-left: 0;
  padding-right: 8px;
}

.create-modal .create-form {
  max-height: 60vh;
  overflow-y: auto;
  padding: 0 12px;
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

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
  margin-bottom: 20px;
}

.detail-drawer {
  .ant-descriptions-item-label {
    width: 120px;
  }
}

:deep(.ant-form-item) {
  margin-bottom: 20px;
}

:deep(.ant-tag) {
  margin-right: 0;
}

.detail-drawer :deep(.ant-table-small) {
  border: 1px solid #f0f0f0;
  border-radius: 2px;
}
</style>
