<template>
  <div class="template-manager-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateTemplate" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新模板
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索模板..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="categoryFilter" placeholder="分类" style="width: 120px" @change="handleCategoryChange">
          <a-select-option :value="null">全部分类</a-select-option>
          <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
            {{ cat.name }}
          </a-select-option>
        </a-select>
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部状态</a-select-option>
          <a-select-option :value="1">启用</a-select-option>
          <a-select-option :value="0">禁用</a-select-option>
        </a-select>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="模板总数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FileOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="常规模板" :value="stats.regular" :value-style="{ color: '#1890ff' }">
              <template #prefix>
                <FileTextOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="系统模板" :value="stats.system" :value-style="{ color: '#722ed1' }">
              <template #prefix>
                <SettingOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="近7天新增" :value="stats.recentAdded" :value-style="{ color: '#fa8c16' }">
              <template #prefix>
                <PlusCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table :data-source="paginatedTemplates" :columns="columns" :pagination="false" :loading="loading"
          row-key="id" bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="template-name-cell">
                <div class="template-badge" :class="record.status ? 'status-enabled' : 'status-disabled'"></div>
                <span class="template-name-text">{{ record.name }}</span>
                <a-tag v-if="record.isSystem" color="purple" size="small">系统</a-tag>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <span class="description-text">{{ record.description || '无描述' }}</span>
            </template>

            <template v-if="column.key === 'category'">
              <a-tag :color="getCategoryColor(record.categoryId)">{{ getCategoryName(record.categoryId) }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-switch :checked="record.status === 1" disabled />
            </template>

            <template v-if="column.key === 'creator'">
              <div class="creator-info">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creatorName) }">
                  {{ getInitials(record.creatorName) }}
                </a-avatar>
                <span class="creator-name">{{ record.creatorName }}</span>
              </div>
            </template>

            <template v-if="column.key === 'createdAt'">
              <div class="date-info">
                <span class="date">{{ formatDate(record.createdAt) }}</span>
                <span class="time">{{ formatTime(record.createdAt) }}</span>
              </div>
            </template>

            <template v-if="column.key === 'action'">
              <div class="action-buttons">
                <a-button type="primary" size="small" @click="handlePreviewTemplate(record)">
                  预览
                </a-button>
                <a-button type="default" size="small" @click="handleEditTemplate(record)" :disabled="record.isSystem">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="handleCommand(column.key, record)">
                      <a-menu-item key="enable" v-if="record.status === 0">启用</a-menu-item>
                      <a-menu-item key="disable" v-if="record.status === 1">禁用</a-menu-item>
                      <a-menu-item key="clone">克隆</a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="delete" danger :disabled="record.isSystem">删除</a-menu-item>
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

        <div class="pagination-container">
          <a-pagination v-model:current="currentPage" :total="totalItems" :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" :show-size-changer="true" @change="handlePageChange"
            @show-size-change="handleSizeChange" :show-total="(total: number) => `共 ${total} 条`" />
        </div>
      </a-card>
    </div>

    <!-- 模板创建/编辑对话框 -->
    <a-modal v-model:visible="templateDialog.visible" :title="templateDialog.isEdit ? '编辑模板' : '创建模板'" width="760px"
      @ok="saveTemplate" :destroy-on-close="true">
      <a-form ref="formRef" :model="templateDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="模板名称" name="name">
          <a-input v-model:value="templateDialog.form.name" placeholder="请输入模板名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="templateDialog.form.description" :rows="3" placeholder="请输入模板描述" />
        </a-form-item>

        <a-form-item label="分类" name="categoryId">
          <a-select v-model:value="templateDialog.form.categoryId" placeholder="请选择分类" style="width: 100%">
            <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="templateDialog.form.status">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="0">禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider orientation="left">模板内容</a-divider>

        <div class="template-editor">
          <a-tabs v-model:activeKey="templateDialog.activeTab">
            <a-tab-pane key="html" tab="HTML">
              <a-form-item>
                <a-textarea v-model:value="templateDialog.form.content.html" :rows="12" class="code-editor"
                  placeholder="<div>您的HTML内容</div>" />
              </a-form-item>
            </a-tab-pane>
            <a-tab-pane key="css" tab="CSS">
              <a-form-item>
                <a-textarea v-model:value="templateDialog.form.content.css" :rows="12" class="code-editor"
                  placeholder=".example { color: blue; }" />
              </a-form-item>
            </a-tab-pane>
            <a-tab-pane key="js" tab="JavaScript">
              <a-form-item>
                <a-textarea v-model:value="templateDialog.form.content.js" :rows="12" class="code-editor"
                  placeholder="function example() { return true; }" />
              </a-form-item>
            </a-tab-pane>
          </a-tabs>
        </div>

        <a-form-item label="变量列表">
          <div class="variables-list">
            <a-row v-for="(variable, index) in templateDialog.form.variables" :key="index" :gutter="16"
              style="margin-bottom: 16px;">
              <a-col :span="10">
                <a-input v-model:value="variable.name" placeholder="变量名" addon-before="{{" addon-after="}}" />
              </a-col>
              <a-col :span="10">
                <a-input v-model:value="variable.description" placeholder="变量描述" />
              </a-col>
              <a-col :span="4">
                <a-button type="text" danger @click="removeVariable(index)">
                  <DeleteOutlined />
                </a-button>
              </a-col>
            </a-row>
            <a-button type="dashed" block @click="addVariable" style="margin-top: 16px">
              <PlusOutlined /> 添加变量
            </a-button>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆模板" @ok="confirmClone" :destroy-on-close="true">
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新模板名称" name="name">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新模板名称" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="cloneDialog.form.description" :rows="3" placeholder="请输入模板描述" />
        </a-form-item>
        <a-form-item label="分类" name="categoryId">
          <a-select v-model:value="cloneDialog.form.categoryId" placeholder="请选择分类" style="width: 100%">
            <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 预览对话框 -->
    <a-modal v-model:visible="previewDialog.visible" title="模板预览" width="80%" footer={null} class="preview-dialog">
      <div v-if="previewDialog.template" class="template-details">
        <div class="detail-header">
          <h2>{{ previewDialog.template.name }}</h2>
          <a-tag :color="previewDialog.template.status ? 'green' : 'default'">
            {{ previewDialog.template.status ? '启用' : '禁用' }}
          </a-tag>
          <a-tag v-if="previewDialog.template.isSystem" color="purple">系统</a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ previewDialog.template.id }}</a-descriptions-item>
          <a-descriptions-item label="分类">{{ getCategoryName(previewDialog.template.categoryId) }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ previewDialog.template.creatorName }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(previewDialog.template.createdAt)
          }}</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">{{ previewDialog.template.description || '无描述'
          }}</a-descriptions-item>
        </a-descriptions>

        <div class="template-content-preview">
          <a-tabs>
            <a-tab-pane key="rendered" tab="渲染效果">
              <div class="preview-rendered">
                <div class="preview-frame" v-html="sanitizedHtml(previewDialog.template)"></div>
              </div>
            </a-tab-pane>
            <a-tab-pane key="code" tab="代码">
              <a-tabs>
                <a-tab-pane key="html" tab="HTML">
                  <div class="code-block">
                    <pre>{{ previewDialog.template.content.html }}</pre>
                  </div>
                </a-tab-pane>
                <a-tab-pane key="css" tab="CSS">
                  <div class="code-block">
                    <pre>{{ previewDialog.template.content.css }}</pre>
                  </div>
                </a-tab-pane>
                <a-tab-pane key="js" tab="JavaScript">
                  <div class="code-block">
                    <pre>{{ previewDialog.template.content.js }}</pre>
                  </div>
                </a-tab-pane>
              </a-tabs>
            </a-tab-pane>
            <a-tab-pane key="variables" tab="变量">
              <a-table :data-source="previewDialog.template.variables" :columns="variableColumns" :pagination="false"
                size="small" bordered>
              </a-table>
            </a-tab-pane>
          </a-tabs>
        </div>

        <div class="detail-footer">
          <a-button @click="previewDialog.visible = false">关闭</a-button>
          <a-button type="primary" @click="handleEditTemplate(previewDialog.template)"
            :disabled="previewDialog.template.isSystem">编辑</a-button>
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
  FileOutlined,
  FileTextOutlined,
  SettingOutlined,
  PlusCircleOutlined,
  DeleteOutlined,
  DownOutlined
} from '@ant-design/icons-vue';

// 类型定义
interface Variable {
  name: string;
  description: string;
}

interface TemplateContent {
  html: string;
  css: string;
  js: string;
}

interface Template {
  id: number;
  name: string;
  description: string;
  categoryId: number;
  content: TemplateContent;
  variables: Variable[];
  status: number; // 0-禁用，1-启用
  isSystem: boolean;
  creatorId: number;
  creatorName: string;
  createdAt: Date;
  updatedAt: Date;
}

interface Category {
  id: number;
  name: string;
  color: string;
}

// 列定义
const columns = [
  {
    title: '模板名称',
    dataIndex: 'name',
    key: 'name',
    width: 180,
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    width: 200,
    ellipsis: true,
  },
  {
    title: '分类',
    dataIndex: 'categoryId',
    key: 'category',
    width: 120,
    align: 'center',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    align: 'center',
  },
  {
    title: '创建人',
    dataIndex: 'creatorName',
    key: 'creator',
    width: 150,
  },
  {
    title: '创建时间',
    dataIndex: 'createdAt',
    key: 'createdAt',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    width: 200,
    align: 'center',
  },
];

// 变量列定义
const variableColumns = [
  {
    title: '变量名',
    dataIndex: 'name',
    key: 'name',
    width: 150,
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
  }
];

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const categoryFilter = ref(null);
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);

// 统计数据
const stats = reactive({
  total: 24,
  regular: 18,
  system: 6,
  recentAdded: 3
});

// 分类数据
const categories = ref<Category[]>([
  { id: 1, name: '邮件模板', color: '#1890ff' },
  { id: 2, name: '通知模板', color: '#52c41a' },
  { id: 3, name: '报表模板', color: '#722ed1' },
  { id: 4, name: '文档模板', color: '#fa8c16' },
  { id: 5, name: '销售模板', color: '#eb2f96' }
]);

// 模拟模板数据
const templates = ref<Template[]>([
  {
    id: 1,
    name: '欢迎邮件模板',
    description: '新用户注册后发送的欢迎邮件',
    categoryId: 1,
    content: {
      html: '<div>\n  <h1>欢迎加入我们！</h1>\n  <p>尊敬的 {{userName}}，</p>\n  <p>感谢您注册我们的服务。您的账号已经创建成功。</p>\n  <p>如有任何问题，请随时联系我们。</p>\n  <p>祝您使用愉快！</p>\n</div>',
      css: 'h1 {\n  color: #1890ff;\n}\np {\n  line-height: 1.6;\n}',
      js: ''
    },
    variables: [
      { name: 'userName', description: '用户姓名' }
    ],
    status: 1,
    isSystem: true,
    creatorId: 101,
    creatorName: '系统管理员',
    createdAt: new Date('2025-01-10T09:00:00'),
    updatedAt: new Date('2025-01-10T09:00:00')
  },
  {
    id: 2,
    name: '订单确认模板',
    description: '用户下单后的订单确认邮件',
    categoryId: 1,
    content: {
      html: '<div>\n  <h1>订单确认</h1>\n  <p>尊敬的 {{userName}}，</p>\n  <p>感谢您的订购。您的订单号为：{{orderNumber}}。</p>\n  <p>订单金额：{{amount}} 元</p>\n  <p>订单日期：{{orderDate}}</p>\n  <p>我们将尽快处理您的订单。</p>\n</div>',
      css: 'h1 {\n  color: #52c41a;\n}\np {\n  line-height: 1.6;\n}',
      js: ''
    },
    variables: [
      { name: 'userName', description: '用户姓名' },
      { name: 'orderNumber', description: '订单号' },
      { name: 'amount', description: '订单金额' },
      { name: 'orderDate', description: '订单日期' }
    ],
    status: 1,
    isSystem: false,
    creatorId: 102,
    creatorName: '张三',
    createdAt: new Date('2025-01-15T14:30:00'),
    updatedAt: new Date('2025-02-01T11:20:00')
  },
  {
    id: 3,
    name: '密码重置通知',
    description: '用户重置密码后的通知邮件',
    categoryId: 2,
    content: {
      html: '<div>\n  <h2>密码已成功重置</h2>\n  <p>尊敬的 {{userName}}，</p>\n  <p>您的密码已于 {{resetTime}} 重置成功。</p>\n  <p>如果这不是您本人的操作，请立即联系客服。</p>\n</div>',
      css: 'h2 {\n  color: #fa8c16;\n}\np {\n  line-height: 1.5;\n}',
      js: ''
    },
    variables: [
      { name: 'userName', description: '用户姓名' },
      { name: 'resetTime', description: '重置时间' }
    ],
    status: 1,
    isSystem: true,
    creatorId: 101,
    creatorName: '系统管理员',
    createdAt: new Date('2025-01-20T10:15:00'),
    updatedAt: new Date('2025-01-20T10:15:00')
  },
  {
    id: 4,
    name: '月度报表模板',
    description: '系统生成的月度报表模板',
    categoryId: 3,
    content: {
      html: '<div>\n  <h2>{{month}}月度报表</h2>\n  <p>报表生成时间：{{generatedTime}}</p>\n  <table border="1" style="width:100%">\n    <tr>\n      <th>项目</th>\n      <th>数量</th>\n      <th>金额</th>\n    </tr>\n    <tr>\n      <td>{{item1}}</td>\n      <td>{{quantity1}}</td>\n      <td>{{amount1}}</td>\n    </tr>\n    <tr>\n      <td>{{item2}}</td>\n      <td>{{quantity2}}</td>\n      <td>{{amount2}}</td>\n    </tr>\n  </table>\n  <p>总计：{{total}} 元</p>\n</div>',
      css: 'table {\n  border-collapse: collapse;\n  margin: 15px 0;\n}\nth, td {\n  padding: 8px;\n  text-align: center;\n}\nth {\n  background-color: #f0f0f0;\n}',
      js: ''
    },
    variables: [
      { name: 'month', description: '月份' },
      { name: 'generatedTime', description: '生成时间' },
      { name: 'item1', description: '项目1名称' },
      { name: 'quantity1', description: '项目1数量' },
      { name: 'amount1', description: '项目1金额' },
      { name: 'item2', description: '项目2名称' },
      { name: 'quantity2', description: '项目2数量' },
      { name: 'amount2', description: '项目2金额' },
      { name: 'total', description: '总计金额' }
    ],
    status: 1,
    isSystem: false,
    creatorId: 103,
    creatorName: '李四',
    createdAt: new Date('2025-02-10T16:45:00'),
    updatedAt: new Date('2025-03-01T09:30:00')
  },
  {
    id: 5,
    name: '服务协议模板',
    description: '用户服务协议文档模板',
    categoryId: 4,
    content: {
      html: '<div>\n  <h1>用户服务协议</h1>\n  <h2>1. 总则</h2>\n  <p>欢迎使用{{companyName}}提供的服务。</p>\n  <h2>2. 服务说明</h2>\n  <p>{{companyName}}提供的服务包括但不限于{{serviceScope}}。</p>\n  <h2>3. 用户权利与义务</h2>\n  <p>用户有权根据本协议约定使用{{companyName}}提供的服务。</p>\n  <p>最后更新时间：{{updateTime}}</p>\n</div>',
      css: 'h1 {\n  text-align: center;\n}\nh2 {\n  margin-top: 20px;\n  color: #333;\n}\np {\n  line-height: 1.6;\n  text-indent: 2em;\n}',
      js: ''
    },
    variables: [
      { name: 'companyName', description: '公司名称' },
      { name: 'serviceScope', description: '服务范围' },
      { name: 'updateTime', description: '更新时间' }
    ],
    status: 1,
    isSystem: false,
    creatorId: 104,
    creatorName: '王五',
    createdAt: new Date('2025-02-20T13:40:00'),
    updatedAt: new Date('2025-02-20T13:40:00')
  },
  {
    id: 6,
    name: '促销活动通知',
    description: '向用户发送促销活动信息',
    categoryId: 5,
    content: {
      html: '<div>\n  <h1 style="color: #ff4d4f;">限时特惠活动</h1>\n  <p>尊敬的 {{userName}}，</p>\n  <p>我们正在举办{{activityName}}活动，活动时间为{{startDate}}至{{endDate}}。</p>\n  <p>活动期间，{{discountDesc}}</p>\n  <p>点击了解详情：<a href="{{activityUrl}}">查看详情</a></p>\n</div>',
      css: 'h1 {\n  text-align: center;\n  margin-bottom: 20px;\n}\np {\n  line-height: 1.8;\n}\na {\n  color: #1890ff;\n  text-decoration: none;\n}\na:hover {\n  text-decoration: underline;\n}',
      js: ''
    },
    variables: [
      { name: 'userName', description: '用户姓名' },
      { name: 'activityName', description: '活动名称' },
      { name: 'startDate', description: '开始日期' },
      { name: 'endDate', description: '结束日期' },
      { name: 'discountDesc', description: '折扣描述' },
      { name: 'activityUrl', description: '活动链接' }
    ],
    status: 0,
    isSystem: false,
    creatorId: 105,
    creatorName: '赵六',
    createdAt: new Date('2025-03-15T10:00:00'),
    updatedAt: new Date('2025-03-15T10:00:00')
  }
]);

// 模板对话框
const templateDialog = reactive({
  visible: false,
  isEdit: false,
  activeTab: 'html',
  form: {
    id: 0,
    name: '',
    description: '',
    categoryId: null as number | null,
    content: {
      html: '',
      css: '',
      js: ''
    },
    variables: [] as Variable[],
    status: 1,
    isSystem: false,
    creatorId: 102, // 模拟当前用户ID
    creatorName: '当前用户', // 模拟当前用户名
    createdAt: new Date(),
    updatedAt: new Date()
  }
});

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  form: {
    name: '',
    description: '',
    categoryId: null as number | null,
    originalId: 0
  }
});

// 预览对话框
const previewDialog = reactive({
  visible: false,
  template: {} as Template
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  categoryId: [
    { required: true, message: '请选择分类', trigger: 'change' }
  ]
};

// 计算过滤后的模板
const filteredTemplates = computed(() => {
  let result = [...templates.value];

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(template =>
      template.name.toLowerCase().includes(query) ||
      (template.description && template.description.toLowerCase().includes(query))
    );
  }

  if (categoryFilter.value !== null) {
    result = result.filter(template => template.categoryId === categoryFilter.value);
  }

  if (statusFilter.value !== null) {
    result = result.filter(template => template.status === statusFilter.value);
  }

  return result;
});

const totalItems = computed(() => filteredTemplates.value.length);

const paginatedTemplates = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredTemplates.value.slice(start, end);
});

// 方法
const handleSizeChange = (current: number, size: number) => {
  pageSize.value = size;
  currentPage.value = 1;
};

const handlePageChange = (page: number) => {
  currentPage.value = page;
};

const handleSearch = () => {
  currentPage.value = 1;
};

const handleCategoryChange = () => {
  currentPage.value = 1;
};

const handleStatusChange = () => {
  currentPage.value = 1;
};

const handleCreateTemplate = () => {
  templateDialog.isEdit = false;
  templateDialog.form = {
    id: 0,
    name: '',
    description: '',
    categoryId: null,
    content: {
      html: '',
      css: '',
      js: ''
    },
    variables: [],
    status: 1,
    isSystem: false,
    creatorId: 102,
    creatorName: '当前用户',
    createdAt: new Date(),
    updatedAt: new Date()
  };
  templateDialog.activeTab = 'html';
  templateDialog.visible = true;
};

const handleEditTemplate = (row: Template) => {
  if (row.isSystem) {
    message.warning('系统模板不可编辑');
    return;
  }

  templateDialog.isEdit = true;
  templateDialog.form = JSON.parse(JSON.stringify(row));
  templateDialog.activeTab = 'html';
  templateDialog.visible = true;
  previewDialog.visible = false;
};

const handlePreviewTemplate = (row: Template) => {
  previewDialog.template = row;
  previewDialog.visible = true;
};

const handleCommand = (command: string, row: Template) => {
  switch (command) {
    case 'enable':
      enableTemplate(row);
      break;
    case 'disable':
      disableTemplate(row);
      break;
    case 'clone':
      showCloneDialog(row);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

const enableTemplate = (template: Template) => {
  const index = templates.value.findIndex(t => t.id === template.id);
  if (index !== -1) {
    const t = templates.value[index];
    if (t) {
      t.status = 1;
      t.updatedAt = new Date();
      message.success(`模板 "${template.name}" 已启用`);
    }
  }
};

const disableTemplate = (template: Template) => {
  const index = templates.value.findIndex(t => t.id === template.id);
  if (index !== -1) {
    const t = templates.value[index];
    if (t) {
      t.status = 0;
      t.updatedAt = new Date();
      message.success(`模板 "${template.name}" 已禁用`);
    }
  }
};

const showCloneDialog = (template: Template) => {
  cloneDialog.form.name = `${template.name} 副本`;
  cloneDialog.form.description = template.description;
  cloneDialog.form.categoryId = template.categoryId;
  cloneDialog.form.originalId = template.id;
  cloneDialog.visible = true;
};

const confirmClone = () => {
  const originalTemplate = templates.value.find(t => t.id === cloneDialog.form.originalId);
  if (originalTemplate) {
    const newId = Math.max(...templates.value.map(t => t.id)) + 1;
    const clonedTemplate: Template = {
      ...JSON.parse(JSON.stringify(originalTemplate)),
      id: newId,
      name: cloneDialog.form.name,
      description: cloneDialog.form.description,
      categoryId: cloneDialog.form.categoryId || originalTemplate.categoryId,
      isSystem: false,
      status: 1,
      creatorId: 102,
      creatorName: '当前用户',
      createdAt: new Date(),
      updatedAt: new Date()
    };

    templates.value.push(clonedTemplate);
    cloneDialog.visible = false;
    message.success(`模板 "${originalTemplate.name}" 已克隆为 "${cloneDialog.form.name}"`);
  }
};

const confirmDelete = (template: Template) => {
  if (template.isSystem) {
    message.warning('系统模板不可删除');
    return;
  }

  Modal.confirm({
    title: '警告',
    content: `确定要删除模板 "${template.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      const index = templates.value.findIndex(t => t.id === template.id);
      if (index !== -1) {
        templates.value.splice(index, 1);
        message.success(`模板 "${template.name}" 已删除`);
      }
    }
  });
};

const addVariable = () => {
  templateDialog.form.variables.push({
    name: '',
    description: ''
  });
};

const removeVariable = (index: number) => {
  templateDialog.form.variables.splice(index, 1);
};

const saveTemplate = () => {
  if (templateDialog.form.name.trim() === '') {
    message.error('模板名称不能为空');
    return;
  }

  if (templateDialog.form.categoryId === null) {
    message.error('请选择分类');
    return;
  }

  if (templateDialog.isEdit) {
    // 更新现有模板
    const index = templates.value.findIndex(t => t.id === templateDialog.form.id);
    if (index !== -1) {
      templateDialog.form.updatedAt = new Date();
      templates.value[index] = { ...templateDialog.form } as Template;
      message.success(`模板 "${templateDialog.form.name}" 已更新`);
    }
  } else {
    // 创建新模板
    const newId = Math.max(...templates.value.map(t => t.id)) + 1;
    templateDialog.form.id = newId;
    templates.value.push({ ...templateDialog.form } as Template);
    message.success(`模板 "${templateDialog.form.name}" 已创建`);
  }
  templateDialog.visible = false;
};

// 辅助方法
const formatDate = (date: Date) => {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (date: Date) => {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (date: Date) => {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const getInitials = (name: string) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getAvatarColor = (name: string) => {
  // 根据名称生成一致的颜色
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];

  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length];
};

const getCategoryName = (categoryId: number) => {
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.name : '未分类';
};

const getCategoryColor = (categoryId: number) => {
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.color : '';
};

const sanitizedHtml = (template: Template) => {
  if (!template) return '';

  // 组合HTML、CSS和JavaScript
  let html = template.content.html || '';
  const css = template.content.css || '';
  const js = template.content.js || '';

  // 添加样式和脚本
  let result = html;
  if (css) {
    result = `<style>${css}</style>${result}`;
  }
  if (js) {
    result = `${result}<script>${js}<\/script>`;
  }

  // 这里可以添加安全处理逻辑，如XSS过滤等
  // 简单示例：移除可能的script标签（实际应使用专业库如DOMPurify）
  result = result.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');

  return result;
};

// 初始化
onMounted(() => {
  loading.value = true;
  // 模拟API加载
  setTimeout(() => {
    loading.value = false;
  }, 800);
});
</script>

<style scoped>
.template-manager-container {
  padding: 24px;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  color: #1f2937;
  margin: 0;
  background: linear-gradient(90deg, #1890ff 0%, #52c41a 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  font-weight: 700;
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

.template-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.template-badge {
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

.template-name-text {
  font-weight: 500;
}

.description-text {
  color: #606266;
  display: -webkit-box;
  -webkit-line-clamp: 2;
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

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.template-editor {
  margin-bottom: 20px;
}

.code-editor {
  font-family: 'Courier New', monospace;
}

.variables-list {
  margin-top: 10px;
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
}

.template-content-preview {
  margin-top: 24px;
}

.preview-rendered {
  padding: 20px;
  border-radius: 4px;
  min-height: 300px;
  overflow: auto;
}

.preview-frame {
  width: 100%;
  min-height: 300px;
}

.code-block {
  padding: 16px;
  border-radius: 4px;
  overflow: auto;
  max-height: 400px;
}

.code-block pre {
  margin: 0;
  font-family: 'Courier New', monospace;
  white-space: pre-wrap;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
