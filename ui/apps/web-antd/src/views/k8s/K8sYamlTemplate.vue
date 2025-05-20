<template>
  <div class="service-manager template-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">
        <FileTextOutlined class="dashboard-icon" />
        Kubernetes 模板管理器
      </h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ templates.length }}</div>
          <div class="stat-label">模板总数</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ clusters.length }}</div>
          <div class="stat-label">可用集群</div>
        </div>
      </div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="control-panel">
      <div class="search-filters">
        <a-select
          v-model:value="selectedCluster"
          placeholder="选择集群"
          class="control-item cluster-selector"
          @change="handleClusterChange"
        >
          <template #suffixIcon><ClusterOutlined /></template>
          <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
            <span class="cluster-option">
              <CloudServerOutlined />
              {{ cluster.name }}
            </span>
          </a-select-option>
        </a-select>
        
        <a-input-search
          v-model:value="searchText"
          placeholder="搜索模板名称"
          class="control-item search-input"
          @search="onSearch"
          allow-clear
        >
          <template #prefix><SearchOutlined /></template>
        </a-input-search>
      </div>
      
      <div class="action-buttons">
        <a-tooltip title="刷新数据">
          <a-button type="primary" class="refresh-btn" @click="getTemplates" :loading="loading" :disabled="!selectedCluster">
            <template #icon><ReloadOutlined /></template>
          </a-button>
        </a-tooltip>
        
        <a-button 
          type="primary" 
          class="create-btn" 
          @click="showCreateModal"
          :disabled="!selectedCluster"
        >
          <template #icon><PlusOutlined /></template>
          创建模板
        </a-button>
      </div>
    </div>

    <!-- 提示信息 -->
    <a-alert 
      v-if="!selectedCluster" 
      message="请先选择一个集群来管理模板" 
      type="info" 
      show-icon 
      class="cluster-alert"
    />

    <!-- 状态摘要卡片 -->
    <div class="status-summary" v-if="selectedCluster">
      <div class="summary-card total-card">
        <div class="card-content">
          <div class="card-metric">
            <FileTextOutlined class="metric-icon" />
            <div class="metric-value">{{ templates.length }}</div>
          </div>
          <div class="card-title">模板总数</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">全部模板</div>
        </div>
      </div>
      
      <div class="summary-card running-card">
        <div class="card-content">
          <div class="card-metric">
            <ClockCircleOutlined class="metric-icon" />
            <div class="metric-value">{{ lastUpdateTime || '-' }}</div>
          </div>
          <div class="card-title">最近更新</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">上次模板更新时间</div>
        </div>
      </div>
      
      <div class="summary-card cluster-card">
        <div class="card-content">
          <div class="card-metric">
            <ClusterOutlined class="metric-icon" />
            <div class="metric-value cluster-name">{{ selectedClusterName || '未选择' }}</div>
          </div>
          <div class="card-title">当前集群</div>
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
    <div class="view-toggle" v-if="selectedCluster">
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
      v-if="viewMode === 'table' && selectedCluster"
      :columns="columns"
      :data-source="filteredTemplates"
      :loading="loading"
      row-key="id"
      :pagination="{
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条模板`
      }"
      class="services-table template-table"
    >
      <!-- 模板名称列 -->
      <template #name="{ text }">
        <div class="template-name">
          <FileTextOutlined />
          <span>{{ text }}</span>
        </div>
      </template>
      
      <!-- 创建时间列 -->
      <template #created_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatFullDate(text)">
            {{ formatDate(text) }}
          </a-tooltip>
        </div>
      </template>

      <!-- 更新时间列 -->
      <template #updated_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatFullDate(text)">
            {{ formatDate(text) }}
          </a-tooltip>
        </div>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <div class="action-column">
          <a-tooltip title="检查YAML格式">
            <a-button type="primary" ghost shape="circle" @click="handleCheck(record)">
              <template #icon><CheckOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="编辑模板">
            <a-button type="primary" ghost shape="circle" @click="handleEdit(record)">
              <template #icon><EditOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="删除模板">
            <a-popconfirm
              title="确定要删除该模板吗?"
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
          <FileTextOutlined style="font-size: 48px; color: #d9d9d9; margin-bottom: 16px" />
          <p>当前集群暂无模板数据</p>
          <a-button type="primary" @click="showCreateModal">创建第一个模板</a-button>
        </div>
      </template>
    </a-table>

    <!-- 卡片视图 -->
    <div v-else-if="viewMode === 'card' && selectedCluster" class="card-view">
      <a-spin :spinning="loading">
        <a-empty v-if="filteredTemplates.length === 0" description="暂无模板数据" />
        <div v-else class="service-cards template-cards">
          <div v-for="template in filteredTemplates" :key="template.id" class="service-card template-card">
            <div class="card-header">
              <div class="service-title template-title">
                <FileTextOutlined class="service-icon" />
                <h3>{{ template.name }}</h3>
              </div>
            </div>
            
            <div class="card-content">
              <div class="card-detail created-at-detail">
                <span class="detail-label">创建时间:</span>
                <span class="detail-value">
                  <ClockCircleOutlined />
                  {{ formatDate(template.created_at) }}
                </span>
              </div>
              <div class="card-detail updated-at-detail">
                <span class="detail-label">更新时间:</span>
                <span class="detail-value">
                  <ClockCircleOutlined />
                  {{ formatDate(template.updated_at) }}
                </span>
              </div>
              <div class="card-detail cluster-detail">
                <span class="detail-label">所属集群:</span>
                <span class="detail-value">
                  <CloudServerOutlined />
                  {{ selectedClusterName }}
                </span>
              </div>
            </div>
            
            <div class="card-footer card-action-footer">
              <a-button type="primary" ghost size="small" @click="handleCheck(template)">
                <template #icon><CheckOutlined /></template>
                检查
              </a-button>
              <a-button type="primary" ghost size="small" @click="handleEdit(template)">
                <template #icon><EditOutlined /></template>
                编辑
              </a-button>
              <a-popconfirm
                title="确定要删除该模板吗?"
                @confirm="handleDelete(template)"
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

    <!-- 创建/编辑模板模态框 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑模板' : '创建模板'"
      @ok="handleSubmit"
      width="800px"
      :okText="isEdit ? '保存更改' : '创建模板'"
      :maskClosable="false"
      class="yaml-modal template-modal"
    >
      <a-alert v-if="selectedCluster" class="yaml-info" type="info" show-icon>
        <template #message>
          <span>{{ selectedClusterName }} 集群模板</span>
        </template>
        <template #description>
          <div>{{ isEdit ? '编辑现有模板' : '创建新模板' }}</div>
        </template>
      </a-alert>
      
      <a-form 
        :model="formState" 
        :rules="rules" 
        ref="formRef"
        layout="vertical"
        class="template-form"
      >
        <a-form-item label="模板名称" name="name">
          <a-input 
            v-model:value="formState.name" 
            placeholder="请输入模板名称" 
            class="form-input"
          >
            <template #prefix>
              <FileOutlined />
            </template>
          </a-input>
        </a-form-item>
        
        <a-form-item label="YAML内容" name="content">
          <div class="yaml-actions">
            <a-button type="primary" size="small" @click="formatYaml" :loading="formatting">
              <template #icon><AlignLeftOutlined /></template>
              格式化
            </a-button>
            <a-button size="small" @click="expandEditor = !expandEditor">
              <template #icon>
                <ExpandOutlined v-if="!expandEditor" />
                <CompressOutlined v-else />
              </template>
              {{ expandEditor ? '收起' : '展开' }}
            </a-button>
          </div>
          <a-textarea
            v-model:value="formState.content"
            placeholder="# 请输入标准YAML格式内容"
            :rows="10"
            :auto-size="{ minRows: expandEditor ? 20 : 10, maxRows: expandEditor ? 30 : 15 }"
            class="yaml-editor"
            spellcheck="false"
          />
          <div class="yaml-tips" :class="{ 'error-tips': formatErrorMsg }">
            <InfoCircleOutlined />
            <span>{{ formatErrorMsg || '提示：点击"格式化"按钮可以美化YAML排版' }}</span>
          </div>
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="modal-footer">
          <a-button @click="modalVisible = false">取消</a-button>
          <a-button type="primary" ghost @click="checkCurrentYaml" :loading="checkingYaml">
            <template #icon><CheckOutlined /></template>
            检查YAML
          </a-button>
          <a-button type="primary" @click="handleSubmit" :loading="submitting">
            {{ isEdit ? '保存更改' : '创建模板' }}
          </a-button>
        </div>
      </template>
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
  EditOutlined,
  DeleteOutlined,
  CheckOutlined,
  ReloadOutlined,
  FileTextOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  InfoCircleOutlined,
  AlignLeftOutlined,
  ExpandOutlined,
  CompressOutlined,
  TableOutlined,
  ClusterOutlined,
  DashboardOutlined
} from '@ant-design/icons-vue';
import {
  getYamlTemplateApi,
  createYamlTemplateApi,
  updateYamlTemplateApi,
  deleteYamlTemplateApi,
  checkYamlTemplateApi,
  getAllClustersApi,
} from '#/api';

// @ts-ignore
import yaml from 'js-yaml';

// 类型定义
interface YamlTemplate {
  id: number;
  name: string;
  content: string;
  created_at?: string;
  updated_at?: string;
}

// 状态变量
const loading = ref(false);
const templates = ref<YamlTemplate[]>([]);
const searchText = ref('');
const modalVisible = ref(false);
const isEdit = ref(false);
const formRef = ref<FormInstance>();
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();
const expandEditor = ref(false);
const checkingYaml = ref(false);
const submitting = ref(false);
const formatting = ref(false);
const formatErrorMsg = ref('');
const viewMode = ref<'table' | 'card'>('table');

const formState = ref<Partial<YamlTemplate>>({
  name: '',
  content: '',
});

// 表单校验规则
const rules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' },
    { min: 2, max: 50, message: '模板名称长度应为2-50个字符', trigger: 'blur' }
  ],
  content: [{ required: true, message: '请输入YAML内容', trigger: 'blur' }],
};

// 表格列配置
const columns = [
  {
    title: '模板名称',
    dataIndex: 'name',
    key: 'name',
    width: '35%',
    sorter: (a: YamlTemplate, b: YamlTemplate) => a.name.localeCompare(b.name),
    slots: { customRender: 'name' },
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: '25%',
    sorter: (a: YamlTemplate, b: YamlTemplate) => {
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
    sorter: (a: YamlTemplate, b: YamlTemplate) => {
      if (!a.updated_at || !b.updated_at) return 0;
      return new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
    },
    slots: { customRender: 'updated_at' },
  },
  {
    title: '操作',
    key: 'action',
    width: '15%',
    fixed: 'right',
    slots: { customRender: 'action' },
  },
];

// 计算属性
const selectedClusterName = computed(() => {
  const cluster = clusters.value.find(c => c.id === selectedCluster.value);
  return cluster ? cluster.name : '';
});

// 计算属性：最近更新时间
const lastUpdateTime = computed(() => {
  if (!templates.value.length) return '-';
  
  let latestDate = new Date(0);
  templates.value.forEach(template => {
    if (template.updated_at) {
      const updateDate = new Date(template.updated_at);
      if (updateDate > latestDate) {
        latestDate = updateDate;
      }
    }
  });
  
  if (latestDate.getTime() === 0) return '-';
  
  return formatDate(latestDate.toISOString());
});

// 日期格式化函数
const formatDate = (dateString?: string) => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const formatFullDate = (dateString?: string) => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
};

// 计算属性：过滤后的模板列表
const filteredTemplates = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  if (!searchValue) return templates.value;
  return templates.value.filter(template => template.name.toLowerCase().includes(searchValue));
});

// 获取集群列表
const getClusters = async () => {
  try {
    const res = await getAllClustersApi();
    clusters.value = res || [];
    // 如果有集群，默认选择第一个
    if (clusters.value.length > 0 && !selectedCluster.value) {
      const firstCluster = clusters.value[0];
      if (firstCluster?.id) {
        selectedCluster.value = firstCluster.id;
        await getTemplates();
      }
    }
  } catch (error: any) {
    message.error(error.message || '获取集群列表失败');
  }
};

// 获取模板列表
const getTemplates = async () => {
  if (!selectedCluster.value) {
    message.warning('请先选择集群');
    return;
  }

  loading.value = true;
  try {
    const res = await getYamlTemplateApi(selectedCluster.value);
    templates.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取模板列表失败');
  } finally {
    loading.value = false;
  }
};

// 搜索
const onSearch = () => {
  // 搜索逻辑已经在计算属性中实现，这里可以添加其他触发行为
};

// 切换集群
const handleClusterChange = () => {
  templates.value = [];
  getTemplates();
};

// 显示创建模态框
const showCreateModal = () => {
  isEdit.value = false;
  formState.value = {
    name: '',
    content: '',
  };
  formatErrorMsg.value = '';
  modalVisible.value = true;
};

// 显示编辑模态框
const handleEdit = (record: YamlTemplate) => {
  isEdit.value = true;
  formState.value = {
    id: record.id,
    name: record.name,
    content: record.content,
  };
  formatErrorMsg.value = '';
  modalVisible.value = true;
};

// 检查YAML
const handleCheck = async (record: YamlTemplate) => {
  if (!selectedCluster.value) return;
  
  const hide = message.loading('正在检查YAML格式...', 0);
  try {
    await checkYamlTemplateApi({
      cluster_id: selectedCluster.value,
      name: record.name,
      content: record.content,
    });
    hide();
    message.success('YAML格式检查通过');
  } catch (error: any) {
    hide();
    message.error(error.message || 'YAML格式检查失败');
  }
};

// 检查当前编辑器中的YAML
const checkCurrentYaml = async () => {
  if (!selectedCluster.value || !formState.value.content) {
    message.warning('请先输入YAML内容');
    return;
  }
  
  checkingYaml.value = true;
  try {
    await checkYamlTemplateApi({
      cluster_id: selectedCluster.value,
      name: formState.value.name || '临时检查',
      content: formState.value.content,
    });
    message.success('YAML格式检查通过');
    formatErrorMsg.value = '';
  } catch (error: any) {
    message.error(error.message || 'YAML格式检查失败');
  } finally {
    checkingYaml.value = false;
  }
};

// 格式化YAML 
const formatYaml = () => {
  if (!formState.value.content?.trim()) {
    message.warning('请先输入YAML内容再进行格式化');
    return;
  }
  
  if (!yaml) {
    message.warning('格式化功能未加载，请确保已安装js-yaml库');
    return;
  }
  
  formatting.value = true;
  formatErrorMsg.value = '';
  
  try {
    // 使用js-yaml解析YAML内容
    const parsedYaml = yaml.load(formState.value.content);
    
    // 使用js-yaml重新dump格式化后的内容，设置缩进为2
    const formattedYaml = yaml.dump(parsedYaml, {
      indent: 2,
      lineWidth: -1,  // 不限制行宽
      noRefs: true,   // 不使用引用标记
      noCompatMode: true,  // 使用最新的YAML规范
    });
    
    // 更新文本框内容
    formState.value.content = formattedYaml;
    message.success('YAML格式化成功');
  } catch (error: any) {
    // 如果解析出错，显示错误信息
    message.error('YAML格式化失败，请检查语法');
    formatErrorMsg.value = `格式化错误: ${error.message}`;
    console.error('YAML格式化错误:', error);
  } finally {
    formatting.value = false;
  }
};

// 提交表单
const handleSubmit = async () => {
  if (!selectedCluster.value) return;

  try {
    await formRef.value?.validate();
    submitting.value = true;
    
    const hide = message.loading(isEdit.value ? '正在更新模板...' : '正在创建模板...', 0);
    
    if (isEdit.value) {
      await updateYamlTemplateApi({
        cluster_id: selectedCluster.value,
        id: formState.value.id,
        name: formState.value.name,
        content: formState.value.content,
      });
      hide();
      message.success('模板更新成功');
    } else {
      await createYamlTemplateApi({
        cluster_id: selectedCluster.value,
        name: formState.value.name,
        content: formState.value.content,
      });
      hide();
      message.success('模板创建成功');
    }
    
    modalVisible.value = false;
    getTemplates();
  } catch (error: any) {
    message.error(error.message || (isEdit.value ? '更新模板失败' : '创建模板失败'));
  } finally {
    submitting.value = false;
  }
};

// 删除模板
const handleDelete = async (template: YamlTemplate) => {
  if (!selectedCluster.value) {
    message.error('请选择集群');
    return;
  }

  const hide = message.loading('正在删除模板...', 0);
  try {
    await deleteYamlTemplateApi(template.id, selectedCluster.value);
    hide();
    message.success('删除成功');
    getTemplates();
  } catch (error: any) {
    hide();
    message.error(error.message || '删除失败');
  }
};

// 页面加载时获取数据
onMounted(() => {
  getClusters();
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

.template-manager {
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

.cluster-option {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cluster-option :deep(svg) {
  margin-right: 4px;
}

/* 提示信息 */
.cluster-alert {
  border-radius: 8px;
  margin-bottom: 24px;
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

.cluster-card .metric-icon {
  color: #722ed1;
}

.cluster-name {
  font-size: 22px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
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

/* 模板表格样式 */
.template-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.template-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.template-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.template-name {
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
.template-cards {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

/* 卡片样式优化 */
.template-card {
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

.template-card:hover {
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

.template-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.template-title h3 {
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

/* 创建/编辑模板模态框 */
.template-modal {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
}

.yaml-modal {
  font-family: "Consolas", "Monaco", monospace;
}

.yaml-info {
  margin-bottom: 16px;
}

.yaml-editor {
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
  border-radius: 8px;
  background-color: #f9f9f9;
  padding: 12px;
  transition: all 0.3s;
  tab-size: 2;
}

.yaml-editor:hover {
  background-color: #f5f5f5;
}

.yaml-editor:focus {
  background-color: #f0f0f0;
  border-color: #40a9ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.yaml-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-bottom: 10px;
}

.yaml-tips {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  color: #8c8c8c;
  font-size: 13px;
}

.error-tips {
  color: #ff4d4f;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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
  .template-cards {
    justify-content: space-around;
  }
  
  .template-card {
    width: 320px;
  }
}

@media (max-width: 768px) {
  .template-cards {
    flex-direction: column;
    align-items: center;
  }
  
  .template-card {
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