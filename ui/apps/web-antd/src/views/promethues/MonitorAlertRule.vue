<template>
  <div>
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input v-model:value="searchText" placeholder="请输入AlertRule名称" style="width: 200px" />
        <a-button type="primary" size="middle" @click="handleSearch">
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
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增AlertRule</a-button>
      </div>
    </div>

    <!-- AlertRule 列表表格 -->
    <a-table :columns="columns" :data-source="data" row-key="id" :pagination="false">
      <template #expr="{ record }">
        <div style="max-width: 300px; word-break: break-all">
          {{ record.expr }}
        </div>
      </template>
      <template #labels="{ record }">
        <template v-if="record.labels && record.labels.length && record.labels[0] !== ''">
          <a-tag v-for="label in record.labels" :key="label">
            {{ label.split(',')[0] }}: {{ label.split(',')[1] }}
          </a-tag>
        </template>
        <a-tag v-else color="default">无标签</a-tag>
      </template>
      <template #annotations="{ record }">
        <template v-if="record.annotations && record.annotations.length && record.annotations[0] !== ''">
          <a-tag v-for="annotation in record.annotations" :key="annotation">
            {{ annotation.split(',')[0] }}: {{ annotation.split(',')[1] }}
          </a-tag>
        </template>
        <a-tag v-else color="default">无注解</a-tag>
      </template>
      <template #severity="{ record }">
        <a-tag :color="severityColor(record.severity)">
          {{ record.severity }}
        </a-tag>
      </template>
      <template #enable="{ record }">
        <a-tag :color="record.enable === 1 ? 'green' : 'red'">
          {{ record.enable === 1 ? '启用' : '禁用' }}
        </a-tag>
      </template>
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑资源信息">
            <a-button type="link" @click="showEditModal(record)">
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
    </a-table>

    <!-- 分页器 -->
    <a-pagination v-model:current="current" v-model:pageSize="pageSizeRef" :page-size-options="pageSizeOptions"
      :total="total" show-size-changer @change="handlePageChange" @showSizeChange="handleSizeChange" class="pagination">
      <template #buildOptionText="props">
        <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
        <span v-else>全部</span>
      </template>
    </a-pagination>
    <!-- 新增AlertRule模态框 -->
    <a-modal title="新增AlertRule" v-model:visible="isAddModalVisible" @ok="handleAdd" @cancel="closeAddModal">
      <a-form :model="addForm" layout="vertical">
        <a-form-item label="名称" name="name" :rules="[{ required: true, message: '请输入名称' }]">
          <a-input v-model:value="addForm.name" placeholder="请输入名称" />
        </a-form-item>
        <a-form-item label="所属实例池" name="pool_id" :rules="[{ required: true, message: '请选择所属实例池' }]">
          <a-select v-model:value="addForm.pool_id" placeholder="请选择所属实例池">
            <a-select-option v-for="pool in scrapePools" :key="pool.id" :value="pool.id">
              {{ pool.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="发送组" name="send_group_id">
          <a-select v-model:value="addForm.send_group_id" placeholder="请选择发送组">
            <a-select-option v-for="group in sendGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="树节点" name="treeNodeId">
          <a-tree-select v-model:value="addForm.tree_node_id" :tree-data="leafNodes" :tree-default-expand-all="true"
            placeholder="请选择树节点" style="width: 100%" />
        </a-form-item>
        <a-form-item label="表达式" name="expr">
          <a-input v-model:value="addForm.expr" placeholder="请输入表达式" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="validateAddExpression(addForm.expr)">验证表达式</a-button>
        </a-form-item>
        <a-form-item label="严重性" name="severity">
          <a-select v-model:value="addForm.severity" placeholder="请选择严重性">
            <a-select-option value="critical">Critical</a-select-option>
            <a-select-option value="warning">Warning</a-select-option>
            <a-select-option value="info">Info</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="持续时间" name="for_time">
          <a-input v-model:value="addForm.for_time" placeholder="例如: 10s" />
        </a-form-item>
        <!-- 动态标签表单项 -->
        <a-form-item v-for="(label, index) in addForm.labels" :key="label.key" :label="index === 0 ? '分组标签' : ''">
          <a-input v-model:value="label.labelKey" placeholder="标签名" style="width: 40%; margin-right: 8px" />
          <a-input v-model:value="label.labelValue" placeholder="标签值" style="width: 40%; margin-right: 8px" />
          <MinusCircleOutlined class="dynamic-delete-button" @click="removeLabel(label)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addLabel">
            <PlusOutlined />
            添加标签
          </a-button>
        </a-form-item>
        <a-form-item v-for="(annotation, index) in addForm.annotations" :key="annotation.key"
          :label="index === 0 ? '注解' : ''">
          <a-input v-model:value="annotation.labelKey" placeholder="注解名" style="width: 40%; margin-right: 8px" />
          <a-input v-model:value="annotation.labelValue" placeholder="标签值" style="width: 40%; margin-right: 8px" />
          <MinusCircleOutlined class="dynamic-delete-button" @click="removeAnnotation(annotation)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addAnnotation">
            <PlusOutlined />
            添加注解
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>
    <!-- 编辑AlertRule模态框 -->
    <a-modal title="编辑AlertRule" v-model:visible="isEditModalVisible" @ok="handleEdit" @cancel="closeEditModal">
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="名称" name="name" :rules="[{ required: true, message: '请输入名称' }]">
          <a-input v-model:value="editForm.name" placeholder="请输入名称" />
        </a-form-item>
        <a-form-item label="所属实例池" name="pool_id" :rules="[{ required: true, message: '请选择所属实例池' }]">
          <a-select v-model:value="editForm.pool_id" placeholder="请选择所属实例池">
            <a-select-option v-for="pool in scrapePools" :key="pool.id" :value="pool.id">
              {{ pool.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="发送组" name="send_group_id">
          <a-select v-model:value="editForm.send_group_id" placeholder="请选择发送组">
            <a-select-option v-for="group in sendGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="树节点" name="treeNodeId">
          <a-tree-select v-model:value="editForm.tree_node_id" :tree-data="leafNodes" :tree-default-expand-all="true"
            placeholder="请选择树节点" style="width: 100%" />
        </a-form-item>
        <a-form-item label="启用" name="enable">
          <a-switch v-model:checked="editForm.enable" />
        </a-form-item>
        <a-form-item label="表达式" name="expr">
          <a-input v-model:value="editForm.expr" placeholder="请输入表达式" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="validateEditExpression">验证表达式</a-button>
        </a-form-item>
        <a-form-item label="严重性" name="severity">
          <a-select v-model:value="editForm.severity" placeholder="请选择严重性">
            <a-select-option value="critical">Critical</a-select-option>
            <a-select-option value="warning">Warning</a-select-option>
            <a-select-option value="info">Info</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="持续时间" name="for_time">
          <a-input v-model:value="editForm.for_time" placeholder="例如: 10s" />
        </a-form-item>
        <!-- 动态标签表单项 -->
        <a-form-item v-for="(label, index) in editForm.labels" :key="label.key" :label="index === 0 ? '分组标签' : ''">
          <a-input v-model:value="label.labelKey" placeholder="标签名" style="width: 40%; margin-right: 8px" />
          <a-input v-model:value="label.labelValue" placeholder="标签值" style="width: 40%; margin-right: 8px" />
          <MinusCircleOutlined class="dynamic-delete-button" @click="removeEditLabel(label)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addEditLabel">
            <PlusOutlined />
            添加标签
          </a-button>
        </a-form-item>
        <a-form-item v-for="(annotation, index) in editForm.annotations" :key="annotation.key"
          :label="index === 0 ? '注解' : ''">
          <a-input v-model:value="annotation.labelKey" placeholder="注解名" style="width: 40%; margin-right: 8px" />
          <a-input v-model:value="annotation.labelValue" placeholder="标签值" style="width: 40%; margin-right: 8px" />
          <MinusCircleOutlined class="dynamic-delete-button" @click="removeEditAnnotation(annotation)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addEditAnnotation">
            <PlusOutlined />
            添加注解
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined
} from '@ant-design/icons-vue';
import {
  getAlertRulesListApi,
  createAlertRuleApi,
  updateAlertRuleApi,
  deleteAlertRuleApi,
  getAllTreeNodes,
  validateExprApi,
  getAllAlertManagerPoolApi,
  getAllMonitorSendGroupApi,
  getMonitorAlertRuleTotalApi
} from '#/api';
import { Icon } from '@iconify/vue';
import type { AlertRuleItem } from '#/api/core/prometheus';

// 定义树节点数据类型
interface TreeNode {
  id: string;
  title: string;
  children?: TreeNode[];
  isLeaf?: number;
  value?: string;
  key?: string;
}

interface ScrapePool {
  id: number;
  name: string;
}

interface SendGroup {
  id: number;
  name: string;
}

// 数据源
const data = ref<AlertRuleItem[]>([]);
const scrapePools = ref<ScrapePool[]>([]);
const sendGroups = ref<SendGroup[]>([]);

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 搜索文本
const searchText = ref('');

// 加载状态
const loading = ref(false);

// 树形数据
const treeData = ref<TreeNode[]>([]);
const leafNodes = ref<TreeNode[]>([]);



// 表格列配置
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => a.id - b.id,
  },
  {
    title: '名称',
    dataIndex: 'name',
    key: 'name',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => a.name.localeCompare(b.name),
  },
  {
    title: '所属实例池ID',
    dataIndex: 'pool_id',
    key: 'pool_id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => (a.pool_id || 0) - (b.pool_id || 0),
  },
  {
    title: '绑定发送组ID',
    dataIndex: 'send_group_id',
    key: 'send_group_id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => (a.send_group_id || 0) - (b.send_group_id || 0),
  },
  {
    title: '绑定树节点ID',
    dataIndex: 'tree_node_id',
    key: 'tree_node_id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => (a.tree_node_id || 0) - (b.tree_node_id || 0),
  },
  {
    title: '严重性',
    dataIndex: 'severity',
    key: 'severity',
    slots: { customRender: 'severity' },
    sorter: (a: AlertRuleItem, b: AlertRuleItem) =>
      a.severity.localeCompare(b.severity),
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
  },
  {
    title: '是否启用',
    dataIndex: 'enable',
    key: 'enable',
    slots: { customRender: 'enable' },
  },
  {
    title: '标签',
    dataIndex: 'labels',
    key: 'labels',
    slots: { customRender: 'labels' },
  },
  {
    title: '注解',
    dataIndex: 'annotations',
    key: 'annotations',
    slots: { customRender: 'annotations' },
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


// 处理表格变化
const handlePageChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
};

const removeEditLabel = (label: any) => {
  const index = editForm.labels.indexOf(label);
  if (index !== -1) {
    editForm.labels.splice(index, 1);
  }
};

const removeLabel = (label: any) => {
  const index = addForm.labels.indexOf(label);
  if (index !== -1) {
    addForm.labels.splice(index, 1);
  }
};

const addLabel = () => {
  addForm.labels.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const addEditLabel = () => {
  editForm.labels.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const addAnnotation = () => {
  addForm.annotations.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const addEditAnnotation = () => {
  editForm.annotations.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const removeAnnotation = (annotation: any) => {
  const index = addForm.annotations.indexOf(annotation);
  if (index !== -1) {
    addForm.annotations.splice(index, 1);
  }
};

const removeEditAnnotation = (annotation: any) => {
  const index = editForm.annotations.indexOf(annotation);
  if (index !== -1) {
    editForm.annotations.splice(index, 1);
  }
};

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
    console.error(error);
  }
};

// 获取实例池数据
const fetchScrapePools = async () => {
  try {
    const response = await getAllAlertManagerPoolApi();
    scrapePools.value = response;
  } catch (error: any) {
    message.error(error.message || '获取实例池数据失败');
    console.error(error);
  }
};

// 获取发送组数据
const fetchSendGroups = async () => {
  try {
    const response = await getAllMonitorSendGroupApi();
    sendGroups.value = response;
  } catch (error: any) {
    message.error(error.message || '获取发送组数据失败');
    console.error(error);
  }
};

// 搜索处理
const handleSearch = () => {
  current.value = 1;
  fetchAlertRules();
};

// 重置处理
const handleReset = () => {
  searchText.value = '';
  fetchAlertRules();
};

// 新增表单
const addForm = reactive({
  name: '',
  pool_id: null,
  send_group_id: null,
  tree_node_id: null,
  enable: true,
  expr: '',
  severity: '',
  grafana_link: '',
  for_time: '',
  labels: [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'bind_tree_node', labelValue: '', key: Date.now() + 1 },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 },
    { labelKey: 'alert_rule_id', labelValue: '', key: Date.now() + 3 }
  ],
  annotations: [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'bind_tree_node', labelValue: '', key: Date.now() + 1 },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 }
  ],
});

// 编辑表单
const editForm = reactive({
  id: 0,
  name: '',
  pool_id: null,
  send_group_id: null,
  tree_node_id: null,
  enable: true,
  expr: '',
  severity: '',
  grafana_link: '',
  for_time: '',
  labels: [{ labelKey: '', labelValue: '', key: Date.now() }],
  annotations: [{ labelKey: '', labelValue: '', key: Date.now() }],
});

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

const resetAddForm = () => {
  addForm.name = '';
  addForm.pool_id = null;
  addForm.send_group_id = null;
  addForm.tree_node_id = null;
  addForm.enable = true;
  addForm.expr = '';
  addForm.severity = '';
  addForm.grafana_link = '';
  addForm.for_time = '';
  addForm.labels = [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'bind_tree_node', labelValue: '', key: Date.now() + 1 },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 },
    { labelKey: 'alert_rule_id', labelValue: '', key: Date.now() + 3 }
  ];
  addForm.annotations = [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'bind_tree_node', labelValue: '', key: Date.now() + 1 },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 }
  ];
};

// 关闭新增模态框
const closeAddModal = () => {
  isAddModalVisible.value = false;
};

// 显示编辑模态框
const showEditModal = (record: AlertRuleItem) => {
  Object.assign(editForm, {
    id: record.id,
    name: record.name,
    pool_id: record.pool_id || null,
    send_group_id: record.send_group_id || null,
    tree_node_id: record.tree_node_id || null,
    enable: record.enable,
    expr: record.expr,
    severity: record.severity,
    grafana_link: record.grafana_link,
    for_time: record.for_time,
    labels: record.labels ?
      record.labels.map((value: string) => {
        const [labelKey, labelValue] = value.split(',');
        return {
          labelKey: labelKey || '',
          labelValue: labelValue || '',
          key: Date.now()
        };
      }) : [],
    annotations: record.annotations ?
      record.annotations.map((value: string) => {
        const [labelKey, labelValue] = value.split(',');
        return {
          labelKey: labelKey || '',
          labelValue: labelValue || '',
          key: Date.now()
        };
      }) : [],
  });
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交新增 AlertRule
const handleAdd = async () => {
  try {
    // 表单验证逻辑可以在此添加
    if (addForm.name === '' || addForm.pool_id === 0) {
      message.error('请填写所有必填项');
      return;
    }

    const formData = {
      ...addForm,
      labels: addForm.labels
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
      annotations: addForm.annotations
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
    };

    await createAlertRuleApi(formData); // 调用创建 API
    message.success('新增AlertRule成功');
    fetchAlertRules();
    closeAddModal();
  } catch (error: any) {
    message.error(error.message || '新增AlertRule失败');
    console.error(error);
  }
};

// 提交更新AlertRule
const handleEdit = async () => {
  try {
    if (editForm.name === '' || editForm.pool_id === 0) {
      message.error('请填写所有必填项');
      return;
    }

    const formData = {
      ...editForm,
      labels: editForm.labels
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
      annotations: editForm.annotations
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
    };

    await updateAlertRuleApi(formData); // 调用更新 API
    message.success('更新AlertRule成功');
    fetchAlertRules();
    closeEditModal();
  } catch (error: any) {
    message.error(error.message || '更新AlertRule失败');
    console.error(error);
  }
};

// 处理删除AlertRule
const handleDelete = (record: AlertRuleItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除AlertRule "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await deleteAlertRuleApi(record.id); // 调用删除 API
        message.success('AlertRule已删除');
        fetchAlertRules();
      } catch (error: any) {

        message.error(error.message || '删除AlertRule失败');
        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 获取AlertRules数据
const fetchAlertRules = async () => {
  try {
    loading.value = true;
    const response = await getAlertRulesListApi(current.value, pageSizeRef.value, searchText.value);
    data.value = response;
    total.value = await getMonitorAlertRuleTotalApi();

  } catch (error: any) {
    message.error(error.message || '获取AlertRules数据失败');
    console.error(error);
  } finally {
    loading.value = false;
  }
};

// 定义Severity颜色映射
const severityColor = (severity: string) => {
  switch (severity) {
    case 'critical':
      return 'red';
    case 'warning':
      return 'orange';
    case 'info':
      return 'blue';
    default:
      return 'default';
  }
};

const validateAddExpression = async (expr: string) => {
  try {
    const payload = { promql_expr: expr };
    const result = await validateExprApi(payload);
    message.success('验证表达式成功', result.message);
    return true;
  } catch (error: any) {
    message.error(error.message || '验证表达式失败');
    console.error(error);
    return false;
  }
};

// 验证表达式的方法（编辑）
const validateEditExpression = async () => {
  try {
    const payload = { promql_expr: editForm.expr };
    const result = await validateExprApi(payload);
    message.success('验证表达式成功', result.message);
    return true;
  } catch (error: any) {
    message.error(error.message || '验证表达式失败');
    console.error(error);
    return false;
  }
};

// 在组件加载时获取数据
onMounted(() => {
  fetchAlertRules();
  fetchTreeNodes();
  fetchScrapePools();
  fetchSendGroups();
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
