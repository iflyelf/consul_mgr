<template>
  <div class="instances-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Consul Instances 管理</span>
          <div>
            <el-button type="primary" @click="handleRegister">
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
            <el-button @click="handleRefresh" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :inline="true" :model="searchForm">
          <el-form-item label="服务组">
            <el-select v-model="searchForm.group_id" placeholder="请选择服务组" clearable @change="handleGroupChange">
              <el-option
                v-for="group in groups"
                :key="group.id"
                :label="group.name"
                :value="group.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="服务名称">
            <el-select v-model="searchForm.service" placeholder="请选择服务" clearable @change="handleServiceChange">
              <el-option
                v-for="service in services"
                :key="service.name"
                :label="service.name"
                :value="service.name"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="searchForm.status" placeholder="全部" clearable>
              <el-option label="健康" value="passing" />
              <el-option label="警告" value="warning" />
              <el-option label="异常" value="critical" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch" :loading="loading">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 批量操作 -->
      <div class="batch-actions" v-if="selectedInstances.length > 0">
        <el-alert
          :title="`已选择 ${selectedInstances.length} 个实例`"
          type="info"
          :closable="false"
        />
        <el-button type="danger" @click="handleBatchDelete" :loading="batchLoading">
          批量删除
        </el-button>
      </div>

      <!-- 实例列表 -->
      <el-table
        :data="instances"
        v-loading="loading"
        @selection-change="handleSelectionChange"
        stripe
        border
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="instance_id" label="实例ID" min-width="200" show-overflow-tooltip />
        <el-table-column prop="service_name" label="服务名称" min-width="150" />
        <el-table-column label="地址" min-width="150">
          <template #default="{ row }">
            {{ row.address }}:{{ row.port }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
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
        <el-table-column label="Meta" min-width="150" align="center">
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
            <el-tooltip
              v-if="row.checks && row.checks.length > 0"
              :content="`${row.checks.length} 个检查`"
              placement="top"
            >
              <el-icon color="#67C23A" :size="20"><CircleCheck /></el-icon>
            </el-tooltip>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确定要删除这个实例吗？"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 空状态 -->
      <el-empty v-if="!loading && instances.length === 0" description="暂无实例" />

      <!-- 分页 -->
      <div class="pagination" v-if="total > 0">
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

    <!-- 注册/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="700px"
    >
      <el-form :model="instanceForm" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="服务组" prop="group_id">
          <el-select v-model="instanceForm.group_id" placeholder="请选择" :disabled="isEdit">
            <el-option
              v-for="group in groups"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="实例ID" prop="id" v-if="!isEdit">
          <el-input v-model="instanceForm.id" placeholder="唯一标识，如: instance-001" />
        </el-form-item>
        <el-form-item label="服务名称" prop="service">
          <el-input v-model="instanceForm.service" placeholder="服务名称" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="地址" prop="address">
          <el-input v-model="instanceForm.address" placeholder="IP 或域名" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="instanceForm.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="Tags">
          <el-select
            v-model="instanceForm.tags"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="添加标签，可输入自定义标签"
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
            <el-input v-model="meta.key" placeholder="Key" style="width: 35%" />
            <el-input v-model="meta.value" placeholder="Value" style="width: 45%; margin-left: 10px" />
            <el-button type="danger" size="small" @click="removeMetaItem(index)" style="margin-left: 10px">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button type="primary" size="small" @click="addMetaItem">
            <el-icon><Plus /></el-icon>
            添加 Meta
          </el-button>
        </el-form-item>
        
        <el-divider content-position="left">健康检查配置</el-divider>
        
        <el-form-item label="启用检查">
          <el-switch v-model="instanceForm.enableHealthCheck" />
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
          
          <el-form-item label="检查地址" prop="checkUrl" v-if="instanceForm.checkType === 'http'">
            <el-input v-model="instanceForm.checkUrl" placeholder="http://localhost:8080/health">
              <template #prepend>http://</template>
            </el-input>
          </el-form-item>
          
          <el-form-item label="检查端口" prop="checkPort" v-if="instanceForm.checkType === 'tcp'">
            <el-input-number v-model="instanceForm.checkPort" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
          
          <el-form-item label="GRPC地址" prop="grpc" v-if="instanceForm.checkType === 'grpc'">
            <el-input v-model="instanceForm.grpc" placeholder="localhost:9090" />
          </el-form-item>
          
          <el-form-item label="检查间隔" prop="interval" v-if="instanceForm.checkType !== 'ttl'">
            <el-input v-model="instanceForm.interval" placeholder="10s">
              <template #append>秒</template>
            </el-input>
          </el-form-item>
          
          <el-form-item label="超时时间" prop="timeout" v-if="instanceForm.checkType !== 'ttl'">
            <el-input v-model="instanceForm.timeout" placeholder="5s">
              <template #append>秒</template>
            </el-input>
          </el-form-item>
          
          <el-form-item label="TTL时间" prop="ttl" v-if="instanceForm.checkType === 'ttl'">
            <el-input v-model="instanceForm.ttl" placeholder="30s">
              <template #append>秒</template>
            </el-input>
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
      <el-form label-width="100px">
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
    <el-dialog v-model="importDialogVisible" title="导入实例" width="500px">
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :limit="1"
        :on-change="handleFileChange"
        drag
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          拖拽文件到此处或 <em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            支持 JSON/YAML/CSV 格式，单个文件不超过 10MB
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Download, Upload, Refresh, CircleCheck, Delete, UploadFilled } from '@element-plus/icons-vue'
import { getGroups } from '@/api/group'
import { getServices } from '@/api/service'
import { 
  getInstances, 
  registerInstance, 
  updateInstance, 
  deleteInstance,
  batchDeleteInstances,
  exportInstances,
  importInstances 
} from '@/api/instance'

const loading = ref(false)
const submitLoading = ref(false)
const batchLoading = ref(false)
const exportLoading = ref(false)
const importLoading = ref(false)

const groups = ref([])
const services = ref([])
const instances = ref([])
const selectedInstances = ref([])

const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const searchForm = reactive({
  group_id: '',
  service: '',
  status: ''
})

const dialogVisible = ref(false)
const dialogTitle = ref('注册实例')
const isEdit = ref(false)

const exportDialogVisible = ref(false)
const exportFormat = ref('json')

const importDialogVisible = ref(false)
const uploadFile = ref(null)

const commonTags = ref(['production', 'staging', 'development', 'canary', 'v1', 'v2', 'v3'])

const instanceForm = reactive({
  group_id: '',
  id: '',
  service: '',
  address: '',
  port: 8080,
  tags: [],
  metaList: [],
  enableHealthCheck: false,
  checkType: 'http',
  checkUrl: '',
  checkPort: 8080,
  grpc: '',
  interval: '10s',
  timeout: '5s',
  ttl: '30s'
})

const rules = {
  group_id: [{ required: true, message: '请选择服务组', trigger: 'change' }],
  id: [{ required: true, message: '请输入实例ID', trigger: 'blur' }],
  service: [{ required: true, message: '请输入服务名称', trigger: 'blur' }],
  address: [{ required: true, message: '请输入地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }]
}

const formRef = ref(null)
const uploadRef = ref(null)

// 获取服务组列表
const fetchGroups = async () => {
  try {
    const res = await getGroups()
    groups.value = res.list || []
    if (groups.value.length > 0 && !searchForm.group_id) {
      searchForm.group_id = groups.value[0].id
      await fetchServices()
    }
  } catch (error) {
    ElMessage.error('获取服务组失败')
  }
}

// 获取服务列表
const fetchServices = async () => {
  if (!searchForm.group_id) return
  
  try {
    const res = await getServices({ group_id: searchForm.group_id })
    services.value = res.list || []
  } catch (error) {
    ElMessage.error('获取服务列表失败')
  }
}

// 获取实例列表
const fetchInstances = async () => {
  if (!searchForm.group_id) {
    ElMessage.warning('请先选择服务组')
    return
  }
  
  loading.value = true
  try {
    const res = await getInstances(searchForm)
    instances.value = res.list || []
    total.value = res.total || 0
  } catch (error) {
    ElMessage.error('获取实例列表失败')
  } finally {
    loading.value = false
  }
}

// 服务组切换
const handleGroupChange = async () => {
  searchForm.service = ''
  await fetchServices()
  fetchInstances()
}

// 服务切换
const handleServiceChange = () => {
  fetchInstances()
}

// 搜索
const handleSearch = () => {
  currentPage.value = 1
  fetchInstances()
}

// 重置
const handleReset = () => {
  searchForm.service = ''
  searchForm.status = ''
  currentPage.value = 1
  fetchInstances()
}

// 刷新
const handleRefresh = () => {
  fetchInstances()
}

// 注册实例
const handleRegister = () => {
  if (!searchForm.group_id) {
    ElMessage.warning('请先选择服务组')
    return
  }
  
  dialogTitle.value = '注册实例'
  isEdit.value = false
  resetForm()
  instanceForm.group_id = searchForm.group_id
  dialogVisible.value = true
}

// 编辑实例
const handleEdit = (row) => {
  dialogTitle.value = '编辑实例'
  isEdit.value = true
  instanceForm.group_id = searchForm.group_id
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
  instanceForm.checkUrl = ''
  instanceForm.checkPort = 8080
  instanceForm.grpc = ''
  instanceForm.interval = '10s'
  instanceForm.timeout = '5s'
  instanceForm.ttl = '30s'
}

// 添加 Meta
const addMetaItem = () => {
  instanceForm.metaList.push({ key: '', value: '' })
}

// 删除 Meta
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
      group_id: instanceForm.group_id,
      service: instanceForm.service,
      address: instanceForm.address,
      port: instanceForm.port,
      tags: instanceForm.tags,
      meta
    }
    
    if (!isEdit.value) {
      data.id = instanceForm.id
      if (instanceForm.enableHealthCheck) {
        const check = {
          interval: instanceForm.interval,
          timeout: instanceForm.timeout
        }
        
        switch (instanceForm.checkType) {
          case 'http':
            check.http = instanceForm.checkUrl
            break
          case 'tcp':
            check.tcp = `${instanceForm.address}:${instanceForm.checkPort}`
            break
          case 'grpc':
            check.grpc = instanceForm.grpc
            break
          case 'ttl':
            check.ttl = instanceForm.ttl
            break
        }
        
        data.check = check
      }
      await registerInstance(data)
      ElMessage.success('注册成功')
    } else {
      data.instance_id = instanceForm.id
      await updateInstance(data)
      ElMessage.success('更新成功')
    }
    
    dialogVisible.value = false
    fetchInstances()
  } catch (error) {
    ElMessage.error(isEdit.value ? '更新失败' : '注册失败')
  } finally {
    submitLoading.value = false
  }
}

// 删除实例
const handleDelete = async (row) => {
  try {
    await deleteInstance({
      group_id: searchForm.group_id,
      instance_id: row.id
    })
    ElMessage.success('删除成功')
    fetchInstances()
  } catch (error) {
    ElMessage.error('删除失败')
  }
}

// 选择变化
const handleSelectionChange = (selection) => {
  selectedInstances.value = selection
}

// 批量删除
const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedInstances.value.length} 个实例吗？`,
      '批量删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    batchLoading.value = true
    const instanceIds = selectedInstances.value.map(i => i.id)
    await batchDeleteInstances({
      group_id: searchForm.group_id,
      instance_ids: instanceIds
    })
    
    ElMessage.success('批量删除成功')
    selectedInstances.value = []
    fetchInstances()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  } finally {
    batchLoading.value = false
  }
}

// 导出
const handleExport = () => {
  if (!searchForm.group_id) {
    ElMessage.warning('请先选择服务组')
    return
  }
  exportDialogVisible.value = true
}

// 确认导出
const handleConfirmExport = async () => {
  exportLoading.value = true
  try {
    const res = await exportInstances({
      group_id: searchForm.group_id,
      service: searchForm.service,
      format: exportFormat.value
    })
    
    const blob = new Blob([res], { type: 'application/octet-stream' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `instances_${searchForm.service || 'all'}.${exportFormat.value}`
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
  if (!searchForm.group_id) {
    ElMessage.warning('请先选择服务组')
    return
  }
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
    formData.append('group_id', searchForm.group_id)
    
    await importInstances(formData)
    ElMessage.success('导入成功')
    importDialogVisible.value = false
    uploadRef.value.clearFiles()
    fetchInstances()
  } catch (error) {
    ElMessage.error('导入失败')
  } finally {
    importLoading.value = false
  }
}

// 分页
const handleSizeChange = (val) => {
  pageSize.value = val
  fetchInstances()
}

const handleCurrentChange = (val) => {
  currentPage.value = val
  fetchInstances()
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

// 获取状态文本
const getStatusText = (status) => {
  const statusMap = {
    passing: '健康',
    warning: '警告',
    critical: '异常'
  }
  return statusMap[status] || status
}

onMounted(() => {
  fetchGroups()
})
</script>

<style scoped>
.instances-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-bar {
  margin-bottom: 20px;
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}

.text-muted {
  color: #909399;
}

.meta-item {
  padding: 5px 0;
  border-bottom: 1px solid #eee;
}

.meta-item:last-child {
  border-bottom: none;
}

.meta-input {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
