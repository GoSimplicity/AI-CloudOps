<template>
  <div>
    <!-- 查询和操作工具栏 -->
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input v-model:value="searchText" placeholder="请输入采集池名称" style="width: 200px" />
        <a-button type="primary" size="middle" @click="handleSearch">
          <template #icon>
            <SearchOutlined />
          </template>
          搜索
        </a-button>
        <a-button @click="handleReset">
          <template #icon>
            <ReloadOutlined />
          </template>
          重置
        </a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增采集池</a-button>
      </div>
    </div>

    <!-- 采集池列表表格 -->
    <a-table :columns="columns" :data-source="data" row-key="id" :pagination="false">
      <!-- Prometheus实例列 -->
      <template #prometheus_instances="{ record }">
        <a-tag v-for="instance in record.prometheus_instances" :key="instance">{{ instance }}</a-tag>
      </template>
      <!-- AlertManager实例列 -->
      <template #alert_manager_instances="{ record }">
        <a-tag v-for="instance in record.alert_manager_instances" :key="instance">{{ instance }}</a-tag>
      </template>
      <!-- IP标签列 -->
      <template #external_labels="{ record }">
        <template
          v-if="record.external_labels && record.external_labels.filter((label: string) => label.trim() !== '').length > 0">
          <a-tag v-for="label in record.external_labels" :key="label">
            {{ label.split(',')[0] }}: {{ label.split(',')[1] }}
          </a-tag>
        </template>
        <a-tag v-else color="default">无标签</a-tag>
      </template>
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑资源信息">
            <a-button type="link" @click="handleEdit(record)">
              <template #icon>
                <Icon icon="clarity:note-edit-line" style="font-size: 22px" />
              </template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="删除资源">
            <a-button type="link" danger @click="handleDelete(record)">
              <template #icon>
                <Icon icon="ant-design:delete-outlined" style="font-size: 22px" />
              </template>
            </a-button>
          </a-tooltip>
        </a-space>
      </template>
    </a-table>

    <!-- 分页器 -->
    <a-pagination v-model:current="current" v-model:pageSize="pageSizeRef" :page-size-options="pageSizeOptions"
      :total="total" show-size-changer @change="handlePageChange" @showSizeChange="handleSizeChange" class="pagination">
      <template #buildOptionText="props">
        <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
        <span v-else>全部</span>
      </template>
    </a-pagination>

    <!-- 新增采集池模态框 -->
    <a-modal title="新增采集池" v-model:visible="isAddModalVisible" @ok="handleAdd" @cancel="closeAddModal">
      <a-form ref="addFormRef" :model="addForm" layout="vertical">
        <a-form-item label="采集池名称" name="name" :rules="[{ required: true, message: '请输入采集池名称' }]">
          <a-input v-model:value="addForm.name" placeholder="请输入采集池名称" />
        </a-form-item>

        <!-- 动态Prometheus实例表单项 -->
        <a-form-item v-for="(instance, index) in addForm.prometheus_instances" :key="instance.key"
          :label="index === 0 ? 'Prometheus实例' : ''" :name="['prometheus_instances', index, 'value']"
          :rules="{ required: true, message: '请输入Prometheus实例IP' }">
          <a-input v-model:value="instance.value" placeholder="请输入Prometheus实例IP"
            style="width: 60%; margin-right: 8px" />
          <MinusCircleOutlined v-if="addForm.prometheus_instances.length > 1" class="dynamic-delete-button"
            @click="removePrometheusInstance(instance)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addPrometheusInstance">
            <PlusOutlined />
            添加Prometheus实例
          </a-button>
        </a-form-item>

        <!-- 动态AlertManager实例表单项 -->
        <a-form-item v-for="(instance, index) in addForm.alert_manager_instances" :key="instance.key"
          :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
          :rules="{ required: true, message: '请输入AlertManager实例IP' }">
          <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP"
            style="width: 60%; margin-right: 8px" />
          <MinusCircleOutlined v-if="addForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
            @click="removeAlertManagerInstance(instance)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addAlertManagerInstance">
            <PlusOutlined />
            添加AlertManager实例
          </a-button>
        </a-form-item>

        <a-form-item label="采集间隔" name="scrape_interval">
          <a-input-number v-model:value="addForm.scrape_interval" :min="1" placeholder="请输入采集间隔（秒）" />
        </a-form-item>
        <a-form-item label="采集超时" name="scrape_timeout">
          <a-input-number v-model:value="addForm.scrape_timeout" :min="1" placeholder="请输入采集超时时间（秒）" />
        </a-form-item>
        <a-form-item label="支持告警" name="support_alert">
          <a-switch v-model:checked="addForm.support_alert" />
        </a-form-item>
        <a-form-item label="支持记录" name="support_record">
          <a-switch v-model:checked="addForm.support_record" />
        </a-form-item>
        <a-form-item label="远程写入地址" name="remote_write_url">
          <a-input v-model:value="addForm.remote_write_url" placeholder="请输入远程写入地址" />
        </a-form-item>
        <a-form-item label="远程超时（秒）" name="remote_timeout_seconds">
          <a-input-number v-model:value="addForm.remote_timeout_seconds" :min="1" placeholder="请输入远程超时（秒）" />
        </a-form-item>
        <a-form-item label="远程读取地址" name="remote_read_url">
          <a-input v-model:value="addForm.remote_read_url" placeholder="请输入远程读取地址" />
        </a-form-item>
        <a-form-item label="AlertManager地址" name="alert_manager_url">
          <a-input v-model:value="addForm.alert_manager_url" placeholder="请输入AlertManager地址" />
        </a-form-item>
        <a-form-item label="规则文件路径" name="rule_file_path">
          <a-input v-model:value="addForm.rule_file_path" placeholder="请输入规则文件路径" />
        </a-form-item>
        <a-form-item label="预聚合文件路径" name="record_file_path">
          <a-input v-model:value="addForm.record_file_path" placeholder="请输入预聚合文件路径" />
        </a-form-item>

        <!-- 动态IP标签表单项 -->
        <a-form-item v-for="(label, index) in addForm.external_labels" :key="label.key"
          :label="index === 0 ? '采集池IP标签' : ''">
          <a-space>
            <a-input v-model:value="label.labelKey" placeholder="请输入标签Key" style="width: 120px" />
            <a-input v-model:value="label.labelValue" placeholder="请输入标签Value" style="width: 120px" />
            <MinusCircleOutlined v-if="addForm.external_labels.length > 1" class="dynamic-delete-button"
              @click="removeExternalLabel(label)" />
          </a-space>
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addExternalLabel">
            <PlusOutlined />
            添加IP标签
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑采集池模态框 -->
    <a-modal title="编辑采集池" v-model:visible="isEditModalVisible" @ok="handleUpdate" @cancel="closeEditModal">
      <a-form ref="editFormRef" :model="editForm" layout="vertical">
        <a-form-item label="采集池名称" name="name" :rules="[{ required: true, message: '请输入采集池名称' }]">
          <a-input v-model:value="editForm.name" placeholder="请输入采集池名称" />
        </a-form-item>

        <!-- 动态Prometheus实例表单项 -->
        <a-form-item v-for="(instance, index) in editForm.prometheus_instances" :key="instance.key"
          :label="index === 0 ? 'Prometheus实例' : ''" :name="['prometheus_instances', index, 'value']"
          :rules="{ required: true, message: '请输入Prometheus实例IP' }">
          <a-input v-model:value="instance.value" placeholder="请输入Prometheus实例IP"
            style="width: 60%; margin-right: 8px" />
          <MinusCircleOutlined v-if="editForm.prometheus_instances.length > 1" class="dynamic-delete-button"
            @click="removePrometheusInstanceEdit(instance)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addPrometheusInstanceEdit">
            <PlusOutlined />
            添加Prometheus实例
          </a-button>
        </a-form-item>

        <!-- 动态AlertManager实例表单项 -->
        <a-form-item v-for="(instance, index) in editForm.alert_manager_instances" :key="instance.key"
          :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
          :rules="{ required: true, message: '请输入AlertManager实例IP' }">
          <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP"
            style="width: 60%; margin-right: 8px" />
          <MinusCircleOutlined v-if="editForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
            @click="removeAlertManagerInstanceEdit(instance)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addAlertManagerInstanceEdit">
            <PlusOutlined />
            添加AlertManager实例
          </a-button>
        </a-form-item>

        <a-form-item label="采集间隔" name="scrape_interval">
          <a-input-number v-model:value="editForm.scrape_interval" :min="1" placeholder="请输入采集间隔（秒）" />
        </a-form-item>
        <a-form-item label="采集超时" name="scrape_timeout">
          <a-input-number v-model:value="editForm.scrape_timeout" :min="1" placeholder="请输入采集超时时间（秒）" />
        </a-form-item>
        <a-form-item label="支持告警" name="support_alert">
          <a-switch v-model:checked="editForm.support_alert" />
        </a-form-item>
        <a-form-item label="支持记录" name="support_record">
          <a-switch v-model:checked="editForm.support_record" />
        </a-form-item>
        <a-form-item label="远程写入地址" name="remote_write_url">
          <a-input v-model:value="editForm.remote_write_url" placeholder="请输入远程写入地址" />
        </a-form-item>
        <a-form-item label="远程超时（秒）" name="remote_timeout_seconds">
          <a-input-number v-model:value="editForm.remote_timeout_seconds" :min="1" placeholder="请输入远程超时（秒）" />
        </a-form-item>
        <a-form-item label="远程读取地址" name="remote_read_url">
          <a-input v-model:value="editForm.remote_read_url" placeholder="请输入远程读取地址" />
        </a-form-item>
        <a-form-item label="AlertManager地址" name="alert_manager_url">
          <a-input v-model:value="editForm.alert_manager_url" placeholder="请输入AlertManager地址" />
        </a-form-item>
        <a-form-item label="规则文件路径" name="rule_file_path">
          <a-input v-model:value="editForm.rule_file_path" placeholder="请输入规则文件路径" />
        </a-form-item>
        <a-form-item label="预聚合文件路径" name="record_file_path">
          <a-input v-model:value="editForm.record_file_path" placeholder="请输入预聚合文件路径" />
        </a-form-item>

        <!-- 动态IP标签表单项 -->
        <a-form-item v-for="(label, index) in editForm.external_labels" :key="label.key"
          :label="index === 0 ? '采集池IP标签' : ''">
          <a-space>
            <a-input v-model:value="label.labelKey" placeholder="请输入标签Key" style="width: 120px" />
            <a-input v-model:value="label.labelValue" placeholder="请输入标签Value" style="width: 120px" />
            <MinusCircleOutlined v-if="editForm.external_labels.length > 1" class="dynamic-delete-button"
              @click="removeExternalLabelEdit(label)" />
          </a-space>
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addExternalLabelEdit">
            <PlusOutlined />
            添加IP标签
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined
} from '@ant-design/icons-vue';
import { getMonitorScrapePoolListApi, createMonitorScrapePoolApi, deleteMonitorScrapePoolApi, updateMonitorScrapePoolApi, getMonitorScrapePoolTotalApi } from '#/api'
import type { createMonitorScrapePoolReq, updateMonitorScrapePoolReq, MonitorScrapePoolItem } from '#/api'
import { Icon } from '@iconify/vue';
import type { FormInstance } from 'ant-design-vue';
interface DynamicItem {
  value: string;
  key: number;
}

interface LabelItem {
  labelKey: string;
  labelValue: string;
  key: number;
}

// 从后端获取数据
const data = ref<MonitorScrapePoolItem[]>([]);

// 搜索文本
const searchText = ref('');

const handleReset = () => {
  searchText.value = '';
  fetchResources();
};

// 处理搜索
const handleSearch = () => {
  current.value = 1;
  fetchResources();
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchResources();
};

// 处理分页变化
const handlePageChange = (page: number) => {
  current.value = page;
  fetchResources();
};

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 表格列配置
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '采集池名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: 'Prometheus实例',
    dataIndex: 'prometheus_instances',
    key: 'prometheus_instances',
    slots: { customRender: 'prometheus_instances' },
  },
  {
    title: 'AlertManager实例',
    dataIndex: 'alert_manager_instances',
    key: 'alert_manager_instances',
    slots: { customRender: 'alert_manager_instances' },
  },
  {
    title: '采集池IP标签',
    dataIndex: 'external_labels',
    key: 'external_labels',
    slots: { customRender: 'external_labels' },
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
  },
  {
    title: '远程写入地址',
    dataIndex: 'remote_write_url',
    key: 'remote_write_url',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

const addFormRef = ref<FormInstance>();
const editFormRef = ref<FormInstance>();

const isAddModalVisible = ref(false);
const addForm = reactive({
  name: '',
  prometheus_instances: [] as DynamicItem[],
  alert_manager_instances: [] as DynamicItem[],
  scrape_interval: 10,
  scrape_timeout: 10,
  remote_timeout_seconds: 10,
  support_alert: false,
  support_record: false,
  external_labels: [] as LabelItem[],
  remote_write_url: '',
  remote_read_url: '',
  alert_manager_url: '',
  rule_file_path: '',
  record_file_path: '',
});

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.scrape_interval = 10;
  addForm.scrape_timeout = 10;
  addForm.support_alert = false;
  addForm.support_record = false;
  addForm.remote_write_url = '';
  addForm.record_file_path = '';
  addForm.remote_timeout_seconds = 10;
  addForm.remote_read_url = '';
  addForm.alert_manager_url = '';
  addForm.rule_file_path = '';
  addForm.external_labels = [];
  addForm.prometheus_instances = [];
  addForm.alert_manager_instances = [];
};

// 编辑相关状态
const isEditModalVisible = ref(false);
const editForm = reactive({
  id: 0,
  name: '',
  prometheus_instances: [] as DynamicItem[],
  alert_manager_instances: [] as DynamicItem[],
  scrape_interval: 10,
  scrape_timeout: 10,
  remote_timeout_seconds: 10,
  support_alert: false,
  support_record: false,
  external_labels: [] as LabelItem[],
  remote_write_url: '',
  remote_read_url: '',
  alert_manager_url: '',
  rule_file_path: '',
  record_file_path: '',
});

// 动态表单项操作方法 - 新增表单
const addPrometheusInstance = () => {
  addForm.prometheus_instances.push({
    value: '',
    key: Date.now(),
  });
};

const removePrometheusInstance = (item: DynamicItem) => {
  const index = addForm.prometheus_instances.indexOf(item);
  if (index !== -1) {
    addForm.prometheus_instances.splice(index, 1);
  }
};

const addAlertManagerInstance = () => {
  addForm.alert_manager_instances.push({
    value: '',
    key: Date.now(),
  });
};

const removeAlertManagerInstance = (item: DynamicItem) => {
  const index = addForm.alert_manager_instances.indexOf(item);
  if (index !== -1) {
    addForm.alert_manager_instances.splice(index, 1);
  }
};

const addExternalLabel = () => {
  addForm.external_labels.push({
    labelKey: '',
    labelValue: '',
    key: Date.now(),
  });
};

const removeExternalLabel = (item: LabelItem) => {
  const index = addForm.external_labels.indexOf(item);
  if (index !== -1) {
    addForm.external_labels.splice(index, 1);
  }
};

// 动态表单项操作方法 - 编辑表单
const addPrometheusInstanceEdit = () => {
  editForm.prometheus_instances.push({
    value: '',
    key: Date.now(),
  });
};

const removePrometheusInstanceEdit = (item: DynamicItem) => {
  const index = editForm.prometheus_instances.indexOf(item);
  if (index !== -1) {
    editForm.prometheus_instances.splice(index, 1);
  }
};

const addAlertManagerInstanceEdit = () => {
  editForm.alert_manager_instances.push({
    value: '',
    key: Date.now(),
  });
};

const removeAlertManagerInstanceEdit = (item: DynamicItem) => {
  const index = editForm.alert_manager_instances.indexOf(item);
  if (index !== -1) {
    editForm.alert_manager_instances.splice(index, 1);
  }
};

const addExternalLabelEdit = () => {
  editForm.external_labels.push({
    labelKey: '',
    labelValue: '',
    key: Date.now(),
  });
};

const removeExternalLabelEdit = (item: LabelItem) => {
  const index = editForm.external_labels.indexOf(item);
  if (index !== -1) {
    editForm.external_labels.splice(index, 1);
  }
};

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 显示编辑模态框
const handleEdit = (record: MonitorScrapePoolItem) => {
  // 预填充表单数据
  editForm.id = record.id;
  editForm.name = record.name;
  editForm.scrape_interval = record.scrape_interval;
  editForm.scrape_timeout = record.scrape_timeout;
  editForm.support_alert = record.support_alert;
  editForm.support_record = record.support_record;
  editForm.remote_write_url = record.remote_write_url;
  editForm.remote_timeout_seconds = record.remote_timeout_seconds;
  editForm.remote_read_url = record.remote_read_url;
  editForm.alert_manager_url = record.alert_manager_url;
  editForm.rule_file_path = record.rule_file_path;
  editForm.record_file_path = record.record_file_path;
  editForm.external_labels = record.external_labels
    ? record.external_labels
      .filter((value: string) => value.trim() !== '') // 过滤空字符串
      .map((value: string) => {
        const parts = value.split(',');
        return {
          labelKey: parts[0] || '',
          labelValue: parts[1] || '',
          key: Date.now()
        };
      })
    : [];
  editForm.prometheus_instances = record.prometheus_instances ?
    record.prometheus_instances.map(value => ({ value, key: Date.now() })) : [];
  editForm.alert_manager_instances = record.alert_manager_instances ?
    record.alert_manager_instances.map(value => ({ value, key: Date.now() })) : [];

  isEditModalVisible.value = true;
};

// 关闭新增模态框
const closeAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = false;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交新增采集池
const handleAdd = async () => {
  try {
    await addFormRef.value?.validate();

    // 转换动态表单项数据为API所需格式
    const formData: createMonitorScrapePoolReq = {
      ...addForm,
      prometheus_instances: addForm.prometheus_instances.map(item => item.value),
      alert_manager_instances: addForm.alert_manager_instances.map(item => item.value),
      external_labels: addForm.external_labels
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '') // 过滤空键值
        .map(item => `${item.labelKey},${item.labelValue}`),
    };

    await createMonitorScrapePoolApi(formData);
    message.success('新增采集池成功');
    fetchResources();
    closeAddModal();
  } catch (error: any) {
    message.error(error.message || '新增采集池失败');
    console.error(error);
  }
};

// 处理删除采集池
const handleDelete = (record: MonitorScrapePoolItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除采集池 "${record.name}" 吗？`,
    onOk: async () => {
      try {
        await deleteMonitorScrapePoolApi(record.id);
        message.success('删除采集池成功');
        fetchResources();

      } catch (error: any) {
        message.error(error.message || '删除采集池失败');
        console.error(error);
      }
    },
  });
};

// 提交更新采集池
const handleUpdate = async () => {
  try {
    await editFormRef.value?.validate();

    // 转换动态表单项数据为API所需格式
    const formData: updateMonitorScrapePoolReq = {
      ...editForm,
      prometheus_instances: editForm.prometheus_instances.map(item => item.value),
      alert_manager_instances: editForm.alert_manager_instances.map(item => item.value),
      external_labels: editForm.external_labels
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '') // 过滤空键值
        .map(item => `${item.labelKey},${item.labelValue}`),
    };

    await updateMonitorScrapePoolApi(formData);
    message.success('更新采集池成功');
    fetchResources();
    closeEditModal();
  } catch (error: any) {
    message.error(error.message || '更新采集池失败');
    console.error(error);
  }
};

const fetchResources = async () => {
  try {
    const response = await getMonitorScrapePoolListApi(current.value, pageSizeRef.value, searchText.value);
    data.value = response;
    total.value = await getMonitorScrapePoolTotalApi();
  } catch (error: any) {

    message.error(error.message || '获取采集池数据失败');
    console.error(error);
  }
};

onMounted(() => {
  fetchResources();
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
  gap: 16px;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.pagination {
  margin-top: 16px;
  text-align: right;
  margin-right: 12px;
}

.dynamic-delete-button {
  cursor: pointer;
  position: relative;
  top: 4px;
  font-size: 24px;
  color: #999;
  transition: all 0.3s;
}

.dynamic-delete-button:hover {
  color: #777;
}

.dynamic-delete-button[disabled] {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
