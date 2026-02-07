import { useState, useCallback, useEffect } from "react"
import { Header } from "@/components/Header"
import { MessageList } from "@/components/MessageList"
import { StatePanel } from "@/components/StatePanel"
import { Footer } from "@/components/Footer"
import { useGameWebSocket } from "@/hooks/useGameWebSocket"
import { useLLMResponse } from "@/hooks/useLLMResponse"
import { useAuth } from "@/contexts/AuthContext"
import type { ServerMessage, KeeperMessage, StateUpdate } from "@/types/websocket"
import { toast } from "sonner"

interface Message {
  id: string
  role: 'kp' | 'player' | 'system'
  content: string
  timestamp: Date
  sender?: string
}

interface CharacterState {
  hp: number
  hpMax: number
  san: number
  sanMax: number
  luck: number
  luckMax: number
  mp: number
  mpMax: 10
}

interface WorldState {
  currentScene: string
  location: string
  timer?: number
  leads: Array<{ id: string; text: string; verified: boolean }>
}

export function GameConsole() {
  const { user } = useAuth()

  // State management
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "1",
      role: "kp",
      content: "欢迎来到 Monika！你是调查员，刚刚醒来发现自己在一间陌生的房间里。",
      timestamp: new Date(),
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

  // Get sessionId from user or localStorage (for demo purposes)
  // In production, this would come from the game session state
  const sessionId = user?.id?.toString() || localStorage.getItem('monika_session_id') || null

  // LLM streaming response management
  const { streamingText, isStreaming, currentResponse, processStream, finalizeResponse, reset: resetLLM } = useLLMResponse()

  /**
   * Add a keeper message to the message list
   * Converts LLM response to Message format
   */
  const addKeeperMessage = useCallback((llmResponse: import('@/types/websocket').LLMResponse) => {
    const keeperMessage: Message = {
      id: Date.now().toString(),
      role: "kp",
      content: llmResponse.narrative,
      timestamp: new Date(),
    }
    setMessages(prev => [...prev, keeperMessage])
  }, [])

  /**
   * Handle incoming messages from the WebSocket server
   * Processes keeper messages and updates the message list
   */
  const handleServerMessage = useCallback((message: ServerMessage) => {
    if (message.type === 'keeper_message') {
      const keeperMsg = message as KeeperMessage

      if (keeperMsg.is_streaming) {
        // Process streaming chunks for progressive display
        processStream(keeperMsg.content)
      } else {
        // Finalize the response when streaming is complete
        finalizeResponse(keeperMsg.content)

        // Add complete message to the message list
        addKeeperMessage(keeperMsg.content)
      }
    } else if (message.type === 'error') {
      toast.error(message.content)
    }
  }, [processStream, finalizeResponse, addKeeperMessage])

  /**
   * Handle state updates from the WebSocket server
   * Updates world state and character state based on server changes
   */
  const handleStateUpdate = useCallback((update: StateUpdate) => {
    // Update world state based on server changes
    if (update.content.world_state) {
      setWorld((prevWorld) => ({
        ...prevWorld,
        ...(update.content.world_state.location && { location: update.content.world_state.location }),
        ...(update.content.world_state.leads && {
          leads: update.content.world_state.leads.map((text: string, idx: number) => ({
            id: `lead-${Date.now()}-${idx}`,
            text,
            verified: false
          }))
        })
      }))
    }

    // Update current scene if provided
    if (update.content.current_scene) {
      setWorld((prevWorld) => ({
        ...prevWorld,
        currentScene: update.content.current_scene
      }))
    }
  }, [])

  /**
   * Simulate mock response when WebSocket is not connected
   * This is a fallback for development/testing
   */
  const simulateMockResponse = useCallback((content: string) => {
    setTimeout(() => {
      const responseMessages: Record<string, Message> = {
        "观察": {
          id: (Date.now() + 1).toString(),
          role: "kp",
          content: "你仔细观察房间...房间里有一张旧床、一个破旧的柜子，还有一扇看起来紧闭的木门。墙壁上有些奇怪的符号。",
          timestamp: new Date(),
        },
        "default": {
          id: (Date.now() + 1).toString(),
          role: "kp",
          content: "你做出了行动...",
          timestamp: new Date(),
        }
      }

      const response = Object.entries(responseMessages).find(([key]) =>
        content.includes(key)
      )?.[1] || responseMessages.default

      setMessages((prev) => [...prev, response])
    }, 500)
  }, [])

  // WebSocket connection for real-time game communication
  const { isConnected, error, sendMessage } = useGameWebSocket(sessionId, {
    onMessage: handleServerMessage,
    onStateUpdate: handleStateUpdate
  })

  /**
   * Handle sending user messages
   * Sends message via WebSocket and resets LLM state
   */
  const handleSendMessage = useCallback((content: string) => {
    // Add player message to the message list
    const playerMessage: Message = {
      id: Date.now().toString(),
      role: "player",
      content,
      timestamp: new Date(),
      sender: user?.username || "调查员",
    }
    setMessages((prev) => [...prev, playerMessage])

    // Send message via WebSocket if connected
    if (isConnected) {
      sendMessage(content)
      resetLLM()
    } else {
      // Fallback to mock response when not connected
      toast.warning('WebSocket未连接，使用模拟响应')
      simulateMockResponse(content)
    }
  }, [isConnected, sendMessage, resetLLM, user?.username, simulateMockResponse])

  /**
   * Show error toast when WebSocket connection fails
   */
  useEffect(() => {
    if (error) {
      toast.error(`WebSocket错误: ${error}`)
    }
  }, [error])

  return (
    <div className="flex flex-col h-screen">
      <Header characterName="调查员" />
      <div className="flex-1 flex overflow-hidden">
        <div className="flex-1 flex flex-col min-w-0">
          <MessageList messages={messages} />
          <Footer onSendMessage={handleSendMessage} />
        </div>
        <StatePanel
          character={{
            id: 1,
            name: user?.username || "调查员",
            hp: character.hp,
            maxHp: character.hpMax,
            mp: character.mp,
            maxMp: character.mpMax,
            san: character.san,
            maxSan: character.sanMax,
            luck: character.luck,
          }}
        />
      </div>
    </div>
  )
}
