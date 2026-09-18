import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiUrl } from '@/config/api'

// 仅缓存非敏感的用户展示信息；登录凭证由后端 HttpOnly Cookie 承载，
// 前端不接触 token（避免 localStorage 被 XSS 窃取）。
const USER_KEY = 'userInfo'

export const useUserStore = defineStore('user', () => {
  // 状态
  const userInfo = ref(JSON.parse(localStorage.getItem(USER_KEY) || 'null'))
  // 登录态以服务端探测为准（Cookie 不可被 JS 读取）
  const isLoggedIn = ref(false)
  const initialized = ref(false)

  // 计算属性
  const username = computed(() => userInfo.value?.name || '')
  const displayName = computed(() => userInfo.value?.display_name || userInfo.value?.name || '')
  const email = computed(() => userInfo.value?.email || '')
  const avatar = computed(() => userInfo.value?.avatar || '')
  const isAdmin = computed(() => userInfo.value?.is_admin || false)
  const roles = computed(() => userInfo.value?.roles || [])

  /**
   * 设置用户信息
   */
  const setUser = (info) => {
    userInfo.value = info
    localStorage.setItem(USER_KEY, JSON.stringify(info))
  }

  /**
   * 清除认证信息（本地用户信息；Cookie 由后端清除）
   */
  const clearAuth = () => {
    userInfo.value = null
    isLoggedIn.value = false
    localStorage.removeItem(USER_KEY)
  }

  /**
   * 检查是否有指定角色
   */
  const hasRole = (role) => roles.value.includes(role)

  /**
   * 检查是否有任一角色
   */
  const hasAnyRole = (...roleList) => roleList.some(role => roles.value.includes(role))

  /**
   * 从后端刷新当前用户信息（凭证由 Cookie 自动携带）
   * @param {boolean} silent 不触发全局 401 跳转（用于登录态探测）
   */
  const fetchUserInfo = async (silent = false) => {
    try {
      const resp = await fetch(apiUrl('/auth/userinfo'), {
        credentials: 'include'
      })
      const result = await resp.json()
      if (result.code === 200 && result.data) {
        setUser(result.data)
        isLoggedIn.value = true
        return result.data
      }
    } catch (error) {
      if (!silent) console.error('获取用户信息失败:', error)
    }
    return null
  }

  /**
   * 初始化：探测登录态（应用启动/路由守卫调用一次）
   */
  const init = async () => {
    if (initialized.value) return
    try {
      await fetchUserInfo(true)
    } finally {
      initialized.value = true
    }
  }

  /**
   * 登出：通知后端清除 Cookie
   */
  const logout = async () => {
    try {
      await fetch(apiUrl('/auth/logout'), { method: 'POST', credentials: 'include' })
    } catch (error) {
      console.error('登出请求失败:', error)
    } finally {
      clearAuth()
    }
  }

  return {
    // 状态
    userInfo,
    isLoggedIn,
    initialized,

    // 计算属性
    username,
    displayName,
    email,
    avatar,
    isAdmin,
    roles,

    // 方法
    setUser,
    clearAuth,
    hasRole,
    hasAnyRole,
    fetchUserInfo,
    init,
    logout
  }
})
