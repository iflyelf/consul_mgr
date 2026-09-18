<template>
  <div class="settings-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>系统设置</span>
            <el-tag type="info" size="small">保存后即时生效（Casdoor 连接类需重启）</el-tag>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="load">刷新</el-button>
            <el-button type="primary" :loading="saving" @click="save">保存配置</el-button>
          </div>
        </div>
      </template>

      <el-collapse v-model="activeGroups">
        <el-collapse-item v-for="(items, group) in grouped" :key="group" :name="group">
          <template #title><span class="group-title">{{ group }}</span></template>
          <el-form label-width="220px">
            <el-form-item v-for="it in items" :key="it.key" :label="it.label">
              <el-switch v-if="it.type === 'bool'" v-model="form[it.key]" />
              <el-input-number v-else-if="it.type === 'int'" v-model="form[it.key]" :controls="false" style="width: 220px" />
              <el-input
                v-else
                v-model="form[it.key]"
                :type="it.secret ? 'password' : 'text'"
                :show-password="it.secret"
                :placeholder="it.secret ? '留空/保持占位符表示不修改' : ''"
                style="max-width: 460px"
              />
              <span class="key-tip">{{ it.key }}</span>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { listSettings, updateSettings } from '@/api/setting'

const saving = ref(false)
const items = ref([])
const form = ref({})
const activeGroups = ref([])

const grouped = computed(() => {
  const out = {}
  for (const it of items.value) {
    if (!out[it.group]) out[it.group] = []
    out[it.group].push(it)
  }
  if (activeGroups.value.length === 0 && Object.keys(out).length > 0) {
    activeGroups.value = [Object.keys(out)[0]]
  }
  return out
})

const load = async () => {
  try {
    const res = await listSettings()
    items.value = res || []
    const f = {}
    for (const it of items.value) {
      f[it.key] = it.type === 'bool' ? it.value === 'true' : it.value
    }
    form.value = f
  } catch (e) {
    /* 拦截器已提示 */
  }
}

const save = async () => {
  saving.value = true
  try {
    const payload = {}
    for (const it of items.value) {
      const v = form.value[it.key]
      payload[it.key] = it.type === 'bool' ? String(!!v) : String(v ?? '')
    }
    await updateSettings(payload)
    ElMessage.success('配置已保存并生效')
    load()
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.settings-container { width: 100%; }
.card-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; }
.title { display: flex; align-items: center; gap: 10px; font-weight: 600; }
.header-actions { display: flex; gap: 8px; }
.group-title { font-weight: 600; }
.key-tip { margin-left: 12px; color: var(--text-secondary); font-size: 12px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>
