<template>
  <div class="ai-assistant-container">
    <!-- 优化的悬浮按钮 -->
    <a-float-button class="assistant-float-button" type="primary" shape="circle" @click="handleClick">
      <template #icon>
        <div class="float-button-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M12 2C6.48 2 2 6.48 2 12C2 17.52 6.48 22 12 22C17.52 22 22 17.52 22 12C22 6.48 17.52 2 12 2ZM12 20C7.59 20 4 16.41 4 12C4 7.59 7.59 4 12 4C16.41 4 20 7.59 20 12C20 16.41 16.41 20 12 20Z"
              fill="currentColor" transform="translate(-2, 0)" />
            <path d="M13 7H11V11H7V13H11V17H13V13H17V11H13V7Z" fill="currentColor" transform="translate(-2, 0)" />
          </svg>
        </div>
      </template>
      <template #tooltip>
        <div class="tooltip-content">
          <span class="tooltip-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M9 16.17L4.83 12L3.41 13.41L9 19L21 7L19.59 5.59L9 16.17Z" fill="currentColor" />
            </svg>
          </span>
          <span>AI-CloudOps助手</span>
        </div>
      </template>
    </a-float-button>

    <!-- 优化的抽屉 -->
    <a-drawer :title="false" placement="right" :width="380" :visible="drawerVisible" @close="toggleChatDrawer"
      class="ai-chat-drawer" :headerStyle="headerStyle" :bodyStyle="bodyStyle" closable :mask="true"
      :maskClosable="true">
      <!-- 自定义抽屉头部 -->
      <template #header>
        <div class="drawer-header">
          <div class="drawer-title">
            <div class="title-icon">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M21 11.5C21 16.75 16.75 21 11.5 21C6.25 21 2 16.75 2 11.5C2 6.25 6.25 2 11.5 2"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round" />
                <path d="M22 22L20 20" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
                <path d="M11.5 8V12.5L14.5 15.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
              </svg>
            </div>
            <span>AI-CloudOps助手</span>
          </div>
          <div class="drawer-actions">
            <a-button type="text" class="action-button" @click="clearChat" title="清空聊天">
              <template #icon>
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path
                    d="M19 6.41L17.59 5L12 10.59L6.41 5L5 6.41L10.59 12L5 17.59L6.41 19L12 13.41L17.59 19L19 17.59L13.41 12L19 6.41Z"
                    fill="currentColor" />
                </svg>
              </template>
            </a-button>
            <a-button type="text" class="action-button" @click="toggleChatDrawer" title="关闭">
              <template #icon>
                <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M6 18L18 6M6 6L18 18" stroke="currentColor" stroke-width="2" stroke-linecap="round"
                    stroke-linejoin="round" />
                </svg>
              </template>
            </a-button>
          </div>
        </div>
      </template>

      <div class="chat-container">
        <!-- 消息内容区域 -->
        <div class="chat-messages" ref="messagesContainer">
          <div v-for="(msg, index) in chatMessages" :key="index" :class="['message', msg.type]">
            <div class="avatar">
              <div class="avatar-container" :class="msg.type === 'ai' ? 'ai-avatar' : 'user-avatar'">
                <template v-if="msg.type === 'ai'">
                  <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M9 16.17L4.83 12L3.41 13.41L9 19L21 7L19.59 5.59L9 16.17Z" fill="currentColor" />
                  </svg>
                </template>
                <template v-else>
                  <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path
                      d="M12 12C14.21 12 16 10.21 16 8C16 5.79 14.21 4 12 4C9.79 4 8 5.79 8 8C8 10.21 9.79 12 12 12ZM12 14C9.33 14 4 15.34 4 18V20H20V18C20 15.34 14.67 14 12 14Z"
                      fill="currentColor" />
                  </svg>
                </template>
              </div>
            </div>
            <div class="content">
              <div class="name">{{ msg.type === 'user' ? '您' : 'AI助手' }}</div>
              <div class="text" v-html="renderMarkdown(msg.content || '')"></div>
              <div class="time">{{ msg.time }}</div>
            </div>
          </div>
          <!-- 加载指示器 -->
          <div v-if="sending && chatMessages[chatMessages.length - 1]?.type === 'ai'" class="typing-indicator">
            <span></span>
            <span></span>
            <span></span>
          </div>
        </div>

        <!-- 分隔线 -->
        <div class="chat-divider"></div>

        <!-- 输入区域 -->
        <div class="chat-input">
          <div class="textarea-container">
            <a-textarea v-model:value="globalInputMessage" placeholder="请输入您的问题..." :rows="1" :disabled="sending"
              :auto-size="{ minRows: 1, maxRows: 5 }" @keydown.enter.prevent="handleEnterPress" />
            <a-button type="primary" :disabled="sending" @click="handleSearch" class="send-button"
              :class="{ 'button-active': globalInputMessage.trim().length > 0 }">
              <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M2.01 21L23 12L2.01 3L2 10L17 12L2 14L2.01 21Z" fill="currentColor" />
              </svg>
            </a-button>
          </div>
          <div class="input-tools">
            <div class="tools-hint">按Enter发送，Shift+Enter换行</div>
            <div class="shortcuts-hint">
              <span class="shortcut-key">Ctrl+/</span>
              <span>快速打开助手</span>
            </div>
          </div>
        </div>
      </div>
    </a-drawer>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, nextTick, onBeforeUnmount, watch } from 'vue';
import { message } from 'ant-design-vue';
// @ts-ignore
import MarkdownIt from 'markdown-it';
// @ts-ignore
import MarkdownItHighlight from 'markdown-it-highlight';

// 初始化markdown解析器
const md = new MarkdownIt({
  breaks: true,
  linkify: true,
  html: false,
  typographer: true
}).use(MarkdownItHighlight);

// 状态管理
const drawerVisible = ref(false);
const globalInputMessage = ref('');
const sending = ref(false);
const messagesContainer = ref(null);
let socket: WebSocket | null = null;
let currentResponse = '';

// 抽屉样式
const headerStyle = {
  background: '#0a0f1a',
  borderBottom: '1px solid rgba(75, 85, 99, 0.3)',
  padding: '0'
};

const bodyStyle = {
  background: '#0d1424',
  padding: 0
};

// 聊天消息
interface ChatMessage {
  content: string;
  type: 'user' | 'ai';
  time: string;
}

// WebSocket消息历史
interface ChatHistoryItem {
  role: string;
  content: string;
}

const chatMessages = reactive<ChatMessage[]>([
  {
    content: '您好！我是AI-CloudOps助手，可以回答您关于云运维的问题。请问有什么我可以帮助您的？',
    type: 'ai',
    time: formatTime(new Date())
  }
]);

// 聊天历史记录
const chatHistory = reactive<ChatHistoryItem[]>([
  {
    role: 'CloudOps小助手',
    content: '您好！我是AI-CloudOps助手，可以回答您关于云运维的问题。请问有什么我可以帮助您的？'
  }
]);

// Enter键处理
const handleEnterPress = (e: KeyboardEvent) => {
  if (e.shiftKey) return;

  e.preventDefault();
  const msg = globalInputMessage.value.trim();
  if (!msg) return;

  // 发送消息
  sendMessage(msg);

  // 确保在消息发送后清空输入框
  globalInputMessage.value = '';

  setTimeout(() => {
    globalInputMessage.value = '';
  }, 10);
};

// 确保点击按钮发送也使用相同的逻辑
const handleSearch = () => {
  const msg = globalInputMessage.value.trim();
  if (!msg) return;

  // 发送消息
  sendMessage(msg);

  // 清空输入框
  globalInputMessage.value = '';
};

// 清空聊天
const clearChat = () => {
  // 显示确认提示
  if (confirm('确定要清空所有聊天记录吗？')) {
    chatMessages.length = 0;
    chatHistory.length = 0;
    // 重新添加初始欢迎消息
    chatMessages.push({
      content: '您好！我是AI-CloudOps助手，可以回答您关于云运维的问题。请问有什么我可以帮助您的？',
      type: 'ai',
      time: formatTime(new Date())
    });
    chatHistory.push({
      role: 'CloudOps小助手',
      content: '您好！我是AI-CloudOps助手，可以回答您关于云运维的问题。请问有什么我可以帮助您的？'
    });
  }
};

// 渲染Markdown内容
const renderMarkdown = (content: string): string => {
  if (!content) return '';
  try {
    return md.render(content);
  } catch (e) {
    console.error('Markdown渲染错误:', e);
    return content;
  }
};

// 初始化WebSocket连接
const initWebSocket = () => {
  if (socket !== null) {
    return;
  }

  socket = new WebSocket('ws://localhost:8889/api/ai/chat/ws');

  socket.onopen = () => {
    console.log('WebSocket连接已建立');
  };

  socket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'message') {
      if (!data.done) {
        // 累积响应内容
        currentResponse += data.content || '';

        // 更新最后一条AI消息
        if (chatMessages[chatMessages.length - 1]?.type === 'ai') {
          chatMessages[chatMessages.length - 1]!.content = currentResponse;
        }

        // 滚动到底部
        nextTick(() => {
          scrollToBottom();
        });
      } else {
        // 消息完成，更新聊天历史
        sending.value = false;

        // 将完整回复添加到聊天历史
        chatHistory.push({
          role: 'assistant',
          content: currentResponse
        });

        // 重置当前响应
        currentResponse = '';
      }
    }
  };

  socket.onerror = (error) => {
    console.error('WebSocket错误:', error);
    message.error('连接服务器失败，请稍后重试');
    sending.value = false;
  };

  socket.onclose = () => {
    console.log('WebSocket连接已关闭');
    socket = null;
  };
};

// 切换聊天抽屉显示状态
const toggleChatDrawer = () => {
  drawerVisible.value = !drawerVisible.value;
  if (drawerVisible.value) {
    initWebSocket();
    nextTick(() => {
      scrollToBottom();
    });
  } else {
    // 关闭抽屉时关闭WebSocket连接
    if (socket) {
      socket.close();
      socket = null;
    }
  }
};

// 发送消息
const sendMessage = async (value: string) => {
  if (!value.trim()) {
    message.warning('请输入消息内容');
    return;
  }

  if (!socket || socket.readyState !== WebSocket.OPEN) {
    initWebSocket();
    message.warning('正在连接服务器，请稍后重试');
    return;
  }

  // 添加用户消息
  chatMessages.push({
    content: value,
    type: 'user',
    time: formatTime(new Date())
  });

  // 添加到聊天历史
  chatHistory.push({
    role: 'user',
    content: value
  });

  sending.value = true;

  // 滚动到底部
  await nextTick();
  scrollToBottom();

  // 添加AI消息占位
  chatMessages.push({
    content: '',
    type: 'ai',
    time: formatTime(new Date())
  });

  // 重置当前响应
  currentResponse = '';

  // 准备发送的消息
  const messageToSend = {
    role: "assistant",
    style: "专业",
    question: value,
    chatHistory: chatHistory.slice(0, -1) // 不包括当前问题
  };

  // 发送消息到WebSocket
  socket.send(JSON.stringify(messageToSend));
};

// 格式化时间
function formatTime(date: Date): string {
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
}

// 滚动到底部
const scrollToBottom = () => {
  if (messagesContainer.value) {
    const container = messagesContainer.value as HTMLElement;
    container.scrollTop = container.scrollHeight;
  }
};

// 初始点击处理函数
const handleClick = () => {
  toggleChatDrawer();
};

// 监听消息变化自动滚动
watch(chatMessages, () => {
  nextTick(() => {
    scrollToBottom();
  });
}, { deep: true });

onMounted(() => {
  // 初始化操作
  window.addEventListener('keydown', (e) => {
    // 添加快捷键 Ctrl+/ 打开聊天窗口
    if (e.ctrlKey && e.key === '/') {
      e.preventDefault();
      toggleChatDrawer();
    }
  });
});

onBeforeUnmount(() => {
  // 移除事件监听
  window.removeEventListener('keydown', () => { });

  // 关闭WebSocket连接
  if (socket) {
    socket.close();
    socket = null;
  }
});
</script>

<style>
/* 代码高亮样式 */
.hljs {
  display: block;
  overflow-x: auto;
  padding: 1em;
  background: #0f172a;
  color: #e2e8f0;
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  border-radius: 8px;
  margin: 14px 0;
  font-size: 13px;
  line-height: 1.5;
}

.hljs-keyword,
.hljs-selector-tag,
.hljs-addition {
  color: #f472b6;
}

.hljs-number,
.hljs-string,
.hljs-meta .hljs-meta-string,
.hljs-literal,
.hljs-doctag,
.hljs-regexp {
  color: #4ade80;
}

.hljs-title,
.hljs-section,
.hljs-name,
.hljs-selector-id,
.hljs-selector-class {
  color: #60a5fa;
}

.hljs-attribute,
.hljs-attr,
.hljs-variable,
.hljs-template-variable,
.hljs-class .hljs-title,
.hljs-type {
  color: #fbbf24;
}

.hljs-symbol,
.hljs-bullet,
.hljs-subst,
.hljs-meta,
.hljs-meta .hljs-keyword,
.hljs-selector-attr,
.hljs-selector-pseudo,
.hljs-link {
  color: #e879f9;
}

.hljs-built_in,
.hljs-deletion {
  color: #a78bfa;
}

.hljs-comment,
.hljs-quote {
  color: #94a3b8;
  font-style: italic;
}
</style>

<style scoped>
.ai-assistant-container {
  position: relative;
}

/* 悬浮按钮样式 */
.assistant-float-button {
  box-shadow: 0 4px 20px rgba(0, 118, 255, 0.4);
  transition: all 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

.assistant-float-button:hover {
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 8px 25px rgba(0, 118, 255, 0.5);
}

.float-button-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
}

.tooltip-content {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  font-size: 13px;
}

.tooltip-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
}

/* 抽屉样式 */
.ai-chat-drawer :deep(.ant-drawer-content) {
  background-color: #0d1424;
  /* 更深的暗蓝色背景 */
  border-radius: 20px 0 0 20px;
  overflow: hidden;
  box-shadow: -10px 0 30px rgba(0, 0, 0, 0.6);
}

.ai-chat-drawer :deep(.ant-drawer-body) {
  padding: 0;
  display: flex;
  flex-direction: column;
}

/* 自定义抽屉头部 */
.drawer-header {
  padding: 18px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
  background: linear-gradient(to right, #0a0f1a, #111827);
}

.drawer-title {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #f8fafc;
  font-weight: 600;
  font-size: 16px;
  letter-spacing: 0.2px;
}

.title-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b82f6;
  width: 24px;
  height: 24px;
}

.drawer-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.action-button {
  color: #94a3b8;
  transition: all 0.2s ease;
  width: 36px;
  height: 36px;
  border-radius: 10px;
}

.action-button:hover {
  color: #f8fafc;
  background-color: rgba(100, 116, 139, 0.15);
}

.action-button :deep(svg) {
  width: 20px;
  height: 20px;
}

/* 聊天容器 */
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: linear-gradient(to bottom, #0d1424, #121a30);
}

/* 消息区域 */
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  scroll-behavior: smooth;
}

/* 自定义滚动条 */
.chat-messages::-webkit-scrollbar {
  width: 5px;
}

.chat-messages::-webkit-scrollbar-track {
  background: rgba(15, 23, 42, 0.1);
}

.chat-messages::-webkit-scrollbar-thumb {
  background: rgba(100, 116, 139, 0.3);
  border-radius: 10px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: rgba(100, 116, 139, 0.5);
}

/* 消息样式 */
.message {
  display: flex;
  margin-bottom: 28px;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.message .avatar {
  margin-right: 14px;
  flex-shrink: 0;
}

.avatar-container {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.avatar-container svg {
  width: 24px;
  height: 24px;
}

.user-avatar {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white;
}

.ai-avatar {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
}

.message .content {
  background-color: #1e293b;
  padding: 16px 20px;
  border-radius: 18px;
  max-width: 82%;
  color: #f1f5f9;
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);
  position: relative;
  transition: all 0.3s ease;
}

.message.user .content {
  background: linear-gradient(135deg, #1d4ed8, #3b82f6);
  box-shadow: 0 5px 15px rgba(29, 78, 216, 0.25);
}

.message.ai .content {
  background: linear-gradient(135deg, #1e293b, #273446);
  box-shadow: 0 5px 15px rgba(15, 23, 42, 0.2);
}

.message .name {
  font-weight: 600;
  margin-bottom: 10px;
  font-size: 14px;
  color: rgba(248, 250, 252, 0.9);
  letter-spacing: 0.3px;
}

.message .text {
  word-break: break-word;
  color: #f8fafc;
  line-height: 1.7;
  font-size: 14px;
  letter-spacing: 0.2px;
}

.message .text :deep(pre) {
  margin: 16px 0;
  border-radius: 10px;
  overflow-x: auto;
  position: relative;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(30, 41, 59, 0.8);
}

.message .text :deep(pre)::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: linear-gradient(to right, transparent 0%, rgba(15, 23, 42, 0.1) 100%);
  pointer-events: none;
}

.message .text :deep(code:not(pre code)) {
  background-color: rgba(30, 41, 59, 0.5);
  padding: 3px 6px;
  border-radius: 5px;
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  color: #7dd3fc;
}

.message .text :deep(a) {
  color: #38bdf8;
  text-decoration: none;
  border-bottom: 1px dotted #38bdf8;
  transition: all 0.2s ease;
  padding-bottom: 1px;
}

.message .text :deep(a:hover) {
  color: #0ea5e9;
  border-bottom: 1px solid #0ea5e9;
}

.message .text :deep(ul),
.message .text :deep(ol) {
  padding-left: 22px;
  margin: 12px 0;
}

.message .text :deep(li) {
  margin-bottom: 8px;
}

.message .text :deep(p) {
  margin: 10px 0;
}

.message .text :deep(h1),
.message .text :deep(h2),
.message .text :deep(h3),
.message .text :deep(h4),
.message .text :deep(h5),
.message .text :deep(h6) {
  margin-top: 20px;
  margin-bottom: 12px;
  color: #f8fafc;
  font-weight: 600;
}

.message .text :deep(h1) {
  font-size: 22px;
}

.message .text :deep(h2) {
  font-size: 20px;
}

.message .text :deep(h3) {
  font-size: 18px;
}

.message .text :deep(h4) {
  font-size: 16px;
}

.message .text :deep(blockquote) {
  border-left: 4px solid #3b82f6;
  padding: 10px 18px;
  margin: 16px 0;
  background-color: rgba(59, 130, 246, 0.1);
  border-radius: 0 8px 8px 0;
  font-style: italic;
  color: #e2e8f0;
}

.message .time {
  font-size: 12px;
  color: rgba(148, 163, 184, 0.8);
  margin-top: 10px;
  text-align: right;
}

/* 分隔线 */
.chat-divider {
  height: 1px;
  background: linear-gradient(to right, transparent, rgba(100, 116, 139, 0.2), transparent);
  margin: 0;
}

/* 输入区域 */
.chat-input {
  padding: 18px 20px 20px;
  background-color: #121a30;
  border-top: 1px solid rgba(51, 65, 85, 0.5);
}

.textarea-container {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  background-color: #1e293b;
  border-radius: 16px;
  padding: 14px 18px;
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.3s ease;
}

.textarea-container:focus-within {
  box-shadow: 0 5px 20px rgba(37, 99, 235, 0.2);
  background-color: #334155;
  transform: translateY(-1px);
}

.textarea-container :deep(.ant-input) {
  background-color: transparent;
  color: #f1f5f9;
  border: none;
  resize: none;
  flex: 1;
  padding: 0;
  line-height: 1.6;
  font-size: 15px;
  letter-spacing: 0.2px;
}

.textarea-container :deep(.ant-input:focus) {
  box-shadow: none;
}

.send-button {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border: none;
  height: 38px;
  width: 38px;
  min-width: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  padding: 0;
  transition: all 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
}

.send-button svg {
  width: 22px;
  height: 22px;
}

.send-button:hover {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  transform: translateY(-2px) scale(1.05);
  box-shadow: 0 6px 16px rgba(37, 99, 235, 0.4);
}

.button-active {
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(37, 99, 235, 0.5);
  }

  70% {
    box-shadow: 0 0 0 10px rgba(37, 99, 235, 0);
  }

  100% {
    box-shadow: 0 0 0 0 rgba(37, 99, 235, 0);
  }
}

.chat-input :deep(.ant-input::placeholder) {
  color: #94a3b8;
}

.input-tools {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
  padding: 0 8px;
}

.tools-hint {
  font-size: 12px;
  color: #64748b;
}

.shortcuts-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #64748b;
}

.shortcut-key {
  background-color: rgba(51, 65, 85, 0.5);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'JetBrains Mono', monospace;
  color: #94a3b8;
  font-size: 11px;
}

/* 打字指示器 */
.typing-indicator {
  padding: 8px 14px;
  background: rgba(51, 65, 85, 0.4);
  border-radius: 24px;
  display: inline-flex;
  align-items: center;
  margin-left: 54px;
  margin-bottom: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.typing-indicator span {
  height: 8px;
  width: 8px;
  margin: 0 3px;
  background-color: #94a3b8;
  display: block;
  border-radius: 50%;
  opacity: 0.5;
}

.typing-indicator span:nth-of-type(1) {
  animation: typing 1.3s infinite 0s;
}

.typing-indicator span:nth-of-type(2) {
  animation: typing 1.3s infinite 0.2s;
}

.typing-indicator span:nth-of-type(3) {
  animation: typing 1.3s infinite 0.4s;
}

@keyframes typing {
  0% {
    transform: scale(1);
    opacity: 0.5;
  }

  50% {
    transform: scale(1.5);
    opacity: 1;
  }

  100% {
    transform: scale(1);
    opacity: 0.5;
  }
}

/* 响应式调整 */
@media (max-width: 768px) {
  .ai-chat-drawer :deep(.ant-drawer-content-wrapper) {
    width: 100% !important;
  }

  .message .content {
    max-width: 90%;
  }

  .chat-messages {
    padding: 16px;
  }

  .chat-input {
    padding: 16px;
  }
}
</style>
