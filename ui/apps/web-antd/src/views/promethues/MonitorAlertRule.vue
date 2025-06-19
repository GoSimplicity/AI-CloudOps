<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">AlertRule管理</h2>
      <div class="page-description">管理和配置Prometheus告警规则及其相关设置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入AlertRule名称" 
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
          新增AlertRule
        </a-button>
      </div>
    </div>

    <!-- AlertRule列表表格 -->
    <div class="dashboard-card table-container">
      <a-table 
        :columns="columns" 
        :data-source="data" 
        row-key="id" 
        :loading="loading" 
        :pagination="false"
        class="custom-table"
        :scroll="{ x: 1200 }"
      >
        <template #expr="{ record }">
          <div style="max-width: 300px; word-break: break-all">
            {{ record.expr }}
          </div>
        </template>
        
        <!-- 标签组列 -->
        <template #labels="{ record }">
          <div class="tag-container">
            <template v-if="record.labels && record.labels.length && record.labels[0] !== ''">
              <a-tag v-for="label in record.labels" :key="label" class="tech-tag label-tag">
                <span class="label-key">{{ label.split(',')[0] }}</span>
                <span class="label-separator">:</span>
                <span class="label-value">{{ label.split(',')[1] }}</span>
              </a-tag>
            </template>
            <a-tag v-else class="tech-tag empty-tag">无标签</a-tag>
          </div>
        </template>
        
        <!-- 注解列 -->
        <template #annotations="{ record }">
          <div class="tag-container">
            <template v-if="record.annotations && record.annotations.length && record.annotations[0] !== ''">
              <a-tag v-for="annotation in record.annotations" :key="annotation" class="tech-tag annotation-tag">
                <span class="label-key">{{ annotation.split(',')[0] }}</span>
                <span class="label-separator">:</span>
                <span class="label-value">{{ annotation.split(',')[1] }}</span>
              </a-tag>
            </template>
            <a-tag v-else class="tech-tag empty-tag">无注解</a-tag>
          </div>
        </template>
        
        <!-- 严重性列 -->
        <template #severity="{ record }">
          <a-tag :class="['tech-tag', `severity-${record.severity}`]">
            {{ record.severity }}
          </a-tag>
        </template>
        
        <!-- 启用状态列 -->
        <template #enable="{ record }">
          <a-tag :class="['tech-tag', record.enable === 1 ? 'status-enabled' : 'status-disabled']">
            {{ record.enable === 1 ? '启用' : '禁用' }}
          </a-tag>
        </template>
        
        <!-- IP地址列 -->
        <template #ip_address="{ record }">
          <span>{{ record.ip_address || '-' }}</span>
        </template>
        
        <!-- 操作列 -->
        <template #action="{ record }">
          <div class="action-column">
            <a-tooltip title="编辑资源信息">
              <a-button type="primary" shape="circle" class="edit-button" @click="showEditModal(record)">
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

    <!-- 新增AlertRule模态框 -->
    <a-modal 
      title="新增AlertRule" 
      v-model:visible="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :width="700"
      class="custom-modal"
    >
      <a-form :model="addForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="名称" name="name" :rules="[{ required: true, message: '请输入名称' }]">
                <a-input v-model:value="addForm.name" placeholder="请输入名称" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="所属实例池" name="pool_id" :rules="[{ required: true, message: '请选择所属实例池' }]">
                <a-select v-model:value="addForm.pool_id" placeholder="请选择所属实例池">
                  <a-select-option v-for="pool in scrapePools" :key="pool.id" :value="pool.id">
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="发送组" name="send_group_id">
                <a-select v-model:value="addForm.send_group_id" placeholder="请选择发送组">
                  <a-select-option v-for="group in sendGroups" :key="group.id" :value="group.id">
                    {{ group.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="目标地址" name="ip_address" :rules="[{ required: true, message: '请输入IP地址和端口' }]">
                <div class="ip-port-container">
                  <a-input 
                    v-model:value="addForm.ip" 
                    placeholder="请输入IP地址" 
                    class="ip-input"
                  />
                  <span class="separator">:</span>
                  <a-input 
                    v-model:value="addForm.port" 
                    placeholder="端口" 
                    class="port-input"
                  />
                </div>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">规则配置</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="表达式" name="expr">
                <a-input v-model:value="addForm.expr" placeholder="请输入表达式" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item>
                <a-button type="primary" class="action-button" @click="validateAddExpression(addForm.expr)">验证表达式</a-button>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="严重性" name="severity">
                <a-select v-model:value="addForm.severity" placeholder="请选择严重性">
                  <a-select-option value="critical">Critical</a-select-option>
                  <a-select-option value="warning">Warning</a-select-option>
                  <a-select-option value="info">Info</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="持续时间" name="for_time">
                <a-input v-model:value="addForm.for_time" placeholder="例如: 10s" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">标签配置</div>
          <!-- 动态标签表单项 -->
          <a-form-item v-for="(label, index) in addForm.labels" :key="label.key"
            :label="index === 0 ? '分组标签' : ''">
            <div class="label-input-group">
              <a-input v-model:value="label.labelKey" placeholder="标签名" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="label.labelValue" placeholder="标签值" class="label-value-input" />
              <MinusCircleOutlined v-if="index > 0" class="dynamic-delete-button"
                @click="removeLabel(label)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addLabel">
              <PlusOutlined />
              添加标签
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">注解配置</div>
          <!-- 动态注解表单项 -->
          <a-form-item v-for="(annotation, index) in addForm.annotations" :key="annotation.key"
            :label="index === 0 ? '注解' : ''">
            <div class="label-input-group">
              <a-input v-model:value="annotation.labelKey" placeholder="注解名" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="annotation.labelValue" placeholder="标签值" class="label-value-input" />
              <MinusCircleOutlined v-if="index > 0" class="dynamic-delete-button"
                @click="removeAnnotation(annotation)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addAnnotation">
              <PlusOutlined />
              添加注解
            </a-button>
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑AlertRule模态框 -->
    <a-modal 
      title="编辑AlertRule" 
      v-model:visible="isEditModalVisible" 
      @ok="handleEdit" 
      @cancel="closeEditModal"
      :width="700"
      class="custom-modal"
    >
      <a-form :model="editForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="名称" name="name" :rules="[{ required: true, message: '请输入名称' }]">
                <a-input v-model:value="editForm.name" placeholder="请输入名称" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="所属实例池" name="pool_id" :rules="[{ required: true, message: '请选择所属实例池' }]">
                <a-select v-model:value="editForm.pool_id" placeholder="请选择所属实例池">
                  <a-select-option v-for="pool in scrapePools" :key="pool.id" :value="pool.id">
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="发送组" name="send_group_id">
                <a-select v-model:value="editForm.send_group_id" placeholder="请选择发送组">
                  <a-select-option v-for="group in sendGroups" :key="group.id" :value="group.id">
                    {{ group.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="目标地址" name="ip_address" :rules="[{ required: true, message: '请输入IP地址和端口' }]">
                <div class="ip-port-container">
                  <a-input 
                    v-model:value="editForm.ip" 
                    placeholder="请输入IP地址" 
                    class="ip-input"
                  />
                  <span class="separator">:</span>
                  <a-input 
                    v-model:value="editForm.port" 
                    placeholder="端口" 
                    class="port-input"
                  />
                </div>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="启用" name="enable">
                <a-switch v-model:checked="editForm.enable" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">规则配置</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="表达式" name="expr">
                <a-input v-model:value="editForm.expr" placeholder="请输入表达式" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item>
                <a-button type="primary" class="action-button" @click="validateEditExpression">验证表达式</a-button>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="严重性" name="severity">
                <a-select v-model:value="editForm.severity" placeholder="请选择严重性">
                  <a-select-option value="critical">Critical</a-select-option>
                  <a-select-option value="warning">Warning</a-select-option>
                  <a-select-option value="info">Info</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="持续时间" name="for_time">
                <a-input v-model:value="editForm.for_time" placeholder="例如: 10s" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">标签配置</div>
          <!-- 动态标签表单项 -->
          <a-form-item v-for="(label, index) in editForm.labels" :key="label.key"
            :label="index === 0 ? '分组标签' : ''">
            <div class="label-input-group">
              <a-input v-model:value="label.labelKey" placeholder="标签名" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="label.labelValue" placeholder="标签值" class="label-value-input" />
              <MinusCircleOutlined v-if="index > 0" class="dynamic-delete-button"
                @click="removeEditLabel(label)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addEditLabel">
              <PlusOutlined />
              添加标签
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">注解配置</div>
          <!-- 动态注解表单项 -->
          <a-form-item v-for="(annotation, index) in editForm.annotations" :key="annotation.key"
            :label="index === 0 ? '注解' : ''">
            <div class="label-input-group">
              <a-input v-model:value="annotation.labelKey" placeholder="注解名" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="annotation.labelValue" placeholder="标签值" class="label-value-input" />
              <MinusCircleOutlined v-if="index > 0" class="dynamic-delete-button"
                @click="removeEditAnnotation(annotation)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addEditAnnotation">
              <PlusOutlined />
              添加注解
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
import {
  getAlertRulesListApi,
  createAlertRuleApi,
  updateAlertRuleApi,
  deleteAlertRuleApi,
} from '#/api/core/prometheus_alert_rule';
import { getAllAlertManagerPoolApi } from '#/api/core/prometheus_alert_pool';
import { validateExprApi } from '#/api/core/prometheus_alert_rule';
import { getAllMonitorSendGroupApi } from '#/api/core/prometheus_send_group';
import { Icon } from '@iconify/vue';
import type { AlertRuleItem } from '#/api/core/prometheus_alert_rule';

interface ScrapePool {
  id: number;
  name: string;
}

interface SendGroup {
  id: number;
  name: string;
}

// 数据源
const data = ref<AlertRuleItem[]>([]);
const scrapePools = ref<ScrapePool[]>([]);
const sendGroups = ref<SendGroup[]>([]);

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 搜索文本
const searchText = ref('');

// 加载状态
const loading = ref(false);

// 表格列配置
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => a.id - b.id,
  },
  {
    title: '名称',
    dataIndex: 'name',
    key: 'name',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => a.name.localeCompare(b.name),
  },
  {
    title: '所属实例池ID',
    dataIndex: 'pool_id',
    key: 'pool_id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => (a.pool_id || 0) - (b.pool_id || 0),
  },
  {
    title: '绑定发送组ID',
    dataIndex: 'send_group_id',
    key: 'send_group_id',
    sorter: (a: AlertRuleItem, b: AlertRuleItem) => (a.send_group_id || 0) - (b.send_group_id || 0),
  },
  {
    title: 'IP地址',
    dataIndex: 'ip_address',
    key: 'ip_address',
    slots: { customRender: 'ip_address' },
  },
  {
    title: '严重性',
    dataIndex: 'severity',
    key: 'severity',
    slots: { customRender: 'severity' },
    sorter: (a: AlertRuleItem, b: AlertRuleItem) =>
      a.severity.localeCompare(b.severity),
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
  },
  {
    title: '是否启用',
    dataIndex: 'enable',
    key: 'enable',
    slots: { customRender: 'enable' },
  },
  {
    title: '标签',
    dataIndex: 'labels',
    key: 'labels',
    slots: { customRender: 'labels' },
  },
  {
    title: '注解',
    dataIndex: 'annotations',
    key: 'annotations',
    slots: { customRender: 'annotations' },
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 120,
  },
];

// 模态框状态和表单
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);

// 处理表格变化
const handlePageChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchAlertRules();
};

const handleSizeChange = (_: number, size: number) => {
  pageSizeRef.value = size;
  fetchAlertRules();
};

const removeEditLabel = (label: any) => {
  const index = editForm.labels.indexOf(label);
  if (index !== -1) {
    editForm.labels.splice(index, 1);
  }
};

const removeLabel = (label: any) => {
  const index = addForm.labels.indexOf(label);
  if (index !== -1) {
    addForm.labels.splice(index, 1);
  }
};

const addLabel = () => {
  addForm.labels.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const addEditLabel = () => {
  editForm.labels.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const addAnnotation = () => {
  addForm.annotations.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const addEditAnnotation = () => {
  editForm.annotations.push({ labelKey: '', labelValue: '', key: Date.now() });
};

const removeAnnotation = (annotation: any) => {
  const index = addForm.annotations.indexOf(annotation);
  if (index !== -1) {
    addForm.annotations.splice(index, 1);
  }
};

const removeEditAnnotation = (annotation: any) => {
  const index = editForm.annotations.indexOf(annotation);
  if (index !== -1) {
    editForm.annotations.splice(index, 1);
  }
};

// 获取实例池数据
const fetchScrapePools = async () => {
  try {
    const response = await getAllAlertManagerPoolApi();
    scrapePools.value = response.items;
  } catch (error: any) {
    message.error(error.message || '获取实例池数据失败');
    console.error(error);
  }
};

// 获取发送组数据
const fetchSendGroups = async () => {
  try {
    const response = await getAllMonitorSendGroupApi();
    sendGroups.value = response.items;
  } catch (error: any) {
    message.error(error.message || '获取发送组数据失败');
    console.error(error);
  }
};

// 搜索处理
const handleSearch = () => {
  current.value = 1;
  fetchAlertRules();
};

// 重置处理
const handleReset = () => {
  searchText.value = '';
  fetchAlertRules();
};

// 新增表单
const addForm = reactive({
  name: '',
  pool_id: null,
  send_group_id: null,
  ip: '',
  port: '',
  enable: true,
  expr: '',
  severity: '',
  grafana_link: '',
  for_time: '',
  labels: [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 },
    { labelKey: 'alert_rule_id', labelValue: '', key: Date.now() + 3 }
  ],
  annotations: [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 }
  ],
});

// 编辑表单
const editForm = reactive({
  id: 0,
  name: '',
  pool_id: null,
  send_group_id: null,
  ip: '',
  port: '',
  enable: true,
  expr: '',
  severity: '',
  grafana_link: '',
  for_time: '',
  labels: [{ labelKey: '', labelValue: '', key: Date.now() }],
  annotations: [{ labelKey: '', labelValue: '', key: Date.now() }],
});

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

const resetAddForm = () => {
  addForm.name = '';
  addForm.pool_id = null;
  addForm.send_group_id = null;
  addForm.ip = '';
  addForm.port = '';
  addForm.enable = true;
  addForm.expr = '';
  addForm.severity = '';
  addForm.grafana_link = '';
  addForm.for_time = '';
  addForm.labels = [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 },
    { labelKey: 'alert_rule_id', labelValue: '', key: Date.now() + 3 }
  ];
  addForm.annotations = [
    { labelKey: 'severity', labelValue: '', key: Date.now() },
    { labelKey: 'alert_send_group', labelValue: '', key: Date.now() + 2 }
  ];
};

// 关闭新增模态框
const closeAddModal = () => {
  isAddModalVisible.value = false;
};

// 显示编辑模态框
const showEditModal = (record: AlertRuleItem) => {
  // 解析IP地址（假设格式为ip:port）
  const ipParts = record.ip_address?.split(':') || ['', ''];
  
  Object.assign(editForm, {
    id: record.id,
    name: record.name,
    pool_id: record.pool_id || null,
    send_group_id: record.send_group_id || null,
    ip: ipParts[0] || '',
    port: ipParts[1] || '',
    enable: record.enable,
    expr: record.expr,
    severity: record.severity,
    grafana_link: record.grafana_link,
    for_time: record.for_time,
    labels: record.labels ?
      record.labels.map((value: string) => {
        const [labelKey, labelValue] = value.split(',');
        return {
          labelKey: labelKey || '',
          labelValue: labelValue || '',
          key: Date.now()
        };
      }) : [],
    annotations: record.annotations ?
      record.annotations.map((value: string) => {
        const [labelKey, labelValue] = value.split(',');
        return {
          labelKey: labelKey || '',
          labelValue: labelValue || '',
          key: Date.now()
        };
      }) : [],
  });
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交新增 AlertRule
const handleAdd = async () => {
  try {
    // 表单验证逻辑
    if (addForm.name === '' || addForm.pool_id === 0) {
      message.error('请填写所有必填项');
      return;
    }

    if (!addForm.ip || !addForm.port) {
      message.error('请填写IP地址和端口');
      return;
    }

    // 组合IP地址
    const ip_address = `${addForm.ip}:${addForm.port}`;

    // 创建符合 createAlertRuleReq 类型的数据
    const formData = {
      name: addForm.name,
      pool_id: addForm.pool_id,
      send_group_id: addForm.send_group_id,
      ip_address,
      enable: addForm.enable,
      expr: addForm.expr,
      severity: addForm.severity,
      grafana_link: addForm.grafana_link,
      for_time: addForm.for_time,
      labels: addForm.labels
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
      annotations: addForm.annotations
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
    };

    await createAlertRuleApi(formData);
    message.success('新增AlertRule成功');
    fetchAlertRules();
    closeAddModal();
  } catch (error: any) {
    message.error(error.message || '新增AlertRule失败');
    console.error(error);
  }
};

// 提交更新AlertRule
const handleEdit = async () => {
  try {
    if (editForm.name === '' || editForm.pool_id === 0) {
      message.error('请填写所有必填项');
      return;
    }

    if (!editForm.ip || !editForm.port) {
      message.error('请填写IP地址和端口');
      return;
    }

    // 组合IP地址
    const ip_address = `${editForm.ip}:${editForm.port}`;

    // 创建符合 updateAlertRuleReq 类型的数据
    const formData = {
      id: editForm.id,
      name: editForm.name,
      pool_id: editForm.pool_id,
      send_group_id: editForm.send_group_id,
      ip_address,
      enable: editForm.enable,
      expr: editForm.expr,
      severity: editForm.severity,
      grafana_link: editForm.grafana_link,
      for_time: editForm.for_time,
      labels: editForm.labels
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
      annotations: editForm.annotations
        .filter(item => item.labelKey.trim() !== '' && item.labelValue.trim() !== '')
        .map(item => `${item.labelKey},${item.labelValue}`),
    };

    await updateAlertRuleApi(formData);
    message.success('更新AlertRule成功');
    fetchAlertRules();
    closeEditModal();
  } catch (error: any) {
    message.error(error.message || '更新AlertRule失败');
    console.error(error);
  }
};

// 处理删除AlertRule
const handleDelete = (record: AlertRuleItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除AlertRule "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await deleteAlertRuleApi(record.id);
        message.success('AlertRule已删除');
        fetchAlertRules();
      } catch (error: any) {
        message.error(error.message || '删除AlertRule失败');
        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 获取AlertRules数据
const fetchAlertRules = async () => {
  try {
    loading.value = true;
    const response = await getAlertRulesListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    data.value = response.items;
    total.value = response.total;
  } catch (error: any) {
    message.error(error.message || '获取AlertRules数据失败');
    console.error(error);
  } finally {
    loading.value = false;
  }
};

// 验证表达式的方法（新增）
const validateAddExpression = async (expr: string) => {
  try {
    const payload = { promql_expr: expr };
    const result = await validateExprApi(payload);
    message.success('验证表达式成功', result.message);
    return true;
  } catch (error: any) {
    message.error(error.message || '验证表达式失败');
    console.error(error);
    return false;
  }
};

// 验证表达式的方法（编辑）
const validateEditExpression = async () => {
  try {
    const payload = { promql_expr: editForm.expr };
    const result = await validateExprApi(payload);
    message.success('验证表达式成功', result.message);
    return true;
  } catch (error: any) {
    message.error(error.message || '验证表达式失败');
    console.error(error);
    return false;
  }
};

// 在组件加载时获取数据
onMounted(() => {
  fetchAlertRules();
  fetchScrapePools();
  fetchSendGroups();
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

.severity-critical {
  background-color: #fff1f0;
  color: #cf1322;
  border-left: 3px solid #ff4d4f;
}

.severity-warning {
  background-color: #fff7e6;
  color: #d46b08;
  border-left: 3px solid #fa8c16;
}

.severity-info {
  background-color: #e6f7ff;
  color: #0958d9;
  border-left: 3px solid #1890ff;
}

.status-enabled {
  background-color: #f6ffed;
  color: #389e0d;
  border-left: 3px solid #52c41a;
}

.status-disabled {
  background-color: #fff1f0;
  color: #cf1322;
  border-left: 3px solid #ff4d4f;
}

.label-tag {
  background-color: #f6ffed;
  color: #389e0d;
  border-left: 3px solid #52c41a;
}

.annotation-tag {
  background-color: #f0f5ff;
  color: #1d39c4;
  border-left: 3px solid #2f54eb;
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

/* IP地址和端口输入框样式 */
.ip-port-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ip-input {
  flex: 3;
}

.port-input {
  flex: 1;
}

.separator {
  font-weight: bold;
  color: #8c8c8c;
  font-size: 16px;
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