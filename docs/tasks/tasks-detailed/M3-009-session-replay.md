# M3-009: 实现会话回放功能

**任务ID**: M3-009
**标题**: 实现会话回放功能
**类型**: backend + frontend (全栈开发)
**预估工时**: 6h
**依赖**: M3-006, M1-080

---

## 任务描述

实现游戏会话的回放功能，允许玩家和 KP 按时间轴回放历史会话中的所有事件，包括聊天消息、骰子检定、状态变化等，支持播放控制和速度调节。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-009-01 | 设计回放数据结构 | 定义 ReplaySession 和 ReplayEvent | 30min |
| M3-009-02 | 实现回放数据生成服务 | 从事件日志生成回放数据 | 1h |
| M3-009-03 | 实现回放查询 API | 获取回放数据的接口 | 45min |
| M3-009-04 | 实现回放播放器组件 | React 播放器 UI | 1.5h |
| M3-009-05 | 实现播放控制逻辑 | 播放/暂停/进度控制 | 1h |
| M3-009-06 | 实现速度调节功能 | 0.5x/1x/2x 速度 | 30min |
| M3-009-07 | 实现事件过滤功能 | 按类型过滤事件 | 30min |
| M3-009-08 | 编写回放测试 | 单元测试和集成测试 | 45min |

---

## 后端代码示例

### 回放数据模型

```python
# app/db/models/replay.py
from sqlalchemy import Column, String, DateTime, ForeignKey, JSON, Integer
from sqlalchemy.orm import relationship
from datetime import datetime

from app.db.database import Base

class ReplaySession(Base):
    """回放会话"""
    __tablename__ = "replay_sessions"

    id = Column(String, primary_key=True, index=True)
    session_id = Column(String, ForeignKey("game_sessions.id"), nullable=False)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)

    # 回放元数据
    title = Column(String)
    description = Column(String)
    started_at = Column(DateTime, nullable=False)
    ended_at = Column(DateTime)
    duration_seconds = Column(Integer)

    # 回放配置
    events_count = Column(Integer, default=0)
    participants = Column(JSON)  # 参与者列表

    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

class ReplayEvent(Base):
    """回放事件"""
    __tablename__ = "replay_events"

    id = Column(String, primary_key=True, index=True)
    replay_session_id = Column(String, ForeignKey("replay_sessions.id"), nullable=False)

    # 事件数据
    sequence_number = Column(Integer, nullable=False)  # 播放顺序
    timestamp = Column(DateTime, nullable=False)  # 原始时间戳
    event_type = Column(String, nullable=False)  # 消息/检定/状态变化等
    event_data = Column(JSON, nullable=False)  # 完整事件数据

    # 播放相关
    delay_ms = Column(Integer, default=0)  # 相对上一事件的延迟
    is_skippable = Column(Boolean, default=True)  # 是否可跳过
```

### 回放服务

```python
# app/services/replay.py
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
from sqlalchemy.orm import Session

from app.db.models.replay import ReplaySession, ReplayEvent
from app.db.models.gamestate import GameEvent
from app.core.logger import EventLogger

class ReplayService:
    """回放服务"""

    def __init__(self, db: Session):
        self.db = db
        self.logger = EventLogger()

    async def create_replay_session(
        self,
        session_id: str,
        campaign_id: str,
        start_time: datetime,
        end_time: Optional[datetime] = None,
    ) -> ReplaySession:
        """创建回放会话"""
        # 获取时间范围内的事件
        events = await self.logger.get_events(
            campaign_id=campaign_id,
            start_time=start_time,
            end_time=end_time or datetime.utcnow(),
        )

        if not events:
            raise ValueError("指定时间范围内没有事件")

        # 创建回放会话
        replay_session = ReplaySession(
            id=self._generate_replay_id(),
            session_id=session_id,
            campaign_id=campaign_id,
            started_at=start_time,
            ended_at=end_time,
            duration_seconds=int((end_time - start_time).total_seconds()) if end_time else None,
            events_count=len(events),
            participants=self._extract_participants(events),
        )

        self.db.add(replay_session)
        self.db.flush()

        # 创建回放事件
        await self._create_replay_events(replay_session.id, events)

        self.db.commit()
        return replay_session

    async def _create_replay_events(
        self,
        replay_session_id: str,
        events: List[GameEvent],
    ):
        """创建回放事件"""
        if not events:
            return

        base_timestamp = events[0].timestamp
        replay_events = []

        for idx, event in enumerate(events):
            # 计算延迟
            delay_ms = int((event.timestamp - base_timestamp).total_seconds() * 1000)
            if idx > 0:
                delay_ms = int((event.timestamp - events[idx - 1].timestamp).total_seconds() * 1000)

            replay_event = ReplayEvent(
                id=self._generate_event_id(),
                replay_session_id=replay_session_id,
                sequence_number=idx + 1,
                timestamp=event.timestamp,
                event_type=event.type,
                event_data={
                    "description": event.description,
                    "data": event.data,
                    "user_id": event.user_id,
                    "character_id": event.character_id,
                },
                delay_ms=delay_ms,
                is_skippable=self._is_skippable(event),
            )
            replay_events.append(replay_event)

        self.db.bulk_save_objects(replay_events)

    async def get_replay_data(
        self,
        replay_session_id: str,
        event_types: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """获取回放数据"""
        replay_session = self.db.query(ReplaySession).filter(
            ReplaySession.id == replay_session_id
        ).first()

        if not replay_session:
            raise ValueError("回放会话不存在")

        # 获取事件
        query = self.db.query(ReplayEvent).filter(
            ReplayEvent.replay_session_id == replay_session_id
        )

        if event_types:
            query = query.filter(ReplayEvent.event_type.in_(event_types))

        replay_events = query.order_by(ReplayEvent.sequence_number).all()

        return {
            "session": {
                "id": replay_session.id,
                "title": replay_session.title,
                "started_at": replay_session.started_at.isoformat(),
                "ended_at": replay_session.ended_at.isoformat() if replay_session.ended_at else None,
                "duration_seconds": replay_session.duration_seconds,
                "participants": replay_session.participants,
            },
            "events": [
                {
                    "sequence": event.sequence_number,
                    "timestamp": event.timestamp.isoformat(),
                    "type": event.event_type,
                    "data": event.event_data,
                    "delay_ms": event.delay_ms,
                    "is_skippable": event.is_skippable,
                }
                for event in replay_events
            ],
        }

    def _extract_participants(self, events: List[GameEvent]) -> List[Dict[str, Any]]:
        """提取参与者"""
        participants = {}
        for event in events:
            if event.user_id not in participants:
                participants[event.user_id] = {
                    "user_id": event.user_id,
                    "character_id": event.character_id,
                    "messages": 0,
                }
            participants[event.user_id]["messages"] += 1

        return list(participants.values())

    def _is_skippable(self, event: GameEvent) -> bool:
        """判断事件是否可跳过"""
        # 系统事件可跳过
        if event.type in ["system", "info"]:
            return True
        # 重要事件不可跳过
        if event.type in ["combat", "san_check", "clue_discovered"]:
            return False
        return True

    def _generate_replay_id(self) -> str:
        """生成回放 ID"""
        import uuid
        return f"replay_{uuid.uuid4().hex[:12]}"

    def _generate_event_id(self) -> str:
        """生成事件 ID"""
        import uuid
        return f"rev_evt_{uuid.uuid4().hex[:12]}"
```

### 回放 API

```python
# app/api/replay.py
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import Optional, List

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.replay import ReplayService

router = APIRouter(prefix="/replay", tags=["replay"])

class CreateReplayRequest(BaseModel):
    session_id: str
    campaign_id: str
    start_time: str  # ISO format
    end_time: Optional[str] = None

class ReplayResponse(BaseModel):
    session: dict
    events: List[dict]

@router.post("/create")
async def create_replay(
    request: CreateReplayRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建回放"""
    service = ReplayService(db)

    replay_session = await service.create_replay_session(
        session_id=request.session_id,
        campaign_id=request.campaign_id,
        start_time=datetime.fromisoformat(request.start_time),
        end_time=datetime.fromisoformat(request.end_time) if request.end_time else None,
    )

    return {"replay_id": replay_session.id}

@router.get("/{replay_id}", response_model=ReplayResponse)
async def get_replay(
    replay_id: str,
    event_types: Optional[List[str]] = Query(None),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取回放数据"""
    service = ReplayService(db)

    replay_data = await service.get_replay_data(
        replay_session_id=replay_id,
        event_types=event_types,
    )

    return replay_data
```

---

## 前端代码示例

### 回放播放器组件

```typescript
// frontend/src/components/replay/ReplayPlayer.tsx
import React, { useState, useEffect, useRef } from 'react';
import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
import { Play, Pause, SkipBack, SkipForward, Settings } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';

interface ReplayEvent {
  sequence: number;
  timestamp: string;
  type: string;
  data: any;
  delay_ms: number;
  is_skippable: boolean;
}

interface ReplayData {
  session: {
    id: string;
    title: string;
    started_at: string;
    ended_at: string | null;
    duration_seconds: number | null;
    participants: Array<{
      user_id: string;
      character_id: string | null;
      messages: number;
    }>;
  };
  events: ReplayEvent[];
}

type PlaybackSpeed = 0.5 | 1 | 2;

export function ReplayPlayer({ replayId }: { replayId: string }) {
  const [replayData, setReplayData] = useState<ReplayData | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentEventIndex, setCurrentEventIndex] = useState(0);
  const [playbackSpeed, setPlaybackSpeed] = useState<PlaybackSpeed>(1);
  const [selectedEventTypes, setSelectedEventTypes] = useState<Set<string>>(
    new Set(['message', 'roll', 'combat', 'san_check'])
  );

  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  // 加载回放数据
  useEffect(() => {
    const loadReplay = async () => {
      const response = await fetch(`/api/replay/${replayId}`);
      const data = await response.json();
      setReplayData(data);
    };
    loadReplay();
  }, [replayId]);

  // 播放逻辑
  useEffect(() => {
    if (!isPlaying || !replayData) return;

    const currentEvent = replayData.events[currentEventIndex];
    if (!currentEvent) {
      setIsPlaying(false);
      return;
    }

    // 计算延迟时间
    const delay = currentEvent.delay_ms / playbackSpeed;

    timeoutRef.current = setTimeout(() => {
      if (currentEventIndex < replayData.events.length - 1) {
        setCurrentEventIndex(prev => prev + 1);
      } else {
        setIsPlaying(false);
      }
    }, delay);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isPlaying, currentEventIndex, replayData, playbackSpeed]);

  const togglePlay = () => {
    setIsPlaying(!isPlaying);
  };

  const skipBack = () => {
    setCurrentEventIndex(Math.max(0, currentEventIndex - 5));
  };

  const skipForward = () => {
    if (replayData) {
      setCurrentEventIndex(Math.min(replayData.events.length - 1, currentEventIndex + 5));
    }
  };

  const seekTo = (index: number) => {
    setCurrentEventIndex(index);
  };

  const toggleEventType = (eventType: string) => {
    const newSet = new Set(selectedEventTypes);
    if (newSet.has(eventType)) {
      newSet.delete(eventType);
    } else {
      newSet.add(eventType);
    }
    setSelectedEventTypes(newSet);
  };

  if (!replayData) {
    return <div>加载中...</div>;
  }

  const filteredEvents = replayData.events.filter(e =>
    selectedEventTypes.has(e.type)
  );

  const currentEvent = filteredEvents[currentEventIndex];
  const progress = (currentEventIndex / (filteredEvents.length - 1)) * 100;

  return (
    <div className="space-y-4">
      {/* 回放信息 */}
      <Card>
        <CardHeader>
          <CardTitle>{replayData.session.title || '会话回放'}</CardTitle>
          <div className="text-sm text-muted-foreground">
            {replayData.session.started_at} - {replayData.session.ended_at || '进行中'}
            {replayData.session.duration_seconds && (
              <span> · {Math.floor(replayData.session.duration_seconds / 60)} 分钟</span>
            )}
          </div>
        </CardHeader>
      </Card>

      {/* 播放控制 */}
      <Card>
        <CardContent className="pt-6">
          <div className="space-y-4">
            {/* 进度条 */}
            <div className="space-y-2">
              <Slider
                value={[currentEventIndex]}
                max={filteredEvents.length - 1}
                step={1}
                onValueChange={(value) => seekTo(value[0])}
                className="cursor-pointer"
              />
              <div className="flex justify-between text-sm text-muted-foreground">
                <span>{currentEventIndex + 1} / {filteredEvents.length}</span>
                <span>{Math.floor(progress)}%</span>
              </div>
            </div>

            {/* 控制按钮 */}
            <div className="flex items-center justify-center gap-2">
              <Button variant="outline" size="icon" onClick={skipBack}>
                <SkipBack className="h-4 w-4" />
              </Button>
              <Button size="icon" onClick={togglePlay}>
                {isPlaying ? (
                  <Pause className="h-4 w-4" />
                ) : (
                  <Play className="h-4 w-4" />
                )}
              </Button>
              <Button variant="outline" size="icon" onClick={skipForward}>
                <SkipForward className="h-4 w-4" />
              </Button>

              <div className="ml-4 flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPlaybackSpeed(0.5)}
                  className={playbackSpeed === 0.5 ? 'bg-accent' : ''}
                >
                  0.5x
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPlaybackSpeed(1)}
                  className={playbackSpeed === 1 ? 'bg-accent' : ''}
                >
                  1x
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPlaybackSpeed(2)}
                  className={playbackSpeed === 2 ? 'bg-accent' : ''}
                >
                  2x
                </Button>
              </div>
            </div>

            {/* 事件类型过滤 */}
            <div className="flex items-center gap-4 flex-wrap">
              <span className="text-sm font-medium">显示事件:</span>
              {['message', 'roll', 'combat', 'san_check', 'clue_discovered'].map(type => (
                <div key={type} className="flex items-center gap-2">
                  <Checkbox
                    id={type}
                    checked={selectedEventTypes.has(type)}
                    onCheckedChange={() => toggleEventType(type)}
                  />
                  <label htmlFor={type} className="text-sm">{type}</label>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 当前事件显示 */}
      {currentEvent && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">
              事件 #{currentEvent.sequence} · {currentEvent.type}
            </CardTitle>
            <div className="text-sm text-muted-foreground">
              {currentEvent.timestamp}
            </div>
          </CardHeader>
          <CardContent>
            <ReplayEventRenderer event={currentEvent} />
          </CardContent>
        </Card>
      )}

      {/* 参与者 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">参与者</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {replayData.session.participants.map(participant => (
              <div key={participant.user_id} className="flex justify-between text-sm">
                <span>{participant.character_id || participant.user_id}</span>
                <span className="text-muted-foreground">
                  {participant.messages} 条消息
                </span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// 事件渲染器
function ReplayEventRenderer({ event }: { event: ReplayEvent }) {
  switch (event.type) {
    case 'message':
      return (
        <div className="p-4 bg-muted rounded-lg">
          <p>{event.data.description}</p>
        </div>
      );

    case 'roll':
      return (
        <div className="p-4 bg-blue-50 dark:bg-blue-950 rounded-lg">
          <div className="font-semibold">骰子检定</div>
          <div className="mt-2 text-sm">
            <div>结果: {event.data.data?.result}</div>
            <div>判定: {event.data.data?.success ? '成功' : '失败'}</div>
          </div>
        </div>
      );

    case 'combat':
      return (
        <div className="p-4 bg-red-50 dark:bg-red-950 rounded-lg">
          <div className="font-semibold">战斗事件</div>
          <div className="mt-2 text-sm">
            <div>伤害: {event.data.data?.damage}</div>
            <div>目标: {event.data.data?.target}</div>
          </div>
        </div>
      );

    case 'san_check':
      return (
        <div className="p-4 bg-purple-50 dark:bg-purple-950 rounded-lg">
          <div className="font-semibold">SAN 检定</div>
          <div className="mt-2 text-sm">
            <div>检定结果: {event.data.data?.success ? '成功' : '失败'}</div>
            <div>SAN 变化: {event.data.data?.san_change}</div>
          </div>
        </div>
      );

    case 'clue_discovered':
      return (
        <div className="p-4 bg-yellow-50 dark:bg-yellow-950 rounded-lg">
          <div className="font-semibold">发现线索</div>
          <div className="mt-2 text-sm">
            {event.data.description}
          </div>
        </div>
      );

    default:
      return (
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-sm">{event.data.description}</p>
          <pre className="mt-2 text-xs overflow-auto">
            {JSON.stringify(event.data.data, null, 2)}
          </pre>
        </div>
      );
  }
}
```

### 回放列表组件

```typescript
// frontend/src/components/replay/ReplayList.tsx
import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Play, Clock } from 'lucide-react';

interface ReplaySession {
  id: string;
  title: string;
  started_at: string;
  ended_at: string | null;
  duration_seconds: number | null;
}

export function ReplayList({ campaignId }: { campaignId: string }) {
  const [replays, setReplays] = useState<ReplaySession[]>([]);

  useEffect(() => {
    const loadReplays = async () => {
      // 实现获取回放列表
      const response = await fetch(`/api/replay?campaign_id=${campaignId}`);
      const data = await response.json();
      setReplays(data);
    };
    loadReplays();
  }, [campaignId]);

  return (
    <div className="space-y-3">
      <h3 className="text-lg font-semibold">历史回放</h3>
      {replays.map(replay => (
        <Card key={replay.id}>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">
                {replay.title || '未命名回放'}
              </CardTitle>
              <Button size="sm" onClick={() => window.location.href = `/replay/${replay.id}`}>
                <Play className="h-4 w-4 mr-2" />
                播放
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <div className="flex items-center gap-1">
                <Clock className="h-3 w-3" />
                {replay.started_at}
              </div>
              {replay.duration_seconds && (
                <div>
                  {Math.floor(replay.duration_seconds / 60)} 分钟
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/replay.py` | 创建 | 回放数据模型 |
| `app/services/replay.py` | 创建 | 回放服务 |
| `app/api/replay.py` | 创建 | 回放 API |
| `frontend/src/components/replay/ReplayPlayer.tsx` | 创建 | 回放播放器组件 |
| `frontend/src/components/replay/ReplayList.tsx` | 创建 | 回放列表组件 |
| `tests/test_replay.py` | 创建 | 回放测试 |
| `frontend/src/pages/ReplayPage.tsx` | 创建 | 回放页面 |

---

## 验收标准

- [ ] 能正确生成回放数据
- [ ] 播放控制功能正常（播放/暂停/进度）
- [ ] 速度调节功能正常（0.5x/1x/2x）
- [ ] 事件类型过滤有效
- [ ] 不同事件类型正确渲染
- [ ] 参与者信息准确显示
- [ ] 回放数据与原始事件一致

---

## 参考文档

- M1-080: 事件日志系统
- M3-006: 事件写入服务
- React Hook Form 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
