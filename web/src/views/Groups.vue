<template>
  <div class="page-container">
    <el-card class="card-container">
      <template #header>
        <div class="card-header">
          <span>服务组管理</span>
          <el-button type="primary" :icon="Plus" @click="handleAdd">添加服务组</el-button>
        </div>
      </template>
      
      <el-table
        v-loading="loading"
        :data="tableData"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="code" label="代码" min-width="100" />
        <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip />
        <el-table-column prop="consul_address" label="Consul 地址" min-width="180" show-overflow-tooltip />
        <el-table-column prop="consul_datacenter" label="数据中心" width="100" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="success" size="small" @click="handleTest(row)">测试</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </el-card>
    
    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入服务组名称" />
        </el-form-item>
        
        <el-form-item label="代码" prop="code">
          <el-input v-model="formData.code" placeholder="请输入服务组代码" :disabled="isEdit" />
        </el-form-item>
        
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入描述"
          />
        </el-form-item>
        
        <el-form-item label="Consul 地址" prop="consul_address">
          <el-input v-model="formData.consul_address" placeholder="http://consul.example.com:8500" />
        </el-form-item>
        
        <el-form-item label="Consul Token">
          <el-input v-model="formData.consul_token" placeholder="可选" show-password />
        </el-form-item>
        
        <el-form-item label="数据中心">
          <el-input v-model="formData.consul_datacenter" placeholder="默认 dc1" />
        </el-form-item>
        
        <el-form-item label="状态" v-if="isEdit">
          <el-switch
            v-model="formData.status"
            :active-value="1"
            :inactive-value="0"
          />
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getGroupList, createGroup, updateGroup, deleteGroup, testConnection } from '@/api/group'

const loading = ref(false)
const tableData = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const dialogVisible = ref(false)
const dialogTitle = ref('添加服务组')
const isEdit = ref(false)
const submitLoading = ref(false)
const formRef = ref(null)

const formData = reactive({
  id: null,
  name: '',
  code: '',
  description: '',
  consul_address: '',
  consul_token: '',
  consul_datacenter: '',
  status: 1
})

const rules = {
  name: [{ required: true, message: '请输入服务组名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入服务组代码', trigger: 'blur' }],
  consul_address: [
    { required: true, message: '请输入 Consul 地址', trigger: 'blur' },
    { type: 'url', message: '请输入正确的 URL 格式', trigger: 'blur' }
  ]
}

const loadData = async () => {
  loading.value = true
  try {
    const data = await getGroupList({
      page: page.value,
      page_size: pageSize.value
    })
    tableData.value = data.list || []
    total.value = data.total || 0
  } catch (error) {
    console.error('加载数据失败:', error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(formData, {
    id: null,
    name: '',
    code: '',
    description: '',
    consul_address: '',
    consul_token: '',
    consul_datacenter: '',
    status: 1
  })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  isEdit.value = false
  dialogTitle.value = '添加服务组'
  dialogVisible.value = true
}

const handleEdit = (row) => {
  resetForm()
  Object.assign(formData, {
    id: row.id,
    name: row.name,
    code: row.code,
    description: row.description,
    consul_address: row.consul_address,
    consul_token: row.consul_token || '',
    consul_datacenter: row.consul_datacenter || '',
    status: row.status
  })
  isEdit.value = true
  dialogTitle.value = '编辑服务组'
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    submitLoading.value = true
    try {
      const data = {
        name: formData.name,
        code: formData.code,
        description: formData.description,
        consul_address: formData.consul_address,
        consul_token: formData.consul_token || undefined,
        consul_datacenter: formData.consul_datacenter || undefined
      }
      
      if (isEdit.value) {
        // 编辑时只发送修改的字段
        const updateData = {
          description: formData.description
        }
        if (formData.name !== '') updateData.name = formData.name
        if (formData.consul_address !== '') updateData.consul_address = formData.consul_address
        if (formData.consul_token !== '') updateData.consul_token = formData.consul_token
        if (formData.consul_datacenter !== '') updateData.consul_datacenter = formData.consul_datacenter
        updateData.status = formData.status
        
        await updateGroup(formData.id, updateData)
        ElMessage.success('更新成功')
      } else {
        await createGroup(data)
        ElMessage.success('创建成功')
      }
      
      dialogVisible.value = false
      loadData()
    } catch (error) {
      console.error('提交失败:', error)
    } finally {
      submitLoading.value = false
    }
  })
}

const handleTest = async (row) => {
  try {
    await testConnection(row.id)
    ElMessage.success('连接测试成功')
  } catch (error) {
    console.error('测试失败:', error)
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定要删除服务组 "${row.name}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await deleteGroup(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (error) {
    // 取消或错误
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 768px) {
  :deep(.el-table) {
    font-size: 12px;
  }
  
  :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
  }
}
</style>
