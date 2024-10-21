<template>
  <a-layout style="height: 100vh;">
    <a-layout-sider width="320" theme="light">
      <div style="padding: 24px;">
        <a-tree
          :field-names="{ title: 'title', key: 'key', children: 'children' }"
          :tree-data="treeData"
          default-expand-all
          show-line
          draggable
          @select="onSelect"
          @drop="onDrop"
          @dragend="onDragEnd"
        />
        <a-button type="primary" block style="margin-top: 16px" @click="showAddModal">
          <a-icon type="plus" /> 新增节点
        </a-button>
      </div>
    </a-layout-sider>

    <a-layout>
      <a-layout-content style="padding: 24px">
        <div v-if="selectedNode">
          <a-card title="节点详情" bordered>
            <template #extra>
              <span>
                <a-button type="primary" size="small" @click="showEditModal">
                  <a-icon type="edit" /> 编辑节点
                </a-button>
                <a-button type="danger" size="small" style="margin-left: 8px" @click="deleteNode">
                  <a-icon type="delete" /> 删除节点
                </a-button>
              </span>
            </template>
            <a-descriptions bordered column="2">
              <a-descriptions-item label="描述">{{ selectedNode.description }}</a-descriptions-item>
              <a-descriptions-item label="Level 等级">{{ selectedNode.level }}</a-descriptions-item>
              <a-descriptions-item label="运维负责人">
                <a-tag v-for="person in selectedNode.opsAdmins" :key="person.id" color="blue">
                  {{ person.name }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="研发负责人">
                <a-tag v-for="person in selectedNode.rdAdmins" :key="person.id" color="green">
                  {{ person.name }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="研发工程师">
                <a-tag v-for="engineer in selectedNode.rdMembers" :key="engineer.id" color="purple">
                  {{ engineer.name }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="绑定的 ECS 数量">{{ selectedNode.ecsNum }}</a-descriptions-item>
              <a-descriptions-item label="绑定的 ELB 数量">{{ selectedNode.elbNum }}</a-descriptions-item>
              <a-descriptions-item label="绑定的 RDS 数量">{{ selectedNode.rdsNum }}</a-descriptions-item>
            </a-descriptions>

            <a-divider orientation="left">ECS 资源详情</a-divider>
            <a-table
              :columns="ecsColumns"
              :data-source="selectedNode.bindEcs"
              rowKey="id"
              :pagination="false"
              size="small"
              bordered
            />

            <a-divider orientation="left">ELB 资源详情</a-divider>
            <a-table
              :columns="elbColumns"
              :data-source="selectedNode.bindElb"
              rowKey="id"
              :pagination="false"
              size="small"
              bordered
            />

            <a-divider orientation="left">RDS 资源详情</a-divider>
            <a-table
              :columns="rdsColumns"
              :data-source="selectedNode.bindRds"
              rowKey="id"
              :pagination="false"
              size="small"
              bordered
            />
          </a-card>
        </div>

        <div v-else style="display: flex; justify-content: center; align-items: center; height: 100%;">
          <a-result title="请选择一个节点查看详情" icon="info-circle" status="info" />
        </div>
      </a-layout-content>
    </a-layout>

    <a-modal
      v-model:visible="isAddModalVisible"
      title="新增节点"
      @ok="handleAdd"
      @cancel="handleCancel"
      okText="新增"
      cancelText="取消"
    >
      <a-form :model="addForm" label-col="{ span: 6 }" wrapper-col="{ span: 16 }">
        <a-form-item label="节点名称" :rules="[{ required: true, message: '请输入节点名称' }]">
          <a-input v-model:value="addForm.title" placeholder="请输入节点名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-input v-model:value="addForm.description" placeholder="请输入描述" />
        </a-form-item>
        <a-form-item label="父节点">
          <a-select v-model:value="addForm.pId" placeholder="请选择父节点">
            <a-select-option v-for="node in flatTreeData" :key="node.id" :value="node.id">
              {{ node.title }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="isEditModalVisible"
      title="编辑节点"
      @ok="handleEdit"
      @cancel="handleCancel"
      okText="保存"
      cancelText="取消"
    >
      <a-form :model="editForm" label-col="{ span: 6 }" wrapper-col="{ span: 16 }">
        <a-form-item label="节点名称" :rules="[{ required: true, message: '请输入节点名称' }]">
          <a-input v-model:value="editForm.title" placeholder="请输入节点名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-input v-model:value="editForm.description" placeholder="请输入描述" />
        </a-form-item>
      </a-form>
    </a-modal>
  </a-layout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import type { TreeNode } from './types'; // 确保类型定义文件路径正确
import { message } from 'ant-design-vue';

// 树数据
const treeData = ref<TreeNode[]>([]);
const flatTreeData = ref<TreeNode[]>([]);
const selectedNode = ref<TreeNode | null>(null);
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);

const addForm = reactive({
  title: '',
  description: '',
  pId: 0,
});

const editForm = reactive({
  id: 0,
  title: '',
  description: '',
});

const ecsColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '操作系统类型', dataIndex: 'osType', key: 'osType' },
  { title: '实例类型', dataIndex: 'instanceType', key: 'instanceType' },
  { title: 'CPU 核数', dataIndex: 'cpu', key: 'cpu' },
  { title: '内存 (GiB)', dataIndex: 'memory', key: 'memory' },
  { title: '磁盘 (GiB)', dataIndex: 'disk', key: 'disk' },
  { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
];

const elbColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '负载均衡类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
  { title: '带宽容量 (Mb)', dataIndex: 'bandwidthCapacity', key: 'bandwidthCapacity' },
  { title: '地址类型', dataIndex: 'addressType', key: 'addressType' },
  { title: 'DNS 名称', dataIndex: 'dnsName', key: 'dnsName' },
];

const rdsColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '引擎类型', dataIndex: 'engine', key: 'engine' },
  { title: '网络类型', dataIndex: 'dbInstanceNetType', key: 'dbInstanceNetType' },
  { title: '实例规格', dataIndex: 'dbInstanceClass', key: 'dbInstanceClass' },
  { title: '实例类型', dataIndex: 'dbInstanceType', key: 'dbInstanceType' },
  { title: '版本', dataIndex: 'engineVersion', key: 'engineVersion' },
  { title: '状态', dataIndex: 'dbInstanceStatus', key: 'dbInstanceStatus' },
];

// 初始化树数据
const fetchTreeData = () => {
  treeData.value = mockData;
  flatTreeData.value = [];
  flattenTree(treeData.value, flatTreeData.value);
};

// Mock 数据
const mockData: TreeNode[] = [
  {
    id: 1,
    title: '系统 A',
    pId: 0,
    key: '1',
    description: '这是系统 A 的描述',
    level: 1,
    opsAdmins: [{ id: 1, name: '张三' }],
    rdAdmins: [{ id: 2, name: '李四' }],
    rdMembers: [{ id: 3, name: '王五' }],
    ecsNum: 2,
    elbNum: 3, // 更新为3个 ELB
    rdsNum: 3, // 更新为3个 RDS
    bindEcs: [
      { id: 101, osType: 'Linux', instanceType: 't2.medium', cpu: 2, memory: 4, disk: 50, hostname: 'ecs-101' },
      { id: 102, osType: 'Windows', instanceType: 't2.large', cpu: 4, memory: 8, disk: 100, hostname: 'ecs-102' },
    ],
    bindElb: [
      { id: 201, loadBalancerType: '公网型', bandwidthCapacity: 100, addressType: '公网', dnsName: 'elb-201.example.com' },
      { id: 202, loadBalancerType: '内部型', bandwidthCapacity: 50, addressType: '内部', dnsName: 'elb-202.internal.example.com' },
      { id: 203, loadBalancerType: '混合型', bandwidthCapacity: 150, addressType: '公网/内部', dnsName: 'elb-203.example.com' },
    ],
    bindRds: [
      { id: 301, engine: 'MySQL', dbInstanceNetType: 'Internet', dbInstanceClass: 'db.m1.small', dbInstanceType: '主实例', engineVersion: '5.7', dbInstanceStatus: 'Running' },
      { id: 302, engine: 'PostgreSQL', dbInstanceNetType: 'VPC', dbInstanceClass: 'db.m1.medium', dbInstanceType: '从实例', engineVersion: '12', dbInstanceStatus: 'Stopped' },
      { id: 303, engine: 'MongoDB', dbInstanceNetType: 'Internet', dbInstanceClass: 'db.m2.large', dbInstanceType: '主实例', engineVersion: '4.4', dbInstanceStatus: 'Running' },
    ],
    children: [
      {
        id: 2,
        title: '子系统 A1',
        pId: 1,
        key: '2',
        description: '这是子系统 A1 的描述',
        level: 2,
        opsAdmins: [{ id: 1, name: '张三' }],
        rdAdmins: [{ id: 2, name: '李四' }],
        rdMembers: [{ id: 4, name: '赵六' }],
        ecsNum: 1,
        elbNum: 1, // 更新为1个 ELB
        rdsNum: 1, // 更新为1个 RDS
        bindEcs: [
          { id: 103, osType: 'Linux', instanceType: 't2.small', cpu: 1, memory: 2, disk: 20, hostname: 'ecs-103' },
        ],
        bindElb: [
          { id: 204, loadBalancerType: '公网型', bandwidthCapacity: 80, addressType: '公网', dnsName: 'elb-204.example.com' },
        ],
        bindRds: [
          { id: 304, engine: 'SQL Server', dbInstanceNetType: 'VPC', dbInstanceClass: 'db.m3.medium', dbInstanceType: '主实例', engineVersion: '2019', dbInstanceStatus: 'Running' },
        ],
        children: [],
      },
    ],
  },
];

const flattenTree = (nodes: TreeNode[], flatList: TreeNode[]) => {
  nodes.forEach(node => {
    flatList.push(node);
    if (node.children && node.children.length > 0) {
      flattenTree(node.children, flatList);
    }
  });
};

const onSelect = (keys: string[]) => {
  if (keys.length > 0) {
    selectedNode.value = findNodeByKey(treeData.value, keys[0]);
  }
};

const findNodeByKey = (data: TreeNode[], key: string): TreeNode | null => {
  for (const node of data) {
    if (node.key === key) return node;
    if (node.children) {
      const result = findNodeByKey(node.children, key);
      if (result) return result;
    }
  }
  return null;
};

const showAddModal = () => {
  isAddModalVisible.value = true;
};

const handleAdd = () => {
  if (!addForm.title) {
    message.error('节点名称不能为空');
    return;
  }

  const newId = flatTreeData.value.length > 0 ? Math.max(...flatTreeData.value.map(node => node.id)) + 1 : 1;
  const newNode: TreeNode = {
    ...addForm,
    id: newId,
    key: `${newId}`,
    nodePath: '',
    children: [],
  };

  if (addForm.pId === 0) {
    newNode.nodePath = newNode.title;
    treeData.value.push(newNode);
  } else {
    const parentNode = flatTreeData.value.find(node => node.id === addForm.pId);
    if (parentNode) {
      newNode.nodePath = `${parentNode.nodePath} > ${newNode.title}`;
      parentNode.children = parentNode.children || [];
      parentNode.children.push(newNode);
      parentNode.isLeaf = false;
    }
  }

  updateTreeData();
  message.success('新增节点成功');
  isAddModalVisible.value = false;
  resetForm(addForm);
};

const resetForm = (form: any) => {
  Object.keys(form).forEach(key => (form[key] = key === 'pId' ? 0 : ''));
};

const showEditModal = () => {
  if (selectedNode.value) {
    Object.assign(editForm, selectedNode.value);
    isEditModalVisible.value = true;
  }
};

const handleEdit = () => {
  if (!editForm.title) {
    message.error('节点名称不能为空');
    return;
  }

  const node = flatTreeData.value.find(n => n.id === editForm.id);
  if (node) {
    Object.assign(node, editForm);
    node.nodePath = node.pId === 0 ? node.title : `${findNodeByKey(treeData.value, node.pId)?.nodePath} > ${node.title}`;

    if (node.children && node.children.length > 0) {
      updateChildPaths(node);
    }

    updateTreeData();
    message.success('编辑节点成功');
    isEditModalVisible.value = false;
  }
};

const updateChildPaths = (node: TreeNode) => {
  node.children.forEach(child => {
    child.nodePath = `${node.nodePath} > ${child.title}`;
    if (child.children && child.children.length > 0) {
      updateChildPaths(child);
    }
  });
};

const updateTreeData = () => {
  treeData.value = [...treeData.value];
  flatTreeData.value = [];
  flattenTree(treeData.value, flatTreeData.value);
};

const deleteNode = () => {
  if (selectedNode.value) {
    const deleteKey = selectedNode.value.key;
    treeData.value = removeNodeByKey(treeData.value, deleteKey);
    message.success('删除节点成功');
    updateTreeData();
    selectedNode.value = null;
  }
};

const removeNodeByKey = (nodes: TreeNode[], key: string): TreeNode[] => {
  return nodes
    .filter(node => node.key !== key)
    .map(node => ({
      ...node,
      children: node.children ? removeNodeByKey(node.children, key) : [],
    }));
};

onMounted(fetchTreeData);
</script>


<style scoped>
h2 {
  margin-bottom: 16px;
  font-size: 24px;
  color: #1890ff;
}

h3 {
  margin-top: 24px;
  margin-bottom: 16px;
  font-size: 20px;
  color: #595959;
}

ul {
  padding-left: 0;
  list-style: none;
  margin: 0;
}

li {
  margin-bottom: 4px;
}

.a-button .anticon {
  margin-right: 4px;
}

.a-card {
  margin-bottom: 24px;
}
</style>
