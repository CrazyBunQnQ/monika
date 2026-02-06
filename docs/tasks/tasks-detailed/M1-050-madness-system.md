# M1-050: 疯狂机制

**任务类型**: backend
**预估工时**: 3h
**依赖**: M1-040
**状态**: [ ]

---

## 子任务拆解

### 1.1 疯狂数据模型 (35min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-050-01 | [ ] 创建 `app/core/madness.py` | [ ] |
| M1-050-02 | [ ] 定义 `MadnessType` 枚举 | [ ] |
| M1-050-03 | [ ] 定义 `MadnessPhase` 枚举 | [ ] |
| M1-050-04 | [ ] 定义 `MadnessState` | [ ] |

```python
# app/core/madness.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from datetime import datetime

class MadnessType(str, Enum):
    """疯狂类型"""
    TEMPORARY = "temporary"     # 临时疯狂（1d10 小时）
    INDEFINITE = "indefinite"   # 不定疯狂（治愈疾病方式治疗）
    CHRONIC = "chronic"         # 慢性疯狂（长期症状）

class MadnessPhase(str, Enum):
    """疯狂阶段"""
    NONE = "none"
    ONSET = "onset"           # 发作期
    ACTIVE = "active"         # 活跃期
    SUBSIDING = "subsiding"   # 消退期
    RECOVERED = "recovered"   # 已恢复

class DelusionType(str, Enum):
    """妄想类型"""
    PERSECUTION = "persecution"   # 被害妄想
    GRANDIOSE = "grandiose"       # 夸大妄想
    SOMATIC = "somatic"           # 躯体妄想
    JEALOUS = "jealous"           # 嫉妒妄想
    EROTIC = "erotic"             # 色情妄想

@dataclass
class MadnessSymptom:
    """疯狂症状"""
    id: int
    type: str                     # "delusion", "phobia", "mania", "panic"
    description: str
    severity: int = 1             # 1-5
    trigger: Optional[str] = None # 触发条件
    effect: str = ""              # 效果描述

@dataclass
class MadnessState:
    """疯狂状态"""
    character_id: int

    # 基本信息
    madness_type: Optional[MadnessType] = None
    phase: MadnessPhase = MadnessPhase.NONE

    # 时间
    onset_at: Optional[datetime] = None
    active_until: Optional[datetime] = None
    duration_hours: int = 0

    # 症状
    symptoms: List[MadnessSymptom] = field(default_factory=list)

    # 行动限制
    cannot_speak: bool = False
    cannot_move: bool = False
    cannot_combat: bool = False
    cannot_reason: bool = False

    # 叙述
    current_episode_narrative: str = ""

    # 恢复记录
    recovery_history: List[Dict] = field(default_factory=list)

    def is_active(self) -> bool:
        """是否处于疯狂状态"""
        return self.phase in [MadnessPhase.ONSET, MadnessPhase.ACTIVE]

    def get_action_restrictions(self) -> List[str]:
        """获取行动限制"""
        restrictions = []
        if self.cannot_speak:
            restrictions.append("无法说话")
        if self.cannot_move:
            restrictions.append("无法移动")
        if self.cannot_combat:
            restrictions.append("无法战斗")
        if self.cannot_reason:
            restrictions.append("无法理性思考")
        return restrictions
```

---

### 1.2 疯狂生成器 (45min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-050-05 | [ ] 创建 `app/services/madness.py` | [ ] |
| M1-050-06 | [ ] 实现 `trigger_temporary_madness()` | [ ] |
| M1-050-07 | [ ] 实现 `trigger_indefinite_madness()` | [ ] |
| M1-050-08 | [ ] 实现 `generate_symptoms()` | [ ] |

```python
# app/services/madness.py
import random
from typing import Optional, List, Tuple
from datetime import datetime, timedelta
from app.core.madness import (
    MadnessState, MadnessType, MadnessPhase,
    MadnessSymptom, DelusionType
)

class MadnessService:
    """疯狂服务"""

    # 临时疯狂症状表
    TEMPORARY_SYMPTOMS = [
        # 恐惧症
        ("phobia", "黑暗恐惧症", "害怕黑暗", 2, "黑暗环境"),
        ("phobia", "幽闭恐惧症", "害怕封闭空间", 2, "封闭空间"),
        ("phobia", "血液恐惧症", "害怕血液", 3, "看到血液"),
        ("phobia", "深海恐惧症", "害怕深水", 2, "深水环境"),
        ("phobia", "蜘蛛恐惧症", "害怕蜘蛛", 3, "看到蜘蛛"),
        # 躁狂症
        ("mania", "纵火癖", "想要纵火", 3, "看到火焰"),
        ("mania", "盗窃癖", "无法控制偷窃冲动", 2, "看到有价值物品"),
        ("mania", "杀戮冲动", "强烈杀戮欲望", 4, "看到武器"),
        # 妄想
        ("delusion", "被迫害妄想", "认为有人要杀他", 3, "任何陌生人"),
        ("delusion", "身份妄想", "认为自己是他人", 2, "被称呼真名"),
        ("delusion", "躯体妄想", "身体某个部位消失", 2, "被问及身体状况"),
        # 恐慌反应
        ("panic", "惊恐发作", "突然恐惧", 2, "任何压力"),
        ("panic", "木僵状态", "僵硬无法行动", 3, "特定触发词"),
    ]

    # 不定疯狂表
    INDEFINITE_SYMPTOMS = [
        ("delusion", "慢性妄想", "持续的被迫害妄想", 4),
        ("delusion", "人格分裂", "多重人格", 5),
        ("phobia", "严重恐惧症", "无法控制的恐惧", 4),
        ("mania", "慢性躁狂", "持续的危险行为", 4),
        ("psychosis", "幻觉", "看到不存在的事物", 5),
        ("psychosis", "幻听", "听到声音", 5),
    ]

    # 疯狂叙述
    MADNESS_NARRATIVES = {
        "temporary": [
            "你的视野开始模糊，恐惧吞噬了你的心智...",
            "你的双手开始颤抖，无法控制地尖叫...",
            "你的意识开始模糊，周围的声音变得扭曲...",
            "心脏剧烈跳动，你感到窒息般的恐惧...",
            "脑海中充斥着诡异的声音，你无法思考...",
        ],
        "indefinite": [
            "你的心智已经支离破碎...",
            "现实的界限在你眼中变得模糊...",
            "你已经分不清现实与幻想...",
            "你的意识陷入了永无止境的噩梦...",
        ]
    }

    def trigger_temporary_madness(
        self,
        state: MadnessState,
        trigger_event: str = ""
    ) -> MadnessState:
        """触发临时疯狂（1d10 小时）"""
        state.madness_type = MadnessType.TEMPORARY
        state.phase = MadnessPhase.ONSET
        state.onset_at = datetime.now()

        # 持续时间 1-10 小时
        state.duration_hours = random.randint(1, 10)

        # 随机选择症状
        symptoms = self._generate_symptoms(1, "temporary")
        state.symptoms = symptoms

        # 随机生成行动限制
        self._apply_action_restrictions(state)

        # 生成叙述
        narrative = random.choice(self.MADNESS_NARRATIVES["temporary"])
        if trigger_event:
            narrative += f" 触发事件: {trigger_event}"
        state.current_episode_narrative = narrative

        return state

    def trigger_indefinite_madness(
        self,
        state: MadnessState,
        cause: str = ""
    ) -> MadnessState:
        """触发不定疯狂"""
        state.madness_type = MadnessType.INDEFINITE
        state.phase = MadnessPhase.ACTIVE
        state.onset_at = datetime.now()

        # 生成 2-3 个严重症状
        symptoms = self._generate_symptoms(2, "indefinite")
        state.symptoms = symptoms

        # 更严重的行动限制
        state.cannot_reason = True
        state.cannot_combat = True

        # 生成叙述
        narrative = random.choice(self.MADNESS_NARRATIVES["indefinite"])
        if cause:
            narrative += f" 病因: {cause}"
        state.current_episode_narrative = narrative

        return state

    def _generate_symptoms(
        self,
        count: int,
        severity_type: str
    ) -> List[MadnessSymptom]:
        """生成症状"""
        if severity_type == "temporary":
            pool = self.TEMPORARY_SYMPTOMS
        else:
            pool = self.INDEFINITE_SYMPTOMS

        selected = random.sample(pool, min(count, len(pool)))

        symptoms = []
        for i, (s_type, name, desc, severity, trigger) in enumerate(selected):
            symptoms.append(MadnessSymptom(
                id=i,
                type=s_type,
                description=f"{name}: {desc}",
                severity=severity,
                trigger=trigger,
                effect=self._get_symptom_effect(s_type)
            ))

        return symptoms

    def _get_symptom_effect(self, symptom_type: str) -> str:
        """获取症状效果"""
        effects = {
            "phobia": "必须进行意志检定，否则逃离现场",
            "mania": "必须进行意志检定，否则执行冲动行为",
            "delusion": "行为变得怪异，可能说出奇怪的话",
            "panic": "进入木僵状态 1-3 轮",
            "psychosis": "可能产生幻觉，影响判断",
        }
        return effects.get(symptom_type, "行为异常")

    def _apply_action_restrictions(self, state: MadnessState):
        """应用行动限制"""
        restrictions = random.sample([
            ("cannot_speak", "无法说话"),
            ("cannot_move", "无法移动"),
            ("cannot_combat", "无法战斗"),
        ], k=random.randint(1, 2))

        for attr, desc in restrictions:
            setattr(state, attr, True)
```

---

### 1.3 疯狂处理 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-050-09 | [ ] 实现 `process_madness_phase()` | [ ] |
| M1-050-10 | [ ] 实现 `attempt_recovery()` | [ ] |
| M1-050-11 | [ ] 实现 `get_action_penalty()` | [ ] |

```python
class MadnessService:
    # ...

    def process_madness_phase(
        self,
        state: MadnessState
    ) -> Tuple[MadnessPhase, str]:
        """处理疯狂阶段变化"""
        if state.phase == MadnessPhase.NONE:
            return MadnessPhase.NONE, "正常"

        if state.phase == MadnessPhase.ONSET:
            # 发作期结束，进入活跃期
            state.phase = MadnessPhase.ACTIVE
            return MadnessPhase.ACTIVE, f"疯狂发作，持续 {state.duration_hours} 小时"

        if state.phase == MadnessPhase.ACTIVE:
            if state.madness_type == MadnessType.TEMPORARY:
                # 检查时间是否结束
                if state.duration_hours <= 0:
                    state.phase = MadnessPhase.RECOVERED
                    return MadnessPhase.RECOVERED, "临时疯狂已恢复"

            return MadnessPhase.ACTIVE, "仍处于疯狂状态"

        if state.phase == MadnessPhase.RECOVERED:
            return MadnessPhase.RECOVERED, "已恢复健康"

        return state.phase, ""

    def attempt_recovery(
        self,
        state: MadnessState,
        medicine_skill: int,
        care_skill: int
    ) -> Tuple[bool, str]:
        """尝试治愈疯狂

        临时疯狂：1d6 小时后自动恢复
        不定疯狂：需要 POW x5 成功治愈检定 + 心理治疗
        """
        if state.madness_type == MadnessType.TEMPORARY:
            # 1d6 小时后恢复
            hours_remaining = random.randint(1, 6)
            state.duration_hours = max(0, state.duration_hours - hours_remaining)

            if state.duration_hours <= 0:
                self._recover(state)
                return True, f"临时疯狂恢复！"

            return False, f"还需要 {state.duration_hours} 小时"

        if state.madness_type == MadnessType.INDEFINITE:
            # 治愈检定
            roll = random.randint(1, 100)
            target = state.character_id  # 使用 POW 作为目标（简化）

            if roll <= target:
                # 成功：1d6 个月后恢复
                months = random.randint(1, 6)
                state.recovery_history.append({
                    "date": datetime.now().isoformat(),
                    "method": "medicine_check",
                    "result": "success",
                    "months_remaining": months
                })
                return True, f"治愈成功！还需要 {months} 个月治疗"

            return False, "治愈失败"

        return False, "当前无疯狂状态"

    def _recover(self, state: MadnessState):
        """恢复健康"""
        state.phase = MadnessPhase.RECOVERED
        state.cannot_speak = False
        state.cannot_move = False
        state.cannot_combat = False
        state.cannot_reason = False
        state.madness_type = None

        state.recovery_history.append({
            "date": datetime.now().isoformat(),
            "method": "time",
            "result": "full_recovery"
        })

    def get_action_penalty(
        self,
        state: MadnessState
    ) -> Dict[str, int]:
        """获取疯狂导致的检定惩罚"""
        if not state.is_active():
            return {"penalty": 0}

        total_penalty = 0

        for symptom in state.symptoms:
            total_penalty += symptom.severity

        # 根据类型调整
        if state.madness_type == MadnessType.INDEFINITE:
            total_penalty *= 2

        return {
            "penalty": min(total_penalty, 20),  # 最大惩罚 20
            "symptom_count": len(state.symptoms),
            "restrictions": state.get_action_restrictions()
        }
```

---

### 1.4 疯狂 API (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-050-12 | [ ] 创建 `app/api/madness.py` | [ ] |
| M1-050-13 | [ ] 实现 POST /madness/trigger | [ ] |
| M1-050-14 | [ ] 实现 POST /madness/recover | [ ] |
| M1-050-15 | [ ] 实现 GET /madness/status | [ ] |

```python
# app/api/madness.py
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from typing import Optional, Dict
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.services.madness import MadnessService

router = APIRouter(prefix="/madness", tags=["疯狂"])

class TriggerMadnessRequest(BaseModel):
    """触发疯狂请求"""
    character_id: int
    madness_type: str  # "temporary" / "indefinite"
    trigger_event: Optional[str] = None

class RecoveryAttemptRequest(BaseModel):
    """恢复尝试请求"""
    character_id: int
    medicine_skill: int
    care_skill: int

@router.post("/trigger")
async def trigger_madness(
    request: TriggerMadnessRequest,
    current_user: User = Depends(get_current_active_user)
):
    """触发疯狂状态"""
    from app.core.madness import MadnessType

    try:
        m_type = MadnessType(request.madness_type)
    except ValueError:
        raise HTTPException(status_code=400, detail="无效的疯狂类型")

    state = _get_madness_state(request.character_id)
    service = MadnessService()

    if m_type == MadnessType.TEMPORARY:
        service.trigger_temporary_madness(state, request.trigger_event)
    else:
        service.trigger_indefinite_madness(state, request.trigger_event)

    _save_madness_state(state)

    return {
        "madness_type": state.madness_type.value,
        "phase": state.phase.value,
        "duration_hours": state.duration_hours,
        "symptoms": [
            {"description": s.description, "effect": s.effect}
            for s in state.symptoms
        ],
        "narrative": state.current_episode_narrative,
        "restrictions": state.get_action_restrictions()
    }

@router.post("/recover")
async def attempt_recovery(
    request: RecoveryAttemptRequest,
    current_user: User = Depends(get_current_active_user)
):
    """尝试恢复"""
    state = _get_madness_state(request.character_id)

    if not state.is_active():
        return {"message": "当前无疯狂状态", "recovered": False}

    service = MadnessService()
    success, message = service.attempt_recovery(
        state,
        request.medicine_skill,
        request.care_skill
    )

    _save_madness_state(state)

    return {
        "recovered": success,
        "message": message,
        "current_phase": state.phase.value
    }

@router.get("/status/{character_id}")
async def get_madness_status(
    character_id: int,
    current_user: User = Depends(get_current_active_user)
):
    """获取疯狂状态"""
    state = _get_madness_state(character_id)

    service = MadnessService()
    penalties = service.get_action_penalty(state)

    return {
        "is_active": state.is_active(),
        "type": state.madness_type.value if state.madness_type else None,
        "phase": state.phase.value,
        "duration_hours": state.duration_hours,
        "symptoms": [
            {"description": s.description, "severity": s.severity}
            for s in state.symptoms
        ],
        "restrictions": state.get_action_restrictions(),
        "penalties": penalties
    }
```

---

### 1.5 单元测试 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-050-16 | [ ] 创建 `tests/test_madness.py` | [ ] |
| M1-050-17 | [ ] 测试临时疯狂触发 | [ ] |
| M1-050-18 | [ ] 测试不定疯狂触发 | [ ] |
| M1-050-19 | [ ] 测试恢复机制 | [ ] |

```python
# tests/test_madness.py
import pytest
from app.services.madness import MadnessService
from app.core.madness import MadnessState, MadnessType, MadnessPhase

class TestMadnessService:
    def setup_method(self):
        self.service = MadnessService()
        self.state = MadnessState(character_id=1)

    def test_trigger_temporary_madness(self):
        """测试临时疯狂触发"""
        result = self.service.trigger_temporary_madness(
            self.state,
            "目睹怪物"
        )

        assert result.madness_type == MadnessType.TEMPORARY
        assert result.phase == MadnessPhase.ONSET
        assert len(result.symptoms) == 1
        assert 1 <= result.duration_hours <= 10

    def test_trigger_indefinite_madness(self):
        """测试不定疯狂触发"""
        result = self.service.trigger_indefinite_madness(
            self.state,
            "长期接触神话生物"
        )

        assert result.madness_type == MadnessType.INDEFINITE
        assert result.phase == MadnessPhase.ACTIVE
        assert len(result.symptoms) == 2
        assert result.cannot_reason is True

    def test_action_restrictions(self):
        """测试行动限制"""
        self.service.trigger_temporary_madness(self.state)

        restrictions = self.state.get_action_restrictions()
        assert len(restrictions) > 0

    def test_recovery_from_temporary(self):
        """测试临时疯狂恢复"""
        self.service.trigger_temporary_madness(self.state)
        self.state.duration_hours = 1

        success, message = self.service.attempt_recovery(
            self.state,
            medicine_skill=50,
            care_skill=30
        )

        assert "还需要" in message or "恢复" in message

    def test_action_penalty(self):
        """测试行动惩罚"""
        self.service.trigger_temporary_madness(self.state)

        penalty = self.service.get_action_penalty(self.state)

        assert penalty["penalty"] > 0
        assert len(penalty["restrictions"]) > 0
```

---

## 验收标准

- [ ] 临时疯狂生成正确症状
- [ ] 不定疯狂生成严重症状
- [ ] 行动限制正确应用
- [ ] 恢复机制符合 CoC 规则
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/madness.py` | 创建 | 疯狂数据模型 |
| `app/services/madness.py` | 创建 | 疯狂服务 |
| `app/api/madness.py` | 创建 | 疯狂 API |
| `tests/test_madness.py` | 创建 | 单元测试 |

---

## CoC 7e 疯狂规则

| 疯狂类型 | 持续时间 | 触发条件 | 恢复方式 |
|----------|----------|----------|----------|
| 临时疯狂 | 1d10 小时 | 单次损失 >=5 | 时间 |
| 不定疯狂 | 1d6 月 | SAN 归零 | POW x5 治愈检定 + 治疗 |
