# M5-011: 实现条件触发器编辑器

**任务ID**: M5-011
**标题**: 实现条件触发器编辑器
**类型**: fullstack (全栈开发)
**预估工时**: 8h
**依赖**: M0, M1 完成

---

## 任务描述

实现一个可视化的条件触发器编辑器，允许 KP 配置复杂的游戏逻辑触发条件。例如："当 HP < 10 且 SAN < 20 时触发某个剧情事件"。编辑器需要支持条件组合、运算符选择、动作定义等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-011-01 | 设计触发器数据模型 | 定义条件、动作、触发器结构 | 1h |
| M5-011-02 | 实现后端触发器存储 | CRUD API + 数据库表 | 1.5h |
| M5-011-03 | 实现触发器评估引擎 | 条件判断与执行逻辑 | 2h |
| M5-011-04 | 实现前端触发器编辑器 UI | 可视化条件构建器 | 2h |
| M5-011-05 | 实现触发器测试功能 | 实时预览与调试 | 1h |
| M5-011-06 | 编写单元测试 | 后端评估逻辑测试 | 0.5h |

---

## 完整后端代码示例 (Python + Agno)

### 数据模型定义

```python
# backend/app/models/triggers.py
from datetime import datetime
from typing import Dict, Any, List, Optional
from enum import Enum
from sqlalchemy import Column, String, JSON, DateTime, Boolean, Integer, ForeignKey
from sqlalchemy.dialects.postgresql import UUID
import uuid

from app.db.base_class import Base


class TriggerConditionOperator(str, Enum):
    """条件运算符"""
    EQUAL = "eq"
    NOT_EQUAL = "ne"
    GREATER_THAN = "gt"
    LESS_THAN = "lt"
    GREATER_EQUAL = "gte"
    LESS_EQUAL = "lte"
    CONTAINS = "contains"
    NOT_CONTAINS = "not_contains"
    IN = "in"
    NOT_IN = "not_in"
    AND = "and"
    OR = "or"


class TriggerActionType(str, Enum):
    """触发动作类型"""
    NARRATIVE = "narrative"  # 叙事描述
    SAN_CHECK = "san_check"  # SAN 检定
    DAMAGE = "damage"  # 造成伤害
    HEAL = "heal"  # 治疗
    STATE_CHANGE = "state_change"  # 状态变更
    EVENT_LOG = "event_log"  # 记录事件
    CUSTOM = "custom"  # 自定义动作


class Trigger(Base):
    """触发器表"""
    __tablename__ = "triggers"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    campaign_id = Column(UUID(as_uuid=True), ForeignKey("campaigns.id"), nullable=False)
    name = Column(String(255), nullable=False)
    description = Column(String(1000), nullable=True)

    # 条件配置 (JSON 格式)
    conditions = Column(JSON, nullable=False)

    # 动作配置 (JSON 格式)
    actions = Column(JSON, nullable=False)

    # 是否启用
    is_active = Column(Boolean, default=True)

    # 是否一次性触发
    once_only = Column(Boolean, default=False)

    # 已触发次数
    trigger_count = Column(Integer, default=0)

    # 创建时间
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 触发历史
    triggered_sessions = Column(JSON, default=list)  # 记录已触发的 session_id


class TriggerCondition:
    """条件定义"""

    @staticmethod
    def simple_condition(field: str, operator: TriggerConditionOperator, value: Any) -> Dict:
        """简单条件"""
        return {
            "type": "simple",
            "field": field,  # e.g., "character.hp", "character.san", "scene.id"
            "operator": operator.value,
            "value": value
        }

    @staticmethod
    def composite_condition(operator: TriggerConditionOperator, conditions: List[Dict]) -> Dict:
        """复合条件 (AND/OR)"""
        return {
            "type": "composite",
            "operator": operator.value,  # "and" or "or"
            "conditions": conditions
        }


class TriggerAction:
    """动作定义"""

    @staticmethod
    def narrative_action(text: str, visibility: str = "public") -> Dict:
        """叙事动作"""
        return {
            "type": TriggerActionType.NARRATIVE.value,
            "params": {
                "text": text,
                "visibility": visibility
            }
        }

    @staticmethod
    def san_check_action(loss_success: int, loss_failure: int) -> Dict:
        """SAN 检定动作"""
        return {
            "type": TriggerActionType.SAN_CHECK.value,
            "params": {
                "loss_success": loss_success,
                "loss_failure": loss_failure
            }
        }

    @staticmethod
    def damage_action(target: str, amount: int, damage_type: str = "physical") -> Dict:
        """伤害动作"""
        return {
            "type": TriggerActionType.DAMAGE.value,
            "params": {
                "target": target,  # "current_character" or character_id
                "amount": amount,
                "damage_type": damage_type
            }
        }

    @staticmethod
    def state_change_action(state_key: str, value: Any) -> Dict:
        """状态变更动作"""
        return {
            "type": TriggerActionType.STATE_CHANGE.value,
            "params": {
                "key": state_key,
                "value": value
            }
        }
```

### 触发器评估引擎

```python
# backend/app/services/trigger_service.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from sqlalchemy import and_

from app.models.triggers import Trigger, TriggerConditionOperator, TriggerActionType
from app.models.session import Session
from app.models.character import Character


class TriggerEvaluator:
    """触发器条件评估器"""

    @staticmethod
    def evaluate_condition(
        condition: Dict,
        context: Dict[str, Any],
        session: Session
    ) -> bool:
        """
        评估单个条件

        Args:
            condition: 条件定义
            context: 评估上下文 (包含 character, scene, state 等)
            session: 数据库会话
        """
        condition_type = condition.get("type")

        if condition_type == "simple":
            return TriggerEvaluator._evaluate_simple_condition(
                condition, context, session
            )
        elif condition_type == "composite":
            return TriggerEvaluator._evaluate_composite_condition(
                condition, context, session
            )

        return False

    @staticmethod
    def _evaluate_simple_condition(
        condition: Dict,
        context: Dict[str, Any],
        session: Session
    ) -> bool:
        """评估简单条件"""
        field = condition.get("field")
        operator = condition.get("operator")
        expected_value = condition.get("value")

        # 解析字段路径
        actual_value = TriggerEvaluator._get_field_value(field, context, session)

        if actual_value is None:
            return False

        # 执行比较
        if operator == TriggerConditionOperator.EQUAL.value:
            return actual_value == expected_value
        elif operator == TriggerConditionOperator.NOT_EQUAL.value:
            return actual_value != expected_value
        elif operator == TriggerConditionOperator.GREATER_THAN.value:
            return actual_value > expected_value
        elif operator == TriggerConditionOperator.LESS_THAN.value:
            return actual_value < expected_value
        elif operator == TriggerConditionOperator.GREATER_EQUAL.value:
            return actual_value >= expected_value
        elif operator == TriggerConditionOperator.LESS_EQUAL.value:
            return actual_value <= expected_value
        elif operator == TriggerConditionOperator.CONTAINS.value:
            return expected_value in str(actual_value)
        elif operator == TriggerConditionOperator.IN.value:
            return actual_value in expected_value
        elif operator == TriggerConditionOperator.NOT_IN.value:
            return actual_value not in expected_value

        return False

    @staticmethod
    def _evaluate_composite_condition(
        condition: Dict,
        context: Dict[str, Any],
        session: Session
    ) -> bool:
        """评估复合条件"""
        operator = condition.get("operator")
        sub_conditions = condition.get("conditions", [])

        results = [
            TriggerEvaluator.evaluate_condition(cond, context, session)
            for cond in sub_conditions
        ]

        if operator == TriggerConditionOperator.AND.value:
            return all(results)
        elif operator == TriggerConditionOperator.OR.value:
            return any(results)

        return False

    @staticmethod
    def _get_field_value(
        field: str,
        context: Dict[str, Any],
        session: Session
    ) -> Any:
        """
        获取字段值

        支持的字段格式:
        - "character.hp" -> 当前角色 HP
        - "character.san" -> 当前角色 SAN
        - "scene.id" -> 当前场景 ID
        - "state.turn_count" -> 回合数
        """
        parts = field.split(".")

        if parts[0] == "character":
            # 角色字段
            character: Character = context.get("character")
            if not character:
                return None

            field_name = parts[1]
            if field_name == "hp":
                return character.derived.get("HP")
            elif field_name == "san":
                return character.derived.get("SAN")
            elif field_name == "luck":
                return character.derived.get("Luck")
            elif hasattr(character, field_name):
                return getattr(character, field_name)

        elif parts[0] == "scene":
            # 场景字段
            session_obj: Session = context.get("session")
            if not session_obj:
                return None

            if parts[1] == "id":
                return session_obj.current_scene_id

        elif parts[0] == "state":
            # 状态字段
            session_obj: Session = context.get("session")
            if not session_obj:
                return None

            state = session_obj.state
            if state and parts[1] in state:
                return state[parts[1]]

        return None


class TriggerService:
    """触发器服务"""

    @staticmethod
    def get_campaign_triggers(
        db: Session,
        campaign_id: str,
        active_only: bool = True
    ) -> List[Trigger]:
        """获取 Campaign 的所有触发器"""
        query = db.query(Trigger).filter(Trigger.campaign_id == campaign_id)

        if active_only:
            query = query.filter(Trigger.is_active == True)

        return query.all()

    @staticmethod
    def evaluate_and_execute_triggers(
        db: Session,
        campaign_id: str,
        session_id: str,
        context: Dict[str, Any]
    ) -> List[Dict]:
        """
        评估并执行触发器

        Args:
            db: 数据库会话
            campaign_id: Campaign ID
            session_id: Session ID
            context: 上下文 (包含 character, scene, state 等)

        Returns:
            触发的动作列表
        """
        triggers = TriggerService.get_campaign_triggers(db, campaign_id)
        triggered_actions = []

        for trigger in triggers:
            # 检查是否一次性触发且已触发
            if trigger.once_only and session_id in trigger.triggered_sessions:
                continue

            # 评估条件
            if TriggerEvaluator.evaluate_condition(trigger.conditions, context, db):
                # 执行动作
                actions = TriggerService._execute_actions(
                    trigger.actions,
                    context,
                    db
                )

                triggered_actions.extend(actions)

                # 更新触发器状态
                trigger.trigger_count += 1
                if trigger.once_only:
                    trigger.triggered_sessions.append(session_id)

                db.commit()

        return triggered_actions

    @staticmethod
    def _execute_actions(
        actions: List[Dict],
        context: Dict[str, Any],
        db: Session
    ) -> List[Dict]:
        """执行触发动作"""
        executed = []

        for action in actions:
            action_type = action.get("type")
            params = action.get("params", {})

            if action_type == TriggerActionType.NARRATIVE.value:
                # 叙事动作 - 返回文本供 AI 使用
                executed.append({
                    "type": "narrative",
                    "text": params.get("text"),
                    "visibility": params.get("visibility", "public")
                })

            elif action_type == TriggerActionType.SAN_CHECK.value:
                # SAN 检定动作
                executed.append({
                    "type": "san_check",
                    "loss_success": params.get("loss_success", 0),
                    "loss_failure": params.get("loss_failure", 1)
                })

            elif action_type == TriggerActionType.DAMAGE.value:
                # 伤害动作
                target_id = params.get("target")
                amount = params.get("amount", 0)

                # TODO: 实际应用伤害到角色
                executed.append({
                    "type": "damage",
                    "target": target_id,
                    "amount": amount
                })

            elif action_type == TriggerActionType.STATE_CHANGE.value:
                # 状态变更动作
                state_key = params.get("key")
                state_value = params.get("value")

                # TODO: 更新 session 状态
                executed.append({
                    "type": "state_change",
                    "key": state_key,
                    "value": state_value
                })

        return executed

    @staticmethod
    def create_trigger(
        db: Session,
        campaign_id: str,
        name: str,
        conditions: Dict,
        actions: List[Dict],
        description: Optional[str] = None,
        once_only: bool = False
    ) -> Trigger:
        """创建触发器"""
        trigger = Trigger(
            campaign_id=campaign_id,
            name=name,
            description=description,
            conditions=conditions,
            actions=actions,
            once_only=once_only
        )

        db.add(trigger)
        db.commit()
        db.refresh(trigger)

        return trigger

    @staticmethod
    def update_trigger(
        db: Session,
        trigger_id: str,
        **kwargs
    ) -> Optional[Trigger]:
        """更新触发器"""
        trigger = db.query(Trigger).filter(Trigger.id == trigger_id).first()

        if not trigger:
            return None

        for key, value in kwargs.items():
            if hasattr(trigger, key):
                setattr(trigger, key, value)

        db.commit()
        db.refresh(trigger)

        return trigger

    @staticmethod
    def delete_trigger(db: Session, trigger_id: str) -> bool:
        """删除触发器"""
        trigger = db.query(Trigger).filter(Trigger.id == trigger_id).first()

        if not trigger:
            return False

        db.delete(trigger)
        db.commit()

        return True

    @staticmethod
    def test_trigger(
        db: Session,
        trigger_id: str,
        test_context: Dict[str, Any]
    ) -> Dict:
        """
        测试触发器

        Args:
            db: 数据库会话
            trigger_id: 触发器 ID
            test_context: 测试上下文

        Returns:
            测试结果
        """
        trigger = db.query(Trigger).filter(Trigger.id == trigger_id).first()

        if not trigger:
            return {
                "success": False,
                "error": "Trigger not found"
            }

        # 评估条件
        condition_met = TriggerEvaluator.evaluate_condition(
            trigger.conditions,
            test_context,
            db
        )

        if not condition_met:
            return {
                "success": True,
                "triggered": False,
                "reason": "Conditions not met"
            }

        # 预览动作
        action_preview = TriggerService._execute_actions(
            trigger.actions,
            test_context,
            db
        )

        return {
            "success": True,
            "triggered": True,
            "actions": action_preview
        }
```

### API 路由

```python
# backend/app/api/triggers.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.models.triggers import TriggerCondition, TriggerAction
from app.services.trigger_service import TriggerService
from app.schemas.triggers import (
    TriggerCreate,
    TriggerUpdate,
    TriggerResponse,
    TriggerTestRequest,
    TriggerTestResponse
)

router = APIRouter()


@router.post("/", response_model=TriggerResponse)
def create_trigger(
    trigger_in: TriggerCreate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """创建触发器"""
    trigger = TriggerService.create_trigger(
        db,
        campaign_id=trigger_in.campaign_id,
        name=trigger_in.name,
        conditions=trigger_in.conditions,
        actions=trigger_in.actions,
        description=trigger_in.description,
        once_only=trigger_in.once_only
    )
    return trigger


@router.get("/{trigger_id}", response_model=TriggerResponse)
def get_trigger(
    trigger_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取触发器详情"""
    trigger = db.query(Trigger).filter(Trigger.id == trigger_id).first()

    if not trigger:
        raise HTTPException(status_code=404, detail="Trigger not found")

    return trigger


@router.put("/{trigger_id}", response_model=TriggerResponse)
def update_trigger(
    trigger_id: str,
    trigger_in: TriggerUpdate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """更新触发器"""
    trigger = TriggerService.update_trigger(
        db,
        trigger_id,
        **trigger_in.dict(exclude_unset=True)
    )

    if not trigger:
        raise HTTPException(status_code=404, detail="Trigger not found")

    return trigger


@router.delete("/{trigger_id}")
def delete_trigger(
    trigger_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """删除触发器"""
    success = TriggerService.delete_trigger(db, trigger_id)

    if not success:
        raise HTTPException(status_code=404, detail="Trigger not found")

    return {"message": "Trigger deleted successfully"}


@router.post("/{trigger_id}/test", response_model=TriggerTestResponse)
def test_trigger(
    trigger_id: str,
    test_request: TriggerTestRequest,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """测试触发器"""
    result = TriggerService.test_trigger(
        db,
        trigger_id,
        test_context=test_request.context
    )

    return result


@router.get("/campaign/{campaign_id}", response_model=List[TriggerResponse])
def list_campaign_triggers(
    campaign_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """列出 Campaign 的所有触发器"""
    triggers = TriggerService.get_campaign_triggers(db, campaign_id)
    return triggers
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 类型定义

```typescript
// frontend/src/types/triggers.ts
export enum TriggerConditionOperator {
  EQUAL = "eq",
  NOT_EQUAL = "ne",
  GREATER_THAN = "gt",
  LESS_THAN = "lt",
  GREATER_EQUAL = "gte",
  LESS_EQUAL = "lte",
  CONTAINS = "contains",
  NOT_CONTAINS = "not_contains",
  IN = "in",
  NOT_IN = "not_in",
  AND = "and",
  OR = "or"
}

export enum TriggerActionType {
  NARRATIVE = "narrative",
  SAN_CHECK = "san_check",
  DAMAGE = "damage",
  HEAL = "heal",
  STATE_CHANGE = "state_change",
  EVENT_LOG = "event_log",
  CUSTOM = "custom"
}

export interface TriggerCondition {
  type: "simple" | "composite";
  field?: string;
  operator: string;
  value?: any;
  conditions?: TriggerCondition[];
}

export interface TriggerAction {
  type: TriggerActionType;
  params: Record<string, any>;
}

export interface Trigger {
  id: string;
  campaign_id: string;
  name: string;
  description?: string;
  conditions: TriggerCondition;
  actions: TriggerAction[];
  is_active: boolean;
  once_only: boolean;
  trigger_count: number;
  created_at: string;
  updated_at: string;
}

export interface TriggerTestContext {
  character?: {
    hp: number;
    san: number;
    luck: number;
  };
  scene?: {
    id: string;
  };
  state?: Record<string, any>;
}
```

### 触发器编辑器组件

```tsx
// frontend/src/components/triggers/TriggerEditor.tsx
import React, { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Plus, Trash2, Play, Save } from "lucide-react";

import {
  Trigger,
  TriggerCondition,
  TriggerAction,
  TriggerConditionOperator,
  TriggerActionType
} from "@/types/triggers";
import { ConditionBuilder } from "./ConditionBuilder";
import { ActionBuilder } from "./ActionBuilder";
import { TriggerTestPanel } from "./TriggerTestPanel";

interface TriggerEditorProps {
  trigger?: Trigger;
  campaignId: string;
  onSave: (trigger: Partial<Trigger>) => Promise<void>;
  onCancel: () => void;
}

export function TriggerEditor({
  trigger,
  campaignId,
  onSave,
  onCancel
}: TriggerEditorProps) {
  const [name, setName] = useState(trigger?.name || "");
  const [description, setDescription] = useState(trigger?.description || "");
  const [conditions, setConditions] = useState<TriggerCondition>(
    trigger?.conditions || {
      type: "simple",
      field: "character.hp",
      operator: TriggerConditionOperator.LESS_THAN,
      value: 10
    }
  );
  const [actions, setActions] = useState<TriggerAction[]>(
    trigger?.actions || []
  );
  const [onceOnly, setOnceOnly] = useState(trigger?.once_only || false);
  const [isActive, setIsActive] = useState(trigger?.is_active ?? true);
  const [saving, setSaving] = useState(false);
  const [testContext, setTestContext] = useState<any>({});
  const [testResult, setTestResult] = useState<any>(null);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave({
        id: trigger?.id,
        campaign_id: campaignId,
        name,
        description,
        conditions,
        actions,
        once_only: onceOnly,
        is_active: isActive
      });
    } finally {
      setSaving(false);
    }
  };

  const handleAddAction = useCallback(() => {
    setActions([...actions, {
      type: TriggerActionType.NARRATIVE,
      params: {
        text: "",
        visibility: "public"
      }
    }]);
  }, [actions]);

  const handleUpdateAction = useCallback((index: number, action: TriggerAction) => {
    const newActions = [...actions];
    newActions[index] = action;
    setActions(newActions);
  }, [actions]);

  const handleRemoveAction = useCallback((index: number) => {
    setActions(actions.filter((_, i) => i !== index));
  }, [actions]);

  const handleTest = async () => {
    const result = await fetch(`/api/triggers/${trigger?.id}/test`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ context: testContext })
    });
    const data = await result.json();
    setTestResult(data);
  };

  return (
    <div className="max-w-6xl mx-auto p-6 space-y-6">
      {/* 基本信息卡片 */}
      <Card>
        <CardHeader>
          <CardTitle>触发器配置</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">触发器名称</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="例如: 低血量警告"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Input
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="描述触发器的作用"
              />
            </div>
          </div>

          <div className="flex items-center gap-6">
            <div className="flex items-center space-x-2">
              <Switch
                id="once-only"
                checked={onceOnly}
                onCheckedChange={setOnceOnly}
              />
              <Label htmlFor="once-only">仅触发一次</Label>
            </div>
            <div className="flex items-center space-x-2">
              <Switch
                id="is-active"
                checked={isActive}
                onCheckedChange={setIsActive}
              />
              <Label htmlFor="is-active">启用</Label>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 条件和动作编辑 */}
      <Tabs defaultValue="conditions">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="conditions">触发条件</TabsTrigger>
          <TabsTrigger value="actions">触发动作</TabsTrigger>
          <TabsTrigger value="test">测试</TabsTrigger>
        </TabsList>

        <TabsContent value="conditions">
          <Card>
            <CardHeader>
              <CardTitle>条件配置</CardTitle>
            </CardHeader>
            <CardContent>
              <ConditionBuilder
                condition={conditions}
                onChange={setConditions}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="actions">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>动作配置</CardTitle>
                <Button onClick={handleAddAction} size="sm">
                  <Plus className="w-4 h-4 mr-2" />
                  添加动作
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {actions.map((action, index) => (
                <ActionBuilder
                  key={index}
                  action={action}
                  onChange={(updated) => handleUpdateAction(index, updated)}
                  onRemove={() => handleRemoveAction(index)}
                />
              ))}
              {actions.length === 0 && (
                <div className="text-center text-muted-foreground py-8">
                  暂无动作，点击上方按钮添加
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="test">
          <Card>
            <CardHeader>
              <CardTitle>触发器测试</CardTitle>
            </CardHeader>
            <CardContent>
              <TriggerTestPanel
                testContext={testContext}
                onContextChange={setTestContext}
                onTest={handleTest}
                testResult={testResult}
              />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 操作按钮 */}
      <div className="flex justify-end gap-4">
        <Button variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "保存中..." : "保存触发器"}
        </Button>
      </div>
    </div>
  );
}
```

### 条件构建器组件

```tsx
// frontend/src/components/triggers/ConditionBuilder.tsx
import React from "react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2 } from "lucide-react";

import { TriggerCondition, TriggerConditionOperator } from "@/types/triggers";

interface ConditionBuilderProps {
  condition: TriggerCondition;
  onChange: (condition: TriggerCondition) => void;
}

const FIELD_OPTIONS = [
  { value: "character.hp", label: "角色 HP" },
  { value: "character.san", label: "角色 SAN" },
  { value: "character.luck", label: "角色幸运" },
  { value: "scene.id", label: "场景 ID" },
  { value: "state.turn_count", label: "回合数" }
];

const OPERATOR_OPTIONS = [
  { value: TriggerConditionOperator.EQUAL, label: "等于" },
  { value: TriggerConditionOperator.NOT_EQUAL, label: "不等于" },
  { value: TriggerConditionOperator.GREATER_THAN, label: "大于" },
  { value: TriggerConditionOperator.LESS_THAN, label: "小于" },
  { value: TriggerConditionOperator.GREATER_EQUAL, label: "大于等于" },
  { value: TriggerConditionOperator.LESS_EQUAL, label: "小于等于" },
  { value: TriggerConditionOperator.CONTAINS, label: "包含" },
  { value: TriggerConditionOperator.IN, label: "在列表中" }
];

export function ConditionBuilder({ condition, onChange }: ConditionBuilderProps) {
  const handleTypeChange = (type: "simple" | "composite") => {
    if (type === "simple") {
      onChange({
        type: "simple",
        field: "character.hp",
        operator: TriggerConditionOperator.LESS_THAN,
        value: 10
      });
    } else {
      onChange({
        type: "composite",
        operator: TriggerConditionOperator.AND,
        conditions: []
      });
    }
  };

  const handleAddSubCondition = () => {
    if (condition.type === "composite") {
      onChange({
        ...condition,
        conditions: [
          ...condition.conditions,
          {
            type: "simple",
            field: "character.hp",
            operator: TriggerConditionOperator.LESS_THAN,
            value: 10
          }
        ]
      });
    }
  };

  const handleRemoveSubCondition = (index: number) => {
    if (condition.type === "composite") {
      onChange({
        ...condition,
        conditions: condition.conditions.filter((_, i) => i !== index)
      });
    }
  };

  const handleUpdateSubCondition = (index: number, subCondition: TriggerCondition) => {
    if (condition.type === "composite") {
      const newConditions = [...condition.conditions];
      newConditions[index] = subCondition;
      onChange({
        ...condition,
        conditions: newConditions
      });
    }
  };

  if (condition.type === "simple") {
    return (
      <div className="flex items-center gap-2">
        <Select
          value={condition.field}
          onValueChange={(value) => onChange({ ...condition, field: value })}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="选择字段" />
          </SelectTrigger>
          <SelectContent>
            {FIELD_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={condition.operator}
          onValueChange={(value) => onChange({ ...condition, operator: value })}
        >
          <SelectTrigger className="w-[120px]">
            <SelectValue placeholder="运算符" />
          </SelectTrigger>
          <SelectContent>
            {OPERATOR_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          type="number"
          value={condition.value}
          onChange={(e) => onChange({ ...condition, value: parseFloat(e.target.value) || 0 })}
          className="w-[100px]"
        />

        <Button
          variant="outline"
          size="sm"
          onClick={() => handleTypeChange("composite")}
        >
          转为复合条件
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Badge variant="outline">
          {condition.operator === TriggerConditionOperator.AND ? "AND (且)" : "OR (或)"}
        </Badge>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onChange({
            type: "simple",
            field: "character.hp",
            operator: TriggerConditionOperator.LESS_THAN,
            value: 10
          })}
        >
          转为简单条件
        </Button>
      </div>

      {condition.conditions?.map((subCondition, index) => (
        <Card key={index} className="p-3">
          <div className="flex items-start gap-2">
            <div className="flex-1">
              <ConditionBuilder
                condition={subCondition}
                onChange={(updated) => handleUpdateSubCondition(index, updated)}
              />
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleRemoveSubCondition(index)}
            >
              <Trash2 className="w-4 h-4" />
            </Button>
          </div>
        </Card>
      ))}

      <Button variant="outline" size="sm" onClick={handleAddSubCondition}>
        <Plus className="w-4 h-4 mr-2" />
        添加子条件
      </Button>
    </div>
  );
}
```

### 动作构建器组件

```tsx
// frontend/src/components/triggers/ActionBuilder.tsx
import React from "react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card } from "@/components/ui/card";
import { Trash2 } from "lucide-react";

import { TriggerAction, TriggerActionType } from "@/types/triggers";

interface ActionBuilderProps {
  action: TriggerAction;
  onChange: (action: TriggerAction) => void;
  onRemove: () => void;
}

const ACTION_TYPE_OPTIONS = [
  { value: TriggerActionType.NARRATIVE, label: "叙事描述" },
  { value: TriggerActionType.SAN_CHECK, label: "SAN 检定" },
  { value: TriggerActionType.DAMAGE, label: "造成伤害" },
  { value: TriggerActionType.HEAL, label: "治疗" },
  { value: TriggerActionType.STATE_CHANGE, label: "状态变更" }
];

export function ActionBuilder({ action, onChange, onRemove }: ActionBuilderProps) {
  const handleTypeChange = (type: TriggerActionType) => {
    switch (type) {
      case TriggerActionType.NARRATIVE:
        onChange({
          type,
          params: { text: "", visibility: "public" }
        });
        break;
      case TriggerActionType.SAN_CHECK:
        onChange({
          type,
          params: { loss_success: 0, loss_failure: 1 }
        });
        break;
      case TriggerActionType.DAMAGE:
        onChange({
          type,
          params: { target: "current_character", amount: 5, damage_type: "physical" }
        });
        break;
      case TriggerActionType.HEAL:
        onChange({
          type,
          params: { target: "current_character", amount: 5 }
        });
        break;
      case TriggerActionType.STATE_CHANGE:
        onChange({
          type,
          params: { key: "", value: "" }
        });
        break;
    }
  };

  const handleParamChange = (key: string, value: any) => {
    onChange({
      ...action,
      params: {
        ...action.params,
        [key]: value
      }
    });
  };

  return (
    <Card className="p-4">
      <div className="flex items-start gap-4">
        <div className="flex-1 space-y-4">
          {/* 动作类型选择 */}
          <div className="space-y-2">
            <Label>动作类型</Label>
            <Select
              value={action.type}
              onValueChange={(value) => handleTypeChange(value as TriggerActionType)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ACTION_TYPE_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* 根据类型显示不同的参数配置 */}
          {action.type === TriggerActionType.NARRATIVE && (
            <div className="space-y-2">
              <Label>叙事文本</Label>
              <Textarea
                value={action.params.text || ""}
                onChange={(e) => handleParamChange("text", e.target.value)}
                placeholder="输入要显示的叙事文本..."
                rows={3}
              />
              <Label>可见性</Label>
              <Select
                value={action.params.visibility || "public"}
                onValueChange={(value) => handleParamChange("visibility", value)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="public">公开</SelectItem>
                  <SelectItem value="kp">仅 KP</SelectItem>
                  <SelectItem value="private">私密</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {action.type === TriggerActionType.SAN_CHECK && (
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>成功损失</Label>
                <Input
                  type="number"
                  value={action.params.loss_success || 0}
                  onChange={(e) => handleParamChange("loss_success", parseInt(e.target.value) || 0)}
                />
              </div>
              <div className="space-y-2">
                <Label>失败损失</Label>
                <Input
                  type="number"
                  value={action.params.loss_failure || 1}
                  onChange={(e) => handleParamChange("loss_failure", parseInt(e.target.value) || 0)}
                />
              </div>
            </div>
          )}

          {action.type === TriggerActionType.DAMAGE && (
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>伤害量</Label>
                <Input
                  type="number"
                  value={action.params.amount || 0}
                  onChange={(e) => handleParamChange("amount", parseInt(e.target.value) || 0)}
                />
              </div>
              <div className="space-y-2">
                <Label>伤害类型</Label>
                <Select
                  value={action.params.damage_type || "physical"}
                  onValueChange={(value) => handleParamChange("damage_type", value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="physical">物理</SelectItem>
                    <SelectItem value="san">精神</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}

          {action.type === TriggerActionType.HEAL && (
            <div className="space-y-2">
              <Label>治疗量</Label>
              <Input
                type="number"
                value={action.params.amount || 0}
                onChange={(e) => handleParamChange("amount", parseInt(e.target.value) || 0)}
              />
            </div>
          )}

          {action.type === TriggerActionType.STATE_CHANGE && (
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>状态键</Label>
                <Input
                  value={action.params.key || ""}
                  onChange={(e) => handleParamChange("key", e.target.value)}
                  placeholder="例如: current_scene"
                />
              </div>
              <div className="space-y-2">
                <Label>状态值</Label>
                <Input
                  value={action.params.value || ""}
                  onChange={(e) => handleParamChange("value", e.target.value)}
                  placeholder="新值"
                />
              </div>
            </div>
          )}
        </div>

        <Button variant="ghost" size="sm" onClick={onRemove}>
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
    </Card>
  );
}
```

### 触发器测试面板组件

```tsx
// frontend/src/components/triggers/TriggerTestPanel.tsx
import React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Play, CheckCircle, XCircle } from "lucide-react";

interface TriggerTestPanelProps {
  testContext: any;
  onContextChange: (context: any) => void;
  onTest: () => void;
  testResult: any;
}

export function TriggerTestPanel({
  testContext,
  onContextChange,
  onTest,
  testResult
}: TriggerTestPanelProps) {
  return (
    <div className="space-y-6">
      {/* 测试上下文输入 */}
      <Card>
        <CardContent className="pt-6 space-y-4">
          <h3 className="font-semibold">测试上下文</h3>

          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>角色 HP</Label>
              <Input
                type="number"
                value={testContext.character?.hp || 15}
                onChange={(e) => onContextChange({
                  ...testContext,
                  character: {
                    ...testContext.character,
                    hp: parseInt(e.target.value) || 15
                  }
                })}
              />
            </div>
            <div className="space-y-2">
              <Label>角色 SAN</Label>
              <Input
                type="number"
                value={testContext.character?.san || 60}
                onChange={(e) => onContextChange({
                  ...testContext,
                  character: {
                    ...testContext.character,
                    san: parseInt(e.target.value) || 60
                  }
                })}
              />
            </div>
            <div className="space-y-2">
              <Label>角色幸运</Label>
              <Input
                type="number"
                value={testContext.character?.luck || 50}
                onChange={(e) => onContextChange({
                  ...testContext,
                  character: {
                    ...testContext.character,
                    luck: parseInt(e.target.value) || 50
                  }
                })}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>场景 ID</Label>
            <Input
              value={testContext.scene?.id || ""}
              onChange={(e) => onContextChange({
                ...testContext,
                scene: {
                  id: e.target.value
                }
              })}
              placeholder="输入场景 ID"
            />
          </div>

          <Button onClick={onTest} className="w-full">
            <Play className="w-4 h-4 mr-2" />
            运行测试
          </Button>
        </CardContent>
      </Card>

      {/* 测试结果 */}
      {testResult && (
        <Card>
          <CardContent className="pt-6 space-y-4">
            <div className="flex items-center gap-2">
              {testResult.triggered ? (
                <CheckCircle className="w-5 h-5 text-green-500" />
              ) : (
                <XCircle className="w-5 h-5 text-red-500" />
              )}
              <h3 className="font-semibold">
                {testResult.triggered ? "触发器会被执行" : "触发器不会被执行"}
              </h3>
            </div>

            {!testResult.triggered && testResult.reason && (
              <p className="text-sm text-muted-foreground">{testResult.reason}</p>
            )}

            {testResult.triggered && testResult.actions && (
              <div className="space-y-2">
                <Label>将执行的动作:</Label>
                {testResult.actions.map((action: any, index: number) => (
                  <Badge key={index} variant="outline">
                    {action.type}: {JSON.stringify(action.params || {})}
                  </Badge>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/models/triggers.py` | 创建 | 触发器数据模型 |
| `backend/app/services/trigger_service.py` | 创建 | 触发器服务与评估引擎 |
| `backend/app/api/triggers.py` | 创建 | 触发器 API 路由 |
| `backend/app/schemas/triggers.py` | 创建 | Pydantic 模型 |
| `backend/app/db/migrations/versions/xxx_create_triggers.py` | 创建 | 数据库迁移 |
| `frontend/src/types/triggers.ts` | 创建 | TypeScript 类型定义 |
| `frontend/src/components/triggers/TriggerEditor.tsx` | 创建 | 触发器编辑器主组件 |
| `frontend/src/components/triggers/ConditionBuilder.tsx` | 创建 | 条件构建器组件 |
| `frontend/src/components/triggers/ActionBuilder.tsx` | 创建 | 动作构建器组件 |
| `frontend/src/components/triggers/TriggerTestPanel.tsx` | 创建 | 测试面板组件 |
| `frontend/src/pages/CampaignTriggers.tsx` | 创建 | 触发器列表页面 |

---

## 验收标准

- [ ] KP 可以创建条件触发器
- [ ] 支持简单条件 (字段 + 运算符 + 值)
- [ ] 支持复合条件 (AND/OR 组合)
- [ ] 支持多种触发动作类型
- [ ] 触发器可以实时测试并预览结果
- [ ] 触发器在游戏中正确评估和执行
- [ ] 一次性触发器只触发一次
- [ ] 触发器历史可追溯

---

## 参考文档

- CoC 7e 规则书 - 特殊规则章节
- 游戏事件系统设计最佳实践
- React 表单管理指南
- shadcn/ui 组件库文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
