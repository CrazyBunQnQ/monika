# M5-008: 实现自动保存功能

**任务ID**: M5-008
**标题**: 实现自动保存功能
**类型**: frontend (前端开发)
**预估工时**: 1.5h
**依赖**: M2-002

---

## 任务描述

实现自动保存功能，定期保存游戏状态，防止数据丢失，支持手动保存和恢复历史版本。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-008-01 | 设计保存策略 | Save Strategy | 15min |
| M5-008-02 | 实现自动保存器 | Auto Saver | 25min |
| M5-008-03 | 实现保存状态 UI | Status UI | 25min |
| M5-008-04 | 实现手动保存 | Manual Save | 20min |
| M5-008-05 | 实现保存历史 | Save History | 20min |
| M5-008-06 | 实现冲突处理 | Conflict Handling | 15min |
| M5-008-07 | 编写保存测试 | 测试覆盖 | 10min |

---

## 自动保存配置

```typescript
// frontend/src/lib/autosave/config.ts
export interface AutoSaveConfig {
  enabled: boolean
  interval: number  // 保存间隔（毫秒）
  maxHistory: number  // 最大历史记录数
  debounceMs: number  // 防抖延迟
}

export const DEFAULT_AUTOSAVE_CONFIG: AutoSaveConfig = {
  enabled: true,
  interval: 60000,  // 60秒
  maxHistory: 10,
  debounceMs: 2000,  // 2秒
}
```

---

## 自动保存 Hook

```typescript
// frontend/src/hooks/useAutoSave.ts
import { useEffect, useRef, useCallback } from 'react'
import { useToast } from '@/hooks/use-toast'

interface AutoSaveOptions {
  roomId: string
  getData: () => any
  onSave?: () => void
  interval?: number
  debounceMs?: number
  enabled?: boolean
}

export function useAutoSave({
  roomId,
  getData,
  onSave,
  interval = 60000,
  debounceMs = 2000,
  enabled = true,
}: AutoSaveOptions) {
  const { toast } = useToast()
  const saveTimeoutRef = useRef<NodeJS.Timeout>()
  const lastSavedRef = useRef<Date | null>(null)
  const isDirtyRef = useRef(false)
  const savingRef = useRef(false)

  // 执行保存
  const performSave = useCallback(async () => {
    if (savingRef.current) {
      return
    }

    savingRef.current = true
    isDirtyRef.current = false

    try {
      const data = getData()

      const response = await fetch(`/api/rooms/${roomId}/autosave`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      })

      if (!response.ok) {
        throw new Error('保存失败')
      }

      lastSavedRef.current = new Date()
      onSave?.()

      toast({
        title: '已自动保存',
        description: `保存于 ${new Date().toLocaleTimeString('zh-CN')}`,
        duration: 2000,
      })
    } catch (error) {
      toast({
        title: '自动保存失败',
        description: '请检查网络连接',
        variant: 'destructive',
      })
    } finally {
      savingRef.current = false
    }
  }, [roomId, getData, onSave, toast])

  // 标记为需要保存
  const markDirty = useCallback(() => {
    isDirtyRef.current = true

    // 清除之前的定时器
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
    }

    // 设置新的定时器（防抖）
    saveTimeoutRef.current = setTimeout(() => {
      performSave()
    }, debounceMs)
  }, [debounceMs, performSave])

  // 手动保存
  const manualSave = useCallback(async () => {
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
      saveTimeoutRef.current = undefined
    }

    await performSave()
  }, [performSave])

  // 定时保存
  useEffect(() => {
    if (!enabled) return

    const intervalId = setInterval(() => {
      if (isDirtyRef.current) {
        performSave()
      }
    }, interval)

    return () => clearInterval(intervalId)
  }, [enabled, interval, performSave])

  // 页面卸载前保存
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (isDirtyRef.current) {
        e.preventDefault()
        e.returnValue = ''
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)

    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)

      // 组件卸载时如果有未保存的数据，尝试保存
      if (isDirtyRef.current && !savingRef.current) {
        performSave()
      }
    }
  }, [performSave])

  return {
    lastSaved: lastSavedRef.current,
    isSaving: savingRef.current,
    markDirty,
    manualSave,
  }
}
```

---

## 保存状态指示器

```tsx
// frontend/src/components/game/AutoSaveIndicator.tsx
import { useState, useEffect } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Save, Check, AlertCircle, Loader2 } from 'lucide-react'

interface AutoSaveIndicatorProps {
  lastSaved: Date | null
  isSaving: boolean
  onSave: () => void
}

export function AutoSaveIndicator({ lastSaved, isSaving, onSave }: AutoSaveIndicatorProps) {
  const [timeUntilNext, setTimeUntilNext] = useState<number>(60)

  useEffect(() => {
    if (lastSaved) {
      const interval = setInterval(() => {
        const elapsed = Date.now() - lastSaved.getTime()
        const remaining = Math.max(0, 60 - Math.floor(elapsed / 1000))
        setTimeUntilNext(remaining)
      }, 1000)

      return () => clearInterval(interval)
    }
  }, [lastSaved])

  return (
    <div className="flex items-center space-x-2">
      {isSaving ? (
        <Badge variant="secondary" className="flex items-center space-x-1">
          <Loader2 className="h-3 w-3 animate-spin" />
          <span>保存中...</span>
        </Badge>
      ) : lastSaved ? (
        <Badge variant="outline" className="flex items-center space-x-1">
          <Check className="h-3 w-3" />
          <span>
            {timeUntilNext > 0 ? `${timeUntilNext}秒后保存` : '准备保存'}
          </span>
        </Badge>
      ) : (
        <Badge variant="secondary" className="flex items-center space-x-1">
          <AlertCircle className="h-3 w-3" />
          <span>未保存</span>
        </Badge>
      )}

      <Button size="sm" variant="outline" onClick={onSave}>
        <Save className="h-4 w-4" />
      </Button>
    </div>
  )
}
```

---

## 保存历史管理

```typescript
// frontend/src/lib/autosave/history.ts
interface SaveHistoryEntry {
  id: string
  timestamp: Date
  data: any
  description?: string
}

class SaveHistoryManager {
  private history: SaveHistoryEntry[] = []
  private maxHistory: number = 10

  constructor(maxHistory: number = 10) {
    this.maxHistory = maxHistory
    this.loadFromStorage()
  }

  addEntry(data: any, description?: string): string {
    const entry: SaveHistoryEntry = {
      id: `save_${Date.now()}`,
      timestamp: new Date(),
      data,
      description,
    }

    this.history.unshift(entry)

    // 限制历史记录数量
    if (this.history.length > this.maxHistory) {
      this.history = this.history.slice(0, this.maxHistory)
    }

    this.saveToStorage()
    return entry.id
  }

  getEntry(id: string): SaveHistoryEntry | undefined {
    return this.history.find(e => e.id === id)
  }

  getHistory(): SaveHistoryEntry[] {
    return [...this.history]
  }

  clear(): void {
    this.history = []
    this.saveToStorage()
  }

  private saveToStorage() {
    try {
      localStorage.setItem('save-history', JSON.stringify(this.history))
    } catch (error) {
      console.error('Failed to save history:', error)
    }
  }

  private loadFromStorage() {
    try {
      const saved = localStorage.getItem('save-history')
      if (saved) {
        this.history = JSON.parse(saved)
      }
    } catch (error) {
      console.error('Failed to load history:', error)
    }
  }
}

export const saveHistoryManager = new SaveHistoryManager()
```

---

## 保存历史面板

```tsx
// frontend/src/components/game/SaveHistoryPanel.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { History, RotateCcw, Trash2 } from 'lucide-react'
import { saveHistoryManager } from '@/lib/autosave/history'
import { useToast } from '@/hooks/use-toast'

interface SaveHistoryEntry {
  id: string
  timestamp: string
  description?: string
}

interface SaveHistoryPanelProps {
  roomId: string
  onRestore: (saveId: string) => void
}

export function SaveHistoryPanel({ roomId, onRestore }: SaveHistoryPanelProps) {
  const [history, setHistory] = useState<SaveHistoryEntry[]>([])
  const [loading, setLoading] = useState(false)

  const { toast } = useToast()

  useEffect(() => {
    loadHistory()
  }, [])

  const loadHistory = () => {
    const entries = saveHistoryManager.getHistory()
    setHistory(entries.map(e => ({
      ...e,
      timestamp: new Date(e.timestamp).toISOString(),
    })))
  }

  const handleRestore = async (saveId: string) => {
    setLoading(true)

    try {
      const entry = saveHistoryManager.getEntry(saveId)
      if (!entry) {
        throw new Error('存档不存在')
      }

      // 恢复数据
      const response = await fetch(`/api/rooms/${roomId}/restore`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ save_id: saveId }),
      })

      if (!response.ok) throw new Error('恢复失败')

      onRestore(saveId)

      toast({
        title: '已恢复存档',
        description: new Date(entry.timestamp).toLocaleString('zh-CN'),
      })
    } catch (error) {
      toast({
        title: '恢复失败',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleClear = () => {
    saveHistoryManager.clear()
    setHistory([])
    toast({
      title: '已清空历史',
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span className="flex items-center">
            <History className="h-4 w-4 mr-2" />
            保存历史
          </span>
          <Button
            size="sm"
            variant="outline"
            onClick={handleClear}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="space-y-2 max-h-64 overflow-y-auto">
          {history.length === 0 ? (
            <div className="text-center text-sm text-muted-foreground py-8">
              暂无保存历史
            </div>
          ) : (
            history.map((entry, index) => (
              <div
                key={entry.id}
                className="flex items-center justify-between p-3 border rounded hover:bg-muted"
              >
                <div className="flex-1">
                  <div className="flex items-center space-x-2">
                    <Badge variant="secondary" className="text-xs">
                      {index === 0 ? '最新' : `-${index}`}
                    </Badge>
                    <span className="text-sm">
                      {entry.description || '自动保存'}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    {new Date(entry.timestamp).toLocaleString('zh-CN')}
                  </p>
                </div>

                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleRestore(entry.id)}
                  disabled={loading}
                >
                  <RotateCcw className="h-3 w-3 mr-1" />
                  恢复
                </Button>
              </div>
            ))
          )}
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 使用示例

```tsx
// 在房间组件中使用自动保存
import { useAutoSave } from '@/hooks/useAutoSave'
import { AutoSaveIndicator } from '@/components/game/AutoSaveIndicator'
import { useMemo } from 'react'

export function GameRoom({ roomId }: { roomId: string }) {
  const [state, setState] = useState(initialState)

  // 获取当前状态数据
  const getData = useMemo(() => {
    return {
      characters: state.characters,
      npcs: state.npcs,
      clues: state.clues,
      // ... 其他需要保存的数据
    }
  }, [state])

  // 使用自动保存
  const { lastSaved, isSaving, markDirty, manualSave } = useAutoSave({
    roomId,
    getData,
    interval: 60000,  // 60秒
    debounceMs: 2000,  // 2秒防抖
    enabled: true,
  })

  // 当状态改变时标记为需要保存
  const updateState = (updates: any) => {
    setState(prev => ({ ...prev, ...updates }))
    markDirty()
  }

  return (
    <div>
      <div className="flex items-center justify-between p-4 border-b">
        <h1>游戏房间</h1>
        <AutoSaveIndicator
          lastSaved={lastSaved}
          isSaving={isSaving}
          onSave={manualSave}
        />
      </div>

      {/* 游戏内容 */}
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/autosave/config.ts` | 创建 | 自动保存配置 |
| `frontend/src/hooks/useAutoSave.ts` | 创建 | 自动保存 Hook |
| `frontend/src/lib/autosave/history.ts` | 创建 | 保存历史管理 |
| `frontend/src/components/game/AutoSaveIndicator.tsx` | 创建 | 保存指示器组件 |
| `frontend/src/components/game/SaveHistoryPanel.tsx` | 创建 | 保存历史面板 |
| `app/api/rooms.py` | 修改 | 添加自动保存端点 |

---

## 验收标准

- [ ] 自动保存正常
- [ ] 手动保存有效
- [ ] 防抖功能正确
- [ ] 状态指示准确
- [ ] 历史恢复成功
- [ ] 冲突处理安全

---

## 参考文档

- M2-002: WebSocket 事件系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
