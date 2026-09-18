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

// 从 FlyIAM 同步字段定义
export function syncUserFields() {
  return request({ url: '/user-fields/sync', method: 'post' })
}
