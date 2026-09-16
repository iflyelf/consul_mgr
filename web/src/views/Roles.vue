<template>
  <div class="roles-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>角色管理</span>
          <div>
            <el-input
              v-model="keyword"
              placeholder="搜索角色"
              clearable
              style="width: 220px; margin-right: 10px"
              @keyup.enter="loadData"
              @clear="loadData"
            />
            <el-button type="primary" @click="handleAdd">
              <el-icon><Plus /></el-icon>
              新建角色
            </el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="角色 = 可复用的权限集合（read/write/delete），可被团队引用"
        style="margin-bottom: 16px"
      />

      <el-table :data="roles" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="角色名称" min-width="150" />
        <el-table-column prop="code" label="代码" min-width="130" />
        <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column label="权限" min-width="220">
          <template #default="{ row }">
            <el-tag
              v-for="p in row.permissions"
              :key="p"
              size="small"
              style="margin-right: 6px"
              :type="permType(p)"
            >{{ permLabel(p) }}</el-tag>
            <span v-if="!row.permissions || row.permissions.length === 0" class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除该角色吗？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="520px">
      <el-form :model="form" ref="formRef" :rules="rules" label-width="90px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="form.name" placeholder="如：运维人员" />
        </el-form-item>
        <el-form-item label="代码">
          <el-input v-model="form.code" placeholder="如：operator（留空则同名称）" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="用途说明" />
        </el-form-item>
        <el-form-item label="权限">
          <el-checkbox-group v-model="form.permissions">
            <el-checkbox label="read">只读 (read)</el-checkbox>
            <el-checkbox label="write">读写 (write)</el-checkbox>
            <el-checkbox label="delete">删除 (delete)</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getRoles, createRole, updateRole, deleteRole } from '@/api/org'

const loading = ref(false)
const submitLoading = ref(false)
const keyword = ref('')
const roles = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('新建角色')
const isEdit = ref(false)
const formRef = ref(null)

const form = reactive({ id: null, name: '', code: '', description: '', permissions: [] })
const rules = { name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }] }

const permLabel = (p) => ({ read: '只读', write: '读写', delete: '删除', '*': '全部' }[p] || p)
const permType = (p) => ({ read: 'info', write: 'warning', delete: 'danger', '*': 'danger' }[p] || 'info')

const loadData = async () => {
  loading.value = true
  try {
    const res = await getRoles({ keyword: keyword.value })
    roles.value = res.list || []
  } catch (e) {
    ElMessage.error('获取角色列表失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(form, { id: null, name: '', code: '', description: '', permissions: [] })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  isEdit.value = false
  dialogTitle.value = '新建角色'
  dialogVisible.value = true
}

const handleEdit = (row) => {
  resetForm()
  Object.assign(form, {
    id: row.id, name: row.name, code: row.code,
    description: row.description, permissions: [...(row.permissions || [])]
  })
  isEdit.value = true
  dialogTitle.value = '编辑角色'
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    if (isEdit.value) {
      await updateRole(form.id, {
        name: form.name, description: form.description, permissions: form.permissions
      })
      ElMessage.success('更新成功')
    } else {
      await createRole({
        name: form.name, code: form.code, description: form.description, permissions: form.permissions
      })
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
    await deleteRole(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || '删除失败')
  }
}

onMounted(loadData)
</script>

<style scoped>
.roles-container { width: 100%; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.text-muted { color: var(--text-secondary); }
</style>
