# M1-010: 检定系统 API

**任务ID**: M1-010
**标题**: 检定系统 API
**类型**: backend (后端开发)
**预估工时**: 6h
**依赖**: M1-057

---

## 任务描述

实现检定系统的 API，包括技能检定、属性检定、难度调整、修正值等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-010-01 | 设计检定请求格式 | Request Schema | 30min |
| M1-010-02 | 设计检定响应格式 | Response Schema | 30min |
| M1-010-03 | 实现检定计算服务 | 核心检定逻辑 | 45min |
| M1-010-04 | 实现 POST /game/check | 检定端点 | 30min |
| M1-010-05 | 实现难度调整逻辑 | 难度计算 | 30min |
| M1-010-06 | 实现修正值处理 | 正负修正 | 20min |
| M1-010-07 | 实现推骰检查 | 检查是否可推 | 20min |
| M1-010-08 | 编写检定测试 | 单元测试 | 40min |
| M1-010-09 | 添加 OpenAPI 文档 | API 文档 | 15min |

---

## 检定请求格式

```typescript
interface CheckRequest {
  // 必填
  character_id: string;
  /** 角色ID */

  skill: string;
  /** 技能名称或属性名称 */

  // 可选
  difficulty?: 'regular' | 'hard' | 'extreme';
  /** 难度级别 */

  modifier?: number;
  /** 修正值 (-99 到 +99) */

  bonus_dice?: number;
  /** 奖励骰数量 (0-3) */

  penalty_dice?: number;
  /** 惩罚骰数量 (0-3) */

  luck?: number;
  /** 花幸运的点数 */

  reason?: string;
  /** 检定原因描述 */
}
```

---

## 检定响应格式

```typescript
interface CheckResponse {
  // 基本信息
  check_id: string;
  character_id: string;
  skill: string;
  skill_value: number;

  // 掷骰结果
  roll_result: number;
  /** 最终掷骰结果 */

  raw_rolls: number[];
  /** 原始掷骰结果 (多个骰子) */

  // 难度信息
  difficulty: string;
  difficulty_modifier: number;
  /** 难度修正值 */

  // 成功等级
  success_level: SuccessLevel;
  /** 成功等级 */

  passed: boolean;
  /** 是否通过 */

  // 修正信息
  modifier: number;
  /** 修正值 */

  bonus_dice_used?: number;
  /** 使用的奖励骰数 */

  penalty_dice_used?: number;
  /** 使用的惩罚骰数 */

  // 奖励骰详情
  bonus_rolls?: {
    count: number;
    rolls: number[];
    chosen: number;
  };

  // 惩罚骰详情
  penalty_rolls?: {
    count: number;
    rolls: number[];
    chosen: number;
  };

  // 特殊结果
  critical?: boolean;
  /** 是否大成功 */

  fumble?: boolean;
  /** 是否大失败 */

  // 描述
  description: string;
  /** 结果描述 */

  can_push?: boolean;
  /** 是否可以推骰 */

  push_cost?: number;
  /** 推骰消耗 (SAN/Luck) */

  // 元数据
  timestamp: string;
  /** 时间戳 */
}

type SuccessLevel =
  | 'critical'      // 大成功 (1)
  | 'extreme'       // 极难成功
  | 'hard'          // 困难成功
  | 'regular'       // 普通成功
  | 'failure'       // 失败
  | 'fumble';       // 大失败 (100)
```

---

## 检定计算服务

```python
# app/services/check.py
from typing import Optional, List
from app.core.dice import roll_d100, choose_highest, choose_lowest
from app.core.success import calculate_success_level, get_success_description

class CheckService:
    def __init__(self, db: Session):
        self.db = db

    def perform_check(self, request: CheckRequest) -> CheckResponse:
        """执行检定"""
        # 1. 获取角色和技能值
        character = self._get_character(request.character_id)
        skill_value = self._get_skill_value(character, request.skill)

        # 2. 应用修正值
        effective_value = skill_value + request.modifier

        # 3. 计算难度阈值
        difficulty_threshold = self._get_difficulty_threshold(
            effective_value,
            request.difficulty
        )

        # 4. 处理奖励骰
        if request.bonus_dice and request.bonus_dice > 0:
            roll_result, bonus_info = self._roll_with_bonus(
                request.bonus_dice
            )
        # 5. 处理惩罚骰
        elif request.penalty_dice and request.penalty_dice > 0:
            roll_result, penalty_info = self._roll_with_penalty(
                request.penalty_dice
            )
        # 6. 普通掷骰
        else:
            roll_result = roll_d100()
            bonus_info = None
            penalty_info = None

        # 7. 计算成功等级
        success_level, passed = calculate_success_level(
            roll_result,
            effective_value,
            request.difficulty
        )

        # 8. 检查是否大成功/大失败
        is_critical = (roll_result == 1)
        is_fumble = (roll_result == 100)

        # 9. 检查是否可以推骰
        can_push = (
            not passed and
            not is_fumble and
            skill_value >= request.skill_value  # 没有临时调整
        )

        # 10. 计算推骰成本
        push_cost = self._calculate_push_cost(character)

        # 11. 生成描述
        description = get_success_description(
            success_level,
            request.skill,
            character.name
        )

        return CheckResponse(
            check_id=self._generate_check_id(),
            character_id=request.character_id,
            skill=request.skill,
            skill_value=skill_value,
            roll_result=roll_result,
            raw_rolls=[roll_result],
            difficulty=request.difficulty or 'regular',
            difficulty_modifier=difficulty_threshold - skill_value,
            success_level=success_level,
            passed=passed,
            modifier=request.modifier or 0,
            bonus_dice_used=request.bonus_dice,
            penalty_dice_used=request.penalty_dice,
            bonus_rolls=bonus_info,
            penalty_rolls=penalty_info,
            critical=is_critical,
            fumble=is_fumble,
            description=description,
            can_push=can_push,
            push_cost=push_cost,
            timestamp=datetime.now().isoformat()
        )

    def _roll_with_bonus(self, count: int):
        """使用奖励骰"""
        rolls = [roll_d100() for _ in range(count)]
        chosen = choose_highest(rolls)
        return chosen, {
            'count': count,
            'rolls': rolls,
            'chosen': chosen
        }

    def _roll_with_penalty(self, count: int):
        """使用惩罚骰"""
        rolls = [roll_d100() for _ in range(count)]
        chosen = choose_lowest(rolls)
        return chosen, {
            'count': count,
            'rolls': rolls,
            'chosen': chosen
        }

    def _get_difficulty_threshold(self, skill_value: int, difficulty: str) -> int:
        """获取难度阈值"""
        if difficulty == 'regular':
            return skill_value
        elif difficulty == 'hard':
            return skill_value // 2
        elif difficulty == 'extreme':
            return skill_value // 5
        else:
            return skill_value

    def _calculate_push_cost(self, character: Character) -> int:
        """计算推骰成本"""
        # 推骰消耗 Luck 点数，至少 1 点
        # 某些情况可能消耗更多
        return max(1, character.derived.luck // 10)
```

---

## API 端点

```python
# app/api/check.py
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from app.api.deps import get_current_user, get_db
from app.schemas.check import CheckRequest, CheckResponse
from app.services.check import CheckService

router = APIRouter(prefix="/game", tags=["check"])

@router.post("/check", response_model=CheckResponse)
async def perform_check(
    request: CheckRequest,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """执行技能检定"""
    service = CheckService(db)

    # 验证权限
    if current_user.id != request.character_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You can only check for your own character"
        )

    # 执行检定
    try:
        result = service.perform_check(request)
        return result
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )

@router.get("/check/skills")
async def list_checkable_skills(
    character_id: str,
    current_user = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """列出可检定的技能"""
    # 返回技能列表和属性
    pass
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/schemas/check.py` | 创建 | 检定 Schema |
| `app/services/check.py` | 创建 | 检定服务 |
| `app/api/check.py` | 创建 | 检定 API |
| `tests/test_check.py` | 创建 | 检定测试 |

---

## 验收标准

- [ ] 检定计算正确
- [ ] 难度调整准确
- [ ] 奖励/惩罚骰正常
- [ ] 推骰检查正确
- [ ] 单元测试覆盖
- [ ] API 文档完整

---

## 参考文档

- M1-057: d100 随机数生成
- M1-058: 大成功/大失败判定
- CoC 7e 规则书 - 检定章节

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
