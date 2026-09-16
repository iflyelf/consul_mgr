<template>
  <div class="users-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>用户管理</span>
          <div>
            <el-input
              v-model="keyword"
              placeholder="搜索用户名 / 姓名 / 邮箱"
              clearable
              style="width: 260px; margin-right: 10px"
              @keyup.enter="handleSearch"
              @clear="handleSearch"
            />
            <el-button type="primary" @click="handleSearch" :loading="loading">搜索</el-button>
            <el-button @click="handleRefresh" :loading="loading">刷新</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="用户体系由 Casdoor 统一维护（认证 / 密码 / 角色），此处为只读展示"
        style="margin-bottom: 16px"
      />

      <el-table :data="users" v-loading="loading" stripe border>
        <el-table-column label="用户" min-width="220">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="28" :src="row.avatar" />
              <div>
                <div class="user-name">{{ row.displayName || row.name }}</div>
                <div class="user-sub">{{ row.name }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱" min-width="200" show-overflow-tooltip />
        <el-table-column prop="phone" label="电话" width="150" />
        <el-table-column label="类型" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isAdmin" type="danger" size="small">管理员</el-tag>
            <el-tag v-else type="info" size="small">普通用户</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="owner" label="所属组织" width="130" />
        <el-table-column prop="createdTime" label="创建时间" width="180" />
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
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getUsers } from '@/api/org'

const loading = ref(false)
const keyword = ref('')
const users = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)

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
    ElMessage.error('获取用户列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索：回到第一页
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

onMounted(loadData)
</script>

<style scoped>
.users-container { width: 100%; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.user-cell { display: flex; align-items: center; gap: 10px; }
.user-name { font-weight: 500; }
.user-sub { font-size: 12px; color: var(--text-secondary); }
.pagination { margin-top: 16px; text-align: right; }
.total { color: var(--text-secondary); font-size: 13px; }
</style>
