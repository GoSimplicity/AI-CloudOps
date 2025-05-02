<template>
  <div>
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-select
          v-model:value="selectedCluster"
          placeholder="请选择集群"
          style="width: 200px; margin-right: 16px"
          @change="handleClusterChange"
        >
          <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
            {{ cluster.name }}
          </a-select-option>
        </a-select>
        <a-input v-model:value="searchText" placeholder="请输入模板名称" style="width: 200px" />
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="showCreateModal" :disabled="!selectedCluster">
          创建模板
        </a-button>
      </div>
    </div>

    <!-- 模板列表 -->
    <a-table
      :columns="columns"
      :data-source="filteredTemplates"
      :loading="loading"
      row-key="id"
    >
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="handleCheck(record)">
            <template #icon><CheckOutlined /></template>
            检查
          </a-button>
          <a-button type="primary" ghost size="small" @click="handleEdit(record)">
            <template #icon><EditOutlined /></template>
            编辑
          </a-button>
          <a-popconfirm
            title="确定要删除该模板吗？"
            @confirm="handleDelete(record)"
            ok-text="确定"
            cancel-text="取消"
          >
            <a-button type="primary" danger ghost size="small">
              <template #icon><DeleteOutlined /></template>
              删除
            </a-button>
          </a-popconfirm>
        </a-space>
      </template>
    </a-table>

    <!-- 创建/编辑模板模态框 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑模板' : '创建模板'"
      @ok="handleSubmit"
      width="800px"
    >
      <a-form :model="formState" :rules="rules" ref="formRef">
        <a-form-item label="模板名称" name="name">
          <a-input v-model:value="formState.name" placeholder="请输入模板名称" />
        </a-form-item>
        <a-form-item label="YAML内容" name="content">
          <a-textarea
            v-model:value="formState.content"
            placeholder="请输入YAML内容"
            :rows="10"
            :auto-size="{ minRows: 10, maxRows: 20 }"
          />
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
  getYamlTemplateApi,
  createYamlTemplateApi,
  updateYamlTemplateApi,
  deleteYamlTemplateApi,
  checkYamlTemplateApi,
  getAllClustersApi,
} from '#/api';

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
const formState = ref<Partial<YamlTemplate>>({
  name: '',
  content: '',
});

// 表单校验规则
const rules = {
  name: [{ required: true, message: '请输入模板名称', trigger: 'blur' }],
  content: [{ required: true, message: '请输入YAML内容', trigger: 'blur' }],
};

// 表格列配置
const columns = [
  {
    title: '模板名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
  },
  {
    title: '更新时间',
    dataIndex: 'updated_at',
    key: 'updated_at',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 计算属性：过滤后的模板列表
const filteredTemplates = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return templates.value.filter(template => template.name.toLowerCase().includes(searchValue));
});

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
  modalVisible.value = true;
};

// 检查YAML
const handleCheck = async (record: YamlTemplate) => {
  if (!selectedCluster.value) return;
  
  try {
    await checkYamlTemplateApi({
      cluster_id: selectedCluster.value,
      name: record.name,
      content: record.content,
    });
    message.success('YAML格式检查通过');
  } catch (error: any) {
    message.error(error.message || 'YAML格式检查失败');
  }
};

// 提交表单
const handleSubmit = async () => {
  if (!selectedCluster.value) return;

  try {
    await formRef.value?.validate();
    
    if (isEdit.value) {
      await updateYamlTemplateApi({
        cluster_id: selectedCluster.value,
        id: formState.value.id,
        name: formState.value.name,
        content: formState.value.content,
      });
      message.success('模板更新成功');
    } else {
      await createYamlTemplateApi({
        cluster_id: selectedCluster.value,
        name: formState.value.name,
        content: formState.value.content,
      });
      message.success('模板创建成功');
    }
    
    modalVisible.value = false;
    getTemplates();
  } catch (error: any) {
    message.error(error.message || (isEdit.value ? '更新模板失败' : '创建模板失败'));
  }
};

// 删除模板
const handleDelete = async (template: YamlTemplate) => {
  if (!selectedCluster.value) {
    message.error('请选择集群');
    return;
  }

  try {
    await deleteYamlTemplateApi(template.id, selectedCluster.value);
    message.success('删除成功');
    getTemplates();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

// 页面加载时获取数据
onMounted(() => {
  getClusters();
});
</script>

<style scoped>
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
  gap: 16px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

:deep(.ant-form-item-label) {
  width: 80px;
  text-align: right;
}

:deep(.ant-input) {
  font-family: monospace;
}

.custom-toolbar {
  padding: 6px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  align-items: center;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 16px;
}
</style>
