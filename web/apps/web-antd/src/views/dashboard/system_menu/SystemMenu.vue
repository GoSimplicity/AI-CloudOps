<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入菜单名称"
          style="width: 200px; margin-right: 16px;"
        />
        <!-- 状态过滤 -->
        <a-select
          v-model="selectedStatus"
          placeholder="请选择状态"
          style="width: 150px; margin-right: 16px;"
          allowClear
        >
          <a-select-option value="true">启用</a-select-option>
          <a-select-option value="false">禁用</a-select-option>
        </a-select>
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">创建菜单</a-button>
      </div>
    </div>

    <!-- 菜单列表表格 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="id">
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleEdit(record)">编辑</a-button>
          <a-button type="link" danger @click="handleDelete(record)">删除</a-button>
        </a-space>
      </template>

      <!-- 状态列 -->
      <template #status="{ record }">
        <a-switch
          :checked="record.status"
          @change="handleStatusChange(record)"
        />
      </template>

      <!-- 图标列 -->
      <template #icon="{ record }">
        <a-icon :type="record.icon || 'menu'" /> <!-- 设置默认图标 -->
      </template>
    </a-table>

    <!-- 编辑菜单对话框 -->
    <a-modal
      v-model:visible="isEditModalVisible"
      title="编辑菜单"
      @ok="handleEditSubmit"
      @cancel="handleEditCancel"
      :okText="'保存'"
      :cancelText="'取消'"
    >
      <a-form layout="vertical">
        <a-form-item label="中文名称" required>
          <a-input v-model="editForm.chineseName" placeholder="请输入中文名称" />
        </a-form-item>
        <a-form-item label="英文名称" required>
          <a-input v-model="editForm.englishName" placeholder="请输入英文名称" />
        </a-form-item>
        <a-form-item label="图标">
          <a-input v-model="editForm.icon" placeholder="请输入图标名称" />
        </a-form-item>
        <a-form-item label="权限标识" required>
          <a-input v-model="editForm.permission" placeholder="请输入权限标识" />
        </a-form-item>
        <a-form-item label="组件" required>
          <a-input v-model="editForm.component" placeholder="请输入组件名称" />
        </a-form-item>
        <a-form-item label="路径" required>
          <a-input v-model="editForm.path" placeholder="请输入菜单路径" />
        </a-form-item>
        <a-form-item label="类型" required>
          <a-input v-model="editForm.type" placeholder="请输入类型" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model="editForm.status" placeholder="请选择状态">
            <a-select-option value="1">启用</a-select-option>
            <a-select-option value="0">禁用</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import {ref, onMounted, reactive} from 'vue';
import { message } from 'ant-design-vue';
import {requestClient} from "#/api/request";

// 定义数据类型
interface BackendMenu {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  name: string;
  title: string;
  pid: number;
  parentMenu: string;
  icon: string;
  type: string;
  show: boolean;
  orderNo: number;
  component: string;
  redirect: string;
  path: string;
  remark: string;
  homePath: string;
  status: string; // "1" 表示启用，"0" 表示禁用
  meta: {
    title: string;
    icon: string;
    showMenu: boolean;
    hideMenu: boolean;
    ignoreKeepAlive: boolean;
  };
  children: any;
  roles: any;
  key: number;
  value: number;
}

interface Menu {
  id: number;
  chineseName: string;
  englishName: string;
  permission: string;
  icon: string;
  component: string;
  status: boolean;
  path: string;
  type: string;
  createTime: string;
}

// 搜索文本
const searchText = ref('');
// 状态过滤
const selectedStatus = ref<string | null>(null);

// 原始数据
const data = ref<Menu[]>([]);

// 过滤后的数据
const filteredData = ref<Menu[]>([]);

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '中文名称',
    dataIndex: 'chineseName',
    key: 'chineseName',
  },
  {
    title: '英文名称',
    dataIndex: 'englishName',
    key: 'englishName',
  },
  {
    title: '图标',
    dataIndex: 'icon',
    key: 'icon',
    slots: { customRender: 'icon' },
  },
  {
    title: '权限标识',
    dataIndex: 'permission',
    key: 'permission',
  },
  {
    title: '组件',
    dataIndex: 'component',
    key: 'component',
  },
  {
    title: '路径',
    dataIndex: 'path',
    key: 'path',
  },
  {
    title: '类型',
    dataIndex: 'type',
    key: 'type',
  },
  {
    title: '创建时间',
    dataIndex: 'createTime',
    key: 'createTime',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    slots: { customRender: 'status' },
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 处理搜索
const handleSearch = () => {
  const searchValue = searchText.value.trim().toLowerCase();
  filteredData.value = data.value.filter(item => {
    const nameMatch = item.chineseName.toLowerCase().includes(searchValue);
    const statusMatch =
      selectedStatus.value === null ||
      item.status.toString() === selectedStatus.value;
    return nameMatch && statusMatch;
  });
};

// 处理修改状态
const handleStatusChange = async (record: Menu) => {
  const originalStatus = record.status; // 保存原始状态

  // 临时修改前端状态，防止重复点击
  record.status = !record.status;

  try {
    // 发送状态更新请求，将布尔值转换为字符串 "1" 或 "0"
    await requestClient.post(`/auth/menu/update_status`, {
      id: record.id,  // 传递菜单的 ID
      status: record.status ? '1' : '0',  // 将布尔值转换为 "1" 或 "0"
    });

    // 显示成功消息
    message.success(`菜单 "${record.chineseName}" 的状态已修改`);
  } catch (error) {
    console.error('状态修改失败:', error);

    // 恢复原始状态
    record.status = originalStatus;

    // 显示错误提示
    message.error('状态修改失败，请稍后再试');
  }
};

// 处理创建菜单
const handleAdd = () => {
  // 这里可以打开一个对话框，填写新菜单的信息
  message.info('点击了创建菜单按钮');
};

// 编辑菜单对话框的可见状态
const isEditModalVisible = ref(false);

// 当前编辑的菜单记录
const currentEditRecord = ref<Menu | null>(null);

// 编辑菜单表单数据
const editForm = reactive({
  id: 0,
  chineseName: '',
  englishName: '',
  icon: '',
  permission: '',
  component: '',
  status: '1', // "1" 表示启用，"0" 表示禁用
  path: '',
  type: '',
});

const handleEdit = (record: Menu) => {
  currentEditRecord.value = record;

  // 填充编辑表单数据
  editForm.id = record.id;
  editForm.chineseName = record.chineseName;
  editForm.englishName = record.englishName;
  editForm.icon = record.icon;
  editForm.permission = record.permission;
  editForm.component = record.component;
  editForm.status = record.status ? '1' : '0';
  editForm.path = record.path;
  editForm.type = record.type;

  // 打开编辑对话框
  isEditModalVisible.value = true;
};

const handleEditSubmit = async () => {
  try {
    // 发送状态更新请求，将布尔值转换为字符串 "1" 或 "0"
    await requestClient.post(`/auth/menu/update`, {
      id: editForm.id,  // 传递菜单的 ID
      chineseName: editForm.chineseName,
      englishName: editForm.englishName,
      icon: editForm.icon,
      permission: editForm.permission,
      component: editForm.component,
      status: editForm.status,  // "1" 或 "0"
      path: editForm.path,
      type: editForm.type,
    });

    // 显示成功消息
    message.success(`菜单 "${editForm.chineseName}" 已成功更新`);

    // 关闭编辑对话框
    isEditModalVisible.value = false;

    // 刷新菜单数据
    fetchData();
  } catch (error) {
    console.error('编辑菜单失败:', error);
    message.error('编辑菜单失败，请稍后再试');
  }
};

const handleEditCancel = () => {
  isEditModalVisible.value = false;
};

const handleDelete = async (record: Menu) => {
  try {
    // 调用后端接口删除菜单
    await requestClient.delete(`/auth/menu/${record.id}`);

    message.success(`菜单 "${record.chineseName}" 已删除`);

    // 只刷新菜单数据，而不是刷新整个页面
    fetchData();
  } catch (error) {
    console.error('删除菜单失败:', error);
    message.error('删除失败，请稍后再试');
  }
};

// 获取后端数据并映射到前端接口
const fetchData = async () => {
  try {
    const response = await requestClient.get("/auth/menu/all");
    console.log(response); // 打印整个响应对象，应该是数组

    // 直接使用 response 作为数据数组
    const backendData: BackendMenu[] = response;

    // 映射后端数据到前端 Menu 接口
    data.value = backendData.map(item => ({
      id: item.ID,
      chineseName: item.title,
      englishName: item.name,
      icon: item.icon || 'menu', // 设置默认图标
      permission: item.roles || '',
      component: item.component,
      status: item.status === '1',
      path: item.path,
      type: item.type || '',
      createTime: new Date(item.CreatedAt).toLocaleString(),
    }));

    // 初始化过滤后的数据
    handleSearch();
  } catch (error) {
    console.error(error);
    message.error('请求失败，请稍后再试');
  }
};

// 在组件挂载时获取数据
onMounted(() => {
  fetchData();
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
