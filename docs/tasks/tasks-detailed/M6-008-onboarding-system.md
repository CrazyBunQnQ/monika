# M6-008: 实现引导系统

**任务ID**: M6-008
**标题**: 实现引导系统
**类型**: fullstack (全栈开发)
**预估工时**: 8h
**依赖**: M6-056, M1-040

---

## 任务描述

实现新用户引导系统，包括首次登录引导、功能介绍、交互式教程等，帮助新用户快速理解游戏玩法和系统功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-008-01 | 设计引导流程 | 确定引导步骤和内容 | 1h |
| M6-008-02 | 实现引导状态管理 | 后端引导进度存储 | 1.5h |
| M6-008-03 | 实现引导数据模型 | 数据库表结构 | 1h |
| M6-008-04 | 实现引导 API | 进度保存/获取接口 | 1h |
| M6-008-05 | 实现 Tour 组件 | 前端引导组件 | 1.5h |
| M6-008-06 | 实现引导提示系统 | Tooltip/Hint 组件 | 1h |
| M6-008-07 | 编写引导内容 | 各功能模块说明 | 1h |

---

## 后端实现

### 引导数据模型

```python
# backend/models/onboarding.py
from datetime import datetime
from enum import Enum
from typing import Optional, List, Dict, Any
from pydantic import BaseModel, Field
from sqlalchemy import Column, Integer, String, Boolean, DateTime, JSON, ForeignKey
from sqlalchemy.orm import relationship

from database import Base


class OnboardingStep(str, Enum):
    """引导步骤枚举"""
    WELCOME = "welcome"
    CHARACTER_CREATE = "character_create"
    COMMAND_INPUT = "command_input"
    DICE_ROLL = "dice_roll"
    INVENTORY = "inventory"
    SANITY_CHECK = "sanity_check"
    COMBAT = "combat"
    SOCIAL = "social"
    MEMORY = "memory"
    COMPLETE = "complete"


class OnboardingProgress(Base):
    """引导进度表"""
    __tablename__ = "onboarding_progress"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, unique=True)

    # 引导状态
    is_started = Column(Boolean, default=False)
    is_completed = Column(Boolean, default=False)
    current_step = Column(String, default=OnboardingStep.WELCOME)

    # 完成的步骤列表
    completed_steps = Column(JSON, default=list)

    # 跳过的步骤
    skipped_steps = Column(JSON, default=list)

    # 额外数据（如选择的角色等）
    metadata = Column(JSON, default=dict)

    # 时间戳
    started_at = Column(DateTime, nullable=True)
    completed_at = Column(DateTime, nullable=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 关系
    user = relationship("User", back_populates="onboarding_progress")


class OnboardingContent(Base):
    """引导内容表"""
    __tablename__ = "onboarding_content"

    id = Column(Integer, primary_key=True, index=True)
    step_key = Column(String, unique=True, nullable=False)

    # 内容
    title = Column(String, nullable=False)
    description = Column(String, nullable=False)
    content = Column(JSON, nullable=False)  # 支持多语言

    # 目标元素选择器
    target_selector = Column(String)  # CSS选择器
    position = Column(String)  # top, bottom, left, right

    # 配置
    order = Column(Integer, default=0)
    is_skippable = Column(Boolean, default=True)
    is_required = Column(Boolean, default=False)

    # 触发条件
    trigger_condition = Column(JSON)  # JSON表达式

    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)


# Pydantic 模型
class OnboardingProgressCreate(BaseModel):
    """创建引导进度"""
    current_step: Optional[OnboardingStep] = OnboardingStep.WELCOME
    metadata: Optional[Dict[str, Any]] = {}


class OnboardingProgressUpdate(BaseModel):
    """更新引导进度"""
    current_step: Optional[OnboardingStep] = None
    completed_step: Optional[OnboardingStep] = None
    skipped_step: Optional[OnboardingStep] = None
    metadata: Optional[Dict[str, Any]] = None


class OnboardingProgressResponse(BaseModel):
    """引导进度响应"""
    is_started: bool
    is_completed: bool
    current_step: Optional[str]
    completed_steps: List[str]
    skipped_steps: List[str]
    metadata: Dict[str, Any]
    next_step: Optional[str]
    completion_percentage: float

    class Config:
        from_attributes = True


class OnboardingContentResponse(BaseModel):
    """引导内容响应"""
    step_key: str
    title: str
    description: str
    content: Dict[str, Any]
    target_selector: Optional[str]
    position: Optional[str]
    is_skippable: bool
    is_required: bool

    class Config:
        from_attributes = True
```

### 引导服务

```python
# backend/services/onboarding_service.py
from typing import List, Optional, Dict, Any
from datetime import datetime
from sqlalchemy.orm import Session

from models.onboarding import (
    OnboardingProgress,
    OnboardingContent,
    OnboardingStep,
    OnboardingProgressCreate,
    OnboardingProgressUpdate,
    OnboardingProgressResponse,
    OnboardingContentResponse,
)


class OnboardingService:
    """引导系统服务"""

    def __init__(self, db: Session):
        self.db = db

    def start_onboarding(self, user_id: int) -> OnboardingProgressResponse:
        """开始引导"""
        progress = self.db.query(OnboardingProgress).filter(
            OnboardingProgress.user_id == user_id
        ).first()

        if not progress:
            progress = OnboardingProgress(
                user_id=user_id,
                is_started=True,
                current_step=OnboardingStep.WELCOME.value,
                started_at=datetime.utcnow()
            )
            self.db.add(progress)
        else:
            progress.is_started = True
            progress.started_at = datetime.utcnow()

        self.db.commit()
        self.db.refresh(progress)
        return self._to_response(progress)

    def update_progress(
        self,
        user_id: int,
        update: OnboardingProgressUpdate
    ) -> OnboardingProgressResponse:
        """更新引导进度"""
        progress = self.db.query(OnboardingProgress).filter(
            OnboardingProgress.user_id == user_id
        ).first()

        if not progress:
            raise ValueError("Onboarding not started")

        # 更新当前步骤
        if update.current_step:
            progress.current_step = update.current_step.value

        # 标记完成步骤
        if update.completed_step:
            if update.completed_step.value not in progress.completed_steps:
                progress.completed_steps.append(update.completed_step.value)

        # 跳过步骤
        if update.skipped_step:
            if update.skipped_step.value not in progress.skipped_steps:
                progress.skipped_steps.append(update.skipped_step.value)

        # 更新元数据
        if update.metadata:
            progress.metadata.update(update.metadata)

        # 检查是否完成
        if update.completed_step == OnboardingStep.COMPLETE:
            progress.is_completed = True
            progress.completed_at = datetime.utcnow()

        self.db.commit()
        self.db.refresh(progress)
        return self._to_response(progress)

    def get_progress(self, user_id: int) -> OnboardingProgressResponse:
        """获取引导进度"""
        progress = self.db.query(OnboardingProgress).filter(
            OnboardingProgress.user_id == user_id
        ).first()

        if not progress:
            return OnboardingProgressResponse(
                is_started=False,
                is_completed=False,
                current_step=None,
                completed_steps=[],
                skipped_steps=[],
                metadata={},
                next_step=OnboardingStep.WELCOME.value,
                completion_percentage=0.0
            )

        return self._to_response(progress)

    def get_step_content(
        self,
        step: OnboardingStep,
        locale: str = "zh"
    ) -> Optional[OnboardingContentResponse]:
        """获取步骤内容"""
        content = self.db.query(OnboardingContent).filter(
            OnboardingContent.step_key == step.value
        ).first()

        if not content:
            return None

        # 返回本地化内容
        localized_content = content.content.get(locale, content.content.get("en", {}))

        return OnboardingContentResponse(
            step_key=content.step_key,
            title=localized_content.get("title", content.title),
            description=localized_content.get("description", content.description),
            content=localized_content,
            target_selector=content.target_selector,
            position=content.position,
            is_skippable=content.is_skippable,
            is_required=content.is_required
        )

    def get_all_steps(self, locale: str = "zh") -> List[OnboardingContentResponse]:
        """获取所有引导步骤"""
        contents = self.db.query(OnboardingContent).order_by(
            OnboardingContent.order
        ).all()

        result = []
        for content in contents:
            localized_content = content.content.get(locale, content.content.get("en", {}))
            result.append(OnboardingContentResponse(
                step_key=content.step_key,
                title=localized_content.get("title", content.title),
                description=localized_content.get("description", content.description),
                content=localized_content,
                target_selector=content.target_selector,
                position=content.position,
                is_skippable=content.is_skippable,
                is_required=content.is_required
            ))

        return result

    def reset_onboarding(self, user_id: int) -> OnboardingProgressResponse:
        """重置引导"""
        progress = self.db.query(OnboardingProgress).filter(
            OnboardingProgress.user_id == user_id
        ).first()

        if progress:
            self.db.delete(progress)
            self.db.commit()

        return self.start_onboarding(user_id)

    def _to_response(self, progress: OnboardingProgress) -> OnboardingProgressResponse:
        """转换为响应对象"""
        all_steps = [step.value for step in OnboardingStep]
        completed_count = len(progress.completed_steps)
        completion_percentage = (completed_count / len(all_steps)) * 100

        # 计算下一步
        next_step = None
        if not progress.is_completed:
            for step in all_steps:
                if step not in progress.completed_steps:
                    next_step = step
                    break

        return OnboardingProgressResponse(
            is_started=progress.is_started,
            is_completed=progress.is_completed,
            current_step=progress.current_step,
            completed_steps=progress.completed_steps or [],
            skipped_steps=progress.skipped_steps or [],
            metadata=progress.metadata or {},
            next_step=next_step,
            completion_percentage=completion_percentage
        )
```

### 引导 API 路由

```python
# backend/api/routes/onboarding.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from database import get_db
from services.onboarding_service import OnboardingService
from models.onboarding import (
    OnboardingStep,
    OnboardingProgressCreate,
    OnboardingProgressUpdate,
    OnboardingProgressResponse,
    OnboardingContentResponse,
)
from middleware.auth import get_current_user

router = APIRouter(prefix="/onboarding", tags=["onboarding"])


@router.post("/start", response_model=OnboardingProgressResponse)
async def start_onboarding(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """开始新用户引导"""
    service = OnboardingService(db)
    return service.start_onboarding(current_user.id)


@router.get("/progress", response_model=OnboardingProgressResponse)
async def get_progress(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取引导进度"""
    service = OnboardingService(db)
    return service.get_progress(current_user.id)


@router.put("/progress", response_model=OnboardingProgressResponse)
async def update_progress(
    update: OnboardingProgressUpdate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """更新引导进度"""
    service = OnboardingService(db)
    return service.update_progress(current_user.id, update)


@router.get("/steps", response_model=list[OnboardingContentResponse])
async def get_all_steps(
    locale: str = "zh",
    db: Session = Depends(get_db)
):
    """获取所有引导步骤"""
    service = OnboardingService(db)
    return service.get_all_steps(locale)


@router.get("/steps/{step}", response_model=OnboardingContentResponse)
async def get_step_content(
    step: OnboardingStep,
    locale: str = "zh",
    db: Session = Depends(get_db)
):
    """获取特定步骤内容"""
    service = OnboardingService(db)
    content = service.get_step_content(step, locale)
    if not content:
        raise HTTPException(status_code=404, detail="Step not found")
    return content


@router.post("/reset", response_model=OnboardingProgressResponse)
async def reset_onboarding(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """重置引导"""
    service = OnboardingService(db)
    return service.reset_onboarding(current_user.id)
```

### 引导内容初始化

```python
# backend/scripts/init_onboarding_content.py
from database import SessionLocal
from models.onboarding import OnboardingContent, OnboardingStep


def init_onboarding_content():
    """初始化引导内容"""
    db = SessionLocal()

    contents = [
        OnboardingContent(
            step_key=OnboardingStep.WELCOME.value,
            title="欢迎来到 CoC 跑团平台",
            description="了解如何开始你的第一次冒险",
            target_selector=".game-container",
            position="bottom",
            order=0,
            is_skippable=False,
            is_required=True,
            content={
                "zh": {
                    "title": "欢迎来到 CoC 跑团平台",
                    "description": "这是一个基于克苏鲁的呼唤 7 版规则的跑团平台。我们将引导你完成基础设置。",
                    "body": "点击「下一步」开始创建你的第一个角色",
                    "image": "/images/onboarding/welcome.png"
                },
                "en": {
                    "title": "Welcome to CoC TRPG Platform",
                    "description": "This is a Call of Cthulhu 7th Edition TRPG platform. We'll guide you through basic setup.",
                    "body": "Click 'Next' to create your first character",
                    "image": "/images/onboarding/welcome.png"
                }
            }
        ),
        OnboardingContent(
            step_key=OnboardingStep.CHARACTER_CREATE.value,
            title="创建角色",
            description="学习如何创建调查员角色",
            target_selector="[data-tour='character-create']",
            position="right",
            order=1,
            is_skippable=False,
            is_required=True,
            content={
                "zh": {
                    "title": "创建你的调查员",
                    "description": "点击角色卡片创建你的第一个调查员角色",
                    "body": "你需要设定姓名、职业、属性和技能",
                    "tips": ["姓名可以自由设定", "职业决定了基础技能", "属性可以通过掷骰获得"]
                }
            }
        ),
        OnboardingContent(
            step_key=OnboardingStep.COMMAND_INPUT.value,
            title="输入命令",
            description="学习如何与游戏交互",
            target_selector="[data-tour='command-input']",
            position="top",
            order=2,
            is_skippable=False,
            is_required=True,
            content={
                "zh": {
                    "title": "输入游戏命令",
                    "description": "在这里输入你的行动命令",
                    "body": "试试输入「我仔细查看房间」",
                    "examples": [
                        "我检查桌子",
                        "我询问守卫",
                        "我使用侦查技能"
                    ]
                }
            }
        ),
        OnboardingContent(
            step_key=OnboardingStep.DICE_ROLL.value,
            title="掷骰检定",
            description="了解如何进行技能检定",
            target_selector="[data-tour='dice-roll']",
            position="left",
            order=3,
            is_skippable=False,
            is_required=True,
            content={
                "zh": {
                    "title": "掷骰检定",
                    "description": "当需要判定行动结果时，系统会自动进行检定",
                    "body": "检定结果会显示在消息区域",
                    "tips": [
                        "检定结果 ≤ 技能值为成功",
                        "大成功（1）和大失败（100）有特殊效果"
                    ]
                }
            }
        ),
        OnboardingContent(
            step_key=OnboardingStep.INVENTORY.value,
            title="物品管理",
            description="管理你的物品和装备",
            target_selector="[data-tour='inventory']",
            position="right",
            order=4,
            is_skippable=True,
            is_required=False,
            content={
                "zh": {
                    "title": "物品栏",
                    "description": "在这里查看和管理你的物品",
                    "body": "你可以使用、丢弃或查看物品详情"
                }
            }
        ),
        OnboardingContent(
            step_key=OnboardingStep.SANITY_CHECK.value,
            title="理智检定",
            description="了解 SAN 值系统",
            target_selector="[data-tour='sanity-check']",
            position="bottom",
            order=5,
            is_skippable=True,
            is_required=False,
            content={
                "zh": {
                    "title": "理智值 (SAN)",
                    "description": "SAN 值代表你的角色精神稳定度",
                    "body": "遇到恐怖事物时会进行 SAN 检定",
                    "tips": [
                        "SAN 降为 0 会陷入永久疯狂",
                        "小心保护你的理智值"
                    ]
                }
            }
        ),
        OnboardingContent(
            step_key=OnboardingStep.COMPLETE.value,
            title="引导完成",
            description="开始你的冒险",
            target_selector=".game-container",
            position="bottom",
            order=6,
            is_skippable=False,
            is_required=True,
            content={
                "zh": {
                    "title": "准备就绪！",
                    "description": "你已经了解了基础操作",
                    "body": "现在开始你的第一次冒险吧！",
                    "next_actions": [
                        "开始新游戏",
                        "查看帮助文档",
                        "调整设置"
                    ]
                }
            }
        )
    ]

    try:
        for content in contents:
            existing = db.query(OnboardingContent).filter(
                OnboardingContent.step_key == content.step_key
            ).first()
            if not existing:
                db.add(content)

        db.commit()
        print(f"初始化了 {len(contents)} 个引导步骤")
    except Exception as e:
        db.rollback()
        print(f"初始化失败: {e}")
    finally:
        db.close()


if __name__ == "__main__":
    init_onboarding_content()
```

---

## 前端实现

### Tour 引导组件

```tsx
// frontend/src/components/onboarding/Tour.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { X, ChevronLeft, ChevronRight, Info } from 'lucide-react'
import { cn } from '@/lib/utils'

interface TourStep {
  step_key: string
  title: string
  description: string
  content: {
    body?: string
    tips?: string[]
    examples?: string[]
    next_actions?: string[]
    image?: string
  }
  target_selector?: string
  position?: 'top' | 'bottom' | 'left' | 'right'
  is_skippable: boolean
  is_required: boolean
}

interface TourProps {
  steps: TourStep[]
  currentStepIndex: number
  onComplete: () => void
  onSkip: () => void
  onNext: () => void
  onPrevious: () => void
  isOpen: boolean
}

export function Tour({
  steps,
  currentStepIndex,
  onComplete,
  onSkip,
  onNext,
  onPrevious,
  isOpen
}: TourProps) {
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const currentStep = steps[currentStepIndex]

  useEffect(() => {
    if (isOpen && currentStep?.target_selector) {
      const target = document.querySelector(currentStep.target_selector)
      if (target) {
        const rect = target.getBoundingClientRect()
        setPosition({
          top: rect.top + rect.height / 2,
          left: rect.left + rect.width / 2
        })

        // 高亮目标元素
        target.classList.add('ring-2', 'ring-primary', 'ring-offset-2')
        return () => {
          target.classList.remove('ring-2', 'ring-primary', 'ring-offset-2')
        }
      }
    }
  }, [isOpen, currentStep])

  if (!isOpen || !currentStep) return null

  const progress = ((currentStepIndex + 1) / steps.length) * 100

  return (
    <>
      {/* 遮罩 */}
      <div className="fixed inset-0 bg-black/50 z-40" onClick={onSkip} />

      {/* 引导卡片 */}
      <Card
        className={cn(
          "fixed z-50 w-96 shadow-2xl",
          currentStep.position === 'top' && "transform -translate-y-full -translate-x-1/2",
          currentStep.position === 'bottom' && "transform translate-y-2 -translate-x-1/2",
          currentStep.position === 'left' && "transform -translate-x-full -translate-y-1/2",
          currentStep.position === 'right' && "transform translate-x-2 -translate-y-1/2"
        )}
        style={{
          top: currentStep.position === 'top' || currentStep.position === 'bottom'
            ? position.top
            : position.top,
          left: position.left
        }}
      >
        <CardContent className="p-6">
          {/* 关闭按钮 */}
          <div className="flex justify-between items-start mb-4">
            <Badge variant="outline">
              步骤 {currentStepIndex + 1} / {steps.length}
            </Badge>
            {currentStep.is_skippable && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onSkip}
                className="h-6 w-6 p-0"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>

          {/* 进度条 */}
          <div className="w-full bg-secondary h-1 rounded-full mb-4">
            <div
              className="bg-primary h-1 rounded-full transition-all"
              style={{ width: `${progress}%` }}
            />
          </div>

          {/* 内容 */}
          <div className="space-y-4">
            {/* 标题 */}
            <h3 className="text-lg font-semibold flex items-center">
              <Info className="h-4 w-4 mr-2 text-primary" />
              {currentStep.title}
            </h3>

            {/* 描述 */}
            <p className="text-sm text-muted-foreground">
              {currentStep.description}
            </p>

            {/* 正文 */}
            {currentStep.content.body && (
              <p className="text-sm">
                {currentStep.content.body}
              </p>
            )}

            {/* 提示 */}
            {currentStep.content.tips && currentStep.content.tips.length > 0 && (
              <div className="bg-muted p-3 rounded-lg">
                <div className="text-xs font-semibold mb-2">提示</div>
                <ul className="text-xs space-y-1">
                  {currentStep.content.tips.map((tip, idx) => (
                    <li key={idx} className="flex items-start">
                      <span className="mr-2">•</span>
                      <span>{tip}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* 示例 */}
            {currentStep.content.examples && currentStep.content.examples.length > 0 && (
              <div className="space-y-2">
                <div className="text-xs font-semibold">示例命令</div>
                {currentStep.content.examples.map((example, idx) => (
                  <code
                    key={idx}
                    className="block text-xs bg-muted p-2 rounded"
                  >
                    {example}
                  </code>
                ))}
              </div>
            )}

            {/* 图片 */}
            {currentStep.content.image && (
              <div className="relative aspect-video rounded-lg overflow-hidden">
                <img
                  src={currentStep.content.image}
                  alt={currentStep.title}
                  className="object-cover w-full h-full"
                />
              </div>
            )}
          </div>

          {/* 操作按钮 */}
          <div className="flex justify-between mt-6">
            <Button
              variant="outline"
              size="sm"
              onClick={onPrevious}
              disabled={currentStepIndex === 0}
            >
              <ChevronLeft className="h-4 w-4 mr-1" />
              上一步
            </Button>

            <div className="flex gap-2">
              {currentStep.is_skippable && (
                <Button variant="ghost" size="sm" onClick={onSkip}>
                  跳过
                </Button>
              )}

              {currentStepIndex === steps.length - 1 ? (
                <Button size="sm" onClick={onComplete}>
                  完成
                </Button>
              ) : (
                <Button size="sm" onClick={onNext}>
                  下一步
                  <ChevronRight className="h-4 w-4 ml-1" />
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </>
  )
}
```

### 引导管理 Hook

```tsx
// frontend/src/hooks/useOnboarding.ts
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { useToast } from '@/hooks/use-toast'

interface OnboardingProgress {
  is_started: boolean
  is_completed: boolean
  current_step: string | null
  completed_steps: string[]
  skipped_steps: string[]
  metadata: Record<string, any>
  next_step: string | null
  completion_percentage: number
}

interface OnboardingStep {
  step_key: string
  title: string
  description: string
  content: any
  target_selector?: string
  position?: string
  is_skippable: boolean
  is_required: boolean
}

export function useOnboarding() {
  const { user } = useAuth()
  const { toast } = useToast()

  const [progress, setProgress] = useState<OnboardingProgress | null>(null)
  const [steps, setSteps] = useState<OnboardingStep[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isTourOpen, setIsTourOpen] = useState(false)
  const [currentStepIndex, setCurrentStepIndex] = useState(0)

  // 获取引导进度
  const fetchProgress = useCallback(async () => {
    if (!user) return

    setIsLoading(true)
    try {
      const response = await fetch('/api/onboarding/progress', {
        headers: {
          Authorization: `Bearer ${await user.getIdToken()}`
        }
      })

      if (!response.ok) throw new Error('Failed to fetch progress')

      const data: OnboardingProgress = await response.json()
      setProgress(data)

      // 如果未完成且有下一步，自动打开引导
      if (!data.is_completed && data.next_step) {
        // 等待页面加载后再打开
        setTimeout(() => {
          setIsTourOpen(true)
        }, 1000)
      }
    } catch (error) {
      console.error('Failed to fetch onboarding progress:', error)
    } finally {
      setIsLoading(false)
    }
  }, [user])

  // 获取所有步骤
  const fetchSteps = useCallback(async () => {
    if (!user) return

    try {
      const response = await fetch('/api/onboarding/steps?locale=zh', {
        headers: {
          Authorization: `Bearer ${await user.getIdToken()}`
        }
      })

      if (!response.ok) throw new Error('Failed to fetch steps')

      const data: OnboardingStep[] = await response.json()
      setSteps(data)
    } catch (error) {
      console.error('Failed to fetch onboarding steps:', error)
    }
  }, [user])

  // 开始引导
  const startOnboarding = useCallback(async () => {
    if (!user) return

    setIsLoading(true)
    try {
      const response = await fetch('/api/onboarding/start', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${await user.getIdToken()}`
        }
      })

      if (!response.ok) throw new Error('Failed to start onboarding')

      const data: OnboardingProgress = await response.json()
      setProgress(data)
      setIsTourOpen(true)
      setCurrentStepIndex(0)

      toast({
        title: "引导开始",
        description: "让我们开始了解 CoC 跑团平台"
      })
    } catch (error) {
      toast({
        variant: "destructive",
        title: "启动失败",
        description: "无法启动引导系统"
      })
    } finally {
      setIsLoading(false)
    }
  }, [user, toast])

  // 更新进度
  const updateProgress = useCallback(async (updates: {
    current_step?: string
    completed_step?: string
    skipped_step?: string
    metadata?: Record<string, any>
  }) => {
    if (!user) return

    try {
      const response = await fetch('/api/onboarding/progress', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${await user.getIdToken()}`
        },
        body: JSON.stringify(updates)
      })

      if (!response.ok) throw new Error('Failed to update progress')

      const data: OnboardingProgress = await response.json()
      setProgress(data)
      return data
    } catch (error) {
      console.error('Failed to update progress:', error)
      return null
    }
  }, [user])

  // 下一步
  const handleNext = useCallback(async () => {
    const currentStep = steps[currentStepIndex]
    if (!currentStep) return

    const newProgress = await updateProgress({
      completed_step: currentStep.step_key
    })

    if (newProgress && newProgress.next_step) {
      const nextIndex = steps.findIndex(s => s.step_key === newProgress.next_step)
      setCurrentStepIndex(nextIndex)
    } else {
      // 完成
      setIsTourOpen(false)
      toast({
        title: "引导完成！",
        description: "你现在可以开始你的冒险了"
      })
    }
  }, [currentStepIndex, steps, updateProgress, toast])

  // 上一步
  const handlePrevious = useCallback(() => {
    setCurrentStepIndex(Math.max(0, currentStepIndex - 1))
  }, [currentStepIndex])

  // 跳过
  const handleSkip = useCallback(async () => {
    const currentStep = steps[currentStepIndex]
    if (!currentStep) return

    await updateProgress({
      skipped_step: currentStep.step_key
    })
    setIsTourOpen(false)

    toast({
      title: "引导已跳过",
      description: "你可以随时从设置中重新开始引导"
    })
  }, [currentStepIndex, steps, updateProgress, toast])

  // 完成
  const handleComplete = useCallback(async () => {
    const currentStep = steps[currentStepIndex]
    if (!currentStep) return

    await updateProgress({
      completed_step: currentStep.step_key
    })
    setIsTourOpen(false)

    toast({
      title: "引导完成！",
      description: "你现在可以开始你的冒险了"
    })
  }, [currentStepIndex, steps, updateProgress, toast])

  // 重置引导
  const resetOnboarding = useCallback(async () => {
    if (!user) return

    try {
      const response = await fetch('/api/onboarding/reset', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${await user.getIdToken()}`
        }
      })

      if (!response.ok) throw new Error('Failed to reset onboarding')

      const data: OnboardingProgress = await response.json()
      setProgress(data)
      setIsTourOpen(true)
      setCurrentStepIndex(0)

      toast({
        title: "引导已重置",
        description: "让我们重新开始"
      })
    } catch (error) {
      toast({
        variant: "destructive",
        title: "重置失败",
        description: "无法重置引导系统"
      })
    }
  }, [user, toast])

  // 初始化
  useEffect(() => {
    if (user) {
      fetchProgress()
      fetchSteps()
    }
  }, [user, fetchProgress, fetchSteps])

  return {
    progress,
    steps,
    isLoading,
    isTourOpen,
    currentStepIndex,
    setIsTourOpen,
    startOnboarding,
    handleNext,
    handlePrevious,
    handleSkip,
    handleComplete,
    resetOnboarding,
    completionPercentage: progress?.completion_percentage || 0
  }
}
```

### 引导入口组件

```tsx
// frontend/src/components/onboarding/OnboardingProvider.tsx
import { ReactNode } from 'react'
import { Tour } from './Tour'
import { useOnboarding } from '@/hooks/useOnboarding'
import { Button } from '@/components/ui/button'
import { GraduationCap } from 'lucide-react'

interface OnboardingProviderProps {
  children: ReactNode
}

export function OnboardingProvider({ children }: OnboardingProviderProps) {
  const {
    steps,
    isTourOpen,
    currentStepIndex,
    setIsTourOpen,
    startOnboarding,
    handleNext,
    handlePrevious,
    handleSkip,
    handleComplete,
    completionPercentage
  } = useOnboarding()

  return (
    <>
      {children}

      {/* 引导入口按钮 */}
      {completionPercentage < 100 && (
        <Button
          variant="outline"
          size="sm"
          className="fixed bottom-4 right-4 z-30"
          onClick={() => setIsTourOpen(!isTourOpen)}
        >
          <GraduationCap className="h-4 w-4 mr-2" />
          教程 {Math.round(completionPercentage)}%
        </Button>
      )}

      {/* 引导 Tour */}
      <Tour
        steps={steps}
        currentStepIndex={currentStepIndex}
        isOpen={isTourOpen}
        onComplete={handleComplete}
        onSkip={handleSkip}
        onNext={handleNext}
        onPrevious={handlePrevious}
      />
    </>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/models/onboarding.py` | 创建 | 引导数据模型 |
| `backend/services/onboarding_service.py` | 创建 | 引导业务逻辑 |
| `backend/api/routes/onboarding.py` | 创建 | 引导 API 路由 |
| `backend/scripts/init_onboarding_content.py` | 创建 | 引导内容初始化脚本 |
| `frontend/src/components/onboarding/Tour.tsx` | 创建 | Tour 引导组件 |
| `frontend/src/components/onboarding/OnboardingProvider.tsx` | 创建 | 引导提供者组件 |
| `frontend/src/hooks/useOnboarding.ts` | 创建 | 引导管理 Hook |
| `frontend/src/types/onboarding.ts` | 创建 | 引导类型定义 |
| `frontend/src/App.tsx` | 修改 | 集成 OnboardingProvider |

---

## 验收标准

- [ ] 新用户首次登录自动触发引导
- [ ] 引导覆盖核心功能模块
- [ ] 支持跳过和重新开始
- [ ] 引导进度正确保存和恢复
- [ ] 目标元素正确高亮定位
- [ ] 支持中英文内容
- [ ] 移动端适配良好
- [ ] 无障碍支持

---

## 参考文档

- M6-056: 实现工具提示
- M1-040: 角色系统
- M0-001: 核心命令清单
- React Joyride: https://react-joyride.com/
- shadcn/ui Tour 组件文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
