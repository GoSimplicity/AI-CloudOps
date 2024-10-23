<template>
    <div>
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model="searchText"
            placeholder="请输入发送组名称"
            style="width: 200px; margin-right: 16px;"
          />
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd">新增发送组</a-button>
        </div>
      </div>
  
      <!-- 发送组列表表格 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- 是否启用列 -->
        <template #enable="{ record }">
          <!-- 根据 enable 字段值显示是否启用 -->
          {{ record.enable === 1 ? '启用' : '禁用' }}
        </template>
        <!-- 是否发送恢复消息列 -->
        <template #sendResolved="{ record }">
          <!-- 根据 sendResolved 字段值显示是否发送恢复消息 -->
          {{ record.sendResolved === 1 ? '是' : '否' }}
        </template>
        <!-- 升级人列表列 -->
        <template #upgradeUsers="{ record }">
          <!-- 显示升级人列表，使用逗号分隔 -->
          {{ record.upgradeUsers.join(', ') }}
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <!-- 编辑按钮，点击后调用 handleEdit 方法 -->
            <a-button type="link" @click="handleEdit(record)">编辑发送组</a-button>
            <!-- 删除按钮，点击后调用 handleDelete 方法 -->
            <a-button type="link" danger @click="handleDelete(record)">删除发送组</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  
  // 定义数据类型
  interface SendGroup {
    id: number; // 发送组ID
    key: string; // 唯一标识符，用于区分不同的发送组
    name: string; // 发送组名称
    nameZh: string; // 发送组中文名称
    enable: number; // 是否启用发送组：1启用，2禁用
    poolName: string; // 关联的AlertManager实例名称
    createUserName: string; // 创建该发送组的用户名称
    repeatInterval: string; // 默认重复发送时间
    createTime: string; // 发送组的创建时间
    onDutyGroupName: string; // 关联的值班组名称
    sendResolved: number; // 是否发送恢复消息：1发送，2不发送
    upgradeUsers: string[]; // 升级人列表
  }
  
  // 示例数据
  const data = reactive<SendGroup[]>([
    {
      id: 1,
      key: '1',
      name: 'default-send-group',
      nameZh: '默认发送组',
      enable: 1,
      poolName: '默认AlertManager实例',
      createUserName: '管理员',
      repeatInterval: '5m',
      createTime: '2023-10-01 10:00:00',
      onDutyGroupName: '默认值班组',
      sendResolved: 1,
      upgradeUsers: ['用户A', '用户B'],
    },
    // 可添加更多示例数据
  ]);
  
  // 搜索文本
  const searchText = ref('');
  // 过滤后的数据，通过 computed 属性动态计算
  const filteredData = computed(() => {
    // 将搜索文本转换为小写并去除前后空格
    const searchValue = searchText.value.trim().toLowerCase();
    // 根据发送组名称或中文名称进行过滤
    return data.filter(item => item.name.toLowerCase().includes(searchValue) || item.nameZh.toLowerCase().includes(searchValue));
  });
  
  // 表格列配置
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: '发送组名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '发送组中文名称',
      dataIndex: 'nameZh',
      key: 'nameZh',
    },
    {
      title: '是否启用',
      dataIndex: 'enable',
      key: 'enable',
      slots: { customRender: 'enable' }, // 使用自定义插槽来渲染是否启用
    },
    {
      title: '关联实例名称',
      dataIndex: 'poolName',
      key: 'poolName',
    },
    {
      title: '重复发送时间',
      dataIndex: 'repeatInterval',
      key: 'repeatInterval',
    },
    {
      title: '关联值班组',
      dataIndex: 'onDutyGroupName',
      key: 'onDutyGroupName',
    },
    {
      title: '是否发送恢复消息',
      dataIndex: 'sendResolved',
      key: 'sendResolved',
      slots: { customRender: 'sendResolved' }, // 使用自定义插槽来渲染是否发送恢复消息
    },
    {
      title: '升级人列表',
      dataIndex: 'upgradeUsers',
      key: 'upgradeUsers',
      slots: { customRender: 'upgradeUsers' }, // 使用自定义插槽来渲染升级人列表
    },
    {
      title: '创建者',
      dataIndex: 'createUserName',
      key: 'createUserName',
    },
    {
      title: '创建时间',
      dataIndex: 'createTime',
      key: 'createTime',
    },
    {
      title: '操作',
      key: 'action',
      slots: { customRender: 'action' }, // 使用自定义插槽来渲染操作按钮
    },
  ];
  
  // 处理新增发送组
  const handleAdd = () => {
    // 弹出信息，表示点击了新增发送组按钮
    message.info('点击了新增发送组按钮');
  };
  
  // 处理编辑发送组
  const handleEdit = (record: SendGroup) => {
    // 弹出信息，表示编辑了指定的发送组
    message.info(`编辑发送组 "${record.nameZh}"`);
  };
  
  // 处理删除发送组
  const handleDelete = (record: SendGroup) => {
    // 弹出确认对话框，提示是否确认删除发送组
    Modal.confirm({
      title: '确认删除',
      content: `您确定要删除发送组 "${record.nameZh}" 吗？`,
      onOk: () => {
        // 查找要删除的数据索引
        const index = data.findIndex(item => item.key === record.key);
        if (index !== -1) {
          // 删除指定索引的数据
          data.splice(index, 1);
          // 弹出删除成功的消息
          message.success(`发送组 "${record.nameZh}" 已删除`);
        }
      },
    });
  };
  </script>
  
  <style scoped>
  .custom-toolbar {
    padding: 8px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }
  
  .search-filters {
    display: flex;
    align-items: center;
  }
  </style>
  