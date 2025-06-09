<template>
  <div class="overview-container">
    <a-page-header title="服务树概览" subtitle="查看和管理企业服务树结构" :backIcon="false">
      <template #extra>
        <a-space>
          <a-button type="primary" @click="refreshData" :loading="loading">
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
        <a-col :xs="24" :lg="12">
          <a-card title="树形视图" :bordered="false" class="tree-card">
            <div class="tree-content">
              <a-spin :spinning="loading">
                <a-tree 
                  v-if="treeData.length > 0"
                  :tree-data="treeData" 
                  :defaultExpandedKeys="defaultExpandedKeys"
                  :showLine="{ showLeafIcon: false }" 
                  @select="onTreeNodeSelect" 
                  class="service-tree"
                  :selectable="true"
                  :checkable="false"
                  :blockNode="true"
                >
                  <template #title="{ title, key }">
                    <div class="tree-node-title">
                      <span class="node-name">{{ title }}</span>
                      <div class="node-tags">
                        <a-tag v-if="getNodeResourceCount(key) > 0" color="blue" size="small">
                          资源: {{ getNodeResourceCount(key) }}
                        </a-tag>
                        <a-tag v-if="getNodeMemberCount(key) > 0" color="green" size="small">
                          成员: {{ getNodeMemberCount(key) }}
                        </a-tag>
                      </div>
                    </div>
                  </template>
                </a-tree>
                <a-empty v-else description="暂无树形数据" />
              </a-spin>
            </div>
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card title="网络视图" :bordered="false" class="graph-card">
            <div class="graph-content">
              <a-spin :spinning="loading">
                <div class="graph-view">
                  <div ref="chartContainer" class="chart-container"></div>
                  <a-empty v-if="treeData.length === 0" description="暂无树形数据" />
                </div>
              </a-spin>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <a-row :gutter="16" class="node-details-row">
      <a-col :xs="24" :lg="12">
        <a-card title="节点详情" :bordered="false" v-if="selectedNode" class="details-card">
          <a-spin :spinning="nodeDetailLoading">
            <a-descriptions :column="1" bordered size="small">
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
              <a-descriptions-item label="管理员">
                <div v-if="selectedNode.adminUsers && selectedNode.adminUsers.length > 0">
                  <div class="member-list">
                    <a-tag 
                      v-for="admin in selectedNode.adminUsers.slice(0, 3)" 
                      :key="admin"
                      color="blue"
                      size="small"
                    >
                      {{ admin }}
                    </a-tag>
                    <a-tag 
                      v-if="selectedNode.adminUsers.length > 3" 
                      color="blue"
                      size="small"
                    >
                      +{{ selectedNode.adminUsers.length - 3 }}...
                    </a-tag>
                  </div>
                  <div class="member-count">
                    共 {{ selectedNode.adminUsers.length }} 名管理员
                  </div>
                </div>
                <span v-else class="empty-text">暂无管理员</span>
              </a-descriptions-item>
              <a-descriptions-item label="普通成员">
                <div v-if="selectedNode.memberUsers && selectedNode.memberUsers.length > 0">
                  <div class="member-list">
                    <a-tag 
                      v-for="member in selectedNode.memberUsers.slice(0, 3)" 
                      :key="member"
                      color="green"
                      size="small"
                    >
                      {{ member }}
                    </a-tag>
                    <a-tag 
                      v-if="selectedNode.memberUsers.length > 3" 
                      color="green"
                      size="small"
                    >
                      +{{ selectedNode.memberUsers.length - 3 }}...
                    </a-tag>
                  </div>
                  <div class="member-count">
                    共 {{ selectedNode.memberUsers.length }} 名成员
                  </div>
                </div>
                <span v-else class="empty-text">暂无普通成员</span>
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
          </a-spin>
        </a-card>
        <a-empty v-else description="请选择节点查看详情" class="empty-detail" />
      </a-col>

      <a-col :xs="24" :lg="12">
        <a-card title="绑定资源" :bordered="false" v-if="selectedNode" class="resources-card">
          <a-spin :spinning="resourceLoading">
            <a-table 
              :dataSource="nodeResources" 
              :columns="resourceColumns" 
              :pagination="{ pageSize: 8, size: 'small', showQuickJumper: true, showSizeChanger: false }"
              size="small"
              :locale="{ emptyText: '暂无绑定资源' }"
              :scroll="{ x: 400 }"
              row-key="id"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'resourceStatus'">
                  <a-tag :color="getResourceStatusColor(record.resourceStatus)" size="small">
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
        <a-empty v-else description="请选择节点查看资源" class="empty-resource" />
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick, onBeforeUnmount, onUnmounted } from 'vue';
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
  getNodeMembers,
  type TreeNodeDetail,
  type TreeNodeListItem,
  type TreeStatistics,
  type TreeNodeResource,
  type GetTreeListParams,
} from '#/api/core/tree_node';

interface UserInfo {
  id: number;
  username: string;
  real_name: string;
  mobile: string;
  account_type: number;
  enable: number;
}

// 定义树节点接口
interface TreeNode {
  key: string;
  title: string;
  isLeaf: boolean;
  children?: TreeNode[];
  [key: string]: any;
}

const router = useRouter();
const loading = ref(false);
const resourceLoading = ref(false);
const nodeDetailLoading = ref(false);
const selectedNode = ref<TreeNodeDetail | null>(null);
const chartContainer = ref<HTMLElement | null>(null);
let chart: echarts.ECharts | null = null;

// 组件卸载标志
const isUnmounted = ref(false);

// 统计数据
const statistics = ref<TreeStatistics | null>(null);

// 树形数据
const treeData = ref<TreeNode[]>([]);
const defaultExpandedKeys = ref<string[]>([]);

// 节点详情缓存
const nodeDetails = ref<Record<string, TreeNodeDetail>>({});

// 节点资源数据
const nodeResources = ref<TreeNodeResource[]>([]);

// 资源表格列定义
const resourceColumns = [
  { 
    title: '资源名称', 
    dataIndex: 'resourceName', 
    key: 'resourceName',
    ellipsis: true,
    width: 120
  },
  { 
    title: '资源类型', 
    dataIndex: 'resourceType', 
    key: 'resourceType',
    width: 100
  },
  { 
    title: '状态', 
    dataIndex: 'resourceStatus', 
    key: 'resourceStatus',
    width: 80
  },
  { 
    title: '创建时间', 
    dataIndex: 'resourceCreateTime', 
    key: 'resourceCreateTime',
    width: 140
  },
];

// 防抖处理
const debounce = <T extends (...args: any[]) => void>(fn: T, delay: number): T => {
  let timeoutId: NodeJS.Timeout;
  return ((...args: any[]) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  }) as T;
};

// 工具函数
const formatDateTime = (dateStr: string | number): string => {
  if (!dateStr) return '-';
  
  try {
    let date: Date;
    if (typeof dateStr === 'number') {
      date = new Date(dateStr * 1000);
    } else {
      date = new Date(dateStr);
    }
    
    if (isNaN(date.getTime())) {
      return '-';
    }
    
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
  } catch (error) {
    console.error('日期格式化错误:', error);
    return '-';
  }
};

const getResourceStatusColor = (status: string): string => {
  const colorMap: Record<string, string> = {
    'running': 'green',
    'stopped': 'red',
    'starting': 'orange',
    'stopping': 'orange',
    'active': 'green',
    'inactive': 'red',
    'online': 'green',
    'offline': 'red',
    'pending': 'orange',
    'error': 'red',
    'warning': 'orange',
  };
  return colorMap[status?.toLowerCase()] || 'default';
};

// 获取节点资源数量（使用computed优化性能）
const getNodeResourceCount = (key: string | number): number => {
  if (isUnmounted.value) return 0;
  const nodeId = parseInt(key.toString());
  return nodeDetails.value[nodeId]?.resourceCount || 0;
};

// 获取节点成员总数
const getNodeMemberCount = (key: string | number): number => {
  if (isUnmounted.value) return 0;
  const nodeId = parseInt(key.toString());
  const node = nodeDetails.value[nodeId];
  if (!node) return 0;
  
  const adminUsers = node.adminUsers || [];
  const memberUsers = node.memberUsers || [];
  
  // 使用Set去重
  const allUsers = new Set([...adminUsers, ...memberUsers]);
  return allUsers.size;
};

// 错误处理函数
const handleError = (error: any, operation: string) => {
  if (isUnmounted.value) return;
  
  console.error(`${operation}失败:`, error);
  
  let errorMessage = `${operation}失败`;
  if (error?.response?.data?.message) {
    errorMessage = error.response.data.message;
  } else if (error?.message) {
    errorMessage = error.message;
  }
  
  message.error(errorMessage);
};

// 异步请求重试机制
const withRetry = async <T>(
  fn: () => Promise<T>, 
  retries = 3, 
  delay = 1000
): Promise<T> => {
  for (let i = 0; i < retries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === retries - 1) throw error;
      if (isUnmounted.value) throw new Error('Component unmounted');
      await new Promise(resolve => setTimeout(resolve, delay * Math.pow(2, i)));
    }
  }
  throw new Error('Max retries exceeded');
};

// 加载节点成员信息
const loadNodeMembers = async (nodeId: number): Promise<{ adminUsers: UserInfo[], memberUsers: UserInfo[] }> => {
  if (isUnmounted.value) {
    return { adminUsers: [], memberUsers: [] };
  }
  
  try {
    const [adminRes, memberRes] = await Promise.all([
      getNodeMembers(nodeId, { type: 'admin' }).catch(() => []),
      getNodeMembers(nodeId, { type: 'member' }).catch(() => [])
    ]);
    
    if (isUnmounted.value) {
      return { adminUsers: [], memberUsers: [] };
    }
    
    return {
      adminUsers: Array.isArray(adminRes) ? adminRes : [],
      memberUsers: Array.isArray(memberRes) ? memberRes : []
    };
  } catch (error) {
    if (!isUnmounted.value) {
      console.warn('获取节点成员失败:', error);
    }
    return {
      adminUsers: [],
      memberUsers: []
    };
  }
};

// 数据加载函数优化
const loadTreeData = async () => {
  if (isUnmounted.value) return;
  
  try {
    const params: GetTreeListParams = {};
    const response = await withRetry(() => getTreeList(params));
    
    if (isUnmounted.value) return;
    
    // 处理响应数据
    const data = response?.data || response;
    const items = data?.items || data;
    
    if (!Array.isArray(items)) {
      throw new Error('API返回的数据格式不正确');
    }
    
    // 批量处理节点（减少API调用）
    const nodeIds = new Set<number>();
    const collectNodeIds = (node: TreeNodeListItem) => {
      nodeIds.add(node.id);
      if (node.children && node.children.length > 0) {
        node.children.forEach(collectNodeIds);
      }
    };
    
    items.forEach(collectNodeIds);
    
    // 批量加载成员信息
    const memberPromises = Array.from(nodeIds).map(async (nodeId) => {
      try {
        const members = await loadNodeMembers(nodeId);
        return { nodeId, members };
      } catch (error) {
        return { nodeId, members: { adminUsers: [], memberUsers: [] } };
      }
    });
    
    const memberResults = await Promise.allSettled(memberPromises);
    
    if (isUnmounted.value) return;
    
    // 处理树节点数据
    const processNode = (node: TreeNodeListItem) => {
      const memberResult = memberResults.find(result => 
        result.status === 'fulfilled' && result.value.nodeId === node.id
      );
      
      const members = memberResult?.status === 'fulfilled' 
        ? memberResult.value.members 
        : { adminUsers: [], memberUsers: [] };
      
      nodeDetails.value[node.id] = {
        id: node.id,
        name: node.name,
        parentId: node.parentId,
        level: node.level,
        description: '',
        creatorId: node.creatorId,
        status: node.status || 'active',
        isLeaf: node.isLeaf,
        createdAt: node.created_at,
        updatedAt: node.updated_at,
        creatorName: '',
        parentName: '',
        childCount: node.children?.length || 0,
        adminUsers: members.adminUsers.map(user => user.username),
        memberUsers: members.memberUsers.map(user => user.username),
        resourceCount: 0,
      };
      
      if (node.children && node.children.length > 0) {
        node.children.forEach(processNode);
      }
    };
    
    // 处理所有节点
    items.forEach(processNode);
    
    if (isUnmounted.value) return;
    
    // 构建树形结构
    const transformNode = (node: TreeNodeListItem): TreeNode => ({
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
      defaultExpandedKeys.value = [treeData.value[0]?.key || ''];
    }
    
    console.log('树形数据加载成功, 节点数量:', nodeIds.size);
  } catch (error) {
    handleError(error, '加载树形数据');
  }
};

const loadStatistics = async () => {
  if (isUnmounted.value) return;
  
  try {
    const res = await withRetry(() => getTreeStatistics());
    if (!isUnmounted.value) {
      statistics.value = res;
    }
  } catch (error) {
    handleError(error, '获取统计数据');
  }
};

const loadNodeResources = async (nodeId: number) => {
  if (!nodeId || isUnmounted.value) return;
  
  resourceLoading.value = true;
  try {
    const res = await withRetry(() => getNodeResources(nodeId));
    
    if (isUnmounted.value) return;
    
    nodeResources.value = Array.isArray(res) ? res : [];
    
    // 更新节点详情中的资源数量
    if (nodeDetails.value[nodeId]) {
      nodeDetails.value[nodeId].resourceCount = nodeResources.value.length;
    }
  } catch (error) {
    if (!isUnmounted.value) {
      handleError(error, '获取节点资源');
      nodeResources.value = [];
    }
  } finally {
    if (!isUnmounted.value) {
      resourceLoading.value = false;
    }
  }
};

// 图表相关函数
const destroyChart = () => {
  if (chart && !chart.isDisposed()) {
    try {
      chart.dispose();
    } catch (error) {
      console.warn('图表销毁时出现错误:', error);
    } finally {
      chart = null;
    }
  }
};

const initChart = () => {
  if (isUnmounted.value || !chartContainer.value) return;
  
  destroyChart();
  
  try {
    chart = echarts.init(chartContainer.value, null, {
      renderer: 'svg',
      useDirtyRect: false
    });
    updateChart();
  } catch (error) {
    console.error('初始化图表失败:', error);
  }
};

const updateChart = () => {
  if (isUnmounted.value || !chart || chart.isDisposed() || treeData.value.length === 0) return;

  try {
    const nodes: any[] = [];
    const links: any[] = [];
    const nodeRelations = new Map();
    
    const buildRelationsMap = (node: TreeNode, parentKey?: string) => {
      if (parentKey) {
        nodeRelations.set(node.key, parentKey);
      }
      
      if (node.children && node.children.length > 0) {
        node.children.forEach((child) => {
          buildRelationsMap(child, node.key);
        });
      }
    };
    
    treeData.value.forEach((rootNode) => {
      buildRelationsMap(rootNode);
    });

    const processNode = (node: TreeNode) => {
      const resourceCount = getNodeResourceCount(node.key);
      const memberCount = getNodeMemberCount(node.key);
      const totalValue = resourceCount + memberCount;
      
      nodes.push({
        name: node.title,
        id: node.key,
        value: totalValue,
        symbolSize: Math.max(20, Math.min(80, 20 + (totalValue * 2))),
        itemStyle: {
          color: node.isLeaf ? '#52c41a' : '#1890ff'
        },
        label: {
          show: true,
          position: 'inside',
          color: '#fff',
          fontWeight: 'bold',
          fontSize: Math.max(10, Math.min(14, 10 + totalValue * 0.3))
        }
      });
      
      if (nodeRelations.has(node.key)) {
        const parentKey = nodeRelations.get(node.key);
        links.push({
          source: parentKey,
          target: node.key,
          lineStyle: {
            width: 2,
            curveness: 0.2,
            opacity: 0.8
          }
        });
      }
      
      if (node.children && node.children.length > 0) {
        node.children.forEach((child) => {
          processNode(child);
        });
      }
    };
    
    treeData.value.forEach((rootNode) => {
      processNode(rootNode);
    });

    const option = {
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(0, 0, 0, 0.8)',
        borderColor: 'rgba(0, 0, 0, 0.8)',
        textStyle: {
          color: '#fff'
        },
        formatter: (params: any) => {
          const nodeId = parseInt(params.data.id);
          const nodeDetail = nodeDetails.value[nodeId];
          const resourceCount = nodeDetail?.resourceCount || 0;
          const adminCount = nodeDetail?.adminUsers?.length || 0;
          const memberCount = nodeDetail?.memberUsers?.length || 0;
          
          return `
            <div style="padding: 8px;">
              <div style="font-weight: bold; margin-bottom: 6px;">${params.data.name}</div>
              <div>资源数量: ${resourceCount}</div>
              <div>管理员: ${adminCount}</div>
              <div>成员: ${memberCount}</div>
            </div>
          `;
        }
      },
      animationDurationUpdate: 1000,
      animationEasingUpdate: 'quinticInOut' as const,
      series: [
        {
          type: 'graph',
          layout: 'force',
          data: nodes,
          links: links,
          roam: true,
          focusNodeAdjacency: true,
          draggable: true,
          itemStyle: {
            borderColor: '#fff',
            borderWidth: 2,
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
            },
            itemStyle: {
              shadowBlur: 20,
              shadowColor: 'rgba(0, 0, 0, 0.5)'
            }
          },
          force: {
            repulsion: 300,
            edgeLength: [80, 150],
            gravity: 0.05,
            friction: 0.6,
            layoutAnimation: true
          }
        }
      ]
    };

    if (!isUnmounted.value && chart && !chart.isDisposed()) {
      chart.setOption(option, true);
    }
  } catch (error) {
    console.error('更新图表失败:', error);
  }
};

// 树节点选择事件
const onTreeNodeSelect = async (selectedKeys: string[]) => {
  if (isUnmounted.value) return;
  
  if (selectedKeys.length > 0) {
    const nodeId = parseInt(selectedKeys[0] || '0');
    
    if (nodeId > 0) {
      nodeDetailLoading.value = true;
      try {
        let nodeDetail = nodeDetails.value[nodeId];
        if (!nodeDetail) {
          // 重新加载节点详情
          const [detailRes, membersRes] = await Promise.all([
            getNodeDetail(nodeId).catch(() => null),
            loadNodeMembers(nodeId)
          ]);
          
          if (isUnmounted.value) return;
          
          if (detailRes) {
            nodeDetail = {
              ...detailRes,
              adminUsers: membersRes.adminUsers.map(user => user.username),
              memberUsers: membersRes.memberUsers.map(user => user.username)
            };
            nodeDetails.value[nodeId] = nodeDetail as TreeNodeDetail;
          }
        }
        
        if (!isUnmounted.value) {
          if (nodeDetail) {
            selectedNode.value = nodeDetail;
            await loadNodeResources(nodeId);
          } else {
            selectedNode.value = null;
            message.error('获取节点详情失败');
          }
        }
      } catch (error) {
        if (!isUnmounted.value) {
          handleError(error, '获取节点数据');
          selectedNode.value = null;
        }
      } finally {
        if (!isUnmounted.value) {
          nodeDetailLoading.value = false;
        }
      }
    }
  } else {
    if (!isUnmounted.value) {
      selectedNode.value = null;
      nodeResources.value = [];
    }
  }
};

// 刷新数据
const refreshData = async () => {
  if (isUnmounted.value) return;
  
  loading.value = true;
  try {
    await Promise.all([
      loadTreeData(),
      loadStatistics(),
    ]);
    
    if (isUnmounted.value) return;
    
    // 延迟初始化图表，确保DOM已更新
    await nextTick();
    setTimeout(() => {
      if (!isUnmounted.value && chartContainer.value) {
        if (chart && !chart.isDisposed()) {
          updateChart();
        } else {
          initChart();
        }
      }
    }, 100);
    
    message.success('数据刷新成功');
  } catch (error) {
    handleError(error, '刷新数据');
  } finally {
    if (!isUnmounted.value) {
      loading.value = false;
    }
  }
};

// 导航函数
const navigateToManagePage = () => {
  if (!isUnmounted.value) {
    router.push('/tree_node_manager');
  }
};

// 窗口大小变化处理
const handleResize = debounce(() => {
  if (!isUnmounted.value && chart && !chart.isDisposed()) {
    try {
      chart.resize();
    } catch (error) {
      console.warn('图表resize失败:', error);
    }
  }
}, 300);

// 监听树数据变化
watch(treeData, () => {
  if (!isUnmounted.value && chart && !chart.isDisposed() && chartContainer.value) {
    // 防抖更新图表
    setTimeout(() => {
      if (!isUnmounted.value) {
        updateChart();
      }
    }, 200);
  }
}, { deep: true });

// 生命周期钩子
onMounted(() => {
  refreshData();
  window.addEventListener('resize', handleResize);
});

onBeforeUnmount(() => {
  isUnmounted.value = true;
  window.removeEventListener('resize', handleResize);
});

onUnmounted(() => {
  destroyChart();
  selectedNode.value = null;
  nodeResources.value = [];
  treeData.value = [];
  nodeDetails.value = {};
  statistics.value = null;
});
</script>

<style scoped lang="scss">
.overview-container {
  padding: 16px;
  min-height: 100vh;
  background-color: #f5f5f5;

  .dashboard-cards {
    margin-top: 16px;
    margin-bottom: 24px;

    .stat-card {
      text-align: center;
      padding: 24px 16px;
      background: linear-gradient(135deg, #1890ff0a 0%, #1890ff1a 100%);
      border-radius: 8px;

      .card-icon {
        font-size: 36px;
        color: #1890ff;
        margin-bottom: 16px;
        display: block;
        transition: all 0.3s ease;
      }

      .stat-number {
        font-size: 28px;
        font-weight: 600;
        margin-bottom: 8px;
        color: #262626;
        line-height: 1;
      }

      .stat-label {
        font-size: 14px;
        color: #8c8c8c;
        font-weight: 500;
      }

      &:hover .card-icon {
        transform: scale(1.1);
        color: #096dd9;
      }
    }
  }

  .tree-visualization {
    margin-bottom: 24px;

    .tree-card, .graph-card {
      min-height: 420px;

      :deep(.ant-card-body) {
        padding: 16px;
        height: calc(100% - 57px);
      }

      .tree-content, .graph-content {
        height: 100%;

        .service-tree {
          height: 100%;
          overflow: auto;
          padding: 8px;

          :deep(.ant-tree-node-content-wrapper) {
            width: 100%;
            padding: 4px 8px;
            border-radius: 4px;
            transition: all 0.2s ease;

            &:hover {
              background-color: #f0f8ff;
            }

            &.ant-tree-node-selected {
              background-color: #e6f7ff;
            }
          }

          .tree-node-title {
            display: flex;
            align-items: center;
            justify-content: space-between;
            width: 100%;
            gap: 8px;

            .node-name {
              flex: 1;
              font-weight: 500;
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
            }

            .node-tags {
              display: flex;
              gap: 4px;
              flex-shrink: 0;

              .ant-tag {
                margin: 0;
                font-size: 11px;
                line-height: 18px;
              }
            }
          }
        }
      }
    }

    .graph-view {
      height: 350px;
      width: 100%;
      border-radius: 6px;
      position: relative;
      border: 1px solid #e8e8e8;
      background: #fff;

      .chart-container {
        width: 100%;
        height: 100%;
        border-radius: 6px;
      }
    }
  }

  .node-details-row {
    .details-card,
    .resources-card {
      min-height: 450px;

      :deep(.ant-card-body) {
        padding: 16px;
      }
    }
    
    .member-list {
      margin-bottom: 8px;
      
      .ant-tag {
        margin-right: 4px;
        margin-bottom: 4px;
      }
    }
    
    .member-count {
      font-size: 12px;
      color: #666;
      margin-top: 4px;
      font-style: italic;
    }
    
    .empty-text {
      color: #999;
      font-style: italic;
    }

    .empty-detail,
    .empty-resource {
      margin-top: 100px;
    }
  }

  // 表格优化
  :deep(.ant-table) {
    .ant-table-thead > tr > th {
      background-color: #fafafa;
      font-weight: 600;
      color: #262626;
    }

    .ant-table-tbody > tr:hover > td {
      background-color: #f5f5f5;
    }
  }

  // 描述列表优化
  :deep(.ant-descriptions) {
    .ant-descriptions-item-label {
      font-weight: 600;
      color: #262626;
      background-color: #fafafa;
    }

    .ant-descriptions-item-content {
      color: #595959;
    }
  }
}

// 响应式调整
@media (max-width: 1200px) {
  .overview-container {
    .tree-visualization {
      .tree-card, .graph-card {
        margin-bottom: 16px;
      }
    }
    
    .node-details-row {
      .details-card,
      .resources-card {
        margin-bottom: 16px;
      }
    }
  }
}

@media (max-width: 768px) {
  .overview-container {
    padding: 8px;
    
    .dashboard-cards {
      .stat-card {
        padding: 16px 8px;
        
        .card-icon {
          font-size: 28px;
          margin-bottom: 12px;
        }
        
        .stat-number {
          font-size: 24px;
        }

        .stat-label {
          font-size: 13px;
        }
      }
    }
    
    .tree-visualization {
      .tree-card, .graph-card {
        height: 300px;
        margin-bottom: 12px;
      }

      .graph-view {
        height: 250px;
      }
    }

    .node-details-row {
      .details-card,
      .resources-card {
        min-height: 350px;
        margin-bottom: 12px;
      }
    }
  }
}

@media (max-width: 576px) {
  .overview-container {
    .tree-visualization {
      .tree-content {
        .tree-node-title {
          flex-direction: column;
          align-items: flex-start;
          gap: 4px;

          .node-tags {
            align-self: flex-end;
          }
        }
      }
    }
  }
}
</style>