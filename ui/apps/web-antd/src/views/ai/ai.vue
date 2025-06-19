<template>
  <div class="ai-assistant-container">
    <!-- 优化的悬浮按钮 -->
    <a-float-button 
      class="assistant-float-button" 
      type="primary" 
      shape="circle" 
      @click="handleClick"
    >
      <template #icon>
        <div class="float-button-icon">
          <MessageCircle :size="24" />
        </div>
      </template>
      <template #tooltip>
        <div class="tooltip-content">
          <Sparkles :size="16" />
          <span>AI-CloudOps助手</span>
        </div>
      </template>
    </a-float-button>

    <!-- 优化的抽屉 -->
    <a-drawer 
      :title="false" 
      placement="right" 
      :width="420" 
      :visible="drawerVisible" 
      @close="toggleChatDrawer"
      class="ai-chat-drawer" 
      :headerStyle="headerStyle" 
      :bodyStyle="bodyStyle" 
      closable 
      :mask="true"
      :maskClosable="true"
    >
      <!-- 自定义抽屉头部 -->
      <template #header>
        <div class="drawer-header">
          <div class="drawer-title">
            <div class="title-icon">
              <Bot :size="24" />
            </div>
            <div class="title-content">
              <span class="title-text">AI-CloudOps助手</span>
              <span class="title-subtitle">智能运维助手</span>
            </div>
          </div>
          <div class="drawer-actions">
            <a-button 
              type="text" 
              class="action-button" 
              @click="clearChat" 
              title="清空聊天"
            >
              <Trash2 :size="18" />
            </a-button>
            <a-button 
              type="text" 
              class="action-button" 
              @click="toggleChatDrawer" 
              title="关闭"
            >
              <X :size="18" />
            </a-button>
          </div>
        </div>
      </template>

      <div class="chat-container">
        <!-- 状态栏 -->
        <div class="status-bar">
          <div class="status-indicator">
            <div class="status-dot" :class="{ 'online': isConnected }"></div>
            <span class="status-text">
              {{ isConnected ? '已连接' : '连接中...' }}
            </span>
          </div>
          <div class="message-count">
            {{ chatMessages.length - 1 }} 条对话
          </div>
        </div>

        <!-- 消息内容区域 -->
        <div class="chat-messages" ref="messagesContainer">
          <div v-for="(msg, index) in chatMessages" :key="index" :class="['message', msg.type]">
            <div class="message-wrapper">
              <div class="avatar">
                <div class="avatar-container" :class="msg.type === 'ai' ? 'ai-avatar' : 'user-avatar'">
                  <Bot v-if="msg.type === 'ai'" :size="20" />
                  <User v-else :size="20" />
                </div>
              </div>
              <div class="content">
                <div class="message-header">
                  <span class="name">{{ msg.type === 'user' ? '您' : 'AI助手' }}</span>
                  <span class="time">{{ msg.time }}</span>
                </div>
                <div class="text" v-html="renderMarkdown(msg.content || '')"></div>
                <div class="message-actions" v-if="msg.type === 'ai' && msg.content">
                  <a-button 
                    type="text" 
                    size="small" 
                    class="message-action-btn"
                    @click="copyMessage(msg.content)"
                  >
                    <Copy :size="14" />
                  </a-button>
                  <a-button 
                    type="text" 
                    size="small" 
                    class="message-action-btn"
                    @click="toggleLike(index)"
                  >
                    <ThumbsUp :size="14" :class="{ 'liked': msg.liked }" />
                  </a-button>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 加载指示器 -->
          <div v-if="sending" class="typing-indicator">
            <div class="avatar">
              <div class="avatar-container ai-avatar">
                <Bot :size="20" />
              </div>
            </div>
            <div class="typing-content">
              <div class="typing-animation">
                <span></span>
                <span></span>
                <span></span>
              </div>
              <span class="typing-text">AI正在思考中...</span>
            </div>
          </div>
        </div>

        <!-- 快捷操作 -->
        <div class="quick-actions" v-if="!sending">
          <div class="quick-action-buttons">
            <a-button 
              v-for="action in quickActions" 
              :key="action.text"
              type="text" 
              size="small" 
              class="quick-action-btn"
              @click="sendQuickMessage(action.text)"
            >
              <component :is="action.icon" :size="14" />
              {{ action.text }}
            </a-button>
          </div>
        </div>

        <!-- 输入区域 -->
        <div class="chat-input">
          <div class="textarea-container">
            <div class="input-wrapper">
              <a-textarea 
                v-model:value="globalInputMessage" 
                placeholder="请输入您的问题..." 
                :rows="1" 
                :disabled="sending"
                :auto-size="{ minRows: 1, maxRows: 4 }" 
                @keydown.enter="handleEnterKey"
                class="message-input"
              />

              <div class="input-actions">
                <a-button 
                  type="text" 
                  size="small" 
                  class="input-action-btn"
                  title="添加附件"
                >
                  <Paperclip :size="16" />
                </a-button>
                <a-button 
                  type="primary" 
                  :disabled="!globalInputMessage.trim() || sending" 
                  @click="handleSearch" 
                  class="send-button"
                  :loading="sending"
                >
                  <Send :size="18" v-if="!sending" />
                </a-button>
              </div>
            </div>
          </div>
          
          <div class="input-hints">
            <div class="hints-left">
              <span class="hint-item">
                <Keyboard :size="12" />
                Enter换行
              </span>
              <span class="hint-item">
                <Command :size="12" />
                Shift+Enter发送
              </span>
            </div>
            <div class="hints-right">
              <span class="shortcut-hint">
                <span class="shortcut-key">Ctrl + /</span>
                快速打开
              </span>
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
import { 
  MessageCircle, 
  Sparkles, 
  Bot, 
  User, 
  Trash2, 
  X, 
  Copy, 
  ThumbsUp, 
  Send, 
  Paperclip,
  Keyboard,
  Command,
  HelpCircle,
  Settings,
  Zap,
  FileText
} from 'lucide-vue-next';
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
const isConnected = ref(false);
const messagesContainer = ref(null);
let socket: WebSocket | null = null;
let currentResponse = '';

// 抽屉样式
const headerStyle = {
  background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 100%)',
  borderBottom: '1px solid rgba(148, 163, 184, 0.1)',
  padding: '0',
  boxShadow: '0 2px 20px rgba(0, 0, 0, 0.1)'
};

const bodyStyle = {
  background: 'linear-gradient(to bottom, #0f172a 0%, #1e293b 100%)',
  padding: 0
};

// 快捷操作
const quickActions = [
  { text: '云服务器状态', icon: Settings },
  { text: '性能监控', icon: Zap },
  { text: '日志分析', icon: FileText },
  { text: '帮助文档', icon: HelpCircle }
];

// 聊天消息接口
interface ChatMessage {
  content: string;
  type: 'user' | 'ai';
  time: string;
  liked?: boolean;
}

// WebSocket消息历史
interface ChatHistoryItem {
  role: string;
  content: string;
}

const chatMessages = reactive<ChatMessage[]>([
  {
    content: '👋 您好！我是AI-CloudOps助手，专注于为您提供智能运维服务。\n\n我可以帮助您：\n• 🔍 监控云服务器状态\n• 📊 分析性能指标\n• 🛠️ 故障诊断与修复\n• 📋 生成运维报告\n\n请问有什么我可以为您服务的吗？',
    type: 'ai',
    time: formatTime(new Date())
  }
]);

// 聊天历史记录
const chatHistory = reactive<ChatHistoryItem[]>([
  {
    role: 'assistant',
    content: '您好！我是AI-CloudOps助手，可以回答您关于云运维的问题。请问有什么我可以帮助您的？'
  }
]);

// 初始化聊天记录
const initChatMessages = () => {
  chatMessages.length = 0;
  chatHistory.length = 0;
  
  chatMessages.push({
    content: '👋 您好！我是AI-CloudOps助手，专注于为您提供智能运维服务。\n\n我可以帮助您：\n• 🔍 监控云服务器状态\n• 📊 分析性能指标\n• 🛠️ 故障诊断与修复\n• 📋 生成运维报告\n\n请问有什么我可以为您服务的吗？',
    type: 'ai',
    time: formatTime(new Date())
  });
  
  chatHistory.push({
    role: 'assistant',
    content: '您好！我是AI-CloudOps助手，可以回答您关于云运维的问题。请问有什么我可以帮助您的？'
  });
};

// 快捷消息发送
const sendQuickMessage = (text: string) => {
  globalInputMessage.value = text;
  handleSearch();
};

// 切换点赞状态
const toggleLike = (index: number) => {
  if (chatMessages[index]) {
    chatMessages[index].liked = !chatMessages[index].liked;
    message.success(chatMessages[index].liked ? '已点赞' : '已取消点赞');
  }
};

// 复制消息
const copyMessage = async (content: string) => {
  try {
    await navigator.clipboard.writeText(content);
    message.success('已复制到剪贴板');
  } catch (err) {
    message.error('复制失败');
  }
};

// 发送按钮处理
const handleSearch = () => {
  const msg = globalInputMessage.value.trim();
  if (!msg || sending.value) return;

  sendMessage(msg);
};

// 清空聊天
const clearChat = () => {
  if (chatMessages.length <= 1) {
    message.info('暂无聊天记录');
    return;
  }

  const modal = {
    title: '确认清空',
    content: '确定要清空所有聊天记录吗？此操作不可恢复。',
    okText: '确定',
    cancelText: '取消',
    onOk: () => {
      initChatMessages();
      message.success('聊天记录已清空');
    }
  };
  
  // 使用 Ant Design Vue 的 Modal.confirm
  import('ant-design-vue').then(({ Modal }) => {
    Modal.confirm(modal);
  });
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
  // 确保每次都是新的连接
  if (socket !== null) {
    socket.close();
    socket = null;
  }

  socket = new WebSocket('ws://localhost:8889/api/ai/chat/ws');

  socket.onopen = () => {
    console.log('WebSocket连接已建立');
    isConnected.value = true;
  };

  socket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'message') {
      if (!data.done) {
        currentResponse += data.content || '';

        if (chatMessages[chatMessages.length - 1]?.type === 'ai') {
          chatMessages[chatMessages.length - 1]!.content = currentResponse;
        }

        nextTick(() => {
          scrollToBottom();
        });
      } else {
        sending.value = false;

        chatHistory.push({
          role: 'assistant',
          content: currentResponse
        });

        currentResponse = '';
      }
    }
  };

  socket.onerror = (error) => {
    console.error('WebSocket错误:', error);
    message.error('连接服务器失败，请稍后重试');
    sending.value = false;
    isConnected.value = false;
  };

  socket.onclose = () => {
    console.log('WebSocket连接已关闭');
    isConnected.value = false;
    socket = null;
  };
};

// 切换聊天抽屉显示状态
const toggleChatDrawer = () => {
  drawerVisible.value = !drawerVisible.value;
  if (drawerVisible.value) {
    // 打开抽屉时，始终创建新的WebSocket连接
    initWebSocket();
    nextTick(() => {
      scrollToBottom();
    });
  } else {
    // 关闭抽屉时，关闭WebSocket连接
    if (socket) {
      socket.close();
      socket = null;
    }
    
    // 关闭抽屉时重置发送状态
    sending.value = false;
    currentResponse = '';
    
    // 如果最后一条消息是AI且内容为空，则移除它
    if (chatMessages.length > 0 && chatMessages[chatMessages.length - 1]?.type === 'ai' && !chatMessages[chatMessages.length - 1]?.content.trim()) {
      chatMessages.pop();
      // 同时也要移除对应的历史记录
      if (chatHistory && chatHistory.length > 0 && chatHistory[chatHistory.length - 1]?.role === 'user') {
        chatHistory.pop();
      }
    }
    
    // 关闭抽屉时清空聊天历史
    initChatMessages();
  }
};

// 发送消息
const sendMessage = async (value: string) => {
  const trimmedValue = value.trim();
  if (!trimmedValue) {
    message.warning('请输入消息内容');
    return;
  }

  globalInputMessage.value = '';

  if (!socket || socket.readyState !== WebSocket.OPEN) {
    initWebSocket();
    message.warning('正在连接服务器，请稍后重试');
    return;
  }

  // 添加用户消息
  chatMessages.push({
    content: trimmedValue,
    type: 'user',
    time: formatTime(new Date())
  });

  // 添加到聊天历史
  chatHistory.push({
    role: 'user',
    content: trimmedValue
  });

  sending.value = true;

  await nextTick();
  scrollToBottom();

  // 添加AI消息占位
  chatMessages.push({
    content: '',
    type: 'ai',
    time: formatTime(new Date())
  });

  currentResponse = '';

  const messageToSend = {
    role: "assistant",
    style: "专业",
    question: trimmedValue,
    chatHistory: chatHistory.slice(0, -1)
  };

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
    container.scrollTo({
      top: container.scrollHeight,
      behavior: 'smooth'
    });
  }
};

const handleEnterKey = (e: KeyboardEvent) => {
  if (e.shiftKey) {
    e.preventDefault();
    handleSearch();
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

// 键盘快捷键处理
const handleKeydown = (e: KeyboardEvent) => {
  if (e.ctrlKey && e.key === '/') {
    e.preventDefault();
    toggleChatDrawer();
  }
};

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown);

  if (socket) {
    socket.close();
    socket = null;
  }
});
</script>

<style scoped>
/* 代码高亮样式 */
:deep(.hljs) {
  display: block;
  overflow-x: auto;
  padding: 1.2em;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  color: #e2e8f0;
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  border-radius: 12px;
  margin: 16px 0;
  font-size: 14px;
  line-height: 1.6;
  border: 1px solid rgba(148, 163, 184, 0.1);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

:deep(.hljs-keyword), :deep(.hljs-selector-tag), :deep(.hljs-addition) {
  color: #f472b6;
}

:deep(.hljs-number), :deep(.hljs-string), :deep(.hljs-meta .hljs-meta-string), 
:deep(.hljs-literal), :deep(.hljs-doctag), :deep(.hljs-regexp) {
  color: #4ade80;
}

:deep(.hljs-title), :deep(.hljs-section), :deep(.hljs-name), 
:deep(.hljs-selector-id), :deep(.hljs-selector-class) {
  color: #60a5fa;
}

:deep(.hljs-attribute), :deep(.hljs-attr), :deep(.hljs-variable), 
:deep(.hljs-template-variable), :deep(.hljs-class .hljs-title), :deep(.hljs-type) {
  color: #fbbf24;
}

:deep(.hljs-symbol), :deep(.hljs-bullet), :deep(.hljs-subst), :deep(.hljs-meta), 
:deep(.hljs-meta .hljs-keyword), :deep(.hljs-selector-attr), 
:deep(.hljs-selector-pseudo), :deep(.hljs-link) {
  color: #e879f9;
}

:deep(.hljs-built_in), :deep(.hljs-deletion) {
  color: #a78bfa;
}

:deep(.hljs-comment), :deep(.hljs-quote) {
  color: #94a3b8;
  font-style: italic;
}

.ai-assistant-container {
  position: relative;
}

/* 悬浮按钮样式 */
.assistant-float-button {
  box-shadow: 0 8px 32px rgba(59, 130, 246, 0.3);
  transition: all 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.assistant-float-button:hover {
  transform: translateY(-4px) scale(1.08);
  box-shadow: 0 12px 40px rgba(59, 130, 246, 0.4);
}

.float-button-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.tooltip-content {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  font-size: 14px;
  font-weight: 500;
}

/* 抽屉样式 */
.ai-chat-drawer :deep(.ant-drawer-content) {
  background: linear-gradient(to bottom, #0f172a 0%, #1e293b 100%);
  border-radius: 24px 0 0 24px;
  overflow: hidden;
  box-shadow: -20px 0 60px rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(20px);
}

.ai-chat-drawer :deep(.ant-drawer-body) {
  padding: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* 抽屉头部 */
.drawer-header {
  padding: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
  backdrop-filter: blur(10px);
}

.drawer-title {
  display: flex;
  align-items: center;
  gap: 16px;
  color: #f8fafc;
}

.title-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border-radius: 12px;
  color: white;
  box-shadow: 0 4px 20px rgba(59, 130, 246, 0.3);
}

.title-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-text {
  font-weight: 700;
  font-size: 18px;
  letter-spacing: -0.025em;
}

.title-subtitle {
  font-size: 13px;
  color: #94a3b8;
  font-weight: 500;
}

.drawer-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-button {
  color: #94a3b8;
  transition: all 0.3s ease;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.action-button:hover {
  color: #f8fafc;
  background: rgba(148, 163, 184, 0.1);
  transform: scale(1.05);
}

/* 状态栏 */
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: rgba(15, 23, 42, 0.5);
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  backdrop-filter: blur(10px);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
  transition: all 0.3s ease;
}

.status-dot.online {
  background: #10b981;
  box-shadow: 0 0 10px rgba(16, 185, 129, 0.5);
}

.status-text {
  font-size: 13px;
  color: #94a3b8;
  font-weight: 500;
}

.message-count {
  font-size: 13px;
  color: #64748b;
}

/* 聊天容器 */
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

/* 消息区域 */
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  scroll-behavior: smooth;
  min-height: 0;
}

.chat-messages::-webkit-scrollbar {
  width: 4px;
}

.chat-messages::-webkit-scrollbar-track {
  background: transparent;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.2);
  border-radius: 2px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.4);
}

/* 消息样式 */
.message {
  margin-bottom: 32px;
  animation: messageSlideIn 0.4s ease-out;
}

@keyframes messageSlideIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.message-wrapper {
  display: flex;
  gap: 16px;
}

.avatar {
  flex-shrink: 0;
}

.avatar-container {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.user-avatar {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white;
}

.ai-avatar {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
}

.avatar-container:hover {
  transform: scale(1.05);
}

.content {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.name {
  font-weight: 600;
  font-size: 15px;
  color: #f1f5f9;
  letter-spacing: -0.025em;
}

.time {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

.text {
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  padding: 20px 24px;
  border-radius: 16px;
  color: #f1f5f9;
  line-height: 1.7;
  font-size: 15px;
  word-break: break-word;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.05);
  position: relative;
  transition: all 0.3s ease;
}

.message.user .text {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(37, 99, 235, 0.2);
}

.text:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.15);
}

.text :deep(p) {
  margin: 12px 0;
}

.text :deep(p:first-child) {
  margin-top: 0;
}

.text :deep(p:last-child) {
  margin-bottom: 0;
}

.text :deep(ul), .text :deep(ol) {
  padding-left: 24px;
  margin: 16px 0;
}

.text :deep(li) {
  margin-bottom: 8px;
  line-height: 1.6;
}

.text :deep(code:not(pre code)) {
  background: rgba(15, 23, 42, 0.6);
  padding: 4px 8px;
  border-radius: 6px;
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  color: #7dd3fc;
  border: 1px solid rgba(148, 163, 184, 0.1);
}

.text :deep(a) {
  color: #38bdf8;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s ease;
}

.text :deep(a:hover) {
  color: #0ea5e9;
  text-decoration: underline;
}

.text :deep(h1), .text :deep(h2), .text :deep(h3), 
.text :deep(h4), .text :deep(h5), .text :deep(h6) {
  margin-top: 24px;
  margin-bottom: 16px;
  color: #f8fafc;
  font-weight: 700;
  letter-spacing: -0.025em;
}

.text :deep(blockquote) {
  border-left: 4px solid #3b82f6;
  padding: 16px 20px;
  margin: 20px 0;
  background: rgba(59, 130, 246, 0.05);
  border-radius: 0 12px 12px 0;
  font-style: italic;
  color: #e2e8f0;
}

.message-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  opacity: 0;
  transition: all 0.3s ease;
}

.message:hover .message-actions {
  opacity: 1;
}

.message-action-btn {
  color: #64748b;
  transition: all 0.2s ease;
  padding: 4px 8px;
  border-radius: 8px;
  height: auto;
}

.message-action-btn:hover {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.message-action-btn .liked {
  color: #3b82f6;
}

/* 打字指示器 */
.typing-indicator {
  display: flex;
  gap: 16px;
  margin-bottom: 32px;
  animation: messageSlideIn 0.4s ease-out;
}

.typing-content {
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  padding: 20px 24px;
  border-radius: 16px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.05);
  display: flex;
  align-items: center;
  gap: 16px;
}

.typing-animation {
  display: flex;
  gap: 4px;
}

.typing-animation span {
  height: 8px;
  width: 8px;
  background: #3b82f6;
  border-radius: 50%;
  display: block;
  animation: typing 1.4s infinite ease-in-out;
}

.typing-animation span:nth-child(1) {
  animation-delay: 0s;
}

.typing-animation span:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-animation span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes typing {
  0%, 80%, 100% {
    transform: scale(0.8);
    opacity: 0.5;
  }
  40% {
    transform: scale(1.2);
    opacity: 1;
  }
}

.typing-text {
  color: #94a3b8;
  font-size: 14px;
  font-weight: 500;
}

/* 快捷操作 */
.quick-actions {
  padding: 16px 24px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  background: rgba(15, 23, 42, 0.3);
}

.quick-action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.quick-action-btn {
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  color: #3b82f6;
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.quick-action-btn:hover {
  background: rgba(59, 130, 246, 0.2);
  border-color: rgba(59, 130, 246, 0.4);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
}

/* 输入区域 */
.chat-input {
  padding: 24px;
  background: rgba(15, 23, 42, 0.5);
  backdrop-filter: blur(10px);
  border-top: 1px solid rgba(148, 163, 184, 0.1);
}

.textarea-container {
  margin-bottom: 12px;
}

.input-wrapper {
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  border-radius: 20px;
  padding: 16px 20px;
  border: 1px solid rgba(148, 163, 184, 0.1);
  transition: all 0.3s ease;
  display: flex;
  align-items: flex-end;
  gap: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.input-wrapper:focus-within {
  border-color: rgba(59, 130, 246, 0.3);
  box-shadow: 0 4px 20px rgba(59, 130, 246, 0.1);
  background: linear-gradient(135deg, #334155 0%, #475569 100%);
}

.message-input {
  flex: 1;
  background: transparent !important;
  border: none !important;
  padding: 0 !important;
  color: #f1f5f9 !important;
  font-size: 15px !important;
  line-height: 1.6 !important;
  resize: none !important;
}

.message-input:focus {
  box-shadow: none !important;
  outline: none !important;
}

.message-input::placeholder {
  color: #64748b !important;
}

.input-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.input-action-btn {
  color: #64748b;
  transition: all 0.2s ease;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.input-action-btn:hover {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.send-button {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border: none;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  box-shadow: 0 4px 16px rgba(59, 130, 246, 0.3);
}

.send-button:hover {
  transform: translateY(-2px) scale(1.05);
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.4);
}

.send-button:disabled {
  opacity: 0.6;
  transform: none;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.2);
}

/* 输入提示 */
.input-hints {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
  padding: 0 4px;
}

.hints-left {
  display: flex;
  gap: 16px;
}

.hint-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

.hints-right {
  display: flex;
  align-items: center;
}

.shortcut-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #64748b;
}

.shortcut-key {
  background: rgba(51, 65, 85, 0.6);
  padding: 4px 8px;
  border-radius: 6px;
  font-family: 'JetBrains Mono', monospace;
  color: #94a3b8;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  border: 1px solid rgba(148, 163, 184, 0.1);
}

/* 响应式调整 */
@media (max-width: 768px) {
  .ai-chat-drawer :deep(.ant-drawer-content-wrapper) {
    width: 100% !important;
  }

  .chat-messages {
    padding: 16px;
  }

  .chat-input {
    padding: 16px;
  }

  .quick-action-buttons {
    justify-content: center;
  }

  .input-hints {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }

  .hints-left {
    gap: 12px;
  }
}

/* 暗黑模式优化 */
@media (prefers-color-scheme: dark) {
  .ai-chat-drawer :deep(.ant-drawer-content) {
    background: linear-gradient(to bottom, #0a0f1a 0%, #111827 100%);
  }
}
</style>