<template>
  <div class="console">
    <div class="header">
      <a-button v-if="isConnected" type="primary" danger @click="handleClose">关闭连接</a-button>
    </div>
    <div v-if="isConnected" ref="terminalElement" id="terminal"></div>
    <div v-else class="reconnect-message">
      <a-result
        status="warning"
        title="连接已断开"
        sub-title="终端连接已关闭,请重新连接"
      >
        <template #extra>
          <a-button type="primary" @click="handleReconnect">重新连接</a-button>
        </template>
      </a-result>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, ref, onBeforeUnmount } from 'vue';
import { useRoute} from 'vue-router';
import { Terminal } from 'xterm';
import { message } from 'ant-design-vue';
import { useAccessStore } from '@vben/stores';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

// 常量定义
const INACTIVITY_TIMEOUT = 60000; // 不活动超时时间(毫秒)
const TERMINAL_CONFIG = {
  cols: 120,
  rows: 30,
  convertEol: true,
  scrollback: 1000,
  disableStdin: false,
  cursorStyle: 'block',
  cursorBlink: true,
  fontFamily: 'Menlo, Monaco, Consolas, monospace',
  fontSize: 14,
  theme: {
    foreground: '#ffffff',
    background: '#000000',
    cursor: '#ffffff',
    black: '#000000',
    red: '#cd3131',
    green: '#0dbc79',
    yellow: '#e5e510',
    blue: '#2472c8',
    magenta: '#bc3fbc',
    cyan: '#11a8cd',
    white: '#e5e5e5',
    brightBlack: '#666666',
    brightRed: '#f14c4c',
    brightGreen: '#23d18b',
    brightYellow: '#f5f543',
    brightBlue: '#3b8eea',
    brightMagenta: '#d670d6',
    brightCyan: '#29b8db',
    brightWhite: '#e5e5e5'
  }
};

// 状态管理
const route = useRoute();
const terminalElement = ref<HTMLElement | null>(null);
const terminal = ref<Terminal | null>(null);
const isConnected = ref(true);
const ws = ref<WebSocket | null>(null);
const currentCommand = ref('');
const inactivityTimer = ref<NodeJS.Timeout | null>(null);
const fitAddon = ref<FitAddon | null>(null);

// 重置不活动计时器
const resetInactivityTimer = () => {
  if (inactivityTimer.value) {
    clearTimeout(inactivityTimer.value);
  }
  inactivityTimer.value = setTimeout(() => {
    if (ws.value) {
      message.warning('由于长时间未操作,连接已自动关闭');
      ws.value.close(1000, '用户不活动超时');
    }
  }, INACTIVITY_TIMEOUT);
};

// 手动关闭连接
const handleClose = () => {
  if (ws.value) {
    ws.value.close(1000, '用户手动关闭连接');
    message.success('已关闭终端连接');
  }
};

// 重新连接
const handleReconnect = () => {
  const id = route.query.id as string;
  if (id) {
    isConnected.value = true;
    window.location.reload();
  }
};

// 处理终端输入
const handleTerminalInput = (data: string) => {
  if (!ws.value || ws.value.readyState !== WebSocket.OPEN || !terminal.value) return;
  
  resetInactivityTimer();
  
  switch(data) {
    case '\r': // 回车键
      ws.value.send(currentCommand.value + '\n');
      currentCommand.value = '';
      terminal.value.write('\r\n');
      break;
    case '\u007f': // 退格键
      if (currentCommand.value.length > 0) {
        currentCommand.value = currentCommand.value.slice(0, -1);
        terminal.value.write('\b \b');
      }
      break;
    default:
      currentCommand.value += data;
      terminal.value.write(data);
  }
};

// WebSocket连接函数
const connectWebSocket = (id: string) => {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken;
  
  if (!token) {
    message.error('未获取到认证信息');
    return;
  }

  const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const wsUrl = `${wsProtocol}://localhost:8888/api/tree/ecs/console/${id}?token=${encodeURIComponent(token)}`;

  try {
    ws.value = new WebSocket(wsUrl);

    ws.value.onopen = () => {
      console.log('WebSocket连接已建立');
      message.success('终端连接成功');
      resetInactivityTimer();
    };

    ws.value.onmessage = (event) => {
      terminal.value?.write(event.data);
    };

    ws.value.onerror = (error) => {
      console.error('WebSocket错误:', error);
      message.error('终端连接错误');
      isConnected.value = false;
    };

    ws.value.onclose = () => {
      isConnected.value = false;
      if (inactivityTimer.value) {
        clearTimeout(inactivityTimer.value);
      }
    };
  } catch (error) {
    console.error('创建WebSocket失败:', error);
    message.error('创建终端连接失败');
    isConnected.value = false;
  }
};

// 初始化终端
const initTerminal = () => {
  terminal.value = new Terminal({
    ...TERMINAL_CONFIG,
    cursorStyle: 'block' as 'block' | 'underline' | 'bar'
  });
  fitAddon.value = new FitAddon();
  terminal.value.loadAddon(fitAddon.value);

  if (terminalElement.value) {
    terminal.value.open(terminalElement.value);
    fitAddon.value.fit();
  }

  terminal.value.onData(handleTerminalInput);
};

// 处理窗口大小变化
const handleResize = () => {
  fitAddon.value?.fit();
};

// 生命周期钩子
onMounted(() => {
  const id = route.query.id as string;
  if (!id) {
    message.error('缺少必要的参数');
    return;
  }

  initTerminal();
  connectWebSocket(id);
  window.addEventListener('resize', handleResize);
});

onBeforeUnmount(() => {
  // 清理资源
  if (ws.value) {
    try {
      ws.value.close(1000, '用户关闭终端');
    } catch (error) {
      console.error('关闭WebSocket失败:', error);
    }
  }

  if (terminal.value) {
    try {
      terminal.value.dispose();
    } catch (error) {
      console.error('销毁终端失败:', error);
    }
  }

  if (inactivityTimer.value) {
    clearTimeout(inactivityTimer.value);
  }

  window.removeEventListener('resize', handleResize);
});
</script>

<style scoped>
.console {
  width: 100%;
  height: calc(100vh - 100px);
  padding: 16px;
  background: #000000;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.header {
  margin-bottom: 16px;
  text-align: right;
}

#terminal {
  width: 100%;
  height: calc(100% - 40px);
  border-radius: 4px;
  overflow: hidden;
}

.reconnect-message {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  border-radius: 4px;
}
</style>
