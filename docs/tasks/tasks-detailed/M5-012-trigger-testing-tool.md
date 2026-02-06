# M5-012: 实现触发器测试工具

**任务ID**: M5-012
**标题**: 实现触发器测试工具
**类型**: fullstack (全栈开发)
**预估工时**: 6h
**依赖**: M5-011 完成

---

## 任务描述

实现一个完整的触发器测试工具，允许 KP 在真实游戏环境之外测试触发器的行为。工具需要支持模拟各种游戏状态、快速切换测试场景、查看详细的执行日志等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-012-01 | 设计测试用例数据结构 | 测试场景定义 | 45min |
| M5-012-02 | 实现测试场景预设库 | 常见场景模板 | 1h |
| M5-012-03 | 实现执行历史记录 | 测试结果存储 | 1h |
| M5-012-04 | 实现前端测试工作台 UI | 完整测试界面 | 2h |
| M5-012-05 | 实现批量测试功能 | 多触发器并行测试 | 1h |
| M5-012-06 | 编写文档与示例 | 用户指南 | 15min |

---

## 完整后端代码示例 (Python + Agno)

### 测试场景模型

```python
# backend/app/models/trigger_testing.py
from datetime import datetime
from typing import Dict, Any, List, Optional
from sqlalchemy import Column, String, JSON, DateTime, Boolean, Integer, ForeignKey, Text
from sqlalchemy.dialects.postgresql import UUID
import uuid

from app.db.base_class import Base


class TestScenario(Base):
    """测试场景预设"""
    __tablename__ = "test_scenarios"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    name = Column(String(255), nullable=False)
    description = Column(Text, nullable=True)

    # 场景类型
    scenario_type = Column(String(50), nullable=False)  # "character", "combat", "exploration", etc.

    # 测试上下文
    context = Column(JSON, nullable=False)

    # 预期结果 (用于验证)
    expected_results = Column(JSON, nullable=True)

    # 是否为系统预设
    is_system = Column(Boolean, default=False)

    # 创建者
    created_by = Column(UUID(as_uuid=True), ForeignKey("accounts.id"))

    created_at = Column(DateTime, default=datetime.utcnow)


class TriggerTestHistory(Base):
    """触发器测试历史"""
    __tablename__ = "trigger_test_history"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    trigger_id = Column(UUID(as_uuid=True), ForeignKey("triggers.id"))

    # 测试上下文
    test_context = Column(JSON, nullable=False)

    # 测试结果
    result = Column(JSON, nullable=False)

    # 执行时间
    executed_at = Column(DateTime, default=datetime.utcnow)

    # 执行耗时 (毫秒)
    execution_time_ms = Column(Integer, nullable=True)

    # 是否通过
    passed = Column(Boolean, default=True)

    # 错误信息
    error_message = Column(Text, nullable=True)


class TriggerTestService:
    """触发器测试服务"""

    @staticmethod
    def create_test_scenario(
        db,
        name: str,
        scenario_type: str,
        context: Dict[str, Any],
        description: Optional[str] = None,
        expected_results: Optional[Dict] = None,
        created_by: Optional[str] = None,
        is_system: bool = False
    ) -> TestScenario:
        """创建测试场景"""
        scenario = TestScenario(
            name=name,
            description=description,
            scenario_type=scenario_type,
            context=context,
            expected_results=expected_results,
            created_by=created_by,
            is_system=is_system
        )

        db.add(scenario)
        db.commit()
        db.refresh(scenario)

        return scenario

    @staticmethod
    def get_system_scenarios(db) -> List[TestScenario]:
        """获取系统预设场景"""
        return db.query(TestScenario).filter(
            TestScenario.is_system == True
        ).all()

    @staticmethod
    def get_user_scenarios(db, user_id: str) -> List[TestScenario]:
        """获取用户自定义场景"""
        return db.query(TestScenario).filter(
            TestScenario.created_by == user_id,
            TestScenario.is_system == False
        ).all()

    @staticmethod
    def run_trigger_test(
        db,
        trigger_id: str,
        test_context: Dict[str, Any],
        save_history: bool = True
    ) -> Dict[str, Any]:
        """
        运行触发器测试

        Args:
            db: 数据库会话
            trigger_id: 触发器 ID
            test_context: 测试上下文
            save_history: 是否保存历史记录

        Returns:
            测试结果
        """
        import time
        from app.services.trigger_service import TriggerService
        from app.models.triggers import Trigger

        start_time = time.time()

        try:
            # 获取触发器
            trigger = db.query(Trigger).filter(Trigger.id == trigger_id).first()

            if not trigger:
                result = {
                    "success": False,
                    "error": "Trigger not found",
                    "passed": False
                }
            else:
                # 执行测试
                result = TriggerService.test_trigger(
                    db,
                    trigger_id,
                    test_context
                )
                result["passed"] = result.get("success", False)

            execution_time = int((time.time() - start_time) * 1000)
            result["execution_time_ms"] = execution_time

            # 保存历史
            if save_history:
                history = TriggerTestHistory(
                    trigger_id=trigger_id,
                    test_context=test_context,
                    result=result,
                    execution_time_ms=execution_time,
                    passed=result.get("passed", False)
                )
                db.add(history)
                db.commit()

            return result

        except Exception as e:
            execution_time = int((time.time() - start_time) * 1000)
            error_result = {
                "success": False,
                "error": str(e),
                "passed": False,
                "execution_time_ms": execution_time
            }

            if save_history:
                history = TriggerTestHistory(
                    trigger_id=trigger_id,
                    test_context=test_context,
                    result=error_result,
                    execution_time_ms=execution_time,
                    passed=False,
                    error_message=str(e)
                )
                db.add(history)
                db.commit()

            return error_result

    @staticmethod
    def run_batch_test(
        db,
        trigger_ids: List[str],
        scenario_id: str
    ) -> List[Dict[str, Any]]:
        """
        批量测试多个触发器

        Args:
            db: 数据库会话
            trigger_ids: 触发器 ID 列表
            scenario_id: 测试场景 ID

        Returns:
            所有测试结果
        """
        # 获取场景上下文
        scenario = db.query(TestScenario).filter(
            TestScenario.id == scenario_id
        ).first()

        if not scenario:
            return [{"error": "Scenario not found"}]

        results = []
        for trigger_id in trigger_ids:
            result = TriggerTestService.run_trigger_test(
                db,
                trigger_id,
                scenario.context,
                save_history=True
            )
            results.append({
                "trigger_id": trigger_id,
                ...result
            })

        return results

    @staticmethod
    def get_test_history(
        db,
        trigger_id: str,
        limit: int = 50
    ) -> List[TriggerTestHistory]:
        """获取测试历史"""
        return db.query(TriggerTestHistory).filter(
            TriggerTestHistory.trigger_id == trigger_id
        ).order_by(
            TriggerTestHistory.executed_at.desc()
        ).limit(limit).all()

    @staticmethod
    def get_test_statistics(
        db,
        trigger_id: str
    ) -> Dict[str, Any]:
        """获取测试统计信息"""
        from sqlalchemy import func

        history = db.query(TriggerTestHistory).filter(
            TriggerTestHistory.trigger_id == trigger_id
        )

        total = history.count()
        passed = history.filter(TriggerTestHistory.passed == True).count()
        failed = total - passed

        avg_time = db.query(
            func.avg(TriggerTestHistory.execution_time_ms)
        ).filter(
            TriggerTestHistory.trigger_id == trigger_id
        ).scalar()

        return {
            "total_tests": total,
            "passed": passed,
            "failed": failed,
            "pass_rate": round(passed / total * 100, 2) if total > 0 else 0,
            "avg_execution_time_ms": round(avg_time, 2) if avg_time else 0
        }
```

### 预设场景数据

```python
# backend/app/data/default_test_scenarios.py

DEFAULT_TEST_SCENARIOS = [
    {
        "name": "低血量状态",
        "description": "角色 HP 低于 10 的状态",
        "scenario_type": "character",
        "context": {
            "character": {
                "hp": 8,
                "hp_max": 15,
                "san": 45,
                "san_max": 60,
                "luck": 50
            },
            "scene": {
                "id": "scene_danger"
            },
            "state": {
                "turn_count": 5,
                "in_combat": True
            }
        }
    },
    {
        "name": "濒临疯狂",
        "description": "SAN 值极低的状态",
        "scenario_type": "character",
        "context": {
            "character": {
                "hp": 12,
                "hp_max": 15,
                "san": 15,
                "san_max": 60,
                "luck": 50
            },
            "scene": {
                "id": "scene_horror"
            },
            "state": {
                "turn_count": 3,
                "in_combat": False
            }
        }
    },
    {
        "name": "战斗中",
        "description": "战斗状态",
        "scenario_type": "combat",
        "context": {
            "character": {
                "hp": 10,
                "hp_max": 15,
                "san": 50,
                "san_max": 60,
                "luck": 50
            },
            "scene": {
                "id": "scene_combat"
            },
            "state": {
                "turn_count": 2,
                "in_combat": True,
                "enemies_count": 3
            }
        }
    },
    {
        "name": "探索场景",
        "description": "普通探索状态",
        "scenario_type": "exploration",
        "context": {
            "character": {
                "hp": 15,
                "hp_max": 15,
                "san": 60,
                "san_max": 60,
                "luck": 50
            },
            "scene": {
                "id": "scene_library"
            },
            "state": {
                "turn_count": 1,
                "in_combat": False
            }
        }
    },
    {
        "name": "重伤濒死",
        "description": "HP 低于 3 的危险状态",
        "scenario_type": "character",
        "context": {
            "character": {
                "hp": 2,
                "hp_max": 15,
                "san": 30,
                "san_max": 60,
                "luck": 50
            },
            "scene": {
                "id": "scene_critical"
            },
            "state": {
                "turn_count": 8,
                "in_combat": True
            }
        }
    }
]


def seed_default_scenarios(db):
    """初始化系统预设场景"""
    from app.models.trigger_testing import TestScenario
    from app.services.trigger_testing import TriggerTestService

    for scenario_data in DEFAULT_TEST_SCENARIOS:
        existing = db.query(TestScenario).filter(
            TestScenario.name == scenario_data["name"],
            TestScenario.is_system == True
        ).first()

        if not existing:
            TriggerTestService.create_test_scenario(
                db,
                **scenario_data,
                is_system=True
            )
```

### API 路由

```python
# backend/app/api/trigger_testing.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.schemas.trigger_testing import (
    TestScenarioCreate,
    TestScenarioResponse,
    BatchTestRequest,
    BatchTestResponse,
    TestStatisticsResponse
)
from app.services.trigger_testing import TriggerTestService
from app.data.default_test_scenarios import seed_default_scenarios

router = APIRouter()


@router.post("/scenarios/seed")
def seed_scenarios(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """初始化系统预设场景"""
    seed_default_scenarios(db)
    return {"message": "Scenarios seeded successfully"}


@router.get("/scenarios", response_model=List[TestScenarioResponse])
def list_scenarios(
    system_only: bool = False,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """列出所有测试场景"""
    if system_only:
        return TriggerTestService.get_system_scenarios(db)
    else:
        return TriggerTestService.get_user_scenarios(
            db,
            current_user.id
        ) + TriggerTestService.get_system_scenarios(db)


@router.post("/scenarios", response_model=TestScenarioResponse)
def create_scenario(
    scenario_in: TestScenarioCreate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """创建自定义测试场景"""
    return TriggerTestService.create_test_scenario(
        db,
        **scenario_in.dict(),
        created_by=current_user.id
    )


@router.post("/test/{trigger_id}")
def test_trigger(
    trigger_id: str,
    test_context: dict,
    save_history: bool = True,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """测试单个触发器"""
    return TriggerTestService.run_trigger_test(
        db,
        trigger_id,
        test_context,
        save_history
    )


@router.post("/batch", response_model=BatchTestResponse)
def batch_test(
    request: BatchTestRequest,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """批量测试触发器"""
    results = TriggerTestService.run_batch_test(
        db,
        request.trigger_ids,
        request.scenario_id
    )

    passed = sum(1 for r in results if r.get("passed", False))
    return {
        "total": len(results),
        "passed": passed,
        "failed": len(results) - passed,
        "results": results
    }


@router.get("/history/{trigger_id}")
def get_test_history(
    trigger_id: str,
    limit: int = 50,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取测试历史"""
    return TriggerTestService.get_test_history(
        db,
        trigger_id,
        limit
    )


@router.get("/statistics/{trigger_id}", response_model=TestStatisticsResponse)
def get_test_statistics(
    trigger_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取测试统计"""
    return TriggerTestService.get_test_statistics(db, trigger_id)
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 类型定义

```typescript
// frontend/src/types/trigger-testing.ts
export interface TestScenario {
  id: string;
  name: string;
  description?: string;
  scenario_type: string;
  context: TestContext;
  expected_results?: any;
  is_system: boolean;
  created_at: string;
}

export interface TestContext {
  character?: {
    hp: number;
    hp_max: number;
    san: number;
    san_max: number;
    luck: number;
  };
  scene?: {
    id: string;
  };
  state?: {
    turn_count: number;
    in_combat: boolean;
    [key: string]: any;
  };
}

export interface TriggerTestResult {
  success: boolean;
  triggered?: boolean;
  passed?: boolean;
  actions?: any[];
  reason?: string;
  error?: string;
  execution_time_ms?: number;
}

export interface TestHistory {
  id: string;
  trigger_id: string;
  test_context: TestContext;
  result: TriggerTestResult;
  executed_at: string;
  execution_time_ms: number;
  passed: boolean;
  error_message?: string;
}

export interface TestStatistics {
  total_tests: number;
  passed: number;
  failed: number;
  pass_rate: number;
  avg_execution_time_ms: number;
}
```

### 触发器测试工作台组件

```tsx
// frontend/src/components/triggers/TriggerTestWorkbench.tsx
import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Play, RefreshCw, Clock, CheckCircle, XCircle, BarChart3 } from "lucide-react";

import { Trigger } from "@/types/triggers";
import { TestScenario, TriggerTestResult, TestHistory, TestStatistics } from "@/types/trigger-testing";
import { ScenarioSelector } from "./ScenarioSelector";
import { ContextEditor } from "./ContextEditor";
import { TestResultViewer } from "./TestResultViewer";
import { TestHistoryList } from "./TestHistoryList";
import { TestStatisticsView } from "./TestStatisticsView";

interface TriggerTestWorkbenchProps {
  trigger: Trigger;
}

export function TriggerTestWorkbench({ trigger }: TriggerTestWorkbenchProps) {
  const [scenarios, setScenarios] = useState<TestScenario[]>([]);
  const [selectedScenario, setSelectedScenario] = useState<TestScenario | null>(null);
  const [customContext, setCustomContext] = useState<any>(null);
  const [testResult, setTestResult] = useState<TriggerTestResult | null>(null);
  const [testing, setTesting] = useState(false);
  const [history, setHistory] = useState<TestHistory[]>([]);
  const [statistics, setStatistics] = useState<TestStatistics | null>(null);
  const [activeTab, setActiveTab] = useState("test");

  useEffect(() => {
    loadScenarios();
    loadHistory();
    loadStatistics();
  }, [trigger.id]);

  const loadScenarios = async () => {
    const res = await fetch("/api/trigger-testing/scenarios");
    const data = await res.json();
    setScenarios(data);
  };

  const loadHistory = async () => {
    const res = await fetch(`/api/trigger-testing/history/${trigger.id}`);
    const data = await res.json();
    setHistory(data);
  };

  const loadStatistics = async () => {
    const res = await fetch(`/api/trigger-testing/statistics/${trigger.id}`);
    const data = await res.json();
    setStatistics(data);
  };

  const handleScenarioSelect = (scenarioId: string) => {
    const scenario = scenarios.find(s => s.id === scenarioId);
    if (scenario) {
      setSelectedScenario(scenario);
      setCustomContext(scenario.context);
    }
  };

  const handleRunTest = async () => {
    setTesting(true);
    try {
      const res = await fetch(`/api/trigger-testing/test/${trigger.id}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          test_context: customContext,
          save_history: true
        })
      });

      const result: TriggerTestResult = await res.json();
      setTestResult(result);

      // 刷新历史和统计
      await loadHistory();
      await loadStatistics();
    } finally {
      setTesting(false);
    }
  };

  const currentContext = customContext || selectedScenario?.context || {
    character: { hp: 15, hp_max: 15, san: 60, san_max: 60, luck: 50 },
    scene: { id: "" },
    state: { turn_count: 1, in_combat: false }
  };

  return (
    <div className="space-y-6">
      {/* 顶部信息栏 */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">触发器测试</h2>
          <p className="text-muted-foreground">{trigger.name}</p>
        </div>
        {statistics && (
          <div className="flex items-center gap-4">
            <Badge variant={statistics.pass_rate > 80 ? "default" : "destructive"}>
              通过率: {statistics.pass_rate}%
            </Badge>
            <Badge variant="outline">
              总测试: {statistics.total_tests}
            </Badge>
          </div>
        )}
      </div>

      {/* 主内容区 */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="test">测试运行</TabsTrigger>
          <TabsTrigger value="history">
            测试历史
            {history.length > 0 && (
              <Badge variant="secondary" className="ml-2">
                {history.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="statistics">统计分析</TabsTrigger>
          <TabsTrigger value="scenarios">场景管理</TabsTrigger>
        </TabsList>

        <TabsContent value="test" className="space-y-6">
          <div className="grid grid-cols-2 gap-6">
            {/* 左侧: 场景选择与上下文编辑 */}
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>选择测试场景</CardTitle>
                </CardHeader>
                <CardContent>
                  <ScenarioSelector
                    scenarios={scenarios}
                    selectedId={selectedScenario?.id}
                    onSelect={handleScenarioSelect}
                  />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>测试上下文</CardTitle>
                </CardHeader>
                <CardContent>
                  <ContextEditor
                    context={currentContext}
                    onChange={setCustomContext}
                  />
                </CardContent>
              </Card>
            </div>

            {/* 右侧: 测试结果 */}
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle>测试结果</CardTitle>
                    <Button
                      onClick={handleRunTest}
                      disabled={testing}
                      size="sm"
                    >
                      {testing ? (
                        <>
                          <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                          测试中...
                        </>
                      ) : (
                        <>
                          <Play className="w-4 h-4 mr-2" />
                          运行测试
                        </>
                      )}
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  {testResult ? (
                    <TestResultViewer result={testResult} />
                  ) : (
                    <div className="text-center text-muted-foreground py-8">
                      点击"运行测试"开始测试
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="history">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>测试历史</CardTitle>
                <Button variant="outline" size="sm" onClick={loadHistory}>
                  <RefreshCw className="w-4 h-4 mr-2" />
                  刷新
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <ScrollArea className="h-[600px]">
                <TestHistoryList
                  history={history}
                  onSelect={(h) => {
                    setCustomContext(h.test_context);
                    setTestResult(h.result);
                    setActiveTab("test");
                  }}
                />
              </ScrollArea>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="statistics">
          <Card>
            <CardHeader>
              <CardTitle>统计分析</CardTitle>
            </CardHeader>
            <CardContent>
              <TestStatisticsView statistics={statistics} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="scenarios">
          <Card>
            <CardHeader>
              <CardTitle>场景管理</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-center text-muted-foreground py-8">
                场景管理功能开发中...
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
```

### 场景选择器组件

```tsx
// frontend/src/components/triggers/ScenarioSelector.tsx
import React from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";

import { TestScenario } from "@/types/trigger-testing";

interface ScenarioSelectorProps {
  scenarios: TestScenario[];
  selectedId?: string;
  onSelect: (scenarioId: string) => void;
}

export function ScenarioSelector({ scenarios, selectedId, onSelect }: ScenarioSelectorProps) {
  const systemScenarios = scenarios.filter(s => s.is_system);
  const userScenarios = scenarios.filter(s => !s.is_system);

  return (
    <ScrollArea className="h-[300px]">
      <div className="space-y-4">
        {systemScenarios.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-2">系统预设</h4>
            <div className="space-y-2">
              {systemScenarios.map(scenario => (
                <ScenarioCard
                  key={scenario.id}
                  scenario={scenario}
                  selected={selectedId === scenario.id}
                  onSelect={() => onSelect(scenario.id)}
                />
              ))}
            </div>
          </div>
        )}

        {userScenarios.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-2">自定义</h4>
            <div className="space-y-2">
              {userScenarios.map(scenario => (
                <ScenarioCard
                  key={scenario.id}
                  scenario={scenario}
                  selected={selectedId === scenario.id}
                  onSelect={() => onSelect(scenario.id)}
                />
              ))}
            </div>
          </div>
        )}

        {scenarios.length === 0 && (
          <div className="text-center text-muted-foreground py-4">
            暂无测试场景
          </div>
        )}
      </div>
    </ScrollArea>
  );
}

interface ScenarioCardProps {
  scenario: TestScenario;
  selected: boolean;
  onSelect: () => void;
}

function ScenarioCard({ scenario, selected, onSelect }: ScenarioCardProps) {
  return (
    <Card
      className={`p-3 cursor-pointer transition-colors ${
        selected ? "border-primary bg-primary/5" : ""
      }`}
      onClick={onSelect}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="font-medium">{scenario.name}</span>
            {scenario.is_system && (
              <Badge variant="secondary" className="text-xs">预设</Badge>
            )}
          </div>
          {scenario.description && (
            <p className="text-xs text-muted-foreground mt-1">
              {scenario.description}
            </p>
          )}
        </div>
      </div>
    </Card>
  );
}
```

### 上下文编辑器组件

```tsx
// frontend/src/components/triggers/ContextEditor.tsx
import React from "react";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";

import { TestContext } from "@/types/trigger-testing";

interface ContextEditorProps {
  context: TestContext;
  onChange: (context: TestContext) => void;
}

export function ContextEditor({ context, onChange }: ContextEditorProps) {
  const handleCharacterChange = (field: string, value: number) => {
    onChange({
      ...context,
      character: {
        ...context.character,
        [field]: value
      }
    });
  };

  const handleSceneChange = (value: string) => {
    onChange({
      ...context,
      scene: {
        id: value
      }
    });
  };

  const handleStateChange = (field: string, value: any) => {
    onChange({
      ...context,
      state: {
        ...context.state,
        [field]: value
      }
    });
  };

  return (
    <div className="space-y-4">
      {/* 角色状态 */}
      <div>
        <h4 className="text-sm font-semibold mb-2">角色状态</h4>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label className="text-xs">HP</Label>
            <Input
              type="number"
              value={context.character?.hp || 15}
              onChange={(e) => handleCharacterChange("hp", parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">HP Max</Label>
            <Input
              type="number"
              value={context.character?.hp_max || 15}
              onChange={(e) => handleCharacterChange("hp_max", parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">SAN</Label>
            <Input
              type="number"
              value={context.character?.san || 60}
              onChange={(e) => handleCharacterChange("san", parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">SAN Max</Label>
            <Input
              type="number"
              value={context.character?.san_max || 60}
              onChange={(e) => handleCharacterChange("san_max", parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">幸运</Label>
            <Input
              type="number"
              value={context.character?.luck || 50}
              onChange={(e) => handleCharacterChange("luck", parseInt(e.target.value) || 0)}
            />
          </div>
        </div>
      </div>

      <Separator />

      {/* 场景信息 */}
      <div>
        <h4 className="text-sm font-semibold mb-2">场景信息</h4>
        <div className="space-y-1">
          <Label className="text-xs">场景 ID</Label>
          <Input
            value={context.scene?.id || ""}
            onChange={(e) => handleSceneChange(e.target.value)}
            placeholder="输入场景 ID"
          />
        </div>
      </div>

      <Separator />

      {/* 游戏状态 */}
      <div>
        <h4 className="text-sm font-semibold mb-2">游戏状态</h4>
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">回合数</Label>
              <Input
                type="number"
                value={context.state?.turn_count || 1}
                onChange={(e) => handleStateChange("turn_count", parseInt(e.target.value) || 0)}
              />
            </div>
            <div className="flex items-center space-x-2 pt-6">
              <Switch
                id="in-combat"
                checked={context.state?.in_combat || false}
                onCheckedChange={(checked) => handleStateChange("in_combat", checked)}
              />
              <Label htmlFor="in-combat" className="text-xs">战斗中</Label>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
```

### 测试结果查看器组件

```tsx
// frontend/src/components/triggers/TestResultViewer.tsx
import React from "react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CheckCircle, XCircle, Clock, AlertCircle } from "lucide-react";

import { TriggerTestResult } from "@/types/trigger-testing";

interface TestResultViewerProps {
  result: TriggerTestResult;
}

export function TestResultViewer({ result }: TestResultViewerProps) {
  return (
    <div className="space-y-4">
      {/* 执行状态 */}
      <Card className="p-4">
        <div className="flex items-center gap-3">
          {result.passed ? (
            <CheckCircle className="w-6 h-6 text-green-500" />
          ) : (
            <XCircle className="w-6 h-6 text-red-500" />
          )}
          <div className="flex-1">
            <p className="font-semibold">
              {result.passed ? "测试通过" : "测试失败"}
            </p>
            {result.reason && (
              <p className="text-sm text-muted-foreground">{result.reason}</p>
            )}
            {result.error && (
              <p className="text-sm text-red-500">{result.error}</p>
            )}
          </div>
          {result.execution_time_ms && (
            <div className="flex items-center gap-1 text-muted-foreground">
              <Clock className="w-4 h-4" />
              <span className="text-sm">{result.execution_time_ms}ms</span>
            </div>
          )}
        </div>
      </Card>

      {/* 触发状态 */}
      {result.triggered !== undefined && (
        <Card className="p-4">
          <p className="text-sm text-muted-foreground mb-2">触发状态</p>
          {result.triggered ? (
            <Badge variant="default">触发器将被执行</Badge>
          ) : (
            <Badge variant="outline">触发器不会执行</Badge>
          )}
        </Card>
      )}

      {/* 将执行的动作 */}
      {result.actions && result.actions.length > 0 && (
        <Card className="p-4">
          <p className="text-sm text-muted-foreground mb-3">将执行的动作</p>
          <div className="space-y-2">
            {result.actions.map((action, index) => (
              <div key={index} className="flex items-center gap-2">
                <Badge variant="outline">{action.type}</Badge>
                <span className="text-sm text-muted-foreground">
                  {JSON.stringify(action.params)}
                </span>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
```

### 测试历史列表组件

```tsx
// frontend/src/components/triggers/TestHistoryList.tsx
import React from "react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CheckCircle, XCircle } from "lucide-react";
import { formatDistanceToNow } from "date-fns";

import { TestHistory } from "@/types/trigger-testing";

interface TestHistoryListProps {
  history: TestHistory[];
  onSelect: (history: TestHistory) => void;
}

export function TestHistoryList({ history, onSelect }: TestHistoryListProps) {
  if (history.length === 0) {
    return (
      <div className="text-center text-muted-foreground py-8">
        暂无测试历史
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {history.map((item) => (
        <Card
          key={item.id}
          className="p-3 cursor-pointer hover:bg-accent/50 transition-colors"
          onClick={() => onSelect(item)}
        >
          <div className="flex items-center gap-3">
            {item.passed ? (
              <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0" />
            ) : (
              <XCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
            )}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <Badge variant={item.passed ? "default" : "destructive"} className="text-xs">
                  {item.passed ? "通过" : "失败"}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  {formatDistanceToNow(new Date(item.executed_at), { addSuffix: true })}
                </span>
              </div>
              {item.error_message && (
                <p className="text-xs text-red-500 truncate mt-1">
                  {item.error_message}
                </p>
              )}
            </div>
            <div className="text-xs text-muted-foreground">
              {item.execution_time_ms}ms
            </div>
          </div>
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
| `backend/app/models/trigger_testing.py` | 创建 | 测试场景和历史模型 |
| `backend/app/services/trigger_testing.py` | 创建 | 测试服务 |
| `backend/app/data/default_test_scenarios.py` | 创建 | 预设场景数据 |
| `backend/app/api/trigger_testing.py` | 创建 | 测试 API 路由 |
| `frontend/src/types/trigger-testing.ts` | 创建 | 类型定义 |
| `frontend/src/components/triggers/TriggerTestWorkbench.tsx` | 创建 | 测试工作台主组件 |
| `frontend/src/components/triggers/ScenarioSelector.tsx` | 创建 | 场景选择器 |
| `frontend/src/components/triggers/ContextEditor.tsx` | 创建 | 上下文编辑器 |
| `frontend/src/components/triggers/TestResultViewer.tsx` | 创建 | 结果查看器 |
| `frontend/src/components/triggers/TestHistoryList.tsx` | 创建 | 历史列表 |

---

## 验收标准

- [ ] KP 可以从预设场景快速选择测试上下文
- [ ] 支持自定义测试上下文
- [ ] 测试结果清晰显示触发状态和执行动作
- [ ] 测试历史可追溯和重放
- [ ] 提供测试统计信息（通过率、平均耗时）
- [ ] 测试执行速度快（< 100ms）
- [ ] 支持批量测试多个触发器

---

## 参考文档

- 软件测试最佳实践
- React 性能优化指南
- 测试驱动开发 (TDD) 方法论

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
