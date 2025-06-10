<template>
    <div class="gpu-topology-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">GPU拓扑</h2>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入节点名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="节点状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="Ready">正常</a-select-option>
            <a-select-option value="NotReady">异常</a-select-option>
            <a-select-option value="Unknown">未知</a-select-option>
          </a-select>
          <a-select 
            v-model:value="gpuTypeFilter" 
            placeholder="GPU类型" 
            class="gpu-type-filter"
            allowClear
          >
            <a-select-option value="">全部类型</a-select-option>
            <a-select-option value="A100">NVIDIA A100</a-select-option>
            <a-select-option value="V100">NVIDIA V100</a-select-option>
            <a-select-option value="RTX3090">NVIDIA RTX 3090</a-select-option>
            <a-select-option value="RTX4090">NVIDIA RTX 4090</a-select-option>
          </a-select>
          <a-button type="primary" class="action-button" @click="handleSearch">
            <template #icon>
              <SearchOutlined />
            </template>
            搜索
          </a-button>
          <a-button class="action-button reset-button" @click="handleReset">
            <template #icon>
              <ReloadOutlined />
            </template>
            重置
          </a-button>
        </div>
        <div class="action-buttons">
          <a-button type="primary" class="refresh-button" @click="handleRefresh">
            <template #icon>
              <ReloadOutlined />
            </template>
            刷新拓扑
          </a-button>
        </div>
      </div>
  
      <!-- GPU拓扑视图 -->
      <div class="dashboard-card topology-container">
        <div class="topology-header">
          <h3 class="section-title">集群GPU拓扑视图</h3>
          <div class="topology-stats">
            <div class="stat-item">
              <span class="stat-label">节点总数:</span>
              <span class="stat-value">{{ nodeStats.total }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">GPU总数:</span>
              <span class="stat-value">{{ nodeStats.totalGPUs }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">可用GPU:</span>
              <span class="stat-value available">{{ nodeStats.availableGPUs }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">使用中GPU:</span>
              <span class="stat-value used">{{ nodeStats.usedGPUs }}</span>
            </div>
          </div>
        </div>
  
        <div class="topology-view">
          <div class="node-grid">
            <div 
              v-for="node in filteredNodes" 
              :key="node.id"
              class="node-card"
              :class="{ 'node-offline': node.status !== 'Ready' }"
              @click="handleNodeClick(node)"
            >
              <div class="node-header">
                <div class="node-info">
                  <h4 class="node-name">{{ node.name }}</h4>
                  <a-tag :color="getNodeStatusColor(node.status)" class="node-status">
                    {{ getNodeStatusText(node.status) }}
                  </a-tag>
                </div>
                <div class="node-metrics">
                  <div class="metric-item">
                    <span class="metric-label">CPU:</span>
                    <span class="metric-value">{{ node.cpuUsage }}%</span>
                  </div>
                  <div class="metric-item">
                    <span class="metric-label">内存:</span>
                    <span class="metric-value">{{ node.memoryUsage }}%</span>
                  </div>
                </div>
              </div>
  
              <div class="gpu-grid">
                <div 
                  v-for="gpu in node.gpus" 
                  :key="gpu.id"
                  class="gpu-card"
                  :class="{ 
                    'gpu-used': gpu.status === 'Used',
                    'gpu-available': gpu.status === 'Available',
                    'gpu-error': gpu.status === 'Error'
                  }"
                  @click.stop="handleGPUClick(node, gpu)"
                >
                  <div class="gpu-header">
                    <span class="gpu-id">GPU {{ gpu.index }}</span>
                    <span class="gpu-type">{{ gpu.type }}</span>
                  </div>
                  <div class="gpu-usage">
                    <div class="usage-bar">
                      <div 
                        class="usage-fill" 
                        :style="{ width: gpu.utilization + '%' }"
                      ></div>
                    </div>
                    <span class="usage-text">{{ gpu.utilization }}%</span>
                  </div>
                  <div class="gpu-memory">
                    <span class="memory-text">{{ gpu.memoryUsed }}GB / {{ gpu.memoryTotal }}GB</span>
                  </div>
                  <div class="gpu-process" v-if="gpu.process">
                    <span class="process-text">{{ gpu.process }}</span>
                  </div>
                </div>
              </div>
  
              <div class="node-footer">
                <div class="node-labels">
                  <a-tag v-for="label in node.labels" :key="label" size="small" class="node-label">
                    {{ label }}
                  </a-tag>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
  
      <!-- 节点详情模态框 -->
      <a-modal 
        title="节点详情" 
        v-model:visible="isNodeModalVisible" 
        @cancel="closeNodeModal"
        :width="800"
        class="custom-modal"
        :footer="null"
      >
        <div class="node-detail-container" v-if="selectedNode">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="节点名称">{{ selectedNode.name }}</a-descriptions-item>
              <a-descriptions-item label="节点状态">
                <a-tag :color="getNodeStatusColor(selectedNode.status)">{{ getNodeStatusText(selectedNode.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="节点IP">{{ selectedNode.ip }}</a-descriptions-item>
              <a-descriptions-item label="操作系统">{{ selectedNode.os }}</a-descriptions-item>
              <a-descriptions-item label="内核版本">{{ selectedNode.kernelVersion }}</a-descriptions-item>
              <a-descriptions-item label="容器运行时">{{ selectedNode.containerRuntime }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ selectedNode.createdAt }}</a-descriptions-item>
              <a-descriptions-item label="最后心跳">{{ selectedNode.lastHeartbeat }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">资源信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="CPU核心">{{ selectedNode.cpuCores }}</a-descriptions-item>
              <a-descriptions-item label="CPU使用率">{{ selectedNode.cpuUsage }}%</a-descriptions-item>
              <a-descriptions-item label="总内存">{{ selectedNode.memoryTotal }}GB</a-descriptions-item>
              <a-descriptions-item label="内存使用率">{{ selectedNode.memoryUsage }}%</a-descriptions-item>
              <a-descriptions-item label="GPU数量">{{ selectedNode.gpus.length }}</a-descriptions-item>
              <a-descriptions-item label="可用GPU">{{ selectedNode.gpus.filter(g => g.status === 'Available').length }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">GPU详情</div>
            <div class="gpu-detail-list">
              <div v-for="gpu in selectedNode.gpus" :key="gpu.id" class="gpu-detail-item">
                <div class="gpu-detail-header">
                  <span class="gpu-detail-name">GPU {{ gpu.index }} - {{ gpu.type }}</span>
                  <a-tag :color="getGPUStatusColor(gpu.status)" class="gpu-detail-status">
                    {{ getGPUStatusText(gpu.status) }}
                  </a-tag>
                </div>
                <div class="gpu-detail-metrics">
                  <div class="metric-row">
                    <span class="metric-label">利用率:</span>
                    <div class="metric-bar">
                      <div class="bar-bg">
                        <div class="bar-fill" :style="{ width: gpu.utilization + '%' }"></div>
                      </div>
                      <span class="metric-value">{{ gpu.utilization }}%</span>
                    </div>
                  </div>
                  <div class="metric-row">
                    <span class="metric-label">显存:</span>
                    <div class="metric-bar">
                      <div class="bar-bg">
                        <div class="bar-fill memory" :style="{ width: (gpu.memoryUsed / gpu.memoryTotal * 100) + '%' }"></div>
                      </div>
                      <span class="metric-value">{{ gpu.memoryUsed }}GB / {{ gpu.memoryTotal }}GB</span>
                    </div>
                  </div>
                  <div class="metric-row" v-if="gpu.temperature">
                    <span class="metric-label">温度:</span>
                    <span class="metric-value">{{ gpu.temperature }}°C</span>
                  </div>
                  <div class="metric-row" v-if="gpu.powerUsage">
                    <span class="metric-label">功耗:</span>
                    <span class="metric-value">{{ gpu.powerUsage }}W / {{ gpu.powerLimit }}W</span>
                  </div>
                </div>
                <div class="gpu-detail-process" v-if="gpu.process">
                  <span class="process-label">运行进程:</span>
                  <span class="process-name">{{ gpu.process }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-modal>
  
      <!-- GPU详情模态框 -->
      <a-modal 
        title="GPU详情" 
        v-model:visible="isGPUModalVisible" 
        @cancel="closeGPUModal"
        :width="600"
        class="custom-modal"
        :footer="null"
      >
        <div class="gpu-detail-container" v-if="selectedGPU">
          <div class="detail-section">
            <div class="section-title">GPU信息</div>
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="GPU ID">{{ selectedGPU.id }}</a-descriptions-item>
              <a-descriptions-item label="GPU索引">{{ selectedGPU.index }}</a-descriptions-item>
              <a-descriptions-item label="GPU型号">{{ selectedGPU.type }}</a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getGPUStatusColor(selectedGPU.status)">{{ getGPUStatusText(selectedGPU.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="驱动版本">{{ selectedGPU.driverVersion }}</a-descriptions-item>
              <a-descriptions-item label="CUDA版本">{{ selectedGPU.cudaVersion }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">实时监控</div>
            <div class="gpu-metrics">
              <div class="metric-card">
                <div class="metric-title">GPU利用率</div>
                <div class="metric-progress">
                  <a-progress 
                    :percent="selectedGPU.utilization" 
                    status="active"
                    :stroke-color="getUtilizationColor(selectedGPU.utilization)"
                  />
                </div>
              </div>
              <div class="metric-card">
                <div class="metric-title">显存使用</div>
                <div class="metric-progress">
                  <a-progress 
                    :percent="Math.round(selectedGPU.memoryUsed / selectedGPU.memoryTotal * 100)" 
                    status="active"
                    stroke-color="#722ed1"
                  />
                </div>
                <div class="metric-text">{{ selectedGPU.memoryUsed }}GB / {{ selectedGPU.memoryTotal }}GB</div>
              </div>
              <div class="metric-card" v-if="selectedGPU.temperature">
                <div class="metric-title">温度</div>
                <div class="metric-value large">{{ selectedGPU.temperature }}°C</div>
              </div>
              <div class="metric-card" v-if="selectedGPU.powerUsage">
                <div class="metric-title">功耗</div>
                <div class="metric-value large">{{ selectedGPU.powerUsage }}W</div>
              </div>
            </div>
          </div>
  
          <div class="detail-section" v-if="selectedGPU.process">
            <div class="section-title">运行进程</div>
            <div class="process-info">
              <div class="process-item">
                <span class="process-label">进程名称:</span>
                <span class="process-value">{{ selectedGPU.process }}</span>
              </div>
              <div class="process-item" v-if="selectedGPU.pid">
                <span class="process-label">进程ID:</span>
                <span class="process-value">{{ selectedGPU.pid }}</span>
              </div>
              <div class="process-item" v-if="selectedGPU.user">
                <span class="process-label">用户:</span>
                <span class="process-value">{{ selectedGPU.user }}</span>
              </div>
            </div>
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, onMounted, computed } from 'vue';
  import { message } from 'ant-design-vue';
  import {
    SearchOutlined,
    ReloadOutlined
  } from '@ant-design/icons-vue';
  
  interface GPU {
    id: string;
    index: number;
    type: string;
    status: 'Available' | 'Used' | 'Error';
    utilization: number;
    memoryUsed: number;
    memoryTotal: number;
    temperature?: number;
    powerUsage?: number;
    powerLimit?: number;
    process?: string;
    pid?: number;
    user?: string;
    driverVersion?: string;
    cudaVersion?: string;
  }
  
  interface Node {
    id: string;
    name: string;
    status: 'Ready' | 'NotReady' | 'Unknown';
    ip: string;
    os: string;
    kernelVersion: string;
    containerRuntime: string;
    cpuCores: number;
    cpuUsage: number;
    memoryTotal: number;
    memoryUsage: number;
    gpus: GPU[];
    labels: string[];
    createdAt: string;
    lastHeartbeat: string;
  }
  
  // 搜索和筛选
  const searchText = ref('');
  const statusFilter = ref('');
  const gpuTypeFilter = ref('');
  
  // 节点数据
  const nodes = ref<Node[]>([]);
  
  // 模态框状态
  const isNodeModalVisible = ref(false);
  const isGPUModalVisible = ref(false);
  
  // 选中的节点和GPU
  const selectedNode = ref<Node | null>(null);
  const selectedGPU = ref<GPU | null>(null);
  
  // 节点统计
  const nodeStats = computed(() => {
    const total = nodes.value.length;
    const totalGPUs = nodes.value.reduce((sum, node) => sum + node.gpus.length, 0);
    const availableGPUs = nodes.value.reduce((sum, node) => 
      sum + node.gpus.filter(gpu => gpu.status === 'Available').length, 0);
    const usedGPUs = nodes.value.reduce((sum, node) => 
      sum + node.gpus.filter(gpu => gpu.status === 'Used').length, 0);
    
    return {
      total,
      totalGPUs,
      availableGPUs,
      usedGPUs
    };
  });
  
  // 过滤后的节点
  const filteredNodes = computed(() => {
    return nodes.value.filter(node => {
      const matchesSearch = !searchText.value || 
        node.name.toLowerCase().includes(searchText.value.toLowerCase());
      const matchesStatus = !statusFilter.value || node.status === statusFilter.value;
      const matchesGPUType = !gpuTypeFilter.value || 
        node.gpus.some(gpu => gpu.type.includes(gpuTypeFilter.value));
      
      return matchesSearch && matchesStatus && matchesGPUType;
    });
  });
  
  // 初始化数据
  onMounted(() => {
    loadData();
  });
  
  // 获取节点状态颜色
  const getNodeStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'Ready': 'green',
      'NotReady': 'red',
      'Unknown': 'orange'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取节点状态文本
  const getNodeStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'Ready': '正常',
      'NotReady': '异常',
      'Unknown': '未知'
    };
    return textMap[status] || status;
  };
  
  // 获取GPU状态颜色
  const getGPUStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'Available': 'green',
      'Used': 'blue',
      'Error': 'red'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取GPU状态文本
  const getGPUStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'Available': '可用',
      'Used': '使用中',
      'Error': '错误'
    };
    return textMap[status] || status;
  };
  
  // 获取利用率颜色
  const getUtilizationColor = (utilization: number) => {
    if (utilization >= 80) return '#f5222d';
    if (utilization >= 60) return '#fa8c16';
    if (utilization >= 40) return '#fadb14';
    return '#52c41a';
  };
  
  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockNodes: Node[] = [
      {
        id: 'node-1',
        name: 'gpu-node-001',
        status: 'Ready',
        ip: '192.168.1.101',
        os: 'Ubuntu 20.04.6 LTS',
        kernelVersion: '5.4.0-150-generic',
        containerRuntime: 'containerd://1.6.20',
        cpuCores: 32,
        cpuUsage: 45,
        memoryTotal: 128,
        memoryUsage: 62,
        gpus: [
          {
            id: 'gpu-1-0',
            index: 0,
            type: 'NVIDIA A100',
            status: 'Used',
            utilization: 85,
            memoryUsed: 35,
            memoryTotal: 40,
            temperature: 72,
            powerUsage: 320,
            powerLimit: 400,
            process: 'pytorch-training',
            pid: 12345,
            user: 'user1',
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          },
          {
            id: 'gpu-1-1',
            index: 1,
            type: 'NVIDIA A100',
            status: 'Available',
            utilization: 0,
            memoryUsed: 0,
            memoryTotal: 40,
            temperature: 35,
            powerUsage: 45,
            powerLimit: 400,
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          }
        ],
        labels: ['gpu=a100', 'zone=us-west1-a'],
        createdAt: '2024-06-01 10:30:00',
        lastHeartbeat: '2024-06-09 14:30:15'
      },
      {
        id: 'node-2',
        name: 'gpu-node-002',
        status: 'Ready',
        ip: '192.168.1.102',
        os: 'Ubuntu 20.04.6 LTS',
        kernelVersion: '5.4.0-150-generic',
        containerRuntime: 'containerd://1.6.20',
        cpuCores: 24,
        cpuUsage: 23,
        memoryTotal: 96,
        memoryUsage: 38,
        gpus: [
          {
            id: 'gpu-2-0',
            index: 0,
            type: 'NVIDIA V100',
            status: 'Used',
            utilization: 92,
            memoryUsed: 14,
            memoryTotal: 16,
            temperature: 78,
            powerUsage: 280,
            powerLimit: 300,
            process: 'tensorflow-train',
            pid: 23456,
            user: 'user2',
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          },
          {
            id: 'gpu-2-1',
            index: 1,
            type: 'NVIDIA V100',
            status: 'Available',
            utilization: 0,
            memoryUsed: 0,
            memoryTotal: 16,
            temperature: 42,
            powerUsage: 55,
            powerLimit: 300,
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          },
          {
            id: 'gpu-2-2',
            index: 2,
            type: 'NVIDIA V100',
            status: 'Error',
            utilization: 0,
            memoryUsed: 0,
            memoryTotal: 16,
            temperature: 95,
            powerUsage: 0,
            powerLimit: 300,
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          }
        ],
        labels: ['gpu=v100', 'zone=us-west1-b'],
        createdAt: '2024-06-01 11:15:00',
        lastHeartbeat: '2024-06-09 14:30:12'
      },
      {
        id: 'node-3',
        name: 'gpu-node-003',
        status: 'Ready',
        ip: '192.168.1.103',
        os: 'Ubuntu 20.04.6 LTS',
        kernelVersion: '5.4.0-150-generic',
        containerRuntime: 'containerd://1.6.20',
        cpuCores: 16,
        cpuUsage: 12,
        memoryTotal: 64,
        memoryUsage: 25,
        gpus: [
          {
            id: 'gpu-3-0',
            index: 0,
            type: 'NVIDIA RTX 3090',
            status: 'Available',
            utilization: 0,
            memoryUsed: 0,
            memoryTotal: 24,
            temperature: 38,
            powerUsage: 35,
            powerLimit: 350,
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          },
          {
            id: 'gpu-3-1',
            index: 1,
            type: 'NVIDIA RTX 3090',
            status: 'Available',
            utilization: 0,
            memoryUsed: 0,
            memoryTotal: 24,
            temperature: 36,
            powerUsage: 32,
            powerLimit: 350,
            driverVersion: '535.86.10',
            cudaVersion: '12.2'
          }
        ],
        labels: ['gpu=rtx3090', 'zone=us-west1-c'],
        createdAt: '2024-06-02 09:20:00',
        lastHeartbeat: '2024-06-09 14:30:08'
      }
    ];
    
    nodes.value = mockNodes;
  };
  
  // 搜索处理
  const handleSearch = () => {
    message.success('搜索完成');
  };
  
  // 重置处理
  const handleReset = () => {
    searchText.value = '';
    statusFilter.value = '';
    gpuTypeFilter.value = '';
    message.success('重置成功');
  };
  
  // 刷新拓扑
  const handleRefresh = () => {
    loadData();
    message.success('拓扑已刷新');
  };
  
  // 节点点击处理
  const handleNodeClick = (node: Node) => {
    selectedNode.value = node;
    isNodeModalVisible.value = true;
  };
  
  // 关闭节点模态框
  const closeNodeModal = () => {
    isNodeModalVisible.value = false;
    selectedNode.value = null;
  };
  
  // GPU点击处理
  const handleGPUClick = (node: Node, gpu: GPU) => {
    selectedGPU.value = gpu;
    isGPUModalVisible.value = true;
  };
  
  // 关闭GPU模态框
  const closeGPUModal = () => {
    isGPUModalVisible.value = false;
    selectedGPU.value = null;
  };
  </script>
  
  <style scoped>
  .gpu-topology-page {
    padding: 20px;
    background-color: #f5f7fa;
    min-height: 100vh;
  }
  
  .page-header {
    margin-bottom: 24px;
  }
  
  .page-title {
    font-size: 24px;
    font-weight: 600;
    color: #1a202c;
    margin: 0;
  }
  
  .dashboard-card {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    margin-bottom: 24px;
  }
  
  .custom-toolbar {
    padding: 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
  }
  
  .search-filters {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    align-items: center;
  }
  
  .search-input {
    width: 200px;
  }
  
  .status-filter,
  .gpu-type-filter {
    width: 150px;
  }
  
  .action-button {
    height: 32px;
  }
  
  .reset-button {
    background: #f1f5f9;
    border-color: #e2e8f0;
    color: #475569;
  }
  
  .action-buttons {
    display: flex;
    gap: 12px;
  }
  
  .refresh-button {
    background: #3b82f6;
    border-color: #3b82f6;
  }
  
  .topology-container {
    padding: 20px;
  }
  
  .topology-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    flex-wrap: wrap;
    gap: 16px;
  }
  
  .section-title {
    font-size: 18px;
    font-weight: 600;
    color: #1a202c;
    margin: 0;
  }
  
  .topology-stats {
    display: flex;
    gap: 24px;
    flex-wrap: wrap;
  }
  
  .stat-item {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .stat-label {
    color: #64748b;
    font-size: 14px;
  }
  
  .stat-value {
    font-weight: 600;
    font-size: 16px;
    color: #1a202c;
  }
  
  .stat-value.available {
    color: #10b981;
  }
  
  .stat-value.used {
    color: #3b82f6;
  }
  
  .topology-view {
    margin-top: 20px;
  }
  
  .node-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
    gap: 20px;
  }
  
  .node-card {
    border: 2px solid #e2e8f0;
    border-radius: 8px;
    padding: 16px;
    cursor: pointer;
    transition: all 0.3s ease;
    background: #fafbfc;
  }
  
  .node-card:hover {
    border-color: #3b82f6;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
  }
  
  .node-card.node-offline {
    border-color: #ef4444;
    background: #fef2f2;
  }
  
  .node-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 16px;
  }
  
  .node-info h4 {
    margin: 0 0 8px 0;
    font-size: 16px;
    font-weight: 600;
    color: #1a202c;
  }
  
  .node-status {
    font-size: 12px;
    font-weight: 500;
  }
  
  .node-metrics {
    display: flex;
    flex-direction: column;
    gap: 4px;
    text-align: right;
  }
  
  .metric-item {
    font-size: 12px;
    color: #64748b;
  }
  
  .metric-label {
    font-weight: 500;
  }
  
  .metric-value {
    color: #1a202c;
    font-weight: 600;
    margin-left: 4px;
  }
  
  .gpu-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 12px;
    margin-bottom: 16px;
  }
  
  .gpu-card {
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 12px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .gpu-card:hover {
    border-color: #3b82f6;
    transform: translateY(-2px);
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  }
  
  .gpu-card.gpu-available {
    border-color: #10b981;
    background: #f0fdf4;
  }
  
  .gpu-card.gpu-used {
    border-color: #3b82f6;
    background: #eff6ff;
  }
  
  .gpu-card.gpu-error {
    border-color: #ef4444;
    background: #fef2f2;
  }
  
  .gpu-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  
  .gpu-id {
    font-weight: 600;
    font-size: 12px;
    color: #1a202c;
  }
  
  .gpu-type {
    font-size: 10px;
    color: #64748b;
    background: #f1f5f9;
    padding: 2px 6px;
    border-radius: 3px;
  }
  
  .gpu-usage {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
  }
  
  .usage-bar {
    flex: 1;
    height: 4px;
    background: #e2e8f0;
    border-radius: 2px;
    overflow: hidden;
  }
  
  .usage-fill {
    height: 100%;
    background: linear-gradient(90deg, #10b981, #3b82f6, #f59e0b, #ef4444);
    transition: width 0.3s ease;
  }
  
  .usage-text {
    font-size: 10px;
    font-weight: 600;
    color: #1a202c;
    min-width: 30px;
  }
  
  .gpu-memory {
    margin-bottom: 6px;
  }
  
  .memory-text {
    font-size: 10px;
    color: #64748b;
  }
  
  .gpu-process {
    background: #f8fafc;
    padding: 4px 6px;
    border-radius: 3px;
    border: 1px solid #e2e8f0;
  }
  
  .process-text {
    font-size: 10px;
    color: #374151;
    font-weight: 500;
  }
  
  .node-footer {
    border-top: 1px solid #e2e8f0;
    padding-top: 12px;
  }
  
  .node-labels {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
  
  .node-label {
    font-size: 10px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    color: #64748b;
  }
  
  .custom-modal :deep(.ant-modal-header) {
    border-bottom: 1px solid #e2e8f0;
    padding: 16px 24px;
  }
  
  .custom-modal :deep(.ant-modal-title) {
    font-size: 18px;
    font-weight: 600;
    color: #1a202c;
  }
  
  .node-detail-container,
  .gpu-detail-container {
    max-height: 600px;
    overflow-y: auto;
  }
  
  .detail-section {
    margin-bottom: 24px;
  }
  
  .detail-section .section-title {
    font-size: 14px;
    font-weight: 600;
    color: #1a202c;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #e2e8f0;
  }
  
  .gpu-detail-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  
  .gpu-detail-item {
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 16px;
    background: #fafbfc;
  }
  
  .gpu-detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }
  
  .gpu-detail-name {
    font-weight: 600;
    color: #1a202c;
  }
  
  .gpu-detail-status {
    font-size: 12px;
  }
  
  .gpu-detail-metrics {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .metric-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  
  .metric-row .metric-label {
    min-width: 60px;
    font-size: 12px;
    color: #64748b;
    font-weight: 500;
  }
  
  .metric-bar {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .bar-bg {
    flex: 1;
    height: 6px;
    background: #e2e8f0;
    border-radius: 3px;
    overflow: hidden;
  }
  
  .bar-fill {
    height: 100%;
    background: #3b82f6;
    transition: width 0.3s ease;
  }
  
  .bar-fill.memory {
    background: #722ed1;
  }
  
  .metric-row .metric-value {
    font-size: 12px;
    color: #1a202c;
    font-weight: 600;
    min-width: 80px;
  }
  
  .gpu-detail-process {
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid #e2e8f0;
  }
  
  .process-label {
    font-size: 12px;
    color: #64748b;
    font-weight: 500;
  }
  
  .process-name {
    font-size: 12px;
    color: #1a202c;
    font-weight: 600;
    margin-left: 8px;
  }
  
  .gpu-metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }
  
  .metric-card {
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 16px;
    background: #fafbfc;
  }
  
  .metric-title {
    font-size: 14px;
    font-weight: 600;
    color: #1a202c;
    margin-bottom: 12px;
  }
  
  .metric-progress {
    margin-bottom: 8px;
  }
  
  .metric-text {
    font-size: 12px;
    color: #64748b;
    text-align: center;
  }
  
  .metric-value.large {
    font-size: 24px;
    font-weight: 700;
    color: #1a202c;
    text-align: center;
    margin-top: 8px;
  }
  
  .process-info {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 16px;
  }
  
  .process-item {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }
  
  .process-item:last-child {
    margin-bottom: 0;
  }
  
  .process-item .process-label {
    min-width: 80px;
    font-size: 12px;
    color: #64748b;
    font-weight: 500;
  }
  
  .process-item .process-value {
    font-size: 12px;
    color: #1a202c;
    font-weight: 600;
  }
  
  @media (max-width: 768px) {
    .custom-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
    
    .search-filters {
      justify-content: stretch;
    }
    
    .search-input,
    .status-filter,
    .gpu-type-filter {
      width: 100%;
      min-width: auto;
    }
    
    .action-buttons {
      justify-content: center;
    }
  
    .node-grid {
      grid-template-columns: 1fr;
    }
    
    .topology-header {
      flex-direction: column;
      align-items: flex-start;
    }
    
    .topology-stats {
      width: 100%;
      justify-content: space-between;
    }
  
    .gpu-grid {
      grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    }
  
    .gpu-metrics {
      grid-template-columns: 1fr;
    }
  }
  </style>