<template>
    <div>
      <a-page-header title="排班表" :breadcrumb="{ routes: breadcrumb }" />
      <a-card>
        <a-table :columns="columns" :data-source="scheduleData" row-key="key">
          <!-- 成员列 -->
          <template #members="{ record }">
            <a-tag v-for="member in record.members" :key="member">{{ member }}</a-tag>
          </template>
        </a-table>
      </a-card>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { ref } from 'vue';
  import { useRoute } from 'vue-router';
  
  // 使用 Vue Router
  const route = useRoute();
  const onDutyGroupId = route.params.id;
  
  // 面包屑导航
  const breadcrumb = ref([
    { path: '/', breadcrumbName: '首页' },
    { path: '/on-duty', breadcrumbName: '值班组管理' },
    { path: `/schedule/${onDutyGroupId}`, breadcrumbName: '排班表' },
  ]);
  
  // 定义数据类型
  interface Schedule {
    key: string;
    members: string[]; // 成员列表
    date: string; // 日期
    currentUser: string; // 当前值班人员
  }
  
  // 示例数据
  const scheduleData = ref<Schedule[]>([
    {
      key: '1',
      members: ['张三', '李四', '王五'],
      date: '2023-10-01',
      currentUser: '李四',
    },
    {
      key: '2',
      members: ['张三', '李四', '王五'],
      date: '2023-10-02',
      currentUser: '王五',
    },
    // 可添加更多示例数据
  ]);
  
  // 表格列配置
  const columns = [
    {
      title: '日期',
      dataIndex: 'date',
      key: 'date',
    },
    {
      title: '值班人员',
      dataIndex: 'currentUser',
      key: 'currentUser',
    },
    {
      title: '成员列表',
      key: 'members',
      slots: { customRender: 'members' },
    },
  ];
  </script>
  
  <style scoped>
  .page-header {
    margin-bottom: 16px;
  }
  </style>
  