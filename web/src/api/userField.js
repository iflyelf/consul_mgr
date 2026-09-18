import request from '@/utils/request'

// 用户字段定义列表
export function listUserFields() {
  return request({ url: '/user-fields', method: 'get' })
}

// 新增字段定义
export function createUserField(data) {
  return request({ url: '/user-fields', method: 'post', data })
}

// 更新字段定义
export function updateUserField(id, data) {
  return request({ url: `/user-fields/${id}`, method: 'put', data })
}

// 删除字段定义
export function deleteUserField(id) {
  return request({ url: `/user-fields/${id}`, method: 'delete' })
}

// 立即从 FlyIAM 同步字段定义
export function syncUserFields() {
  return request({ url: '/user-fields/sync', method: 'post' })
}

// 自动同步配置
export function getSyncConfig() {
  return request({ url: '/user-fields/sync/config', method: 'get' })
}

export function updateSyncConfig(data) {
  return request({ url: '/user-fields/sync/config', method: 'put', data })
}

// 同步进度（轮询）
export function getSyncProgress() {
  return request({ url: '/user-fields/sync/progress', method: 'get' })
}

// 同步日志
export function getSyncLogs(limit = 20) {
  return request({ url: '/user-fields/sync/logs', method: 'get', params: { limit } })
}
