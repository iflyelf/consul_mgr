<template>
  <el-container class="layout-container">
    <el-header class="layout-header">
      <div class="header-left">
        <h2>Consul Manager</h2>
      </div>
      
      <div class="header-right">
        <el-dropdown class="theme-switcher" @command="handleThemeChange">
          <el-button :icon="Sunny" circle />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="light">浅色主题</el-dropdown-item>
              <el-dropdown-item command="dark">深色主题</el-dropdown-item>
              <el-dropdown-item command="blue">蓝色主题</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <el-dropdown @command="handleCommand">
          <span class="user-info">
            <el-avatar :icon="UserFilled" />
            <span class="username">{{ userStore.userInfo.username || '用户' }}</span>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout" :icon="SwitchButton">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>
    
    <el-container>
      <el-aside width="200px" class="layout-aside">
        <el-menu
          :default-active="$route.path"
          router
        >
          <el-menu-item index="/groups">
            <el-icon><Grid /></el-icon>
            <span>服务组管理</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
      
      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { UserFilled, SwitchButton, Sunny, Grid } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import { useThemeStore } from '@/store/theme'
import { logout } from '@/api/auth'

const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()

const handleCommand = async (command) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      
      await logout()
      userStore.clearAuth()
      router.push('/login')
    } catch (error) {
      // 取消或错误
    }
  }
}

const handleThemeChange = (theme) => {
  themeStore.setTheme(theme)
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.layout-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--card-bg);
  border-bottom: 1px solid var(--border-color);
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.header-left h2 {
  color: var(--primary-color);
  font-size: 20px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 15px;
}

.theme-switcher {
  margin-right: 10px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  color: var(--text-color);
}

.username {
  font-size: 14px;
}

@media (max-width: 768px) {
  .username {
    display: none;
  }
}

.layout-aside {
  background: var(--card-bg);
  border-right: 1px solid var(--border-color);
}

.layout-main {
  background: var(--bg-color);
  overflow-y: auto;
}

@media (max-width: 768px) {
  .layout-aside {
    width: 60px !important;
  }
  
  .el-menu-item span {
    display: none;
  }
}
</style>
