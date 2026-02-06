# M2-010: 实现房间录制功能

**任务ID**: M2-010
**标题**: 实现房间录制功能
**类型**: fullstack (全栈开发)
**预估工时**: 3h
**依赖**: M2-001

---

## 任务描述

实现房间录制功能，支持录制游戏会话的视频/音频，包括多方通话、屏幕共享等内容，便于会话回顾和分享。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-010-01 | 设计录制系统架构 | Recording Architecture | 30min |
| M2-010-02 | 实现媒体流录制 | Stream Recording | 45min |
| M2-010-03 | 实现多路混音 | Audio Mixing | 35min |
| M2-010-04 | 实现录制控制 | Recording Controls | 30min |
| M2-010-05 | 实现录制存储 | Storage | 25min |
| M2-010-06 | 实现录制回放 | Playback | 30min |
| M2-010-07 | 编写测试 | 测试覆盖 | 25min |

---

## 录制配置模型

```python
# app/db/models/recording.py
from sqlalchemy import Column, String, Integer, Text, ForeignKey, Boolean, JSON, DateTime
from sqlalchemy.orm import relationship
from app.db.database import Base
from datetime import datetime

class RoomRecording(Base):
    """房间录制"""
    __tablename__ = 'room_recordings'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 录制信息
    started_by = Column(String, ForeignKey('users.id'), nullable=False)
    started_at = Column(DateTime, default=datetime.utcnow)
    ended_at = Column(DateTime)
    duration_seconds = Column(Integer)

    # 录制配置
    record_video = Column(Boolean, default=True)
    record_audio = Column(Boolean, default=True)
    record_screen_shares = Column(Boolean, default=True)
    record_chat = Column(Boolean, default=True)

    # 输出配置
    output_format = Column(String, default='webm')  # webm, mp4
    video_quality = Column(String, default='medium')  # low, medium, high
    audio_bitrate = Column(Integer, default=128)

    # 文件信息
    file_path = Column(String)
    file_size = Column(Integer)
    thumbnail_path = Column(String)

    # 元数据
    participants = Column(JSON)  # 参与者列表
    tags = Column(JSON)
    description = Column(Text)

    # 权限
    is_public = Column(Boolean, default=False)
    download_allowed = Column(Boolean, default=True)

    # 状态
    status = Column(String, default='recording')  # recording, processing, completed, failed

    def __repr__(self):
        return f"<RoomRecording {self.id}>"

class RecordingHighlight(Base):
    """录制高光时刻"""
    __tablename__ = 'recording_highlights'

    id = Column(String, primary_key=True, index=True)
    recording_id = Column(String, ForeignKey('room_recordings.id'), nullable=False)

    # 时间信息
    start_time = Column(Integer, nullable=False)  # 从开始秒数
    end_time = Column(Integer, nullable=False)
    title = Column(String)
    description = Column(Text)
    thumbnail_path = Column(String)

    # 创建者
    created_by = Column(String, ForeignKey('users.id'))
    created_at = Column(DateTime, default=datetime.utcnow)

    # 关系
    recording = relationship("RoomRecording", back_populates="highlights")
```

---

## 录制服务

```python
# app/services/recording.py
from typing import Dict, Any, Optional
from sqlalchemy.orm import Session
import os
import asyncio
from pathlib import Path

from app.db.models.recording import RoomRecording
from app.core.security import generate_id
from app.core.config import settings

class RecordingService:
    """录制服务"""

    def __init__(self, db: Session):
        self.db = db
        self.recordings_dir = Path(settings.RECORDINGS_DIR)
        self.recordings_dir.mkdir(parents=True, exist_ok=True)
        self.active_recordings = {}

    async def start_recording(
        self,
        room_id: str,
        user_id: str,
        config: Dict[str, Any] = None,
    ) -> Dict[str, Any]:
        """开始录制"""
        # 检查是否已有活跃录制
        if room_id in self.active_recordings:
            return {
                "success": False,
                "error": "房间已有活跃录制",
            }

        # 创建录制记录
        recording = RoomRecording(
            id=generate_id('recording'),
            room_id=room_id,
            started_by=user_id,
            record_video=config.get('record_video', True),
            record_audio=config.get('record_audio', True),
            record_screen_shares=config.get('record_screen_shares', True),
            record_chat=config.get('record_chat', True),
            output_format=config.get('output_format', 'webm'),
            video_quality=config.get('video_quality', 'medium'),
            audio_bitrate=config.get('audio_bitrate', 128),
        )

        self.db.add(recording)
        self.db.commit()
        self.db.refresh(recording)

        # 创建输出文件
        output_path = self.recordings_dir / f"{recording.id}.{recording.output_format}"

        # 启动录制
        self.active_recordings[room_id] = {
            "recording_id": recording.id,
            "output_path": str(output_path),
            "started_at": recording.started_at,
            "config": config or {},
        }

        # 通知房间成员
        # TODO: WebSocket 通知

        return {
            "success": True,
            "recording_id": recording.id,
            "started_at": recording.started_at.isoformat(),
        }

    async def stop_recording(
        self,
        room_id: str,
        user_id: str,
    ) -> Dict[str, Any]:
        """停止录制"""
        if room_id not in self.active_recordings:
            return {
                "success": False,
                "error": "没有活跃录制",
            }

        recording_info = self.active_recordings[room_id]
        recording_id = recording_info['recording_id']

        # 获取录制记录
        recording = self.db.query(RoomRecording)\
            .filter(RoomRecording.id == recording_id)\
            .first()

        if not recording:
            return {
                "success": False,
                "error": "录制不存在",
            }

        # 检查权限
        if recording.started_by != user_id:
            return {
                "success": False,
                "error": "只有发起者可以停止录制",
            }

        # 停止录制
        ended_at = datetime.utcnow()
        duration = int((ended_at - recording.started_at).total_seconds())

        recording.ended_at = ended_at
        recording.duration_seconds = duration
        recording.status = 'processing'

        self.db.commit()

        # 从活跃录制中移除
        del self.active_recordings[room_id]

        # 处理录制文件
        await self._process_recording(recording)

        return {
            "success": True,
            "recording_id": recording.id,
            "duration": duration,
        }

    async def _process_recording(
        self,
        recording: RoomRecording,
    ):
        """处理录制文件"""
        # 这里可以进行后处理，如生成缩略图、转码等
        output_path = Path(recording.output_path) if recording.output_path else None

        if output_path and output_path.exists():
            # 获取文件大小
            recording.file_size = output_path.stat().st_size
            recording.status = 'completed'
            self.db.commit()

        # TODO: 生成缩略图
        # TODO: 可选的转码

    def get_room_recordings(
        self,
        room_id: str,
        limit: int = 50,
    ) -> list:
        """获取房间录制列表"""
        recordings = self.db.query(RoomRecording)\
            .filter(RoomRecording.room_id == room_id)\
            .order_by(RoomRecording.started_at.desc())\
            .limit(limit)\
            .all()

        return [
            {
                "id": r.id,
                "started_at": r.started_at.isoformat(),
                "duration": r.duration_seconds,
                "file_size": r.file_size,
                "status": r.status,
                "is_public": r.is_public,
            }
            for r in recordings
        ]

    def get_recording(
        self,
        recording_id: str,
    ) -> Optional[Dict[str, Any]]:
        """获取录制详情"""
        recording = self.db.query(RoomRecording)\
            .filter(RoomRecording.id == recording_id)\
            .first()

        if not recording:
            return None

        return {
            "id": recording.id,
            "room_id": recording.room_id,
            "started_at": recording.started_at.isoformat(),
            "ended_at": recording.ended_at.isoformat() if recording.ended_at else None,
            "duration": recording.duration_seconds,
            "file_size": recording.file_size,
            "thumbnail_path": recording.thumbnail_path,
            "participants": recording.participants,
            "description": recording.description,
            "status": recording.status,
        }

    def delete_recording(
        self,
        recording_id: str,
        user_id: str,
    ) -> bool:
        """删除录制"""
        recording = self.db.query(RoomRecording)\
            .filter(RoomRecording.id == recording_id)\
            .first()

        if not recording:
            return False

        # 检查权限
        if recording.started_by != user_id:
            return False

        # 删除文件
        if recording.file_path:
            file_path = Path(recording.file_path)
            if file_path.exists():
                file_path.unlink()

        # 删除缩略图
        if recording.thumbnail_path:
            thumbnail_path = Path(recording.thumbnail_path)
            if thumbnail_path.exists():
                thumbnail_path.unlink()

        # 删除数据库记录
        self.db.delete(recording)
        self.db.commit()

        return True
```

---

## 录制 API

```python
# app/api/recording.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.api.deps.permissions import require_room_role
from app.db.models.user import User
from app.services.recording import RecordingService

router = APIRouter(prefix="/recording", tags=["recording"])

class StartRecordingRequest(BaseModel):
    room_id: str
    record_video: bool = True
    record_audio: bool = True
    record_screen_shares: bool = True
    record_chat: bool = True
    output_format: str = 'webm'
    video_quality: str = 'medium'
    audio_bitrate: int = 128

@router.post("/start")
async def start_recording(
    request: StartRecordingRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """开始录制"""
    service = RecordingService(db)

    result = await service.start_recording(
        request.room_id,
        current_user.id,
        request.dict(),
    )

    if not result["success"]:
        raise HTTPException(status_code=400, detail=result["error"])

    return result

@router.post("/stop")
async def stop_recording(
    room_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """停止录制"""
    service = RecordingService(db)

    result = await service.stop_recording(room_id, current_user.id)

    if not result["success"]:
        raise HTTPException(status_code=400, detail=result["error"])

    return result

@router.get("/room/{room_id}")
async def get_room_recordings(
    room_id: str,
    limit: int = 50,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间录制列表"""
    service = RecordingService(db)
    recordings = service.get_room_recordings(room_id, limit)

    return {"recordings": recordings}

@router.get("/{recording_id}")
async def get_recording(
    recording_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取录制详情"""
    service = RecordingService(db)
    recording = service.get_recording(recording_id)

    if not recording:
        raise HTTPException(status_code=404, detail="录制不存在")

    return recording

@router.delete("/{recording_id}")
async def delete_recording(
    recording_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除录制"""
    service = RecordingService(db)
    success = service.delete_recording(recording_id, current_user.id)

    if not success:
        raise HTTPException(status_code=404, detail="录制不存在或无权删除")

    return {"message": "录制已删除"}
```

---

## 前端录制组件

```tsx
// frontend/src/components/recording/RecordingControls.tsx
import { useState, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Record, Stop, Circle } from 'lucide-react'

interface RecordingControlsProps {
  roomId: string
}

export function RecordingControls({ roomId }: RecordingControlsProps) {
  const [isRecording, setIsRecording] = useState(false)
  const [recordingTime, setRecordingTime] = useState(0)
  const [showSettings, setShowSettings] = useState(false)
  const [config, setConfig] = useState({
    record_video: true,
    record_audio: true,
    record_screen_shares: true,
    record_chat: true,
    video_quality: 'medium',
  })

  const timerRef = useRef<NodeJS.Timeout>()

  useEffect(() => {
    // 检查当前是否有活跃录制
    checkRecordingStatus()
  }, [roomId])

  const checkRecordingStatus = async () => {
    try {
      const response = await fetch(`/api/recording/room/${roomId}`)
      if (response.ok) {
        const data = await response.json()
        const activeRecording = data.recordings.find((r: any) => r.status === 'recording')
        setIsRecording(!!activeRecording)
      }
    } catch (error) {
      console.error('Failed to check recording status:', error)
    }
  }

  const startRecording = async () => {
    try {
      const response = await fetch('/api/recording/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          room_id: roomId,
          ...config,
        }),
      })

      if (response.ok) {
        setIsRecording(true)
        startTimer()
      }
    } catch (error) {
      console.error('Failed to start recording:', error)
    }
  }

  const stopRecording = async () => {
    try {
      const response = await fetch('/api/recording/stop', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ room_id: roomId }),
      })

      if (response.ok) {
        setIsRecording(false)
        stopTimer()
        setRecordingTime(0)
      }
    } catch (error) {
      console.error('Failed to stop recording:', error)
    }
  }

  const startTimer = () => {
    timerRef.current = setInterval(() => {
      setRecordingTime((prev) => prev + 1)
    }, 1000)
  }

  const stopTimer = () => {
    if (timerRef.current) {
      clearInterval(timerRef.current)
    }
  }

  const formatTime = (seconds: number) => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
    }
    return `${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  return (
    <Dialog open={showSettings} onOpenChange={setShowSettings}>
      <div className="flex items-center gap-2">
        {isRecording && (
          <div className="flex items-center gap-2 text-red-500">
            <Circle className="h-3 w-3 animate-pulse fill-current" />
            <span className="text-sm font-mono">{formatTime(recordingTime)}</span>
          </div>
        )}

        <DialogTrigger asChild>
          <Button
            variant={isRecording ? "destructive" : "outline"}
            size="sm"
            onClick={() => {
              if (isRecording) {
                stopRecording()
              }
            }}
          >
            {isRecording ? (
              <>
                <Stop className="h-4 w-4 mr-2" />
                停止录制
              </>
            ) : (
              <>
                <Record className="h-4 w-4 mr-2" />
                开始录制
              </>
            )}
          </Button>
        </DialogTrigger>
      </div>

      <DialogContent>
        <DialogHeader>
          <DialogTitle>录制设置</DialogTitle>
          <DialogDescription>
            配置录制的选项
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="record-video">录制视频</Label>
            <Switch
              id="record-video"
              checked={config.record_video}
              onCheckedChange={(checked) =>
                setConfig({ ...config, record_video: checked })
              }
            />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="record-audio">录制音频</Label>
            <Switch
              id="record-audio"
              checked={config.record_audio}
              onCheckedChange={(checked) =>
                setConfig({ ...config, record_audio: checked })
              }
            />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="record-screen">录制屏幕共享</Label>
            <Switch
              id="record-screen"
              checked={config.record_screen_shares}
              onCheckedChange={(checked) =>
                setConfig({ ...config, record_screen_shares: checked })
              }
            />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="record-chat">录制聊天</Label>
            <Switch
              id="record-chat"
              checked={config.record_chat}
              onCheckedChange={(checked) =>
                setConfig({ ...config, record_chat: checked })
              }
            />
          </div>

          <Button
            onClick={() => {
              setShowSettings(false)
              startRecording()
            }}
            className="w-full"
          >
            开始录制
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
```

---

## 录制列表组件

```tsx
// frontend/src/components/recording/RecordingList.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Play, Download, Trash2, Clock } from 'lucide-react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

interface Recording {
  id: string
  started_at: string
  duration: number
  file_size: number
  status: string
}

interface RecordingListProps {
  roomId: string
}

export function RecordingList({ roomId }: RecordingListProps) {
  const [recordings, setRecordings] = useState<Recording[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchRecordings()
  }, [roomId])

  const fetchRecordings = async () => {
    try {
      const response = await fetch(`/api/recording/room/${roomId}`)
      if (response.ok) {
        const data = await response.json()
        setRecordings(data.recordings)
      }
    } catch (error) {
      console.error('Failed to fetch recordings:', error)
    } finally {
      setLoading(false)
    }
  }

  const deleteRecording = async (recordingId: string) => {
    try {
      const response = await fetch(`/api/recording/${recordingId}`, {
        method: 'DELETE',
      })

      if (response.ok) {
        fetchRecordings()
      }
    } catch (error) {
      console.error('Failed to delete recording:', error)
    }
  }

  const formatDuration = (seconds: number) => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) {
      return `${hours}时${minutes}分${secs}秒`
    }
    return `${minutes}分${secs}秒`
  }

  const formatFileSize = (bytes: number) => {
    const mb = bytes / (1024 * 1024)
    return `${mb.toFixed(1)} MB`
  }

  if (loading) {
    return <div>加载中...</div>
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>录制记录</CardTitle>
      </CardHeader>
      <CardContent>
        {recordings.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">暂无录制记录</p>
        ) : (
          <div className="space-y-3">
            {recordings.map((recording) => (
              <div
                key={recording.id}
                className="flex items-center justify-between p-3 rounded-lg border"
              >
                <div className="flex items-center gap-3">
                  <Clock className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <div className="font-medium">
                      {new Date(recording.started_at).toLocaleString()}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      时长: {formatDuration(recording.duration)}
                      {' · '}
                      大小: {formatFileSize(recording.file_size)}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {recording.status === 'completed' && (
                    <>
                      <Button size="sm" variant="ghost">
                        <Play className="h-4 w-4" />
                      </Button>
                      <Button size="sm" variant="ghost">
                        <Download className="h-4 w-4" />
                      </Button>
                    </>
                  )}

                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button size="sm" variant="ghost">
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>确认删除</AlertDialogTitle>
                        <AlertDialogDescription>
                          删除后无法恢复，确定要删除这个录制吗？
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => deleteRecording(recording.id)}
                          className="bg-destructive text-destructive-foreground"
                        >
                          删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            ))}
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
| `app/db/models/recording.py` | 创建 | 录制数据模型 |
| `app/services/recording.py` | 创建 | 录制服务 |
| `app/api/recording.py` | 创建 | 录制 API |
| `frontend/src/components/recording/RecordingControls.tsx` | 创建 | 录制控制组件 |
| `frontend/src/components/recording/RecordingList.tsx` | 创建 | 录制列表组件 |

---

## 验收标准

- [ ] 录制启动/停止正常
- [ ] 音视频混合正确
- [ ] 文件存储完整
- [ ] 回放功能可用
- [ ] 权限控制有效
- [ ] 文件大小合理

---

## 参考文档

- M2-001: 房间管理系统
- M2-006: 屏幕共享功能
- MediaRecorder API

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
