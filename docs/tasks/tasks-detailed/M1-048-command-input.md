# M1-048: 实现 CommandInput 命令输入组件

**任务ID**: M1-048
**标题**: 实现 CommandInput 命令输入组件
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M0-011

---

## 任务描述

实现命令输入组件，支持斜杠命令解析、自动补全、命令历史等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-048-01 | 设计输入框 UI | UI 设计 | 20min |
| M1-048-02 | 实现命令解析 | Parser | 30min |
| M1-048-03 | 实现自动补全 | Autocomplete | 30min |
| M1-048-04 | 实现命令历史 | History | 25min |
| M1-048-05 | 实现快捷提示 | Suggestions | 20min |
| M1-048-06 | 集成 API 调用 | API Integration | 15min |
| M1-048-07 | 编写组件测试 | 测试覆盖 | 10min |

---

## 命令输入组件

```tsx
// frontend/src/components/game/CommandInput.tsx
import { useState, useRef, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Command, Slash } from 'lucide-react'
import { useCommandParser } from '@/hooks/useCommandParser'

interface CommandInputProps {
  onCommand?: (command: string, args: string[]) => void
  roomId: string
}

export function CommandInput({ onCommand, roomId }: CommandInputProps) {
  const [input, setInput] = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const [showSuggestions, setShowSuggestions] = useState(false)

  const inputRef = useRef<HTMLInputElement>(null)

  const { parse, validate } = useCommandParser()

  const commands = [
    '/roll', '/r', '/check', '/c',
    '/attack', '/damage', '/heal',
    '/san', '/sancheck',
    '/character', '/char', '/switch',
    '/help', '/status', '/clear',
  ]

  useEffect(() => {
    // 监听快捷键
    const handleKeyDown = (e: KeyboardEvent) => {
      // 按 / 键聚焦输入框
      if (e.key === '/' && document.activeElement !== inputRef.current) {
        e.preventDefault()
        inputRef.current?.focus()
      }

      // ESC 关闭建议
      if (e.key === 'Escape') {
        setShowSuggestions(false)
        setSelectedIndex(-1)
      }

      // 方向键选择建议
      if (showSuggestions) {
        if (e.key === 'ArrowDown') {
          e.preventDefault()
          setSelectedIndex((i) =>
            i < suggestions.length - 1 ? i + 1 : i
          )
        } else if (e.key === 'ArrowUp') {
          e.preventDefault()
          setSelectedIndex((i) => (i > 0 ? i - 1 : -1))
        } else if (e.key === 'Enter' && selectedIndex >= 0) {
          e.preventDefault()
          setInput(suggestions[selectedIndex])
          setShowSuggestions(false)
          setSelectedIndex(-1)
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [suggestions, showSuggestions, selectedIndex])

  const handleChange = (value: string) => {
    setInput(value)

    // 显示建议
    if (value.startsWith('/')) {
      const matches = commands.filter(cmd =>
        cmd.toLowerCase().startsWith(value.toLowerCase())
      )
      setSuggestions(matches)
      setShowSuggestions(matches.length > 0)
      setSelectedIndex(-1)
    } else {
      setShowSuggestions(false)
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!input.trim()) return

    // 解析命令
    const result = parse(input)

    if (result.isValid) {
      onCommand?.(result.command, result.args)
      setInput('')
      setShowSuggestions(false)
      setSelectedIndex(-1)
    } else {
      // 显示错误
      console.error('Invalid command:', result.error)
    }
  }

  const handleSuggestionClick = (suggestion: string) => {
    setInput(suggestion)
    setShowSuggestions(false)
    inputRef.current?.focus()
  }

  return (
    <div className="relative">
      <form onSubmit={handleSubmit} className="relative">
        <div className="flex items-center space-x-2">
          <div className="relative flex-1">
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
              <Slash className="h-4 w-4" />
            </div>
            <Input
              ref={inputRef}
              value={input}
              onChange={(e) => handleChange(e.target.value)}
              placeholder="输入命令... (按 / 开始)"
              className="pl-10"
              autoComplete="off"
            />
          </div>

          <Button type="submit" size="icon">
            <Command className="h-4 w-4" />
          </Button>
        </div>

        {/* 命令建议 */}
        {showSuggestions && (
          <Card className="absolute top-full left-0 right-0 mt-2 z-10 max-h-48 overflow-y-auto">
            <div className="p-2 space-y-1">
              {suggestions.map((suggestion, index) => (
                <div
                  key={suggestion}
                  className={`px-3 py-2 rounded cursor-pointer text-sm ${
                    index === selectedIndex
                      ? 'bg-primary text-primary-foreground'
                      : 'hover:bg-muted'
                  }`}
                  onClick={() => handleSuggestionClick(suggestion)}
                >
                  {suggestion}
                </div>
              ))}
            </div>
          </Card>
        )}
      </form>
    </div>
  )
}
```

---

## 命令解析 Hook

```tsx
// frontend/src/hooks/useCommandParser.ts
import { useCallback } from 'react'
import { useCommandRegex } from '@/hooks/useCommandRegex'

interface ParseResult {
  isValid: boolean
  command: string
  args: string[]
  error?: string
}

export function useCommandParser() {
  const { patterns } = useCommandRegex()

  const parse = useCallback((input: string): ParseResult => {
    if (!input.startsWith('/')) {
      return {
        isValid: false,
        command: '',
        args: [],
        error: '命令必须以 / 开头',
      }
    }

    // 匹配各种命令模式
    for (const [name, pattern] of Object.entries(patterns)) {
      const match = input.match(new RegExp(`^${pattern}`)))
      if (match) {
        // 提取参数
        const args = match.slice(1).filter(Boolean)

        return {
          isValid: true,
          command: name,
          args: args,
        }
      }
    }

    return {
      isValid: false,
      command: '',
      args: [],
      error: '未知命令',
    }
  }, [patterns])

  const validate = useCallback((command: string, args: string[]): boolean => {
    // 验证命令和参数
    return true
  }, [])

  return { parse, validate }
}
```

---

## 命令正则配置

```tsx
// frontend/src/config/commands.ts
export const COMMAND_PATTERNS = {
  // /roll 1d100
  'roll': '\\/roll(?:|r)\\s+(.+)',

  // /check 侦查
  'check': '\\/(?:check|c)\\s+([\\w\\u4e00-\\u9fa5]+)(?:\\s+([+-]\\d+))?',

  // /attack 怪物A 1d6+2
  'attack': '\\/attack(?:\\s+([^\\s]+))?(?:\\s+(\\d+d\\d+(?:[+-]\\d+)?))?',

  // /damage 怪物A 8 穿刺
  'damage': '\\/damage\\s+([^\\s]+)\\s+(\\d+d\\d+(?:[+-]\\d+)?|\\d+)(?:\\s+([^\\s]+))?',

  // /heal 5
  'heal': '\\/heal\\s+(?:([^\\s]+)\\s+)?(\\d+)',

  // /san 1d6
  'san': '\\/san\\s+(\\d+d\\d+(?:[+-]\\d+)?|\\d+)(?:\\s*\\/\\s*(\\d+d\\d+(?:[+-]\\d+)?|\\d+))?(?:\\s+(.+))?',

  // /character list
  'character': '\\/(?:character|char)\\s+(list|view|create|edit|delete)',

  // /switch 张三
  'switch': '\\/(?:switch|use)\\s+(.+)',

  // /help
  'help': '\\/help(?:\\s+(.+))?',

  // /status
  'status': '\\/status',

  // /clear
  'clear': '\\/clear',
}
```

---

## 命令历史组件

```tsx
// frontend/src/components/game/CommandHistory.tsx
import { useState, useEffect } from 'react'

interface CommandHistoryProps {
  limit?: number
}

export function CommandHistory({ limit = 50 }: CommandHistoryProps) {
  const [history, setHistory] = useState<string[]>([])
  const [currentIndex, setCurrentIndex] = useState(-1)

  useEffect(() => {
    // 从本地存储加载历史
    const saved = localStorage.getItem('command_history')
    if (saved) {
      setHistory(JSON.parse(saved))
    }
  }, [])

  const addCommand = (command: string) => {
    setHistory(prev => {
      const newHistory = [command, ...prev].slice(0, limit)
      localStorage.setItem('command_history', JSON.stringify(newHistory))
      return newHistory
    })
    setCurrentIndex(-1)
  }

  const navigateUp = (): string | null => {
    if (history.length === 0) return null

    const newIndex = Math.min(currentIndex + 1, history.length - 1)
    setCurrentIndex(newIndex)
    return history[newIndex]
  }

  const navigateDown = (): string | null => {
    if (currentIndex < 0) return null

    const newIndex = Math.max(currentIndex - 1, -1)
    setCurrentIndex(newIndex)
    return newIndex >= 0 ? history[newIndex] : ''
  }

  return {
    history,
    currentIndex,
    addCommand,
    navigateUp,
    navigateDown,
  }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/CommandInput.tsx` | 创建 | 命令输入组件 |
| `frontend/src/hooks/useCommandParser.ts` | 创建 | 命令解析 Hook |
| `frontend/src/config/commands.ts` | 创建 | 命令配置 |
| `frontend/src/components/game/CommandHistory.tsx` | 创建 | 命令历史组件 |

---

## 验收标准

- [ ] 命令解析正确
- [ ] 自动补全有效
- [ ] 命令历史可用
- [ ] 快捷键正常
- [ ] 错误提示友好
- [ ] API 调用成功

---

## 参考文档

- M0-011: 命令参数正则表达式
- M0-001: 核心命令清单

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
