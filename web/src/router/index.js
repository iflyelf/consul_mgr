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
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  // 设置页面标题
  document.title = to.meta.title ? `${to.meta.title} - Consul Manager` : 'Consul Manager'
  
  // 获取用户 Store
  const userStore = useUserStore()
  const isLoggedIn = userStore.isLoggedIn
  
  // 如果路由需要认证
  if (to.meta.requiresAuth !== false) {
    if (!isLoggedIn) {
      // 未登录，跳转到登录页
      next('/login')
      return
    }
  }
  
  // 如果已登录访问登录页，跳转到首页
  if (to.path === '/login' && isLoggedIn) {
    next('/')
    return
  }
  
  next()
})

export default router
