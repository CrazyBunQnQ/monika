# M1-020: 战斗系统

**任务类型**: backend
**预估工时**: 5h
**依赖**: M1-003, M1-010
**状态**: [ ]

---

## 子任务拆解

### 1.1 战斗状态数据模型 (50min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-020-01 | [ ] 创建 `app/core/combat.py` | [ ] |
| M1-020-02 | [ ] 定义 `CombatPhase` 枚举 | [ ] |
| M1-020-03 | [ ] 定义 `Combatant` 数据类 | [ ] |
| M1-020-04 | [ ] 定义 `CombatState` 数据类 | [ ] |

```python
# app/core/combat.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from datetime import datetime
from app.db.models.character import Character

class CombatPhase(str, Enum):
    """战斗阶段"""
    NOT_STARTED = "not_started"
    INITIATIVE = "initiative"     # 回合顺序
    ROUND = "round"               # 战斗轮
    ENDED = "ended"

class ActionType(str, Enum):
    """动作类型"""
    ATTACK = "attack"
    DEFEND = "defend"
    DODGE = "dodge"
    ITEM = "item"
    CAST = "cast"
    FLEE = "flee"
    SPECIAL = "special"

class DamageType(str, Enum):
    """伤害类型"""
    BLUNT = "blunt"       # 钝击
    SLASH = "slash"       # 切割
    PIERCE = "pierce"      # 穿刺
    FIRE = "fire"          # 火焰
    COLD = "cold"          # 寒冷
    ELECTRIC = "electric"  # 电击
    TOXIC = "toxic"        # 毒素
    MAGICAL = "magical"    # 魔法
    SANITY = "sanity"      # 精神

@dataclass
class Combatant:
    """战斗者"""
    character: Character
    initiative: int = 0         # 回合值
    hp_current: int = 0          # 当前 HP
    mp_current: int = 0          # 当前 MP
    status_effects: List[str] = field(default_factory=list)  # 状态效果

    # 战斗属性
    armor_class: int = 10       # 护甲等级
    attack_bonus: int = 0        # 攻击加值
    damage_bonus: int = 0        # 伤害加值

    # 行动
    has_acted: bool = False
    actions_left: int = 1        # 本轮剩余动作

    # 临时修正
    temporary_modifiers: Dict[str, int] = field(default_factory=dict)

@dataclass
class CombatAction:
    """战斗动作"""
    combatant_id: int
    action_type: ActionType
    target_id: Optional[int] = None

    # 攻击详情
    attack_roll: Optional[int] = None
    damage_rolls: List[int] = field(default_factory=list)
    damage_type: DamageType = DamageType.BLUNT

    # 结果
    hit: bool = False
    damage: int = 0
    critical: bool = False
    fumble: bool = False

    # 叙述
    narrative: str = ""

@dataclass
class CombatRound:
    """战斗轮"""
    round_number: int
    phase: CombatPhase
    combatants: List[Combatant]
    actions: List[CombatAction] = field(default_factory=list)
    started_at: datetime = field(default_factory=datetime.utcnow)
    ended_at: Optional[datetime] = None

@dataclass
class CombatState:
    """战斗状态"""
    id: Optional[int] = None
    campaign_id: int
    phase: CombatPhase = CombatPhase.NOT_STARTED

    # 战斗者
    combatants: List[Combatant] = field(default_factory=list)
    player_ids: List[int] = field(default_factory=list)
    enemy_ids: List[int] = field(default_factory=list)

    # 回合
    current_round: int = 0
    current_turn: int = 0  # 当前行动索引
    turn_order: List[int] = field(default_factory=list)  # combatant_id 顺序

    # 历史
    rounds: List[CombatRound] = field(default_factory=list)
    combat_log: List[str] = field(default_factory=list)

    # 时间
    started_at: Optional[datetime] = None
    ended_at: Optional[datetime] = None

    def add_log(self, message: str):
        """添加战斗日志"""
        timestamp = datetime.now().strftime("%H:%M:%S")
        self.combat_log.append(f"[{timestamp}] {message}")
```

---

### 1.2 战斗引擎 (60min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-020-05 | [ ] 创建 `app/services/combat.py` | [ ] |
| M1-020-06 | [ ] 实现 `start_combat()` | [ ] |
| M1-020-07 | [ ] 实现 `calculate_initiative()` | [ ] |
| M1-020-08 | [ ] 实现 `execute_action()` | [ ] |
| M1-020-09 | [ ] 实现 `calculate_damage()` | [ ] |

```python
# app/services/combat.py
import random
from typing import Optional, Tuple
from app.core.combat import (
    CombatState, Combatant, CombatAction, CombatRound,
    CombatPhase, ActionType, DamageType
)

class CombatService:
    """战斗服务"""

    def start_combat(
        self,
        player_ids: List[int],
        enemy_ids: List[int]
    ) -> CombatState:
        """开始战斗"""
        state = CombatState(
            phase=CombatPhase.INITIATIVE,
            player_ids=player_ids,
            enemy_ids=enemy_ids,
            started_at=datetime.utcnow()
        )

        # 初始化战斗者
        for cid in player_ids + enemy_ids:
            combatant = Combatant(character_id=cid)
            state.combatants.append(combatant)

        # 计算先攻
        self._roll_initiatives(state)

        # 设置回合顺序
        state.turn_order = sorted(
            state.combatants,
            key=lambda c: c.initiative,
            reverse=True
        )

        return state

    def _roll_initiatives(self, state: CombatState):
        """掷先攻值"""
        for combatant in state.combatants:
            # d100 + DEX/5 + 其他修正
            base_initiative = random.randint(1, 100)
            dex_bonus = (combatant.character.dex) // 5 if combatant.character else 0
            modifier = combatant.temporary_modifiers.get("initiative", 0)

            combatant.initiative = base_initiative + dex_bonus + modifier

    def execute_attack(
        self,
        state: CombatState,
        attacker_id: int,
        target_id: int,
        attack_type: str = "melee",
        weapon_bonus: int = 0
    ) -> CombatAction:
        """执行攻击"""
        attacker = self._get_combatant(state, attacker_id)
        target = self._get_combatant(state, target_id)

        action = CombatAction(
            combatant_id=attacker_id,
            action_type=ActionType.ATTACK,
            target_id=target_id,
            damage_type=DamageType.BLUNT
        )

        # 1. 攻击检定 (d100)
        attack_roll = random.randint(1, 100)
        attack_modifier = attacker.attack_bonus + weapon_bonus
        attack_value = attack_roll + attack_modifier

        # 目标护甲等级 (AC)
        target_ac = target.armor_class

        # 判定命中
        # 大成功: 1-5% (自动命中 + 双倍伤害)
        # 普通命中: 攻击值 >= AC
        # 失败: 攻击值 < AC

        if attack_roll <= 5:
            action.hit = True
            action.critical = True
            action.narrative = f"攻击命中！大成功！(原始: {attack_roll})"
        elif attack_roll >= 96:
            action.hit = False
            action.fumble = True
            action.narrative = f"攻击失误！大失败！(原始: {attack_roll})"
        elif attack_value >= target_ac:
            action.hit = True
            action.narrative = f"攻击命中！({attack_value} vs AC {target_ac})"
        else:
            action.hit = False
            action.narrative = f"攻击未命中！({attack_value} vs AC {target_ac})"

        # 2. 计算伤害
        if action.hit:
            damage = self._calculate_damage(
                attacker, target, action.critical, attack_type
            )
            action.damage_rolls = damage["rolls"]
            action.damage = damage["total"]
            action.damage_type = damage["type"]

            # 应用伤害
            target.hp_current = max(0, target.hp_current - damage["total"])
            action.narrative += f" 造成 {damage['total']} 点{damage['type'].value}伤害"

            # 检查是否倒下
            if target.hp_current <= 0:
                action.narrative += " 目标倒下！"

        state.combat_log.append(action.narrative)
        return action

    def _calculate_damage(
        self,
        attacker: Combatant,
        target: Combatant,
        is_critical: bool,
        attack_type: str
    ) -> Dict[str, Any]:
        """计算伤害"""
        # 基础伤害
        if attack_type == "melee":
            damage_dice = (1, 6)  # 1d6
        elif attack_type == "ranged":
            damage_dice = (1, 4)  # 1d4
        elif attack_type == "unarmed":
            damage_dice = (1, 3)  # 1d3
        else:
            damage_dice = (1, 4)

        dice_count, dice_sides = damage_dice

        # 掷伤害骰
        rolls = [random.randint(1, dice_sides) for _ in range(dice_count)]
        base_damage = sum(rolls)

        # 加成
        bonus = attacker.damage_bonus

        # 大成功双倍
        if is_critical:
            base_damage *= 2
            bonus *= 2

        total = base_damage + bonus

        return {
            "rolls": rolls,
            "total": total,
            "bonus": bonus,
            "type": DamageType.BLUNT
        }

    def _get_combatant(
        self,
        state: CombatState,
        combatant_id: int
    ) -> Optional[Combatant]:
        """获取战斗者"""
        for c in state.combatants:
            if c.character.id == combatant_id:
                return c
        return None
```

---

### 1.3 闪避与格挡 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-020-10 | [ ] 实现 `execute_dodge()` | [ ] |
| M1-020-11 | [ ] 实现 `execute_defend()` | [ ] |

```python
class CombatService:
    # ...

    def execute_dodge(
        self,
        state: CombatState,
        dodger_id: int
    ) -> CombatAction:
        """执行闪避"""
        dodger = self._get_combatant(state, dodger_id)

        action = CombatAction(
            combatant_id=dodger_id,
            action_type=ActionType.DODGE
        )

        # 闪避检定 (d100)
        dodge_value = (dodger.character.dex) // 5 if dodger.character else 10
        roll = random.randint(1, 100)

        # 闪避成功条件
        if roll <= dodge_value:
            action.hit = True  # 表示闪避成功
            action.narrative = f"闪避成功！(检定: {roll} <= {dodge_value})"
        else:
            action.hit = False
            action.narrative = f"闪避失败！(检定: {roll} > {dodge_value})"

        state.combat_log.append(action.narrative)
        return action

    def execute_defend(
        self,
        state: CombatState,
        defender_id: int
    ) -> CombatAction:
        """执行格挡"""
        defender = self._get_combatant(state, defender_id)

        action = CombatAction(
            combatant_id=defender_id,
            action_type=ActionType.DEFEND
        )

        # 格挡给目标增加临时护甲
        block_value = (defender.character.str) // 10 if defender.character else 5

        # 格挡不是立即生效，而是在被攻击时应用
        defender.temporary_modifiers["block"] = block_value
        action.narrative = f"进入防御姿态，获得 +{block_value} 护甲等级"

        state.combat_log.append(action.narrative)
        return action
```

---

### 1.4 战斗状态管理 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-020-12 | [ ] 实现 `next_turn()` | [ ] |
| M1-020-13 | [ ] 实现 `end_combat()` | [ ] |
| M1-020-14 | [ ] 实现 `apply_status_effect()` | [ ] |

```python
class CombatService:
    # ...

    def next_turn(self, state: CombatState) -> Optional[Combatant]:
        """进入下一轮"""
        if state.phase == CombatPhase.NOT_STARTED:
            state.phase = CombatPhase.ROUND
            state.current_round = 1
        elif state.phase == CombatPhase.ROUND:
            # 检查是否需要新回合
            if state.current_turn >= len(state.turn_order):
                state.current_turn = 0
                state.current_round += 1

                # 重置所有战斗者行动
                for c in state.combatants:
                    c.has_acted = False
                    c.actions_left = 1

        if state.current_turn < len(state.turn_order):
            return state.turn_order[state.current_turn]

        return None

    def end_combat(self, state: CombatState) -> Dict[str, Any]:
        """结束战斗"""
        state.phase = CombatPhase.ENDED
        state.ended_at = datetime.utcnow()

        # 统计
        survivors = [c for c in state.combatants if c.hp_current > 0]
        casualties = [c for c in state.combatants if c.hp_current <= 0]

        return {
            "duration_seconds": (state.ended_at - state.started_at).total_seconds(),
            "rounds": state.current_round,
            "survivors": len(survivors),
            "casualties": len(casualties),
            "combat_log": state.combat_log
        }

    def apply_status_effect(
        self,
        state: CombatState,
        combatant_id: int,
        effect: str,
        duration: int = 1
    ) -> bool:
        """应用状态效果"""
        combatant = self._get_combatant(state, combatant_id)
        if not combatant:
            return False

        effect_entry = f"{effect}:{duration}"
        if effect_entry not in combatant.status_effects:
            combatant.status_effects.append(effect_entry)

        # 应用效果
        self._apply_effect(combatant, effect)
        return True

    def _apply_effect(self, combatant: Combatant, effect: str):
        """应用具体状态效果"""
        effects = {
            "prone": lambda c: setattr(c, 'armor_class', c.armor_class - 2),
            "stunned": lambda c: setattr(c, 'actions_left', 0),
            "blinded": lambda c: setattr(c, 'attack_bonus', c.attack_bonus - 5),
            "poisoned": lambda c: setattr(c, 'damage_bonus', c.damage_bonus - 2),
        }

        if effect in effects:
            effects[effect](combatant)
```

---

### 1.5 战斗 API (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-020-15 | [ ] 创建 `app/api/combat.py` | [ ] |
| M1-020-16 | [ ] 实现 POST /combat/start | [ ] |
| M1-020-17 | [ ] 实现 POST /combat/action | [ ] |
| M1-020-18 | [ ] 实现 GET /combat/state | [ ] |

```python
# app/api/combat.py
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field
from typing import Optional, List
from sqlmodel import Session
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.db.connection import get_session
from app.services.combat import CombatService

router = APIRouter(prefix="/combat", tags=["战斗"])

# 简化：使用内存存储，实际应使用数据库
combat_states: dict[int, CombatState] = {}

class StartCombatRequest(BaseModel):
    """开始战斗请求"""
    campaign_id: int
    player_character_ids: List[int]
    enemy_ids: List[int]

class ExecuteActionRequest(BaseModel):
    """执行动作请求"""
    combat_state_id: int
    attacker_id: int
    target_id: int
    action_type: str = "attack"
    attack_type: str = "melee"

@router.post("/start")
async def start_combat(
    request: StartCombatRequest,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """开始战斗"""
    combat_service = CombatService()

    state = combat_service.start_combat(
        player_ids=request.player_character_ids,
        enemy_ids=request.enemy_ids
    )

    combat_states[request.campaign_id] = state

    return {
        "combat_id": request.campaign_id,
        "phase": state.phase.value,
        "round": state.current_round,
        "turn_order": [
            {"id": c.character.id, "initiative": c.initiative}
            for c in state.turn_order
        ]
    }

@router.post("/action")
async def execute_action(
    request: ExecuteActionRequest,
    current_user: User = Depends(get_current_active_user)
):
    """执行战斗动作"""
    state = combat_states.get(request.combat_state_id)
    if not state:
        raise HTTPException(status_code=404, detail="战斗不存在")

    combat_service = CombatService()

    if request.action_type == "attack":
        action = combat_service.execute_attack(
            state,
            request.attacker_id,
            request.target_id,
            request.attack_type
        )
    elif request.action_type == "dodge":
        action = combat_service.execute_dodge(state, request.attacker_id)
    elif request.action_type == "defend":
        action = combat_service.execute_defend(state, request.attacker_id)
    else:
        raise HTTPException(status_code=400, detail="不支持的动作类型")

    # 下一轮
    next_combatant = combat_service.next_turn(state)

    return {
        "action": {
            "type": action.action_type.value,
            "hit": action.hit,
            "damage": action.damage,
            "critical": action.critical,
            "narrative": action.narrative
        },
        "next_turn": {
            "character_id": next_combatant.character.id if next_combatant else None
        } if next_combatant else {"combat_ended": True}
    }
```

---

### 1.6 单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-020-19 | [ ] 创建 `tests/test_combat.py` | [ ] |
| M1-020-20 | [ ] 测试战斗开始 | [ ] |
| M1-020-21 | [ ] 测试攻击判定 | [ ] |
| M1-020-22 | [ ] 测试伤害计算 | [ ] |
| M1-020-23 | [ ] 测试回合轮转 | [ ] |

```python
# tests/test_combat.py
import pytest
from app.services.combat import CombatService
from app.core.combat import CombatState, Combatant

class TestCombatService:
    def test_start_combat(self):
        """测试开始战斗"""
        service = CombatService()

        state = service.start_combat(
            player_ids=[1, 2],
            enemy_ids=[101, 102]
        )

        assert state.phase.value == "initiative"
        assert len(state.combatants) == 4
        assert len(state.turn_order) == 4

    def test_attack_critical(self):
        """测试大成功攻击"""
        service = CombatService()
        state = service.start_combat([1], [101])

        # Mock 攻击掷骰为大成功
        with patch('random.randint', side_effect=[3, 6, 6]):
            action = service.execute_attack(state, 1, 101, "melee")

        assert action.critical is True
        assert action.hit is True

    def test_attack_fumble(self):
        """测试大失败攻击"""
        service = CombatService()
        state = service.start_combat([1], [101])

        with patch('random.randint', side_effect=[97, 6]):
            action = service.execute_attack(state, 1, 101, "melee")

        assert action.fumble is True
        assert action.hit is False

    def test_damage_calculation(self):
        """测试伤害计算"""
        service = CombatService()
        state = service.start_combat([1], [101])

        with patch('random.randint', side_effect=[25, 4]):  # 命中 + 4伤害
            action = service.execute_attack(state, 1, 101, "melee")

        assert action.hit is True
        assert action.damage == 4  # 1d6 = 4

    def test_combat_end(self):
        """测试战斗结束"""
        service = CombatService()
        state = service.start_combat([1], [101])

        result = service.end_combat(state)

        assert result["rounds"] == 1
        assert state.phase.value == "ended"
```

---

## 验收标准

- [ ] 战斗开始/结束流程完整
- [ ] 攻击检定正确（大成功/大失败）
- [ ] 伤害计算正确
- [ ] 先攻顺序正确
- [ ] 闪避/格挡可用
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/combat.py` | 创建 | 战斗数据模型 |
| `app/services/combat.py` | 创建 | 战斗引擎 |
| `app/api/combat.py` | 创建 | 战斗 API |
| `tests/test_combat.py` | 创建 | 单元测试 |

---

## CoC 7e 战斗规则

| 攻击结果 | 条件 | 效果 |
|----------|------|------|
| 大成功 | 1-5% | 伤害 x2 |
| 命中 | 攻击值 >= AC | 正常伤害 |
| 失误 | 攻击值 < AC | 无伤害 |
| 大失败 | 96-100% | 可能自伤 |
