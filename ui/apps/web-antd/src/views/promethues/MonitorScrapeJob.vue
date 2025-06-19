<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">采集任务管理</h2>
      <div class="page-description">管理和配置Prometheus的采集任务及相关设置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入采集任务名称" 
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
        <a-button type="primary" class="add-button" @click="openAddModal">
          <template #icon>
            <PlusOutlined />
          </template>
          新增采集任务
        </a-button>
      </div>
    </div>

    <!-- 采集任务列表表格 -->
    <div class="dashboard-card table-container">
      <a-spin :spinning="loading">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1200 }"
        >
          <!-- 服务发现类型列 -->
          <template #serviceDiscoveryType="{ record }">
            <a-tag :color="record.service_discovery_type === 'k8s' ? 'blue' : 'green'" class="tech-tag">
              {{ record.service_discovery_type === 'k8s' ? 'Kubernetes' : 'HTTP' }}
            </a-tag>
          </template>
          
          <!-- 关联采集池列 -->
          <template #poolName="{ record }">
            <a-tag color="purple" class="tech-tag">{{ getPoolName(record.pool_id) }}</a-tag>
          </template>
          
          <!-- IP地址+端口列 -->
          <template #ipAddress="{ record }">
            <div class="tag-container">
              <a-tag color="blue" class="tech-tag">
                {{ record.ip_address }}:{{ record.port }}
              </a-tag>
            </div>
          </template>
          
          <!-- 创建者列 -->
          <template #createUserName="{ record }">
            <a-tag color="cyan" class="tech-tag">{{ record.create_user_name }}</a-tag>
          </template>
          
          <!-- 创建时间列 -->
          <template #created_at="{ record }">
            <a-tooltip :title="formatDate(record.created_at)">
              {{ record.created_at }}
            </a-tooltip>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-tooltip title="编辑资源信息">
                <a-button type="primary" shape="circle" class="edit-button" @click="openEditModal(record)">
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
              <span v-else">全部</span>
            </template>
          </a-pagination>
        </div>
      </a-spin>
    </div>

    <!-- 新增采集任务模态框 -->
    <a-modal 
      title="新增采集任务" 
      v-model:visible="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :width="700"
      :confirmLoading="confirmLoading"
      :maskClosable="false"
      class="custom-modal"
    >
      <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集任务名称" name="name" :rules="[{ required: true, message: '请输入采集任务名称' }]">
                <a-input v-model:value="addForm.name" placeholder="请输入采集任务名称" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="启用" name="enable">
                <a-switch v-model:checked="addForm.enable" :checked-children="'启用'" :un-checked-children="'禁用'" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">服务配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="服务发现类型" name="service_discovery_type" :rules="[{ required: true, message: '请选择服务发现类型' }]">
                <a-select v-model:value="addForm.service_discovery_type" placeholder="请选择服务发现类型">
                  <a-select-option value="http">HTTP</a-select-option>
                  <a-select-option value="k8s">Kubernetes</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="协议方案" name="scheme" :rules="[{ required: true, message: '请选择协议方案' }]">
                <a-select v-model:value="addForm.scheme" placeholder="请选择协议方案">
                  <a-select-option value="http">HTTP</a-select-option>
                  <a-select-option value="https">HTTPS</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="监控采集路径" name="metrics_path" :rules="[{ required: true, message: '请输入监控采集路径' }]">
                <a-input v-model:value="addForm.metrics_path" placeholder="请输入监控采集路径" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="关联采集池" name="pool_id" :rules="[{ required: true, message: '请选择关联采集池' }]">
                <a-select v-model:value="addForm.pool_id" placeholder="请选择关联采集池">
                  <a-select-option v-for="pool in pools" :key="pool.id" :value="pool.id">
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">采集配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集间隔（秒）" name="scrape_interval" :rules="[
                { required: true, message: '请输入采集间隔' },
                { type: 'number', min: 1, message: '采集间隔必须大于0' }
              ]">
                <a-input-number v-model:value="addForm.scrape_interval" :min="1" class="full-width" placeholder="请输入采集间隔（秒）" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集超时（秒）" name="scrape_timeout" :rules="[
                { required: true, message: '请输入采集超时' },
                { type: 'number', min: 1, message: '采集超时必须大于0' }
              ]">
                <a-input-number v-model:value="addForm.scrape_timeout" :min="1" class="full-width" placeholder="请输入采集超时（秒）" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="刷新间隔（秒）" name="refresh_interval" :rules="[
                { required: true, message: '请输入刷新间隔' },
                { type: 'number', min: 1, message: '刷新间隔必须大于0' }
              ]">
                <a-input-number v-model:value="addForm.refresh_interval" :min="1" class="full-width" placeholder="请输入刷新间隔（秒）" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="端口" name="port" :rules="[
                { required: true, message: '请输入端口' },
                { type: 'number', min: 1, max: 65535, message: '端口必须在1-65535之间' }
              ]">
                <a-input-number v-model:value="addForm.port" :min="1" :max="65535" class="full-width" placeholder="请输入端口" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">目标地址配置</div>
          <a-form-item label="IP地址" name="ip_address" :rules="[
            { required: true, message: '请输入IP地址' },
            { pattern: /^(\d{1,3}\.){3}\d{1,3}$/, message: '请输入正确的IP地址格式' }
          ]">
            <a-input 
              v-model:value="addForm.ip_address" 
              placeholder="请输入IP地址，如：192.168.1.100"
            />
            <div class="form-help-text">
              <Icon icon="ant-design:info-circle-outlined" />
              请输入单个IP地址，端口配置在上方端口字段
            </div>
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑采集任务模态框 -->
    <a-modal 
      title="编辑采集任务" 
      v-model:visible="isEditModalVisible" 
      @ok="handleUpdate" 
      @cancel="closeEditModal"
      :width="700"
      :confirmLoading="confirmLoading"
      :maskClosable="false"
      class="custom-modal"
    >
      <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form" @submit.prevent>
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集任务名称" name="name" :rules="[{ required: true, message: '请输入采集任务名称' }]">
                <a-input v-model:value="editForm.name" placeholder="请输入采集任务名称" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="启用" name="enable">
                <a-switch v-model:checked="editForm.enable" :checked-children="'启用'" :un-checked-children="'禁用'" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">服务配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="服务发现类型" name="service_discovery_type" :rules="[{ required: true, message: '请选择服务发现类型' }]">
                <a-select v-model:value="editForm.service_discovery_type" placeholder="请选择服务发现类型">
                  <a-select-option value="http">HTTP</a-select-option>
                  <a-select-option value="k8s">Kubernetes</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="协议方案" name="scheme" :rules="[{ required: true, message: '请选择协议方案' }]">
                <a-select v-model:value="editForm.scheme" placeholder="请选择协议方案">
                  <a-select-option value="http">HTTP</a-select-option>
                  <a-select-option value="https">HTTPS</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="监控采集路径" name="metrics_path" :rules="[{ required: true, message: '请输入监控采集路径' }]">
                <a-input v-model:value="editForm.metrics_path" placeholder="请输入监控采集路径" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="关联采集池" name="pool_id" :rules="[{ required: true, message: '请选择关联采集池' }]">
                <a-select v-model:value="editForm.pool_id" placeholder="请选择关联采集池">
                  <a-select-option v-for="pool in pools" :key="pool.id" :value="pool.id">
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">采集配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集间隔（秒）" name="scrape_interval" :rules="[
                { required: true, message: '请输入采集间隔' },
                { type: 'number', min: 1, message: '采集间隔必须大于0' }
              ]">
                <a-input-number v-model:value="editForm.scrape_interval" :min="1" class="full-width" placeholder="请输入采集间隔（秒）" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="采集超时（秒）" name="scrape_timeout" :rules="[
                { required: true, message: '请输入采集超时' },
                { type: 'number', min: 1, message: '采集超时必须大于0' }
              ]">
                <a-input-number v-model:value="editForm.scrape_timeout" :min="1" class="full-width" placeholder="请输入采集超时（秒）" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="刷新间隔（秒）" name="refresh_interval" :rules="[
                { required: true, message: '请输入刷新间隔' },
                { type: 'number', min: 1, message: '刷新间隔必须大于0' }
              ]">
                <a-input-number v-model:value="editForm.refresh_interval" :min="1" class="full-width" placeholder="请输入刷新间隔（秒）" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="端口" name="port" :rules="[
                { required: true, message: '请输入端口' },
                { type: 'number', min: 1, max: 65535, message: '端口必须在1-65535之间' }
              ]">
                <a-input-number v-model:value="editForm.port" :min="1" :max="65535" class="full-width" placeholder="请输入端口" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">目标地址配置</div>
          <a-form-item label="IP地址" name="ip_address" :rules="[
            { required: true, message: '请输入IP地址' },
            { pattern: /^(\d{1,3}\.){3}\d{1,3}$/, message: '请输入正确的IP地址格式' }
          ]">
            <a-input 
              v-model:value="editForm.ip_address" 
              placeholder="请输入IP地址，如：192.168.1.100"
            />
            <div class="form-help-text">
              <Icon icon="ant-design:info-circle-outlined" />
              请输入单个IP地址，端口配置在上方端口字段
            </div>
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import dayjs from 'dayjs';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
} from '@ant-design/icons-vue';
import {
  getMonitorScrapeJobListApi,
  createScrapeJobApi,
  updateScrapeJobApi,
  deleteScrapeJobApi,
} from '#/api/core/prometheus_scrape_job';
import { getAllMonitorScrapePoolApi } from '#/api/core/prometheus_scrape_pool';
import type { MonitorScrapeJobItem, createScrapeJobReq, updateScrapeJobReq } from '#/api/core/prometheus_scrape_job';

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 搜索处理
const handleSearch = () => {
  fetchResources();
};

const handleSizeChange = (_current: number, size: number) => {
  pageSizeRef.value = size;
  fetchResources();
};

// 处理分页变化
const handlePageChange = (page: number) => {
  current.value = page;
  fetchResources();
};

const handleReset = () => {
  searchText.value = '';
  fetchResources();
};

interface Pool {
  id: number;
  name: string;
}

// 数据源（待从后端获取）
const data = ref<MonitorScrapeJobItem[]>([]);

// 搜索文本
const searchText = ref('');

// 格式化日期
const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss');
};

// 表格列配置 - 更新IP地址列显示
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    width: 80,
  },
  {
    title: '采集任务名称',
    dataIndex: 'name',
    key: 'name',
    width: 150,
  },
  {
    title: '服务发现类型',
    dataIndex: 'service_discovery_type',
    key: 'service_discovery_type',
    slots: { customRender: 'serviceDiscoveryType' },
    width: 120,
  },
  {
    title: '监控采集路径',
    dataIndex: 'metrics_path',
    key: 'metrics_path',
    width: 130,
  },
  {
    title: '关联采集池',
    dataIndex: 'pool_id',
    key: 'pool_id',
    slots: { customRender: 'poolName' },
    width: 120,
  },
  {
    title: '目标地址',
    dataIndex: 'ip_address',
    key: 'ip_address',
    slots: { customRender: 'ipAddress' },
    width: 150,
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
    slots: { customRender: 'createUserName' },
    width: 100,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    slots: { customRender: 'created_at' },
    width: 170,
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 120,
  },
];

// 模态框相关状态
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);
const currentRecord = ref<MonitorScrapeJobItem | null>(null);
const confirmLoading = ref(false);

// 表单引用
const addFormRef = ref();
const editFormRef = ref();

// 表单数据模型 - 修改为字符串类型
const addForm = reactive({
  name: '',
  enable: true,
  service_discovery_type: 'http',
  metrics_path: '/metrics',
  scheme: 'http',
  scrape_interval: 15,
  scrape_timeout: 5,
  pool_id: null as number | null,
  refresh_interval: 30,
  port: 9100,
  ip_address: '', // 字符串类型，单个IP
  relabel_configs_yaml_string: '',
  kube_config_file_path: '',
  tls_ca_file_path: '',
  tls_ca_content: '',
  bearer_token: '',
  bearer_token_file: '',
  kubernetes_sd_role: '',
});

const editForm = reactive({
  id: 0,
  name: '',
  enable: true,
  service_discovery_type: 'http',
  metrics_path: '/metrics',
  scheme: 'http',
  scrape_interval: 15,
  scrape_timeout: 5,
  pool_id: null as number | null,
  refresh_interval: 30,
  port: 9100,
  ip_address: '', // 字符串类型，单个IP
  relabel_configs_yaml_string: '',
  kube_config_file_path: '',
  tls_ca_file_path: '',
  tls_ca_content: '',
  bearer_token: '',
  bearer_token_file: '',
  kubernetes_sd_role: '',
});

// 采集池列表
const pools = ref<Pool[]>([]);

// 获取采集池数据
const fetchPools = async () => {
  try {
    const response = await getAllMonitorScrapePoolApi();
    if (response && response.items) {
      pools.value = response.items.map((pool: any) => ({
        id: pool.id,
        name: pool.name,
      }));
    } else {
      pools.value = [];
      console.error('获取采集池数据格式异常', response);
    }
  } catch (error: any) {
    pools.value = [];
    message.error(error.message || '获取采集池数据失败');
  }
};

// 获取采集任务数据
const loading = ref(false);
const fetchResources = async () => {
  if (current.value < 1) current.value = 1;
  loading.value = true;
  try {
    const response = await getMonitorScrapeJobListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    if (response && response.items) {
      data.value = response.items.map((item: any) => ({
        ...item,
        // 处理IP地址字段 - 如果是数组则取第一个，否则保持原值
        ip_address: Array.isArray(item.ip_address) ? item.ip_address[0] || '' : (item.ip_address || '')
      }));
      total.value = response.total;
    } else {
      data.value = [];
      total.value = 0;
      console.error('获取采集任务数据格式异常', response);
    }
  } catch (error: any) {
    data.value = [];
    total.value = 0;
    message.error(error.message || '获取采集任务数据失败');
  } finally {
    loading.value = false;
  }
};

// 获取采集池名称
const getPoolName = (poolId: number) => {
  const pool = pools.value.find(p => p.id === poolId);
  return pool ? pool.name : '未知';
};

// 打开新增模态框
const openAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 关闭新增模态框
const closeAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = false;
};

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.enable = true;
  addForm.service_discovery_type = 'http';
  addForm.metrics_path = '/metrics';
  addForm.scheme = 'http';
  addForm.scrape_interval = 15;
  addForm.scrape_timeout = 5;
  addForm.pool_id = null;
  addForm.refresh_interval = 30;
  addForm.port = 9100;
  addForm.ip_address = ''; // 重置为空字符串
  addForm.relabel_configs_yaml_string = '';
  addForm.kube_config_file_path = '';
  addForm.tls_ca_file_path = '';
  addForm.tls_ca_content = '';
  addForm.bearer_token = '';
  addForm.bearer_token_file = '';
  addForm.kubernetes_sd_role = '';
};

// 提交新增采集任务
const handleAdd = async () => {
  try {
    confirmLoading.value = true;
    // 表单验证
    await addFormRef.value.validateFields();

    // 检查IP地址
    if (!addForm.ip_address || addForm.ip_address.trim() === '') {
      message.error('请输入IP地址');
      return;
    }

    const formData: createScrapeJobReq = {
      name: addForm.name,
      enable: addForm.enable,
      service_discovery_type: addForm.service_discovery_type,
      metrics_path: addForm.metrics_path,
      scheme: addForm.scheme,
      scrape_interval: addForm.scrape_interval,
      scrape_timeout: addForm.scrape_timeout,
      pool_id: addForm.pool_id!,
      refresh_interval: addForm.refresh_interval,
      port: addForm.port,
      ip_address: addForm.ip_address, 
      relabel_configs_yaml_string: addForm.relabel_configs_yaml_string,
      kube_config_file_path: addForm.kube_config_file_path,
      tls_ca_file_path: addForm.tls_ca_file_path,
      tls_ca_content: addForm.tls_ca_content,
      bearer_token: addForm.bearer_token,
      bearer_token_file: addForm.bearer_token_file,
      kubernetes_sd_role: addForm.kubernetes_sd_role,
    };

    // 提交数据
    await createScrapeJobApi(formData);
    message.success('新增采集任务成功');
    fetchResources(); // 重新获取数据
    closeAddModal();
  } catch (error: any) {
    message.error(error.message || '新增采集任务失败');
  } finally {
    confirmLoading.value = false;
  }
};

// 打开编辑模态框
const openEditModal = (record: MonitorScrapeJobItem) => {
  currentRecord.value = record;
  Object.assign(editForm, {
    id: record.id,
    name: record.name,
    enable: record.enable,
    service_discovery_type: record.service_discovery_type,
    metrics_path: record.metrics_path,
    scheme: record.scheme,
    scrape_interval: record.scrape_interval,
    scrape_timeout: record.scrape_timeout,
    pool_id: record.pool_id,
    refresh_interval: record.refresh_interval,
    port: record.port,
    // 处理IP地址 - 如果是数组取第一个，否则保持原值
    ip_address: Array.isArray(record.ip_address) 
      ? (record.ip_address[0] || '') 
      : (record.ip_address || ''),
  });
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交更新采集任务
const handleUpdate = async () => {
  try {
    confirmLoading.value = true;
    // 表单验证
    await editFormRef.value.validateFields();

    // 检查IP地址
    if (!editForm.ip_address || editForm.ip_address.trim() === '') {
      message.error('请输入IP地址');
      return;
    }

    const formData: updateScrapeJobReq = {
      id: editForm.id,
      name: editForm.name,
      enable: editForm.enable,
      service_discovery_type: editForm.service_discovery_type,
      metrics_path: editForm.metrics_path,
      scheme: editForm.scheme,
      scrape_interval: editForm.scrape_interval,
      scrape_timeout: editForm.scrape_timeout,
      pool_id: editForm.pool_id!,
      refresh_interval: editForm.refresh_interval,
      port: editForm.port,
      ip_address: editForm.ip_address, 
      relabel_configs_yaml_string: editForm.relabel_configs_yaml_string,
      kube_config_file_path: editForm.kube_config_file_path,
      tls_ca_file_path: editForm.tls_ca_file_path,
      tls_ca_content: editForm.tls_ca_content,
      bearer_token: editForm.bearer_token,
      bearer_token_file: editForm.bearer_token_file,
      kubernetes_sd_role: editForm.kubernetes_sd_role
    };

    // 提交数据
    await updateScrapeJobApi(formData);
    message.success('更新采集任务成功');
    fetchResources(); // 重新获取数据
    closeEditModal();
  } catch (error: any) {
    message.error(error.message || '更新采集任务失败');
  } finally {
    confirmLoading.value = false;
  }
};

// 处理删除采集任务
const handleDelete = (record: MonitorScrapeJobItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除采集任务 "${record.name}" 吗？`,
    onOk: async () => {
      try {
        await deleteScrapeJobApi(record.id);
        message.success('删除采集任务成功');
        fetchResources(); // 重新获取数据
      } catch (error: any) {
        message.error(error.message || '删除采集任务失败');
      }
    },
  });
};

// 在组件挂载时获取数据
onMounted(() => {
  fetchResources();
  fetchPools();
});
</script>

<style scoped>
/* 保持原有样式，只添加表单帮助文本样式 */
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

/* 表单帮助文本样式 */
.form-help-text {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #666;
  font-size: 12px;
  margin-top: 4px;
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
</style>