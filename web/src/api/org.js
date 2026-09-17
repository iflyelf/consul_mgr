import request from '@/utils/request'

// ==================== 用户（来自 Casdoor，与 FlyIAM 对齐） ====================
export function getUsers(params) {
  return request({ url: '/users', method: 'get', params })
}

export function createUser(data) {
  return request({ url: '/users', method: 'post', data })
}

export function updateUser(name, data) {
  return request({ url: `/users/${encodeURIComponent(name)}`, method: 'put', data })
}

export function deleteUser(name) {
  return request({ url: `/users/${encodeURIComponent(name)}`, method: 'delete' })
}

export function resetUserPassword(name, password = '') {
  return request({ url: `/users/${encodeURIComponent(name)}/reset-password`, method: 'post', data: { password } })
}

export function setUserAdmin(name, isAdmin) {
  return request({ url: `/users/${encodeURIComponent(name)}/admin`, method: 'post', data: { isAdmin } })
}

// ==================== 角色 ====================
export function getRoles(params) {
  return request({ url: '/roles', method: 'get', params })
}

export function createRole(data) {
  return request({ url: '/roles', method: 'post', data })
}

export function updateRole(id, data) {
  return request({ url: `/roles/${id}`, method: 'put', data })
}

export function deleteRole(id) {
  return request({ url: `/roles/${id}`, method: 'delete' })
}

// ==================== 团队 ====================
export function getTeams(params) {
  return request({ url: '/teams', method: 'get', params })
}

export function createTeam(data) {
  return request({ url: '/teams', method: 'post', data })
}

export function updateTeam(id, data) {
  return request({ url: `/teams/${id}`, method: 'put', data })
}

export function deleteTeam(id) {
  return request({ url: `/teams/${id}`, method: 'delete' })
}

// ==================== 团队成员 ====================
export function getTeamMembers(id) {
  return request({ url: `/teams/${id}/members`, method: 'get' })
}

export function addTeamMember(id, data) {
  return request({ url: `/teams/${id}/members`, method: 'post', data })
}

export function removeTeamMember(id, userId) {
  return request({ url: `/teams/${id}/members`, method: 'delete', params: { user_id: userId } })
}

// ==================== 团队 → 服务组授权 ====================
export function getTeamPermissions(id) {
  return request({ url: `/teams/${id}/permissions`, method: 'get' })
}

export function grantTeamPermission(id, data) {
  return request({ url: `/teams/${id}/permissions`, method: 'post', data })
}

export function revokeTeamPermission(id, groupId) {
  return request({ url: `/teams/${id}/permissions`, method: 'delete', params: { group_id: groupId } })
}
