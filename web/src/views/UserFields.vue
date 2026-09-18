<template>
  <div class="user-fields-container">
    <!-- 自动同步配置 -->
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <div class="title">
            <el-icon><Refresh /></el-icon>
            <span>字段自动同步</span>
          </div>
          <el-tag :type="flyiamConfigured ? 'success' : 'warning'" size="small" effect="plain">
            {{ flyiamConfigured ? 'FlyIAM 已配置' : 'FlyIAM 未配置' }}
          </el-tag>
        </div>
      </template>

      <el-alert
        v-if="!flyiamConfigured"
        type="warning"
        :closable="false"
        show-icon
        title="未配置 FlyIAM 对接，无法同步。请设置 CONSUL_MGR_FLYIAM_API_ENDPOINT 与 CONSUL_MGR_FLYIAM_SERVICE_TOKEN。"
        style="margin-bottom: 16px"
      />

      <el-form :model="syncForm" label-width="96px" class="sync-form">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12" :md="8">
            <el-form-item label="定时同步">
              <el-switch v-model="syncForm.enabled" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8">
            <el-form-item label="启动时同步">
              <el-switch v-model="syncForm.syncOnStartup" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8">
            <el-form-item label="同步间隔">
              <el-input v-model="syncForm.interval" placeholder="如 6h、30m、1d" />
            </el-form-item>
          </el-col>
        </el-row>
        <div class="sync-actions">
          <el-button type="primary" :loading="savingCfg" @click="saveConfig">保存配置</el-button>
          <el-button type="success" :icon="Refresh" :loading="syncing" :disabled="!flyiamConfigured" @click="handleSync">
            立即同步
          </el-button>
          <span v-if="lastRunText" class="last-run">{{ lastRunText }}</span>
        </div>
      </el-form>

      <!-- 进度 -->
      <div v-if="progress.running" class="progress-box">
        <el-progress :percentage="100" :indeterminate="true" :duration="3" :stroke-width="10" :show-text="false" />
        <div class="progress-text">{{ progress.message || '同步中...' }}</div>
      </div>
      <div v-else-if="progress.status" class="progress-box">
        <el-tag :type="progress.status === 'success' ? 'success' : 'danger'" size="small" effect="dark">
          {{ progress.status === 'success' ? '成功' : '失败' }}
        </el-tag>
        <span class="progress-text">{{ progress.message }}</span>
      </div>
    </el-card>

    <!-- 字段定义 -->
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <div class="title">
            <el-icon><SetUp /></el-icon>
            <span>用户字段定义</span>
            <el-tag type="info" size="small">数据源字段变化无需改代码</el-tag>
          </div>
          <div class="header-actions">
            <el-button type="primary" :icon="Plus" @click="openCreate">新增字段</el-button>
            <el-button :icon="Refresh" @click="loadFields">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="label" label="显示名" min-width="150">
          <template #default="{ row }">
            <span>{{ row.label }}</span>
            <el-tag v-if="row.builtin" type="info" size="small" effect="plain" class="tag-gap">内置</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fieldKey" label="字段键" min-width="150" show-overflow-tooltip>
          <template #default="{ row }"><code class="key">{{ row.fieldKey }}</code></template>
        </el-table-column>
        <el-table-column prop="fieldType" label="类型" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ typeLabel(row.fieldType) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="列表显示" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.showInList ? 'success' : 'info'" size="small" effect="plain">
              {{ row.showInList ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="表单显示" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.showInForm ? 'success' : 'info'" size="small" effect="plain">
              {{ row.showInForm ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sortOrder" label="排序" width="80" align="center" />
        <el-table-column label="操作" width="130" align="center" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" :disabled="row.builtin" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="formVisible" :title="formIsEdit ? '编辑字段' : '新增字段'" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="字段键" required>
          <el-input v-model="form.fieldKey" :disabled="formIsEdit && form.builtin" placeholder="英文标识，如 jobLevel" />
          <div class="form-tip">作为用户属性键名，创建后不建议修改</div>
        </el-form-item>
        <el-form-item label="显示名" required>
          <el-input v-model="form.label" placeholder="如 职级" />
        </el-form-item>
        <el-form-item label="字段类型">
          <el-select v-model="form.fieldType" style="width: 100%">
            <el-option v-for="t in fieldTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.fieldType === 'select'" label="选项">
          <el-input v-model="form.options" type="textarea" :rows="3" placeholder='JSON 数组，如 ["P5","P6","P7"]' />
        </el-form-item>
        <el-form-item label="列表显示"><el-switch v-model="form.showInList" /></el-form-item>
        <el-form-item label="表单显示"><el-switch v-model="form.showInForm" /></el-form-item>
        <el-form-item label="可编辑"><el-switch v-model="form.editable" /></el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sortOrder" :min="0" :max="9999" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, SetUp } from '@element-plus/icons-vue'
import {
  listUserFields,
  createUserField,
  updateUserField,
  deleteUserField,
  syncUserFields,
  getSyncConfig,
  updateSyncConfig,
  getSyncProgress
} from '@/api/userField'

// formatDateTime 将时间格式化为 YYYY-MM-DD HH:mm:ss（本地时区）
const formatDateTime = (v) => {
  if (!v) return '-'
  const d = new Date(v)
  if (isNaN(d.getTime())) return String(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

const fieldTypes = [
  { value: 'text', label: '单行文本' },
  { value: 'textarea', label: '多行文本' },
  { value: 'number', label: '数字' },
  { value: 'select', label: '下拉选择' },
  { value: 'date', label: '日期' }
]

const loading = ref(false)
const saving = ref(false)
const savingCfg = ref(false)
const syncing = ref(false)
const list = ref([])

const flyiamConfigured = ref(false)
const syncForm = ref({ enabled: false, interval: '6h', syncOnStartup: false })
const progress = ref({ running: false, status: '', message: '' })
const lastRunText = ref('')

const formVisible = ref(false)
const formIsEdit = ref(false)
const emptyForm = () => ({
  fieldKey: '',
  label: '',
  fieldType: 'text',
  options: '',
  showInList: true,
  showInForm: true,
  editable: true,
  sortOrder: 100
})
const form = ref(emptyForm())

let pollTimer = null

const typeLabel = (t) => fieldTypes.find((x) => x.value === t)?.label || t

const loadFields = async () => {
  loading.value = true
  try {
    const res = await listUserFields()
    list.value = res || []
  } catch (e) {
    /* 拦截器已提示 */
  } finally {
    loading.value = false
  }
}

const loadConfig = async () => {
  try {
    const res = await getSyncConfig()
    const cfg = res?.config || {}
    syncForm.value = {
      enabled: !!cfg.enabled,
      interval: cfg.interval || '6h',
      syncOnStartup: !!cfg.syncOnStartup
    }
    flyiamConfigured.value = !!res?.flyiamConfigured
    if (cfg.lastRunAt) {
      const status = cfg.lastStatus === 'success' ? '成功' : cfg.lastStatus === 'failed' ? '失败' : '-'
      lastRunText.value = `上次同步：${formatDateTime(cfg.lastRunAt)}（${status}）`
    }
  } catch (e) {
    /* 拦截器已提示 */
  }
}

const loadProgress = async () => {
  try {
    const p = await getSyncProgress()
    progress.value = p || { running: false, status: '', message: '' }
  } catch (e) {
    /* 忽略 */
  }
}

const saveConfig = async () => {
  savingCfg.value = true
  try {
    await updateSyncConfig(syncForm.value)
    ElMessage.success('配置已保存')
    loadConfig()
  } finally {
    savingCfg.value = false
  }
}

const handleSync = async () => {
  syncing.value = true
  progress.value = { running: true, status: 'running', message: '正在同步...' }
  try {
    const res = await syncUserFields()
    ElMessage.success(`同步完成：新增 ${res?.added || 0}，更新 ${res?.updated || 0}，共 ${res?.total || 0}`)
    progress.value = {
      running: false,
      status: 'success',
      message: `同步完成：新增 ${res?.added || 0}，更新 ${res?.updated || 0}，共 ${res?.total || 0}`
    }
    loadFields()
    loadConfig()
  } catch (e) {
    progress.value = { running: false, status: 'failed', message: '同步失败' }
  } finally {
    syncing.value = false
  }
}

const openCreate = () => {
  form.value = emptyForm()
  formIsEdit.value = false
  formVisible.value = true
}

const openEdit = (row) => {
  form.value = { ...row }
  formIsEdit.value = true
  formVisible.value = true
}

const handleSave = async () => {
  if (!form.value.fieldKey || !form.value.label) {
    ElMessage.warning('请填写字段键与显示名')
    return
  }
  saving.value = true
  try {
    if (formIsEdit.value) {
      await updateUserField(form.value.id, form.value)
      ElMessage.success('更新成功')
    } else {
      await createUserField(form.value)
      ElMessage.success('新增成功')
    }
    formVisible.value = false
    loadFields()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除字段「${row.label}（${row.fieldKey}）」吗？`, '提示', { type: 'warning' })
  } catch (e) {
    return
  }
  await deleteUserField(row.id)
  ElMessage.success('已删除')
  loadFields()
}

onMounted(() => {
  loadFields()
  loadConfig()
  loadProgress()
  // 轮询进度（10s），同步运行时可及时刷新
  pollTimer = setInterval(loadProgress, 10000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.user-fields-container {
  width: 100%;
}
.section-card {
  margin-bottom: 16px;
}
.section-card:last-child {
  margin-bottom: 0;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
}
.title .el-icon {
  color: var(--primary-color, #409eff);
}
.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.sync-form {
  margin-bottom: 4px;
}
.sync-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.last-run {
  color: var(--text-secondary);
  font-size: 13px;
}
.progress-box {
  margin-top: 14px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.progress-box .el-progress {
  flex: 1;
}
.progress-text {
  color: var(--text-secondary);
  font-size: 13px;
}
.tag-gap {
  margin-left: 6px;
}
.key {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  background: var(--panel-bg, #f5f7fa);
  padding: 1px 6px;
  border-radius: 4px;
}
.form-tip {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.6;
}
</style>
