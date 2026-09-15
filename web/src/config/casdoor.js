// Casdoor 回调处理
//
// 说明：
//   Casdoor 服务地址、Client ID、回调地址等均由后端按当前访问域名动态生成，
//   前端不再在构建期写死任何 Casdoor 配置，跨域名/IP 部署自动适配。
//
// 作者: iflyelf

/**
 * 处理登录回调
 *
 * 从 URL 中取出 code/state，交给后端换取 Token。
 * @returns {Promise<Object>} 登录结果，包含 access_token / user_info
 */
export async function handleCallback() {
  const params = new URLSearchParams(window.location.search)
  const code = params.get('code')
  const state = params.get('state')

  if (!code || !state) {
    throw new Error('缺少授权码或状态码')
  }

  const response = await fetch(`/api/auth/callback?code=${code}&state=${state}`)
  const result = await response.json()

  if (result.code !== 200) {
    throw new Error(result.message || '登录失败')
  }

  return result.data
}
