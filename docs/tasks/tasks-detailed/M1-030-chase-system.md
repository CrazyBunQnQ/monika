# M1-030: 追逐系统

**任务类型**: backend
**预估工时**: 4h
**依赖**: M1-010, M1-020
**状态**: [ ]

---

## 子任务拆解

### 1.1 追逐状态数据模型 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-030-01 | [ ] 创建 `app/core/chase.py` | [ ] |
| M1-030-02 | [ ] 定义 `ChasePhase` 枚举 | [ ] |
| M1-030-03 | [ ] 定义 `ChaseParticipant` | [ ] |
| M1-030-04 | [ ] 定义 `ChaseState` | [ ] |

```python
# app/core/chase.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from datetime import datetime

class ChasePhase(str, Enum):
    """追逐阶段"""
    NOT_STARTED = "not_started"
    SETUP = "setup"           # 设定阶段
    MAIN = "main"             # 追逐阶段
    INTERACTION = "interaction"  # 互动阶段
    ENDED = "ended"

class MovementType(str, Enum):
    """移动类型"""
    WALK = "walk"
    RUN = "run"
    SPRINT = "sprint"
    CRAWL = "crawl"
    CLIMB = "climb"
    JUMP = "jump"
    SWIM = "swim"

class ObstacleType(str, Enum):
    """障碍物类型"""
    DOOR = "door"           # 门
    FENCE = "fence"         # 栅栏
    STAIRS = "stairs"       # 楼梯
    VEHICLE = "vehicle"     # 车辆
    CROWWD = "crowd"        # 人群
    TERRAIN = "terrain"     # 地形

@dataclass
class ChaseParticipant:
    """追逐参与者"""
    character_id: int
    name: str

    # 位置
    position: int = 0         # 当前位置 (距离起点)
    distance_ahead: int = 0   # 领先距离

    # 状态
    movement_type: MovementType = MovementType.RUN
    action_points: int = 1    # 行动点数
    is_exhausted: bool = False

    # 修正值
    speed_modifier: int = 0
    obstacle_modifier: int = 0

    # 临时状态
    status_effects: List[str] = field(default_factory=list)

@dataclass
class Obstacle:
    """障碍物"""
    id: int
    type: ObstacleType
    position: int          # 出现位置
    difficulty: int        # 难度值
    description: str      # 描述

    # 互动结果
    passed: bool = False
    failed: bool = False
    time_penalty: int = 0  # 时间惩罚

@dataclass
class ChaseRound:
    """追逐轮"""
    round_number: int
    participants: List[ChaseParticipant]
    actions: List[str] = field(default_factory=list)
    obstacles_passed: List[int] = field(default_factory=list)
    started_at: datetime = field(default_factory=datetime.utcnow)

@dataclass
class ChaseState:
    """追逐状态"""
    id: Optional[int] = None
    campaign_id: int
    phase: ChasePhase = ChasePhase.NOT_STARTED

    # 参与者
    pursuers: List[ChaseParticipant] = field(default_factory=list)  # 追逐者
    targets: List[ChaseParticipant] = field(default_factory=list)    # 被追逐者

    # 距离
    initial_distance: int = 0      # 初始距离
    current_distance: int = 0       # 当前距离
    escape_distance: int = 100     # 逃脱距离

    # 障碍物
    obstacles: List[Obstacle] = field(default_factory=list)
    next_obstacle_id: int = 1

    # 回合
    current_round: int = 0

    # 历史
    rounds: List[ChaseRound] = field(default_factory=list)
    chase_log: List[str] = field(default_factory=list)

    # 时间
    started_at: Optional[datetime] = None
    ended_at: Optional[datetime] = None

    def add_log(self, message: str):
        """添加日志"""
        timestamp = datetime.now().strftime("%H:%M:%S")
        self.chase_log.append(f"[{timestamp}] {message}")
```

---

### 1.2 追逐引擎 (60min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-030-05 | [ ] 创建 `app/services/chase.py` | [ ] |
| M1-030-06 | [ ] 实现 `start_chase()` | [ ] |
| M1-030-07 | [ ] 实现 `calculate_movement()` | [ ] |
| M1-030-08 | [ ] 实现 `execute_turn()` | [ ] |
| M1-030-09 | [ ] 实现 `handle_obstacle()` | [ ] |

```python
# app/services/chase.py
import random
from typing import Optional, Tuple, List
from datetime import timedelta
from app.core.chase import (
    ChaseState, ChaseParticipant, Obstacle, ChaseRound,
    ChasePhase, MovementType, ObstacleType
)

class ChaseService:
    """追逐服务"""

    # 基础移动距离 (米/轮)
    MOVEMENT_DISTANCE = {
        MovementType.WALK: 5,
        MovementType.RUN: 10,
        MovementType.SPRINT: 20,
        MovementType.CRAWL: 2,
        MovementType.CLIMB: 5,
        MovementType.JUMP: 8,
        MovementType.SWIM: 4,
    }

    def start_chase(
        self,
        pursuer_ids: List[int],
        target_ids: List[int],
        initial_distance: int = 20,
        escape_distance: int = 100
    ) -> ChaseState:
        """开始追逐"""
        state = ChaseState(
            phase=ChasePhase.SETUP,
            pursuers=[
                ChaseParticipant(character_id=pid, name=f"追逐者_{pid}")
                for pid in pursuer_ids
            ],
            targets=[
                ChaseParticipant(character_id=tid, name=f"目标_{tid}")
                for tid in target_ids
            ],
            initial_distance=initial_distance,
            current_distance=initial_distance,
            escape_distance=escape_distance,
            started_at=datetime.utcnow()
        )

        return state

    def calculate_movement(
        self,
        participant: ChaseParticipant,
        movement_type: MovementType,
        is_chase: bool = True
    ) -> int:
        """计算移动距离

        规则:
        - 基础距离由移动类型决定
        - DEX/5 作为敏捷修正
        - 负重/状态可能减少距离
        - 追逐中额外 +5 米
        """
        base_distance = self.MOVEMENT_DISTANCE[movement_type]

        # 敏捷修正 (简化)
        dex_modifier = random.randint(1, 10) if is_chase else 0

        # 状态修正
        status_modifier = 0
        if "exhausted" in participant.status_effects:
            status_modifier = -5
        if "injured" in participant.status_effects:
            status_modifier = -3

        total = base_distance + dex_modifier + status_modifier + participant.speed_modifier

        return max(1, total)  # 至少移动 1 米

    def execute_turn(
        self,
        state: ChaseState,
        participant_id: int,
        movement_type: MovementType = MovementType.RUN,
        action: str = "move"
    ) -> dict:
        """执行一回合"""
        if state.phase == ChasePhase.NOT_STARTED:
            state.phase = ChasePhase.MAIN

        # 找到参与者
        participant = self._find_participant(state, participant_id)
        if not participant:
            return {"error": "参与者不存在"}

        # 计算移动
        distance = self.calculate_movement(participant, movement_type)

        if action == "move":
            return self._handle_movement(state, participant, distance)
        elif action == "obstacle":
            return self._handle_obstacle(state, participant, distance)
        elif action == "dash":
            return self._handle_dash(state, participant, distance)

        return {"error": "不支持的动作"}

    def _handle_movement(
        self,
        state: ChaseState,
        participant: ChaseParticipant,
        distance: int
    ) -> dict:
        """处理移动"""
        is_pursuer = participant in state.pursuers
        is_target = participant in state.targets

        # 追逐者移动
        if is_pursuer:
            # 拉近距离
            state.current_distance = max(0, state.current_distance - distance)
            participant.position += distance

            message = f"{participant.name} 移动了 {distance} 米"

        # 被追逐者移动
        elif is_target:
            # 拉大距离
            state.current_distance += distance
            participant.position += distance

            message = f"{participant.name} 移动了 {distance} 米"

        state.add_log(message)

        # 检查是否逃脱
        if state.current_distance >= state.escape_distance:
            return self._end_chase(state, "targets_escaped")

        return {
            "action": "move",
            "distance": distance,
            "current_gap": state.current_distance,
            "message": message
        }

    def _handle_obstacle(
        self,
        state: ChaseState,
        participant: ChaseParticipant,
        distance: int
    ) -> dict:
        """处理障碍物"""
        # 随机或指定障碍物
        if not state.obstacles:
            self._generate_obstacle(state)

        obstacle = state.obstacles[0]

        # 障碍物检定
        from app.services.check import CheckService
        check_service = CheckService()

        # 简化：使用难度值作为目标
        success = random.randint(1, 100) <= (100 - obstacle.difficulty)

        if success:
            obstacle.passed = True
            message = f"{participant.name} 成功越过 {obstacle.type.value}: {obstacle.description}"
            state.obstacles.pop(0)
        else:
            obstacle.failed = True
            # 失败惩罚
            distance = max(0, distance - obstacle.time_penalty)
            message = f"{participant.name} 未能越过 {obstacle.type.value}，损失 {obstacle.time_penalty} 米"

        state.add_log(message)

        return {
            "action": "obstacle",
            "obstacle": obstacle.type.value,
            "success": success,
            "distance": distance,
            "message": message
        }

    def _generate_obstacle(self, state: ChaseState):
        """生成障碍物"""
        obstacles_pool = [
            (ObstacleType.DOOR, "一扇紧闭的门", 30),
            (ObstacleType.FENCE, "一道高栅栏", 40),
            (ObstacleType.STAIRS, "向下的楼梯", 35),
            (ObstacleType.CROWWD, "拥挤的人群", 25),
            (ObstacleType.TERRAIN, "崎岖的地形", 45),
        ]

        import random
        ob_type, desc, diff = random.choice(obstacles_pool)

        obstacle = Obstacle(
            id=state.next_obstacle_id,
            type=ob_type,
            position=state.current_distance,
            difficulty=diff,
            description=desc
        )

        state.obstacles.append(obstacle)
        state.next_obstacle_id += 1

    def _handle_dash(
        self,
        state: ChaseState,
        participant: ChaseParticipant,
        distance: int
    ) -> dict:
        """处理冲刺（消耗额外1点DB）"""
        # 冲刺移动距离 x2，但消耗更多体力
        doubled_distance = distance * 2

        if participant in state.pursuers:
            state.current_distance = max(0, state.current_distance - doubled_distance)
        elif participant in state.targets:
            state.current_distance += doubled_distance

        message = f"{participant.name} 冲刺了 {doubled_distance} 米！"

        state.add_log(message)

        return {
            "action": "dash",
            "distance": doubled_distance,
            "current_gap": state.current_distance,
            "message": message
        }

    def _end_chase(self, state: ChaseState, result: str) -> dict:
        """结束追逐"""
        state.phase = ChasePhase.ENDED
        state.ended_at = datetime.utcnow()

        duration = (state.ended_at - state.started_at).total_seconds()

        state.add_log(f"追逐结束: {result}")

        return {
            "ended": True,
            "result": result,
            "duration_seconds": duration,
            "distance_traveled": state.escape_distance,
            "chase_log": state.chase_log
        }
```

---

### 1.3 距离计算 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-030-10 | [ ] 实现 `calculate_distance_change()` | [ ] |
| M1-030-11 | [ ] 实现 `get_interaction_range()` | [ ] |

```python
class ChaseService:
    # ...

    def calculate_distance_change(
        self,
        pursuer_movement: int,
        target_movement: int,
        pursuer_check: int,
        target_check: int
    ) -> int:
        """计算距离变化

        正数 = 拉近距离
        负数 = 远离
        """
        # 基础变化
        base_change = pursuer_movement - target_movement

        # 检定修正
        check_modifier = 0
        if pursuer_check < target_check:
            check_modifier = (target_check - pursuer_check) // 10

        return base_change - check_modifier

    def get_interaction_range(self, state: ChaseState) -> bool:
        """检查是否在互动范围内（5米内）"""
        return state.current_distance <= 5

    def can_attack(self, state: ChaseState) -> bool:
        """检查是否在攻击范围内（2米内）"""
        return state.current_distance <= 2
```

---

### 1.4 追逐 API (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-030-12 | [ ] 创建 `app/api/chase.py` | [ ] |
| M1-030-13 | [ ] 实现 POST /chase/start | [ ] |
| M1-030-14 | [ ] 实现 POST /chase/turn | [ ] |
| M1-030-15 | [ ] 实现 GET /chase/status | [ ] |

```python
# app/api/chase.py
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from typing import Optional, List
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.services.chase import ChaseService

router = APIRouter(prefix="/chase", tags=["追逐"])

# 内存存储
chase_states: dict[int, ChaseState] = {}

class StartChaseRequest(BaseModel):
    """开始追逐请求"""
    campaign_id: int
    pursuer_ids: List[int]
    target_ids: List[int]
    initial_distance: int = 20
    escape_distance: int = 100

class TurnRequest(BaseModel):
    """回合请求"""
    chase_id: int
    participant_id: int
    action: str = "move"  # move / obstacle / dash
    movement_type: str = "run"

@router.post("/start")
async def start_chase(
    request: StartChaseRequest,
    current_user: User = Depends(get_current_active_user)
):
    """开始追逐"""
    service = ChaseService()

    state = service.start_chase(
        pursuer_ids=request.pursuer_ids,
        target_ids=request.target_ids,
        initial_distance=request.initial_distance,
        escape_distance=request.escape_distance
    )

    chase_states[request.campaign_id] = state

    return {
        "chase_id": request.campaign_id,
        "phase": state.phase.value,
        "initial_gap": state.current_distance,
        "escape_distance": state.escape_distance
    }

@router.post("/turn")
async def execute_turn(
    request: TurnRequest,
    current_user: User = Depends(get_current_active_user)
):
    """执行回合"""
    state = chase_states.get(request.chase_id)
    if not state:
        raise HTTPException(status_code=404, detail="追逐不存在")

    from app.core.chase import MovementType

    try:
        movement_type = MovementType(request.movement_type)
    except ValueError:
        movement_type = MovementType.RUN

    service = ChaseService()
    result = service.execute_turn(
        state,
        request.participant_id,
        movement_type,
        request.action
    )

    return result

@router.get("/status/{chase_id}")
async def get_chase_status(
    chase_id: int,
    current_user: User = Depends(get_current_active_user)
):
    """获取追逐状态"""
    state = chase_states.get(chase_id)
    if not state:
        raise HTTPException(status_code=404, detail="追逐不存在")

    return {
        "phase": state.phase.value,
        "current_gap": state.current_distance,
        "escape_distance": state.escape_distance,
        "can_interact": state.current_distance <= 5,
        "can_attack": state.current_distance <= 2,
        "round": state.current_round,
        "pursuers": [
            {"id": p.character_id, "position": p.position}
            for p in state.pursuers
        ],
        "targets": [
            {"id": t.character_id, "position": t.position}
            for t in state.targets
        ]
    }
```

---

### 1.5 单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-030-16 | [ ] 创建 `tests/test_chase.py` | [ ] |
| M1-030-17 | [ ] 测试追逐开始 | [ ] |
| M1-030-18 | [ ] 测试移动计算 | [ ] |
| M1-030-19 | [ ] 测试障碍物 | [ ] |
| M1-030-20 | [ ] 测试逃脱判定 | [ ] |

```python
# tests/test_chase.py
import pytest
from app.services.chase import ChaseService
from app.core.chase import ChaseState, MovementType

class TestChaseService:
    def test_start_chase(self):
        """测试开始追逐"""
        service = ChaseService()

        state = service.start_chase(
            pursuer_ids=[1],
            target_ids=[2],
            initial_distance=20,
            escape_distance=100
        )

        assert state.phase.value == "setup"
        assert state.current_distance == 20
        assert len(state.pursuers) == 1
        assert len(state.targets) == 1

    def test_calculate_movement_walk(self):
        """测试行走移动"""
        participant = ChaseParticipant(character_id=1, name="测试")
        service = ChaseService()

        distance = service.calculate_movement(
            participant,
            MovementType.WALK,
            is_chase=False
        )

        assert 1 <= distance <= 15  # 基础5 + 修正0-10

    def test_calculate_movement_run(self):
        """测试奔跑移动"""
        participant = ChaseParticipant(character_id=1, name="测试")
        service = ChaseService()

        distance = service.calculate_movement(
            participant,
            MovementType.RUN,
            is_chase=True
        )

        # 基础10 + 敏捷修正
        assert 1 <= distance <= 30

    def test_obstacle_generation(self):
        """测试障碍物生成"""
        service = ChaseService()
        state = service.start_chase([1], [2])

        service._generate_obstacle(state)

        assert len(state.obstacles) == 1
        assert state.obstacles[0].difficulty > 0

    def test_escape_condition(self):
        """测试逃脱条件"""
        service = ChaseService()
        state = service.start_chase([1], [2], escape_distance=100)

        # 模拟逃跑
        state.current_distance = 100

        # 应该在下一回合检测到逃脱
        assert state.current_distance >= state.escape_distance
```

---

## 验收标准

- [ ] 追逐可以开始/结束
- [ ] 移动距离计算正确
- [ ] 障碍物检定可用
- [ ] 距离变化正确计算
- [ ] 逃脱条件正确判定
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/chase.py` | 创建 | 追逐数据模型 |
| `app/services/chase.py` | 创建 | 追逐引擎 |
| `app/api/chase.py` | 创建 | 追逐 API |
| `tests/test_chase.py` | 创建 | 单元测试 |

---

## 追逐规则速查

| 移动类型 | 基础距离 | 追逐加成 |
|----------|----------|----------|
| 行走 | 5米 | +5米 |
| 奔跑 | 10米 | +5米 |
| 冲刺 | 20米 | +10米 (消耗DB) |
| 爬行 | 2米 | +2米 |
| 攀爬 | 5米 | +5米 |

| 距离 | 效果 |
|------|------|
| >100米 | 逃脱成功 |
| 5-100米 | 普通追逐 |
| 2-5米 | 可互动 |
| <2米 | 可攻击 |
