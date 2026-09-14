import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('access_token') || '')
  const userInfo = ref(JSON.parse(localStorage.getItem('user_info') || '{}'))

  const setToken = (newToken) => {
    token.value = newToken
    localStorage.setItem('access_token', newToken)
  }

  const setUserInfo = (info) => {
    userInfo.value = info
    localStorage.setItem('user_info', JSON.stringify(info))
  }

  const clearAuth = () => {
    token.value = ''
    userInfo.value = {}
    localStorage.removeItem('access_token')
    localStorage.removeItem('user_info')
  }

  return {
    token,
    userInfo,
    setToken,
    setUserInfo,
    clearAuth
  }
})
