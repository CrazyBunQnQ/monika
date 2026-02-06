# M2-003: 实现视频流同步

**任务ID**: M2-003
**标题**: 实现视频流同步
**类型**: backend (后端开发)
**预估工时**: 3h
**依赖**: M2-022

---

## 任务描述

实现视频流的同步和广播功能，支持 KP 视频共享给所有玩家。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-003-01 | 设计视频流架构 | Architecture | 25min |
| M2-003-02 | 实现 WebRTC 信令 | Signaling | 40min |
| M2-003-03 | 实现流转发 | Stream Relay | 35min |
| M2-003-04 | 实现客户端连接 | Client Connect | 30min |
| M2-003-05 | 实现流控制 | Control | 25min |
| M2-003-06 | 编写流测试 | 测试覆盖 | 25min |

---

## WebRTC 信令服务

```python
# app/services/webrtc.py
from typing import Dict, Optional
import json
import uuid
from dataclasses import dataclass

@dataclass
class WebRTCSession:
    """WebRTC 会话"""
    session_id: str
    room_id: str
    host_id: str
    sdp: Optional[str] = None
    candidates: list = None

class WebRTCService:
    """WebRTC 信令服务"""

    def __init__(self, sio):
        self.sio = sio
        self.sessions: Dict[str, WebRTCSession] = {}
        self._setup_handlers()

    def _setup_handlers(self):
        """设置信令处理器"""
        self.sio.on('webrtc_offer', self._handle_offer)
        self.sio.on('webrtc_answer', self._handle_answer)
        self.sio.on('webrtc_ice_candidate', self._handle_ice_candidate)
        self.sio.on('webrtc_join', self._handle_join)
        self.sio.on('webrtc_leave', self._handle_leave)

    async def _handle_offer(self, sid, data):
        """处理 offer"""
        room_id = data.get('room_id')
        sdp = data.get('sdp')

        # 创建会话
        session_id = str(uuid.uuid4())
        session = WebRTCSession(
            session_id=session_id,
            room_id=room_id,
            host_id=sid,
            sdp=sdp,
            candidates=[]
        )
        self.sessions[session_id] = session

        # 通知房间内其他人
        await self.sio.emit('webrtc_offer', {
            'session_id': session_id,
            'sdp': sdp,
            'host_id': sid,
        }, room=room_id, skip_sid=sid)

    async def _handle_answer(self, sid, data):
        """处理 answer"""
        session_id = data.get('session_id')
        sdp = data.get('sdp')

        session = self.sessions.get(session_id)
        if not session:
            return

        # 转发 answer 给 host
        await self.sio.emit('webrtc_answer', {
            'session_id': session_id,
            'sdp': sdp,
            'client_id': sid,
        }, room=session.host_id)

    async def _handle_ice_candidate(self, sid, data):
        """处理 ICE candidate"""
        session_id = data.get('session_id')
        candidate = data.get('candidate')

        session = self.sessions.get(session_id)
        if not session:
            return

        # 判断是 host 还是 client
        if sid == session.host_id:
            # 转发给房间内其他人
            await self.sio.emit('webrtc_ice_candidate', {
                'session_id': session_id,
                'candidate': candidate,
            }, room=session.room_id, skip_sid=sid)
        else:
            # 转发给 host
            await self.sio.emit('webrtc_ice_candidate', {
                'session_id': session_id,
                'candidate': candidate,
                'client_id': sid,
            }, room=session.host_id)

    async def _handle_join(self, sid, data):
        """处理加入视频流"""
        room_id = data.get('room_id')

        # 加入 Socket.io 房间
        self.sio.enter_room(sid, f"video_{room_id}")

        # 通知 host 有新观众
        await self.sio.emit('webrtc_viewer_joined', {
            'viewer_id': sid,
        }, room=f"video_{room_id}")

    async def _handle_leave(self, sid, data):
        """处理离开视频流"""
        room_id = data.get('room_id')

        # 离开 Socket.io 房间
        self.sio.leave_room(sid, f"video_{room_id}")

        # 通知 host
        await self.sio.emit('webrtc_viewer_left', {
            'viewer_id': sid,
        }, room=f"video_{room_id}")
```

---

## 前端 WebRTC 客户端

```tsx
// frontend/src/hooks/useWebRTC.ts
import { useEffect, useRef, useState } from 'react'
import { socket } from '@/lib/socket'

interface WebRTCConfig {
  roomId: string
  isHost: boolean
  onStream?: (stream: MediaStream) => void
}

export function useWebRTC({ roomId, isHost, onStream }: WebRTCConfig) {
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  const STUN_SERVERS = [
    'stun:stun.l.google.com:19302',
    'stun:stun1.l.google.com:19302',
  ]

  useEffect(() => {
    if (isHost) {
      setupHost()
    } else {
      setupViewer()
    }

    return () => {
      cleanup()
    }
  }, [roomId, isHost])

  // Host 设置
  const setupHost = async () => {
    try {
      // 获取本地流
      const stream = await navigator.mediaDevices.getUserMedia({
        video: true,
        audio: true,
      })
      localStreamRef.current = stream

      // 创建 PeerConnection
      const pc = new RTCPeerConnection({
        iceServers: STUN_SERVERS.map(server => ({ urls: server })),
      })

      peerConnectionRef.current = pc

      // 添加本地流
      stream.getTracks().forEach(track => {
        pc.addTrack(track, stream)
      })

      // 监听 ICE candidates
      pc.onicecandidate = (event) => {
        if (event.candidate) {
          socket.emit('webrtc_ice_candidate', {
            room_id: roomId,
            candidate: event.candidate,
          })
        }
      }

      socket.on('webrtc_answer', handleAnswer)
      socket.on('webrtc_ice_candidate', handleIceCandidate)
      socket.on('webrtc_viewer_joined', handleViewerJoined)

    } catch (error) {
      console.error('Failed to setup host:', error)
    }
  }

  // Viewer 设置
  const setupViewer = async () => {
    try {
      const pc = new RTCPeerConnection({
        iceServers: STUN_SERVERS.map(server => ({ urls: server })),
      })

      peerConnectionRef.current = pc

      // 监听远程流
      pc.ontrack = (event) => {
        if (event.streams[0]) {
          onStream?.(event.streams[0])
          setIsConnected(true)
        }
      }

      // 监听 ICE candidates
      pc.onicecandidate = (event) => {
        if (event.candidate) {
          socket.emit('webrtc_ice_candidate', {
            room_id: roomId,
            candidate: event.candidate,
          })
        }
      }

      socket.on('webrtc_offer', handleOffer)
      socket.on('webrtc_ice_candidate', handleIceCandidate)

      // 通知加入
      socket.emit('webrtc_join', { room_id: roomId })

    } catch (error) {
      console.error('Failed to setup viewer:', error)
    }
  }

  // 处理 offer (viewer)
  const handleOffer = async (data: any) => {
    const pc = peerConnectionRef.current
    if (!pc) return

    await pc.setRemoteDescription(new RTCSessionDescription(data.sdp))

    const answer = await pc.createAnswer()
    await pc.setLocalDescription(answer)

    socket.emit('webrtc_answer', {
      session_id: data.session_id,
      sdp: answer,
    })
  }

  // 处理 answer (host)
  const handleAnswer = async (data: any) => {
    const pc = peerConnectionRef.current
    if (!pc) return

    await pc.setRemoteDescription(new RTCSessionDescription(data.sdp))
  }

  // 处理 ICE candidate
  const handleIceCandidate = async (data: any) => {
    const pc = peerConnectionRef.current
    if (!pc || !data.candidate) return

    await pc.addIceCandidate(new RTCIceCandidate(data.candidate))
  }

  // 新观众加入 (host)
  const handleViewerJoined = async (data: any) => {
    const pc = peerConnectionRef.current
    if (!pc) return

    // 为新观众创建 offer
    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)

    socket.emit('webrtc_offer', {
      room_id: roomId,
      sdp: offer,
    })
  }

  // 清理
  const cleanup = () => {
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach(track => track.stop())
    }

    if (peerConnectionRef.current) {
      peerConnectionRef.current.close()
    }

    socket.emit('webrtc_leave', { room_id: roomId })

    socket.off('webrtc_offer')
    socket.off('webrtc_answer')
    socket.off('webrtc_ice_candidate')
    socket.off('webrtc_viewer_joined')
    socket.off('webrtc_viewer_left')
  }

  return {
    isConnected,
    localStream: localStreamRef.current,
  }
}
```

---

## 视频播放组件

```tsx
// frontend/src/components/game/VideoStream.tsx
import { useEffect, useRef } from 'react'
import { useWebRTC } from '@/hooks/useWebRTC'

interface VideoStreamProps {
  roomId: string
  isHost: boolean
}

export function VideoStream({ roomId, isHost }: VideoStreamProps) {
  const localVideoRef = useRef<HTMLVideoElement>(null)
  const remoteVideoRef = useRef<HTMLVideoElement>(null)

  const { isConnected, localStream } = useWebRTC({
    roomId,
    isHost,
    onStream: (stream) => {
      if (remoteVideoRef.current) {
        remoteVideoRef.current.srcObject = stream
      }
    },
  })

  // 显示本地流（仅 host）
  useEffect(() => {
    if (isHost && localStream && localVideoRef.current) {
      localVideoRef.current.srcObject = localStream
    }
  }, [isHost, localStream])

  return (
    <div className="space-y-4">
      {isHost && (
        <div>
          <h3 className="text-sm font-medium mb-2">你的画面</h3>
          <video
            ref={localVideoRef}
            autoPlay
            muted
            playsInline
            className="w-full rounded-lg bg-black aspect-video"
          />
        </div>
      )}

      <div>
        <h3 className="text-sm font-medium mb-2">
          {isHost ? '预览' : 'KP 画面'}
        </h3>
        <video
          ref={remoteVideoRef}
          autoPlay
          playsInline
          className="w-full rounded-lg bg-black aspect-video"
        />
      </div>

      {!isConnected && !isHost && (
        <div className="text-center text-sm text-muted-foreground">
          等待 KP 开始视频流...
        </div>
      )}
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/webrtc.py` | 创建 | WebRTC 服务 |
| `frontend/src/hooks/useWebRTC.ts` | 创建 | WebRTC Hook |
| `frontend/src/components/game/VideoStream.tsx` | 创建 | 视频流组件 |
| `frontend/src/lib/socket.ts` | 创建 | Socket.io 客户端 |

---

## 验收标准

- [ ] 视频流建立成功
- [ ] 音视频同步正常
- [ ] 多人观看支持
- [ ] ICE 穿透有效
- [ ] 断线重连可用

---

## 参考文档

- M2-022: Socket.io 服务配置
- WebRTC API 文档
- STUN/TURN 协议

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
