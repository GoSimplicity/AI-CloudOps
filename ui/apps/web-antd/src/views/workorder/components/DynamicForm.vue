<template>
    <div class="dynamic-form">
      <a-form
        ref="formRef"
        :model="localData"
        :rules="rules"
        layout="vertical"
        @finish="handleFinish"
        class="form-container"
      >
        <template v-for="field in fields" :key="field.id">
          <a-form-item
            v-if="!field.hidden"
            :name="field.name"
            :label="field.label"
            :required="field.required"
            class="form-field"
          >
            <template #label>
              <div class="field-label">
                <span>{{ field.label }}</span>
                <span v-if="field.required" class="required-mark">*</span>
              </div>
            </template>
  
            <!-- 文本输入框 -->
            <a-input
              v-if="field.type === 'text'"
              v-model:value="localData[field.name]"
              :placeholder="field.placeholder"
              :disabled="field.disabled"
              class="form-input"
              size="large"
            />
  
            <!-- 数字输入框 -->
            <a-input-number
              v-else-if="field.type === 'number'"
              v-model:value="localData[field.name]"
              :placeholder="field.placeholder"
              :disabled="field.disabled"
              style="width: 100%"
              class="form-input"
              size="large"
            />
  
            <!-- 日期选择器 -->
            <a-date-picker
              v-else-if="field.type === 'date'"
              v-model:value="localData[field.name]"
              :placeholder="field.placeholder"
              :disabled="field.disabled"
              style="width: 100%"
              class="form-input"
              size="large"
            />
  
            <!-- 下拉选择 -->
            <a-select
              v-else-if="field.type === 'select'"
              v-model:value="localData[field.name]"
              :placeholder="field.placeholder || '请选择'"
              :disabled="field.disabled"
              style="width: 100%"
              class="form-input"
              size="large"
            >
              <a-select-option
                v-for="option in field.options"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </a-select-option>
            </a-select>
  
            <!-- 单选框组 -->
            <a-radio-group
              v-else-if="field.type === 'radio'"
              v-model:value="localData[field.name]"
              :disabled="field.disabled"
              class="radio-group"
            >
              <div class="radio-options">
                <a-radio
                  v-for="option in field.options"
                  :key="option.value"
                  :value="option.value"
                  class="radio-option"
                >
                  {{ option.label }}
                </a-radio>
              </div>
            </a-radio-group>
  
            <!-- 复选框组 -->
            <a-checkbox-group
              v-else-if="field.type === 'checkbox'"
              v-model:value="localData[field.name]"
              :disabled="field.disabled"
              class="checkbox-group"
            >
              <div class="checkbox-options">
                <a-checkbox
                  v-for="option in field.options"
                  :key="option.value"
                  :value="option.value"
                  class="checkbox-option"
                >
                  {{ option.label }}
                </a-checkbox>
              </div>
            </a-checkbox-group>
  
            <!-- 多行文本 -->
            <a-textarea
              v-else-if="field.type === 'textarea'"
              v-model:value="localData[field.name]"
              :placeholder="field.placeholder"
              :disabled="field.disabled"
              :rows="4"
              class="form-input"
              size="large"
            />
          </a-form-item>
        </template>
  
        <div class="form-actions" v-if="showActions">
          <a-space size="large">
            <a-button type="primary" html-type="submit" size="large" class="submit-btn">
              <FormOutlined />
              提交表单
            </a-button>
            <a-button @click="resetForm" size="large" class="reset-btn">
              <ReloadOutlined />
              重置表单
            </a-button>
          </a-space>
        </div>
      </a-form>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, watch } from 'vue';
  import { FormOutlined, ReloadOutlined } from '@ant-design/icons-vue';
  import type { FormInstance } from 'ant-design-vue';
  
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
  }
  
  interface Props {
    fields: FormField[];
    data?: Record<string, any>;
    rules?: Record<string, any[]>;
    showActions?: boolean;
  }
  
  interface Emits {
    (e: 'update:data', data: Record<string, any>): void;
    (e: 'submit', data: Record<string, any>): void;
  }
  
  const props = withDefaults(defineProps<Props>(), {
    data: () => ({}),
    rules: () => ({}),
    showActions: true
  });
  
  const emit = defineEmits<Emits>();
  
  const formRef = ref<FormInstance>();
  const localData = ref<Record<string, any>>({ ...(props.data || {}) });
  
  // 标记数据来源以防止循环更新
  let isUpdatingFromProps = false;
  let isUpdatingFromLocal = false;
  
  // 监听数据变化
  watch(
    localData,
    (newData) => {
      if (!isUpdatingFromProps) {
        isUpdatingFromLocal = true;
        emit('update:data', JSON.parse(JSON.stringify(newData)));
        // 重置标志以允许后续更新
        setTimeout(() => {
          isUpdatingFromLocal = false;
        }, 0);
      }
    },
    { deep: true }
  );
  
  // 监听外部数据变化
  watch(
    () => props.data,
    (newData) => {
      if (!isUpdatingFromLocal && JSON.stringify(localData.value) !== JSON.stringify(newData)) {
        isUpdatingFromProps = true;
        // 使用深拷贝避免引用问题
        localData.value = JSON.parse(JSON.stringify(newData || {}));
        // 重置标志以允许后续更新
        setTimeout(() => {
          isUpdatingFromProps = false;
        }, 0);
      }
    },
    { deep: true }
  );
  
  // 处理表单提交
  const handleFinish = (values: Record<string, any>) => {
    emit('submit', values);
  };
  
  // 重置表单
  const resetForm = () => {
    formRef.value?.resetFields();
  };
  
  // 验证表单
  const validate = () => {
    return formRef.value?.validate();
  };
  
  // 暴露方法给父组件
  defineExpose({
    validate,
    resetForm
  });
  </script>
  
  <style scoped>
  .dynamic-form {
    max-width: 100%;
    margin: 0 auto;
  }
  
  .form-container {
    background: white;
    border-radius: 12px;
    padding: 32px;
    border: 1px solid #e8e8e8;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  }
  
  .form-field {
    margin-bottom: 28px;
  }
  
  .field-label {
    display: flex;
    align-items: center;
    gap: 4px;
    font-weight: 600;
    color: #374151;
    font-size: 15px;
  }
  
  .required-mark {
    color: #ef4444;
    font-weight: 500;
  }
  
  .form-input {
    border-radius: 8px;
    border: 1.5px solid #e5e7eb;
    transition: all 0.3s ease;
    font-size: 15px;
  }
  
  .form-input:hover {
    border-color: #93c5fd;
    box-shadow: 0 2px 4px rgba(147, 197, 253, 0.1);
  }
  
  .form-input:focus,
  .form-input:focus-within {
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }
  
  .radio-group,
  .checkbox-group {
    width: 100%;
  }
  
  .radio-options,
  .checkbox-options {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: #f8fafc;
    border: 1.5px solid #e5e7eb;
    border-radius: 8px;
    transition: all 0.3s ease;
  }
  
  .radio-options:hover,
  .checkbox-options:hover {
    border-color: #93c5fd;
    background: #eff6ff;
  }
  
  .radio-option,
  .checkbox-option {
    padding: 12px 16px;
    border-radius: 6px;
    transition: all 0.2s ease;
    background: white;
    border: 1px solid #e5e7eb;
    font-size: 15px;
    font-weight: 500;
  }
  
  .radio-option:hover,
  .checkbox-option:hover {
    border-color: #3b82f6;
    background: #eff6ff;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(59, 130, 246, 0.1);
  }
  
  .radio-option :deep(.ant-radio-checked .ant-radio-inner),
  .checkbox-option :deep(.ant-checkbox-checked .ant-checkbox-inner) {
    background-color: #3b82f6;
    border-color: #3b82f6;
  }
  
  .form-actions {
    margin-top: 40px;
    padding-top: 32px;
    border-top: 2px solid #f3f4f6;
    text-align: center;
  }
  
  .submit-btn {
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    border: none;
    border-radius: 8px;
    font-weight: 600;
    height: 48px;
    padding: 0 32px;
    font-size: 16px;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
    transition: all 0.3s ease;
  }
  
  .submit-btn:hover {
    background: linear-gradient(135deg, #1d4ed8 0%, #1e40af 100%);
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(59, 130, 246, 0.3);
  }
  
  .reset-btn {
    border: 2px solid #e5e7eb;
    border-radius: 8px;
    font-weight: 600;
    height: 48px;
    padding: 0 32px;
    font-size: 16px;
    color: #6b7280;
    transition: all 0.3s ease;
  }
  
  .reset-btn:hover {
    border-color: #9ca3af;
    color: #374151;
    transform: translateY(-1px);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
  
  /* 响应式设计 */
  @media (max-width: 768px) {
    .form-container {
      padding: 20px;
      border-radius: 8px;
    }
    
    .form-field {
      margin-bottom: 20px;
    }
    
    .field-label {
      font-size: 14px;
    }
    
    .form-input {
      font-size: 14px;
    }
    
    .radio-options,
    .checkbox-options {
      padding: 12px;
    }
    
    .radio-option,
    .checkbox-option {
      padding: 10px 12px;
      font-size: 14px;
    }
    
    .form-actions {
      margin-top: 32px;
      padding-top: 24px;
    }
    
    .submit-btn,
    .reset-btn {
      width: 100%;
      margin-bottom: 12px;
      height: 44px;
      font-size: 15px;
    }
    
    .form-actions .ant-space {
      width: 100%;
      flex-direction: column;
    }
  }
  
  @media (max-width: 480px) {
    .form-container {
      padding: 16px;
    }
    
    .radio-options,
    .checkbox-options {
      gap: 8px;
    }
    
    .submit-btn,
    .reset-btn {
      height: 40px;
      font-size: 14px;
    }
  }
  
  /* 动画效果 */
  .form-field {
    animation: fadeInUp 0.3s ease-out;
  }
  
  @keyframes fadeInUp {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  
  /* 输入状态指示 */
  .form-input:invalid {
    border-color: #ef4444;
  }
  
  .form-input:valid {
    border-color: #10b981;
  }
  
  /* 禁用状态 */
  .form-input:disabled {
    background-color: #f9fafb;
    border-color: #d1d5db;
    color: #9ca3af;
    cursor: not-allowed;
  }
  
  .radio-option:has(.ant-radio-wrapper-disabled),
  .checkbox-option:has(.ant-checkbox-wrapper-disabled) {
    background-color: #f9fafb;
    border-color: #d1d5db;
    color: #9ca3af;
    cursor: not-allowed;
  }
  
  .radio-option:has(.ant-radio-wrapper-disabled):hover,
  .checkbox-option:has(.ant-checkbox-wrapper-disabled):hover {
    transform: none;
    box-shadow: none;
  }
  </style>