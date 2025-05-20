<template>
  <div class="service-manager task-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">
        <FileProtectOutlined class="dashboard-icon" />
        Kubernetes YAML 任务管理器
      </h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ tasks.length }}</div>
          <div class="stat-label">任务</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ templates.length }}</div>
          <div class="stat-label">可用模板</div>
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
        
        <a-button type="primary" class="create-btn" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          创建任务
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
          <div class="footer-text">已配置YAML任务</div>
        </div>
      </div>
      
      <div class="summary-card template-card">
        <div class="card-content">
          <div class="card-metric">
            <FileOutlined class="metric-icon" />
            <div class="metric-value">{{ templates.length }}</div>
          </div>
          <div class="card-title">可用模板</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="templates.length > 0 ? 100 : 0" 
            :stroke-color="{ from: '#1890ff', to: '#52c41a' }" 
            size="small" 
            :show-info="false" 
          />
          <div class="footer-text">模板可直接使用</div>
        </div>
      </div>
      
      <div class="summary-card cluster-card">
        <div class="card-content">
          <div class="card-metric">
            <CloudServerOutlined class="metric-icon" />
            <div class="metric-value">{{ clusters.length }}</div>
          </div>
          <div class="card-title">可用集群</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="clusters.length > 0 ? 100 : 0" 
            :status="clusters.length > 0 ? 'success' : 'exception'" 
            size="small" 
            :show-info="false"
          />
          <div class="footer-text">{{ clusters.length > 0 ? '集群连接正常' : '无可用集群' }}</div>
        </div>
      </div>
      
      <div class="summary-card system-card">
        <div class="card-content">
          <div class="card-metric">
            <AppstoreOutlined class="metric-icon" />
            <div class="metric-value">运行中</div>
          </div>
          <div class="card-title">系统状态</div>
        </div>
        <div class="card-footer">
          <div class="system-status">
            <span class="status-indicator"></span>
            <span class="status-text">系统在线</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 视图切换 -->
    <div class="view-toggle">
      <a-radio-group v-model:value="viewMode" button-style="solid">
        <a-radio-button value="table">
          <TableOutlined />
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
      class="services-table task-table"
    >
      <!-- 任务名称列 -->
      <template #name="{ text }">
        <div class="task-name">
          <FileProtectOutlined />
          <span>{{ text }}</span>
        </div>
      </template>
      
      <!-- 创建时间列 -->
      <template #created_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatFullDate(text)">
            <span>{{ formatDate(text) }}</span>
          </a-tooltip>
        </div>
      </template>

      <!-- 更新时间列 -->
      <template #updated_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatFullDate(text)">
            <span>{{ formatDate(text) }}</span>
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
          <InboxOutlined style="font-size: 48px; color: #d9d9d9; margin-bottom: 16px" />
          <p>暂无任务数据</p>
          <a-button type="primary" @click="showCreateModal">创建第一个任务</a-button>
        </div>
      </template>
    </a-table>

    <!-- 卡片视图 -->
    <div v-else class="card-view">
      <a-spin :spinning="loading">
        <a-empty v-if="filteredTasks.length === 0" description="暂无任务数据">
          <template #extra>
            <a-button type="primary" @click="showCreateModal">创建第一个任务</a-button>
          </template>
        </a-empty>
        <div v-else class="service-cards task-cards">
          <div v-for="task in filteredTasks" :key="task.id" class="service-card task-card">
            <div class="card-header">
              <div class="service-title task-title">
                <FileProtectOutlined class="service-icon" />
                <h3>{{ task.name }}</h3>
              </div>
            </div>
            
            <div class="card-content">
              <div class="card-detail created-detail">
                <span class="detail-label">创建时间:</span>
                <span class="detail-value">
                  <ClockCircleOutlined />
                  <a-tooltip :title="formatFullDate(task.created_at)">
                    {{ formatDate(task.created_at) }}
                  </a-tooltip>
                </span>
              </div>
              <div class="card-detail updated-detail">
                <span class="detail-label">更新时间:</span>
                <span class="detail-value">
                  <ClockCircleOutlined />
                  <a-tooltip :title="formatFullDate(task.updated_at)">
                    {{ formatDate(task.updated_at) }}
                  </a-tooltip>
                </span>
              </div>
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
                <span class="detail-label">变量数量:</span>
                <span class="detail-value">{{ task.variables?.length || 0 }}</span>
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
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑任务' : '创建任务'"
      @ok="handleSubmit"
      width="800px"
      :okText="isEdit ? '保存更改' : '创建任务'"
      :maskClosable="false"
      class="task-modal"
    >
      <a-form 
        :model="formState" 
        :rules="rules" 
        ref="formRef"
        layout="vertical"
        class="task-form"
      >
        <a-form-item label="任务名称" name="name">
          <a-input 
            v-model:value="formState.name" 
            placeholder="请输入任务名称" 
            class="form-input"
          >
            <template #prefix>
              <FileOutlined />
            </template>
          </a-input>
        </a-form-item>
        
        <div class="form-row">
          <a-form-item label="选择模板" name="template_id" class="form-col">
            <a-select
              v-model:value="formState.template_id"
              placeholder="请选择模板"
              class="form-select"
              :options="templates.map(item => ({
                value: item.id,
                label: item.name
              }))"
            >
              <template #suffixIcon>
                <DownOutlined />
              </template>
            </a-select>
          </a-form-item>
          
          <a-form-item label="选择集群" name="cluster_id" class="form-col">
            <a-select
              v-model:value="formState.cluster_id"
              placeholder="请选择集群"
              class="form-select"
              :options="clusters.map(item => ({
                value: item.id,
                label: item.name
              }))"
            >
              <template #suffixIcon>
                <DownOutlined />
              </template>
            </a-select>
          </a-form-item>
        </div>
        
        <a-form-item label="变量列表" name="variables">
          <div class="variables-header">
            <span class="variables-title">设置任务变量</span>
            <a-button type="primary" ghost @click="addVariable" class="add-var-btn">
              <template #icon><PlusOutlined /></template>
              添加变量
            </a-button>
          </div>
          
          <a-empty v-if="!formState.variables?.length" class="variables-empty">
            <template #description>
              <span>暂无变量，点击上方按钮添加</span>
            </template>
          </a-empty>
          
          <div v-else class="variables-container">
            <div v-for="(variable, index) in formState.variables" :key="index" class="variable-item">
              <a-input
                v-model:value="formState.variables[index]"
                placeholder="key=value"
                class="variable-input"
              >
                <template #prefix>
                  <CodeOutlined />
                </template>
              </a-input>
              <a-button type="text" danger @click="removeVariable(index)" class="remove-var-btn">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </div>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
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
  DownOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  CodeOutlined,
  InboxOutlined,
  TableOutlined,
  ClockCircleOutlined
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

// 状态变量
const loading = ref(false);
const tasks = ref<YamlTask[]>([]);
const searchText = ref('');
const modalVisible = ref(false);
const isEdit = ref(false);
const formRef = ref<FormInstance>();
const clusters = ref<Array<{id: number, name: string}>>([]);
const templates = ref<Array<{id: number, name: string}>>([]);
const viewMode = ref<'table' | 'card'>('table');

const formState = ref<Partial<YamlTask>>({
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
    sorter: (a: YamlTask, b: YamlTask) => new Date(a.created_at || '').getTime() - new Date(b.created_at || '').getTime(),
    slots: { customRender: 'created_at' },
  },
  {
    title: '更新时间',
    dataIndex: 'updated_at',
    key: 'updated_at',
    width: '25%',
    sorter: (a: YamlTask, b: YamlTask) => new Date(a.updated_at || '').getTime() - new Date(b.updated_at || '').getTime(),
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

// 日期格式化函数
const formatDate = (dateString?: string) => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getMonth() + 1}月${date.getDate()}日 ${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const formatFullDate = (dateString?: string) => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
};

// 根据ID获取模板名称
const getTemplateName = (id?: number) => {
  if (!id) return '-';
  const template = templates.value.find(t => t.id === id);
  return template ? template.name : '-';
};

// 根据ID获取集群名称
const getClusterName = (id?: number) => {
  if (!id) return '-';
  const cluster = clusters.value.find(c => c.id === id);
  return cluster ? cluster.name : '-';
};

// 计算属性：过滤后的任务列表
const filteredTasks = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  if (!searchValue) return tasks.value;
  return tasks.value.filter(task => task.name.toLowerCase().includes(searchValue));
});

// 搜索
const onSearch = () => {
  // 搜索逻辑已经在计算属性中实现，这里可以添加其他触发行为
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
    // 这里暂时使用第一个集群的模板列表
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
  formState.value = {
    name: '',
    template_id: undefined,
    cluster_id: undefined,
    variables: [],
  };
  modalVisible.value = true;
};

// 显示编辑模态框
const handleEdit = (record: YamlTask) => {
  isEdit.value = true;
  formState.value = {
    id: record.id,
    name: record.name,
    template_id: record.template_id,
    cluster_id: record.cluster_id,
    variables: [...record.variables],
  };
  modalVisible.value = true;
};

// 添加变量
const addVariable = () => {
  if (!formState.value.variables) {
    formState.value.variables = [];
  }
  formState.value.variables.push('');
};

// 删除变量
const removeVariable = (index: number) => {
  formState.value.variables?.splice(index, 1);
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
    
    // 过滤掉空的变量
    const variables = formState.value.variables?.filter(v => v.trim()) || [];
    const hide = message.loading(isEdit.value ? '正在更新任务...' : '正在创建任务...', 0);
    
    if (isEdit.value) {
      await updateYamlTaskApi({
        id: formState.value.id,
        name: formState.value.name,
        template_id: formState.value.template_id,
        cluster_id: formState.value.cluster_id,
        variables,
      });
      hide();
      message.success('任务更新成功');
    } else {
      await createYamlTaskApi({
        name: formState.value.name,
        template_id: formState.value.template_id,
        cluster_id: formState.value.cluster_id,
        variables,
      });
      hide();
      message.success('任务创建成功');
    }
    
    modalVisible.value = false;
    getTasks();
  } catch (error: any) {
    message.error(error.message || (isEdit.value ? '更新任务失败' : '创建任务失败'));
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

// 页面加载时获取数据
onMounted(async () => {
  await getClusters();
  await getTemplates();
  await getTasks();
});
</script>

<style>
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

.task-manager {
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

.template-card .metric-icon {
  color: #52c41a;
}

.cluster-card .metric-icon {
  color: #722ed1;
}

.system-card .metric-icon {
  color: #fa8c16;
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

.system-status {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #52c41a;
  display: inline-block;
}

.status-text {
  font-size: 13px;
  color: #52c41a;
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

/* 任务表格样式 */
.task-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.task-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.task-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.task-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
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

/* 卡片容器布局优化 - 横向排列 */
.task-cards {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

/* 卡片样式优化 */
.task-card, .service-card {
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

.task-card:hover, .service-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

/* 卡片头部样式 */
.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
  position: relative;
}

.task-title, .service-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 45px;
}

.task-title h3, .service-title h3 {
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

/* 卡片内容区域 */
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

/* 卡片底部按钮区域 */
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

/* 空状态样式 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 0;
}

/* 表单样式 */
.task-form {
  padding: 10px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-input, .form-select {
  border-radius: 8px;
  height: 42px;
}

.variables-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.variables-title {
  font-weight: 500;
  color: #333;
}

.add-var-btn {
  border-radius: 6px;
  height: 36px;
}

.variables-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 8px;
}

.variable-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.variable-input {
  flex: 1;
  border-radius: 6px;
  border: 1px solid #d9d9d9;
  background: white;
  transition: all 0.3s;
}

.variable-input:hover {
  border-color: #40a9ff;
}

.remove-var-btn {
  border-radius: 50%;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
}

.remove-var-btn:hover {
  background: #fff2f0;
  color: #ff4d4f;
}

.variables-empty {
  padding: 24px 0;
  background: #f9f9f9;
  border-radius: 8px;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .task-cards {
    justify-content: space-around;
  }
  
  .task-card, .service-card {
    width: 320px;
  }
}

@media (max-width: 768px) {
  .search-filters {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .control-panel {
    flex-direction: column;
    gap: 20px;
  }
  
  .action-buttons {
    margin-left: 0;
    justify-content: space-between;
    width: 100%;
  }
  
  .form-row {
    grid-template-columns: 1fr;
  }
  
  .task-cards {
    flex-direction: column;
    align-items: center;
  }
  
  .task-card, .service-card {
    width: 100%;
    max-width: 450px;
  }
}
</style>