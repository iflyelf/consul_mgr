<template>
  <div class="meta-editor">
    <!-- 模式切换 -->
    <div class="mode-switch">
      <el-radio-group v-model="mode" size="small">
        <el-radio-button value="form">表单模式</el-radio-button>
        <el-radio-button value="json">JSON 模式</el-radio-button>
      </el-radio-group>
      <el-button 
        v-if="mode === 'form'" 
        type="primary" 
        size="small" 
        :icon="Plus"
        @click="addItem"
      >
        添加字段
      </el-button>
    </div>
    
    <!-- 表单模式 -->
    <div v-if="mode === 'form'" class="form-mode">
      <el-empty v-if="formData.length === 0" description="暂无数据，点击添加字段" />
      <div
        v-for="(item, index) in formData"
        :key="index"
        class="meta-item"
      >
        <el-input
          v-model="item.key"
          placeholder="Key"
          style="width: 40%"
          @change="handleFormChange"
        />
        <el-input
          v-model="item.value"
          placeholder="Value"
          style="width: 45%; margin-left: 10px"
          @change="handleFormChange"
        />
        <el-button
          type="danger"
          :icon="Delete"
          circle
          @click="removeItem(index)"
          style="margin-left: 10px"
        />
      </div>
    </div>
    
    <!-- JSON 模式 -->
    <div v-else class="json-mode">
      <el-input
        v-model="jsonData"
        type="textarea"
        :rows="12"
        placeholder='请输入 JSON 格式的 Meta 数据，例如：
{
  "version": "1.0.0",
  "env": "production",
  "region": "cn-north"
}'
        @change="handleJsonChange"
        class="json-textarea"
      />
      <div v-if="jsonError" class="json-error">
        <el-alert
          :title="jsonError"
          type="error"
          :closable="false"
        />
      </div>
      <div v-if="!jsonError && jsonData" class="json-success">
        <el-alert
          title="JSON 格式正确"
          type="success"
          :closable="false"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

// 编辑模式：form（表单） / json（JSON）
const mode = ref('form')

// 表单模式数据
const formData = ref([])

// JSON 模式数据
const jsonData = ref('')
const jsonError = ref('')

// 初始化
watch(() => props.modelValue, (val) => {
  initData(val)
}, { immediate: true })

function initData(val) {
  if (!val || Object.keys(val).length === 0) {
    formData.value = []
    jsonData.value = ''
    return
  }
  
  // 转换为表单数据
  formData.value = Object.entries(val).map(([key, value]) => ({
    key,
    value: String(value)
  }))
  
  // 转换为 JSON 数据
  try {
    jsonData.value = JSON.stringify(val, null, 2)
    jsonError.value = ''
  } catch (error) {
    jsonData.value = ''
    jsonError.value = 'JSON 序列化失败'
  }
}

// 表单模式：添加字段
function addItem() {
  formData.value.push({ key: '', value: '' })
}

// 表单模式：删除字段
function removeItem(index) {
  formData.value.splice(index, 1)
  handleFormChange()
}

// 表单模式：数据变更
function handleFormChange() {
  const meta = {}
  formData.value.forEach(item => {
    if (item.key) {
      meta[item.key] = item.value
    }
  })
  
  emit('update:modelValue', meta)
  
  // 同步更新 JSON 数据
  try {
    jsonData.value = JSON.stringify(meta, null, 2)
    jsonError.value = ''
  } catch (error) {
    jsonError.value = 'JSON 序列化失败'
  }
}

// JSON 模式：数据变更
function handleJsonChange() {
  try {
    if (!jsonData.value.trim()) {
      emit('update:modelValue', {})
      formData.value = []
      jsonError.value = ''
      return
    }
    
    const meta = JSON.parse(jsonData.value)
    jsonError.value = ''
    
    // 更新表单数据
    formData.value = Object.entries(meta).map(([key, value]) => ({
      key,
      value: String(value)
    }))
    
    emit('update:modelValue', meta)
  } catch (error) {
    jsonError.value = 'JSON 格式错误: ' + error.message
  }
}
</script>

<style scoped>
.meta-editor {
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 15px;
  background: #ffffff;
}

.mode-switch {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.form-mode {
  min-height: 200px;
}

.meta-item {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.json-mode {
  min-height: 200px;
}

.json-textarea {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
}

.json-error {
  margin-top: 10px;
}

.json-success {
  margin-top: 10px;
}

:deep(.el-textarea__inner) {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}
</style>
