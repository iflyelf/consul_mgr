import { defineStore } from 'pinia'
import { ref } from 'vue'

// 三套主题：warm(暖沙米·默认) / cool(冷蓝) / dark(暗黑)
export const THEMES = [
  { value: 'warm', label: '暖沙米', icon: 'Sunny' },
  { value: 'cool', label: '冷蓝', icon: 'MostlyCloudy' },
  { value: 'dark', label: '暗黑', icon: 'Moon' }
]

const VALID = THEMES.map(t => t.value)

// 兼容历史主题名
const LEGACY_MAP = {
  light: 'warm',
  blue: 'cool'
}

function resolveInitialTheme() {
  let theme = localStorage.getItem('consul_mgr_theme') || localStorage.getItem('theme')
  if (theme && LEGACY_MAP[theme]) {
    theme = LEGACY_MAP[theme]
  }
  if (!theme || !VALID.includes(theme)) {
    // 首次访问：按系统深色偏好选择
    theme = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'warm'
  }
  return theme
}

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref(resolveInitialTheme())

  const setTheme = (theme) => {
    if (!VALID.includes(theme)) {
      theme = 'warm'
    }
    currentTheme.value = theme
    localStorage.setItem('consul_mgr_theme', theme)
    document.documentElement.setAttribute('data-theme', theme)
    // 同步浏览器地址栏/移动端主题色
    const meta = document.querySelector('meta[name="theme-color"]')
    if (meta) {
      const bg = getComputedStyle(document.body).backgroundColor
      meta.setAttribute('content', bg)
    }
  }

  // 初始化
  setTheme(currentTheme.value)

  return {
    currentTheme,
    setTheme,
    themes: THEMES
  }
})
