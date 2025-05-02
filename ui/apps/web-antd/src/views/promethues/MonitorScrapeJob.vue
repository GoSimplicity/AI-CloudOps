<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input v-model:value="searchText" placeholder="请输入采集任务名称" style="width: 200px" allow-clear
          @pressEnter="handleSearch" />
        <a-button type="primary" @click="handleSearch">
          <template #icon>
            <SearchOutlined />
          </template>
          搜索
        </a-button>
        <a-button @click="handleReset">
          <template #icon>
            <ReloadOutlined />
          </template>
          重置
        </a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="openAddModal">
          <template #icon>
            <PlusOutlined />
          </template>
          新增采集任务
        </a-button>
      </div>
    </div>

    <!-- 数据加载状态 -->
    <a-spin :spinning="loading">
      <!-- 表格 -->
      <a-table :dataSource="data" :columns="columns" :pagination="false">
        <!-- 服务发现类型列 -->
        <template #serviceDiscoveryType="{ record }">
          <a-tag :color="record.service_discovery_type === 'k8s' ? 'blue' : 'green'">
            {{ record.service_discovery_type === 'k8s' ? 'Kubernetes' : 'HTTP' }}
          </a-tag>
        </template>
        <!-- 关联采集池列 -->
        <template #poolName="{ record }">
          <a-tag color="purple">{{ getPoolName(record.pool_id) }}</a-tag>
        </template>
        <!-- 创建者列 -->
        <template #createUserName="{ record }">
          <a-tag color="cyan">{{ record.create_user_name }}</a-tag>
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-tooltip title="编辑资源信息">
              <a-button type="link" @click="openEditModal(record)">
                <template #icon>
                  <Icon icon="clarity:note-edit-line" style="font-size: 22px" />
                </template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除资源">
              <a-button type="link" danger @click="handleDelete(record)">
                <template #icon>
                  <Icon icon="ant-design:delete-outlined" style="font-size: 22px" />
                </template>
              </a-button>
            </a-tooltip>
          </a-space>
        </template>
        <!-- 树节点列 -->
        <template #treeNodeNames="{ record }">
          <a-tooltip :title="formatTreeNodeNames(record.tree_node_names)">
            <span>{{ formatTreeNodeNames(record.tree_node_names) }}</span>
          </a-tooltip>
        </template>
        <!-- 创建时间列 -->
        <template #created_at="{ record }">
          <a-tooltip :title="formatDate(record.created_at)">
            {{ formatDate(record.created_at) }}
          </a-tooltip>
        </template>
      </a-table>

      <!-- 分页器 -->
      <a-pagination v-model:current="current" v-model:pageSize="pageSizeRef" :page-size-options="pageSizeOptions"
        :total="total" show-size-changer @change="handlePageChange" @showSizeChange="handleSizeChange"
        class="pagination">
        <template #buildOptionText="props">
          <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
          <span v-else>全部</span>
        </template>
      </a-pagination>
    </a-spin>

    <!-- 新增采集任务模态框 -->
    <a-modal v-model:visible="isAddModalVisible" title="新增采集任务" @ok="handleAdd" @cancel="closeAddModal" :okText="'提交'"
      :cancelText="'取消'" :confirmLoading="confirmLoading" :maskClosable="false" width="600px">
      <a-form :model="addForm" layout="vertical" ref="addFormRef">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="采集任务名称" name="name" :rules="[{ required: true, message: '请输入采集任务名称' }]">
              <a-input v-model:value="addForm.name" placeholder="请输入采集任务名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="启用" name="enable">
              <a-switch v-model:checked="addForm.enable" :checked-children="'启用'" :un-checked-children="'禁用'" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="服务发现类型" name="service_discovery_type"
              :rules="[{ required: true, message: '请选择服务发现类型' }]">
              <a-select v-model:value="addForm.service_discovery_type" placeholder="请选择服务发现类型">
                <a-select-option value="http">HTTP</a-select-option>
                <a-select-option value="k8s">Kubernetes</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="协议方案" name="scheme" :rules="[{ required: true, message: '请选择协议方案' }]">
              <a-select v-model:value="addForm.scheme" placeholder="请选择协议方案">
                <a-select-option value="http">HTTP</a-select-option>
                <a-select-option value="https">HTTPS</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="监控采集路径" name="metrics_path" :rules="[{ required: true, message: '请输入监控采集路径' }]">
              <a-input v-model:value="addForm.metrics_path" placeholder="请输入监控采集路径" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="采集间隔（秒）" name="scrape_interval" :rules="[
              { required: true, message: '请输入采集间隔' },
              { type: 'number', min: 1, message: '采集间隔必须大于0' }
            ]">
              <a-input-number v-model:value="addForm.scrape_interval" :min="1" style="width: 100%;"
                placeholder="请输入采集间隔（秒）" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="采集超时（秒）" name="scrape_timeout" :rules="[
              { required: true, message: '请输入采集超时' },
              { type: 'number', min: 1, message: '采集超时必须大于0' }
            ]">
              <a-input-number v-model:value="addForm.scrape_timeout" :min="1" style="width: 100%;"
                placeholder="请输入采集超时（秒）" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="关联采集池" name="pool_id" :rules="[{ required: true, message: '请选择关联采集池' }]">
              <a-select v-model:value="addForm.pool_id" placeholder="请选择关联采集池">
                <a-select-option v-for="pool in pools" :key="pool.id" :value="pool.id">
                  {{ pool.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="刷新间隔（秒）" name="refresh_interval" :rules="[
              { required: true, message: '请输入刷新间隔' },
              { type: 'number', min: 1, message: '刷新间隔必须大于0' }
            ]">
              <a-input-number v-model:value="addForm.refresh_interval" :min="1" style="width: 100%;"
                placeholder="请输入刷新间隔（秒）" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="端口" name="port" :rules="[
              { required: true, message: '请输入端口' },
              { type: 'number', min: 1, max: 65535, message: '端口必须在1-65535之间' }
            ]">
              <a-input-number v-model:value="addForm.port" :min="1" :max="65535" style="width: 100%;"
                placeholder="请输入端口" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="树节点" name="tree_node_ids">
          <a-tree-select v-model:value="addForm.tree_node_ids" :tree-data="leafNodes" :tree-checkable="true"
            :tree-default-expand-all="true" :show-checked-strategy="SHOW_PARENT" placeholder="请选择树节点"
            style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑采集任务模态框 -->
    <a-modal v-model:visible="isEditModalVisible" title="编辑采集任务" @ok="handleUpdate" @cancel="closeEditModal"
      :okText="'提交'" :cancelText="'取消'" :confirmLoading="confirmLoading" :maskClosable="false" width="600px">
      <a-form :model="editForm" layout="vertical" ref="editFormRef" @submit.prevent>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="采集任务名称" name="name" :rules="[{ required: true, message: '请输入采集任务名称' }]">
              <a-input v-model:value="editForm.name" placeholder="请输入采集任务名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="启用" name="enable">
              <a-switch v-model:checked="editForm.enable" :checked-children="'启用'" :un-checked-children="'禁用'" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="服务发现类型" name="service_discovery_type"
              :rules="[{ required: true, message: '请选择服务发现类型' }]">
              <a-select v-model:value="editForm.service_discovery_type" placeholder="请选择服务发现类型">
                <a-select-option value="http">HTTP</a-select-option>
                <a-select-option value="k8s">Kubernetes</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="协议方案" name="scheme" :rules="[{ required: true, message: '请选择协议方案' }]">
              <a-select v-model:value="editForm.scheme" placeholder="请选择协议方案">
                <a-select-option value="http">HTTP</a-select-option>
                <a-select-option value="https">HTTPS</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="监控采集路径" name="metrics_path" :rules="[{ required: true, message: '请输入监控采集路径' }]">
              <a-input v-model:value="editForm.metrics_path" placeholder="请输入监控采集路径" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="采集间隔（秒）" name="scrape_interval" :rules="[
              { required: true, message: '请输入采集间隔' },
              { type: 'number', min: 1, message: '采集间隔必须大于0' }
            ]">
              <a-input-number v-model:value="editForm.scrape_interval" :min="1" style="width: 100%;"
                placeholder="请输入采集间隔（秒）" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="采集超时（秒）" name="scrape_timeout" :rules="[
              { required: true, message: '请输入采集超时' },
              { type: 'number', min: 1, message: '采集超时必须大于0' }
            ]">
              <a-input-number v-model:value="editForm.scrape_timeout" :min="1" style="width: 100%;"
                placeholder="请输入采集超时（秒）" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="关联采集池" name="pool_id" :rules="[{ required: true, message: '请选择关联采集池' }]">
              <a-select v-model:value="editForm.pool_id" placeholder="请选择关联采集池">
                <a-select-option v-for="pool in pools" :key="pool.id" :value="pool.id">
                  {{ pool.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="刷新间隔（秒）" name="refresh_interval" :rules="[
              { required: true, message: '请输入刷新间隔' },
              { type: 'number', min: 1, message: '刷新间隔必须大于0' }
            ]">
              <a-input-number v-model:value="editForm.refresh_interval" :min="1" style="width: 100%;"
                placeholder="请输入刷新间隔（秒）" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="端口" name="port" :rules="[
              { required: true, message: '请输入端口' },
              { type: 'number', min: 1, max: 65535, message: '端口必须在1-65535之间' }
            ]">
              <a-input-number v-model:value="editForm.port" :min="1" :max="65535" style="width: 100%;"
                placeholder="请输入端口" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="树节点" name="tree_node_ids" :rules="[{ required: true, message: '请选择树节点' }]">
          <a-tree-select v-model:value="editForm.tree_node_ids" :tree-data="leafNodes" :tree-checkable="true"
            :tree-default-expand-all="true" :show-checked-strategy="SHOW_PARENT" placeholder="请选择树节点"
            style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { TreeSelect } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import dayjs from 'dayjs';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
} from '@ant-design/icons-vue';
import {
  getMonitorScrapeJobListApi,
  createScrapeJobApi,
  updateScrapeJobApi,
  deleteScrapeJobApi,
  getAllMonitorScrapePoolApi,
  getAllTreeNodes,
  getMonitorScrapeJobTotalApi
} from '#/api';
import type { MonitorScrapeJobItem, createScrapeJobReq, updateScrapeJobReq } from '#/api/core/prometheus';
const { SHOW_PARENT } = TreeSelect;

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 搜索处理
const handleSearch = () => {
  fetchResources();
};

const handleSizeChange = (_current: number, size: number) => {
  pageSizeRef.value = size;
  fetchResources();
};

// 处理分页变化
const handlePageChange = (page: number) => {
  current.value = page;
  fetchResources();
};

const handleReset = () => {
  searchText.value = '';
  fetchResources();
};

interface Pool {
  id: number;
  name: string;
}

interface TreeNode {
  id: string;
  title: string;
  children?: TreeNode[];
  isLeaf?: number;
  value?: string;
  key?: string;
}

// 数据源（待从后端获取）
const data = ref<MonitorScrapeJobItem[]>([]);

// 搜索文本
const searchText = ref('');

// 格式化日期
const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss');
};

// 表格列配置 - 精简后的列
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '采集任务名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '服务发现类型',
    dataIndex: 'service_discovery_type',
    key: 'service_discovery_type',
    slots: { customRender: 'serviceDiscoveryType' },
  },
  {
    title: '监控采集路径',
    dataIndex: 'metrics_path',
    key: 'metrics_path',
  },
  {
    title: '关联采集池',
    dataIndex: 'pool_id',
    key: 'pool_id',
    slots: { customRender: 'poolName' },
  },
  {
    title: '树节点',
    dataIndex: 'tree_node_names',
    key: 'tree_node_names',
    slots: { customRender: 'treeNodeNames' },
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
    slots: { customRender: 'createUserName' },
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    slots: { customRender: 'created_at' },
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 树形数据
const treeData = ref<TreeNode[]>([]);
const leafNodes = ref<TreeNode[]>([]);

// 递归处理树节点数据
const processTreeData = (nodes: any[]): TreeNode[] => {
  return nodes.map(node => {
    const processedNode: TreeNode = {
      id: node.id,
      title: node.name || node.title,
      key: node.id,
      value: node.id,
      isLeaf: node.isLeaf
    };

    if (node.children && node.children.length > 0) {
      processedNode.children = processTreeData(node.children);
    }

    return processedNode;
  });
};

// 递归获取所有叶子节点
const getLeafNodes = (nodes: TreeNode[]): TreeNode[] => {
  let leaves: TreeNode[] = [];
  nodes.forEach(node => {
    if (node.isLeaf === 1) {
      leaves.push(node);
    } else if (node.children) {
      leaves = leaves.concat(getLeafNodes(node.children));
    }
  });
  return leaves;
};

// 获取树节点数据
const fetchTreeNodes = async () => {
  try {
    const response = await getAllTreeNodes();
    if (!response) {
      treeData.value = [];
      leafNodes.value = [];
      return;
    }
    treeData.value = processTreeData(response);
    leafNodes.value = getLeafNodes(treeData.value);
  } catch (error: any) {
    message.error(error.message || '获取树节点数据失败');
  }
};

// 模态框相关状态
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);
const currentRecord = ref<MonitorScrapeJobItem | null>(null);
const confirmLoading = ref(false);

// 表单引用
const addFormRef = ref();
const editFormRef = ref();

// 表单数据模型
const addForm = reactive({
  name: '',
  enable: true,
  service_discovery_type: 'http',
  metrics_path: '/metrics',
  scheme: 'http',
  scrape_interval: 15,
  scrape_timeout: 5,
  pool_id: null as number | null,
  refresh_interval: 30,
  port: 9100,
  tree_node_ids: [] as string[],
  relabel_configs_yaml_string: '',
  kube_config_file_path: '',
  tls_ca_file_path: '',
  tls_ca_content: '',
  bearer_token: '',
  bearer_token_file: '',
  kubernetes_sd_role: '',
});

const editForm = reactive({
  id: 0,
  name: '',
  enable: true,
  service_discovery_type: 'http',
  metrics_path: '/metrics',
  scheme: 'http',
  scrape_interval: 15,
  scrape_timeout: 5,
  pool_id: null as number | null,
  refresh_interval: 30,
  port: 9100,
  tree_node_ids: [] as string[],
  relabel_configs_yaml_string: '',
  kube_config_file_path: '',
  tls_ca_file_path: '',
  tls_ca_content: '',
  bearer_token: '',
  bearer_token_file: '',
  kubernetes_sd_role: '',
});

// 采集池列表
const pools = ref<Pool[]>([]);

// 获取采集池数据
const fetchPools = async () => {
  try {
    const response = await getAllMonitorScrapePoolApi();
    pools.value = response.map((pool: any) => ({
      id: pool.id,
      name: pool.name,
    }));
  } catch (error: any) {
    message.error(error.message || '获取采集池数据失败');
  }
};

// 获取采集任务数据
const loading = ref(false);
const fetchResources = async () => {
  if (current.value < 1) current.value = 1;
  loading.value = true;
  try {
    const response = await getMonitorScrapeJobListApi(current.value, pageSizeRef.value, searchText.value);
    data.value = response.map((item: any) => ({
      ...item,
      // 确保 treeNodeIds 始终是字符串数组
      tree_node_ids: Array.isArray(item.tree_node_ids) ? item.tree_node_ids.map(String) : [],
      tree_node_names: Array.isArray(item.tree_node_names) ? item.tree_node_names : []
    }));
    total.value = await getMonitorScrapeJobTotalApi();

  } catch (error: any) {
    message.error(error.message || '获取采集任务数据失败');
  } finally {
    loading.value = false;
  }
};

// 获取采集池名称
const getPoolName = (poolId: number) => {
  const pool = pools.value.find(p => p.id === poolId);
  return pool ? pool.name : '未知';
};

// 格式化树节点名称
const formatTreeNodeNames = (treeNodeNames: string[]) => {
  if (!Array.isArray(treeNodeNames)) return '';
  return treeNodeNames.join(', ');
};

// 在组件挂载时获取数据
onMounted(() => {
  fetchResources();
  fetchPools();
  fetchTreeNodes();
});

// 打开新增模态框
const openAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 关闭新增模态框
const closeAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = false;
};

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.enable = true;
  addForm.service_discovery_type = 'http';
  addForm.metrics_path = '/metrics';
  addForm.scheme = 'http';
  addForm.scrape_interval = 15;
  addForm.scrape_timeout = 5;
  addForm.pool_id = null;
  addForm.refresh_interval = 30;
  addForm.port = 9100;
  addForm.tree_node_ids = [];
  addForm.relabel_configs_yaml_string = '';
  addForm.kube_config_file_path = '';
  addForm.tls_ca_file_path = '';
  addForm.tls_ca_content = '';
  addForm.bearer_token = '';
  addForm.bearer_token_file = '';
  addForm.kubernetes_sd_role = '';
};

// 提交新增采集任务
const handleAdd = async () => {
  try {
    confirmLoading.value = true;
    // 表单验证
    await addFormRef.value.validateFields();

    // 确保 treeNodeIds 是字符串数组
    const formData: createScrapeJobReq = {
      name: addForm.name,
      enable: addForm.enable,
      service_discovery_type: addForm.service_discovery_type,
      metrics_path: addForm.metrics_path,
      scheme: addForm.scheme,
      scrape_interval: addForm.scrape_interval,
      scrape_timeout: addForm.scrape_timeout,
      pool_id: addForm.pool_id!,
      refresh_interval: addForm.refresh_interval,
      port: addForm.port,
      tree_node_ids: addForm.tree_node_ids.map(String),
      relabel_configs_yaml_string: addForm.relabel_configs_yaml_string,
      kube_config_file_path: addForm.kube_config_file_path,
      tls_ca_file_path: addForm.tls_ca_file_path,
      tls_ca_content: addForm.tls_ca_content,
      bearer_token: addForm.bearer_token,
      bearer_token_file: addForm.bearer_token_file,
      kubernetes_sd_role: addForm.kubernetes_sd_role,
    };

    // 提交数据
    await createScrapeJobApi(formData);
    message.success('新增采集任务成功');
    fetchResources(); // 重新获取数据
    closeAddModal();
  } catch (error: any) {
    message.error(error.message || '新增采集任务失败');
  } finally {
    confirmLoading.value = false;
  }
};

// 打开编辑模态框
const openEditModal = (record: MonitorScrapeJobItem) => {
  currentRecord.value = record;
  Object.assign(editForm, {
    id: record.id,
    name: record.name,
    enable: record.enable,
    service_discovery_type: record.service_discovery_type,
    metrics_path: record.metrics_path,
    scheme: record.scheme,
    scrape_interval: record.scrape_interval,
    scrape_timeout: record.scrape_timeout,
    pool_id: record.pool_id,
    refresh_interval: record.refresh_interval,
    port: record.port,
    tree_node_ids: record.tree_node_ids?.filter(id => id && id.trim() !== '').map(String) || [],
  });
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交更新采集任务
const handleUpdate = async () => {
  try {
    confirmLoading.value = true;
    // 表单验证
    await editFormRef.value.validateFields();

    // 确保 treeNodeIds 是字符串数组
    const formData: updateScrapeJobReq = {
      id: editForm.id,
      name: editForm.name,
      enable: editForm.enable,
      service_discovery_type: editForm.service_discovery_type,
      metrics_path: editForm.metrics_path,
      scheme: editForm.scheme,
      scrape_interval: editForm.scrape_interval,
      scrape_timeout: editForm.scrape_timeout,
      pool_id: editForm.pool_id!,
      refresh_interval: editForm.refresh_interval,
      port: editForm.port,
      tree_node_ids: editForm.tree_node_ids.map(String),
      relabel_configs_yaml_string: editForm.relabel_configs_yaml_string,
      kube_config_file_path: editForm.kube_config_file_path,
      tls_ca_file_path: editForm.tls_ca_file_path,
      tls_ca_content: editForm.tls_ca_content,
      bearer_token: editForm.bearer_token,
      bearer_token_file: editForm.bearer_token_file,
      kubernetes_sd_role: editForm.kubernetes_sd_role
    };

    // 提交数据
    await updateScrapeJobApi(formData);
    message.success('更新采集任务成功');
    fetchResources(); // 重新获取数据
    closeEditModal();
  } catch (error: any) {
    message.error(error.message || '更新采集任务失败');
  } finally {
    confirmLoading.value = false;
  }
};

// 处理删除采集任务
const handleDelete = (record: MonitorScrapeJobItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除采集任务 "${record.name}" 吗？`,
    onOk: async () => {
      try {
        await deleteScrapeJobApi(record.id);
        message.success('删除采集任务成功');
        fetchResources(); // 重新获取数据
      } catch (error: any) {
        message.error(error.message || '删除采集任务失败');
      }
    },
  });
};
</script>

<style scoped>
.custom-toolbar {
  padding: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
  gap: 16px;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.pagination {
  margin-top: 16px;
  text-align: right;
  margin-right: 12px;
}

.dynamic-delete-button {
  cursor: pointer;
  position: relative;
  top: 4px;
  font-size: 24px;
  color: #999;
  transition: all 0.3s;
}

.dynamic-delete-button:hover {
  color: #777;
}

.dynamic-delete-button[disabled] {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
