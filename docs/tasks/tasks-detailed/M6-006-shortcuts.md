# M6-006: 实现快捷键系统

**任务ID**: M6-006
**标题**: 实现快捷键系统
**类型**: frontend (前端开发)
**预估工时**: 1.5h
**依赖**: 无

---

## 任务描述

实现全局快捷键系统，允许用户通过键盘快捷方式快速执行常用操作，提升操作效率。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-006-01 | 设计快捷键架构 | Architecture | 15min |
| M6-006-02 | 实现快捷键注册 | Registration | 25min |
| M6-006-03 | 实现快捷键处理 | Handler | 25min |
| M6-006-04 | 实现冲突检测 | Conflict Detection | 20min |
| M6-006-05 | 实现快捷键设置 UI | Settings UI | 25min |
| M6-006-06 | 实现快捷键提示 | Help Tooltip | 15min |
| M6-006-07 | 编写快捷键测试 | 测试覆盖 | 10min |

---

## 快捷键定义

```typescript
// frontend/src/lib/shortcuts/types.ts
export interface Shortcut {
  id: string
  name: string
  description: string
  defaultKey: string
  category: 'global' | 'chat' | 'dice' | 'combat' | 'general'
  action: () => void
  customizable: boolean
}

export interface ShortcutBinding {
  shortcutId: string
  key: string
  ctrl?: boolean
  alt?: boolean
  shift?: boolean
  meta?: boolean
}
```

---

## 快捷键管理器

```typescript
// frontend/src/lib/shortcuts/shortcut-manager.ts
type ShortcutHandler = (e: KeyboardEvent) => void

class ShortcutManager {
  private bindings: Map<string, ShortcutHandler> = new Map()
  private disabled: boolean = false

  constructor() {
    this.init()
  }

  private init() {
    document.addEventListener('keydown', this.handleKeyDown)
    this.loadCustomBindings()
  }

  private handleKeyDown = (e: KeyboardEvent) => {
    if (this.disabled) return

    // 忽略在输入框中的按键
    const target = e.target as HTMLElement
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.contentEditable === 'true'
    ) {
      return
    }

    const key = this.getKeyString(e)
    const handler = this.bindings.get(key)

    if (handler) {
      e.preventDefault()
      handler(e)
    }
  }

  private getKeyString(e: KeyboardEvent): string {
    const parts: string[] = []

    if (e.ctrlKey) parts.push('ctrl')
    if (e.altKey) parts.push('alt')
    if (e.shiftKey) parts.push('shift')
    if (e.metaKey) parts.push('meta')

    parts.push(e.key.toLowerCase())

    return parts.join('+')
  }

  register(key: string, handler: ShortcutHandler) {
    const normalizedKey = key.toLowerCase().replace(/\s+/g, '')
    this.bindings.set(normalizedKey, handler)
  }

  unregister(key: string) {
    const normalizedKey = key.toLowerCase().replace(/\s+/g, '')
    this.bindings.delete(normalizedKey)
  }

  disable() {
    this.disabled = true
  }

  enable() {
    this.disabled = false
  }

  loadCustomBindings() {
    const custom = localStorage.getItem('custom-shortcuts')
    if (custom) {
      // TODO: 应用自定义快捷键
    }
  }

  saveCustomBinding(shortcutId: string, key: string) {
    const custom = JSON.parse(localStorage.getItem('custom-shortcuts') || '{}')
    custom[shortcutId] = key
    localStorage.setItem('custom-shortcuts', JSON.stringify(custom))
  }

  destroy() {
    document.removeEventListener('keydown', this.handleKeyDown)
  }
}

export const shortcutManager = new ShortcutManager()
```

---

## 默认快捷键配置

```typescript
// frontend/src/lib/shortcuts/default-shortcuts.ts
import { Shortcut } from './types'

export const DEFAULT_SHORTCUTS: Shortcut[] = [
  // 全局快捷键
  {
    id: 'toggle-sidebar',
    name: '切换侧边栏',
    description: '显示/隐藏侧边栏',
    defaultKey: 'ctrl+b',
    category: 'global',
    action: () => {
      // TODO: 切换侧边栏
      console.log('Toggle sidebar')
    },
    customizable: true,
  },
  {
    id: 'toggle-settings',
    name: '打开设置',
    description: '打开设置面板',
    defaultKey: 'ctrl+,',
    category: 'global',
    action: () => {
      // TODO: 打开设置
      console.log('Open settings')
    },
    customizable: true,
  },
  {
    id: 'toggle-shortcuts-help',
    name: '快捷键帮助',
    description: '显示快捷键列表',
    defaultKey: 'ctrl+/',
    category: 'global',
    action: () => {
      // TODO: 显示帮助
      console.log('Show shortcuts help')
    },
    customizable: false,
  },

  // 聊天快捷键
  {
    id: 'focus-chat-input',
    name: '聚焦聊天输入',
    description: '将焦点移到聊天输入框',
    defaultKey: 'ctrl+/',
    category: 'chat',
    action: () => {
      document.getElementById('chat-input')?.focus()
    },
    customizable: true,
  },
  {
    id: 'send-message',
    name: '发送消息',
    description: '发送聊天消息',
    defaultKey: 'enter',
    category: 'chat',
    action: () => {
      // TODO: 发送消息
      console.log('Send message')
    },
    customizable: false,
  },

  // 掷骰快捷键
  {
    id: 'quick-roll-d100',
    name: '快速掷 d100',
    description: '立即执行 d100 掷骰',
    defaultKey: 'ctrl+r',
    category: 'dice',
    action: () => {
      // TODO: 掷骰
      console.log('Roll d100')
    },
    customizable: true,
  },
  {
    id: 'quick-roll-d20',
    name: '快速掷 d20',
    description: '立即执行 d20 掷骰',
    defaultKey: 'ctrl+shift+r',
    category: 'dice',
    action: () => {
      // TODO: 掷骰
      console.log('Roll d20')
    },
    customizable: true,
  },

  // 战斗快捷键
  {
    id: 'next-turn',
    name: '下一回合',
    description: '推进到下一个战斗回合',
    defaultKey: 'ctrl+t',
    category: 'combat',
    action: () => {
      // TODO: 下一回合
      console.log('Next turn')
    },
    customizable: true,
  },
  {
    id: 'toggle-combat',
    name: '切换战斗模式',
    description: '开始或结束战斗',
    defaultKey: 'ctrl+shift+t',
    category: 'combat',
    action: () => {
      // TODO: 切换战斗
      console.log('Toggle combat')
    },
    customizable: true,
  },

  // 通用快捷键
  {
    id: 'undo',
    name: '撤销',
    description: '撤销上一个操作',
    defaultKey: 'ctrl+z',
    category: 'general',
    action: () => {
      // TODO: 撤销
      console.log('Undo')
    },
    customizable: false,
  },
  {
    id: 'redo',
    name: '重做',
    description: '重做上一个撤销的操作',
    defaultKey: 'ctrl+y',
    category: 'general',
    action: () => {
      // TODO: 重做
      console.log('Redo')
    },
    customizable: false,
  },
  {
    id: 'save',
    name: '保存',
    description: '保存当前状态',
    defaultKey: 'ctrl+s',
    category: 'general',
    action: () => {
      // TODO: 保存
      console.log('Save')
    },
    customizable: false,
  },
  {
    id: 'find',
    name: '搜索',
    description: '在当前页面搜索',
    defaultKey: 'ctrl+f',
    category: 'general',
    action: () => {
      // TODO: 搜索
      console.log('Find')
    },
    customizable: false,
  },
]
```

---

## 快捷键 Hook

```typescript
// frontend/src/hooks/useShortcuts.ts
import { useEffect } from 'react'
import { shortcutManager } from '@/lib/shortcuts/shortcut-manager'

export interface ShortcutConfig {
  key: string
  handler: (e: KeyboardEvent) => void
  disabled?: boolean
}

export function useShortcuts(shortcuts: ShortcutConfig[]) {
  useEffect(() => {
    shortcuts.forEach(({ key, handler }) => {
      shortcutManager.register(key, handler)
    })

    return () => {
      shortcuts.forEach(({ key }) => {
        shortcutManager.unregister(key)
      })
    }
  }, [shortcuts])
}

// 使用示例
export function useChatShortcuts(onSend: () => void) {
  return useShortcuts([
    {
      key: 'enter',
      handler: (e) => {
        if (!(e.target instanceof HTMLElement) || e.target.tagName !== 'TEXTAREA') {
          return
        }
        onSend()
      },
    },
  ])
}
```

---

## 快捷键设置组件

```tsx
// frontend/src/components/settings/ShortcutSettings.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Keyboard, RefreshCw } from 'lucide-react'
import { DEFAULT_SHORTCUTS } from '@/lib/shortcuts/default-shortcuts'
import { shortcutManager } from '@/lib/shortcuts/shortcut-manager'
import { useToast } from '@/hooks/use-toast'

interface ShortcutBinding {
  shortcutId: string
  currentKey: string
}

export function ShortcutSettings() {
  const [bindings, setBindings] = useState<ShortcutBinding[]>([])
  const [recording, setRecording] = useState<string | null>(null)
  const [pendingKey, setPendingKey] = useState<string>('')

  const { toast } = useToast()

  useEffect(() => {
    loadBindings()
  }, [])

  const loadBindings = () => {
    const custom = JSON.parse(localStorage.getItem('custom-shortcuts') || '{}')

    const loadedBindings = DEFAULT_SHORTCUTS.map(shortcut => ({
      shortcutId: shortcut.id,
      currentKey: custom[shortcut.id] || shortcut.defaultKey,
    }))

    setBindings(loadedBindings)
  }

  const handleRecordStart = (shortcutId: string) => {
    setRecording(shortcutId)
    setPendingKey('')

    const handleKeyDown = (e: KeyboardEvent) => {
      e.preventDefault()
      e.stopPropagation()

      const key = shortcutManager.getKeyString(e)
      setPendingKey(key)

      // 自动完成录制
      setTimeout(() => {
        handleRecordEnd(shortcutId, key)
      }, 100)
    }

    document.addEventListener('keydown', handleKeyDown, { once: true })
  }

  const handleRecordEnd = (shortcutId: string, key: string) => {
    if (!key) return

    // 检查冲突
    const conflict = bindings.find(b => b.currentKey === key && b.shortcutId !== shortcutId)
    if (conflict) {
      const conflictShortcut = DEFAULT_SHORTCUTS.find(s => s.id === conflict.shortcutId)
      toast({
        title: '快捷键冲突',
        description: `此快捷键已被 "${conflictShortcut?.name}" 使用`,
        variant: 'destructive',
      })
      return
    }

    // 保存自定义快捷键
    shortcutManager.saveCustomBinding(shortcutId, key)

    // 更新本地状态
    setBindings(prev =>
      prev.map(b =>
        b.shortcutId === shortcutId ? { ...b, currentKey: key } : b
      )
    )

    setRecording(null)
    setPendingKey('')

    toast({
      title: '快捷键已更新',
      description: `已将快捷键设置为 ${key}`,
    })
  }

  const handleReset = (shortcutId: string) => {
    const shortcut = DEFAULT_SHORTCUTS.find(s => s.id === shortcutId)
    if (!shortcut) return

    // 删除自定义快捷键
    const custom = JSON.parse(localStorage.getItem('custom-shortcuts') || '{}')
    delete custom[shortcutId]
    localStorage.setItem('custom-shortcuts', JSON.stringify(custom))

    // 更新本地状态
    setBindings(prev =>
      prev.map(b =>
        b.shortcutId === shortcutId ? { ...b, currentKey: shortcut.defaultKey } : b
      )
    )

    toast({
      title: '已恢复默认快捷键',
    })
  }

  const handleResetAll = () => {
    localStorage.removeItem('custom-shortcuts')
    loadBindings()
    toast({
      title: '已恢复所有默认快捷键',
    })
  }

  const groupedShortcuts = DEFAULT_SHORTCUTS.reduce((acc, shortcut) => {
    if (!acc[shortcut.category]) {
      acc[shortcut.category] = []
    }
    acc[shortcut.category].push(shortcut)
    return acc
  }, {} as Record<string, typeof DEFAULT_SHORTCUTS>)

  const formatKey = (key: string) => {
    return key
      .split('+')
      .map(k => {
        const map: Record<string, string> = {
          'ctrl': 'Ctrl',
          'alt': 'Alt',
          'shift': 'Shift',
          'meta': 'Cmd',
        }
        return map[k] || k.toUpperCase()
      })
      .join(' + ')
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span className="flex items-center">
            <Keyboard className="h-4 w-4 mr-2" />
            快捷键设置
          </span>
          <Button
            size="sm"
            variant="outline"
            onClick={handleResetAll}
          >
            <RefreshCw className="h-4 w-4 mr-1" />
            全部重置
          </Button>
        </CardTitle>
      </CardHeader>

      <CardContent>
        <Tabs defaultValue="global">
          <TabsList className="mb-4">
            <TabsTrigger value="global">全局</TabsTrigger>
            <TabsTrigger value="chat">聊天</TabsTrigger>
            <TabsTrigger value="dice">掷骰</TabsTrigger>
            <TabsTrigger value="combat">战斗</TabsTrigger>
            <TabsTrigger value="general">通用</TabsTrigger>
          </TabsList>

          {Object.entries(groupedShortcuts).map(([category, shortcuts]) => (
            <TabsContent key={category} value={category}>
              <div className="space-y-2">
                {shortcuts.map(shortcut => {
                  const binding = bindings.find(b => b.shortcutId === shortcut.id)
                  const isRecording = recording === shortcut.id

                  return (
                    <div
                      key={shortcut.id}
                      className="flex items-center justify-between p-3 border rounded"
                    >
                      <div className="flex-1">
                        <div className="font-medium text-sm">{shortcut.name}</div>
                        <div className="text-xs text-muted-foreground mt-1">
                          {shortcut.description}
                        </div>
                      </div>

                      <div className="flex items-center space-x-2">
                        {shortcut.customizable ? (
                          <>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={isRecording}
                              onClick={() => handleRecordStart(shortcut.id)}
                            >
                              <Keyboard className="h-4 w-4 mr-1" />
                              {isRecording ? '按下按键...' : formatKey(binding?.currentKey || shortcut.defaultKey)}
                            </Button>

                            {binding?.currentKey !== shortcut.defaultKey && (
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => handleReset(shortcut.id)}
                              >
                                重置
                              </Button>
                            )}
                          </>
                        ) : (
                          <Badge variant="secondary">{formatKey(shortcut.defaultKey)}</Badge>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  )
}
```

---

## 快捷键帮助提示

```tsx
// frontend/src/components/shortcuts/ShortcutTooltip.tsx
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface ShortcutTooltipProps {
  shortcut: string
  children: React.ReactNode
}

export function ShortcutTooltip({ shortcut, children }: ShortcutTooltipProps) {
  const formatKey = (key: string) => {
    return key
      .split('+')
      .map(k => {
        const map: Record<string, string> = {
          'ctrl': 'Ctrl',
          'alt': 'Alt',
          'shift': 'Shift',
          'meta': '⌘',
        }
        return map[k] || k.toUpperCase()
      })
      .join(' + ')
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{children}</TooltipTrigger>
        <TooltipContent>
          <p className="font-mono text-xs">{formatKey(shortcut)}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/shortcuts/types.ts` | 创建 | 快捷键类型定义 |
| `frontend/src/lib/shortcuts/shortcut-manager.ts` | 创建 | 快捷键管理器 |
| `frontend/src/lib/shortcuts/default-shortcuts.ts` | 创建 | 默认快捷键配置 |
| `frontend/src/hooks/useShortcuts.ts` | 创建 | 快捷键 Hook |
| `frontend/src/components/settings/ShortcutSettings.tsx` | 创建 | 快捷键设置组件 |
| `frontend/src/components/shortcuts/ShortcutTooltip.tsx` | 创建 | 快捷键提示组件 |

---

## 验收标准

- [ ] 快捷键注册成功
- [ ] 全局快捷键有效
- [ ] 冲突检测正确
- [ ] 自定义功能可用
- [ ] 帮助提示准确
- [ ] 重置功能正常

---

## 参考文档

- KeyboardEvent API
- React Keybind 库

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
