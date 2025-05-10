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
              <div class="stat-number">{{ statistics.totalAdmins }}</div>
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
              <div class="stat-number">{{ statistics.activeNodes }}</div>
              <div class="stat-label">活跃节点</div>
            </div>
          </template>
        </a-card>
      </a-col>
    </a-row>

    <div class="tree-visualization">
      <a-row :gutter="16">
        <a-col :span="12">
          <a-card title="树形视图" :bordered="false" class="tree-card">
            <div class="tree-content">
              <a-spin :spinning="loading">
                <a-tree :tree-data="treeData" :defaultExpandedKeys="['1']"
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
              </a-spin>
            </div>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="网络视图" :bordered="false" class="graph-card">
            <div class="graph-content">
              <a-spin :spinning="loading">
                <div class="graph-view">
                  <div ref="chartContainer" style="width: 100%; height: 350px;"></div>
                  <a-empty v-if="treeData.length === 0" description="暂无树形数据" />
                </div>
              </a-spin>
            </div>
          </a-card>
        </a-col>
      </a-row>
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
            <a-descriptions-item label="父节点">
              {{ selectedNode.parentName || '无' }}
            </a-descriptions-item>
            <a-descriptions-item label="层级">
              {{ selectedNode.level }}
            </a-descriptions-item>
            <a-descriptions-item label="管理员">
              <a-tag v-for="admin in selectedNode.adminUsers" :key="admin" color="blue">
                {{ admin }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="成员">
              <a-tag v-for="member in selectedNode.memberUsers" :key="member" color="green">
                {{ member }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">
              {{ selectedNode.createdAt }}
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
import { ref, reactive, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import {
  ReloadOutlined,
  SettingOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  TeamOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons-vue';
import { getTreeStatistics, getTreeList, getNodeDetail } from '#/api/core/tree';
import type { TreeNodeListReq, TreeStatisticsResp, TreeNodeDetailResp } from '#/api/core/tree';
import { message } from 'ant-design-vue';
import * as echarts from 'echarts';

const router = useRouter();
const loading = ref(false);
const selectedNode = ref<TreeNodeDetailResp | null>(null);
const chartContainer = ref<HTMLElement | null>(null);
let chart: echarts.ECharts | null = null;

// 统计数据
const statistics = reactive<TreeStatisticsResp>({
  totalNodes: 0,
  totalResources: 0,
  totalAdmins: 0,
  totalMembers: 0,
  activeNodes: 0,
  inactiveNodes: 0
});

// 树形数据
const treeData = ref<any[]>([]);

// 节点资源数据映射
const nodeResources = reactive<Record<string, { count: number }>>({});

// 资源表格列定义
const ecsColumns = [
  { title: '实例名称', dataIndex: 'name', key: 'name' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'IP地址', dataIndex: 'ip', key: 'ip' },
  { title: '规格', dataIndex: 'type', key: 'type' },
];

const rdsColumns = [
  { title: '实例名称', dataIndex: 'name', key: 'name' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '类型', dataIndex: 'dbType', key: 'dbType' },
  { title: '规格', dataIndex: 'type', key: 'type' },
];

const elbColumns = [
  { title: '实例名称', dataIndex: 'name', key: 'name' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: 'IP地址', dataIndex: 'ip', key: 'ip' },
  { title: '类型', dataIndex: 'loadBalancerType', key: 'loadBalancerType' },
];

// 选中节点的资源
const selectedNodeResources = reactive({
  ecs: [],
  rds: [],
  elb: [],
});

// 获取节点资源数量
const getNodeResourceCount = (key: number): number => {
  return nodeResources[key]?.count || 0;
};

// 初始化ECharts图表
const initChart = () => {
  if (chartContainer.value) {
    chart = echarts.init(chartContainer.value);
    updateChart();
  }
};

// 更新图表数据
const updateChart = () => {
  if (!chart || treeData.value.length === 0) return;

  // 准备节点数据
  const nodes: any[] = [];
  const links: any[] = [];
  
  // 创建实际节点关系的映射，用于检查连线
  const nodeRelations = new Map();
  
  // 将树形数据中的父子关系存储到映射中
  const buildRelationsMap = (node: any) => {
    if (node.children && node.children.length > 0) {
      node.children.forEach((child: any) => {
        // 记录父子关系: 子节点ID -> 父节点ID
        nodeRelations.set(child.key.toString(), node.key.toString());
        buildRelationsMap(child);
      });
    }
  };
  
  // 为每个根节点建立关系映射
  treeData.value.forEach((rootNode) => {
    buildRelationsMap(rootNode);
  });

  // 递归处理树节点及其关系
  const processNode = (node: any) => {
    // 添加当前节点
    nodes.push({
      name: node.title,
      id: node.key.toString(),
      value: getNodeResourceCount(node.key),
      symbolSize: Math.max(30, 40 + (getNodeResourceCount(node.key) * 2)),
      itemStyle: {
        color: nodeRelations.has(node.key.toString()) ? '#1890ff' : '#722ed1'
      },
      label: {
        show: true,
        position: 'inside',
        formatter: (params: any) => {
          return params.data.name;
        }
      }
    });
    
    // 如果节点存在于关系映射中，添加连接关系
    if (nodeRelations.has(node.key.toString())) {
      const parentId = nodeRelations.get(node.key.toString());
      links.push({
        source: parentId,
        target: node.key.toString(),
        lineStyle: {
          width: 2,
          curveness: 0.2
        }
      });
    }
    
    // 递归处理子节点
    if (node.children && node.children.length > 0) {
      node.children.forEach((child: any) => {
        processNode(child);
      });
    }
  };
  
  // 处理所有顶级节点及其子节点
  treeData.value.forEach((rootNode) => {
    processNode(rootNode);
  });

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}'
    },
    animationDurationUpdate: 1500,
    animationEasingUpdate: 'quinticInOut' as const, 
    series: [
      {
        type: 'graph',
        layout: 'force',
        data: nodes,
        links: links,
        roam: true,
        label: {
          show: true,
          position: 'inside',
          color: '#fff',
          fontWeight: 'bold'
        },
        lineStyle: {
          color: 'source',
          curveness: 0.3
        },
        emphasis: {
          focus: 'adjacency',
          lineStyle: {
            width: 4
          }
        },
        force: {
          repulsion: 300,
          edgeLength: 150
        }
      }
    ]
  };

  chart.setOption(option);
};

// 监听窗口大小变化，调整图表大小
window.addEventListener('resize', () => {
  if (chart) {
    chart.resize();
  }
});

// 监听树数据变化，更新图表
watch(treeData, () => {
  if (chart) {
    updateChart();
  }
}, { deep: true });

// 树节点选择事件
const onTreeNodeSelect = async (selectedKeys: number[]): Promise<void> => {
  if (selectedKeys.length > 0) {
    const key = selectedKeys[0];
    try {
      loading.value = true;
      // 获取节点详情
      const nodeDetailRes = await getNodeDetail(Number(key));
      if (nodeDetailRes) {
        selectedNode.value = nodeDetailRes;
      } else {
        selectedNode.value = null;
        message.error('获取节点详情失败');
      }
    } catch (error) {
      console.error('获取节点数据失败:', error);
      message.error('获取节点数据失败');
      selectedNode.value = null;
    } finally {
      loading.value = false;
    }
  } else {
    selectedNode.value = null;
  }
};

// 刷新数据
const refreshData = async () => {
  loading.value = true;
  try {
    // 获取统计数据
    const statsRes = await getTreeStatistics();
    if (statsRes) {
      // 直接使用返回的数据对象，不需要检查code
      Object.assign(statistics, statsRes);
    } else {
      message.error('获取统计数据失败：返回格式不正确');
    }

    // 获取树节点数据
    const listReq: TreeNodeListReq = {}; // 可以添加筛选条件
    const treeRes = await getTreeList(listReq);
    if (treeRes && Array.isArray(treeRes)) {
      // 将后端返回的树状结构转换为前端所需格式
      treeData.value = treeRes.map((node: any) => {
        // 递归处理子节点
        if (node.children && Array.isArray(node.children)) {
          node.children = processTreeNodes(node.children);
        }
        
        // 记录资源数量
        if (node.resourceCount && node.resourceCount > 0) {
          nodeResources[node.id] = { count: node.resourceCount };
        }
        
        return {
          key: node.id,
          title: node.name,
          ...node
        };
      });
      
      // 更新图表
      setTimeout(() => {
        updateChart();
      }, 100);
    } else if (treeRes.code !== 0) {
      message.error(`获取服务树数据失败: ${treeRes.message || '未知错误'}`);
    } else {
      message.error('获取服务树数据失败: 返回数据格式不正确');
    }
    
    // 递归处理树节点的辅助函数
    function processTreeNodes(nodes: any[]): any[] {
      return nodes.map((node: any) => {
        if (node.children && Array.isArray(node.children)) {
          node.children = processTreeNodes(node.children);
        }
        
        // 记录资源数量
        if (node.resourceCount && node.resourceCount > 0) {
          nodeResources[node.id] = { count: node.resourceCount };
        }
        
        return {
          key: node.id,
          title: node.name,
          ...node
        };
      });
    }
  } catch (error) {
    console.error('加载数据失败:', error);
    message.error('加载数据失败，请稍后重试');
  } finally {
    loading.value = false;
  }
};

// 导航到节点管理页面
const navigateToManagePage = () => {
  router.push('/tree_node_manager');
};

onMounted(() => {
  refreshData();
  initChart();
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

    .tree-card, .graph-card {
      height: 400px;
      overflow: auto;

      .tree-content, .graph-content {
        height: 100%;

        .service-tree {
          margin-top: 16px;
        }

        .tree-node-title {
          display: flex;
          align-items: center;
          gap: 8px;
        }
      }
    }

    .graph-view {
      height: 350px;
      width: 100%;
      border-radius: 4px;
      position: relative;
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