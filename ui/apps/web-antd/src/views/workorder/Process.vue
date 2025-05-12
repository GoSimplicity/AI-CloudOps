<template>
  <div class="process-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateProcess" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新流程
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索流程..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部</a-select-option>
          <a-select-option :value="0">草稿</a-select-option>
          <a-select-option :value="1">已发布</a-select-option>
          <a-select-option :value="2">已禁用</a-select-option>
        </a-select>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="总流程数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <ApartmentOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已发布" :value="stats.published" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="草稿" :value="stats.draft" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <EditOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已禁用" :value="stats.disabled" :value-style="{ color: '#cf1322' }">
              <template #prefix>
                <StopOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table :data-source="processList" :columns="columns" :pagination="false" :loading="loading"
          row-key="id" bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="process-name-cell">
                <div class="process-badge" :class="getStatusClass(record.status)"></div>
                <span class="process-name-text">{{ record.name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <span class="description-text">{{ record.description || '无描述' }}</span>
            </template>

            <template v-if="column.key === 'version'">
              <a-tag color="blue">v{{ record.version }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 0 ? 'orange' : record.status === 1 ? 'green' : 'default'">
                {{ record.status === 0 ? '草稿' : record.status === 1 ? '已发布' : '已禁用' }}
              </a-tag>
            </template>

            <template v-if="column.key === 'creator'">
              <div class="creator-info">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name) }">
                  {{ getInitials(record.creator_name) }}
                </a-avatar>
                <span class="creator-name">{{ record.creator_name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'createdAt'">
              <div class="date-info">
                <span class="date">{{ formatDate(record.created_at) }}</span>
                <span class="time">{{ formatTime(record.created_at) }}</span>
              </div>
            </template>

            <template v-if="column.key === 'action'">
              <div class="action-buttons">
                <a-button type="primary" size="small" @click="handleViewProcess(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditProcess(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleCommand(e.key, record)">
                      <a-menu-item key="publish" v-if="record.status === 0">发布</a-menu-item>
                      <a-menu-item key="unpublish" v-if="record.status === 1">取消发布</a-menu-item>
                      <a-menu-item key="clone">克隆</a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="delete" danger>删除</a-menu-item>
                    </a-menu>
                  </template>
                  <a-button size="small">
                    更多
                    <DownOutlined />
                  </a-button>
                </a-dropdown>
              </div>
            </template>
          </template>
        </a-table>

        <div class="pagination-container">
          <a-pagination v-model:current="currentPage" :total="total" :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" :show-size-changer="true" 
            @change="handleCurrentChange" @showSizeChange="handleSizeChange" 
            :show-total="(total: number) => `共 ${total} 条`" />
        </div>
      </a-card>
    </div>

    <!-- 流程创建/编辑对话框 -->
    <a-modal v-model:visible="processDialog.visible" :title="processDialog.isEdit ? '编辑流程' : '创建流程'" width="760px"
      @ok="saveProcess" :destroy-on-close="true">
      <a-form ref="formRef" :model="processDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="流程名称" name="name">
          <a-input v-model:value="processDialog.form.name" placeholder="请输入流程名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="processDialog.form.description" :rows="3" placeholder="请输入流程描述" />
        </a-form-item>

        <a-form-item label="关联表单" name="form_design_id">
          <a-select v-model:value="processDialog.form.form_design_id" placeholder="请选择关联表单" style="width: 100%">
            <a-select-option v-for="form in forms" :key="form.id" :value="form.id">
              {{ form.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="processDialog.form.status">
            <a-radio :value="0">草稿</a-radio>
            <a-radio :value="1">已发布</a-radio>
            <a-radio :value="2">已禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider orientation="left">流程节点</a-divider>

        <div class="nodes-editor">
          <div class="node-list">
            <a-collapse>
              <a-collapse-panel v-for="(node, index) in processDialog.form.definition.steps" :key="index"
                :header="node.step || `节点 ${index + 1}`">
                <template #extra>
                  <a-button type="text" danger @click.stop="removeNode(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </template>

                <a-form-item label="节点名称">
                  <a-input v-model:value="node.step" placeholder="节点名称" />
                </a-form-item>

                <a-form-item label="角色">
                  <a-input v-model:value="node.role" placeholder="节点角色" />
                </a-form-item>

                <a-form-item label="动作">
                  <a-input v-model:value="node.action" placeholder="节点动作" />
                </a-form-item>
              </a-collapse-panel>
            </a-collapse>

            <div class="add-node-button">
              <a-button type="dashed" block @click="addNode" style="margin-top: 16px">
                <PlusOutlined /> 添加节点
              </a-button>
            </div>
          </div>
        </div>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆流程" @ok="confirmClone" :destroy-on-close="true">
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新流程名称" name="name">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新流程名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情对话框 -->
    <a-modal v-model:visible="detailDialog.visible" title="流程详情" width="70%" :footer="null" class="detail-dialog">
      <div v-if="detailDialog.process" class="process-details">
        <div class="detail-header">
          <h2>{{ detailDialog.process.name }}</h2>
          <a-tag
            :color="detailDialog.process.status === 0 ? 'orange' : detailDialog.process.status === 1 ? 'green' : 'default'">
            {{ detailDialog.process.status === 0 ? '草稿' : detailDialog.process.status === 1 ? '已发布' : '已禁用' }}
          </a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ detailDialog.process.id }}</a-descriptions-item>
          <a-descriptions-item label="版本">v{{ detailDialog.process.version }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.process.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.process.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="关联表单">{{ getFormName(detailDialog.process.form_design_id) }}</a-descriptions-item>
          <a-descriptions-item label="描述">{{ detailDialog.process.description || '无描述' }}</a-descriptions-item>
        </a-descriptions>

        <div class="process-preview">
          <h3>流程节点</h3>
          <div class="process-flow-chart">
            <div v-for="(step, index) in parsedDefinition(detailDialog.process)" :key="index" class="process-node"
              :class="`node-type-${getNodeTypeClass(step.action)}`">
              <div class="node-header">
                <span class="node-type-badge">{{ getNodeTypeName(step.action) }}</span>
                <span class="node-name">{{ step.step }}</span>
              </div>
              <div class="node-content">
                <div class="node-role">
                  <div>角色：{{ step.role }}</div>
                </div>
                <div class="node-action">
                  <div>动作：{{ step.action }}</div>
                </div>
              </div>
              <div class="node-footer" v-if="index < parsedDefinition(detailDialog.process).length - 1">
                <ArrowDownOutlined />
                <div>下一节点：{{ parsedDefinition(detailDialog.process)[index + 1]?.step || '结束' }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="detail-footer">
          <a-button @click="detailDialog.visible = false">关闭</a-button>
          <a-button type="primary" @click="handleEditProcess(detailDialog.process)">编辑</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  ApartmentOutlined,
  CheckCircleOutlined,
  EditOutlined,
  StopOutlined,
  DeleteOutlined,
  DownOutlined,
  ArrowDownOutlined
} from '@ant-design/icons-vue';

import {
  type Process,
  type ProcessReq,
  type FormDesign,
  type ListProcessReq,
  type DetailProcessReqReq,
  type PublishProcessReq,
  type CloneProcessReq,
  type DeleteProcessReqReq,
  type Step,
  type Definition,
  listProcess,
  detailProcess,
  createProcess,
  updateProcess,
  deleteProcess,
  publishProcess as publishProcessApi,
  cloneProcess as cloneProcessApi,
  listFormDesign,
  type ListFormDesignReq
} from '#/api/core/workorder';

// 列定义
const columns = [
  {
    title: '流程名称',
    dataIndex: 'name',
    key: 'name',
    width: 180,
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    width: 200,
    ellipsis: true,
  },
  {
    title: '版本',
    dataIndex: 'version',
    key: 'version',
    width: 100,
    align: 'center',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    align: 'center',
  },
  {
    title: '创建人',
    dataIndex: 'creator_name',
    key: 'creator',
    width: 150,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'createdAt',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    width: 200,
    align: 'center',
  },
];

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);
const total = ref(0);

// 统计数据
const stats = reactive({
  total: 0,
  published: 0,
  draft: 0,
  disabled: 0
});

// 流程列表
const processList = ref<Process[]>([]);

// 表单列表
const forms = ref<FormDesign[]>([]);

// 流程对话框
const processDialog = reactive({
  visible: false,
  isEdit: false,
  form: {
    id: undefined,
    name: '',
    description: '',
    form_design_id: undefined as number | undefined,
    definition: {
      steps: [] as Step[]
    } as Definition,
    version: undefined,
    status: 0,
    category_id: undefined,
    creator_id: undefined,
    creator_name: undefined
  } as ProcessReq
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入流程名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  form_design_id: [
    { required: true, message: '请选择关联表单', trigger: 'change' }
  ]
};

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  form: {
    name: '',
    id: 0
  } as CloneProcessReq
});

// 详情对话框
const detailDialog = reactive({
  visible: false,
  process: null as Process | null
});

// 初始化加载数据
const loadProcesses = async () => {
  loading.value = true;
  try {
    const params: ListProcessReq = {
      page: currentPage.value,
      size: pageSize.value,
      search: searchQuery.value || undefined,
      status: statusFilter.value || undefined
    };
    
    const res = await listProcess(params);
    if (res) {
      processList.value = res.items || [];
      total.value = res.total || 0;
      
      // 更新统计数据
      stats.total = res.total || 0;
      stats.published = processList.value.filter((p: any) => p.status === 1).length;
      stats.draft = processList.value.filter((p: any) => p.status === 0).length;
      stats.disabled = processList.value.filter((p: any) => p.status === 2).length;
    }
  } catch (error) {
    message.error('加载流程数据失败');
    console.error('Failed to load processes:', error);
  } finally {
    loading.value = false;
  }
};

// 加载表单列表
const loadForms = async () => {
  try {
    const params: ListFormDesignReq = {
      page: 1,
      size: 100,
      status: 1 // 只获取已发布的表单
    };
    
    const res = await listFormDesign(params);
    if (res) {
      forms.value = res.items || [];
    }
  } catch (error) {
    message.error('加载表单数据失败');
    console.error('Failed to load forms:', error);
  }
};

// 方法
const handleSizeChange = (current: number, size: number) => {
  pageSize.value = size;
  currentPage.value = 1;
  loadProcesses();
};

const handleCurrentChange = (page: number) => {
  currentPage.value = page;
  loadProcesses();
};

const handleSearch = () => {
  currentPage.value = 1;
  loadProcesses();
};

const handleStatusChange = () => {
  currentPage.value = 1;
  loadProcesses();
};

const handleCreateProcess = () => {
  processDialog.isEdit = false;
  processDialog.form = {
    name: '',
    description: '',
    form_design_id: undefined as unknown as number,
    definition: {
      steps: [
        { step: '开始', role: '系统', action: 'start' }
      ]
    },
    status: 0
  };
  processDialog.visible = true;
};

const parsedDefinition = (process: Process): Step[] => {
  try {
    if (typeof process.definition === 'string') {
      return JSON.parse(process.definition).steps || [];
    }
    return [];
  } catch (error) {
    console.error('解析流程定义失败:', error);
    return [];
  }
};

const handleEditProcess = async (row: Process) => {
  processDialog.isEdit = true;
  loading.value = true;
  
  try {
    const res = await detailProcess({ id: row.id });
    if (res) {
      const process = res;
      
      processDialog.form = {
        id: process.id,
        name: process.name,
        description: process.description,
        form_design_id: process.form_design_id,
        definition: typeof process.definition === 'string' 
          ? JSON.parse(process.definition) 
          : { steps: [] },
        version: process.version,
        status: process.status,
        category_id: process.category_id,
        creator_id: process.creator_id,
        creator_name: process.creator_name
      };
      
      processDialog.visible = true;
      detailDialog.visible = false;
    }
  } catch (error) {
    message.error('获取流程详情失败');
    console.error('Failed to get process details:', error);
  } finally {
    loading.value = false;
  }
};

const handleViewProcess = async (row: Process) => {
  loading.value = true;
  
  try {
    const res = await detailProcess({ id: row.id });
    if (res) {
      detailDialog.process = res;
      detailDialog.visible = true;
    }
  } catch (error) {
    message.error('获取流程详情失败');
    console.error('Failed to get process details:', error);
  } finally {
    loading.value = false;
  }
};

const handleCommand = async (command: string, row: Process) => {
  switch (command) {
    case 'publish':
      await publishProcess(row);
      break;
    case 'unpublish':
      // 后端可能没有取消发布的接口，这里可以通过更新状态实现
      await updateProcessStatus(row, 0);
      break;
    case 'clone':
      showCloneDialog(row);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

const publishProcess = async (process: Process) => {
  try {
    const params: PublishProcessReq = {
      id: process.id
    };
    
    const res = await publishProcessApi(params);
    if (res) {
      message.success(`流程 "${process.name}" 已发布`);
      loadProcesses();
    }
  } catch (error) {
    message.error('发布流程失败');
    console.error('Failed to publish process:', error);
  }
};

const updateProcessStatus = async (process: Process, status: number) => {
  try {
    const updatedProcess: ProcessReq = {
      id: process.id,
      name: process.name,
      description: process.description,
      form_design_id: process.form_design_id,
      definition: typeof process.definition === 'string' 
        ? JSON.parse(process.definition) 
        : { steps: [] },
      status: status
    };
    
    const res = await updateProcess(updatedProcess);
    if (res) {
      message.success(`流程 "${process.name}" 状态已更新`);
      loadProcesses();
    }
  } catch (error) {
    message.error('更新流程状态失败');
    console.error('Failed to update process status:', error);
  }
};

const showCloneDialog = (process: Process) => {
  cloneDialog.form = {
    id: process.id,
    name: `${process.name} 的副本`
  };
  cloneDialog.visible = true;
};

const confirmClone = async () => {
  try {
    const res = await cloneProcessApi(cloneDialog.form);
    if (res) {
      message.success(`流程已克隆为 "${cloneDialog.form.name}"`);
      cloneDialog.visible = false;
      loadProcesses();
    }
  } catch (error) {
    message.error('克隆流程失败');
    console.error('Failed to clone process:', error);
  }
};

const confirmDelete = (process: Process) => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除流程 "${process.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        const params: DeleteProcessReqReq = {
          id: process.id
        };
        
        const res = await deleteProcess(params);
        if (res) {
          message.success(`流程 "${process.name}" 已删除`);
          loadProcesses();
        }
      } catch (error) {
        message.error('删除流程失败');
        console.error('Failed to delete process:', error);
      }
    }
  });
};

const addNode = () => {
  processDialog.form.definition.steps.push({
    step: '',
    role: '',
    action: ''
  });
};

const removeNode = (index: number) => {
  processDialog.form.definition.steps.splice(index, 1);
};

const saveProcess = async () => {
  try {
    if (processDialog.form.name.trim() === '') {
      message.error('流程名称不能为空');
      return;
    }

    if (!processDialog.form.form_design_id) {
      message.error('请选择关联表单');
      return;
    }

    if (processDialog.form.definition.steps.length === 0) {
      message.error('流程至少需要一个节点');
      return;
    }

    // 验证流程节点是否有效
    for (let i = 0; i < processDialog.form.definition.steps.length; i++) {
      const step = processDialog.form.definition.steps[i];
      if (!step?.step) {
        message.error(`节点 ${i + 1} 名称不能为空`);
        return;
      }
      if (!step?.role) {
        message.error(`节点 ${i + 1} 角色不能为空`);
        return;
      }
      if (!step?.action) {
        message.error(`节点 ${i + 1} 动作不能为空`);
        return;
      }
    }

    if (processDialog.isEdit) {
      // 更新现有流程
      const res = await updateProcess(processDialog.form);
      if (res) {
        message.success(`流程 "${processDialog.form.name}" 已更新`);
      }
    } else {
      // 创建新流程
      const res = await createProcess(processDialog.form);
      if (res) {
        message.success(`流程 "${processDialog.form.name}" 已创建`);
      }
    }
    
    processDialog.visible = false;
    loadProcesses();
  } catch (error) {
    message.error(processDialog.isEdit ? '更新流程失败' : '创建流程失败');
    console.error('Failed to save process:', error);
  }
};

// 辅助方法
const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (dateStr: string | undefined) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (dateStr: string | undefined) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const getInitials = (name: string | undefined) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: number) => {
  switch (status) {
    case 0: return 'status-draft';
    case 1: return 'status-published';
    case 2: return 'status-disabled';
    default: return '';
  }
};

const getAvatarColor = (name: string | undefined) => {
  if (!name) return '#1890ff';
  
  // 根据名称生成一致的颜色
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];

  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length];
};

const getFormName = (formId: number) => {
  const form = forms.value.find((f: any) => f.id === formId);
  return form ? form.name : '未知表单';
};

const getNodeTypeClass = (action: string) => {
  // 根据action映射到UI展示类型
  const map: Record<string, string> = {
    'start': 'start',
    'approve': 'approval',
    'notify': 'notice',
    'condition': 'condition',
    'end': 'end',
    'review': 'approval'
  };
  return map[action] || 'approval';
};

const getNodeTypeName = (action: string) => {
  // 根据action映射到展示的节点类型名称
  const typeMap: Record<string, string> = {
    'start': '开始',
    'approve': '审批',
    'notify': '通知',
    'condition': '条件',
    'end': '结束',
    'review': '审核'
  };
  return typeMap[action] || action;
};

// 加载数据
onMounted(() => {
  loadForms();
  loadProcesses();
});
</script>

<style scoped>
.process-container {
  padding: 24px;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.btn-create {
  background: linear-gradient(135deg, #1890ff 0%);
  border: none;
}

.stats-row {
  margin-bottom: 24px;
}

.stats-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  height: 100%;
}

.table-container {
  margin-bottom: 24px;
}

.process-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.process-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-draft {
  background-color: #faad14;
}

.status-published {
  background-color: #52c41a;
}

.status-disabled {
  background-color: #d9d9d9;
}

.process-name-text {
  font-weight: 500;
}

.description-text {
  color: #606266;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.creator-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.creator-name {
  font-size: 14px;
}

.date-info {
  display: flex;
  flex-direction: column;
}

.date {
  font-weight: 500;
  font-size: 14px;
}

.time {
  font-size: 12px;
  color: #8c8c8c;
}

.action-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.nodes-editor {
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 20px;
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.add-node-button {
  text-align: center;
  margin-top: 16px;
}

.detail-dialog .process-details {
  margin-bottom: 20px;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.detail-header h2 {
  margin: 0;
  font-size: 24px;
  color: #1f2937;
}

.process-preview {
  margin-top: 24px;
}

.process-preview h3 {
  margin-bottom: 16px;
  color: #1f2937;
  font-size: 18px;
}

.process-flow-chart {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.process-node {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 4px;
  transition: all 0.3s;
  position: relative;
}

.node-type-start {
  background-color: #e6f7ff;
  border-color: #91d5ff;
}

.node-type-approval {
  background-color: #f6ffed;
  border-color: #b7eb8f;
}

.node-type-notice {
  background-color: #fefce6;
  border-color: #ffe58f;
}

.node-type-condition {
  background-color: #fff7e6;
  border-color: #ffd591;
}

.node-type-end {
  background-color: #f9f0ff;
  border-color: #d3adf7;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.node-type-badge {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: #fff;
  background-color: #1890ff;
}

.node-type-start .node-type-badge {
  background-color: #1890ff;
}

.node-type-approval .node-type-badge {
  background-color: #52c41a;
}

.node-type-notice .node-type-badge {
  background-color: #faad14;
}

.node-type-condition .node-type-badge {
  background-color: #fa8c16;
}

.node-type-end .node-type-badge {
  background-color: #722ed1;
}

.node-name {
  font-weight: bold;
  font-size: 16px;
}

.node-content {
  margin-bottom: 12px;
}

.node-role, .node-action {
  margin-bottom: 8px;
}

.node-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 8px;
  color: #8c8c8c;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>