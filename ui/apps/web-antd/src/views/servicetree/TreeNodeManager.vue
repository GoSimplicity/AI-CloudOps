<template>
  <div class="tree-manager-container">
    <a-page-header title="服务树节点管理" subtitle="创建、编辑和管理服务树节点" :backIcon="true" @back="goBack">
      <template #extra>
        <a-space>
          <a-button type="primary" @click="showCreateNodeModal">
            <template #icon>
              <PlusOutlined />
            </template>
            创建节点
          </a-button>
          <a-button @click="refreshData">
            <template #icon>
              <ReloadOutlined />
            </template>
            刷新
          </a-button>
        </a-space>
      </template>
    </a-page-header>

    <a-row :gutter="16" class="main-content">
      <!-- 左侧树形结构 -->
      <a-col :span="6">
        <a-card title="服务树结构" :bordered="false" class="tree-card">
          <template #extra>
            <a-dropdown>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="1" @click="expandAll">展开所有</a-menu-item>
                  <a-menu-item key="2" @click="collapseAll">收起所有</a-menu-item>
                </a-menu>
              </template>
              <a-button type="text">
                <SettingOutlined />
              </a-button>
            </a-dropdown>
          </template>

          <a-spin :spinning="loading">
            <a-input-search v-model:value="searchValue" placeholder="搜索节点" style="margin-bottom: 16px"
              @change="onSearchChange" allowClear />

            <a-tree v-model:expandedKeys="expandedKeys" v-model:selectedKeys="selectedKeys"
              :tree-data="filteredTreeData" :showLine="{ showLeafIcon: false }" @select="onSelect">
              <template #title="{ title, key }">
                <span class="tree-node-title">
                  {{ title }}
                  <div class="node-actions">
                    <a-tooltip title="新增子节点">
                      <PlusCircleOutlined @click.stop="showCreateChildNodeModal(key)" />
                    </a-tooltip>
                    <a-tooltip title="编辑节点">
                      <EditOutlined @click.stop="showEditNodeModal(key)" />
                    </a-tooltip>
                    <a-tooltip title="删除节点">
                      <DeleteOutlined @click.stop="confirmDeleteNode(key)" />
                    </a-tooltip>
                  </div>
                </span>
              </template>
            </a-tree>
          </a-spin>
        </a-card>
      </a-col>

      <!-- 右侧详情与管理 -->
      <a-col :span="18">
        <div v-if="selectedNode">
          <a-card :bordered="false" class="detail-card">
            <a-tabs v-model:activeKey="activeTabKey">
              <a-tab-pane key="basicInfo" tab="基本信息">
                <a-descriptions title="节点详情" :column="{ xxl: 3, xl: 3, lg: 3, md: 2, sm: 1, xs: 1 }" bordered>
                  <a-descriptions-item label="节点ID">{{ selectedNode.id }}</a-descriptions-item>
                  <a-descriptions-item label="节点名称">{{ selectedNode.name }}</a-descriptions-item>
                  <a-descriptions-item label="节点路径">{{ selectedNode.path }}</a-descriptions-item>
                  <a-descriptions-item label="父节点">{{ selectedNode.parentName || '无' }}</a-descriptions-item>
                  <a-descriptions-item label="创建时间">{{ selectedNode.createTime }}</a-descriptions-item>
                  <a-descriptions-item label="更新时间">{{ selectedNode.updateTime }}</a-descriptions-item>
                  <a-descriptions-item label="创建者">{{ selectedNode.creator }}</a-descriptions-item>
                  <a-descriptions-item label="子节点数">{{ selectedNode.childCount }}</a-descriptions-item>
                  <a-descriptions-item label="资源数">{{ selectedNode.resourceCount }}</a-descriptions-item>
                  <a-descriptions-item label="描述" :span="3">
                    {{ selectedNode.description || '无描述' }}
                  </a-descriptions-item>
                </a-descriptions>

                <a-divider orientation="left">快捷操作</a-divider>
                <a-space>
                  <a-button type="primary" @click="showEditNodeModal(selectedNode.key)">
                    <template #icon>
                      <EditOutlined />
                    </template>
                    编辑节点
                  </a-button>
                  <a-button @click="showCreateChildNodeModal(selectedNode.key)">
                    <template #icon>
                      <PlusOutlined />
                    </template>
                    添加子节点
                  </a-button>
                  <a-button danger @click="confirmDeleteNode(selectedNode.key)">
                    <template #icon>
                      <DeleteOutlined />
                    </template>
                    删除节点
                  </a-button>
                  <a-button @click="showBindResourceModal">
                    <template #icon>
                      <LinkOutlined />
                    </template>
                    绑定资源
                  </a-button>
                </a-space>
              </a-tab-pane>

              <a-tab-pane key="resources" tab="绑定资源">
                <a-tabs v-model:activeKey="resourceTabKey">
                  <a-tab-pane key="ecs" tab="云服务器 (ECS)">
                    <div class="resource-header">
                      <a-button type="primary" size="small" @click="showBindResourceModal('ecs')">
                        <template #icon>
                          <PlusOutlined />
                        </template>
                        绑定 ECS
                      </a-button>
                    </div>
                    <a-table :dataSource="resourcesData.ecs" :columns="ecsColumns" :pagination="{ pageSize: 10 }"
                      size="middle">
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'action'">
                          <a-space>
                            <a-button size="small" type="link" @click="viewResourceDetail(record, 'ecs')">
                              查看
                            </a-button>
                            <a-button size="small" type="link" danger @click="confirmUnbindResource(record, 'ecs')">
                              解绑
                            </a-button>
                          </a-space>
                        </template>
                      </template>
                    </a-table>
                  </a-tab-pane>

                  <a-tab-pane key="rds" tab="数据库 (RDS)">
                    <div class="resource-header">
                      <a-button type="primary" size="small" @click="showBindResourceModal('rds')">
                        <template #icon>
                          <PlusOutlined />
                        </template>
                        绑定 RDS
                      </a-button>
                    </div>
                    <a-table :dataSource="resourcesData.rds" :columns="rdsColumns" :pagination="{ pageSize: 10 }"
                      size="middle">
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'action'">
                          <a-space>
                            <a-button size="small" type="link" @click="viewResourceDetail(record, 'rds')">
                              查看
                            </a-button>
                            <a-button size="small" type="link" danger @click="confirmUnbindResource(record, 'rds')">
                              解绑
                            </a-button>
                          </a-space>
                        </template>
                      </template>
                    </a-table>
                  </a-tab-pane>

                  <a-tab-pane key="elb" tab="负载均衡 (ELB)">
                    <div class="resource-header">
                      <a-button type="primary" size="small" @click="showBindResourceModal('elb')">
                        <template #icon>
                          <PlusOutlined />
                        </template>
                        绑定 ELB
                      </a-button>
                    </div>
                    <a-table :dataSource="resourcesData.elb" :columns="elbColumns" :pagination="{ pageSize: 10 }"
                      size="middle">
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'action'">
                          <a-space>
                            <a-button size="small" type="link" @click="viewResourceDetail(record, 'elb')">
                              查看
                            </a-button>
                            <a-button size="small" type="link" danger @click="confirmUnbindResource(record, 'elb')">
                              解绑
                            </a-button>
                          </a-space>
                        </template>
                      </template>
                    </a-table>
                  </a-tab-pane>
                </a-tabs>
              </a-tab-pane>

              <a-tab-pane key="members" tab="成员管理">
                <a-tabs v-model:activeKey="memberTabKey">
                  <a-tab-pane key="admins" tab="管理员">
                    <div class="member-header">
                      <a-button type="primary" size="small" @click="showAddMemberModal('admin')">
                        <template #icon>
                          <PlusOutlined />
                        </template>
                        添加管理员
                      </a-button>
                    </div>
                    <a-table :dataSource="membersData.admins" :columns="adminColumns" :pagination="{ pageSize: 10 }"
                      size="middle">
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'action'">
                          <a-button size="small" type="link" danger @click="confirmRemoveMember(record, 'admin')">
                            移除
                          </a-button>
                        </template>
                      </template>
                    </a-table>
                  </a-tab-pane>

                  <a-tab-pane key="members" tab="普通成员">
                    <div class="member-header">
                      <a-button type="primary" size="small" @click="showAddMemberModal('member')">
                        <template #icon>
                          <PlusOutlined />
                        </template>
                        添加成员
                      </a-button>
                    </div>
                    <a-table :dataSource="membersData.members" :columns="memberColumns" :pagination="{ pageSize: 10 }"
                      size="middle">
                      <template #bodyCell="{ column, record }">
                        <template v-if="column.key === 'action'">
                          <a-button size="small" type="link" danger @click="confirmRemoveMember(record, 'member')">
                            移除
                          </a-button>
                        </template>
                      </template>
                    </a-table>
                  </a-tab-pane>
                </a-tabs>
              </a-tab-pane>
            </a-tabs>
          </a-card>
        </div>
        <a-empty v-else description="请选择服务树节点" />
      </a-col>
    </a-row>

    <!-- 创建节点模态框 -->
    <a-modal v-model:visible="createNodeModalVisible"
      :title="isEditMode ? '编辑节点' : currentParentKey ? '添加子节点' : '创建顶级节点'" @ok="handleCreateOrUpdateNode"
      :confirmLoading="confirmLoading" width="600px">
      <a-form :model="nodeForm" :rules="nodeFormRules" ref="nodeFormRef" layout="vertical">
        <a-form-item label="节点名称" name="name">
          <a-input v-model:value="nodeForm.name" placeholder="请输入节点名称" />
        </a-form-item>
        <a-form-item label="父节点" name="parentId" v-if="!isEditMode">
          <a-select v-model:value="nodeForm.parentId" placeholder="请选择父节点" :disabled="!!currentParentKey">
            <a-select-option :value="0">无 (创建顶级节点)</a-select-option>
            <a-select-option v-for="option in parentNodeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="nodeForm.description" placeholder="请输入节点描述" :rows="4" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 绑定资源模态框 -->
    <a-modal v-model:visible="bindResourceModalVisible" title="绑定资源" @ok="handleBindResource"
      :confirmLoading="confirmLoading" width="800px">
      <a-form :model="bindResourceForm" layout="vertical">
        <a-form-item label="资源类型" name="resourceType">
          <a-select v-model:value="bindResourceForm.resourceType" placeholder="请选择资源类型">
            <a-select-option value="ecs">云服务器 (ECS)</a-select-option>
            <a-select-option value="rds">数据库 (RDS)</a-select-option>
            <a-select-option value="elb">负载均衡 (ELB)</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="选择资源" name="resourceIds">
          <a-table :dataSource="availableResources"
            :rowSelection="{ selectedRowKeys: bindResourceForm.resourceIds, onChange: onSelectedResourcesChange }"
            :columns="availableResourceColumns" size="middle" :pagination="{ pageSize: 5 }" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 添加成员模态框 -->
    <a-modal v-model:visible="addMemberModalVisible" :title="memberForm.type === 'admin' ? '添加管理员' : '添加成员'"
      @ok="handleAddMember" :confirmLoading="confirmLoading" width="600px">
      <a-form :model="memberForm" layout="vertical">
        <a-form-item label="选择用户" name="userIds">
          <a-select v-model:value="memberForm.userIds" mode="multiple" placeholder="请选择用户" :options="userOptions"
            style="width: 100%" :filter-option="filterUserOption"></a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 资源详情模态框 -->
    <a-modal v-model:visible="resourceDetailModalVisible" title="资源详情" footer={null} width="800px">
      <a-descriptions bordered :column="1" size="middle">
        <a-descriptions-item v-for="(value, key) in currentResourceDetail" :key="key" :label="formatResourceLabel(key)">
          <template v-if="Array.isArray(value)">
            <a-tag v-for="(item, index) in value" :key="index">{{ item }}</a-tag>
          </template>
          <template v-else>{{ value }}</template>
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import {
  PlusOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  SettingOutlined,
  PlusCircleOutlined,
  LinkOutlined,
} from '@ant-design/icons-vue';
import { message, Modal } from 'ant-design-vue';

const router = useRouter();
const loading = ref(false);
const confirmLoading = ref(false);
const searchValue = ref('');
const expandedKeys = ref<string[]>(['1']);
const selectedKeys = ref<string[]>([]);
const activeTabKey = ref('basicInfo');
const resourceTabKey = ref('ecs');
const memberTabKey = ref('admins');

// 模态框状态
const createNodeModalVisible = ref(false);
const bindResourceModalVisible = ref(false);
const addMemberModalVisible = ref(false);
const resourceDetailModalVisible = ref(false);

// 节点表单状态
const nodeFormRef = ref<any>(null);
const nodeForm = reactive({
  id: 0,
  name: '',
  parentId: 0,
  description: '',
});

const nodeFormRules = {
  name: [{ required: true, message: '请输入节点名称', trigger: 'blur' }],
};

// 资源绑定表单
const bindResourceForm = reactive({
  resourceType: 'ecs',
  resourceIds: [] as string[],
});

// 成员表单
const memberForm = reactive({
  type: 'admin', // 'admin' 或 'member'
  userIds: [] as string[],
});

// 其他状态
const isEditMode = ref(false);
const currentParentKey = ref('');
const currentResourceType = ref('');
const currentResourceDetail = ref({});

// Mock 树形数据
const treeData = ref([
  {
    title: '总部',
    key: '1',
    children: [
      {
        title: '技术部',
        key: '1-1',
        children: [
          {
            title: '后端组',
            key: '1-1-1',
          },
          {
            title: '前端组',
            key: '1-1-2',
          },
          {
            title: '运维组',
            key: '1-1-3',
          },
        ],
      },
      {
        title: '产品部',
        key: '1-2',
        children: [
          {
            title: '产品设计组',
            key: '1-2-1',
          },
          {
            title: '用户体验组',
            key: '1-2-2',
          },
        ],
      },
      {
        title: '运营部',
        key: '1-3',
      },
      {
        title: '财务部',
        key: '1-4',
      },
    ],
  },
]);

// 节点详细信息
const nodesDetails = {
  '1': {
    id: 1,
    key: '1',
    name: '总部',
    path: '/总部',
    parentName: null,
    childCount: 4,
    resourceCount: 15,
    creator: 'admin',
    createTime: '2023-05-10 10:00:00',
    updateTime: '2023-06-15 14:30:00',
    description: '公司总部节点，所有业务的顶层结构',
  },
  '1-1': {
    id: 2,
    key: '1-1',
    name: '技术部',
    path: '/总部/技术部',
    parentName: '总部',
    childCount: 3,
    resourceCount: 42,
    creator: 'admin',
    createTime: '2023-05-15 14:30:00',
    updateTime: '2023-06-15 14:35:00',
    description: '负责公司所有技术相关工作的部门',
  },
  '1-1-1': {
    id: 3,
    key: '1-1-1',
    name: '后端组',
    path: '/总部/技术部/后端组',
    parentName: '技术部',
    childCount: 0,
    resourceCount: 28,
    creator: 'tech_lead',
    createTime: '2023-06-01 09:15:00',
    updateTime: '2023-06-15 14:40:00',
    description: '负责后端服务开发和维护',
  },
  '1-1-2': {
    id: 4,
    key: '1-1-2',
    name: '前端组',
    path: '/总部/技术部/前端组',
    parentName: '技术部',
    childCount: 0,
    resourceCount: 15,
    creator: 'tech_lead',
    createTime: '2023-06-01 09:20:00',
    updateTime: '2023-06-15 14:45:00',
    description: '负责前端界面开发和用户交互',
  },
  '1-1-3': {
    id: 5,
    key: '1-1-3',
    name: '运维组',
    path: '/总部/技术部/运维组',
    parentName: '技术部',
    childCount: 0,
    resourceCount: 45,
    creator: 'tech_lead',
    createTime: '2023-06-01 09:25:00',
    updateTime: '2023-06-15 14:50:00',
    description: '负责基础设施和服务运维',
  },
  '1-2': {
    id: 6,
    key: '1-2',
    name: '产品部',
    path: '/总部/产品部',
    parentName: '总部',
    childCount: 2,
    resourceCount: 14,
    creator: 'admin',
    createTime: '2023-05-15 15:00:00',
    updateTime: '2023-06-15 15:00:00',
    description: '负责产品规划和设计',
  },
  '1-2-1': {
    id: 7,
    key: '1-2-1',
    name: '产品设计组',
    path: '/总部/产品部/产品设计组',
    parentName: '产品部',
    childCount: 0,
    resourceCount: 8,
    creator: 'product_lead',
    createTime: '2023-06-02 10:30:00',
    updateTime: '2023-06-15 15:05:00',
    description: '负责产品原型和详细设计',
  },
  '1-2-2': {
    id: 8,
    key: '1-2-2',
    name: '用户体验组',
    path: '/总部/产品部/用户体验组',
    parentName: '产品部',
    childCount: 0,
    resourceCount: 6,
    creator: 'product_lead',
    createTime: '2023-06-02 10:35:00',
    updateTime: '2023-06-15 15:10:00',
    description: '负责用户体验研究和改进',
  },
  '1-3': {
    id: 9,
    key: '1-3',
    name: '运营部',
    path: '/总部/运营部',
    parentName: '总部',
    childCount: 0,
    resourceCount: 12,
    creator: 'admin',
    createTime: '2023-05-15 15:30:00',
    updateTime: '2023-06-15 15:15:00',
    description: '负责市场营销和运营',
  },
  '1-4': {
    id: 10,
    key: '1-4',
    name: '财务部',
    path: '/总部/财务部',
    parentName: '总部',
    childCount: 0,
    resourceCount: 7,
    creator: 'admin',
    createTime: '2023-05-15 16:00:00',
    updateTime: '2023-06-15 15:20:00',
    description: '负责财务管理和预算控制',
  },
};

// 资源数据模拟
const resourcesData = reactive({
  ecs: [
    { key: '1', instanceName: 'web-server-1', instanceId: 'i-2ze0xvx82ozr4f9j12ab', status: '运行中', ipAddr: '10.0.0.1', instanceType: 'ecs.g6.xlarge', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-15 08:30:00' },
    { key: '2', instanceName: 'web-server-2', instanceId: 'i-2ze0xvx82ozr4f9j12ac', status: '运行中', ipAddr: '10.0.0.2', instanceType: 'ecs.g6.xlarge', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-15 08:35:00' },
    { key: '3', instanceName: 'api-server-1', instanceId: 'i-2ze0xvx82ozr4f9j12ad', status: '运行中', ipAddr: '10.0.0.3', instanceType: 'ecs.g6.2xlarge', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-15 08:40:00' },
  ],
  rds: [
    { key: '1', instanceName: 'main-db', instanceId: 'rm-2zekx8vh5n177d4y1', status: '运行中', dbType: 'MySQL', instanceType: 'rds.mysql.s3.large', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-16 09:30:00' },
    { key: '2', instanceName: 'read-db', instanceId: 'rm-2zekx8vh5n177d4y2', status: '运行中', dbType: 'MySQL', instanceType: 'rds.mysql.s3.large', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-16 09:35:00' },
  ],
  elb: [
    { key: '1', instanceName: 'web-lb', instanceId: 'lb-2zekx8vh5n177d4y1', status: '运行中', ipAddr: '47.100.123.45', loadBalancerType: '公网', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-17 10:30:00' },
    { key: '2', instanceName: 'api-lb', instanceId: 'lb-2zekx8vh5n177d4y2', status: '运行中', ipAddr: '10.0.0.100', loadBalancerType: '内网', provider: 'Alibaba Cloud', regionId: 'cn-beijing', createTime: '2023-01-17 10:35:00' },
  ],
});

// 可绑定的资源模拟数据
const availableResourcesData = {
  ecs: [
    { key: '4', instanceName: 'cache-server-1', instanceId: 'i-2ze0xvx82ozr4f9j12ae', status: '运行中', ipAddr: '10.0.0.4', instanceType: 'ecs.g6.large', provider: 'Alibaba Cloud', regionId: 'cn-beijing' },
    { key: '5', instanceName: 'cache-server-2', instanceId: 'i-2ze0xvx82ozr4f9j12af', status: '运行中', ipAddr: '10.0.0.5', instanceType: 'ecs.g6.large', provider: 'Alibaba Cloud', regionId: 'cn-beijing' },
    { key: '6', instanceName: 'job-server-1', instanceId: 'i-2ze0xvx82ozr4f9j12ag', status: '运行中', ipAddr: '10.0.0.6', instanceType: 'ecs.g6.xlarge', provider: 'Alibaba Cloud', regionId: 'cn-beijing' },
  ],
  rds: [
    { key: '3', instanceName: 'archive-db', instanceId: 'rm-2zekx8vh5n177d4y3', status: '运行中', dbType: 'MySQL', instanceType: 'rds.mysql.s3.medium', provider: 'Alibaba Cloud', regionId: 'cn-beijing' },
    { key: '4', instanceName: 'report-db', instanceId: 'rm-2zekx8vh5n177d4y4', status: '运行中', dbType: 'PostgreSQL', instanceType: 'rds.pg.s3.large', provider: 'Alibaba Cloud', regionId: 'cn-beijing' },
  ],
  elb: [
    { key: '3', instanceName: 'internal-lb', instanceId: 'lb-2zekx8vh5n177d4y3', status: '运行中', ipAddr: '10.0.0.200', loadBalancerType: '内网', provider: 'Alibaba Cloud', regionId: 'cn-beijing' },
  ],
};

// 成员数据模拟
const membersData = reactive({
  admins: [
    { key: '1', userId: 'admin01', username: 'admin01', displayName: '管理员01', email: 'admin01@example.com', department: '技术部', addedTime: '2023-05-15 14:35:00' },
    { key: '2', userId: 'tech_lead', username: 'tech_lead', displayName: '技术负责人', email: 'tech_lead@example.com', department: '技术部', addedTime: '2023-05-15 14:40:00' },
  ],
  members: [
    { key: '1', userId: 'dev01', username: 'dev01', displayName: '开发者01', email: 'dev01@example.com', department: '技术部', addedTime: '2023-06-01 09:30:00' },
    { key: '2', userId: 'dev02', username: 'dev02', displayName: '开发者02', email: 'dev02@example.com', department: '技术部', addedTime: '2023-06-01 09:35:00' },
    { key: '3', userId: 'dev03', username: 'dev03', displayName: '开发者03', email: 'dev03@example.com', department: '技术部', addedTime: '2023-06-01 09:40:00' },
  ],
});

// 资源表格列定义
const ecsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '实例ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
  { title: '操作', key: 'action' },
];

const rdsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '实例ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '类型', dataIndex: 'dbType', key: 'dbType' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
  { title: '操作', key: 'action' },
];

const elbColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '实例ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr' },
  { title: '类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
  { title: '操作', key: 'action' },
];

// 可绑定资源表格列定义
const availableResourceColumns = computed(() => {
  if (bindResourceForm.resourceType === 'ecs') {
    return [
      { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
      { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr' },
      { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
      { title: '状态', dataIndex: 'status', key: 'status' },
    ];
  } else if (bindResourceForm.resourceType === 'rds') {
    return [
      { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
      { title: '类型', dataIndex: 'dbType', key: 'dbType' },
      { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
      { title: '状态', dataIndex: 'status', key: 'status' },
    ];
  } else {
    return [
      { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
      { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr' },
      { title: '类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
      { title: '状态', dataIndex: 'status', key: 'status' },
    ];
  }
});

// 成员表格列定义
const adminColumns = [
  { title: '用户ID', dataIndex: 'userId', key: 'userId' },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '显示名称', dataIndex: 'displayName', key: 'displayName' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '部门', dataIndex: 'department', key: 'department' },
  { title: '添加时间', dataIndex: 'addedTime', key: 'addedTime' },
  { title: '操作', key: 'action' },
];

const memberColumns = [
  { title: '用户ID', dataIndex: 'userId', key: 'userId' },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '显示名称', dataIndex: 'displayName', key: 'displayName' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '部门', dataIndex: 'department', key: 'department' },
  { title: '添加时间', dataIndex: 'addedTime', key: 'addedTime' },
  { title: '操作', key: 'action' },
];

// 用户选项模拟数据
const userOptions = [
  { label: 'admin02 (管理员02)', value: 'admin02' },
  { label: 'dev04 (开发者04)', value: 'dev04' },
  { label: 'dev05 (开发者05)', value: 'dev05' },
  { label: 'ops01 (运维01)', value: 'ops01' },
  { label: 'ops02 (运维02)', value: 'ops02' },
  { label: 'pm01 (产品经理01)', value: 'pm01' },
  { label: 'pm02 (产品经理02)', value: 'pm02' },
];

// 获取父节点选项
const parentNodeOptions = computed(() => {
  const options: Array<{ label: string; value: number }> = [];

  const traverseTree = (nodes: any[], path = '') => {
    for (const node of nodes) {
      const nodePath = path ? `${path} / ${node.title}` : node.title;
      options.push({
        label: nodePath,
        value: parseInt(node.key, 10),
      });

      if (node.children && node.children.length > 0) {
        traverseTree(node.children, nodePath);
      }
    }
  };

  traverseTree(treeData.value);
  return options;
});

// 获取当前选中的节点
const selectedNode = computed(() => {
  if (selectedKeys.value.length > 0 && typeof selectedKeys.value[0] === 'string' && selectedKeys.value[0] in nodesDetails) {
    return nodesDetails[selectedKeys.value[0] as keyof typeof nodesDetails];
  }
  return null;
});

// 获取可用资源列表
const availableResources = computed(() => {
  const resourceType = bindResourceForm.resourceType as keyof typeof availableResourcesData;
  return availableResourcesData[resourceType] || [];
});

// 过滤树数据
const filteredTreeData = computed(() => {
  if (!searchValue.value) {
    return treeData.value;
  }

  const search = searchValue.value.toLowerCase();

  const filterNode = (node: any, path: string[] = []) => {
    const newPath = [...path, node.title];

    if (node.title.toLowerCase().includes(search) || newPath.join('/').toLowerCase().includes(search)) {
      return { ...node };
    }

    if (node.children) {
      const filteredChildren = node.children
        .map((child: any) => filterNode(child, newPath))
        .filter(Boolean);

      if (filteredChildren.length > 0) {
        return {
          ...node,
          children: filteredChildren,
        };
      }
    }

    return null;
  };

  return treeData.value
    .map(node => filterNode(node))
    .filter(Boolean);
});

// 用户选择过滤
const filterUserOption = (input: string, option: { label: string }) => {
  return option.label.toLowerCase().includes(input.toLowerCase());
};

// 根据 key 格式化资源标签
const formatResourceLabel = (key: string) => {
  const labelMap: Record<string, string> = {
    instanceName: '实例名称',
    instanceId: '实例ID',
    status: '状态',
    ipAddr: 'IP地址',
    instanceType: '规格',
    provider: '云服务提供商',
    regionId: '地区',
    createTime: '创建时间',
    dbType: '数据库类型',
    loadBalancerType: '负载均衡类型',
  };
  return labelMap[key] || key;
};

// 事件处理函数
const goBack = () => {
  router.push('/tree/overview');
};

const refreshData = () => {
  loading.value = true;
  setTimeout(() => {
    loading.value = false;
    message.success('数据已刷新');
  }, 1000);
};

const onSearchChange = () => {
  // 如果搜索值不为空，展开所有节点以便查看
  if (searchValue.value) {
    expandAll();
  }
};

const expandAll = () => {
  const keys: string[] = [];

  const traverse = (nodes: any[]) => {
    for (const node of nodes) {
      keys.push(node.key);
      if (node.children) {
        traverse(node.children);
      }
    }
  };

  traverse(treeData.value);
  expandedKeys.value = keys;
};

const collapseAll = () => {
  expandedKeys.value = [];
};

const onSelect = (keys: string[]) => {
  if (keys.length > 0) {
    selectedKeys.value = keys;
  }
};

const showCreateNodeModal = () => {
  isEditMode.value = false;
  currentParentKey.value = '';

  nodeForm.id = 0;
  nodeForm.name = '';
  nodeForm.parentId = 0;
  nodeForm.description = '';

  createNodeModalVisible.value = true;
};

const showCreateChildNodeModal = (parentNodeKey: string) => {
  isEditMode.value = false;
  currentParentKey.value = parentNodeKey;

  const parentId = parseInt(parentNodeKey, 10);

  nodeForm.id = 0;
  nodeForm.name = '';
  nodeForm.parentId = parentId;
  nodeForm.description = '';

  createNodeModalVisible.value = true;
};

const showEditNodeModal = (nodeKey: string) => {
  isEditMode.value = true;
  currentParentKey.value = '';

  if (nodeKey in nodesDetails) {
    const nodeDetail = nodesDetails[nodeKey as keyof typeof nodesDetails];
    if (nodeDetail) {
      nodeForm.id = nodeDetail.id;
      nodeForm.name = nodeDetail.name;
      nodeForm.description = nodeDetail.description || '';
    }
  }

  createNodeModalVisible.value = true;
};

const confirmDeleteNode = (nodeKey: string) => {
  if (!(nodeKey in nodesDetails)) {
    message.error('未找到节点信息');
    return;
  }

  const nodeDetail = nodesDetails[nodeKey as keyof typeof nodesDetails];

  if (!nodeDetail) {
    message.error('未找到节点信息');
    return;
  }

  Modal.confirm({
    title: '确认删除',
    content: `确定要删除节点 "${nodeDetail.name}" 吗？该操作无法撤销，且会影响该节点下的所有资源和子节点。`,
    okText: '确认',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      // 模拟删除操作
      message.success(`节点 "${nodeDetail.name}" 已删除`);
      // 实际应用中这里应该调用API删除节点，并重新加载树数据
    },
  });
};

const handleCreateOrUpdateNode = () => {
  if (nodeFormRef.value) {
    nodeFormRef.value.validate().then(() => {
      confirmLoading.value = true;

      setTimeout(() => {
        confirmLoading.value = false;
        createNodeModalVisible.value = false;

        const actionText = isEditMode.value ? '更新' : '创建';
        message.success(`节点${actionText}成功！`);

        // 实际应用中这里应该调用API创建或更新节点，并重新加载树数据
      }, 1000);
    }).catch((error: any) => {
      console.log('表单验证失败:', error);
    });
  }
};

const showBindResourceModal = (resourceType = 'ecs') => {
  bindResourceForm.resourceType = resourceType;
  bindResourceForm.resourceIds = [];
  bindResourceModalVisible.value = true;
};

const onSelectedResourcesChange = (selectedRowKeys: string[]) => {
  bindResourceForm.resourceIds = selectedRowKeys;
};

const handleBindResource = () => {
  if (bindResourceForm.resourceIds.length === 0) {
    message.warning('请至少选择一个资源');
    return;
  }

  confirmLoading.value = true;

  setTimeout(() => {
    confirmLoading.value = false;
    bindResourceModalVisible.value = false;

    message.success(`成功绑定 ${bindResourceForm.resourceIds.length} 个资源到当前节点`);

    // 实际应用中这里应该调用API绑定资源，并刷新资源列表
  }, 1000);
};

const confirmUnbindResource = (resource: { instanceName: string }, type: string) => {
  Modal.confirm({
    title: '确认解绑',
    content: `确定要解绑资源 "${resource.instanceName}" 吗？`,
    okText: '确认',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      // 模拟解绑操作
      message.success(`资源 "${resource.instanceName}" 已解绑`);
      // 实际应用中这里应该调用API解绑资源，并刷新资源列表
    },
  });
};

const viewResourceDetail = (resource: Record<string, any>, type: string) => {
  currentResourceDetail.value = { ...resource };
  resourceDetailModalVisible.value = true;
};

const showAddMemberModal = (type: string) => {
  memberForm.type = type;
  memberForm.userIds = [];
  addMemberModalVisible.value = true;
};

const handleAddMember = () => {
  if (memberForm.userIds.length === 0) {
    message.warning('请至少选择一个用户');
    return;
  }

  confirmLoading.value = true;

  setTimeout(() => {
    confirmLoading.value = false;
    addMemberModalVisible.value = false;

    const roleText = memberForm.type === 'admin' ? '管理员' : '成员';
    message.success(`成功添加 ${memberForm.userIds.length} 个${roleText}到当前节点`);

    // 实际应用中这里应该调用API添加成员，并刷新成员列表
  }, 1000);
};

const confirmRemoveMember = (member: { displayName: string }, type: string) => {
  const roleText = type === 'admin' ? '管理员' : '成员';

  Modal.confirm({
    title: '确认移除',
    content: `确定要移除${roleText} "${member.displayName}" 吗？`,
    okText: '确认',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      // 模拟移除操作
      message.success(`${roleText} "${member.displayName}" 已移除`);
      // 实际应用中这里应该调用API移除成员，并刷新成员列表
    },
  });
};

onMounted(() => {
  refreshData();
});

// 监听搜索值变化，更新展开的节点
watch(searchValue, (newVal) => {
  if (newVal) {
    // 如果搜索值不为空，可能需要展开特定的节点
    // 这里可以实现更复杂的逻辑
  }
});
</script>

<style scoped lang="scss">
.tree-manager-container {
  padding: 16px;
  min-height: 100vh;

  .main-content {
    margin-top: 16px;
  }

  .tree-card {
    height: calc(100vh - 130px);
    overflow: auto;

    .tree-node-title {
      display: flex;
      align-items: center;
      width: 100%;
      justify-content: space-between;

      .node-actions {
        visibility: hidden;
        display: flex;
        gap: 8px;
      }

      &:hover .node-actions {
        visibility: visible;
      }
    }
  }

  .detail-card {
    margin-bottom: 24px;
  }

  .resource-header,
  .member-header {
    margin-bottom: 16px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
