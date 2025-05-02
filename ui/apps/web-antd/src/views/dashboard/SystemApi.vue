<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model:value="searchText" 
          placeholder="请输入API名称"
          style="width: 200px; margin-right: 16px;"
        />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">新增API</a-button>
      </div>
    </div>

    <!-- API列表表格 -->
    <a-table :columns="columns" :data-source="filteredApiList" row-key="id" :loading="loading">
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑API">
            <a-button type="link" @click="handleEdit(record)">
              <template #icon><Icon icon="clarity:note-edit-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-popconfirm
            title="确定要删除这个API吗?"
            ok-text="确定"
            cancel-text="取消"
            placement="left"
            @confirm="handleDelete(record)"
          >
            <a-tooltip title="删除API">
              <a-button type="link" danger>
                <template #icon><Icon icon="ant-design:delete-outlined" style="font-size: 22px" /></template>
              </a-button>
            </a-tooltip>
          </a-popconfirm>
        </a-space>
      </template>

      <!-- 请求方法列 -->
      <template #method="{ record }">
        <a-tag :color="getMethodColor(record.method)">
          {{ getMethodName(record.method) }}
        </a-tag>
      </template>

      <!-- 公开状态列 -->
      <template #isPublic="{ record }">
        <a-switch
          :checked="record.is_public === 1"
          @change="(checked: boolean) => handlePublicChange(record, checked ? 1 : 0)"
        />
      </template>
    </a-table>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="modalTitle"
      @ok="handleModalOk"
      @cancel="handleModalCancel"
      :okText="'保存'"
      :cancelText="'取消'"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="API名称" required>
          <a-input v-model:value="formData.name" placeholder="请输入API名称" />
        </a-form-item>
        <a-form-item label="API路径" required>
          <a-input v-model:value="formData.path" placeholder="请输入API路径" />
        </a-form-item>
        <a-form-item label="请求方法" required>
          <a-select v-model:value="formData.method">
            <a-select-option :value="1">GET</a-select-option>
            <a-select-option :value="2">POST</a-select-option>
            <a-select-option :value="3">PUT</a-select-option>
            <a-select-option :value="4">DELETE</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="API描述">
          <a-textarea v-model:value="formData.description" placeholder="请输入API描述" />
        </a-form-item>
        <a-form-item label="API版本">
          <a-input v-model:value="formData.version" placeholder="请输入API版本" />
        </a-form-item>
        <a-form-item label="API分类">
          <a-input-number v-model:value="formData.category" placeholder="请输入API分类" />
        </a-form-item>
        <a-form-item label="是否公开">
          <a-switch v-model:checked="formData.is_public" :checkedValue="1" :unCheckedValue="0" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, computed } from 'vue';
import { message } from 'ant-design-vue';
import { listApisApi, createApiApi, updateApiApi, deleteApiApi } from '#/api/core/system';
import type { SystemApi } from '#/api/core/system';
import { Icon } from '@iconify/vue';

// 表格加载状态
const loading = ref(false);

// 搜索文本
const searchText = ref('');

// API列表数据
const apiList = ref<any[]>([]);

// 过滤后的API列表
const filteredApiList = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  if (!searchValue) return apiList.value;
  
  return apiList.value.filter(api => 
    api.name.toLowerCase().includes(searchValue) ||
    api.path.toLowerCase().includes(searchValue) ||
    api.description?.toLowerCase().includes(searchValue)
  );
});

// 对话框相关
const modalVisible = ref(false);
const modalTitle = ref('新增API');
const formData = reactive<SystemApi.CreateApiReq>({
  name: '',
  path: '',
  method: 1,
  description: '',
  version: '',
  category: undefined,
  is_public: 0
});

// 获取API列表
const fetchApiList = async () => {
  loading.value = true;
  try {
    const res = await listApisApi({
      page_number: 1,
      page_size: 999
    });
    apiList.value = res.list;
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
  }
  loading.value = false;
};

// 表格列配置
const columns = [
  {
    title: 'API名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: 'API路径', 
    dataIndex: 'path',
    key: 'path',
  },
  {
    title: '请求方法',
    dataIndex: 'method',
    key: 'method', 
    slots: { customRender: 'method' },
  },
  {
    title: 'API描述',
    dataIndex: 'description',
    key: 'description',
  },
  {
    title: 'API版本',
    dataIndex: 'version',
    key: 'version',
  },
  {
    title: '是否公开',
    dataIndex: 'is_public',
    key: 'is_public',
    slots: { customRender: 'isPublic' },
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 获取请求方法名称
const getMethodName = (method: number) => {
  const methodMap: Record<number, string> = {
    1: 'GET',
    2: 'POST', 
    3: 'PUT',
    4: 'DELETE'
  };
  return methodMap[method] || '未知';
};

// 获取请求方法颜色
const getMethodColor = (method: number) => {
  const colorMap: Record<number, string> = {
    1: 'blue',
    2: 'green',
    3: 'orange', 
    4: 'red'
  };
  return colorMap[method] || 'default';
};

// 处理搜索
const handleSearch = () => {
  // 搜索功能已通过 computed 属性 filteredApiList 实现
  // 不需要额外的处理逻辑
};

// 处理新增
const handleAdd = () => {
  modalTitle.value = '新增API';
  Object.assign(formData, {
    name: '',
    path: '',
    method: 1,
    description: '',
    version: '',
    category: undefined,
    is_public: 0
  });
  modalVisible.value = true;
};

// 处理编辑
const handleEdit = (record: any) => {
  modalTitle.value = '编辑API';
  Object.assign(formData, record);
  modalVisible.value = true;
};

// 处理删除
const handleDelete = async (record: any) => {
  try {
    await deleteApiApi(record.id);
    message.success('删除成功');
    fetchApiList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

// 处理公开状态切换
const handlePublicChange = async (record: any, isPublic: number) => {
  try {
    await updateApiApi({
      ...record,
      is_public: isPublic,
    });
    message.success('更新成功');
    fetchApiList();
  } catch (error: any) {
    message.error(error.message || '更新失败');
  }
};

// 处理对话框确认
const handleModalOk = async () => {
  try {
    if (modalTitle.value === '新增API') {
      await createApiApi(formData);
      message.success('新增API成功');
    } else {
      await updateApiApi(formData as SystemApi.UpdateApiReq);
      message.success('编辑API成功');
    }
    modalVisible.value = false;
    fetchApiList();
  } catch (error: any) {
    message.error(error.message || `${modalTitle.value}失败`);
  }
};

// 处理对话框取消
const handleModalCancel = () => {
  modalVisible.value = false;
};

// 页面加载时获取数据
onMounted(() => {
  fetchApiList();
});
</script>

<style scoped>
.custom-toolbar {
  padding: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  align-items: center;
}
</style>
