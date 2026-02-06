# M6-005: 实现主题系统

**任务ID**: M6-005
**标题**: 实现主题系统
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M0-039

---

## 任务描述

实现主题切换系统，支持亮色/暗色主题，以及未来扩展的自定义主题功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-005-01 | 设计主题系统架构 | Architecture | 20min |
| M6-005-02 | 实现 Tailwind 主题配置 | Tailwind Config | 25min |
| M6-005-03 | 实现主题 Context | Theme Context | 25min |
| M6-005-04 | 实现主题切换组件 | Theme Switcher | 25min |
| M6-005-05 | 实现主题持久化 | Persistence | 15min |
| M6-005-06 | 实现组件主题适配 | Component Theming | 20min |
| M6-005-07 | 编写主题测试 | 测试覆盖 | 10min |

---

## 主题系统架构

```typescript
// frontend/src/lib/themes/index.ts
export interface Theme {
  name: string
  type: 'light' | 'dark'
  colors: ThemeColors
}

export interface ThemeColors {
  // 主色调
  primary: string
  primaryForeground: string

  // 次要色调
  secondary: string
  secondaryForeground: string

  // 背景
  background: string
  foreground: string

  // 卡片
  card: string
  cardForeground: string

  // 输入
  input: string
  ring: string

  // 边框
  border: string

  // 危险/警告/成功
  destructive: string
  destructiveForeground: string
  warning: string
  success: string

  // 中性色
  muted: string
  mutedForeground: string

  // 附加色
  accent: string
  accentForeground: string

  // 弹出层
  popover: string
  popoverForeground: string
}

export const themes: Record<string, Theme> = {
  light: {
    name: 'Light',
    type: 'light',
    colors: {
      primary: 'hsl(222.2 47.4% 11.2%)',
      primaryForeground: 'hsl(210 40% 98%)',
      secondary: 'hsl(210 40% 96.1%)',
      secondaryForeground: 'hsl(222.2 47.4% 11.2%)',
      background: 'hsl(0 0% 100%)',
      foreground: 'hsl(222.2 84% 4.9%)',
      card: 'hsl(0 0% 100%)',
      cardForeground: 'hsl(222.2 84% 4.9%)',
      input: 'hsl(214.3 31.8% 91.4%)',
      ring: 'hsl(222.2 84% 4.9%)',
      border: 'hsl(214.3 31.8% 91.4%)',
      destructive: 'hsl(0 84.2% 60.2%)',
      destructiveForeground: 'hsl(210 40% 98%)',
      warning: 'hsl(38 92% 50%)',
      success: 'hsl(142 76% 36%)',
      muted: 'hsl(210 40% 96.1%)',
      mutedForeground: 'hsl(215.4 16.3% 46.9%)',
      accent: 'hsl(210 40% 96.1%)',
      accentForeground: 'hsl(222.2 47.4% 11.2%)',
      popover: 'hsl(0 0% 100%)',
      popoverForeground: 'hsl(222.2 84% 4.9%)',
    },
  },
  dark: {
    name: 'Dark',
    type: 'dark',
    colors: {
      primary: 'hsl(217.2 91.2% 59.8%)',
      primaryForeground: 'hsl(222.2 47.4% 11.2%)',
      secondary: 'hsl(217.2 32.6% 17.5%)',
      secondaryForeground: 'hsl(210 40% 98%)',
      background: 'hsl(222.2 84% 4.9%)',
      foreground: 'hsl(210 40% 98%)',
      card: 'hsl(217.2 32.6% 17.5%)',
      cardForeground: 'hsl(210 40% 98%)',
      input: 'hsl(217.2 32.6% 17.5%)',
      ring: 'hsl(212.7 26.8% 83.9%)',
      border: 'hsl(217.2 32.6% 17.5%)',
      destructive: 'hsl(0 62.8% 30.6%)',
      destructiveForeground: 'hsl(210 40% 98%)',
      warning: 'hsl(38 92% 50%)',
      success: 'hsl(142 71% 45%)',
      muted: 'hsl(217.2 32.6% 17.5%)',
      mutedForeground: 'hsl(215 20.2% 65.1%)',
      accent: 'hsl(217.2 32.6% 17.5%)',
      accentForeground: 'hsl(210 40% 98%)',
      popover: 'hsl(217.2 32.6% 17.5%)',
      popoverForeground: 'hsl(210 40% 98%)',
    },
  },
  sepia: {
    name: 'Sepia',
    type: 'light',
    colors: {
      primary: 'hsl(30 60% 40%)',
      primaryForeground: 'hsl(0 0% 98%)',
      secondary: 'hsl(30 30% 85%)',
      secondaryForeground: 'hsl(30 60% 20%)',
      background: 'hsl(30 40% 96%)',
      foreground: 'hsl(30 60% 15%)',
      card: 'hsl(30 30% 92%)',
      cardForeground: 'hsl(30 60% 15%)',
      input: 'hsl(30 20% 90%)',
      ring: 'hsl(30 60% 40%)',
      border: 'hsl(30 20% 85%)',
      destructive: 'hsl(0 60% 45%)',
      destructiveForeground: 'hsl(0 0% 98%)',
      warning: 'hsl(38 92% 50%)',
      success: 'hsl(142 60% 40%)',
      muted: 'hsl(30 20% 88%)',
      mutedForeground: 'hsl(30 40% 35%)',
      accent: 'hsl(30 30% 85%)',
      accentForeground: 'hsl(30 60% 20%)',
      popover: 'hsl(30 30% 92%)',
      popoverForeground: 'hsl(30 60% 15%)',
    },
  },
}
```

---

## Tailwind 配置

```javascript
// frontend/tailwind.config.js
import type { Config } from 'tailwindcss'

const config: Config = {
  darkMode: ['class'],
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        warning: 'hsl(var(--warning))',
        success: 'hsl(var(--success))',
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}

export default config
```

---

## CSS 变量

```css
/* frontend/src/styles/globals.css */
@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --card: 0 0% 100%;
    --card-foreground: 222.2 84% 4.9%;
    --popover: 0 0% 100%;
    --popover-foreground: 222.2 84% 4.9%;
    --primary: 222.2 47.4% 11.2%;
    --primary-foreground: 210 40% 98%;
    --secondary: 210 40% 96.1%;
    --secondary-foreground: 222.2 47.4% 11.2%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;
    --accent: 210 40% 96.1%;
    --accent-foreground: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 210 40% 98%;
    --border: 214.3 31.8% 91.4%;
    --input: 214.3 31.8% 91.4%;
    --ring: 222.2 84% 4.9%;
    --warning: 38 92% 50%;
    --success: 142 76% 36%;
    --radius: 0.5rem;
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --card: 217.2 32.6% 17.5%;
    --card-foreground: 210 40% 98%;
    --popover: 217.2 32.6% 17.5%;
    --popover-foreground: 210 40% 98%;
    --primary: 217.2 91.2% 59.8%;
    --primary-foreground: 222.2 47.4% 11.2%;
    --secondary: 217.2 32.6% 17.5%;
    --secondary-foreground: 210 40% 98%;
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
    --accent: 217.2 32.6% 17.5%;
    --accent-foreground: 210 40% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 210 40% 98%;
    --border: 217.2 32.6% 17.5%;
    --input: 217.2 32.6% 17.5%;
    --ring: 212.7 26.8% 83.9%;
    --warning: 38 92% 50%;
    --success: 142 71% 45%;
  }

  .sepia {
    --background: 30 40% 96%;
    --foreground: 30 60% 15%;
    --card: 30 30% 92%;
    --card-foreground: 30 60% 15%;
    --popover: 30 30% 92%;
    --popover-foreground: 30 60% 15%;
    --primary: 30 60% 40%;
    --primary-foreground: 0 0% 98%;
    --secondary: 30 30% 85%;
    --secondary-foreground: 30 60% 20%;
    --muted: 30 20% 88%;
    --muted-foreground: 30 40% 35%;
    --accent: 30 30% 85%;
    --accent-foreground: 30 60% 20%;
    --destructive: 0 60% 45%;
    --destructive-foreground: 0 0% 98%;
    --border: 30 20% 85%;
    --input: 30 20% 90%;
    --ring: 30 60% 40%;
    --warning: 38 92% 50%;
    --success: 142 60% 40%;
  }
}
```

---

## 主题 Context

```tsx
// frontend/src/contexts/ThemeContext.tsx
import { createContext, useContext, useEffect, useState } from 'react'
import type { Theme, ThemeColors } from '@/lib/themes'

interface ThemeContextValue {
  theme: string
  themeType: 'light' | 'dark'
  colors: ThemeColors
  setTheme: (theme: string) => void
  toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider')
  }
  return context
}

interface ThemeProviderProps {
  children: React.ReactNode
  defaultTheme?: string
  storageKey?: string
}

export function ThemeProvider({
  children,
  defaultTheme = 'light',
  storageKey = 'monika-theme',
}: ThemeProviderProps) {
  const [theme, setThemeState] = useState<string>(defaultTheme)

  useEffect(() => {
    // 从 localStorage 读取
    const stored = localStorage.getItem(storageKey)
    if (stored) {
      setThemeState(stored)
    }
  }, [storageKey])

  useEffect(() => {
    // 应用主题到 DOM
    const root = document.documentElement

    // 移除所有主题类
    root.classList.remove('light', 'dark', 'sepia')

    // 添加当前主题类
    const currentTheme = themes[theme]
    root.classList.add(currentTheme.type)

    // 设置 CSS 变量
    const colors = currentTheme.colors
    Object.entries(colors).forEach(([key, value]) => {
      const cssVar = `--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`
      root.style.setProperty(cssVar, value)
    })

    // 保存到 localStorage
    localStorage.setItem(storageKey, theme)
  }, [theme, storageKey])

  const setTheme = (newTheme: string) => {
    if (themes[newTheme]) {
      setThemeState(newTheme)
    }
  }

  const toggleTheme = () => {
    const currentTheme = themes[theme]
    const newType = currentTheme.type === 'light' ? 'dark' : 'light'
    const newTheme = Object.values(themes).find(t => t.type === newType)?.name.toLowerCase()
    if (newTheme) {
      setThemeState(newTheme)
    }
  }

  const currentTheme = themes[theme]

  return (
    <ThemeContext.Provider
      value={{
        theme,
        themeType: currentTheme.type,
        colors: currentTheme.colors,
        setTheme,
        toggleTheme,
      }}
    >
      {children}
    </ThemeContext.Provider>
  )
}
```

---

## 主题切换组件

```tsx
// frontend/src/components/game/ThemeSwitcher.tsx
import { Moon, Sun, Monitor } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/contexts/ThemeContext'

export function ThemeSwitcher() {
  const { theme, setTheme, toggleTheme } = useTheme()

  return (
    <div className="flex items-center space-x-2 border rounded-md p-1">
      <Button
        size="sm"
        variant={theme === 'light' ? 'default' : 'ghost'}
        onClick={() => setTheme('light')}
        title="亮色主题"
      >
        <Sun className="h-4 w-4" />
      </Button>
      <Button
        size="sm"
        variant={theme === 'dark' ? 'default' : 'ghost'}
        onClick={() => setTheme('dark')}
        title="暗色主题"
      >
        <Moon className="h-4 w-4" />
      </Button>
      <Button
        size="sm"
        variant={theme === 'sepia' ? 'default' : 'ghost'}
        onClick={() => setTheme('sepia')}
        title="复古主题"
      >
        <Monitor className="h-4 w-4" />
      </Button>
    </div>
  )
}
```

---

## 使用示例

```tsx
// frontend/src/App.tsx
import { ThemeProvider } from '@/contexts/ThemeContext'
import { ThemeSwitcher } from '@/components/game/ThemeSwitcher'

export default function App() {
  return (
    <ThemeProvider defaultTheme="dark">
      <div className="min-h-screen bg-background text-foreground">
        <header className="border-b">
          <div className="container flex items-center justify-between py-4">
            <h1 className="text-2xl font-bold">Monika</h1>
            <ThemeSwitcher />
          </div>
        </header>

        {/* 内容 */}
      </div>
    </ThemeProvider>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/themes/index.ts` | 创建 | 主题定义 |
| `frontend/src/contexts/ThemeContext.tsx` | 创建 | 主题 Context |
| `frontend/src/components/game/ThemeSwitcher.tsx` | 创建 | 主题切换组件 |
| `frontend/tailwind.config.js` | 修改 | Tailwind 配置 |
| `frontend/src/styles/globals.css` | 修改 | CSS 变量 |

---

## 验收标准

- [ ] 主题切换流畅
- [ ] 亮色/暗色模式正常
- [ ] 主题持久化有效
- [ ] 所有组件适配良好
- [ ] 自定义主题支持
- [ ] 系统主题同步

---

## 参考文档

- M0-039: UI 设计规范
- Tailwind CSS Dark Mode
- shadcn/ui Theming

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
