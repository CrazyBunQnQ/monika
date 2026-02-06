# M2-004: 实现语音聊天功能

**任务ID**: M2-004
**标题**: 实现语音聊天功能
**类型**: backend (后端开发)
**预估工时**: 3h
**依赖**: M2-022

---

## 任务描述

实现基于 WebRTC 的语音聊天功能，支持房间内语音通话。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-004-01 | 设计语音架构 | Architecture | 25min |
| M2-004-02 | 实现 WebRTC 音频 | Audio Stream | 40min |
| M2-004-03 | 实现混音器 | Audio Mixer | 35min |
| M2-004-04 | 实现静音/取消静音 | Mute Control | 25min |
| M2-004-05 | 实现语音活动检测 | VAD | 30min |
| M2-004-06 | 编写语音测试 | 测试覆盖 | 25min |

---

## 语音聊天服务

```python
# app/services/voice_chat.py
from typing import Dict, List, Optional
from fastapi import WebSocket

class VoiceChatService:
    """语音聊天服务"""

    def __init__(self):
        self.rooms: Dict[str, Dict[str, WebSocket]] = {}
        self.peer_connections: Dict[str, Dict[str, any]] = {}

    async def join_voice_room(
        self,
        room_id: str,
        user_id: str,
        websocket: WebSocket,
    ):
        """加入语音房间"""
        if room_id not in self.rooms:
            self.rooms[room_id] = {}

        self.rooms[room_id][user_id] = websocket

        # 通知房间内其他人
        for uid, ws in self.rooms[room_id].items():
            if uid != user_id:
                await ws.send_json({
                    "type": "user_joined",
                    "user_id": user_id,
                })

    async def leave_voice_room(
        self,
        room_id: str,
        user_id: str,
    ):
        """离开语音房间"""
        if room_id not in self.rooms:
            return

        if user_id in self.rooms[room_id]:
            del self.rooms[room_id][user_id]

        # 通知房间内其他人
        for uid, ws in self.rooms[room_id].items():
            await ws.send_json({
                "type": "user_left",
                "user_id": user_id,
            })

        # 清理空房间
        if not self.rooms[room_id]:
            del self.rooms[room_id]

    async def toggle_mute(
        self,
        room_id: str,
        user_id: str,
        muted: bool,
    ):
        """切换静音状态"""
        if room_id not in self.rooms:
            return

        # 通知房间内所有人
        for uid, ws in self.rooms[room_id].items():
            await ws.send_json({
                "type": "user_muted",
                "user_id": user_id,
                "muted": muted,
            })

    async def handle_audio_data(
        self,
        room_id: str,
        user_id: str,
        audio_data: bytes,
    ):
        """转发音频数据"""
        if room_id not in self.rooms:
            return

        # 转发给房间内其他人
        for uid, ws in self.rooms[room_id].items():
            if uid != user_id:
                try:
                    await ws.send_bytes(audio_data)
                except:
                    # 连接可能已关闭
                    pass

    def get_participants(self, room_id: str) -> List[str]:
        """获取房间参与者"""
        if room_id not in self.rooms:
            return []
        return list(self.rooms[room_id].keys())

    def is_user_in_room(self, room_id: str, user_id: str) -> bool:
        """检查用户是否在语音房间"""
        return room_id in self.rooms and user_id in self.rooms[room_id]
```

---

## WebSocket 语音端点

```python
# app/api/voice.py
from fastapi import APIRouter, WebSocket, WebSocketDisconnect, Depends
from sqlalchemy.orm import Session

from app.services.voice_chat import VoiceChatService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/voice", tags=["voice"])
voice_service = VoiceChatService()

@router.websocket("/room/{room_id}")
async def voice_websocket(
    websocket: WebSocket,
    room_id: str,
    token: str,
    db: Session = Depends(get_db),
):
    """语音聊天 WebSocket"""
    # 验证用户
    user = await _get_user_from_token(token, db)
    if not user:
        await websocket.close(code=1008)
        return

    await websocket.accept()

    try:
        # 加入语音房间
        await voice_service.join_voice_room(
            room_id,
            user.id,
            websocket
        )

        while True:
            data = await websocket.receive()

            # 处理音频数据
            if isinstance(data, bytes):
                await voice_service.handle_audio_data(
                    room_id,
                    user.id,
                    data,
                )
            # 处理控制消息
            elif isinstance(data, str):
                message = json.loads(data)

                if message["type"] == "toggle_mute":
                    await voice_service.toggle_mute(
                        room_id,
                        user.id,
                        message["muted"],
                    )

    except WebSocketDisconnect:
        await voice_service.leave_voice_room(room_id, user.id)
```

---

## 前端语音聊天 Hook

```tsx
// frontend/src/hooks/useVoiceChat.ts
import { useEffect, useRef, useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'

interface VoiceChatOptions {
  roomId: string
}

export function useVoiceChat({ roomId }: VoiceChatOptions) {
  const { user } = useAuth()
  const [isConnected, setIsConnected] = useState(false)
  const [isMuted, setIsMuted] = useState(false)
  const [participants, setParticipants] = useState<string[]>([])

  const wsRef = useRef<WebSocket | null>(null)
  const audioContextRef = useRef<AudioContext | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)

  useEffect(() => {
    connect()
    return () => {
      disconnect()
    }
  }, [roomId])

  const connect = async () => {
    try {
      // 获取麦克风权限
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
        video: false,
      })

      mediaStreamRef.current = stream

      // 创建音频上下文
      const audioContext = new AudioContext()
      audioContextRef.current = audioContext

      // 创建 WebSocket 连接
      const ws = new WebSocket(
        `wss://api.example.com/voice/room/${roomId}?token=${localStorage.getItem('access_token')}`
      )

      ws.binaryType = 'arraybuffer'

      ws.onopen = () => {
        setIsConnected(true)
      }

      ws.onmessage = async (event) => {
        if (event.data instanceof ArrayBuffer) {
          // 接收音频数据
          const audioData = event.data
          await playAudio(audioData)
        } else {
          // 控制消息
          const message = JSON.parse(event.data)
          handleControlMessage(message)
        }
      }

      ws.onclose = () => {
        setIsConnected(false)
      }

      wsRef.current = ws

      // 发送音频数据
      const processor = audioContext.createScriptProcessor(1024, 1, 1)
      const source = audioContext.createMediaStreamSource(stream)
      source.connect(processor)

      processor.connect(audioContext.destination)

      processor.onaudioprocess = (e) => {
        if (ws.readyState === WebSocket.OPEN && !isMuted) {
          const audioData = e.inputBuffer.getChannelData(0)
          ws.send(audioData.buffer)
        }
      }

    } catch (error) {
      console.error('Voice chat connection failed:', error)
    }
  }

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close()
    }

    if (mediaStreamRef.current) {
      mediaStreamRef.current.getTracks().forEach(track => track.stop())
    }

    setIsConnected(false)
  }

  const toggleMute = () => {
    const newMuted = !isMuted
    setIsMuted(newMuted)

    if (wsRef.current) {
      wsRef.current.send(JSON.stringify({
        type: 'toggle_mute',
        muted: newMuted,
      }))
    }
  }

  const playAudio = async (arrayBuffer: ArrayBuffer) => {
    if (!audioContextRef.current) return

    const audioContext = audioContextRef.current

    const audioBuffer = await audioContext.decodeAudioData(arrayBuffer)
    const source = audioContext.createBufferSource()
    source.buffer = audioBuffer
    source.connect(audioContext.destination)
    source.start()
  }

  const handleControlMessage = (message: any) => {
    switch (message.type) {
      case 'user_joined':
        setParticipants([...participants, message.user_id])
        break
      case 'user_left':
        setParticipants(participants.filter(id => id !== message.user_id))
        break
      case 'user_muted':
        // 更新静音状态
        break
    }
  }

  return {
    isConnected,
    isMuted,
    participants,
    connect,
    disconnect,
    toggleMute,
  }
```

---

## 语音控制组件

```tsx
// frontend/src/components/game/VoiceControl.tsx
import { useVoiceChat } from '@/hooks/useVoiceChat'

interface VoiceControlProps {
  roomId: string
}

export function VoiceControl({ roomId }: VoiceControlProps) {
  const { isConnected, isMuted, participants, connect, disconnect, toggleMute } = useVoiceChat({ roomId })

  return (
    <div className="space-y-4">
      {/* 连接状态 */}
      <div className="flex items-center justify-between">
        <span className="text-sm">语音聊天</span>
        <span className={`text-xs ${isConnected ? 'text-green-500' : 'text-gray-500'}`}>
          {isConnected ? '已连接' : '未连接'}
        </span>
      </div>

      {/* 参与者 */}
      {isConnected && participants.length > 0 && (
        <div className="space-y-2">
          <span className="text-sm font-medium">参与者 ({participants.length})</span>
          <div className="flex flex-wrap gap-2">
            {participants.map((id) => (
              <div key={id} className="text-sm bg-muted px-2 py-1 rounded">
                {id}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 控制按钮 */}
      <div className="flex space-x-2">
        {!isConnected ? (
          <button
            onClick={connect}
            className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90"
          >
            连接语音
          </button>
        ) : (
          <>
            <button
              onClick={toggleMute}
              className={`px-4 py-2 rounded ${
                isMuted
                  ? 'bg-red-500 text-white'
                  : 'bg-secondary text-secondary-foreground'
              }`}
            >
              {isMuted ? '已静音' : '静音'}
            </button>
            <button
              onClick={disconnect}
              className="flex-1 px-4 py-2 bg-destructive text-destructive-foreground rounded hover:bg-destructive/90"
            >
              断开连接
            </button>
          </>
        )}
      </div>
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/voice_chat.py` | 创建 | 语音聊天服务 |
| `app/api/voice.py` | 创建 | 语音 WebSocket 端点 |
| `frontend/src/hooks/useVoiceChat.ts` | 创建 | 语音聊天 Hook |
| `frontend/src/components/game/VoiceControl.tsx` | 创建 | 语音控制组件 |

---

## 验收标准

- [ ] 语音连接稳定
- [ ] 音质清晰
- [ ] 静音功能正常
- [ ] 参与者列表准确
- [ ] 断线重连可用
- [ ] 延迟在可接受范围内

---

## 参考文档

- M2-002: WebSocket 事件系统
- M2-003: 视频流同步
- WebRTC API 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
