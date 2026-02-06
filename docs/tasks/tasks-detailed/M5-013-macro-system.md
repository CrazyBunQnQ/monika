# M5-013: 实现宏命令系统

**任务ID**: M5-013
**标题**: 实现宏命令系统
**类型**: fullstack (全栈开发)
**预估工时**: 10h
**依赖**: M0 完成

---

## 任务描述

实现一个宏命令系统，允许玩家和 KP 定义快捷命令别名和复杂操作序列。例如：定义 `/attack` 为一系列战斗操作的组合，或定义 `/search` 自动执行搜索检定并展示结果。宏命令应支持参数传递、条件执行、嵌套调用等高级功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-013-01 | 设计宏命令数据结构 | 变量、参数、动作序列 | 1.5h |
| M5-013-02 | 实现宏命令解析器 | 变量替换、参数解析 | 2.5h |
| M5-013-03 | 实现宏命令执行引擎 | 动作序列执行 | 2h |
| M5-013-04 | 实现后端宏命令管理 API | CRUD + 权限控制 | 1.5h |
| M5-013-05 | 实现前端宏命令编辑器 | 可视化编辑界面 | 2h |
| M5-013-06 | 编写宏命令示例文档 | 常用场景示例 | 0.5h |

---

## 完整后端代码示例 (Python + Agno)

### 宏命令数据模型

```python
# backend/app/models/macros.py
from datetime import datetime
from typing import Dict, Any, List, Optional
from enum import Enum
from sqlalchemy import Column, String, JSON, DateTime, Boolean, Integer, ForeignKey, Text
from sqlalchemy.dialects.postgresql import UUID
import uuid

from app.db.base_class import Base


class MacroScope(str, Enum):
    """宏命令作用域"""
    GLOBAL = "global"  # 全局（管理员）
    CAMPAIGN = "campaign"  # Campaign 级别（KP）
    CHARACTER = "character"  # 角色级别（玩家）
    PERSONAL = "personal"  # 个人级别


class MacroActionType(str, Enum):
    """宏命令动作类型"""
    COMMAND = "command"  # 执行命令
    MESSAGE = "message"  # 发送消息
    DELAY = "delay"  # 延迟
    CONDITION = "condition"  # 条件分支
    LOOP = "loop"  # 循环
    SET_VAR = "set_var"  # 设置变量
    ROLL = "roll"  # 掷骰


class Macro(Base):
    """宏命令表"""
    __tablename__ = "macros"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)

    # 基本信息
    name = Column(String(100), nullable=False, unique=True)  # 宏命令名称（不带斜杠）
    description = Column(Text, nullable=True)
    aliases = Column(JSON, default=list)  # 别名列表

    # 作用域
    scope = Column(String(50), nullable=False, default=MacroScope.PERSONAL)

    # 归属
    campaign_id = Column(UUID(as_uuid=True), ForeignKey("campaigns.id"), nullable=True)
    character_id = Column(UUID(as_uuid=True), ForeignKey("characters.id"), nullable=True)
    account_id = Column(UUID(as_uuid=True), ForeignKey("accounts.id"), nullable=True)

    # 参数定义
    parameters = Column(JSON, default=list)  # 参数定义列表

    # 变量定义
    variables = Column(JSON, default=dict)  # 默认变量

    # 动作序列
    actions = Column(JSON, nullable=False)  # 动作列表

    # 是否启用
    is_active = Column(Boolean, default=True)

    # 使用统计
    usage_count = Column(Integer, default=0)

    # 创建信息
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    created_by = Column(UUID(as_uuid=True), ForeignKey("accounts.id"))


class MacroExecution:
    """宏命令执行器"""

    @staticmethod
    def parse_parameters(
        command_text: str,
        macro: Macro
    ) -> Dict[str, Any]:
        """
        解析宏命令参数

        Args:
            command_text: 完整命令文本 (例如: /attack target=goblin bonus=2)
            macro: 宏命令定义

        Returns:
            参数字典
        """
        import re

        # 移除宏命令名称
        parts = command_text[len(macro.name) + 1:].strip().split()

        params = {}

        # 解析键值对参数
        for part in parts:
            if "=" in part:
                key, value = part.split("=", 1)
                params[key.strip()] = value.strip()
            else:
                # 位置参数
                params[part] = True

        return params

    @staticmethod
    def resolve_variables(
        text: str,
        variables: Dict[str, Any],
        parameters: Dict[str, Any]
    ) -> str:
        """
        解析变量替换

        支持的变量格式:
        - ${var_name} - 变量
        - ${param_name} - 参数
        - ${roll.d20} - 掷骰结果
        """
        import re

        def replace_var(match):
            var_name = match.group(1)
            # 优先使用参数
            if var_name in parameters:
                return str(parameters[var_name])
            # 其次使用变量
            if var_name in variables:
                return str(variables[var_name])
            # 处理嵌套表达式
            if "." in var_name:
                parts = var_name.split(".")
                if parts[0] == "roll" and len(parts) == 2:
                    # 掷骰表达式 ${roll.d20}
                    from app.services.game_service import DiceService
                    return str(DiceService.roll_expression(parts[1]))
            return match.group(0)

        return re.sub(r"\$\{([^}]+)\}", replace_var, text)

    @staticmethod
    def execute_action(
        action: Dict,
        context: Dict[str, Any],
        variables: Dict[str, Any],
        parameters: Dict[str, Any]
    ) -> Any:
        """
        执行单个动作

        Args:
            action: 动作定义
            context: 执行上下文
            variables: 当前变量
            parameters: 宏参数

        Returns:
            动作结果
        """
        action_type = action.get("type")
        params = action.get("params", {})

        # 解析参数中的变量
        resolved_params = {}
        for key, value in params.items():
            if isinstance(value, str):
                resolved_params[key] = MacroExecution.resolve_variables(
                    value,
                    variables,
                    parameters
                )
            else:
                resolved_params[key] = value

        if action_type == MacroActionType.COMMAND:
            # 执行命令
            from app.services.command_service import CommandService
            command = resolved_params.get("command")
            return CommandService.execute(command, context)

        elif action_type == MacroActionType.MESSAGE:
            # 发送消息
            message = resolved_params.get("message")
            visibility = resolved_params.get("visibility", "public")
            # TODO: 发送消息到游戏台
            return {"type": "message", "content": message, "visibility": visibility}

        elif action_type == MacroActionType.DELAY:
            # 延迟执行
            import asyncio
            delay_ms = resolved_params.get("delay_ms", 0)
            if delay_ms > 0:
                import asyncio
                await asyncio.sleep(delay_ms / 1000)
            return None

        elif action_type == MacroActionType.SET_VAR:
            # 设置变量
            var_name = resolved_params.get("name")
            var_value = resolved_params.get("value")
            variables[var_name] = var_value
            return {"type": "set_var", "name": var_name, "value": var_value}

        elif action_type == MacroActionType.ROLL:
            # 掷骰
            from app.services.game_service import DiceService
            expression = resolved_params.get("expression", "1d20")
            result = DiceService.roll_expression(expression)
            return {"type": "roll", "expression": expression, "result": result}

        elif action_type == MacroActionType.CONDITION:
            # 条件分支
            condition = resolved_params.get("condition")
            if_actions = resolved_params.get("if_actions", [])
            else_actions = resolved_params.get("else_actions", [])

            # 评估条件
            if MacroExecution._evaluate_condition(condition, variables, parameters):
                results = []
                for act in if_actions:
                    result = MacroExecution.execute_action(act, context, variables, parameters)
                    results.append(result)
                return {"type": "condition", "result": True, "actions": results}
            else:
                results = []
                for act in else_actions:
                    result = MacroExecution.execute_action(act, context, variables, parameters)
                    results.append(result)
                return {"type": "condition", "result": False, "actions": results}

        elif action_type == MacroActionType.LOOP:
            # 循环
            loop_var = resolved_params.get("variable", "i")
            loop_count = int(resolved_params.get("count", 1))
            loop_actions = resolved_params.get("actions", [])

            results = []
            for i in range(loop_count):
                variables[loop_var] = i + 1
                for act in loop_actions:
                    result = MacroExecution.execute_action(act, context, variables, parameters)
                    results.append(result)

            return {"type": "loop", "iterations": loop_count, "results": results}

        return None

    @staticmethod
    def _evaluate_condition(
        condition: str,
        variables: Dict[str, Any],
        parameters: Dict[str, Any]
    ) -> bool:
        """评估条件表达式"""
        # 简单实现：支持 ==, !=, >, <, >=, <=
        import re

        # 解析条件
        pattern = r"(\w+)\s*(==|!=|>=|<=|>|<)\s*(\w+)"
        match = re.match(pattern, condition.strip())

        if not match:
            return False

        var_name = match.group(1)
        operator = match.group(2)
        value = match.group(3)

        # 获取变量值
        if var_name in parameters:
            left = parameters[var_name]
        elif var_name in variables:
            left = variables[var_name]
        else:
            return False

        # 尝试转换为数字
        try:
            left = float(left)
            value = float(value)
        except (ValueError, TypeError):
            pass

        # 执行比较
        if operator == "==":
            return str(left) == str(value)
        elif operator == "!=":
            return str(left) != str(value)
        elif operator == ">":
            return left > value
        elif operator == "<":
            return left < value
        elif operator == ">=":
            return left >= value
        elif operator == "<=":
            return left <= value

        return False

    @staticmethod
    async def execute_macro(
        macro: Macro,
        command_text: str,
        context: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        执行宏命令

        Args:
            macro: 宏命令定义
            command_text: 完整命令文本
            context: 执行上下文

        Returns:
            执行结果
        """
        # 解析参数
        parameters = MacroExecution.parse_parameters(command_text, macro)

        # 初始化变量（包含默认变量和宏变量）
        variables = {**macro.variables}

        # 执行动作序列
        results = []
        for action in macro.actions:
            result = MacroExecution.execute_action(
                action,
                context,
                variables,
                parameters
            )
            results.append(result)

        # 更新使用统计
        macro.usage_count += 1

        return {
            "macro_id": str(macro.id),
            "macro_name": macro.name,
            "parameters": parameters,
            "variables": variables,
            "results": results
        }


class MacroService:
    """宏命令服务"""

    @staticmethod
    def create_macro(
        db,
        name: str,
        actions: List[Dict],
        scope: MacroScope = MacroScope.PERSONAL,
        description: Optional[str] = None,
        aliases: Optional[List[str]] = None,
        parameters: Optional[List[Dict]] = None,
        variables: Optional[Dict] = None,
        campaign_id: Optional[str] = None,
        character_id: Optional[str] = None,
        account_id: Optional[str] = None
    ) -> Macro:
        """创建宏命令"""
        macro = Macro(
            name=name.lstrip("/"),  # 移除开头的斜杠
            description=description,
            aliases=aliases or [],
            scope=scope,
            parameters=parameters or [],
            variables=variables or {},
            actions=actions,
            campaign_id=campaign_id,
            character_id=character_id,
            account_id=account_id
        )

        db.add(macro)
        db.commit()
        db.refresh(macro)

        return macro

    @staticmethod
    def get_available_macros(
        db,
        campaign_id: Optional[str] = None,
        character_id: Optional[str] = None,
        account_id: Optional[str] = None
    ) -> List[Macro]:
        """获取可用的宏命令"""
        query = db.query(Macro).filter(Macro.is_active == True)

        # 获取全局宏
        macros = query.filter(Macro.scope == MacroScope.GLOBAL).all()

        # 获取 Campaign 宏
        if campaign_id:
            campaign_macros = query.filter(
                Macro.scope == MacroScope.CAMPAIGN,
                Macro.campaign_id == campaign_id
            ).all()
            macros.extend(campaign_macros)

        # 获取角色宏
        if character_id:
            character_macros = query.filter(
                Macro.scope == MacroScope.CHARACTER,
                Macro.character_id == character_id
            ).all()
            macros.extend(character_macros)

        # 获取个人宏
        if account_id:
            personal_macros = query.filter(
                Macro.scope == MacroScope.PERSONAL,
                Macro.account_id == account_id
            ).all()
            macros.extend(personal_macros)

        return macros

    @staticmethod
    def find_macro_by_name(
        db,
        name: str,
        campaign_id: Optional[str] = None,
        character_id: Optional[str] = None,
        account_id: Optional[str] = None
    ) -> Optional[Macro]:
        """根据名称查找宏命令"""
        macros = MacroService.get_available_macros(
            db,
            campaign_id,
            character_id,
            account_id
        )

        # 精确匹配
        for macro in macros:
            if macro.name == name.lstrip("/"):
                return macro

            # 检查别名
            if name.lstrip("/") in macro.aliases:
                return macro

        return None

    @staticmethod
    def update_macro(
        db,
        macro_id: str,
        **kwargs
    ) -> Optional[Macro]:
        """更新宏命令"""
        macro = db.query(Macro).filter(Macro.id == macro_id).first()

        if not macro:
            return None

        for key, value in kwargs.items():
            if hasattr(macro, key):
                setattr(macro, key, value)

        db.commit()
        db.refresh(macro)

        return macro

    @staticmethod
    def delete_macro(db, macro_id: str) -> bool:
        """删除宏命令"""
        macro = db.query(Macro).filter(Macro.id == macro_id).first()

        if not macro:
            return False

        db.delete(macro)
        db.commit()

        return True
```

### API 路由

```python
# backend/app/api/macros.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.schemas.macros import (
    MacroCreate,
    MacroUpdate,
    MacroResponse,
    MacroExecuteRequest,
    MacroExecuteResponse
)
from app.services.macro_service import MacroService, MacroExecution

router = APIRouter()


@router.post("/", response_model=MacroResponse)
def create_macro(
    macro_in: MacroCreate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """创建宏命令"""
    macro = MacroService.create_macro(
        db,
        **macro_in.dict(),
        account_id=current_user.id
    )
    return macro


@router.get("/", response_model=List[MacroResponse])
def list_macros(
    campaign_id: str = None,
    character_id: str = None,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """列出可用的宏命令"""
    return MacroService.get_available_macros(
        db,
        campaign_id,
        character_id,
        current_user.id
    )


@router.get("/{macro_id}", response_model=MacroResponse)
def get_macro(
    macro_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取宏命令详情"""
    macro = db.query(Macro).filter(Macro.id == macro_id).first()

    if not macro:
        raise HTTPException(status_code=404, detail="Macro not found")

    return macro


@router.put("/{macro_id}", response_model=MacroResponse)
def update_macro(
    macro_id: str,
    macro_in: MacroUpdate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """更新宏命令"""
    macro = MacroService.update_macro(
        db,
        macro_id,
        **macro_in.dict(exclude_unset=True)
    )

    if not macro:
        raise HTTPException(status_code=404, detail="Macro not found")

    return macro


@router.delete("/{macro_id}")
def delete_macro(
    macro_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """删除宏命令"""
    success = MacroService.delete_macro(db, macro_id)

    if not success:
        raise HTTPException(status_code=404, detail="Macro not found")

    return {"message": "Macro deleted successfully"}


@router.post("/execute", response_model=MacroExecuteResponse)
async def execute_macro(
    request: MacroExecuteRequest,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """执行宏命令"""
    # 查找宏命令
    macro = MacroService.find_macro_by_name(
        db,
        request.command,
        request.campaign_id,
        request.character_id,
        current_user.id
    )

    if not macro:
        raise HTTPException(status_code=404, detail="Macro not found")

    # 执行宏命令
    result = await MacroExecution.execute_macro(
        macro,
        request.command,
        request.context
    )

    return result
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 类型定义

```typescript
// frontend/src/types/macros.ts
export enum MacroScope {
  GLOBAL = "global",
  CAMPAIGN = "campaign",
  CHARACTER = "character",
  PERSONAL = "personal"
}

export enum MacroActionType {
  COMMAND = "command",
  MESSAGE = "message",
  DELAY = "delay",
  CONDITION = "condition",
  LOOP = "loop",
  SET_VAR = "set_var",
  ROLL = "roll"
}

export interface MacroParameter {
  name: string;
  type: "string" | "number" | "boolean";
  required: boolean;
  default?: any;
  description?: string;
}

export interface MacroAction {
  type: MacroActionType;
  params: Record<string, any>;
}

export interface Macro {
  id: string;
  name: string;
  description?: string;
  aliases: string[];
  scope: MacroScope;
  parameters: MacroParameter[];
  variables: Record<string, any>;
  actions: MacroAction[];
  is_active: boolean;
  usage_count: number;
  created_at: string;
  updated_at: string;
}

export interface MacroExecutionContext {
  session_id: string;
  campaign_id?: string;
  character_id?: string;
}
```

### 宏命令编辑器组件

```tsx
// frontend/src/components/macros/MacroEditor.tsx
import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2, Save, Play } from "lucide-react";

import { Macro, MacroAction, MacroActionType, MacroParameter } from "@/types/macros";
import { ActionBuilder } from "./ActionBuilder";
import { ParameterEditor } from "./ParameterEditor";

interface MacroEditorProps {
  macro?: Macro;
  onSave: (macro: Partial<Macro>) => Promise<void>;
  onTest?: (macro: Partial<Macro>) => Promise<void>;
  onCancel: () => void;
}

export function MacroEditor({ macro, onSave, onTest, onCancel }: MacroEditorProps) {
  const [name, setName] = useState(macro?.name || "");
  const [description, setDescription] = useState(macro?.description || "");
  const [aliases, setAliases] = useState<string[]>(macro?.aliases || []);
  const [aliasInput, setAliasInput] = useState("");
  const [parameters, setParameters] = useState<MacroParameter[]>(macro?.parameters || []);
  const [variables, setVariables] = useState<Record<string, any>>(macro?.variables || {});
  const [actions, setActions] = useState<MacroAction[]>(macro?.actions || []);
  const [isActive, setIsActive] = useState(macro?.is_active ?? true);
  const [saving, setSaving] = useState(false);

  const handleAddAlias = () => {
    if (aliasInput.trim() && !aliases.includes(aliasInput.trim())) {
      setAliases([...aliases, aliasInput.trim()]);
      setAliasInput("");
    }
  };

  const handleRemoveAlias = (alias: string) => {
    setAliases(aliases.filter(a => a !== alias));
  };

  const handleAddParameter = () => {
    setParameters([...parameters, {
      name: "",
      type: "string",
      required: false
    }]);
  };

  const handleUpdateParameter = (index: number, param: MacroParameter) => {
    const newParams = [...parameters];
    newParams[index] = param;
    setParameters(newParams);
  };

  const handleRemoveParameter = (index: number) => {
    setParameters(parameters.filter((_, i) => i !== index));
  };

  const handleAddAction = () => {
    setActions([...actions, {
      type: MacroActionType.MESSAGE,
      params: { message: "" }
    }]);
  };

  const handleUpdateAction = (index: number, action: MacroAction) => {
    const newActions = [...actions];
    newActions[index] = action;
    setActions(newActions);
  };

  const handleRemoveAction = (index: number) => {
    setActions(actions.filter((_, i) => i !== index));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave({
        id: macro?.id,
        name: name.replace(/^\//, ""), // 移除开头的斜杠
        description,
        aliases,
        parameters,
        variables,
        actions,
        is_active: isActive
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      {/* 基本信息 */}
      <Card>
        <CardHeader>
          <CardTitle>基本信息</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">宏命令名称</Label>
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">/</span>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="attack"
                />
              </div>
              <p className="text-xs text-muted-foreground">
                输入时: /{name || "command"} 参数1=value1 参数2=value2
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Input
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="描述这个宏命令的作用"
              />
            </div>
          </div>

          {/* 别名 */}
          <div className="space-y-2">
            <Label>别名</Label>
            <div className="flex gap-2">
              <Input
                value={aliasInput}
                onChange={(e) => setAliasInput(e.target.value)}
                onKeyPress={(e) => e.key === "Enter" && handleAddAlias()}
                placeholder="添加别名"
              />
              <Button onClick={handleAddAlias} variant="outline" size="sm">
                <Plus className="w-4 h-4" />
              </Button>
            </div>
            {aliases.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-2">
                {aliases.map((alias) => (
                  <Badge key={alias} variant="secondary">
                    /{alias}
                    <button
                      onClick={() => handleRemoveAlias(alias)}
                      className="ml-1 hover:text-destructive"
                    >
                      ×
                    </button>
                  </Badge>
                ))}
              </div>
            )}
          </div>

          {/* 启用开关 */}
          <div className="flex items-center space-x-2">
            <Switch
              id="is-active"
              checked={isActive}
              onCheckedChange={setIsActive}
            />
            <Label htmlFor="is-active">启用此宏命令</Label>
          </div>
        </CardContent>
      </Card>

      {/* 参数和动作 */}
      <Tabs defaultValue="parameters">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="parameters">参数</TabsTrigger>
          <TabsTrigger value="variables">变量</TabsTrigger>
          <TabsTrigger value="actions">动作序列</TabsTrigger>
        </TabsList>

        <TabsContent value="parameters">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>参数定义</CardTitle>
                <Button onClick={handleAddParameter} size="sm">
                  <Plus className="w-4 h-4 mr-2" />
                  添加参数
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {parameters.map((param, index) => (
                <ParameterEditor
                  key={index}
                  parameter={param}
                  onChange={(p) => handleUpdateParameter(index, p)}
                  onRemove={() => handleRemoveParameter(index)}
                />
              ))}
              {parameters.length === 0 && (
                <div className="text-center text-muted-foreground py-8">
                  暂无参数，点击上方按钮添加
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="variables">
          <Card>
            <CardHeader>
              <CardTitle>默认变量</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={JSON.stringify(variables, null, 2)}
                onChange={(e) => {
                  try {
                    setVariables(JSON.parse(e.target.value));
                  } catch {
                    // 忽略无效 JSON
                  }
                }}
                placeholder='{"key": "value"}'
                rows={10}
                className="font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground mt-2">
                使用 JSON 格式定义默认变量，可在动作中通过 ${variable_name} 引用
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="actions">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>动作序列</CardTitle>
                <Button onClick={handleAddAction} size="sm">
                  <Plus className="w-4 h-4 mr-2" />
                  添加动作
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {actions.map((action, index) => (
                <div key={index} className="relative">
                  <div className="absolute left-2 top-0 bottom-0 w-0.5 bg-border" />
                  <div className="pl-6">
                    <ActionBuilder
                      action={action}
                      onChange={(a) => handleUpdateAction(index, a)}
                      onRemove={() => handleRemoveAction(index)}
                    />
                  </div>
                </div>
              ))}
              {actions.length === 0 && (
                <div className="text-center text-muted-foreground py-8">
                  暂无动作，点击上方按钮添加
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 操作按钮 */}
      <div className="flex justify-end gap-4">
        <Button variant="outline" onClick={onCancel}>
          取消
        </Button>
        {onTest && (
          <Button variant="outline" onClick={() => onTest({
            name: name.replace(/^\//, ""),
            description,
            aliases,
            parameters,
            variables,
            actions
          })}>
            <Play className="w-4 h-4 mr-2" />
            测试
          </Button>
        )}
        <Button onClick={handleSave} disabled={saving}>
          <Save className="w-4 h-4 mr-2" />
          {saving ? "保存中..." : "保存宏命令"}
        </Button>
      </div>
    </div>
  );
}
```

### 动作构建器组件

```tsx
// frontend/src/components/macros/ActionBuilder.tsx
import React from "react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card } from "@/components/ui/card";
import { Trash2 } from "lucide-react";

import { MacroAction, MacroActionType } from "@/types/macros";

interface ActionBuilderProps {
  action: MacroAction;
  onChange: (action: MacroAction) => void;
  onRemove: () => void;
}

const ACTION_TYPE_OPTIONS = [
  { value: MacroActionType.COMMAND, label: "执行命令" },
  { value: MacroActionType.MESSAGE, label: "发送消息" },
  { value: MacroActionType.SET_VAR, label: "设置变量" },
  { value: MacroActionType.ROLL, label: "掷骰" },
  { value: MacroActionType.CONDITION, label: "条件分支" },
  { value: MacroActionType.LOOP, label: "循环" }
];

export function ActionBuilder({ action, onChange, onRemove }: ActionBuilderProps) {
  const handleTypeChange = (type: MacroActionType) => {
    switch (type) {
      case MacroActionType.COMMAND:
        onChange({ type, params: { command: "" } });
        break;
      case MacroActionType.MESSAGE:
        onChange({ type, params: { message: "", visibility: "public" } });
        break;
      case MacroActionType.SET_VAR:
        onChange({ type, params: { name: "", value: "" } });
        break;
      case MacroActionType.ROLL:
        onChange({ type, params: { expression: "1d20" } });
        break;
      case MacroActionType.CONDITION:
        onChange({ type, params: { condition: "", if_actions: [], else_actions: [] } });
        break;
      case MacroActionType.LOOP:
        onChange({ type, params: { variable: "i", count: 1, actions: [] } });
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
          {/* 动作类型 */}
          <div className="space-y-2">
            <Label>动作类型</Label>
            <Select value={action.type} onValueChange={(v) => handleTypeChange(v as MacroActionType)}>
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
          {action.type === MacroActionType.COMMAND && (
            <div className="space-y-2">
              <Label>命令</Label>
              <Input
                value={action.params.command || ""}
                onChange={(e) => handleParamChange("command", e.target.value)}
                placeholder="/roll d20"
              />
              <p className="text-xs text-muted-foreground">
                可使用变量: ${"${variable}"} 或 ${"${param}"}
              </p>
            </div>
          )}

          {action.type === MacroActionType.MESSAGE && (
            <div className="space-y-2">
              <Label>消息内容</Label>
              <Textarea
                value={action.params.message || ""}
                onChange={(e) => handleParamChange("message", e.target.value)}
                placeholder="要显示的消息"
                rows={3}
              />
              <Label>可见性</Label>
              <Select
                value={action.params.visibility || "public"}
                onValueChange={(v) => handleParamChange("visibility", v)}
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

          {action.type === MacroActionType.SET_VAR && (
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>变量名</Label>
                <Input
                  value={action.params.name || ""}
                  onChange={(e) => handleParamChange("name", e.target.value)}
                  placeholder="result"
                />
              </div>
              <div className="space-y-2">
                <Label>变量值</Label>
                <Input
                  value={action.params.value || ""}
                  onChange={(e) => handleParamChange("value", e.target.value)}
                  placeholder="${roll.d20}"
                />
              </div>
            </div>
          )}

          {action.type === MacroActionType.ROLL && (
            <div className="space-y-2">
              <Label>掷骰表达式</Label>
              <Input
                value={action.params.expression || "1d20"}
                onChange={(e) => handleParamChange("expression", e.target.value)}
                placeholder="1d20"
              />
              <p className="text-xs text-muted-foreground">
                结果将存储为 ${"${roll.expression}"}
              </p>
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

### 参数编辑器组件

```tsx
// frontend/src/components/macros/ParameterEditor.tsx
import React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card } from "@/components/ui/card";
import { Trash2 } from "lucide-react";

import { MacroParameter } from "@/types/macros";

interface ParameterEditorProps {
  parameter: MacroParameter;
  onChange: (parameter: MacroParameter) => void;
  onRemove: () => void;
}

export function ParameterEditor({ parameter, onChange, onRemove }: ParameterEditorProps) {
  return (
    <Card className="p-4">
      <div className="flex items-start gap-4">
        <div className="flex-1 grid grid-cols-4 gap-4">
          <div className="space-y-2">
            <Label>参数名</Label>
            <Input
              value={parameter.name}
              onChange={(e) => onChange({ ...parameter, name: e.target.value })}
              placeholder="target"
            />
          </div>
          <div className="space-y-2">
            <Label>类型</Label>
            <Select
              value={parameter.type}
              onValueChange={(v) => onChange({ ...parameter, type: v as any })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="string">字符串</SelectItem>
                <SelectItem value="number">数字</SelectItem>
                <SelectItem value="boolean">布尔值</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>默认值</Label>
            <Input
              value={parameter.default || ""}
              onChange={(e) => onChange({ ...parameter, default: e.target.value })}
              placeholder="(可选)"
            />
          </div>
          <div className="flex items-center space-x-2 pt-6">
            <Switch
              checked={parameter.required}
              onCheckedChange={(checked) => onChange({ ...parameter, required: checked })}
            />
            <Label>必填</Label>
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={onRemove}>
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
    </Card>
  );
}
```

### 宏命令列表组件

```tsx
// frontend/src/components/macros/MacroList.tsx
import React, { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Plus, Edit, Trash2, Play } from "lucide-react";

import { Macro } from "@/types/macros";

interface MacroListProps {
  campaignId?: string;
  characterId?: string;
  onEdit: (macro: Macro) => void;
  onExecute: (macro: Macro) => void;
  onCreate: () => void;
}

export function MacroList({ campaignId, characterId, onEdit, onExecute, onCreate }: MacroListProps) {
  const [macros, setMacros] = useState<Macro[]>([]);

  useEffect(() => {
    loadMacros();
  }, [campaignId, characterId]);

  const loadMacros = async () => {
    const params = new URLSearchParams();
    if (campaignId) params.set("campaign_id", campaignId);
    if (characterId) params.set("character_id", characterId);

    const res = await fetch(`/api/macros?${params}`);
    const data = await res.json();
    setMacros(data);
  };

  const handleDelete = async (macroId: string) => {
    if (!confirm("确定要删除这个宏命令吗？")) return;

    await fetch(`/api/macros/${macroId}`, { method: "DELETE" });
    loadMacros();
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">宏命令</h2>
        <Button onClick={onCreate}>
          <Plus className="w-4 h-4 mr-2" />
          创建宏命令
        </Button>
      </div>

      <div className="grid gap-4">
        {macros.map((macro) => (
          <Card key={macro.id}>
            <CardContent className="p-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <code className="text-sm bg-muted px-2 py-1 rounded">
                      /{macro.name}
                    </code>
                    {macro.aliases.map((alias) => (
                      <Badge key={alias} variant="secondary">/{alias}</Badge>
                    ))}
                    {!macro.is_active && (
                      <Badge variant="outline">已禁用</Badge>
                    )}
                  </div>
                  {macro.description && (
                    <p className="text-sm text-muted-foreground mt-2">
                      {macro.description}
                    </p>
                  )}
                  <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                    <span>{macro.actions.length} 个动作</span>
                    <span>使用 {macro.usage_count} 次</span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onExecute(macro)}
                    disabled={!macro.is_active}
                  >
                    <Play className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onEdit(macro)}
                  >
                    <Edit className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(macro.id)}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {macros.length === 0 && (
        <div className="text-center text-muted-foreground py-12">
          暂无宏命令，点击上方按钮创建
        </div>
      )}
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/models/macros.py` | 创建 | 宏命令数据模型 |
| `backend/app/services/macro_service.py` | 创建 | 宏命令服务与执行引擎 |
| `backend/app/api/macros.py` | 创建 | 宏命令 API 路由 |
| `backend/app/schemas/macros.py` | 创建 | Pydantic 模型 |
| `backend/app/db/migrations/versions/xxx_create_macros.py` | 创建 | 数据库迁移 |
| `frontend/src/types/macros.ts` | 创建 | TypeScript 类型定义 |
| `frontend/src/components/macros/MacroEditor.tsx` | 创建 | 宏命令编辑器 |
| `frontend/src/components/macros/ActionBuilder.tsx` | 创建 | 动作构建器 |
| `frontend/src/components/macros/ParameterEditor.tsx` | 创建 | 参数编辑器 |
| `frontend/src/components/macros/MacroList.tsx` | 创建 | 宏命令列表 |
| `frontend/src/pages/MacroManagement.tsx` | 创建 | 宏命令管理页面 |

---

## 验收标准

- [ ] 用户可以创建自定义宏命令
- [ ] 支持参数定义和变量使用
- [ ] 支持多种动作类型（命令、消息、掷骰等）
- [ ] 支持条件分支和循环
- [ ] 宏命令可以在游戏中正确执行
- [ ] 不同作用域的宏命令正确隔离
- [ ] 宏命令使用次数正确统计

---

## 参考文档

- 命令行界面设计最佳实践
- 宏命令系统设计模式
- 变量作用域管理

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
