# M1-046: 实现 ChatPanel 聊天面板组件

**任务ID**: M1-046
**标题**: 实现 ChatPanel 聊天面板组件
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M2-002

---

## 任务描述

实现游戏聊天面板组件，支持公共聊天、私聊、OOC、表情动作等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-046-01 | 设计聊天面板布局 | UI 设计 | 20min |
| M1-046-02 | 实现消息列表 | Message List | 30min |
| M1-046-03 | 实现输入框 | Input | 25min |
| M1-046-04 | 实现 WebSocket 连接 | Socket.io | 30min |
| M1-046-05 | 实现消息过滤 | Message Filter | 20min |
| M1-046-06 | 实现快速命令 | Quick Commands | 15min |
| M1-046-07 | 编写面板测试 | 测试覆盖 | 10min |

---

## 聊天面板组件

```tsx
// frontend/src/components/game/ChatPanel.tsx
import { useState, useEffect, useRef } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Send, Smile } from 'lucide-react'
import { socket } from '@/lib/socket'

interface ChatMessage {
  id: string
  type: 'chat' | 'private' | 'ooc' | 'me' | 'system'
  sender_id: string
  sender_name: string
  content: string
  timestamp: Date
  target_id?: string
}

interface ChatPanelProps {
  roomId: string
}

export function ChatPanel({ roomId }: ChatPanelProps) {
  const { user } = useAuth()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [activeTab, setActiveTab] = useState('all')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    // 监听新消息
    socket.on('chat_message', handleNewMessage)
    socket.on('private_message', handlePrivateMessage)
    socket.on('system_message', handleSystemMessage)

    return () => {
      socket.off('chat_message')
      socket.off('private_message')
      socket.off('system_message')
    }
  }, [])

  useEffect(() => {
    // 自动滚动到底部
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleNewMessage = (data: any) => {
    setMessages(prev => [...prev, {
      id: Date.now().toString(),
      type: 'chat',
      sender_id: data.sender_id,
      sender_name: data.sender_name,
      content: data.message,
      timestamp: new Date(data.timestamp),
    }])
  }

  const handlePrivateMessage = (data: any) => {
    setMessages(prev => [...prev, {
      id: Date.now().toString(),
      type: 'private',
      sender_id: data.sender_id,
      sender_name: data.sender_name,
      content: data.message,
      timestamp: new Date(data.timestamp),
      target_id: data.target_id || user?.id,
    }])
  }

  const handleSystemMessage = (data: any) => {
    setMessages(prev => [...prev, {
      id: Date.now().toString(),
      type: 'system',
      sender_id: 'system',
      sender_name: '系统',
      content: data.message,
      timestamp: new Date(),
    }])
  }

  const handleSend = () => {
    if (!input.trim()) return

    socket.emit('chat_message', {
      room_id: roomId,
      message: input.trim(),
      sender: {
        id: user?.id,
        name: user?.username,
      },
    })

    setInput('')
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  // 过滤消息
  const filteredMessages = messages.filter(msg => {
    if (activeTab === 'all') return msg.type !== 'private'
    if (activeTab === 'ooc') return msg.type === 'ooc'
    if (activeTab === 'ic') return msg.type === 'chat' || msg.type === 'me'
    return true
  })

  // 渲染消息
  const renderMessage = (msg: ChatMessage) => {
    switch (msg.type) {
      case 'chat':
        return (
          <div key={msg.id} className="flex items-start space-x-2">
            <span className="font-medium text-sm">{msg.sender_name}:</span>
            <span className="text-sm">{msg.content}</span>
          </div>
        )
      case 'private':
        const isToMe = msg.target_id === user?.id
        const isFromMe = msg.sender_id === user?.id
        return (
          <div key={msg.id} className={`flex items-start space-x-2 ${isToMe || isFromMe ? 'text-purple-500' : ''}`}>
            <span className="font-medium text-sm">
              [私聊] {msg.sender_name}:
            </span>
            <span className="text-sm">{msg.content}</span>
          </div>
        )
      case 'ooc':
        return (
          <div key={msg.id} className="text-sm text-muted-foreground italic">
            ((OOC)) {msg.sender_name}: {msg.content}
          </div>
        )
      case 'me':
        return (
          <div key={msg.id} className="text-sm italic">
            * {msg.sender_name} {msg.content}
          </div>
        )
      case 'system':
        return (
          <div key={msg.id} className="text-xs text-muted-foreground text-center py-1">
            {msg.content}
          </div>
        )
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* 消息列表 */}
      <Card className="flex-1 flex flex-col">
        <CardContent className="flex-1 p-4 overflow-y-auto">
          {filteredMessages.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              暂无消息，开始聊天吧！
            </div>
          ) : (
            <div className="space-y-2">
              {filteredMessages.map(renderMessage)}
            </div>
          )}
          <div ref={messagesEndRef} />
        </CardContent>
      </Card>

      {/* 输入框 */}
      <div className="space-y-2">
        {/* 标签页 */}
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="all">全部</TabsTrigger>
            <TabsTrigger value="ic">IC</TabsTrigger>
            <TabsTrigger value="ooc">OOC</TabsTrigger>
          </TabsList>
        </Tabs>

        {/* 输入区域 */}
        <div className="flex space-x-2">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="输入消息..."
            className="flex-1"
          />
          <Button
            size="icon"
            variant="ghost"
            onClick={() => {/* 打开表情选择器 */}}
          >
            <Smile className="h-4 w-4" />
          </Button>
          <Button
            size="icon"
            onClick={handleSend}
            disabled={!input.trim()}
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>

        {/* 快捷命令 */}
        <div className="flex flex-wrap gap-2 text-xs">
          <Badge
            variant="outline"
            className="cursor-pointer"
            onClick={() => setInput('/me ')}
          >
            /me 动作
          </Badge>
          <Badge
            variant="outline"
            className="cursor-pointer"
            onClick={() => setInput('// ')}
          >
            // OOC
          </Badge>
          <Badge
            variant="outline"
            className="cursor-pointer"
            onClick={() => setInput('/roll 1d100')}
          >
            /roll 1d100
          </Badge>
        </div>
      </div>
    </div>
  )
}
```

---

## Socket.io 客户端

```tsx
// frontend/src/lib/socket.ts
import { io, Socket } from 'socket.io-client'

let socketInstance: Socket | null = null

export function initSocket(token: string) {
  if (socketInstance) {
    return socketInstance
  }

  socketInstance = io({
    auth: { token },
    autoConnect: true,
  })

  socketInstance.on('connect', () => {
    console.log('Connected to WebSocket')
  })

  socketInstance.on('disconnect', () => {
    console.log('Disconnected from WebSocket')
  })

  return socketInstance
}

export function getSocket(): Socket {
  if (!socketInstance) {
    throw new Error('Socket not initialized')
  }
  return socketInstance

// 导出单例
export const socket = new Proxy({} as { socket: Socket | null }, {
  get(target, prop) {
    if (prop === 'socket') {
      return socketInstance
    }
    return socketInstance?.[prop]
  },
})
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/ChatPanel.tsx` | 创建 | 聊天面板主组件 |
| `frontend/src/lib/socket.ts` | 创建 | Socket.io 客户端 |
| `frontend/src/types/chat.ts` | 创建 | 聊天类型定义 |

---

## 验收标准

- [ ] 消息发送正常
- [ ] WebSocket 连接稳定
- [ ] 消息过滤有效
- [ ] 私聊功能正常
- [ ] OOC 消息区分明显
- [ ] 快捷命令可用

---

## 参考文档

- M2-002: WebSocket 事件系统
- M0-005: 聊天命令规范

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
