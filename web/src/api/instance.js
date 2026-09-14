import request from '@/utils/request'

// 获取实例列表
export function getInstances(params) {
  return request({
    url: '/api/instances',
    method: 'get',
    params
  })
}

// 注册实例
export function registerInstance(data) {
  return request({
    url: '/api/instances',
    method: 'post',
    data
  })
}

// 更新实例
export function updateInstance(params, data) {
  return request({
    url: '/api/instances',
    method: 'put',
    params,
    data
  })
}

// 删除实例
export function deleteInstance(params) {
  return request({
    url: '/api/instances',
    method: 'delete',
    params
  })
}

// 批量删除实例
export function batchDeleteInstances(data) {
  return request({
    url: '/api/instances/batch-delete',
    method: 'post',
    data
  })
}

// 导出实例
export function exportInstances(params) {
  return request({
    url: '/api/instances/export',
    method: 'get',
    params,
    responseType: 'blob'
  })
}

// 导入实例
export function importInstances(data) {
  return request({
    url: '/api/instances/import',
    method: 'post',
    data,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}
