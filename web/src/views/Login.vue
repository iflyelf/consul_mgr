<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="card-header">
          <h2>Consul Manager</h2>
          <p>统一服务注册与配置管理平台</p>
        </div>
      </template>
      
      <div class="login-content">
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          @click="handleCasdoorLogin"
          class="login-button"
        >
          <el-icon class="el-icon--left"><User /></el-icon>
          使用 Casdoor 登录
        </el-button>
        
        <div class="login-tips">
          <el-alert
            title="首次登录请联系管理员开通账号"
            type="info"
            :closable="false"
            show-icon
          />
        </div>
        
        <div class="login-info">
          <el-divider />
          <p class="info-text">
            <el-icon><InfoFilled /></el-icon>
            使用 Casdoor 统一认证平台登录
          </p>
          <p class="info-text">
            <el-icon><Lock /></el-icon>
            支持 SSO 单点登录
          </p>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { User, InfoFilled, Lock } from '@element-plus/icons-vue'

const loading = ref(false)

/**
 * 跳转到 Casdoor 登录
 *
 * 说明：后端 /api/auth/login 会按当前访问的域名动态生成 Casdoor 登录地址并 302 跳转，
 * 前端无需知道 Casdoor 地址，跨域名/IP 部署自动适配。
 */
const handleCasdoorLogin = () => {
  loading.value = true
  window.location.href = '/api/auth/login'
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-card {
  width: 450px;
  max-width: 100%;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
}

.card-header {
  text-align: center;
  padding: 10px 0;
}

.card-header h2 {
  margin: 0 0 10px 0;
  color: #303133;
  font-size: 28px;
  font-weight: 600;
}

.card-header p {
  margin: 0;
  color: #909399;
  font-size: 14px;
}

.login-content {
  padding: 20px 0;
}

.login-button {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 500;
  border-radius: 6px;
}

.login-tips {
  margin-top: 20px;
}

.login-info {
  margin-top: 20px;
}

.info-text {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #606266;
  font-size: 14px;
  margin: 8px 0;
}

.info-text .el-icon {
  color: #409eff;
}

@media (max-width: 768px) {
  .login-card {
    width: 100%;
  }
  
  .card-header h2 {
    font-size: 24px;
  }
}
</style>
