import { useState } from "react"
import { Header } from "@/components/Header"
import { MessageList } from "@/components/MessageList"
import { StatePanel, type CharacterState, type WorldState, type Lead } from "@/components/StatePanel"
import { Footer } from "@/components/Footer"
import { MessageBubble, type Message } from "@/components/MessageBubble"

export function GameConsole() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "1",
      role: "kp",
      content: "欢迎来到 Monika！你是调查员，刚刚醒来发现自己在一间陌生的房间里。",
      timestamp: new Date(),
      stateChanges: [
        { type: "generic", value: "Scene started" }
      ],
      next: [
        "观察房间的环境",
        "检查门是否锁着",
        "搜寻房间里的物品",
        "大声呼救"
      ]
    },
  ])

  const [character, setCharacter] = useState<CharacterState>({
    hp: 12,
    hpMax: 12,
    san: 60,
    sanMax: 60,
    luck: 50,
    luckMax: 50,
    mp: 10,
    mpMax: 10,
  })

  const [world, setWorld] = useState<WorldState>({
    currentScene: "陌生的房间",
    location: "未知地点",
    timer: undefined,
    leads: [
      { id: "1", text: "调查房间环境", verified: false },
      { id: "2", text: "检查门锁", verified: false },
      { id: "3", text: "搜寻物品", verified: false },
    ],
  })

  const handleSendMessage = (content: string) => {
    const playerMessage: Message = {
      id: Date.now().toString(),
      role: "player",
      content,
      timestamp: new Date(),
      sender: "调查员",
    }
    setMessages((prev) => [...prev, playerMessage])

    // Simulate KP response (in real app, this would be an API call)
    setTimeout(() => {
      const responseMessages: Record<string, Message> = {
        "观察": {
          id: (Date.now() + 1).toString(),
          role: "kp",
          content: "你仔细观察房间...房间里有一张旧床、一个破旧的柜子，还有一扇看起来紧闭的木门。墙壁上有些奇怪的符号。",
          timestamp: new Date(),
          stateChanges: [
            { type: "san", change: 0, reason: "Observation check" }
          ],
          next: [
            "检查墙上的符号 (需要检定)",
            "搜索柜子",
            "尝试开门",
          ]
        },
        "default": {
          id: (Date.now() + 1).toString(),
          role: "kp",
          content: "你做出了行动...",
          timestamp: new Date(),
          stateChanges: [],
          next: world.leads.map(l => l.text)
        }
      }

      const response = Object.entries(responseMessages).find(([key]) =>
        content.includes(key)
      )?.[1] || responseMessages.default

      setMessages((prev) => [...prev, response])
    }, 500)
  }

  const handleRoll = () => {
    // Simulate a dice roll (in real app, this would be an API call)
    const rollResult = Math.floor(Math.random() * 100) + 1
    const successLevel = rollResult <= 50 ? "regular_success" : "failure"

    const rollMessage: Message = {
      id: Date.now().toString(),
      role: "system",
      content: ` rolled ${rollResult} - ${successLevel.replace("_", " ")}`,
      timestamp: new Date(),
      stateChanges: [
        {
          type: "roll",
          value: `${rollResult} - ${successLevel.replace("_", " ")}`
        }
      ]
    }
    setMessages((prev) => [...prev, rollMessage])
  }

  return (
    <div className="flex flex-col h-screen">
      <Header characterName="调查员" />
      <div className="flex-1 flex overflow-hidden">
        <div className="flex-1 flex flex-col min-w-0">
          <MessageList messages={messages} />
          <Footer onSendMessage={handleSendMessage} onRoll={handleRoll} />
        </div>
        <StatePanel character={character} world={world} />
      </div>
    </div>
  )
}
