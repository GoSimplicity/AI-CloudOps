<template>
  <a-layout class="calendar-layout">
    <a-layout-header class="header">
      <div class="header-title">值班表</div>
      <div class="header-info">
        <span>值班组名称：{{ dutyGroupName }};</span>
        <span>总值班人数：{{ totalOnDutyUsers }}</span>
      </div>
    </a-layout-header>

    <a-layout-content class="calendar-content">
      <a-card bordered="{false}" class="calendar-card">
        <div class="calendar">
          <div
            class="day prev-month"
            v-for="day in previousMonthDays"
            :key="'prev-' + day.date"
          >
            <div class="day-number">{{ day.date }}</div>
            <div class="day-weekday">{{ day.weekday }}</div>
            <div class="day-user">
              {{ day.user_id ? `值班人: ${createUserName}` : '没有找到值班人' }}
            </div>
          </div>

          <div
            class="day"
            v-for="day in daysInMonth"
            :key="day.date"
            @click="isCurrentMonth(day.date) ? openSwapModal(day) : null"
            :class="{
              'has-user': day.user_id,
              'no-user': !day.user_id,
              disabled: !isCurrentMonth(day.date),
            }"
          >
            <div class="day-number">{{ day.date }}</div>
            <div class="day-weekday">{{ day.weekday }}</div>
            <div class="day-user">
              {{ day.user_id ? `值班人: ${createUserName}` : '没有找到值班人' }}
            </div>
          </div>
        </div>
      </a-card>

      <a-modal
        title="换班记录"
        v-model:visible="isSwapModalVisible"
        @ok="handleSwap"
        @cancel="closeSwapModal"
      >
        <a-form layout="vertical">
          <a-form-item label="值班人ID" required>
            <a-input-number
              v-model:value="swapForm.on_duty_user_id"
              placeholder="请输入新值班人ID"
            />
          </a-form-item>
          <p>调换日期: {{ swapForm.date }}</p>
        </a-form>
      </a-modal>
    </a-layout-content>
  </a-layout>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { message } from 'ant-design-vue';
import { getOnDutyApi, getOnDutyFuturePlanApi, createOnDutyChangeApi } from '#/api'; 

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
.calendar-layout {
  height: 100vh;
  background-color: var(--ant-layout-background);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  background: var(--ant-layout-header-background);
}

.header-title {
  font-size: 1.5em;
}

.header-info {
  font-size: 1em;
  color: var(--ant-text-color-secondary);
}

.calendar-content {
  padding: 20px;
}

.calendar-card {
  border-radius: 8px;
  padding: 20px;
}

.calendar {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 10px;
}

.day {
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 10px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
  transition: background-color 0.3s, transform 0.2s, box-shadow 0.3s;
}

.day:hover {
  transform: scale(1.03);
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.15);
}

.prev-month {
  background-color: #e0e0e0;
  color: #757575;
}

.has-user {
  background-color: #a5d6a7;
  color: #2e7d32;
}

.no-user {
  background-color: #ffccbc;
  color: #d84315;
}

.disabled {
  background-color: #e0e0e0; /* 灰色背景 */
  color: #757575; /* 灰色文字 */
  cursor: not-allowed; /* 禁止指针 */
}

.day-number {
  font-size: 1.1em;
  font-weight: bold;
}

.day-weekday {
  font-size: 0.85em;
  color: #888;
}

.day-user {
  margin-top: 8px;
  font-size: 0.85em;
}
</style>
