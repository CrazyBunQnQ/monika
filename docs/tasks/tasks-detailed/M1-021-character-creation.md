# M1-021: 实现角色创建向导

**任务ID**: M1-021
**标题**: 实现角色创建向导
**类型**: fullstack (全栈开发)
**预估工时**: 2.5h
**依赖**: M1-001

---

## 任务描述

实现一个分步骤的角色创建向导，引导用户完成 CoC 7e 角色的创建，包括属性分配、职业选择、技能点分配等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-021-01 | 设计向导流程 | Wizard Flow | 25min |
| M1-021-02 | 实现属性生成器 | Attribute Generator | 35min |
| M1-021-03 | 实现职业选择器 | Occupation Selector | 30min |
| M1-021-04 | 实现技能分配器 | Skill Allocator | 35min |
| M1-021-05 | 实现背景定制 | Background Customization | 25min |
| M1-021-06 | 实现向导UI组件 | Wizard UI | 25min |
| M1-021-07 | 编写测试 | 测试覆盖 | 25min |

---

## 属性生成服务

```python
# app/services/character/creation.py
from typing import Dict, Any, List
from sqlalchemy.orm import Session
import random

from app.db.models.character import Character
from app.core.security import generate_id

class AttributeGenerator:
    """属性生成器 - CoC 7e 规则"""

    @staticmethod
    def roll_attributes() -> Dict[str, int]:
        """使用掷骰方法生成属性

        CoC 7e 规则:
        - 力量: 3d6×5
        - 体质: 3d6×5
        - 体型: (2d6+6)×5
        - 敏捷: 3d6×5
        - 外貌: 3d6×5
        - 智力: (2d6+6)×5
        - 意志: 3d6×5
        - 教育: (2d6+6)×5
        """
        return {
            'strength': AttributeGenerator._roll_3d6_times_5(),
            'constitution': AttributeGenerator._roll_3d6_times_5(),
            'size': AttributeGenerator._roll_2d6_plus_6_times_5(),
            'dexterity': AttributeGenerator._roll_3d6_times_5(),
            'appearance': AttributeGenerator._roll_3d6_times_5(),
            'intelligence': AttributeGenerator._roll_2d6_plus_6_times_5(),
            'power': AttributeGenerator._roll_3d6_times_5(),
            'education': AttributeGenerator._roll_2d6_plus_6_times_5(),
        }

    @staticmethod
    def _roll_3d6_times_5() -> int:
        """3d6 × 5"""
        return sum(random.randint(1, 6) for _ in range(3)) * 5

    @staticmethod
    def _roll_2d6_plus_6_times_5() -> int:
        """(2d6 + 6) × 5"""
        return (sum(random.randint(1, 6) for _ in range(2)) + 6) * 5

    @staticmethod
    def roll_quick_attributes() -> Dict[str, int]:
        """快速生成方法（可选）

        使用预设的属性模板快速创建角色
        """
        templates = [
            {
                'strength': 70, 'constitution': 60, 'size': 60,
                'dexterity': 60, 'appearance': 60, 'intelligence': 70,
                'power': 60, 'education': 70,
            },
            {
                'strength': 60, 'constitution': 70, 'size': 70,
                'dexterity': 50, 'appearance': 50, 'intelligence': 60,
                'power': 70, 'education': 60,
            },
            {
                'strength': 50, 'constitution': 50, 'size': 50,
                'dexterity': 70, 'appearance': 70, 'intelligence': 80,
                'power': 50, 'education': 80,
            },
        ]
        return random.choice(templates)

    @staticmethod
    def calculate_derived_attributes(attributes: Dict[str, int]) -> Dict[str, int]:
        """计算衍生属性"""
        strength = attributes['strength']
        constitution = attributes['constitution']
        size = attributes['size']
        dexterity = attributes['dexterity']
        intelligence = attributes['intelligence']
        power = attributes['power']
        education = attributes['education']

        # HP (伤害吸收) = (体质 + 体型) / 10
        hp = max(1, (constitution + size) // 10)

        # MP (魔法值) = 意志 / 5
        mp = max(1, power // 5)

        # SAN (理智) = 意志
        san = power

        # 幸运 = 3d6 × 5
        luck = AttributeGenerator._roll_3d6_times_5()

        # 移动速率 = 敏捷 / 体质 / 力量
        move_rate = AttributeGenerator._calculate_move_rate(
            dexterity, constitution, strength, size
        )

        # 豁免 = 敏捷
        dodge = dexterity // 2

        return {
            'hp': hp,
            'max_hp': hp,
            'mp': mp,
            'max_mp': mp,
            'san': san,
            'max_san': san,
            'luck': luck,
            'move_rate': move_rate,
            'dodge': dodge,
        }

    @staticmethod
    def _calculate_move_rate(
        dexterity: int,
        constitution: int,
        strength: int,
        size: int,
    ) -> int:
        """计算移动速率

        规则:
        - 敏捷、体质、力量都大于体型: 9
        - 敏捷或体质或力量小于体型: 8
        - 敏捷和体质都小于体型: 7
        """
        if dexterity > size and constitution > size and strength > size:
            return 9
        elif dexterity < size and constitution < size:
            return 7
        else:
            return 8


class OccupationService:
    """职业服务"""

    OCCUPATIONS = {
        'detective': {
            'name': '私家侦探',
            'name_en': 'Private Investigator',
            'credit_rating': '20-60',
            'skills': ['快速交谈', '心理学', '取证', '手枪'],
            'occupation_points': 8,
            'personal_points': None,
        },
        'doctor': {
            'name': '医生',
            'name_en': 'Doctor',
            'credit_rating': '30-80',
            'skills': ['急救', '医学', '心理学', '其他/专精'],
            'occupation_points': 8,
            'personal_points': '智力×2',
        },
        'journalist': {
            'name': '记者',
            'name_en': 'Journalist',
            'credit_rating': '10-50',
            'skills': ['快速交谈', '艺术/手艺', '外语', '历史', '说服'],
            'occupation_points': 8,
            'personal_points': '教育×2',
        },
        'professor': {
            'name': '教授',
            'name_en': 'Professor',
            'credit_rating': '30-70',
            'skills': ['其他/专精', '历史', '图书馆', '说服'],
            'occupation_points': 8,
            'personal_points': '教育×2',
        },
        'antiquarian': {
            'name': '古董商',
            'name_en': 'Antiquarian',
            'credit_rating': '10-50',
            'skills': ['鉴定', '历史', '图书馆', '说服', '其他/专精'],
            'occupation_points': 8,
            'personal_points': '教育×2',
        },
        'artist': {
            'name': '艺术家',
            'name_en': 'Artist',
            'credit_rating': '10-40',
            'skills': ['艺术/手艺', '艺术/手艺(另一项)', '心理学'],
            'occupation_points': 8,
            'personal_points': '智力×2',
        },
        'policeman': {
            'name': '警察',
            'name_en': 'Policeman',
            'credit_rating': '20-50',
            'skills': ['快速交谈', '斗殴', '法律', '射击', '驾驶'],
            'occupation_points': 8,
            'personal_points': None,
        },
        'soldier': {
            'name': '军人',
            'name_en': 'Soldier',
            'credit_rating': '20-50',
            'skills': ['射击', '急救', '侦查', ' scouting', '武器维修'],
            'occupation_points': 8,
            'personal_points': None,
        },
    }

    @classmethod
    def get_occupation(cls, occupation_id: str) -> Dict[str, Any]:
        """获取职业信息"""
        return cls.OCCUPATIONS.get(occupation_id)

    @classmethod
    def get_all_occupations(cls) -> List[Dict[str, Any]]:
        """获取所有职业"""
        return [
            {'id': k, **v}
            for k, v in cls.OCCUPATIONS.items()
        ]


class SkillPointAllocator:
    """技能点分配器"""

    @staticmethod
    def calculate_occupation_points(
        occupation: Dict[str, Any],
        attributes: Dict[str, int],
    ) -> int:
        """计算职业技能点"""
        occupation_points = occupation['occupation_points']

        # 某些职业使用特定属性计算点数
        personal_points = occupation.get('personal_points')
        if personal_points:
            if '智力' in personal_points:
                return attributes['intelligence'] * 2
            elif '教育' in personal_points:
                return attributes['education'] * 2

        return occupation_points

    @staticmethod
    def calculate_personal_points(
        attributes: Dict[str, int],
        occupation: Dict[str, Any] = None,
    ) -> int:
        """计算个人兴趣技能点"""
        if occupation and occupation.get('personal_points'):
            if '智力' in occupation['personal_points']:
                return attributes['intelligence']
            elif 'education' in occupation['personal_points']:
                return attributes['education']

        # 默认: 智力
        return attributes['intelligence']

    @staticmethod
    def allocate_points(
        skills: List[Dict[str, Any]],
        points: int,
    ) -> Dict[str, int]:
        """分配技能点

        Args:
            skills: 技能列表，每个包含 {id, name, default, max}
            points: 可分配点数

        Returns:
            Dict[skill_id, allocated_points]
        """
        allocation = {}

        # 为默认技能分配基础点数
        for skill in skills:
            if skill.get('default'):
                allocation[skill['id']] = skill['default']
                # 从可用点数中扣除默认点数
                points -= skill['default']

        return {
            'allocation': allocation,
            'remaining': points,
        }

    @staticmethod
    def validate_allocation(
        allocation: Dict[str, int],
        max_points: int,
        skill_limits: Dict[str, int] = None,
    ) -> bool:
        """验证技能点分配是否合法

        Args:
            allocation: 技能点分配 {skill_id: points}
            max_points: 最大点数
            skill_limits: 技能上限 {skill_id: max}

        Returns:
            是否合法
        """
        # 检查总点数
        total = sum(allocation.values())
        if total > max_points:
            return False

        # 检查每个技能的上限
        if skill_limits:
            for skill_id, points in allocation.items():
                limit = skill_limits.get(skill_id)
                if limit and points > limit:
                    return False

        return True


class CharacterCreationService:
    """角色创建服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_character(
        self,
        user_id: str,
        name: str,
        player_name: str,
        attributes: Dict[str, int],
        occupation_id: str,
        skills: Dict[str, int],
        age: int = None,
        backstory: str = None,
    ) -> Character:
        """创建角色"""
        # 生成衍生属性
        derived = AttributeGenerator.calculate_derived_attributes(attributes)

        # 获取职业信息
        occupation = OccupationService.get_occupation(occupation_id)

        # 创建角色
        character = Character(
            id=generate_id('character'),
            user_id=user_id,
            name=name,
            player_name=player_name,
            occupation=occupation['name'],
            occupation_en=occupation['name_en'],
            age=age or self._calculate_age(attributes['education']),
            backstory=backstory or '',
            strength=attributes['strength'],
            constitution=attributes['constitution'],
            size=attributes['size'],
            dexterity=attributes['dexterity'],
            appearance=attributes['appearance'],
            intelligence=attributes['intelligence'],
            power=attributes['power'],
            education=attributes['education'],
            hp=derived['hp'],
            max_hp=derived['max_hp'],
            mp=derived['mp'],
            max_mp=derived['max_mp'],
            san=derived['san'],
            max_san=derived['max_san'],
            luck=derived['luck'],
            move_rate=derived['move_rate'],
            dodge=derived['dodge'],
        )

        self.db.add(character)
        self.db.commit()
        self.db.refresh(character)

        # 创建技能记录
        # TODO: 创建技能

        return character

    @staticmethod
    def _calculate_age(education: int) -> int:
        """根据教育计算年龄"""
        base_age = 18 + (education - 50) // 10
        return max(18, min(base_age + random.randint(0, 10), 70))
```

---

## 角色创建 API

```python
# app/api/character/creation.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import Dict, List

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.character.creation import (
    AttributeGenerator,
    OccupationService,
    SkillPointAllocator,
    CharacterCreationService,
)

router = APIRouter(prefix="/character/creation", tags=["character_creation"])

@router.post("/roll-attributes")
async def roll_attributes(
    method: str = "standard",  # standard, quick
):
    """生成属性"""
    if method == "quick":
        attributes = AttributeGenerator.roll_quick_attributes()
    else:
        attributes = AttributeGenerator.roll_attributes()

    derived = AttributeGenerator.calculate_derived_attributes(attributes)

    return {
        "attributes": attributes,
        "derived": derived,
    }

@router.get("/occupations")
async def get_occupations():
    """获取所有职业"""
    occupations = OccupationService.get_all_occupations()
    return {"occupations": occupations}

@router.get("/occupations/{occupation_id}")
async def get_occupation_details(
    occupation_id: str,
):
    """获取职业详情"""
    occupation = OccupationService.get_occupation(occupation_id)
    if not occupation:
        raise HTTPException(status_code=404, detail="职业不存在")
    return occupation

@router.post("/calculate-points")
async def calculate_skill_points(
    occupation_id: str,
    attributes: Dict[str, int],
):
    """计算技能点"""
    occupation = OccupationService.get_occupation(occupation_id)
    if not occupation:
        raise HTTPException(status_code=404, detail="职业不存在")

    occupation_points = SkillPointAllocator.calculate_occupation_points(
        occupation, attributes
    )
    personal_points = SkillPointAllocator.calculate_personal_points(
        attributes, occupation
    )

    return {
        "occupation_points": occupation_points,
        "personal_points": personal_points,
        "total_points": occupation_points + personal_points,
    }

@router.post("/validate-allocation")
async def validate_allocation(
    allocation: Dict[str, int],
    max_points: int,
    skill_limits: Dict[str, int] = None,
):
    """验证技能点分配"""
    is_valid = SkillPointAllocator.validate_allocation(
        allocation, max_points, skill_limits
    )
    return {"valid": is_valid}

@router.post("/create")
async def create_character(
    name: str,
    player_name: str,
    attributes: Dict[str, int],
    occupation_id: str,
    skills: Dict[str, int],
    age: int = None,
    backstory: str = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建角色"""
    service = CharacterCreationService(db)

    character = service.create_character(
        user_id=current_user.id,
        name=name,
        player_name=player_name,
        attributes=attributes,
        occupation_id=occupation_id,
        skills=skills,
        age=age,
        backstory=backstory,
    )

    return {
        "character_id": character.id,
        "name": character.name,
    }
```

---

## 前端角色创建向导组件

```tsx
// frontend/src/components/character/CharacterCreationWizard.tsx
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { ChevronLeft, ChevronRight, Check } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { AttributeStep } from './steps/AttributeStep'
import { OccupationStep } from './steps/OccupationStep'
import { SkillStep } from './steps/SkillStep'
import { BackgroundStep } from './steps/BackgroundStep'
import { ReviewStep } from './steps/ReviewStep'

type Step = 'attributes' | 'occupation' | 'skills' | 'background' | 'review'

interface CharacterData {
  attributes?: Record<string, number>
  derived?: Record<string, number>
  occupation?: string
  skills?: Record<string, number>
  occupationPoints?: number
  personalPoints?: number
  name?: string
  playerName?: string
  age?: number
  backstory?: string
}

export function CharacterCreationWizard() {
  const [currentStep, setCurrentStep] = useState<Step>('attributes')
  const [characterData, setCharacterData] = useState<CharacterData>({})
  const [loading, setLoading] = useState(false)
  const { toast } = useToast()

  const steps: Step[] = ['attributes', 'occupation', 'skills', 'background', 'review']
  const currentStepIndex = steps.indexOf(currentStep)
  const progress = ((currentStepIndex + 1) / steps.length) * 100

  const handleNext = () => {
    if (currentStepIndex < steps.length - 1) {
      setCurrentStep(steps[currentStepIndex + 1])
    }
  }

  const handlePrevious = () => {
    if (currentStepIndex > 0) {
      setCurrentStep(steps[currentStepIndex - 1])
    }
  }

  const handleCreate = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/character/creation/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(characterData),
      })

      if (response.ok) {
        const data = await response.json()
        toast({
          title: '角色创建成功',
          description: `角色 ${data.name} 已创建`,
        })
        // 跳转到角色页面
        window.location.href = `/characters/${data.character_id}`
      }
    } catch (error) {
      toast({
        title: '创建失败',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>创建新角色</CardTitle>
          <Progress value={progress} className="mt-4" />
        </CardHeader>
        <CardContent>
          {currentStep === 'attributes' && (
            <AttributeStep
              data={characterData}
              onChange={(data) => setCharacterData({ ...characterData, ...data })}
              onNext={handleNext}
            />
          )}

          {currentStep === 'occupation' && (
            <OccupationStep
              data={characterData}
              onChange={(data) => setCharacterData({ ...characterData, ...data })}
              onNext={handleNext}
              onPrevious={handlePrevious}
            />
          )}

          {currentStep === 'skills' && (
            <SkillStep
              data={characterData}
              onChange={(data) => setCharacterData({ ...characterData, ...data })}
              onNext={handleNext}
              onPrevious={handlePrevious}
            />
          )}

          {currentStep === 'background' && (
            <BackgroundStep
              data={characterData}
              onChange={(data) => setCharacterData({ ...characterData, ...data })}
              onNext={handleNext}
              onPrevious={handlePrevious}
            />
          )}

          {currentStep === 'review' && (
            <ReviewStep
              data={characterData}
              onCreate={handleCreate}
              onPrevious={handlePrevious}
              loading={loading}
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
```

---

## 属性生成步骤组件

```tsx
// frontend/src/components/character/steps/AttributeStep.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Dices } from 'lucide-react'

interface AttributeStepProps {
  data: CharacterData
  onChange: (data: CharacterData) => void
  onNext: () => void
}

export function AttributeStep({ data, onChange, onNext }: AttributeStepProps) {
  const [rolling, setRolling] = useState(false)

  const handleRoll = async (method: 'standard' | 'quick') => {
    setRolling(true)
    try {
      const response = await fetch(`/api/character/creation/roll-attributes?method=${method}`)
      if (response.ok) {
        const result = await response.json()
        onChange({
          ...data,
          attributes: result.attributes,
          derived: result.derived,
        })
      }
    } finally {
      setRolling(false)
    }
  }

  const attributeNames = {
    strength: '力量',
    constitution: '体质',
    size: '体型',
    dexterity: '敏捷',
    appearance: '外貌',
    intelligence: '智力',
    power: '意志',
    education: '教育',
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium mb-2">属性生成</h3>
        <p className="text-sm text-muted-foreground">
          选择一种方式生成你的角色属性
        </p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <Button
          variant="outline"
          onClick={() => handleRoll('standard')}
          disabled={rolling}
        >
          <Dices className="h-4 w-4 mr-2" />
          标准掷骰 (3d6×5)
        </Button>
        <Button
          variant="outline"
          onClick={() => handleRoll('quick')}
          disabled={rolling}
        >
          <Dices className="h-4 w-4 mr-2" />
          快速创建
        </Button>
      </div>

      {data.attributes && (
        <div className="space-y-4">
          <h4 className="font-medium">属性值</h4>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {Object.entries(data.attributes).map(([key, value]) => (
              <div key={key} className="text-center p-4 rounded-lg bg-muted">
                <div className="text-sm text-muted-foreground">
                  {attributeNames[key]}
                </div>
                <div className="text-2xl font-bold">{value}</div>
              </div>
            ))}
          </div>

          {data.derived && (
            <div className="mt-6">
              <h4 className="font-medium mb-3">衍生属性</h4>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <div className="text-center p-3 rounded bg-secondary">
                  <div className="text-xs text-muted-foreground">HP</div>
                  <div className="text-xl font-bold">{data.derived.hp}</div>
                </div>
                <div className="text-center p-3 rounded bg-secondary">
                  <div className="text-xs text-muted-foreground">MP</div>
                  <div className="text-xl font-bold">{data.derived.mp}</div>
                </div>
                <div className="text-center p-3 rounded bg-secondary">
                  <div className="text-xs text-muted-foreground">SAN</div>
                  <div className="text-xl font-bold">{data.derived.san}</div>
                </div>
                <div className="text-center p-3 rounded bg-secondary">
                  <div className="text-xs text-muted-foreground">幸运</div>
                  <div className="text-xl font-bold">{data.derived.luck}</div>
                </div>
              </div>
            </div>
          )}

          <div className="flex justify-end">
            <Button onClick={onNext}>
              下一步
            </Button>
          </div>
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
| `app/services/character/creation.py` | 创建 | 角色创建服务 |
| `app/api/character/creation.py` | 创建 | 角色创建 API |
| `frontend/src/components/character/CharacterCreationWizard.tsx` | 创建 | 角色创建向导 |
| `frontend/src/components/character/steps/AttributeStep.tsx` | 创建 | 属性步骤 |
| `frontend/src/components/character/steps/OccupationStep.tsx` | 创建 | 职业步骤 |
| `frontend/src/components/character/steps/SkillStep.tsx` | 创建 | 技能步骤 |
| `frontend/src/components/character/steps/BackgroundStep.tsx` | 创建 | 背景步骤 |
| `frontend/src/components/character/steps/ReviewStep.tsx` | 创建 | 审查步骤 |

---

## 验收标准

- [ ] 属性生成符合 CoC 7e 规则
- [ ] 职业选择完整
- [ ] 技能点计算正确
- [ ] 向导流程流畅
- [ ] 表单验证有效
- [ ] 创建成功跳转正确

---

## 参考文档

- M1-001: 角色系统
- M1-005: 技能系统
- CoC 7e 规则书 - 角色创建

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
