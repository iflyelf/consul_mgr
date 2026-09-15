import request from '@/utils/request'

// 获取服务列表
export function getServices(params) {
  return request({
    url: '/services',
    method: 'get',
    params
  })
}

// 获取服务详情
export function getServiceDetail(params) {
  return request({
    url: '/services/detail',
    method: 'get',
    params
  })
}

// 删除服务
export function deleteService(params) {
  return request({
    url: '/services',
    method: 'delete',
    params
  })
}

// 批量删除服务
export function batchDeleteServices(data) {
  return request({
    url: '/services/batch-delete',
    method: 'post',
    data
  })
}
