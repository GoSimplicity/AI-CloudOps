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
              <div class="stat-number">{{ statistics?.totalNodes || 0 }}</div>
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
              <div class="stat-number">{{ statistics?.totalResources || 0 }}</div>
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
              <div class="stat-number">{{ statistics?.totalAdmins || 0 }}</div>
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
              <div class="stat-number">{{ statistics?.activeNodes || 0 }}</div>
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
                <a-tree 
                  :tree-data="treeData" 
                  :defaultExpandedKeys="defaultExpandedKeys"
                  :showLine="{ showLeafIcon: false }" 
                  @select="onTreeNodeSelect" 
                  class="service-tree"
                >
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
            <a-descriptions-item label="状态">
              <a-tag :color="selectedNode.status === 'active' ? 'green' : 'red'">
                {{ selectedNode.status === 'active' ? '活跃' : '非活跃' }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="节点类型">
              <a-tag :color="selectedNode.isLeaf ? 'blue' : 'orange'">
                {{ selectedNode.isLeaf ? '叶子节点' : '目录节点' }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="管理员数量">
              {{ selectedNode.adminUsers?.length || 0 }}
            </a-descriptions-item>
            <a-descriptions-item label="成员数量">
              {{ selectedNode.memberUsers?.length || 0 }}
            </a-descriptions-item>
            <a-descriptions-item label="子节点数">
              {{ selectedNode.childCount || 0 }}
            </a-descriptions-item>
            <a-descriptions-item label="资源数">
              {{ selectedNode.resourceCount || 0 }}
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">
              {{ formatDateTime(selectedNode.createdAt) }}
            </a-descriptions-item>
            <a-descriptions-item label="更新时间">
              {{ formatDateTime(selectedNode.updatedAt) }}
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
          <a-spin :spinning="resourceLoading">
            <a-table 
              :dataSource="nodeResources" 
              :columns="resourceColumns" 
              :pagination="{ pageSize: 8, size: 'small' }"
              size="small"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'resourceStatus'">
                  <a-tag :color="getResourceStatusColor(record.resourceStatus)">
                    {{ record.resourceStatus }}
                  </a-tag>
                </template>
                <template v-if="column.key === 'resourceCreateTime'">
                  {{ formatDateTime(record.resourceCreateTime) }}
                </template>
              </template>
            </a-table>
          </a-spin>
        </a-card>
        <a-empty v-else description="请选择节点查看资源" />
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, nextTick } from 'vue';
import { useRouter } from 'vue-router';
import {
  ReloadOutlined,
  SettingOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  TeamOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons-vue';
import { message } from 'ant-design-vue';
import * as echarts from 'echarts';
import { 
  getTreeList,
  getNodeDetail,
  getTreeStatistics,
  getNodeResources,
  type TreeNodeDetail,
  type TreeNodeListItem,
  type TreeStatistics,
  type TreeNodeResource,
  type GetTreeListParams,
} from '#/api/core/tree_node';

const router = useRouter();
const loading = ref(false);
const resourceLoading = ref(false);
const selectedNode = ref<TreeNodeDetail | null>(null);
const chartContainer = ref<HTMLElement | null>(null);
let chart: echarts.ECharts | null = null;

// 统计数据
const statistics = ref<TreeStatistics | null>(null);

// 树形数据
const treeData = ref<any[]>([]);
const defaultExpandedKeys = ref<string[]>([]);

// 节点详情缓存
const nodeDetails = ref<Record<string, TreeNodeDetail>>({});

// 节点资源数据
const nodeResources = ref<TreeNodeResource[]>([]);

// 资源表格列定义
const resourceColumns = [
  { title: '资源名称', dataIndex: 'resourceName', key: 'resourceName' },
  { title: '资源类型', dataIndex: 'resourceType', key: 'resourceType' },
  { title: '状态', dataIndex: 'resourceStatus', key: 'resourceStatus' },
  { title: '创建时间', dataIndex: 'resourceCreateTime', key: 'resourceCreateTime' },
];

// 工具函数
const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString('zh-CN');
};

const getResourceStatusColor = (status: string) => {
  const colorMap: Record<string, string> = {
    'running': 'green',
    'stopped': 'red',
    'starting': 'orange',
    'stopping': 'orange',
    'active': 'green',
    'inactive': 'red',
  };
  return colorMap[status] || 'default';
};

// 获取节点资源数量
const getNodeResourceCount = (key: string | number): number => {
  const nodeId = parseInt(key.toString());
  return nodeDetails.value[nodeId]?.resourceCount || 0;
};

// 修复后的数据加载函数
const loadTreeData = async () => {
  try {
    const params: GetTreeListParams = {};
    const response = await getTreeList(params);
    
    // 检查响应数据结构
    const data = response.data || response;
    const items = data.items || data;
    
    if (!Array.isArray(items)) {
      console.error('API返回的数据格式不正确:', response);
      message.error('数据格式错误');
      return;
    }
    
    // 处理树节点数据并缓存详情
    const processNode = (node: TreeNodeListItem) => {
      nodeDetails.value[node.id] = {
        id: node.id,
        name: node.name,
        parentId: node.parentId,
        level: node.level,
        description: '',
        creatorId: node.creatorId,
        status: node.status,
        isLeaf: node.isLeaf,
        createdAt: node.created_at,
        updatedAt: node.updated_at,
        creatorName: '',
        parentName: '',
        childCount: node.children?.length || 0,
        adminUsers: [],
        memberUsers: [],
        resourceCount: 0,
      };
      
      if (node.children && node.children.length > 0) {
        node.children.forEach(processNode);
      }
    };
    
    items.forEach(processNode);
    
    // 构建树形结构
    const transformNode = (node: TreeNodeListItem): any => ({
      key: node.id.toString(),
      title: node.name,
      isLeaf: node.isLeaf,
      children: node.children && node.children.length > 0 
        ? node.children.map(transformNode) 
        : undefined
    });
    
    treeData.value = items.map(transformNode);
    
    // 设置默认展开的键
    if (treeData.value.length > 0) {
      defaultExpandedKeys.value = [treeData.value[0].key];
    }
    
    console.log('树形数据加载成功:', treeData.value);
  } catch (error) {
    console.error('加载树形数据失败:', error);
    message.error('加载树形数据失败');
  }
};

const loadStatistics = async () => {
  try {
    const res = await getTreeStatistics();
    statistics.value = res;
  } catch (error) {
    console.error('获取统计数据失败:', error);
    message.error('获取统计数据失败');
  }
};

const loadNodeResources = async (nodeId: number) => {
  if (!nodeId) return;
  
  resourceLoading.value = true;
  try {
    const res = await getNodeResources(nodeId);
    nodeResources.value = res;
  } catch (error) {
    console.error('获取节点资源失败:', error);
    message.error('获取节点资源失败');
  } finally {
    resourceLoading.value = false;
  }
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

  const nodes: any[] = [];
  const links: any[] = [];
  
  // 创建节点关系映射
  const nodeRelations = new Map();
  
  const buildRelationsMap = (node: any, parentKey?: string) => {
    if (parentKey) {
      nodeRelations.set(node.key, parentKey);
    }
    
    if (node.children && node.children.length > 0) {
      node.children.forEach((child: any) => {
        buildRelationsMap(child, node.key);
      });
    }
  };
  
  // 建立关系映射
  treeData.value.forEach((rootNode) => {
    buildRelationsMap(rootNode);
  });

  // 递归处理树节点
  const processNode = (node: any) => {
    const resourceCount = getNodeResourceCount(node.key);
    
    nodes.push({
      name: node.title,
      id: node.key,
      value: resourceCount,
      symbolSize: Math.max(30, 30 + (resourceCount * 2)),
      itemStyle: {
        color: node.isLeaf ? '#52c41a' : '#1890ff'
      },
      label: {
        show: true,
        position: 'inside',
        color: '#fff',
        fontWeight: 'bold',
        fontSize: 12
      }
    });
    
    // 添加父子连接关系
    if (nodeRelations.has(node.key)) {
      const parentKey = nodeRelations.get(node.key);
      links.push({
        source: parentKey,
        target: node.key,
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
  
  // 处理所有节点
  treeData.value.forEach((rootNode) => {
    processNode(rootNode);
  });

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        return `${params.data.name}<br/>资源数量: ${params.data.value}`;
      }
    },
    animationDurationUpdate: 1500,
    animationEasingUpdate: 'quinticInOut',
    series: [
      {
        type: 'graph',
        layout: 'force',
        data: nodes,
        links: links,
        roam: true,
        focusNodeAdjacency: true,
        itemStyle: {
          borderColor: '#fff',
          borderWidth: 1,
          shadowBlur: 10,
          shadowColor: 'rgba(0, 0, 0, 0.3)'
        },
        label: {
          show: true,
          position: 'inside'
        },
        lineStyle: {
          color: 'source',
          curveness: 0.3,
          opacity: 0.9
        },
        emphasis: {
          focus: 'adjacency',
          lineStyle: {
            width: 4
          }
        },
        force: {
          repulsion: 400,
          edgeLength: [100, 200],
          gravity: 0.1
        }
      }
    ]
  };

  chart.setOption(option as any);
};

// 树节点选择事件
const onTreeNodeSelect = async (selectedKeys: string[]) => {
  if (selectedKeys.length > 0) {
    const nodeId = parseInt(selectedKeys[0] || '0');
    
    if (nodeId > 0) {
      loading.value = true;
      try {
        // 从缓存获取或加载节点详情
        let nodeDetail = nodeDetails.value[nodeId];
        if (!nodeDetail) {
          nodeDetail = await getNodeDetail(nodeId);
          if (nodeDetail) {
            nodeDetails.value[nodeId] = nodeDetail;
          }
        }
        
        if (nodeDetail) {
          selectedNode.value = nodeDetail;
          // 加载节点资源
          await loadNodeResources(nodeId);
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
    }
  } else {
    selectedNode.value = null;
    nodeResources.value = [];
  }
};

// 刷新数据
const refreshData = async () => {
  loading.value = true;
  try {
    await Promise.all([
      loadTreeData(),
      loadStatistics(),
    ]);
    
    // 初始化或更新图表
    await nextTick();
    if (chart) {
      updateChart();
    } else {
      initChart();
    }
  } catch (error) {
    console.error('刷新数据失败:', error);
    message.error('刷新数据失败');
  } finally {
    loading.value = false;
  }
};

// 导航到节点管理页面
const navigateToManagePage = () => {
  router.push('/tree_node_manager');
};

// 监听窗口大小变化
const handleResize = () => {
  if (chart) {
    chart.resize();
  }
};

// 监听树数据变化，更新图表
watch(treeData, () => {
  if (chart) {
    updateChart();
  }
}, { deep: true });

onMounted(() => {
  refreshData();
  window.addEventListener('resize', handleResize);
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
        display: block;
      }

      .stat-number {
        font-size: 28px;
        font-weight: 600;
        margin-bottom: 8px;
        color: #262626;
      }

      .stat-label {
        font-size: 14px;
        color: #8c8c8c;
      }
    }
  }

  .tree-visualization {
    margin-bottom: 24px;

    .tree-card, .graph-card {
      height: 420px;

      .tree-content, .graph-content {
        height: 100%;

        .service-tree {
          margin-top: 16px;
          height: calc(100% - 16px);
          overflow: auto;
        }

        .tree-node-title {
          display: flex;
          align-items: center;
          gap: 8px;
          justify-content: space-between;
          width: 100%;
        }
      }
    }

    .graph-view {
      height: 350px;
      width: 100%;
      border-radius: 4px;
      position: relative;
      border: 1px solid #f0f0f0;
    }
  }

  .node-details-row {
    .details-card,
    .resources-card {
      min-height: 400px;
    }
  }
}

// 响应式调整
@media (max-width: 768px) {
  .overview-container {
    padding: 8px;
    
    .dashboard-cards {
      .stat-card {
        padding: 16px 0;
        
        .card-icon {
          font-size: 28px;
        }
        
        .stat-number {
          font-size: 24px;
        }
      }
    }
    
    .tree-visualization {
      .tree-card, .graph-card {
        height: 300px;
        margin-bottom: 16px;
      }
    }
  }
}
</style>