<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input v-model="searchText" placeholder="请输入节点名称" style="width: 200px; margin-right: 16px" />
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="isAddLabelModalVisible = true" style="margin-right: 8px">
          添加节点标签
        </a-button>
        <a-button type="primary" @click="isAddTaintModalVisible = true" style="margin-right: 8px">
          添加Taint
        </a-button>
        <a-button type="primary" @click="handleToggleSchedule" style="margin-right: 8px">
          启用/禁用调度
        </a-button>
        <a-button type="primary" @click="isDeleteTaintModalVisible = true" style="margin-right: 8px">
          删除Taint
        </a-button>
        <a-button type="primary" @click="handleClearTaints">
          清空Taint
        </a-button>
      </div>
    </div>
    <!-- 节点表格 -->
    <a-table :columns="columns" :data-source="filteredData" pagination="{false}" row-key="name">
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="handleViewDetails(record)">
            <template #icon><EyeOutlined /></template>
            查看详情
          </a-button>
          <a-button type="primary" ghost size="small" @click="showDeleteLabelModal(record)">
            <template #icon><DeleteOutlined /></template>
            删除标签
          </a-button>
          <a-button type="primary" ghost size="small" @click="handleToggleSchedule(record)">
            <template #icon><ReloadOutlined /></template>
            {{ record.schedulable ? '禁用调度' : '启用调度' }}
          </a-button>
        </a-space>
      </template>
    </a-table>
    <!-- 添加标签模态框 -->
    <a-modal v-model:visible="isAddLabelModalVisible" title="添加节点标签" @cancel="closeAddLabelModal" @ok="handleAddLabel">
      <a-form :model="labelForm" layout="vertical">
        <a-form-item :rules="[{ required: true, message: '请选择节点名称' }]" label="节点名称" name="nodeName">
          <a-select v-model:value="labelForm.nodeName" placeholder="请选择节点名称">
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              {{ node.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :rules="[{ required: true, message: '请输入标签键' }]" label="标签键" name="key">
          <a-input v-model:value="labelForm.key" placeholder="请输入标签键" />
        </a-form-item>
        <a-form-item :rules="[{ required: true, message: '请输入标签值' }]" label="标签值" name="value">
          <a-input v-model:value="labelForm.value" placeholder="请输入标签值" />
        </a-form-item>
      </a-form>
    </a-modal>
    <!-- 添加Taint模态框 -->
    <a-modal v-model:visible="isAddTaintModalVisible" title="添加节点Taint" @cancel="closeAddTaintModal" @ok="handleAddTaint(taintForm.nodeName)">
      <a-form :model="taintForm" layout="vertical">
        <a-form-item :rules="[{ required: true, message: '请选择节点名称' }]" label="节点名称" name="nodeName">
          <a-select v-model:value="taintForm.nodeName" placeholder="请选择节点名称">
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              {{ node.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :rules="[{ required: true, message: '请输入Taint YAML' }]" label="Taint YAML" name="taintYaml">
          <a-textarea
            v-model:value="taintForm.taintYaml"
            :rows="6"
            placeholder="示例：- key: &quot;example-key&quot;
  value: &quot;example-value&quot; 
  effect: &quot;NoSchedule&quot;"
          />
        </a-form-item>
        <a-button type="primary" @click="checkTaintYaml(taintForm.nodeName)" style="margin-bottom: 16px">
          检查YAML格式
        </a-button>
      </a-form>
    </a-modal>
    <!-- 删除Taint模态框 -->
    <a-modal v-model:visible="isDeleteTaintModalVisible" title="删除节点Taint" @cancel="closeDeleteTaintModal" @ok="handleDeleteTaint(deleteTaintForm.nodeName)">
      <a-form :model="deleteTaintForm" layout="vertical">
        <a-form-item :rules="[{ required: true, message: '请选择节点名称' }]" label="节点名称" name="nodeName">
          <a-select v-model:value="deleteTaintForm.nodeName" placeholder="请选择节点名称">
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              {{ node.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :rules="[{ required: true, message: '请输入Taint YAML' }]" label="Taint YAML" name="taintYaml">
          <a-textarea
            v-model:value="deleteTaintForm.taintYaml"
            :rows="6"
            placeholder="示例：- key: &quot;example-key&quot;
  value: &quot;example-value&quot; 
  effect: &quot;NoSchedule&quot;"
          />
        </a-form-item>
        <a-button type="primary" @click="checkTaintYaml(deleteTaintForm.nodeName)" style="margin-bottom: 16px">
          检查YAML格式
        </a-button>
      </a-form>
    </a-modal>
    <!-- 删除标签模态框 -->
    <a-modal v-model:visible="isDeleteLabelModalVisible" title="删除节点标签" @cancel="closeDeleteLabelModal"
      @ok="handleDeleteLabel">
      <a-form :model="deleteLabelForm" layout="vertical">
        <a-form-item :rules="[{ required: true, message: '请选择标签' }]" label="选择标签" name="label">
          <a-select v-model:value="deleteLabelForm.label" placeholder="请选择标签">
            <a-select-option v-for="(label, index) in labelOptions" :key="index" :value="label">
              {{ label }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
    <!-- 查看节点详情模态框 -->
    <a-modal v-model:visible="isViewDetailsModalVisible" title="节点详情" @cancel="closeViewDetailsModal" @ok="closeViewDetailsModal">
      <div v-if="selectedNodeDetails" class="node-details">
        <div class="node-details-header">
          <h3>{{ selectedNodeDetails.name }}</h3>
          <p><strong>节点状态:</strong> {{ selectedNodeDetails.status }}</p>
        </div>
        <div class="node-details-section">
          <h4>基本信息：</h4>
          <div class="info-item">
            <span><strong>IP 地址:</strong></span>
            <span>{{ selectedNodeDetails.ip }}</span>
          </div>
          <div class="info-item">
            <span><strong>创建时间:</strong></span>
            <span>{{ selectedNodeDetails.age }}</span>
          </div>
          <div class="info-item">
            <span><strong>CPU 请求:</strong></span>
            <span>{{ selectedNodeDetails.cpu_request_info }}</span>
          </div>
          <div class="info-item">
            <span><strong>内存请求:</strong></span>
            <span>{{ selectedNodeDetails.memory_request_info }}</span>
          </div>
        </div>
        <div class="node-details-section">
          <h4>资源信息：</h4>
          <div class="info-item">
            <span><strong>CPU 使用:</strong></span>
            <span>{{ selectedNodeDetails.cpu_usage_info }}</span>
          </div>
          <div class="info-item">
            <span><strong>内存使用:</strong></span>
            <span>{{ selectedNodeDetails.memory_usage_info }}</span>
          </div>
          <div class="info-item">
            <span><strong>磁盘存储:</strong></span>
            <span>{{ selectedNodeDetails.ephemeral_storage }}</span>
          </div>
        </div>
        <div class="node-details-section">
          <h4>标签</h4>
          <ul>
            <li v-for="(label, index) in selectedNodeDetails.labels" :key="index">
              <span>{{ label }}</span>
            </li>
          </ul>
        </div>
        <div class="node-details-section">
          <h4>节点污点：</h4>
          <ul v-if="selectedNodeDetails.taints && selectedNodeDetails.taints.length > 0">
            <li v-for="(taint, index) in selectedNodeDetails.taints" :key="index">
              <span>{{ taint }}</span>
            </li>
          </ul>
          <div v-else>
            <span>暂无污点</span>
          </div>
        </div>
        <div class="node-details-section">
          <h4>事件：</h4>
          <ul>
            <li v-for="(event, index) in selectedNodeDetails.events" :key="index" class="event-item">
              <p>
                <strong>{{ event.reason }}</strong>: {{ event.message }}
              </p>
              <p>
                <em>
                  {{ formatTime(event.first_time) }} -
                  {{ formatTime(event.last_time) }}
                </em>
              </p>
              <p><strong>类型:</strong> {{ event.type }}</p>
              <p><strong>组件:</strong> {{ event.component }}</p>
              <p><strong>对象:</strong> {{ event.object }}</p>
              <p><strong>发生次数:</strong> {{ event.count }}</p>
            </li>
          </ul>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import type { GetNodeDetailRes } from '#/api';

import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { message } from 'ant-design-vue';

import {
  addNodeLabelApi,
  deleteNodeLabelApi,
  getNodeDetailsApi,
  getNodeListApi,
  addNodeTaintApi,
  checkTaintYamlApi,
} from '#/api';

const selectedNodeDetails = ref<GetNodeDetailRes>();
const nodes = ref([]);
const searchText = ref('');
const isAddLabelModalVisible = ref(false);
const isAddTaintModalVisible = ref(false);
const isDeleteTaintModalVisible = ref(false);
const isViewDetailsModalVisible = ref(false);
const isDeleteLabelModalVisible = ref(false);
const route = useRoute();
// 删除标签模态框表单
const deleteLabelForm = reactive({
  label: [],
});

// 标签选项（从节点详情中获取）
const labelOptions = computed(() => {
  return selectedNodeDetails.value?.labels || [];
});

// 过滤后的数据
const filteredData = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  return nodes.value.filter((node: { name: string }) =>
    node.name.toLowerCase().includes(searchValue),
  );
});

// 表格列配置
const columns = [
  { dataIndex: 'name', key: 'name', title: '节点名称' },
  { dataIndex: 'cluster_id', key: 'cluster_id', title: '关联集群id' },
  { dataIndex: 'status', key: 'status', title: '节点状态' },
  { dataIndex: 'ip', key: 'ip', title: 'IP 地址' },
  { dataIndex: 'roles', key: 'roles', title: '角色' },
  { dataIndex: 'age', key: 'age', title: '创建时间' },
  { key: 'action', slots: { customRender: 'action' }, title: '操作' },
];

// 标签添加表单
const labelForm = reactive({
  key: '',
  nodeName: '',
  value: '',
});

// 添加Taint表单
const taintForm = reactive({
  nodeName: '',
  taintYaml: '',
});

// 删除Taint表单
const deleteTaintForm = reactive({
  nodeName: '',
  taintYaml: '',
});

// 获取节点列表
const getNodes = async () => {
  try {
    const cluster_id = Number(route.query.cluster_id);
    if (isNaN(cluster_id)) {
      throw new Error('无效的集群ID');
    }
    const res = await getNodeListApi(cluster_id);
    nodes.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取节点数据失败');
  }
};

// 获取节点详情
const handleViewDetails = async (data: any) => {
  try {
    const res = await getNodeDetailsApi(data.name, data.cluster_id);
    selectedNodeDetails.value = res;
    isViewDetailsModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取节点详情失败');
  }
};

// 添加节点标签
const handleAddLabel = async () => {
  const { key, nodeName, value } = labelForm;
  const cluster_id = Number(route.query.cluster_id);
    if (isNaN(cluster_id)) {
      throw new Error('无效的集群ID');
    }
  const mod_type = 'add';
  try {
    // 标签传递格式为 key, val, key, val
    await addNodeLabelApi({
      cluster_id,
      labels: [key, value], // 标签是交替格式 key, val
      mod_type,
      node_name: nodeName,
    });
    message.success('标签添加成功');
    // 清除表单数据
    labelForm.nodeName = '';
    labelForm.key = '';
    labelForm.value = '';
    getNodes();
    isAddLabelModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '标签添加失败');
  }
};

const handleDeleteLabel = async () => {
  const selectedLabel = deleteLabelForm.label; // "key=val"
  const cluster_id = Number(route.query.cluster_id);
    if (isNaN(cluster_id)) {
      throw new Error('无效的集群ID');
    }
  const mod_type = 'del';
  if (!selectedLabel) {
    message.error('请选择一个标签');
    return;
  }

  const [key, val] = selectedLabel.split('=');

  if (!key || !val) {
    message.error('标签格式不正确');
    return;
  }

  try {
    await deleteNodeLabelApi({
      cluster_id,
      labels: [key, val],
      mod_type,
      node_name: selectedNodeDetails.value?.name,
    });
    message.success('标签删除成功');
    closeDeleteLabelModal();
  } catch (error: any) {
    message.error(error.message || '删除标签失败');
  }
};

// 添加Taint
const handleAddTaint = async (nodeName: string) => {
  try {
    await addNodeTaintApi({
      cluster_id: Number(route.query.cluster_id),
      mod_type: 'add', 
      node_name: nodeName,
      taint_yaml: taintForm.taintYaml,
    });
    message.success('Taint添加成功');
    // 清除表单数据
    taintForm.nodeName = '';
    taintForm.taintYaml = '';
    // 关闭模态框
    isAddTaintModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '添加Taint失败');
  }
};

// 删除Taint
const handleDeleteTaint = async (nodeName: string) => {
  try {
    await addNodeTaintApi({
      cluster_id: Number(route.query.cluster_id),
      mod_type: 'del',
      node_name: nodeName,
      taint_yaml: deleteTaintForm.taintYaml,
    });
    message.success('Taint删除成功');
    // 清除表单数据
    deleteTaintForm.nodeName = '';
    deleteTaintForm.taintYaml = '';
    // 关闭模态框
    isDeleteTaintModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '删除Taint失败');
  }
};

// 检查Taint YAML格式
const checkTaintYaml = async (nodeName: string) => {
  try {
    await checkTaintYamlApi({
      cluster_id: Number(route.query.cluster_id),
      node_name: nodeName,
      taint_yaml: taintForm.taintYaml,
    });
    message.success('YAML格式校验通过');
  } catch (error: any) {
    message.error(error.message || 'YAML格式校验失败');
  }
};

// 关闭添加标签模态框
const closeAddLabelModal = () => {
  isAddLabelModalVisible.value = false;
};
// 关闭添加Taint模态框
const closeAddTaintModal = () => {
  isAddTaintModalVisible.value = false;
};
// 关闭删除标签模态框
const closeDeleteLabelModal = () => {
  isDeleteLabelModalVisible.value = false;
};

// 关闭查看详情模态框
const closeViewDetailsModal = () => {
  isViewDetailsModalVisible.value = false;
};

// 弹出删除标签模态框
const showDeleteLabelModal = (record: any) => {
  selectedNodeDetails.value = record;
  isDeleteLabelModalVisible.value = true;
};

// 关闭删除Taint模态框
const closeDeleteTaintModal = () => {
  isDeleteTaintModalVisible.value = false;
};

// 时间格式化
const formatTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()} ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}`;
};

// 初始化数据
onMounted(() => {
  getNodes();
});
</script>

<style scoped>
/* 样式 */
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
}

.action-buttons {
  display: flex;
}

.node-details-section {
  margin-top: 16px;
}

.node-details-header h3 {
  margin: 0;
}

.info-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}
.custom-toolbar {
  padding: 6px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  align-items: center;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 16px;
}
</style>
