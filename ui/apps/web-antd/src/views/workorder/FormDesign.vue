<template>
  <div class="dynamic-form-container">
    <!-- 表单设计器 -->
    <div class="form-designer" v-if="showDesigner">
      <a-card class="designer-card">
        <template #title>
          <div class="card-title">
            <FormOutlined />
            <span>表单设计器</span>
          </div>
        </template>
        <template #extra>
          <a-space>
            <a-button @click="togglePreview" type="primary" class="btn-primary">
              <template #icon>
                <EyeOutlined />
              </template>
              预览表单
            </a-button>
            <a-button @click="clearForm" type="default">
              <template #icon>
                <DeleteOutlined />
              </template>
              清空表单
            </a-button>
          </a-space>
        </template>

        <div class="designer-content">
          <!-- 字段类型选择区域 -->
          <div class="field-controls">
            <div class="controls-header">
              <h4>选择字段类型</h4>
              <span class="controls-desc">点击下方按钮添加表单字段</span>
            </div>
            <div class="field-buttons">
              <a-button @click="addFieldType('text')" class="field-btn">
                <PlusOutlined />
                <span>文本框</span>
              </a-button>
              <a-button @click="addFieldType('number')" class="field-btn">
                <PlusOutlined />
                <span>数字</span>
              </a-button>
              <a-button @click="addFieldType('date')" class="field-btn">
                <PlusOutlined />
                <span>日期</span>
              </a-button>
              <a-button @click="addFieldType('select')" class="field-btn">
                <PlusOutlined />
                <span>下拉选项</span>
              </a-button>
              <a-button @click="addFieldType('checkbox')" class="field-btn">
                <PlusOutlined />
                <span>复选框</span>
              </a-button>
              <a-button @click="addFieldType('radio')" class="field-btn">
                <PlusOutlined />
                <span>单选框</span>
              </a-button>
              <a-button @click="addFieldType('textarea')" class="field-btn">
                <PlusOutlined />
                <span>多行文本</span>
              </a-button>
            </div>
          </div>

          <!-- 配置管理操作区域 -->
          <div class="config-actions">
            <div class="actions-header">
              <h4>配置管理</h4>
              <span class="actions-desc">导入、导出和管理表单配置</span>
            </div>
            <div class="action-buttons">
              <a-button @click="toggleJsonViewer" class="action-btn">
                <template #icon>
                  <CodeOutlined />
                </template>
                <span>查看 JSON</span>
              </a-button>
              <a-button @click="copyJson" class="action-btn">
                <template #icon>
                  <CopyOutlined />
                </template>
                <span>复制配置</span>
              </a-button>
              <a-button @click="exportConfig" class="action-btn">
                <template #icon>
                  <DownloadOutlined />
                </template>
                <span>导出文件</span>
              </a-button>
              <a-upload 
                :show-upload-list="false"
                :before-upload="importConfig"
                accept=".json"
                class="upload-wrapper"
              >
                <a-button class="action-btn">
                  <template #icon>
                    <UploadOutlined />
                  </template>
                  <span>导入配置</span>
                </a-button>
              </a-upload>
            </div>
          </div>

          <!-- 字段列表 -->
          <div class="field-list" v-if="formConfig.fields.length > 0">
            <div class="list-header">
              <h4>表单字段配置</h4>
              <span class="field-count">共 {{ formConfig.fields.length }} 个字段</span>
            </div>
            <a-collapse>
              <a-collapse-panel 
                v-for="(field, index) in formConfig.fields" 
                :key="field.id"
                :header="getFieldDisplayName(field)"
                class="field-panel"
              >
                <template #extra>
                  <div class="field-actions" @click.stop>
                    <a-tooltip title="上移">
                      <a-button 
                        type="text" 
                        size="small" 
                        @click="moveField(index, -1)"
                        :disabled="index === 0"
                        class="action-btn-small"
                      >
                        <UpOutlined />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip title="下移">
                      <a-button 
                        type="text" 
                        size="small" 
                        @click="moveField(index, 1)"
                        :disabled="index === formConfig.fields.length - 1"
                        class="action-btn-small"
                      >
                        <DownOutlined />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip title="删除">
                      <a-button 
                        type="text" 
                        danger 
                        size="small" 
                        @click="removeField(index)"
                        class="action-btn-small"
                      >
                        <DeleteOutlined />
                      </a-button>
                    </a-tooltip>
                  </div>
                </template>

                <FieldConfig 
                  :field="field" 
                  @update="updateField(index, $event)"
                />
              </a-collapse-panel>
            </a-collapse>
          </div>

          <div v-else class="empty-form">
            <a-empty description="暂无表单字段">
              <template #image>
                <FormOutlined style="font-size: 48px; color: #d9d9d9;" />
              </template>
              <div class="empty-desc">
                <p>请在上方选择字段类型开始设计表单</p>
                <a-button type="primary" @click="addFieldType('text')" ghost>
                  <PlusOutlined /> 添加第一个字段
                </a-button>
              </div>
            </a-empty>
          </div>
        </div>
      </a-card>
    </div>

    <!-- 表单预览 -->
    <div class="form-preview" v-if="!showDesigner">
      <a-card class="preview-card">
        <template #title>
          <div class="card-title">
            <EyeOutlined />
            <span>表单预览</span>
          </div>
        </template>
        <template #extra>
          <a-space>
            <a-button @click="togglePreview" type="default">
              <template #icon>
                <EditOutlined />
              </template>
              返回设计
            </a-button>
            <a-button @click="resetForm" type="default">
              重置数据
            </a-button>
            <a-button @click="submitForm" type="primary" class="btn-primary">
              提交表单
            </a-button>
          </a-space>
        </template>

        <div class="preview-content">
          <div class="preview-header">
            <h3>{{ formConfig.title }}</h3>
            <p v-if="formConfig.description" class="preview-description">
              {{ formConfig.description }}
            </p>
            <a-alert
              message="预览模式"
              description="您可以查看和填写表单字段，点击提交按钮查看表单数据。"
              type="info"
              show-icon
              banner
              class="preview-notice"
            />
          </div>
          
          <DynamicForm 
            :fields="formConfig.fields" 
            v-model:data="formData"
            :rules="formRules"
            @submit="handleSubmit"
            ref="dynamicFormRef"
          />
        </div>
      </a-card>
    </div>

    <!-- JSON 配置查看 -->
    <a-modal 
      :open="showJsonViewer" 
      title="表单配置 JSON" 
      :width="800"
      :footer="null" 
      @cancel="toggleJsonViewer"
      class="json-modal"
    >
      <div class="json-viewer">
        <div class="json-actions">
          <a-space>
            <a-button @click="copyJson" size="small">
              <CopyOutlined /> 复制配置
            </a-button>
            <a-button @click="exportConfig" size="small">
              <DownloadOutlined /> 导出文件
            </a-button>
          </a-space>
        </div>
        <pre class="json-content">{{ JSON.stringify(formConfig.fields, null, 2) }}</pre>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue';
import { message } from 'ant-design-vue';
import {
  PlusOutlined,
  DeleteOutlined,
  EyeOutlined,
  EditOutlined,
  FormOutlined,
  UpOutlined,
  DownOutlined,
  CodeOutlined,
  DownloadOutlined,
  UploadOutlined,
  CopyOutlined
} from '@ant-design/icons-vue';
import FieldConfig from './components/FieldConfig.vue';
import DynamicForm from './components/DynamicForm.vue';

// 字段类型定义
interface FormField {
  id: string;
  type: 'text' | 'number' | 'date' | 'select' | 'checkbox' | 'radio' | 'textarea';
  label: string;
  name: string;
  required: boolean;
  placeholder?: string;
  defaultValue?: any;
  options?: Array<{ label: string; value: any }>;
  rules?: any[];
  disabled?: boolean;
  hidden?: boolean;
  sort_order?: number;
}

interface FormConfig {
  title: string;
  description?: string;
  fields: FormField[];
}

// 响应式数据
const showDesigner = ref(true);
const showJsonViewer = ref(false);
const dynamicFormRef = ref();

const formConfig = reactive<FormConfig>({
  title: '动态表单',
  description: '这是一个通过可视化设计器生成的动态表单',
  fields: []
});

const formData = ref<Record<string, any>>({});
const formRules = computed(() => {
  const rules: Record<string, any[]> = {};
  formConfig.fields.forEach(field => {
    if (field.required) {
      rules[field.name] = [
        { required: true, message: `请输入${field.label}`, trigger: 'blur' }
      ];
    }
    if (field.rules) {
      rules[field.name] = [...(rules[field.name] || []), ...field.rules];
    }
  });
  return rules;
});

// 字段类型映射
const fieldTypeMap = {
  text: '文本框',
  number: '数字',
  date: '日期',
  select: '下拉选项',
  checkbox: '复选框',
  radio: '单选框',
  textarea: '多行文本'
};

// 生成唯一ID
const generateId = () => `field_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

// 获取字段显示名称
const getFieldDisplayName = (field: FormField) => {
  const typeText = fieldTypeMap[field.type];
  const labelText = field.label || '未命名';
  const requiredText = field.required ? '（必填）' : '（可选）';
  return `${typeText} - ${labelText} ${requiredText}`;
};

// 添加字段
const addFieldType = (type: FormField['type']) => {
  const fieldNumber = formConfig.fields.length + 1;
  const field: FormField = {
    id: generateId(),
    type,
    label: `${fieldTypeMap[type]}${fieldNumber}`,
    name: `field_${fieldNumber}`,
    required: false,
    placeholder: `请输入${fieldTypeMap[type]}`,
    defaultValue: getDefaultValue(type),
    options: needsOptions(type) ? [
      { label: '选项1', value: 'option1' },
      { label: '选项2', value: 'option2' }
    ] : undefined,
    sort_order: fieldNumber,
    disabled: false,
    hidden: false
  };
  
  formConfig.fields.push(field);
  message.success(`已添加${fieldTypeMap[type]}字段`);
};

// 获取默认值
const getDefaultValue = (type: FormField['type']) => {
  switch (type) {
    case 'checkbox':
      return [];
    case 'number':
      return undefined;
    case 'date':
      return undefined;
    default:
      return '';
  }
};

// 判断是否需要选项
const needsOptions = (type: FormField['type']) => {
  return ['select', 'radio', 'checkbox'].includes(type);
};

// 更新字段
const updateField = (index: number, updatedField: FormField) => {
  // 使用深拷贝而非直接引用，避免响应式引用导致的循环更新
  formConfig.fields[index] = JSON.parse(JSON.stringify(updatedField));
};

// 移除字段
const removeField = (index: number) => {
  const field = formConfig.fields[index];
  formConfig.fields.splice(index, 1);
  
  // 从表单数据中移除对应字段
  if (field && field.name in formData.value) {
    delete formData.value[field.name];
  }
  
  message.success(`已删除字段: ${field?.label}`);
};

// 移动字段
const moveField = (index: number, direction: number) => {
  const newIndex = index + direction;
  if (newIndex >= 0 && newIndex < formConfig.fields.length) {
    const temp = formConfig.fields[index];
    formConfig.fields[index] = formConfig.fields[newIndex]!;
    formConfig.fields[newIndex] = temp!;
    
    // 更新移动后的字段顺序
    updateSortOrders();
    
    message.success('字段位置已调整');
  }
};

// 更新所有字段的排序顺序
const updateSortOrders = () => {
  formConfig.fields.forEach((field, index) => {
    field.sort_order = index + 1;
  });
};

// 切换预览模式
const togglePreview = () => {
  if (showDesigner.value && formConfig.fields.length === 0) {
    message.warning('请先添加表单字段');
    return;
  }
  
  showDesigner.value = !showDesigner.value;
  if (!showDesigner.value) {
    initFormData();
  }
};

// 初始化表单数据
const initFormData = () => {
  const data: Record<string, any> = {};
  formConfig.fields.forEach(field => {
    if (field.defaultValue !== undefined) {
      data[field.name] = field.defaultValue;
    } else {
      data[field.name] = getDefaultValue(field.type);
    }
  });
  formData.value = data;
};

// 重置表单数据
const resetForm = () => {
  initFormData();
  if (dynamicFormRef.value) {
    dynamicFormRef.value.resetForm();
  }
  message.success('表单数据已重置');
};

// 清空表单配置
const clearForm = () => {
  if (formConfig.fields.length === 0) {
    message.info('表单已为空');
    return;
  }
  
  formConfig.fields = [];
  formData.value = {};
  message.success('表单配置已清空');
};

// 提交表单
const submitForm = async () => {
  if (dynamicFormRef.value) {
    try {
      await dynamicFormRef.value.validate();
      handleSubmit(formData.value);
    } catch (error) {
      message.error('表单验证失败，请检查输入');
    }
  }
};

// 处理表单提交
const handleSubmit = (data: Record<string, any>) => {
  console.log('表单数据:', data);
  message.success('表单提交成功！数据已输出到控制台');
  // 这里可以调用API提交数据
};

// 切换JSON查看器
const toggleJsonViewer = () => {
  showJsonViewer.value = !showJsonViewer.value;
};

// 复制JSON配置
const copyJson = async () => {
  try {
    await navigator.clipboard.writeText(JSON.stringify(formConfig.fields, null, 2));
    message.success('配置已复制到剪贴板');
  } catch (error) {
    message.error('复制失败，请手动复制');
  }
};

// 导出配置
const exportConfig = () => {
  const dataStr = JSON.stringify(formConfig.fields, null, 2);
  const dataBlob = new Blob([dataStr], { type: 'application/json' });
  const url = URL.createObjectURL(dataBlob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `form-fields-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success('字段配置导出成功');
};

// 导入配置
const importConfig = (file: File) => {
  const reader = new FileReader();
  reader.onload = (e) => {
    try {
      const fields = JSON.parse(e.target?.result as string);
      if (Array.isArray(fields)) {
        formConfig.fields = fields;
        message.success('字段配置导入成功');
      } else if (fields.fields && Array.isArray(fields.fields)) {
        // 兼容旧格式
        formConfig.fields = fields.fields;
        message.success('字段配置导入成功');
      } else {
        message.error('字段配置文件格式不正确');
      }
    } catch (error) {
      message.error('字段配置文件解析失败');
    }
  };
  reader.readAsText(file);
  return false;
};
</script>

<style scoped>
.dynamic-form-container {
  padding: 20px;
  background-color: #f5f5f5;
  min-height: 100vh;
}

/* 卡片样式 */
.designer-card,
.preview-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #e8e8e8;
  margin-bottom: 20px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #1f2937;
}

/* 主要按钮样式 */
.btn-primary {
  background: linear-gradient(135deg, #1890ff 0%, #40a9ff 100%);
  border: none;
  box-shadow: 0 2px 4px rgba(24, 144, 255, 0.2);
}

.btn-primary:hover {
  background: linear-gradient(135deg, #40a9ff 0%, #1890ff 100%);
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.3);
}

/* 设计器内容 */
.designer-content {
  padding: 16px;
}

/* 字段控制区域 */
.field-controls {
  background: white;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 24px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.controls-header {
  margin-bottom: 20px;
  text-align: center;
}

.controls-header h4 {
  margin: 0 0 8px 0;
  color: #1f2937;
  font-size: 16px;
  font-weight: 600;
}

.controls-desc {
  color: #6b7280;
  font-size: 14px;
}

.field-buttons {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 12px;
}

.field-btn {
  height: 48px;
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.3s ease;
  background: #fafafa;
}

.field-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
  background: #f0f8ff;
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.15);
}

.field-btn span {
  font-size: 12px;
  font-weight: 500;
}

/* 配置管理区域 */
.config-actions {
  background: white;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 24px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.actions-header {
  margin-bottom: 20px;
  text-align: center;
}

.actions-header h4 {
  margin: 0 0 8px 0;
  color: #1f2937;
  font-size: 16px;
  font-weight: 600;
}

.actions-desc {
  color: #6b7280;
  font-size: 14px;
}

.action-buttons {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
}

.action-btn {
  height: 48px;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.3s ease;
  background: #fafafa;
  width: 100%;
}

.action-btn:hover {
  border-color: #40a9ff;
  color: #40a9ff;
  background: #f0f8ff;
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(64, 169, 255, 0.15);
}

.action-btn span {
  font-size: 12px;
  font-weight: 500;
}

/* 上传组件包装器 */
.upload-wrapper {
  width: 100%;
}

.upload-wrapper :deep(.ant-upload) {
  width: 100%;
  display: block;
}

/* 字段列表 */
.field-list {
  background: white;
  border-radius: 8px;
  padding: 24px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.list-header h4 {
  margin: 0;
  color: #1f2937;
  font-size: 16px;
  font-weight: 600;
}

.field-count {
  color: #6b7280;
  font-size: 14px;
  background: #f3f4f6;
  padding: 4px 12px;
  border-radius: 12px;
}

/* 字段面板 */
.field-panel :deep(.ant-collapse-header) {
  background: #fafafa;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  margin-bottom: 8px;
  padding: 12px 16px;
  font-weight: 500;
}

.field-panel :deep(.ant-collapse-content) {
  border: 1px solid #e8e8e8;
  border-top: none;
  border-radius: 0 0 8px 8px;
  margin-bottom: 8px;
}

.field-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.action-btn-small {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.action-btn-small:hover {
  background: #f0f0f0;
}

/* 空状态 */
.empty-form {
  background: white;
  border-radius: 8px;
  padding: 60px 24px;
  text-align: center;
  border: 1px solid #e8e8e8;
}

.empty-desc {
  margin-top: 16px;
}

.empty-desc p {
  color: #6b7280;
  margin-bottom: 16px;
}

/* 预览内容 */
.preview-content {
  padding: 24px;
  background: #fafafa;
  border-radius: 8px;
  margin: 16px;
}

.preview-header {
  text-align: center;
  margin-bottom: 32px;
  padding: 24px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}

.preview-header h3 {
  margin: 0 0 12px 0;
  font-size: 24px;
  color: #1f2937;
  font-weight: 600;
}

.preview-description {
  margin: 0 0 20px 0;
  color: #6b7280;
  font-size: 16px;
  line-height: 1.6;
}

.preview-notice {
  margin: 20px 0 0 0;
}

/* JSON 查看器 */
.json-modal :deep(.ant-modal-content) {
  border-radius: 8px;
}

.json-viewer {
  max-height: 60vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.json-actions {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.json-content {
  background-color: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow: auto;
  max-height: 50vh;
  color: #495057;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .dynamic-form-container {
    padding: 12px;
  }
  
  .field-buttons,
  .action-buttons {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .field-btn,
  .action-btn {
    height: 40px;
    font-size: 12px;
  }
  
  .field-btn span,
  .action-btn span {
    font-size: 11px;
  }
  
  .preview-content {
    padding: 16px;
    margin: 8px;
  }
  
  .preview-header {
    padding: 16px;
  }
  
  .preview-header h3 {
    font-size: 20px;
  }
  
  .list-header {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
  
  .field-actions {
    flex-wrap: wrap;
  }
}

@media (max-width: 480px) {
  .field-buttons,
  .action-buttons {
    grid-template-columns: 1fr;
  }
  
  .controls-header,
  .actions-header {
    text-align: left;
  }
  
  .preview-header h3 {
    font-size: 18px;
  }
  
  .preview-description {
    font-size: 14px;
  }
}

/* 动画效果 */
.field-panel {
  transition: all 0.3s ease;
}

.field-panel:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

/* 滚动条优化 */
.json-content::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.json-content::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.json-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.json-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>