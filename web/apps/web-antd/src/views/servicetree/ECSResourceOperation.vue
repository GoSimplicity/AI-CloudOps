<template>
  <div>
    <!-- 操作工具栏 -->
    <div class="toolbar">
      <div class="search-area">
        <a-input v-model="searchText" placeholder="请输入ECS资源名称" style="width: 200px; margin-right: 16px;"
          @keyup.enter="handleSearch" />
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="handleAddResource">新增ECS资源</a-button>
      </div>
    </div>

    <!-- 资源列表 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="ID" :pagination="{ pageSize: 10 }">
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleEditResource(record)">编辑</a-button>
          <a-button type="link" danger @click="handleDeleteResource(record)">删除</a-button>
          <a-button type="link" @click="handleBindToNode(record)">绑定到服务树</a-button>
          <a-button type="link" @click="handleUnbindFromNode(record)">解绑服务树</a-button>
        </a-space>
      </template>
    </a-table>

    <!-- 新增资源模态框 -->
    <a-modal v-model:visible="isCreateModalVisible" title="新增资源" @ok="handleCreateECS" @cancel="handleCancel">
      <a-form :model="createForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" ref="createFormRef">
        <a-form-item label="资源名称" name="instanceName" :rules="[{ required: true, message: '请输入资源名称' }]">
          <a-input v-model:value="createForm.instanceName" placeholder="请输入资源名称" />
        </a-form-item>

        <a-form-item label="IP地址" name="ipAddr" :rules="[{ required: true, message: '请输入IP地址' }]">
          <a-input v-model:value="createForm.ipAddr" placeholder="请输入IP地址" />
        </a-form-item>

        <a-form-item label="主机名" name="hostname">
          <a-input v-model:value="createForm.hostname" placeholder="请输入主机名" />
        </a-form-item>

        <a-form-item label="操作系统" name="osName">
          <a-input v-model:value="createForm.osName" placeholder="请输入系统名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-input v-model:value="createForm.description" placeholder="请输入资源描述" />
        </a-form-item>

        <!-- 支持多标签输入 -->
        <a-form-item label="标签" name="tags">
          <a-select mode="tags" v-model:value="createForm.tags" placeholder="请输入标签" style="width: 100%">
            <a-select-option v-for="tag in createForm.tags" :key="tag" :value="tag">
              {{ tag }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="供应商" name="vendor" :rules="[
          { required: true, type: 'string', message: '请选择供应商' }
        ]">
          <a-select v-model:value="createForm.vendor" placeholder="请选择供应商" style="width: 100%">
            <a-select-option :value="'1'">个人</a-select-option>
            <a-select-option :value="'2'">阿里云</a-select-option>
            <a-select-option :value="'3'">华为云</a-select-option>
            <a-select-option :value="'4'">腾讯云</a-select-option>
            <a-select-option :value="'5'">AWS</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { getAllECSResources, createECSResources, deleteECSResources } from '#/api';
import type { ResourceEcs } from '#/api';

const vendorMap: { [key: string]: string } = {
  '1': '个人',
  '2': '阿里云',
  '3': '华为云',
  '4': '腾讯云',
  '5': 'AWS',
};

// 资源数据类型
interface Resource {
  ID: number;
  instanceName: string;
  createResourceType: number;
  status: string;
  description: string;
  CreatedAt: string;
  vendor: string;
  ipAddr: string;
}

const createForm = reactive({
  instanceName: '',
  description: '',
  tags: [] as string[],
  vendor: null as string | null, // 初始化为 null 或字符串
  hostname: '',
  ipAddr: '',
  osName: '',
});

// 资源数据
const data = reactive<ResourceEcs[]>([]);

// 获取资源数据
const fetchResources = async () => {
  try {
    const response = await getAllECSResources();
    data.splice(0, data.length, ...response);
    handleSearch(); // 初始化过滤后的数据
  } catch (error) {
    console.log(error);
    message.error('获取ECS资源数据失败');
  }
};

// 搜索文本
const searchText = ref('');
// 过滤后的数据
const filteredData = ref<ResourceEcs[]>(data);

const isCreateModalVisible = ref(false);

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'ID',
    key: 'ID',
  },
  {
    title: '资源名称',
    dataIndex: 'instanceName',
    key: 'instanceName',
  },
  {
    title: '供应商',
    dataIndex: 'vendor',
    key: 'vendor',
    customRender: (vendor: string) => vendorMap[vendor.value] || '未知',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
  },
  {
    title: 'IP地址',
    dataIndex: 'ipAddr',
    key: 'ipAddr',
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    ellipsis: true,
  },
  {
    title: '创建时间',
    dataIndex: 'CreatedAt',
    key: 'CreatedAt',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 250,
  },
];

// 处理搜索
const handleSearch = () => {
  if (searchText.value.trim() === '') {
    filteredData.value = data;
  } else {
    const search = searchText.value.trim().toLowerCase();
    filteredData.value = data.filter(resource =>
      resource.instanceName.toLowerCase().includes(search)
    );
  }
};

const handleCreateECS = async () => {
  // 校验供应商是否选择
  if (createForm.vendor === null) {
    message.error('请选择供应商');
    return;
  }

  // 校验资源名称是否填写
  if (!createForm.instanceName) {
    message.error('请输入资源名称');
    return;
  }

  // 清理标签数据，移除空白标签
  createForm.tags = createForm.tags.filter(tag => tag.trim() !== '');

  try {
    // 调用接口创建资源，注意使用 await，并确保 createECSResources 是异步函数
    await createECSResources({
      instanceName: createForm.instanceName,
      description: createForm.description,
      tags: createForm.tags,
      vendor: createForm.vendor,
      hostname: createForm.hostname,
      ipAddr: createForm.ipAddr,
      osName: createForm.osName,
    });

    // 显示成功提示
    message.success('新增ECS资源成功');

    setTimeout(() => {
      location.reload();
    }, 500);

    // 隐藏模态框
    isCreateModalVisible.value = false;

  } catch (error) {
    // 捕获异常并显示错误提示
    console.error('创建ECS资源失败', error);
    message.error('创建ECS资源失败，请稍后再试');
  }
};


const handleCancel = () => {
  isCreateModalVisible.value = false;
};

// 处理新增资源
const handleAddResource = () => {
  Object.assign(createForm, {
    instanceName: '',
    description: '',
    tags: [],
    vendor: null,
    hostname: '',
    ipAddr: '',
    osName: '',
  });
  isCreateModalVisible.value = true;
};

// 处理编辑资源
const handleEditResource = (record: Resource) => {
  // TODO: 打开编辑资源对话框并调用接口保存更新的资源信息
  message.info(`编辑ECS资源 "${record.instanceName}"`);
};

const handleDeleteResource = (record: Resource) => {
  // 弹出确认对话框，防止误删除
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除资源 "${record.instanceName}" 吗？`,
    onOk: async () => {
      try {
        await deleteECSResources(record.ID);
        // 从本地数据中删除该资源
        const index = data.findIndex(item => item.ID === record.ID);
        if (index !== -1) {
          data.splice(index, 1);  // 删除资源
          handleSearch();  // 重新过滤数据
        message.success(`资源 "${record.instanceName}" 已成功删除`);
        }
      } catch (error) {
        console.error('删除资源失败', error);
        message.error(`删除资源 "${record.instanceName}" 失败，请稍后再试`);
      }
    },
  });
};

// 处理绑定到服务树
const handleBindToNode = (record: Resource) => {
  // TODO: 打开绑定对话框并调用接口绑定资源到服务树节点
  message.info(`绑定ECS资源 "${record.instanceName}" 到服务树节点`);
};

// 处理解绑服务树
const handleUnbindFromNode = (record: Resource) => {
  // TODO: 调用接口解绑资源从服务树节点
  message.info(`解绑ECS资源 "${record.instanceName}" 从服务树节点`);
};

onMounted(() => {
  fetchResources();
});
</script>

<style scoped>
.toolbar {
  padding: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-area {
  display: flex;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 8px;
}
</style>
