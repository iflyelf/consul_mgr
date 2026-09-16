<template>
  <div class="teams-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>团队管理</span>
          <div>
            <el-input
              v-model="keyword"
              placeholder="搜索团队"
              clearable
              style="width: 220px; margin-right: 10px"
              @keyup.enter="loadData"
              @clear="loadData"
            />
            <el-button type="primary" @click="handleAdd">
              <el-icon><Plus /></el-icon>
              新建团队
            </el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="团队是权限分配的主体：为团队添加成员并授权服务组，成员即可访问对应服务组"
        style="margin-bottom: 16px"
      />

      <el-table :data="teams" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="团队名称" min-width="160" />
        <el-table-column prop="code" label="代码" min-width="130" />
        <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column label="成员数" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.member_count }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="授权服务组" width="120" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="success">{{ row.group_count }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="openMembers(row)">成员</el-button>
            <el-button type="success" size="small" @click="openPermissions(row)">服务组授权</el-button>
            <el-button link type="warning" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除该团队吗？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button link type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新建/编辑团队 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px">
      <el-form :model="form" ref="formRef" :rules="rules" label-width="90px">
        <el-form-item label="团队名称" prop="name">
          <el-input v-model="form.name" placeholder="如：运维一组" />
        </el-form-item>
        <el-form-item label="代码">
          <el-input v-model="form.code" placeholder="如：ops1（留空则同名称）" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="状态" v-if="isEdit">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 成员管理 -->
    <el-dialog v-model="memberVisible" :title="`团队成员 - ${currentTeam?.name || ''}`" width="640px">
      <div class="add-row">
        <el-select
          v-model="selectedUser"
          filterable
          placeholder="选择用户（来自 Casdoor）"
          style="flex: 1"
        >
          <el-option
            v-for="u in availableUsers"
            :key="u.id"
            :label="`${u.displayName || u.name} (${u.name})`"
            :value="u.id"
          >
            <span>{{ u.displayName || u.name }}</span>
            <span class="opt-sub">{{ u.name }}</span>
          </el-option>
        </el-select>
        <el-button type="primary" style="margin-left: 10px" @click="handleAddMember">添加成员</el-button>
      </div>

      <el-table :data="members" v-loading="memberLoading" stripe border style="margin-top: 14px">
        <el-table-column label="用户" min-width="180">
          <template #default="{ row }">{{ row.displayName || row.username }}</template>
        </el-table-column>
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="handleRemoveMember(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 服务组授权 -->
    <el-dialog v-model="permVisible" :title="`服务组授权 - ${currentTeam?.name || ''}`" width="760px">
      <div class="add-row">
        <el-select v-model="permForm.group_id" filterable placeholder="选择服务组" style="width: 240px">
          <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
        </el-select>
        <el-select v-model="permForm.role_ids" multiple placeholder="引用角色（可选）" style="width: 220px; margin-left: 10px">
          <el-option v-for="r in roles" :key="r.id" :label="r.name" :value="r.id" />
        </el-select>
        <el-button type="primary" style="margin-left: 10px" @click="handleGrant">授权 / 更新</el-button>
      </div>
      <div class="add-row" style="margin-top: 10px">
        <span class="label">直接权限：</span>
        <el-checkbox-group v-model="permForm.permissions">
          <el-checkbox label="read">只读</el-checkbox>
          <el-checkbox label="write">读写</el-checkbox>
          <el-checkbox label="delete">删除</el-checkbox>
        </el-checkbox-group>
      </div>

      <el-table :data="permissions" v-loading="permLoading" stripe border style="margin-top: 14px">
        <el-table-column prop="group_name" label="服务组" min-width="150" />
        <el-table-column label="直接权限" min-width="170">
          <template #default="{ row }">
            <el-tag v-for="p in row.permissions" :key="p" size="small" style="margin-right: 6px">{{ p }}</el-tag>
            <span v-if="!row.permissions || row.permissions.length === 0" class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="引用角色" min-width="170">
          <template #default="{ row }">
            <el-tag v-for="rid in row.role_ids" :key="rid" size="small" type="warning" style="margin-right: 6px">
              {{ roleName(rid) }}
            </el-tag>
            <span v-if="!row.role_ids || row.role_ids.length === 0" class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="handleRevoke(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  getTeams, createTeam, updateTeam, deleteTeam,
  getTeamMembers, addTeamMember, removeTeamMember,
  getTeamPermissions, grantTeamPermission, revokeTeamPermission,
  getUsers, getRoles
} from '@/api/org'
import { getGroups } from '@/api/group'

const loading = ref(false)
const submitLoading = ref(false)
const keyword = ref('')
const teams = ref([])
const groups = ref([])
const roles = ref([])
const allUsers = ref([])

const dialogVisible = ref(false)
const dialogTitle = ref('新建团队')
const isEdit = ref(false)
const formRef = ref(null)
const form = reactive({ id: null, name: '', code: '', description: '', status: 1 })
const rules = { name: [{ required: true, message: '请输入团队名称', trigger: 'blur' }] }

// 成员
const memberVisible = ref(false)
const memberLoading = ref(false)
const members = ref([])
const selectedUser = ref('')
const currentTeam = ref(null)

// 授权
const permVisible = ref(false)
const permLoading = ref(false)
const permissions = ref([])
const permForm = reactive({ group_id: null, permissions: [], role_ids: [] })

const availableUsers = computed(() =>
  allUsers.value.filter(u => !members.value.some(m => m.user_id === u.id))
)

const roleName = (id) => roles.value.find(r => r.id === id)?.name || `#${id}`

const loadData = async () => {
  loading.value = true
  try {
    const res = await getTeams({ keyword: keyword.value })
    teams.value = res.list || []
  } catch (e) {
    ElMessage.error('获取团队列表失败')
  } finally {
    loading.value = false
  }
}

const loadRefData = async () => {
  try {
    const [g, r, u] = await Promise.all([getGroups({ page: 1, page_size: 100 }), getRoles({}), getUsers({})])
    groups.value = g.list || []
    roles.value = r.list || []
    allUsers.value = u.list || []
  } catch (e) { /* 忽略，页面仍可用 */ }
}

const resetForm = () => {
  Object.assign(form, { id: null, name: '', code: '', description: '', status: 1 })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  isEdit.value = false
  dialogTitle.value = '新建团队'
  dialogVisible.value = true
}

const handleEdit = (row) => {
  resetForm()
  Object.assign(form, { id: row.id, name: row.name, code: row.code, description: row.description, status: row.status })
  isEdit.value = true
  dialogTitle.value = '编辑团队'
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    if (isEdit.value) {
      await updateTeam(form.id, { name: form.name, description: form.description, status: form.status })
      ElMessage.success('更新成功')
    } else {
      await createTeam({ name: form.name, code: form.code, description: form.description })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '提交失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await deleteTeam(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '删除失败')
  }
}

// ---------- 成员 ----------
const openMembers = async (row) => {
  currentTeam.value = row
  memberVisible.value = true
  selectedUser.value = ''
  await loadMembers(row.id)
}

const loadMembers = async (id) => {
  memberLoading.value = true
  try {
    const res = await getTeamMembers(id)
    members.value = res.list || []
  } catch (e) {
    ElMessage.error('获取成员失败')
  } finally {
    memberLoading.value = false
  }
}

const handleAddMember = async () => {
  if (!selectedUser.value) {
    ElMessage.warning('请选择用户')
    return
  }
  const u = allUsers.value.find(x => x.id === selectedUser.value)
  try {
    await addTeamMember(currentTeam.value.id, {
      user_id: u.id, username: u.name, display_name: u.displayName || u.name
    })
    ElMessage.success('添加成功')
    selectedUser.value = ''
    await loadMembers(currentTeam.value.id)
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '添加失败')
  }
}

const handleRemoveMember = async (row) => {
  try {
    await removeTeamMember(currentTeam.value.id, row.user_id)
    ElMessage.success('移除成功')
    await loadMembers(currentTeam.value.id)
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '移除失败')
  }
}

// ---------- 服务组授权 ----------
const openPermissions = async (row) => {
  currentTeam.value = row
  permVisible.value = true
  Object.assign(permForm, { group_id: null, permissions: [], role_ids: [] })
  await loadPermissions(row.id)
}

const loadPermissions = async (id) => {
  permLoading.value = true
  try {
    const res = await getTeamPermissions(id)
    permissions.value = res.list || []
  } catch (e) {
    ElMessage.error('获取授权失败')
  } finally {
    permLoading.value = false
  }
}

const handleGrant = async () => {
  if (!permForm.group_id) {
    ElMessage.warning('请选择服务组')
    return
  }
  if (permForm.permissions.length === 0 && permForm.role_ids.length === 0) {
    ElMessage.warning('请至少选择直接权限或引用角色')
    return
  }
  try {
    await grantTeamPermission(currentTeam.value.id, {
      group_id: permForm.group_id,
      permissions: permForm.permissions,
      role_ids: permForm.role_ids
    })
    ElMessage.success('授权成功')
    Object.assign(permForm, { group_id: null, permissions: [], role_ids: [] })
    await loadPermissions(currentTeam.value.id)
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '授权失败')
  }
}

const handleRevoke = async (row) => {
  try {
    await revokeTeamPermission(currentTeam.value.id, row.group_id)
    ElMessage.success('撤销成功')
    await loadPermissions(currentTeam.value.id)
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '撤销失败')
  }
}

onMounted(() => {
  loadData()
  loadRefData()
})
</script>

<style scoped>
.teams-container { width: 100%; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.add-row { display: flex; align-items: center; gap: 4px; }
.label { color: var(--text-secondary); font-size: 13px; margin-right: 8px; }
.text-muted { color: var(--text-secondary); }
.opt-sub { float: right; color: var(--text-secondary); font-size: 12px; }
</style>
