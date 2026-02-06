# M1-005: 技能系统

**任务ID**: M1-005
**标题**: 技能系统
**类型**: backend (后端开发)
**预估工时**: 6h
**依赖**: M1-003

---

## 任务描述

实现 CoC 7e 技能系统，包括技能定义、技能值计算、成长检定、职业加成等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-005-01 | 定义标准技能列表 | CoC 7e 官方技能 | 30min |
| M1-005-02 | 设计技能数据结构 | Skill 模型 | 20min |
| M1-005-03 | 实现技能计算服务 | 基础技能值计算 | 45min |
| M1-005-04 | 实现职业加成逻辑 | 职业技能加成 | 30min |
| M1-005-05 | 实现技能成长检定 | 经验获取和提升 | 45min |
| M1-005-06 | 实现技能 API | 技能查询和更新 | 30min |
| M1-005-07 | 实现技能导入/导出 | 预设技能模板 | 30min |
| M1-005-08 | 编写技能测试 | 单元测试 | 30min |
| M1-005-09 | 编写技能文档 | 技能系统说明 | 15min |
| M1-005-10 | 添加技能数据种子 | 预置技能数据 | 15min |

---

## 标准技能列表

```typescript
// 基础技能
const BASIC_SKILLS = {
  // 体质类
  dodge: { name: '闪避', base: 'DEX/2', category: 'physical' },
  jump: { name: '跳跃', base: 'STR*3', category: 'physical' },
  climb: { name: '攀爬', base: 'STR*3', category: 'physical' },
  swim: { name: '游泳', base: 'STR*5', category: 'physical' },
  throw: { name: '投掷', base: 'DEX*4', category: 'physical' },

  // 知识类
  accounting: { name: '会计', base: 'EDU*10', category: 'knowledge' },
  anthropology: { name: '人类学', base: 'EDU*5', category: 'knowledge' },
  archaeology: { name: '考古学', base: 'EDU*5', category: 'knowledge' },
  astronomy: { name: '天文学', base: 'EDU*5', category: 'knowledge' },
  biology: { name: '生物学', base: 'EDU*5', category: 'knowledge' },
  botany: { name: '植物学', base: 'EDU*5', category: 'knowledge' },
  chemistry: { name: '化学', base: 'EDU*5', category: 'knowledge' },
  geology: { name: '地质学', base: 'EDU*5', category: 'knowledge' },
  history: { name: '历史', base: 'EDU*5', category: 'knowledge' },
  law: { name: '法律', base: 'EDU*5', category: 'knowledge' },
  medicine: { name: '医学', base: 'EDU*5', category: 'knowledge' },
  occult: { name: '神秘学', base: 'EDU*5', category: 'knowledge' },
  physics: { name: '物理学', base: 'EDU*5', category: 'knowledge' },
  psychology: { name: '心理学', base: 'EDU*5', category: 'knowledge' },
  zoology: { name: '动物学', base: 'EDU*5', category: 'knowledge' },

  // 交际类
  charm: { name: '说服', base: 'POW*5', category: 'social' },
  fast_talk: { name: '话术', base: 'EDU*5', category: 'social' },
  intimidate: { name: '恐吓', base: 'POW*5', category: 'social' },
  negotiate: { name: '讨价还价', base: 'EDU*5', category: 'social' },

  // 调查类
  art_craft: { name: '艺术/手艺', base: 'EDU*5', category: 'investigation' },
  first_aid: { name: '急救', base: 'EDU*5', category: 'investigation' },
  spot_hidden: { name: '侦查', base: 'INT*5', category: 'investigation' },
  listen: { name: '聆听', base: 'INT*5', category: 'investigation' },
  psychology_knowledge: { name: '心理学', base: 'EDU*5', category: 'investigation' },
  track: { name: '追踪', base: 'INT*5', category: 'investigation' },

  // 语言类
  own_language: { name: '母语', base: 'EDU', category: 'language' },
  other_language: { name: '外语', base: 'EDU*5', category: 'language' },

  // 战斗类
  firearms_handgun: { name: '手枪', base: 'DEX*4', category: 'combat' },
  firearms_rifle: { name: '步枪/霰弹枪', base: 'DEX*4', category: 'combat' },
  brawl: { name: '格斗(斗殴)', base: 'DEX*2', category: 'combat' },
  // ... 其他武器技能
};
```

---

## 技能数据结构

```typescript
interface Skill {
  id: string;
  name: string;
  name_en: string;  // 英文标识

  // 基础值计算
  base_formula: string;  // 如 "DEX*2", "EDU*5"
  base_value: number;    // 根据属性计算的基础值

  // 当前值
  current_value: number;
  adjustments: {
    occupation?: number;    // 职业加成
    interest?: number;      // 兴趣点数
    experience?: number;    // 经验提升
    other?: number;         // 其他调整
  };

  // 元数据
  category: 'physical' | 'knowledge' | 'social' | 'investigation' | 'language' | 'combat';
  can_growth: boolean;      // 是否可成长
  uses: number;             // 使用次数
  last_used?: datetime;
}

interface CharacterSkills {
  character_id: string;
  skills: Record<string, Skill>;
  skill_points_used: number;  // 已使用的兴趣点数
  skill_points_total: number; // 总兴趣点数 (INT*10)
}
```

---

## 技能计算服务

```python
# app/services/skill.py
from typing import Dict, List
import re

class SkillService:
    def __init__(self, db: Session):
        self.db = db

    def calculate_base_value(self, formula: str, attributes: Dict[str, int]) -> int:
        """根据公式计算基础技能值"""
        # 解析公式，如 "DEX*2" 或 "EDU*5"
        match = re.match(r'(\w+)\*(\d+)', formula)
        if match:
            attr = match.group(1)
            multiplier = int(match.group(2))
            return attributes.get(attr, 0) * multiplier

        # 直接属性值，如 "EDU"
        if formula in attributes:
            return attributes[formula]

        raise ValueError(f"Invalid skill formula: {formula}")

    def calculate_all_skills(self, attributes: Dict[str, int]) -> Dict[str, int]:
        """计算所有技能的基础值"""
        skills = {}
        for skill_id, skill_def in BASIC_SKILLS.items():
            base_value = self.calculate_base_value(
                skill_def['base_formula'],
                attributes
            )
            skills[skill_id] = base_value
        return skills

    def apply_occupation_bonuses(
        self,
        skills: Dict[str, int],
        occupation_skills: List[str],
        bonus_points: int = 20
    ) -> Dict[str, int]:
        """应用职业加成"""
        result = skills.copy()
        available_points = bonus_points

        for skill_id in occupation_skills:
            if skill_id in result and available_points > 0:
                # 每个职业技能最多加 20 点
                max_add = min(20, available_points)
                result[skill_id] += max_add
                available_points -= max_add

        return result

    def add_interest_points(
        self,
        skills: Dict[str, int],
        interest_skills: Dict[str, int],  # skill_id -> points
        total_points: int
    ) -> Dict[str, int]:
        """添加兴趣点数"""
        result = skills.copy()

        used_points = sum(interest_skills.values())
        if used_points > total_points:
            raise ValueError(f"Interest points exceed total: {used_points} > {total_points}")

        for skill_id, points in interest_skills.items():
            if skill_id in result:
                result[skill_id] += points

        return result
```

---

## 技能成长检定

```python
# app/services/growth.py
class SkillGrowthService:
    def __init__(self, db: Session):
        self.db = db

    def can_attempt_growth(self, skill: Skill, character: Character) -> bool:
        """检查是否可以尝试成长"""
        # 技能必须使用过
        if skill.uses == 0:
            return False

        # 技能值不能超过 99 (极少数例外)
        if skill.current_value >= 99:
            return False

        # 需要足够的游戏时间
        # 这里简化处理，实际应该检查游戏内时间
        return True

    def attempt_growth(self, skill: Skill, roll_result: int) -> Dict:
        """执行技能成长检定"""
        # 成长检定：d100 <= 当前技能值
        success = roll_result <= skill.current_value

        result = {
            'skill_id': skill.id,
            'success': success,
            'old_value': skill.current_value,
            'roll_result': roll_result,
        }

        if success:
            # 大成功：+2d10
            if roll_result == 1:
                increase = sum(random.randint(1, 10) for _ in range(2))
            # 普通成功：+1d10
            else:
                increase = random.randint(1, 10)

            skill.current_value = min(99, skill.current_value + increase)
            skill.uses = 0  # 重置使用次数
            result['increase'] = increase
            result['new_value'] = skill.current_value
        else:
            result['increase'] = 0
            result['new_value'] = skill.current_value

        return result

    def mark_skill_used(self, character_id: str, skill_id: str):
        """标记技能已使用"""
        # 实现使用次数记录
        pass
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/skills.py` | 创建 | 标准技能定义 |
| `app/services/skill.py` | 创建 | 技能计算服务 |
| `app/services/growth.py` | 创建 | 技能成长服务 |
| `app/api/skills.py` | 创建 | 技能 API |
| `app/db/models/skill.py` | 创建 | 技能数据模型 |
| `tests/test_skills.py` | 创建 | 技能测试 |

---

## API 端点

```
GET    /characters/:id/skills        - 获取角色技能列表
GET    /characters/:id/skills/:skill  - 获取单个技能
POST   /characters/:id/skills/:skill/use  - 标记技能使用
POST   /characters/:id/skills/:skill/growth - 尝试成长
PUT    /characters/:id/skills        - 更新技能值
GET    /skills/list                  - 获取标准技能列表
```

---

## 验收标准

- [ ] 标准技能列表完整
- [ ] 基础值计算正确
- [ ] 职业加成正确应用
- [ ] 兴趣点数限制正确
- [ ] 成长检定符合规则
- [ ] 技能上限正确 (99)
- [ ] API 文档完整

---

## 参考文档

- CoC 7e 规则书 - 技能章节
- M1-003: 角色卡数据模型

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
