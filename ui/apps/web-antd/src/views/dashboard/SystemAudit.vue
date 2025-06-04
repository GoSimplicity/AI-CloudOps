<template>
    <div class="audit-page">
      <!-- 统计卡片 -->
      <div class="stats-section">
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-icon total">📊</div>
            <div class="stat-content">
              <div class="stat-number">{{ statistics?.total_count || 0 }}</div>
              <div class="stat-label">总数</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon today">📈</div>
            <div class="stat-content">
              <div class="stat-number">{{ statistics?.today_count || 0 }}</div>
              <div class="stat-label">今日</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon error">⚠️</div>
            <div class="stat-content">
              <div class="stat-number">{{ statistics?.error_count || 0 }}</div>
              <div class="stat-label">错误</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon duration">⏱️</div>
            <div class="stat-content">
              <div class="stat-number">{{ (statistics?.avg_duration || 0).toFixed(0) }}ms</div>
              <div class="stat-label">平均耗时</div>
            </div>
          </div>
        </div>
      </div>
  
      <!-- 搜索和筛选 -->
      <div class="search-section">
        <div class="search-form">
          <div class="search-row">
            <input
              v-model="searchForm.search"
              type="text"
              placeholder="搜索关键词、Trace ID..."
              class="search-input"
              @keyup.enter="handleSearch"
            />
            <select v-model="searchForm.operation_type" class="select-input">
              <option value="">所有操作类型</option>
              <option v-for="type in auditTypes" :key="type.type" :value="type.type">
                {{ type.description }}
              </option>
            </select>
            <select v-model="searchForm.status_code" class="select-input">
              <option value="">所有状态码</option>
              <option value="200">200 - 成功</option>
              <option value="400">400 - 客户端错误</option>
              <option value="401">401 - 未授权</option>
              <option value="403">403 - 禁止访问</option>
              <option value="404">404 - 未找到</option>
              <option value="500">500 - 服务器错误</option>
            </select>
          </div>
          <div class="search-row">
            <input
              v-model="searchForm.start_time"
              type="datetime-local"
              class="date-input"
            />
            <span class="date-separator">至</span>
            <input
              v-model="searchForm.end_time"
              type="datetime-local"
              class="date-input"
            />
            <input
              v-model="searchForm.user_id"
              type="number"
              placeholder="用户ID"
              class="number-input"
            />
            <button @click="handleSearch" class="search-btn">搜索</button>
            <button @click="handleReset" class="reset-btn">重置</button>
            <button @click="handleExport" class="export-btn">导出</button>
          </div>
        </div>
      </div>
  
      <!-- 数据表格 -->
      <div class="table-section">
        <div class="table-header">
          <h2>审计日志列表</h2>
          <div class="table-actions">
            <button 
              v-if="selectedIds.length > 0" 
              @click="handleBatchDelete" 
              class="delete-btn"
            >
              删除选中 ({{ selectedIds.length }})
            </button>
          </div>
        </div>
  
        <div class="table-container" v-loading="loading">
          <table class="audit-table">
            <thead>
              <tr>
                <th>
                  <input
                    type="checkbox"
                    :checked="isAllSelected"
                    @change="handleSelectAll"
                  />
                </th>
                <th>时间</th>
                <th>用户</th>
                <th>操作类型</th>
                <th>请求方法</th>
                <th>端点</th>
                <th>状态码</th>
                <th>耗时</th>
                <th>IP地址</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="log in auditLogs"
                :key="log.id"
                :class="{ 'error-row': log.status_code >= 400 }"
              >
                <td>
                  <input
                    type="checkbox"
                    :checked="selectedIds.includes(log.id)"
                    @change="handleSelectItem(log.id)"
                  />
                </td>
                <td class="time-cell">{{ formatTime(log.created_at) }}</td>
                <td>{{ log.user_id }}</td>
                <td>
                  <span class="operation-tag" :class="getOperationClass(log.operation_type)">
                    {{ log.operation_type }}
                  </span>
                </td>
                <td>
                  <span class="method-tag" :class="getMethodClass(log.http_method)">
                    {{ log.http_method }}
                  </span>
                </td>
                <td class="endpoint-cell">{{ log.endpoint }}</td>
                <td>
                  <span class="status-tag" :class="getStatusClass(log.status_code)">
                    {{ log.status_code }}
                  </span>
                </td>
                <td class="duration-cell">{{ log.duration }}ms</td>
                <td>{{ log.ip_address }}</td>
                <td class="actions-cell">
                  <button @click="viewDetail(log)" class="view-btn">查看</button>
                  <button @click="deleteLog(log.id)" class="delete-btn-small">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
  
          <div v-if="auditLogs.length === 0 && !loading" class="empty-state">
            <div class="empty-icon">📋</div>
            <div class="empty-text">暂无审计日志数据</div>
          </div>
        </div>
  
        <!-- 分页 -->
        <div class="pagination" v-if="total > 0">
          <button
            :disabled="currentPage <= 1"
            @click="changePage(currentPage - 1)"
            class="page-btn"
          >
            上一页
          </button>
          <span class="page-info">
            第 {{ currentPage }} 页，共 {{ totalPages }} 页，总计 {{ total }} 条
          </span>
          <button
            :disabled="currentPage >= totalPages"
            @click="changePage(currentPage + 1)"
            class="page-btn"
          >
            下一页
          </button>
        </div>
      </div>
  
      <!-- 详情模态框 -->
      <div v-if="showDetail" class="modal-overlay" @click="closeDetail">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>审计日志详情</h3>
            <button @click="closeDetail" class="close-btn">×</button>
          </div>
          <div class="modal-body">
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
                  <span>{{ selectedLog.trace_id }}</span>
                </div>
                <div class="detail-item">
                  <label>IP地址:</label>
                  <span>{{ selectedLog.ip_address }}</span>
                </div>
                <div class="detail-item">
                  <label>请求方法:</label>
                  <span>{{ selectedLog.http_method }}</span>
                </div>
                <div class="detail-item">
                  <label>端点:</label>
                  <span>{{ selectedLog.endpoint }}</span>
                </div>
                <div class="detail-item">
                  <label>操作类型:</label>
                  <span>{{ selectedLog.operation_type }}</span>
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
                  <span>{{ selectedLog.status_code }}</span>
                </div>
                <div class="detail-item">
                  <label>耗时:</label>
                  <span>{{ selectedLog.duration }}ms</span>
                </div>
                <div class="detail-item">
                  <label>时间:</label>
                  <span>{{ formatTime(selectedLog.created_at) }}</span>
                </div>
              </div>
  
              <div class="detail-section">
                <h4>User Agent</h4>
                <pre class="code-block">{{ selectedLog.user_agent }}</pre>
              </div>
  
              <div class="detail-section">
                <h4>请求体</h4>
                <pre class="code-block">{{ formatJSON(selectedLog.request_body) }}</pre>
              </div>
  
              <div class="detail-section">
                <h4>响应体</h4>
                <pre class="code-block">{{ formatJSON(selectedLog.response_body) }}</pre>
              </div>
  
              <div v-if="selectedLog.error_msg" class="detail-section error-section">
                <h4>错误信息</h4>
                <pre class="code-block error">{{ selectedLog.error_msg }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, onMounted, computed } from 'vue'
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
  
  // 响应式数据
  const loading = ref(false)
  const auditLogs = ref<AuditLog[]>([])
  const statistics = ref<AuditStatistics>()
  const auditTypes = ref<AuditTypeInfo[]>([])
  const selectedIds = ref<number[]>([])
  const selectedLog = ref<AuditLog>()
  const showDetail = ref(false)
  
  // 分页相关
  const currentPage = ref(1)
  const pageSize = ref(20)
  const total = ref(0)
  
  // 搜索表单
  const searchForm = ref<ListAuditLogsRequest>({
    page: 1,
    size: 20,
    search: '',
    operation_type: '',
    user_id: undefined,
    status_code: undefined,
    start_time: undefined,
    end_time: undefined
  })
  
  // 计算属性
  const totalPages = computed(() => Math.ceil(total.value / pageSize.value))
  const isAllSelected = computed(() => 
    auditLogs.value.length > 0 && selectedIds.value.length === auditLogs.value.length
  )
  
  // 加载数据
  const loadAuditLogs = async () => {
    loading.value = true
    try {
      const params = {
        ...searchForm.value,
        page: currentPage.value,
        size: pageSize.value,
        start_time: searchForm.value.start_time ? new Date(searchForm.value.start_time).getTime() : undefined,
        end_time: searchForm.value.end_time ? new Date(searchForm.value.end_time).getTime() : undefined
      }
      
      const response = await listAuditLogsApi(params)
      auditLogs.value = response.items || []
      total.value = response.total || 0
    } catch (error) {
      console.error('加载审计日志失败:', error)
    } finally {
      loading.value = false
    }
  }
  
  const loadStatistics = async () => {
    try {
      const response = await getAuditStatisticsApi()
      statistics.value = response
    } catch (error) {
      console.error('加载统计数据失败:', error)
    }
  }
  
  const loadAuditTypes = async () => {
    try {
      const response = await getAuditTypesApi()
      auditTypes.value = response.items || []
    } catch (error) {
      console.error('加载审计类型失败:', error)
    }
  }
  
  // 事件处理
  const handleSearch = () => {
    currentPage.value = 1
    selectedIds.value = []
    loadAuditLogs()
  }
  
  const handleReset = () => {
    searchForm.value = {
      page: 1,
      size: 20,
      search: '',
      operation_type: '',
      user_id: undefined,
      status_code: undefined,
      start_time: undefined,
      end_time: undefined
    }
    currentPage.value = 1
    selectedIds.value = []
    loadAuditLogs()
  }
  
  const handleSelectAll = (event: Event) => {
    const target = event.target as HTMLInputElement
    if (target.checked) {
      selectedIds.value = auditLogs.value.map(log => log.id)
    } else {
      selectedIds.value = []
    }
  }
  
  const handleSelectItem = (id: number) => {
    const index = selectedIds.value.indexOf(id)
    if (index > -1) {
      selectedIds.value.splice(index, 1)
    } else {
      selectedIds.value.push(id)
    }
  }
  
  const changePage = (page: number) => {
    currentPage.value = page
    loadAuditLogs()
  }
  
  const viewDetail = (log: AuditLog) => {
    selectedLog.value = log
    showDetail.value = true
  }
  
  const closeDetail = () => {
    showDetail.value = false
    selectedLog.value = undefined
  }
  
  const deleteLog = async (id: number) => {
    if (!confirm('确定要删除这条审计日志吗？')) return
    
    try {
      await deleteAuditLogApi(id)
      await loadAuditLogs()
      await loadStatistics()
    } catch (error) {
      console.error('删除失败:', error)
    }
  }
  
  const handleBatchDelete = async () => {
    if (!confirm(`确定要删除选中的 ${selectedIds.value.length} 条记录吗？`)) return
    
    try {
      await batchDeleteLogsApi({ ids: selectedIds.value })
      selectedIds.value = []
      await loadAuditLogs()
      await loadStatistics()
    } catch (error) {
      console.error('批量删除失败:', error)
    }
  }
  
  const handleExport = async () => {
    try {
      const params = {
        ...searchForm.value,
        format: 'excel' as const,
        max_rows: 10000
      }
      await exportAuditLogsApi(params)
    } catch (error) {
      console.error('导出失败:', error)
    }
  }
  
  // 工具函数
  const formatTime = (timeStr: string) => {
    return new Date(timeStr).toLocaleString('zh-CN')
  }
  
  const formatJSON = (obj: any) => {
    if (!obj) return '无'
    return JSON.stringify(obj, null, 2)
  }
  
  const getOperationClass = (operation: string) => {
    const classMap: Record<string, string> = {
      'CREATE': 'create',
      'UPDATE': 'update', 
      'DELETE': 'delete',
      'VIEW': 'view',
      'LOGIN': 'login',
      'LOGOUT': 'logout'
    }
    return classMap[operation] || 'default'
  }
  
  const getMethodClass = (method: string) => {
    const classMap: Record<string, string> = {
      'GET': 'get',
      'POST': 'post',
      'PUT': 'put',
      'DELETE': 'delete',
      'PATCH': 'patch'
    }
    return classMap[method] || 'default'
  }
  
  const getStatusClass = (status: number) => {
    if (status >= 200 && status < 300) return 'success'
    if (status >= 300 && status < 400) return 'redirect'
    if (status >= 400 && status < 500) return 'client-error'
    if (status >= 500) return 'server-error'
    return 'default'
  }
  
  // 生命周期
  onMounted(() => {
    loadAuditLogs()
    loadStatistics()
    loadAuditTypes()
  })
  </script>
  
  <style scoped>
  .audit-page {
    padding: 20px;
    background-color: #f5f5f5;
    min-height: 100vh;
  }
  
  /* 统计卡片样式 */
  .stats-section {
    margin-bottom: 20px;
  }
  
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }
  
  .stat-card {
    background: white;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    display: flex;
    align-items: center;
    gap: 16px;
  }
  
  .stat-icon {
    font-size: 32px;
    width: 60px;
    height: 60px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .stat-icon.total { background-color: #e3f2fd; }
  .stat-icon.today { background-color: #e8f5e8; }
  .stat-icon.error { background-color: #ffeaa7; }
  .stat-icon.duration { background-color: #f3e5f5; }
  
  .stat-number {
    font-size: 24px;
    font-weight: bold;
    color: #333;
  }
  
  .stat-label {
    color: #666;
    font-size: 14px;
  }
  
  /* 搜索区域样式 */
  .search-section {
    background: white;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    margin-bottom: 20px;
  }
  
  .search-row {
    display: flex;
    gap: 12px;
    margin-bottom: 12px;
    flex-wrap: wrap;
  }
  
  .search-row:last-child {
    margin-bottom: 0;
  }
  
  .search-input, .select-input, .date-input, .number-input {
    padding: 8px 12px;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 14px;
  }
  
  .search-input {
    flex: 1;
    min-width: 200px;
  }
  
  .select-input, .number-input {
    min-width: 120px;
  }
  
  .date-input {
    min-width: 180px;
  }
  
  .date-separator {
    align-self: center;
    color: #666;
  }
  
  .search-btn, .reset-btn, .export-btn {
    padding: 8px 16px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
  }
  
  .search-btn {
    background-color: #007bff;
    color: white;
  }
  
  .reset-btn {
    background-color: #6c757d;
    color: white;
  }
  
  .export-btn {
    background-color: #28a745;
    color: white;
  }
  
  /* 表格区域样式 */
  .table-section {
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    overflow: hidden;
  }
  
  .table-header {
    padding: 20px;
    border-bottom: 1px solid #eee;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .table-header h2 {
    margin: 0;
    color: #333;
  }
  
  .delete-btn {
    background-color: #dc3545;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
  }
  
  .table-container {
    overflow-x: auto;
  }
  
  .audit-table {
    width: 100%;
    border-collapse: collapse;
  }
  
  .audit-table th,
  .audit-table td {
    padding: 12px;
    text-align: left;
    border-bottom: 1px solid #eee;
  }
  
  .audit-table th {
    background-color: #f8f9fa;
    font-weight: 600;
    color: #333;
  }
  
  .audit-table tr:hover {
    background-color: #f8f9fa;
  }
  
  .error-row {
    background-color: #fff5f5;
  }
  
  .time-cell {
    font-size: 12px;
    color: #666;
    white-space: nowrap;
  }
  
  .endpoint-cell {
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .duration-cell {
    font-family: monospace;
    font-size: 12px;
  }
  
  /* 标签样式 */
  .operation-tag, .method-tag, .status-tag {
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
  }
  
  .operation-tag.create { background-color: #d4edda; color: #155724; }
  .operation-tag.update { background-color: #d1ecf1; color: #0c5460; }
  .operation-tag.delete { background-color: #f8d7da; color: #721c24; }
  .operation-tag.view { background-color: #e2e3e5; color: #383d41; }
  .operation-tag.login { background-color: #cce5ff; color: #004085; }
  .operation-tag.logout { background-color: #f0f0f0; color: #6c757d; }
  
  .method-tag.get { background-color: #d4edda; color: #155724; }
  .method-tag.post { background-color: #cce5ff; color: #004085; }
  .method-tag.put { background-color: #fff3cd; color: #856404; }
  .method-tag.delete { background-color: #f8d7da; color: #721c24; }
  .method-tag.patch { background-color: #e2e3e5; color: #383d41; }
  
  .status-tag.success { background-color: #d4edda; color: #155724; }
  .status-tag.redirect { background-color: #d1ecf1; color: #0c5460; }
  .status-tag.client-error { background-color: #fff3cd; color: #856404; }
  .status-tag.server-error { background-color: #f8d7da; color: #721c24; }
  
  .actions-cell {
    white-space: nowrap;
  }
  
  .view-btn, .delete-btn-small {
    padding: 4px 8px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    margin-right: 4px;
  }
  
  .view-btn {
    background-color: #007bff;
    color: white;
  }
  
  .delete-btn-small {
    background-color: #dc3545;
    color: white;
  }
  
  /* 空状态样式 */
  .empty-state {
    text-align: center;
    padding: 60px 20px;
    color: #666;
  }
  
  .empty-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }
  
  .empty-text {
    font-size: 16px;
  }
  
  /* 分页样式 */
  .pagination {
    padding: 20px;
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 16px;
    border-top: 1px solid #eee;
  }
  
  .page-btn {
    padding: 8px 16px;
    border: 1px solid #ddd;
    background-color: white;
    border-radius: 4px;
    cursor: pointer;
  }
  
  .page-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .page-info {
    color: #666;
    font-size: 14px;
  }
  
  /* 模态框样式 */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  
  .modal-content {
    background: white;
    border-radius: 8px;
    max-width: 800px;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  }
  
  .modal-header {
    padding: 20px;
    border-bottom: 1px solid #eee;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .modal-header h3 {
    margin: 0;
    color: #333;
  }
  
  .close-btn {
    background: none;
    border: none;
    font-size: 24px;
    cursor: pointer;
    color: #666;
  }
  
  .modal-body {
    padding: 20px;
  }
  
  .detail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 16px;
    margin-bottom: 20px;
  }
  
  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .detail-item label {
    font-weight: 600;
    color: #333;
    font-size: 14px;
  }
  
  .detail-item span {
    color: #666;
    word-break: break-all;
  }
  
  .detail-section {
    margin-bottom: 20px;
  }
  
  .detail-section h4 {
    margin: 0 0 8px 0;
    color: #333;
    font-size: 16px;
  }
  
  .code-block {
    background-color: #f8f9fa;
    border: 1px solid #e9ecef;
    border-radius: 4px;
    padding: 12px;
    font-family: 'Courier New', monospace;
    font-size: 12px;
    white-space: pre-wrap;
    overflow-x: auto;
    max-height: 200px;
  }
  
  .error-section .code-block {
    background-color: #fff5f5;
    border-color: #f5c6cb;
    color: #721c24;
  }
  
  /* 响应式设计 */
  @media (max-width: 768px) {
    .audit-page {
      padding: 10px;
    }
    
    .stats-grid {
      grid-template-columns: 1fr;
    }
    
    .search-row {
      flex-direction: column;
    }
    
    .search-input, .select-input, .date-input, .number-input {
      width: 100%;
    }
    
    .table-header {
      flex-direction: column;
      gap: 10px;
      align-items: flex-start;
    }
    
    .detail-grid {
      grid-template-columns: 1fr;
    }
  }
  </style>