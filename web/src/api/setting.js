import request from '@/utils/request'

// 获取全部可配置项
export function listSettings() {
  return request({ url: '/settings', method: 'get' })
}

// 更新设置（写 DB 并即时生效，无需重启）
export function updateSettings(settings) {
  return request({ url: '/settings', method: 'put', data: { settings } })
}
