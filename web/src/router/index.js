import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/store/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { 
      title: '登录',
      requiresAuth: false
    }
  },
  {
    path: '/callback',
    name: 'Callback',
    component: () => import('@/views/Callback.vue'),
    meta: { 
      title: 'OAuth 回调',
      requiresAuth: false
    }
  },
  {
    path: '/',
    name: 'Layout',
    component: () => import('@/views/Layout.vue'),
    redirect: '/groups',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'groups',
        name: 'Groups',
        component: () => import('@/views/Groups.vue'),
        meta: { title: '服务组管理' }
      },
      {
        path: 'services',
        name: 'Services',
        component: () => import('@/views/Services.vue'),
        meta: { title: 'Services 管理' }
      },
      {
        path: 'service-detail',
        name: 'ServiceDetail',
        component: () => import('@/views/ServiceDetail.vue'),
        meta: { title: '服务详情' }
      },
      {
        path: 'instances',
        name: 'Instances',
        component: () => import('@/views/Instances.vue'),
        meta: { title: 'Instances 管理' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { title: '用户管理', requiresAdmin: true }
      },
      {
        path: 'teams',
        name: 'Teams',
        component: () => import('@/views/Teams.vue'),
        meta: { title: '团队管理', requiresAdmin: true }
      },
      {
        path: 'roles',
        name: 'Roles',
        component: () => import('@/views/Roles.vue'),
        meta: { title: '角色管理', requiresAdmin: true }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 动态导入（懒加载 chunk）失败处理
//
// 场景：发版后旧页面仍引用已删除的旧 hash chunk，服务端返回 404/HTML，
// 浏览器报 "Failed to fetch dynamically imported module"。
// 处理：本地标记 + 强制刷新一次，让浏览器重新获取最新 index.html 与 chunk。
const CHUNK_RELOAD_KEY = 'consul_mgr:chunk-reloaded'
const isChunkLoadError = (err) => {
  const msg = String(err?.message || err || '')
  return (
    msg.includes('Failed to fetch dynamically imported module') ||
    msg.includes('Importing a module script failed') ||
    msg.includes('error loading dynamically imported module') ||
    err?.name === 'ChunkLoadError'
  )
}
const reloadOnce = () => {
  if (sessionStorage.getItem(CHUNK_RELOAD_KEY)) {
    sessionStorage.removeItem(CHUNK_RELOAD_KEY)
    return
  }
  sessionStorage.setItem(CHUNK_RELOAD_KEY, '1')
  window.location.reload()
}
router.onError((err) => {
  if (isChunkLoadError(err)) reloadOnce()
})
window.addEventListener('vite:preloadError', () => reloadOnce())
window.addEventListener('load', () => sessionStorage.removeItem(CHUNK_RELOAD_KEY))

// 路由守卫
router.beforeEach(async (to, from, next) => {
  // 设置页面标题
  document.title = to.meta.title ? `${to.meta.title} - Consul Manager` : 'Consul Manager'
  
  // 获取用户 Store
  const userStore = useUserStore()

  // 登录态探测（凭证在 HttpOnly Cookie，需向后端确认，仅执行一次）
  if (!userStore.initialized) {
    await userStore.init()
  }

  // 如果路由需要认证
  if (to.meta.requiresAuth !== false) {
    if (!userStore.isLoggedIn) {
      // 未登录，跳转到登录页
      next('/login')
      return
    }

    // 用户信息缺失管理员标记时，向后端刷新一次（兼容旧缓存）
    if (userStore.userInfo?.is_admin === undefined) {
      await userStore.fetchUserInfo()
    }

    // 需要管理员权限的页面，普通用户重定向到首页
    if (to.meta.requiresAdmin && !userStore.isAdmin) {
      next('/groups')
      return
    }
  }

  // 如果已登录访问登录页，跳转到首页
  if (to.path === '/login' && userStore.isLoggedIn) {
    next('/')
    return
  }

  next()
})

export default router
