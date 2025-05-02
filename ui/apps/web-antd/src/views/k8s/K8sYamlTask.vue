<template>
  <div>
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input v-model:value="searchText" placeholder="请输入任务名称" style="width: 200px" />
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="showCreateModal">
          创建任务
        </a-button>
      </div>
    </div>

    <!-- 任务列表 -->
    <a-table
      :columns="columns"
      :data-source="filteredTasks"
      :loading="loading"
      row-key="id"
    >
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="handleApply(record)">
            <template #icon><PlayCircleOutlined /></template>
            应用
          </a-button>
          <a-button type="primary" ghost size="small" @click="handleEdit(record)">
            <template #icon><EditOutlined /></template>
            编辑
          </a-button>
          <a-popconfirm
            title="确定要删除该任务吗？"
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

    <!-- 创建/编辑任务模态框 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑任务' : '创建任务'"
      @ok="handleSubmit"
      width="800px"
    >
      <a-form :model="formState" :rules="rules" ref="formRef">
        <a-form-item label="任务名称" name="name">
          <a-input v-model:value="formState.name" placeholder="请输入任务名称" />
        </a-form-item>
        <a-form-item label="选择模板" name="template_id">
          <a-select
            v-model:value="formState.template_id"
            placeholder="请选择模板"
          >
            <a-select-option v-for="template in templates" :key="template.id" :value="template.id">
              {{ template.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="选择集群" name="cluster_id">
          <a-select
            v-model:value="formState.cluster_id"
            placeholder="请选择集群"
          >
            <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
              {{ cluster.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="变量列表" name="variables">
          <a-button type="link" @click="addVariable" style="margin-bottom: 8px">
            添加变量
          </a-button>
          <div v-for="(_, index) in formState.variables" :key="index" style="display: flex; gap: 8px; margin-bottom: 8px">
            <a-input
              v-model:value="formState.variables[index]"
              placeholder="key=value"
              style="flex: 1"
            />
            <a-button type="link" danger @click="removeVariable(index)">
              删除
            </a-button>
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

const formState = ref<Partial<YamlTask>>({
  name: '',
  template_id: undefined,
  cluster_id: undefined,
  variables: [],
});

// 表单校验规则
const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  template_id: [{ required: true, message: '请选择模板', trigger: 'change' }],
  cluster_id: [{ required: true, message: '请选择集群', trigger: 'change' }],
};

// 表格列配置
const columns = [
  {
    title: '任务名称',
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

// 计算属性：过滤后的任务列表
const filteredTasks = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return tasks.value.filter(task => task.name.toLowerCase().includes(searchValue));
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
  try {
    await applyYamlTaskApi(record.id);
    message.success('任务应用成功');
  } catch (error: any) {
    message.error(error.message || '任务应用失败');
  }
};

// 提交表单
const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    
    // 过滤掉空的变量
    const variables = formState.value.variables?.filter(v => v.trim()) || [];
    
    if (isEdit.value) {
      await updateYamlTaskApi({
        id: formState.value.id,
        name: formState.value.name,
        template_id: formState.value.template_id,
        cluster_id: formState.value.cluster_id,
        variables,
      });
      message.success('任务更新成功');
    } else {
      await createYamlTaskApi({
        name: formState.value.name,
        template_id: formState.value.template_id,
        cluster_id: formState.value.cluster_id,
        variables,
      });
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
  try {
    await deleteYamlTaskApi(task.id);
    message.success('删除成功');
    getTasks();
  } catch (error: any) {
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

<style scoped>
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
  padding: 6px;
  align-items: center;
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
  margin-left: 16px;
}

:deep(.ant-form-item-label) {
  width: 80px;
  text-align: right;
}

:deep(.ant-input) {
  font-family: monospace;
}
</style>
