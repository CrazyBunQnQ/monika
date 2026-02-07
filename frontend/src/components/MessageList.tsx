import { useEffect, useRef } from 'react'
import { cn } from '@/lib/utils'

interface MessageListProps {
  messages: Array<{
    id: string
    role: 'kp' | 'player' | 'system'
    content: string
    timestamp: Date
  }>
}

export function MessageList({ messages }: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {messages.map((message) => (
        <div
          key={message.id}
          className={cn(
            'max-w-[80%] rounded-lg p-3',
            message.role === 'kp' && 'bg-primary text-primary-foreground ml-auto',
            message.role === 'player' && 'bg-secondary text-secondary-foreground mr-auto',
            message.role === 'system' && 'bg-muted text-muted-foreground mx-auto text-center'
          )}
        >
          <p className="text-sm">{message.content}</p>
          <p className="text-xs mt-1 opacity-70">
            {message.timestamp.toLocaleTimeString()}
          </p>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}
