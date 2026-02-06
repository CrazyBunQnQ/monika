# M1-022 实现删除角色卡 DELETE /characters/:id

## 概述
实现删除角色卡的 RESTful API 端点,支持软删除和硬删除,确保数据安全和完整性。

## 验收标准
- [ ] 实现 DELETE /characters/:id 端点
- [ ] 支持软删除(默认)
- [ ] 支持硬删除(查询参数)
- [ ] 验证所有权权限
- [ ] 处理关联数据(会话引用)
- [ ] 返回适当的响应码

## 技术方案

### API 端点

```python
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from typing import Literal

router = APIRouter(prefix="/characters", tags=["characters"])

@router.delete("/{character_id}")
async def delete_character(
    character_id: str,
    delete_type: Literal["soft", "hard"] = Query("soft", description="删除类型"),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    删除角色卡

    参数:
    - character_id: 角色 ID
    - delete_type: soft(软删除) | hard(硬删除)

    返回:
    - 200: 删除成功
    - 403: 无权限
    - 404: 角色不存在
    """
    # 查询角色
    character = db.query(Character).filter(
        Character.id == character_id
    ).first()

    if not character:
        raise HTTPException(status_code=404, detail="角色不存在")

    # 权限检查
    if character.user_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权限删除此角色")

    # 软删除
    if delete_type == "soft":
        if character.deleted_at:
            raise HTTPException(status_code=400, detail="角色已被删除")

        character.deleted_at = datetime.utcnow()
        db.commit()

        return {
            "message": "角色已删除",
            "character_id": character_id,
            "delete_type": "soft"
        }

    # 硬删除
    elif delete_type == "hard":
        # 检查关联
        active_sessions = db.query(Session).filter(
            Session.characters.contains(character_id),
            Session.status != "completed"
        ).count()

        if active_sessions > 0:
            raise HTTPException(
                status_code=400,
                detail=f"角色正在 {active_sessions} 个活跃会话中使用"
            )

        # 删除关联数据
        db.query(CharacterState).filter(
            CharacterState.character_id == character_id
        ).delete()

        # 删除角色
        db.delete(character)
        db.commit()

        return {
            "message": "角色已永久删除",
            "character_id": character_id,
            "delete_type": "hard"
        }
```

### 软删除实现

```python
from datetime import datetime
from sqlalchemy import Column, DateTime

class Character(Base):
    __tablename__ = "characters"

    # ... 其他字段 ...

    deleted_at: Optional[DateTime] = Column(
        DateTime,
        nullable=True,
        index=True,
        comment="软删除时间戳"
    )

    @property
    def is_deleted(self) -> bool:
        """是否已删除"""
        return self.deleted_at is not None
```

### 硬删除检查

```python
def check_character_deletion(character_id: str, db: Session) -> dict:
    """
    检查角色是否可以硬删除

    返回:
    {
        "can_delete": bool,
        "active_sessions": int,
        "reasons": List[str]
    }
    """
    reasons = []

    # 检查活跃会话
    active_sessions = db.query(Session).filter(
        Session.characters.contains(character_id),
        Session.status.in_(["active", "paused"])
    ).all()

    if active_sessions:
        reasons.append(f"正在 {len(active_sessions)} 个活跃会话中使用")

    # 检查最近的会话(7天内)
    recent_date = datetime.utcnow() - timedelta(days=7)
    recent_sessions = db.query(Session).filter(
        Session.characters.contains(character_id),
        Session.updated_at >= recent_date
    ).count()

    if recent_sessions > 0:
        reasons.append(f"最近 {recent_sessions} 天内有会话记录")

    return {
        "can_delete": len(reasons) == 0,
        "active_sessions": len(active_sessions),
        "recent_sessions": recent_sessions,
        "reasons": reasons
    }
```

### 权限验证

```python
def can_delete_character(
    character: Character,
    user: User
) -> bool:
    """
    检查用户是否有权限删除角色
    """
    # 角色所有者
    if character.user_id == user.id:
        return True

    # 管理员
    if user.is_admin:
        return True

    return False
```

### 批量删除

```python
@router.post("/delete-batch")
async def delete_characters_batch(
    character_ids: List[str],
    delete_type: Literal["soft", "hard"] = Query("soft"),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    批量删除角色卡

    返回:
    {
        "successful": List[str],
        "failed": List[{"id": str, "reason": str}]
    }
    """
    successful = []
    failed = []

    for character_id in character_ids:
        try:
            # 查询角色
            character = db.query(Character).filter(
                Character.id == character_id
            ).first()

            if not character:
                failed.append({
                    "id": character_id,
                    "reason": "角色不存在"
                })
                continue

            # 权限检查
            if character.user_id != current_user.id:
                failed.append({
                    "id": character_id,
                    "reason": "无权限"
                })
                continue

            # 删除
            if delete_type == "soft":
                character.deleted_at = datetime.utcnow()
                successful.append(character_id)
            else:
                check = check_character_deletion(character_id, db)
                if not check["can_delete"]:
                    failed.append({
                        "id": character_id,
                        "reason": ", ".join(check["reasons"])
                    })
                    continue

                db.delete(character)
                successful.append(character_id)

        except Exception as e:
            failed.append({
                "id": character_id,
                "reason": str(e)
            })

    db.commit()

    return {
        "total": len(character_ids),
        "successful_count": len(successful),
        "failed_count": len(failed),
        "successful": successful,
        "failed": failed
    }
```

### 响应格式

```python
# 成功响应
{
    "message": "角色已删除",
    "character_id": "abc123",
    "delete_type": "soft",
    "deleted_at": "2026-02-06T12:00:00Z"
}

# 错误响应
{
    "detail": "角色正在 2 个活跃会话中使用"
}

# 批量删除响应
{
    "total": 5,
    "successful_count": 3,
    "failed_count": 2,
    "successful": ["id1", "id2", "id3"],
    "failed": [
        {"id": "id4", "reason": "角色不存在"},
        {"id": "id5", "reason": "无权限"}
    ]
}
```

## 依赖关系
- 前置任务: M1-019 实现创建角色卡 POST /characters
- 被依赖: M1-028 实现角色卡列表组件

## 预估工时
2h
