import request from '@/utils/request'

// 获取服务组列表
export const getGroupList = (params) => {
  return request({
    url: '/groups',
    method: 'get',
    params
  })
}

// 别名：getGroups
export const getGroups = getGroupList

// 获取服务组详情
export const getGroupDetail = (id) => {
  return request({
    url: `/groups/${id}`,
    method: 'get'
  })
}

// 创建服务组
export const createGroup = (data) => {
  return request({
    url: '/groups',
    method: 'post',
    data
  })
}

// 更新服务组
export const updateGroup = (id, data) => {
  return request({
    url: `/groups/${id}`,
    method: 'put',
    data
  })
}

// 删除服务组
export const deleteGroup = (id) => {
  return request({
    url: `/groups/${id}`,
    method: 'delete'
  })
}

// 测试连接
export const testConnection = (id) => {
  return request({
    url: `/groups/${id}/test`,
    method: 'post'
  })
}
