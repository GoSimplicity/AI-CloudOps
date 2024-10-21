<template>
    <div>
      <!-- 配置文件下载 -->
      <div class="config-download">
        <a-button type="primary" @click="downloadConfig('prometheus')">下载 Prometheus 配置文件</a-button>
        <a-button type="primary" @click="downloadConfig('prometheus_alert')">下载 Prometheus 告警配置文件</a-button>
        <a-button type="primary" @click="downloadConfig('prometheus_record')">下载 Prometheus 记录配置文件</a-button>
        <a-button type="primary" @click="downloadConfig('alertManager')">下载 AlertManager 配置文件</a-button>
      </div>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { message } from 'ant-design-vue';
  
  // 下载配置文件
  const downloadConfig = (type: string) => {
    let url = '';
    switch (type) {
      case 'prometheus':
        url = '/api/monitor/prometheus_configs/prometheus';
        break;
      case 'prometheus_alert':
        url = '/api/monitor/prometheus_configs/prometheus_alert';
        break;
      case 'prometheus_record':
        url = '/api/monitor/prometheus_configs/prometheus_record';
        break;
      case 'alertManager':
        url = '/api/monitor/prometheus_configs/alertManager';
        break;
      default:
        message.error('未知的配置类型');
        return;
    }
  
    // 创建下载链接并执行下载
    const link = document.createElement('a');
    link.href = url;
    link.download = `${type}_config.yaml`;
    link.click();
    message.success(`${type} 配置文件下载开始`);
  };
  </script>
  
  <style scoped>
  .config-download {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 16px;
  }
  </style>
  