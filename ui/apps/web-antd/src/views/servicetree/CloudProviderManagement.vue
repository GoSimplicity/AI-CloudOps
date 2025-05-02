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
      <!-- 云账户列表 -->
      <a-card title="云账户列表" class="account-card">
        <template #extra>
          <a-input-search v-model:value="searchValue" placeholder="搜索账户" style="width: 250px" @search="onSearch" />
        </template>

        <a-table :dataSource="cloudAccounts" :columns="columns" :loading="loading" :pagination="{ pageSize: 10 }"
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
                <a-button type="link" size="small" @click="syncAccount(record)">
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

      <!-- 资源统计卡片 -->
      <a-row :gutter="16" style="margin-top: 16px">
        <a-col :span="8">
          <a-card>
            <template #title>
              <div class="card-title">
                <cloud-server-outlined />
                <span>区域分布</span>
              </div>
            </template>
            <a-statistic :value="regionCount" suffix="个区域" />
            <div class="chart-placeholder">
              <div class="chart-mock"></div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card>
            <template #title>
              <div class="card-title">
                <api-outlined />
                <span>实例类型</span>
              </div>
            </template>
            <a-statistic :value="instanceTypeCount" suffix="种规格" />
            <div class="chart-placeholder">
              <div class="chart-mock"></div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card>
            <template #title>
              <div class="card-title">
                <safety-outlined />
                <span>安全组</span>
              </div>
            </template>
            <a-statistic :value="securityGroupCount" suffix="个安全组" />
            <div class="chart-placeholder">
              <div class="chart-mock"></div>
            </div>
          </a-card>
        </a-col>
      </a-row>
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
import { ref, reactive, onMounted, computed } from 'vue';
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
  ApiOutlined,
  SafetyOutlined
} from '@ant-design/icons-vue';

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

// 表格列定义
const columns = [
  {
    title: '账户名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '云提供商',
    dataIndex: 'provider',
    key: 'provider',
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
  },
  {
    title: '操作',
    key: 'action',
  },
];

// 状态变量
const loading = ref(false);
const searchValue = ref('');
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

// 统计数据
const regionCount = ref(12);
const instanceTypeCount = ref(45);
const securityGroupCount = ref(18);

// 计算属性
const regionOptions = computed(() => {
  return regions.value.map(region => ({
    label: `${region.localName} (${region.regionId})`,
    value: region.regionId,
  }));
});

// 生命周期钩子
onMounted(() => {
  fetchProviders();
  fetchCloudAccounts();
});

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
        id: 1,
        name: '生产环境-阿里云',
        provider: 'aliyun',
        accountId: 'aliyun123456',
        accessKey: 'LTAI4*********',
        regions: ['cn-beijing', 'cn-shanghai', 'cn-hangzhou'],
        isEnabled: true,
        lastSyncTime: '2023-05-15 14:30:22',
        description: '阿里云生产环境账户',
      },
      {
        id: 2,
        name: '测试环境-腾讯云',
        provider: 'tencent',
        accountId: 'tencent789012',
        accessKey: 'AKIDz8*********',
        regions: ['ap-beijing', 'ap-shanghai'],
        isEnabled: true,
        lastSyncTime: '2023-05-14 09:15:36',
        description: '腾讯云测试环境账户',
      },
      {
        id: 3,
        name: '开发环境-华为云',
        provider: 'huawei',
        accountId: 'huawei345678',
        accessKey: 'HWSK0*********',
        regions: ['cn-north-4', 'cn-east-3'],
        isEnabled: false,
        lastSyncTime: '2023-05-10 16:42:18',
        description: '华为云开发环境账户',
      },
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
    } else {
      regions.value = [];
    }
    regionsLoading.value = false;
  }, 300);
};

const handleProviderChange = (value: string) => {
  accountForm.regions = [];
  fetchRegions(value);
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
  message.success('数据已刷新');
};

const onSearch = (value: string) => {
  searchValue.value = value;
  // 实际应用中这里应该调用API进行搜索
  message.info(`搜索: ${value}`);
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

const editAccount = (account: CloudAccount) => {
  isEditing.value = true;
  selectedAccount.value = account;
  accountForm.id = account.id;
  accountForm.name = account.name;
  accountForm.provider = account.provider;
  accountForm.accountId = account.accountId;
  accountForm.accessKey = account.accessKey;
  accountForm.secretKey = ''; // 出于安全考虑，不回显密钥
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
            id: accountForm.id as number, // 确保id是number类型
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
  message.loading({ content: `正在同步 ${account.name} 的资源数据...`, key: 'sync' });
  // 模拟同步操作
  setTimeout(() => {
    message.success({ content: `${account.name} 资源数据同步完成`, key: 'sync' });
    // 更新最后同步时间
    const index = cloudAccounts.value.findIndex(item => item.id === account.id);
    if (index !== -1 && cloudAccounts.value[index]) {
      cloudAccounts.value[index].lastSyncTime = new Date().toLocaleString();
    }
  }, 1500);
};

const toggleAccountStatus = (account: CloudAccount) => {
  const index = cloudAccounts.value.findIndex(item => item.id === account.id);
  if (index !== -1 && cloudAccounts.value[index]) {
    cloudAccounts.value[index].isEnabled = !cloudAccounts.value[index].isEnabled;
    const isEnabled = cloudAccounts.value[index]?.isEnabled;
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

    .account-card {
      margin-bottom: 16px;
    }
  }

  .card-title {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .chart-placeholder {
    margin-top: 16px;
    height: 120px;
    display: flex;
    justify-content: center;
    align-items: center;

    .chart-mock {
      width: 100%;
      height: 100%;
      background-size: 20px 20px;
      border-radius: 4px;
    }
  }

  .drawer-actions {
    position: absolute;
    bottom: 24px;
    width: calc(100% - 48px);
    display: flex;
    justify-content: space-between;
  }
}
</style>
