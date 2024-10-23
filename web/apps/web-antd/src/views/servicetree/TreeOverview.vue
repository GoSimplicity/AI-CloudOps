<template>
  <a-layout style="height: 100vh;">
    <a-layout-sider width="320" theme="light">
      <div style="padding: 24px;">
        <a-tree :field-names="{ title: 'title', key: 'key', children: 'children' }" :tree-data="treeData"
          default-expand-all show-line draggable @select="onSelect" />
        <a-button type="primary" block style="margin-top: 16px" @click="showAddModal">
          <a-icon type="plus" /> 新增节点
          <a-modal v-model:visible="isSelectVisible" title="新增节点" @ok="handleAdd" okText="新增" cancelText="取消">
            <a-form :model="addForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" ref="addFormRef">
              <a-form-item label="节点名称" name="title" :rules="[{ required: true, message: '请输入节点名称' }]">
                <a-input v-model:value="addForm.title" placeholder="请输入节点名称" />
              </a-form-item>
              <a-form-item label="level层级" name="level">
                <a-input-number v-model:value="addForm.level" />
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
                  <a-select-option :key="0" :value="0">顶级节点</a-select-option>
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
                  <a-modal v-model:visible="isEditVisible" title="编辑节点" @ok="handleEdit" okText="保存" cancelText="取消">
                    <a-form :model="editForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" ref="editFormRef">
                      <a-form-item label="节点名称" name="title" :rules="[{ required: true, message: '请输入节点名称' }]">
                        <a-input v-model:value="editForm.title" placeholder="请输入节点名称" />
                      </a-form-item>
                      <a-form-item label="描述" name="description">
                        <a-input v-model:value="editForm.desc" placeholder="请输入描述" />
                      </a-form-item>
                      <a-form-item label="运维负责人" name="ops_admins">
                        <a-select v-model:value="editForm.ops_admins" mode="tags" placeholder="请选择运维负责人">
                          <a-select-option v-for="person in flatTreeData" :key="person.ID" :value="person.ops_admin_users">
                            {{ person.ops_admin_users }}
                          </a-select-option>
                        </a-select>
                      </a-form-item>
                      <a-form-item label="研发负责人" name="rd_admins">
                        <a-select v-model:value="editForm.rd_admins" mode="tags" placeholder="请选择研发负责人">
                          <a-select-option v-for="person in flatTreeData" :key="person.ID" :value="person.ops_admin_users">
                            {{ person.rd_admin_users }}
                          </a-select-option>
                        </a-select>
                      </a-form-item>
                      <a-form-item label="研发工程师" name="rd_members">
                        <a-select v-model:value="editForm.rd_members" mode="tags" placeholder="请选择研发工程师">
                          <a-select-option v-for="person in flatTreeData" :key="person.ID" :value="person.ops_admin_users">
                            {{ person.rd_member_users }}
                          </a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-form>
                  </a-modal>
                </a-button>
              </span>
            </template>
            <a-descriptions bordered column="2">
              <a-descriptions-item label="描述">{{ selectedNode.desc }}</a-descriptions-item>
              <a-descriptions-item label="Level 等级">{{ selectedNode.level }}</a-descriptions-item>
              <a-descriptions-item label="运维负责人">
                <a-tag v-for="person in selectedNode.ops_admins" :key="person.id" color="blue">
                  {{ person.name }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="研发负责人">
                <a-tag v-for="person in selectedNode.rd_admins" :key="person.id" color="green">
                  {{ person.name }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="研发工程师">
                <a-tag v-for="engineer in selectedNode.rd_members" :key="engineer.id" color="purple">
                  {{ engineer.name }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="绑定的 ECS 数量">{{ selectedNode.ecsNum }}</a-descriptions-item>
              <a-descriptions-item label="绑定的 ELB 数量">{{ selectedNode.elbNum }}</a-descriptions-item>
              <a-descriptions-item label="绑定的 RDS 数量">{{ selectedNode.rdsNum }}</a-descriptions-item>
            </a-descriptions>

            <a-divider orientation="left">ECS 资源详情</a-divider>
            <a-table :columns="ecsColumns" :data-source="selectedNode.bind_ecs" rowKey="ID" :pagination="false"
              size="small" bordered />

            <a-divider orientation="left">ELB 资源详情</a-divider>
            <a-table :columns="elbColumns" :data-source="selectedNode.bind_elb" rowKey="ID" :pagination="false"
              size="small" bordered />

            <a-divider orientation="left">RDS 资源详情</a-divider>
            <a-table :columns="rdsColumns" :data-source="selectedNode.bind_rds" rowKey="ID" :pagination="false"
              size="small" bordered />
          </a-card>
        </div>

        <div v-else style="display: flex; justify-content: center; align-items: center; height: 100%;">
          <a-result title="请选择一个节点查看详情" icon="info-circle" status="info" />
        </div>
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import type { TreeNode } from '#/api/core/tree';
import { message } from 'ant-design-vue';
import { getAllTreeNodes, createTreeNode, updateTreeNode } from '#/api';

const treeData = ref<TreeNode[]>([]);
const flatTreeData = ref<TreeNode[]>([]);
const selectedNode = ref<TreeNode | null>(null);
const isSelectVisible = ref(false);
const isEditVisible = ref(false);
const addForm = reactive({
  title: '',
  description: '',
  pId: 0,
  isLeaf: 0,
  level: 1,
});

interface Person {
  id: number;
  name: string;
  realName: string;
  roles: string[];
  userId: number;
  username: string;
}

const editForm = reactive({
  ID: 0,
  title: '',
  desc: '',
  ops_admins: [] as Person[],
  rd_admins: [] as Person[],
  rd_members: [] as Person[],
});

const ecsColumns = [
  { title: 'ID', dataIndex: 'ID', key: 'ID' },
  { title: '操作系统类型', dataIndex: 'osType', key: 'osType' },
  { title: '实例类型', dataIndex: 'instanceType', key: 'instanceType' },
  { title: 'CPU 核数', dataIndex: 'cpu', key: 'cpu' },
  { title: '内存 (GiB)', dataIndex: 'memory', key: 'memory' },
  { title: '磁盘 (GiB)', dataIndex: 'disk', key: 'disk' },
  { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
];

const elbColumns = [
  { title: 'ID', dataIndex: 'ID', key: 'ID' },
  { title: '载均衡类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
  { title: '带宽容量 (Mb)', dataIndex: 'bandwidthCapacity', key: 'bandwidthCapacity' },
  { title: '地址类型', dataIndex: 'addressType', key: 'addressType' },
  { title: 'DNS 名称', dataIndex: 'dnsName', key: 'dnsName' },
];

const rdsColumns = [
  { title: 'ID', dataIndex: 'ID', key: 'ID' },
  { title: '引擎类型', dataIndex: 'engine', key: 'engine' },
  { title: '网络类型', dataIndex: 'dbInstanceNetType', key: 'dbInstanceNetType' },
  { title: '实例规格', dataIndex: 'dbInstanceClass', key: 'dbInstanceClass' },
  { title: '实例类型', dataIndex: 'dbInstanceType', key: 'dbInstanceType' },
  { title: '版本', dataIndex: 'engineVersion', key: 'engineVersion' },
  { title: '状态', dataIndex: 'dbInstanceStatus', key: 'dbInstanceStatus' },
];

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
    const key = keys[0];
    if (key) {
      selectedNode.value = findNodeByKey(treeData.value, key);
    }
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
  isSelectVisible.value = true;
};

const handleAdd = async () => {
  if (!addForm.title) {
    message.error('节点名称不能为空');
    return;
  }

  try {
    const response = await createTreeNode({
      title: addForm.title,
      pId: addForm.pId,
      desc: addForm.description,
      isLeaf: addForm.isLeaf,
      level: addForm.level,
    });

    const newNode = response.data;
    treeData.value.push(newNode);
    message.success('新增节点成功');
    isSelectVisible.value = false;
    resetForm(addForm);
    setTimeout(() => {
      location.reload();
    }, 500);
  } catch (error) {
    message.error('新增节点失败');
    console.error(error);
  }
};

const showEditModal = () => {
  if (selectedNode.value) {
    // 将当前选中的节点数据复制到 editForm 中
    editForm.ID = selectedNode.value.ID;
    editForm.title = selectedNode.value.title;
    editForm.desc = selectedNode.value.desc;
    editForm.ops_admins = selectedNode.value.ops_admins ? [...selectedNode.value.ops_admins] : [];
    editForm.rd_admins = selectedNode.value.rd_admins ? [...selectedNode.value.rd_admins] : [];
    editForm.rd_members = selectedNode.value.rd_members ? [...selectedNode.value.rd_members] : [];
    
    isEditVisible.value = true; // 显示编辑模态框
  }
};

const handleEdit = async () => {
  if (!editForm.title) {
    message.error('节点名称不能为空');
    return;
  }

  try {
    await updateTreeNode({
      ID: editForm.ID,
      title: editForm.title,
      desc: editForm.desc,
      ops_admins: editForm.ops_admins,
      rd_admins: editForm.rd_admins,
      rd_members: editForm.rd_members,
    });

    message.success('编辑节点成功');
    isEditVisible.value = false;
    fetchTreeData();
  } catch (error) {
    message.error('编辑节点失败');
    console.error(error);
  }
};


const resetForm = (form: any) => {
  Object.keys(form).forEach(key => (form[key] = key === 'pId' ? 0 : ''));
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

.a-button .anticon {
  margin-right: 4px;
}

.a-card {
  margin-bottom: 24px;
}
</style>
