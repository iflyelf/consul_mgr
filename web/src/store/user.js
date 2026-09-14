import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useUserStore = defineStore('user', () => {
  // 状态
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref(JSON.parse(localStorage.getItem('userInfo') || 'null'))

  // 计算属性
  const isLoggedIn = computed(() => !!token.value)
  const username = computed(() => userInfo.value?.name || '')
  const displayName = computed(() => userInfo.value?.display_name || userInfo.value?.name || '')
  const email = computed(() => userInfo.value?.email || '')
  const avatar = computed(() => userInfo.value?.avatar || '')
  const isAdmin = computed(() => userInfo.value?.is_admin || false)
  const roles = computed(() => userInfo.value?.roles || [])

  /**
   * 设置 Token
   */
  const setToken = (newToken) => {
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  /**
   * 设置用户信息
   */
  const setUser = (info) => {
    userInfo.value = info
    localStorage.setItem('userInfo', JSON.stringify(info))
  }

  /**
   * 设置 Token 和用户信息
   */
  const setAuth = (newToken, info) => {
    setToken(newToken)
    setUser(info)
  }

  /**
   * 清除认证信息
   */
  const clearAuth = () => {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('userInfo')
  }

  /**
   * 检查是否有指定角色
   */
  const hasRole = (role) => {
    return roles.value.includes(role)
  }

  /**
   * 检查是否有任一角色
   */
  const hasAnyRole = (...roleList) => {
    return roleList.some(role => roles.value.includes(role))
  }

  /**
   * 登出
   */
  const logout = async () => {
    try {
      // 调用后端登出接口
      await fetch('/api/auth/logout', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token.value}`
        }
      })
    } catch (error) {
      console.error('登出请求失败:', error)
    } finally {
      // 无论接口成功与否，都清除本地信息
      clearAuth()
    }
  }

  return {
    // 状态
    token,
    userInfo,
    
    // 计算属性
    isLoggedIn,
    username,
    displayName,
    email,
    avatar,
    isAdmin,
    roles,
    
    // 方法
    setToken,
    setUser,
    setAuth,
    clearAuth,
    hasRole,
    hasAnyRole,
    logout
  }
})
