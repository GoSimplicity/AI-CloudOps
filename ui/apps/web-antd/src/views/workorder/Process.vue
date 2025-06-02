<template>
  <div class="process-container">
    <!-- 优化头部布局 -->
    <div class="page-header">
      <div class="header-left">
        <a-button type="primary" @click="handleCreateProcess" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新流程
        </a-button>
      </div>
      
      <div class="header-right">
        <a-input-search 
          v-model:value="searchQuery" 
          placeholder="搜索流程..." 
          style="width: 280px" 
          @search="handleSearch"
          allow-clear 
        />
        <a-select 
          v-model:value="statusFilter" 
          placeholder="全部" 
          style="width: 100px" 
          @change="handleStatusChange"
        >
          <a-select-option :value="null">全部</a-select-option>
          <a-select-option :value="0">草稿</a-select-option>
          <a-select-option :value="1">已发布</a-select-option>
          <a-select-option :value="2">已禁用</a-select-option>
        </a-select>
        <a-select 
          v-model:value="categoryFilter" 
          placeholder="全部分类" 
          style="width: 120px" 
          @change="handleCategoryChange"
        >
          <a-select-option :value="null">全部分类</a-select-option>
          <a-select-option v-for="category in categories" :key="category.id" :value="category.id">
            {{ category.name }}
          </a-select-option>
        </a-select>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-section">
      <a-row :gutter="16">
        <a-col :xs="24" :sm="12" :md="6">
          <a-card class="stats-card">
            <a-statistic title="总流程数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <ApartmentOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :xs="24" :sm="12" :md="6">
          <a-card class="stats-card">
            <a-statistic title="已发布" :value="stats.published" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :xs="24" :sm="12" :md="6">
          <a-card class="stats-card">
            <a-statistic title="草稿" :value="stats.draft" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <EditOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :xs="24" :sm="12" :md="6">
          <a-card class="stats-card">
            <a-statistic title="已禁用" :value="stats.disabled" :value-style="{ color: '#cf1322' }">
              <template #prefix>
                <StopOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <!-- 表格区域 -->
    <div class="table-section">
      <a-card :bordered="false" class="table-card">
        <a-table 
          :data-source="processList" 
          :columns="columns" 
          :pagination="false" 
          :loading="loading"
          row-key="id" 
          :scroll="{ x: 1200 }"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="process-name-cell">
                <div class="process-badge" :class="getStatusClass(record.status)"></div>
                <span class="process-name-text">{{ record.name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <div class="description-text" :title="record.description">
                {{ record.description || '无描述' }}
              </div>
            </template>

            <template v-if="column.key === 'version'">
              <a-tag color="blue">v{{ record.version }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 0 ? 'orange' : record.status === 1 ? 'green' : 'default'">
                {{ record.status === 0 ? '草稿' : record.status === 1 ? '已发布' : '已禁用' }}
              </a-tag>
            </template>

            <template v-if="column.key === 'form_design'">
              <span>{{ getFormName(record.form_design_id) }}</span>
            </template>

            <template v-if="column.key === 'category'">
              <span>{{ getCategoryName(record.category_id) }}</span>
            </template>

            <template v-if="column.key === 'creator'">
              <div class="creator-info">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name) }">
                  {{ getInitials(record.creator_name) }}
                </a-avatar>
                <span class="creator-name">{{ record.creator_name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'createdAt'">
              <div class="date-info">
                <div class="date">{{ formatDate(record.created_at) }}</div>
                <div class="time">{{ formatTime(record.created_at) }}</div>
              </div>
            </template>

            <template v-if="column.key === 'action'">
              <div class="action-buttons">
                <a-button type="primary" size="small" @click="handleViewProcess(record)">
                  查看
                </a-button>
                <a-button size="small" @click="handleEditProcess(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleCommand(e.key, record)">
                      <a-menu-item key="publish" v-if="record.status === 0">发布</a-menu-item>
                      <a-menu-item key="unpublish" v-if="record.status === 1">取消发布</a-menu-item>
                      <a-menu-item key="validate">验证流程</a-menu-item>
                      <a-menu-item key="clone">克隆</a-menu-item>
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

        <!-- 分页 -->
        <div class="pagination-wrapper">
          <a-pagination 
            v-model:current="currentPage" 
            :total="total" 
            :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" 
            :show-size-changer="true" 
            @change="handleCurrentChange" 
            @showSizeChange="handleSizeChange" 
            :show-total="(total: number) => `共 ${total} 条`"
            show-quick-jumper
          />
        </div>
      </a-card>
    </div>

    <!-- 流程创建/编辑对话框 -->
    <a-modal v-model:visible="processDialog.visible" :title="processDialog.isEdit ? '编辑流程' : '创建流程'" width="900px"
      @ok="saveProcess" :destroy-on-close="true">
      <a-form ref="formRef" :model="processDialog.form" :rules="formRules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="流程名称" name="name">
              <a-input v-model:value="processDialog.form.name" placeholder="请输入流程名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="关联表单" name="form_design_id">
              <a-select v-model:value="processDialog.form.form_design_id" placeholder="请选择关联表单" style="width: 100%">
                <a-select-option v-for="form in forms" :key="form.id" :value="form.id">
                  {{ form.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="分类" name="category_id">
              <a-select v-model:value="processDialog.form.category_id" placeholder="请选择分类" style="width: 100%" allow-clear>
                <a-select-option v-for="category in categories" :key="category.id" :value="category.id">
                  {{ category.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" name="status" v-if="processDialog.isEdit">
              <a-radio-group v-model:value="processDialog.form.status">
                <a-radio :value="0">草稿</a-radio>
                <a-radio :value="1">已发布</a-radio>
                <a-radio :value="2">已禁用</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="processDialog.form.description" :rows="3" placeholder="请输入流程描述" />
        </a-form-item>

        <a-divider orientation="left">流程步骤定义</a-divider>

        <div class="steps-editor">
          <div class="step-list">
            <a-collapse v-model:activeKey="activeStepKeys">
              <a-collapse-panel v-for="(step, index) in processDialog.form.definition.steps" :key="index"
                :header="step.name || `步骤 ${index + 1}`">
                <template #extra>
                  <a-button type="text" danger @click.stop="removeStep(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </template>

                <a-row :gutter="16">
                  <a-col :span="12">
                    <a-form-item label="步骤名称">
                      <a-input v-model:value="step.name" placeholder="步骤名称" />
                    </a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item label="步骤类型">
                      <a-select v-model:value="step.type" placeholder="选择步骤类型">
                        <a-select-option value="start">开始</a-select-option>
                        <a-select-option value="approval">审批</a-select-option>
                        <a-select-option value="condition">条件</a-select-option>
                        <a-select-option value="notification">通知</a-select-option>
                        <a-select-option value="end">结束</a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                </a-row>

                <a-row :gutter="16">
                  <a-col :span="12">
                    <a-form-item label="角色">
                      <a-select v-model:value="step.roles" mode="multiple" placeholder="选择角色">
                        <a-select-option v-for="role in roles" :key="role" :value="role">
                          {{ role }}
                        </a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item label="用户ID">
                      <a-select v-model:value="step.users" mode="multiple" placeholder="选择用户">
                        <a-select-option v-for="user in users" :key="user.id" :value="user.id">
                          {{ user.name }}
                        </a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                </a-row>

                <a-row :gutter="16">
                  <a-col :span="12">
                    <a-form-item label="可执行动作">
                      <a-select v-model:value="step.actions" mode="multiple" placeholder="选择动作">
                        <a-select-option value="approve">同意</a-select-option>
                        <a-select-option value="reject">拒绝</a-select-option>
                        <a-select-option value="return">退回</a-select-option>
                        <a-select-option value="transfer">转交</a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item label="时间限制(分钟)">
                      <a-input-number v-model:value="step.time_limit" placeholder="时间限制" style="width: 100%" />
                    </a-form-item>
                  </a-col>
                </a-row>

                <a-row :gutter="16">
                  <a-col :span="12">
                    <a-form-item>
                      <a-checkbox v-model:checked="step.auto_assign">自动分配</a-checkbox>
                    </a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item>
                      <a-checkbox v-model:checked="step.parallel">并行处理</a-checkbox>
                    </a-form-item>
                  </a-col>
                </a-row>

                <a-form-item label="步骤位置">
                  <a-row :gutter="16">
                    <a-col :span="12">
                      <a-input-number v-model:value="step.position.x" placeholder="X坐标" style="width: 100%" />
                    </a-col>
                    <a-col :span="12">
                      <a-input-number v-model:value="step.position.y" placeholder="Y坐标" style="width: 100%" />
                    </a-col>
                  </a-row>
                </a-form-item>
              </a-collapse-panel>
            </a-collapse>

            <div class="add-step-button">
              <a-button type="dashed" block @click="addStep" style="margin-top: 16px">
                <PlusOutlined /> 添加步骤
              </a-button>
            </div>
          </div>
        </div>

        <a-divider orientation="left">流程连接</a-divider>

        <div class="connections-editor">
          <div class="connection-list">
            <div v-for="(connection, index) in processDialog.form.definition.connections" :key="index" class="connection-item">
              <a-row :gutter="16" align="middle">
                <a-col :span="10">
                  <a-select v-model:value="connection.from" placeholder="来源步骤" class="select-step">
                    <a-select-option v-for="step in processDialog.form.definition.steps" :key="step.id" :value="step.id">
                      {{ step.name }}
                    </a-select-option>
                  </a-select>
                </a-col>
                <a-col :span="10">
                  <a-select v-model:value="connection.to" placeholder="目标步骤" class="select-step">
                    <a-select-option v-for="step in processDialog.form.definition.steps" :key="step.id" :value="step.id">
                      {{ step.name }}
                    </a-select-option>
                  </a-select>
                </a-col>
                <a-col :span="4">
                  <a-button type="text" danger @click="removeConnection(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </a-col>
              </a-row>
              <a-row :gutter="16" style="margin-top: 8px">
                <a-col :span="12">
                  <a-input v-model:value="connection.condition" placeholder="条件表达式" />
                </a-col>
                <a-col :span="12">
                  <a-input v-model:value="connection.label" placeholder="连接标签" />
                </a-col>
              </a-row>
            </div>
            <a-button type="dashed" block @click="addConnection" style="margin-top: 16px">
              <PlusOutlined /> 添加连接
            </a-button>
          </div>
        </div>

        <a-divider orientation="left">流程变量</a-divider>

        <div class="variables-editor">
          <div class="variable-list">
            <div v-for="(variable, index) in processDialog.form.definition.variables" :key="index" class="variable-item">
              <a-row :gutter="16" align="middle">
                <a-col :span="5">
                  <a-input v-model:value="variable.name" placeholder="变量名" />
                </a-col>
                <a-col :span="4">
                  <a-select v-model:value="variable.type" placeholder="类型">
                    <a-select-option value="string">字符串</a-select-option>
                    <a-select-option value="number">数字</a-select-option>
                    <a-select-option value="boolean">布尔</a-select-option>
                    <a-select-option value="object">对象</a-select-option>
                  </a-select>
                </a-col>
                <a-col :span="5">
                  <a-input v-model:value="variable.default_value" placeholder="默认值" />
                </a-col>
                <a-col :span="8">
                  <a-input v-model:value="variable.description" placeholder="描述" />
                </a-col>
                <a-col :span="2">
                  <a-button type="text" danger @click="removeVariable(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </a-col>
              </a-row>
            </div>
            <a-button type="dashed" block @click="addVariable" style="margin-top: 16px">
              <PlusOutlined /> 添加变量
            </a-button>
          </div>
        </div>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆流程" @ok="confirmClone" :destroy-on-close="true">
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新流程名称" name="name">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新流程名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情对话框 -->
    <a-modal v-model:visible="detailDialog.visible" title="流程详情" width="80%" :footer="null" class="detail-dialog">
      <div v-if="detailDialog.process" class="process-details">
        <div class="detail-header">
          <h2>{{ detailDialog.process.name }}</h2>
          <a-tag
            :color="detailDialog.process.status === 0 ? 'orange' : detailDialog.process.status === 1 ? 'green' : 'default'">
            {{ detailDialog.process.status === 0 ? '草稿' : detailDialog.process.status === 1 ? '已发布' : '已禁用' }}
          </a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ detailDialog.process.id }}</a-descriptions-item>
          <a-descriptions-item label="版本">v{{ detailDialog.process.version }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.process.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.process.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="关联表单">{{ getFormName(detailDialog.process.form_design_id) }}</a-descriptions-item>
          <a-descriptions-item label="分类">{{ getCategoryName(detailDialog.process.category_id) }}</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">{{ detailDialog.process.description || '无描述' }}</a-descriptions-item>
        </a-descriptions>

        <div class="process-preview">
          <h3>流程步骤</h3>
          <div class="process-flow-chart">
            <div v-for="(step, index) in parseProcessSteps(detailDialog.process)" :key="index" class="process-node"
              :class="`node-type-${getNodeTypeClass(step.type)}`">
              <div class="node-header">
                <span class="node-type-badge">{{ getNodeTypeName(step.type) }}</span>
                <span class="node-name">{{ step.name }}</span>
              </div>
              <div class="node-content">
                <div class="node-info">
                  <div><strong>角色：</strong>{{ step.roles?.join(', ') || '无' }}</div>
                  <div><strong>动作：</strong>{{ step.actions?.join(', ') || '无' }}</div>
                  <div v-if="step.time_limit"><strong>时间限制：</strong>{{ step.time_limit }}分钟</div>
                  <div v-if="step.auto_assign"><strong>自动分配：</strong>是</div>
                  <div v-if="step.parallel"><strong>并行处理：</strong>是</div>
                </div>
              </div>
              <div class="node-footer" v-if="getNextStepName(step.id, detailDialog.process)">
                <ArrowDownOutlined />
                <div>下一步骤：{{ getNextStepName(step.id, detailDialog.process) }}</div>
              </div>
            </div>
          </div>

          <div v-if="parseProcessConnections(detailDialog.process)?.length" class="connections-section">
            <h3>流程连接</h3>
            <a-table :data-source="parseProcessConnections(detailDialog.process)" :columns="connectionDisplayColumns" :pagination="false" size="small">
            </a-table>
          </div>

          <div v-if="parseProcessVariables(detailDialog.process)?.length" class="variables-section">
            <h3>流程变量</h3>
            <a-table :data-source="parseProcessVariables(detailDialog.process)" :columns="variableColumns" :pagination="false" size="small">
            </a-table>
          </div>
        </div>

        <div class="detail-footer">
          <a-button @click="detailDialog.visible = false">关闭</a-button>
          <a-button type="primary" @click="handleEditProcess(detailDialog.process)">编辑</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  ApartmentOutlined,
  CheckCircleOutlined,
  EditOutlined,
  StopOutlined,
  DeleteOutlined,
  DownOutlined,
  ArrowDownOutlined
} from '@ant-design/icons-vue';

import {
  type ProcessItem,
  type ProcessResp,
  type CreateProcessReq,
  type UpdateProcessReq,
  type DeleteProcessReq,
  type PublishProcessReq,
  type CloneProcessReq,
  type ListProcessReq,
  type ProcessStep,
  type ProcessConnection,
  type ProcessVariable,
  type ProcessDefinition,
  listProcess,
  detailProcess,
  createProcess,
  updateProcess,
  deleteProcess,
  publishProcess,
  cloneProcess,
  validateProcess
} from '#/api/core/workorder_process';

import {listFormDesign} from '#/api/core/workorder_form_design'
import type { ListFormDesignReq } from '#/api/core/workorder';

// 添加分类接口导入
import type { Category } from '#/api/core/workorder_category'
import { listCategory } from '#/api/core/workorder_category'

// 列定义
const columns = [
  {
    title: '流程名称',
    dataIndex: 'name',
    key: 'name',
    width: 180,
    fixed: 'left',
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    width: 200,
    ellipsis: true,
  },
  {
    title: '关联表单',
    dataIndex: 'form_design_id',
    key: 'form_design',
    width: 150,
  },
  {
    title: '分类',
    dataIndex: 'category_id',
    key: 'category',
    width: 120,
  },
  {
    title: '版本',
    dataIndex: 'version',
    key: 'version',
    width: 100,
    align: 'center',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    align: 'center',
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
    align: 'center',
    fixed: 'right',
  },
];

// 连接表格列定义
const connectionColumns = [
  { title: '来源步骤', dataIndex: 'from', key: 'from' },
  { title: '目标步骤', dataIndex: 'to', key: 'to' },
  { title: '条件', dataIndex: 'condition', key: 'condition' },
  { title: '标签', dataIndex: 'label', key: 'label' },
];

// 用于显示连接的表格列
const connectionDisplayColumns = [
  { 
    title: '来源步骤',
    dataIndex: 'from', 
    key: 'from',
    customRender: ({ text, record }: any) => {
      return getStepNameById(text, record._process);
    }
  },
  { 
    title: '目标步骤', 
    dataIndex: 'to', 
    key: 'to',
    customRender: ({ text, record }: any) => {
      return getStepNameById(text, record._process);
    }
  },
  { title: '条件', dataIndex: 'condition', key: 'condition' },
  { title: '标签', dataIndex: 'label', key: 'label' },
];

// 变量表格列定义
const variableColumns = [
  { title: '变量名', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'type', key: 'type' },
  { title: '默认值', dataIndex: 'default_value', key: 'default_value' },
  { title: '描述', dataIndex: 'description', key: 'description' },
];

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref<number | null>(null);
const categoryFilter = ref<number | null>(null);
const currentPage = ref(1);
const pageSize = ref(10);
const total = ref(0);

// 统计数据
const stats = reactive({
  total: 0,
  published: 0,
  draft: 0,
  disabled: 0
});

// 数据列表 - 修复分类数据类型
const processList = ref<ProcessItem[]>([]);
const forms = ref<any[]>([]);
const categories = ref<Category[]>([]); // 使用正确的类型
const roles = ref<string[]>(['admin', 'user', 'manager', 'reviewer']);
const users = ref<any[]>([]);

// 流程对话框
const processDialog = reactive({
  visible: false,
  isEdit: false,
  form: {
    id: undefined,
    name: '',
    description: '',
    form_design_id: undefined as number | undefined,
    category_id: undefined as number | undefined,
    status: 0,
    definition: {
      steps: [],
      connections: [],
      variables: []
    } as ProcessDefinition
  } as CreateProcessReq & { id?: number; status?: number }
});

// 激活的步骤键
const activeStepKeys = ref<string[]>([]);

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入流程名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  form_design_id: [
    { required: true, message: '请选择关联表单', trigger: 'change' }
  ]
};

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  form: {
    name: '',
    id: 0
  } as CloneProcessReq
});

// 详情对话框
const detailDialog = reactive({
  visible: false,
  process: null as ProcessResp | null
});

// 生成唯一ID
const generateId = () => {
  return 'step_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
};

// 解析流程定义JSON字符串
const parseProcessDefinition = (process: any): ProcessDefinition => {
  if (!process) return { steps: [], connections: [], variables: [] };
  
  if (typeof process.definition === 'string') {
    try {
      return JSON.parse(process.definition);
    } catch (e) {
      console.error('Error parsing process definition:', e);
      return { steps: [], connections: [], variables: [] };
    }
  }
  
  return process.definition || { steps: [], connections: [], variables: [] };
};

// 解析流程步骤
const parseProcessSteps = (process: any): ProcessStep[] => {
  const definition = parseProcessDefinition(process);
  return definition.steps || [];
};

// 解析流程连接
const parseProcessConnections = (process: any): any[] => {
  const definition = parseProcessDefinition(process);
  return (definition.connections || []).map(conn => ({
    ...conn,
    _process: process // 添加流程引用以便在渲染时获取步骤名称
  }));
};

// 解析流程变量
const parseProcessVariables = (process: any): ProcessVariable[] => {
  const definition = parseProcessDefinition(process);
  return definition.variables || [];
};

// 根据步骤ID获取步骤名称
const getStepNameById = (stepId: string, process: any): string => {
  if (!process || !stepId) return '未知步骤';
  
  const steps = parseProcessSteps(process);
  const step = steps.find(s => s.id === stepId);
  return step ? step.name : '未知步骤';
};

// 获取下一个步骤的名称
const getNextStepName = (stepId: string, process: any): string => {
  if (!process || !stepId) return '';
  
  const connections = parseProcessConnections(process);
  const nextConnection = connections.find(conn => conn.from === stepId);
  
  if (nextConnection) {
    return getStepNameById(nextConnection.to, process);
  }
  
  return '';
};

// 初始化加载数据
const loadProcesses = async () => {
  loading.value = true;
  try {
    const params: ListProcessReq = {
      page: currentPage.value,
      size: pageSize.value,
      name: searchQuery.value || undefined,
      status: statusFilter.value || undefined,
      category_id: categoryFilter.value || undefined
    };
    
    const res = await listProcess(params);
    if (res && res.items) {
      processList.value = res.items || [];
      total.value = res.total || 0;
      
      // 更新统计数据
      stats.total = res.total || 0;
      stats.published = processList.value.filter((p: ProcessItem) => p.status === 1).length;
      stats.draft = processList.value.filter((p: ProcessItem) => p.status === 0).length;
      stats.disabled = processList.value.filter((p: ProcessItem) => p.status === 2).length;
    }
  } catch (error) {
    message.error('加载流程数据失败');
    console.error('Failed to load processes:', error);
  } finally {
    loading.value = false;
  }
};

// 加载表单列表
const loadForms = async () => {
  try {
    const params: ListFormDesignReq = {
      page: 1,
      size: 100,
    };
    const res = await listFormDesign(params)
    if (res && res.items) {
      forms.value = res.items;
      console.log('表单数据加载成功:', forms.value);
    } else {
      console.warn('表单接口返回数据格式异常:', res);
      forms.value = [];
    }
  } catch (error) {
    console.error('Failed to load forms:', error);
    message.error('加载表单列表失败');
    forms.value = [];
  }
};

// 修复加载分类列表函数
const loadCategories = async (): Promise<void> => {
  try {
    // 使用真实的分类接口
    const response = await listCategory({ page: 1, size: 100 });
    if (response && response.items) {
      categories.value = response.items;
      console.log('分类数据加载成功:', categories.value);
    } else {
      console.warn('分类接口返回数据格式异常:', response);
      categories.value = [];
    }
  } catch (error) {
    console.error('Failed to load categories:', error);
    message.error('加载分类列表失败');
    // 出错时提供默认的分类数据
    categories.value = [];
  }
};

// 加载用户列表
const loadUsers = async () => {
  try {
    // 这里模拟用户列表数据，实际应调用相关API
    // 如果有用户管理接口，可以替换为真实接口调用
    users.value = [
      { id: 1, name: '张三' },
      { id: 2, name: '李四' },
      { id: 3, name: '王五' }
    ];
    console.log('用户数据加载成功:', users.value);
  } catch (error) {
    console.error('Failed to load users:', error);
    users.value = [];
  }
};

// 方法
const handleSizeChange = (_: number, size: number) => {
  pageSize.value = size;
  currentPage.value = 1;
  loadProcesses();
};

const handleCurrentChange = (page: number) => {
  currentPage.value = page;
  loadProcesses();
};

const handleSearch = () => {
  currentPage.value = 1;
  loadProcesses();
};

const handleStatusChange = () => {
  currentPage.value = 1;
  loadProcesses();
};

const handleCategoryChange = () => {
  currentPage.value = 1;
  loadProcesses();
};

const handleCreateProcess = () => {
  processDialog.isEdit = false;
  processDialog.form = {
    name: '',
    description: '',
    form_design_id: undefined,
    category_id: undefined,
    definition: {
      steps: [
        {
          id: generateId(),
          name: '开始',
          type: 'start',
          roles: [],
          users: [],
          actions: [],
          conditions: [],
          auto_assign: false,
          parallel: false,
          props: {},
          position: { x: 100, y: 100 }
        }
      ],
      connections: [],
      variables: []
    }
  };
  activeStepKeys.value = ['0'];
  processDialog.visible = true;
};

const handleEditProcess = async (row: ProcessItem) => {
  processDialog.isEdit = true;
  loading.value = true;
  
  try {
    const res = await detailProcess({ id: row.id });
    if (res) {
      // 确保从字符串转换为对象
      const definition = typeof res.definition === 'string' 
        ? JSON.parse(res.definition) 
        : res.definition;
      
      processDialog.form = {
        id: res.id,
        name: res.name,
        description: res.description,
        form_design_id: res.form_design_id,
        category_id: res.category_id,
        status: res.status,
        definition: definition
      };
      
      processDialog.visible = true;
      detailDialog.visible = false;
      activeStepKeys.value = processDialog.form.definition.steps.map((step: any, index: number) => index.toString());
    }
  } catch (error) {
    message.error('获取流程详情失败');
    console.error('Failed to get process details:', error);
  } finally {
    loading.value = false;
  }
};

const handleViewProcess = async (row: ProcessItem) => {
  loading.value = true;
  
  try {
    const res = await detailProcess({ id: row.id });
    if (res) {
      detailDialog.process = res;
      detailDialog.visible = true;
    }
  } catch (error) {
    message.error('获取流程详情失败');
    console.error('Failed to get process details:', error);
  } finally {
    loading.value = false;
  }
};

const handleCommand = async (command: string, row: ProcessItem) => {
  switch (command) {
    case 'publish':
      await publishProcessHandler(row);
      break;
    case 'unpublish':
      // 实际项目中应使用unpublishProcess API，这里简化处理
      message.info('取消发布功能需要实现相应的API');
      break;
    case 'validate':
      await validateProcessHandler(row);
      break;
    case 'clone':
      showCloneDialog(row);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

const publishProcessHandler = async (process: ProcessItem) => {
  try {
    const params: PublishProcessReq = {
      id: process.id
    };
    
    await publishProcess(params);
    message.success(`流程 "${process.name}" 已发布`);
    loadProcesses();
  } catch (error) {
    message.error('发布流程失败');
    console.error('Failed to publish process:', error);
  }
};

const validateProcessHandler = async (process: ProcessItem) => {
  try {
    const res = await validateProcess(process.id);
    if (res && res.is_valid) {
      message.success(`流程 "${process.name}" 验证通过`);
    } else {
      message.error(`流程验证失败：${res?.errors?.join(', ') || '未知错误'}`);
    }
  } catch (error) {
    message.error('验证流程失败');
    console.error('Failed to validate process:', error);
  }
};

const showCloneDialog = (process: ProcessItem) => {
  cloneDialog.form = {
    id: process.id,
    name: `${process.name} 的副本`
  };
  cloneDialog.visible = true;
};

const confirmClone = async () => {
  try {
    if (!cloneDialog.form.name.trim()) {
      message.error('请输入新流程名称');
      return;
    }
    
    await cloneProcess(cloneDialog.form);
    message.success(`流程已克隆为 "${cloneDialog.form.name}"`);
    cloneDialog.visible = false;
    loadProcesses();
  } catch (error) {
    message.error('克隆流程失败');
    console.error('Failed to clone process:', error);
  }
};

const confirmDelete = (process: ProcessItem) => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除流程 "${process.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        const params: DeleteProcessReq = {
          id: process.id
        };
        
        await deleteProcess(params);
        message.success(`流程 "${process.name}" 已删除`);
        loadProcesses();
      } catch (error) {
        message.error('删除流程失败');
        console.error('Failed to delete process:', error);
      }
    }
  });
};

// 步骤管理
const addStep = () => {
  const newStep: ProcessStep = {
    id: generateId(),
    name: '',
    type: 'approval',
    roles: [],
    users: [],
    actions: [],
    conditions: [],
    auto_assign: false,
    parallel: false,
    props: {},
    position: { x: 100, y: 100 + processDialog.form.definition.steps.length * 150 }
  };
  
  processDialog.form.definition.steps.push(newStep);
  activeStepKeys.value.push((processDialog.form.definition.steps.length - 1).toString());
};

const removeStep = (index: number) => {
  processDialog.form.definition.steps.splice(index, 1);
  activeStepKeys.value = activeStepKeys.value.filter(key => key !== index.toString());
};

// 连接管理
const addConnection = () => {
  const newConnection: ProcessConnection = {
    from: '',
    to: '',
    condition: '',
    label: ''
  };
  
  processDialog.form.definition.connections.push(newConnection);
};

const removeConnection = (index: number) => {
  processDialog.form.definition.connections.splice(index, 1);
};

// 变量管理
const addVariable = () => {
  const newVariable: ProcessVariable = {
    name: '',
    type: 'string',
    default_value: '',
    description: ''
  };
  
  processDialog.form.definition.variables.push(newVariable);
};

const removeVariable = (index: number) => {
  processDialog.form.definition.variables.splice(index, 1);
};

const saveProcess = async () => {
  try {
    // 基础验证
    if (!processDialog.form.name.trim()) {
      message.error('流程名称不能为空');
      return;
    }

    if (!processDialog.form.form_design_id) {
      message.error('请选择关联表单');
      return;
    }

    if (processDialog.form.definition.steps.length === 0) {
      message.error('流程至少需要一个步骤');
      return;
    }

    // 验证步骤
    for (let i = 0; i < processDialog.form.definition.steps.length; i++) {
      const step = processDialog.form.definition.steps[i];
      if (!step || !step.name || !step.name.trim()) {
        message.error(`步骤 ${i + 1} 名称不能为空`);
        return;
      }
      if (!step || !step.type) {
        message.error(`步骤 ${i + 1} 类型不能为空`);
        return;
      }
    }

    if (processDialog.isEdit && processDialog.form.id) {
      // 更新现有流程
      const updateData: UpdateProcessReq = {
        id: processDialog.form.id,
        name: processDialog.form.name,
        description: processDialog.form.description || '',
        form_design_id: processDialog.form.form_design_id!,
        definition: processDialog.form.definition,
        category_id: processDialog.form.category_id
      };
      
      await updateProcess(updateData);
      message.success(`流程 "${processDialog.form.name}" 已更新`);
    } else {
      // 创建新流程
      const createData: CreateProcessReq = {
        name: processDialog.form.name,
        description: processDialog.form.description,
        form_design_id: processDialog.form.form_design_id!,
        definition: processDialog.form.definition,
        category_id: processDialog.form.category_id
      };
      
      await createProcess(createData);
      message.success(`流程 "${processDialog.form.name}" 已创建`);
    }
    
    processDialog.visible = false;
    loadProcesses();
  } catch (error) {
    message.error(processDialog.isEdit ? '更新流程失败' : '创建流程失败');
    console.error('Failed to save process:', error);
  }
};

// 检查流程名是否存在（简化实现，实际应调用API）
const checkProcessNameExists = async (name: string) => {
  // 模拟API调用
  return { exists: false };
};

// 更新流程状态（简化实现，实际应调用API）
const updateProcessStatus = async (id: number, status: number) => {
  message.info(`更新流程状态为 ${status}`);
  // 实际项目中应调用相应的API
};

// 辅助方法
const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (dateStr: string | undefined) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (dateStr: string | undefined) => {
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

const getInitials = (name: string | undefined) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: number) => {
  switch (status) {
    case 0: return 'status-draft';
    case 1: return 'status-published';
    case 2: return 'status-disabled';
    default: return '';
  }
};

const getAvatarColor = (name: string | undefined) => {
  if (!name) return '#1890ff';
  
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

const getFormName = (formId: number | undefined) => {
  if (!formId) return '未知表单';
  const form = forms.value.find(f => f.id === formId);
  return form ? form.name : '未知表单';
};

const getCategoryName = (categoryId: number | undefined) => {
  if (!categoryId) return '无分类';
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.name : '无分类';
};

const getNodeTypeClass = (type: string) => {
  const map: Record<string, string> = {
    'start': 'start',
    'approval': 'approval',
    'condition': 'condition',
    'notification': 'notice',
    'end': 'end'
  };
  return map[type] || 'approval';
};

const getNodeTypeName = (type: string) => {
  const typeMap: Record<string, string> = {
    'start': '开始',
    'approval': '审批',
    'condition': '条件',
    'notification': '通知',
    'end': '结束'
  };
  return typeMap[type] || type;
};

// 修复初始化加载 - 并行加载所有数据
onMounted(async () => {
  try {
    // 并行加载所有数据，提高加载效率
    await Promise.all([
      loadForms(),
      loadCategories(),
      loadUsers(),
      loadProcesses()
    ]);
    console.log('所有数据加载完成');
  } catch (error) {
    console.error('初始化数据加载失败:', error);
    message.error('初始化数据加载失败，请刷新页面重试');
  }
});
</script>

<style scoped>
.process-container {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
}

/* 头部布局 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.header-left .btn-create {
  height: 38px;
  padding: 0 20px;
  font-weight: 500;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.2);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* 统计区域 */
.stats-section {
  margin-bottom: 20px;
}

.stats-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: none;
  transition: all 0.3s ease;
}

.stats-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

/* 表格区域 */
.table-section {
  margin-bottom: 20px;
}

.table-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: none;
}

/* 表格单元格样式 */
.process-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.process-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-draft {
  background-color: #faad14;
}

.status-published {
  background-color: #52c41a;
}

.status-disabled {
  background-color: #d9d9d9;
}

.process-name-text {
  font-weight: 500;
  color: #262626;
}

.description-text {
  color: #8c8c8c;
  font-size: 14px;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.creator-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.creator-name {
  font-size: 14px;
  color: #595959;
}

.date-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.date {
  font-weight: 500;
  font-size: 14px;
  color: #262626;
}

.time {
  font-size: 12px;
  color: #8c8c8c;
}

.action-buttons {
  display: flex;
  gap: 6px;
  justify-content: center;
  flex-wrap: wrap;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

/* 步骤编辑器 */
.steps-editor {
  background: #fafafa;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 20px;
}

.step-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.add-step-button {
  text-align: center;
  margin-top: 16px;
}

/* 连接和变量编辑器 */
.connections-editor,
.variables-editor {
  background: #fafafa;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 20px;
}

.connection-item,
.variable-item {
  margin-bottom: 12px;
  padding: 12px;
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
}

.connection-item:last-child,
.variable-item:last-child {
  margin-bottom: 0;
}

/* 详情对话框 */
.detail-dialog .process-details {
  margin-bottom: 20px;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.detail-header h2 {
  margin: 0;
  font-size: 24px;
  color: #1f2937;
  font-weight: 600;
}

.process-preview {
  margin-top: 24px;
}

.process-preview h3 {
  margin: 24px 0 16px 0;
  color: #1f2937;
  font-size: 18px;
  font-weight: 600;
}

.process-flow-chart {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 400px;
  overflow-y: auto;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
}

.process-node {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 16px;
  background: #fff;
  transition: all 0.3s ease;
  position: relative;
}

.process-node:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.node-type-start {
  background: linear-gradient(135deg, #e6f7ff 0%, #bae7ff 100%);
  border-color: #91d5ff;
}

.node-type-approval {
  background: linear-gradient(135deg, #f6ffed 0%, #d9f7be 100%);
  border-color: #b7eb8f;
}

.node-type-notice {
  background: linear-gradient(135deg, #fffbe6 0%, #fff1b8 100%);
  border-color: #ffe58f;
}

.node-type-condition {
  background: linear-gradient(135deg, #fff7e6 0%, #ffd591 100%);
  border-color: #ffcc02;
}

.node-type-end {
  background: linear-gradient(135deg, #f9f0ff 0%, #efdbff 100%);
  border-color: #d3adf7;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.node-type-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  color: #fff;
  background-color: #1890ff;
}

.node-type-start .node-type-badge {
  background-color: #1890ff;
}

.node-type-approval .node-type-badge {
  background-color: #52c41a;
}

.node-type-notice .node-type-badge {
  background-color: #faad14;
}

.node-type-condition .node-type-badge {
  background-color: #fa8c16;
}

.node-type-end .node-type-badge {
  background-color: #722ed1;
}

.node-name {
  font-weight: 600;
  font-size: 16px;
  color: #262626;
}

.node-content {
  margin-bottom: 12px;
}

.node-info {
  font-size: 14px;
  line-height: 1.6;
}

.node-info div {
  margin-bottom: 4px;
  color: #595959;
}

.node-info strong {
  color: #262626;
  font-weight: 500;
}

.node-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed #d9d9d9;
  color: #8c8c8c;
  font-size: 13px;
}

.connections-section,
.variables-section {
  margin-top: 24px;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

/* 下拉框宽度修复 */
.select-step {
  width: 100% !important;
  min-width: 180px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .process-container {
    padding: 12px;
  }
  
  .page-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  
  .header-right {
    flex-wrap: wrap;
    gap: 8px;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 4px;
  }
  
  .stats-section .ant-col {
    margin-bottom: 12px;
  }
}

@media (max-width: 576px) {
  .header-right > * {
    width: 100%;
  }
  
  .action-buttons .ant-btn {
    width: 100%;
  }
}
</style>