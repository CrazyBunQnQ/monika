# M1-080: 事件日志系统

**任务类型**: backend
**预估工时**: 2.5h
**依赖**: M1-001, M1-003
**状态**: [ ]

---

## 子任务拆解

### 1.1 日志数据模型 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-080-01 | [ ] 创建 `app/core/logger.py` | [ ] |
| M1-080-02 | [ ] 定义 `LogLevel` 枚举 | [ ] |
| M1-080-03 | [ ] 定义 `LogCategory` 枚举 | [ ] |
| M1-080-04 | [ ] 定义 `GameLogEntry` | [ ] |

```python
# app/core/logger.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any, Generic, TypeVar
from datetime import datetime
import json

T = TypeVar("T")

class LogLevel(str, Enum):
    """日志级别"""
    DEBUG = "debug"
    INFO = "info"
    WARNING = "warning"
    ERROR = "error"
    CRITICAL = "critical"

class LogCategory(str, Enum):
    """日志分类"""
    NARRATIVE = "narrative"       # 叙述
    ACTION = "action"             # 动作
    CHECK = "check"               # 检定
    COMBAT = "combat"             # 战斗
    CHASE = "chase"               # 追逐
    SOCIAL = "social"             # 社交
    SAN = "san"                   # SAN
    MOVEMENT = "movement"         # 移动
    ITEM = "item"                 # 物品
    SYSTEM = "system"             # 系统
    CHAT = "chat"                 # 聊天
    DISCOVERY = "discovery"       # 发现

@dataclass
class LogMetadata:
    """日志元数据"""
    # 角色信息
    character_id: Optional[int] = None
    character_name: Optional[str] = None

    # 位置信息
    location: Optional[str] = None

    # 检定相关
    skill_key: Optional[str] = None
    skill_name: Optional[str] = None
    roll_value: Optional[int] = None
    target_value: Optional[int] = None
    success_level: Optional[str] = None

    # 战斗相关
    combat_id: Optional[int] = None
    damage: Optional[int] = None
    damage_type: Optional[str] = None

    # 扩展数据
    extra: Dict[str, Any] = field(default_factory=dict)

    def to_json(self) -> str:
        return json.dumps(self.extra)

    @classmethod
    def from_json(cls, json_str: str) -> "LogMetadata":
        if not json_str:
            return cls()
        return cls(extra=json.loads(json_str))

@dataclass
class GameLogEntry:
    """游戏日志条目"""
    id: Optional[int] = None
    session_id: int

    # 内容
    category: LogCategory
    level: LogLevel = LogLevel.INFO

    # 文本
    title: str
    content: str

    # 角色
    character_id: Optional[int] = None
    narrator: bool = False  # 是否为守密人叙述

    # 元数据
    metadata: LogMetadata = field(default_factory=LogMetadata)

    # 时间
    timestamp: datetime = field(default_factory=datetime.utcnow)
    order_index: int = 0

    # 标记
    is_important: bool = False
    tags: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "session_id": self.session_id,
            "category": self.category.value,
            "level": self.level.value,
            "title": self.title,
            "content": self.content,
            "character_id": self.character_id,
            "narrator": self.narrator,
            "metadata": {
                "character_name": self.metadata.character_name,
                "location": self.metadata.location,
                "skill_key": self.metadata.skill_key,
                "skill_name": self.metadata.skill_name,
                "roll_value": self.metadata.roll_value,
                "target_value": self.metadata.target_value,
                "success_level": self.metadata.success_level,
            },
            "timestamp": self.timestamp.isoformat(),
            "order_index": self.order_index,
            "is_important": self.is_important,
            "tags": self.tags
        }
```

---

### 1.2 日志服务 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-080-05 | [ ] 创建 `app/services/logger.py` | [ ] |
| M1-080-06 | [ ] 实现 `add_entry()` | [ ] |
| M1-080-07 | [ ] 实现 `get_session_logs()` | [ ] |
| M1-080-08 | [ ] 实现 `search_logs()` | [ ] |
| M1-080-09 | [ ] 实现 `add_check_result()` | [ ] |

```python
# app/services/logger.py
from typing import Optional, List, Dict, Any, Tuple
from datetime import datetime, timedelta
from sqlmodel import Session, select
from app.core.logger import (
    GameLogEntry, LogCategory, LogLevel, LogMetadata
)

class LoggerService:
    """日志服务"""

    def add_entry(
        self,
        session_id: int,
        category: LogCategory,
        title: str,
        content: str,
        character_id: Optional[int] = None,
        character_name: Optional[str] = None,
        level: LogLevel = LogLevel.INFO,
        metadata: Optional[LogMetadata] = None,
        is_important: bool = False,
        tags: Optional[List[str]] = None,
        **kwargs
    ) -> GameLogEntry:
        """添加日志条目"""
        entry = GameLogEntry(
            session_id=session_id,
            category=category,
            title=title,
            content=content,
            character_id=character_id,
            level=level,
            metadata=metadata or LogMetadata(
                character_name=character_name,
                **kwargs
            ),
            is_important=is_important,
            tags=tags or []
        )

        return entry

    def add_narrative(
        self,
        session_id: int,
        content: str,
        location: Optional[str] = None,
        is_important: bool = False
    ) -> GameLogEntry:
        """添加守密人叙述"""
        return self.add_entry(
            session_id=session_id,
            category=LogCategory.NARRATIVE,
            title="叙述",
            content=content,
            narrator=True,
            location=location,
            is_important=is_important
        )

    def add_action(
        self,
        session_id: int,
        character_id: int,
        character_name: str,
        action: str,
        content: str
    ) -> GameLogEntry:
        """添加角色动作"""
        return self.add_entry(
            session_id=session_id,
            category=LogCategory.ACTION,
            title=f"{character_name} 的动作",
            content=content,
            character_id=character_id,
            character_name=character_name
        )

    def add_check_result(
        self,
        session_id: int,
        character_id: int,
        character_name: str,
        skill_key: str,
        skill_name: str,
        roll_value: int,
        target_value: int,
        success_level: str,
        content: str,
        metadata: Optional[Dict] = None
    ) -> GameLogEntry:
        """添加检定结果"""
        meta = LogMetadata(
            character_id=character_id,
            character_name=character_name,
            skill_key=skill_key,
            skill_name=skill_name,
            roll_value=roll_value,
            target_value=target_value,
            success_level=success_level,
            extra=metadata or {}
        )

        # 根据成功等级设置日志级别
        level = LogLevel.INFO
        if success_level in ["critical"]:
            level = LogLevel.DEBUG  # 好结果
        elif success_level in ["fumble"]:
            level = LogLevel.WARNING  # 坏结果

        return self.add_entry(
            session_id=session_id,
            category=LogCategory.CHECK,
            title=f"检定 - {skill_name}",
            content=content,
            character_id=character_id,
            character_name=character_name,
            level=level,
            metadata=meta
        )

    def add_combat_event(
        self,
        session_id: int,
        combat_id: int,
        character_id: Optional[int],
        character_name: Optional[str],
        action: str,
        result: str,
        damage: Optional[int] = None,
        damage_type: Optional[str] = None
    ) -> GameLogEntry:
        """添加战斗事件"""
        meta = LogMetadata(
            character_id=character_id,
            character_name=character_name,
            combat_id=combat_id,
            damage=damage,
            damage_type=damage_type
        )

        content = f"{action}: {result}"
        if damage:
            content += f" 造成 {damage} 点{damage_type or ''}伤害"

        return self.add_entry(
            session_id=session_id,
            category=LogCategory.COMBAT,
            title="战斗事件",
            content=content,
            character_id=character_id,
            character_name=character_name,
            metadata=meta
        )
```

---

### 1.3 日志查询 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-080-10 | [ ] 实现 `get_session_logs()` | [ ] |
| M1-080-11 | [ ] 实现 `get_character_logs()` | [ ] |
| M1-080-12 | [ ] 实现 `search_logs()` | [ ] |
| M1-080-13 | [ ] 实现 `get_discovery_logs()` | [ ] |

```python
class LoggerService:
    # ...

    def get_session_logs(
        self,
        session: Session,
        session_id: int,
        category: Optional[LogCategory] = None,
        character_id: Optional[int] = None,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
        limit: int = 100,
        offset: int = 0,
        important_only: bool = False
    ) -> Tuple[List[GameLogEntry], int]:
        """获取会话日志"""
        statement = select(GameLogEntry).where(
            GameLogEntry.session_id == session_id
        )

        if category:
            statement = statement.where(GameLogEntry.category == category)

        if character_id:
            statement = statement.where(
                GameLogEntry.character_id == character_id
            )

        if start_time:
            statement = statement.where(
                GameLogEntry.timestamp >= start_time
            )

        if end_time:
            statement = statement.where(
                GameLogEntry.timestamp <= end_time
            )

        if important_only:
            statement = statement.where(GameLogEntry.is_important == True)

        # 统计总数
        total = len(session.exec(statement).all())

        # 分页
        statement = statement.order_by(
            GameLogEntry.timestamp.asc(),
            GameLogEntry.order_index.asc()
        )
        statement = statement.offset(offset).limit(limit)

        logs = session.exec(statement).all()

        return logs, total

    def get_character_logs(
        self,
        session: Session,
        character_id: int,
        limit: int = 50
    ) -> List[GameLogEntry]:
        """获取角色相关日志"""
        statement = select(GameLogEntry).where(
            GameLogEntry.character_id == character_id
        ).order_by(GameLogEntry.timestamp.desc()).limit(limit)

        return session.exec(statement).all()

    def search_logs(
        self,
        session: Session,
        session_id: int,
        keyword: str,
        category: Optional[LogCategory] = None
    ) -> List[GameLogEntry]:
        """搜索日志"""
        statement = select(GameLogEntry).where(
            GameLogEntry.session_id == session_id,
            GameLogEntry.title.contains(keyword) |
            GameLogEntry.content.contains(keyword)
        )

        if category:
            statement = statement.where(GameLogEntry.category == category)

        return session.exec(statement).all()

    def get_discovery_logs(
        self,
        session: Session,
        session_id: int
    ) -> List[GameLogEntry]:
        """获取发现类日志"""
        statement = select(GameLogEntry).where(
            GameLogEntry.session_id == session_id,
            GameLogEntry.category == LogCategory.DISCOVERY
        ).order_by(GameLogEntry.timestamp.asc())

        return session.exec(statement).all()
```

---

### 1.4 日志摘要生成 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-080-14 | [ ] 实现 `generate_session_summary()` | [ ] |
| M1-080-15 | [ ] 实现 `generate_daily_summary()` | [ ] |

```python
class LoggerService:
    # ...

    def generate_session_summary(
        self,
        session: Session,
        session_id: int
    ) -> Dict[str, Any]:
        """生成会话摘要"""
        logs, total = self.get_session_logs(session, session_id, limit=10000)

        # 统计各类日志数量
        category_counts: Dict[str, int] = {}
        for log in logs:
            cat = log.category.value
            category_counts[cat] = category_counts.get(cat, 0) + 1

        # 重要事件
        important = [log for log in logs if log.is_important]

        # 检定统计
        checks = [log for log in logs if log.category == LogCategory.CHECK]
        check_stats = self._calculate_check_stats(checks)

        # 战斗统计
        combats = [log for log in logs if log.category == LogCategory.COMBAT]
        combat_stats = self._calculate_combat_stats(combats)

        return {
            "session_id": session_id,
            "total_entries": total,
            "duration": self._calculate_duration(logs),
            "category_counts": category_counts,
            "important_events": [log.to_dict() for log in important[:5]],
            "check_stats": check_stats,
            "combat_stats": combat_stats
        }

    def _calculate_check_stats(
        self,
        checks: List[GameLogEntry]
    ) -> Dict[str, Any]:
        """计算检定统计"""
        if not checks:
            return {"total": 0}

        levels: Dict[str, int] = {}
        for check in checks:
            level = check.metadata.success_level
            levels[level] = levels.get(level, 0) + 1

        return {
            "total": len(checks),
            "by_level": levels,
            "success_rate": (
                sum(levels.get(k, 0) for k in ["critical", "extreme", "hard", "regular"])
                / len(checks) * 100 if checks else 0
            )
        }

    def _calculate_combat_stats(
        self,
        combats: List[GameLogEntry]
    ) -> Dict[str, Any]:
        """计算战斗统计"""
        total_damage = sum(
            log.metadata.damage or 0 for log in combats
        )
        combat_count = len(combats)

        return {
            "total_events": combat_count,
            "total_damage": total_damage,
            "average_damage": total_damage / combat_count if combat_count > 0 else 0
        }

    def _calculate_duration(
        self,
        logs: List[GameLogEntry]
    ) -> Optional[int]:
        """计算持续时间（分钟）"""
        if len(logs) < 2:
            return None

        first = min(log.timestamp for log in logs)
        last = max(log.timestamp for log in logs)
        return int((last - first).total_seconds() / 60)
```

---

### 1.5 日志 API (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-080-16 | [ ] 创建 `app/api/logger.py` | [ ] |
| M1-080-17 | [ ] 实现 POST /logs | [ ] |
| M1-080-18 | [ ] 实现 GET /logs/session | [ ] |
| M1-080-19 | [ ] 实现 GET /logs/summary | [ ] |

```python
# app/api/logger.py
from fastapi import APIRouter, Depends, HTTPException, Query
from pydantic import BaseModel
from typing import Optional, List
from datetime import datetime
from sqlmodel import Session
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.db.connection import get_session
from app.services.logger import LoggerService
from app.core.logger import LogCategory, LogLevel

router = APIRouter(prefix="/logs", tags=["日志"])

class CreateLogRequest(BaseModel):
    """创建日志请求"""
    session_id: int
    category: str
    title: str
    content: str
    character_id: Optional[int] = None
    character_name: Optional[str] = None
    is_important: bool = False
    tags: Optional[List[str]] = None

class LogResponse(BaseModel):
    """日志响应"""
    id: int
    session_id: int
    category: str
    title: str
    content: str
    character_id: Optional[int]
    timestamp: datetime
    is_important: bool

@router.post("")
async def create_log(
    request: CreateLogRequest,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """创建日志条目"""
    try:
        category = LogCategory(request.category)
    except ValueError:
        raise HTTPException(status_code=400, detail="无效的日志分类")

    service = LoggerService()
    entry = service.add_entry(
        session_id=request.session_id,
        category=category,
        title=request.title,
        content=request.content,
        character_id=request.character_id,
        character_name=request.character_name,
        is_important=request.is_important,
        tags=request.tags
    )

    # 保存到数据库
    session.add(entry)
    session.commit()
    session.refresh(entry)

    return entry.to_dict()

@router.get("/session/{session_id}")
async def get_session_logs(
    session_id: int,
    category: Optional[str] = None,
    character_id: Optional[int] = None,
    important_only: bool = False,
    limit: int = 100,
    offset: int = 0,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """获取会话日志"""
    service = LoggerService()

    cat_enum = None
    if category:
        try:
            cat_enum = LogCategory(category)
        except ValueError:
            raise HTTPException(status_code=400, detail="无效的日志分类")

    logs, total = service.get_session_logs(
        session=session,
        session_id=session_id,
        category=cat_enum,
        character_id=character_id,
        important_only=important_only,
        limit=limit,
        offset=offset
    )

    return {
        "logs": [log.to_dict() for log in logs],
        "total": total,
        "limit": limit,
        "offset": offset
    }

@router.get("/session/{session_id}/summary")
async def get_session_summary(
    session_id: int,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """获取会话摘要"""
    service = LoggerService()
    return service.generate_session_summary(session, session_id)
```

---

### 1.6 单元测试 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-080-20 | [ ] 创建 `tests/test_logger.py` | [ ] |
| M1-080-21 | [ ] 测试日志创建 | [ ] |
| M1-080-22 | [ ] 测试日志查询 | [ ] |
| M1-080-23 | [ ] 测试摘要生成 | [ ] |

```python
# tests/test_logger.py
import pytest
from datetime import datetime, timedelta
from app.services.logger import LoggerService
from app.core.logger import (
    GameLogEntry, LogCategory, LogLevel, LogMetadata
)

class TestLoggerService:
    def test_add_entry(self):
        """测试添加日志"""
        service = LoggerService()

        entry = service.add_entry(
            session_id=1,
            category=LogCategory.NARRATIVE,
            title="测试标题",
            content="测试内容",
            character_id=1,
            character_name="测试角色"
        )

        assert entry.session_id == 1
        assert entry.category == LogCategory.NARRATIVE
        assert entry.title == "测试标题"
        assert entry.content == "测试内容"

    def test_add_check_result(self):
        """测试添加检定结果"""
        service = LoggerService()

        entry = service.add_check_result(
            session_id=1,
            character_id=1,
            character_name="调查员",
            skill_key="library_use",
            skill_name="图书馆使用",
            roll_value=35,
            target_value=50,
            success_level="success",
            content="在图书馆查找线索"
        )

        assert entry.category == LogCategory.CHECK
        assert entry.metadata.roll_value == 35
        assert entry.metadata.success_level == "success"

    def test_add_combat_event(self):
        """测试添加战斗事件"""
        service = LoggerService()

        entry = service.add_combat_event(
            session_id=1,
            combat_id=1,
            character_id=1,
            character_name="调查员",
            action="攻击",
            result="命中",
            damage=5,
            damage_type="blunt"
        )

        assert entry.category == LogCategory.COMBAT
        assert entry.metadata.damage == 5
        assert "5" in entry.content

    def test_entry_to_dict(self):
        """测试转换为字典"""
        service = LoggerService()

        entry = service.add_entry(
            session_id=1,
            category=LogCategory.ACTION,
            title="测试",
            content="内容"
        )

        data = entry.to_dict()

        assert data["session_id"] == 1
        assert data["category"] == "action"
        assert "timestamp" in data
```

---

## 验收标准

- [ ] 日志条目结构完整
- [ ] 支持多种日志分类
- [ ] 日志查询支持分页和筛选
- [ ] 会话摘要生成正确
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/logger.py` | 创建 | 日志数据模型 |
| `app/services/logger.py` | 创建 | 日志服务 |
| `app/api/logger.py` | 创建 | 日志 API |
| `tests/test_logger.py` | 创建 | 单元测试 |
