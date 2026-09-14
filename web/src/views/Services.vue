<template>
  <div class="services-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Consul Services 管理</span>
          <el-button type="primary" @click="handleRefresh" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :inline="true" :model="searchForm" @submit.prevent="handleSearch">
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
          <el-form-item label="关键词">
            <el-input v-model="searchForm.keyword" placeholder="搜索服务名称" clearable />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch" :loading="loading">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 批量操作 -->
      <div class="batch-actions" v-if="selectedServices.length > 0">
        <el-alert
          :title="`已选择 ${selectedServices.length} 个服务`"
          type="info"
          :closable="false"
        />
        <el-button type="danger" @click="handleBatchDelete" :loading="batchLoading">
          批量删除
        </el-button>
      </div>

      <!-- 服务列表 -->
      <el-table
        :data="services"
        v-loading="loading"
        @selection-change="handleSelectionChange"
        stripe
        border
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" label="服务名称" min-width="150">
          <template #default="{ row }">
            <el-link type="primary" @click="handleViewDetail(row)">{{ row.name }}</el-link>
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
        <el-table-column prop="instance_count" label="实例数" width="100" align="center" />
        <el-table-column label="健康状态" width="200" align="center">
          <template #default="{ row }">
            <div v-if="row.health_summary">
              <el-progress
                :percentage="getHealthPercentage(row.health_summary)"
                :color="getHealthColor(row.health_summary)"
                :format="() => `${row.health_summary.passing}/${row.health_summary.total}`"
              />
            </div>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleViewDetail(row)">详情</el-button>
            <el-popconfirm
              title="确定要删除这个服务吗？"
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
      <el-empty v-if="!loading && services.length === 0" description="暂无服务" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getGroups } from '@/api/group'
import { getServices, deleteService, batchDeleteServices } from '@/api/service'

const router = useRouter()

const loading = ref(false)
const batchLoading = ref(false)
const groups = ref([])
const services = ref([])
const selectedServices = ref([])

const searchForm = reactive({
  group_id: '',
  keyword: ''
})

// 获取服务组列表
const fetchGroups = async () => {
  try {
    const res = await getGroups()
    groups.value = res.data.list || []
    // 默认选择第一个服务组
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
  if (!searchForm.group_id) {
    ElMessage.warning('请先选择服务组')
    return
  }
  
  loading.value = true
  try {
    const res = await getServices(searchForm)
    services.value = res.data.list || []
  } catch (error) {
    ElMessage.error('获取服务列表失败')
  } finally {
    loading.value = false
  }
}

// 服务组切换
const handleGroupChange = () => {
  fetchServices()
}

// 搜索
const handleSearch = () => {
  fetchServices()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  fetchServices()
}

// 刷新
const handleRefresh = () => {
  fetchServices()
}

// 查看详情
const handleViewDetail = (row) => {
  router.push({
    name: 'ServiceDetail',
    query: {
      group_id: searchForm.group_id,
      service: row.name
    }
  })
}

// 删除服务
const handleDelete = async (row) => {
  try {
    await deleteService({
      group_id: searchForm.group_id,
      service: row.name
    })
    ElMessage.success('删除成功')
    fetchServices()
  } catch (error) {
    ElMessage.error('删除失败')
  }
}

// 选择变化
const handleSelectionChange = (selection) => {
  selectedServices.value = selection
}

// 批量删除
const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedServices.value.length} 个服务吗？`,
      '批量删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    batchLoading.value = true
    const serviceNames = selectedServices.value.map(s => s.name)
    await batchDeleteServices({
      group_id: searchForm.group_id,
      services: serviceNames
    })
    
    ElMessage.success('批量删除成功')
    selectedServices.value = []
    fetchServices()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  } finally {
    batchLoading.value = false
  }
}

// 计算健康度百分比
const getHealthPercentage = (health) => {
  if (!health || health.total === 0) return 0
  return Math.round((health.passing / health.total) * 100)
}

// 获取健康度颜色
const getHealthColor = (health) => {
  const percentage = getHealthPercentage(health)
  if (percentage >= 90) return '#67C23A'
  if (percentage >= 60) return '#E6A23C'
  return '#F56C6C'
}

onMounted(() => {
  fetchGroups()
})
</script>

<style scoped>
.services-container {
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
</style>
