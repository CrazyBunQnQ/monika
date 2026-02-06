# M4-016: 实现场景包评论

**任务ID**: M4-016
**任务名称**: 实现场景包评论
**预估时间**: 4 小时
**优先级**: P1
**依赖**: M4-015 (场景包评分)
**状态**: 待开始

---

## 任务概述

实现场景包的评论功能，支持用户发表、查看、回复、编辑、删除评论。支持富文本、表情，实现评论点赞功能，提供评论审核和敏感词过滤机制。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-016-01 | 设计评论数据模型和数据库表结构 | 0.5h | M4-015 | 待开始 |
| M4-016-02 | 实现评论CRUD服务 | 1h | M4-016-01 | 待开始 |
| M4-016-03 | 实现评论回复和嵌套结构 | 1h | M4-016-02 | 待开始 |
| M4-016-04 | 实现评论点赞和敏感词过滤 | 1h | M4-016-03 | 待开始 |
| M4-016-05 | 实现评论API和前端组件 | 0.5h | M4-016-04 | 待开始 |

**总预估时间**: 4 小时

---

## Python 后端实现

### 1. 数据库模型

```python
# backend/app/models/comment.py
from sqlalchemy import Column, Integer, String, DateTime, ForeignKey, Text, Boolean, Index
from sqlalchemy.orm import relationship
from datetime import datetime
import uuid

from app.db.base_class import Base

class ScenarioComment(Base):
    """场景包评论"""
    __tablename__ = "scenario_comments"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    scenario_id = Column(String(36), ForeignKey("scenarios.id"), nullable=False, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)

    # 评论内容
    content = Column(Text, nullable=False)
    rich_content = Column(Text, nullable=True)  # 富文本内容（JSON）

    # 回复结构
    parent_id = Column(String(36), ForeignKey("scenario_comments.id"), nullable=True, index=True)
    reply_to_user_id = Column(Integer, ForeignKey("users.id"), nullable=True)  # 回复的目标用户

    # 状态
    is_edited = Column(Boolean, default=False, nullable=False)
    is_deleted = Column(Boolean, default=False, nullable=False, index=True)
    is_pinned = Column(Boolean, default=False, nullable=False)  # 是否置顶
    is_hidden = Column(Boolean, default=False, nullable=False)  # 是否隐藏（审核/举报）

    # 点赞数
    like_count = Column(Integer, default=0, nullable=False)

    # 审核信息
    moderation_status = Column(String(20), default="pending", nullable=False)  # pending, approved, rejected
    moderated_by = Column(Integer, ForeignKey("users.id"), nullable=True)
    moderated_at = Column(DateTime, nullable=True)
    moderation_reason = Column(String(500), nullable=True)

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False, index=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    deleted_at = Column(DateTime, nullable=True)

    # 关系
    scenario = relationship("Scenario", back_populates="comments")
    user = relationship("User", foreign_keys=[user_id], back_populates="scenario_comments")
    parent = relationship("ScenarioComment", remote_side=[id], backref="replies")
    reply_to_user = relationship("User", foreign_keys=[reply_to_user_id])
    moderator = relationship("User", foreign_keys=[moderated_by])

    __table_args__ = (
        Index('ix_scenario_comments_scenario_created', 'scenario_id', 'created_at'),
        Index('ix_scenario_comments_parent_created', 'parent_id', 'created_at'),
    )

    def __repr__(self):
        return f"<ScenarioComment(id={self.id}, scenario={self.scenario_id}, user={self.user_id})>"

    @property
    def is_reply(self) -> bool:
        """是否为回复"""
        return self.parent_id is not None

    @property
    def excerpt(self) -> str:
        """获取评论摘要（前100字符）"""
        if self.is_deleted:
            return "[此评论已被删除]"
        return self.content[:100] + "..." if len(self.content) > 100 else self.content

class CommentLike(Base):
    """评论点赞记录"""
    __tablename__ = "comment_likes"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    comment_id = Column(String(36), ForeignKey("scenario_comments.id"), nullable=False, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)

    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)

    # 关系
    comment = relationship("ScenarioComment", backref="likes")
    user = relationship("User")

    # 唯一约束：每个用户对每条评论只能点赞一次
    __table_args__ = (
        Index('ix_comment_likes_comment_user', 'comment_id', 'user_id', unique=True),
    )
```

### 2. 评论服务

```python
# backend/app/services/comment_service.py
from typing import Optional, List, Dict, Any
from datetime import datetime
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, desc

from app.models.comment import ScenarioComment, CommentLike
from app.core.exceptions import ParseError

class SensitiveWordFilter:
    """敏感词过滤器"""

    def __init__(self):
        # 简单的敏感词列表（实际应该从数据库或配置文件加载）
        self.sensitive_words = set([
            # 这里应该配置实际的敏感词
            "敏感词1", "敏感词2"
        ])

    def filter(self, text: str) -> tuple[str, bool]:
        """
        过滤敏感词

        Args:
            text: 待过滤文本

        Returns:
            tuple: (过滤后的文本, 是否包含敏感词)
        """
        filtered = text
        has_sensitive = False

        for word in self.sensitive_words:
            if word in text:
                has_sensitive = True
                filtered = filtered.replace(word, "*" * len(word))

        return filtered, has_sensitive

class CommentService:
    """评论服务"""

    def __init__(self, db: Session):
        self.db = db
        self.word_filter = SensitiveWordFilter()

    def create_comment(
        self,
        scenario_id: str,
        user_id: int,
        content: str,
        parent_id: Optional[str] = None,
        rich_content: Optional[str] = None
    ) -> ScenarioComment:
        """
        创建评论

        Args:
            scenario_id: 场景包ID
            user_id: 用户ID
            content: 评论内容
            parent_id: 父评论ID（回复时）
            rich_content: 富文本内容（JSON）

        Returns:
            ScenarioComment: 评论记录

        Raises:
            ParseError: 创建失败时抛出
        """
        # 验证内容
        if not content or not content.strip():
            raise ParseError("评论内容不能为空")

        if len(content) > 5000:
            raise ParseError("评论内容不能超过5000字符")

        # 过滤敏感词
        filtered_content, has_sensitive = self.word_filter.filter(content)

        # 验证父评论
        reply_to_user_id = None
        if parent_id:
            parent_comment = self.db.query(ScenarioComment).filter(
                ScenarioComment.id == parent_id,
                ScenarioComment.scenario_id == scenario_id,
                ScenarioComment.is_deleted == False
            ).first()

            if not parent_comment:
                raise ParseError("父评论不存在")

            reply_to_user_id = parent_comment.user_id

        # 创建评论
        comment = ScenarioComment(
            scenario_id=scenario_id,
            user_id=user_id,
            content=filtered_content,
            rich_content=rich_content,
            parent_id=parent_id,
            reply_to_user_id=reply_to_user_id,
            moderation_status="auto_approved" if not has_sensitive else "pending"
        )

        self.db.add(comment)
        self.db.commit()
        self.db.refresh(comment)

        return comment

    def update_comment(
        self,
        comment_id: str,
        user_id: int,
        content: str,
        rich_content: Optional[str] = None
    ) -> Optional[ScenarioComment]:
        """
        更新评论

        Args:
            comment_id: 评论ID
            user_id: 用户ID
            content: 新内容
            rich_content: 富文本内容

        Returns:
            ScenarioComment: 更新后的评论，如果不存在返回None
        """
        comment = self.db.query(ScenarioComment).filter(
            ScenarioComment.id == comment_id,
            ScenarioComment.user_id == user_id,
            ScenarioComment.is_deleted == False
        ).first()

        if not comment:
            return None

        # 验证内容
        if not content or not content.strip():
            raise ParseError("评论内容不能为空")

        if len(content) > 5000:
            raise ParseError("评论内容不能超过5000字符")

        # 过滤敏感词
        filtered_content, _ = self.word_filter.filter(content)

        # 更新评论
        comment.content = filtered_content
        comment.rich_content = rich_content
        comment.is_edited = True
        comment.updated_at = datetime.utcnow()

        self.db.commit()
        self.db.refresh(comment)

        return comment

    def delete_comment(
        self,
        comment_id: str,
        user_id: int,
        is_moderator: bool = False
    ) -> bool:
        """
        删除评论（软删除）

        Args:
            comment_id: 评论ID
            user_id: 用户ID
            is_moderator: 是否为管理员

        Returns:
            bool: 是否删除成功
        """
        query = self.db.query(ScenarioComment).filter(
            ScenarioComment.id == comment_id,
            ScenarioComment.is_deleted == False
        )

        if not is_moderator:
            query = query.filter(ScenarioComment.user_id == user_id)

        comment = query.first()

        if not comment:
            return False

        comment.is_deleted = True
        comment.deleted_at = datetime.utcnow()
        comment.content = "[此评论已被删除]"

        self.db.commit()

        return True

    def get_comment(
        self,
        comment_id: str
    ) -> Optional[ScenarioComment]:
        """获取单条评论"""
        return self.db.query(ScenarioComment).filter(
            ScenarioComment.id == comment_id,
            ScenarioComment.is_deleted == False
        ).first()

    def get_scenario_comments(
        self,
        scenario_id: str,
        include_replies: bool = True,
        skip: int = 0,
        limit: int = 20,
        sort_by: str = "created_at"
    ) -> List[ScenarioComment]:
        """
        获取场景包的评论列表

        Args:
            scenario_id: 场景包ID
            include_replies: 是否包含回复
            skip: 跳过记录数
            limit: 返回记录数
            sort_by: 排序字段 (created_at, like_count)

        Returns:
            List[ScenarioComment]: 评论列表
        """
        query = self.db.query(ScenarioComment).filter(
            ScenarioComment.scenario_id == scenario_id,
            ScenarioComment.parent_id == None,  # 只获取顶级评论
            ScenarioComment.is_deleted == False,
            ScenarioComment.is_hidden == False
        )

        # 排序
        if sort_by == "like_count":
            query = query.order_by(desc(ScenarioComment.like_count), desc(ScenarioComment.created_at))
        else:
            query = query.order_by(desc(ScenarioComment.is_pinned), desc(ScenarioComment.created_at))

        comments = query.offset(skip).limit(limit).all()

        # 如果需要包含回复
        if include_replies:
            for comment in comments:
                replies = self.db.query(ScenarioComment).filter(
                    ScenarioComment.parent_id == comment.id,
                    ScenarioComment.is_deleted == False,
                    ScenarioComment.is_hidden == False
                ).order_by(desc(ScenarioComment.created_at)).limit(5).all()

                # 动态添加属性（不保存到数据库）
                comment.replies_list = replies

        return comments

    def get_comment_replies(
        self,
        comment_id: str,
        skip: int = 0,
        limit: int = 20
    ) -> List[ScenarioComment]:
        """获取评论的回复列表"""
        return self.db.query(ScenarioComment).filter(
            ScenarioComment.parent_id == comment_id,
            ScenarioComment.is_deleted == False,
            ScenarioComment.is_hidden == False
        ).order_by(desc(ScenarioComment.created_at)).offset(skip).limit(limit).all()

    def like_comment(
        self,
        comment_id: str,
        user_id: int
    ) -> tuple[bool, int]:
        """
        点赞/取消点赞评论

        Returns:
            tuple: (是否点赞, 当前点赞数)
        """
        # 检查是否已点赞
        existing_like = self.db.query(CommentLike).filter(
            CommentLike.comment_id == comment_id,
            CommentLike.user_id == user_id
        ).first()

        comment = self.db.query(ScenarioComment).filter(
            ScenarioComment.id == comment_id
        ).first()

        if not comment:
            raise ParseError("评论不存在")

        if existing_like:
            # 取消点赞
            self.db.delete(existing_like)
            comment.like_count = max(0, comment.like_count - 1)
            self.db.commit()
            return False, comment.like_count
        else:
            # 添加点赞
            like = CommentLike(
                comment_id=comment_id,
                user_id=user_id
            )
            self.db.add(like)
            comment.like_count += 1
            self.db.commit()
            return True, comment.like_count

    def get_user_liked_comments(
        self,
        user_id: int,
        comment_ids: List[str]
    ) -> set:
        """获取用户点赞的评论ID集合"""
        likes = self.db.query(CommentLike).filter(
            CommentLike.user_id == user_id,
            CommentLike.comment_id.in_(comment_ids)
        ).all()

        return {like.comment_id for like in likes}

    def get_user_comments(
        self,
        user_id: int,
        skip: int = 0,
        limit: int = 20
    ) -> List[ScenarioComment]:
        """获取用户的评论列表"""
        return self.db.query(ScenarioComment).filter(
            ScenarioComment.user_id == user_id,
            ScenarioComment.is_deleted == False
        ).order_by(
            desc(ScenarioComment.created_at)
        ).offset(skip).limit(limit).all()

    def pin_comment(
        self,
        comment_id: str,
        pinned: bool,
        scenario_id: str
    ) -> bool:
        """
        置顶/取消置顶评论

        Args:
            comment_id: 评论ID
            pinned: 是否置顶
            scenario_id: 场景包ID（用于验证）

        Returns:
            bool: 是否成功
        """
        comment = self.db.query(ScenarioComment).filter(
            ScenarioComment.id == comment_id,
            ScenarioComment.scenario_id == scenario_id
        ).first()

        if not comment:
            return False

        comment.is_pinned = pinned
        self.db.commit()

        return True

    def moderate_comment(
        self,
        comment_id: str,
        moderator_id: int,
        action: str,
        reason: Optional[str] = None
    ) -> bool:
        """
        审核评论

        Args:
            comment_id: 评论ID
            moderator_id: 管理员ID
            action: 操作 (approve, reject, hide)
            reason: 原因

        Returns:
            bool: 是否成功
        """
        comment = self.db.query(ScenarioComment).filter(
            ScenarioComment.id == comment_id
        ).first()

        if not comment:
            return False

        comment.moderated_by = moderator_id
        comment.moderated_at = datetime.utcnow()
        comment.moderation_reason = reason

        if action == "approve":
            comment.moderation_status = "approved"
            comment.is_hidden = False
        elif action == "reject":
            comment.moderation_status = "rejected"
            comment.is_hidden = True
        elif action == "hide":
            comment.is_hidden = True

        self.db.commit()

        return True

    def get_comment_statistics(
        self,
        scenario_id: str
    ) -> Dict[str, int]:
        """获取场景包评论统计"""
        total = self.db.query(ScenarioComment).filter(
            ScenarioComment.scenario_id == scenario_id,
            ScenarioComment.parent_id == None,
            ScenarioComment.is_deleted == False
        ).count()

        replies = self.db.query(ScenarioComment).filter(
            ScenarioComment.scenario_id == scenario_id,
            ScenarioComment.parent_id != None,
            ScenarioComment.is_deleted == False
        ).count()

        return {
            "total_comments": total,
            "total_replies": replies
        }
```

### 3. API 路由

```python
# backend/app/api/v1/endpoints/comment.py
from fastapi import APIRouter, Depends, HTTPException, Query
from typing import Optional, List
from sqlalchemy.orm import Session
from pydantic import BaseModel

from app.api.deps import get_current_user, get_db
from app.models.user import User
from app.services.comment_service import CommentService

router = APIRouter()

class CommentCreate(BaseModel):
    """创建评论请求"""
    content: str
    parent_id: Optional[str] = None
    rich_content: Optional[str] = None

class CommentUpdate(BaseModel):
    """更新评论请求"""
    content: str
    rich_content: Optional[str] = None

@router.post("/scenarios/{scenario_id}/comments")
async def create_comment(
    scenario_id: str,
    comment_data: CommentCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """创建评论"""
    comment_service = CommentService(db)

    try:
        comment = comment_service.create_comment(
            scenario_id=scenario_id,
            user_id=current_user.id,
            content=comment_data.content,
            parent_id=comment_data.parent_id,
            rich_content=comment_data.rich_content
        )

        return {
            "success": True,
            "comment": {
                "id": comment.id,
                "content": comment.content,
                "user_id": comment.user_id,
                "parent_id": comment.parent_id,
                "created_at": comment.created_at
            }
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/scenarios/{scenario_id}/comments")
async def get_scenario_comments(
    scenario_id: str,
    include_replies: bool = True,
    skip: int = Query(0, ge=0),
    limit: int = Query(20, ge=1, le=100),
    sort_by: str = Query("created_at", regex="^(created_at|like_count)$"),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """获取场景包评论列表"""
    comment_service = CommentService(db)
    comments = comment_service.get_scenario_comments(
        scenario_id=scenario_id,
        include_replies=include_replies,
        skip=skip,
        limit=limit,
        sort_by=sort_by
    )

    # 获取当前用户点赞的评论
    comment_ids = [c.id for c in comments]
    liked_ids = comment_service.get_user_liked_comments(current_user.id, comment_ids) if comment_ids else set()

    return {
        "comments": [
            {
                "id": c.id,
                "content": c.content,
                "user_id": c.user_id,
                "parent_id": c.parent_id,
                "reply_to_user_id": c.reply_to_user_id,
                "like_count": c.like_count,
                "is_edited": c.is_edited,
                "is_pinned": c.is_pinned,
                "is_liked": c.id in liked_ids,
                "created_at": c.created_at,
                "updated_at": c.updated_at,
                "replies": [
                    {
                        "id": r.id,
                        "content": r.content,
                        "user_id": r.user_id,
                        "reply_to_user_id": r.reply_to_user_id,
                        "like_count": r.like_count,
                        "created_at": r.created_at
                    }
                    for r in getattr(c, 'replies_list', [])
                ] if include_replies else []
            }
            for c in comments
        ]
    }

@router.get("/comments/{comment_id}/replies")
async def get_comment_replies(
    comment_id: str,
    skip: int = Query(0, ge=0),
    limit: int = Query(20, ge=1, le=100),
    db: Session = Depends(get_db)
):
    """获取评论回复列表"""
    comment_service = CommentService(db)
    replies = comment_service.get_comment_replies(
        comment_id=comment_id,
        skip=skip,
        limit=limit
    )

    return {
        "replies": [
            {
                "id": r.id,
                "content": r.content,
                "user_id": r.user_id,
                "reply_to_user_id": r.reply_to_user_id,
                "like_count": r.like_count,
                "created_at": r.created_at
            }
            for r in replies
        ]
    }

@router.put("/comments/{comment_id}")
async def update_comment(
    comment_id: str,
    comment_data: CommentUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """更新评论"""
    comment_service = CommentService(db)

    comment = comment_service.update_comment(
        comment_id=comment_id,
        user_id=current_user.id,
        content=comment_data.content,
        rich_content=comment_data.rich_content
    )

    if not comment:
        raise HTTPException(status_code=404, detail="评论不存在")

    return {
        "success": True,
        "comment": {
            "id": comment.id,
            "content": comment.content,
            "is_edited": comment.is_edited,
            "updated_at": comment.updated_at
        }
    }

@router.delete("/comments/{comment_id}")
async def delete_comment(
    comment_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """删除评论"""
    comment_service = CommentService(db)
    success = comment_service.delete_comment(
        comment_id=comment_id,
        user_id=current_user.id,
        is_moderator=current_user.is_moderator if hasattr(current_user, 'is_moderator') else False
    )

    if not success:
        raise HTTPException(status_code=404, detail="评论不存在")

    return {"success": True}

@router.post("/comments/{comment_id}/like")
async def toggle_like_comment(
    comment_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """点赞/取消点赞评论"""
    comment_service = CommentService(db)

    try:
        liked, count = comment_service.like_comment(
            comment_id=comment_id,
            user_id=current_user.id
        )

        return {
            "success": True,
            "liked": liked,
            "like_count": count
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/scenarios/{scenario_id}/comments/statistics")
async def get_comment_statistics(
    scenario_id: str,
    db: Session = Depends(get_db)
):
    """获取评论统计"""
    comment_service = CommentService(db)
    stats = comment_service.get_comment_statistics(scenario_id)

    return stats
```

---

## TypeScript/React 前端实现

### 1. 评论服务

```typescript
// frontend/src/services/api/comment.ts
import api from './client';

export interface CommentCreate {
  content: string;
  parent_id?: string;
  rich_content?: string;
}

export interface Comment {
  id: string;
  content: string;
  user_id: number;
  parent_id: string | null;
  reply_to_user_id: number | null;
  like_count: number;
  is_edited: boolean;
  is_pinned: boolean;
  is_liked: boolean;
  created_at: string;
  updated_at: string;
  replies?: Comment[];
}

class CommentService {
  /**
   * 创建评论
   */
  async createComment(
    scenarioId: string,
    data: CommentCreate
  ): Promise<{ success: boolean; comment: Comment }> {
    try {
      const response = await api.post(
        `/api/v1/comment/scenarios/${scenarioId}/comments`,
        data
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '发表评论失败');
    }
  }

  /**
   * 获取场景包评论列表
   */
  async getScenarioComments(
    scenarioId: string,
    params?: {
      include_replies?: boolean;
      skip?: number;
      limit?: number;
      sort_by?: string;
    }
  ): Promise<{ comments: Comment[] }> {
    try {
      const response = await api.get(
        `/api/v1/comment/scenarios/${scenarioId}/comments`,
        { params }
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取评论失败');
    }
  }

  /**
   * 获取评论回复
   */
  async getCommentReplies(
    commentId: string,
    params?: { skip?: number; limit?: number }
  ): Promise<{ replies: Comment[] }> {
    try {
      const response = await api.get(
        `/api/v1/comment/comments/${commentId}/replies`,
        { params }
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取回复失败');
    }
  }

  /**
   * 更新评论
   */
  async updateComment(
    commentId: string,
    data: CommentCreate
  ): Promise<{ success: boolean; comment: Comment }> {
    try {
      const response = await api.put(
        `/api/v1/comment/comments/${commentId}`,
        data
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '更新评论失败');
    }
  }

  /**
   * 删除评论
   */
  async deleteComment(commentId: string): Promise<{ success: boolean }> {
    try {
      const response = await api.delete(
        `/api/v1/comment/comments/${commentId}`
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '删除评论失败');
    }
  }

  /**
   * 点赞/取消点赞评论
   */
  async toggleLikeComment(
    commentId: string
  ): Promise<{ success: boolean; liked: boolean; like_count: number }> {
    try {
      const response = await api.post(
        `/api/v1/comment/comments/${commentId}/like`
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '操作失败');
    }
  }

  /**
   * 获取评论统计
   */
  async getCommentStatistics(
    scenarioId: string
  ): Promise<{ total_comments: number; total_replies: number }> {
    try {
      const response = await api.get(
        `/api/v1/comment/scenarios/${scenarioId}/comments/statistics`
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取统计失败');
    }
  }
}

export default new CommentService();
```

### 2. 评论组件

```typescript
// frontend/src/components/scenario/CommentList.tsx
import React, { useState, useEffect } from 'react';
import {
  List,
  Comment,
  Avatar,
  Button,
  Input,
  message,
  Pagination,
  Tooltip,
  Popconfirm,
} from 'antd';
import {
  LikeOutlined,
  LikeFilled,
  MessageOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import commentService, { Comment as CommentType } from '@/services/api/comment';

const { TextArea } = Input;

interface CommentListProps {
  scenarioId: string;
}

const CommentList: React.FC<CommentListProps> = ({ scenarioId }) => {
  const [comments, setComments] = useState<CommentType[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [replyContent, setReplyContent] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editContent, setEditContent] = useState('');

  useEffect(() => {
    loadComments();
  }, [scenarioId, page]);

  const loadComments = async () => {
    setLoading(true);
    try {
      const result = await commentService.getScenarioComments(scenarioId, {
        skip: (page - 1) * 20,
        limit: 20,
      });
      setComments(result.comments);
      setTotal(result.comments.length);
    } catch (error: any) {
      message.error(error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleLike = async (commentId: string) => {
    try {
      const result = await commentService.toggleLikeComment(commentId);

      // 更新评论列表
      setComments(comments.map(c => {
        if (c.id === commentId) {
          return {
            ...c,
            is_liked: result.liked,
            like_count: result.like_count
          };
        }
        return c;
      }));
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const handleReply = async (commentId: string) => {
    if (!replyContent.trim()) {
      message.warning('请输入回复内容');
      return;
    }

    try {
      await commentService.createComment(scenarioId, {
        content: replyContent,
        parent_id: commentId,
      });

      message.success('回复成功');
      setReplyContent('');
      setReplyingTo(null);
      loadComments();
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const handleEdit = async (commentId: string) => {
    if (!editContent.trim()) {
      message.warning('请输入评论内容');
      return;
    }

    try {
      await commentService.updateComment(commentId, {
        content: editContent,
      });

      message.success('修改成功');
      setEditingId(null);
      setEditContent('');
      loadComments();
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const handleDelete = async (commentId: string) => {
    try {
      await commentService.deleteComment(commentId);
      message.success('删除成功');
      loadComments();
    } catch (error: any) {
      message.error(error.message);
    }
  };

  const renderComment = (item: CommentType) => {
    const actions = [
      <Tooltip key="like" title="点赞">
        <span
          onClick={() => handleLike(item.id)}
          style={{ cursor: 'pointer' }}
        >
          {item.is_liked ? <LikeFilled /> : <LikeOutlined />}
          <span style={{ paddingLeft: 8 }}>{item.like_count}</span>
        </span>
      </Tooltip>,
      <Tooltip key="reply" title="回复">
        <span
          onClick={() => setReplyingTo(item.id)}
          style={{ cursor: 'pointer' }}
        >
          <MessageOutlined />
          回复
        </span>
      </Tooltip>,
      <Tooltip key="edit" title="编辑">
        <span
          onClick={() => {
            setEditingId(item.id);
            setEditContent(item.content);
          }}
          style={{ cursor: 'pointer' }}
        >
          <EditOutlined />
        </span>
      </Tooltip>,
      <Popconfirm
        key="delete"
        title="确定删除这条评论吗？"
        onConfirm={() => handleDelete(item.id)}
      >
        <DeleteOutlined style={{ cursor: 'pointer' }} />
      </Popconfirm>,
    ];

    return (
      <Comment
        key={item.id}
        actions={actions}
        author={`用户 ${item.user_id}`}
        avatar={<Avatar>{item.user_id.toString().slice(-2)}</Avatar>}
        content={
          editingId === item.id ? (
            <div>
              <TextArea
                rows={4}
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
              />
              <div style={{ marginTop: 8 }}>
                <Button
                  type="primary"
                  size="small"
                  onClick={() => handleEdit(item.id)}
                >
                  保存
                </Button>
                <Button
                  size="small"
                  style={{ marginLeft: 8 }}
                  onClick={() => {
                    setEditingId(null);
                    setEditContent('');
                  }}
                >
                  取消
                </Button>
              </div>
            </div>
          ) : (
            <p>{item.content}</p>
          )
        }
        datetime={item.created_at}
      >
        {replyingTo === item.id && (
          <div style={{ marginTop: 16 }}>
            <TextArea
              rows={3}
              placeholder="写下你的回复..."
              value={replyContent}
              onChange={(e) => setReplyContent(e.target.value)}
            />
            <div style={{ marginTop: 8 }}>
              <Button
                type="primary"
                size="small"
                onClick={() => handleReply(item.id)}
              >
                发表回复
              </Button>
              <Button
                size="small"
                style={{ marginLeft: 8 }}
                onClick={() => {
                  setReplyingTo(null);
                  setReplyContent('');
                }}
              >
                取消
              </Button>
            </div>
          </div>
        )}

        {item.replies && item.replies.length > 0 && (
          <List
            dataSource={item.replies}
            renderItem={renderComment}
            style={{ marginTop: 16 }}
          />
        )}
      </Comment>
    );
  };

  return (
    <div>
      <List
        loading={loading}
        dataSource={comments}
        renderItem={renderComment}
        pagination={{
          current: page,
          pageSize: 20,
          total,
          onChange: setPage,
        }}
      />
    </div>
  );
};

export default CommentList;
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/models/comment.py` | 评论数据模型 |
| `/backend/app/services/comment_service.py` | 评论服务 |
| `/backend/app/api/v1/endpoints/comment.py` | 评论API路由 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/comment.ts` | 评论服务API |
| `/frontend/src/components/scenario/CommentList.tsx` | 评论列表组件 |

---

## 验收标准

### 功能验收

- [ ] 用户可以发表评论
- [ ] 用户可以回复评论
- [ ] 用户可以编辑自己的评论
- [ ] 用户可以删除自己的评论
- [ ] 实现评论点赞功能
- [ ] 支持评论嵌套显示
- [ ] 支持按时间/热度排序
- [ ] 敏感词自动过滤

### 安全性验收

- [ ] 只能编辑/删除自己的评论
- [ ] 防止XSS攻击
- [ ] 敏感词正确过滤
- [ ] 评论长度限制生效

### 性能验收

- [ ] 评论列表加载时间 < 1s
- [ ] 发表评论响应时间 < 500ms
- [ ] 支持分页加载

---

## 参考文档

### 内部文档

- [M4-015: 场景包评分](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-015-package-rating.md)

### 技术文档

- [Ant Design Comment Component](https://ant.design/components/comment-cn/)
- [XSS Prevention](https://owasp.org/www-community/attacks/xss/)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
