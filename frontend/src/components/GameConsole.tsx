import { useState } from 'react'
import { Header } from '@/components/Header'
import { MessageList } from '@/components/MessageList'
import { StatePanel } from '@/components/StatePanel'
import { Footer } from '@/components/Footer'
import { Button } from '@/components/ui/button'

interface GameConsoleProps {
  onLogout?: () => void
}

export function GameConsole({ onLogout }: GameConsoleProps) {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      role: 'kp',
      content: '欢迎来到 Monika！你是调查员，刚刚醒来发现自己在一间陌生的房间里。',
      timestamp: new Date(),
    },
  ])

  const [character] = useState<Character>({
    id: 1,
    name: '调查员',
    hp: 10,
    maxHp: 12,
    mp: 12,
    maxMp: 12,
    san: 60,
    maxSan: 60,
    luck: 50,
  })

  const handleSendMessage = (content: string) => {
    const newMessage: Message = {
      id: Date.now().toString(),
      role: 'player',
      content,
      timestamp: new Date(),
    }
    setMessages([...messages, newMessage])
  }

  return (
    <div className="flex flex-col h-screen">
      <Header characterName={character.name} onLogout={onLogout} />
      <div className="flex-1 flex overflow-hidden">
        <div className="flex-1 flex flex-col">
          <MessageList messages={messages} />
          <Footer onSendMessage={handleSendMessage} />
        </div>
        <StatePanel character={character} />
      </div>
    </div>
  )
}

interface Message {
  id: string
  role: 'kp' | 'player' | 'system'
  content: string
  timestamp: Date
}

interface Character {
  id: number
  name: string
  hp: number
  maxHp: number
  mp: number
  maxMp: number
  san: number
  maxSan: number
  luck: number
}
