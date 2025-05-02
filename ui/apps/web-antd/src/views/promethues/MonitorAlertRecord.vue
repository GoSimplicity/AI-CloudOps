<template>
  <div>
    <!-- 查询和操作工具栏 -->
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input
          v-model:value="searchText"
          placeholder="请输入记录名称"
          style="width: 200px"
        />
        <a-button type="primary" size="middle" @click="handleSearch">
          <template #icon><SearchOutlined /></template>
          搜索
        </a-button>
        <a-button @click="handleReset">
          <template #icon><ReloadOutlined /></template>
          重置
        </a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增记录</a-button>
      </div>
    </div>

    <!-- 记录列表表格 -->
    <a-table
      :columns="columns"
      :data-source="data"
      row-key="id"
      :loading="loading"
      :pagination="false"
    >
      <!-- 标签组列 -->
      <template #labels="{ record }">
        <a-tag v-for="label in record.labels" :key="label">{{ label }}</a-tag>
      </template>
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑资源信息">
            <a-button type="link" @click="showEditModal(record)">
              <template #icon><Icon icon="clarity:note-edit-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="删除资源">
            <a-button type="link" danger @click="handleDelete(record)">
              <template #icon><Icon icon="ant-design:delete-outlined" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
        </a-space>
      </template>
    </a-table>

        <!-- 分页器 -->
      <a-pagination
      v-model:current="current"
      v-model:pageSize="pageSizeRef"
      :page-size-options="pageSizeOptions"
      :total="total"
      show-size-changer
      @change="handlePageChange"
      @showSizeChange="handleSizeChange"
      class="pagination"
    >
      <template #buildOptionText="props">
        <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
        <span v-else>全部</span>
      </template>
    </a-pagination>

    <!-- 新增记录规则模态框 -->
    <a-modal
      title="新增记录规则"
      v-model:visible="isAddModalVisible"
      @ok="handleAdd"
      @cancel="closeAddModal"
      :confirmLoading="loading"
      ok-text="提交"
      cancel-text="取消"
    >
      <a-form :model="addForm" layout="vertical">
        <a-form-item
          label="记录名称"
          name="name"
          :rules="[{ required: true, message: '请输入记录名称' }]"
        >
          <a-input
            v-model:value="addForm.name"
            placeholder="请输入记录名称"
          />
        </a-form-item>
        <a-form-item
          label="Prometheus 实例池"
          name="poolId"
          :rules="[{ required: true, message: '请选择实例池' }]"
        >
          <a-select
            v-model:value="addForm.pool_id"
            placeholder="请选择实例池"
            style="width: 100%"
          >
            <a-select-option
              v-for="pool in poolOptions"
              :key="pool.id"
              :value="pool.id"
            >
              {{ pool.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item
          label="树节点"
          name="tree_node_id"
        >
          <a-select
            v-model:value="addForm.tree_node_id"
            placeholder="请选择树节点"
            style="width: 100%"
          >
            <a-select-option
              v-for="node in treeNodeOptions"
              :key="node.id"
              :value="node.id"
            >
              {{ node.title }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item
          label="是否启用"
          name="enable"
          :rules="[{ required: true, message: '请选择是否启用' }]"
        >
          <a-switch v-model:checked="addForm.enable" />
        </a-form-item>

        <a-form-item
          label="持续时间"
          name="forTime"
        >
          <a-input v-model:value="addForm.for_time" placeholder="例如: 15s" />
        </a-form-item>

        <a-form-item
          label="表达式"
          name="expr"
        >
          <a-input v-model:value="addForm.expr" placeholder="请输入表达式" />
        </a-form-item>

        <a-form-item>
          <a-button type="primary" @click="validateAddExpression"
            >验证表达式</a-button
          >
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑记录规则模态框 -->
    <a-modal
      title="编辑记录"
      v-model:visible="isEditModalVisible"
      @ok="handleUpdate"
      @cancel="closeEditModal"
      :confirmLoading="loading"
      ok-text="提交"
      cancel-text="取消"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item
          label="记录名称"
          name="name"
          :rules="[{ required: true, message: '请输入记录名称' }]"
        >
          <a-input
            v-model:value="editForm.name"
            placeholder="请输入记录名称"
          />
        </a-form-item>
        <a-form-item
          label="Prometheus 实例池"
          name="poolId"
          :rules="[{ required: true, message: '请选择实例池' }]"
        >
          <a-select
            v-model:value="editForm.pool_id"
            placeholder="请选择实例池"
            style="width: 100%"
          >
            <a-select-option
              v-for="pool in poolOptions"
              :key="pool.id"
              :value="pool.id"
            >
              {{ pool.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="树节点" name="TreeNodeID">
          <a-select
            v-model:value="editForm.tree_node_id"
            placeholder="请选择树节点"
            style="width: 100%"
          >
            <a-select-option
              v-for="node in treeNodeOptions"
              :key="node.id"
              :value="node.id"
            >
              {{ node.title }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item
          label="是否启用"
          name="enable"
          :rules="[{ required: true, message: '请选择是否启用' }]"
        >
          <a-switch v-model:checked="editForm.enable" />
        </a-form-item>

        <a-form-item
          label="持续时间"
          name="forTime"
        >
          <a-input v-model:value="editForm.for_time" placeholder="例如: 15s" />
        </a-form-item>

        <a-form-item
          label="表达式"
          name="expr"
        >
          <a-input v-model:value="editForm.expr" placeholder="请输入表达式" />
        </a-form-item>

        <a-form-item>
          <a-button type="primary" @click="validateEditExpression"
            >验证表达式</a-button
          >
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  getRecordRulesListApi,
  createRecordRuleApi,
  updateRecordRuleApi,
  deleteRecordRuleApi,
  getAllMonitorScrapePoolApi,
  getRecordRulesTotalApi,
  getAllTreeNodes,
  validateExprApi,
} from '#/api';
import { Icon } from '@iconify/vue';
import {
  SearchOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue';
import type { AlertRecordItem } from '#/api';


// 定义 Pool 和 TreeNode 类型
interface Pool {
  id: number;
  name: string;
}

interface TreeNode {
  id: number;
  title: string;
}

// 数据源
const data = ref<AlertRecordItem[]>([]);

// 下拉框数据源
const poolOptions = ref<Pool[]>([]);
const treeNodeOptions = ref<TreeNode[]>([]);

// 搜索文本
const searchText = ref('');

const handleReset = () => {
  searchText.value = '';
  fetchRecordRules();
};

// 处理搜索
const handleSearch = () => {
  current.value = 1;
  fetchRecordRules();
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchRecordRules();
};

// 处理分页变化
const handlePageChange = (page: number) => {
  current.value = page;
  fetchRecordRules();
};

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 加载状态
const loading = ref(false);

// 表格列配置
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    sorter: (a: AlertRecordItem, b: AlertRecordItem) => a.id - b.id,
  },
  {
    title: '记录名称',
    dataIndex: 'name',
    key: 'name',
    sorter: (a: AlertRecordItem, b: AlertRecordItem) => a.name.localeCompare(b.name),
  },
  {
    title: '关联 Prometheus 实例池',
    dataIndex: 'pool_name',
    key: 'pool_name',
    sorter: (a: AlertRecordItem, b: AlertRecordItem) => a.pool_id - b.pool_id,
  },
  {
    title: '绑定树节点id',
    dataIndex: 'tree_node_id',
    key: 'tree_node_id',
    sorter: (a: AlertRecordItem, b: AlertRecordItem) => a.tree_node_id - b.tree_node_id,
  },
  {
    title: '是否启用',
    dataIndex: 'enable',
    key: 'enable',
    customRender: ({ text }: { text: boolean }) =>
      text ? '启用' : '禁用',
  },
  {
    title: '持续时间',
    dataIndex: 'for_time',
    key: 'for_time',
    sorter: (a: AlertRecordItem, b: AlertRecordItem) =>
      a.for_time.localeCompare(b.for_time),
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 模态框状态和表单
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);

// 新增表单
const addForm = reactive({
  name: '',
  pool_id: null,
  tree_node_id: null,
  enable: false,
  for_time: '15s',
  expr: '',
  labels: [],
  annotations: [],
});

// 编辑表单
const editForm = reactive({
  id: 0,
  name: '',
  pool_id: null,
  tree_node_id: null,
  enable: true,
  for_time: '',
  expr: '',
  labels: [],
  annotations: [],
});

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.pool_id = null;
  addForm.tree_node_id = null;
  addForm.enable = false;
  addForm.for_time = '15s';
  addForm.expr = '';
  addForm.labels = [];
  addForm.annotations = [];
};

// 关闭新增模态框
const closeAddModal = () => {
  isAddModalVisible.value = false;
};

// 显示编辑模态框
const showEditModal = (record: AlertRecordItem) => {
  Object.assign(editForm, {
    id: record.id,
    name: record.name,
    pool_id: record.pool_id,
    tree_node_id: record.tree_node_id,
    enable: record.enable,
    for_time: record.for_time,
    expr: record.expr,
    labels: record.labels,
    annotations: record.annotations,
  });
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交新增记录
const handleAdd = async () => {
  try {
    const payload = {
      name: addForm.name,
      pool_id: addForm.pool_id,
      tree_node_id: addForm.tree_node_id,
      enable: addForm.enable,
      for_time: addForm.for_time,
      expr: addForm.expr,
      labels: addForm.labels,
      annotations: addForm.annotations,
    };

    loading.value = true;
    await createRecordRuleApi(payload); // 调用创建 API
    loading.value = false;
    message.success('新增记录成功');
    fetchRecordRules();
    closeAddModal();
  } catch (error: any) {
    loading.value = false;
    message.error(error.message || '新增记录失败，请稍后重试');

    console.error(error);
  }
};

// 提交更新记录
const handleUpdate = async () => {
  try {
    const payload = {
      id: editForm.id,
      name: editForm.name,
      pool_id: editForm.pool_id,
      tree_node_id: editForm.tree_node_id,
      enable: editForm.enable,
      for_time: editForm.for_time,
      expr: editForm.expr,
      labels: editForm.labels,
      annotations: editForm.annotations,
    };

    loading.value = true;
    await updateRecordRuleApi(payload); // 调用更新 API
    loading.value = false;
    message.success('更新记录规则成功');
    fetchRecordRules();
    closeEditModal();
  } catch (error: any) {
    loading.value = false;
    message.error(error.message || '更新记录规则失败，请稍后重试');
    console.error(error);
  }
};

// 处理删除记录规则
const handleDelete = (record: AlertRecordItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除记录规则 "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await deleteRecordRuleApi(record.id); // 调用删除 API
        loading.value = false;
        message.success('记录规则已删除');
        fetchRecordRules();
      } catch (error: any) {
        loading.value = false;
        message.error(error.message || '删除记录规则失败，请稍后重试');
        console.error(error);
      }
    },
  });
};

// 获取记录规则数据
const fetchRecordRules = async () => {
  try {
    const response = await getRecordRulesListApi(current.value, pageSizeRef.value, searchText.value); // 调用获取数据 API
    data.value = response;
    total.value = await getRecordRulesTotalApi();
  } catch (error: any) {

    message.error(error.message || '获取记录规则数据失败，请稍后重试');
    console.error(error);
  }
};

// 获取所有实例池数据
const fetchPools = async () => {
  try {
    const response = await getAllMonitorScrapePoolApi(); // 调用获取实例池 API
    poolOptions.value = response;
  } catch (error: any) {
    message.error(error.message || '获取实例池数据失败，请稍后重试');
    console.error(error);
  }
};

// 获取所有树节点数据
const fetchTreeNodes = async () => {
  try {
    const response = await getAllTreeNodes(); // 调用获取树节点 API
    treeNodeOptions.value = response;
  } catch (error: any) {
    message.error(error.message || '获取树节点数据失败，请稍后重试');
    console.error(error);
  }
};

// 表达式验证（新增）
const validateAddExpression = async () => {
  try {
    if (!addForm.expr) {
      message.error('表达式不能为空');
      return;
    }
    const payload = { promql_expr: addForm.expr };
    await validateExprApi(payload); // 调用验证 API
    message.success('表达式验证成功');
  } catch (error: any) {
    message.error(error.message || '表达式验证失败，请稍后重试');
    console.error(error);
  }
};

// 表达式验证（编辑）
const validateEditExpression = async () => {
  try {
    if (!editForm.expr) {
      message.error('表达式不能为空');
      return;
    }
    const payload = { promql_expr: editForm.expr };
    await validateExprApi(payload); // 调用验证 API
    message.success('表达式验证成功');
  } catch (error: any) {
    message.error(error.message || '表达式验证失败，请稍后重试');
    console.error(error);
  }
};

// 在组件加载时获取数据
onMounted(() => {
  fetchRecordRules();
  fetchPools();
  fetchTreeNodes();
});
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
