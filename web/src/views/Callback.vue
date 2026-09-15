<template>
  <div class="callback-container">
    <el-card>
      <div class="loading-content">
        <el-icon class="is-loading" :size="48" color="#409eff">
          <Loading />
        </el-icon>
        <p class="loading-text">{{ statusText }}</p>
        <el-progress 
          v-if="showProgress"
          :percentage="progress" 
          :stroke-width="8"
          :show-text="false"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { handleCallback } from '@/config/casdoor'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()

const statusText = ref('正在登录，请稍候...')
const showProgress = ref(true)
const progress = ref(0)

// 进度条动画
const startProgress = () => {
  const interval = setInterval(() => {
    if (progress.value < 90) {
      progress.value += 10
    }
  }, 200)
  
  return () => clearInterval(interval)
}

onMounted(async () => {
  const stopProgress = startProgress()
  
  try {
    // 1. 更新状态
    statusText.value = '正在验证授权码...'
    
    // 2. 处理回调，换取 Token
    const data = await handleCallback()
    
    statusText.value = '正在获取用户信息...'
    progress.value = 60
    
    // 3. 保存 Token（同时更新 Pinia Store 与 localStorage）
    const token = data.access_token
    userStore.setToken(token)

    // 4. 保存用户信息到 Store
    if (data.user_info) {
      userStore.setUser(data.user_info)
    }
    
    statusText.value = '登录成功，正在跳转...'
    progress.value = 100
    
    // 5. 延迟跳转，让用户看到成功提示
    setTimeout(() => {
      stopProgress()
      ElMessage.success('登录成功')
      router.push('/')
    }, 500)
    
  } catch (error) {
    stopProgress()
    console.error('登录失败:', error)
    
    ElMessage.error({
      message: error.message || '登录失败，请重试',
      duration: 3000
    })
    
    // 延迟跳转回登录页
    setTimeout(() => {
      router.push('/login')
    }, 2000)
  }
})
</script>

<style scoped>
.callback-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.loading-content {
  text-align: center;
  padding: 60px 40px;
  min-width: 300px;
}

.is-loading {
  animation: rotate 2s linear infinite;
}

@keyframes rotate {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.loading-text {
  margin-top: 24px;
  margin-bottom: 20px;
  color: #606266;
  font-size: 16px;
  font-weight: 500;
}

.el-progress {
  margin-top: 10px;
}
</style>
