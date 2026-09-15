<template>
  <el-container class="layout-container">
    <el-header class="layout-header">
      <div class="header-left">
        <h2>Consul Manager</h2>
      </div>
      
      <div class="header-right">
        <el-dropdown class="theme-switcher" @command="handleThemeChange">
          <el-button :icon="themeIcon" circle />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="t in themeStore.themes"
                :key="t.value"
                :command="t.value"
                :class="{ 'is-active-theme': themeStore.currentTheme === t.value }"
              >
                {{ t.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <el-dropdown @command="handleCommand">
          <span class="user-info">
            <el-avatar :src="userStore.avatar" :icon="UserFilled" />
            <span class="username">{{ userStore.displayName }}</span>
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item disabled>
                <div class="user-info-dropdown">
                  <div>{{ userStore.displayName }}</div>
                  <div class="user-email">{{ userStore.email }}</div>
                </div>
              </el-dropdown-item>
              <el-dropdown-item divided command="logout" :icon="SwitchButton">退出登录</el-dropdown-item>
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
          <el-menu-item index="/services">
            <el-icon><Connection /></el-icon>
            <span>Services 管理</span>
          </el-menu-item>
          <el-menu-item index="/instances">
            <el-icon><Monitor /></el-icon>
            <span>Instances 管理</span>
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
import { computed } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { UserFilled, SwitchButton, Sunny, MostlyCloudy, Moon, Grid, Connection, Monitor, ArrowDown } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import { useThemeStore } from '@/store/theme'

const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()

// 当前主题对应的切换按钮图标
const themeIcon = computed(() => {
  switch (themeStore.currentTheme) {
    case 'cool': return MostlyCloudy
    case 'dark': return Moon
    default: return Sunny
  }
})

const handleCommand = async (command) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      
      // 调用 Store 的 logout 方法
      await userStore.logout()
      
      ElMessage.success('已退出登录')
      router.push('/login')
    } catch (error) {
      // 取消或错误
      if (error !== 'cancel') {
        console.error('登出失败:', error)
      }
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

.is-active-theme {
  color: var(--primary-color);
  font-weight: 600;
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

.user-info-dropdown {
  padding: 5px 0;
}

.user-email {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
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
