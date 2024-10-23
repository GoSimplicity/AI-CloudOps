<template>
  <a-layout style="height: 100vh;">
    <a-layout-sider width="320" theme="light">
      <div style="padding: 24px;">
        <a-tree :field-names="{ title: 'title', key: 'key', children: 'children' }" :tree-data="treeData"
          default-expand-all show-line draggable @select="onSelect" @drop="onDrop" @dragend="onDragEnd" />
        <a-button type="primary" block style="margin-top: 16px" @click="showAddModal">
          <a-icon type="plus" /> 新增节点
          <a-modal v-model:visible="isAddModalVisible" style="z-index: 1000" title="新增节点" @ok="handleAdd" okText="新增"
            cancelText="取消">
            <a-form :model="addForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" :rules="formRules"
              ref="addFormRef">
              <a-form-item label="节点名称" name="title" :rules="[{ required: true, message: '请输入节点名称' }]">
                <a-input v-model:value="addForm.title" placeholder="请输入节点名称" />
              </a-form-item>
              <a-form-item label="level层级" name="level">
                <a-input-number v-model:value=addForm.level />
              </a-form-item>
              <a-form-item label="创建类型" name="isLeaf" :rules="[{ required: true, message: '请选择是否为叶节点' }]">
                <a-select v-model:value="addForm.isLeaf" placeholder="请选择">
                  <a-select-option :value=0>目录</a-select-option>
                  <a-select-option :value=1>叶节点</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="描述" name="description">
                <a-input v-model:value="addForm.description" placeholder="请输入描述" />
              </a-form-item>
              <a-form-item label="父节点" name="pId">
                <a-select v-if="isSelectVisible" v-model:value="addForm.pId" placeholder="请选择父节点">
                  <!-- 默认顶级节点选项，值为 0 -->
                  <a-select-option :key="0" :value="0">顶级节点</a-select-option>
                  <!-- 动态生成的节点选项 -->
                  <a-select-option v-for="node in flatTreeData" :key="node.ID" :value="node.ID">
                    {{ node.title }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-form>
          </a-modal>
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
            <a-table :columns="ecsColumns" :data-source="selectedNode.bindEcs" rowKey="id" :pagination="false"
              size="small" bordered />

            <a-divider orientation="left">ELB 资源详情</a-divider>
            <a-table :columns="elbColumns" :data-source="selectedNode.bindElb" rowKey="id" :pagination="false"
              size="small" bordered />

            <a-divider orientation="left">RDS 资源详情</a-divider>
            <a-table :columns="rdsColumns" :data-source="selectedNode.bindRds" rowKey="id" :pagination="false"
              size="small" bordered />
          </a-card>
        </div>

        <div v-else style="display: flex; justify-content: center; align-items: center; height: 100%;">
          <a-result title="请选择一个节点查看详情" icon="info-circle" status="info" />
        </div>
      </a-layout-content>
    </a-layout>

    <a-modal v-model:visible="isEditModalVisible" title="编辑节点" @ok="handleEdit" okText="保存" cancelText="取消">
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
import type { TreeNode } from '#/api/core/tree';
import { message } from 'ant-design-vue';
import { getAllTreeNodes, createTreeNode } from '#/api';
// 树数据
const treeData = ref<TreeNode[]>([]);
const flatTreeData = ref<TreeNode[]>([]);
const selectedNode = ref<TreeNode | null>(null);
const isSelectVisible = ref(true);
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);

const addForm = reactive({
  title: '',
  description: '',
  pId: 0,
  isLeaf: 0,
  level: 1,
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
  { title: '载均衡类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
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
  getAllTreeNodes().then(response => {
    treeData.value = response;
    flatTreeData.value = [];
    flattenTree(treeData.value, flatTreeData.value);
  }).catch(error => {
    message.error('获取树数据失败');
    console.error(error);
  });
};

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
  console.log('isAddModalVisible:', isAddModalVisible.value);
};


const handleAdd = async () => {
  // 检查节点名称是否为空
  if (!addForm.title) {
    message.error('节点名称不能为空');
    return;
  }

  try {
    // 调用 API 创建新节点
    const response = await createTreeNode({
      title: addForm.title,
      pId: addForm.pId,
      description: addForm.description,
      isLeaf: addForm.isLeaf,
      level: addForm.level,
    });

    console.log('createTreeNode response:', response);

    // 从响应中获取新节点
    const newNode = response.data;

    // 将新节点添加到树数据中
    treeData.value.push(newNode);

    // 提示用户新增成功
    message.success('新增节点成功');

    // 关闭 select 框
    isSelectVisible.value = false;

    // 重置表单数据
    resetForm(addForm);

    // 延迟 1-2 秒后刷新页面
    setTimeout(() => {
      location.reload();  // 刷新页面
    }, 1500);  // 1.5 秒延迟
  } catch (error) {
    // 处理错误
    message.error('新增节点失败');
    console.error(error);
  }
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
