// Casdoor SDK 配置
//
// 功能：
//   - Casdoor SDK 初始化
//   - 获取登录 URL
//   - 处理 OAuth 回调
//   - 用户信息获取
//
// 作者: iflyelf
// 创建时间: 2024-01-15

import Sdk from 'casdoor-js-sdk'

// Casdoor 配置
export const casdoorConfig = {
  serverUrl: import.meta.env.VITE_CASDOOR_ENDPOINT || 'http://localhost:8000',
  clientId: import.meta.env.VITE_CASDOOR_CLIENT_ID || '',
  appName: import.meta.env.VITE_CASDOOR_APPLICATION || 'consul_manager',
  organizationName: import.meta.env.VITE_CASDOOR_ORGANIZATION || 'consul_mgr',
  redirectPath: '/callback'
}

// 创建 Casdoor SDK 实例
export const casdoorSdk = new Sdk(casdoorConfig)

/**
 * 获取登录 URL
 * @returns {string} Casdoor 登录 URL
 */
export function getSigninUrl() {
  const redirectUri = `${window.location.origin}${casdoorConfig.redirectPath}`
  return casdoorSdk.getSigninUrl(redirectUri)
}

/**
 * 获取注册 URL
 * @returns {string} Casdoor 注册 URL
 */
export function getSignupUrl() {
  const redirectUri = `${window.location.origin}${casdoorConfig.redirectPath}`
  return casdoorSdk.getSignupUrl(redirectUri)
}

/**
 * 处理登录回调
 * @returns {Promise<string>} Access Token
 */
export async function handleCallback() {
  // 获取 URL 参数
  const params = new URLSearchParams(window.location.search)
  const code = params.get('code')
  const state = params.get('state')
  
  if (!code || !state) {
    throw new Error('缺少授权码或状态码')
  }
  
  // 使用后端接口换取 Token
  // 注意：这里不使用 SDK 的 signin 方法，而是调用我们的后端接口
  const response = await fetch(`/api/auth/callback?code=${code}&state=${state}`)
  const result = await response.json()
  
  if (result.code !== 200) {
    throw new Error(result.message || '登录失败')
  }
  
  return result.data
}

/**
 * 获取登出 URL
 * @returns {string} Casdoor 登出 URL
 */
export function getSignoutUrl() {
  return casdoorSdk.getSignoutUrl(`${window.location.origin}/login`)
}

export default casdoorSdk
