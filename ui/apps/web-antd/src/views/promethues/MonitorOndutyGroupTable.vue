<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">值班表</h2>
      <div class="page-description">管理和查看值班人员安排及换班记录</div>
    </div>

    <!-- 值班信息卡片 -->
    <div class="dashboard-card custom-toolbar">
      <div class="duty-info">
        <div class="info-item">
          <span class="info-label">值班组名称：</span>
          <span class="info-value">{{ dutyGroupName }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">总值班人数：</span>
          <span class="info-value">{{ totalOnDutyUsers }}</span>
        </div>
      </div>
    </div>

    <!-- 日历展示区 -->
    <div class="dashboard-card calendar-container">
      <div class="calendar">
        <div
          class="calendar-day prev-month"
          v-for="day in previousMonthDays"
          :key="'prev-' + day.date"
        >
          <div class="day-header">
            <div class="day-number">{{ day.date.split('-')[2] }}</div>
            <div class="day-weekday">{{ day.weekday }}</div>
          </div>
          <div class="day-content">
            <div class="day-user" :class="{'no-user': !day.user_id}">
              {{ day.user_id ? `值班人: ${createUserName}` : '没有找到值班人' }}
            </div>
          </div>
        </div>

        <div
          class="calendar-day"
          v-for="day in daysInMonth"
          :key="day.date"
          @click="isCurrentMonth(day.date) ? openSwapModal(day) : null"
          :class="{
            'has-user': day.user_id,
            'no-user': !day.user_id,
            'disabled': !isCurrentMonth(day.date),
          }"
        >
          <div class="day-header">
            <div class="day-number">{{ day.date.split('-')[2] }}</div>
            <div class="day-weekday">{{ day.weekday }}</div>
          </div>
          <div class="day-content">
            <div class="day-user" :class="{'no-user': !day.user_id}">
              {{ day.user_id ? `值班人: ${createUserName}` : '没有找到值班人' }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 换班模态框 -->
    <a-modal 
      title="换班记录" 
      v-model:visible="isSwapModalVisible" 
      @ok="handleSwap" 
      @cancel="closeSwapModal"
      :width="500"
      class="custom-modal"
    >
      <a-form ref="swapFormRef" :model="swapForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">换班信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="调换日期" name="date">
                <a-input v-model:value="swapForm.date" disabled />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="值班人ID" name="on_duty_user_id" :rules="[{ required: true, message: '请输入新值班人ID' }]">
                <a-input-number v-model:value="swapForm.on_duty_user_id" placeholder="请输入新值班人ID" class="full-width" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { message } from 'ant-design-vue';
import { getOnDutyApi, getOnDutyFuturePlanApi, createOnDutyChangeApi } from '#/api/core/prometheus_onduty'; 

const dutyGroupName = ref('');
const createUserName = ref('');
const totalOnDutyUsers = ref(0);
const isSwapModalVisible = ref(false);
const swapForm = reactive({
  date: '',
  on_duty_group_id: 0,
  origin_user_id: 0,
  on_duty_user_id: 0,
});

interface Day {
  date: string;
  weekday: string;
  user_id: number | null;
  create_user_name: string;
}
const daysInMonth = ref<Day[]>([]);
const previousMonthDays = ref<Day[]>([]);
const route = useRoute();

const getWeekday = (date: Date) => {
  const weekdays = [
    '星期日',
    '星期一',
    '星期二',
    '星期三',
    '星期四',
    '星期五',
    '星期六',
  ];
  return weekdays[date.getDay()];
};

const isCurrentMonth = (dateStr: string | number | Date) => {
  const date = new Date(dateStr);
  const currentMonth = new Date();
  return (
    date.getFullYear() === currentMonth.getFullYear() &&
    date.getMonth() === currentMonth.getMonth()
  );
};

const fetchDutyGroups = async () => {
  try {
    const id = parseInt(route.query.id as string) || 0;
    const response = await getOnDutyApi(id);
    if (response) {
      dutyGroupName.value = response.name || '未知值班组';
      totalOnDutyUsers.value = response.members ? response.members.length : 0;
      createUserName.value = response.create_user_name || '';
    } else {
      message.error('返回数据格式不正确');
    }
  } catch (error: any) {
    console.error('获取值班组信息失败:', error);
    message.error(error.message || '获取值班组信息失败');
  }
};

const fetchDutySchedule = async () => {
  const currentMonth = new Date();
  currentMonth.setDate(1);

  const startTime = new Date(Date.UTC(currentMonth.getFullYear(), currentMonth.getMonth(), 1));
  const endTime = new Date(Date.UTC(currentMonth.getFullYear(), currentMonth.getMonth() + 1, 0));

  const previousMonthStartTime = new Date(Date.UTC(currentMonth.getFullYear(), currentMonth.getMonth() - 1, 1));
  const previousMonthEndTime = new Date(Date.UTC(currentMonth.getFullYear(), currentMonth.getMonth(), 0));

  const id = parseInt(route.query.id as string) || 0;

  try {
    const currentMonthResponse = await getOnDutyFuturePlanApi({
      id: id,
      start_time: startTime.toISOString().split('T')[0] as string,
      end_time: endTime.toISOString().split('T')[0] as string,
    });

    const previousMonthResponse = await getOnDutyFuturePlanApi({
      id: id,
      start_time: previousMonthStartTime.toISOString().split('T')[0] as string,
      end_time: previousMonthEndTime.toISOString().split('T')[0] as string,
    });

    const currentDutyDetails = currentMonthResponse.details || [];
    const previousDutyDetails = previousMonthResponse.details || [];

    daysInMonth.value = [];

    for (let d = new Date(startTime); d <= endTime; d.setDate(d.getDate() + 1)) {
      const currentDate = new Date(d);
      const dateStr = currentDate.toLocaleDateString('en-CA');
      const detail = currentDutyDetails.find((detail: { date: string; }) => detail.date === dateStr) || {};
      daysInMonth.value.push({
        date: dateStr,
        weekday: getWeekday(currentDate) as string,
        user_id: detail.user?.id || null,
        create_user_name: createUserName.value || '',
      });
    }

    previousMonthDays.value = [];
    const firstDayOfCurrentMonth = new Date(currentMonth.getFullYear(), currentMonth.getMonth(), 1);
    const firstDayWeekday = firstDayOfCurrentMonth.getDay();

    const lastDayOfLastMonth = new Date(previousMonthEndTime);

    for (let j = 0; j < firstDayWeekday; j++) {
      const day = new Date(lastDayOfLastMonth);
      day.setDate(lastDayOfLastMonth.getDate() - j);
      if (day.getDay() === 0) break; // 如果是星期日则停止
      const dateStr = day.toLocaleDateString('en-CA');
      const detail = previousDutyDetails.find((detail: { date: string; }) => detail.date === dateStr) || {};
      previousMonthDays.value.unshift({
        date: dateStr,
        weekday: getWeekday(day) as string,
        user_id: detail.user?.id || null,
        create_user_name: createUserName.value || '',
      });
    }
  } catch (error: any) {
    message.error(error.message || '获取值班情况失败');
  }
};

const openSwapModal = (day: { date: string }) => {
  swapForm.date = day.date;
  isSwapModalVisible.value = true;
};

const handleSwap = async () => {
  const payload = {
    on_duty_group_id: parseInt(route.query.id as string),
    date: swapForm.date,
    origin_user_id: getOriginUserId(swapForm.date) as number, // 动态获取原值班人ID
    on_duty_user_id: swapForm.on_duty_user_id,
  };
  
  try {
    await createOnDutyChangeApi(payload);
    message.success('换班成功');
    closeSwapModal();
    fetchDutySchedule(); // 刷新值班表
  } catch (error: any) {
    message.error(error.message || '换班失败');
  }
};

// 获取指定日期的原值班人ID
const getOriginUserId = (date: string) => {
  const day = daysInMonth.value.find(d => d.date === date);
  return day ? day.user_id : null;
};

const closeSwapModal = () => {
  isSwapModalVisible.value = false;
};

onMounted(() => {
  fetchDutyGroups();
  fetchDutySchedule();
});
</script>

<style scoped>
.monitor-page {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 8px;
}

.page-description {
  color: #666;
  font-size: 14px;
}

.dashboard-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  padding: 20px;
  margin-bottom: 24px;
  transition: all 0.3s;
}

.custom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.duty-info {
  display: flex;
  gap: 24px;
  align-items: center;
}

.info-item {
  display: flex;
  align-items: center;
}

.info-label {
  font-weight: 500;
  color: #555;
  margin-right: 8px;
}

.info-value {
  font-weight: 600;
  color: #1890ff;
}

.calendar-container {
  padding: 24px;
}

.calendar {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 16px;
}

.calendar-day {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: all 0.3s ease;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  min-height: 120px;
  border: 1px solid #f0f0f0;
}

.calendar-day:hover {
  transform: translateY(-3px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.calendar-day.has-user {
  border-left: 4px solid #52c41a;
}

.calendar-day.no-user {
  border-left: 4px solid #ff4d4f;
}

.calendar-day.prev-month {
  background: #f9f9f9;
  color: #999;
  cursor: default;
}

.calendar-day.disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.day-header {
  padding: 8px 12px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.day-number {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.day-weekday {
  font-size: 12px;
  color: #888;
}

.day-content {
  padding: 12px;
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.day-user {
  font-size: 13px;
  line-height: 1.5;
  padding: 6px 10px;
  border-radius: 4px;
  background: #e6f7ff;
  color: #1890ff;
  margin-bottom: 8px;
}

.day-user.no-user {
  background: #fff2f0;
  color: #ff4d4f;
}

/* 模态框样式 */
:deep(.custom-modal .ant-modal-content) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.custom-modal .ant-modal-header) {
  padding: 20px 24px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

:deep(.custom-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.custom-modal .ant-modal-body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

:deep(.custom-modal .ant-modal-footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 表单样式 */
.custom-form {
  width: 100%;
}

.form-section {
  margin-bottom: 28px;
  padding: 0;
  position: relative;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 16px;
  padding-left: 12px;
  border-left: 4px solid #1890ff;
}

:deep(.custom-form .ant-form-item-label > label) {
  font-weight: 500;
  color: #333;
}

.full-width {
  width: 100%;
}
</style>