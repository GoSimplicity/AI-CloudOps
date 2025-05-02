<template>
  <div class="process-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateProcess" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新流程
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索流程..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部</a-select-option>
          <a-select-option :value="0">草稿</a-select-option>
          <a-select-option :value="1">已发布</a-select-option>
          <a-select-option :value="2">已禁用</a-select-option>
        </a-select>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="总流程数" :value="stats?.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <ApartmentOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已发布" :value="stats?.published" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="草稿" :value="stats?.draft" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <EditOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已禁用" :value="stats?.disabled" :value-style="{ color: '#cf1322' }">
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
        <a-table :data-source="paginatedProcesses" :columns="columns" :pagination="false" :loading="loading"
          row-key="id" bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="process-name-cell">
                <div class="process-badge" :class="getStatusClass(record.status)"></div>
                <span class="process-name-text">{{ record.name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <span class="description-text">{{ record.description || '无描述' }}</span>
            </template>

            <template v-if="column.key === 'version'">
              <a-tag color="blue">v{{ record.version }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 0 ? 'orange' : record.status === 1 ? 'green' : 'default'">
                {{ record.status === 0 ? '草稿' : record.status === 1 ? '已发布' : '已禁用' }}
              </a-tag>
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
                <a-button type="primary" size="small" @click="handleViewProcess(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditProcess(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="handleCommand(column.key, record)">
                      <a-menu-item key="publish" v-if="record.status === 0">发布</a-menu-item>
                      <a-menu-item key="unpublish" v-if="record.status === 1">取消发布</a-menu-item>
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

        <div class="pagination-container">
          <a-pagination v-model:current="currentPage" :total="totalItems" :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" :show-size-changer="true" @change="handleCurrentChange"
            @show-size-change="handleSizeChange" :show-total="(total: number) => `共 ${total} 条`" />
        </div>
      </a-card>
    </div>

    <!-- 流程创建/编辑对话框 -->
    <a-modal v-model:visible="processDialog.visible" :title="processDialog.isEdit ? '编辑流程' : '创建流程'" width="760px"
      @ok="saveProcess" :destroy-on-close="true">
      <a-form ref="formRef" :model="processDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="流程名称" name="name">
          <a-input v-model:value="processDialog.form.name" placeholder="请输入流程名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="processDialog.form.description" :rows="3" placeholder="请输入流程描述" />
        </a-form-item>

        <a-form-item label="关联表单" name="formID">
          <a-select v-model:value="processDialog.form.formID" placeholder="请选择关联表单" style="width: 100%">
            <a-select-option v-for="form in forms" :key="form.id" :value="form.id">
              {{ form.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="processDialog.form.status">
            <a-radio :value="0">草稿</a-radio>
            <a-radio :value="1">已发布</a-radio>
            <a-radio :value="2">已禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider orientation="left">流程节点</a-divider>

        <div class="nodes-editor">
          <div class="node-list">
            <a-collapse>
              <a-collapse-panel v-for="(node, index) in processDialog.form.nodes" :key="index"
                :header="node.name || `节点 ${index + 1}`">
                <template #extra>
                  <a-button type="text" danger @click.stop="removeNode(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </template>

                <a-form-item label="节点名称">
                  <a-input v-model:value="node.name" placeholder="节点名称" />
                </a-form-item>

                <a-form-item label="节点类型">
                  <a-select v-model:value="node.type" style="width: 100%">
                    <a-select-option value="start">开始节点</a-select-option>
                    <a-select-option value="approval">审批节点</a-select-option>
                    <a-select-option value="notice">通知节点</a-select-option>
                    <a-select-option value="condition">条件节点</a-select-option>
                    <a-select-option value="end">结束节点</a-select-option>
                  </a-select>
                </a-form-item>

                <a-form-item label="处理人" v-if="node.type === 'approval'">
                  <a-select v-model:value="node.approvers" mode="multiple" style="width: 100%" placeholder="选择处理人">
                    <a-select-option v-for="user in users" :key="user.id" :value="user.id">
                      {{ user.name }}
                    </a-select-option>
                  </a-select>
                </a-form-item>

                <a-form-item label="通知人" v-if="node.type === 'notice'">
                  <a-select v-model:value="node.notifyUsers" mode="multiple" style="width: 100%" placeholder="选择通知人">
                    <a-select-option v-for="user in users" :key="user.id" :value="user.id">
                      {{ user.name }}
                    </a-select-option>
                  </a-select>
                </a-form-item>

                <a-form-item label="条件表达式" v-if="node.type === 'condition'">
                  <a-textarea v-model:value="node.condition" placeholder="请输入条件表达式" :rows="2" />
                </a-form-item>

                <a-form-item label="下一节点">
                  <a-select v-model:value="node.nextNode" style="width: 100%" placeholder="选择下一节点">
                    <a-select-option :value="null">无（结束流程）</a-select-option>
                    <a-select-option v-for="(otherNode, otherIndex) in processDialog.form.nodes" :key="otherIndex"
                      :value="otherIndex" :disabled="otherIndex === index">
                      {{ otherNode.name || `节点 ${otherIndex + 1}` }}
                    </a-select-option>
                  </a-select>
                </a-form-item>
              </a-collapse-panel>
            </a-collapse>

            <div class="add-node-button">
              <a-button type="dashed" block @click="addNode" style="margin-top: 16px">
                <PlusOutlined /> 添加节点
              </a-button>
            </div>
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
    <a-modal v-model:visible="detailDialog.visible" title="流程详情" width="70%" :footer="null" class="detail-dialog">
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
          <a-descriptions-item label="创建人">{{ detailDialog.process.creatorName }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.process.createdAt)
          }}</a-descriptions-item>
          <a-descriptions-item label="关联表单">{{ getFormName(detailDialog.process.formID) }}</a-descriptions-item>
          <a-descriptions-item label="描述">{{ detailDialog.process.description || '无描述' }}</a-descriptions-item>
        </a-descriptions>

        <div class="process-preview">
          <h3>流程节点</h3>
          <div class="process-flow-chart">
            <div v-for="(node, index) in detailDialog.process.nodes" :key="index" class="process-node"
              :class="'node-type-' + node.type">
              <div class="node-header">
                <span class="node-type-badge">{{ getNodeTypeName(node.type) }}</span>
                <span class="node-name">{{ node.name }}</span>
              </div>
              <div class="node-content">
                <div v-if="node.type === 'approval'" class="node-approvers">
                  <div>处理人：</div>
                  <a-avatar-group :max-count="3" size="small">
                    <a-tooltip v-for="userId in node.approvers" :key="userId" :title="getUserName(userId)">
                      <a-avatar :style="{ backgroundColor: getAvatarColor(getUserName(userId)) }">
                        {{ getInitials(getUserName(userId)) }}
                      </a-avatar>
                    </a-tooltip>
                  </a-avatar-group>
                </div>
                <div v-if="node.type === 'notice'" class="node-notify-users">
                  <div>通知人：</div>
                  <a-avatar-group :max-count="3" size="small">
                    <a-tooltip v-for="userId in node.notifyUsers" :key="userId" :title="getUserName(userId)">
                      <a-avatar :style="{ backgroundColor: getAvatarColor(getUserName(userId)) }">
                        {{ getInitials(getUserName(userId)) }}
                      </a-avatar>
                    </a-tooltip>
                  </a-avatar-group>
                </div>
                <div v-if="node.type === 'condition'" class="node-condition">
                  <div>条件：</div>
                  <div class="condition-expression">{{ node.condition }}</div>
                </div>
              </div>
              <div class="node-footer" v-if="node.nextNode !== null">
                <ArrowDownOutlined />
                <div>下一节点：{{ getNodeName(detailDialog.process.nodes, node.nextNode) }}</div>
              </div>
            </div>
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
import { ref, reactive, computed, onMounted } from 'vue';
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

// 基于Golang模型的类型定义
interface ProcessNode {
  name: string;
  type: string; // start, approval, notice, condition, end
  approvers?: number[]; // 用于审批节点
  notifyUsers?: number[]; // 用于通知节点
  condition?: string; // 用于条件节点
  nextNode: number | null; // 下一个节点的索引，null表示结束
}

interface Process {
  id: number;
  name: string;
  description: string;
  formID: number;
  nodes: ProcessNode[];
  version: number;
  status: number; // 0-草稿，1-已发布，2-已禁用
  creatorID: number;
  creatorName: string;
  createdAt: Date;
  updatedAt: Date;
}

interface Form {
  id: number;
  name: string;
}

interface User {
  id: number;
  name: string;
}

// 列定义
const columns = [
  {
    title: '流程名称',
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

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);

// 统计数据
const stats = reactive({
  total: 36,
  published: 24,
  draft: 10,
  disabled: 2
});

// 模拟流程数据
const processes = ref<Process[]>([
  {
    id: 1,
    name: '员工入职审批流程',
    description: '新员工入职审批流程',
    formID: 1,
    nodes: [
      { name: '开始', type: 'start', nextNode: 1 },
      { name: '部门经理审批', type: 'approval', approvers: [101, 102], nextNode: 2 },
      { name: '人力资源审批', type: 'approval', approvers: [103], nextNode: 3 },
      { name: '通知入职', type: 'notice', notifyUsers: [101, 103, 104], nextNode: 4 },
      { name: '结束', type: 'end', nextNode: null }
    ],
    version: 2,
    status: 1, // 已发布
    creatorID: 101,
    creatorName: '张三',
    createdAt: new Date('2025-01-15T08:30:00'),
    updatedAt: new Date('2025-02-10T14:45:00')
  },
  {
    id: 2,
    name: '休假申请流程',
    description: '员工休假申请审批流程',
    formID: 2,
    nodes: [
      { name: '开始', type: 'start', nextNode: 1 },
      { name: '判断休假天数', type: 'condition', condition: '休假天数 > 3', nextNode: 2 },
      { name: '部门经理审批', type: 'approval', approvers: [101, 102], nextNode: 3 },
      { name: '人力资源备案', type: 'notice', notifyUsers: [103], nextNode: 4 },
      { name: '结束', type: 'end', nextNode: null }
    ],
    version: 1,
    status: 1, // 已发布
    creatorID: 102,
    creatorName: '李四',
    createdAt: new Date('2025-01-20T10:15:00'),
    updatedAt: new Date('2025-01-20T10:15:00')
  },
  {
    id: 3,
    name: 'IT设备申请流程',
    description: 'IT设备申请审批流程',
    formID: 4,
    nodes: [
      { name: '开始', type: 'start', nextNode: 1 },
      { name: '部门经理审批', type: 'approval', approvers: [101, 102], nextNode: 2 },
      { name: '判断金额', type: 'condition', condition: '金额 > 5000', nextNode: 3 },
      { name: '财务审批', type: 'approval', approvers: [105], nextNode: 4 },
      { name: 'IT部门处理', type: 'approval', approvers: [106], nextNode: 5 },
      { name: '结束', type: 'end', nextNode: null }
    ],
    version: 3,
    status: 1, // 已发布
    creatorID: 103,
    creatorName: '王五',
    createdAt: new Date('2025-01-05T09:20:00'),
    updatedAt: new Date('2025-03-15T11:30:00')
  },
  {
    id: 4,
    name: '报销审批流程',
    description: '员工报销审批流程',
    formID: 5,
    nodes: [
      { name: '开始', type: 'start', nextNode: 1 },
      { name: '部门经理审批', type: 'approval', approvers: [101, 102], nextNode: 2 },
      { name: '判断金额', type: 'condition', condition: '金额 > 10000', nextNode: 3 },
      { name: '财务总监审批', type: 'approval', approvers: [107], nextNode: 4 },
      { name: '财务处理', type: 'approval', approvers: [105], nextNode: 5 },
      { name: '结束', type: 'end', nextNode: null }
    ],
    version: 1,
    status: 0, // 草稿
    creatorID: 104,
    creatorName: '赵六',
    createdAt: new Date('2025-03-10T16:45:00'),
    updatedAt: new Date('2025-03-10T16:45:00')
  },
  {
    id: 5,
    name: '项目立项流程',
    description: '新项目立项审批流程',
    formID: 3,
    nodes: [
      { name: '开始', type: 'start', nextNode: 1 },
      { name: '部门经理审批', type: 'approval', approvers: [101, 102], nextNode: 2 },
      { name: '技术评估', type: 'approval', approvers: [106, 108], nextNode: 3 },
      { name: '财务评估', type: 'approval', approvers: [105], nextNode: 4 },
      { name: '总经理审批', type: 'approval', approvers: [109], nextNode: 5 },
      { name: '通知相关部门', type: 'notice', notifyUsers: [101, 103, 105, 106], nextNode: 6 },
      { name: '结束', type: 'end', nextNode: null }
    ],
    version: 2,
    status: 1, // 已发布
    creatorID: 102,
    creatorName: '李四',
    createdAt: new Date('2025-02-05T11:30:00'),
    updatedAt: new Date('2025-02-20T09:15:00')
  }
]);

// 模拟表单数据
const forms = ref<Form[]>([
  { id: 1, name: '员工入职表单' },
  { id: 2, name: '休假申请表' },
  { id: 3, name: 'IT支持请求' },
  { id: 4, name: '设备采购申请' },
  { id: 5, name: '差旅报销单' }
]);

// 模拟用户数据
const users = ref<User[]>([
  { id: 101, name: '张三' },
  { id: 102, name: '李四' },
  { id: 103, name: '王五' },
  { id: 104, name: '赵六' },
  { id: 105, name: '财务经理' },
  { id: 106, name: 'IT主管' },
  { id: 107, name: '财务总监' },
  { id: 108, name: '技术总监' },
  { id: 109, name: '总经理' }
]);

// 过滤和分页
const filteredProcesses = computed(() => {
  let result = [...processes.value];

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(process =>
      process.name.toLowerCase().includes(query) ||
      (process.description && process.description.toLowerCase().includes(query))
    );
  }

  if (statusFilter.value !== null) {
    result = result.filter(process => process.status === statusFilter.value);
  }

  return result;
});

const totalItems = computed(() => filteredProcesses.value.length);

const paginatedProcesses = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredProcesses.value.slice(start, end);
});

// 流程对话框
const processDialog = reactive({
  visible: false,
  isEdit: false,
  form: {
    id: 0,
    name: '',
    description: '',
    formID: null as number | null,
    nodes: [] as ProcessNode[],
    version: 1,
    status: 0,
    creatorID: 101, // 模拟用户ID
    creatorName: '当前用户', // 模拟用户名
    createdAt: new Date(),
    updatedAt: new Date()
  }
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入流程名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  formID: [
    { required: true, message: '请选择关联表单', trigger: 'change' }
  ]
};

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  form: {
    name: '',
    originalId: 0
  }
});

// 详情对话框
const detailDialog = reactive({
  visible: false,
  process: null as Process | null
});

// 方法
const handleSizeChange = (current: number, size: number) => {
  pageSize.value = size;
  currentPage.value = 1;
};

const handleCurrentChange = (page: number) => {
  currentPage.value = page;
};

const handleSearch = () => {
  currentPage.value = 1;
};

const handleStatusChange = () => {
  currentPage.value = 1;
};

const handleCreateProcess = () => {
  processDialog.isEdit = false;
  processDialog.form = {
    id: 0,
    name: '',
    description: '',
    formID: null,
    nodes: [
      { name: '开始', type: 'start', nextNode: null }
    ],
    version: 1,
    status: 0,
    creatorID: 101,
    creatorName: '当前用户',
    createdAt: new Date(),
    updatedAt: new Date()
  };
  processDialog.visible = true;
};

const handleEditProcess = (row: Process) => {
  processDialog.isEdit = true;
  processDialog.form = JSON.parse(JSON.stringify(row));
  processDialog.visible = true;
  detailDialog.visible = false;
};

const handleViewProcess = (row: Process) => {
  detailDialog.process = row;
  detailDialog.visible = true;
};

const handleCommand = (command: string, row: Process) => {
  switch (command) {
    case 'publish':
      publishProcess(row);
      break;
    case 'unpublish':
      unpublishProcess(row);
      break;
    case 'clone':
      showCloneDialog(row);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

const publishProcess = (process: Process) => {
  const index = processes.value.findIndex(p => p.id === process.id);
  if (index !== -1) {
    const process = processes.value[index];
    if (process) {
      process.status = 1;
      process.updatedAt = new Date();
      message.success(`流程 "${process.name}" 已发布`);
    }
  }
};

const unpublishProcess = (process: Process) => {
  const index = processes.value.findIndex(p => p.id === process.id);
  if (index !== -1) {
    const process = processes.value[index];
    if (process) {
      process.status = 0;
      process.updatedAt = new Date();
      message.success(`流程 "${process.name}" 已取消发布`);
    }
  }
};

const showCloneDialog = (process: Process) => {
  cloneDialog.form.name = `${process.name} 的副本`;
  cloneDialog.form.originalId = process.id;
  cloneDialog.visible = true;
};

const confirmClone = () => {
  const originalProcess = processes.value.find(p => p.id === cloneDialog.form.originalId);
  if (originalProcess) {
    const newId = Math.max(...processes.value.map(p => p.id)) + 1;
    const clonedProcess: Process = {
      ...JSON.parse(JSON.stringify(originalProcess)),
      id: newId,
      name: cloneDialog.form.name,
      status: 0, // 总是草稿
      version: 1,
      createdAt: new Date(),
      updatedAt: new Date()
    };

    processes.value.push(clonedProcess);
    cloneDialog.visible = false;
    message.success(`流程 "${originalProcess.name}" 已克隆为 "${cloneDialog.form.name}"`);
  }
};

const confirmDelete = (process: Process) => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除流程 "${process.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      const index = processes.value.findIndex(p => p.id === process.id);
      if (index !== -1) {
        processes.value.splice(index, 1);
        message.success(`流程 "${process.name}" 已删除`);
      }
    }
  });
};

const addNode = () => {
  processDialog.form.nodes.push({
    name: '',
    type: 'approval',
    approvers: [],
    nextNode: null
  });
};

const removeNode = (index: number) => {
  // 检查是否有其他节点引用了这个要删除的节点
  const hasReferences = processDialog.form.nodes.some(
    (node, nodeIndex) => nodeIndex !== index && node.nextNode === index
  );

  if (hasReferences) {
    message.warning('该节点被其他节点引用，请先修改相关节点的引用关系');
    return;
  }

  processDialog.form.nodes.splice(index, 1);

  // 更新其他节点的引用关系
  processDialog.form.nodes.forEach(node => {
    if (node.nextNode !== null && node.nextNode > index) {
      node.nextNode -= 1;
    }
  });
};

const saveProcess = () => {
  if (processDialog.form.name.trim() === '') {
    message.error('流程名称不能为空');
    return;
  }

  if (processDialog.form.formID === null) {
    message.error('请选择关联表单');
    return;
  }

  if (processDialog.form.nodes.length === 0) {
    message.error('流程至少需要一个节点');
    return;
  }

  // 验证流程节点是否有效
  for (let i = 0; i < processDialog.form.nodes.length; i++) {
    const node = processDialog.form.nodes[i];
    if (!node) continue;

    if (!node.name) {
      message.error(`节点 ${i + 1} 名称不能为空`);
      return;
    }

    if (node.type === 'approval' && (!node.approvers || node.approvers.length === 0)) {
      message.error(`审批节点 "${node.name}" 必须指定处理人`);
      return;
    }

    if (node.type === 'notice' && (!node.notifyUsers || node.notifyUsers.length === 0)) {
      message.error(`通知节点 "${node.name}" 必须指定通知人`);
      return;
    }

    if (node.type === 'condition' && (!node.condition || node.condition.trim() === '')) {
      message.error(`条件节点 "${node.name}" 必须指定条件表达式`);
      return;
    }
  }

  // 确保 formID 不为 null
  const formToSave = {
    ...processDialog.form,
    formID: processDialog.form.formID as number
  };

  if (processDialog.isEdit) {
    // 更新现有流程
    const index = processes.value.findIndex(p => p.id === formToSave.id);
    if (index !== -1) {
      formToSave.updatedAt = new Date();
      processes.value[index] = formToSave;
      message.success(`流程 "${formToSave.name}" 已更新`);
    }
  } else {
    // 创建新流程
    const newId = Math.max(...processes.value.map(p => p.id)) + 1;
    formToSave.id = newId;
    processes.value.push(formToSave);
    message.success(`流程 "${formToSave.name}" 已创建`);
  }
  processDialog.visible = false;
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

const getStatusClass = (status: number) => {
  switch (status) {
    case 0: return 'status-draft';
    case 1: return 'status-published';
    case 2: return 'status-disabled';
    default: return '';
  }
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

const getFormName = (formId: number) => {
  const form = forms.value.find(f => f.id === formId);
  return form ? form.name : '未知表单';
};

const getUserName = (userId: number) => {
  const user = users.value.find(u => u.id === userId);
  return user ? user.name : '未知用户';
};

const getNodeTypeName = (type: string) => {
  const typeMap: Record<string, string> = {
    'start': '开始',
    'approval': '审批',
    'notice': '通知',
    'condition': '条件',
    'end': '结束'
  };
  return typeMap[type] || type;
};

const getNodeName = (nodes: ProcessNode[], index: number | null) => {
  if (index === null) return '无';
  return nodes[index] ? (nodes[index].name || `节点 ${index + 1}`) : '无效节点';
};

// 模拟初始化
onMounted(() => {
  loading.value = true;
  // 模拟API加载
  setTimeout(() => {
    loading.value = false;
  }, 800);
});
</script>

<style scoped>
.process-container {
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

.process-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.process-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
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

.nodes-editor {
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 20px;
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.add-node-button {
  text-align: center;
  margin-top: 16px;
}

.detail-dialog .process-details {
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
}

.process-preview {
  margin-top: 24px;
}

.process-preview h3 {
  margin-bottom: 16px;
  color: #1f2937;
  font-size: 18px;
}

.process-flow-chart {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.process-node {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 4px;
  transition: all 0.3s;
  position: relative;
}

.node-type-start {
  background-color: #e6f7ff;
  border-color: #91d5ff;
}

.node-type-approval {
  background-color: #f6ffed;
  border-color: #b7eb8f;
}

.node-type-notice {
  background-color: #fefce6;
  border-color: #ffe58f;
}

.node-type-condition {
  background-color: #fff7e6;
  border-color: #ffd591;
}

.node-type-end {
  background-color: #f9f0ff;
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
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
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
  font-weight: bold;
  font-size: 16px;
}

.node-content {
  margin-bottom: 12px;
}

.node-approvers,
.node-notify-users {
  display: flex;
  align-items: center;
  gap: 8px;
}

.node-condition {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.condition-expression {
  background-color: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  font-family: monospace;
}

.node-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 8px;
  color: #8c8c8c;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
