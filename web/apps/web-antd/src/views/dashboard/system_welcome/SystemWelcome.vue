<template>
  <div class="welcome-page">
    <!-- 顶部欢迎语 -->
    <div class="welcome-header">
      <h1>欢迎回来，管理员！</h1>
      <p>今天是 {{ currentDate }}</p>
    </div>

    <!-- 统计卡片 -->
    <div class="statistics-cards">
      <a-row gutter="16">
        <a-col :span="6">
          <a-card class="stat-card">
            <a-icon type="desktop" class="stat-icon" />
            <div class="card-content">
              <div class="card-title">机器数量</div>
              <div class="card-number">{{ machineCount }}</div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stat-card">
            <a-icon type="user" class="stat-icon" />
            <div class="card-content">
              <div class="card-title">在线用户</div>
              <div class="card-number">{{ onlineUsers }}</div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stat-card">
            <a-icon type="message" class="stat-icon" />
            <div class="card-content">
              <div class="card-title">未处理告警</div>
              <div class="card-number">{{ newMessages }}</div>
            </div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stat-card">
            <a-icon type="schedule" class="stat-icon" />
            <div class="card-content">
              <div class="card-title">待处理工单</div>
              <div class="card-number">{{ pendingTasks }}</div>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <!-- 最近上线用户列表 -->
    <div class="recent-users">
      <a-card title="最近上线用户">
        <a-list :data-source="recentUsers">
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar :src="item.avatar" />
                </template>
                <template #title>
                  {{ item.name }}
                </template>
                <template #description>
                  上线时间：{{ item.loginTime }}
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </a-card>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';

const currentDate = ref(new Date().toLocaleDateString());

// 统计数据
const machineCount = ref(42);
const onlineUsers = ref(128);
const newMessages = ref(7);
const pendingTasks = ref(5);

// 最近上线用户数据
const recentUsers = ref([
  {
    name: '用户A',
    avatar: 'https://randomuser.me/api/portraits/men/1.jpg',
    loginTime: '2024-9-14 10:00',
  },
  {
    name: '用户B',
    avatar: 'https://randomuser.me/api/portraits/women/2.jpg',
    loginTime: '2024-9-14 09:45',
  },
  {
    name: '用户C',
    avatar: 'https://randomuser.me/api/portraits/men/3.jpg',
    loginTime: '2024-9-14 09:30',
  },
  // 更多用户
]);
</script>

<style scoped>
.welcome-page {
  padding: 24px;
}

.welcome-header {
  text-align: center;
  margin-bottom: 24px;
}

.welcome-header h1 {
  font-size: 32px;
  margin-bottom: 8px;
}

.welcome-header p {
  font-size: 16px;
  color: #888;
}

.statistics-cards .stat-card {
  display: flex;
  align-items: center;
  padding: 24px;
}

.stat-icon {
  font-size: 48px;
  color: #1890ff;
}

.card-content {
  margin-left: 16px;
}

.card-title {
  font-size: 16px;
  color: #888;
}

.card-number {
  font-size: 24px;
  font-weight: bold;
}

.recent-users {
  margin-top: 24px;
}

.recent-users .ant-list-item {
  padding: 16px 0;
}

.recent-users .ant-list-item-meta-title {
  font-size: 16px;
}

.recent-users .ant-list-item-meta-description {
  color: #888;
}
</style>
