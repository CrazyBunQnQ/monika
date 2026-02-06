# M2-007: 实现白板功能

**任务ID**: M2-007
**标题**: 实现白板功能
**类型**: fullstack (全栈开发)
**预估工时**: 3h
**依赖**: M2-002

---

## 任务描述`

实现在线白板功能，允许 KP 和玩家在共享画板上绘制、标注、添加便签等，用于战术规划和场景说明。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-007-01 | 设计白板数据结构 | Data Model | 25min |
| M2-007-02 | 实现绘图引擎 | Drawing Engine | 40min |
| M2-007-03 | 实现白板同步 | Whiteboard Sync | 35min |
| M2-007-04 | 实现工具栏 | Toolbar | 25min |
| M2-007-05 | 实现形状绘制 | Shapes | 30min |
| M2-007-06 | 实现便签功能 | Sticky Notes | 25min |
| M2-007-07 | 编写白板测试 | 测试覆盖 | 15min |

---

## 白板数据结构

```python
# app/db/models/whiteboard.py
from sqlalchemy import Column, String, ForeignKey, JSON, DateTime
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class Whiteboard(Base):
    """白板"""
    __tablename__ = 'whiteboards'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 基本信息
    name = Column(String, nullable=False)
    background_color = Column(String, default='#ffffff')
    background_image = Column(String)

    # 白板内容（JSON 存储）
    elements = Column(JSON, default=list)

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False)
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    # 关系
    room = relationship("Room", back_populates="whiteboards")
    creator = relationship("User", back_populates="created_whiteboards")

    def __repr__(self):
        return f"<Whiteboard {self.name}>"
```

---

## 白板元素类型

```typescript
// frontend/src/lib/whiteboard/types.ts
export type WhiteboardElement =
  | { type: 'path'; points: Point[]; color: string; width: number; id: string }
  | { type: 'rect'; x: number; y: number; width: number; height: number; color: string; fill?: string; id: string }
  | { type: 'circle'; x: number; y: number; radius: number; color: string; fill?: string; id: string }
  | { type: 'arrow'; start: Point; end: Point; color: string; width: number; id: string }
  | { type: 'text'; x: number; y: number; content: string; fontSize: number; color: string; id: string }
  | { type: 'stickyNote'; x: number; y: number; width: number; height: number; content: string; color: string; id: string }
  | { type: 'image'; x: number; y: number; width: number; height: number; src: string; id: string }

export interface Point {
  x: number
  y: number
}

export interface WhiteboardState {
  elements: WhiteboardElement[]
  selectedIds: string[]
  tool: 'select' | 'pen' | 'rect' | 'circle' | 'arrow' | 'text' | 'stickyNote' | 'eraser'
  color: string
  strokeWidth: number
}
```

---

## 白板服务

```python
# app/services/whiteboard.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime

from app.db.models.whiteboard import Whiteboard
from app.core.security import generate_id

class WhiteboardService:
    """白板服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_whiteboard(
        self,
        room_id: str,
        name: str,
        created_by: str,
        background_color: str = '#ffffff',
    ) -> Whiteboard:
        """创建白板"""
        whiteboard = Whiteboard(
            id=generate_id('whiteboard'),
            room_id=room_id,
            name=name,
            background_color=background_color,
            elements=[],
            created_by=created_by,
        )

        self.db.add(whiteboard)
        self.db.commit()
        self.db.refresh(whiteboard)

        return whiteboard

    def get_whiteboard(self, whiteboard_id: str) -> Optional[Whiteboard]:
        """获取白板"""
        return self.db.query(Whiteboard)\
            .filter(Whiteboard.id == whiteboard_id)\
            .first()

    def get_room_whiteboards(self, room_id: str) -> List[Whiteboard]:
        """获取房间白板列表"""
        return self.db.query(Whiteboard)\
            .filter(Whiteboard.room_id == room_id)\
            .order_by(Whiteboard.created_at.desc())\
            .all()

    def add_element(
        self,
        whiteboard_id: str,
        element: Dict[str, Any],
    ) -> Whiteboard:
        """添加元素"""
        whiteboard = self.get_whiteboard(whiteboard_id)
        if not whiteboard:
            return None

        whiteboard.elements.append(element)
        whiteboard.updated_at = datetime.now()

        self.db.commit()
        self.db.refresh(whiteboard)

        return whiteboard

    def update_elements(
        self,
        whiteboard_id: str,
        elements: List[Dict[str, Any]],
    ) -> Whiteboard:
        """更新所有元素"""
        whiteboard = self.get_whiteboard(whiteboard_id)
        if not whiteboard:
            return None

        whiteboard.elements = elements
        whiteboard.updated_at = datetime.now()

        self.db.commit()
        self.db.refresh(whiteboard)

        return whiteboard

    def delete_element(
        self,
        whiteboard_id: str,
        element_id: str,
    ) -> Whiteboard:
        """删除元素"""
        whiteboard = self.get_whiteboard(whiteboard_id)
        if not whiteboard:
            return None

        whiteboard.elements = [
            e for e in whiteboard.elements
            if e.get('id') != element_id
        ]
        whiteboard.updated_at = datetime.now()

        self.db.commit()
        self.db.refresh(whiteboard)

        return whiteboard

    def clear_whiteboard(self, whiteboard_id: str) -> Whiteboard:
        """清空白板"""
        whiteboard = self.get_whiteboard(whiteboard_id)
        if not whiteboard:
            return None

        whiteboard.elements = []
        whiteboard.updated_at = datetime.now()

        self.db.commit()
        self.db.refresh(whiteboard)

        return whiteboard
```

---

## 前端白板组件

```tsx
// frontend/src/components/game/Whiteboard.tsx
import { useState, useRef, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  MousePointer,
  Pen,
  Square,
  Circle,
  Minus,
  Type,
  StickyNote,
  Eraser,
  Trash2,
  Undo,
  Redo,
} from 'lucide-react'
import { WhiteboardState, WhiteboardElement, Point } from '@/lib/whiteboard/types'

interface WhiteboardProps {
  roomId: string
  whiteboardId: string
}

export function Whiteboard({ roomId, whiteboardId }: WhiteboardProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [state, setState] = useState<WhiteboardState>({
    elements: [],
    selectedIds: [],
    tool: 'pen',
    color: '#000000',
    strokeWidth: 2,
  })
  const [isDrawing, setIsDrawing] = useState(false)
  const [currentPath, setCurrentPath] = useState<Point[]>([])

  // 绘制画布
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // 清空画布
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // 绘制所有元素
    state.elements.forEach(element => {
      drawElement(ctx, element)
    })

    // 绘制当前路径
    if (isDrawing && currentPath.length > 0) {
      ctx.beginPath()
      ctx.strokeStyle = state.color
      ctx.lineWidth = state.strokeWidth
      ctx.lineCap = 'round'
      ctx.lineJoin = 'round'

      currentPath.forEach((point, index) => {
        if (index === 0) {
          ctx.moveTo(point.x, point.y)
        } else {
          ctx.lineTo(point.x, point.y)
        }
      })

      ctx.stroke()
    }
  }, [state.elements, isDrawing, currentPath, state.color, state.strokeWidth])

  const drawElement = (ctx: CanvasRenderingContext2D, element: WhiteboardElement) => {
    switch (element.type) {
      case 'path':
        ctx.beginPath()
        ctx.strokeStyle = element.color
        ctx.lineWidth = element.width
        ctx.lineCap = 'round'
        ctx.lineJoin = 'round'

        element.points.forEach((point, index) => {
          if (index === 0) {
            ctx.moveTo(point.x, point.y)
          } else {
            ctx.lineTo(point.x, point.y)
          }
        })

        ctx.stroke()
        break

      case 'rect':
        ctx.strokeStyle = element.color
        ctx.lineWidth = 2
        if (element.fill) {
          ctx.fillStyle = element.fill
          ctx.fillRect(element.x, element.y, element.width, element.height)
        }
        ctx.strokeRect(element.x, element.y, element.width, element.height)
        break

      case 'circle':
        ctx.strokeStyle = element.color
        ctx.lineWidth = 2
        ctx.beginPath()
        ctx.arc(element.x, element.y, element.radius, 0, Math.PI * 2)
        if (element.fill) {
          ctx.fillStyle = element.fill
          ctx.fill()
        }
        ctx.stroke()
        break

      case 'text':
        ctx.fillStyle = element.color
        ctx.font = `${element.fontSize}px Arial`
        ctx.fillText(element.content, element.x, element.y)
        break

      case 'stickyNote':
        ctx.fillStyle = element.color
        ctx.fillRect(element.x, element.y, element.width, element.height)
        ctx.fillStyle = '#000'
        ctx.font = '14px Arial'
        // 简化的文本绘制
        ctx.fillText(element.content.substring(0, 20), element.x + 5, element.y + 20)
        break
    }
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    if (state.tool === 'select') return

    const rect = canvasRef.current!.getBoundingClientRect()
    const point = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    }

    setIsDrawing(true)

    if (state.tool === 'pen') {
      setCurrentPath([point])
    }
  }

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDrawing || state.tool !== 'pen') return

    const rect = canvasRef.current!.getBoundingClientRect()
    const point = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    }

    setCurrentPath(prev => [...prev, point])
  }

  const handleMouseUp = () => {
    if (!isDrawing) return

    if (state.tool === 'pen' && currentPath.length > 1) {
      const newElement: WhiteboardElement = {
        type: 'path',
        id: `element_${Date.now()}`,
        points: currentPath,
        color: state.color,
        width: state.strokeWidth,
      }

      setState(prev => ({
        ...prev,
        elements: [...prev.elements, newElement],
      }))

      // 同步到服务器
      syncElement(newElement)
    }

    setIsDrawing(false)
    setCurrentPath([])
  }

  const syncElement = async (element: WhiteboardElement) => {
    try {
      await fetch(`/api/whiteboards/${whiteboardId}/elements`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ element }),
      })
    } catch (error) {
      console.error('Failed to sync element:', error)
    }
  }

  const setTool = (tool: WhiteboardState['tool']) => {
    setState(prev => ({ ...prev, tool }))
  }

  const clearCanvas = () => {
    setState(prev => ({ ...prev, elements: [] }))
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span>白板</span>
          <div className="flex space-x-2">
            <Button size="sm" variant="outline" onClick={clearCanvas}>
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </CardTitle>
      </CardHeader>

      <CardContent>
        {/* 工具栏 */}
        <div className="flex items-center space-x-1 mb-3 pb-3 border-b">
          <Button
            size="sm"
            variant={state.tool === 'select' ? 'default' : 'ghost'}
            onClick={() => setTool('select')}
          >
            <MousePointer className="h-4 w-4" />
          </Button>
          <Separator orientation="vertical" className="h-6" />
          <Button
            size="sm"
            variant={state.tool === 'pen' ? 'default' : 'ghost'}
            onClick={() => setTool('pen')}
          >
            <Pen className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant={state.tool === 'rect' ? 'default' : 'ghost'}
            onClick={() => setTool('rect')}
          >
            <Square className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant={state.tool === 'circle' ? 'default' : 'ghost'}
            onClick={() => setTool('circle')}
          >
            <Circle className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant={state.tool === 'arrow' ? 'default' : 'ghost'}
            onClick={() => setTool('arrow')}
          >
            <Minus className="h-4 w-4" />
          </Button>
          <Separator orientation="vertical" className="h-6" />
          <Button
            size="sm"
            variant={state.tool === 'text' ? 'default' : 'ghost'}
            onClick={() => setTool('text')}
          >
            <Type className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant={state.tool === 'stickyNote' ? 'default' : 'ghost'}
            onClick={() => setTool('stickyNote')}
          >
            <StickyNote className="h-4 w-4" />
          </Button>
          <Separator orientation="vertical" className="h-6" />
          <Button
            size="sm"
            variant={state.tool === 'eraser' ? 'default' : 'ghost'}
            onClick={() => setTool('eraser')}
          >
            <Eraser className="h-4 w-4" />
          </Button>

          {/* 颜色选择器 */}
          <input
            type="color"
            value={state.color}
            onChange={(e) => setState(prev => ({ ...prev, color: e.target.value }))}
            className="w-8 h-8 rounded cursor-pointer"
          />

          {/* 线宽 */}
          <select
            value={state.strokeWidth}
            onChange={(e) => setState(prev => ({ ...prev, strokeWidth: parseInt(e.target.value) }))}
            className="h-8 rounded px-2 border"
          >
            <option value={1}>1px</option>
            <option value={2}>2px</option>
            <option value={4}>4px</option>
            <option value={8}>8px</option>
          </select>
        </div>

        {/* 画布 */}
        <canvas
          ref={canvasRef}
          width={800}
          height={600}
          className="border rounded bg-white cursor-crosshair"
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
        />
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/whiteboard.py` | 创建 | 白板数据模型 |
| `app/services/whiteboard.py` | 创建 | 白板服务 |
| `app/api/whiteboards.py` | 创建 | 白板 API |
| `frontend/src/lib/whiteboard/types.ts` | 创建 | 白板类型定义 |
| `frontend/src/components/game/Whiteboard.tsx` | 创建 | 白板组件 |

---

## 验收标准

- [ ] 绘图功能正常
- [ ] 实时同步有效
- [ ] 工具切换流畅
- [ ] 撤销重做可用
- [ ] 形状绘制准确
- [ ] 便签功能正常

---

## 参考文档

- M2-002: WebSocket 事件系统
- Fabric.js 或 Konva.js 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
