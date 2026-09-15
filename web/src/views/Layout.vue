<template>
  <div class="layout">
    <!-- 顶栏 -->
    <header class="topbar">
      <div class="brand">
        <img class="brand-logo" src="/favicon.svg" alt="logo" />
        <span class="brand-text">Consul Manager</span>
      </div>

      <!-- 桌面端导航 -->
      <nav class="tabs desktop-tabs">
        <router-link
          v-for="m in menus"
          :key="m.path"
          :to="m.path"
          class="tab"
          :class="{ active: $route.path.startsWith(m.path) }"
        >
          <el-icon><component :is="m.icon" /></el-icon>
          <span>{{ m.label }}</span>
        </router-link>
      </nav>

      <div class="topbar-actions">
        <!-- 主题切换（药丸分组，右上角） -->
        <div class="theme-switcher">
          <button
            v-for="t in themeStore.themes"
            :key="t.value"
            class="theme-btn"
            :class="{ active: themeStore.currentTheme === t.value }"
            :title="t.label"
            @click="themeStore.setTheme(t.value)"
          >
            {{ t.emoji }}
          </button>
        </div>

        <!-- 用户 -->
        <el-dropdown @command="handleCommand">
          <span class="user-info">
            <el-avatar :size="30" :src="userStore.avatar" :icon="UserFilled" />
            <span class="username">{{ userStore.displayName }}</span>
            <el-icon class="user-arrow"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item disabled>
                <div class="user-info-dropdown">
                  <div class="user-name">{{ userStore.displayName }}</div>
                  <div class="user-email">{{ userStore.email }}</div>
                </div>
              </el-dropdown-item>
              <el-dropdown-item divided command="logout" :icon="SwitchButton">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>

    <!-- 移动端导航（横向滚动） -->
    <nav class="tabs mobile-tabs">
      <router-link
        v-for="m in menus"
        :key="m.path"
        :to="m.path"
        class="tab"
        :class="{ active: $route.path.startsWith(m.path) }"
      >
        <el-icon><component :is="m.icon" /></el-icon>
        <span>{{ m.label }}</span>
      </router-link>
    </nav>

    <!-- 主内容 -->
    <main class="layout-main">
      <div class="page-container">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { UserFilled, SwitchButton, Grid, Connection, Monitor, ArrowDown } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import { useThemeStore } from '@/store/theme'

const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()

const menus = [
  { path: '/groups', label: '服务组管理', icon: Grid },
  { path: '/services', label: 'Services 管理', icon: Connection },
  { path: '/instances', label: 'Instances 管理', icon: Monitor }
]

const handleCommand = async (command) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      await userStore.logout()
      ElMessage.success('已退出登录')
      router.push('/login')
    } catch (error) {
      if (error !== 'cancel') {
        console.error('登出失败:', error)
      }
    }
  }
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* ---------------- 顶栏 ---------------- */
.topbar {
  position: sticky;
  top: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 20px;
  background: color-mix(in srgb, var(--card-bg) 88%, transparent);
  backdrop-filter: saturate(180%) blur(14px);
  -webkit-backdrop-filter: saturate(180%) blur(14px);
  border-bottom: 1px solid var(--border-color);
  box-shadow: var(--shadow);
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.brand-logo {
  width: 30px;
  height: 30px;
  border-radius: 9px;
}

.brand-text {
  font-size: 17px;
  font-weight: 700;
  color: var(--primary-color);
  white-space: nowrap;
}

/* ---------------- 导航 ---------------- */
.tabs {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
}

.tabs::-webkit-scrollbar {
  display: none;
}

/* 桌面导航占据中间剩余空间 */
.desktop-tabs {
  flex: 1;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  text-decoration: none;
  white-space: nowrap;
  transition: all 0.2s;
}

.tab:hover {
  background: color-mix(in srgb, var(--primary-color) 10%, transparent);
}

.tab.active {
  background: var(--primary-color);
  color: #fffaf3;
}

/* 移动端导航默认隐藏 */
.mobile-tabs {
  display: none;
}

/* ---------------- 右侧操作 ---------------- */
.topbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

/* 药丸主题切换 */
.theme-switcher {
  display: flex;
  gap: 4px;
  background: var(--panel-bg);
  padding: 4px;
  border-radius: var(--radius);
  border: 1px solid var(--border-color);
}

.theme-btn {
  background: transparent;
  border: none;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  font-size: 15px;
  line-height: 1;
  cursor: pointer;
  opacity: 0.55;
  transition: all 0.2s;
}

.theme-btn:hover {
  opacity: 1;
  background: color-mix(in srgb, var(--primary-color) 12%, transparent);
}

.theme-btn.active {
  opacity: 1;
  background: color-mix(in srgb, var(--primary-color) 22%, transparent);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: var(--text-color);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  transition: background 0.2s;
}

.user-info:hover {
  background: color-mix(in srgb, var(--primary-color) 10%, transparent);
}

.username {
  font-size: 14px;
}

.user-arrow {
  font-size: 12px;
  color: var(--text-secondary);
}

.user-info-dropdown {
  padding: 4px 0;
}

.user-name {
  font-weight: 600;
  color: var(--text-color);
}

.user-email {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}

/* ---------------- 主内容 ---------------- */
.layout-main {
  flex: 1;
  overflow-x: hidden;
}

.layout-main .page-container {
  max-width: 1400px;
  margin: 0 auto;
}

/* ---------------- H5 自适应 ---------------- */
@media (max-width: 860px) {
  .desktop-tabs {
    display: none;
  }

  .topbar {
    padding: 10px 14px;
  }

  /* 移动端导航置顶第二行，横向滚动 */
  .mobile-tabs {
    display: flex;
    flex: none;
    align-items: center;
    position: sticky;
    top: 51px;
    z-index: 190;
    padding: 8px 12px;
    background: color-mix(in srgb, var(--card-bg) 92%, transparent);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border-bottom: 1px solid var(--border-color);
  }

  .mobile-tabs .tab {
    flex: 0 0 auto;
    justify-content: center;
    min-width: 96px;
    height: 36px;
    padding: 0 12px;
  }
}

@media (max-width: 640px) {
  .brand-text {
    font-size: 15px;
  }

  .username {
    display: none;
  }

  .theme-btn {
    padding: 4px 6px;
    font-size: 14px;
  }

  .layout-main .page-container {
    padding: 12px;
  }
}

@media (max-width: 400px) {
  .brand-text {
    display: none;
  }
}
</style>
