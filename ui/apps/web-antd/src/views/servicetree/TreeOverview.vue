<template>
  <div class="overview-container">
    <a-page-header title="服务树概览" subtitle="查看和管理企业服务树结构" :backIcon="false">
      <template #extra>
        <a-space>
          <a-button type="primary" @click="refreshData">
            <template #icon>
              <ReloadOutlined />
            </template>
            刷新
          </a-button>
          <a-button type="primary" @click="navigateToManagePage">
            <template #icon>
              <SettingOutlined />
            </template>
            节点管理
          </a-button>
        </a-space>
      </template>
    </a-page-header>

    <a-row :gutter="16" class="dashboard-cards">
      <a-col :xs="24" :sm="12" :md="8" :lg="6">
        <a-card hoverable>
          <template #cover>
            <div class="stat-card">
              <AppstoreOutlined class="card-icon" />
              <div class="stat-number">{{ statistics.totalNodes }}</div>
              <div class="stat-label">总节点数</div>
            </div>
          </template>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="8" :lg="6">
        <a-card hoverable>
          <template #cover>
            <div class="stat-card">
              <CloudServerOutlined class="card-icon" />
              <div class="stat-number">{{ statistics.totalResources }}</div>
              <div class="stat-label">资源总数</div>
            </div>
          </template>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="8" :lg="6">
        <a-card hoverable>
          <template #cover>
            <div class="stat-card">
              <TeamOutlined class="card-icon" />
              <div class="stat-number">{{ statistics.totalUsers }}</div>
              <div class="stat-label">管理员数</div>
            </div>
          </template>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="8" :lg="6">
        <a-card hoverable>
          <template #cover>
            <div class="stat-card">
              <ClockCircleOutlined class="card-icon" />
              <div class="stat-number">{{ statistics.lastUpdate }}</div>
              <div class="stat-label">最近更新</div>
            </div>
          </template>
        </a-card>
      </a-col>
    </a-row>

    <div class="tree-visualization">
      <a-card title="服务树结构可视化" :bordered="false" class="tree-card">
        <a-radio-group v-model:value="viewMode" button-style="solid" class="view-selector">
          <a-radio-button value="tree">树形视图</a-radio-button>
          <a-radio-button value="graph">网络视图</a-radio-button>
        </a-radio-group>

        <div class="tree-content">
          <a-spin :spinning="loading">
            <a-tree v-if="viewMode === 'tree'" :tree-data="treeData" :defaultExpandedKeys="['1']"
              :showLine="{ showLeafIcon: false }" @select="onTreeNodeSelect" class="service-tree">
              <template #title="{ title, key }">
                <span class="tree-node-title">
                  {{ title }}
                  <a-tag v-if="getNodeResourceCount(key) > 0" color="blue">
                    {{ getNodeResourceCount(key) }}
                  </a-tag>
                </span>
              </template>
            </a-tree>

            <div v-else-if="viewMode === 'graph'" class="graph-view">
              <!-- 这里可以放置假的网络图 -->
              <div class="graph-placeholder">
                <div class="graph-node central-node">根节点</div>
                <div class="graph-connections">
                  <div v-for="(node, index) in 5" :key="index" class="graph-node child-node" :style="{
                    left: `${Math.cos(index * Math.PI * 2 / 5) * 120 + 150}px`,
                    top: `${Math.sin(index * Math.PI * 2 / 5) * 120 + 150}px`
                  }">
                    子节点 {{ index + 1 }}
                  </div>
                </div>
              </div>
            </div>
          </a-spin>
        </div>
      </a-card>
    </div>

    <a-row :gutter="16" class="node-details-row">
      <a-col :span="12">
        <a-card title="节点详情" :bordered="false" v-if="selectedNode" class="details-card">
          <a-descriptions :column="1" bordered>
            <a-descriptions-item label="节点ID">
              {{ selectedNode.id }}
            </a-descriptions-item>
            <a-descriptions-item label="节点名称">
              {{ selectedNode.name }}
            </a-descriptions-item>
            <a-descriptions-item label="路径">
              {{ selectedNode.path }}
            </a-descriptions-item>
            <a-descriptions-item label="管理员">
              <a-tag v-for="admin in selectedNode.admins" :key="admin" color="blue">
                {{ admin }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">
              {{ selectedNode.createTime }}
            </a-descriptions-item>
            <a-descriptions-item label="描述">
              {{ selectedNode.description || '无' }}
            </a-descriptions-item>
          </a-descriptions>
        </a-card>
        <a-empty v-else description="请选择节点查看详情" />
      </a-col>

      <a-col :span="12">
        <a-card title="绑定资源" :bordered="false" v-if="selectedNode" class="resources-card">
          <a-tabs default-active-key="ecs">
            <a-tab-pane key="ecs" tab="云服务器">
              <a-table :dataSource="selectedNodeResources.ecs" :columns="ecsColumns" :pagination="{ pageSize: 5 }"
                size="small" />
            </a-tab-pane>
            <a-tab-pane key="rds" tab="数据库">
              <a-table :dataSource="selectedNodeResources.rds" :columns="rdsColumns" :pagination="{ pageSize: 5 }"
                size="small" />
            </a-tab-pane>
            <a-tab-pane key="elb" tab="负载均衡">
              <a-table :dataSource="selectedNodeResources.elb" :columns="elbColumns" :pagination="{ pageSize: 5 }"
                size="small" />
            </a-tab-pane>
          </a-tabs>
        </a-card>
        <a-empty v-else description="请选择节点查看资源" />
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import {
  ReloadOutlined,
  SettingOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  TeamOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons-vue';

const router = useRouter();
const loading = ref(false);
const viewMode = ref('tree');
const selectedNode = ref<{
  id: number;
  name: string;
  path: string;
  admins: string[];
  createTime: string;
  description: string;
} | null>(null);

// 统计数据
const statistics = reactive({
  totalNodes: 35,
  totalResources: 187,
  totalUsers: 42,
  lastUpdate: '2小时前',
});

// 树形数据
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

// 节点资源数据
const nodeResources: Record<string, { count: number }> = reactive({
  '1-1-1': { count: 28 },
  '1-1-2': { count: 15 },
  '1-1-3': { count: 45 },
  '1-2-1': { count: 8 },
  '1-2-2': { count: 6 },
  '1-3': { count: 12 },
  '1-4': { count: 7 },
});

// 节点详细信息模拟数据
interface NodeData {
  id: number;
  name: string;
  path: string;
  admins: string[];
  createTime: string;
  description: string;
}

const nodesData: Record<string, NodeData> = {
  '1': {
    id: 1,
    name: '总部',
    path: '/总部',
    admins: ['admin', 'root'],
    createTime: '2023-05-10 10:00:00',
    description: '公司总部节点，所有业务的顶层结构',
  },
  '1-1': {
    id: 2,
    name: '技术部',
    path: '/总部/技术部',
    admins: ['tech_lead', 'cto'],
    createTime: '2023-05-15 14:30:00',
    description: '负责公司所有技术相关工作的部门',
  },
  '1-1-1': {
    id: 3,
    name: '后端组',
    path: '/总部/技术部/后端组',
    admins: ['backend_lead', 'tech_lead'],
    createTime: '2023-06-01 09:15:00',
    description: '负责后端服务开发和维护',
  },
  '1-1-2': {
    id: 4,
    name: '前端组',
    path: '/总部/技术部/前端组',
    admins: ['frontend_lead', 'tech_lead'],
    createTime: '2023-06-01 09:20:00',
    description: '负责前端界面开发和用户交互',
  },
  '1-1-3': {
    id: 5,
    name: '运维组',
    path: '/总部/技术部/运维组',
    admins: ['ops_lead', 'tech_lead'],
    createTime: '2023-06-01 09:25:00',
    description: '负责基础设施和服务运维',
  },
  '1-2': {
    id: 6,
    name: '产品部',
    path: '/总部/产品部',
    admins: ['product_lead', 'cpo'],
    createTime: '2023-05-15 15:00:00',
    description: '负责产品规划和设计',
  },
  '1-2-1': {
    id: 7,
    name: '产品设计组',
    path: '/总部/产品部/产品设计组',
    admins: ['design_lead', 'product_lead'],
    createTime: '2023-06-02 10:30:00',
    description: '负责产品原型和详细设计',
  },
  '1-2-2': {
    id: 8,
    name: '用户体验组',
    path: '/总部/产品部/用户体验组',
    admins: ['ux_lead', 'product_lead'],
    createTime: '2023-06-02 10:35:00',
    description: '负责用户体验研究和改进',
  },
  '1-3': {
    id: 9,
    name: '运营部',
    path: '/总部/运营部',
    admins: ['operation_lead', 'coo'],
    createTime: '2023-05-15 15:30:00',
    description: '负责市场营销和运营',
  },
  '1-4': {
    id: 10,
    name: '财务部',
    path: '/总部/财务部',
    admins: ['finance_lead', 'cfo'],
    createTime: '2023-05-15 16:00:00',
    description: '负责财务管理和预算控制',
  },
};

// 资源表格列定义
const ecsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
];

const rdsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '类型', dataIndex: 'dbType', key: 'dbType' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
];

const elbColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr' },
  { title: '类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
];

// 模拟资源数据
const selectedNodeResources = reactive({
  ecs: [
    { key: '1', instanceName: 'web-server-1', status: '运行中', ipAddr: '10.0.0.1', instanceType: 'ecs.g6.xlarge' },
    { key: '2', instanceName: 'web-server-2', status: '运行中', ipAddr: '10.0.0.2', instanceType: 'ecs.g6.xlarge' },
    { key: '3', instanceName: 'api-server-1', status: '运行中', ipAddr: '10.0.0.3', instanceType: 'ecs.g6.2xlarge' },
  ],
  rds: [
    { key: '1', instanceName: 'main-db', status: '运行中', dbType: 'MySQL', instanceType: 'rds.mysql.s3.large' },
    { key: '2', instanceName: 'read-db', status: '运行中', dbType: 'MySQL', instanceType: 'rds.mysql.s3.large' },
  ],
  elb: [
    { key: '1', instanceName: 'web-lb', status: '运行中', ipAddr: '47.100.123.45', loadBalancerType: '公网' },
    { key: '2', instanceName: 'api-lb', status: '运行中', ipAddr: '10.0.0.100', loadBalancerType: '内网' },
  ],
});

// 获取节点资源数量
const getNodeResourceCount = (key: string): number => {
  return nodeResources[key]?.count || 0;
};

// 树节点选择事件
const onTreeNodeSelect = (selectedKeys: string[]): void => {
  if (selectedKeys.length > 0) {
    const key = selectedKeys[0];
    if (typeof key === 'string') {
      const node = nodesData[key]; // 此处可能为 undefined
      selectedNode.value = node ? { ...node } : null; // 安全赋值
    } else {
      selectedNode.value = null;
    }
  } else {
    selectedNode.value = null;
  }
};

// 刷新数据
const refreshData = () => {
  loading.value = true;
  setTimeout(() => {
    loading.value = false;
  }, 1000);
};

// 导航到节点管理页面
const navigateToManagePage = () => {
  router.push('/tree/manage');
};

onMounted(() => {
  refreshData();
});
</script>

<style scoped lang="scss">
.overview-container {
  padding: 16px;
  min-height: 100vh;

  .dashboard-cards {
    margin-top: 16px;
    margin-bottom: 24px;

    .stat-card {
      text-align: center;
      padding: 24px 0;
      background: linear-gradient(135deg, #1890ff0a 0%, #1890ff1a 100%);

      .card-icon {
        font-size: 36px;
        color: #1890ff;
        margin-bottom: 16px;
      }

      .stat-number {
        font-size: 28px;
        font-weight: 600;
        margin-bottom: 8px;
      }

      .stat-label {
        font-size: 14px;
      }
    }
  }

  .tree-visualization {
    margin-bottom: 24px;

    .tree-card {
      .view-selector {
        margin-bottom: 16px;
      }

      .tree-content {
        min-height: 300px;

        .service-tree {
          margin-top: 16px;
        }

        .tree-node-title {
          display: flex;
          align-items: center;
          gap: 8px;
        }

        .graph-view {
          height: 350px;
          width: 100%;
          border-radius: 4px;
          position: relative;

          .graph-placeholder {
            height: 100%;
            position: relative;

            .graph-node {
              position: absolute;
              padding: 8px 16px;
              border-radius: 20px;
              background: #1890ff;
              color: white;
              font-weight: 500;
              box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
              z-index: 2;
            }

            .central-node {
              top: 150px;
              left: 150px;
              background: #722ed1;
            }

            .child-node {
              transition: all 0.3s;

              &:hover {
                transform: scale(1.1);
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
              }
            }

            .graph-connections {
              &::before {
                content: '';
                position: absolute;
                top: 160px;
                left: 160px;
                width: 240px;
                height: 240px;
                border-radius: 50%;
                z-index: 1;
              }
            }
          }
        }
      }
    }
  }

  .node-details-row {

    .details-card,
    .resources-card {
      height: 100%;
    }
  }
}
</style>
