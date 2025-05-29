<template>
    <div class="category-management-container">
      <div class="page-header">
        <div class="header-actions">
          <a-button type="primary" @click="handleCreateCategory" class="btn-create">
            <template #icon>
              <PlusOutlined />
            </template>
            创建分类
          </a-button>
          <a-input-search 
            v-model:value="searchQuery" 
            placeholder="搜索分类..." 
            style="width: 250px" 
            @search="handleSearch"
            allow-clear 
          />
          <a-select 
            v-model:value="statusFilter" 
            placeholder="状态" 
            style="width: 120px" 
            @change="handleStatusChange"
          >
            <a-select-option :value="undefined">全部</a-select-option>
            <a-select-option :value="1">启用</a-select-option>
            <a-select-option :value="0">禁用</a-select-option>
          </a-select>
        </div>
      </div>
  
      <div class="stats-row">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-card class="stats-card">
              <a-statistic title="总分类数" :value="stats.total" :value-style="{ color: '#3f8600' }">
                <template #prefix>
                  <FolderOutlined />
                </template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card class="stats-card">
              <a-statistic title="启用分类" :value="stats.enabled" :value-style="{ color: '#52c41a' }">
                <template #prefix>
                  <CheckCircleOutlined />
                </template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card class="stats-card">
              <a-statistic title="禁用分类" :value="stats.disabled" :value-style="{ color: '#cf1322' }">
                <template #prefix>
                  <StopOutlined />
                </template>
              </a-statistic>
            </a-card>
          </a-col>
        </a-row>
      </div>
  
      <div class="table-container">
        <a-card>
          <a-table 
            :data-source="paginatedCategories" 
            :columns="columns" 
            :pagination="false" 
            :loading="loading" 
            row-key="id"
            bordered
            :row-selection="rowSelection"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <div class="category-name-cell">
                  <div class="category-badge" :class="getStatusClass(record.status)"></div>
                  <span v-if="record.icon" class="category-icon">{{ record.icon }}</span>
                  <span class="category-name-text">{{ record.name }}</span>
                </div>
              </template>
  
              <template v-if="column.key === 'parent'">
                <span v-if="record.parent_id && categories.find(c => c.id === record.parent_id)" class="parent-category">
                  {{ categories.find(c => c.id === record.parent_id)?.name }}
                </span>
                <span v-else class="no-parent">根分类</span>
              </template>
  
              <template v-if="column.key === 'description'">
                <span class="description-text">{{ record.description || '无描述' }}</span>
              </template>
  
              <template v-if="column.key === 'sort_order'">
                <a-tag color="blue">{{ record.sort_order }}</a-tag>
              </template>
  
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 1 ? 'green' : 'default'">
                  {{ record.status === 1 ? '启用' : '禁用' }}
                </a-tag>
              </template>
  
              <template v-if="column.key === 'creator'">
                <div class="creator-info">
                  <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name || '') }">
                    {{ getInitials(record.creator_name) }}
                  </a-avatar>
                  <span class="creator-name">{{ record.creator_name }}</span>
                </div>
              </template>
  
              <template v-if="column.key === 'createdAt'">
                <div class="date-info">
                  <span class="date">{{ formatDate(record.created_at) }}</span>
                  <span class="time">{{ formatTime(record.created_at) }}</span>
                </div>
              </template>
  
              <template v-if="column.key === 'action'">
                <div class="action-buttons">
                  <a-button type="primary" size="small" @click="handleViewCategory(record)">
                    查看
                  </a-button>
                  <a-button type="default" size="small" @click="handleEditCategory(record)">
                    编辑
                  </a-button>
                  <a-dropdown>
                    <template #overlay>
                      <a-menu @click="(e: any) => handleMenuClick(e.key, record)">
                        <a-menu-item key="enable" v-if="record.status === 0">启用</a-menu-item>
                        <a-menu-item key="disable" v-if="record.status === 1">禁用</a-menu-item>
                        <a-menu-divider />
                        <a-menu-item key="delete" danger>删除</a-menu-item>
                      </a-menu>
                    </template>
                    <a-button size="small">
                      更多
                      <DownOutlined />
                    </a-button>
                  </a-dropdown>
                </div>
              </template>
            </template>
          </a-table>
  
          <!-- 批量操作 -->
          <div v-if="selectedRowKeys.length > 0" class="batch-actions">
            <a-alert
              :message="`已选择 ${selectedRowKeys.length} 个分类`"
              type="info"
              show-icon
              style="margin-bottom: 16px"
            >
              <template #action>
                <a-space>
                  <a-button size="small" @click="batchEnable">批量启用</a-button>
                  <a-button size="small" @click="batchDisable">批量禁用</a-button>
                  <a-button size="small" @click="clearSelection">取消选择</a-button>
                </a-space>
              </template>
            </a-alert>
          </div>
  
          <div class="pagination-container">
            <a-pagination 
              v-model:current="currentPage" 
              :total="totalItems" 
              :page-size="pageSize"
              :page-size-options="['10', '20', '50', '100']" 
              :show-size-changer="true" 
              @change="handleCurrentChange"
              @showSizeChange="handleSizeChange" 
              :show-total="(total: number) => `共 ${total} 条`" 
            />
          </div>
        </a-card>
      </div>
  
      <!-- 分类创建/编辑对话框 -->
      <a-modal 
        :open="categoryDialogVisible" 
        :title="categoryDialog.isEdit ? '编辑分类' : '创建分类'" 
        width="600px"
        @ok="saveCategory" 
        @cancel="closeCategoryDialog"
        :destroy-on-close="true"
      >
        <a-form ref="formRef" :model="categoryDialog.form" :rules="categoryRules" layout="vertical">
          <a-form-item label="分类名称" name="name">
            <a-input v-model:value="categoryDialog.form.name" placeholder="请输入分类名称" />
          </a-form-item>
  
          <a-form-item label="父分类" name="parent_id">
            <a-tree-select
              v-model:value="categoryDialog.form.parent_id"
              :tree-data="parentCategoryOptions"
              placeholder="请选择父分类（可选）"
              allow-clear
              tree-default-expand-all
              :field-names="{ label: 'name', value: 'id', children: 'children' }"
            />
          </a-form-item>
  
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="图标" name="icon">
                <a-input v-model:value="categoryDialog.form.icon" placeholder="请输入图标（如：📁）" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="排序" name="sort_order">
                <a-input-number 
                  v-model:value="categoryDialog.form.sort_order" 
                  :min="0" 
                  :max="999" 
                  placeholder="排序值"
                  style="width: 100%"
                />
              </a-form-item>
            </a-col>
          </a-row>
  
          <a-form-item label="描述" name="description">
            <a-textarea v-model:value="categoryDialog.form.description" :rows="3" placeholder="请输入分类描述" />
          </a-form-item>
  
          <a-form-item v-if="categoryDialog.isEdit" label="状态" name="status">
            <a-radio-group v-model:value="categoryDialog.form.status">
              <a-radio :value="1">启用</a-radio>
              <a-radio :value="0">禁用</a-radio>
            </a-radio-group>
          </a-form-item>
        </a-form>
      </a-modal>
  
      <!-- 详情对话框 -->
      <a-modal 
        :open="detailDialogVisible" 
        title="分类详情" 
        width="70%" 
        :footer="null" 
        @cancel="closeDetailDialog"
        class="detail-dialog"
      >
        <div v-if="detailDialog.category" class="category-details">
          <div class="detail-header">
            <h2>
              <span v-if="detailDialog.category.icon" class="detail-icon">{{ detailDialog.category.icon }}</span>
              {{ detailDialog.category.name }}
            </h2>
            <a-tag :color="detailDialog.category.status === 1 ? 'green' : 'default'">
              {{ detailDialog.category.status === 1 ? '启用' : '禁用' }}
            </a-tag>
          </div>
  
          <a-descriptions bordered :column="2">
            <a-descriptions-item label="ID">{{ detailDialog.category.id }}</a-descriptions-item>
            <a-descriptions-item label="排序">{{ detailDialog.category.sort_order }}</a-descriptions-item>
            <a-descriptions-item label="创建人">{{ detailDialog.category.creator_name }}</a-descriptions-item>
            <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.category.created_at || '') }}</a-descriptions-item>
            <a-descriptions-item label="更新时间" :span="2">{{ formatFullDateTime(detailDialog.category.updated_at || '') }}</a-descriptions-item>
            <a-descriptions-item label="描述" :span="2">{{ detailDialog.category.description || '无描述' }}</a-descriptions-item>
          </a-descriptions>
  
          <div class="detail-footer">
            <a-button @click="closeDetailDialog">关闭</a-button>
            <a-button type="primary" @click="handleEditCategory(detailDialog.category)">编辑</a-button>
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, reactive, computed, onMounted } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import {
    PlusOutlined,
    FolderOutlined,
    CheckCircleOutlined,
    StopOutlined,
    DownOutlined
  } from '@ant-design/icons-vue';
  import {
    listCategory,
    detailCategory,
    createCategory,
    updateCategory,
    deleteCategory,
    getCategoryTree,
    type Category,
    type CategoryResp,
    type CreateCategoryReq,
    type UpdateCategoryReq,
    type DeleteCategoryReq,
    type ListCategoryReq,
    type TreeCategoryReq
  } from '#/api/core/workorder_category';
  
  // 响应式数据类型
  interface Statistics {
    total: number;
    enabled: number;
    disabled: number;
  }
  
  interface CategoryDialogState {
    isEdit: boolean;
    form: CreateCategoryReq & { id?: number; status?: number };
  }
  
  interface DetailDialogState {
    category: CategoryResp | null;
  }
  
  // 列定义
  const columns = [
    {
      title: '分类名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '父分类',
      dataIndex: 'parent_id',
      key: 'parent',
      width: 150,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: true,
    },
    {
      title: '排序',
      dataIndex: 'sort_order',
      key: 'sort_order',
      width: 100,
      align: 'center' as const,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      align: 'center' as const,
    },
    {
      title: '创建人',
      dataIndex: 'creator_name',
      key: 'creator',
      width: 150,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'createdAt',
      width: 180,
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      align: 'center' as const,
    },
  ];
  
  // 状态数据
  const loading = ref<boolean>(false);
  const searchQuery = ref<string>('');
  const statusFilter = ref<number | undefined>(undefined);
  const currentPage = ref<number>(1);
  const pageSize = ref<number>(10);
  const categories = ref<CategoryResp[]>([]);
  const selectedRowKeys = ref<number[]>([]);
  
  // 模态框控制
  const categoryDialogVisible = ref<boolean>(false);
  const detailDialogVisible = ref<boolean>(false);
  
  // 统计数据
  const stats = reactive<Statistics>({
    total: 0,
    enabled: 0,
    disabled: 0
  });
  
  // 父分类选项
  const parentCategoryOptions = ref<Category[]>([]);
  
  // 行选择配置
  const rowSelection = {
    selectedRowKeys: selectedRowKeys,
    onChange: (keys: number[]) => {
      selectedRowKeys.value = keys;
    },
  };
  
  // 过滤和分页
  const filteredCategories = computed(() => {
    let result = [...categories.value];
  
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase();
      result = result.filter(category =>
        category.name.toLowerCase().includes(query) ||
        (category.description && category.description.toLowerCase().includes(query))
      );
    }
  
    if (statusFilter.value !== undefined) {
      result = result.filter(category => category.status === statusFilter.value);
    }
  
    return result;
  });
  
  const totalItems = computed(() => filteredCategories.value.length);
  
  const paginatedCategories = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    const end = start + pageSize.value;
    return filteredCategories.value.slice(start, end);
  });
  
  // 分类对话框
  const categoryDialog = reactive<CategoryDialogState>({
    isEdit: false,
    form: {
      name: '',
      parent_id: null,
      icon: '',
      sort_order: 0,
      description: ''
    }
  });
  
  // 分类验证规则
  const categoryRules = {
    name: [
      { required: true, message: '请输入分类名称', trigger: 'blur' },
      { min: 2, max: 50, message: '长度应为2到50个字符', trigger: 'blur' }
    ],
    sort_order: [
      { required: true, message: '请输入排序值', trigger: 'blur' },
      { type: 'number', min: 0, max: 999, message: '排序值应在0-999之间', trigger: 'blur' }
    ]
  };
  
  // 详情对话框
  const detailDialog = reactive<DetailDialogState>({
    category: null
  });
  
  // 加载分类列表
  const loadCategories = async (): Promise<void> => {
    loading.value = true;
    try {
      const params: ListCategoryReq = {
        page: 1,
        size: 100, // 获取所有分类用于统计和筛选
        status: statusFilter.value
      };
      const response = await listCategory(params);
      if (response && response.items) {
        categories.value = response.items;
        updateStats();
      }
    } catch (error) {
      console.error('加载分类列表失败:', error);
      message.error('加载分类列表失败');
    } finally {
      loading.value = false;
    }
  };
  
  // 加载父分类选项
  const loadParentCategoryOptions = async (): Promise<void> => {
    try {
      const params: TreeCategoryReq = { status: 1 }; // 只获取启用的分类
      const response = await getCategoryTree(params);
      if (response) {
        parentCategoryOptions.value = response;
      }
    } catch (error) {
      console.error('加载父分类选项失败:', error);
    }
  };
  
  // 更新统计数据
  const updateStats = (): void => {
    stats.total = categories.value.length;
    stats.enabled = categories.value.filter(category => category.status === 1).length;
    stats.disabled = categories.value.filter(category => category.status === 0).length;
  };
  
  // 分页处理
  const handleSizeChange = (current: number, size: number): void => {
    pageSize.value = size;
    currentPage.value = current;
  };
  
  const handleCurrentChange = (page: number): void => {
    currentPage.value = page;
  };
  
  const handleSearch = (): void => {
    currentPage.value = 1;
  };
  
  const handleStatusChange = (): void => {
    currentPage.value = 1;
    loadCategories();
  };
  
  // 分类操作
  const handleCreateCategory = (): void => {
    categoryDialog.isEdit = false;
    categoryDialog.form = {
      name: '',
      parent_id: null,
      icon: '',
      sort_order: 0,
      description: ''
    };
    categoryDialogVisible.value = true;
    loadParentCategoryOptions();
  };
  
  const handleEditCategory = async (row: CategoryResp): Promise<void> => {
    loading.value = true;
    try {
      const response = await detailCategory({ id: row.id });
      if (response) {
        categoryDialog.isEdit = true;
        categoryDialog.form = {
          id: response.id,
          name: response.name,
          parent_id: response.parent_id,
          icon: response.icon,
          sort_order: response.sort_order,
          description: response.description,
          status: response.status
        };
        categoryDialogVisible.value = true;
        detailDialogVisible.value = false;
        loadParentCategoryOptions();
      }
    } catch (error) {
      console.error('加载分类详情失败:', error);
      message.error('加载分类详情失败');
    } finally {
      loading.value = false;
    }
  };
  
  const handleViewCategory = async (row: CategoryResp): Promise<void> => {
    loading.value = true;
    try {
      const response = await detailCategory({ id: row.id });
      if (response) {
        detailDialog.category = response;
        detailDialogVisible.value = true;
      }
    } catch (error) {
      console.error('加载分类详情失败:', error);
      message.error('加载分类详情失败');
    } finally {
      loading.value = false;
    }
  };
  
  const handleMenuClick = (command: string, row: CategoryResp): void => {
    switch (command) {
      case 'enable':
        updateCategoryStatus(row, 1);
        break;
      case 'disable':
        updateCategoryStatus(row, 0);
        break;
      case 'delete':
        confirmDelete(row);
        break;
    }
  };
  
  // 更新分类状态
  const updateCategoryStatus = async (category: CategoryResp, status: number): Promise<void> => {
    try {
      const params: UpdateCategoryReq = {
        id: category.id,
        name: category.name,
        parent_id: category.parent_id,
        icon: category.icon,
        sort_order: category.sort_order,
        description: category.description,
        status: status
      };
      
      await updateCategory(params);
      message.success(`分类 "${category.name}" ${status === 1 ? '已启用' : '已禁用'}`);
      loadCategories();
    } catch (error) {
      console.error('更新分类状态失败:', error);
      message.error('更新分类状态失败');
    }
  };
  
  // 删除分类
  const confirmDelete = (category: CategoryResp): void => {
    Modal.confirm({
      title: '警告',
      content: `确定要删除分类 "${category.name}" 吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      async onOk() {
        try {
          const params: DeleteCategoryReq = { id: category.id };
          await deleteCategory(params);
          message.success(`分类 "${category.name}" 已删除`);
          loadCategories();
        } catch (error) {
          console.error('删除分类失败:', error);
          message.error('删除分类失败');
        }
      }
    });
  };
  
  // 批量操作
  const batchEnable = (): void => {
    batchUpdateStatus(1);
  };
  
  const batchDisable = (): void => {
    batchUpdateStatus(0);
  };
  
  const batchUpdateStatus = async (status: number): Promise<void> => {
    if (selectedRowKeys.value.length === 0) {
      message.warning('请先选择要操作的分类');
      return;
    }
  
    try {
      // 这里需要遍历每个分类进行更新，因为接口不支持批量更新
      const promises = selectedRowKeys.value.map(async (id) => {
        const category = categories.value.find(c => c.id === id);
        if (category) {
          const params: UpdateCategoryReq = {
            id: category.id,
            name: category.name,
            parent_id: category.parent_id,
            icon: category.icon,
            sort_order: category.sort_order,
            description: category.description,
            status: status
          };
          return updateCategory(params);
        }
      });
  
      await Promise.all(promises);
      message.success(`已${status === 1 ? '启用' : '禁用'} ${selectedRowKeys.value.length} 个分类`);
      selectedRowKeys.value = [];
      loadCategories();
    } catch (error) {
      console.error('批量更新状态失败:', error);
      message.error('批量更新状态失败');
    }
  };
  
  const clearSelection = (): void => {
    selectedRowKeys.value = [];
  };
  
  // 保存分类
  const saveCategory = async (): Promise<void> => {
    if (!categoryDialog.form.name.trim()) {
      message.error('分类名称不能为空');
      return;
    }
  
    if (categoryDialog.form.sort_order < 0 || categoryDialog.form.sort_order > 999) {
      message.error('排序值应在0-999之间');
      return;
    }
  
    try {
      if (categoryDialog.isEdit) {
        const params: UpdateCategoryReq = {
          id: categoryDialog.form.id!,
          name: categoryDialog.form.name,
          parent_id: categoryDialog.form.parent_id,
          icon: categoryDialog.form.icon,
          sort_order: categoryDialog.form.sort_order,
          description: categoryDialog.form.description,
          status: categoryDialog.form.status || 1
        };
        await updateCategory(params);
        message.success(`分类 "${categoryDialog.form.name}" 已更新`);
      } else {
        const params: CreateCategoryReq = {
          name: categoryDialog.form.name,
          parent_id: categoryDialog.form.parent_id,
          icon: categoryDialog.form.icon,
          sort_order: categoryDialog.form.sort_order,
          description: categoryDialog.form.description
        };
        await createCategory(params);
        message.success(`分类 "${categoryDialog.form.name}" 已创建`);
      }
      
      categoryDialogVisible.value = false;
      loadCategories();
    } catch (error) {
      console.error('保存分类失败:', error);
      message.error('保存分类失败');
    }
  };
  
  // 对话框控制
  const closeCategoryDialog = (): void => {
    categoryDialogVisible.value = false;
  };
  
  const closeDetailDialog = (): void => {
    detailDialogVisible.value = false;
  };
  
  // 辅助方法
  const formatDate = (dateStr: string): string => {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
  };
  
  const formatTime = (dateStr: string): string => {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  };
  
  const formatFullDateTime = (dateStr: string): string => {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    return d.toLocaleString('zh-CN', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };
  
  const getInitials = (name: string): string => {
    if (!name) return '';
    return name
      .split('')
      .slice(0, 2)
      .join('')
      .toUpperCase();
  };
  
  const getStatusClass = (status: number): string => {
    return status === 1 ? 'status-enabled' : 'status-disabled';
  };
  
  const getAvatarColor = (name: string): string => {
    const colors = [
      '#1890ff', '#52c41a', '#faad14', '#f5222d',
      '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
    ];
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
  
    return colors[Math.abs(hash) % colors.length]!;
  };
  
  // 初始化
  onMounted(() => {
    loadCategories();
  });
  </script>
  
  <style scoped>
  .category-management-container {
    padding: 24px;
    min-height: 100vh;
  }
  
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }
  
  .header-actions {
    display: flex;
    gap: 12px;
  }
  
  .btn-create {
    background: linear-gradient(135deg, #1890ff 0%);
    border: none;
  }
  
  .stats-row {
    margin-bottom: 24px;
  }
  
  .stats-card {
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
    height: 100%;
  }
  
  .table-container {
    margin-bottom: 24px;
  }
  
  .category-name-cell {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  
  .category-badge {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  
  .status-enabled {
    background-color: #52c41a;
  }
  
  .status-disabled {
    background-color: #d9d9d9;
  }
  
  .category-icon {
    font-size: 16px;
  }
  
  .category-name-text {
    font-weight: 500;
  }
  
  .parent-category {
    color: #1890ff;
    font-weight: 500;
  }
  
  .no-parent {
    color: #8c8c8c;
    font-style: italic;
  }
  
  .description-text {
    color: #606266;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .creator-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .creator-name {
    font-size: 14px;
  }
  
  .date-info {
    display: flex;
    flex-direction: column;
  }
  
  .date {
    font-weight: 500;
    font-size: 14px;
  }
  
  .time {
    font-size: 12px;
    color: #8c8c8c;
  }
  
  .action-buttons {
    display: flex;
    gap: 8px;
    justify-content: center;
  }
  
  .batch-actions {
    margin-bottom: 16px;
  }
  
  .pagination-container {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
  }
  
  .detail-dialog .category-details {
    margin-bottom: 20px;
  }
  
  .detail-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 20px;
  }
  
  .detail-header h2 {
    margin: 0;
    font-size: 24px;
    color: #1f2937;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .detail-icon {
    font-size: 24px;
  }
  
  .detail-footer {
    margin-top: 24px;
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
  </style>
  