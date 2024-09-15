<script setup lang="ts">
import { ref } from 'vue';

// 模拟树状数据
const treeData = ref([
  {
    childNodeCount: 3,
    description: '这是业务系统A的描述信息。',
    devEngineers: ['工程师A', '工程师B'],
    devLeads: ['王五'],
    ecsCount: 5,
    ecsDetails: {
      bandwidth: 200,
      cpuCores: 20,
      disk: 1000,
      memory: 64,
    },
    elbCount: 2,
    key: '0-0',
    leafNodeCount: 2,
    level: 1,
    opsLeads: ['张三', '李四'],
    rdsCount: 3,
    title: '业务系统A',
    children: [
      {
        title: '子系统A1',
        key: '0-0-0',
        description: '这是子系统A1的描述信息。',
        level: 2,
        opsLeads: ['张三'],
        devLeads: ['王五'],
        devEngineers: ['工程师A'],
        childNodeCount: 1,
        leafNodeCount: 1,
        ecsCount: 3,
        elbCount: 1,
        rdsCount: 1,
        ecsDetails: {
          bandwidth: 100,
          cpuCores: 10,
          disk: 500,
          memory: 32,
        },
        isLeaf: true,
      },
      {
        title: '子系统A2',
        key: '0-0-1',
        description: '这是子系统A2的描述信息。',
        level: 2,
        opsLeads: ['李四'],
        devLeads: ['王五'],
        devEngineers: ['工程师B'],
        childNodeCount: 0,
        leafNodeCount: 0,
        ecsCount: 2,
        elbCount: 1,
        rdsCount: 1,
        ecsDetails: {
          bandwidth: 50,
          cpuCores: 5,
          disk: 250,
          memory: 16,
        },
        isLeaf: true,
      },
    ],
  },
  {
    childNodeCount: 2,
    description: '这是业务系统B的描述信息。',
    devEngineers: ['工程师B'],
    devLeads: ['王五'],
    ecsCount: 6,
    ecsDetails: {
      bandwidth: 150,
      cpuCores: 15,
      disk: 750,
      memory: 48,
    },
    elbCount: 3,
    key: '0-1',
    leafNodeCount: 2,
    level: 1,
    opsLeads: ['李四'],
    rdsCount: 2,
    title: '业务系统B',
  },
]);

// 用于存储当前选中的节点详情
const selectedNode = ref(null);

// 树节点点击事件处理
const onSelect = (keys: string[]) => {
  if (keys.length > 0) {
    const findNode = (data: any[], key: string) => {
      for (const node of data) {
        if (node.key === key) {
          return node;
        }
        if (node.children) {
          const result = findNode(node.children, key);
          if (result) return result;
        }
      }
      return null;
    };
    selectedNode.value = findNode(treeData.value, keys[0]);
  }
};
</script>

<template>
  <div style="display: flex">
    <!-- 服务树 -->
    <div style="width: 300px; margin-right: 24px">
      <a-tree
        :field-names="{ title: 'title', key: 'key', children: 'children' }"
        :tree-data="treeData"
        default-expand-all
        show-line
        @select="onSelect"
      />
    </div>

    <!-- 节点详情 -->
    <div v-if="selectedNode" style="flex: 1">
      <h2>{{ selectedNode.title }} 详情</h2>
      <a-descriptions bordered column="2">
        <a-descriptions-item label="描述">
          {{ selectedNode.description }}
        </a-descriptions-item>
        <a-descriptions-item label="Level 等级">
          {{ selectedNode.level }}
        </a-descriptions-item>
        <a-descriptions-item label="运维负责人">
          <ul>
            <li v-for="person in selectedNode.opsLeads" :key="person">
              {{ person }}
            </li>
          </ul>
        </a-descriptions-item>
        <a-descriptions-item label="研发负责人">
          <ul>
            <li v-for="person in selectedNode.devLeads" :key="person">
              {{ person }}
            </li>
          </ul>
        </a-descriptions-item>
        <a-descriptions-item label="研发工程师">
          <ul>
            <li v-for="engineer in selectedNode.devEngineers" :key="engineer">
              {{ engineer }}
            </li>
          </ul>
        </a-descriptions-item>
        <a-descriptions-item label="子节点数量">
          {{ selectedNode.childNodeCount }}
        </a-descriptions-item>
        <a-descriptions-item label="叶子节点数量">
          {{ selectedNode.leafNodeCount }}
        </a-descriptions-item>
        <a-descriptions-item label="绑定的 ECS 数量">
          {{ selectedNode.ecsCount }}
        </a-descriptions-item>
        <a-descriptions-item label="绑定的 ELB 数量">
          {{ selectedNode.elbCount }}
        </a-descriptions-item>
        <a-descriptions-item label="绑定的 RDS 数量">
          {{ selectedNode.rdsCount }}
        </a-descriptions-item>
      </a-descriptions>

      <h3 style="margin-top: 24px">ECS 资源详情</h3>
      <a-descriptions bordered column="2">
        <a-descriptions-item label="CPU 总核数">
          {{ selectedNode.ecsDetails.cpuCores }}
        </a-descriptions-item>
        <a-descriptions-item label="内存总 GB">
          {{ selectedNode.ecsDetails.memory }}
        </a-descriptions-item>
        <a-descriptions-item label="本地磁盘总 GB">
          {{ selectedNode.ecsDetails.disk }}
        </a-descriptions-item>
        <a-descriptions-item label="带宽包上限">
          {{ selectedNode.ecsDetails.bandwidth }}
        </a-descriptions-item>
      </a-descriptions>
    </div>

    <!-- 未选中任何节点时的提示 -->
    <div v-else style="flex: 1; text-align: center">
      <p>请选择一个节点查看详情。</p>
    </div>
  </div>
</template>

<style scoped>
h2 {
  margin-bottom: 16px;
}

h3 {
  margin-top: 24px;
  margin-bottom: 16px;
}
</style>
