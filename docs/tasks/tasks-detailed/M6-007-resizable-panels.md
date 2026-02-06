# M6-007: 实现可调整面板布局

**任务ID**: M6-007
**标题**: 实现可调整面板布局
**类型**: frontend (前端开发)
**预估工时**: 2.5h
**依赖**: M6-001

---

## 任务描述

实现可拖拽调整大小的面板布局系统，允许用户自定义各个面板的大小和位置，提供灵活的界面布局。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-007-01 | 设计布局系统架构 | Layout Architecture | 30min |
| M6-007-02 | 实现拖拽调整大小 | Resize Logic | 40min |
| M6-007-03 | 实现面板拖拽移动 | Drag Logic | 35min |
| M6-007-04 | 实现布局持久化 | Layout Persistence | 25min |
| M6-007-05 | 实现预设布局 | Layout Presets | 25min |
| M6-007-06 | 编写测试 | 测试覆盖 | 25min |

---

## 布局系统核心

```typescript
// frontend/src/lib/layout/resizable-layout.ts
import { useRef, useState, useCallback, useEffect } from 'react'
import { useDebounce } from '@/hooks/use-debounce'

export interface PanelConfig {
  id: string
  minSize?: number
  maxSize?: number
  defaultSize?: number
  collapsible?: boolean
}

export interface LayoutState {
  panels: Record<string, {
    size: number
    collapsed?: boolean
    order: number
  }>
  direction: 'horizontal' | 'vertical'
}

export function useResizableLayout(
  panels: PanelConfig[],
  direction: 'horizontal' | 'vertical' = 'horizontal',
  storageKey?: string
) {
  const [layout, setLayout] = useState<LayoutState>(() => {
    // 尝试从 localStorage 加载
    if (storageKey) {
      try {
        const saved = localStorage.getItem(storageKey)
        if (saved) {
          return JSON.parse(saved)
        }
      } catch {
        // 忽略错误
      }
    }

    // 初始化默认布局
    const panelCount = panels.length
    const defaultSize = 100 / panelCount

    return {
      panels: panels.reduce((acc, panel, index) => {
        acc[panel.id] = {
          size: panel.defaultSize ?? defaultSize,
          order: index,
        }
        return acc
      }, {} as LayoutState['panels']),
      direction,
    }
  })

  const [resizing, setResizing] = useState<string | null>(null)
  const [dragging, setDragging] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // 持久化布局
  const debouncedLayout = useDebounce(layout, 500)

  useEffect(() => {
    if (storageKey) {
      localStorage.setItem(storageKey, JSON.stringify(debouncedLayout))
    }
  }, [debouncedLayout, storageKey])

  // 调整面板大小
  const startResize = useCallback((panelId: string) => {
    setResizing(panelId)
  }, [])

  const stopResize = useCallback(() => {
    setResizing(null)
  }, [])

  const resize = useCallback((panelId: string, delta: number) => {
    setLayout((prev) => {
      const panelConfigs = panels.reduce((acc, p) => {
        acc[p.id] = p
        return acc
      }, {} as Record<string, PanelConfig>)

      const panel = prev.panels[panelId]
      const panelConfig = panelConfigs[panelId]

      if (!panel || !panelConfig) return prev

      // 获取面板顺序
      const panelOrder = panel.order
      const nextPanelId = Object.keys(prev.panels).find(
        (id) => prev.panels[id].order === panelOrder + 1
      )

      if (!nextPanelId) return prev

      const nextPanel = prev.panels[nextPanelId]
      const nextPanelConfig = panelConfigs[nextPanelId]

      // 计算新大小
      const containerSize = direction === 'horizontal'
        ? containerRef.current?.offsetWidth ?? 0
        : containerRef.current?.offsetHeight ?? 0

      const deltaPercent = (delta / containerSize) * 100

      let newSize = panel.size + deltaPercent
      let nextSize = nextPanel.size - deltaPercent

      // 应用最小/最大尺寸限制
      if (panelConfig.minSize !== undefined) {
        newSize = Math.max(newSize, panelConfig.minSize)
        nextSize = nextPanel.size - (newSize - panel.size)
      }

      if (panelConfig.maxSize !== undefined) {
        newSize = Math.min(newSize, panelConfig.maxSize)
        nextSize = nextPanel.size - (newSize - panel.size)
      }

      if (nextPanelConfig?.minSize !== undefined) {
        nextSize = Math.max(nextSize, nextPanelConfig.minSize)
        newSize = panel.size - (nextSize - nextPanel.size)
      }

      if (nextPanelConfig?.maxSize !== undefined) {
        nextSize = Math.min(nextSize, nextPanelConfig.maxSize)
        newSize = panel.size - (nextSize - nextPanel.size)
      }

      return {
        ...prev,
        panels: {
          ...prev.panels,
          [panelId]: { ...panel, size: newSize },
          [nextPanelId]: { ...nextPanel, size: nextSize },
        },
      }
    })
  }, [panels, direction])

  // 切换折叠状态
  const toggleCollapse = useCallback((panelId: string) => {
    setLayout((prev) => {
      const panel = prev.panels[panelId]
      const panelConfig = panels.find((p) => p.id === panelId)

      if (!panel || !panelConfig?.collapsible) return prev

      return {
        ...prev,
        panels: {
          ...prev.panels,
          [panelId]: {
            ...panel,
            collapsed: !panel.collapsed,
            size: panel.collapsed ? panel._sizeBeforeCollapse ?? panelConfig.defaultSize ?? 20 : 0,
            _sizeBeforeCollapse: panel.collapsed ? undefined : panel.size,
          } as any,
        },
      }
    })
  }, [panels])

  // 重置布局
  const resetLayout = useCallback(() => {
    const panelCount = panels.length
    const defaultSize = 100 / panelCount

    setLayout({
      panels: panels.reduce((acc, panel, index) => {
        acc[panel.id] = {
          size: panel.defaultSize ?? defaultSize,
          order: index,
        }
        return acc
      }, {} as LayoutState['panels']),
      direction,
    })

    if (storageKey) {
      localStorage.removeItem(storageKey)
    }
  }, [panels, direction, storageKey])

  return {
    layout,
    containerRef,
    resizing,
    dragging,
    startResize,
    stopResize,
    resize,
    toggleCollapse,
    resetLayout,
  }
}
```

---

## 可调整大小容器组件

```tsx
// frontend/src/components/layout/ResizablePanels.tsx
import { ReactNode, useRef, useEffect, useCallback } from 'react'
import { useResizableLayout, PanelConfig } from '@/lib/layout/resizable-layout'
import { GripVertical, GripHorizontal } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ResizablePanelsProps {
  panels: PanelConfig[]
  direction?: 'horizontal' | 'vertical'
  children: Record<string, ReactNode>
  className?: string
  storageKey?: string
}

export function ResizablePanels({
  panels,
  direction = 'horizontal',
  children,
  className,
  storageKey,
}: ResizablePanelsProps) {
  const {
    layout,
    containerRef,
    resizing,
    startResize,
    stopResize,
    resize,
    toggleCollapse,
    resetLayout,
  } = useResizableLayout(panels, direction, storageKey)

  const startPosRef = useRef<number>(0)
  const startSizeRef = useRef<Record<string, number>>({})

  const handleMouseDown = useCallback(
    (panelId: string) => (e: React.MouseEvent) => {
      e.preventDefault()
      startPosRef.current = direction === 'horizontal' ? e.clientX : e.clientY

      // 记录初始大小
      startSizeRef.current = {}
      Object.entries(layout.panels).forEach(([id, panel]) => {
        startSizeRef.current[id] = panel.size
      })

      startResize(panelId)
    },
    [direction, layout.panels, startResize]
  )

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!resizing) return

      const currentPos = direction === 'horizontal' ? e.clientX : e.clientY
      const delta = currentPos - startPosRef.current

      resize(resizing, delta)
    },
    [direction, resizing, resize]
  )

  const handleMouseUp = useCallback(() => {
    stopResize()
  }, [stopResize])

  useEffect(() => {
    if (resizing) {
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)

      return () => {
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
      }
    }
  }, [resizing, handleMouseMove, handleMouseUp])

  // 按顺序排列面板
  const sortedPanels = Object.entries(layout.panels)
    .sort(([, a], [, b]) => a.order - b.order)
    .map(([id, panel]) => ({ id, ...panel }))

  return (
    <div
      ref={containerRef}
      className={cn(
        'flex w-full h-full',
        direction === 'horizontal' ? 'flex-row' : 'flex-col',
        className
      )}
    >
      {sortedPanels.map((panel, index) => {
        const panelConfig = panels.find((p) => p.id === panel.id)
        const isCollapsed = panel.collapsed

        return (
          <div
            key={panel.id}
            style={{
              [direction === 'horizontal' ? 'width' : 'height']:
                isCollapsed ? 0 : `${panel.size}%`,
              display: isCollapsed ? 'none' : 'flex',
              minWidth: isCollapsed ? 0 : panelConfig?.minSize ? `${panelConfig.minSize}%` : undefined,
              minHeight: isCollapsed ? 0 : panelConfig?.minSize ? `${panelConfig.minSize}%` : undefined,
            }}
            className="overflow-hidden"
          >
            {children[panel.id]}

            {/* 调整大小的手柄 */}
            {!isCollapsed && index < sortedPanels.length - 1 && (
              <div
                className={cn(
                  'flex-shrink-0 bg-border hover:bg-primary/20 cursor-col-resize transition-colors',
                  direction === 'horizontal'
                    ? 'w-1 h-full cursor-col-resize'
                    : 'h-1 w-full cursor-row-resize',
                  resizing === panel.id && 'bg-primary'
                )}
                onMouseDown={handleMouseDown(panel.id)}
              >
                {direction === 'horizontal' ? (
                  <GripVertical className="h-4 w-4 mx-auto" />
                ) : (
                  <GripHorizontal className="h-4 w-4 my-auto" />
                )}
              </div>
            )}
          </div>
        )
      })}

      {/* 折叠的面板按钮 */}
      <div className="flex gap-1">
        {panels
          .filter((p) => p.collapsible)
          .map((panel) => {
            const isCollapsed = layout.panels[panel.id]?.collapsed
            return (
              <button
                key={panel.id}
                onClick={() => toggleCollapse(panel.id)}
                className="px-2 py-1 text-xs bg-muted hover:bg-muted-foreground/20 rounded"
              >
                {isCollapsed ? `展开 ${panel.id}` : `折叠 ${panel.id}`}
              </button>
            )
          })}
      </div>

      {/* 重置按钮 */}
      <button
        onClick={resetLayout}
        className="px-2 py-1 text-xs bg-muted hover:bg-muted-foreground/20 rounded"
      >
        重置布局
      </button>
    </div>
  )
}
```

---

## 游戏房间布局组件

```tsx
// frontend/src/components/game/GameRoomLayout.tsx
import { ReactNode } from 'react'
import { ResizablePanels } from '@/components/layout/ResizablePanels'
import { GameChat } from '@/components/game/GameChat'
import { GameBoard } from '@/components/game/GameBoard'
import { PlayerList } from '@/components/game/PlayerList'
import { DicePanel } from '@/components/game/DicePanel'
import { CharacterSheet } from '@/components/game/CharacterSheet'

interface GameRoomLayoutProps {
  roomId: string
}

export function GameRoomLayout({ roomId }: GameRoomLayoutProps) {
  const panels = [
    { id: 'sidebar', minSize: 15, maxSize: 30, defaultSize: 20, collapsible: true },
    { id: 'main', minSize: 40, defaultSize: 50 },
    { id: 'right', minSize: 15, maxSize: 30, defaultSize: 30, collapsible: true },
  ]

  return (
    <ResizablePanels
      panels={panels}
      direction="horizontal"
      storageKey={`room-layout-${roomId}`}
    >
      {{
        sidebar: (
          <div className="flex flex-col h-full">
            <PlayerList roomId={roomId} />
            <DicePanel roomId={roomId} />
          </div>
        ),
        main: (
          <ResizablePanels
            panels={[
              { id: 'board', minSize: 30, defaultSize: 70 },
              { id: 'chat', minSize: 20, maxSize: 50, defaultSize: 30, collapsible: true },
            ]}
            direction="vertical"
            storageKey={`room-layout-vertical-${roomId}`}
          >
            {{
              board: <GameBoard roomId={roomId} />,
              chat: <GameChat roomId={roomId} />,
            }}
          </ResizablePanels>
        ),
        right: (
          <div className="flex flex-col h-full">
            <CharacterSheet roomId={roomId} />
          </div>
        ),
      }}
    </ResizablePanels>
  )
}
```

---

## 布局预设管理

```typescript
// frontend/src/lib/layout/layout-presets.ts
export interface LayoutPreset {
  id: string
  name: string
  description: string
  icon: string
  layout: {
    panels: Record<string, {
      size: number
      collapsed?: boolean
      order: number
    }>
    direction: 'horizontal' | 'vertical'
  }
}

export const LAYOUT_PRESETS: LayoutPreset[] = [
  {
    id: 'classic',
    name: '经典布局',
    description: '左侧玩家列表，中间游戏区，右侧聊天',
    icon: '🎮',
    layout: {
      direction: 'horizontal',
      panels: {
        sidebar: { size: 20, order: 0 },
        main: { size: 50, order: 1 },
        chat: { size: 30, order: 2 },
      },
    },
  },
  {
    id: 'focus',
    name: '专注模式',
    description: '最大化游戏区域',
    icon: '🎯',
    layout: {
      direction: 'horizontal',
      panels: {
        main: { size: 100, order: 0 },
        sidebar: { size: 0, collapsed: true, order: 1 },
        chat: { size: 0, collapsed: true, order: 2 },
      },
    },
  },
  {
    id: 'social',
    name: '社交模式',
    description: '放大聊天区域',
    icon: '💬',
    layout: {
      direction: 'horizontal',
      panels: {
        sidebar: { size: 15, order: 0 },
        chat: { size: 40, order: 1 },
        main: { size: 45, order: 2 },
      },
    },
  },
  {
    id: 'vertical',
    name: '垂直布局',
    description: '上下分屏布局',
    icon: '↕️',
    layout: {
      direction: 'vertical',
      panels: {
        board: { size: 60, order: 0 },
        chat: { size: 40, order: 1 },
        sidebar: { size: 0, collapsed: true, order: 2 },
      },
    },
  },
]

export function getPreset(id: string): LayoutPreset | undefined {
  return LAYOUT_PRESETS.find((preset) => preset.id === id)
}

export function saveCustomPreset(
  userId: string,
  name: string,
  layout: LayoutPreset['layout']
): LayoutPreset {
  return {
    id: `custom-${Date.now()}`,
    name,
    description: '自定义布局',
    icon: '⚙️',
    layout,
  }
}
```

---

## 布局预设选择器

```tsx
// frontend/src/components/layout/LayoutPresetSelector.tsx
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { LayoutGrid } from 'lucide-react'
import { LAYOUT_PRESETS, getPreset } from '@/lib/layout/layout-presets'

interface LayoutPresetSelectorProps {
  onSelect: (preset: LayoutPreset) => void
  currentPreset?: string
}

export function LayoutPresetSelector({
  onSelect,
  currentPreset,
}: LayoutPresetSelectorProps) {
  const [open, setOpen] = useState(false)

  const handleSelect = (presetId: string) => {
    const preset = getPreset(presetId)
    if (preset) {
      onSelect(preset)
      setOpen(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <LayoutGrid className="h-4 w-4 mr-2" />
          布局
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>选择布局</DialogTitle>
          <DialogDescription>
            选择一个预设布局或自定义您的界面布局
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-4 mt-4">
          {LAYOUT_PRESETS.map((preset) => (
            <button
              key={preset.id}
              onClick={() => handleSelect(preset.id)}
              className={`
                p-4 rounded-lg border-2 transition-all
                ${currentPreset === preset.id
                  ? 'border-primary bg-primary/10'
                  : 'border-border hover:border-primary/50'
                }
              `}
            >
              <div className="text-3xl mb-2">{preset.icon}</div>
              <div className="font-medium">{preset.name}</div>
              <div className="text-sm text-muted-foreground">
                {preset.description}
              </div>
            </button>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
```

---

## 拖拽移动面板

```typescript
// frontend/src/lib/layout/draggable-panel.ts
import { useRef, useCallback, useState, useEffect } from 'react'

interface Position {
  x: number
  y: number
}

export function useDraggablePanel(
  panelId: string,
  onPositionChange?: (position: Position) => void
) {
  const panelRef = useRef<HTMLDivElement>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [position, setPosition] = useState<Position>({ x: 0, y: 0 })
  const startPosRef = useRef<Position>({ x: 0, y: 0 })
  const offsetRef = useRef<Position>({ x: 0, y: 0 })

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    // 只在标题栏上可以拖拽
    if (!(e.target as HTMLElement).closest('.drag-handle')) return

    e.preventDefault()
    setIsDragging(true)
    startPosRef.current = { x: e.clientX, y: e.clientY }
    offsetRef.current = position
  }, [position])

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isDragging) return

    const deltaX = e.clientX - startPosRef.current.x
    const deltaY = e.clientY - startPosRef.current.y

    const newPosition = {
      x: offsetRef.current.x + deltaX,
      y: offsetRef.current.y + deltaY,
    }

    setPosition(newPosition)
    onPositionChange?.(newPosition)
  }, [isDragging, onPositionChange])

  const handleMouseUp = useCallback(() => {
    setIsDragging(false)
  }, [])

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)

      return () => {
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
      }
    }
  }, [isDragging, handleMouseMove, handleMouseUp])

  return {
    panelRef,
    isDragging,
    position,
    handleMouseDown,
  }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/lib/layout/resizable-layout.ts` | 创建 | 可调整布局核心 |
| `frontend/src/lib/layout/layout-presets.ts` | 创建 | 布局预设 |
| `frontend/src/lib/layout/draggable-panel.ts` | 创建 | 拖拽面板 |
| `frontend/src/components/layout/ResizablePanels.tsx` | 创建 | 可调整面板组件 |
| `frontend/src/components/layout/LayoutPresetSelector.tsx` | 创建 | 布局预设选择器 |
| `frontend/src/components/game/GameRoomLayout.tsx` | 创建 | 游戏房间布局 |

---

## 验收标准

- [ ] 面板大小可拖拽调整
- [ ] 最小/最大尺寸限制有效
- [ ] 折叠功能正常
- [ ] 布局可持久化
- [ ] 预设布局可应用
- [ ] 响应式适配正常

---

## 参考文档

- M6-001: 响应式布局
- react-resizable-panels

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
