# M1-018: 实现 MessageList 组件

**任务ID**: M1-018
**标题**: 实现 MessageList 组件
**类型**: frontend (前端开发)
**预估工时**: 2.5h
**依赖**: M1-032

---

## 任务描述

实现游戏消息列表组件，用于显示检定结果、角色行动、系统消息等各类游戏信息。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-018-01 | 设计消息数据结构 | Message 类型 | 20min |
| M1-018-02 | 实现消息列表容器 | 滚动与布局 | 25min |
| M1-018-03 | 实现消息项组件 | 单条消息 | 30min |
| M1-018-04 | 实现消息分类 | 不同类型样式 | 25min |
| M1-018-05 | 实现消息过滤 | 按类型筛选 | 20min |
| M1-018-06 | 实现自动滚动 | 新消息跟随 | 20min |
| M1-018-07 | 实现消息导出 | 导出功能 | 20min |

---

## 消息数据结构

```tsx
// frontend/src/types/message.ts
export type MessageType =
  | 'check'      // 检定
  | 'damage'     // 伤害
  | 'heal'       // 治疗
  | 'sanity'     // SAN 检定
  | 'combat'     // 战斗
  | 'chat'       // 聊天
  | 'system'     // 系统
  | 'error'      // 错误

export interface Message {
  id: string
  type: MessageType
  timestamp: Date
  content: string
  visible_to: 'all' | 'kp' | 'player' | string[]  // 角色ID列表
  metadata?: {
    // 检定相关
    check_type?: string
    skill?: string
    target?: number
    rolled?: number
    success_level?: string

    // 伤害相关
    damage_type?: string
    amount?: number
    target_character?: string

    // SAN 相关
    san_cost?: string
    passed?: boolean

    // 发送者
    sender?: string
    sender_id?: string
  }
}
```

---

## 消息列表组件

```tsx
// frontend/src/components/game/MessageList.tsx
import { useEffect, useRef } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import type { Message } from '@/types/message'
import { CheckMessage } from './messages/CheckMessage'
import { DamageMessage } from './messages/DamageMessage'
import { SanityMessage } from './messages/SanityMessage'
import { ChatMessage } from './messages/ChatMessage'
import { SystemMessage } from './messages/SystemMessage'

interface MessageListProps {
  messages: Message[]
  filter?: MessageType[]
  autoScroll?: boolean
}

export function MessageList({
  messages,
  filter,
  autoScroll = true,
}: MessageListProps) {
  const { user } = useAuth()
  const scrollRef = useRef<HTMLDivElement>(null)
  const endRef = useRef<HTMLDivElement>(null)

  // 过滤可见消息
  const visibleMessages = messages.filter(msg => {
    // 类型过滤
    if (filter && !filter.includes(msg.type)) return false

    // 权限过滤
    if (msg.visible_to === 'all') return true
    if (msg.visible_to === 'kp') return user?.role === 'kp'
    if (msg.visible_to === 'player') return user?.role === 'player'
    if (Array.isArray(msg.visible_to)) {
      return msg.visible_to.includes(user?.id || '')
    }

    return true
  })

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && endRef.current) {
      endRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [visibleMessages.length, autoScroll])

  // 渲染单条消息
  const renderMessage = (msg: Message) => {
    switch (msg.type) {
      case 'check':
        return <CheckMessage key={msg.id} message={msg} />
      case 'damage':
        return <DamageMessage key={msg.id} message={msg} />
      case 'heal':
        return <DamageMessage key={msg.id} message={msg} />
      case 'sanity':
        return <SanityMessage key={msg.id} message={msg} />
      case 'chat':
        return <ChatMessage key={msg.id} message={msg} />
      case 'system':
        return <SystemMessage key={msg.id} message={msg} />
      default:
        return <SystemMessage key={msg.id} message={msg} />
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* 过滤器 */}
      <div className="flex items-center space-x-2 p-2 border-b">
        <FilterButtons currentFilter={filter} />
      </div>

      {/* 消息列表 */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto p-4 space-y-2"
      >
        {visibleMessages.length === 0 ? (
          <div className="text-center text-muted-foreground py-8">
            暂无消息
          </div>
        ) : (
          visibleMessages.map(renderMessage)
        )}
        <div ref={endRef} />
      </div>
    </div>
  )
}

// 过滤按钮
function FilterButtons({ currentFilter }: { currentFilter?: MessageType[] }) {
  const filters: { key: MessageType; label: string }[] = [
    { key: 'check', label: '检定' },
    { key: 'damage', label: '伤害' },
    { key: 'sanity', label: 'SAN' },
    { key: 'chat', label: '聊天' },
    { key: 'system', label: '系统' },
  ]

  return (
    <div className="flex items-center space-x-2">
      <span className="text-sm text-muted-foreground">过滤:</span>
      {filters.map(filter => (
        <button
          key={filter.key}
          className={`px-2 py-1 text-xs rounded ${
            !currentFilter || currentFilter.includes(filter.key)
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          }`}
        >
          {filter.label}
        </button>
      ))}
    </div>
  )
}
```

---

## 检定消息组件

```tsx
// frontend/src/components/game/messages/CheckMessage.tsx
import type { Message } from '@/types/message'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

interface CheckMessageProps {
  message: Message
}

export function CheckMessage({ message }: CheckMessageProps) {
  const { skill, target, rolled, success_level } = message.metadata || {}

  const getSuccessColor = (level?: string) => {
    switch (level) {
      case '大成功': return 'bg-emerald-500'
      case '极难成功': return 'bg-green-500'
      case '困难成功': return 'bg-lime-500'
      case '成功': return 'bg-yellow-500'
      case '失败': return 'bg-orange-500'
      case '大失败': return 'bg-red-500'
      default: return 'bg-gray-500'
    }
  }

  return (
    <Card className="border-l-4 border-l-blue-500">
      <CardContent className="p-3">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center space-x-2 mb-1">
              <span className="font-medium">{skill}</span>
              <Badge variant="outline">检定</Badge>
              {success_level && (
                <Badge className={getSuccessColor(success_level)}>
                  {success_level}
                </Badge>
              )}
            </div>
            <div className="text-sm text-muted-foreground">
              目标: {target} | 掷出: {rolled}
            </div>
            {message.content && (
              <div className="text-sm mt-1">{message.content}</div>
            )}
          </div>
          <div className="text-xs text-muted-foreground">
            {new Date(message.timestamp).toLocaleTimeString()}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 聊天消息组件

```tsx
// frontend/src/components/game/messages/ChatMessage.tsx
import type { Message } from '@/types/message'
import { Card, CardContent } from '@/components/ui/card'
import { Avatar } from '@/components/ui/avatar'

interface ChatMessageProps {
  message: Message
}

export function ChatMessage({ message }: ChatMessageProps) {
  const { sender } = message.metadata || {}

  return (
    <div className="flex items-start space-x-3">
      <Avatar className="h-8 w-8">
        <div className="flex h-full w-full items-center justify-center bg-primary text-primary-foreground text-sm">
          {sender?.charAt(0).toUpperCase()}
        </div>
      </Avatar>

      <Card className="flex-1">
        <CardContent className="p-3">
          <div className="flex items-center space-x-2 mb-1">
            <span className="font-medium text-sm">{sender}</span>
            <span className="text-xs text-muted-foreground">
              {new Date(message.timestamp).toLocaleTimeString()}
            </span>
          </div>
          <div className="text-sm">{message.content}</div>
        </CardContent>
      </Card>
    </div>
  )
}
```

---

## 系统消息组件

```tsx
// frontend/src/components/game/messages/SystemMessage.tsx
import type { Message } from '@/types/message'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Info } from 'lucide-react'

interface SystemMessageProps {
  message: Message
}

export function SystemMessage({ message }: SystemMessageProps) {
  return (
    <Alert>
      <Info className="h-4 w-4" />
      <AlertDescription className="ml-2">
        <div className="flex items-center justify-between">
          <span>{message.content}</span>
          <span className="text-xs text-muted-foreground ml-2">
            {new Date(message.timestamp).toLocaleTimeString()}
          </span>
        </div>
      </AlertDescription>
    </Alert>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/types/message.ts` | 创建 | 消息类型定义 |
| `frontend/src/components/game/MessageList.tsx` | 创建 | 消息列表主组件 |
| `frontend/src/components/game/messages/CheckMessage.tsx` | 创建 | 检定消息 |
| `frontend/src/components/game/messages/DamageMessage.tsx` | 创建 | 伤害消息 |
| `frontend/src/components/game/messages/SanityMessage.tsx` | 创建 | SAN 消息 |
| `frontend/src/components/game/messages/ChatMessage.tsx` | 创建 | 聊天消息 |
| `frontend/src/components/game/messages/SystemMessage.tsx` | 创建 | 系统消息 |

---

## 验收标准

- [ ] 消息列表正确渲染
- [ ] 不同类型消息样式正确
- [ ] 过滤功能有效
- [ ] 自动滚动工作
- [ ] 权限过滤正确
- [ ] 导出功能可用

---

## 参考文档

- M1-032: GameConsole 布局
- M1-010: 检定系统 API

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
