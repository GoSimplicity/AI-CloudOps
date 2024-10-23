<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入值班组名称"
          style="width: 200px; margin-right: 16px;"
        />
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">新增换班记录</a-button>
      </div>
    </div>

    <!-- 值班换班记录列表表格 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="key">
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleEdit(record)">编辑换班记录</a-button>
          <a-button type="link" @click="viewSchedule(record)">查看排班表</a-button>
          <a-button type="link" danger @click="handleDelete(record)">删除换班记录</a-button>
        </a-space>
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from 'vue';
import { message, Modal } from 'ant-design-vue';

// 定义数据类型
interface OnDutyChange {
  key: string; // 唯一标识符，用于区分不同的值班记录
  onDutyGroupId: number; // 值班组ID
  userId: number; // 创建该换班记录的用户ID
  date: string; // 计划哪一天进行换班的日期
  targetUserName: string; // 换班后值班人员用户名
  members: string[]; // 成员列表
  shiftDays: number; // 轮班周期（天）
  currentUserName: string; // 当前值班人员用户名
  createUserName: string; // 创建者用户名
  createTime: string; // 创建时间
}

// 示例数据
const data = reactive<OnDutyChange[]>([
  {
    key: '1',
    onDutyGroupId: 101,
    userId: 1,
    date: '2023-10-01',
    targetUserName: '李四',
    members: ['张三', '李四', '王五'],
    shiftDays: 7,
    currentUserName: '李四',
    createUserName: '管理员',
    createTime: '2023-09-28 10:00:00',
  },
  // 可添加更多示例数据
]);

// 搜索文本
const searchText = ref('');
// 过滤后的数据，通过 computed 属性动态计算
const filteredData = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  return data.filter(item => item.targetUserName.toLowerCase().includes(searchValue) || item.currentUserName.toLowerCase().includes(searchValue));
});

// 表格列配置
const columns = [
  {
    title: '值班组ID',
    dataIndex: 'onDutyGroupId',
    key: 'onDutyGroupId',
  },
  {
    title: '换班日期',
    dataIndex: 'date',
    key: 'date',
  },
  {
    title: '换班后人员',
    dataIndex: 'targetUserName',
    key: 'targetUserName',
  },
  {
    title: '成员列表',
    dataIndex: 'members',
    key: 'members',
    customRender: ({ text }: { text: string[] }) => text.join(', '),
  },
  {
    title: '轮班周期（天）',
    dataIndex: 'shiftDays',
    key: 'shiftDays',
  },
  {
    title: '当前值班人员',
    dataIndex: 'currentUserName',
    key: 'currentUserName',
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

// 处理新增换班记录
const handleAdd = () => {
  // 这里可以打开一个对话框，填写新换班记录的信息
  message.info('点击了新增换班记录按钮');
};

// 处理编辑换班记录
const handleEdit = (record: OnDutyChange) => {
  // 这里可以打开一个对话框，编辑换班记录的信息
  message.info(`编辑换班记录，值班组ID: ${record.onDutyGroupId}`);
};

// 处理查看排班表
const viewSchedule = (record: OnDutyChange) => {
  // 这里可以跳转到查看排班表的页面或弹出排班表的对话框
  message.info(`查看值班组 ${record.onDutyGroupId} 的排班表`);
};

// 处理删除换班记录
const handleDelete = (record: OnDutyChange) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除换班记录吗？值班组ID: ${record.onDutyGroupId}`,
    onOk: () => {
      // 查找要删除的数据索引
      const index = data.findIndex(item => item.key === record.key);
      if (index !== -1) {
        // 删除指定索引的数据
        data.splice(index, 1);
        message.success('换班记录已删除');
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
