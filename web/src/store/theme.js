import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref(localStorage.getItem('theme') || 'light')

  const setTheme = (theme) => {
    currentTheme.value = theme
    localStorage.setItem('theme', theme)
    document.documentElement.setAttribute('data-theme', theme)
  }

  // 初始化主题
  setTheme(currentTheme.value)

  return {
    currentTheme,
    setTheme
  }
})
