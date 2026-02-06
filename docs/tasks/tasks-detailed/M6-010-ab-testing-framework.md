# M6-010: 实现 A/B 测试框架

**任务ID**: M6-010
**标题**: 实现 A/B 测试框架
**类型**: fullstack (全栈开发)
**预估工时**: 10h
**依赖**: M1-040

---

## 任务描述

实现 A/B 测试框架，支持功能开关、实验分组、效果追踪、数据统计分析，用于优化用户体验和功能迭代。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-010-01 | 设计 A/B 测试数据模型 | 实验配置结构 | 1.5h |
| M6-010-02 | 实现实验管理服务 | 实验创建/更新 | 2h |
| M6-010-03 | 实现用户分组算法 | 分桶逻辑 | 1.5h |
| M6-010-04 | 实现事件追踪系统 | 行为记录 | 1.5h |
| M6-010-05 | 实现 A/B 测试 API | 配置/查询接口 | 1h |
| M6-010-06 | 实现前端 SDK | 客户端集成 | 1.5h |
| M6-010-07 | 实现统计分析面板 | 效果对比 | 1h |
| M6-010-08 | 编写使用文档 | 开发者指南 | 1h |

---

## 后端实现

### A/B 测试数据模型

```python
# backend/models/ab_testing.py
from datetime import datetime, timedelta
from enum import Enum
from typing import Optional, List, Dict, Any
from pydantic import BaseModel, Field
from sqlalchemy import Column, Integer, String, Text, DateTime, Boolean, Float, JSON, ForeignKey
from sqlalchemy.orm import relationship

from database import Base


class ExperimentStatus(str, Enum):
    """实验状态"""
    DRAFT = "draft"            # 草稿
    RUNNING = "running"        # 进行中
    PAUSED = "paused"          # 暂停
    COMPLETED = "completed"    # 已完成
    ARCHIVED = "archived"      # 已归档


class TrafficAllocationType(str, Enum):
    """流量分配类型"""
    UNIFORM = "uniform"        # 均匀分配
    MANUAL = "manual"          # 手动分配
    WEIGHTED = "weighted"      # 权重分配


class Experiment(Base):
    """实验表"""
    __tablename__ = "ab_experiments"

    id = Column(Integer, primary_key=True, index=True)

    # 基本信息
    name = Column(String(200), nullable=False, unique=True)
    key = Column(String(100), nullable=False, unique=True, index=True)
    description = Column(Text)

    # 状态和配置
    status = Column(String, default=ExperimentStatus.DRAFT)
    traffic_allocation = Column(Float, default=1.0)  # 流量百分比 (0-1)

    # 变体配置
    variants = Column(JSON, nullable=False)  # [{"name": "control", "weight": 50}, ...]
    allocation_type = Column(String, default=TrafficAllocationType.UNIFORM)

    # 目标指标
    metrics = Column(JSON)  # [{"name": "conversion", "type": "binary"}, ...]

    # 分桶配置
    bucketing_key = Column(String, default="user_id")  # user_id, session_id, custom
    hash_salt = Column(String)  # 用于一致性哈希的盐值

    # 时间配置
    start_date = Column(DateTime, nullable=True)
    end_date = Column(DateTime, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 统计
    total Participants = Column(Integer, default=0)
    required_sample_size = Column(Integer)  # 所需样本量

    # 决策
    winning_variant = Column(String)  # 获胜变体
    decision_reason = Column(Text)    # 决策原因


class ExperimentParticipant(Base):
    """实验参与者表"""
    __tablename__ = "ab_participants"

    id = Column(Integer, primary_key=True, index=True)
    experiment_id = Column(Integer, ForeignKey("ab_experiments.id"), nullable=False)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=True)  # 可为空（匿名用户）

    # 分配信息
    variant = Column(String, nullable=False)  # 分配到的变体
    bucket = Column(Integer, nullable=False)  # 分桶值 (0-99)

    # 时间戳
    enrolled_at = Column(DateTime, default=datetime.utcnow, index=True)

    # 唯一约束
    __table_args__ = (
        {'sqlite_autoincrement': True}
    )


class ExperimentEvent(Base):
    """实验事件表"""
    __tablename__ = "ab_events"

    id = Column(Integer, primary_key=True, index=True)
    experiment_id = Column(Integer, ForeignKey("ab_experiments.id"), nullable=False)
    participant_id = Column(Integer, ForeignKey("ab_participants.id"), nullable=False)

    # 事件信息
    event_type = Column(String, nullable=False)  # conversion, click, view, etc.
    event_name = Column(String)
    value = Column(Float)  # 事件值（用于数值型指标）

    # 上下文
    properties = Column(JSON)  # 额外属性

    # 时间戳
    occurred_at = Column(DateTime, default=datetime.utcnow, index=True)


class ExperimentSnapshot(Base):
    """实验快照表（用于统计）"""
    __tablename__ = "ab_snapshots"

    id = Column(Integer, primary_key=True, index=True)
    experiment_id = Column(Integer, ForeignKey("ab_experiments.id"), nullable=False)

    # 快照数据
    variant_stats = Column(JSON, nullable=False)  # 各变体统计数据

    # 统计结果
    statistical_significance = Column(Float)  # 统计显著性
    confidence_level = Column(Float, default=0.95)
    p_value = Column(Float)

    # 时间戳
    captured_at = Column(DateTime, default=datetime.utcnow, index=True)


# Pydantic 模型
class VariantCreate(BaseModel):
    name: str
    weight: Optional[float] = None
    config: Optional[Dict[str, Any]] = {}


class ExperimentCreate(BaseModel):
    name: str
    key: str
    description: Optional[str] = None
    traffic_allocation: float = 1.0
    variants: List[VariantCreate]
    metrics: List[Dict[str, str]]
    start_date: Optional[datetime] = None
    end_date: Optional[datetime] = None
    required_sample_size: Optional[int] = None


class ExperimentResponse(BaseModel):
    id: int
    name: str
    key: str
    description: Optional[str]
    status: ExperimentStatus
    traffic_allocation: float
    variants: List[Dict[str, Any]]
    metrics: List[Dict[str, str]]
    start_date: Optional[datetime]
    end_date: Optional[datetime]
    total_participants: int
    winning_variant: Optional[str]

    class Config:
        from_attributes = True


class VariantAssignment(BaseModel):
    """变体分配结果"""
    experiment_key: str
    variant: str
    config: Dict[str, Any]
    is_in_experiment: bool
```

### A/B 测试服务

```python
# backend/services/ab_testing_service.py
import hashlib
from typing import List, Optional, Dict, Any
from datetime import datetime
from sqlalchemy.orm import Session
from sqlalchemy import and_

from models.ab_testing import (
    Experiment,
    ExperimentParticipant,
    ExperimentEvent,
    ExperimentStatus,
    ExperimentCreate,
    ExperimentResponse,
    VariantAssignment,
)
from models.user import User


class ABTestingService:
    """A/B 测试服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_experiment(self, data: ExperimentCreate) -> ExperimentResponse:
        """创建实验"""
        # 生成哈希盐值
        hash_salt = hashlib.md5(data.key.encode()).hexdigest()[:8]

        # 标准化变体权重
        variants = self._normalize_variants(data.variants)

        experiment = Experiment(
            name=data.name,
            key=data.key,
            description=data.description,
            traffic_allocation=data.traffic_allocation,
            variants=variants,
            metrics=data.metrics,
            start_date=data.start_date,
            end_date=data.end_date,
            required_sample_size=data.required_sample_size,
            hash_salt=hash_salt,
            status=ExperimentStatus.DRAFT
        )

        self.db.add(experiment)
        self.db.commit()
        self.db.refresh(experiment)

        return self._to_response(experiment)

    def get_assignment(
        self,
        experiment_key: str,
        user_id: Optional[int],
        session_id: Optional[str] = None
    ) -> VariantAssignment:
        """获取用户在实验中的变体分配"""
        experiment = self.db.query(Experiment).filter(
            Experiment.key == experiment_key,
            Experiment.status == ExperimentStatus.RUNNING
        ).first()

        if not experiment:
            return VariantAssignment(
                experiment_key=experiment_key,
                variant="control",
                config={},
                is_in_experiment=False
            )

        # 检查是否在实验流量中
        bucket = self._get_bucket(experiment, user_id, session_id)
        if bucket >= int(experiment.traffic_allocation * 100):
            return VariantAssignment(
                experiment_key=experiment_key,
                variant="control",
                config={},
                is_in_experiment=False
            )

        # 检查是否已分配
        assignment = self._get_existing_assignment(experiment.id, user_id, session_id)

        if assignment:
            variant_config = self._get_variant_config(experiment, assignment.variant)
            return VariantAssignment(
                experiment_key=experiment_key,
                variant=assignment.variant,
                config=variant_config,
                is_in_experiment=True
            )

        # 新分配
        variant = self._assign_variant(experiment, bucket)
        self._record_participation(experiment.id, user_id, variant, bucket)

        variant_config = self._get_variant_config(experiment, variant)

        # 更新参与者数
        experiment.total_participants += 1
        self.db.commit()

        return VariantAssignment(
            experiment_key=experiment_key,
            variant=variant,
            config=variant_config,
            is_in_experiment=True
        )

    def track_event(
        self,
        experiment_key: str,
        user_id: Optional[int],
        session_id: Optional[str],
        event_type: str,
        event_name: Optional[str] = None,
        value: Optional[float] = None,
        properties: Optional[Dict[str, Any]] = None
    ) -> bool:
        """追踪实验事件"""
        # 获取参与记录
        assignment = self._get_existing_assignment_by_key(
            experiment_key, user_id, session_id
        )

        if not assignment:
            return False

        event = ExperimentEvent(
            experiment_id=assignment.experiment_id,
            participant_id=assignment.id,
            event_type=event_type,
            event_name=event_name,
            value=value,
            properties=properties or {}
        )

        self.db.add(event)
        self.db.commit()

        return True

    def get_experiment_stats(self, experiment_id: int) -> Dict[str, Any]:
        """获取实验统计数据"""
        experiment = self.db.query(Experiment).filter(
            Experiment.id == experiment_id
        ).first()

        if not experiment:
            raise ValueError("Experiment not found")

        # 参与者统计
        participants = self.db.query(ExperimentParticipant).filter(
            ExperimentParticipant.experiment_id == experiment_id
        ).all()

        variant_counts = {}
        for p in participants:
            variant_counts[p.variant] = variant_counts.get(p.variant, 0) + 1

        # 事件统计
        events = self.db.query(ExperimentEvent).filter(
            ExperimentEvent.experiment_id == experiment_id
        ).all()

        variant_metrics = {}
        for variant in experiment.variants:
            variant_name = variant["name"]
            variant_participants = [
                p for p in participants if p.variant == variant_name
            ]

            if not variant_participants:
                continue

            participant_ids = [p.id for p in variant_participants]

            variant_metrics[variant_name] = {
                "participants": len(variant_participants),
                "events": {}
            }

            for metric in experiment.metrics or []:
                metric_name = metric.get("name")
                metric_type = metric.get("type", "binary")

                metric_events = [
                    e for e in events
                    if e.participant_id in participant_ids
                    and e.event_type == metric_name
                ]

                if metric_type == "binary":
                    conversion_count = len(set(e.participant_id for e in metric_events))
                    conversion_rate = conversion_count / len(variant_participants)
                    variant_metrics[variant_name]["events"][metric_name] = {
                        "type": "binary",
                        "count": conversion_count,
                        "rate": round(conversion_rate, 4)
                    }
                else:
                    total_value = sum(e.value or 0 for e in metric_events)
                    avg_value = total_value / len(metric_events) if metric_events else 0
                    variant_metrics[variant_name]["events"][metric_name] = {
                        "type": "numeric",
                        "total": total_value,
                        "average": round(avg_value, 2),
                        "count": len(metric_events)
                    }

        return {
            "experiment_id": experiment_id,
            "experiment_name": experiment.name,
            "status": experiment.status,
            "total_participants": len(participants),
            "variant_metrics": variant_metrics,
            "is_significant": self._check_significance(variant_metrics)
        }

    def _normalize_variants(self, variants: List) -> List[Dict[str, Any]]:
        """标准化变体权重"""
        if not variants:
            return [{"name": "control", "weight": 100}]

        # 如果没有权重，均分
        if not any(v.get("weight") for v in variants):
            weight = 100 / len(variants)
            return [
                {**v, "weight": weight, "config": v.get("config", {})}
                for v in variants
            ]

        # 标准化权重到 100
        total_weight = sum(v.get("weight", 0) for v in variants)
        return [
            {
                **v,
                "weight": (v.get("weight", 0) / total_weight) * 100,
                "config": v.get("config", {})
            }
            for v in variants
        ]

    def _get_bucket(
        self,
        experiment: Experiment,
        user_id: Optional[int],
        session_id: Optional[str]
    ) -> int:
        """获取分桶值 (0-99)"""
        if experiment.bucketing_key == "user_id" and user_id:
            key = f"{experiment.hash_salt}:{user_id}"
        elif session_id:
            key = f"{experiment.hash_salt}:{session_id}"
        else:
            key = f"{experiment.hash_salt}:{id(self)}"

        hash_val = int(hashlib.md5(key.encode()).hexdigest(), 16)
        return hash_val % 100

    def _get_existing_assignment(
        self,
        experiment_id: int,
        user_id: Optional[int],
        session_id: Optional[str]
    ) -> Optional[ExperimentParticipant]:
        """获取现有分配"""
        query = self.db.query(ExperimentParticipant).filter(
            ExperimentParticipant.experiment_id == experiment_id
        )

        if user_id:
            query = query.filter(ExperimentParticipant.user_id == user_id)
        else:
            # 匿名用户通过时间戳近似匹配
            pass

        return query.first()

    def _get_existing_assignment_by_key(
        self,
        experiment_key: str,
        user_id: Optional[int],
        session_id: Optional[str]
    ) -> Optional[ExperimentParticipant]:
        """通过 key 获取分配"""
        experiment = self.db.query(Experiment).filter(
            Experiment.key == experiment_key
        ).first()

        if not experiment:
            return None

        return self._get_existing_assignment(experiment.id, user_id, session_id)

    def _assign_variant(self, experiment: Experiment, bucket: int) -> str:
        """根据分桶值分配变体"""
        cumulative = 0
        for variant in experiment.variants:
            cumulative += variant["weight"]
            if bucket < cumulative:
                return variant["name"]

        return experiment.variants[0]["name"]

    def _get_variant_config(self, experiment: Experiment, variant_name: str) -> Dict[str, Any]:
        """获取变体配置"""
        for variant in experiment.variants:
            if variant["name"] == variant_name:
                return variant.get("config", {})
        return {}

    def _record_participation(
        self,
        experiment_id: int,
        user_id: Optional[int],
        variant: str,
        bucket: int
    ):
        """记录参与"""
        participant = ExperimentParticipant(
            experiment_id=experiment_id,
            user_id=user_id,
            variant=variant,
            bucket=bucket
        )
        self.db.add(participant)

    def _check_significance(self, variant_metrics: Dict) -> bool:
        """检查统计显著性（简化版）"""
        # TODO: 实现真正的统计检验（如 Z-test, Chi-square test）
        return False

    def _to_response(self, experiment: Experiment) -> ExperimentResponse:
        """转换为响应对象"""
        return ExperimentResponse(
            id=experiment.id,
            name=experiment.name,
            key=experiment.key,
            description=experiment.description,
            status=experiment.status,
            traffic_allocation=experiment.traffic_allocation,
            variants=experiment.variants,
            metrics=experiment.metrics,
            start_date=experiment.start_date,
            end_date=experiment.end_date,
            total_participants=experiment.total_participants,
            winning_variant=experiment.winning_variant
        )
```

### A/B 测试 API

```python
# backend/api/routes/ab_testing.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from database import get_db
from services.ab_testing_service import ABTestingService
from models.ab_testing import (
    ExperimentCreate,
    ExperimentResponse,
    VariantAssignment,
)
from middleware.auth import get_current_user, optional_auth

router = APIRouter(prefix="/ab-testing", tags=["ab-testing"])


@router.post("/experiments", response_model=ExperimentResponse)
async def create_experiment(
    data: ExperimentCreate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """创建实验（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    service = ABTestingService(db)
    return service.create_experiment(data)


@router.get("/experiments", response_model=list[ExperimentResponse])
async def list_experiments(
    status: str = None,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """列出实验"""
    # TODO: 实现列表查询
    pass


@router.get("/experiments/{experiment_id}/stats")
async def get_experiment_stats(
    experiment_id: int,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_user)
):
    """获取实验统计（管理员）"""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin only")

    service = ABTestingService(db)
    return service.get_experiment_stats(experiment_id)


@router.post("/assign", response_model=VariantAssignment)
async def get_variant_assignment(
    experiment_key: str,
    db: Session = Depends(get_db),
    current_user = Depends(optional_auth)
):
    """获取变体分配"""
    user_id = current_user.id if current_user else None
    session_id = None  # TODO: 从 cookie/header 获取

    service = ABTestingService(db)
    return service.get_assignment(experiment_key, user_id, session_id)


@router.post("/track")
async def track_event(
    experiment_key: str,
    event_type: str,
    event_name: str = None,
    value: float = None,
    properties: dict = None,
    db: Session = Depends(get_db),
    current_user = Depends(optional_auth)
):
    """追踪实验事件"""
    user_id = current_user.id if current_user else None
    session_id = None  # TODO: 从 cookie/header 获取

    service = ABTestingService(db)
    success = service.track_event(
        experiment_key, user_id, session_id,
        event_type, event_name, value, properties
    )

    return {"success": success}
```

---

## 前端实现

### A/B 测试 SDK

```tsx
// frontend/src/lib/ab-testing/ABTestClient.ts
class ABTestClient {
  private baseURL: string
  private assignments: Map<string, string> = new Map()
  private sessionId: string

  constructor(baseURL: string = '/api/ab-testing') {
    this.baseURL = baseURL
    this.sessionId = this.getOrCreateSessionId()
  }

  /**
   * 获取变体分配
   */
  async getVariant(experimentKey: string): Promise<{
    variant: string
    config: Record<string, any>
    isInExperiment: boolean
  }> {
    // 检查缓存
    if (this.assignments.has(experimentKey)) {
      const variant = this.assignments.get(experimentKey)!
      return {
        variant,
        config: {},
        isInExperiment: true
      }
    }

    try {
      const response = await fetch(`${this.baseURL}/assign`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ experiment_key: experimentKey })
      })

      if (!response.ok) {
        console.warn('A/B test assignment failed:', experimentKey)
        return { variant: 'control', config: {}, isInExperiment: false }
      }

      const data = await response.json()
      this.assignments.set(experimentKey, data.variant)

      return {
        variant: data.variant,
        config: data.config || {},
        isInExperiment: data.is_in_experiment
      }
    } catch (error) {
      console.error('A/B test error:', error)
      return { variant: 'control', config: {}, isInExperiment: false }
    }
  }

  /**
   * 追踪事件
   */
  async track(
    experimentKey: string,
    eventType: string,
    options?: {
      eventName?: string
      value?: number
      properties?: Record<string, any>
    }
  ): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseURL}/track`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          experiment_key: experimentKey,
          event_type: eventType,
          event_name: options?.eventName,
          value: options?.value,
          properties: options?.properties
        })
      })

      const data = await response.json()
      return data.success || false
    } catch (error) {
      console.error('Event tracking error:', error)
      return false
    }
  }

  /**
   * 转化事件快捷方法
   */
  async conversion(experimentKey: string, value?: number): Promise<boolean> {
    return this.track(experimentKey, 'conversion', { value })
  }

  /**
   * 点击事件快捷方法
   */
  async click(experimentKey: string, element?: string): Promise<boolean> {
    return this.track(experimentKey, 'click', {
      eventName: element,
      properties: { element }
    })
  }

  /**
   * 页面查看事件快捷方法
   */
  async pageview(experimentKey: string, page?: string): Promise<boolean> {
    return this.track(experimentKey, 'pageview', {
      properties: { page: page || window.location.pathname }
    })
  }

  /**
   * 获取或创建会话 ID
   */
  private getOrCreateSessionId(): string {
    let sessionId = localStorage.getItem('ab_session_id')
    if (!sessionId) {
      sessionId = `sess_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
      localStorage.setItem('ab_session_id', sessionId)
    }
    return sessionId
  }
}

// 导出单例
export const abTestClient = new ABTestClient()
```

### React Hook

```tsx
// frontend/src/hooks/useABTest.ts
import { useState, useEffect } from 'react'
import { abTestClient } from '@/lib/ab-testing/ABTestClient'

interface UseABTestOptions {
  autoTrack?: boolean
  trackOnMount?: boolean
}

export function useABTest(
  experimentKey: string,
  options: UseABTestOptions = {}
) {
  const { autoTrack = true, trackOnMount = true } = options

  const [variant, setVariant] = useState<string>('control')
  const [config, setConfig] = useState<Record<string, any>>({})
  const [isInExperiment, setIsInExperiment] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadVariant = async () => {
      setLoading(true)
      const result = await abTestClient.getVariant(experimentKey)
      setVariant(result.variant)
      setConfig(result.config)
      setIsInExperiment(result.isInExperiment)
      setLoading(false)

      // 自动追踪页面查看
      if (autoTrack && trackOnMount && result.isInExperiment) {
        abTestClient.pageview(experimentKey)
      }
    }

    loadVariant()
  }, [experimentKey, autoTrack, trackOnMount])

  const track = async (
    eventType: string,
    options?: {
      eventName?: string
      value?: number
      properties?: Record<string, any>
    }
  ) => {
    if (!isInExperiment) return false
    return abTestClient.track(experimentKey, eventType, options)
  }

  const trackConversion = async (value?: number) => {
    return abTestClient.conversion(experimentKey, value)
  }

  const trackClick = async (element?: string) => {
    return abTestClient.click(experimentKey, element)
  }

  return {
    variant,
    config,
    isInExperiment,
    loading,
    isControl: variant === 'control',
    track,
    trackConversion,
    trackClick
  }
}
```

### 使用示例组件

```tsx
// frontend/src/components/examples/ABTestExample.tsx
import { useABTest } from '@/hooks/useABTest'
import { Button } from '@/components/ui/button'

/**
 * 示例：测试不同按钮文案的点击率
 */
export function ButtonCopyExperiment() {
  const { variant, trackClick, isInExperiment } = useABTest('button_copy_test', {
    autoTrack: true
  })

  // 变体配置
  const buttonTexts: Record<string, string> = {
    control: '立即开始',
    variant_a: '开始冒险',
    variant_b: '创建角色',
    variant_c: '免费开始'
  }

  const handleClick = () => {
    // 执行实际操作
    console.log('Button clicked')

    // 追踪点击（如果启用自动追踪，可以省略）
    trackClick('start_button')
  }

  return (
    <Button onClick={handleClick}>
      {buttonTexts[variant] || buttonTexts.control}
    </Button>
  )
}

/**
 * 示例：测试不同的引导流程
 */
export function OnboardingFlowExperiment() {
  const { variant, trackConversion } = useABTest('onboarding_flow', {
    trackOnMount: true
  })

  const handleComplete = () => {
    // 完成引导后追踪转化
    trackConversion()
  }

  if (variant === 'variant_a') {
    return (
      <div>
        <h1>欢迎来到 CoC 世界</h1>
        <p>让我们创建你的第一个角色...</p>
        <button onClick={handleComplete}>开始创建</button>
      </div>
    )
  } else if (variant === 'variant_b') {
    return (
      <div>
        <h1>开始你的冒险</h1>
        <p>你可以选择预设角色或自定义...</p>
        <button onClick={handleComplete}>继续</button>
      </div>
    )
  }

  // Control
  return (
    <div>
      <h1>欢迎</h1>
      <button onClick={handleComplete}>下一步</button>
    </div>
  )
}

/**
 * 示例：测试不同的功能展示方式
 */
export function FeatureDisplayExperiment() {
  const { variant, config } = useABTest('feature_display')

  return (
    <div className={config.containerClass || ''}>
      {variant === 'cards' ? (
        <div className="grid grid-cols-3 gap-4">
          {/* 卡片布局 */}
        </div>
      ) : variant === 'list' ? (
        <div className="space-y-2">
          {/* 列表布局 */}
        </div>
      ) : (
        <div>
          {/* Control 布局 */}
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
| `backend/models/ab_testing.py` | 创建 | A/B 测试数据模型 |
| `backend/services/ab_testing_service.py` | 创建 | A/B 测试业务逻辑 |
| `backend/api/routes/ab_testing.py` | 创建 | A/B 测试 API 路由 |
| `frontend/src/lib/ab-testing/ABTestClient.ts` | 创建 | A/B 测试客户端 SDK |
| `frontend/src/hooks/useABTest.ts` | 创建 | React Hook |
| `frontend/src/types/ab-testing.ts` | 创建 | 类型定义 |

---

## 验收标准

- [ ] 支持创建和管理多个实验
- [ ] 用户分桶一致性保证
- [ ] 支持多种流量分配策略
- [ ] 事件追踪准确无误
- [ ] 统计分析结果可靠
- [ ] SDK 易用性良好
- [ ] 性能影响可忽略
- [ ] 支持匿名用户

---

## 参考文档

- M1-040: 用户系统
- Google Optimize 文档
- Optimizely 文档
- 统计显著性检验方法

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
