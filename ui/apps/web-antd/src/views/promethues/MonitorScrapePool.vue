<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">采集池管理</h2>
      <div class="page-description">管理和监控Prometheus采集池及其相关配置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入采集池名称" 
          class="search-input"
        >
          <template #prefix>
            <SearchOutlined class="search-icon" />
          </template>
        </a-input>
        <a-button type="primary" class="action-button" @click="handleSearch">
          <template #icon>
            <SearchOutlined />
          </template>
          搜索
        </a-button>
        <a-button class="action-button reset-button" @click="handleReset">
          <template #icon>
            <ReloadOutlined />
          </template>
          重置
        </a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" class="add-button" @click="showAddModal">
          <template #icon>
            <PlusOutlined />
          </template>
          新增采集池
        </a-button>
      </div>
    </div>

    <!-- 采集池列表表格 -->
    <div class="dashboard-card table-container">
      <a-table 
        :columns="columns" 
        :data-source="data" 
        row-key="id" 
        :pagination="false"
        class="custom-table"
        :scroll="{ x: 1200 }"
      >
        <!-- Prometheus实例列 -->
        <template #prometheus_instances="{ record }">
          <div class="tag-container">
            <a-tag v-for="instance in record.prometheus_instances" :key="instance" class="tech-tag prometheus-tag">
              {{ instance }}
            </a-tag>
          </div>
        </template>
        
        <!-- AlertManager实例列 -->
        <template #alert_manager_instances="{ record }">
          <div class="tag-container">
            <a-tag v-for="instance in record.alert_manager_instances" :key="instance" class="tech-tag alert-tag">
              {{ instance }}
            </a-tag>
          </div>
        </template>
        
        <!-- IP标签列 -->
        <template #external_labels="{ record }">
          <div class="tag-container">
            <template v-if="record.external_labels && record.external_labels.filter((label: string) => label.trim() !== '').length > 0">
              <a-tag v-for="label in record.external_labels" :key="label" class="tech-tag label-tag">
                <span class="label-key">{{ label.split(',')[0] }}</span>
                <span class="label-separator">:</span>
                <span class="label-value">{{ label.split(',')[1] }}</span>
              </a-tag>
            </template>
            <a-tag v-else class="tech-tag empty-tag">无标签</a-tag>
          </div>
        </template>
        
        <!-- 操作列 -->
        <template #action="{ record }">
          <div class="action-column">
            <a-tooltip title="编辑资源信息">
              <a-button type="primary" shape="circle" class="edit-button" @click="handleEdit(record)">
                <template #icon>
                  <Icon icon="clarity:note-edit-line" />
                </template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除资源">
              <a-button type="primary" danger shape="circle" class="delete-button" @click="handleDelete(record)">
                <template #icon>
                  <Icon icon="ant-design:delete-outlined" />
                </template>
              </a-button>
            </a-tooltip>
          </div>
        </template>
      </a-table>

      <!-- 分页器 -->
      <div class="pagination-container">
        <a-pagination 
          v-model:current="current" 
          v-model:pageSize="pageSizeRef" 
          :page-size-options="pageSizeOptions"
          :total="total" 
          show-size-changer 
          @change="handlePageChange" 
          @showSizeChange="handleSizeChange" 
          class="custom-pagination"
        >
          <template #buildOptionText="props">
            <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
            <span v-else>全部</span>
          </template>
        </a-pagination>
      </div>
    </div>

    <!-- 新增采集池模态框 -->
    <a-modal 
      title="新增采集池" 
      v-model:visible="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :width="700"
      class="custom-modal"
    >
      <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="采集池名称" name="name" :rules="[{ required: true, message: '请输入采集池名称' }]">
                <a-input v-model:value="addForm.name" placeholder="请输入采集池名称" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">实例配置</div>
          <!-- 动态Prometheus实例表单项 -->
          <a-form-item v-for="(instance, index) in addForm.prometheus_instances" :key="instance.key"
            :label="index === 0 ? 'Prometheus实例' : ''" :name="['prometheus_instances', index, 'value']"
            :rules="{ required: true, message: '请输入Prometheus实例IP' }">
            <div class="dynamic-input-container">
              <a-input v-model:value="instance.value" placeholder="请输入Prometheus实例IP" class="dynamic-input" />
              <MinusCircleOutlined v-if="addForm.prometheus_instances.length > 1" class="dynamic-delete-button"
                @click="removePrometheusInstance(instance)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addPrometheusInstance">
              <PlusOutlined />
              添加Prometheus实例
            </a-button>
          </a-form-item>

          <!-- 动态AlertManager实例表单项 -->
          <a-form-item v-for="(instance, index) in addForm.alert_manager_instances" :key="instance.key"
            :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
            :rules="{ required: true, message: '请输入AlertManager实例IP' }">
            <div class="dynamic-input-container">
              <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP" class="dynamic-input" />
              <MinusCircleOutlined v-if="addForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
                @click="removeAlertManagerInstance(instance)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addAlertManagerInstance">
              <PlusOutlined />
              添加AlertManager实例
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">采集配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集间隔" name="scrape_interval">
                <a-input-number v-model:value="addForm.scrape_interval" :min="1" placeholder="请输入采集间隔（秒）" class="full-width" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集超时" name="scrape_timeout">
                <a-input-number v-model:value="addForm.scrape_timeout" :min="1" placeholder="请输入采集超时时间（秒）" class="full-width" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="支持告警" name="support_alert">
                <a-switch v-model:checked="addForm.support_alert" class="tech-switch" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="支持记录" name="support_record">
                <a-switch v-model:checked="addForm.support_record" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">远程配置</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="远程写入地址" name="remote_write_url">
                <a-input v-model:value="addForm.remote_write_url" placeholder="请输入远程写入地址" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="远程超时（秒）" name="remote_timeout_seconds">
                <a-input-number v-model:value="addForm.remote_timeout_seconds" :min="1" placeholder="请输入远程超时（秒）" class="full-width" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="远程读取地址" name="remote_read_url">
                <a-input v-model:value="addForm.remote_read_url" placeholder="请输入远程读取地址" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="AlertManager地址" name="alert_manager_url">
                <a-input v-model:value="addForm.alert_manager_url" placeholder="请输入AlertManager地址" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">文件路径配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="规则文件路径" name="rule_file_path">
                <a-input v-model:value="addForm.rule_file_path" placeholder="请输入规则文件路径" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="预聚合文件路径" name="record_file_path">
                <a-input v-model:value="addForm.record_file_path" placeholder="请输入预聚合文件路径" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">标签配置</div>
          <!-- 动态IP标签表单项 -->
          <a-form-item v-for="(label, index) in addForm.external_labels" :key="label.key"
            :label="index === 0 ? '采集池IP标签' : ''">
            <div class="label-input-group">
              <a-input v-model:value="label.labelKey" placeholder="请输入标签Key" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="label.labelValue" placeholder="请输入标签Value" class="label-value-input" />
              <MinusCircleOutlined v-if="addForm.external_labels.length > 1" class="dynamic-delete-button"
                @click="removeExternalLabel(label)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addExternalLabel">
              <PlusOutlined />
              添加IP标签
            </a-button>
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑采集池模态框 -->
    <a-modal 
      title="编辑采集池" 
      v-model:visible="isEditModalVisible" 
      @ok="handleUpdate" 
      @cancel="closeEditModal"
      :width="700"
      class="custom-modal"
    >
      <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="采集池名称" name="name" :rules="[{ required: true, message: '请输入采集池名称' }]">
                <a-input v-model:value="editForm.name" placeholder="请输入采集池名称" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">实例配置</div>
          <!-- 动态Prometheus实例表单项 -->
          <a-form-item v-for="(instance, index) in editForm.prometheus_instances" :key="instance.key"
            :label="index === 0 ? 'Prometheus实例' : ''" :name="['prometheus_instances', index, 'value']"
            :rules="{ required: true, message: '请输入Prometheus实例IP' }">
            <div class="dynamic-input-container">
              <a-input v-model:value="instance.value" placeholder="请输入Prometheus实例IP" class="dynamic-input" />
              <MinusCircleOutlined v-if="editForm.prometheus_instances.length > 1" class="dynamic-delete-button"
                @click="removePrometheusInstanceEdit(instance)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addPrometheusInstanceEdit">
              <PlusOutlined />
              添加Prometheus实例
            </a-button>
          </a-form-item>

          <!-- 动态AlertManager实例表单项 -->
          <a-form-item v-for="(instance, index) in editForm.alert_manager_instances" :key="instance.key"
            :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
            :rules="{ required: true, message: '请输入AlertManager实例IP' }">
            <div class="dynamic-input-container">
              <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP" class="dynamic-input" />
              <MinusCircleOutlined v-if="editForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
                @click="removeAlertManagerInstanceEdit(instance)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addAlertManagerInstanceEdit">
              <PlusOutlined />
              添加AlertManager实例
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">采集配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集间隔" name="scrape_interval">
                <a-input-number v-model:value="editForm.scrape_interval" :min="1" placeholder="请输入采集间隔（秒）" class="full-width" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集超时" name="scrape_timeout">
                <a-input-number v-model:value="editForm.scrape_timeout" :min="1" placeholder="请输入采集超时时间（秒）" class="full-width" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="支持告警" name="support_alert">
                <a-switch v-model:checked="editForm.support_alert" class="tech-switch" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="支持记录" name="support_record">
                <a-switch v-model:checked="editForm.support_record" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">远程配置</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="远程写入地址" name="remote_write_url">
                <a-input v-model:value="editForm.remote_write_url" placeholder="请输入远程写入地址" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="远程超时（秒）" name="remote_timeout_seconds">
                <a-input-number v-model:value="editForm.remote_timeout_seconds" :min="1" placeholder="请输入远程超时（秒）" class="full-width" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="远程读取地址" name="remote_read_url">
                <a-input v-model:value="editForm.remote_read_url" placeholder="请输入远程读取地址" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="AlertManager地址" name="alert_manager_url">
                <a-input v-model:value="editForm.alert_manager_url" placeholder="请输入AlertManager地址" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">文件路径配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="规则文件路径" name="rule_file_path">
                <a-input v-model:value="editForm.rule_file_path" placeholder="请输入规则文件路径" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="预聚合文件路径" name="record_file_path">
                <a-input v-model:value="editForm.record_file_path" placeholder="请输入预聚合文件路径" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">标签配置</div>
          <!-- 动态IP标签表单项 -->
          <a-form-item v-for="(label, index) in editForm.external_labels" :key="label.key"
            :label="index === 0 ? '采集池IP标签' : ''">
            <div class="label-input-group">
              <a-input v-model:value="label.labelKey" placeholder="请输入标签Key" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="label.labelValue" placeholder="请输入标签Value" class="label-value-input" />
              <MinusCircleOutlined v-if="editForm.external_labels.length > 1" class="dynamic-delete-button"
                @click="removeExternalLabelEdit(label)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addExternalLabelEdit">
              <PlusOutlined />
              添加IP标签
            </a-button>
          </a-form-item>
        </div>
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
import { getMonitorScrapePoolListApi, createMonitorScrapePoolApi, deleteMonitorScrapePoolApi, updateMonitorScrapePoolApi } from '#/api/core/prometheus_scrape_pool'
import type { createMonitorScrapePoolReq, updateMonitorScrapePoolReq, MonitorScrapePoolItem } from '#/api/core/prometheus_scrape_pool'
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
    width: 80,
  },
  {
    title: '采集池名称',
    dataIndex: 'name',
    key: 'name',
    width: 150,
  },
  {
    title: 'Prometheus实例',
    dataIndex: 'prometheus_instances',
    key: 'prometheus_instances',
    slots: { customRender: 'prometheus_instances' },
    width: 200,
  },
  {
    title: 'AlertManager实例',
    dataIndex: 'alert_manager_instances',
    key: 'alert_manager_instances',
    slots: { customRender: 'alert_manager_instances' },
    width: 200,
  },
  {
    title: '采集池IP标签',
    dataIndex: 'external_labels',
    key: 'external_labels',
    slots: { customRender: 'external_labels' },
    width: 200,
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
    width: 120,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
  },
  {
    title: '远程写入地址',
    dataIndex: 'remote_write_url',
    key: 'remote_write_url',
    width: 200,
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 120,
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
    const response = await getMonitorScrapePoolListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    data.value = response.items;
    total.value = response.total;
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
.monitor-page {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 8px;
}

.page-description {
  color: #666;
  font-size: 14px;
}

.dashboard-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  padding: 20px;
  margin-bottom: 24px;
  transition: all 0.3s;
}

.custom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  gap: 16px;
  align-items: center;
}

.search-input {
  width: 250px;
  border-radius: 4px;
  transition: all 0.3s;
}

.search-input:hover,
.search-input:focus {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.search-icon {
  color: #bfbfbf;
}

.action-button {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  border-radius: 4px;
  transition: all 0.3s;
}

.reset-button {
  background-color: #f5f5f5;
  color: #595959;
  border-color: #d9d9d9;
}

.reset-button:hover {
  background-color: #e6e6e6;
  border-color: #b3b3b3;
}

.add-button {
  background: linear-gradient(45deg, #1890ff, #36bdf4);
  border: none;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.4);
}

.add-button:hover {
  background: linear-gradient(45deg, #096dd9, #1890ff);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.5);
}

.table-container {
  overflow: hidden;
}

.custom-table {
  margin-top: 8px;
}

:deep(.ant-table) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.ant-table-thead > tr > th) {
  background-color: #f7f9fc;
  font-weight: 600;
  color: #1f1f1f;
  padding: 16px 12px;
}

:deep(.ant-table-tbody > tr > td) {
  padding: 12px;
}

:deep(.ant-table-tbody > tr:hover > td) {
  background-color: #f0f7ff;
}

.tag-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tech-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}

.prometheus-tag {
  background-color: #e6f7ff;
  color: #0958d9;
  border-left: 3px solid #1890ff;
}

.alert-tag {
  background-color: #fff7e6;
  color: #d46b08;
  border-left: 3px solid #fa8c16;
}

.label-tag {
  background-color: #f6ffed;
  color: #389e0d;
  border-left: 3px solid #52c41a;
}

.empty-tag {
  background-color: #f5f5f5;
  color: #8c8c8c;
}

.label-key {
  font-weight: 600;
}

.label-separator {
  margin: 0 4px;
  color: #8c8c8c;
}

.label-value {
  color: #555;
}

.action-column {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.edit-button {
  background: #1890ff;
  border: none;
  box-shadow: 0 2px 4px rgba(24, 144, 255, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.edit-button:hover {
  background: #096dd9;
  transform: scale(1.05);
}

.delete-button {
  background: #ff4d4f;
  border: none;
  box-shadow: 0 2px 4px rgba(255, 77, 79, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.delete-button:hover {
  background: #cf1322;
  transform: scale(1.05);
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

.custom-pagination {
  margin-right: 12px;
}

/* 模态框样式 */
:deep(.custom-modal .ant-modal-content) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.custom-modal .ant-modal-header) {
  padding: 20px 24px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

:deep(.custom-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.custom-modal .ant-modal-body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

:deep(.custom-modal .ant-modal-footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 表单样式 */
.custom-form {
  width: 100%;
}

.form-section {
  margin-bottom: 28px;
  padding: 0;
  position: relative;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 16px;
  padding-left: 12px;
  border-left: 4px solid #1890ff;
}

:deep(.custom-form .ant-form-item-label > label) {
  font-weight: 500;
  color: #333;
}

.full-width {
  width: 100%;
}

:deep(.tech-switch) {
  background-color: rgba(0, 0, 0, 0.25);
}

:deep(.tech-switch.ant-switch-checked) {
  background: linear-gradient(45deg, #1890ff, #36cfc9);
}

.dynamic-input-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dynamic-input {
  width: 100%;
}

.dynamic-delete-button {
  cursor: pointer;
  color: #ff4d4f;
  font-size: 18px;
  transition: all 0.3s;
}

.dynamic-delete-button:hover {
  color: #cf1322;
  transform: scale(1.1);
}

.add-dynamic-button {
  width: 100%;
  margin-top: 8px;
  background: #f5f5f5;
  border: 1px dashed #d9d9d9;
  color: #595959;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.add-dynamic-button:hover {
  color: #1890ff;
  border-color: #1890ff;
  background: #f0f7ff;
}

.label-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label-key-input,
.label-value-input {
  flex: 1;
}

.label-separator {
  font-weight: bold;
  color: #8c8c8c;
}
</style>