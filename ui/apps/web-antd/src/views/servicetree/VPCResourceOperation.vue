<template>
  <div class="vpc-resource-container">
    <a-page-header title="VPC资源管理" subtitle="管理您的云上VPC网络资源" class="page-header">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <template #icon>
            <plus-outlined />
          </template>
          创建VPC
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
      <!-- 筛选条件 -->
      <a-card class="filter-card" :bordered="false">
        <a-form layout="inline">
          <a-form-item label="云提供商">
            <a-select v-model:value="queryParams.provider" style="width: 160px" placeholder="选择云提供商" allow-clear>
              <a-select-option v-for="item in PROVIDER_OPTIONS" :key="item.value" :value="item.value">
                {{ item.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="区域">
            <a-select v-model:value="queryParams.region" style="width: 160px" placeholder="选择区域" allow-clear>
              <a-select-option v-for="item in REGION_OPTIONS" :key="item.value" :value="item.value">
                {{ item.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item>
            <a-button type="primary" @click="handleSearch">
              <template #icon>
                <search-outlined />
              </template>
              查询
            </a-button>
            <a-button style="margin-left: 8px" @click="resetQuery">
              <template #icon>
                <clear-outlined />
              </template>
              重置
            </a-button>
          </a-form-item>
        </a-form>
      </a-card>

      <!-- VPC资源列表 -->
      <a-card title="VPC资源列表" class="resource-card">
        <a-table :loading="loading" :columns="columns" :data-source="vpcList" :pagination="pagination"
          @change="handleTableChange" row-key="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'provider'">
              <a-tag :color="getProviderColor(record.provider)">
                {{ getProviderName(record.provider) }}
              </a-tag>
            </template>
            <template v-if="column.key === 'isDefault'">
              <a-badge :status="record.isDefault ? 'success' : 'default'" :text="record.isDefault ? '是' : '否'" />
            </template>
            <template v-if="column.key === 'action'">
              <a-space>
                <a-button type="link" size="small" @click="viewVpcDetails(record)">
                  <template #icon><eye-outlined /></template>
                  详情
                </a-button>
                <a-button type="link" size="small" @click="showDeleteConfirm(record)">
                  <template #icon><delete-outlined /></template>
                  删除
                </a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- VPC创建弹窗 -->
    <a-modal v-model:visible="createModalVisible" title="创建VPC资源" :confirm-loading="submitLoading"
      @ok="handleCreateSubmit" width="700px">
      <a-form :model="createForm" :rules="rules" ref="createFormRef" :label-col="{ span: 6 }"
        :wrapper-col="{ span: 16 }">
        <a-form-item label="云提供商" name="provider">
          <a-select v-model:value="createForm.provider" placeholder="选择云提供商" @change="handleProviderChange">
            <a-select-option v-for="item in PROVIDER_OPTIONS" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="区域" name="region">
          <a-select v-model:value="createForm.region" placeholder="选择区域" @change="handleRegionChange"
            :disabled="!createForm.provider">
            <a-select-option v-for="item in REGION_OPTIONS" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="可用区" name="zoneId">
          <a-select v-model:value="createForm.zoneId" placeholder="选择可用区" :disabled="!createForm.region">
            <a-select-option v-for="item in ZONE_OPTIONS" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="VPC名称" name="vpcName">
          <a-input v-model:value="createForm.vpcName" placeholder="请输入VPC名称" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="createForm.description" placeholder="请输入描述信息" :rows="2" />
        </a-form-item>
        <a-form-item label="IPv4网段" name="cidrBlock">
          <a-select v-model:value="createForm.cidrBlock" placeholder="选择IPv4网段">
            <a-select-option v-for="item in CIDR_BLOCK_OPTIONS" :key="item.value" :value="item.value">
              {{ item.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="交换机名称" name="vSwitchName">
          <a-input v-model:value="createForm.vSwitchName" placeholder="请输入交换机名称" />
        </a-form-item>
        <a-form-item label="交换机网段" name="vSwitchCidrBlock">
          <a-input v-model:value="createForm.vSwitchCidrBlock" placeholder="请输入交换机网段，如：192.168.0.0/24" />
        </a-form-item>
        <a-form-item label="仅预览" name="dryRun">
          <a-switch v-model:checked="createForm.dryRun" />
        </a-form-item>
        <a-form-item label="标签">
          <a-button type="dashed" @click="addTag" block>
            <plus-outlined /> 添加标签
          </a-button>
          <div v-for="(tag, index) in tagList" :key="index" style="margin-top: 8px">
            <a-input-group compact>
              <a-input style="width: 40%" v-model:value="tag.key" placeholder="标签键" @change="updateTags" />
              <a-input style="width: 40%" v-model:value="tag.value" placeholder="标签值" @change="updateTags" />
              <a-button type="danger" style="width: 20%" @click="removeTag(index)">
                <delete-outlined />
              </a-button>
            </a-input-group>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- VPC详情抽屉 -->
    <a-drawer v-model:visible="detailDrawerVisible" title="VPC详情" placement="right" width="600"
      :footer-style="{ textAlign: 'right' }" :closable="true">
      <template v-if="selectedVpc">
        <a-descriptions bordered :column="1">
          <a-descriptions-item label="VPC ID">{{ selectedVpc.id }}</a-descriptions-item>
          <a-descriptions-item label="VPC名称">{{ selectedVpc.vpcName }}</a-descriptions-item>
          <a-descriptions-item label="云提供商">{{ getProviderName(selectedVpc.provider) }}</a-descriptions-item>
          <a-descriptions-item label="区域">{{ getRegionName(selectedVpc.region) }}</a-descriptions-item>
          <a-descriptions-item label="IPv4网段">{{ selectedVpc.cidrBlock }}</a-descriptions-item>
          <a-descriptions-item label="IPv6网段">{{ selectedVpc.ipv6CidrBlock || '未配置' }}</a-descriptions-item>
          <a-descriptions-item label="是否默认VPC">{{ selectedVpc.isDefault ? '是' : '否' }}</a-descriptions-item>
          <a-descriptions-item label="资源组ID">{{ selectedVpc.resourceGroupId || '默认资源组' }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ selectedVpc.createdAt }}</a-descriptions-item>
        </a-descriptions>

        <a-divider orientation="left">交换机列表</a-divider>
        <a-table :columns="vSwitchColumns" :data-source="vSwitchList" :pagination="false" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-badge status="success" text="可用" />
            </template>
            <template v-if="column.key === 'zoneId'">
              {{ getZoneName(record.zoneId) }}
            </template>
          </template>
        </a-table>

        <a-divider orientation="left">路由表</a-divider>
        <a-table :columns="routeTableColumns" :data-source="routeTableList" :pagination="false" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-badge status="success" text="可用" />
            </template>
          </template>
        </a-table>

        <a-divider orientation="left">关联服务树节点</a-divider>
        <a-empty v-if="!selectedVpc.vpcTreeNodes || selectedVpc.vpcTreeNodes.length === 0" description="暂无关联节点" />
        <a-tag v-else v-for="node in selectedVpc.vpcTreeNodes" :key="node.id" style="margin-bottom: 8px">
          {{ node.name }}
        </a-tag>
      </template>
      <template #footer>
        <a-button style="margin-right: 8px" @click="detailDrawerVisible = false">关闭</a-button>
        <a-button type="danger" @click="showDeleteConfirm(selectedVpc)">删除</a-button>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  ClearOutlined,
  EyeOutlined,
  DeleteOutlined
} from '@ant-design/icons-vue';

const PROVIDER_OPTIONS = [
  { value: 'ALIYUN', label: '阿里云', color: 'orange' },
  { value: 'AWS', label: 'AWS', color: 'blue' },
  { value: 'TENCENT', label: '腾讯云', color: 'green' },
  { value: 'HUAWEI', label: '华为云', color: 'red' }
];

const REGION_OPTIONS = [
  { value: 'cn-hangzhou', label: '华东1（杭州）' },
  { value: 'cn-beijing', label: '华北2（北京）' },
  { value: 'cn-shanghai', label: '华东2（上海）' },
  { value: 'cn-shenzhen', label: '华南1（深圳）' }
];

const ZONE_OPTIONS = [
  { value: 'cn-hangzhou-h', label: '杭州 可用区H' },
  { value: 'cn-hangzhou-i', label: '杭州 可用区I' },
  { value: 'cn-hangzhou-j', label: '杭州 可用区J' }
];

const CIDR_BLOCK_OPTIONS = [
  { value: '192.168.0.0/16', label: '192.168.0.0/16' },
  { value: '172.16.0.0/12', label: '172.16.0.0/12' },
  { value: '10.0.0.0/8', label: '10.0.0.0/8' }
];

interface VpcTreeNode {
  id: string;
  name: string;
}

interface VpcResource {
  id: string;
  provider: string;
  region: string;
  vpcName: string;
  cidrBlock: string;
  ipv6CidrBlock?: string;
  vSwitchIds: string[];
  routeTableIds: string[];
  natGatewayIds: string[];
  isDefault: boolean;
  resourceGroupId?: string;
  createdAt: string;
  updatedAt: string;
  vpcTreeNodes?: VpcTreeNode[];
}

interface VSwitch {
  id: string;
  name: string;
  cidrBlock: string;
  zoneId: string;
  status: string;
}

interface RouteTable {
  id: string;
  name: string;
  type: string;
  status: string;
}

interface QueryParams {
  provider: string | null;
  region: string | null;
  pageNumber: number;
  pageSize: number;
}

interface CreateVpcForm {
  provider: string | null;
  region: string | null;
  zoneId: string | null;
  vpcName: string;
  description: string;
  cidrBlock: string | null;
  vSwitchName: string;
  vSwitchCidrBlock: string;
  dryRun: boolean;
  tags: Record<string, string>;
}

interface TagItem {
  key: string;
  value: string;
}

const loading = ref(false);
const createModalVisible = ref(false);
const detailDrawerVisible = ref(false);
const submitLoading = ref(false);
const createFormRef = ref<FormInstance>();
const selectedVpc = ref<VpcResource | null>(null);
const vpcList = ref<VpcResource[]>([]);
const vSwitchList = ref<VSwitch[]>([]);
const routeTableList = ref<RouteTable[]>([]);
const tagList = ref<TagItem[]>([]);

// 查询参数
const queryParams = reactive<QueryParams>({
  provider: null,
  region: null,
  pageNumber: 1,
  pageSize: 10
});

// 创建表单数据
const createForm = reactive<CreateVpcForm>({
  provider: null,
  region: null,
  zoneId: null,
  vpcName: '',
  description: '',
  cidrBlock: null,
  vSwitchName: '',
  vSwitchCidrBlock: '',
  dryRun: false,
  tags: {}
});

// 表单验证规则
const rules = {
  provider: [{ required: true, message: '请选择云提供商', trigger: 'change' }],
  region: [{ required: true, message: '请选择区域', trigger: 'change' }],
  zoneId: [{ required: true, message: '请选择可用区', trigger: 'change' }],
  vpcName: [{ required: true, message: '请输入VPC名称', trigger: 'blur' }],
  cidrBlock: [{ required: true, message: '请选择IPv4网段', trigger: 'change' }],
  vSwitchName: [{ required: true, message: '请输入交换机名称', trigger: 'blur' }],
  vSwitchCidrBlock: [
    { required: true, message: '请输入交换机网段', trigger: 'blur' },
    { pattern: /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/, message: '请输入有效的CIDR格式，如：192.168.0.0/24', trigger: 'blur' }
  ]
};

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

const columns = [
  { title: 'VPC ID', dataIndex: 'id', key: 'id', width: 180, ellipsis: true },
  { title: 'VPC名称', dataIndex: 'vpcName', key: 'vpcName', ellipsis: true },
  { title: '云提供商', dataIndex: 'provider', key: 'provider', width: 120 },
  { title: '区域', dataIndex: 'region', key: 'region', width: 150 },
  { title: 'IPv4网段', dataIndex: 'cidrBlock', key: 'cidrBlock', width: 150 },
  { title: '默认VPC', dataIndex: 'isDefault', key: 'isDefault', width: 100 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' as const }
];

const vSwitchColumns = [
  { title: '交换机ID', dataIndex: 'id', key: 'id', width: 180, ellipsis: true },
  { title: '名称', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: '可用区', dataIndex: 'zoneId', key: 'zoneId', width: 150 },
  { title: '网段', dataIndex: 'cidrBlock', key: 'cidrBlock', width: 150 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 }
];

const routeTableColumns = [
  { title: '路由表ID', dataIndex: 'id', key: 'id', width: 180, ellipsis: true },
  { title: '名称', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 }
];

onMounted(() => {
  fetchVpcList();
});

const getProviderName = (providerValue: string): string => {
  return PROVIDER_OPTIONS.find(p => p.value === providerValue)?.label || providerValue;
};

const getProviderColor = (providerValue: string): string => {
  return PROVIDER_OPTIONS.find(p => p.value === providerValue)?.color || 'default';
};

const getRegionName = (regionValue: string): string => {
  return REGION_OPTIONS.find(r => r.value === regionValue)?.label || regionValue;
};

const getZoneName = (zoneValue: string): string => {
  return ZONE_OPTIONS.find(z => z.value === zoneValue)?.label || zoneValue;
};

// --- 模拟数据生成 ---

const generateMockVpc = (index: number): VpcResource => {
  const providerOption = PROVIDER_OPTIONS[Math.floor(Math.random() * PROVIDER_OPTIONS.length)];
  const provider = providerOption ? providerOption.value : 'ALIYUN';

  const regionOption = REGION_OPTIONS[Math.floor(Math.random() * REGION_OPTIONS.length)];
  const region = regionOption ? regionOption.value : 'cn-hangzhou';

  const cidrBlockOption = CIDR_BLOCK_OPTIONS[Math.floor(Math.random() * CIDR_BLOCK_OPTIONS.length)];
  const cidrBlock = cidrBlockOption ? cidrBlockOption.value : '192.168.0.0/16';

  return {
    id: `vpc-${Math.random().toString(36).substring(2, 10)}`,
    provider,
    region,
    vpcName: `测试VPC-${index}`,
    cidrBlock,
    ipv6CidrBlock: Math.random() > 0.5 ? '2001:db8::/64' : undefined,
    vSwitchIds: [`vsw-${Math.random().toString(36).substring(2, 10)}`],
    routeTableIds: [`rtb-${Math.random().toString(36).substring(2, 10)}`],
    natGatewayIds: Math.random() > 0.5 ? [`nat-${Math.random().toString(36).substring(2, 10)}`] : [],
    isDefault: Math.random() > 0.8,
    resourceGroupId: Math.random() > 0.5 ? `rg-${Math.random().toString(36).substring(2, 10)}` : undefined,
    createdAt: new Date(Date.now() - Math.floor(Math.random() * 10000000000)).toLocaleString(),
    updatedAt: new Date(Date.now() - Math.floor(Math.random() * 1000000000)).toLocaleString(),
    vpcTreeNodes: Math.random() > 0.3 ? [{ id: `node-${index}`, name: `服务节点-${index}` }] : []
  };
};

const generateMockVSwitch = (id: string, vpcCidr: string): VSwitch => {
  const zone = ZONE_OPTIONS[Math.floor(Math.random() * ZONE_OPTIONS.length)];
  const vSwitchCidr = vpcCidr.replace(/\/\d+$/, '/24');
  return {
    id,
    name: `交换机-${id.substring(4, 8)}`,
    cidrBlock: vSwitchCidr,
    zoneId: zone ? zone.value : 'cn-hangzhou-k',
    status: 'Available'
  };
};

const generateMockRouteTable = (id: string): RouteTable => {
  return {
    id,
    name: `路由表-${id.substring(4, 8)}`,
    type: Math.random() > 0.5 ? '系统路由表' : '自定义路由表',
    status: 'Available'
  };
};

// --- 方法定义 ---

const fetchVpcList = () => {
  loading.value = true;
  console.log('获取VPC列表，参数:', JSON.parse(JSON.stringify(queryParams)));
  setTimeout(() => {
    const mockData: VpcResource[] = Array.from({ length: 35 }, (_, i) => generateMockVpc(i + 1));

    let filteredData = mockData.filter(item => {
      const providerMatch = !queryParams.provider || item.provider === queryParams.provider;
      const regionMatch = !queryParams.region || item.region === queryParams.region;
      return providerMatch && regionMatch;
    });

    pagination.total = filteredData.length;

    const start = (queryParams.pageNumber - 1) * queryParams.pageSize;
    const end = start + queryParams.pageSize;
    vpcList.value = filteredData.slice(start, end);

    loading.value = false;
  }, 500);
};

const handleSearch = () => {
  queryParams.pageNumber = 1;
  pagination.current = 1;
  fetchVpcList();
};

const resetQuery = () => {
  queryParams.provider = null;
  queryParams.region = null;
  queryParams.pageNumber = 1;
  pagination.current = 1;
  fetchVpcList();
};

const refreshData = () => {
  fetchVpcList();
  message.success('数据已刷新');
};

const handleTableChange = (pag: any) => {
  queryParams.pageNumber = pag.current;
  queryParams.pageSize = pag.pageSize;
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchVpcList();
};

const resetCreateForm = () => {
  createForm.provider = null;
  createForm.region = null;
  createForm.zoneId = null;
  createForm.vpcName = '';
  createForm.description = '';
  createForm.cidrBlock = null;
  createForm.vSwitchName = '';
  createForm.vSwitchCidrBlock = '';
  createForm.dryRun = false;
  createForm.tags = {};
  tagList.value = [];
  createFormRef.value?.clearValidate();
};

const showCreateModal = () => {
  resetCreateForm();
  createModalVisible.value = true;
};

const handleProviderChange = (value: string) => {
  createForm.region = null;
  createForm.zoneId = null;
};

const handleRegionChange = (value: string) => {
  createForm.zoneId = null;
};

const addTag = () => {
  tagList.value.push({ key: '', value: '' });
};

const removeTag = (index: number) => {
  tagList.value.splice(index, 1);
  updateTags();
};

const updateTags = () => {
  const tags: Record<string, string> = {};
  tagList.value.forEach(tag => {
    if (tag.key) {
      tags[tag.key] = tag.value;
    }
  });
  createForm.tags = tags;
};

const handleCreateSubmit = async () => {
  if (!createFormRef.value) return;

  try {
    await createFormRef.value.validate();
    submitLoading.value = true;
    console.log('提交VPC创建表单:', JSON.parse(JSON.stringify(createForm)));

    setTimeout(() => {
      const newVpc: VpcResource = {
        id: `vpc-${Math.random().toString(36).substring(2, 10)}`,
        provider: createForm.provider!,
        region: createForm.region!,
        vpcName: createForm.vpcName,
        cidrBlock: createForm.cidrBlock!,
        vSwitchIds: [`vsw-${Math.random().toString(36).substring(2, 10)}`],
        routeTableIds: [`rtb-${Math.random().toString(36).substring(2, 10)}`],
        natGatewayIds: [],
        isDefault: false,
        createdAt: new Date().toLocaleString(),
        updatedAt: new Date().toLocaleString(),
      };

      vpcList.value.unshift(newVpc);
      message.success('VPC创建成功');
      submitLoading.value = false;
      createModalVisible.value = false;

      fetchVpcList();

    }, 1000);
  } catch (errorInfo) {
    console.log('表单验证失败:', errorInfo);
    message.error('请检查表单输入项');
    submitLoading.value = false;
  }
};

const viewVpcDetails = (vpc: VpcResource) => {
  selectedVpc.value = vpc;
  detailDrawerVisible.value = true;

  vSwitchList.value = vpc.vSwitchIds.map(id => generateMockVSwitch(id, vpc.cidrBlock));
  routeTableList.value = vpc.routeTableIds.map(id => generateMockRouteTable(id));
};

const showDeleteConfirm = (vpc: VpcResource | null) => {
  if (!vpc) return;

  Modal.confirm({
    title: '确认删除',
    content: `确定要删除VPC "${vpc.vpcName}" (ID: ${vpc.id}) 吗？此操作通常不可恢复。`,
    okText: '确认删除',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      console.log(`尝试删除VPC: ${vpc.id}`);
      return new Promise((resolve, reject) => {
        setTimeout(() => {
          const index = vpcList.value.findIndex(item => item.id === vpc.id);
          if (index !== -1) {
            vpcList.value.splice(index, 1);
            message.success(`VPC "${vpc.vpcName}" 已删除`);

            if (detailDrawerVisible.value && selectedVpc.value?.id === vpc.id) {
              detailDrawerVisible.value = false;
              selectedVpc.value = null;
            }

            fetchVpcList();
            resolve(true);
          } else {
            message.error('删除失败，未找到该VPC');
            reject(new Error('未找到VPC'));
          }
        }, 500);
      }).catch(() => console.log('操作出错'));
    },
    onCancel() {
      console.log('取消删除');
    },
  });
};

</script>

<style scoped lang="scss">
.vpc-resource-container {
  padding: 0 16px;

  .page-header {
    padding: 16px 24px;
    margin-bottom: 16px;
  }

  .content-layout {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  :deep(.ant-table-cell) {
    .ant-space {
      white-space: nowrap;
    }
  }

  .ant-modal-body {
    .ant-input-group {
      display: flex;

      .ant-input {
        flex: 1;
      }

      .ant-btn {
        width: auto;
        padding: 0 10px;
      }
    }
  }
}
</style>
