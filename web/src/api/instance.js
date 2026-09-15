import request from '@/utils/request'

// 获取实例列表
export function getInstances(params) {
  return request({
    url: '/instances',
    method: 'get',
    params
  })
}

// 注册实例
export function registerInstance(data) {
  return request({
    url: '/instances',
    method: 'post',
    data
  })
}

// 更新实例
export function updateInstance(params, data) {
  return request({
    url: '/instances',
    method: 'put',
    params,
    data
  })
}

// 删除实例
export function deleteInstance(params) {
  return request({
    url: '/instances',
    method: 'delete',
    params
  })
}

// 批量删除实例
export function batchDeleteInstances(data) {
  return request({
    url: '/instances/batch-delete',
    method: 'post',
    data
  })
}

// 批量注册实例（支持 IP 段/CIDR/范围/IP:端口）
export function batchRegisterInstances(data) {
  return request({
    url: '/instances/batch-register',
    method: 'post',
    data
  })
}

// 预览批量注册
export function previewBatchRegister(data) {
  return request({
    url: '/instances/batch-register/preview',
    method: 'post',
    data
  })
}

// 导出实例
export function exportInstances(params) {
  return request({
    url: '/instances/export',
    method: 'get',
    params,
    responseType: 'blob'
  })
}

// 导入实例
export function importInstances(data) {
  return request({
    url: '/instances/import',
    method: 'post',
    data,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}
