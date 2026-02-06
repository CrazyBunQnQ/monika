# M5-009: 实现战绩系统

**任务ID**: M5-009
**标题**: 实现战绩系统
**类型**: fullstack (全栈开发)
**预估工时**: 3h
**依赖**: M1-001

---

## 任务描述

实现战绩系统，记录玩家的游戏统计数据和成就，包括游戏场次、胜利次数、角色存活情况等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-009-01 | 设计统计数据模型 | Statistics Model | 30min |
| M5-009-02 | 设计成就系统 | Achievement System | 35min |
| M5-009-03 | 实现数据追踪 | Data Tracking | 30min |
| M5-009-04 | 实现成就解锁 | Achievement Unlock | 30min |
| M5-009-05 | 实现排行榜 | Leaderboard | 25min |
| M5-009-06 | 实现统计展示 | Statistics Display | 25min |
| M5-009-07 | 编写测试 | 测试覆盖 | 25min |

---

## 统计数据模型

```python
# app/db/models/statistics.py
from sqlalchemy import Column, String, Integer, Float, DateTime, ForeignKey, JSON, Boolean
from sqlalchemy.orm import relationship
from app.db.database import Base
from datetime import datetime

class PlayerStatistics(Base):
    """玩家统计数据"""
    __tablename__ = 'player_statistics'

    id = Column(String, primary_key=True, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False, unique=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False)

    # 游戏次数
    games_played = Column(Integer, default=0)
    games_completed = Column(Integer, default=0)
    games_abandoned = Column(Integer, default=0)

    # 角色统计
    characters_created = Column(Integer, default=0)
    characters_deceased = Column(Integer, default=0)
    characters_insane = Column(Integer, default=0)

    # 掷骰统计
    dice_rolls_total = Column(Integer, default=0)
    critical_successes = Column(Integer, default=0)
    critical_failures = Column(Integer, default=0)
    luck_rolls = Column(Integer, default=0)

    # 检定统计
    skill_checks_total = Column(Integer, default=0)
    skill_checks_passed = Column(Integer, default=0)
    skill_checks_failed = Column(Integer, default=0)

    # 战斗统计
    enemies_defeated = Column(Integer, default=0)
    damage_dealt = Column(Integer, default=0)
    damage_taken = Column(Integer, default=0)
    healing_done = Column(Integer, default=0)

    # SAN 值统计
    san_lost_total = Column(Integer, default=0)
    san_checks_passed = Column(Integer, default=0)
    san_checks_failed = Column(Integer, default=0)
    times_went_insane = Column(Integer, default=0)

    # 时间统计
    total_playtime_seconds = Column(Integer, default=0)
    sessions_joined = Column(Integer, default=0)
    sessions_hosted = Column(Integer, default=0)

    # 社交统计
    messages_sent = Column(Integer, default=0)
    voice_chat_minutes = Column(Integer, default=0)

    # 详细记录
    character_history = Column(JSON, default=list)
    notable_achievements = Column(JSON, default=list)

    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 关系
    user = relationship("User", back_populates="statistics")
    room = relationship("Room", back_populates="statistics")

    def __repr__(self):
        return f"<PlayerStatistics user={self.user_id}>"
```

---

## 成就模型

```python
# app/db/models/achievement.py
from sqlalchemy import Column, String, Integer, Text, ForeignKey, Boolean, JSON, DateTime
from sqlalchemy.orm import relationship
from app.db.database import Base
from datetime import datetime

class Achievement(Base):
    """成就"""
    __tablename__ = 'achievements'

    id = Column(String, primary_key=True, index=True)
    code = Column(String, unique=True, nullable=False, index=True)

    # 基本信息
    name = Column(String, nullable=False)
    name_en = Column(String)
    description = Column(Text)
    description_en = Column(Text)
    icon = Column(String)  # emoji 或图标 URL

    # 成就类型
    category = Column(String)  # combat, exploration, social, survival, special
    rarity = Column(String, default='common')  # common, rare, epic, legendary

    # 解锁条件
    requirements = Column(JSON, nullable=False)
    requirement_type = Column(String)  # count, threshold, condition

    # 奖励
    reward_xp = Column(Integer, default=0)
    reward_badge = Column(String)
    reward_title = Column(String)

    # 统计
    unlock_count = Column(Integer, default=0)

    # 是否隐藏
    is_hidden = Column(Boolean, default=False)

    created_at = Column(DateTime, default=datetime.utcnow)

class UserAchievement(Base):
    """用户成就"""
    __tablename__ = 'user_achievements'

    id = Column(String, primary_key=True, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False, index=True)
    achievement_id = Column(String, ForeignKey('achievements.id'), nullable=False)

    # 解锁信息
    unlocked_at = Column(DateTime, default=datetime.utcnow)
    unlock_context = Column(JSON)  # 解锁时的额外信息

    # 进度（对于未完成的成就）
    progress = Column(JSON)

    # 是否已展示
    has_seen = Column(Boolean, default=False)

    # 关系
    user = relationship("User", back_populates="achievements")
    achievement = relationship("Achievement")

    def __repr__(self):
        return f"<UserAchievement user={self.user_id} achievement={self.achievement_id}>"
```

---

## 统计服务

```python
# app/services/statistics.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session
from sqlalchemy import func, and_
from datetime import datetime, timedelta

from app.db.models.statistics import PlayerStatistics
from app.db.models.achievement import Achievement, UserAchievement
from app.db.models.room import Room
from app.core.security import generate_id

class StatisticsService:
    """统计服务"""

    def __init__(self, db: Session):
        self.db = db

    def get_or_create_statistics(
        self,
        user_id: str,
        room_id: str,
    ) -> PlayerStatistics:
        """获取或创建统计数据"""
        stats = self.db.query(PlayerStatistics)\
            .filter(
                and_(
                    PlayerStatistics.user_id == user_id,
                    PlayerStatistics.room_id == room_id,
                )
            )\
            .first()

        if not stats:
            stats = PlayerStatistics(
                id=generate_id('stats'),
                user_id=user_id,
                room_id=room_id,
            )
            self.db.add(stats)
            self.db.commit()
            self.db.refresh(stats)

        return stats

    def record_dice_roll(
        self,
        user_id: str,
        room_id: str,
        result: int,
        is_critical: bool = False,
        is_critical_failure: bool = False,
    ):
        """记录掷骰"""
        stats = self.get_or_create_statistics(user_id, room_id)

        stats.dice_rolls_total += 1
        if is_critical:
            stats.critical_successes += 1
        if is_critical_failure:
            stats.critical_failures += 1

        self.db.commit()
        return stats

    def record_skill_check(
        self,
        user_id: str,
        room_id: str,
        passed: bool,
        skill_name: str = None,
    ):
        """记录技能检定"""
        stats = self.get_or_create_statistics(user_id, room_id)

        stats.skill_checks_total += 1
        if passed:
            stats.skill_checks_passed += 1
        else:
            stats.skill_checks_failed += 1

        self.db.commit()
        return stats

    def record_combat_action(
        self,
        user_id: str,
        room_id: str,
        action_type: str,  # attack, damage, heal
        value: int = 0,
        defeated: bool = False,
    ):
        """记录战斗行动"""
        stats = self.get_or_create_statistics(user_id, room_id)

        if action_type == 'attack' and defeated:
            stats.enemies_defeated += 1
        elif action_type == 'damage':
            stats.damage_dealt += value
        elif action_type == 'take_damage':
            stats.damage_taken += value
        elif action_type == 'heal':
            stats.healing_done += value

        self.db.commit()
        return stats

    def record_san_check(
        self,
        user_id: str,
        room_id: str,
        san_lost: int,
        passed: bool,
    ):
        """记录 SAN 检定"""
        stats = self.get_or_create_statistics(user_id, room_id)

        stats.san_lost_total += san_lost
        if passed:
            stats.san_checks_passed += 1
        else:
            stats.san_checks_failed += 1

        self.db.commit()
        return stats

    def record_character_fate(
        self,
        user_id: str,
        room_id: str,
        character_id: str,
        character_name: str,
        fate: str,  # deceased, insane, survived
    ):
        """记录角色命运"""
        stats = self.get_or_create_statistics(user_id, room_id)

        # 添加到历史记录
        if not stats.character_history:
            stats.character_history = []

        stats.character_history.append({
            "character_id": character_id,
            "name": character_name,
            "fate": fate,
            "timestamp": datetime.utcnow().isoformat(),
        })

        # 更新计数
        if fate == 'deceased':
            stats.characters_deceased += 1
        elif fate == 'insane':
            stats.characters_insane += 1

        self.db.commit()
        return stats

    def get_room_leaderboard(
        self,
        room_id: str,
        metric: str = 'games_completed',
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        """获取房间排行榜"""
        stats = self.db.query(PlayerStatistics)\
            .filter(PlayerStatistics.room_id == room_id)\
            .order_by(getattr(PlayerStatistics, metric).desc())\
            .limit(limit)\
            .all()

        return [
            {
                "user_id": s.user_id,
                "username": s.user.username,
                "value": getattr(s, metric),
                "rank": idx + 1,
            }
            for idx, s in enumerate(stats)
        ]

    def get_global_leaderboard(
        self,
        metric: str = 'games_completed',
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """获取全局排行榜"""
        # 聚合所有房间的数据
        result = self.db.query(
            PlayerStatistics.user_id,
            func.sum(getattr(PlayerStatistics, metric)).label('total'),
        )\
            .group_by(PlayerStatistics.user_id)\
            .order_by(func.sum(getattr(PlayerStatistics, metric)).desc())\
            .limit(limit)\
            .all()

        return [
            {
                "user_id": row.user_id,
                "value": row.total,
                "rank": idx + 1,
            }
            for idx, row in enumerate(result)
        ]
```

---

## 成就服务

```python
# app/services/achievement.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session

from app.db.models.achievement import Achievement, UserAchievement
from app.db.models.statistics import PlayerStatistics
from app.core.security import generate_id

class AchievementService:
    """成就服务"""

    def __init__(self, db: Session):
        self.db = db

    def check_achievements(
        self,
        user_id: str,
        room_id: str,
    ) -> List[Dict[str, Any]]:
        """检查并解锁成就"""
        stats = self.db.query(PlayerStatistics)\
            .filter(
                PlayerStatistics.user_id == user_id,
                PlayerStatistics.room_id == room_id,
            )\
            .first()

        if not stats:
            return []

        unlocked = []

        # 获取所有未解锁的成就
        existing_achievements = self.db.query(UserAchievement)\
            .filter(UserAchievement.user_id == user_id)\
            .all()
        existing_ids = {ua.achievement_id for ua in existing_achievements}

        achievements = self.db.query(Achievement)\
            .filter(~Achievement.id.in_(existing_ids))\
            .all()

        for achievement in achievements:
            if self._check_requirement(achievement, stats):
                # 解锁成就
                user_achievement = UserAchievement(
                    id=generate_id('user_achievement'),
                    user_id=user_id,
                    achievement_id=achievement.id,
                    unlocked_at=datetime.utcnow(),
                )
                self.db.add(user_achievement)

                # 更新成就计数
                achievement.unlock_count += 1

                unlocked.append({
                    "id": achievement.id,
                    "code": achievement.code,
                    "name": achievement.name,
                    "description": achievement.description,
                    "icon": achievement.icon,
                    "rarity": achievement.rarity,
                    "reward_xp": achievement.reward_xp,
                })

        if unlocked:
            self.db.commit()

        return unlocked

    def _check_requirement(
        self,
        achievement: Achievement,
        stats: PlayerStatistics,
    ) -> bool:
        """检查成就要求"""
        req = achievement.requirements
        req_type = achievement.requirement_type

        if req_type == 'count':
            # 计数类型要求
            for key, value in req.items():
                if hasattr(stats, key):
                    if getattr(stats, key) < value:
                        return False
            return True

        elif req_type == 'threshold':
            # 阈值类型要求
            metric = req.get('metric')
            threshold = req.get('threshold')
            if hasattr(stats, metric):
                return getattr(stats, metric) >= threshold
            return False

        elif req_type == 'condition':
            # 条件类型要求
            if req.get('condition') == 'first_blood':
                return stats.enemies_defeated >= 1
            elif req.get('condition') == 'survivor':
                return stats.characters_deceased == 0 and stats.games_completed >= 1
            elif req.get('condition') == 'lucky':
                return stats.critical_successes >= 10
            elif req.get('condition') == 'unlucky':
                return stats.critical_failures >= 10
            elif req.get('condition') == 'veteran'):
                return stats.games_completed >= 10
            elif req.get('condition') == 'insane'):
                return stats.times_went_insane >= 1

        return False

    def get_user_achievements(
        self,
        user_id: str,
    ) -> List[Dict[str, Any]]:
        """获取用户成就"""
        user_achievements = self.db.query(UserAchievement)\
            .filter(UserAchievement.user_id == user_id)\
            .join(Achievement)\
            .order_by(UserAchievement.unlocked_at.desc())\
            .all()

        return [
            {
                "id": ua.id,
                "achievement_id": ua.achievement_id,
                "achievement": {
                    "code": ua.achievement.code,
                    "name": ua.achievement.name,
                    "description": ua.achievement.description,
                    "icon": ua.achievement.icon,
                    "category": ua.achievement.category,
                    "rarity": ua.achievement.rarity,
                },
                "unlocked_at": ua.unlocked_at.isoformat(),
                "has_seen": ua.has_seen,
            }
            for ua in user_achievements
        ]

    def mark_as_seen(
        self,
        user_id: str,
        achievement_id: str,
    ):
        """标记成就为已查看"""
        ua = self.db.query(UserAchievement)\
            .filter(
                UserAchievement.user_id == user_id,
                UserAchievement.achievement_id == achievement_id,
            )\
            .first()

        if ua:
            ua.has_seen = True
            self.db.commit()

    def get_achievement_progress(
        self,
        user_id: str,
        room_id: str,
    ) -> List[Dict[str, Any]]:
        """获取成就进度"""
        stats = self.db.query(PlayerStatistics)\
            .filter(
                PlayerStatistics.user_id == user_id,
                PlayerStatistics.room_id == room_id,
            )\
            .first()

        if not stats:
            return []

        # 获取未解锁的成就
        existing_ids = [
            ua.achievement_id
            for ua in self.db.query(UserAchievement)
                .filter(UserAchievement.user_id == user_id)
                .all()
        ]

        achievements = self.db.query(Achievement)\
            .filter(~Achievement.id.in_(existing_ids))\
            .all()

        progress = []
        for achievement in achievements:
            req = achievement.requirements
            req_type = achievement.requirement_type

            current = 0
            total = 0

            if req_type == 'threshold':
                metric = req.get('metric')
                total = req.get('threshold', 1)
                if hasattr(stats, metric):
                    current = getattr(stats, metric)

            elif req_type == 'count':
                # 取第一个要求作为参考
                for key, value in req.items():
                    if hasattr(stats, key):
                        current = getattr(stats, key)
                        total = value
                        break

            progress.append({
                "id": achievement.id,
                "code": achievement.code,
                "name": achievement.name if not achievement.is_hidden else "???",
                "description": achievement.description if not achievement.is_hidden else "继续探索解锁",
                "icon": achievement.icon,
                "rarity": achievement.rarity,
                "current": min(current, total),
                "total": total,
                "percentage": int(min(current, total) / total * 100) if total > 0 else 0,
            })

        return sorted(progress, key=lambda x: x['percentage'], reverse=True)
```

---

## 预设成就

```python
# app/services/achievement/presets.py
PRESET_ACHIEVEMENTS = [
    {
        "code": "first_blood",
        "name": "首杀",
        "description": "击败你的第一个敌人",
        "icon": "⚔️",
        "category": "combat",
        "rarity": "common",
        "requirements": {"condition": "first_blood"},
        "requirement_type": "condition",
        "reward_xp": 10,
    },
    {
        "code": "survivor",
        "name": "幸存者",
        "description": "完成一个游戏且角色存活",
        "icon": "🎖️",
        "category": "survival",
        "rarity": "common",
        "requirements": {"condition": "survivor"},
        "requirement_type": "condition",
        "reward_xp": 20,
    },
    {
        "code": "veteran",
        "name": "老手",
        "description": "完成 10 个游戏",
        "icon": "🎖️",
        "category": "special",
        "rarity": "rare",
        "requirements": {"condition": "veteran"},
        "requirement_type": "condition",
        "reward_xp": 50,
    },
    {
        "code": "lucky",
        "name": "幸运儿",
        "description": "获得 10 次大成功",
        "icon": "🍀",
        "category": "special",
        "rarity": "rare",
        "requirements": {"condition": "lucky"},
        "requirement_type": "condition",
        "reward_xp": 30,
    },
    {
        "code": "unlucky",
        "name": "不幸者",
        "description": "遭遇 10 次大失败",
        "icon": "😵",
        "category": "special",
        "rarity": "rare",
        "requirements": {"condition": "unlucky"},
        "requirement_type": "condition",
        "reward_xp": 30,
    },
    {
        "code": "insane",
        "name": "疯狂边缘",
        "description": "角色陷入疯狂",
        "icon": "🌀",
        "category": "survival",
        "rarity": "common",
        "requirements": {"condition": "insane"},
        "requirement_type": "condition",
        "reward_xp": 10,
    },
    {
        "code": "monster_hunter",
        "name": "怪物猎人",
        "description": "击败 100 个敌人",
        "icon": "🏹",
        "category": "combat",
        "rarity": "epic",
        "requirements": {"metric": "enemies_defeated", "threshold": 100},
        "requirement_type": "threshold",
        "reward_xp": 100,
    },
    {
        "code": "dice_roller",
        "name": "掷骰大师",
        "description": "掷骰 1000 次",
        "icon": "🎲",
        "category": "special",
        "rarity": "epic",
        "requirements": {"metric": "dice_rolls_total", "threshold": 1000},
        "requirement_type": "threshold",
        "reward_xp": 75,
    },
]
```

---

## 成就 API

```python
# app/api/achievements.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.achievement import AchievementService

router = APIRouter(prefix="/achievements", tags=["achievements"])

@router.get("")
async def get_user_achievements(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取用户成就"""
    service = AchievementService(db)
    achievements = service.get_user_achievements(current_user.id)

    return {
        "achievements": achievements,
        "total": len(achievements),
    }

@router.get("/progress")
async def get_achievement_progress(
    room_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取成就进度"""
    service = AchievementService(db)
    progress = service.get_achievement_progress(current_user.id, room_id)

    return {"progress": progress}

@router.post("/{achievement_id}/seen")
async def mark_achievement_seen(
    achievement_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """标记成就为已查看"""
    service = AchievementService(db)
    service.mark_as_seen(current_user.id, achievement_id)

    return {"message": "已标记"}
```

---

## 排行榜 API

```python
# app/api/leaderboard.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.statistics import StatisticsService

router = APIRouter(prefix="/leaderboard", tags=["leaderboard"])

@router.get("/room/{room_id}")
async def get_room_leaderboard(
    room_id: str,
    metric: str = Query("games_completed", description="排序指标"),
    limit: int = Query(10, ge=1, le=100),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间排行榜"""
    service = StatisticsService(db)
    leaderboard = service.get_room_leaderboard(room_id, metric, limit)

    return {
        "room_id": room_id,
        "metric": metric,
        "leaderboard": leaderboard,
    }

@router.get("/global")
async def get_global_leaderboard(
    metric: str = Query("games_completed", description="排序指标"),
    limit: int = Query(50, ge=1, le=100),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取全局排行榜"""
    service = StatisticsService(db)
    leaderboard = service.get_global_leaderboard(metric, limit)

    return {
        "metric": metric,
        "leaderboard": leaderboard,
    }
```

---

## 前端统计组件

```tsx
// frontend/src/components/game/PlayerStatistics.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Trophy, Target, Skull, Dices } from 'lucide-react'

interface Statistics {
  games_played: number
  games_completed: number
  characters_deceased: number
  characters_insane: number
  dice_rolls_total: number
  critical_successes: number
  critical_failures: number
  enemies_defeated: number
  damage_dealt: number
}

export function PlayerStatistics({ userId, roomId }: { userId: string; roomId: string }) {
  const [stats, setStats] = useState<Statistics | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchStatistics()
  }, [userId, roomId])

  const fetchStatistics = async () => {
    try {
      const response = await fetch(`/api/statistics/${roomId}/${userId}`)
      if (response.ok) {
        const data = await response.json()
        setStats(data.statistics)
      }
    } catch (error) {
      console.error('Failed to fetch statistics:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div>加载中...</div>
  }

  if (!stats) {
    return <div>暂无统计数据</div>
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <StatCard
        title="游戏场次"
        value={stats.games_completed}
        total={stats.games_played}
        icon={<Trophy className="h-4 w-4" />}
      />
      <StatCard
        title="角色死亡"
        value={stats.characters_deceased}
        icon={<Skull className="h-4 w-4" />}
      />
      <StatCard
        title="击败敌人"
        value={stats.enemies_defeated}
        icon={<Target className="h-4 w-4" />}
      />
      <StatCard
        title="掷骰次数"
        value={stats.dice_rolls_total}
        icon={<Dices className="h-4 w-4" />}
      />
    </div>
  )
}

function StatCard({
  title,
  value,
  total,
  icon,
}: {
  title: string
  value: number
  total?: number
  icon?: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium flex items-center">
          {icon}
          <span className="ml-2">{title}</span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {total !== undefined && (
          <div className="text-xs text-muted-foreground">
            共 {total} 场
          </div>
        )}
      </CardContent>
    </Card>
  )
}
```

---

## 成就展示组件

```tsx
// frontend/src/components/game/AchievementDisplay.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Trophy, Lock, Star } from 'lucide-react'
import { Progress } from '@/components/ui/progress'

interface Achievement {
  id: string
  code: string
  name: string
  description: string
  icon: string
  category: string
  rarity: string
  current?: number
  total?: number
  percentage?: number
  unlocked?: boolean
}

const RARITY_COLORS = {
  common: 'bg-gray-500',
  rare: 'bg-blue-500',
  epic: 'bg-purple-500',
  legendary: 'bg-yellow-500',
}

export function AchievementDisplay({ roomId }: { roomId: string }) {
  const [achievements, setAchievements] = useState<Achievement[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchAchievements()
  }, [roomId])

  const fetchAchievements = async () => {
    try {
      const [unlockedRes, progressRes] = await Promise.all([
        fetch('/api/achievements'),
        fetch(`/api/achievements/progress?room_id=${roomId}`),
      ])

      if (unlockedRes.ok && progressRes.ok) {
        const unlockedData = await unlockedRes.json()
        const progressData = await progressRes.json()

        const unlocked = unlockedData.achievements.map((a: Achievement) => ({
          ...a.achievement,
          unlocked: true,
        }))

        setAchievements([...unlocked, ...progressData.progress])
      }
    } catch (error) {
      console.error('Failed to fetch achievements:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div>加载中...</div>
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">成就</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {achievements.map((achievement) => (
          <AchievementCard key={achievement.id} achievement={achievement} />
        ))}
      </div>
    </div>
  )
}

function AchievementCard({ achievement }: { achievement: Achievement }) {
  const isUnlocked = achievement.unlocked

  return (
    <Card className={`${isUnlocked ? '' : 'opacity-60'}`}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm flex items-center">
            <span className="text-2xl mr-2">
              {isUnlocked ? achievement.icon : <Lock className="h-6 w-6" />}
            </span>
            {achievement.name}
          </CardTitle>
          <Badge className={RARITY_COLORS[achievement.rarity as keyof typeof RARITY_COLORS]}>
            {achievement.rarity}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground mb-2">
          {achievement.description}
        </p>
        {!isUnlocked && achievement.total !== undefined && (
          <div className="space-y-1">
            <Progress value={achievement.percentage} />
            <div className="text-xs text-muted-foreground">
              {achievement.current} / {achievement.total}
            </div>
          </div>
        )}
        {isUnlocked && (
          <div className="flex items-center text-sm text-yellow-600">
            <Star className="h-4 w-4 mr-1" />
            已解锁
          </div>
        )}
      </CardContent>
    </Card>
  )
}
```

---

## 排行榜组件

```tsx
// frontend/src/components/game/Leaderboard.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Trophy, Medal } from 'lucide-react'

interface LeaderboardEntry {
  user_id: string
  username: string
  value: number
  rank: number
}

const METRICS = [
  { key: 'games_completed', label: '完成场次' },
  { key: 'enemies_defeated', label: '击败敌人' },
  { key: 'dice_rolls_total', label: '掷骰次数' },
  { key: 'critical_successes', label: '大成功' },
]

export function Leaderboard({ roomId }: { roomId: string }) {
  const [roomLeaderboard, setRoomLeaderboard] = useState<Record<string, LeaderboardEntry[]>>({})
  const [globalLeaderboard, setGlobalLeaderboard] = useState<Record<string, LeaderboardEntry[]>>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchLeaderboards()
  }, [roomId])

  const fetchLeaderboards = async () => {
    try {
      const promises = METRICS.map(async (metric) => {
        const [roomRes, globalRes] = await Promise.all([
          fetch(`/api/leaderboard/room/${roomId}?metric=${metric.key}`),
          fetch(`/api/leaderboard/global?metric=${metric.key}`),
        ])

        const roomData = roomRes.ok ? await roomRes.json() : null
        const globalData = globalRes.ok ? await globalRes.json() : null

        return {
          key: metric.key,
          room: roomData?.leaderboard || [],
          global: globalData?.leaderboard || [],
        }
      })

      const results = await Promise.all(promises)

      const roomData: Record<string, LeaderboardEntry[]> = {}
      const globalData: Record<string, LeaderboardEntry[]> = {}

      results.forEach((result) => {
        roomData[result.key] = result.room
        globalData[result.key] = result.global
      })

      setRoomLeaderboard(roomData)
      setGlobalLeaderboard(globalData)
    } catch (error) {
      console.error('Failed to fetch leaderboards:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div>加载中...</div>
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center">
          <Trophy className="h-5 w-5 mr-2" />
          排行榜
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="room">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="room">房间排行</TabsTrigger>
            <TabsTrigger value="global">全球排行</TabsTrigger>
          </TabsList>

          <TabsContent value="room" className="space-y-4">
            {METRICS.map((metric) => (
              <MetricLeaderboard
                key={metric.key}
                title={metric.label}
                data={roomLeaderboard[metric.key] || []}
              />
            ))}
          </TabsContent>

          <TabsContent value="global" className="space-y-4">
            {METRICS.map((metric) => (
              <MetricLeaderboard
                key={metric.key}
                title={metric.label}
                data={globalLeaderboard[metric.key] || []}
              />
            ))}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

function MetricLeaderboard({
  title,
  data,
}: {
  title: string
  data: LeaderboardEntry[]
}) {
  return (
    <div>
      <h4 className="text-sm font-medium mb-2">{title}</h4>
      <div className="space-y-1">
        {data.slice(0, 5).map((entry) => (
          <div
            key={entry.user_id}
            className="flex items-center justify-between py-1 px-2 rounded hover:bg-muted"
          >
            <div className="flex items-center">
              {entry.rank <= 3 ? (
                <Medal className={`h-4 w-4 mr-2 ${
                  entry.rank === 1 ? 'text-yellow-500' :
                  entry.rank === 2 ? 'text-gray-400' :
                  'text-orange-600'
                }`} />
              ) : (
                <span className="w-6 text-center text-sm text-muted-foreground mr-2">
                  {entry.rank}
                </span>
              )}
              <span className="text-sm">{entry.username}</span>
            </div>
            <span className="text-sm font-medium">{entry.value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/statistics.py` | 创建 | 统计数据模型 |
| `app/db/models/achievement.py` | 创建 | 成就模型 |
| `app/services/statistics.py` | 创建 | 统计服务 |
| `app/services/achievement.py` | 创建 | 成就服务 |
| `app/services/achievement/presets.py` | 创建 | 预设成就 |
| `app/api/achievements.py` | 创建 | 成就 API |
| `app/api/leaderboard.py` | 创建 | 排行榜 API |
| `frontend/src/components/game/PlayerStatistics.tsx` | 创建 | 统计展示组件 |
| `frontend/src/components/game/AchievementDisplay.tsx` | 创建 | 成就展示组件 |
| `frontend/src/components/game/Leaderboard.tsx` | 创建 | 排行榜组件 |

---

## 验收标准

- [ ] 统计数据准确记录
- [ ] 成就解锁逻辑正确
- [ ] 排行榜排序有效
- [ ] 成就进度显示清晰
- [ ] 隐藏成就正常隐藏
- [ ] 成就通知及时推送

---

## 参考文档

- M1-001: 角色系统
- Steam 成就系统设计

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
