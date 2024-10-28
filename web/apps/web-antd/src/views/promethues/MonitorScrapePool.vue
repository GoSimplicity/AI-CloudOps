<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input v-model="searchText" placeholder="请输入采集池名称" style="width: 200px; margin-right: 16px;" />
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增采集池</a-button>
      </div>
    </div>

    <!-- 采集池列表表格 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="key">
      <!-- Prometheus实例列 -->
      <template #prometheusInstances="{ record }">
        <a-tag v-for="instance in record.prometheusInstances" :key="instance">{{ instance }}</a-tag>
      </template>
      <!-- AlertManager实例列 -->
      <template #alertManagerInstances="{ record }">
        <a-tag v-for="instance in record.alertManagerInstances" :key="instance">{{ instance }}</a-tag>
      </template>
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleEdit(record)">编辑采集池</a-button>
          <a-button type="link" danger @click="handleDelete(record)">删除采集池</a-button>
        </a-space>
      </template>
    </a-table>
    <!-- 新增采集池模态框 -->
    <a-modal title="新增采集池" v-model:visible="isAddModalVisible" @ok="handleAdd" @cancel="closeAddModal">
      <a-form :model="addForm" layout="vertical">
        <a-form-item label="采集池名称" name="name" :rules="[{ required: true, message: '请输入采集池名称' }]">
          <a-input v-model:value="addForm.name" placeholder="请输入采集池名称" />
        </a-form-item>
        <a-form-item label="Prometheus实例" name="prometheusInstances">
          <a-select mode="tags" v-model:value="addForm.prometheusInstances" placeholder="请输入Prometheus实例IP">
          </a-select>
        </a-form-item>
        <a-form-item label="AlertManager实例" name="alertManagerInstances">
          <a-select mode="tags" v-model:value="addForm.alertManagerInstances" placeholder="请输入AlertManager实例IP">
          </a-select>
        </a-form-item>
        <a-form-item label="采集间隔" name="scrapeInterval">
          <a-input-number v-model:value="addForm.scrapeInterval" :min="1" placeholder="请输入采集间隔（秒）" />
        </a-form-item>
        <a-form-item label="采集超时" name="scrapeTimeout">
          <a-input-number v-model:value="addForm.scrapeTimeout" :min="1" placeholder="请输入采集超时时间（秒）" />
        </a-form-item>
        <a-form-item label="支持告警" name="supportAlert">
          <a-select v-model:value="addForm.supportAlert" placeholder="请选择">
            <a-select-option :value="1">支持</a-select-option>
            <a-select-option :value="2">不支持</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="支持记录" name="supportRecord">
          <a-select v-model:value="addForm.supportRecord" placeholder="请选择">
            <a-select-option :value="1">支持</a-select-option>
            <a-select-option :value="2">不支持</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="远程写入地址" name="remoteWriteUrl">
          <a-input v-model:value="addForm.remoteWriteUrl" placeholder="请输入远程写入地址" />
        </a-form-item>
        <a-form-item label="远程超时（秒）" name="remoteTimeoutSeconds">
          <a-input-number v-model:value="addForm.remoteTimeoutSeconds" :min="1" placeholder="请输入远程超时（秒）" />
        </a-form-item>
        <a-form-item label="远程读取地址" name="remoteReadUrl">
          <a-input v-model:value="addForm.remoteReadUrl" placeholder="请输入远程读取地址" />
        </a-form-item>
        <a-form-item label="AlertManager地址" name="alertManagerUrl">
          <a-input v-model:value="addForm.alertManagerUrl" placeholder="请输入AlertManager地址" />
        </a-form-item>
        <a-form-item label="规则文件路径" name="ruleFilePath">
          <a-input v-model:value="addForm.ruleFilePath" placeholder="请输入规则文件路径" />
        </a-form-item>
        <a-form-item label="预聚合文件路径" name="recordFilePath">
          <a-input v-model:value="addForm.recordFilePath" placeholder="请输入预聚合文件路径" />
        </a-form-item>
        <a-form-item label="采集池IP标签" name="externalLabels">
          <a-select mode="tags" v-model:value="addForm.externalLabels" placeholder="请输入采集池IP标签">
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑采集池模态框 -->
    <a-modal title="编辑采集池" v-model:visible="isEditModalVisible" @ok="handleUpdate" @cancel="closeEditModal">
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="采集池名称" name="name" :rules="[{ required: true, message: '请输入采集池名称' }]">
          <a-input v-model:value="editForm.name" placeholder="请输入采集池名称" />
        </a-form-item>
        <a-form-item label="Prometheus实例" name="prometheusInstances">
          <a-select mode="tags" v-model:value="editForm.prometheusInstances" placeholder="请输入Prometheus实例IP">
          </a-select>
        </a-form-item>
        <a-form-item label="AlertManager实例" name="alertManagerInstances">
          <a-select mode="tags" v-model:value="editForm.alertManagerInstances" placeholder="请输入AlertManager实例IP">
          </a-select>
        </a-form-item>
        <a-form-item label="采集间隔" name="scrapeInterval">
          <a-input-number v-model:value="editForm.scrapeInterval" :min="1" placeholder="请输入采集间隔（秒）" />
        </a-form-item>
        <a-form-item label="采集超时" name="scrapeTimeout">
          <a-input-number v-model:value="editForm.scrapeTimeout" :min="1" placeholder="请输入采集超时时间（秒）" />
        </a-form-item>
        <a-form-item label="支持告警" name="supportAlert">
          <a-select v-model:value="editForm.supportAlert" placeholder="请选择">
            <a-select-option :value="1">支持</a-select-option>
            <a-select-option :value="2">不支持</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="支持记录" name="supportRecord">
          <a-select v-model:value="editForm.supportRecord" placeholder="请选择">
            <a-select-option :value="1">支持</a-select-option>
            <a-select-option :value="2">不支持</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="远程写入地址" name="remoteWriteUrl">
          <a-input v-model:value="editForm.remoteWriteUrl" placeholder="请输入远程写入地址" />
        </a-form-item>
        <a-form-item label="远程超时（秒）" name="remoteTimeoutSeconds">
          <a-input-number v-model:value="editForm.remoteTimeoutSeconds" :min="1" placeholder="请输入远程超时（秒）" />
        </a-form-item>
        <a-form-item label="远程读取地址" name="remoteReadUrl">
          <a-input v-model:value="editForm.remoteReadUrl" placeholder="请输入远程读取地址" />
        </a-form-item>
        <a-form-item label="AlertManager地址" name="alertManagerUrl">
          <a-input v-model:value="editForm.alertManagerUrl" placeholder="请输入AlertManager地址" />
        </a-form-item>
        <a-form-item label="规则文件路径" name="ruleFilePath">
          <a-input v-model:value="editForm.ruleFilePath" placeholder="请输入规则文件路径" />
        </a-form-item>
        <a-form-item label="预聚合文件路径" name="recordFilePath">
          <a-input v-model:value="editForm.recordFilePath" placeholder="请输入预聚合文件路径" />
        </a-form-item>
        <a-form-item label="采集池IP标签" name="externalLabels">
          <a-select mode="tags" v-model:value="editForm.externalLabels" placeholder="请输入采集池IP标签">
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>

import { computed, ref, reactive } from 'vue';

import { message, Modal } from 'ant-design-vue';
import { getMonitorScrapePoolApi, createMonitorScrapePoolApi, deleteMonitorScrapePoolApi, updateMonitorScrapePoolApi } from '#/api'

// 定义数据类型
interface ScrapePool {
  ID: number;
  name: string;
  prometheusInstances: string[];
  alertManagerInstances: string[];
  externalLabels: string;
  userId: number;
  remoteReadUrl: string;
  CreatedAt: string;
  scrapeInterval: number;
  scrapeTimeout: number;
  supportAlert: number;
  supportRecord: number;
  remoteWriteUrl: string;
  remoteTimeoutSeconds: number;
  alertManagerUrl: string;
  ruleFilePath: string;
  recordFilePath: string;
}

// 从后端获取数据
const data = ref<ScrapePool[]>([]);

// 搜索文本
const searchText = ref('');
// 过滤后的数据
const filteredData = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  return data.value.filter(item => item.name.toLowerCase().includes(searchValue));
});

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'ID',
    key: 'ID',
  },
  {
    title: '采集池名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: 'Prometheus实例',
    dataIndex: 'prometheusInstances',
    key: 'prometheusInstances',
    slots: { customRender: 'prometheusInstances' },
  },
  {
    title: 'AlertManager实例',
    dataIndex: 'alertManagerInstances',
    key: 'alertManagerInstances',
    slots: { customRender: 'alertManagerInstances' },
  },
  {
    title: '采集池IP标签',
    dataIndex: 'externalLabels',
    key: 'externalLabels',
  },
  {
    title: '创建者',
    dataIndex: 'userId',
    key: 'userId',
  },
  {
    title: '创建时间',
    dataIndex: 'CreatedAt',
    key: 'CreatedAt',
  },
  {
    title: '远程写入地址',
    dataIndex: 'remoteWriteUrl',
    key: 'remoteWriteUrl',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

const isAddModalVisible = ref(false);
const addForm = reactive({
  name: '',
  scrapeInterval: 10,
  scrapeTimeout: 10,
  supportAlert: 1,
  supportRecord: 1,
  remoteWriteUrl: '',
  remoteTimeoutSeconds: 10,
  remoteReadUrl: '',
  alertManagerUrl: '',
  ruleFilePath: '',
  recordFilePath: '',
  externalLabels: [] as string[],
  prometheusInstances: [] as string[],
  alertManagerInstances: [] as string[],
});

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.scrapeInterval = 10;
  addForm.scrapeTimeout = 10;
  addForm.supportAlert = 1;
  addForm.supportRecord = 1;
  addForm.remoteWriteUrl = '';
  addForm.recordFilePath = '';
  addForm.remoteTimeoutSeconds = 10;
  addForm.remoteReadUrl = '';
  addForm.alertManagerUrl = '';
  addForm.ruleFilePath = '';
  addForm.externalLabels = [];
  addForm.prometheusInstances = [];
  addForm.alertManagerInstances = [];
};

// 编辑相关状态
const isEditModalVisible = ref(false);
const editForm = reactive({
  ID: 0,
  name: '',
  scrapeInterval: 10,
  scrapeTimeout: 10,
  supportAlert: 1,
  supportRecord: 1,
  remoteWriteUrl: '',
  remoteTimeoutSeconds: 10,
  remoteReadUrl: '',
  alertManagerUrl: '',
  ruleFilePath: '',
  recordFilePath: '',
  externalLabels: [] as string[],
  prometheusInstances: [] as string[],
  alertManagerInstances: [] as string[],
});

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 显示编辑模态框
const showEditModal = (record: ScrapePool) => {
  // 预填充表单数据
  editForm.ID = record.ID;
  editForm.name = record.name;
  editForm.scrapeInterval = record.scrapeInterval;
  editForm.scrapeTimeout = record.scrapeTimeout;
  editForm.supportAlert = record.supportAlert;
  editForm.supportRecord = record.supportRecord;
  editForm.remoteWriteUrl = record.remoteWriteUrl;
  editForm.remoteTimeoutSeconds = record.remoteTimeoutSeconds;
  editForm.remoteReadUrl = record.remoteReadUrl;
  editForm.alertManagerUrl = record.alertManagerUrl;
  editForm.ruleFilePath = record.ruleFilePath;
  editForm.recordFilePath = record.recordFilePath;
  editForm.externalLabels = record.externalLabels ? record.externalLabels.split(', ') : [];
  editForm.prometheusInstances = record.prometheusInstances || [];
  editForm.alertManagerInstances = record.alertManagerInstances || [];

  isEditModalVisible.value = true;
};

// 关闭新增模态框
const closeAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = false;
};

// 提交新增采集池
const handleAdd = async () => {
  try {
    await createMonitorScrapePoolApi(addForm);
    message.success('新增采集池成功');
    fetchResources(); // 重新获取数据
    closeAddModal();
  } catch (error) {
    message.error('新增采集池失败，请稍后重试');
    console.error(error);
  }
};

// 处理删除采集池
const handleDelete = (record: ScrapePool) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除采集池 "${record.name}" 吗？`,
    onOk: async () => {
      try {
        // 调用删除采集池的 API
        await deleteMonitorScrapePoolApi(record.ID);
        message.success('删除采集池成功');
        fetchResources(); // 重新获取数据，刷新列表
      } catch (error) {
        message.error('删除采集池失败，请稍后重试');
        console.error(error);
      }
    },
  });
};


const handleEdit = (record: ScrapePool) => {
  showEditModal(record);
};

// 提交更新采集池
const handleUpdate = async () => {
  try {
    // 调用更新API
    await updateMonitorScrapePoolApi(editForm);
    message.success('更新采集池成功');
    fetchResources(); // 重新获取数据
    closeEditModal();
  } catch (error) {
    message.error('更新采集池失败，请稍后重试');
    console.error(error);
  }
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

const fetchResources = async () => {
  try {
    // 调用后端 API 获取采集池数据
    const response = await getMonitorScrapePoolApi();

    const formattedResponse = response.map((item: any) => ({
      ...item,
      externalLabels: item.externalLabels.join(', ') // 使用逗号分隔
    }));

    data.value = formattedResponse;
  } catch (error) {
    // 如果请求出错，显示错误信息
    message.error('获取采集池数据失败，请稍后重试');
    console.error(error);
  }
};

// 在组件加载时调用 fetchResources 以获取初始数据
fetchResources();

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
