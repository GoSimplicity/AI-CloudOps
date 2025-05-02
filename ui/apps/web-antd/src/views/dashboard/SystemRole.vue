<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model:value="searchText" 
          placeholder="请输入角色名称"
          style="width: 200px; margin-right: 16px;"
          @pressEnter="handleSearch"
        />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">创建角色</a-button>
      </div>
    </div>

    <!-- 角色列表表格 -->
    <a-table 
      :columns="columns" 
      :data-source="filteredRoleList" 
      row-key="id" 
      :loading="loading"
      :pagination="pagination"
      @change="handleTableChange"
    >
      <!-- 方法列自定义渲染 -->
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'method'">
          {{ record.method }}
        </template>
        <template v-if="column.key === 'action'">
          <a-space>
            <a-tooltip title="编辑角色">
              <a-button type="link" @click="handleEdit(record)">
                <template #icon><Icon icon="clarity:note-edit-line" style="font-size: 22px" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除角色">
              <a-popconfirm
                title="确定要删除这个角色吗?"
                ok-text="确定"
                cancel-text="取消"
                placement="left"
                @confirm="handleDelete(record)"
              >
                <a-button type="link" danger>
                  <template #icon><Icon icon="ant-design:delete-outlined" style="font-size: 22px" /></template>
                </a-button>
              </a-popconfirm>
            </a-tooltip>
          </a-space>
        </template>
      </template>
    </a-table>

    <!-- 角色表单弹窗 -->
    <a-modal
      v-model:visible="isModalVisible"
      :title="modalTitle"
      @ok="handleModalSubmit"
      @cancel="handleModalCancel"
      :okText="'保存'"
      :cancelText="'取消'"
      width="800px"
    >
      <a-form :model="formData" layout="vertical" :rules="formRules">
        <a-form-item label="角色名称" name="name">
          <a-input v-model:value="formData.name" placeholder="请输入角色名称" />
        </a-form-item>
        <a-form-item label="域ID" name="domain">
          <a-input v-model:value="formData.domain" placeholder="请输入域ID" />
        </a-form-item>
        <a-form-item label="路径" name="path">
          <a-input v-model:value="formData.path" placeholder="请输入路径" />
        </a-form-item>
        <a-form-item label="方法" name="method">
          <a-select v-model:value="formData.method" placeholder="请选择方法">
            <a-select-option value="GET">GET</a-select-option>
            <a-select-option value="POST">POST</a-select-option>
            <a-select-option value="PUT">PUT</a-select-option>
            <a-select-option value="DELETE">DELETE</a-select-option>
            <a-select-option value="*">ALL</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, computed } from 'vue';
import { message } from 'ant-design-vue';
import { 
  listRolesApi, 
  createRoleApi, 
  updateRoleApi, 
  deleteRoleApi,
  listApisApi,
} from '#/api/core/system';
import type { SystemApi } from '#/api/core/system';
import { Icon } from '@iconify/vue';

interface ApiItem {
  id: number;
  name: string;
  path: string;
  method: string;
  description?: string;
  version?: string;
  category?: number;
  is_public: number;
}

// 表格加载状态
const loading = ref(false);

// 搜索文本
const searchText = ref('');

// 角色列表数据
const roleList = ref<any[]>([]);

// API选项
const apiOptions = ref<{label: string, value: number}[]>([]);

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入域ID', trigger: 'blur' }],
  path: [{ required: true, message: '请输入路径', trigger: 'blur' }],
  method: [{ required: true, message: '请选择方法', trigger: 'change' }]
};

// 过滤后的角色列表
const filteredRoleList = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  if (!searchValue) return roleList.value;
  
  return roleList.value.filter(role => 
    role.name.toLowerCase().includes(searchValue)
  );
});

// 模态框相关
const isModalVisible = ref(false);
const modalTitle = ref('创建角色');
const formData = reactive<Partial<SystemApi.Role>>({
  name: '',
  domain: '',
  path: '',
  method: ''
});

// 当前编辑的角色信息（用于更新）
const currentRole = reactive<Partial<SystemApi.Role>>({
  name: '',
  domain: '',
  path: '',
  method: ''
});

// 获取所有API
const fetchApis = async () => {
  try {
    // 获取API列表
    const apiRes = await listApisApi({
      page_number: 1,
      page_size: 1000
    });
    if (apiRes && apiRes.list) {
      apiOptions.value = apiRes.list.map((api: ApiItem) => ({
        label: `${api.name} (${api.path}) [${api.method}]`,
        value: api.id
      }));
    }
  } catch (error: any) {
    message.error(error.message || '获取权限数据失败');
  }
};

// 表格列配置
const columns = [
  {
    title: '角色名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '域ID',
    dataIndex: 'domain',
    key: 'domain',
  },
  {
    title: '路径',
    dataIndex: 'path',
    key: 'path',
    ellipsis: true,
  },
  {
    title: '方法',
    dataIndex: 'method',
    key: 'method',
  },
  {
    title: '操作',
    key: 'action',
    width: '150px',
    fixed: 'right'
  },
];

// 处理表格变化（分页、排序、筛选）
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchRoleList();
};

// 获取角色列表
const fetchRoleList = async () => {
  loading.value = true;
  try {
    const res = await listRolesApi({
      page_number: pagination.current,
      page_size: pagination.pageSize
    });
    if (res && res.items) {
      roleList.value = res.items;
      pagination.total = res.total || res.items.length;
    } else {
      roleList.value = [];
      pagination.total = 0;
    }
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
    roleList.value = [];
    pagination.total = 0;
  } finally {
    loading.value = false;
  }
};

// 处理搜索
const handleSearch = () => {
  pagination.current = 1; // 搜索时重置到第一页
  // 搜索功能已通过 computed 属性 filteredRoleList 实现
};

// 处理添加
const handleAdd = () => {
  modalTitle.value = '创建角色';
  Object.assign(formData, {
    name: '',
    domain: '',
    path: '',
    method: ''
  });
  isModalVisible.value = true;
};

// 处理编辑
const handleEdit = async (record: SystemApi.Role) => {
  modalTitle.value = '编辑角色';
  
  // 保存当前角色信息用于更新
  Object.assign(currentRole, {
    name: record.name,
    domain: record.domain,
    path: record.path,
    method: record.method
  });
  
  // 设置表单数据
  Object.assign(formData, {
    name: record.name,
    domain: record.domain,
    path: record.path,
    method: record.method
  });

  isModalVisible.value = true;
};

// 处理删除
const handleDelete = async (record: SystemApi.Role) => {
  try {
    await deleteRoleApi({
      name: record.name,
      domain: record.domain,
      path: record.path,
      method: record.method
    });
    message.success('删除成功');
    fetchRoleList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

// 处理模态框提交
const handleModalSubmit = async () => {
  // 表单验证
  if (!formData.name || !formData.domain || !formData.path || !formData.method) {
    message.warning('请填写完整的角色信息');
    return;
  }
  
  try {
    if (modalTitle.value === '创建角色') {
      await createRoleApi({
        ...formData as SystemApi.CreateRoleReq
      });
    } else {
      await updateRoleApi({
        old_role: {
          ...currentRole as SystemApi.Role
        },
        new_role: {
          ...formData as SystemApi.Role
        }
      });
    }
    message.success(`${modalTitle.value}成功`);
    isModalVisible.value = false;
    fetchRoleList();
  } catch (error: any) {
    message.error(error.message || `${modalTitle.value}失败`);
  }
};

// 处理模态框取消
const handleModalCancel = () => {
  isModalVisible.value = false;
};

onMounted(() => {
  fetchRoleList();
  fetchApis();
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
  align-items: center;
}

.action-buttons {
  margin-left: 16px;
}
</style>
