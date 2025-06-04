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
      <div class="toolbar">
        <!-- 搜索区域 -->
        <div class="search-section">
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

          <a-date-picker
            v-model:value="startTime"
            show-time
            placeholder="开始时间"
            class="date-picker"
            format="YYYY-MM-DD HH:mm:ss"
          />
          
          <a-date-picker
            v-model:value="endTime"
            show-time
            placeholder="结束时间"
            class="date-picker"
            format="YYYY-MM-DD HH:mm:ss"
          />
        </div>
        
        <!-- 操作按钮 -->
        <div class="action-section">
          <a-button type="primary" @click="handleSearch" class="action-btn">
            <template #icon><Icon icon="ri:search-line" /></template>
            搜索
          </a-button>
          
          <a-button @click="handleReset" class="action-btn">
            <template #icon><Icon icon="ri:refresh-line" /></template>
            重置
          </a-button>
          
          <a-button 
            v-if="selectedIds.length > 0" 
            type="primary" 
            danger 
            @click="handleBatchDelete" 
            class="action-btn"
          >
            <template #icon><Icon icon="ant-design:delete-outlined" /></template>
            删除选中 ({{ selectedIds.length }})
          </a-button>
        </div>
      </div>
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
      class="detail-modal"
      :maskClosable="false"
      :destroyOnClose="true"
      :width="900"
    >
      <div class="modal-content">
        <div class="modal-header">
          <div class="header-icon">
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
              <label>请求路径:</label>
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
import { ref, onMounted } from 'vue'
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
  batchDeleteLogsApi
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
    title: '请求路径',
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
      // start_time: startTime.value ? startTime.value.valueOf() : undefined,
      // end_time: endTime.value ? endTime.value.valueOf() : undefined
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

const handlePageSizeChange = (_: number, size: number) => {
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
  padding: 24px;
  background-color: #f5f7fa;
  min-height: 100vh;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* 顶部卡片样式 */
.dashboard-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  padding: 24px;
  margin-bottom: 20px;
  border: 1px solid #f0f0f0;
}

.card-title {
  display: flex;
  align-items: center;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.title-icon {
  font-size: 24px;
  margin-right: 12px;
  color: #1890ff;
}

.card-title h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #262626;
}

/* 统计卡片样式 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: all 0.3s;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border-color: #d9d9d9;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
  flex-shrink: 0;
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
  font-size: 24px;
  font-weight: 700;
  color: #262626;
  line-height: 1.2;
  margin-bottom: 4px;
}

.stat-label {
  color: #8c8c8c;
  font-size: 14px;
  font-weight: 500;
}

/* 工具栏样式 */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  flex-wrap: wrap;
}

.search-section {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  flex: 1;
}

.search-input {
  width: 280px;
  border-radius: 6px;
}

.filter-select {
  min-width: 120px;
  border-radius: 6px;
}

.date-picker {
  width: 200px;
}

:deep(.date-picker .ant-picker) {
  border-radius: 6px;
}

.action-section {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

.action-btn {
  border-radius: 6px;
  height: 32px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-weight: 500;
}

/* 表格容器样式 */
.table-container {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  border: 1px solid #f0f0f0;
}

.audit-table {
  width: 100%;
}

:deep(.audit-table .ant-table-thead > tr > th) {
  background-color: #fafafa;
  font-weight: 600;
  color: #262626;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.audit-table .ant-table-tbody > tr:hover > td) {
  background-color: #f5f5f5;
}

/* 表格内容样式 */
.time-cell {
  font-size: 13px;
  color: #595959;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
}

.endpoint-cell {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  color: #595959;
  background-color: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.duration-cell {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  font-size: 13px;
  color: #595959;
  font-weight: 500;
}

/* 标签样式 */
.operation-tag, .method-tag, .status-tag {
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
  border: none;
}

.method-tag {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  letter-spacing: 0.5px;
}

/* 操作按钮样式 */
.action-button {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  transition: all 0.2s;
  border: none;
}

.action-button:hover {
  transform: translateY(-1px);
}

.view-button {
  color: #1890ff;
}

.view-button:hover {
  background-color: #e6f7ff;
  color: #096dd9;
}

.delete-button {
  color: #ff4d4f;
}

.delete-button:hover {
  background-color: #fff2f0;
  color: #cf1322;
}

/* 详情模态框样式 */
:deep(.detail-modal .ant-modal-content) {
  border-radius: 12px;
  overflow: hidden;
}

:deep(.detail-modal .ant-modal-header) {
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.detail-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #262626;
}

.modal-content {
  padding: 0;
}

.modal-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 0 16px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 20px;
}

.header-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: linear-gradient(135deg, #722ed1, #b37feb);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
}

.header-text {
  font-size: 16px;
  color: #262626;
  font-weight: 600;
}

/* 详情内容样式 */
.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
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
  color: #262626;
  font-size: 14px;
}

.detail-item span {
  color: #595959;
  word-break: break-all;
}

.trace-id {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  font-size: 12px;
  background-color: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
}

.endpoint-path {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  font-size: 12px;
  background-color: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
}

.duration-badge {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  background-color: #f0f2f5;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 500;
}

.detail-section {
  margin-bottom: 20px;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background-color: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  font-weight: 600;
  color: #262626;
}

.section-icon {
  color: #1890ff;
  font-size: 16px;
}

.error-section .section-icon.error-icon {
  color: #ff4d4f;
}

.section-content {
  padding: 16px;
}

.code-block {
  background-color: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 12px;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  font-size: 12px;
  white-space: pre-wrap;
  overflow-x: auto;
  max-height: 200px;
  margin: 0;
  line-height: 1.5;
  color: #262626;
}

.error-section .code-block {
  background-color: #fff2f0;
  border-color: #ffccc7;
  color: #a8071a;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-section {
    margin-bottom: 16px;
  }
  
  .action-section {
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .audit-management-container {
    padding: 16px;
  }
  
  .dashboard-card {
    padding: 20px;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .search-section {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-input {
    width: 100%;
  }
  
  .filter-select,
  .date-picker {
    width: 100%;
  }
  
  .action-section {
    flex-direction: column;
    align-items: stretch;
  }
  
  .detail-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 480px) {
  .action-section {
    gap: 12px;
  }
  
  .action-btn {
    width: 100%;
    justify-content: center;
  }
}
</style>