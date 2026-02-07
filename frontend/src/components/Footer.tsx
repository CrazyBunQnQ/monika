import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Send } from 'lucide-react'

interface FooterProps {
  onSendMessage: (content: string) => void
}

export function Footer({ onSendMessage }: FooterProps) {
  const [input, setInput] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (input.trim()) {
      onSendMessage(input.trim())
      setInput('')
    }
  }

  return (
    <footer className="border-t bg-card p-4">
      <form onSubmit={handleSubmit} className="flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="输入你的行动..."
          className="flex-1 px-4 py-2 bg-muted rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        />
        <Button type="submit" size="icon">
          <Send className="w-4 h-4" />
        </Button>
      </form>
    </footer>
  )
}
