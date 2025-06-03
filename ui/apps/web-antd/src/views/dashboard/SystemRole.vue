<template>
  <div class="role-management-container">
    <!-- 顶部统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon role-icon">
          <Icon icon="material-symbols:badge-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ roleList.length }}</div>
          <div class="stat-label">总角色数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon active-icon">
          <Icon icon="material-symbols:check-circle-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ activeRoles }}</div>
          <div class="stat-label">启用角色</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon system-icon">
          <Icon icon="material-symbols:admin-panel-settings-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ systemRoles }}</div>
          <div class="stat-label">系统角色</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon user-icon">
          <Icon icon="material-symbols:group-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ totalUsers }}</div>
          <div class="stat-label">关联用户</div>
        </div>
      </div>
    </div>

    <!-- 主控制面板 -->
    <div class="main-panel">
      <div class="panel-header">
        <div class="header-title">
          <Icon icon="material-symbols:shield-person-outline" class="title-icon" />
          <h2>角色权限管理</h2>
        </div>
        
        <!-- 搜索和筛选区域 -->
        <div class="search-section">
          <div class="search-group">
            <a-input
              v-model:value="searchParams.search"
              placeholder="搜索角色名称或编码"
              class="search-input"
              @pressEnter="handleSearch"
              allowClear
            >
              <template #prefix>
                <Icon icon="ri:search-line" />
              </template>
            </a-input>
            
            <a-select
              v-model:value="searchParams.status"
              placeholder="状态筛选"
              class="status-filter"
              allowClear
            >
              <a-select-option :value="1">
                <div class="status-option">
                  <div class="status-dot active"></div>
                  <span>启用</span>
                </div>
              </a-select-option>
              <a-select-option :value="0">
                <div class="status-option">
                  <div class="status-dot inactive"></div>
                  <span>禁用</span>
                </div>
              </a-select-option>
            </a-select>
            
            <a-button type="primary" @click="handleSearch" class="search-btn">
              <template #icon><Icon icon="ri:search-line" /></template>
              搜索
            </a-button>
          </div>
          
          <div class="action-group">
            <!-- 视图切换按钮 -->
            <a-radio-group 
              v-model:value="viewMode" 
              button-style="solid" 
              size="small"
              class="view-toggle"
            >
              <a-radio-button value="card">
                卡片
              </a-radio-button>
              <a-radio-button value="list">
                列表
              </a-radio-button>
            </a-radio-group>
            
            <a-button @click="handleRefresh" class="refresh-btn">
              <template #icon><Icon icon="material-symbols:refresh" /></template>
              刷新
            </a-button>
            <a-button type="primary" @click="handleAdd" class="add-btn">
              <template #icon><Icon icon="material-symbols:add" /></template>
              新建角色
            </a-button>
          </div>
        </div>
      </div>

      <!-- 卡片视图 -->
      <div v-if="viewMode === 'card'" class="role-grid">
        <div 
          v-for="role in paginatedRoles" 
          :key="role.id" 
          class="role-card"
          :class="{ 'system-role': role.is_system }"
        >
          <div class="role-header">
            <div class="role-title">
              <div class="role-name">{{ role.name }}</div>
              <div class="role-code">{{ role.code }}</div>
            </div>
            <div class="role-status">
              <a-switch 
                v-model:checked="role.status" 
                :checked-value="1" 
                :unchecked-value="0"
                @change="handleStatusChange(role)"
                :disabled="role.is_system === 1"
                size="small"
              />
            </div>
          </div>
          
          <div class="role-description">
            {{ role.description || '暂无描述' }}
          </div>
          
          <div class="role-stats">
            <div class="stat-item">
              <Icon icon="material-symbols:api" />
              <span>{{ role.apis?.length || 0 }} 个权限</span>
            </div>
            <div class="stat-item">
              <Icon icon="material-symbols:group" />
              <span>{{ role.users?.length || 0 }} 个用户</span>
            </div>
          </div>
          
          <div class="role-tags">
            <a-tag v-if="role.is_system === 1" color="red" class="system-tag">
              <Icon icon="material-symbols:admin-panel-settings" />
              系统角色
            </a-tag>
            <a-tag :color="role.status === 1 ? 'green' : 'default'" class="status-tag">
              {{ role.status === 1 ? '已启用' : '已禁用' }}
            </a-tag>
          </div>
          
          <div class="role-actions">
            <a-tooltip title="查看详情">
              <a-button type="text" @click="handleView(role)" class="action-btn view-btn">
                <Icon icon="material-symbols:visibility-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="编辑角色">
              <a-button type="text" @click="handleEdit(role)" class="action-btn edit-btn">
                <Icon icon="material-symbols:edit-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="权限管理">
              <a-button type="text" @click="handlePermission(role)" class="action-btn permission-btn">
                <Icon icon="material-symbols:key-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除角色">
              <a-popconfirm
                title="确定要删除这个角色吗？"
                @confirm="handleDelete(role)"
                :disabled="role.is_system === 1"
              >
                <a-button 
                  type="text" 
                  danger 
                  :disabled="role.is_system === 1"
                  class="action-btn delete-btn"
                >
                  <Icon icon="material-symbols:delete-outline" />
                </a-button>
              </a-popconfirm>
            </a-tooltip>
          </div>
          
          <div class="role-time">
            <small>创建时间：{{ formatTime(role.created_at) }}</small>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else class="role-table-container">
        <a-table
          :columns="tableColumns"
          :data-source="paginatedRoles"
          :pagination="false"
          :scroll="{ x: 1200 }"
          row-key="id"
          class="role-table"
          size="middle"
        >
          <template #bodyCell="{ column, record, text }">
            <!-- 角色名称列 -->
            <template v-if="column.key === 'name'">
              <div class="name-cell">
                <div class="role-name-text">{{ record.name }}</div>
                <div class="role-code-text">{{ record.code }}</div>
              </div>
            </template>
            
            <!-- 角色类型列 -->
            <template v-if="column.key === 'type'">
              <a-tag v-if="record.is_system === 1" color="orange">
                <Icon icon="material-symbols:admin-panel-settings" />
                系统角色
              </a-tag>
              <a-tag v-else color="blue">
                <Icon icon="material-symbols:person-outline" />
                自定义角色
              </a-tag>
            </template>
            
            <!-- 状态列 -->
            <template v-if="column.key === 'status'">
              <a-switch 
                v-model:checked="record.status" 
                :checked-value="1" 
                :unchecked-value="0"
                @change="handleStatusChange(record)"
                :disabled="record.is_system === 1"
                size="small"
              />
              <div class="status-text">
                {{ record.status === 1 ? '已启用' : '已禁用' }}
              </div>
            </template>
            
            <!-- 描述列 -->
            <template v-if="column.key === 'description'">
              <div class="description-cell">
                {{ record.description || '暂无描述' }}
              </div>
            </template>
            
            <!-- 权限数量列 -->
            <template v-if="column.key === 'apis'">
              <div class="count-cell">
                <Icon icon="material-symbols:api" />
                {{ record.apis?.length || 0 }}
              </div>
            </template>
            
            <!-- 用户数量列 -->
            <template v-if="column.key === 'users'">
              <div class="count-cell">
                <Icon icon="material-symbols:group" />
                {{ record.users?.length || 0 }}
              </div>
            </template>
            
            <!-- 创建时间列 -->
            <template v-if="column.key === 'created_at'">
              <div class="time-cell">
                {{ formatTime(record.created_at) }}
              </div>
            </template>
            
            <!-- 操作列 -->
            <template v-if="column.key === 'actions'">
              <div class="table-actions">
                <a-tooltip title="查看详情">
                  <a-button type="text" @click="handleView(record)" class="table-action-btn view-btn">
                    <Icon icon="material-symbols:visibility-outline" />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="编辑角色">
                  <a-button type="text" @click="handleEdit(record)" class="table-action-btn edit-btn">
                    <Icon icon="material-symbols:edit-outline" />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="权限管理">
                  <a-button type="text" @click="handlePermission(record)" class="table-action-btn permission-btn">
                    <Icon icon="material-symbols:key-outline" />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="删除角色">
                  <a-popconfirm
                    title="确定要删除这个角色吗？"
                    @confirm="handleDelete(record)"
                    :disabled="record.is_system === 1"
                  >
                    <a-button 
                      type="text" 
                      danger 
                      :disabled="record.is_system === 1"
                      class="table-action-btn delete-btn"
                    >
                      <Icon icon="material-symbols:delete-outline" />
                    </a-button>
                  </a-popconfirm>
                </a-tooltip>
              </div>
            </template>
          </template>
        </a-table>
      </div>
      
      <!-- 分页 -->
      <div class="pagination-container">
        <a-pagination
          v-model:current="pagination.current"
          v-model:page-size="pagination.pageSize"
          :total="filteredRoles.length"
          :show-size-changer="true"
          :show-quick-jumper="true"
          :show-total="(total: number, range: [number, number]) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`"
          class="custom-pagination"
        />
      </div>
    </div>

    <!-- 后续的模态框代码保持不变 -->
    <!-- 查看详情模态框 -->
    <a-modal
      v-model:open="viewModalVisible"
      title="角色详情"
      width="900px"
      :mask-closable="false"
      :footer="null"
      class="view-modal"
    >
      <!-- 查看模态框内容保持不变 -->
      <div class="view-content" v-if="viewModalVisible && viewRoleData">
        <!-- 基本信息 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:info-outline" />
            <span>基本信息</span>
          </div>
          <div class="info-grid">
            <div class="info-item">
              <label>角色名称</label>
              <div class="info-value">{{ viewRoleData.name }}</div>
            </div>
            <div class="info-item">
              <label>角色编码</label>
              <div class="info-value code">{{ viewRoleData.code }}</div>
            </div>
            <div class="info-item">
              <label>角色状态</label>
              <div class="info-value">
                <a-tag :color="viewRoleData.status === 1 ? 'green' : 'default'">
                  <div class="status-option">
                    <div class="status-dot" :class="viewRoleData.status === 1 ? 'active' : 'inactive'"></div>
                    <span>{{ viewRoleData.status === 1 ? '已启用' : '已禁用' }}</span>
                  </div>
                </a-tag>
              </div>
            </div>
            <div class="info-item">
              <label>角色类型</label>
              <div class="info-value">
                <a-tag v-if="viewRoleData.is_system === 1" color="orange">
                  <Icon icon="material-symbols:admin-panel-settings" />
                  系统角色
                </a-tag>
                <a-tag v-else color="blue">
                  <Icon icon="material-symbols:person-outline" />
                  自定义角色
                </a-tag>
              </div>
            </div>
            <div class="info-item full-width">
              <label>角色描述</label>
              <div class="info-value description">
                {{ viewRoleData.description || '暂无描述' }}
              </div>
            </div>
          </div>
        </div>

        <!-- 统计信息 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:analytics-outline" />
            <span>统计信息</span>
          </div>
          <div class="stats-row">
            <div class="stats-item">
              <div class="stats-icon api-stats">
                <Icon icon="material-symbols:api" />
              </div>
              <div class="stats-info">
                <div class="stats-number">{{ viewRoleData.apis?.length || 0 }}</div>
                <div class="stats-label">关联权限</div>
              </div>
            </div>
            <div class="stats-item">
              <div class="stats-icon user-stats">
                <Icon icon="material-symbols:group" />
              </div>
              <div class="stats-info">
                <div class="stats-number">{{ viewRoleData.users?.length || 0 }}</div>
                <div class="stats-label">关联用户</div>
              </div>
            </div>
            <div class="stats-item">
              <div class="stats-icon time-stats">
                <Icon icon="material-symbols:schedule" />
              </div>
              <div class="stats-info">
                <div class="stats-number">{{ formatTime(viewRoleData.created_at) }}</div>
                <div class="stats-label">创建时间</div>
              </div>
            </div>
          </div>
        </div>

        <!-- 权限列表 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:key-outline" />
            <span>权限详情</span>
            <div class="section-extra">
              <a-input
                v-model:value="viewApiSearch"
                placeholder="搜索权限"
                size="small"
                class="mini-search"
                style="width: 200px;"
                allowClear
              >
                <template #prefix>
                  <Icon icon="ri:search-line" />
                </template>
              </a-input>
            </div>
          </div>
          
          <div v-if="viewRoleData.apis && viewRoleData.apis.length > 0" class="apis-container">
            <div class="apis-summary">
              <span>共 {{ filteredViewApis.length }} 个权限</span>
              <div class="method-stats">
                <span v-for="(count, method) in viewApiMethodStats" :key="method" class="method-stat">
                  <span class="method-badge" :class="method.toLowerCase()">{{ method }}</span>
                  <span>{{ count }}</span>
                </span>
              </div>
            </div>
            
            <div class="apis-list">
              <div v-for="api in filteredViewApis" :key="api.id" class="api-detail-item">
                <div class="api-method-badge" :class="getMethodClass(api.method)">
                  {{ formatMethod(api.method) }}
                </div>
                <div class="api-detail-info">
                  <div class="api-detail-name">{{ api.name }}</div>
                  <div class="api-detail-path">{{ api.path }}</div>
                  <div class="api-detail-desc" v-if="api.description">
                    {{ api.description }}
                  </div>
                </div>
                <div class="api-detail-meta">
                  <div class="api-category" v-if="api.category">
                    <Icon icon="material-symbols:category-outline" />
                    {{ api.category }}
                  </div>
                  <div class="api-created">
                    <Icon icon="material-symbols:schedule" />
                    {{ formatTime(api.created_at) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-else class="empty-apis">
            <Icon icon="material-symbols:security-outline" class="empty-icon" />
            <div class="empty-text">该角色暂未分配任何权限</div>
          </div>
        </div>

        <!-- 关联用户 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:group-outline" />
            <span>关联用户</span>
            <div class="section-extra">
              <a-input
                v-model:value="viewUserSearch"
                placeholder="搜索用户"
                size="small"
                class="mini-search"
                style="width: 200px;"
                allowClear
              >
                <template #prefix>
                  <Icon icon="ri:search-line" />
                </template>
              </a-input>
            </div>
          </div>
          
          <div v-if="viewRoleData.users && viewRoleData.users.length > 0" class="users-container">
            <div class="users-summary">
              <span>共 {{ filteredViewUsers.length }} 个用户</span>
            </div>
            
            <div class="users-list">
              <div v-for="user in filteredViewUsers" :key="user.id" class="user-detail-item">
                <div class="user-avatar">
                  <Icon icon="material-symbols:person" v-if="!user.avatar" />
                  <img v-else :src="user.avatar" :alt="user.username" />
                </div>
                <div class="user-detail-info">
                  <div class="user-detail-name">
                    {{ user.real_name || user.username }}
                    <a-tag v-if="user.username" size="small" color="blue">{{ user.username }}</a-tag>
                  </div>
                  <div class="user-detail-email" v-if="user.email">{{ user.email }}</div>
                  <div class="user-detail-phone" v-if="user.phone">{{ user.phone }}</div>
                </div>
                <div class="user-detail-status">
                  <a-tag :color="user.status === 1 ? 'green' : 'default'" size="small">
                    {{ user.status === 1 ? '正常' : '禁用' }}
                  </a-tag>
                  <div class="user-login-time" v-if="user.last_login_at">
                    最后登录：{{ formatTime(user.last_login_at) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-else class="empty-users">
            <Icon icon="material-symbols:group-off-outline" class="empty-icon" />
            <div class="empty-text">该角色暂无关联用户</div>
          </div>
        </div>

        <!-- 操作记录 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:history" />
            <span>操作记录</span>
          </div>
          <div class="timeline-container">
            <div class="timeline-item">
              <div class="timeline-dot create"></div>
              <div class="timeline-content">
                <div class="timeline-title">角色创建</div>
                <div class="timeline-time">{{ formatTime(viewRoleData.created_at) }}</div>
              </div>
            </div>
            <div class="timeline-item" v-if="viewRoleData.updated_at && viewRoleData.updated_at !== viewRoleData.created_at">
              <div class="timeline-dot update"></div>
              <div class="timeline-content">
                <div class="timeline-title">最后更新</div>
                <div class="timeline-time">{{ formatTime(viewRoleData.updated_at) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <template #footer>
        <div class="view-footer">
          <a-space>
            <a-button @click="viewModalVisible = false">关闭</a-button>
            <a-button type="primary" @click="handleEditFromView">
              <Icon icon="material-symbols:edit-outline" />
              编辑角色
            </a-button>
            <a-button @click="handlePermissionFromView">
              <Icon icon="material-symbols:key-outline" />
              权限管理
            </a-button>
          </a-space>
        </div>
      </template>
    </a-modal>

    <!-- 角色表单弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      width="800px"
      :mask-closable="false"
      :destroy-on-close="true"
      class="role-modal"
      :key="modalVisible ? 'open' : 'closed'"
    >
      <div class="modal-content" v-if="modalVisible">
        <a-form
          ref="formRef"
          :model="formData"
          :rules="formRules"
          layout="vertical"
          class="role-form"
          :key="formData.id || 'new'"
        >
          <div class="form-section">
            <div class="section-header">
              <Icon icon="material-symbols:info-outline" />
              <span>基本信息</span>
            </div>
            <div class="form-grid">
              <a-form-item label="角色名称" name="name">
                <a-input 
                  v-model:value="formData.name" 
                  placeholder="请输入角色名称"
                />
              </a-form-item>
              <a-form-item label="角色编码" name="code">
                <a-input 
                  v-model:value="formData.code" 
                  placeholder="请输入角色编码"
                />
              </a-form-item>
            </div>
            <a-form-item label="角色描述" name="description">
              <a-textarea 
                v-model:value="formData.description" 
                placeholder="请输入角色描述"
                :rows="3"
              />
            </a-form-item>
            <a-form-item label="角色状态" name="status">
              <a-radio-group v-model:value="formData.status" class="status-radio">
                <a-radio :value="1">
                  <div class="radio-option">
                    <div class="status-dot active"></div>
                    <span>启用</span>
                  </div>
                </a-radio>
                <a-radio :value="0">
                  <div class="radio-option">
                    <div class="status-dot inactive"></div>
                    <span>禁用</span>
                  </div>
                </a-radio>
              </a-radio-group>
            </a-form-item>
          </div>
          
          <div class="form-section">
            <div class="section-header">
              <Icon icon="material-symbols:key-outline" />
              <span>权限配置</span>
            </div>
            <a-form-item label="关联API权限" name="api_ids">
              <a-select
                v-if="apiList.length > 0 && modalVisible"
                v-model:value="formData.api_ids"
                mode="multiple"
                placeholder="请选择API权限"
                class="api-select"
                :filter-option="filterOption"
                show-search
                :key="`api-select-${modalVisible}-${formData.id || 'new'}-${apiList.length}`"
                :get-popup-container="getPopupContainer"
                :dropdown-match-select-width="false"
              >
                <a-select-option 
                  v-for="api in apiList" 
                  :key="api.id" 
                  :value="api.id"
                  :label="api.name"
                >
                  <div class="api-option">
                    <div class="api-method" :class="getMethodClass(api.method)">
                      {{ formatMethod(api.method) }}
                    </div>
                    <div class="api-info">
                      <div class="api-name">{{ api.name }}</div>
                      <div class="api-path">{{ api.path }}</div>
                    </div>
                  </div>
                </a-select-option>
              </a-select>
            </a-form-item>
          </div>
        </a-form>
      </div>
      
      <template #footer>
        <a-space>
          <a-button @click="handleCancel">取消</a-button>
          <a-button type="primary" @click="handleSubmit" :loading="submitLoading">
            <Icon icon="material-symbols:save-outline" />
            保存
          </a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- 权限管理弹窗 -->
    <a-modal
      v-model:open="permissionModalVisible"
      title="权限管理"
      width="1000px"
      :mask-closable="false"
      class="permission-modal"
      :key="permissionModalVisible ? 'perm-open' : 'perm-closed'"
    >
      <div class="permission-content" v-if="permissionModalVisible">
        <div class="permission-header">
          <div class="role-info">
            <Icon icon="material-symbols:badge-outline" />
            <span>{{ currentRole?.name }}</span>
          </div>
          <div class="permission-stats">
            <span>已分配 {{ assignedApis.length }} 个权限</span>
          </div>
        </div>
        
        <a-tabs v-model:activeKey="permissionTab" class="permission-tabs">
          <a-tab-pane key="assigned" tab="已分配权限">
            <div class="api-list">
              <div 
                v-for="api in assignedApis" 
                :key="api.id" 
                class="api-item assigned"
              >
                <div class="api-method" :class="api.method.toLowerCase()">
                  {{ api.method }}
                </div>
                <div class="api-details">
                  <div class="api-name">{{ api.name }}</div>
                  <div class="api-path">{{ api.path }}</div>
                </div>
                <a-button 
                  type="text" 
                  danger 
                  @click="handleRevokeApi(api)"
                  class="revoke-btn"
                >
                  <Icon icon="material-symbols:remove-circle-outline" />
                  移除
                </a-button>
              </div>
            </div>
          </a-tab-pane>
          
          <a-tab-pane key="available" tab="可分配权限">
            <div class="api-search">
              <a-input
                v-model:value="apiSearchText"
                placeholder="搜索API权限"
                class="search-input"
              >
                <template #prefix>
                  <Icon icon="ri:search-line" />
                </template>
              </a-input>
            </div>
            <div class="api-list">
              <div 
                v-for="api in availableApis" 
                :key="api.id" 
                class="api-item available"
              >
                <div class="api-method" :class="api.method.toLowerCase()">
                  {{ api.method }}
                </div>
                <div class="api-details">
                  <div class="api-name">{{ api.name }}</div>
                  <div class="api-path">{{ api.path }}</div>
                </div>
                <a-button 
                  type="primary" 
                  size="small"
                  @click="handleAssignApi(api)"
                  class="assign-btn"
                >
                  <Icon icon="material-symbols:add-circle-outline" />
                  分配
                </a-button>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
      
      <template #footer>
        <a-button @click="permissionModalVisible = false">关闭</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted, nextTick, onBeforeUnmount, watch } from 'vue';
import { message } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import type { FormInstance } from 'ant-design-vue';
import type { Role, ListRolesReq, CreateRoleReq, UpdateRoleReq, DeleteRoleReq } from '#/api/core/system';
import { 
  listRolesApi, 
  createRoleApi, 
  updateRoleApi, 
  deleteRoleApi,
  getRoleDetailApi,
  assignApisToRoleApi,
  revokeApisFromRoleApi,
  getRoleApisApi
} from '#/api/core/system';

import { listApisApi } from '#/api/core/system';

// 表单引用
const formRef = ref<FormInstance>();

// 视图模式状态
const viewMode = ref<'card' | 'list'>('card');

// 表格列配置
const tableColumns = [
  {
    title: '角色名称',
    key: 'name',
    width: 200,
    fixed: 'left'
  },
  {
    title: '角色类型',
    key: 'type',
    width: 120,
    align: 'center'
  },
  {
    title: '状态',
    key: 'status',
    width: 120,
    align: 'center'
  },
  {
    title: '描述',
    key: 'description',
    width: 250,
    ellipsis: true
  },
  {
    title: '权限数',
    key: 'apis',
    width: 100,
    align: 'center'
  },
  {
    title: '用户数',
    key: 'users',
    width: 100,
    align: 'center'
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 160,
    align: 'center'
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    align: 'center',
    fixed: 'right'
  }
];

// 响应式数据
const loading = ref(false);
const submitLoading = ref(false);
const modalVisible = ref(false);
const permissionModalVisible = ref(false);
const viewModalVisible = ref(false);
const modalTitle = ref('');
const roleList = ref<Role[]>([]);
const apiList = ref<any[]>([]);
const currentRole = ref<Role | null>(null);
const assignedApis = ref<any[]>([]);
const apiSearchText = ref('');
const permissionTab = ref('assigned');
const viewRoleData = ref<Role | null>(null);
const viewApiSearch = ref('');
const viewUserSearch = ref('');

// 搜索参数
const searchParams = reactive<ListRolesReq>({
  page: 1,
  size: 10,
  search: '',
  status: undefined
});

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 12,
  total: 0
});

// 初始化表单数据的函数
const initFormData = () => ({
  name: '',
  code: '',
  description: '',
  status: 1 as 0 | 1,
  api_ids: []
});

// 表单数据
const formData = reactive<Partial<CreateRoleReq & UpdateRoleReq>>(initFormData());

// 表单验证规则
const formRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入角色编码', trigger: 'blur' }],
  status: [{ required: true, message: '请选择角色状态', trigger: 'change' }]
};

// 工具函数 - 修复method相关问题
const formatMethod = (method: any): string => {
  if (typeof method === 'string') {
    return method.toUpperCase();
  }
  return String(method || 'UNKNOWN').toUpperCase();
};

const getMethodClass = (method: any): string => {
  const methodStr = formatMethod(method).toLowerCase();
  return ['get', 'post', 'put', 'delete', 'patch'].includes(methodStr) ? methodStr : 'unknown';
};

// 修复getPopupContainer函数
const getPopupContainer = (trigger?: HTMLElement): HTMLElement => {
  // 如果没有传入trigger或trigger已被销毁，返回document.body
  if (!trigger || !trigger.parentNode) {
    return document.body;
  }
  
  // 尝试找到最近的模态框容器
  let container = trigger.parentNode as HTMLElement;
  while (container && container !== document.body) {
    if (container.classList?.contains('ant-modal-body') || 
        container.classList?.contains('ant-modal-wrap')) {
      return container;
    }
    container = container.parentNode as HTMLElement;
  }
  
  return document.body;
};

// 计算属性
const activeRoles = computed(() => 
  roleList.value.filter((role: Role) => role.status === 1).length
);

const systemRoles = computed(() => 
  roleList.value.filter((role: Role) => role.is_system === 1).length
);

const totalUsers = computed(() => 
  roleList.value.reduce((total: number, role: Role) => total + (role.users?.length || 0), 0)
);

const filteredRoles = computed(() => {
  let filtered = roleList.value;
  
  if (searchParams.search) {
    const searchText = searchParams.search.toLowerCase();
    filtered = filtered.filter((role: Role) => 
      role.name.toLowerCase().includes(searchText) ||
      role.code.toLowerCase().includes(searchText)
    );
  }
  
  if (searchParams.status !== undefined) {
    filtered = filtered.filter((role: Role) => role.status === searchParams.status);
  }
  
  return filtered;
});

const paginatedRoles = computed(() => {
  const start = (pagination.current - 1) * pagination.pageSize;
  const end = start + pagination.pageSize;
  return filteredRoles.value.slice(start, end);
});

const availableApis = computed(() => {
  if (!currentRole.value) return [];
  
  const assignedIds = assignedApis.value.map(api => api.id);
  let available = apiList.value.filter(api => !assignedIds.includes(api.id));
  
  if (apiSearchText.value) {
    const searchText = apiSearchText.value.toLowerCase();
    available = available.filter(api => 
      api.name.toLowerCase().includes(searchText) ||
      api.path.toLowerCase().includes(searchText)
    );
  }
  
  return available;
});

// 查看页面的计算属性
const filteredViewApis = computed(() => {
  if (!viewRoleData.value?.apis) return [];
  
  let filtered = viewRoleData.value.apis;
  if (viewApiSearch.value) {
    const searchText = viewApiSearch.value.toLowerCase();
    filtered = filtered.filter((api: any) => 
      api.name.toLowerCase().includes(searchText) ||
      api.path.toLowerCase().includes(searchText) ||
      (api.description && api.description.toLowerCase().includes(searchText))
    );
  }
  
  return filtered;
});

const filteredViewUsers = computed(() => {
  if (!viewRoleData.value?.users) return [];
  
  let filtered = viewRoleData.value.users;
  if (viewUserSearch.value) {
    const searchText = viewUserSearch.value.toLowerCase();
    filtered = filtered.filter((user: any) => 
      (user.username && user.username.toLowerCase().includes(searchText)) ||
      (user.real_name && user.real_name.toLowerCase().includes(searchText)) ||
      (user.email && user.email.toLowerCase().includes(searchText)) ||
      (user.phone && user.phone.toLowerCase().includes(searchText))
    );
  }
  
  return filtered;
});

const viewApiMethodStats = computed(() => {
  if (!viewRoleData.value?.apis) return {};
  
  const stats: Record<string, number> = {};
  viewRoleData.value.apis.forEach((api: any) => {
    const method = formatMethod(api.method);
    stats[method] = (stats[method] || 0) + 1;
  });
  
  return stats;
});

// 其他工具函数
const formatTime = (timeStr: string | undefined) => {
  if (!timeStr) return '-';
  return new Date(timeStr).toLocaleString('zh-CN');
};

const filterOption = (input: string, option: any) => {
  if (!option?.label) return false;
  return option.label.toLowerCase().includes(input.toLowerCase());
};

// 数据获取函数
const fetchRoleList = async () => {
  loading.value = true;
  try {
    const response = await listRolesApi(searchParams);
    if (response) {
      roleList.value = response.items || [];
      pagination.total = response.total || 0;
    }
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
  } finally {
    loading.value = false;
  }
};

const fetchApiList = async () => {
  try {
    const response = await listApisApi({
      page_number: 1,
      page_size: 1000
    });
    
    // 确保API数据的method字段是字符串
    const safeApiList = (response.list || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));
    
    apiList.value = safeApiList;
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
  }
};

const fetchRoleApis = async (roleId: number) => {
  try {
    const response = await getRoleApisApi(roleId);
    
    // 确保返回的API数据也是安全的
    const safeAssignedApis = (response.items || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));
    
    assignedApis.value = safeAssignedApis;
  } catch (error: any) {
    message.error(error.message || '获取角色权限失败');
  }
};

// 事件处理函数
const handleSearch = () => {
  pagination.current = 1;
  fetchRoleList();
};

const handleRefresh = () => {
  searchParams.search = '';
  searchParams.status = undefined;
  pagination.current = 1;
  fetchRoleList();
};

const handleAdd = async () => {
  modalTitle.value = '新建角色';
  
  // 先清理数据
  Object.assign(formData, initFormData());
  
  // 确保DOM更新后再显示模态框
  await nextTick();
  modalVisible.value = true;
  
  // 再次等待确保模态框渲染完成
  await nextTick();
  formRef.value?.resetFields();
};

const handleEdit = async (role: Role) => {
  try {
    modalTitle.value = '编辑角色';
    
    // 先关闭模态框并清理状态
    modalVisible.value = false;
    await nextTick();
    
    // 获取角色详情
    const response = await getRoleDetailApi(role.id);
    const roleDetail = response;
    
    // 设置表单数据
    Object.assign(formData, {
      id: roleDetail.id,
      name: roleDetail.name,
      code: roleDetail.code,
      description: roleDetail.description,
      status: roleDetail.status,
      api_ids: roleDetail.apis?.map((api: any) => api.id) || []
    });
    
    // 打开模态框
    modalVisible.value = true;
    
    // 等待DOM更新后清除验证状态
    await nextTick();
    formRef.value?.clearValidate();
  } catch (error: any) {
    message.error(error.message || '获取角色详情失败');
  }
};

const handleView = async (role: Role) => {
  try {
    // 获取角色详情
    const response = await getRoleDetailApi(role.id);
    viewRoleData.value = response;
    
    // 清空搜索条件
    viewApiSearch.value = '';
    viewUserSearch.value = '';
    
    // 显示模态框
    viewModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取角色详情失败');
  }
};

const handlePermission = async (role: Role) => {
  currentRole.value = role;
  await fetchRoleApis(role.id);
  permissionModalVisible.value = true;
};

const handleDelete = async (role: Role) => {
  try {
    await deleteRoleApi({ id: role.id });
    message.success('删除成功');
    fetchRoleList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

const handleStatusChange = async (role: Role) => {
  const originalStatus = role.status;
  try {
    await updateRoleApi({
      id: role.id,
      name: role.name,
      code: role.code,
      description: role.description,
      status: role.status,
      api_ids: role.apis?.map((api: any) => api.id) || []
    });
    message.success('状态更新成功');
  } catch (error: any) {
    message.error(error.message || '状态更新失败');
    // 回滚状态
    role.status = originalStatus;
  }
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    submitLoading.value = true;
    
    if (formData.id) {
      await updateRoleApi(formData as UpdateRoleReq);
      message.success('更新成功');
    } else {
      await createRoleApi(formData as CreateRoleReq);
      message.success('创建成功');
    }
    
    handleCancel();
    fetchRoleList();
  } catch (error: any) {
    if (error.errorFields) {
      // 表单验证错误
      return;
    }
    message.error(error.message || '操作失败');
  } finally {
    submitLoading.value = false;
  }
};

const handleCancel = async () => {
  modalVisible.value = false;
  
  // 等待模态框关闭动画完成后清理数据
  setTimeout(() => {
    Object.assign(formData, initFormData());
    formRef.value?.resetFields();
  }, 300);
};

const handleAssignApi = async (api: any) => {
  if (!currentRole.value) return;
  
  try {
    await assignApisToRoleApi({
      role_id: currentRole.value.id,
      api_ids: [api.id]
    });
    message.success('权限分配成功');
    await fetchRoleApis(currentRole.value.id);
  } catch (error: any) {
    message.error(error.message || '权限分配失败');
  }
};

const handleRevokeApi = async (api: any) => {
  if (!currentRole.value) return;
  
  try {
    await revokeApisFromRoleApi({
      role_id: currentRole.value.id,
      api_ids: [api.id]
    });
    message.success('权限移除成功');
    await fetchRoleApis(currentRole.value.id);
  } catch (error: any) {
    message.error(error.message || '权限移除失败');
  }
};

// 从查看页面跳转到编辑的函数
const handleEditFromView = () => {
  if (viewRoleData.value) {
    viewModalVisible.value = false;
    handleEdit(viewRoleData.value);
  }
};

// 从查看页面跳转到权限管理的函数
const handlePermissionFromView = () => {
  if (viewRoleData.value) {
    viewModalVisible.value = false;
    handlePermission(viewRoleData.value);
  }
};

// 监听模态框状态变化，确保状态同步
watch(modalVisible, async (newVal) => {
  if (!newVal) {
    // 模态框关闭时，延迟清理数据
    setTimeout(() => {
      Object.assign(formData, initFormData());
    }, 300);
  }
});

// 监听视图模式变化，调整分页大小
watch(viewMode, (newMode) => {
  if (newMode === 'list') {
    pagination.pageSize = 20; // 列表模式下显示更多数据
  } else {
    pagination.pageSize = 12; // 卡片模式下显示较少数据
  }
  pagination.current = 1; // 重置到第一页
});

// 组件卸载时清理状态
onBeforeUnmount(() => {
  modalVisible.value = false;
  permissionModalVisible.value = false;
  viewModalVisible.value = false;
  currentRole.value = null;
  viewRoleData.value = null;
  Object.assign(formData, initFormData());
});

// 初始化
onMounted(() => {
  fetchRoleList();
  fetchApiList();
});
</script>

<style scoped>
/* 原有样式保持不变 */
.role-management-container {
  padding: 24px;
  background-color: #f5f5f5;
  min-height: 100vh;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #e8e8e8;
  transition: all 0.3s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: white;
}

.role-icon {
  background-color: #1890ff;
}

.active-icon {
  background-color: #52c41a;
}

.system-icon {
  background-color: #faad14;
}

.user-icon {
  background-color: #722ed1;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #262626;
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: #8c8c8c;
  margin-top: 4px;
  line-height: 1.4;
}

/* 主面板 */
.main-panel {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #e8e8e8;
}

.panel-header {
  margin-bottom: 24px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
}

.title-icon {
  font-size: 24px;
  color: #1890ff;
}

.header-title h2 {
  font-size: 20px;
  font-weight: 600;
  color: #262626;
  margin: 0;
}

.search-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.search-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.status-filter {
  width: 140px;
}

.status-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 22px;
  font-size: 14px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.active {
  background: #52c41a;
}

.status-dot.inactive {
  background: #d9d9d9;
}

.action-group {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 视图切换按钮样式 */
.view-toggle {
  border-radius: 6px;
  overflow: hidden;
}

.view-toggle :deep(.ant-radio-button-wrapper) {
  height: 32px;
  line-height: 30px;
  padding: 0 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  border: 1px solid #d9d9d9;
}

.view-toggle :deep(.ant-radio-button-wrapper:first-child) {
  border-right: none;
}

.view-toggle :deep(.ant-radio-button-wrapper-checked) {
  background: #1890ff;
  border-color: #1890ff;
  color: white;
}

.view-toggle :deep(.ant-radio-button-wrapper:hover) {
  color: #1890ff;
  border-color: #1890ff;
}

.view-toggle :deep(.ant-radio-button-wrapper-checked:hover) {
  color: white;
}

/* 角色网格保持原有样式 */
.role-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.role-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #e8e8e8;
  transition: all 0.3s ease;
  position: relative;
}

.role-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  border-color: #1890ff;
}

.role-card.system-role {
  border-left: 4px solid #faad14;
}

.role-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.role-title {
  flex: 1;
}

.role-name {
  font-size: 16px;
  font-weight: 600;
  color: #262626;
  margin-bottom: 4px;
  line-height: 1.3;
  word-break: break-word;
}

.role-code {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  display: inline-block;
  line-height: 1.4;
}

.role-description {
  color: #595959;
  line-height: 1.5;
  margin-bottom: 12px;
  min-height: 40px;
  font-size: 14px;
  word-break: break-word;
}

.role-stats {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #8c8c8c;
  line-height: 1.4;
}

.stat-item svg {
  color: #1890ff;
  font-size: 14px;
}

.role-tags {
  display: flex;
  gap: 6px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.system-tag, .status-tag {
  border-radius: 4px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  line-height: 1.4;
  white-space: nowrap;
}

.role-actions {
  display: flex;
  justify-content: center;
  gap: 4px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  margin-bottom: 8px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  border: none;
  font-size: 16px;
  padding: 0;
}

.view-btn:hover {
  background-color: #e6f7ff;
  color: #1890ff;
}

.edit-btn:hover {
  background-color: #f6ffed;
  color: #52c41a;
}

.permission-btn:hover {
  background-color: #fff7e6;
  color: #faad14;
}

.delete-btn:hover {
  background-color: #fff2f0;
  color: #ff4d4f;
}

.role-time {
  text-align: center;
  color: #bfbfbf;
  font-size: 12px;
}

/* 表格容器样式 */
.role-table-container {
  margin-bottom: 24px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #f0f0f0;
}

/* 表格自定义样式 */
.role-table {
  background: white;
}

.role-table :deep(.ant-table) {
  border-radius: 0;
}

.role-table :deep(.ant-table-thead > tr > th) {
  background: #fafafa;
  font-weight: 600;
  color: #262626;
  border-bottom: 2px solid #f0f0f0;
  padding: 16px 12px;
  font-size: 14px;
}

.role-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px 12px;
  border-bottom: 1px solid #f5f5f5;
  vertical-align: middle;
}

.role-table :deep(.ant-table-tbody > tr:hover > td) {
  background: #fafafa;
}

/* 表格单元格样式 */
.name-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.role-name-text {
  font-weight: 600;
  color: #262626;
  font-size: 15px;
  line-height: 1.4;
}

.role-code-text {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  display: inline-block;
  width: fit-content;
  line-height: 1.3;
}

.status-text {
  font-size: 12px;
  color: #8c8c8c;
  margin-top: 4px;
  text-align: center;
}

.description-cell {
  color: #595959;
  line-height: 1.4;
  font-size: 14px;
  max-width: 250px;
  word-break: break-word;
}

.count-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-weight: 500;
  color: #262626;
}

.count-cell svg {
  color: #1890ff;
  font-size: 14px;
}

.time-cell {
  color: #8c8c8c;
  font-size: 13px;
  text-align: center;
}

/* 表格操作按钮 */
.table-actions {
  display: flex;
  justify-content: center;
  gap: 4px;
}

.table-action-btn {
  width: 28px;
  height: 28px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  border: none;
  font-size: 14px;
  padding: 0;
}

.table-action-btn.view-btn:hover {
  background-color: #e6f7ff;
  color: #1890ff;
}

.table-action-btn.edit-btn:hover {
  background-color: #f6ffed;
  color: #52c41a;
}

.table-action-btn.permission-btn:hover {
  background-color: #fff7e6;
  color: #faad14;
}

.table-action-btn.delete-btn:hover {
  background-color: #fff2f0;
  color: #ff4d4f;
}

/* 分页 */
.pagination-container {
  display: flex;
  justify-content: center;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

/* 其他原有样式保持不变... */
/* 查看模态框样式 */
.view-modal :deep(.ant-modal-body) {
  max-height: 80vh;
  overflow-y: auto;
  padding: 24px;
}

.view-content {
  padding: 0;
}

.view-section {
  margin-bottom: 32px;
}

.view-section:last-child {
  margin-bottom: 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 2px solid #f0f0f0;
  color: #262626;
  font-weight: 600;
  font-size: 16px;
}

.section-header svg {
  color: #1890ff;
  font-size: 18px;
}

.section-extra {
  margin-left: auto;
}

.mini-search {
  border-radius: 6px;
}

/* 基本信息网格 */
.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-item.full-width {
  grid-column: 1 / -1;
}

.info-item label {
  font-size: 14px;
  font-weight: 500;
  color: #8c8c8c;
  margin: 0;
}

.info-value {
  font-size: 15px;
  color: #262626;
  line-height: 1.5;
  min-height: 22px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-value.code {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  background: #f5f5f5;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 14px;
  display: inline-block;
  width: fit-content;
}

.info-value.description {
  background: #fafafa;
  padding: 12px;
  border-radius: 6px;
  border: 1px solid #f0f0f0;
  min-height: 50px;
  align-items: flex-start;
}

/* 统计信息行 */
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.stats-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #f0f0f0;
}

.stats-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 18px;
}

.api-stats {
  background: linear-gradient(135deg, #1890ff, #40a9ff);
}

.user-stats {
  background: linear-gradient(135deg, #52c41a, #73d13d);
}

.time-stats {
  background: linear-gradient(135deg, #faad14, #ffc53d);
}

.stats-info {
  flex: 1;
}

.stats-number {
  font-size: 18px;
  font-weight: 600;
  color: #262626;
  line-height: 1.2;
}

.stats-label {
  font-size: 13px;
  color: #8c8c8c;
  margin-top: 2px;
}

/* 权限容器 */
.apis-container, .users-container {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.apis-summary, .users-summary {
  padding: 12px 16px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  color: #595959;
}

.method-stats {
  display: flex;
  gap: 12px;
  align-items: center;
}

.method-stat {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.method-badge {
  padding: 2px 6px;
  border-radius: 3px;
  color: white;
  font-weight: 500;
  font-size: 10px;
  text-transform: uppercase;
}

.method-badge.get {
  background: #1890ff;
}

.method-badge.post {
  background: #52c41a;
}

.method-badge.put {
  background: #faad14;
}

.method-badge.delete {
  background: #f5222d;
}

.method-badge.patch {
  background: #722ed1;
}

/* 权限列表 */
.apis-list {
  max-height: 300px;
  overflow-y: auto;
}

.api-detail-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-bottom: 1px solid #f5f5f5;
  transition: background-color 0.2s ease;
}

.api-detail-item:hover {
  background: #fafafa;
}

.api-detail-item:last-child {
  border-bottom: none;
}

.api-method-badge {
  font-size: 11px;
  font-weight: bold;
  padding: 6px 12px;
  border-radius: 6px;
  color: white;
  min-width: 60px;
  text-align: center;
  text-transform: uppercase;
  line-height: 1;
  flex-shrink: 0;
}

.api-detail-info {
  flex: 1;
  min-width: 0;
}

.api-detail-name {
  font-weight: 600;
  color: #262626;
  font-size: 15px;
  margin-bottom: 4px;
  line-height: 1.4;
}

.api-detail-path {
  font-size: 13px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  margin-bottom: 4px;
  word-break: break-all;
}

.api-detail-desc {
  font-size: 12px;
  color: #595959;
  line-height: 1.4;
}

.api-detail-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: #8c8c8c;
  text-align: right;
  flex-shrink: 0;
}

.api-category, .api-created {
  display: flex;
  align-items: center;
  gap: 4px;
  justify-content: flex-end;
}

/* 用户列表 */
.users-list {
  max-height: 250px;
  overflow-y: auto;
}

.user-detail-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-bottom: 1px solid #f5f5f5;
  transition: background-color 0.2s ease;
}

.user-detail-item:hover {
  background: #fafafa;
}

.user-detail-item:last-child {
  border-bottom: none;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #f5f5f5;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-avatar svg {
  font-size: 20px;
  color: #8c8c8c;
}

.user-detail-info {
  flex: 1;
  min-width: 0;
}

.user-detail-name {
  font-weight: 600;
  color: #262626;
  font-size: 15px;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
  line-height: 1.4;
}

.user-detail-email, .user-detail-phone {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 2px;
}

.user-detail-status {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}

.user-login-time {
  font-size: 11px;
  color: #bfbfbf;
  text-align: right;
}

/* 空状态 */
.empty-apis, .empty-users {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #8c8c8c;
}

.empty-icon {
  font-size: 48px;
  color: #d9d9d9;
  margin-bottom: 12px;
}

.empty-text {
  font-size: 14px;
  color: #8c8c8c;
}

/* 时间线 */
.timeline-container {
  padding-left: 20px;
}

.timeline-item {
  position: relative;
  padding-bottom: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.timeline-item:last-child {
  padding-bottom: 0;
}

.timeline-item::before {
  content: '';
  position: absolute;
  left: -12px;
  top: 12px;
  bottom: -8px;
  width: 2px;
  background: #f0f0f0;
}

.timeline-item:last-child::before {
  display: none;
}

.timeline-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  position: absolute;
  left: -18px;
  top: 6px;
  border: 2px solid white;
  z-index: 1;
}

.timeline-dot.create {
  background: #52c41a;
}

.timeline-dot.update {
  background: #1890ff;
}

.timeline-content {
  flex: 1;
}

.timeline-title {
  font-weight: 500;
  color: #262626;
  font-size: 14px;
  margin-bottom: 4px;
}

.timeline-time {
  font-size: 12px;
  color: #8c8c8c;
}

/* 底部操作栏 */
.view-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

/* 模态框样式修复 */
.role-modal :deep(.ant-select-dropdown) {
  z-index: 1060;
}

.modal-content {
  position: relative;
  max-height: 70vh;
  overflow-y: auto;
}

/* 表单样式 */
.form-section {
  margin-bottom: 24px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.status-radio {
  display: flex;
  gap: 24px;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 22px;
  font-size: 14px;
}

.api-select {
  width: 100%;
}

.api-select :deep(.ant-select-selector) {
  min-height: 32px;
}

.api-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
}

.api-method {
  font-size: 10px;
  font-weight: bold;
  padding: 3px 8px;
  border-radius: 4px;
  color: white;
  min-width: 50px;
  text-align: center;
  text-transform: uppercase;
  line-height: 1;
  flex-shrink: 0;
}

.api-method.get {
  background: #1890ff;
}

.api-method.post {
  background: #52c41a;
}

.api-method.put {
  background: #faad14;
}

.api-method.delete {
  background: #f5222d;
}

.api-info {
  flex: 1;
}

.api-name {
  font-weight: 500;
  color: #262626;
  font-size: 14px;
  line-height: 1.4;
  word-break: break-word;
}

.api-path {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  line-height: 1.4;
  word-break: break-all;
}

/* 权限管理模态框 */
.permission-modal :deep(.ant-select-dropdown) {
  z-index: 1060;
}

.permission-content {
  padding: 16px 0;
}

.permission-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.role-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: #262626;
}

.permission-stats {
  color: #8c8c8c;
  font-size: 14px;
}

.api-search {
  margin-bottom: 16px;
}

.api-list {
  max-height: 400px;
  overflow-y: auto;
}

.api-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  margin-bottom: 8px;
  transition: all 0.3s ease;
}

.api-item:hover {
  border-color: #d9d9d9;
  background: #fafafa;
}

.api-item.assigned {
  background: #f6ffed;
  border-color: #b7eb8f;
}

.api-details {
  flex: 1;
}

.assign-btn, .revoke-btn {
  border-radius: 4px;
  font-size: 12px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 60px;
  padding: 0 12px;
  justify-content: center;
  white-space: nowrap;
}

/* 按钮样式统一 */
.search-btn {
  min-width: 80px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 14px;
  white-space: nowrap;
  padding: 0 16px;
}

.refresh-btn {
  min-width: 72px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 14px;
  white-space: nowrap;
  padding: 0 16px;
}

.add-btn {
  min-width: 100px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 14px;
  white-space: nowrap;
  padding: 0 16px;
}

/* 模态框按钮 */
.role-modal .ant-modal-footer .ant-btn {
  min-width: 80px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 14px;
  white-space: nowrap;
  padding: 0 16px;
}

/* 确保按钮内容居中 */
.ant-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.ant-btn-sm {
  gap: 4px;
}

/* 修复图标按钮 */
.ant-btn[class*="icon-only"] {
  padding: 0;
  width: 32px;
  height: 32px;
  gap: 0;
}

/* 修复开关组件 */
.ant-switch-small {
  min-width: 28px;
  height: 16px;
}

/* 输入框高度统一 */
.ant-input,
.ant-select-selector {
  height: 32px;
  line-height: 30px;
  font-size: 14px;
}

/* 表单标签 */
.ant-form-item-label > label {
  font-size: 14px;
  line-height: 1.4;
}

/* 模态框标题 */
.ant-modal-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

/* 修复分页按钮 */
.custom-pagination .ant-pagination-item,
.custom-pagination .ant-pagination-prev,
.custom-pagination .ant-pagination-next {
  min-width: 32px;
  height: 32px;
  line-height: 30px;
  font-size: 14px;
}

/* 确保所有文字都能完整显示 */
* {
  box-sizing: border-box;
}

.text-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 如果需要文字省略，可以添加这个类 */
.multiline-ellipsis {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .role-management-container {
    padding: 16px;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .search-section {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-group {
    justify-content: center;
  }
  
  .role-grid {
    grid-template-columns: 1fr;
  }
  
  .form-grid {
    grid-template-columns: 1fr;
  }
  
  .info-grid {
    grid-template-columns: 1fr;
  }
  
  .stats-row {
    grid-template-columns: 1fr;
  }
  
  .api-detail-item, .user-detail-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .api-detail-meta, .user-detail-status {
    align-self: stretch;
    text-align: left;
  }
  
  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .section-extra {
    margin-left: 0;
    align-self: stretch;
  }
  
  .mini-search {
    width: 100% !important;
  }
  
  .search-btn,
  .refresh-btn,
  .add-btn {
    min-width: 60px;
    font-size: 13px;
    padding: 0 12px;
  }
  
  .role-name {
    font-size: 15px;
  }
  
  .stat-value {
    font-size: 20px;
  }
  
  /* 移动端表格横向滚动 */
  .role-table-container {
    overflow-x: auto;
  }
  
  /* 移动端视图切换器 */
  .view-toggle {
    width: 100%;
    margin-bottom: 8px;
  }
  
  .view-toggle :deep(.ant-radio-button-wrapper) {
    flex: 1;
    text-align: center;
  }
}

@media (max-width: 480px) {
  .search-input {
    width: 100%;
  }
  
  .search-group {
    flex-direction: column;
  }
  
  .search-btn,
  .refresh-btn,
  .add-btn {
    min-width: 50px;
    font-size: 12px;
    padding: 0 8px;
    height: 30px;
  }
  
  .action-btn {
    width: 28px;
    height: 28px;
    font-size: 14px;
  }
  
  .table-action-btn {
    width: 24px;
    height: 24px;
    font-size: 12px;
  }
  
  .role-name {
    font-size: 14px;
  }
  
  .stat-value {
    font-size: 18px;
  }
  
  .stat-label {
    font-size: 13px;
  }
  
  .view-modal :deep(.ant-modal-body) {
    padding: 16px;
  }
  
  .method-stats {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
  
  .apis-summary, .users-summary {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .view-footer {
    justify-content: center;
  }
  
  .view-footer .ant-space {
    width: 100%;
    justify-content: center;
  }
}
</style>