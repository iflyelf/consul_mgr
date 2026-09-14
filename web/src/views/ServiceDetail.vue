<template>
  <div class="service-detail-container">
    <el-page-header @back="handleBack" :content="`服务详情: ${serviceName}`" />

    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>基本信息</span>
          <el-button type="primary" @click="handleRefresh" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <el-descriptions :column="2" border v-if="serviceDetail">
        <el-descriptions-item label="服务名称">{{ serviceDetail.name }}</el-descriptions-item>
        <el-descriptions-item label="实例数量">{{ serviceDetail.instances?.length || 0 }}</el-descriptions-item>
        <el-descriptions-item label="Tags" :span="2">
          <el-tag v-for="tag in serviceDetail.tags" :key="tag" size="small" style="margin-right: 5px">
            {{ tag }}
          </el-tag>
          <span v-if="!serviceDetail.tags || serviceDetail.tags.length === 0" class="text-muted">-</span>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>实例列表</span>
          <div>
            <el-button type="primary" @click="handleRegisterInstance">
              <el-icon><Plus /></el-icon>
              注册实例
            </el-button>
            <el-button type="success" @click="handleExport">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
            <el-button type="warning" @click="handleImport">
              <el-icon><Upload /></el-icon>
              导入
            </el-button>
          </div>
        </div>
      </template>

      <!-- 实例表格 -->
      <el-table
        :data="instances"
        v-loading="loading"
        stripe
        border
      >
        <el-table-column prop="id" label="实例ID" min-width="200" />
        <el-table-column label="地址" min-width="150">
          <template #default="{ row }">
            {{ row.address }}:{{ row.port }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Tags" min-width="200">
          <template #default="{ row }">
            <el-tag v-for="tag in row.tags" :key="tag" size="small" style="margin-right: 5px">
              {{ tag }}
            </el-tag>
            <span v-if="!row.tags || row.tags.length === 0" class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="Meta" min-width="200">
          <template #default="{ row }">
            <el-popover placement="left" :width="400" trigger="hover" v-if="row.meta && Object.keys(row.meta).length > 0">
              <template #reference>
                <el-link type="primary">{{ Object.keys(row.meta).length }} 个</el-link>
              </template>
              <div>
                <div v-for="(value, key) in row.meta" :key="key" class="meta-item">
                  <strong>{{ key }}:</strong> {{ value }}
                </div>
              </div>
            </el-popover>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="健康检查" width="100" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.checks && row.checks.length > 0" color="#67C23A"><CircleCheck /></el-icon>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确定要删除这个实例吗？"
              @confirm="handleDeleteInstance(row)"
            >
              <template #reference>
                <el-button type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && instances.length === 0" description="暂无实例" />
    </el-card>

    <!-- 注册/编辑实例对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
    >
      <el-form :model="instanceForm" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="实例ID" prop="id" v-if="!isEdit">
          <el-input v-model="instanceForm.id" placeholder="唯一标识" />
        </el-form-item>
        <el-form-item label="服务名称" prop="service">
          <el-input v-model="instanceForm.service" disabled />
        </el-form-item>
        <el-form-item label="地址" prop="address">
          <el-input v-model="instanceForm.address" placeholder="IP 或域名" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="instanceForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="Tags">
          <el-select
            v-model="instanceForm.tags"
            multiple
            filterable
            allow-create
            placeholder="添加标签"
            style="width: 100%"
          >
            <el-option
              v-for="tag in commonTags"
              :key="tag"
              :label="tag"
              :value="tag"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Meta">
          <div v-for="(meta, index) in instanceForm.metaList" :key="index" class="meta-input">
            <el-input v-model="meta.key" placeholder="Key" style="width: 40%" />
            <el-input v-model="meta.value" placeholder="Value" style="width: 40%; margin-left: 10px" />
            <el-button type="danger" size="small" @click="removeMetaItem(index)" style="margin-left: 10px">删除</el-button>
          </div>
          <el-button type="primary" size="small" @click="addMetaItem">添加 Meta</el-button>
        </el-form-item>
        <el-form-item label="健康检查">
          <el-checkbox v-model="instanceForm.enableHealthCheck">启用健康检查</el-checkbox>
        </el-form-item>
        <template v-if="instanceForm.enableHealthCheck">
          <el-form-item label="检查类型" prop="checkType">
            <el-radio-group v-model="instanceForm.checkType">
              <el-radio label="http">HTTP</el-radio>
              <el-radio label="tcp">TCP</el-radio>
              <el-radio label="ttl">TTL</el-radio>
              <el-radio label="grpc">GRPC</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="检查间隔" prop="interval" v-if="instanceForm.checkType !== 'ttl'">
            <el-input v-model="instanceForm.interval" placeholder="10s" />
          </el-form-item>
          <el-form-item label="超时时间" prop="timeout" v-if="instanceForm.checkType !== 'ttl'">
            <el-input v-model="instanceForm.timeout" placeholder="5s" />
          </el-form-item>
          <el-form-item label="检查URL" prop="http" v-if="instanceForm.checkType === 'http'">
            <el-input v-model="instanceForm.http" placeholder="http://localhost:8080/health" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>

    <!-- 导出对话框 -->
    <el-dialog v-model="exportDialogVisible" title="导出实例" width="400px">
      <el-form>
        <el-form-item label="导出格式">
          <el-radio-group v-model="exportFormat">
            <el-radio label="json">JSON</el-radio>
            <el-radio label="yaml">YAML</el-radio>
            <el-radio label="csv">CSV</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="exportDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleConfirmExport" :loading="exportLoading">导出</el-button>
      </template>
    </el-dialog>

    <!-- 导入对话框 -->
    <el-dialog v-model="importDialogVisible" title="导入实例" width="400px">
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :limit="1"
        :on-change="handleFileChange"
        drag
      >
        <el-icon class="el-icon--upload"><upload-filled /></el-icon>
        <div class="el-upload__text">
          拖拽文件到此处或 <em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            支持 JSON/YAML/CSV 格式
          </div>
        </template>
      </el-upload>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleConfirmImport" :loading="importLoading">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Refresh, Plus, Download, Upload, CircleCheck, UploadFilled } from '@element-plus/icons-vue'
import { getServiceDetail } from '@/api/service'
import { 
  registerInstance, 
  updateInstance, 
  deleteInstance,
  exportInstances,
  importInstances 
} from '@/api/instance'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const submitLoading = ref(false)
const exportLoading = ref(false)
const importLoading = ref(false)

const groupId = ref(route.query.group_id)
const serviceName = ref(route.query.service)

const serviceDetail = ref(null)
const instances = computed(() => serviceDetail.value?.instances || [])

const dialogVisible = ref(false)
const dialogTitle = ref('注册实例')
const isEdit = ref(false)

const exportDialogVisible = ref(false)
const exportFormat = ref('json')

const importDialogVisible = ref(false)
const uploadFile = ref(null)

const commonTags = ref(['production', 'staging', 'development', 'v1', 'v2'])

const instanceForm = reactive({
  id: '',
  service: '',
  address: '',
  port: 8080,
  tags: [],
  metaList: [],
  enableHealthCheck: false,
  checkType: 'http',
  interval: '10s',
  timeout: '5s',
  http: ''
})

const rules = {
  id: [{ required: true, message: '请输入实例ID', trigger: 'blur' }],
  service: [{ required: true, message: '请输入服务名称', trigger: 'blur' }],
  address: [{ required: true, message: '请输入地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }]
}

const formRef = ref(null)
const uploadRef = ref(null)

// 获取服务详情
const fetchServiceDetail = async () => {
  loading.value = true
  try {
    const res = await getServiceDetail({
      group_id: groupId.value,
      service: serviceName.value
    })
    serviceDetail.value = res.data
  } catch (error) {
    ElMessage.error('获取服务详情失败')
  } finally {
    loading.value = false
  }
}

// 刷新
const handleRefresh = () => {
  fetchServiceDetail()
}

// 返回
const handleBack = () => {
  router.back()
}

// 获取状态类型
const getStatusType = (status) => {
  const statusMap = {
    passing: 'success',
    warning: 'warning',
    critical: 'danger'
  }
  return statusMap[status] || 'info'
}

// 注册实例
const handleRegisterInstance = () => {
  dialogTitle.value = '注册实例'
  isEdit.value = false
  resetForm()
  instanceForm.service = serviceName.value
  dialogVisible.value = true
}

// 编辑实例
const handleEdit = (row) => {
  dialogTitle.value = '编辑实例'
  isEdit.value = true
  instanceForm.id = row.id
  instanceForm.service = row.service_name
  instanceForm.address = row.address
  instanceForm.port = row.port
  instanceForm.tags = row.tags || []
  instanceForm.metaList = Object.entries(row.meta || {}).map(([key, value]) => ({ key, value }))
  instanceForm.enableHealthCheck = row.checks && row.checks.length > 0
  dialogVisible.value = true
}

// 重置表单
const resetForm = () => {
  instanceForm.id = ''
  instanceForm.service = ''
  instanceForm.address = ''
  instanceForm.port = 8080
  instanceForm.tags = []
  instanceForm.metaList = []
  instanceForm.enableHealthCheck = false
  instanceForm.checkType = 'http'
  instanceForm.interval = '10s'
  instanceForm.timeout = '5s'
  instanceForm.http = ''
}

// 添加 Meta 项
const addMetaItem = () => {
  instanceForm.metaList.push({ key: '', value: '' })
}

// 删除 Meta 项
const removeMetaItem = (index) => {
  instanceForm.metaList.splice(index, 1)
}

// 提交表单
const handleSubmit = async () => {
  await formRef.value.validate()
  
  submitLoading.value = true
  try {
    const meta = {}
    instanceForm.metaList.forEach(item => {
      if (item.key) {
        meta[item.key] = item.value
      }
    })
    
    const data = {
      group_id: groupId.value,
      service: instanceForm.service,
      address: instanceForm.address,
      port: instanceForm.port,
      tags: instanceForm.tags,
      meta
    }
    
    if (!isEdit.value) {
      data.id = instanceForm.id
      if (instanceForm.enableHealthCheck) {
        data.check = {
          [instanceForm.checkType]: instanceForm.http || `${instanceForm.address}:${instanceForm.port}`,
          interval: instanceForm.interval,
          timeout: instanceForm.timeout
        }
      }
      await registerInstance(data)
      ElMessage.success('注册成功')
    } else {
      data.instance_id = instanceForm.id
      await updateInstance(data)
      ElMessage.success('更新成功')
    }
    
    dialogVisible.value = false
    fetchServiceDetail()
  } catch (error) {
    ElMessage.error(isEdit.value ? '更新失败' : '注册失败')
  } finally {
    submitLoading.value = false
  }
}

// 删除实例
const handleDeleteInstance = async (row) => {
  try {
    await deleteInstance({
      group_id: groupId.value,
      instance_id: row.id
    })
    ElMessage.success('删除成功')
    fetchServiceDetail()
  } catch (error) {
    ElMessage.error('删除失败')
  }
}

// 导出
const handleExport = () => {
  exportDialogVisible.value = true
}

// 确认导出
const handleConfirmExport = async () => {
  exportLoading.value = true
  try {
    const res = await exportInstances({
      group_id: groupId.value,
      service: serviceName.value,
      format: exportFormat.value
    })
    
    // 创建下载
    const blob = new Blob([res], { type: 'application/octet-stream' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${serviceName.value}.${exportFormat.value}`
    link.click()
    window.URL.revokeObjectURL(url)
    
    ElMessage.success('导出成功')
    exportDialogVisible.value = false
  } catch (error) {
    ElMessage.error('导出失败')
  } finally {
    exportLoading.value = false
  }
}

// 导入
const handleImport = () => {
  uploadFile.value = null
  importDialogVisible.value = true
}

// 文件选择
const handleFileChange = (file) => {
  uploadFile.value = file.raw
}

// 确认导入
const handleConfirmImport = async () => {
  if (!uploadFile.value) {
    ElMessage.warning('请选择文件')
    return
  }
  
  importLoading.value = true
  try {
    const formData = new FormData()
    formData.append('file', uploadFile.value)
    formData.append('group_id', groupId.value)
    
    await importInstances(formData)
    ElMessage.success('导入成功')
    importDialogVisible.value = false
    uploadRef.value.clearFiles()
    fetchServiceDetail()
  } catch (error) {
    ElMessage.error('导入失败')
  } finally {
    importLoading.value = false
  }
}

onMounted(() => {
  if (!groupId.value || !serviceName.value) {
    ElMessage.error('缺少参数')
    router.back()
    return
  }
  fetchServiceDetail()
})
</script>

<style scoped>
.service-detail-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.text-muted {
  color: #909399;
}

.meta-item {
  padding: 5px 0;
}

.meta-input {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}
</style>
