<template>
  <a-layout class="layout">
    <a-layout-content class="content">
      <div class="card-container">
        <a-card
          class="download-card"
          bordered
          hoverable
          title="Prometheus 配置文件下载"
        >
          <a-form layout="vertical" @submit.prevent>
            <a-form-item
              label="目标服务器 IP"
              :validate-status="ipError ? 'error' : 'success'"
              :help="ipError || '请输入目标服务器的 IP 地址'"
            >
              <a-input
                v-model:value="ip"
                placeholder="例如：192.168.1.100"
              />
            </a-form-item>

            <a-space direction="vertical" size="middle" class="button-group">
              <a-button
                type="primary"
                :disabled="!isIpValid"
                :loading="loading.prometheus"
                @click="downloadConfig('prometheus')"
                block
              >
                下载 Prometheus 配置文件
              </a-button>
              <a-button
                type="primary"
                :disabled="!isIpValid"
                :loading="loading.prometheus_alert"
                @click="downloadConfig('prometheus_alert')"
                block
              >
                下载 Prometheus 告警配置文件
              </a-button>
              <a-button
                type="primary"
                :disabled="!isIpValid"
                :loading="loading.prometheus_record"
                @click="downloadConfig('prometheus_record')"
                block
              >
                下载 Prometheus 记录配置文件
              </a-button>
              <a-button
                type="primary"
                :disabled="!isIpValid"
                :loading="loading.alertManager"
                @click="downloadConfig('alertManager')"
                block
              >
                下载 AlertManager 配置文件
              </a-button>
            </a-space>
          </a-form>
        </a-card>
      </div>
    </a-layout-content>
  </a-layout>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { message } from 'ant-design-vue';

// IP 地址
const ip = ref('');
const ipError = ref('');
const isIpValid = ref(false);

// 加载状态
const loading = ref({
  prometheus: false,
  prometheus_alert: false,
  prometheus_record: false,
  alertManager: false,
});

// 验证 IP 地址格式
const validateIp = () => {
  const ipRegex = /^(25[0-5]|2[0-4]\d|[0-1]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[0-1]?\d{1,2})){3}$/;
  if (!ip.value) {
    console.log(ip)
    ipError.value = 'IP 地址不能为空';
    isIpValid.value = false;
  } else if (!ipRegex.test(ip.value)) {
    ipError.value = '请输入有效的 IP 地址';
    isIpValid.value = false;
  } else {
    ipError.value = '';
    isIpValid.value = true;
  }
};

// 监听 IP 地址的变化并验证
watch(ip, validateIp, { immediate: true });

// 下载配置文件
const downloadConfig = async (type: string) => {
  if (!isIpValid.value) {
    message.error('请先输入有效的 IP 地址');
    return;
  }

  let url = '';
  switch (type) {
    case 'prometheus':
      url = `/api/monitor/prometheus_configs/prometheus?ip=${ip.value}`;
      break;
    case 'prometheus_alert':
      url = `/api/monitor/prometheus_configs/prometheus_alert?ip=${ip.value}`;
      break;
    case 'prometheus_record':
      url = `/api/monitor/prometheus_configs/prometheus_record?ip=${ip.value}`;
      break;
    case 'alertManager':
      url = `/api/monitor/prometheus_configs/alertManager?ip=${ip.value}`;
      break;
    default:
      message.error('未知的配置类型');
      return;
  }

  try {
    loading.value[type] = true;
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error('网络响应不是OK');
    }
    const blob = await response.blob();
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = `${type}_config.yaml`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(downloadUrl);
    message.success(`${type} 配置文件下载开始`);
  } catch (error) {
    message.error('下载配置文件失败');
    console.error(error);
  } finally {
    loading.value[type] = false;
  }
};
</script>

<style scoped>
.layout {
  min-height: 100vh;
}

.content {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 50px;
}

.card-container {
  width: 100%;
  max-width: 500px;
}

.download-card {
  padding: 24px;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.button-group .ant-btn {
  height: 50px;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.footer {
  text-align: center;
  padding: 20px;
  background: #001529;
}
</style>
