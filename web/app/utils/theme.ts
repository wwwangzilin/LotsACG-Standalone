import { Themes, type StyleVars } from '@varlet/ui'

// 大气简约风格: 深色/浅色纯净底 + 单一靛蓝强调色 + 克制的同色系渐变。
// 自定义变量会随 StyleProvider 一并注入为 CSS 变量。

const blueLight = {
  '--hsl-primary': '221, 83%, 53%',
  '--hsl-primary-container': '214, 100%, 90%',
  '--hsl-on-primary-container': '224, 80%, 22%',
  '--hsl-info': '221, 70%, 45%',
  '--hsl-info-container': '214, 100%, 90%',
  '--hsl-on-info-container': '224, 80%, 22%',
  '--hsl-warning': '38, 92%, 45%',
  '--hsl-warning-container': '40, 100%, 92%',
  '--hsl-on-warning-container': '35, 80%, 20%',
  '--hsl-danger': '0, 72%, 45%',
  '--hsl-danger-container': '0, 100%, 93%',
  '--hsl-on-danger-container': '0, 80%, 22%'
}

const blueDark = {
  '--hsl-primary': '217, 91%, 66%',
  '--hsl-primary-container': '221, 45%, 26%',
  '--hsl-on-primary-container': '214, 100%, 88%',
  '--hsl-info': '217, 90%, 70%',
  '--hsl-info-container': '221, 40%, 24%',
  '--hsl-on-info-container': '214, 100%, 88%',
  '--hsl-warning': '38, 92%, 62%',
  '--hsl-warning-container': '38, 40%, 24%',
  '--hsl-on-warning-container': '40, 100%, 88%',
  '--hsl-danger': '0, 84%, 66%',
  '--hsl-danger-container': '0, 45%, 24%',
  '--hsl-on-danger-container': '0, 100%, 90%'
}

export const lightTheme: StyleVars = {
  ...Themes.md3Light,
  // ---- 自定义: 克制的蓝色渐变 & 玻璃面板 & 圆角 ----
  '--gradient-primary': 'linear-gradient(135deg, #1D4ED8 0%, #3B82F6 100%)',
  '--gradient-soft': 'linear-gradient(135deg, rgba(59, 130, 246, 0.10), rgba(147, 197, 253, 0.10))',
  '--glass-bg': 'rgba(255, 255, 255, 0.75)',
  '--glass-border': 'rgba(255, 255, 255, 0.65)',
  '--glass-blur': '16px',
  '--card-radius': '12px',
  '--card-shadow': '0 2px 12px rgba(15, 23, 42, 0.06), 0 1px 4px rgba(15, 23, 42, 0.04)',
  '--card-shadow-hover': '0 10px 28px rgba(15, 23, 42, 0.12), 0 3px 10px rgba(15, 23, 42, 0.08)',

  // ---- Varlet 主色: 靛蓝 ----
  '--hsl-primary': blueLight['--hsl-primary'],
  '--color-primary': 'hsla(var(--hsl-primary), 1)',
  '--hsl-on-primary': '0, 0%, 100%',
  '--color-on-primary': 'hsla(var(--hsl-on-primary), 1)',
  '--hsl-primary-container': blueLight['--hsl-primary-container'],
  '--color-primary-container': 'hsla(var(--hsl-primary-container), 1)',
  '--hsl-on-primary-container': blueLight['--hsl-on-primary-container'],
  '--color-on-primary-container': 'hsla(var(--hsl-on-primary-container), 1)',

  // ---- 辅助色 ----
  '--hsl-info': blueLight['--hsl-info'],
  '--color-info': 'hsla(var(--hsl-info), 1)',
  '--hsl-on-info': '0, 0%, 100%',
  '--color-on-info': 'hsla(var(--hsl-on-info), 1)',
  '--hsl-info-container': blueLight['--hsl-info-container'],
  '--color-info-container': 'hsla(var(--hsl-info-container), 1)',
  '--hsl-on-info-container': blueLight['--hsl-on-info-container'],
  '--color-on-info-container': 'hsla(var(--hsl-on-info-container), 1)',
  '--hsl-warning': blueLight['--hsl-warning'],
  '--color-warning': 'hsla(var(--hsl-warning), 1)',
  '--hsl-on-warning': '0, 0%, 100%',
  '--color-on-warning': 'hsla(var(--hsl-on-warning), 1)',
  '--hsl-warning-container': blueLight['--hsl-warning-container'],
  '--color-warning-container': 'hsla(var(--hsl-warning-container), 1)',
  '--hsl-on-warning-container': blueLight['--hsl-on-warning-container'],
  '--color-on-warning-container': 'hsla(var(--hsl-on-warning-container), 1)',
  '--hsl-danger': blueLight['--hsl-danger'],
  '--color-danger': 'hsla(var(--hsl-danger), 1)',
  '--hsl-on-danger': '0, 0%, 100%',
  '--color-on-danger': 'hsla(var(--hsl-on-danger), 1)',
  '--hsl-danger-container': blueLight['--hsl-danger-container'],
  '--color-danger-container': 'hsla(var(--hsl-danger-container), 1)',
  '--hsl-on-danger-container': blueLight['--hsl-on-danger-container'],
  '--color-on-danger-container': 'hsla(var(--hsl-on-danger-container), 1)',

  // ---- 表面 & 文字: 纯净浅底 ----
  '--hsl-body': '220, 25%, 97%',
  '--color-body': 'hsla(var(--hsl-body), 1)',
  '--hsl-surface-container': '0, 0%, 100%',
  '--color-surface-container': 'hsla(var(--hsl-surface-container), 1)',
  '--hsl-surface-container-low': '220, 25%, 98%',
  '--color-surface-container-low': 'hsla(var(--hsl-surface-container-low), 1)',
  '--hsl-surface-container-high': '220, 20%, 94%',
  '--color-surface-container-high': 'hsla(var(--hsl-surface-container-high), 1)',
  '--hsl-text': '222, 30%, 14%',
  '--color-text': 'hsla(var(--hsl-text), 1)',
  '--hsl-on-surface-variant': '220, 12%, 38%',
  '--color-on-surface-variant': 'hsla(var(--hsl-on-surface-variant), 1)',
  '--hsl-outline': '220, 10%, 60%',
  '--color-outline': 'hsla(var(--hsl-outline), 1)',
  '--hsl-inverse-surface': '222, 30%, 16%',
  '--color-inverse-surface': 'hsla(var(--hsl-inverse-surface), 1)'
}

export const darkTheme: StyleVars = {
  ...Themes.md3Dark,
  // ---- 自定义: 克制的蓝色渐变 & 玻璃面板 & 圆角 ----
  '--gradient-primary': 'linear-gradient(135deg, #2563EB 0%, #60A5FA 100%)',
  '--gradient-soft': 'linear-gradient(135deg, rgba(37, 99, 235, 0.14), rgba(96, 165, 250, 0.08))',
  '--glass-bg': 'rgba(16, 20, 28, 0.72)',
  '--glass-border': 'rgba(255, 255, 255, 0.07)',
  '--glass-blur': '16px',
  '--card-radius': '12px',
  '--card-shadow': '0 2px 12px rgba(0, 0, 0, 0.35)',
  '--card-shadow-hover': '0 10px 30px rgba(37, 99, 235, 0.18), 0 4px 12px rgba(0, 0, 0, 0.45)',

  // ---- Varlet 主色: 亮蓝 (暗色) ----
  '--hsl-primary': blueDark['--hsl-primary'],
  '--color-primary': 'hsla(var(--hsl-primary), 1)',
  '--hsl-on-primary': '220, 90%, 18%',
  '--color-on-primary': 'hsla(var(--hsl-on-primary), 1)',
  '--hsl-primary-container': blueDark['--hsl-primary-container'],
  '--color-primary-container': 'hsla(var(--hsl-primary-container), 1)',
  '--hsl-on-primary-container': blueDark['--hsl-on-primary-container'],
  '--color-on-primary-container': 'hsla(var(--hsl-on-primary-container), 1)',

  // ---- 辅助色 ----
  '--hsl-info': blueDark['--hsl-info'],
  '--color-info': 'hsla(var(--hsl-info), 1)',
  '--hsl-on-info': '220, 90%, 18%',
  '--color-on-info': 'hsla(var(--hsl-on-info), 1)',
  '--hsl-info-container': blueDark['--hsl-info-container'],
  '--color-info-container': 'hsla(var(--hsl-info-container), 1)',
  '--hsl-on-info-container': blueDark['--hsl-on-info-container'],
  '--color-on-info-container': 'hsla(var(--hsl-on-info-container), 1)',
  '--hsl-warning': blueDark['--hsl-warning'],
  '--color-warning': 'hsla(var(--hsl-warning), 1)',
  '--hsl-on-warning': '38, 90%, 16%',
  '--color-on-warning': 'hsla(var(--hsl-on-warning), 1)',
  '--hsl-warning-container': blueDark['--hsl-warning-container'],
  '--color-warning-container': 'hsla(var(--hsl-warning-container), 1)',
  '--hsl-on-warning-container': blueDark['--hsl-on-warning-container'],
  '--color-on-warning-container': 'hsla(var(--hsl-on-warning-container), 1)',
  '--hsl-danger': blueDark['--hsl-danger'],
  '--color-danger': 'hsla(var(--hsl-danger), 1)',
  '--hsl-on-danger': '0, 90%, 18%',
  '--color-on-danger': 'hsla(var(--hsl-on-danger), 1)',
  '--hsl-danger-container': blueDark['--hsl-danger-container'],
  '--color-danger-container': 'hsla(var(--hsl-danger-container), 1)',
  '--hsl-on-danger-container': blueDark['--hsl-on-danger-container'],
  '--color-on-danger-container': 'hsla(var(--hsl-on-danger-container), 1)',

  // ---- 表面 & 文字: 深空蓝黑底 + 深灰卡片 ----
  '--hsl-body': '222, 27%, 6%',
  '--color-body': 'hsla(var(--hsl-body), 1)',
  '--hsl-surface-container': '222, 20%, 10%',
  '--color-surface-container': 'hsla(var(--hsl-surface-container), 1)',
  '--hsl-surface-container-low': '222, 22%, 8%',
  '--color-surface-container-low': 'hsla(var(--hsl-surface-container-low), 1)',
  '--hsl-surface-container-high': '222, 18%, 13%',
  '--color-surface-container-high': 'hsla(var(--hsl-surface-container-high), 1)',
  '--hsl-surface-container-highest': '222, 16%, 17%',
  '--color-surface-container-highest': 'hsla(var(--hsl-surface-container-highest), 1)',
  '--hsl-text': '220, 20%, 92%',
  '--color-text': 'hsla(var(--hsl-text), 1)',
  '--hsl-on-surface-variant': '220, 12%, 70%',
  '--color-on-surface-variant': 'hsla(var(--hsl-on-surface-variant), 1)',
  '--hsl-outline': '220, 10%, 45%',
  '--color-outline': 'hsla(var(--hsl-outline), 1)',
  '--hsl-inverse-surface': '220, 20%, 90%',
  '--color-inverse-surface': 'hsla(var(--hsl-inverse-surface), 1)'
}
