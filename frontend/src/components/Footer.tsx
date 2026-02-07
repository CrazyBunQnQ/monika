import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Send, Dice3 } from "lucide-react"

interface FooterProps {
  onSendMessage: (content: string) => void
  onRoll?: () => void
}

export function Footer({ onSendMessage, onRoll }: FooterProps) {
  const [input, setInput] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (input.trim()) {
      onSendMessage(input.trim())
      setInput("")
    }
  }

  return (
    <footer className="border-t bg-card p-4">
      <div className="flex gap-2">
        <form onSubmit={handleSubmit} className="flex-1 flex gap-2">
          <Input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="描述你的行动..."
            className="flex-1"
          />
          <Button type="submit" size="icon">
            <Send className="h-4 w-4" />
          </Button>
        </form>
        {onRoll && (
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={onRoll}
            title="Quick Roll"
          >
            <Dice3 className="h-4 w-4" />
          </Button>
        )}
      </div>
      {/* Quick action hints */}
      <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
        <span>快捷键:</span>
        <kbd className="px-1.5 py-0.5 bg-muted rounded text-xs">Enter</kbd>
        <span>发送</span>
        <kbd className="px-1.5 py-0.5 bg-muted rounded text-xs">/roll</kbd>
        <span>检定</span>
      </div>
    </footer>
  )
}
