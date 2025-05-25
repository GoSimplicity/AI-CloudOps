<template>
  <div class="form-design-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateForm" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新表单
        </a-button>
        <a-input-search 
          v-model:value="searchQuery" 
          placeholder="搜索表单..." 
          style="width: 250px" 
          @search="handleSearch"
          allow-clear 
        />
        <a-select 
          v-model:value="statusFilter" 
          placeholder="状态" 
          style="width: 120px" 
          @change="handleStatusChange"
        >
          <a-select-option :value="undefined">全部</a-select-option>
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
            <a-statistic title="总表单数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FormOutlined />
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
        <a-table 
          :data-source="paginatedForms" 
          :columns="columns" 
          :pagination="false" 
          :loading="loading" 
          row-key="id"
          bordered
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="form-name-cell">
                <div class="form-badge" :class="getStatusClass(record.status)"></div>
                <span class="form-name-text">{{ record.name }}</span>
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
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name || '') }">
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
                <a-button type="primary" size="small" @click="handleViewForm(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditForm(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleMenuClick(e.key, record)">
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
          <a-pagination 
            v-model:current="currentPage" 
            :total="totalItems" 
            :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" 
            :show-size-changer="true" 
            @change="handleCurrentChange"
            @showSizeChange="handleSizeChange" 
            :show-total="(total: number) => `共 ${total} 条`" 
          />
        </div>
      </a-card>
    </div>

    <!-- 表单创建/编辑对话框 -->
    <a-modal 
      :open="formDialogVisible" 
      :title="formDialog.isEdit ? '编辑表单设计' : '创建表单设计'" 
      width="760px"
      @ok="saveForm" 
      @cancel="closeFormDialog"
      :destroy-on-close="true"
    >
      <a-form ref="formRef" :model="formDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="表单名称" name="name">
          <a-input v-model:value="formDialog.form.name" placeholder="请输入表单名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="formDialog.form.description" :rows="3" placeholder="请输入表单描述" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="24">
            <a-form-item label="分类" name="category_id">
              <a-select v-model:value="formDialog.form.category_id" placeholder="请选择分类" style="width: 100%">
                <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
                  {{ cat.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-divider orientation="left">表单结构</a-divider>

        <div class="schema-editor">
          <div class="field-list">
            <a-collapse>
              <a-collapse-panel 
                v-for="(field, index) in formDialog.form.schema.fields" 
                :key="index"
                :header="field.label || `字段 ${index + 1}`"
              >
                <template #extra>
                  <a-button type="text" danger @click.stop="removeField(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </template>

                <a-form-item label="字段类型">
                  <a-select v-model:value="field.type" style="width: 100%">
                    <a-select-option value="text">文本框</a-select-option>
                    <a-select-option value="number">数字</a-select-option>
                    <a-select-option value="date">日期</a-select-option>
                    <a-select-option value="select">下拉选择</a-select-option>
                    <a-select-option value="checkbox">复选框</a-select-option>
                    <a-select-option value="radio">单选框</a-select-option>
                    <a-select-option value="textarea">多行文本</a-select-option>
                  </a-select>
                </a-form-item>

                <a-form-item label="标签名称">
                  <a-input v-model:value="field.label" placeholder="字段标签" />
                </a-form-item>

                <a-form-item label="字段名称">
                  <a-input v-model:value="field.name" placeholder="字段名称" />
                </a-form-item>

                <a-form-item label="是否必填">
                  <a-switch v-model:checked="field.required" />
                </a-form-item>
              </a-collapse-panel>
            </a-collapse>

            <div class="add-field-button">
              <a-button type="dashed" block @click="addField" style="margin-top: 16px">
                <PlusOutlined /> 添加字段
              </a-button>
            </div>
          </div>
        </div>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal 
      :open="cloneDialogVisible" 
      title="克隆表单" 
      @ok="confirmClone" 
      @cancel="closeCloneDialog"
      :destroy-on-close="true"
    >
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新表单名称" name="name" :rules="[{ required: true, message: '请输入新表单名称' }]">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新表单名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情对话框 -->
    <a-modal 
      :open="detailDialogVisible" 
      title="表单详情" 
      width="70%" 
      :footer="null" 
      @cancel="closeDetailDialog"
      class="detail-dialog"
    >
      <div v-if="detailDialog.form" class="form-details">
        <div class="detail-header">
          <h2>{{ detailDialog.form.name }}</h2>
          <a-tag
            :color="detailDialog.form.status === 0 ? 'orange' : detailDialog.form.status === 1 ? 'green' : 'default'"
          >
            {{ detailDialog.form.status === 0 ? '草稿' : detailDialog.form.status === 1 ? '已发布' : '已禁用' }}
          </a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ detailDialog.form.id }}</a-descriptions-item>
          <a-descriptions-item label="版本">v{{ detailDialog.form.version }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.form.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.form.created_at || '') }}</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">{{ detailDialog.form.description || '无描述' }}</a-descriptions-item>
        </a-descriptions>

        <div class="schema-preview">
          <h3>表单结构</h3>
          <a-table 
            :data-source="parsedSchema.fields" 
            :columns="schemaColumns" 
            :pagination="false" 
            bordered
            size="small" 
            row-key="name"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'required'">
                <a-tag :color="record.required ? 'red' : ''">
                  {{ record.required ? '必填' : '可选' }}
                </a-tag>
              </template>
              <template v-if="column.key === 'type'">
                {{ getFieldTypeName(record.type) }}
              </template>
            </template>
          </a-table>
        </div>

        <div class="detail-footer">
          <a-button @click="closeDetailDialog">关闭</a-button>
          <a-button type="primary" @click="handleEditForm(detailDialog.form)">编辑</a-button>
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
  FormOutlined,
  CheckCircleOutlined,
  EditOutlined,
  StopOutlined,
  DeleteOutlined,
  DownOutlined
} from '@ant-design/icons-vue';
import {
  listFormDesign,
  detailFormDesign,
  createFormDesign,
  updateFormDesign,
  deleteFormDesign,
  publishFormDesign,
  cloneFormDesign,
  type FormDesignResp,
  type FormDesignItem,
  type FormField,
  type FormSchema,
  type FormDesignReq,
  type ListFormDesignReq,
  type DetailFormDesignReq,
  type PublishFormDesignReq,
  type CloneFormDesignReq,
  type Category
} from '#/api/core/workorder';

// 响应式数据类型
interface Statistics {
  total: number;
  published: number;
  draft: number;
  disabled: number;
}

interface FormDialogState {
  isEdit: boolean;
  form: FormDesignReq;
}

interface CloneDialogState {
  form: {
    name: string;
    originalId: number;
  };
}

interface DetailDialogState {
  form: FormDesignResp | null;
}

// 列定义
const columns = [
  {
    title: '表单名称',
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
    align: 'center' as const,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    align: 'center' as const,
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
    align: 'center' as const,
  },
];

// 表单结构列定义
const schemaColumns = [
  {
    title: '类型',
    dataIndex: 'type',
    key: 'type',
    width: 120,
  },
  {
    title: '标签',
    dataIndex: 'label',
    key: 'label',
    width: 180,
  },
  {
    title: '字段名',
    dataIndex: 'name',
    key: 'name',
    width: 180,
  },
  {
    title: '是否必填',
    dataIndex: 'required',
    key: 'required',
    width: 100,
  },
];

// 状态数据
const loading = ref<boolean>(false);
const searchQuery = ref<string>('');
const statusFilter = ref<number | undefined>(undefined);
const currentPage = ref<number>(1);
const pageSize = ref<number>(10);
const formDesigns = ref<FormDesignItem[]>([]);

// 模态框控制
const formDialogVisible = ref<boolean>(false);
const cloneDialogVisible = ref<boolean>(false);
const detailDialogVisible = ref<boolean>(false);

// 统计数据
const stats = reactive<Statistics>({
  total: 0,
  published: 0,
  draft: 0,
  disabled: 0
});

// 分类数据
const categories = ref<Category[]>([
  { id: 1, name: '人力资源', sort_order: 1, status: 1, created_at: '', updated_at: '' },
  { id: 2, name: '财务部门', sort_order: 2, status: 1, created_at: '', updated_at: '' },
  { id: 3, name: 'IT部门', sort_order: 3, status: 1, created_at: '', updated_at: '' },
  { id: 4, name: '运营部门', sort_order: 4, status: 1, created_at: '', updated_at: '' },
  { id: 5, name: '项目管理', sort_order: 5, status: 1, created_at: '', updated_at: '' },
  { id: 6, name: '客户服务', sort_order: 6, status: 1, created_at: '', updated_at: '' },
  { id: 7, name: '采购部门', sort_order: 7, status: 1, created_at: '', updated_at: '' }
]);

// 过滤和分页
const filteredForms = computed(() => {
  let result = [...formDesigns.value];

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(form =>
      form.name.toLowerCase().includes(query) ||
      (form.description && form.description.toLowerCase().includes(query))
    );
  }

  if (statusFilter.value !== undefined) {
    result = result.filter(form => form.status === statusFilter.value);
  }

  return result;
});

const totalItems = computed(() => filteredForms.value.length);

const paginatedForms = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredForms.value.slice(start, end);
});

// 表单对话框
const formDialog = reactive<FormDialogState>({
  isEdit: false,
  form: {
    name: '',
    description: '',
    schema: {
      fields: []
    },
    category_id: undefined
  }
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入表单名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  category_id: [
    { required: true, message: '请选择分类', trigger: 'change' }
  ]
};

// 克隆对话框
const cloneDialog = reactive<CloneDialogState>({
  form: {
    name: '',
    originalId: 0
  }
});

// 详情对话框
const detailDialog = reactive<DetailDialogState>({
  form: null
});

const parsedSchema = computed(() => {
  if (!detailDialog.form || !detailDialog.form.schema) {
    return { fields: [] };
  }
  
  try {
    return typeof detailDialog.form.schema === 'string' 
      ? JSON.parse(detailDialog.form.schema)
      : detailDialog.form.schema;
  } catch (error) {
    console.error('解析schema失败:', error);
    return { fields: [] };
  }
});

// 对话框控制方法
const closeFormDialog = (): void => {
  formDialogVisible.value = false;
};

const closeCloneDialog = (): void => {
  cloneDialogVisible.value = false;
};

const closeDetailDialog = (): void => {
  detailDialogVisible.value = false;
};

// 加载表单列表
const loadFormDesigns = async (): Promise<void> => {
  loading.value = true;
  try {
    const params: ListFormDesignReq = {
      page: currentPage.value,
      size: pageSize.value,
      search: searchQuery.value || undefined,
      status: statusFilter.value !== undefined ? statusFilter.value : undefined
    };
    const response = await listFormDesign(params);
    if (response) {
      formDesigns.value = Array.isArray(response.items) ? response.items : [];
      updateStats(response);
    }
  } catch (error) {
    console.error('加载表单列表失败:', error);
    message.error('加载表单列表失败');
  } finally {
    loading.value = false;
  }
};

// 更新统计数据
const updateStats = (data: any): void => {
  if (data.statistics) {
    stats.total = data.statistics.total || 0;
    stats.published = data.statistics.published || 0;
    stats.draft = data.statistics.draft || 0;
    stats.disabled = data.statistics.disabled || 0;
  } else {
    // 从列表数据计算
    stats.total = formDesigns.value.length;
    stats.published = formDesigns.value.filter((form: { status: number }) => form.status === 1).length;
    stats.draft = formDesigns.value.filter((form: { status: number }) => form.status === 0).length;
    stats.disabled = formDesigns.value.filter((form: { status: number }) => form.status === 2).length;
  }
};

// 方法
const handleSizeChange = (current: number, size: number): void => {
  pageSize.value = size;
  currentPage.value = current;
  loadFormDesigns();
};

const handleCurrentChange = (page: number): void => {
  currentPage.value = page;
  loadFormDesigns();
};

const handleSearch = (): void => {
  currentPage.value = 1;
  loadFormDesigns();
};

const handleStatusChange = (): void => {
  currentPage.value = 1;
  loadFormDesigns();
};

const handleCreateForm = (): void => {
  formDialog.isEdit = false;
  formDialog.form = {
    name: '',
    description: '',
    schema: {
      fields: []
    },
    category_id: undefined
  };
  formDialogVisible.value = true;
};

const handleEditForm = async (row: FormDesignItem | FormDesignResp): Promise<void> => {
  loading.value = true;
  try {
    const response = await detailFormDesign({ id: row.id });
    if (response) {
      const formData = response;
      formDialog.isEdit = true;
      
      // 解析schema字符串为对象
      let schemaObj: FormSchema;
      try {
        schemaObj = typeof formData.schema === 'string' 
          ? JSON.parse(formData.schema)
          : formData.schema;
      } catch (error) {
        console.error('解析schema失败:', error);
        schemaObj = { fields: [] };
      }
      
      formDialog.form = {
        id: formData.id,
        name: formData.name,
        description: formData.description,
        schema: schemaObj,
        category_id: formData.category_id
      };
      
      formDialogVisible.value = true;
      detailDialogVisible.value = false;
    }
  } catch (error) {
    console.error('加载表单详情失败:', error);
    message.error('加载表单详情失败');
  } finally {
    loading.value = false;
  }
};

const handleViewForm = async (row: FormDesignItem): Promise<void> => {
  loading.value = true;
  try {
    const response = await detailFormDesign({ id: row.id });
    if (response) {
      detailDialog.form = response;
      detailDialogVisible.value = true;
    }
  } catch (error) {
    console.error('加载表单详情失败:', error);
    message.error('加载表单详情失败');
  } finally {
    loading.value = false;
  }
};

const handleMenuClick = (command: string, row: FormDesignItem): void => {
  switch (command) {
    case 'publish':
      publishForm(row);
      break;
    case 'unpublish':
      unpublishForm(row);
      break;
    case 'clone':
      showCloneDialog(row);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

const publishForm = async (form: FormDesignItem): Promise<void> => {
  try {
    const params: PublishFormDesignReq = { id: form.id };
    const response = await publishFormDesign(params);
    if (response) {
      message.success(`表单 "${form.name}" 已发布`);
      loadFormDesigns();
    }
  } catch (error) {
    console.error('发布表单失败:', error);
    message.error('发布表单失败');
  }
};

const unpublishForm = async (form: FormDesignItem): Promise<void> => {
  try {
    // 先获取详细信息
    const detailResponse = await detailFormDesign({ id: form.id });
    if (detailResponse && detailResponse.data) {
      const formData = detailResponse.data;
      
      const schemaObj = typeof formData.schema === 'string' 
        ? JSON.parse(formData.schema)
        : formData.schema;
        
      const params: FormDesignReq = {
        id: form.id,
        name: form.name,
        description: form.description,
        schema: schemaObj,
        category_id: form.category_id
      };
      
      const response = await updateFormDesign(params);
      if (response) {
        message.success(`表单 "${form.name}" 已取消发布`);
        loadFormDesigns();
      }
    }
  } catch (error) {
    console.error('取消发布表单失败:', error);
    message.error('取消发布表单失败');
  }
};

const showCloneDialog = (form: FormDesignItem): void => {
  cloneDialog.form.name = `${form.name} 的副本`;
  cloneDialog.form.originalId = form.id;
  cloneDialogVisible.value = true;
};

const confirmClone = async (): Promise<void> => {
  if (!cloneDialog.form.name.trim()) {
    message.error('请输入新表单名称');
    return;
  }
  
  try {
    const params: CloneFormDesignReq = {
      id: cloneDialog.form.originalId, 
      name: cloneDialog.form.name    
    };
    
    const response = await cloneFormDesign(params);
    if (response) {
      message.success(`表单已克隆为 "${cloneDialog.form.name}"`);
      cloneDialogVisible.value = false;
      loadFormDesigns();
    }
  } catch (error) {
    console.error('克隆表单失败:', error);
    message.error('克隆表单失败');
  }
};

const confirmDelete = (form: FormDesignItem): void => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除表单 "${form.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        const params: DetailFormDesignReq = { id: form.id };
        await deleteFormDesign(params);
        message.success(`表单 "${form.name}" 已删除`);
        loadFormDesigns();
      } catch (error) {
        console.error('删除表单失败:', error);
        message.error('删除表单失败');
      }
    }
  });
};

const addField = (): void => {
  const newField: FormField = {
    id: `field_${Date.now()}`,
    type: 'text',
    label: '',
    name: '',
    required: false
  };
  formDialog.form.schema.fields.push(newField);
};

const removeField = (index: number): void => {
  formDialog.form.schema.fields.splice(index, 1);
};

const saveForm = async (): Promise<void> => {
  if (formDialog.form.name.trim() === '') {
    message.error('表单名称不能为空');
    return;
  }

  if (!formDialog.form.category_id) {
    message.error('请选择分类');
    return;
  }
  
  // 验证字段名称是否重复
  const fieldNames = formDialog.form.schema.fields.map(field => field.name);
  const uniqueFieldNames = new Set(fieldNames);
  if (fieldNames.length !== uniqueFieldNames.size) {
    message.error('表单中存在重复的字段名称，请修改');
    return;
  }

  try {
    const formData: FormDesignReq = {
      ...formDialog.form,
      schema: formDialog.form.schema
    };

    if (formDialog.isEdit) {
      await updateFormDesign(formData);
      message.success(`表单 "${formDialog.form.name}" 已更新`);
    } else {
      await createFormDesign(formData);
      message.success(`表单 "${formDialog.form.name}" 已创建`);
    }
    
    formDialogVisible.value = false;
    loadFormDesigns();
  } catch (error) {
    console.error('保存表单失败:', error);
    message.error('保存表单失败');
  }
};

// 获取字段类型名称
const getFieldTypeName = (type: string): string => {
  const typeMap: Record<string, string> = {
    'text': '文本框',
    'number': '数字',
    'date': '日期',
    'select': '下拉选择',
    'checkbox': '复选框',
    'radio': '单选框',
    'textarea': '多行文本'
  };
  return typeMap[type] || type;
};

// 辅助方法
const formatDate = (dateStr: string): string => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (dateStr: string): string => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (dateStr: string): string => {
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

const getInitials = (name: string): string => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: number): string => {
  switch (status) {
    case 0: return 'status-draft';
    case 1: return 'status-published';
    case 2: return 'status-disabled';
    default: return '';
  }
};

const getAvatarColor = (name: string): string => {
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length]!;
};

// 初始化
onMounted(() => {
  loadFormDesigns();
});
</script>

<style scoped>
.form-design-container {
  padding: 24px;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  color: #1f2937;
  margin: 0;
  background: linear-gradient(90deg, #1890ff 0%, #52c41a 100%);
  -webkit-text-fill-color: transparent;
  font-weight: 700;
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

.form-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.form-badge {
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

.form-name-text {
  font-weight: 500;
}

.description-text {
  color: #606266;
  display: -webkit-box;
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

.schema-editor {
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 20px;
}

.field-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.add-field-button {
  text-align: center;
  margin-top: 16px;
}

.detail-dialog .form-details {
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

.schema-preview {
  margin-top: 24px;
}

.schema-preview h3 {
  margin-bottom: 16px;
  color: #1f2937;
  font-size: 18px;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>