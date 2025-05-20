<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">配置文件下载</h2>
      <div class="page-description">根据目标服务器IP下载Prometheus和AlertManager配置文件</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card">
      <a-form layout="vertical" @submit.prevent>
        <a-form-item
          label="目标服务器 IP"
          :validate-status="ipError ? 'error' : 'success'"
          :help="ipError || '请输入目标服务器的 IP 地址'"
        >
          <div class="search-filters">
            <a-input
              v-model:value="ip"
              placeholder="例如：192.168.1.100"
              class="search-input"
            >
              <template #prefix>
                <Icon icon="mdi:ip-network" class="search-icon" />
              </template>
            </a-input>
            <a-button type="primary" class="action-button" @click="validateIp">
              <template #icon>
                <CheckOutlined />
              </template>
              验证IP
            </a-button>
            <a-button class="action-button reset-button" @click="resetIp">
              <template #icon>
                <ReloadOutlined />
              </template>
              重置
            </a-button>
          </div>
        </a-form-item>
      </a-form>
    </div>

    <!-- 下载按钮卡片 -->
    <div class="dashboard-card">
      <div class="section-title">配置文件下载</div>
      <div class="download-buttons">
        <a-button
          type="primary"
          :disabled="!isIpValid"
          :loading="loading.prometheus"
          @click="downloadConfig('prometheus')"
          class="download-button"
        >
          <template #icon>
            <Icon icon="simple-icons:prometheus" />
          </template>
          Prometheus 配置文件
        </a-button>
        <a-button
          type="primary"
          :disabled="!isIpValid"
          :loading="loading.prometheus_alert"
          @click="downloadConfig('prometheus_alert')"
          class="download-button"
        >
          <template #icon>
            <Icon icon="mdi:alert-circle-outline" />
          </template>
          Prometheus 告警配置文件
        </a-button>
        <a-button
          type="primary"
          :disabled="!isIpValid"
          :loading="loading.prometheus_record"
          @click="downloadConfig('prometheus_record')"
          class="download-button"
        >
          <template #icon>
            <Icon icon="material-symbols:document-scanner-outline-rounded" />
          </template>
          Prometheus 记录配置文件
        </a-button>
        <a-button
          type="primary"
          :disabled="!isIpValid"
          :loading="loading.alertManager"
          @click="downloadConfig('alertManager')"
          class="download-button"
        >
          <template #icon>
            <Icon icon="carbon:notification" />
          </template>
          AlertManager 配置文件
        </a-button>
      </div>
    </div>
    
    <!-- 使用说明卡片 -->
    <div class="dashboard-card">
      <div class="section-title">使用说明</div>
      <div class="instruction-content">
        <div class="instruction-item">
          <div class="instruction-number">1</div>
          <div class="instruction-text">输入目标服务器的 IP 地址并验证</div>
        </div>
        <div class="instruction-item">
          <div class="instruction-number">2</div>
          <div class="instruction-text">点击对应的按钮下载所需的配置文件</div>
        </div>
        <div class="instruction-item">
          <div class="instruction-number">3</div>
          <div class="instruction-text">将下载的配置文件放置在对应服务的配置目录中</div>
        </div>
        <div class="instruction-item">
          <div class="instruction-number">4</div>
          <div class="instruction-text">重启对应的服务使配置生效</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { message } from 'ant-design-vue';
import { CheckOutlined, ReloadOutlined } from '@ant-design/icons-vue';
import { Icon } from '@iconify/vue';

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
    ipError.value = 'IP 地址不能为空';
    isIpValid.value = false;
    message.error('IP 地址不能为空');
  } else if (!ipRegex.test(ip.value)) {
    ipError.value = '请输入有效的 IP 地址';
    isIpValid.value = false;
    message.error('请输入有效的 IP 地址');
  } else {
    ipError.value = '';
    isIpValid.value = true;
    message.success('IP 地址有效');
  }
};

// 重置IP
const resetIp = () => {
  ip.value = '';
  ipError.value = '';
  isIpValid.value = false;
};

// 监听 IP 地址的变化并验证
watch(ip, (newValue) => {
  if (newValue) {
    const ipRegex = /^(25[0-5]|2[0-4]\d|[0-1]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[0-1]?\d{1,2})){3}$/;
    if (!ipRegex.test(newValue)) {
      ipError.value = '请输入有效的 IP 地址';
      isIpValid.value = false;
    } else {
      ipError.value = '';
      isIpValid.value = true;
    }
  } else {
    ipError.value = 'IP 地址不能为空';
    isIpValid.value = false;
  }
});

// 下载配置文件
const downloadConfig = async (type: string) => {
  if (!isIpValid.value) {
    message.error('请先输入有效的 IP 地址');
    return;
  }

  let url = '';
  let fileName = '';
  switch (type) {
    case 'prometheus':
      url = `/api/monitor/prometheus_configs/prometheus?ip=${ip.value}`;
      fileName = 'prometheus_config.yaml';
      break;
    case 'prometheus_alert':
      url = `/api/monitor/prometheus_configs/prometheus_alert?ip=${ip.value}`;
      fileName = 'prometheus_alert_config.yaml';
      break;
    case 'prometheus_record':
      url = `/api/monitor/prometheus_configs/prometheus_record?ip=${ip.value}`;
      fileName = 'prometheus_record_config.yaml';
      break;
    case 'alertManager':
      url = `/api/monitor/prometheus_configs/alertManager?ip=${ip.value}`;
      fileName = 'alertmanager_config.yaml';
      break;
    default:
      message.error('未知的配置类型');
      return;
  }

  try {
    loading.value[type] = true;
    const response = await fetch(url);
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || '下载配置文件失败');
    }
    const blob = await response.blob();
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(downloadUrl);
    message.success(`${fileName} 下载开始`);
  } catch (error: any) {
    message.error(error.message || '下载配置文件失败');
    console.error(error);
  } finally {
    loading.value[type] = false;
  }
};
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

.search-filters {
  display: flex;
  gap: 16px;
  align-items: center;
  width: 100%;
}

.search-input {
  flex: 1;
  border-radius: 4px;
  transition: all 0.3s;
}

.search-input:hover,
.search-input:focus {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.search-icon {
  color: #bfbfbf;
  font-size: 16px;
}

.action-button {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  border-radius: 4px;
  transition: all 0.3s;
}

.reset-button {
  background-color: #f5f5f5;
  color: #595959;
  border-color: #d9d9d9;
}

.reset-button:hover {
  background-color: #e6e6e6;
  border-color: #b3b3b3;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 16px;
  padding-left: 12px;
  border-left: 4px solid #1890ff;
}

.download-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.download-button {
  min-width: 220px;
  height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 500;
  background: linear-gradient(45deg, #1890ff, #36bdf4);
  border: none;
  border-radius: 4px;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.4);
  transition: all 0.3s;
}

.download-button:hover:not(:disabled) {
  background: linear-gradient(45deg, #096dd9, #1890ff);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.5);
}

.download-button:disabled {
  background: #f5f5f5;
  color: rgba(0, 0, 0, 0.25);
  box-shadow: none;
}

.instruction-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.instruction-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.instruction-number {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: linear-gradient(45deg, #1890ff, #36bdf4);
  color: white;
  border-radius: 50%;
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0;
}

.instruction-text {
  font-size: 14px;
  color: #333;
  padding-top: 8px;
}

@media (max-width: 768px) {
  .download-buttons {
    flex-direction: column;
  }
  
  .search-filters {
    flex-direction: column;
    align-items: stretch;
  }
  
  .action-button {
    width: 100%;
  }
}
</style>