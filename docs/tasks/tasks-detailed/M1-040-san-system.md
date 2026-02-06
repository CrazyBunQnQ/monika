# M1-040: SAN 值系统

**任务类型**: backend
**预估工时**: 3.5h
**依赖**: M1-003, M1-010
**状态**: [ ]

---

## 子任务拆解

### 1.1 SAN 数据模型 (35min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-040-01 | [ ] 创建 `app/core/san.py` | [ ] |
| M1-040-02 | [ ] 定义 `SanEventType` 枚举 | [ ] |
| M1-040-03 | [ ] 定义 `SanState` 数据类 | [ ] |
| M1-040-04 | [ ] 定义 `SanCheckResult` 数据类 | [ ] |

```python
# app/core/san.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from datetime import datetime

class SanEventType(str, Enum):
    """SAN 事件类型"""
    SEE_MONSTER = "see_monster"           # 目睹怪物
    SEE_CORPSE = "see_corpse"             # 目睹尸体
    HEAR_TERROR = "hear_terror"           # 听到恐怖声音
    READ_FORBIDDEN = "read_forbidden"     # 阅读禁忌文献
    WITNESS_DEATH = "witness_death"      # 目击死亡
    COSMIC_REALITY = "cosmic_reality"     # 认知宇宙真相
    TORTURE = "torture"                  # 遭受折磨
    OTHER = "other"

@dataclass
class SanThreshold:
    """SAN 阈值"""
    success: int = 0       # 成功失去
    failure: int = 0       # 失败失去
    critical: int = 0      # 大成功失去
    fumble: int = 0        # 大失败失去

@dataclass
class SanState:
    """SAN 状态"""
    character_id: int

    # 当前值
    current: int = 99
    maximum: int = 99

    # 累积损失
    lost_total: int = 0
    lost_session: int = 0

    # 恢复
    recovery_rate: int = 1  # 每小时恢复
    last_recovery: Optional[datetime] = None

    # 疯狂阈值
    impending_breakdown: int = 5  # 即将崩溃阈值
    temporary_insanity: int = 0   # 临时疯狂
    indefinite_insanity: bool = False

    # 历史
    check_history: List[Dict] = field(default_factory=list)

    def is_broken(self) -> bool:
        """是否已经崩溃"""
        return self.current <= 0 or self.indefinite_insanity

    def can_recover(self) -> bool:
        """是否可以恢复"""
        return not self.is_broken() and self.current < self.maximum

@dataclass
class SanCheckResult:
    """SAN 检定结果"""
    success_level: str  # success / failure / critical / fumble
    san_loss: int
    roll_value: int
    target_value: int

    # 状态变化
    san_before: int
    san_after: int

    # 叙述
    narrative: str

    # 疯狂检测
    triggers_temporary_insanity: bool = False
    triggers_indefinite_insanity: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return {
            "success_level": self.success_level,
            "san_loss": self.san_loss,
            "roll_value": self.roll_value,
            "target_value": self.target_value,
            "san_before": self.san_before,
            "san_after": self.san_after,
            "narrative": self.narrative,
            "temporary_insanity": self.triggers_temporary_insanity,
            "indefinite_insanity": self.triggers_indefinite_insanity
        }
```

---

### 1.2 SAN 检定引擎 (50min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-040-05 | [ ] 创建 `app/services/san.py` | [ ] |
| M1-040-06 | [ ] 实现 `calculate_san_loss()` | [ ] |
| M1-040-07 | [ ] 实现 `execute_san_check()` | [ ] |
| M1-040-08 | [ ] 实现 `check_insanity()` | [ ] |

```python
# app/services/san.py
import random
from typing import Optional, Tuple
from datetime import datetime, timedelta
from app.core.san import (
    SanState, SanCheckResult, SanEventType, SanThreshold
)

class SanService:
    """SAN 值服务"""

    # 事件基础损失表
    EVENT_LOSS_TABLE = {
        SanEventType.SEE_MONSTER: SanThreshold(success=1, failure=1d6, critical=0, fumble=1d10),
        SanEventType.SEE_CORPSE: SanThreshold(success=0, failure=1, critical=1d4, fumble=1d6),
        SanEventType.HEAR_TERROR: SanThreshold(success=0, failure=1, critical=0, fumble=1d4),
        SanEventType.READ_FORBIDDEN: SanThreshold(success=1, failure=1d6, critical=1d10, fumble=2d10),
        SanEventType.WITNESS_DEATH: SanThreshold(success=1, failure=1d6, critical=0, fumble=1d10),
        SanEventType.COSMIC_REALITY: SanThreshold(success=2, failure=2d10, critical=1d100, fumble=99),
        SanEventType.TORTURE: SanThreshold(success=1d4, failure=2d10, critical=99, fumble=99),
        SanEventType.OTHER: SanThreshold(success=1, failure=1, critical=1, fumble=1),
    }

    def calculate_san_loss(
        self,
        event_type: SanEventType,
        roll: int,
        target: int
    ) -> Tuple[int, str]:
        """计算 SAN 损失

        规则:
        - 大成功 (1/5): 失去 1/5 基础值
        - 成功 (≤目标值): 失去基础值
        - 失败 (>目标值): 失去 1d6 基础值
        - 大失败 (96-100): 失去 1d10 基础值
        """
        thresholds = self.EVENT_LOSS_TABLE.get(event_type, SanThreshold())
        base_loss = thresholds.success

        # 判定成功等级
        if roll <= 5:  # 大成功
            loss = thresholds.critical if thresholds.critical else base_loss // 5
            result = "critical"
        elif roll <= target:  # 成功
            loss = base_loss
            result = "success"
        elif roll >= 96:  # 大失败
            loss = thresholds.fumble
            result = "fumble"
        else:  # 失败
            loss = thresholds.failure if thresholds.failure else base_loss
            result = "failure"

        return loss, result

    def execute_san_check(
        self,
        san_state: SanState,
        event_type: SanEventType,
        modifier: int = 0
    ) -> SanCheckResult:
        """执行 SAN 检定"""
        # 基础成功率 = POW/5
        base_target = san_state.maximum // 5
        target_value = max(1, base_target + modifier)

        # 掷骰
        roll = random.randint(1, 100)

        # 计算损失
        san_loss, result = self.calculate_san_loss(event_type, roll, target_value)

        # 记录之前
        san_before = san_state.current

        # 应用损失
        san_state.current = max(0, san_state.current - san_loss)
        san_state.lost_total += san_loss
        san_state.lost_session += san_loss

        # 检查疯狂
        triggers_temp, triggers_indef = self._check_insanity(
            san_state, san_loss, result
        )

        san_state.check_history.append({
            "timestamp": datetime.now().isoformat(),
            "event": event_type.value,
            "roll": roll,
            "target": target_value,
            "loss": san_loss,
            "result": result
        })

        # 生成叙述
        narrative = self._generate_narrative(
            event_type, result, roll, target_value, san_loss
        )

        return SanCheckResult(
            success_level=result,
            san_loss=san_loss,
            roll_value=roll,
            target_value=target_value,
            san_before=san_before,
            san_after=san_state.current,
            narrative=narrative,
            triggers_temporary_insanity=triggers_temp,
            triggers_indefinite_insanity=triggers_indef
        )

    def _check_insanity(
        self,
        san_state: SanState,
        loss: int,
        result: str
    ) -> Tuple[bool, bool]:
        """检查是否触发疯狂"""
        # 一次损失 >= 5 或 总损失 >= 总 SAN 的 1/5
        total_loss_threshold = san_state.maximum // 5

        # 临时疯狂条件
        temp_trigger = loss >= 5 or (
            san_state.lost_session >= total_loss_threshold
        )

        # 不定疯狂条件
        inde_trigger = san_state.current <= 0

        return temp_trigger, inde_trigger

    def _generate_narrative(
        self,
        event_type: SanEventType,
        result: str,
        roll: int,
        target: int,
        loss: int
    ) -> str:
        """生成叙述文本"""
        event_names = {
            SanEventType.SEE_MONSTER: "目睹怪物",
            SanEventType.SEE_CORPSE: "目睹尸体",
            SanEventType.HEAR_TERROR: "听到恐怖",
            SanEventType.READ_FORBIDDEN: "阅读禁忌",
            SanEventType.WITNESS_DEATH: "目击死亡",
            SanEventType.COSMIC_REALITY: "认知真相",
            SanEventType.TORTURE: "遭受折磨",
        }

        event_name = event_names.get(event_type, event_type.value)

        result_text = {
            "critical": "你竟然保持镇定！",
            "success": "你勉强维持住了心神。",
            "failure": "你的意志受到了冲击！",
            "fumble": "你的精神崩溃了！"
        }

        return f"{event_name}: {roll}/{target} → 失去 {loss} 点 SAN - {result_text.get(result, '')}"
```

---

### 1.3 SAN 恢复机制 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-040-09 | [ ] 实现 `recover_san()` | [ ] |
| M1-040-10 | [ ] 实现 `process_recovery()` | [ ] |
| M1-040-11 | [ ] 实现 `get_san_status()` | [ ] |

```python
class SanService:
    # ...

    def recover_san(
        self,
        san_state: SanState,
        amount: int,
        reason: str = "rest"
    ) -> int:
        """恢复 SAN 值"""
        if san_state.is_broken():
            return 0  # 崩溃中无法恢复

        max_recoverable = san_state.maximum - san_state.lost_total
        actual = min(amount, max_recoverable)

        san_state.current += actual
        san_state.lost_total -= actual
        san_state.lost_session = max(0, san_state.lost_session - actual)

        # 记录恢复
        san_state.check_history.append({
            "timestamp": datetime.now().isoformat(),
            "action": "recover",
            "amount": actual,
            "reason": reason
        })

        return actual

    def hourly_recovery(self, san_state: SanState) -> int:
        """每小时自动恢复（1点/成功故事）"""
        if san_state.is_broken():
            return 0

        # 成功恢复：1d3
        recovery = random.randint(1, 3)
        return self.recover_san(san_state, recovery, "hourly_recovery")

    def dream_recovery(self, san_state: SanState) -> int:
        """长梦恢复"""
        if san_state.is_broken():
            return 0

        # 1d6 + 心理治疗修正
        recovery = random.randint(1, 6)
        return self.recover_san(san_state, recovery, "dream")

    def therapy_recovery(
        self,
        san_state: SanState,
        therapy_skill: int,
        session_hours: int = 1
    ) -> int:
        """心理治疗恢复"""
        if san_state.is_broken():
            return 0

        # 治疗检定
        roll = random.randint(1, 100)

        if roll <= therapy_skill:
            # 成功：恢复 1d6
            recovery = random.randint(1, 6) * session_hours
            return self.recover_san(san_state, recovery, "therapy")
        else:
            # 失败：无恢复
            return 0
```

---

### 1.4 SAN API (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-040-12 | [ ] 创建 `app/api/san.py` | [ ] |
| M1-040-13 | [ ] 实现 POST /san/check | [ ] |
| M1-040-14 | [ ] 实现 POST /san/recover | [ ] |
| M1-040-15 | [ ] 实现 GET /san/status | [ ] |

```python
# app/api/san.py
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field
from typing import Optional
from sqlmodel import Session
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.db.connection import get_session
from app.services.san import SanService
from app.core.san import SanEventType

router = APIRouter(prefix="/san", tags=["SAN值"])

class SanCheckRequest(BaseModel):
    """SAN 检定请求"""
    character_id: int
    event_type: str
    modifier: int = 0

class SanRecoverRequest(BaseModel):
    """SAN 恢复请求"""
    character_id: int
    amount: int
    reason: str = "rest"

@router.post("/check")
async def san_check(
    request: SanCheckRequest,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """执行 SAN 检定"""
    try:
        event_type = SanEventType(request.event_type)
    except ValueError:
        raise HTTPException(status_code=400, detail="无效的事件类型")

    # 获取 SAN 状态
    san_state = _get_san_state(session, request.character_id)

    service = SanService()
    result = service.execute_san_check(san_state, event_type, request.modifier)

    # 保存状态
    _save_san_state(session, san_state)

    return result.to_dict()

@router.post("/recover")
async def san_recover(
    request: SanRecoverRequest,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """恢复 SAN 值"""
    san_state = _get_san_state(session, request.character_id)

    service = SanService()
    recovered = service.recover_san(
        san_state,
        request.amount,
        request.reason
    )

    _save_san_state(session, san_state)

    return {
        "recovered": recovered,
        "current": san_state.current,
        "lost_total": san_state.lost_total
    }

@router.get("/status/{character_id}")
async def get_san_status(
    character_id: int,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """获取 SAN 状态"""
    san_state = _get_san_state(session, character_id)

    return {
        "current": san_state.current,
        "maximum": san_state.maximum,
        "lost_total": san_state.lost_total,
        "lost_session": san_state.lost_session,
        "is_broken": san_state.is_broken(),
        "temporary_insanity": san_state.temporary_insanity > 0,
        "indefinite_insanity": san_state.indefinite_insanity,
        "recovery_rate": san_state.recovery_rate
    }
```

---

### 1.5 单元测试 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-040-16 | [ ] 创建 `tests/test_san.py` | [ ] |
| M1-040-17 | [ ] 测试 SAN 损失计算 | [ ] |
| M1-040-18 | [ ] 测试疯狂检测 | [ ] |
| M1-040-19 | [ ] 测试 SAN 恢复 | [ ] |

```python
# tests/test_san.py
import pytest
from app.services.san import SanService
from app.core.san import SanState, SanEventType

class TestSanService:
    def setup_method(self):
        self.service = SanService()
        self.san_state = SanState(
            character_id=1,
            current=99,
            maximum=99
        )

    def test_san_check_success(self):
        """测试 SAN 检定成功"""
        with patch('random.randint', return_value=50):
            result = self.service.execute_san_check(
                self.san_state,
                SanEventType.SEE_MONSTER,
                modifier=0
            )

        assert result.success_level == "success"
        assert result.san_loss >= 0
        assert result.san_after < result.san_before

    def test_san_check_critical(self):
        """测试 SAN 大成功"""
        with patch('random.randint', return_value=5):
            result = self.service.execute_san_check(
                self.san_state,
                SanEventType.SEE_MONSTER,
                modifier=0
            )

        assert result.success_level == "critical"

    def test_san_check_fumble(self):
        """测试 SAN 大失败"""
        with patch('random.randint', return_value=97):
            result = self.service.execute_san_check(
                self.san_state,
                SanEventType.SEE_MONSTER,
                modifier=0
            )

        assert result.success_level == "fumble"
        assert result.san_loss >= 1

    def test_san_recovery(self):
        """测试 SAN 恢复"""
        self.san_state.current = 90
        recovered = self.service.recover_san(
            self.san_state,
            5,
            "therapy"
        )

        assert recovered == 5
        assert self.san_state.current == 95

    def test_insanity_trigger(self):
        """测试疯狂触发"""
        self.san_state.current = 6

        # 大损失触发临时疯狂
        with patch('random.randint', side_effect=[60, 6]):  # 失败 + 6损失
            result = self.service.execute_san_check(
                self.san_state,
                SanEventType.COSMIC_REALITY,
                modifier=0
            )

        assert result.triggers_temporary_insanity is True

    def test_zero_san_triggers_indefinite(self):
        """测试 SAN 归零触发不定疯狂"""
        self.san_state.current = 1

        with patch('random.randint', return_value=99):  # 大失败
            result = self.service.execute_san_check(
                self.san_state,
                SanEventType.COSMIC_REALITY
            )

        assert result.triggers_indefinite_insanity is True
```

---

## 验收标准

- [ ] SAN 检定正确计算损失
- [ ] 大成功/大失败判定正确
- [ ] 疯狂检测正确触发
- [ ] SAN 恢复机制完整
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/san.py` | 创建 | SAN 数据模型 |
| `app/services/san.py` | 创建 | SAN 服务 |
| `app/api/san.py` | 创建 | SAN API |
| `tests/test_san.py` | 创建 | 单元测试 |
