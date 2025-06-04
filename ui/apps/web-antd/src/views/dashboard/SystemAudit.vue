<template>
  <div class="audit-management-container">
    <!-- 顶部卡片 -->
    <div class="dashboard-card">
      <div class="card-title">
        <Icon icon="material-symbols:history" class="title-icon" />
        <h2>审计管理</h2>
      </div>

      <!-- 统计卡片 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon total">
            <Icon icon="material-symbols:database" />
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ statistics?.total_count || 0 }}</div>
            <div class="stat-label">总数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon today">
            <Icon icon="material-symbols:today" />
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ statistics?.today_count || 0 }}</div>
            <div class="stat-label">今日</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon error">
            <Icon icon="material-symbols:error" />
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ statistics?.error_count || 0 }}</div>
            <div class="stat-label">错误</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon duration">
            <Icon icon="material-symbols:speed" />
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ (statistics?.avg_duration || 0).toFixed(0) }}ms</div>
            <div class="stat-label">平均耗时</div>
          </div>
        </div>
      </div>

      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <a-input
            v-model:value="searchForm.search"
            placeholder="搜索关键词、Trace ID..."
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <Icon icon="ri:search-line" />
            </template>
          </a-input>
          
          <a-select
            v-model:value="searchForm.operation_type"
            placeholder="操作类型"
            class="filter-select"
            allowClear
          >
            <a-select-option v-for="type in auditTypes" :key="type.type" :value="type.type">
              {{ type.description }}
            </a-select-option>
          </a-select>
          
          <a-select
            v-model:value="searchForm.status_code"
            placeholder="状态码"
            class="filter-select"
            allowClear
          >
            <a-select-option value="200">200 - 成功</a-select-option>
            <a-select-option value="400">400 - 客户端错误</a-select-option>
            <a-select-option value="401">401 - 未授权</a-select-option>
            <a-select-option value="403">403 - 禁止访问</a-select-option>
            <a-select-option value="404">404 - 未找到</a-select-option>
            <a-select-option value="500">500 - 服务器错误</a-select-option>
          </a-select>
          
          <a-button type="primary" @click="handleSearch" class="search-button">
            <template #icon><Icon icon="ri:search-line" /></template>
            搜索
          </a-button>
          
          <a-button @click="handleReset" class="reset-button">
            <template #icon><Icon icon="ri:refresh-line" /></template>
            重置
          </a-button>
        </div>
        
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button 
            v-if="selectedIds.length > 0" 
            type="primary" 
            danger 
            @click="handleBatchDelete" 
            class="batch-delete-button"
          >
            <template #icon><Icon icon="ant-design:delete-outlined" /></template>
            删除选中 ({{ selectedIds.length }})
          </a-button>
          
          <a-button type="primary" @click="handleExport" class="export-button">
            <template #icon><Icon icon="material-symbols:download" /></template>
            导出
          </a-button>
        </div>
      </div>

      <!-- 高级筛选 -->
      <a-collapse class="advanced-filters" ghost>
        <a-collapse-panel key="advanced" header="高级筛选">
          <template #extra>
            <Icon icon="ri:filter-line" />
          </template>
          <div class="advanced-form">
            <div class="form-row">
              <a-form-item label="用户ID" class="form-item">
                <a-input-number
                  v-model:value="searchForm.user_id"
                  placeholder="用户ID"
                  class="custom-input"
                  :min="0"
                />
              </a-form-item>
              
              <a-form-item label="每页条数" class="form-item">
                <a-select
                  v-model:value="pagination.size"
                  @change="handlePageSizeChange"
                  class="custom-select"
                >
                  <a-select-option :value="10">10条/页</a-select-option>
                  <a-select-option :value="20">20条/页</a-select-option>
                  <a-select-option :value="50">50条/页</a-select-option>
                  <a-select-option :value="100">100条/页</a-select-option>
                </a-select>
              </a-form-item>
            </div>
            
            <div class="form-row">
              <a-form-item label="开始时间" class="form-item">
                <a-date-picker
                  v-model:value="startTime"
                  show-time
                  placeholder="请选择开始时间"
                  class="date-picker"
                  format="YYYY-MM-DD HH:mm:ss"
                />
              </a-form-item>
              
              <a-form-item label="结束时间" class="form-item">
                <a-date-picker
                  v-model:value="endTime"
                  show-time
                  placeholder="请选择结束时间"
                  class="date-picker"
                  format="YYYY-MM-DD HH:mm:ss"
                />
              </a-form-item>
            </div>
          </div>
        </a-collapse-panel>
      </a-collapse>
    </div>

    <!-- 审计日志列表表格 -->
    <div class="table-container">
      <a-table 
        :columns="columns" 
        :data-source="auditLogs" 
        row-key="id" 
        :loading="loading"
        :pagination="{
          current: pagination.page,
          pageSize: pagination.size,
          total: pagination.total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total: number, range: [number, number]) => `显示第 ${range[0]} - ${range[1]} 条，共 ${total} 条记录`,
          onChange: changePage,
          onShowSizeChange: handlePageSizeChange
        }"
        :row-selection="{
          selectedRowKeys: selectedIds,
          onChange: handleSelectionChange
        }"
        class="audit-table"
      >
        <!-- 时间列 -->
        <template #time="{ record }">
          <div class="time-cell">{{ formatTime(record.created_at) }}</div>
        </template>

        <!-- 操作类型列 -->
        <template #operationType="{ record }">
          <a-tag :color="getOperationColor(record.operation_type)" class="operation-tag">
            {{ record.operation_type }}
          </a-tag>
        </template>

        <!-- 请求方法列 -->
        <template #method="{ record }">
          <a-tag :color="getMethodColor(record.http_method)" class="method-tag">
            {{ record.http_method }}
          </a-tag>
        </template>

        <!-- 端点列 -->
        <template #endpoint="{ record }">
          <div class="endpoint-cell" :title="record.endpoint">{{ record.endpoint }}</div>
        </template>

        <!-- 状态码列 -->
        <template #statusCode="{ record }">
          <a-tag :color="getStatusColor(record.status_code)" class="status-tag">
            {{ record.status_code }}
          </a-tag>
        </template>

        <!-- 耗时列 -->
        <template #duration="{ record }">
          <div class="duration-cell">{{ record.duration }}ms</div>
        </template>

        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-tooltip title="查看详情">
              <a-button type="link" @click="viewDetail(record)" class="action-button view-button">
                <template #icon><Icon icon="clarity:eye-line" /></template>
              </a-button>
            </a-tooltip>
            <a-popconfirm
              title="确定要删除这条审计日志吗?"
              ok-text="确定"
              cancel-text="取消"
              placement="left"
              @confirm="deleteLog(record.id)"
            >
              <a-tooltip title="删除日志">
                <a-button type="link" danger class="action-button delete-button">
                  <template #icon><Icon icon="ant-design:delete-outlined" /></template>
                </a-button>
              </a-tooltip>
            </a-popconfirm>
          </a-space>
        </template>
      </a-table>
    </div>

    <!-- 详情模态框 -->
    <a-modal
      v-model:visible="showDetail"
      title="审计日志详情"
      :footer="null"
      class="custom-modal detail-modal"
      :maskClosable="false"
      :destroyOnClose="true"
      :width="900"
    >
      <div class="modal-content">
        <div class="modal-header-icon">
          <div class="icon-wrapper detail-icon">
            <Icon icon="material-symbols:history-edu" />
          </div>
          <div class="header-text">日志详细信息</div>
        </div>
        
        <div v-if="selectedLog" class="detail-content">
          <div class="detail-grid">
            <div class="detail-item">
              <label>ID:</label>
              <span>{{ selectedLog.id }}</span>
            </div>
            <div class="detail-item">
              <label>用户ID:</label>
              <span>{{ selectedLog.user_id }}</span>
            </div>
            <div class="detail-item">
              <label>Trace ID:</label>
              <span class="trace-id">{{ selectedLog.trace_id }}</span>
            </div>
            <div class="detail-item">
              <label>IP地址:</label>
              <span>{{ selectedLog.ip_address }}</span>
            </div>
            <div class="detail-item">
              <label>请求方法:</label>
              <a-tag :color="getMethodColor(selectedLog.http_method)">
                {{ selectedLog.http_method }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>端点:</label>
              <span class="endpoint-path">{{ selectedLog.endpoint }}</span>
            </div>
            <div class="detail-item">
              <label>操作类型:</label>
              <a-tag :color="getOperationColor(selectedLog.operation_type)">
                {{ selectedLog.operation_type }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>目标类型:</label>
              <span>{{ selectedLog.target_type }}</span>
            </div>
            <div class="detail-item">
              <label>目标ID:</label>
              <span>{{ selectedLog.target_id }}</span>
            </div>
            <div class="detail-item">
              <label>状态码:</label>
              <a-tag :color="getStatusColor(selectedLog.status_code)">
                {{ selectedLog.status_code }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>耗时:</label>
              <span class="duration-badge">{{ selectedLog.duration }}ms</span>
            </div>
            <div class="detail-item">
              <label>时间:</label>
              <span>{{ formatTime(selectedLog.created_at) }}</span>
            </div>
          </div>

          <div class="detail-section">
            <div class="section-header">
              <Icon icon="mdi:account-details" class="section-icon" />
              <span>User Agent</span>
            </div>
            <div class="section-content">
              <pre class="code-block">{{ selectedLog.user_agent }}</pre>
            </div>
          </div>

          <div class="detail-section">
            <div class="section-header">
              <Icon icon="mdi:code-json" class="section-icon" />
              <span>请求体</span>
            </div>
            <div class="section-content">
              <pre class="code-block">{{ formatJSON(selectedLog.request_body) }}</pre>
            </div>
          </div>

          <div class="detail-section">
            <div class="section-header">
              <Icon icon="mdi:code-json" class="section-icon" />
              <span>响应体</span>
            </div>
            <div class="section-content">
              <pre class="code-block">{{ formatJSON(selectedLog.response_body) }}</pre>
            </div>
          </div>

          <div v-if="selectedLog.error_msg" class="detail-section error-section">
            <div class="section-header">
              <Icon icon="mdi:alert-circle" class="section-icon error-icon" />
              <span>错误信息</span>
            </div>
            <div class="section-content">
              <pre class="code-block error">{{ selectedLog.error_msg }}</pre>
            </div>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import dayjs, { Dayjs } from 'dayjs'
import {
  type AuditLog,
  type AuditStatistics,
  type AuditTypeInfo,
  type ListAuditLogsRequest,
  listAuditLogsApi,
  getAuditStatisticsApi,
  getAuditTypesApi,
  deleteAuditLogApi,
  batchDeleteLogsApi,
  exportAuditLogsApi
} from '#/api/core/audit'
import { Icon } from '@iconify/vue'

// 响应式数据
const loading = ref(false)
const auditLogs = ref<AuditLog[]>([])
const statistics = ref<AuditStatistics>()
const auditTypes = ref<AuditTypeInfo[]>([])
const selectedIds = ref<number[]>([])
const selectedLog = ref<AuditLog>()
const showDetail = ref(false)

// 时间选择器
const startTime = ref<Dayjs>()
const endTime = ref<Dayjs>()

// 分页数据结构
const pagination = ref({
  page: 1,
  size: 20,
  total: 0
})

// 搜索表单
const searchForm = ref<Omit<ListAuditLogsRequest, 'page' | 'size'>>({
  search: '',
  operation_type: '',
  user_id: undefined,
  status_code: undefined,
  start_time: undefined,
  end_time: undefined
})

// 表格列配置
const columns = [
  {
    title: '时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
    slots: { customRender: 'time' }
  },
  {
    title: '用户ID',
    dataIndex: 'user_id',
    key: 'user_id',
    width: 100
  },
  {
    title: '操作类型',
    dataIndex: 'operation_type',
    key: 'operation_type',
    slots: { customRender: 'operationType' },
    width: 120
  },
  {
    title: '请求方法',
    dataIndex: 'http_method',
    key: 'http_method',
    slots: { customRender: 'method' },
    width: 100
  },
  {
    title: '端点',
    dataIndex: 'endpoint',
    key: 'endpoint',
    ellipsis: true,
    slots: { customRender: 'endpoint' }
  },
  {
    title: '状态码',
    dataIndex: 'status_code',
    key: 'status_code',
    slots: { customRender: 'statusCode' },
    width: 100
  },
  {
    title: '耗时',
    dataIndex: 'duration',
    key: 'duration',
    slots: { customRender: 'duration' },
    width: 100
  },
  {
    title: 'IP地址',
    dataIndex: 'ip_address',
    key: 'ip_address',
    width: 140
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    width: 120,
    fixed: 'right'
  }
]

// 加载数据
const loadAuditLogs = async () => {
  loading.value = true
  try {
    const params: ListAuditLogsRequest = {
      ...searchForm.value,
      page: pagination.value.page,
      size: pagination.value.size,
      start_time: startTime.value ? startTime.value.valueOf() : undefined,
      end_time: endTime.value ? endTime.value.valueOf() : undefined
    }
    
    const response = await listAuditLogsApi(params)
    auditLogs.value = response.items || []
    pagination.value.total = response.total || 0
    
  } catch (error: any) {
    message.error(error.message || '获取审计日志失败')
  } finally {
    loading.value = false
  }
}

const loadStatistics = async () => {
  try {
    const response = await getAuditStatisticsApi()
    statistics.value = response
  } catch (error: any) {
    message.error(error.message || '获取统计数据失败')
  }
}

const loadAuditTypes = async () => {
  try {
    const response = await getAuditTypesApi()
    auditTypes.value = response.items || []
  } catch (error: any) {
    message.error(error.message || '获取审计类型失败')
  }
}

// 事件处理
const handleSearch = () => {
  pagination.value.page = 1
  selectedIds.value = []
  loadAuditLogs()
}

const handleReset = () => {
  searchForm.value = {
    search: '',
    operation_type: '',
    user_id: undefined,
    status_code: undefined,
    start_time: undefined,
    end_time: undefined
  }
  startTime.value = undefined
  endTime.value = undefined
  pagination.value.page = 1
  pagination.value.size = 20
  selectedIds.value = []
  loadAuditLogs()
}

const handleSelectionChange = (selectedRowKeys: number[]) => {
  selectedIds.value = selectedRowKeys
}

const changePage = (page: number, pageSize?: number) => {
  pagination.value.page = page
  if (pageSize) {
    pagination.value.size = pageSize
  }
  selectedIds.value = []
  loadAuditLogs()
}

const handlePageSizeChange = (current: number, size: number) => {
  pagination.value.page = 1
  pagination.value.size = size
  selectedIds.value = []
  loadAuditLogs()
}

const viewDetail = (log: AuditLog) => {
  selectedLog.value = log
  showDetail.value = true
}

const deleteLog = async (id: number) => {
  try {
    await deleteAuditLogApi(id)
    message.success('删除成功')
    loadAuditLogs()
    loadStatistics()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

const handleBatchDelete = async () => {
  try {
    await batchDeleteLogsApi({ ids: selectedIds.value })
    message.success('批量删除成功')
    selectedIds.value = []
    loadAuditLogs()
    loadStatistics()
  } catch (error: any) {
    message.error(error.message || '批量删除失败')
  }
}

const handleExport = async () => {
  try {
    const params = {
      ...searchForm.value,
      format: 'excel' as const,
      max_rows: 10000,
      page: pagination.value.page,
      size: pagination.value.size,
      start_time: startTime.value ? startTime.value.valueOf() : undefined,
      end_time: endTime.value ? endTime.value.valueOf() : undefined
    }
    await exportAuditLogsApi(params)
    message.success('导出成功')
  } catch (error: any) {
    message.error(error.message || '导出失败')
  }
}

// 工具函数
const formatTime = (timeStr: string) => {
  return dayjs(timeStr).format('YYYY-MM-DD HH:mm:ss')
}

const formatJSON = (obj: any) => {
  if (!obj) return '无数据'
  try {
    return JSON.stringify(obj, null, 2)
  } catch {
    return String(obj)
  }
}

const getOperationColor = (operation: string) => {
  const colorMap: Record<string, string> = {
    'CREATE': '#52c41a',
    'UPDATE': '#1890ff',
    'DELETE': '#f5222d',
    'VIEW': '#8c8c8c',
    'LOGIN': '#722ed1',
    'LOGOUT': '#fa8c16'
  }
  return colorMap[operation] || '#d9d9d9'
}

const getMethodColor = (method: string) => {
  const colorMap: Record<string, string> = {
    'GET': '#1890ff',
    'POST': '#52c41a',
    'PUT': '#faad14',
    'DELETE': '#f5222d',
    'PATCH': '#13c2c2'
  }
  return colorMap[method] || '#d9d9d9'
}

const getStatusColor = (status: number) => {
  if (status >= 200 && status < 300) return '#52c41a'
  if (status >= 300 && status < 400) return '#1890ff'
  if (status >= 400 && status < 500) return '#faad14'
  if (status >= 500) return '#f5222d'
  return '#d9d9d9'
}

// 生命周期
onMounted(() => {
  loadAuditLogs()
  loadStatistics()
  loadAuditTypes()
})
</script>

<style scoped>
/* 整体容器样式 */
.audit-management-container {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
  font-family: 'Roboto', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

/* 顶部卡片样式 */
.dashboard-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 20px;
  margin-bottom: 20px;
  transition: all 0.3s;
}

.card-title {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.title-icon {
  font-size: 28px;
  margin-right: 10px;
  color: #1890ff;
}

.card-title h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: #1e293b;
}

/* 统计卡片样式 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: linear-gradient(135deg, #fff, #f8f9fa);
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
}

.stat-icon.total {
  background: linear-gradient(135deg, #1890ff, #36cfc9);
}

.stat-icon.today {
  background: linear-gradient(135deg, #52c41a, #73d13d);
}

.stat-icon.error {
  background: linear-gradient(135deg, #faad14, #ffc53d);
}

.stat-icon.duration {
  background: linear-gradient(135deg, #722ed1, #b37feb);
}

.stat-content {
  flex: 1;
}

.stat-number {
  font-size: 20px;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.2;
}

.stat-label {
  color: #64748b;
  font-size: 14px;
  margin-top: 4px;
}

/* 工具栏样式 */
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.search-input {
  width: 280px;
  border-radius: 6px;
  transition: all 0.3s;
}

.search-input:hover, 
.search-input:focus {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.filter-select {
  min-width: 140px;
  border-radius: 6px;
}

:deep(.filter-select .ant-select-selector) {
  border-radius: 6px !important;
}

.search-button {
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.reset-button {
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.action-buttons {
  display: flex;
  gap: 12px;
  align-items: center;
}

.batch-delete-button {
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.3s;
}

.export-button {
  border-radius: 6px;
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  border: none;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.3);
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.3s;
}

.export-button:hover {
  background: linear-gradient(90deg, #40a9ff, #5cdbd3);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.4);
}

/* 高级筛选样式 */
.advanced-filters {
  margin-top: 16px;
}

:deep(.advanced-filters .ant-collapse-header) {
  padding: 12px 0 !important;
}

.advanced-form {
  padding-top: 16px;
}

.form-row {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
}

.form-item {
  flex: 1;
  margin-bottom: 0;
}

:deep(.custom-input) {
  border-radius: 6px;
  transition: all 0.3s;
}

:deep(.custom-select .ant-select-selector) {
  border-radius: 6px !important;
}

:deep(.date-picker .ant-picker) {
  width: 100%;
  border-radius: 6px;
}

/* 表格容器样式 */
.table-container {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 20px;
  overflow: hidden;
}

.audit-table {
  width: 100%;
}

/* 表格内容样式 */
.time-cell {
  font-size: 13px;
  color: #595959;
  font-family: 'Roboto Mono', monospace;
}

.endpoint-cell {
  font-family: 'Roboto Mono', 'Courier New', monospace;
  color: #595959;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.duration-cell {
  font-family: 'Roboto Mono', monospace;
  font-size: 13px;
  color: #595959;
}

/* 标签样式 */
.operation-tag, .method-tag, .status-tag {
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.method-tag {
  font-family: 'Roboto Mono', monospace;
  letter-spacing: 0.5px;
}

/* 操作按钮样式 */
.action-button {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s;
}

.action-button:hover {
  background-color: #f0f0f0;
  transform: translateY(-1px);
}

.view-button {
  color: #1890ff;
}

.delete-button {
  color: #f5222d;
}

/* 详情模态框样式 */
:deep(.detail-modal .ant-modal-content) {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

:deep(.detail-modal .ant-modal-header) {
  background: #fff;
  padding: 20px 24px 0;
  border-bottom: none;
}

:deep(.detail-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
}

:deep(.detail-modal .ant-modal-body) {
  padding: 0;
}

.modal-content {
  padding: 20px 24px 24px;
}

.modal-header-icon {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 30px;
}

.icon-wrapper {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.25);
}

.detail-icon {
  background: linear-gradient(135deg, #722ed1, #b37feb);
}

.icon-wrapper svg {
  font-size: 32px;
  color: white;
}

.header-text {
  font-size: 16px;
  color: #1e293b;
  font-weight: 500;
}

/* 详情内容样式 */
.detail-content {
  margin-top: 0;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-item label {
  font-weight: 600;
  color: #1e293b;
  font-size: 14px;
}

.detail-item span {
  color: #64748b;
  word-break: break-all;
}

.trace-id {
  font-family: 'Roboto Mono', monospace;
  font-size: 12px;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
}

.endpoint-path {
  font-family: 'Roboto Mono', monospace;
  font-size: 12px;
  color: #595959;
  background-color: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
}

.duration-badge {
  font-family: 'Roboto Mono', monospace;
  background-color: #f0f2f5;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.detail-section {
  margin-bottom: 24px;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  border: 1px solid #f0f0f0;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background-color: #f9f9f9;
  border-bottom: 1px solid #f0f0f0;
  font-weight: 500;
  color: #1e293b;
}

.section-icon {
  color: #1890ff;
  font-size: 18px;
}

.error-section .section-icon.error-icon {
  color: #f5222d;
}

.section-content {
  padding: 16px;
}

.code-block {
  background-color: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 12px;
  font-family: 'Roboto Mono', 'Courier New', monospace;
  font-size: 12px;
  white-space: pre-wrap;
  overflow-x: auto;
  max-height: 200px;
  margin: 0;
  line-height: 1.5;
}

.error-section .code-block {
  background-color: #fff5f5;
  border-color: #ffccc7;
  color: #a8071a;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .audit-management-container {
    padding: 10px;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .custom-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .search-filters {
    width: 100%;
    margin-bottom: 12px;
  }
  
  .search-input {
    width: 100%;
  }
  
  .form-row {
    flex-direction: column;
  }
  
  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>