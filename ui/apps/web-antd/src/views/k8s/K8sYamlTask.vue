<template>
  <div class="service-manager yaml-task-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">
        <FileProtectOutlined class="dashboard-icon" />
        Kubernetes YAML 任务管理器
      </h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ tasks.length }}</div>
          <div class="stat-label">任务总数</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ templates.length }}</div>
          <div class="stat-label">可用模板</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ clusters.length }}</div>
          <div class="stat-label">集群连接</div>
        </div>
      </div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="control-panel">
      <div class="search-filters">
        <a-input-search
          v-model:value="searchText"
          placeholder="搜索任务名称"
          class="control-item search-input"
          @search="onSearch"
          allow-clear
        >
          <template #prefix><SearchOutlined /></template>
        </a-input-search>
      </div>
      
      <div class="action-buttons">
        <a-tooltip title="刷新数据">
          <a-button type="primary" class="refresh-btn" @click="getTasks" :loading="loading">
            <template #icon><ReloadOutlined /></template>
          </a-button>
        </a-tooltip>
        
        <a-button 
          type="primary" 
          class="create-btn" 
          @click="showCreateModal"
        >
          <template #icon><PlusOutlined /></template>
          新建任务
        </a-button>
      </div>
    </div>

    <!-- 状态摘要卡片 -->
    <div class="status-summary">
      <div class="summary-card total-card">
        <div class="card-content">
          <div class="card-metric">
            <FileProtectOutlined class="metric-icon" />
            <div class="metric-value">{{ tasks.length }}</div>
          </div>
          <div class="card-title">任务总数</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">管理您的全部YAML任务</div>
        </div>
      </div>
      
      <div class="summary-card running-card">
        <div class="card-content">
          <div class="card-metric">
            <FileOutlined class="metric-icon" />
            <div class="metric-value">{{ templates.length }}</div>
          </div>
          <div class="card-title">可用模板</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">部署模板库</div>
        </div>
      </div>
      
      <div class="summary-card env-card">
        <div class="card-content">
          <div class="card-metric">
            <CloudServerOutlined class="metric-icon" />
            <div class="metric-value">{{ clusters.length }}</div>
          </div>
          <div class="card-title">集群连接</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">{{ clusters.length > 0 ? '连接正常' : '等待连接' }}</div>
        </div>
      </div>
    </div>

    <!-- 视图切换 -->
    <div class="view-toggle">
      <a-radio-group v-model:value="viewMode" button-style="solid">
        <a-radio-button value="table">
          <UnorderedListOutlined />
          表格视图
        </a-radio-button>
        <a-radio-button value="card">
          <AppstoreOutlined />
          卡片视图
        </a-radio-button>
      </a-radio-group>
    </div>

    <!-- 表格视图 -->
    <a-table
      v-if="viewMode === 'table'"
      :columns="columns"
      :data-source="filteredTasks"
      :loading="loading"
      row-key="id"
      :pagination="{ 
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条数据`
      }"
      class="services-table yaml-task-table"
    >
      <!-- 任务名称列 -->
      <template #name="{ text }">
        <div class="task-name">
          <FileProtectOutlined />
          <span class="task-title">{{ text }}</span>
        </div>
      </template>
      
      <!-- 创建时间列 -->
      <template #created_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatDateTime(text)">
            {{ formatDate(text) }}
          </a-tooltip>
        </div>
      </template>

      <!-- 更新时间列 -->
      <template #updated_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatDateTime(text)">
            {{ formatDate(text) }}
          </a-tooltip>
        </div>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <div class="action-column">
          <a-tooltip title="应用任务">
            <a-button type="primary" ghost shape="circle" @click="handleApply(record)">
              <template #icon><PlayCircleOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="编辑任务">
            <a-button type="primary" ghost shape="circle" @click="handleEdit(record)">
              <template #icon><EditOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="删除任务">
            <a-popconfirm
              title="确定要删除该任务吗?"
              description="此操作不可撤销"
              @confirm="handleDelete(record)"
              ok-text="确定"
              cancel-text="取消"
            >
              <a-button type="primary" danger ghost shape="circle">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </a-popconfirm>
          </a-tooltip>
        </div>
      </template>

      <!-- 空状态 -->
      <template #emptyText>
        <div class="empty-state">
          <FileProtectOutlined style="font-size: 48px; color: #d9d9d9; margin-bottom: 16px" />
          <p>暂无任务数据</p>
          <a-button type="primary" @click="showCreateModal">创建第一个任务</a-button>
        </div>
      </template>
    </a-table>

    <!-- 卡片视图 -->
    <div v-else class="card-view">
      <a-spin :spinning="loading">
        <a-empty v-if="filteredTasks.length === 0" description="暂无任务数据" />
        <div v-else class="service-cards yaml-task-cards">
          <div v-for="task in filteredTasks" :key="task.id" class="service-card yaml-task-card">
            <div class="card-header">
              <div class="service-title task-title">
                <FileProtectOutlined class="service-icon" />
                <h3>{{ task.name }}</h3>
              </div>
              <a-tag color="blue" class="card-type-tag">
                <span class="status-dot"></span>
                YAML任务
              </a-tag>
            </div>
            
            <div class="card-content">
              <div class="card-detail template-detail">
                <span class="detail-label">模板:</span>
                <span class="detail-value">
                  <FileOutlined />
                  {{ getTemplateName(task.template_id) }}
                </span>
              </div>
              <div class="card-detail cluster-detail">
                <span class="detail-label">集群:</span>
                <span class="detail-value">
                  <CloudServerOutlined />
                  {{ getClusterName(task.cluster_id) }}
                </span>
              </div>
              <div class="card-detail variables-detail">
                <span class="detail-label">变量:</span>
                <span class="detail-value">
                  <CodeOutlined />
                  {{ task.variables?.length || 0 }} 个
                </span>
              </div>
              <div class="card-detail created-detail">
                <span class="detail-label">创建时间:</span>
                <span class="detail-value">
                  <ClockCircleOutlined />
                  {{ formatDate(task.created_at) }}
                </span>
              </div>
            </div>
            
            <div class="card-footer card-action-footer">
              <a-button type="primary" ghost size="small" @click="handleApply(task)">
                <template #icon><PlayCircleOutlined /></template>
                应用
              </a-button>
              <a-button type="primary" ghost size="small" @click="handleEdit(task)">
                <template #icon><EditOutlined /></template>
                编辑
              </a-button>
              <a-popconfirm
                title="确定要删除该任务吗?"
                @confirm="handleDelete(task)"
                ok-text="确定"
                cancel-text="取消"
              >
                <a-button type="primary" danger ghost size="small">
                  <template #icon><DeleteOutlined /></template>
                  删除
                </a-button>
              </a-popconfirm>
            </div>
          </div>
        </div>
      </a-spin>
    </div>

    <!-- 创建/编辑任务模态框 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑任务' : '创建任务'"
      @ok="handleSubmit"
      @cancel="closeModal"
      width="900px"
      :okText="isEdit ? '保存更改' : '创建任务'"
      :confirmLoading="submitLoading"
      class="yaml-task-modal"
    >
      <a-alert type="info" show-icon class="modal-alert">
        <template #message>{{ isEdit ? '编辑YAML任务' : '创建YAML任务' }}</template>
        <template #description>{{ isEdit ? '修改任务配置和变量信息' : '请配置任务的基本信息和部署参数' }}</template>
      </a-alert>
      
      <a-form :model="formState" layout="vertical" class="yaml-task-form" ref="formRef">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-form-item label="任务名称" name="name" :rules="rules.name">
            <a-input v-model:value="formState.name" placeholder="请输入任务名称" class="form-input">
              <template #prefix><FileProtectOutlined /></template>
            </a-input>
          </a-form-item>
        </div>
        
        <div class="form-section">
          <div class="section-title">配置选择</div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="选择模板" name="template_id" :rules="rules.template_id">
                <a-select
                  v-model:value="formState.template_id"
                  placeholder="请选择模板"
                  class="form-select"
                >
                  <template #suffixIcon><FileOutlined /></template>
                  <a-select-option v-for="template in templates" :key="template.id" :value="template.id">
                    {{ template.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="选择集群" name="cluster_id" :rules="rules.cluster_id">
                <a-select
                  v-model:value="formState.cluster_id"
                  placeholder="请选择集群"
                  class="form-select"
                >
                  <template #suffixIcon><CloudServerOutlined /></template>
                  <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
                    {{ cluster.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>
        
        <div class="form-section">
          <div class="section-header">
            <div class="section-title">变量配置</div>
            <a-button type="primary" ghost @click="addVariable" class="add-variable-btn">
              <PlusOutlined />
              添加变量
            </a-button>
          </div>
          
          <div class="variables-area">
            <a-empty v-if="!formState.variables?.length" class="variables-empty">
              <template #image>
                <CodeOutlined style="font-size: 48px; color: #d9d9d9;" />
              </template>
              <template #description>
                <span>暂无变量，点击上方按钮添加变量</span>
              </template>
            </a-empty>
            
            <div v-else class="variables-list">
              <div v-for="(variable, index) in formState.variables" :key="index" class="variable-row">
                <div class="variable-number">{{ index + 1 }}</div>
                <a-input
                  v-model:value="formState.variables[index]"
                  placeholder="key=value"
                  class="variable-input"
                >
                  <template #prefix>
                    <CodeOutlined />
                  </template>
                </a-input>
                <a-button 
                  type="text" 
                  danger 
                  @click="removeVariable(index)" 
                  class="remove-variable-btn"
                >
                  <DeleteOutlined />
                </a-button>
              </div>
            </div>
          </div>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';
import {
  SearchOutlined,
  PlusOutlined,
  FileOutlined,
  FileProtectOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  CodeOutlined,
  ClockCircleOutlined,
  UnorderedListOutlined
} from '@ant-design/icons-vue';
import {
  getYamlTaskListApi,
  createYamlTaskApi,
  updateYamlTaskApi,
  deleteYamlTaskApi,
  applyYamlTaskApi,
  getAllClustersApi,
  getYamlTemplateApi,
} from '#/api';

// 类型定义
interface YamlTask {
  id: number;
  name: string;
  template_id: number;
  cluster_id: number;
  variables: string[];
  created_at?: string;
  updated_at?: string;
}

interface ClusterItem {
  id: number;
  name: string;
}

interface TemplateItem {
  id: number;
  name: string;
}

// 状态变量
const loading = ref(false);
const submitLoading = ref(false);
const tasks = ref<YamlTask[]>([]);
const searchText = ref('');
const modalVisible = ref(false);
const isEdit = ref(false);
const formRef = ref<FormInstance>();
const clusters = ref<ClusterItem[]>([]);
const templates = ref<TemplateItem[]>([]);
const viewMode = ref<'table' | 'card'>('table');

// 表单状态
const formState = reactive<Partial<YamlTask>>({
  name: '',
  template_id: undefined,
  cluster_id: undefined,
  variables: [],
});

// 表单校验规则
const rules = {
  name: [
    { required: true, message: '请输入任务名称', trigger: 'blur' },
    { min: 2, max: 50, message: '任务名称长度应为2-50个字符', trigger: 'blur' }
  ],
  template_id: [{ required: true, message: '请选择模板', trigger: 'change' }],
  cluster_id: [{ required: true, message: '请选择集群', trigger: 'change' }],
};

// 表格列配置
const columns = [
  {
    title: '任务名称',
    dataIndex: 'name',
    key: 'name',
    width: '30%',
    sorter: (a: YamlTask, b: YamlTask) => a.name.localeCompare(b.name),
    slots: { customRender: 'name' },
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: '25%',
    sorter: (a: YamlTask, b: YamlTask) => {
      if (!a.created_at || !b.created_at) return 0;
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
    },
    slots: { customRender: 'created_at' },
  },
  {
    title: '更新时间',
    dataIndex: 'updated_at',
    key: 'updated_at',
    width: '25%',
    sorter: (a: YamlTask, b: YamlTask) => {
      if (!a.updated_at || !b.updated_at) return 0;
      return new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
    },
    slots: { customRender: 'updated_at' },
  },
  {
    title: '操作',
    key: 'action',
    width: '20%',
    fixed: 'right',
    slots: { customRender: 'action' },
  },
];

// 计算属性：过滤后的任务列表
const filteredTasks = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  if (!searchValue) return tasks.value;
  return tasks.value.filter(task => task.name.toLowerCase().includes(searchValue));
});

// 日期格式化函数
const formatDate = (dateString?: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const formatDateTime = (dateString?: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
};

// 根据ID获取模板名称
const getTemplateName = (id?: number): string => {
  if (!id) return '-';
  const template = templates.value.find(t => t.id === id);
  return template ? template.name : '-';
};

// 根据ID获取集群名称
const getClusterName = (id?: number): string => {
  if (!id) return '-';
  const cluster = clusters.value.find(c => c.id === id);
  return cluster ? cluster.name : '-';
};

// 搜索处理
const onSearch = () => {
  // 搜索逻辑已在计算属性中实现
};

// 获取集群列表
const getClusters = async () => {
  try {
    const res = await getAllClustersApi();
    clusters.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取集群列表失败');
  }
};

// 获取模板列表
const getTemplates = async () => {
  try {
    const firstCluster = clusters.value[0];
    if (firstCluster) {
      const res = await getYamlTemplateApi(firstCluster.id);
      templates.value = res || [];
    }
  } catch (error: any) {
    message.error(error.message || '获取模板列表失败');
  }
};

// 获取任务列表
const getTasks = async () => {
  loading.value = true;
  try {
    const res = await getYamlTaskListApi();
    tasks.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取任务列表失败');
  } finally {
    loading.value = false;
  }
};

// 显示创建模态框
const showCreateModal = () => {
  isEdit.value = false;
  Object.assign(formState, {
    name: '',
    template_id: undefined,
    cluster_id: undefined,
    variables: [],
  });
  modalVisible.value = true;
};

// 显示编辑模态框
const handleEdit = (record: YamlTask) => {
  isEdit.value = true;
  Object.assign(formState, {
    id: record.id,
    name: record.name,
    template_id: record.template_id,
    cluster_id: record.cluster_id,
    variables: [...record.variables],
  });
  modalVisible.value = true;
};

// 添加变量
const addVariable = () => {
  if (!formState.variables) {
    formState.variables = [];
  }
  formState.variables.push('');
};

// 删除变量
const removeVariable = (index: number) => {
  formState.variables?.splice(index, 1);
};

// 应用任务
const handleApply = async (record: YamlTask) => {
  const hide = message.loading('正在应用任务...', 0);
  try {
    await applyYamlTaskApi(record.id);
    hide();
    message.success('任务应用成功');
  } catch (error: any) {
    hide();
    message.error(error.message || '任务应用失败');
  }
};

// 提交表单
const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    
    const variables = formState.variables?.filter(v => v.trim()) || [];
    submitLoading.value = true;
    
    if (isEdit.value) {
      await updateYamlTaskApi({
        id: formState.id!,
        name: formState.name!,
        template_id: formState.template_id!,
        cluster_id: formState.cluster_id!,
        variables,
      });
      message.success('任务更新成功');
    } else {
      await createYamlTaskApi({
        name: formState.name!,
        template_id: formState.template_id!,
        cluster_id: formState.cluster_id!,
        variables,
      });
      message.success('任务创建成功');
    }
    
    modalVisible.value = false;
    getTasks();
  } catch (error: any) {
    message.error(error.message || (isEdit.value ? '更新任务失败' : '创建任务失败'));
  } finally {
    submitLoading.value = false;
  }
};

// 删除任务
const handleDelete = async (task: YamlTask) => {
  const hide = message.loading('正在删除任务...', 0);
  try {
    await deleteYamlTaskApi(task.id);
    hide();
    message.success('删除成功');
    getTasks();
  } catch (error: any) {
    hide();
    message.error(error.message || '删除失败');
  }
};

// 关闭模态框
const closeModal = () => {
  modalVisible.value = false;
};

// 页面加载时获取数据
onMounted(async () => {
  await getClusters();
  await getTemplates();
  await getTasks();
});
</script>

<style>
/* 继承集群管理页面的基础样式 */
:root {
  --primary-color: #1890ff;
  --success-color: #52c41a;
  --warning-color: #faad14;
  --error-color: #f5222d;
  --font-size-base: 14px;
  --border-radius-base: 4px;
  --box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  --transition-duration: 0.3s;
}

.yaml-task-manager {
  background-color: #f0f2f5;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

/* 仪表板标题样式 */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
  padding-bottom: 20px;
  border-bottom: 1px solid #f0f0f0;
}

.dashboard-title {
  font-size: 24px;
  font-weight: 600;
  color: #262626;
  margin: 0;
  display: flex;
  align-items: center;
}

.dashboard-icon {
  margin-right: 14px;
  font-size: 28px;
  color: #1890ff;
}

.dashboard-stats {
  display: flex;
  gap: 20px;
}

.stat-item {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border-radius: 8px;
  padding: 10px 18px;
  color: white;
  min-width: 120px;
  text-align: center;
  box-shadow: 0 3px 8px rgba(24, 144, 255, 0.2);
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.3;
}

.stat-label {
  font-size: 12px;
  opacity: 0.9;
  margin-top: 4px;
}

/* 控制面板样式 */
.control-panel {
  display: flex;
  justify-content: space-between;
  margin-bottom: 24px;
  background: white;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.search-filters {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  align-items: center;
  flex: 1;
}

.control-item {
  min-width: 200px;
}

.search-input {
  flex-grow: 1;
  max-width: 300px;
}

.action-buttons {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-left: 20px;
}

.refresh-btn {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border: none;
  height: 36px;
  width: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.create-btn {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border: none;
  height: 36px;
  padding: 0 16px;
  font-weight: 500;
}

/* 状态摘要卡片 */
.status-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 20px;
  margin-bottom: 28px;
}

.summary-card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  transition: transform 0.3s, box-shadow 0.3s;
  display: flex;
  flex-direction: column;
}

.summary-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.1);
}

.card-content {
  padding: 24px;
  flex-grow: 1;
}

.card-title {
  font-size: 14px;
  color: #8c8c8c;
  margin-top: 10px;
}

.card-metric {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.metric-icon {
  font-size: 28px;
  margin-right: 16px;
}

.metric-value {
  font-size: 32px;
  font-weight: 600;
  color: #262626;
}

.total-card .metric-icon {
  color: #1890ff;
}

.running-card .metric-icon {
  color: #52c41a;
}

.env-card .metric-icon {
  color: #722ed1;
}

.card-footer {
  padding: 14px 24px;
  background-color: #fafafa;
  border-top: 1px solid #f0f0f0;
}

.footer-text {
  font-size: 12px;
  color: #8c8c8c;
  margin-top: 6px;
}

/* 视图切换按钮 */
.view-toggle {
  margin-bottom: 20px;
  text-align: right;
}

.view-toggle :deep(.ant-radio-button-wrapper) {
  padding: 0 16px;
  height: 36px;
  line-height: 34px;
  display: inline-flex;
  align-items: center;
}

.view-toggle :deep(.ant-radio-button-wrapper svg) {
  margin-right: 6px;
}

/* YAML任务表格样式 */
.yaml-task-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.yaml-task-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.yaml-task-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.task-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.task-title {
  color: #1890ff;
}

.timestamp {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #595959;
}

.action-column {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.action-column :deep(.ant-btn) {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0;
}

/* 卡片视图容器 */
.card-view {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

/* YAML任务卡片样式 */
.yaml-task-cards {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

.yaml-task-card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
  transition: transform 0.3s, box-shadow 0.3s;
  overflow: hidden;
  position: relative;
  display: flex;
  flex-direction: column;
  width: 350px;
  border: 1px solid #eaeaea;
  margin-bottom: 20px;
}

.yaml-task-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
  position: relative;
}

.service-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 45px;
}

.service-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #333;
  word-break: break-all;
  line-height: 1.4;
}

.service-icon {
  font-size: 20px;
  color: #1890ff;
}

.card-type-tag {
  position: absolute;
  top: 12px;
  right: 12px;
  padding: 2px 10px;
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}

.card-content {
  padding: 20px;
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: #fff;
}

.card-detail {
  display: flex;
  align-items: center;
  line-height: 1.5;
}

.detail-label {
  color: #666;
  min-width: 100px;
  font-size: 14px;
}

.detail-value {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #333;
  flex: 1;
}

.card-action-footer {
  padding: 16px 20px;
  background-color: #f5f7fa;
  border-top: 1px solid #eeeeee;
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.card-action-footer .ant-btn {
  flex: 1;
  min-width: 80px;
  border-radius: 4px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-action-footer .ant-btn svg {
  margin-right: 8px;
}

/* YAML任务模态框样式 */
.yaml-task-modal {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
}

.modal-alert {
  margin-bottom: 16px;
}

.yaml-task-form {
  padding: 10px;
}

.form-section {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid #f1f5f9;
}

.form-section:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-title::before {
  content: '';
  width: 4px;
  height: 20px;
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border-radius: 2px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.form-input,
.form-select {
  border-radius: 8px;
  height: 42px;
}

.add-variable-btn {
  border-radius: 10px;
  height: 40px;
  font-weight: 500;
}

.variables-area {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
}

.variables-empty {
  padding: 40px 0;
  margin: 0;
}

.variables-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.variable-row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: white;
  border-radius: 10px;
  padding: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.variable-number {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
}

.variable-input {
  flex: 1;
  height: 40px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.remove-variable-btn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
}

.remove-variable-btn:hover {
  background: #fef2f2;
  color: #ef4444;
}

/* 空状态样式 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 0;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .yaml-task-cards {
    justify-content: space-around;
  }
  
  .yaml-task-card {
    width: 320px;
  }
}

@media (max-width: 768px) {
  .yaml-task-cards {
    flex-direction: column;
    align-items: center;
  }
  
  .yaml-task-card {
    width: 100%;
    max-width: 450px;
  }
  
  .card-action-footer {
    flex-wrap: wrap;
  }
  
  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .dashboard-stats {
    margin-top: 16px;
    width: 100%;
  }
  
  .control-panel {
    flex-direction: column;
  }
  
  .search-filters {
    margin-bottom: 16px;
  }
  
  .action-buttons {
    margin-left: 0;
    justify-content: flex-end;
  }
}
</style>