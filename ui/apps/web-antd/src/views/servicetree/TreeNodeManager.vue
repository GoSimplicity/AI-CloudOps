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
                <a-descriptions title="节点详情" :column="3" bordered>
                  <a-descriptions-item label="节点ID">{{ selectedNode.id }}</a-descriptions-item>
                  <a-descriptions-item label="节点名称">{{ selectedNode.name }}</a-descriptions-item>
                  <a-descriptions-item label="层级">{{ selectedNode.level }}</a-descriptions-item>
                  <a-descriptions-item label="父节点">{{ selectedNode.parentName || '无' }}</a-descriptions-item>
                  <a-descriptions-item label="创建时间">{{ selectedNode.createdAt }}</a-descriptions-item>
                  <a-descriptions-item label="更新时间">{{ selectedNode.updatedAt }}</a-descriptions-item>
                  <a-descriptions-item label="创建者">{{ selectedNode.creatorId }}</a-descriptions-item>
                  <a-descriptions-item label="子节点数">{{ selectedNode.childCount }}</a-descriptions-item>
                  <a-descriptions-item label="资源数">{{ selectedNode.resourceCount }}</a-descriptions-item>
                  <a-descriptions-item label="状态">{{ selectedNode.status }}</a-descriptions-item>
                  <a-descriptions-item label="叶子节点">{{ selectedNode.isLeaf ? '是' : '否' }}</a-descriptions-item>
                  <a-descriptions-item label="描述" :span="3">
                    {{ selectedNode.description || '无描述' }}
                  </a-descriptions-item>
                </a-descriptions>

                <a-divider orientation="left">快捷操作</a-divider>
                <a-space>
                  <a-button type="primary" @click="showEditNodeModal(String(selectedNode.id))">
                    <template #icon>
                      <EditOutlined />
                    </template>
                    编辑节点
                  </a-button>
                  <a-button @click="showCreateChildNodeModal(String(selectedNode.id))">
                    <template #icon>
                      <PlusOutlined />
                    </template>
                    添加子节点
                  </a-button>
                  <a-button danger @click="confirmDeleteNode(String(selectedNode.id))">
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
                    <a-table :dataSource="selectedNode.adminUsers" :columns="adminColumns" :pagination="{ pageSize: 10 }"
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
                    <a-table :dataSource="selectedNode.memberUsers" :columns="memberColumns" :pagination="{ pageSize: 10 }"
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
    <a-modal v-model:open="createNodeModalVisible"
      :title="isEditMode ? '编辑节点' : currentParentId ? '添加子节点' : '创建顶级节点'" @ok="handleCreateOrUpdateNode"
      :confirmLoading="confirmLoading" width="600px">
      <a-form :model="nodeForm" :rules="nodeFormRules" ref="nodeFormRef" layout="vertical">
        <a-form-item label="节点名称" name="name">
          <a-input v-model:value="nodeForm.name" placeholder="请输入节点名称" />
        </a-form-item>
        <a-form-item label="父节点" name="parentId" v-if="!isEditMode">
          <a-select v-model:value="nodeForm.parentId" placeholder="请选择父节点" :disabled="!!currentParentId">
            <a-select-option :value="0">无 (创建顶级节点)</a-select-option>
            <a-select-option v-for="option in parentNodeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="nodeForm.description" placeholder="请输入节点描述" :rows="4" />
        </a-form-item>
        <a-form-item label="节点类型" name="isLeaf">
          <a-radio-group v-model:value="nodeForm.isLeaf">
            <a-radio :value="false">目录节点</a-radio>
            <a-radio :value="true">叶子节点</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="状态" name="status">
          <a-select v-model:value="nodeForm.status" placeholder="请选择状态">
            <a-select-option value="active">激活</a-select-option>
            <a-select-option value="inactive">未激活</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 绑定资源模态框 -->
    <a-modal v-model:open="bindResourceModalVisible" title="绑定资源" @ok="handleBindResource"
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
    <a-modal v-model:open="addMemberModalVisible" :title="memberForm.type === 'admin' ? '添加管理员' : '添加成员'"
      @ok="handleAddMember" :confirmLoading="confirmLoading" width="600px">
      <a-form :model="memberForm" layout="vertical">
        <a-form-item label="选择用户" name="userId">
          <a-select v-model:value="memberForm.userId" placeholder="请选择用户" :options="userOptions"
            style="width: 100%" :filter-option="filterUserOption"></a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 资源详情模态框 -->
    <a-modal v-model:open="resourceDetailModalVisible" title="资源详情" footer={null} width="800px">
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
import { 
  getTreeList, 
  getNodeDetail, 
  getTreeStatistics, 
  createNode, 
  updateNode, 
  deleteNode, 
  addNodeMember, 
  removeNodeMember,
} from '#/api'; 

import type {
  TreeNodeListReq,
  TreeNodeCreateReq,
  TreeNodeUpdateReq,
  TreeNodeMemberReq,
  TreeNode
} from '#/api'; 

const router = useRouter();
const loading = ref(false);
const confirmLoading = ref(false);
const searchValue = ref('');
const expandedKeys = ref<string[]>([]);
const selectedKeys = ref<string[]>([]);
const activeTabKey = ref('basicInfo');
const resourceTabKey = ref('ecs');
const memberTabKey = ref('admins');

// 模态框状态
const createNodeModalVisible = ref(false);
const bindResourceModalVisible = ref(false);
const addMemberModalVisible = ref(false);
const resourceDetailModalVisible = ref(false);

// 数据状态
const treeData = ref<any[]>([]);
const nodeDetails = ref<Record<string, TreeNode>>({});
const currentNodeDetail = ref<TreeNode | null>(null);
const treeStatistics = ref<any>(null);

// 节点表单状态
const nodeFormRef = ref<any>(null);
const nodeForm = reactive<TreeNodeCreateReq & { id: number }>({
  id: 0,
  name: '',
  parentId: 0,
  creatorId: 1, // 默认值，实际应用中应该从用户会话获取
  description: '',
  isLeaf: false,
  status: 'active',
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
const memberForm = reactive<TreeNodeMemberReq & { type: string }>({
  nodeId: 0,
  userId: 0,
  type: 'admin', // 'admin' 或 'member'
});

// 其他状态
const isEditMode = ref(false);
const currentParentId = ref<number | null>(null);
const currentResourceDetail = ref({});

// 资源数据
const resourcesData = reactive({
  ecs: [],
  rds: [],
  elb: [],
});

// 父节点选项
const parentNodeOptions = ref<{ label: string; value: number }[]>([]);

// 用户选项
const userOptions = ref<{ label: string; value: number }[]>([]);

// 可用资源选项
const availableResources = ref([]);

// 表格列定义
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

// 选中的节点
const selectedNode = computed(() => {
  if (selectedKeys.value.length > 0) {
    const key = selectedKeys.value[0];
    if (key !== undefined) {
      const id = parseInt(key.toString());
      return nodeDetails.value[id] || null;
    }
  }
  return null;
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

// 事件处理函数
const goBack = () => {
  router.push('/tree/overview');
};

// 修复树形数据加载函数
const loadTreeData = async () => {
  loading.value = true;
  try {
    const req: TreeNodeListReq = {};
    const res = await getTreeList(req);
    
    // 构建节点详细信息缓存
    const processNode = (node: any) => {
      nodeDetails.value[node.id] = node;
      if (node.children && node.children.length > 0) {
        node.children.forEach(processNode);
      }
    };
    
    res.forEach(processNode);
    
    // 正确处理树状结构，确保递归处理所有层级的节点
    const transformNode = (node: any) => {
      const result = {
        key: node.id.toString(),
        title: node.name,
        isLeaf: node.isLeaf,
        children: node.children && node.children.length > 0 
          ? node.children.map(transformNode) 
          : undefined
      };
      return result;
    };
    
    treeData.value = res.map(transformNode);
    
    // 更新父节点选项
    updateParentNodeOptions(res);
  } catch (error) {
    console.error('加载树形数据失败:', error);
    message.error('加载树形数据失败');
  } finally {
    loading.value = false;
  }
};

// 更新父节点选项
const updateParentNodeOptions = (nodes: any[]) => {
  parentNodeOptions.value = nodes
    .filter(node => !node.isLeaf) // 只有非叶节点才能作为父节点
    .map(node => ({
      label: node.name,
      value: node.id,
    }));
};

// 获取节点详情
const loadNodeDetail = async (nodeId: number) => {
  // 确保nodeId是有效的正整数
  if (!nodeId || nodeId <= 0) {
    console.warn('无效的节点ID:', nodeId);
    return null;
  }
  
  try {
    const res = await getNodeDetail(nodeId);
    nodeDetails.value[nodeId] = res;
    currentNodeDetail.value = res;
    return res;
  } catch (error) {
    console.error('获取节点详情失败:', error);
    message.error('获取节点详情失败');
    return null;
  }
};

// 加载统计数据
const loadStatistics = async () => {
  try {
    const res = await getTreeStatistics();
    treeStatistics.value = res;
  } catch (error) {
    console.error('获取统计数据失败:', error);
  }
};

const refreshData = () => {
  loadTreeData();
  loadStatistics();
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
const onSelect = async (keys: string[]) => {
  if (keys.length > 0 && keys[0]) {
    const nodeId = parseInt(keys[0].toString());
    // 确保nodeId是有效的正整数
    if (nodeId > 0) {
      await loadNodeDetail(nodeId);
    } else {
      console.warn('选择了无效的节点ID:', nodeId);
    }
  }
};

const showCreateNodeModal = () => {
  isEditMode.value = false;
  currentParentId.value = null;

  // 重置表单
  nodeForm.id = 0;
  nodeForm.name = '';
  nodeForm.parentId = 0;
  nodeForm.description = '';
  nodeForm.isLeaf = false;
  nodeForm.status = 'active';

  createNodeModalVisible.value = true;
};

const showCreateChildNodeModal = (parentNodeKey: string) => {
  isEditMode.value = false;
  const parentId = parseInt(parentNodeKey.toString());
  currentParentId.value = parentId;

  // 重置表单
  nodeForm.id = 0;
  nodeForm.name = '';
  nodeForm.parentId = parentId;
  nodeForm.description = '';
  nodeForm.isLeaf = false;
  nodeForm.status = 'active';

  createNodeModalVisible.value = true;
};

const showEditNodeModal = async (nodeKey: string) => {
  isEditMode.value = true;
  currentParentId.value = null;

  const nodeId = parseInt(nodeKey.toString());
  const nodeDetail = await loadNodeDetail(nodeId);
  
  if (nodeDetail) {
    nodeForm.id = nodeDetail.id;
    nodeForm.name = nodeDetail.name;
    nodeForm.parentId = nodeDetail.parentId;
    nodeForm.description = nodeDetail.description;
    nodeForm.isLeaf = nodeDetail.isLeaf;
    nodeForm.status = nodeDetail.status;
  }

  createNodeModalVisible.value = true;
};

const confirmDeleteNode = (nodeKey: string) => {
  const nodeId = parseInt(nodeKey.toString());
  const nodeDetail = nodeDetails.value[nodeId];

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
    async onOk() {
      try {
        await deleteNode(nodeId);
        message.success(`节点 "${nodeDetail.name}" 已删除`);
        refreshData();
      } catch (error) {
        console.error('删除节点失败:', error);
        message.error('删除节点失败');
      }
    },
  });
};

const handleCreateOrUpdateNode = () => {
  if (nodeFormRef.value) {
    nodeFormRef.value.validate().then(async () => {
      confirmLoading.value = true;

      try {
        if (isEditMode.value) {
          // 更新节点
          const updateReq: TreeNodeUpdateReq = {
            id: nodeForm.id,
            name: nodeForm.name,
            parentId: nodeForm.parentId,
            description: nodeForm.description,
            isLeaf: nodeForm.isLeaf,
            status: nodeForm.status,
          };
          
          await updateNode(updateReq);
          message.success('节点更新成功！');
        } else {
          // 创建节点
          const createReq: TreeNodeCreateReq = {
            name: nodeForm.name,
            parentId: nodeForm.parentId,
            creatorId: nodeForm.creatorId,
            description: nodeForm.description,
            isLeaf: nodeForm.isLeaf,
            status: nodeForm.status,
          };
          
          await createNode(createReq);
          message.success('节点创建成功！');
        }
        
        // 刷新数据
        refreshData();
        createNodeModalVisible.value = false;
      } catch (error) {
        console.error('节点操作失败:', error);
        message.error('节点操作失败');
      } finally {
        confirmLoading.value = false;
      }
    }).catch((error: any) => {
      console.log('表单验证失败:', error);
    });
  }
};

const showBindResourceModal = (resourceType = 'ecs') => {
  bindResourceForm.resourceType = resourceType;
  bindResourceForm.resourceIds = [];
  // 这里需要调用API获取可用资源列表
  // loadAvailableResources(resourceType);
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

  // 实际应用中这里应该调用API绑定资源
  // bindResourcesToNode(selectedNode.value.id, bindResourceForm.resourceType, bindResourceForm.resourceIds);

  setTimeout(() => {
    confirmLoading.value = false;
    bindResourceModalVisible.value = false;
    message.success(`成功绑定 ${bindResourceForm.resourceIds.length} 个资源到当前节点`);
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
      // 实际应用中这里应该调用API解绑资源
      // unbindResourceFromNode(selectedNode.value.id, type, resource.id);
      message.success(`资源 "${resource.instanceName}" 已解绑`);
    },
  });
};

const viewResourceDetail = (resource: Record<string, any>, type: string) => {
  currentResourceDetail.value = { ...resource };
  resourceDetailModalVisible.value = true;
};

const showAddMemberModal = (type: string) => {
  if (!selectedNode.value) {
    message.warning('请先选择节点');
    return;
  }
  
  memberForm.type = type;
  memberForm.nodeId = selectedNode.value.id;
  memberForm.userId = 0;
  addMemberModalVisible.value = true;
  
  // 在实际应用中这里需要加载可选用户列表
  // loadAvailableUsers(type);
};

const handleAddMember = async () => {
  if (!memberForm.userId) {
    message.warning('请选择用户');
    return;
  }

  confirmLoading.value = true;

  try {
    await addNodeMember(memberForm);
    message.success(`成功添加${memberForm.type === 'admin' ? '管理员' : '成员'}`);
    // 刷新节点详情
    if (selectedNode.value) {
      await loadNodeDetail(selectedNode.value.id);
    }
    addMemberModalVisible.value = false;
  } catch (error) {
    console.error('添加成员失败:', error);
    message.error('添加成员失败');
  } finally {
    confirmLoading.value = false;
  }
};

const confirmRemoveMember = async (username: string, type: string) => {
  if (!selectedNode.value) return;
  
  const roleText = type === 'admin' ? '管理员' : '成员';

  Modal.confirm({
    title: '确认移除',
    content: `确定要移除${roleText} "${username}" 吗？`,
    okText: '确认',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        const req: TreeNodeMemberReq = {
          nodeId: selectedNode.value!.id,
          userId: 0, // 这里需要获取用户ID
          type: type
        };
        
        await removeNodeMember(req);
        message.success(`${roleText} "${username}" 已移除`);
        
        // 刷新节点详情
        if (selectedNode.value) {
          await loadNodeDetail(selectedNode.value.id);
        }
      } catch (error) {
        console.error('移除成员失败:', error);
        message.error('移除成员失败');
      }
    },
  });
};

// 格式化资源标签
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

onMounted(() => {
  refreshData();
});

// 监听搜索值变化
watch(searchValue, (newVal) => {
  if (newVal) {
  }
});
</script>

<style scoped lang="scss">
.tree-manager-container {
  padding: 12px;
  min-height: 100vh;

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