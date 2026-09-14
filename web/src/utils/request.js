import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useUserStore } from '@/store/user'

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

// 请求拦截器
request.interceptors.request.use(
  config => {
    // 从 localStorage 获取 Token
    const token = localStorage.getItem('token')
    if (token) {
      // 添加 Authorization Header
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => {
    console.error('请求错误:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  response => {
    const res = response.data
    
    // 后端返回的数据格式：{ code: 200, message: "success", data: {...} }
    if (res.code === 200) {
      return res.data
    }
    
    // 其他状态码视为错误
    ElMessage.error(res.message || '请求失败')
    return Promise.reject(new Error(res.message || '请求失败'))
  },
  error => {
    console.error('响应错误:', error)
    
    // 处理 HTTP 状态码
    if (error.response) {
      const status = error.response.status
      const data = error.response.data
      
      switch (status) {
        case 401:
          // 未认证或 Token 过期
          ElMessage.error(data.message || '未认证，请重新登录')
          
          // 清除认证信息
          const userStore = useUserStore()
          userStore.clearAuth()
          
          // 跳转到登录页
          router.push('/login')
          break
          
        case 403:
          // 权限不足
          ElMessage.error(data.message || '权限不足')
          break
          
        case 404:
          ElMessage.error(data.message || '请求的资源不存在')
          break
          
        case 500:
          ElMessage.error(data.message || '服务器内部错误')
          break
          
        default:
          ElMessage.error(data.message || error.message || '网络错误')
      }
    } else if (error.request) {
      // 请求已发出，但没有收到响应
      ElMessage.error('网络错误，请检查网络连接')
    } else {
      // 请求配置出错
      ElMessage.error('请求配置错误: ' + error.message)
    }
    
    return Promise.reject(error)
  }
)

export default request
