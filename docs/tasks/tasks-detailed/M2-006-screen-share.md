# M2-006: 实现屏幕共享功能

**任务ID**: M2-006
**标题**: 实现屏幕共享功能
**类型**: fullstack (全栈开发)
**预估工时**: 2.5h
**依赖**: M2-002, M2-003

---

## 任务描述

实现屏幕共享功能，允许 KP 共享游戏地图、场景图等内容给所有玩家。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-006-01 | 设计屏幕共享架构 | Architecture | 20min |
| M2-006-02 | 实现共享者端 | Sharer | 35min |
| M2-006-03 | 实现观看者端 | Viewer | 30min |
| M2-006-04 | 实现信令服务 | Signaling | 30min |
| M2-006-05 | 实现权限控制 | Permissions | 20min |
| M2-006-06 | 实现 WebSocket 通知 | WS Events | 15min |
| M2-006-07 | 编写共享测试 | 测试覆盖 | 15min |

---

## 屏幕共享数据结构

```python
# app/db/models/screen_share.py
from sqlalchemy import Column, String, Boolean, ForeignKey, DateTime
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class ScreenShare(Base):
    """屏幕共享会话"""
    __tablename__ = 'screen_shares'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 共享者
    shared_by = Column(String, ForeignKey('users.id'), nullable=False)

    # WebRTC 相关
    offer_sdp = Column(String)  # SDP offer
    active = Column(Boolean, default=True, nullable=False)

    # 元数据
    source_type = Column(String)  # screen, window, browser
    source_name = Column(String)  # 源名称

    # 时间
    started_at = Column(DateTime, default=func.now(), nullable=False)
    ended_at = Column(DateTime)

    # 关系
    room = relationship("Room", back_populates="screen_shares")
    sharer = relationship("User", back_populates="screen_shares")

    def __repr__(self):
        return f"<ScreenShare {self.id}>"
```

---

## WebRTC 屏幕共享服务

```python
# app/services/webrtc/screen_share.py
from typing import Dict, Any, Optional
from fastapi import WebSocket
import json

class ScreenShareService:
    """屏幕共享服务"""

    def __init__(self):
        self.active_shares: Dict[str, Dict[str, Any]] = {}
        self.viewers: Dict[str, Dict[str, WebSocket]] = {}

    async def start_share(
        self,
        room_id: str,
        user_id: str,
        offer_sdp: str,
        source_type: str = "screen",
    ) -> str:
        """开始屏幕共享"""
        share_id = f"share_{room_id}_{user_id}"

        self.active_shares[share_id] = {
            "room_id": room_id,
            "user_id": user_id,
            "offer_sdp": offer_sdp,
            "source_type": source_type,
            "active": True,
        }

        return share_id

    async def stop_share(self, share_id: str):
        """停止屏幕共享"""
        if share_id in self.active_shares:
            self.active_shares[share_id]["active"] = False

            # 通知所有观看者
            if share_id in self.viewers:
                for viewer_id, ws in self.viewers[share_id].items():
                    try:
                        await ws.send_json({
                            "type": "share_ended",
                            "share_id": share_id,
                        })
                    except:
                        pass

            del self.viewers[share_id]

    async def add_viewer(
        self,
        share_id: str,
        viewer_id: str,
        viewer_ws: WebSocket,
    ):
        """添加观看者"""
        if share_id not in self.viewers:
            self.viewers[share_id] = {}

        self.viewers[share_id][viewer_id] = viewer_ws

        # 返回 offer 给观看者
        share = self.active_shares.get(share_id)
        if share:
            return {
                "share_id": share_id,
                "offer_sdp": share["offer_sdp"],
                "source_type": share["source_type"],
            }

        return None

    async def handle_answer(
        self,
        share_id: str,
        viewer_id: str,
        answer_sdp: str,
    ):
        """处理观看者的 answer"""
        # 转发 answer 给共享者
        share = self.active_shares.get(share_id)
        if share:
            # 通知共享者（这里需要共享者的 WebSocket 连接）
            pass

    async def handle_ice_candidate(
        self,
        share_id: str,
        user_id: str,
        candidate: Dict[str, Any],
        target: str,  # "sharer" or "viewer"
    ):
        """处理 ICE candidate"""
        # 转发 ICE candidate
        if target == "sharer":
            # 发送给共享者
            pass
        else:
            # 发送给指定观看者
            if share_id in self.viewers and user_id in self.viewers[share_id]:
                try:
                    await self.viewers[share_id][user_id].send_json({
                        "type": "ice_candidate",
                        "candidate": candidate,
                    })
                except:
                    pass

    def get_active_share(self, room_id: str) -> Optional[Dict[str, Any]]:
        """获取房间活动的共享"""
        for share_id, share in self.active_shares.items():
            if share["room_id"] == room_id and share["active"]:
                return {
                    "share_id": share_id,
                    "user_id": share["user_id"],
                    "source_type": share["source_type"],
                }
        return None
```

---

## 屏幕共享 API

```python
# app/api/screen_share.py
from fastapi import APIRouter, Depends, WebSocket, WebSocketDisconnect
from sqlalchemy.orm import Session
from pydantic import BaseModel

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.webrtc.screen_share import ScreenShareService
from app.core.security import generate_id

router = APIRouter(prefix="/screen-share", tags=["screen-share"])
share_service = ScreenShareService()

class StartShareRequest(BaseModel):
    room_id: str
    offer_sdp: str
    source_type: str = "screen"
    source_name: str = None

@router.post("/start")
async def start_screen_share(
    request: StartShareRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """开始屏幕共享"""
    # 验证用户在房间中且是 KP
    # TODO: 检查权限

    share_id = await share_service.start_share(
        room_id=request.room_id,
        user_id=current_user.id,
        offer_sdp=request.offer_sdp,
        source_type=request.source_type,
    )

    # 通知房间成员
    # TODO: WebSocket 广播

    return {
        "share_id": share_id,
        "status": "started",
    }

@router.post("/stop/{share_id}")
async def stop_screen_share(
    share_id: str,
    current_user: User = Depends(get_current_user),
):
    """停止屏幕共享"""
    await share_service.stop_share(share_id)

    return {"status": "stopped"}

@router.get("/active/{room_id}")
async def get_active_share(
    room_id: str,
    current_user: User = Depends(get_current_user),
):
    """获取房间活动的屏幕共享"""
    share = share_service.get_active_share(room_id)

    if not share:
        return {"active": False}

    return {
        "active": True,
        "share_id": share["share_id"],
        "shared_by": share["user_id"],
        "source_type": share["source_type"],
    }

@router.websocket("/ws/{room_id}")
async def screen_share_websocket(
    websocket: WebSocket,
    room_id: str,
):
    """屏幕共享 WebSocket"""
    await websocket.accept()

    try:
        while True:
            data = await websocket.receive_json()

            message_type = data.get("type")

            if message_type == "start_share":
                # 开始共享
                share_id = await share_service.start_share(
                    room_id=room_id,
                    user_id=data.get("user_id"),
                    offer_sdp=data.get("offer_sdp"),
                    source_type=data.get("source_type", "screen"),
                )
                await websocket.send_json({
                    "type": "share_started",
                    "share_id": share_id,
                })

            elif message_type == "stop_share":
                # 停止共享
                await share_service.stop_share(data.get("share_id"))

            elif message_type == "join_share":
                # 加入观看
                share_info = await share_service.add_viewer(
                    share_id=data.get("share_id"),
                    viewer_id=data.get("viewer_id"),
                    viewer_ws=websocket,
                )
                await websocket.send_json({
                    "type": "share_joined",
                    "share_info": share_info,
                })

            elif message_type == "answer":
                # 处理 answer
                await share_service.handle_answer(
                    share_id=data.get("share_id"),
                    viewer_id=data.get("viewer_id"),
                    answer_sdp=data.get("answer_sdp"),
                )

            elif message_type == "ice_candidate":
                # 处理 ICE candidate
                await share_service.handle_ice_candidate(
                    share_id=data.get("share_id"),
                    user_id=data.get("user_id"),
                    candidate=data.get("candidate"),
                    target=data.get("target"),
                )

    except WebSocketDisconnect:
        pass
```

---

## 前端屏幕共享组件

```tsx
// frontend/src/components/game/ScreenShare.tsx
import { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Monitor, MonitorOff, Users } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useToast } from '@/hooks/use-toast'

interface ScreenShareProps {
  roomId: string
  isKp: boolean
}

export function ScreenShare({ roomId, isKp }: ScreenShareProps) {
  const [isSharing, setIsSharing] = useState(false)
  const [isViewing, setIsViewing] = useState(false)
  const [viewerCount, setViewerCount] = useState(0)
  const [shareId, setShareId] = useState<string | null>(null)

  const videoRef = useRef<HTMLVideoElement>(null)
  const { user } = useAuth()
  const { toast } = useToast()

  // 检查是否有活动的共享
  useEffect(() => {
    checkActiveShare()

    const ws = new WebSocket(`ws://localhost:8000/api/screen-share/ws/${roomId}`)

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data)

      if (data.type === 'share_started') {
        setShareId(data.share_id)
        if (data.user_id !== user?.id) {
          // 其他人开始共享
          setIsViewing(true)
        }
      } else if (data.type === 'share_ended') {
        setIsViewing(false)
        setShareId(null)
      } else if (data.type === 'viewer_joined') {
        setViewerCount(prev => prev + 1)
      } else if (data.type === 'viewer_left') {
        setViewerCount(prev => prev - 1)
      }
    }

    return () => ws.close()
  }, [roomId])

  const checkActiveShare = async () => {
    try {
      const response = await fetch(`/api/screen-share/active/${roomId}`)
      const data = await response.json()

      if (data.active && data.shared_by !== user?.id) {
        setIsViewing(true)
        setShareId(data.share_id)
      }
    } catch (error) {
      console.error('Failed to check active share:', error)
    }
  }

  const handleStartShare = async () => {
    try {
      // 获取屏幕流
      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: {
          cursor: "always"
        },
        audio: false
      })

      // 创建 WebRTC 连接
      const pc = new RTCPeerConnection({
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' }
        ]
      })

      // 添加轨道
      stream.getTracks().forEach(track => {
        pc.addTrack(track, stream)
      })

      // 创建 offer
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)

      // 发送到服务器
      const response = await fetch('/api/screen-share/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          room_id: roomId,
          offer_sdp: JSON.stringify(pc.localDescription),
          source_type: 'screen',
        }),
      })

      if (!response.ok) throw new Error('启动共享失败')

      const data = await response.json()
      setShareId(data.share_id)
      setIsSharing(true)

      // 显示本地预览
      if (videoRef.current) {
        videoRef.current.srcObject = stream
      }

      toast({
        title: '屏幕共享已启动',
        description: '其他玩家现在可以看到你的屏幕',
      })
    } catch (error) {
      console.error('Failed to start share:', error)
      toast({
        title: '启动失败',
        description: error.message,
        variant: 'destructive',
      })
    }
  }

  const handleStopShare = async () => {
    if (!shareId) return

    try {
      await fetch(`/api/screen-share/stop/${shareId}`, {
        method: 'POST',
      })

      // 停止所有轨道
      if (videoRef.current?.srcObject) {
        const stream = videoRef.current.srcObject as MediaStream
        stream.getTracks().forEach(track => track.stop())
        videoRef.current.srcObject = null
      }

      setIsSharing(false)
      setShareId(null)

      toast({
        title: '屏幕共享已停止',
      })
    } catch (error) {
      console.error('Failed to stop share:', error)
    }
  }

  const handleViewShare = async () => {
    // 观看共享
    // TODO: 实现 WebRTC 连接作为观看者
    toast({
      title: '正在连接...',
      description: '正在建立屏幕共享连接',
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between text-base">
          <span className="flex items-center">
            <Monitor className="h-4 w-4 mr-2" />
            屏幕共享
          </span>
          {isSharing && (
            <span className="text-sm text-muted-foreground flex items-center">
              <Users className="h-3 w-3 mr-1" />
              {viewerCount} 人观看
            </span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent>
        {isSharing ? (
          <div className="space-y-3">
            <video
              ref={videoRef}
              autoPlay
              muted
              playsInline
              className="w-full rounded border bg-black"
            />
            <Button
              variant="destructive"
              size="sm"
              className="w-full"
              onClick={handleStopShare}
            >
              <MonitorOff className="h-4 w-4 mr-2" />
              停止共享
            </Button>
          </div>
        ) : isViewing ? (
          <div className="space-y-3">
            <div className="aspect-video bg-black rounded flex items-center justify-center">
              <video
                ref={videoRef}
                autoPlay
                playsInline
                className="w-full h-full"
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={() => setIsViewing(false)}
            >
              关闭观看
            </Button>
          </div>
        ) : (
          <div className="text-center py-4">
            {isKp ? (
              <Button
                onClick={handleStartShare}
                className="w-full"
              >
                <Monitor className="h-4 w-4 mr-2" />
                开始屏幕共享
              </Button>
            ) : (
              <p className="text-sm text-muted-foreground">
                等待 KP 共享屏幕...
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/screen_share.py` | 创建 | 屏幕共享数据模型 |
| `app/services/webrtc/screen_share.py` | 创建 | 屏幕共享服务 |
| `app/api/screen_share.py` | 创建 | 屏幕共享 API |
| `frontend/src/components/game/ScreenShare.tsx` | 创建 | 屏幕共享组件 |

---

## 验收标准

- [ ] 屏幕共享启动成功
- [ ] 观看者能看到画面
- [ ] 画面延迟低（<500ms）
- [ ] 权限控制有效
- [ ] 共享停止正常
- [ ] 观看者计数正确

---

## 参考文档

- M2-002: WebSocket 事件系统
- M2-003: 视频流同步
- WebRTC Screen Sharing API

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
