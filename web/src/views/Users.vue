<template>
  <div class="users-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>用户管理</span>
            <el-tag type="success" size="small">数据来源：Casdoor</el-tag>
          </div>
          <div class="header-actions">
            <el-input
              v-model="keyword"
              placeholder="搜索用户名 / 姓名 / 邮箱"
              clearable
              style="width: 240px"
              @keyup.enter="handleSearch"
              @clear="handleSearch"
            />
            <el-button type="primary" @click="handleSearch" :loading="loading">搜索</el-button>
            <el-button type="primary" :icon="Plus" @click="openCreate">新增用户</el-button>
            <el-button :icon="Refresh" @click="handleRefresh" :loading="loading">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :data="users" v-loading="loading" stripe border>
        <el-table-column label="用户" min-width="220">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="34" :src="row.avatar || defaultAvatar(row)">
                {{ (row.displayName || row.name || '?').slice(0, 1) }}
              </el-avatar>
              <div>
                <div class="user-name">
                  {{ row.displayName || row.name }}
                  <el-tag v-if="row.isAdmin" type="danger" size="small">管理员</el-tag>
                  <el-tag v-if="row.isForbidden" type="warning" size="small">禁用</el-tag>
                </div>
                <div class="user-sub">{{ row.name }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱" min-width="180" show-overflow-tooltip />
        <el-table-column prop="phone" label="电话" width="130" />
        <el-table-column label="工号" width="110" show-overflow-tooltip>
          <template #default="{ row }">{{ prop(row, 'empCode') || '-' }}</template>
        </el-table-column>
        <el-table-column label="一级部门" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ prop(row, 'deptNameLv1') || '-' }}</template>
        </el-table-column>
        <el-table-column label="二级部门" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ prop(row, 'deptNameLv2') || '-' }}</template>
        </el-table-column>
        <el-table-column prop="owner" label="所属组织" width="120" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleView(row)">查看</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-tooltip content="更多操作" placement="top">
              <el-dropdown trigger="click" @command="(cmd) => handleMore(cmd, row)">
                <el-button link type="primary" size="small" :icon="MoreFilled" />
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="reset" :icon="Key">重置密码</el-dropdown-item>
                    <el-dropdown-item command="customPwd" :icon="EditPen">自定义密码</el-dropdown-item>
                    <el-dropdown-item command="admin" :icon="UserFilled">
                      {{ row.isAdmin ? '取消管理员' : '设为管理员' }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </el-tooltip>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 查看详情 -->
    <el-dialog v-model="detailVisible" title="用户详情" width="620px">
      <div class="detail-head" v-if="currentUser">
        <el-avatar :size="56" :src="currentUser.avatar || defaultAvatar(currentUser)">
          {{ (currentUser.displayName || currentUser.name || '?').slice(0, 1) }}
        </el-avatar>
        <div>
          <div class="detail-head-name">
            {{ currentUser.displayName || currentUser.name }}
            <el-tag v-if="currentUser.isAdmin" type="danger" size="small">管理员</el-tag>
            <el-tag v-if="currentUser.isForbidden" type="warning" size="small">禁用</el-tag>
          </div>
          <div class="user-sub">{{ currentUser.name }}</div>
        </div>
      </div>
      <el-descriptions :column="2" border v-if="currentUser" style="margin-top: 16px">
        <el-descriptions-item label="账号">{{ currentUser.name }}</el-descriptions-item>
        <el-descriptions-item label="所属组织">{{ currentUser.owner || '-' }}</el-descriptions-item>
        <el-descriptions-item label="邮箱">{{ currentUser.email || '-' }}</el-descriptions-item>
        <el-descriptions-item label="电话">{{ currentUser.phone || '-' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间" :span="2">{{ currentUser.createdTime || '-' }}</el-descriptions-item>
        <el-descriptions-item
          v-for="p in propertyEntries(currentUser)"
          :key="p.key"
          :label="propLabel(p.key)"
        >
          {{ p.value || '-' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 新增/编辑 -->
    <el-dialog v-model="formVisible" :title="formIsEdit ? '编辑用户' : '新增用户'" width="620px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="域账号" required>
          <el-input v-model="form.name" :disabled="formIsEdit" placeholder="唯一标识，如 zhangsan" />
        </el-form-item>
        <el-form-item label="姓名" required>
          <el-input v-model="form.displayName" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item v-for="f in hrFields" :key="f.key" :label="f.label">
          <el-input v-model="form.properties[f.key]" />
        </el-form-item>
        <el-form-item v-if="!formIsEdit" label="初始密码">
          <el-input v-model="form.password" type="password" show-password placeholder="留空使用系统默认密码" />
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
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, MoreFilled, Key, EditPen, UserFilled } from '@element-plus/icons-vue'
import {
  getUsers,
  createUser,
  updateUser,
  deleteUser,
  resetUserPassword,
  setUserAdmin
} from '@/api/org'

const loading = ref(false)
const saving = ref(false)
const keyword = ref('')
const users = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)

const detailVisible = ref(false)
const currentUser = ref(null)
const formVisible = ref(false)
const formIsEdit = ref(false)

// 人事字段（与 FlyIAM 内置字段键一致，存储于 Casdoor Properties）
const hrFields = [
  { key: 'empCode', label: '工号' },
  { key: 'deptNameLv0', label: '零级部门' },
  { key: 'deptNameLv1', label: '一级部门' },
  { key: 'deptNameLv2', label: '二级部门' },
  { key: 'compileType', label: '编制类型' },
  { key: 'superior', label: '上级账号' }
]

// 属性显示名（未知键直接显示键名）
const propLabels = {
  empCode: '工号',
  deptNameLv0: '零级部门',
  deptNameLv1: '一级部门',
  deptNameLv2: '二级部门',
  deptIdLv0: '零级部门ID',
  deptIdLv1: '一级部门ID',
  deptIdLv2: '二级部门ID',
  compileType: '编制类型',
  superior: '上级账号',
  source: '来源'
}
const propLabel = (key) => propLabels[key] || key
const prop = (row, key) => row?.properties?.[key] || ''
const propertyEntries = (row) =>
  Object.keys(row?.properties || {})
    .filter((k) => row.properties[k] !== '' && row.properties[k] != null)
    .map((k) => ({ key: k, value: row.properties[k] }))

const emptyForm = () => ({
  name: '',
  displayName: '',
  email: '',
  phone: '',
  password: '',
  properties: {}
})
const form = ref(emptyForm())

// 无头像时用姓名首字生成占位（数据 URI）
const defaultAvatar = (row) => {
  const ch = (row?.displayName || row?.name || '?').trim().slice(0, 1).toUpperCase()
  const colors = ['#c8865a', '#2f6fed', '#83a76f', '#d9a94c', '#d9776c', '#8e7cc3']
  let hash = 0
  const key = row?.name || ch
  for (let i = 0; i < key.length; i++) hash = (hash * 31 + key.charCodeAt(i)) % colors.length
  const bg = colors[hash]
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80"><rect width="80" height="80" rx="40" fill="${bg}"/><text x="50%" y="54%" font-size="34" fill="#fff" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif">${ch}</text></svg>`
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await getUsers({
      keyword: keyword.value,
      page: currentPage.value,
      page_size: pageSize.value
    })
    users.value = res.list || []
    total.value = res.total || 0
  } catch (e) {
    // 错误提示由请求拦截器统一处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadData()
}
const handleSizeChange = (val) => {
  pageSize.value = val
  currentPage.value = 1
  loadData()
}
const handleCurrentChange = (val) => {
  currentPage.value = val
  loadData()
}
const handleRefresh = () => loadData()

const handleView = (row) => {
  currentUser.value = row
  detailVisible.value = true
}

const openCreate = () => {
  form.value = emptyForm()
  formIsEdit.value = false
  formVisible.value = true
}

const openEdit = (row) => {
  form.value = {
    ...emptyForm(),
    name: row.name,
    displayName: row.displayName,
    email: row.email,
    phone: row.phone,
    properties: { ...(row.properties || {}) }
  }
  formIsEdit.value = true
  formVisible.value = true
}

const handleSave = async () => {
  if (!form.value.name || !form.value.displayName) {
    ElMessage.warning('请填写域账号与姓名')
    return
  }
  saving.value = true
  try {
    if (formIsEdit.value) {
      await updateUser(form.value.name, form.value)
      ElMessage.success('更新成功')
    } else {
      await createUser(form.value)
      ElMessage.success('新增成功')
    }
    formVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}

const handleMore = async (cmd, row) => {
  try {
    if (cmd === 'reset') {
      await ElMessageBox.confirm(`确定将「${row.displayName || row.name}」的密码重置为系统默认密码吗？`, '提示', { type: 'warning' })
      const res = await resetUserPassword(row.name)
      ElMessage.success(`已重置为：${res?.password || '默认密码'}`)
    } else if (cmd === 'customPwd') {
      const { value } = await ElMessageBox.prompt(`为「${row.displayName || row.name}」设置新密码`, '自定义密码', {
        inputType: 'password',
        inputPlaceholder: '请输入新密码',
        inputValidator: (v) => (v && v.length >= 6 ? true : '密码至少 6 位')
      })
      await resetUserPassword(row.name, value)
      ElMessage.success('密码已设置')
    } else if (cmd === 'admin') {
      const next = !row.isAdmin
      await ElMessageBox.confirm(
        `确定${next ? '将' : '取消'}「${row.displayName || row.name}」的管理员权限吗？`,
        '提示',
        { type: 'warning' }
      )
      await setUserAdmin(row.name, next)
      ElMessage.success('已更新')
      loadData()
    }
  } catch (e) {
    // 取消或失败，忽略
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除用户「${row.displayName || row.name}（${row.name}）」吗？`, '提示', { type: 'warning' })
  } catch (e) {
    return
  }
  await deleteUser(row.name)
  ElMessage.success('已删除')
  loadData()
}

onMounted(loadData)
</script>

<style scoped>
.users-container { width: 100%; }
.card-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; }
.title { display: flex; align-items: center; gap: 10px; font-weight: 600; }
.header-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.user-cell { display: flex; align-items: center; gap: 10px; }
.user-name { display: flex; align-items: center; gap: 6px; font-weight: 500; }
.user-sub { font-size: 12px; color: var(--text-secondary); }
.detail-head { display: flex; align-items: center; gap: 14px; }
.detail-head-name { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; }
.pagination { margin-top: 16px; text-align: right; }
</style>
