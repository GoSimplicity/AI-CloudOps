<template>
  <div class="user-management-container">
    <!-- 顶部统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon user-icon">
          <Icon icon="material-symbols:person-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ userList.length }}</div>
          <div class="stat-label">总用户数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon active-icon">
          <Icon icon="material-symbols:check-circle-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ activeUsers }}</div>
          <div class="stat-label">活跃用户</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon admin-icon">
          <Icon icon="material-symbols:admin-panel-settings-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ adminUsers }}</div>
          <div class="stat-label">管理员用户</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon role-icon">
          <Icon icon="material-symbols:badge-outline" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ totalRoles }}</div>
          <div class="stat-label">关联角色</div>
        </div>
      </div>
    </div>

    <!-- 主控制面板 -->
    <div class="main-panel">
      <div class="panel-header">
        <div class="header-title">
          <Icon icon="material-symbols:group-outline" class="title-icon" />
          <h2>用户权限管理</h2>
        </div>
        
        <!-- 搜索和筛选区域 -->
        <div class="search-section">
          <div class="search-group">
            <a-input
              v-model:value="searchParams.search"
              placeholder="搜索用户名或真实姓名"
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
              新建用户
            </a-button>
          </div>
        </div>
      </div>

      <!-- 卡片视图 -->
      <div v-if="viewMode === 'card'" class="user-grid">
        <div 
          v-for="user in paginatedUsers" 
          :key="user.id" 
          class="user-card"
          :class="{ 'admin-user': isAdminUser(user) }"
        >
          <div class="user-header">
            <div class="user-avatar-section">
              <div class="user-avatar">
                <Icon icon="material-symbols:person" v-if="!user.avatar" />
                <img v-else :src="user.avatar" :alt="user.username" />
              </div>
              <div class="user-title">
                <div class="user-name">{{ user.real_name || user.username }}</div>
                <div class="user-username">@{{ user.username }}</div>
              </div>
            </div>
            <div class="user-status">
              <a-switch 
                v-model:checked="user.enable" 
                :checked-value="1" 
                :unchecked-value="0"
                @change="handleStatusChange(user)"
                size="small"
              />
            </div>
          </div>
          <div class="user-stats">
            <div class="stat-item">
              <Icon icon="material-symbols:badge" />
              <span>{{ user.roles?.length || 0 }} 个角色</span>
            </div>
            <div class="stat-item">
              <Icon icon="material-symbols:api" />
              <span>{{ user.apis?.length || 0 }} 个权限</span>
            </div>
          </div>
          
          <div class="user-tags">
            <a-tag v-if="isAdminUser(user)" color="red" class="admin-tag">
              <Icon icon="material-symbols:admin-panel-settings" />
              管理员
            </a-tag>
            <a-tag :color="user.enable === 1 ? 'green' : 'default'" class="status-tag">
              {{ user.enable === 1 ? '已启用' : '已禁用' }}
            </a-tag>
          </div>
          
          <div class="user-actions">
            <a-tooltip title="查看详情">
              <a-button type="text" @click="handleView(user)" class="action-btn view-btn">
                <Icon icon="material-symbols:visibility-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="编辑用户">
              <a-button type="text" @click="handleEdit(user)" class="action-btn edit-btn">
                <Icon icon="material-symbols:edit-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="修改密码">
              <a-button type="text" @click="handleChangePassword(user)" class="action-btn password-btn">
                <Icon icon="material-symbols:key-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="角色管理">
              <a-button type="text" @click="handleRoleManagement(user)" class="action-btn role-btn">
                <Icon icon="material-symbols:badge-outline" />
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除用户">
              <a-popconfirm
                title="确定要删除这个用户吗？"
                @confirm="handleDelete(user)"
              >
                <a-button 
                  type="text" 
                  danger 
                  class="action-btn delete-btn"
                >
                  <Icon icon="material-symbols:delete-outline" />
                </a-button>
              </a-popconfirm>
            </a-tooltip>
          </div>
          
          <div class="user-time">
            <small>注册时间：{{ formatTime(user.created_at) }}</small>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else class="user-table-container">
        <a-table
          :columns="tableColumns"
          :data-source="paginatedUsers"
          :pagination="false"
          :scroll="{ x: 1400 }"
          row-key="id"
          class="user-table"
          size="middle"
        >
          <template #bodyCell="{ column, record, text }">
            <!-- 用户名称列 -->
            <template v-if="column.key === 'name'">
              <div class="name-cell">
                <div class="user-avatar-table">
                  <Icon icon="material-symbols:person" v-if="!record.avatar" />
                  <img v-else :src="record.avatar" :alt="record.username" />
                </div>
                <div class="user-info-table">
                  <div class="user-name-text">{{ record.real_name || record.username }}</div>
                  <div class="user-username-text">@{{ record.username }}</div>
                </div>
              </div>
            </template>
            
            <!-- 联系方式列 -->
            <template v-if="column.key === 'contact'">
              <div class="contact-cell">
                <div class="contact-item" v-if="record.mobile">
                  <Icon icon="material-symbols:phone" />
                  {{ record.mobile }}
                </div>
                <div class="contact-item" v-if="record.email">
                  <Icon icon="material-symbols:email-outline" />
                  {{ record.email }}
                </div>
              </div>
            </template>
            
            <!-- 用户类型列 -->
            <template v-if="column.key === 'type'">
              <a-tag v-if="isAdminUser(record)" color="orange">
                <Icon icon="material-symbols:admin-panel-settings" />
                管理员
              </a-tag>
              <a-tag v-else color="blue">
                <Icon icon="material-symbols:person-outline" />
                普通用户
              </a-tag>
            </template>
            
            <!-- 状态列 -->
            <template v-if="column.key === 'status'">
              <a-switch 
                v-model:checked="record.enable" 
                :checked-value="1" 
                :unchecked-value="0"
                @change="handleStatusChange(record)"
                size="small"
              />
              <div class="status-text">
                {{ record.enable === 1 ? '已启用' : '已禁用' }}
              </div>
            </template>
            
            <!-- 角色数量列 -->
            <template v-if="column.key === 'roles'">
              <div class="count-cell">
                <Icon icon="material-symbols:badge" />
                {{ record.roles?.length || 0 }}
              </div>
            </template>
            
            <!-- 权限数量列 -->
            <template v-if="column.key === 'apis'">
              <div class="count-cell">
                <Icon icon="material-symbols:api" />
                {{ record.apis?.length || 0 }}
              </div>
            </template>
            
            <!-- 最后登录列 -->
            <template v-if="column.key === 'last_login'">
              <div class="time-cell">
                {{ formatTime(record.last_login_at) }}
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
                <a-tooltip title="编辑用户">
                  <a-button type="text" @click="handleEdit(record)" class="table-action-btn edit-btn">
                    <Icon icon="material-symbols:edit-outline" />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="修改密码">
                  <a-button type="text" @click="handleChangePassword(record)" class="table-action-btn password-btn">
                    <Icon icon="material-symbols:key-outline" />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="角色管理">
                  <a-button type="text" @click="handleRoleManagement(record)" class="table-action-btn role-btn">
                    <Icon icon="material-symbols:badge-outline" />
                  </a-button>
                </a-tooltip>
                <a-tooltip title="删除用户">
                  <a-popconfirm
                    title="确定要删除这个用户吗？"
                    @confirm="handleDelete(record)"
                  >
                    <a-button 
                      type="text" 
                      danger 
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
          :total="filteredUsers.length"
          :show-size-changer="true"
          :show-quick-jumper="true"
          :show-total="(total: number, range: [number, number]) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`"
          class="custom-pagination"
        />
      </div>
    </div>

    <!-- 查看详情模态框 -->
    <a-modal
      v-model:open="viewModalVisible"
      title="用户详情"
      width="900px"
      :mask-closable="false"
      :footer="null"
      class="view-modal"
    >
      <div class="view-content" v-if="viewModalVisible && viewUserData">
        <!-- 基本信息 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:info-outline" />
            <span>基本信息</span>
          </div>
          <div class="info-grid">
            <div class="info-item">
              <label>用户名</label>
              <div class="info-value">{{ viewUserData.username }}</div>
            </div>
            <div class="info-item">
              <label>真实姓名</label>
              <div class="info-value">{{ viewUserData.real_name || '未设置' }}</div>
            </div>
            <div class="info-item">
              <label>用户状态</label>
              <div class="info-value">
                <a-tag :color="viewUserData.enable === 1 ? 'green' : 'red'">
                  <div class="status-option">
                    <div class="status-dot" :class="viewUserData.enable === 1 ? 'active' : 'inactive'"></div>
                    <span>{{ viewUserData.enable === 1 ? '已启用' : '已禁用' }}</span>
                  </div>
                </a-tag>
              </div>
            </div>
            <div class="info-item">
              <label>用户类型</label>
              <div class="info-value">
                <a-tag v-if="isAdminUser(viewUserData)" color="orange">
                  <Icon icon="material-symbols:admin-panel-settings" />
                  管理员用户
                </a-tag>
                <a-tag v-else color="blue">
                  <Icon icon="material-symbols:person-outline" />
                  普通用户
                </a-tag>
              </div>
            </div>
            <div class="info-item">
              <label>手机号码</label>
              <div class="info-value">{{ viewUserData.mobile || '未设置' }}</div>
            </div>
            <div class="info-item">
              <label>邮箱地址</label>
              <div class="info-value">{{ viewUserData.email || '未设置' }}</div>
            </div>
            <div class="info-item">
              <label>飞书用户ID</label>
              <div class="info-value">{{ viewUserData.fei_shu_user_id || '未设置' }}</div>
            </div>
            <div class="info-item">
              <label>首页路径</label>
              <div class="info-value code">{{ viewUserData.home_path || '默认' }}</div>
            </div>
            <div class="info-item full-width">
              <label>用户描述</label>
              <div class="info-value description">
                {{ viewUserData.desc || '暂无描述' }}
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
              <div class="stats-icon role-stats">
                <Icon icon="material-symbols:badge" />
              </div>
              <div class="stats-info">
                <div class="stats-number">{{ viewUserData.roles?.length || 0 }}</div>
                <div class="stats-label">分配角色</div>
              </div>
            </div>
            <div class="stats-item">
              <div class="stats-icon api-stats">
                <Icon icon="material-symbols:api" />
              </div>
              <div class="stats-info">
                <div class="stats-number">{{ viewUserData.apis?.length || 0 }}</div>
                <div class="stats-label">直接权限</div>
              </div>
            </div>
            <div class="stats-item">
              <div class="stats-icon time-stats">
                <Icon icon="material-symbols:schedule" />
              </div>
              <div class="stats-info">
                <div class="stats-number">{{ formatTime(viewUserData.last_login_at) || '从未登录' }}</div>
                <div class="stats-label">最后登录</div>
              </div>
            </div>
          </div>
        </div>

        <!-- 角色列表 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:badge-outline" />
            <span>角色详情</span>
            <div class="section-extra">
              <a-input
                v-model:value="viewRoleSearch"
                placeholder="搜索角色"
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
          
          <div v-if="viewUserData.roles && viewUserData.roles.length > 0" class="roles-container">
            <div class="roles-summary">
              <span>共 {{ filteredViewRoles.length }} 个角色</span>
            </div>
            
            <div class="roles-list">
              <div v-for="role in filteredViewRoles" :key="role.id" class="role-detail-item">
                <div class="role-status-badge" :class="role.status === 1 ? 'active' : 'inactive'">
                  {{ role.status === 1 ? '启用' : '禁用' }}
                </div>
                <div class="role-detail-info">
                  <div class="role-detail-name">{{ role.name }}</div>
                  <div class="role-detail-code">{{ role.code }}</div>
                  <div class="role-detail-desc" v-if="role.description">
                    {{ role.description }}
                  </div>
                </div>
                <div class="role-detail-meta">
                  <div class="role-apis" v-if="role.apis">
                    <Icon icon="material-symbols:api" />
                    {{ role.apis.length }} 个权限
                  </div>
                  <div class="role-created">
                    <Icon icon="material-symbols:schedule" />
                    {{ formatTime(role.created_at) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-else class="empty-roles">
            <Icon icon="material-symbols:badge-off-outline" class="empty-icon" />
            <div class="empty-text">该用户暂未分配任何角色</div>
          </div>
        </div>

        <!-- 直接权限 -->
        <div class="view-section">
          <div class="section-header">
            <Icon icon="material-symbols:key-outline" />
            <span>直接权限</span>
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
          
          <div v-if="viewUserData.apis && viewUserData.apis.length > 0" class="apis-container">
            <div class="apis-summary">
              <span>共 {{ filteredViewApis.length }} 个直接权限</span>
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
            <div class="empty-text">该用户暂未分配任何直接权限</div>
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
                <div class="timeline-title">用户注册</div>
                <div class="timeline-time">{{ formatTime(viewUserData.created_at) }}</div>
              </div>
            </div>
            <div class="timeline-item" v-if="viewUserData.last_login_at">
              <div class="timeline-dot login"></div>
              <div class="timeline-content">
                <div class="timeline-title">最后登录</div>
                <div class="timeline-time">{{ formatTime(viewUserData.last_login_at) }}</div>
              </div>
            </div>
            <div class="timeline-item" v-if="viewUserData.updated_at && viewUserData.updated_at !== viewUserData.created_at">
              <div class="timeline-dot update"></div>
              <div class="timeline-content">
                <div class="timeline-title">最后更新</div>
                <div class="timeline-time">{{ formatTime(viewUserData.updated_at) }}</div>
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
              编辑用户
            </a-button>
            <a-button @click="handleRoleManagementFromView">
              <Icon icon="material-symbols:badge-outline" />
              角色管理
            </a-button>
          </a-space>
        </div>
      </template>
    </a-modal>

    <!-- 用户表单弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      width="800px"
      :mask-closable="false"
      :destroy-on-close="true"
      class="user-modal"
      :key="modalVisible ? 'open' : 'closed'"
    >
      <div class="modal-content" v-if="modalVisible">
        <a-form
          ref="formRef"
          :model="formData"
          :rules="formRules"
          layout="vertical"
          class="user-form"
          :key="formData.id || 'new'"
        >
          <div class="form-section">
            <div class="section-header">
              <Icon icon="material-symbols:info-outline" />
              <span>基本信息</span>
            </div>
            <div class="form-grid">
              <a-form-item label="用户名" name="username">
                <a-input 
                  v-model:value="formData.username" 
                  placeholder="请输入用户名"
                  :disabled="modalTitle === '编辑用户'"
                />
              </a-form-item>
              <a-form-item label="真实姓名" name="real_name">
                <a-input 
                  v-model:value="formData.real_name" 
                  placeholder="请输入真实姓名"
                />
              </a-form-item>
              <a-form-item label="手机号码" name="mobile">
                <a-input 
                  v-model:value="formData.mobile" 
                  placeholder="请输入手机号码"
                />
              </a-form-item>
              <a-form-item label="邮箱地址" name="email">
                <a-input 
                  v-model:value="formData.email" 
                  placeholder="请输入邮箱地址"
                />
              </a-form-item>
              <a-form-item label="飞书用户ID" name="fei_shu_user_id">
                <a-input 
                  v-model:value="formData.fei_shu_user_id" 
                  placeholder="请输入飞书用户ID"
                />
              </a-form-item>
              <a-form-item label="首页路径" name="home_path">
                <a-input 
                  v-model:value="formData.home_path" 
                  placeholder="请输入首页路径"
                />
              </a-form-item>
            </div>
            
            <div class="form-grid" v-if="modalTitle === '新建用户'">
              <a-form-item label="密码" name="password">
                <a-input-password 
                  v-model:value="formData.password" 
                  placeholder="请输入密码"
                />
              </a-form-item>
              <a-form-item label="确认密码" name="confirmPassword">
                <a-input-password 
                  v-model:value="formData.confirmPassword" 
                  placeholder="请再次输入密码"
                />
              </a-form-item>
            </div>
            
            <a-form-item label="用户描述" name="desc">
              <a-textarea 
                v-model:value="formData.desc" 
                placeholder="请输入用户描述"
                :rows="3"
              />
            </a-form-item>
            <a-form-item label="用户状态" name="enable">
              <a-radio-group v-model:value="formData.enable" class="status-radio">
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

    <!-- 修改密码弹窗 -->
    <a-modal
      v-model:open="passwordModalVisible"
      title="修改密码"
      width="500px"
      :mask-closable="false"
      class="password-modal"
    >
      <div class="modal-content" v-if="passwordModalVisible">
        <div class="user-info-banner">
          <div class="user-avatar-small">
            <Icon icon="material-symbols:person" />
          </div>
          <div class="user-info-text">
            <div class="user-name">{{ currentUser?.real_name || currentUser?.username }}</div>
            <div class="user-username">@{{ currentUser?.username }}</div>
          </div>
        </div>
        
        <a-form
          ref="passwordFormRef"
          :model="passwordData"
          :rules="passwordRules"
          layout="vertical"
          class="password-form"
        >
          <a-form-item label="原密码" name="password">
            <a-input-password 
              v-model:value="passwordData.password" 
              placeholder="请输入原密码"
            />
          </a-form-item>
          <a-form-item label="新密码" name="newPassword">
            <a-input-password 
              v-model:value="passwordData.newPassword" 
              placeholder="请输入新密码"
            />
          </a-form-item>
          <a-form-item label="确认新密码" name="confirmPassword">
            <a-input-password 
              v-model:value="passwordData.confirmPassword" 
              placeholder="请再次输入新密码"
            />
          </a-form-item>
        </a-form>
      </div>
      
      <template #footer>
        <a-space>
          <a-button @click="passwordModalVisible = false">取消</a-button>
          <a-button type="primary" @click="handlePasswordSubmit" :loading="passwordLoading">
            确认修改
          </a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- 角色管理弹窗 -->
    <a-modal
      v-model:open="roleModalVisible"
      title="角色管理"
      width="1000px"
      :mask-closable="false"
      class="role-modal"
      :key="roleModalVisible ? 'role-open' : 'role-closed'"
    >
      <div class="role-content" v-if="roleModalVisible">
        <div class="role-header">
          <div class="user-info">
            <Icon icon="material-symbols:person-outline" />
            <span>{{ currentUser?.real_name || currentUser?.username }}</span>
          </div>
          <div class="role-stats">
            <span>已分配 {{ assignedRoles.length }} 个角色</span>
          </div>
        </div>
        
        <a-tabs v-model:activeKey="roleTab" class="role-tabs">
          <a-tab-pane key="assigned" tab="已分配角色">
            <div class="role-list">
              <div 
                v-for="role in assignedRoles" 
                :key="role.id" 
                class="role-item assigned"
              >
                <div class="role-status" :class="role.status === 1 ? 'active' : 'inactive'">
                  {{ role.status === 1 ? '启用' : '禁用' }}
                </div>
                <div class="role-details">
                  <div class="role-name">{{ role.name }}</div>
                  <div class="role-code">{{ role.code }}</div>
                  <div class="role-desc" v-if="role.description">{{ role.description }}</div>
                </div>
                <a-button 
                  type="text" 
                  danger 
                  @click="handleRevokeRole(role)"
                  class="revoke-btn"
                >
                  <Icon icon="material-symbols:remove-circle-outline" />
                  移除
                </a-button>
              </div>
            </div>
          </a-tab-pane>
          
          <a-tab-pane key="available" tab="可分配角色">
            <div class="role-search">
              <a-input
                v-model:value="roleSearchText"
                placeholder="搜索角色"
                class="search-input"
              >
                <template #prefix>
                  <Icon icon="ri:search-line" />
                </template>
              </a-input>
            </div>
            <div class="role-list">
              <div 
                v-for="role in availableRoles" 
                :key="role.id" 
                class="role-item available"
              >
                <div class="role-status" :class="role.status === 1 ? 'active' : 'inactive'">
                  {{ role.status === 1 ? '启用' : '禁用' }}
                </div>
                <div class="role-details">
                  <div class="role-name">{{ role.name }}</div>
                  <div class="role-code">{{ role.code }}</div>
                  <div class="role-desc" v-if="role.description">{{ role.description }}</div>
                </div>
                <a-button 
                  type="primary" 
                  size="small"
                  @click="handleAssignRole(role)"
                  class="assign-btn"
                  :disabled="role.status === 0"
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
        <a-button @click="roleModalVisible = false">关闭</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted, nextTick, onBeforeUnmount, watch } from 'vue';
import { message } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import type { FormInstance } from 'ant-design-vue';
import { 
  getAllUsers, 
  registerApi, 
  updateUserInfo, 
  deleteUser, 
  changePassword,
  getUserDetailApi,
  assignRolesToUserApi,
  revokeRolesFromUserApi,
  getUserRolesApi
} from '#/api';
import { listRolesApi } from '#/api/core/system';

// 表单引用
const formRef = ref<FormInstance>();
const passwordFormRef = ref<FormInstance>();

// 视图模式状态
const viewMode = ref<'card' | 'list'>('card');

// 表格列配置
const tableColumns = [
  {
    title: '用户信息',
    key: 'name',
    width: 220,
    fixed: 'left'
  },
  {
    title: '联系方式',
    key: 'contact',
    width: 200
  },
  {
    title: '用户类型',
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
    title: '角色数',
    key: 'roles',
    width: 100,
    align: 'center'
  },
  {
    title: '权限数',
    key: 'apis',
    width: 100,
    align: 'center'
  },
  {
    title: '最后登录',
    key: 'last_login',
    width: 160,
    align: 'center'
  },
  {
    title: '注册时间',
    key: 'created_at',
    width: 160,
    align: 'center'
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    align: 'center',
    fixed: 'right'
  }
];

// 响应式数据
const loading = ref(false);
const submitLoading = ref(false);
const passwordLoading = ref(false);
const modalVisible = ref(false);
const passwordModalVisible = ref(false);
const roleModalVisible = ref(false);
const viewModalVisible = ref(false);
const modalTitle = ref('');
const userList = ref<any[]>([]);
const roleList = ref<any[]>([]);
const currentUser = ref<any>(null);
const assignedRoles = ref<any[]>([]);
const roleSearchText = ref('');
const roleTab = ref('assigned');
const viewUserData = ref<any>(null);
const viewRoleSearch = ref('');
const viewApiSearch = ref('');

// 搜索参数
const searchParams = reactive({
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
  username: '',
  real_name: '',
  mobile: '',
  email: '',
  fei_shu_user_id: '',
  home_path: '',
  desc: '',
  enable: 1,
  account_type: 1,
  password: '',
  confirmPassword: ''
});

// 表单数据
const formData = reactive<any>(initFormData());

// 密码表单数据
const passwordData = reactive({
  username: '',
  password: '',
  newPassword: '',
  confirmPassword: ''
});

// 表单验证规则
const formRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  real_name: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    {
      validator: (rule: any, value: string) => {
        if (value && value !== formData.password) {
          return Promise.reject('两次输入的密码不一致');
        }
        return Promise.resolve();
      },
      trigger: 'blur'
    }
  ],
  enable: [{ required: true, message: '请选择用户状态', trigger: 'change' }]
};

// 密码验证规则
const passwordRules = {
  password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (rule: any, value: string) => {
        if (value && value !== passwordData.newPassword) {
          return Promise.reject('两次输入的密码不一致');
        }
        return Promise.resolve();
      },
      trigger: 'blur'
    }
  ]
};

// 工具函数
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

const formatTime = (timestamp: number | string | undefined) => {
  if (!timestamp) return '-';
  
  // 如果是时间戳（数字）
  if (typeof timestamp === 'number') {
    return new Date(timestamp * 1000).toLocaleString('zh-CN');
  }
  
  // 如果是字符串
  return new Date(timestamp).toLocaleString('zh-CN');
};

const isAdminUser = (user: any): boolean => {
  return user.roles?.some((role: any) => role.code === 'admin' || role.is_system === 1) || false;
};

// 计算属性
const activeUsers = computed(() => 
  userList.value.filter((user: any) => user.enable === 1).length
);

const adminUsers = computed(() => 
  userList.value.filter((user: any) => isAdminUser(user)).length
);

const totalRoles = computed(() => 
  userList.value.reduce((total: number, user: any) => total + (user.roles?.length || 0), 0)
);

const filteredUsers = computed(() => {
  let filtered = userList.value;
  
  if (searchParams.search) {
    const searchText = searchParams.search.toLowerCase();
    filtered = filtered.filter((user: any) => 
      user.username.toLowerCase().includes(searchText) ||
      (user.real_name && user.real_name.toLowerCase().includes(searchText))
    );
  }
  
  if (searchParams.status !== undefined) {
    filtered = filtered.filter((user: any) => user.enable === searchParams.status);
  }
  
  return filtered;
});

const paginatedUsers = computed(() => {
  const start = (pagination.current - 1) * pagination.pageSize;
  const end = start + pagination.pageSize;
  return filteredUsers.value.slice(start, end);
});

const availableRoles = computed(() => {
  if (!currentUser.value) return [];
  
  const assignedIds = assignedRoles.value.map(role => role.id);
  let available = roleList.value.filter(role => !assignedIds.includes(role.id));
  
  if (roleSearchText.value) {
    const searchText = roleSearchText.value.toLowerCase();
    available = available.filter(role => 
      role.name.toLowerCase().includes(searchText) ||
      role.code.toLowerCase().includes(searchText)
    );
  }
  
  return available;
});

// 查看页面的计算属性
const filteredViewRoles = computed(() => {
  if (!viewUserData.value?.roles) return [];
  
  let filtered = viewUserData.value.roles;
  if (viewRoleSearch.value) {
    const searchText = viewRoleSearch.value.toLowerCase();
    filtered = filtered.filter((role: any) => 
      role.name.toLowerCase().includes(searchText) ||
      role.code.toLowerCase().includes(searchText)
    );
  }
  
  return filtered;
});

const filteredViewApis = computed(() => {
  if (!viewUserData.value?.apis) return [];
  
  let filtered = viewUserData.value.apis;
  if (viewApiSearch.value) {
    const searchText = viewApiSearch.value.toLowerCase();
    filtered = filtered.filter((api: any) => 
      api.name.toLowerCase().includes(searchText) ||
      api.path.toLowerCase().includes(searchText)
    );
  }
  
  return filtered;
});

const viewApiMethodStats = computed(() => {
  if (!viewUserData.value?.apis) return {};
  
  const stats: Record<string, number> = {};
  viewUserData.value.apis.forEach((api: any) => {
    const method = formatMethod(api.method);
    stats[method] = (stats[method] || 0) + 1;
  });
  
  return stats;
});

// 数据获取函数
const fetchUserList = async () => {
  loading.value = true;
  try {
    const response = await getAllUsers();
    userList.value = response || [];
    pagination.total = response?.length || 0;
  } catch (error: any) {
    message.error(error.message || '获取用户列表失败');
  } finally {
    loading.value = false;
  }
};

const fetchRoleList = async () => {
  try {
    const response = await listRolesApi({
      page: 1,
      size: 100
    });
    roleList.value = response.items || [];
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
  }
};

const fetchUserRoles = async (userId: number) => {
  try {
    const response = await getUserRolesApi(userId);
    assignedRoles.value = response.items || [];
  } catch (error: any) {
    message.error(error.message || '获取用户角色失败');
  }
};

// 事件处理函数
const handleSearch = () => {
  pagination.current = 1;
  fetchUserList();
};

const handleRefresh = () => {
  searchParams.search = '';
  searchParams.status = undefined;
  pagination.current = 1;
  fetchUserList();
};

const handleAdd = async () => {
  modalTitle.value = '新建用户';
  Object.assign(formData, initFormData());
  await nextTick();
  modalVisible.value = true;
  await nextTick();
  formRef.value?.resetFields();
};

const handleEdit = async (user: any) => {
  try {
    modalTitle.value = '编辑用户';
    modalVisible.value = false;
    await nextTick();
    
    // 获取用户详情
    const response = await getUserDetailApi(user.id);
    const userDetail = response;
    
    Object.assign(formData, {
      id: userDetail.id,
      username: userDetail.username,
      real_name: userDetail.real_name,
      mobile: userDetail.mobile,
      email: userDetail.email,
      fei_shu_user_id: userDetail.fei_shu_user_id,
      home_path: userDetail.home_path,
      desc: userDetail.desc,
      enable: userDetail.enable,
      account_type: userDetail.account_type
    });
    
    modalVisible.value = true;
    await nextTick();
    formRef.value?.clearValidate();
  } catch (error: any) {
    message.error(error.message || '获取用户详情失败');
  }
};

const handleView = async (user: any) => {
  try {
    const response = await getUserDetailApi(user.id);
    viewUserData.value = response;
    
    viewRoleSearch.value = '';
    viewApiSearch.value = '';
    
    viewModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取用户详情失败');
  }
};

const handleChangePassword = (user: any) => {
  currentUser.value = user;
  Object.assign(passwordData, {
    username: user.username,
    password: '',
    newPassword: '',
    confirmPassword: ''
  });
  passwordModalVisible.value = true;
};

const handleRoleManagement = async (user: any) => {
  currentUser.value = user;
  await fetchUserRoles(user.id);
  roleModalVisible.value = true;
};

const handleDelete = async (user: any) => {
  try {
    await deleteUser(user.id);
    message.success('删除成功');
    fetchUserList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

const handleStatusChange = async (user: any) => {
  const originalStatus = user.enable;
  try {
    const enableValue = user.enable ? 1 : 0;
    
    await updateUserInfo({
      user_id: user.id,
      real_name: user.real_name,
      mobile: user.mobile,
      email: user.email,
      fei_shu_user_id: user.fei_shu_user_id,
      home_path: user.home_path,
      desc: user.desc,
      account_type: user.account_type || 1,
      enable: enableValue 
    });
    message.success('状态更新成功');
  } catch (error: any) {
    message.error(error.message || '状态更新失败');
    user.enable = originalStatus;
  }
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    submitLoading.value = true;
    
    if (modalTitle.value === '新建用户') {
      await registerApi({
        username: formData.username,
        password: formData.password,
        confirmPassword: formData.confirmPassword,
        real_name: formData.real_name,
        mobile: formData.mobile,
        email: formData.email,
        fei_shu_user_id: formData.fei_shu_user_id,
        home_path: formData.home_path,
        desc: formData.desc
      });
      message.success('创建成功');
    } else {
      await updateUserInfo({
        user_id: formData.id,
        real_name: formData.real_name,
        mobile: formData.mobile,
        email: formData.email,
        fei_shu_user_id: formData.fei_shu_user_id,
        home_path: formData.home_path,
        desc: formData.desc,
        account_type: formData.account_type || 1,
        enable: formData.enable
      });
      message.success('更新成功');
    }
    
    handleCancel();
    fetchUserList();
  } catch (error: any) {
    if (error.errorFields) {
      return;
    }
    message.error(error.message || '操作失败');
  } finally {
    submitLoading.value = false;
  }
};

const handlePasswordSubmit = async () => {
  try {
    await passwordFormRef.value?.validate();
    passwordLoading.value = true;
    
    await changePassword({
      username: passwordData.username,
      password: passwordData.password,
      newPassword: passwordData.newPassword,
      confirmPassword: passwordData.confirmPassword
    });
    
    message.success('密码修改成功');
    passwordModalVisible.value = false;
  } catch (error: any) {
    if (error.errorFields) {
      return;
    }
    message.error(error.message || '密码修改失败');
  } finally {
    passwordLoading.value = false;
  }
};

const handleCancel = async () => {
  modalVisible.value = false;
  
  setTimeout(() => {
    Object.assign(formData, initFormData());
    formRef.value?.resetFields();
  }, 300);
};

const handleAssignRole = async (role: any) => {
  if (!currentUser.value) return;
  
  try {
    await assignRolesToUserApi({
      user_id: currentUser.value.id,
      role_ids: [role.id]
    });
    message.success('角色分配成功');
    await fetchUserRoles(currentUser.value.id);
  } catch (error: any) {
    message.error(error.message || '角色分配失败');
  }
};

const handleRevokeRole = async (role: any) => {
  if (!currentUser.value) return;
  
  try {
    await revokeRolesFromUserApi({
      user_id: currentUser.value.id,
      role_ids: [role.id]
    });
    message.success('角色移除成功');
    await fetchUserRoles(currentUser.value.id);
  } catch (error: any) {
    message.error(error.message || '角色移除失败');
  }
};

// 从查看页面跳转到编辑的函数
const handleEditFromView = () => {
  if (viewUserData.value) {
    viewModalVisible.value = false;
    handleEdit(viewUserData.value);
  }
};

// 从查看页面跳转到角色管理的函数
const handleRoleManagementFromView = () => {
  if (viewUserData.value) {
    viewModalVisible.value = false;
    handleRoleManagement(viewUserData.value);
  }
};

// 监听模态框状态变化
watch(modalVisible, async (newVal) => {
  if (!newVal) {
    setTimeout(() => {
      Object.assign(formData, initFormData());
    }, 300);
  }
});

// 监听视图模式变化，调整分页大小
watch(viewMode, (newMode) => {
  if (newMode === 'list') {
    pagination.pageSize = 20;
  } else {
    pagination.pageSize = 12;
  }
  pagination.current = 1;
});

// 组件卸载时清理状态
onBeforeUnmount(() => {
  modalVisible.value = false;
  passwordModalVisible.value = false;
  roleModalVisible.value = false;
  viewModalVisible.value = false;
  currentUser.value = null;
  viewUserData.value = null;
  Object.assign(formData, initFormData());
});

// 初始化
onMounted(() => {
  fetchUserList();
  fetchRoleList();
});
</script>

<style scoped>
/* 基础容器样式 */
.user-management-container {
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

.user-icon {
  background-color: #1890ff;
}

.active-icon {
  background-color: #52c41a;
}

.admin-icon {
  background-color: #faad14;
}

.role-icon {
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

/* 用户网格（卡片视图） */
.user-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.user-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #e8e8e8;
  transition: all 0.3s ease;
  position: relative;
}

.user-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  border-color: #1890ff;
}

.user-card.admin-user {
  border-left: 4px solid #faad14;
}

.user-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.user-avatar-section {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.user-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
  overflow: hidden;
  flex-shrink: 0;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-title {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-size: 16px;
  font-weight: 600;
  color: #262626;
  margin-bottom: 4px;
  line-height: 1.3;
  word-break: break-word;
}

.user-username {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  display: inline-block;
  line-height: 1.4;
}

.user-info {
  margin-bottom: 12px;
  min-height: 60px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #595959;
  margin-bottom: 4px;
  line-height: 1.4;
}

.info-item svg {
  color: #1890ff;
  font-size: 14px;
  flex-shrink: 0;
}

.user-stats {
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

.user-tags {
  display: flex;
  gap: 6px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.admin-tag, .status-tag {
  border-radius: 4px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  line-height: 1.4;
  white-space: nowrap;
}

.user-actions {
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

.password-btn:hover {
  background-color: #fff7e6;
  color: #faad14;
}

.role-btn:hover {
  background-color: #f9f0ff;
  color: #722ed1;
}

.delete-btn:hover {
  background-color: #fff2f0;
  color: #ff4d4f;
}

.user-time {
  text-align: center;
  color: #bfbfbf;
  font-size: 12px;
}

/* 表格容器样式 */
.user-table-container {
  margin-bottom: 24px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #f0f0f0;
}

/* 表格自定义样式 */
.user-table {
  background: white;
}

.user-table :deep(.ant-table) {
  border-radius: 0;
}

.user-table :deep(.ant-table-thead > tr > th) {
  background: #fafafa;
  font-weight: 600;
  color: #262626;
  border-bottom: 2px solid #f0f0f0;
  padding: 16px 12px;
  font-size: 14px;
}

.user-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px 12px;
  border-bottom: 1px solid #f5f5f5;
  vertical-align: middle;
}

.user-table :deep(.ant-table-tbody > tr:hover > td) {
  background: #fafafa;
}

/* 表格单元格样式 */
.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar-table {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 14px;
  overflow: hidden;
  flex-shrink: 0;
}

.user-avatar-table img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-info-table {
  flex: 1;
  min-width: 0;
}

.user-name-text {
  font-weight: 600;
  color: #262626;
  font-size: 15px;
  line-height: 1.4;
  margin-bottom: 2px;
}

.user-username-text {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  background: #f5f5f5;
  padding: 1px 4px;
  border-radius: 3px;
  display: inline-block;
  line-height: 1.3;
}

.contact-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.contact-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #595959;
}

.contact-item svg {
  font-size: 12px;
  color: #1890ff;
}

.status-text {
  font-size: 12px;
  color: #8c8c8c;
  margin-top: 4px;
  text-align: center;
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

.table-action-btn.password-btn:hover {
  background-color: #fff7e6;
  color: #faad14;
}

.table-action-btn.role-btn:hover {
  background-color: #f9f0ff;
  color: #722ed1;
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

/* 角色容器样式 */
.roles-container {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.roles-summary {
  padding: 12px 16px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;
  color: #595959;
}

.roles-list {
  max-height: 300px;
  overflow-y: auto;
}

.role-detail-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-bottom: 1px solid #f5f5f5;
  transition: background-color 0.2s ease;
}

.role-detail-item:hover {
  background: #fafafa;
}

.role-detail-item:last-child {
  border-bottom: none;
}

.role-status-badge {
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

.role-status-badge.active {
  background: #52c41a;
}

.role-status-badge.inactive {
  background: #d9d9d9;
}

.role-detail-info {
  flex: 1;
  min-width: 0;
}

.role-detail-name {
  font-weight: 600;
  color: #262626;
  font-size: 15px;
  margin-bottom: 4px;
  line-height: 1.4;
}

.role-detail-code {
  font-size: 13px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  margin-bottom: 4px;
  word-break: break-all;
}

.role-detail-desc {
  font-size: 12px;
  color: #595959;
  line-height: 1.4;
}

.role-detail-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: #8c8c8c;
  text-align: right;
  flex-shrink: 0;
}

.role-apis, .role-created {
  display: flex;
  align-items: center;
  gap: 4px;
  justify-content: flex-end;
}

/* API 容器样式 */
.apis-container {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.apis-summary {
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

.api-method-badge.get {
  background: #1890ff;
}

.api-method-badge.post {
  background: #52c41a;
}

.api-method-badge.put {
  background: #faad14;
}

.api-method-badge.delete {
  background: #f5222d;
}

.api-method-badge.patch {
  background: #722ed1;
}

.api-method-badge.unknown {
  background: #d9d9d9;
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

/* 空状态 */
.empty-roles, .empty-apis {
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

.timeline-dot.login {
  background: #1890ff;
}

.timeline-dot.update {
  background: #faad14;
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

/* 表单模态框样式 */
.user-modal :deep(.ant-modal-body) {
  max-height: 70vh;
  overflow-y: auto;
  padding: 24px;
}

.modal-content {
  position: relative;
}

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

/* 密码模态框样式 */
.user-info-banner {
  background: #f5f5f5;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar-small {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 18px;
}

.user-info-text {
  flex: 1;
}

.user-name {
  font-weight: 600;
  color: #262626;
  font-size: 16px;
  margin-bottom: 4px;
}

.user-username {
  font-size: 13px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

/* 角色管理模态框样式 */
.role-content {
  padding: 16px 0;
}

.role-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: #262626;
}

.role-stats {
  color: #8c8c8c;
  font-size: 14px;
}

.role-search {
  margin-bottom: 16px;
}

.role-list {
  max-height: 400px;
  overflow-y: auto;
}

.role-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  margin-bottom: 8px;
  transition: all 0.3s ease;
}

.role-item:hover {
  border-color: #d9d9d9;
  background: #fafafa;
}

.role-item.assigned {
  background: #f6ffed;
  border-color: #b7eb8f;
}

.role-status {
  font-size: 11px;
  font-weight: bold;
  padding: 4px 8px;
  border-radius: 4px;
  color: white;
  min-width: 50px;
  text-align: center;
  text-transform: uppercase;
  line-height: 1;
  flex-shrink: 0;
}

.role-status.active {
  background: #52c41a;
}

.role-status.inactive {
  background: #d9d9d9;
}

.role-details {
  flex: 1;
}

.role-name {
  font-weight: 600;
  color: #262626;
  font-size: 14px;
  margin-bottom: 2px;
}

.role-code {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  margin-bottom: 2px;
}

.role-desc {
  font-size: 12px;
  color: #595959;
  line-height: 1.4;
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

/* 通用按钮样式 */
.search-btn, .refresh-btn, .add-btn {
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

.custom-pagination .ant-pagination-item,
.custom-pagination .ant-pagination-prev,
.custom-pagination .ant-pagination-next {
  min-width: 32px;
  height: 32px;
  line-height: 30px;
  font-size: 14px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .user-management-container {
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
  
  .user-grid {
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
  
  .user-avatar-section {
    flex-direction: column;
    text-align: center;
  }
  
  .user-header {
    flex-direction: column;
    gap: 12px;
  }
  
  .user-actions {
    justify-content: space-around;
  }
  
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
  
  .user-name {
    font-size: 14px;
  }
  
  .stat-value {
    font-size: 18px;
  }
  
  .view-modal :deep(.ant-modal-body) {
    padding: 16px;
  }
  
  .method-stats {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
  
  .apis-summary, .roles-summary {
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