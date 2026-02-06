# M6-009: 实现反馈收集

**任务ID**: M6-009
**标题**: 实现反馈收集
**类型**: fullstack (全栈开发)
**预估工时**: 6h
**依赖**: M1-050

---

## 任务描述

实现用户反馈收集系统，包括 bug 报告、功能建议、体验反馈等类型，支持截图上传、优先级标记、反馈状态跟踪。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-009-01 | 设计反馈类型分类 | 反馈类别定义 | 30min |
| M6-009-02 | 实现反馈数据模型 | 数据库结构 | 45min |
| M6-009-03 | 实现反馈收集服务 | 业务逻辑 | 1h |
| M6-009-04 | 实现反馈 API | 提交/查询接口 | 45min |
| M6-009-05 | 实现反馈表单组件 | 前端反馈表单 | 1h |
| M6-009-06 | 实现截图上传功能 | 文件上传处理 | 45min |
| M6-009-07 | 实现反馈历史查看 | 用户反馈列表 | 30min |
| M6-009-08 | 实现反馈管理面板 | 后台管理界面 | 45min |

---

## 后端实现

### 反馈数据模型

```python
# backend/models/feedback.py
from datetime import datetime
from enum import Enum
from typing import Optional, List
from pydantic import BaseModel, Field
from sqlalchemy import Column, Integer, String, Text, DateTime, ForeignKey, Enum as SQLEnum, JSON
from sqlalchemy.orm import relationship

from database import Base


class FeedbackType(str, Enum):
    """反馈类型"""
    BUG = "bug"                    # Bug 报告
    FEATURE = "feature"            # 功能建议
    IMPROVEMENT = "improvement"    # 体验改进
    CONTENT = "content"            # 内容反馈
    OTHER = "other"                # 其他


class FeedbackPriority(str, Enum):
    """反馈优先级"""
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


class FeedbackStatus(str, Enum):
    """反馈状态"""
    SUBMITTED = "submitted"        # 已提交
    REVIEWING = "reviewing"        # 审核中
    CONFIRMED = "confirmed"        # 已确认
    IN_PROGRESS = "in_progress"    # 处理中
    RESOLVED = "resolved"          # 已解决
    CLOSED = "closed"              # 已关闭
    REJECTED = "rejected"          # 已拒绝


class Feedback(Base):
    """反馈表"""
    __tablename__ = "feedbacks"

    id = Column(Integer, primary_key=True, index=True)

    # 用户关联
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)

    # 反馈信息
    type = Column(SQLEnum(FeedbackType), nullable=False)
    category = Column(String)  # 二级分类
    title = Column(String(200), nullable=False)
    description = Column(Text, nullable=False)

    # 优先级和状态
    priority = Column(SQLEnum(FeedbackPriority), default=FeedbackPriority.MEDIUM)
    status = Column(SQLEnum(FeedbackStatus), default=FeedbackStatus.SUBMITTED)

    # 上下文信息
    page_url = Column(String)  # 当前页面 URL
    component = Column(String)  # 相关组件
    reproduction_steps = Column(Text)  # 复现步骤

    # 附件
    screenshots = Column(JSON, default=list)  # 截图 URL 列表
    attachments = Column(JSON, default=list)  # 其他附件

    # 系统信息
    browser_info = Column(JSON)  # 浏览器信息
    device_info = Column(JSON)   # 设备信息
    session_data = Column(JSON)  # 会话数据

    # 处理信息
    assigned_to = Column(Integer, ForeignKey("users.id"), nullable=True)
    response = Column(Text)  # 管理员回复
    resolution = Column(Text)  # 解决方案

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow, index=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    resolved_at = Column(DateTime, nullable=True)

    # 关系
    user = relationship("User", foreign_keys=[user_id], back_populates="feedbacks")
    assignee = relationship("User", foreign_keys=[assigned_to])


class FeedbackVote(Base):
    """反馈投票表（用于功能建议的点赞）"""
    __tablename__ = "feedback_votes"

    id = Column(Integer, primary_key=True, index=True)
    feedback_id = Column(Integer, ForeignKey("feedbacks.id"), nullable=False)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    vote_type = Column(String, default="up")  # up, down

    created_at = Column(DateTime, default=datetime.utcnow)

    # 唯一约束
    __table_args__ = (
        {'sqlite_autoincrement': True}
    )


class FeedbackComment(Base):
    """反馈评论表"""
    __tablename__ = "feedback_comments"

    id = Column(Integer, primary_key=True, index=True)
    feedback_id = Column(Integer, ForeignKey("feedbacks.id"), nullable=False)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    content = Column(Text, nullable=False)

    # 管理员标记
    is_official = Column(Boolean, default=False)  # 是否为官方回复

    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)


# Pydantic 模型
class FeedbackCreate(BaseModel):
    """创建反馈"""
    type: FeedbackType
    category: Optional[str] = None
    title: str = Field(..., max_length=200)
    description: str
    priority: Optional[FeedbackPriority] = FeedbackPriority.MEDIUM
    page_url: Optional[str] = None
    component: Optional[str] = None
    reproduction_steps: Optional[str] = None
    screenshots: Optional[List[str]] = []
    browser_info: Optional[dict] = None
    device_info: Optional[dict] = None
    session_data: Optional[dict] = None


class FeedbackUpdate(BaseModel):
    """更新反馈"""
    status: Optional[FeedbackStatus] = None
    priority: Optional[FeedbackPriority] = None
    assigned_to: Optional[int] = None
    response: Optional[str] = None
    resolution: Optional[str] = None


class FeedbackResponse(BaseModel):
    """反馈响应"""
    id: int
    type: FeedbackType
    category: Optional[str]
    title: str
    description: str
    priority: FeedbackPriority
    status: FeedbackStatus
    page_url: Optional[str]
    component: Optional[str]
    screenshots: List[str]
    response: Optional[str]
    resolution: Optional[str]
    created_at: datetime
    updated_at: datetime
    resolved_at: Optional[datetime]
    vote_count: int = 0

    class Config:
        from_attributes = True


class FeedbackListResponse(BaseModel):
    """反馈列表响应"""
    items: List[FeedbackResponse]
    total: int
    page: int
    page_size: int
```

### 反馈服务

```python
# backend/services/feedback_service.py
from typing import List, Optional, Dict, Any
from datetime import datetime
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from models.feedback import (
    Feedback,
    FeedbackVote,
    FeedbackComment,
    FeedbackCreate,
    FeedbackUpdate,
    FeedbackResponse,
    FeedbackType,
    FeedbackStatus,
    FeedbackPriority,
)


class FeedbackService:
    """反馈系统服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_feedback(
        self,
        user_id: int,
        data: FeedbackCreate
    ) -> FeedbackResponse:
        """创建反馈"""
        feedback = Feedback(
            user_id=user_id,
            type=data.type,
            category=data.category,
            title=data.title,
            description=data.description,
            priority=data.priority,
            page_url=data.page_url,
            component=data.component,
            reproduction_steps=data.reproduction_steps,
            screenshots=data.screenshots or [],
            browser_info=data.browser_info,
            device_info=data.device_info,
            session_data=data.session_data,
            status=FeedbackStatus.SUBMITTED
        )

        self.db.add(feedback)
        self.db.commit()
        self.db.refresh(feedback)

        # 发送通知给管理员
        self._notify_admins(feedback)

        return self._to_response(feedback)

    def get_feedbacks(
        self,
        user_id: Optional[int] = None,
        feedback_type: Optional[FeedbackType] = None,
        status: Optional[FeedbackStatus] = None,
        priority: Optional[FeedbackPriority] = None,
        page: int = 1,
        page_size: int = 20
    ) -> Dict[str, Any]:
        """获取反馈列表"""
        query = self.db.query(Feedback)

        # 筛选条件
        if user_id:
            query = query.filter(Feedback.user_id == user_id)

        if feedback_type:
            query = query.filter(Feedback.type == feedback_type)

        if status:
            query = query.filter(Feedback.status == status)

        if priority:
            query = query.filter(Feedback.priority == priority)

        # 总数
        total = query.count()

        # 分页
        offset = (page - 1) * page_size
        items = query.order_by(Feedback.created_at.desc()).offset(offset).limit(page_size).all()

        return {
            "items": [self._to_response(item) for item in items],
            "total": total,
            "page": page,
            "page_size": page_size
        }

    def get_feedback(self, feedback_id: int) -> Optional[FeedbackResponse]:
        """获取单个反馈"""
        feedback = self.db.query(Feedback).filter(Feedback.id == feedback_id).first()
        if not feedback:
            return None
        return self._to_response(feedback)

    def update_feedback(
        self,
        feedback_id: int,
        update: FeedbackUpdate
    ) -> Optional[FeedbackResponse]:
        """更新反馈（管理员）"""
        feedback = self.db.query(Feedback).filter(Feedback.id == feedback_id).first()
        if not feedback:
            return None

        if update.status:
            feedback.status = update.status
            if update.status == FeedbackStatus.RESOLVED:
                feedback.resolved_at = datetime.utcnow()

        if update.priority:
            feedback.priority = update.priority

        if update.assigned_to:
            feedback.assigned_to = update.assigned_to

        if update.response:
            feedback.response = update.response

        if update.resolution:
            feedback.resolution = update.resolution

        self.db.commit()
        self.db.refresh(feedback)

        # 通知用户
        self._notify_user(feedback)

        return self._to_response(feedback)

    def add_vote(self, feedback_id: int, user_id: int) -> bool:
        """为功能建议投票"""
        existing = self.db.query(FeedbackVote).filter(
            and_(
                FeedbackVote.feedback_id == feedback_id,
                FeedbackVote.user_id == user_id
            )
        ).first()

        if existing:
            return False

        vote = FeedbackVote(
            feedback_id=feedback_id,
            user_id=user_id
        )
        self.db.add(vote)
        self.db.commit()
        return True

    def remove_vote(self, feedback_id: int, user_id: int) -> bool:
        """取消投票"""
        vote = self.db.query(FeedbackVote).filter(
            and_(
                FeedbackVote.feedback_id == feedback_id,
                FeedbackVote.user_id == user_id
            )
        ).first()

        if not vote:
            return False

        self.db.delete(vote)
        self.db.commit()
        return True

    def add_comment(
        self,
        feedback_id: int,
        user_id: int,
        content: str,
        is_official: bool = False
    ) -> FeedbackComment:
        """添加评论"""
        comment = FeedbackComment(
            feedback_id=feedback_id,
            user_id=user_id,
            content=content,
            is_official=is_official
        )
        self.db.add(comment)
        self.db.commit()
        self.db.refresh(comment)
        return comment

    def get_feedback_stats(self) -> Dict[str, Any]:
        """获取反馈统计"""
        total = self.db.query(Feedback).count()

        by_status = {}
        for status in FeedbackStatus:
            count = self.db.query(Feedback).filter(Feedback.status == status).count()
            by_status[status.value] = count

        by_type = {}
        for ftype in FeedbackType:
            count = self.db.query(Feedback).filter(Feedback.type == ftype).count()
            by_type[ftype.value] = count

        by_priority = {}
        for priority in FeedbackPriority:
            count = self.db.query(Feedback).filter(Feedback.priority == priority).count()
            by_priority[priority.value] = count

        # 平均解决时间
        resolved = self.db.query(Feedback).filter(
            Feedback.status == FeedbackStatus.RESOLVED,
            Feedback.resolved_at.isnot(None)
        ).all()

        if resolved:
            avg_resolution_hours = sum(
                (r.resolved_at - r.created_at).total_seconds() / 3600
                for r in resolved
            ) / len(resolved)
        else:
            avg_resolution_hours = 0

        return {
            "total": total,
            "by_status": by_status,
            "by_type": by_type,
            "by_priority": by_priority,
            "avg_resolution_hours": round(avg_resolution_hours, 2)
        }

    def _to_response(self, feedback: Feedback) -> FeedbackResponse:
        """转换为响应对象"""
        vote_count = self.db.query(FeedbackVote).filter(
            FeedbackVote.feedback_id == feedback.id
        ).count()

        return FeedbackResponse(
            id=feedback.id,
            type=feedback.type,
            category=feedback.category,
            title=feedback.title,
            description=feedback.description,
            priority=feedback.priority,
            status=feedback.status,
            page_url=feedback.page_url,
            component=feedback.component,
            screenshots=feedback.screenshots or [],
            response=feedback.response,
            resolution=feedback.resolution,
            created_at=feedback.created_at,
            updated_at=feedback.updated_at,
            resolved_at=feedback.resolved_at,
            vote_count=vote_count
        )

    def _notify_admins(self, feedback: Feedback):
        """通知管理员有新反馈"""
        # TODO: 实现管理员通知
        pass

    def _notify_user(self, feedback: Feedback):
        """通知用户反馈状态更新"""
        # TODO: 实现用户通知
        pass
```

### 反馈 API 路由

```python
# backend/api/routes/feedback.py
from fastapi import APIRouter, Depends, HTTPException, UploadFile, File
from sqlalchemy.orm import Session

from database import get_db
from services.feedback_service import FeedbackService
from models.feedback import (
    FeedbackType,
    FeedbackStatus,
    FeedbackPriority,
    FeedbackCreate,
    FeedbackUpdate,
    FeedbackResponse,
)
from middleware.auth import get_current_user
from utils.file_upload import upload_screenshot

router = APIRouter(prefix="/feedback", tags=["feedback"])


@router.post("", response_model=FeedbackResponse, status_code=201)
async def create_feedback(
    data: FeedbackCreate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """提交反馈"""
    service = FeedbackService(db)
    return service.create_feedback(current_user.id, data)


@router.get("", response_model=dict)
async def get_feedbacks(
    feedback_type: FeedbackType = None,
    status: FeedbackStatus = None,
    priority: FeedbackPriority = None,
    page: int = 1,
    page_size: int = 20,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取反馈列表"""
    service = FeedbackService(db)

    # 普通用户只能看自己的，管理员可以看所有
    user_id = current_user.id if current_user.role != "admin" else None

    return service.get_feedbacks(
        user_id=user_id,
        feedback_type=feedback_type,
        status=status,
        priority=priority,
        page=page,
        page_size=page_size
    )


@router.get("/{feedback_id}", response_model=FeedbackResponse)
async def get_feedback(
    feedback_id: int,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取反馈详情"""
    service = FeedbackService(db)
    feedback = service.get_feedback(feedback_id)

    if not feedback:
        raise HTTPException(status_code=404, detail="Feedback not found")

    return feedback


@router.put("/{feedback_id}", response_model=FeedbackResponse)
async def update_feedback(
    feedback_id: int,
    update: FeedbackUpdate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """更新反馈（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    service = FeedbackService(db)
    feedback = service.update_feedback(feedback_id, update)

    if not feedback:
        raise HTTPException(status_code=404, detail="Feedback not found")

    return feedback


@router.post("/{feedback_id}/vote")
async def vote_feedback(
    feedback_id: int,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """为反馈投票"""
    service = FeedbackService(db)
    success = service.add_vote(feedback_id, current_user.id)
    return {"success": success}


@router.delete("/{feedback_id}/vote")
async def unvote_feedback(
    feedback_id: int,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """取消投票"""
    service = FeedbackService(db)
    success = service.remove_vote(feedback_id, current_user.id)
    return {"success": success}


@router.post("/upload-screenshot")
async def upload_screenshot(
    file: UploadFile = File(...),
    current_user = Depends(get_current_user)
):
    """上传截图"""
    if not file.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="Only images allowed")

    url = await upload_screenshot(file, current_user.id)
    return {"url": url}


@router.get("/stats/overview")
async def get_feedback_stats(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取反馈统计（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    service = FeedbackService(db)
    return service.get_feedback_stats()
```

---

## 前端实现

### 反馈表单组件

```tsx
// frontend/src/components/feedback/FeedbackForm.tsx
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Bug,
  Lightbulb,
  TrendingUp,
  FileText,
  MessageSquare,
  Upload,
  X,
  Send
} from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'

interface FeedbackFormProps {
  onSuccess?: () => void
  onCancel?: () => void
}

const FEEDBACK_TYPES = [
  { value: 'bug', label: 'Bug 报告', icon: Bug, color: 'text-red-500' },
  { value: 'feature', label: '功能建议', icon: Lightbulb, color: 'text-yellow-500' },
  { value: 'improvement', label: '体验改进', icon: TrendingUp, color: 'text-green-500' },
  { value: 'content', label: '内容反馈', icon: FileText, color: 'text-blue-500' },
  { value: 'other', label: '其他', icon: MessageSquare, color: 'text-gray-500' },
]

const BUG_CATEGORIES = [
  '界面问题',
  '功能异常',
  '性能问题',
  '数据错误',
  '其他',
]

const FEATURE_CATEGORIES = [
  '新功能请求',
  '现有功能改进',
  '集成建议',
  '其他',
]

export function FeedbackForm({ onSuccess, onCancel }: FeedbackFormProps) {
  const { toast } = useToast()

  const [type, setType] = useState<'bug' | 'feature' | 'improvement' | 'content' | 'other'>('bug')
  const [category, setCategory] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [reproduction, setReproduction] = useState('')
  const [screenshots, setScreenshots] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectedType = FEEDBACK_TYPES.find(t => t.value === type)
  const Icon = selectedType?.icon || Bug

  // 上传截图
  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files) return

    const file = files[0]
    if (!file.type.startsWith('image/')) {
      toast({
        variant: "destructive",
        title: "上传失败",
        description: "只能上传图片文件"
      })
      return
    }

    const formData = new FormData()
    formData.append('file', file)

    try {
      const response = await fetch('/api/feedback/upload-screenshot', {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) throw new Error('Upload failed')

      const data = await response.json()
      setScreenshots([...screenshots, data.url])
    } catch (error) {
      toast({
        variant: "destructive",
        title: "上传失败",
        description: "请稍后重试"
      })
    }
  }

  // 移除截图
  const removeScreenshot = (index: number) => {
    setScreenshots(screenshots.filter((_, i) => i !== index))
  }

  // 提交反馈
  const handleSubmit = async () => {
    if (!title.trim() || !description.trim()) {
      toast({
        variant: "destructive",
        title: "填写不完整",
        description: "请填写标题和描述"
      })
      return
    }

    setIsSubmitting(true)

    try {
      const response = await fetch('/api/feedback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type,
          category,
          title,
          description,
          reproduction_steps: type === 'bug' ? reproduction : undefined,
          screenshots,
          page_url: window.location.href,
          browser_info: {
            userAgent: navigator.userAgent,
            language: navigator.language,
          },
          device_info: {
            screen: `${window.screen.width}x${window.screen.height}`,
            viewport: `${window.innerWidth}x${window.innerHeight}`,
          },
        }),
      })

      if (!response.ok) throw new Error('Submit failed')

      toast({
        title: "反馈已提交",
        description: "感谢你的反馈，我们会尽快处理"
      })

      onSuccess?.()
    } catch (error) {
      toast({
        variant: "destructive",
        title: "提交失败",
        description: "请稍后重试"
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="w-full max-w-2xl mx-auto">
      <CardHeader>
        <CardTitle className="flex items-center">
          <MessageSquare className="h-5 w-5 mr-2" />
          提交反馈
        </CardTitle>
        <CardDescription>
          帮助我们改进产品，你的反馈很重要
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-6">
        {/* 反馈类型选择 */}
        <div className="space-y-3">
          <Label>反馈类型</Label>
          <div className="grid grid-cols-5 gap-2">
            {FEEDBACK_TYPES.map(({ value, label, icon: Icon, color }) => (
              <button
                key={value}
                onClick={() => setType(value as any)}
                className={cn(
                  "flex flex-col items-center p-3 rounded-lg border-2 transition-all",
                  type === value
                    ? "border-primary bg-primary/5"
                    : "border-transparent hover:bg-muted"
                )}
              >
                <Icon className={cn("h-5 w-5 mb-1", color)} />
                <span className="text-xs">{label}</span>
              </button>
            ))}
          </div>
        </div>

        {/* 分类 */}
        {type === 'bug' && (
          <div className="space-y-2">
            <Label>问题分类</Label>
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue placeholder="选择分类" />
              </SelectTrigger>
              <SelectContent>
                {BUG_CATEGORIES.map(cat => (
                  <SelectItem key={cat} value={cat}>{cat}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {type === 'feature' && (
          <div className="space-y-2">
            <Label>建议分类</Label>
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue placeholder="选择分类" />
              </SelectTrigger>
              <SelectContent>
                {FEATURE_CATEGORIES.map(cat => (
                  <SelectItem key={cat} value={cat}>{cat}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {/* 标题 */}
        <div className="space-y-2">
          <Label>标题 *</Label>
          <Input
            placeholder="简要描述你的反馈"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={200}
          />
          <div className="text-xs text-muted-foreground text-right">
            {title.length} / 200
          </div>
        </div>

        {/* 描述 */}
        <div className="space-y-2">
          <Label>详细描述 *</Label>
          <Textarea
            placeholder={type === 'bug' ? "描述你遇到的问题" : "详细说明你的建议"}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={5}
          />
        </div>

        {/* 复现步骤（Bug） */}
        {type === 'bug' && (
          <div className="space-y-2">
            <Label>复现步骤（可选）</Label>
            <Textarea
              placeholder="1. 点击...\n2. 输入...\n3. 观察到..."
              value={reproduction}
              onChange={(e) => setReproduction(e.target.value)}
              rows={4}
            />
          </div>
        )}

        {/* 截图上传 */}
        <div className="space-y-3">
          <Label>截图（可选）</Label>
          <div className="flex flex-wrap gap-3">
            {screenshots.map((url, index) => (
              <div key={index} className="relative group">
                <img
                  src={url}
                  alt={`Screenshot ${index + 1}`}
                  className="h-24 w-24 object-cover rounded-lg border"
                />
                <button
                  onClick={() => removeScreenshot(index)}
                  className="absolute -top-2 -right-2 h-6 w-6 bg-red-500 text-white rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ))}
            {screenshots.length < 5 && (
              <label className="h-24 w-24 border-2 border-dashed rounded-lg flex flex-col items-center justify-center cursor-pointer hover:bg-muted transition-colors">
                <Upload className="h-6 w-6 text-muted-foreground mb-1" />
                <span className="text-xs text-muted-foreground">上传</span>
                <input
                  type="file"
                  accept="image/*"
                  onChange={handleFileUpload}
                  className="hidden"
                />
              </label>
            )}
          </div>
          <div className="text-xs text-muted-foreground">
            最多上传 5 张截图
          </div>
        </div>

        {/* 提交按钮 */}
        <div className="flex gap-3 pt-4">
          <Button
            variant="outline"
            className="flex-1"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            取消
          </Button>
          <Button
            className="flex-1"
            onClick={handleSubmit}
            disabled={isSubmitting}
          >
            {isSubmitting ? '提交中...' : (
              <>
                <Send className="h-4 w-4 mr-2" />
                提交反馈
              </>
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
```

### 反馈历史组件

```tsx
// frontend/src/components/feedback/FeedbackList.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Bug, Lightbulb, TrendingUp, Clock, CheckCircle, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

const FEEDBACK_TYPE_CONFIG = {
  bug: { label: 'Bug', icon: Bug, color: 'bg-red-500' },
  feature: { label: '建议', icon: Lightbulb, color: 'bg-yellow-500' },
  improvement: { label: '改进', icon: TrendingUp, color: 'bg-green-500' },
  content: { label: '内容', icon: FileText, color: 'bg-blue-500' },
  other: { label: '其他', icon: MessageSquare, color: 'bg-gray-500' },
}

const STATUS_CONFIG = {
  submitted: { label: '已提交', icon: Clock, color: 'text-gray-500' },
  reviewing: { label: '审核中', icon: Clock, color: 'text-blue-500' },
  confirmed: { label: '已确认', icon: CheckCircle, color: 'text-green-500' },
  in_progress: { label: '处理中', icon: TrendingUp, color: 'text-yellow-500' },
  resolved: { label: '已解决', icon: CheckCircle, color: 'text-green-500' },
  closed: { label: '已关闭', icon: XCircle, color: 'text-gray-500' },
  rejected: { label: '已拒绝', icon: XCircle, color: 'text-red-500' },
}

export function FeedbackList() {
  const [feedbacks, setFeedbacks] = useState([])
  const [filter, setFilter] = useState('all')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchFeedbacks()
  }, [filter])

  const fetchFeedbacks = async () => {
    setLoading(true)
    try {
      const url = filter === 'all'
        ? '/api/feedback'
        : `/api/feedback?type=${filter}`

      const response = await fetch(url)
      if (response.ok) {
        const data = await response.json()
        setFeedbacks(data.items)
      }
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div>加载中...</div>
  }

  return (
    <div className="space-y-4">
      {/* 筛选器 */}
      <div className="flex gap-2">
        <Select value={filter} onValueChange={setFilter}>
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部</SelectItem>
            <SelectItem value="bug">Bug</SelectItem>
            <SelectItem value="feature">建议</SelectItem>
            <SelectItem value="improvement">改进</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* 反馈列表 */}
      <div className="space-y-3">
        {feedbacks.map((feedback) => {
          const typeConfig = FEEDBACK_TYPE_CONFIG[feedback.type]
          const statusConfig = STATUS_CONFIG[feedback.status]
          const TypeIcon = typeConfig.icon
          const StatusIcon = statusConfig.icon

          return (
            <Card key={feedback.id}>
              <CardContent className="p-4">
                <div className="flex items-start gap-4">
                  {/* 类型图标 */}
                  <div className={cn("h-10 w-10 rounded-lg flex items-center justify-center", typeConfig.color)}>
                    <TypeIcon className="h-5 w-5 text-white" />
                  </div>

                  {/* 内容 */}
                  <div className="flex-1 space-y-2">
                    <div className="flex items-start justify-between">
                      <h4 className="font-medium">{feedback.title}</h4>
                      <Badge variant="outline" className={cn("flex items-center gap-1", statusConfig.color)}>
                        <StatusIcon className="h-3 w-3" />
                        {statusConfig.label}
                      </Badge>
                    </div>

                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {feedback.description}
                    </p>

                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span>#{feedback.id}</span>
                      <span>{new Date(feedback.created_at).toLocaleDateString()}</span>
                      {feedback.vote_count > 0 && (
                        <span>{feedback.vote_count} 人赞同</span>
                      )}
                    </div>

                    {feedback.response && (
                      <div className="bg-muted p-3 rounded-lg text-sm">
                        <div className="font-semibold text-xs text-muted-foreground mb-1">官方回复</div>
                        {feedback.response}
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}

        {feedbacks.length === 0 && (
          <Card>
            <CardContent className="p-8 text-center text-muted-foreground">
              还没有提交任何反馈
            </CardContent>
          </Card>
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
| `backend/models/feedback.py` | 创建 | 反馈数据模型 |
| `backend/services/feedback_service.py` | 创建 | 反馈业务逻辑 |
| `backend/api/routes/feedback.py` | 创建 | 反馈 API 路由 |
| `backend/utils/file_upload.py` | 创建 | 文件上传工具 |
| `frontend/src/components/feedback/FeedbackForm.tsx` | 创建 | 反馈表单组件 |
| `frontend/src/components/feedback/FeedbackList.tsx` | 创建 | 反馈列表组件 |
| `frontend/src/types/feedback.ts` | 创建 | 反馈类型定义 |

---

## 验收标准

- [ ] 支持多种反馈类型提交
- [ ] 支持截图上传（最多5张）
- [ ] 反馈列表正确展示和筛选
- [ ] 支持功能建议投票
- [ ] 管理员可更新反馈状态
- [ ] 用户可查看自己的反馈历史
- [ ] 反馈统计准确
- [ ] 移动端适配良好

---

## 参考文档

- M1-050: 友好拒绝机制
- M0-001: 核心命令清单
- shadcn/ui Form 组件文档
- AWS S3 文件上传最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
