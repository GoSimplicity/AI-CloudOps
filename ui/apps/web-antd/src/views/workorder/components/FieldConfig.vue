<template>
    <div class="field-config">
      <a-form layout="vertical" class="config-form">
        <div class="config-section">
          <div class="section-title">
            <span>基础配置</span>
          </div>
          <a-row :gutter="16">
            <a-col :span="24" :md="8">
              <a-form-item label="字段类型" class="form-item">
                <a-select v-model:value="localField.type" @change="handleTypeChange" class="field-select">
                  <a-select-option value="text">
                    <span class="option-content">📝 文本框</span>
                  </a-select-option>
                  <a-select-option value="number">
                    <span class="option-content">🔢 数字</span>
                  </a-select-option>
                  <a-select-option value="date">
                    <span class="option-content">📅 日期</span>
                  </a-select-option>
                  <a-select-option value="select">
                    <span class="option-content">📋 下拉选项</span>
                  </a-select-option>
                  <a-select-option value="checkbox">
                    <span class="option-content">☑️ 复选框</span>
                  </a-select-option>
                  <a-select-option value="radio">
                    <span class="option-content">🔘 单选框</span>
                  </a-select-option>
                  <a-select-option value="textarea">
                    <span class="option-content">📄 多行文本</span>
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
  
            <a-col :span="24" :md="8">
              <a-form-item label="标签名称" class="form-item">
                <a-input 
                  v-model:value="localField.label" 
                  placeholder="请输入字段标签" 
                  class="config-input"
                />
              </a-form-item>
            </a-col>
  
            <a-col :span="24" :md="8">
              <a-form-item label="字段名称" class="form-item">
                <a-input 
                  v-model:value="localField.name" 
                  placeholder="英文、数字、下划线" 
                  class="config-input"
                />
              </a-form-item>
            </a-col>
          </a-row>
  
          <a-row :gutter="16">
            <a-col :span="24" :md="12">
              <a-form-item label="占位符文本" class="form-item">
                <a-input 
                  v-model:value="localField.placeholder" 
                  placeholder="输入框的提示文本" 
                  class="config-input"
                />
              </a-form-item>
            </a-col>
  
            <a-col :span="24" :md="6">
              <a-form-item label="是否必填" class="form-item switch-item">
                <a-switch 
                  v-model:checked="localField.required" 
                  checked-children="必填"
                  un-checked-children="可选"
                />
              </a-form-item>
            </a-col>
  
            <a-col :span="24" :md="6">
              <a-form-item label="是否禁用" class="form-item switch-item">
                <a-switch 
                  v-model:checked="localField.disabled" 
                  checked-children="禁用"
                  un-checked-children="启用"
                />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
  
        <!-- 默认值配置 -->
        <div class="config-section" v-if="showDefaultValue">
          <div class="section-title">
            <span>默认值设置</span>
          </div>
          <a-row>
            <a-col :span="24">
              <a-form-item label="默认值" class="form-item">
                <template v-if="localField.type === 'text' || localField.type === 'textarea'">
                  <a-input 
                    v-model:value="localField.defaultValue" 
                    placeholder="请输入默认值" 
                    class="config-input"
                  />
                </template>
                <template v-else-if="localField.type === 'number'">
                  <a-input-number 
                    v-model:value="localField.defaultValue" 
                    placeholder="请输入默认数值"
                    style="width: 100%" 
                    class="config-input"
                  />
                </template>
                <template v-else-if="localField.type === 'select' || localField.type === 'radio'">
                  <a-select 
                    v-model:value="localField.defaultValue" 
                    placeholder="请选择默认值" 
                    class="config-input"
                    allow-clear
                  >
                    <a-select-option 
                      v-for="option in localField.options" 
                      :key="option.value" 
                      :value="option.value"
                    >
                      {{ option.label }}
                    </a-select-option>
                  </a-select>
                </template>
                <template v-else-if="localField.type === 'checkbox'">
                  <a-checkbox-group v-model:value="localField.defaultValue" class="checkbox-group">
                    <a-checkbox 
                      v-for="option in localField.options" 
                      :key="option.value" 
                      :value="option.value"
                      class="checkbox-item"
                    >
                      {{ option.label }}
                    </a-checkbox>
                  </a-checkbox-group>
                </template>
              </a-form-item>
            </a-col>
          </a-row>
        </div>
  
        <!-- 选项配置 -->
        <div class="config-section" v-if="needsOptions">
          <div class="section-title">
            <span>选项配置</span>
            <a-button type="link" @click="addOption" size="small" class="add-option-btn">
              <PlusOutlined /> 添加选项
            </a-button>
          </div>
          <div class="options-list">
            <div v-for="(option, index) in localField.options" :key="index" class="option-item">
              <div class="option-header">
                <span class="option-index">选项 {{ index + 1 }}</span>
                <a-button 
                  type="text" 
                  danger 
                  size="small" 
                  @click="removeOption(index)"
                  class="remove-option-btn"
                >
                  <DeleteOutlined />
                </a-button>
              </div>
              <a-row :gutter="12">
                <a-col :span="12">
                  <a-input 
                    v-model:value="option.label" 
                    placeholder="选项显示文本" 
                    class="option-input"
                  />
                </a-col>
                <a-col :span="12">
                  <a-input 
                    v-model:value="option.value" 
                    placeholder="选项值" 
                    class="option-input"
                  />
                </a-col>
              </a-row>
            </div>
            
            <div class="add-option-area" v-if="!localField.options || localField.options.length === 0">
              <a-button 
                type="dashed" 
                @click="addOption" 
                size="large"
                class="add-option-large"
              >
                <PlusOutlined /> 添加第一个选项
              </a-button>
            </div>
          </div>
        </div>
      </a-form>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, watch, computed, nextTick } from 'vue';
  import { PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue';
  
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
    field: FormField;
  }
  
  interface Emits {
    (e: 'update', field: FormField): void;
  }
  
  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
  
  // 创建本地字段的深拷贝而不是直接引用
  const localField = ref<FormField>(JSON.parse(JSON.stringify(props.field)));
  
  // 计算属性
  const needsOptions = computed(() => {
    return ['select', 'radio', 'checkbox'].includes(localField.value.type);
  });
  
  const showDefaultValue = computed(() => {
    return !['date'].includes(localField.value.type);
  });
  
  // 标记更新是否来自props
  let isUpdatingFromProps = false;
  
  // 监听字段变化并发出更新事件，添加防止循环更新的机制
  watch(
    localField,
    (newField) => {
      if (!isUpdatingFromProps) {
        emit('update', JSON.parse(JSON.stringify(newField)));
      }
    },
    { deep: true }
  );
  
  // 监听props变化
  watch(
    () => props.field,
    (newField) => {
      // 防止无限循环
      if (JSON.stringify(localField.value) !== JSON.stringify(newField)) {
        isUpdatingFromProps = true;
        // 使用nextTick确保DOM更新后再改变标志
        nextTick(() => {
          localField.value = JSON.parse(JSON.stringify(newField));
          // 异步重置标志以允许后续本地更改触发更新
          setTimeout(() => {
            isUpdatingFromProps = false;
          }, 0);
        });
      }
    },
    { deep: true }
  );
  
  // 处理类型变化
  const handleTypeChange = (type: FormField['type']) => {
    localField.value.type = type;
    
    // 根据类型初始化选项和默认值
    if (needsOptions.value && (!localField.value.options || localField.value.options.length === 0)) {
      localField.value.options = [
        { label: '选项1', value: 'option1' },
        { label: '选项2', value: 'option2' }
      ];
    } else if (!needsOptions.value) {
      localField.value.options = undefined;
    }
    
    // 重置默认值
    switch (type) {
      case 'checkbox':
        localField.value.defaultValue = [];
        break;
      case 'number':
        localField.value.defaultValue = undefined;
        break;
      case 'date':
        localField.value.defaultValue = undefined;
        break;
      default:
        localField.value.defaultValue = '';
    }
  };
  
  // 添加选项
  const addOption = () => {
    if (!localField.value.options) {
      localField.value.options = [];
    }
    const optionIndex = localField.value.options.length + 1;
    localField.value.options.push({
      label: `选项${optionIndex}`,
      value: `option${optionIndex}`
    });
  };
  
  // 删除选项
  const removeOption = (index: number) => {
    if (localField.value.options) {
      localField.value.options.splice(index, 1);
    }
  };
  </script>
  
  <style scoped>
  .field-config {
    background: #fafafa;
    border-radius: 8px;
    overflow: hidden;
  }
  
  .config-form {
    padding: 0;
  }
  
  .config-section {
    background: white;
    margin-bottom: 16px;
    border-radius: 8px;
    border: 1px solid #e8e8e8;
    overflow: hidden;
  }
  
  .section-title {
    background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
    padding: 12px 20px;
    border-bottom: 1px solid #e8e8e8;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 600;
    color: #495057;
    font-size: 14px;
  }
  
  .config-section .ant-row {
    padding: 20px;
  }
  
  .form-item {
    margin-bottom: 16px;
  }
  
  .form-item :deep(.ant-form-item-label) {
    font-weight: 500;
    color: #374151;
  }
  
  .switch-item {
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  
  .config-input,
  .field-select {
    border-radius: 6px;
    transition: all 0.3s ease;
  }
  
  .config-input:focus,
  .config-input:hover,
  .field-select:focus,
  .field-select:hover {
    border-color: #1890ff;
    box-shadow: 0 2px 4px rgba(24, 144, 255, 0.1);
  }
  
  .option-content {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .checkbox-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .checkbox-item {
    padding: 8px 12px;
    border: 1px solid #e8e8e8;
    border-radius: 6px;
    transition: all 0.2s ease;
  }
  
  .checkbox-item:hover {
    border-color: #1890ff;
    background: #f0f8ff;
  }
  
  .add-option-btn {
    color: #1890ff;
    padding: 0;
    height: auto;
  }
  
  .options-list {
    padding: 20px;
    padding-top: 0;
  }
  
  .option-item {
    background: #f8f9fa;
    border: 1px solid #e9ecef;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 12px;
    transition: all 0.3s ease;
  }
  
  .option-item:hover {
    border-color: #1890ff;
    box-shadow: 0 2px 8px rgba(24, 144, 255, 0.1);
  }
  
  .option-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }
  
  .option-index {
    font-weight: 500;
    color: #495057;
    font-size: 13px;
  }
  
  .remove-option-btn {
    color: #dc3545;
    padding: 4px;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
  }
  
  .remove-option-btn:hover {
    background: #f8d7da;
  }
  
  .option-input {
    border-radius: 6px;
  }
  
  .add-option-area {
    text-align: center;
    padding: 40px 20px;
  }
  
  .add-option-large {
    height: 48px;
    border: 2px dashed #d9d9d9;
    border-radius: 8px;
    font-weight: 500;
  }
  
  .add-option-large:hover {
    border-color: #1890ff;
    color: #1890ff;
  }
  
  @media (max-width: 768px) {
    .section-title {
      padding: 10px 16px;
      font-size: 13px;
    }
    
    .config-section .ant-row {
      padding: 16px;
    }
    
    .option-item {
      padding: 12px;
    }
    
    .option-header {
      flex-direction: column;
      gap: 8px;
      align-items: flex-start;
    }
    
    .options-list {
      padding: 16px;
      padding-top: 0;
    }
  }
  </style>