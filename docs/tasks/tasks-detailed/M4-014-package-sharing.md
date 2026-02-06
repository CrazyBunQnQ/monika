# M4-014: 实现场景包分享

**任务ID**: M4-014
**任务名称**: 实现场景包分享
**预估时间**: 4 小时
**优先级**: P1
**依赖**: M4-013 (CDN 上传)
**状态**: 待开始

---

## 任务概述

实现场景包的分享功能，允许用户生成分享链接、设置访问权限、管理分享记录。支持公开分享、密码保护、限时分享等多种分享模式，并提供分享统计分析功能。

---

## 子任务拆解

| ID | 子任务描述 | 预估时间 | 依赖 | 状态 |
|----|-----------|----------|------|------|
| M4-014-01 | 设计分享数据模型和数据库表结构 | 1h | M4-013 | 待开始 |
| M4-014-02 | 实现分享链接生成和验证服务 | 1h | M4-014-01 | 待开始 |
| M4-014-03 | 实现访问权限控制（公开/密码/限时） | 1h | M4-014-02 | 待开始 |
| M4-014-04 | 实现分享记录管理和统计 | 0.5h | M4-014-03 | 待开始 |
| M4-014-05 | 实现分享API和前端界面 | 0.5h | M4-014-04 | 待开始 |

**总预估时间**: 4 小时

---

## Python 后端实现

### 1. 数据库模型

```python
# backend/app/models/share.py
from sqlalchemy import Column, Integer, String, DateTime, Boolean, ForeignKey, Text, Enum as SQLEnum
from sqlalchemy.orm import relationship
from datetime import datetime
import enum
import uuid

from app.db.base_class import Base

class SharePermission(str, enum.Enum):
    """分享权限类型"""
    PUBLIC = "public"         # 公开访问
    PASSWORD = "password"     # 密码保护
    PRIVATE = "private"       # 私有（仅指定用户）
    TOKEN = "token"           # Token 访问

class ScenarioShare(Base):
    """场景包分享记录"""
    __tablename__ = "scenario_shares"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    scenario_id = Column(String(36), ForeignKey("scenarios.id"), nullable=False, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)

    # 分享信息
    share_code = Column(String(32), unique=True, nullable=False, index=True)  # 分享码
    title = Column(String(200))  # 分享标题
    description = Column(Text)   # 分享描述

    # 权限设置
    permission = Column(SQLEnum(SharePermission), default=SharePermission.PUBLIC, nullable=False)
    password = Column(String(128))  # 访问密码（bcrypt hash）

    # 访问限制
    max_access_count = Column(Integer, default=None)  # 最大访问次数（None=无限制）
    access_count = Column(Integer, default=0)          # 已访问次数
    max_download_count = Column(Integer, default=None)  # 最大下载次数
    download_count = Column(Integer, default=0)         # 已下载次数

    # 时间限制
    expires_at = Column(DateTime, default=None)  # 过期时间（None=永不过期）

    # 状态
    is_active = Column(Boolean, default=True, nullable=False, index=True)

    # 统计
    view_count = Column(Integer, default=0)  # 浏览次数
    share_count = Column(Integer, default=0)  # 被分享次数

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    last_accessed_at = Column(DateTime, default=None)

    # 关系
    scenario = relationship("Scenario", back_populates="shares")
    user = relationship("User", back_populates="scenario_shares")

    def __repr__(self):
        return f"<ScenarioShare(code={self.share_code}, permission={self.permission})>"

    def is_valid(self) -> bool:
        """检查分享是否有效"""
        if not self.is_active:
            return False
        if self.expires_at and self.expires_at < datetime.utcnow():
            return False
        if self.max_access_count and self.access_count >= self.max_access_count:
            return False
        return True

    def can_download(self) -> bool:
        """检查是否可以下载"""
        if not self.is_valid():
            return False
        if self.max_download_count and self.download_count >= self.max_download_count:
            return False
        return True

class ShareAccessLog(Base):
    """分享访问日志"""
    __tablename__ = "share_access_logs"

    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    share_id = Column(String(36), ForeignKey("scenario_shares.id"), nullable=False, index=True)

    # 访问信息
    ip_address = Column(String(45))  # IPv4 or IPv6
    user_agent = Column(String(500))
    referrer = Column(String(500))

    # 访问类型
    action = Column(String(50), nullable=False)  # view, download, etc

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False, index=True)

    # 关系
    share = relationship("ScenarioShare", back_populates="access_logs")
```

### 2. 分享服务

```python
# backend/app/services/share_service.py
from typing import Optional, List, Dict, Any
from datetime import datetime, timedelta
import secrets
import bcrypt
from sqlalchemy.orm import Session

from app.models.share import ScenarioShare, SharePermission, ShareAccessLog
from app.core.exceptions import ParseError

class ShareService:
    """分享服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_share(
        self,
        scenario_id: str,
        user_id: int,
        title: Optional[str] = None,
        description: Optional[str] = None,
        permission: SharePermission = SharePermission.PUBLIC,
        password: Optional[str] = None,
        max_access_count: Optional[int] = None,
        max_download_count: Optional[int] = None,
        expires_days: Optional[int] = None
    ) -> ScenarioShare:
        """
        创建分享链接

        Args:
            scenario_id: 场景包ID
            user_id: 用户ID
            title: 分享标题
            description: 分享描述
            permission: 权限类型
            password: 访问密码（password 模式）
            max_access_count: 最大访问次数
            max_download_count: 最大下载次数
            expires_days: 有效期（天）

        Returns:
            ScenarioShare: 分享记录
        """
        # 生成唯一分享码
        share_code = self._generate_share_code()

        # 计算过期时间
        expires_at = None
        if expires_days:
            expires_at = datetime.utcnow() + timedelta(days=expires_days)

        # 哈希密码
        password_hash = None
        if password and permission == SharePermission.PASSWORD:
            password_hash = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt()).decode('utf-8')

        # 创建分享记录
        share = ScenarioShare(
            scenario_id=scenario_id,
            user_id=user_id,
            share_code=share_code,
            title=title,
            description=description,
            permission=permission,
            password=password_hash,
            max_access_count=max_access_count,
            max_download_count=max_download_count,
            expires_at=expires_at
        )

        self.db.add(share)
        self.db.commit()
        self.db.refresh(share)

        return share

    def get_share_by_code(self, share_code: str) -> Optional[ScenarioShare]:
        """根据分享码获取分享记录"""
        return self.db.query(ScenarioShare).filter(
            ScenarioShare.share_code == share_code
        ).first()

    def verify_share_access(
        self,
        share_code: str,
        password: Optional[str] = None,
        ip_address: Optional[str] = None
    ) -> tuple[bool, Optional[ScenarioShare], Optional[str]]:
        """
        验证分享访问权限

        Args:
            share_code: 分享码
            password: 访问密码（如果需要）
            ip_address: 访问者IP

        Returns:
            tuple: (是否允许访问, 分享记录, 错误信息)
        """
        share = self.get_share_by_code(share_code)

        if not share:
            return False, None, "分享链接不存在"

        if not share.is_valid():
            if not share.is_active:
                return False, share, "分享链接已被禁用"
            if share.expires_at and share.expires_at < datetime.utcnow():
                return False, share, "分享链接已过期"
            if share.max_access_count and share.access_count >= share.max_access_count:
                return False, share, "访问次数已达上限"

        # 密码验证
        if share.permission == SharePermission.PASSWORD:
            if not password:
                return False, share, "请输入访问密码"
            if not self._verify_password(password, share.password):
                return False, share, "密码错误"

        return True, share, None

    def record_access(
        self,
        share_id: str,
        action: str,
        ip_address: Optional[str] = None,
        user_agent: Optional[str] = None,
        referrer: Optional[str] = None
    ) -> None:
        """记录访问日志"""
        log = ShareAccessLog(
            share_id=share_id,
            ip_address=ip_address,
            user_agent=user_agent,
            referrer=referrer,
            action=action
        )

        self.db.add(log)

        # 更新分享统计
        share = self.db.query(ScenarioShare).filter(
            ScenarioShare.id == share_id
        ).first()

        if share:
            if action == "view":
                share.view_count += 1
                share.access_count += 1
                share.last_accessed_at = datetime.utcnow()
            elif action == "download":
                share.download_count += 1

        self.db.commit()

    def get_user_shares(
        self,
        user_id: int,
        skip: int = 0,
        limit: int = 20
    ) -> List[ScenarioShare]:
        """获取用户的分享列表"""
        return self.db.query(ScenarioShare).filter(
            ScenarioShare.user_id == user_id
        ).order_by(
            ScenarioShare.created_at.desc()
        ).offset(skip).limit(limit).all()

    def update_share(
        self,
        share_id: str,
        user_id: int,
        **kwargs
    ) -> Optional[ScenarioShare]:
        """更新分享设置"""
        share = self.db.query(ScenarioShare).filter(
            ScenarioShare.id == share_id,
            ScenarioShare.user_id == user_id
        ).first()

        if not share:
            return None

        # 更新允许的字段
        updatable_fields = [
            'title', 'description', 'permission', 'max_access_count',
            'max_download_count', 'expires_at', 'is_active'
        ]

        for field in updatable_fields:
            if field in kwargs:
                setattr(share, field, kwargs[field])

        # 如果更新密码
        if 'password' in kwargs and kwargs['password']:
            share.password = bcrypt.hashpw(
                kwargs['password'].encode('utf-8'),
                bcrypt.gensalt()
            ).decode('utf-8')

        self.db.commit()
        self.db.refresh(share)

        return share

    def delete_share(self, share_id: str, user_id: int) -> bool:
        """删除分享"""
        share = self.db.query(ScenarioShare).filter(
            ScenarioShare.id == share_id,
            ScenarioShare.user_id == user_id
        ).first()

        if not share:
            return False

        self.db.delete(share)
        self.db.commit()

        return True

    def get_share_statistics(
        self,
        share_id: str,
        user_id: int
    ) -> Optional[Dict[str, Any]]:
        """获取分享统计"""
        share = self.db.query(ScenarioShare).filter(
            ScenarioShare.id == share_id,
            ScenarioShare.user_id == user_id
        ).first()

        if not share:
            return None

        # 获取访问日志
        logs = self.db.query(ShareAccessLog).filter(
            ShareAccessLog.share_id == share_id
        ).all()

        # 统计最近7天的访问
        now = datetime.utcnow()
        week_ago = now - timedelta(days=7)

        recent_logs = [log for log in logs if log.created_at >= week_ago]

        return {
            "total_views": share.view_count,
            "total_downloads": share.download_count,
            "access_count": share.access_count,
            "recent_views": len([log for log in recent_logs if log.action == "view"]),
            "recent_downloads": len([log for log in recent_logs if log.action == "download"]),
            "created_at": share.created_at,
            "last_accessed_at": share.last_accessed_at,
        }

    def _generate_share_code(self) -> str:
        """生成唯一分享码"""
        while True:
            code = secrets.token_urlsafe(16)
            existing = self.db.query(ScenarioShare).filter(
                ScenarioShare.share_code == code
            ).first()
            if not existing:
                return code

    def _verify_password(self, password: str, password_hash: str) -> bool:
        """验证密码"""
        return bcrypt.checkpw(
            password.encode('utf-8'),
            password_hash.encode('utf-8')
        )
```

### 3. API 路由

```python
# backend/app/api/v1/endpoints/share.py
from fastapi import APIRouter, Depends, HTTPException, Query
from fastapi.security import HTTPBearer
from typing import Optional, List
from sqlalchemy.orm import Session

from app.api.deps import get_current_user, get_db
from app.models.share import SharePermission
from app.services.share_service import ShareService
from app.models.user import User

router = APIRouter()
security = HTTPBearer()

@router.post("/scenarios/{scenario_id}/share")
async def create_scenario_share(
    scenario_id: str,
    title: Optional[str] = None,
    description: Optional[str] = None,
    permission: SharePermission = SharePermission.PUBLIC,
    password: Optional[str] = None,
    max_access_count: Optional[int] = None,
    max_download_count: Optional[int] = None,
    expires_days: Optional[int] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    创建场景包分享链接

    - **scenario_id**: 场景包ID
    - **title**: 分享标题
    - **description**: 分享描述
    - **permission**: 权限类型 (public/password/private/token)
    - **password**: 访问密码（password 模式需要）
    - **max_access_count**: 最大访问次数
    - **max_download_count**: 最大下载次数
    - **expires_days**: 有效期（天）
    """
    share_service = ShareService(db)

    try:
        share = share_service.create_share(
            scenario_id=scenario_id,
            user_id=current_user.id,
            title=title,
            description=description,
            permission=permission,
            password=password,
            max_access_count=max_access_count,
            max_download_count=max_download_count,
            expires_days=expires_days
        )

        return {
            "success": True,
            "share_code": share.share_code,
            "share_url": f"/share/{share.share_code}",
            "permission": share.permission.value,
            "expires_at": share.expires_at
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/share/{share_code}")
async def get_share_info(
    share_code: str,
    db: Session = Depends(get_db)
):
    """
    获取分享信息（访问前调用）

    - **share_code**: 分享码
    """
    share_service = ShareService(db)
    share = share_service.get_share_by_code(share_code)

    if not share:
        raise HTTPException(status_code=404, detail="分享链接不存在")

    # 返回基本信息（不包含敏感数据）
    return {
        "share_code": share.share_code,
        "title": share.title,
        "description": share.description,
        "permission": share.permission.value,
        "requires_password": share.permission == SharePermission.PASSWORD,
        "is_valid": share.is_valid()
    }

@router.post("/share/{share_code}/access")
async def access_share(
    share_code: str,
    password: Optional[str] = None,
    db: Session = Depends(get_db),
    request: Request = None
):
    """
    访问分享的场景包

    - **share_code**: 分享码
    - **password**: 访问密码（如果需要）
    """
    share_service = ShareService(db)

    # 验证访问权限
    allowed, share, error = share_service.verify_share_access(
        share_code=share_code,
        password=password,
        ip_address=request.client.host if request else None
    )

    if not allowed:
        raise HTTPException(status_code=403, detail=error)

    # 记录访问
    share_service.record_access(
        share_id=share.id,
        action="view",
        ip_address=request.client.host if request else None,
        user_agent=request.headers.get("user-agent")
    )

    # 返回场景包信息
    return {
        "success": True,
        "scenario_id": share.scenario_id,
        "can_download": share.can_download(),
        "download_count": share.download_count,
        "max_download_count": share.max_download_count
    }

@router.get("/users/me/shares")
async def get_my_shares(
    skip: int = Query(0, ge=0),
    limit: int = Query(20, ge=1, le=100),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    获取我的分享列表

    - **skip**: 跳过记录数
    - **limit**: 返回记录数
    """
    share_service = ShareService(db)
    shares = share_service.get_user_shares(
        user_id=current_user.id,
        skip=skip,
        limit=limit
    )

    return {
        "shares": [
            {
                "id": share.id,
                "share_code": share.share_code,
                "title": share.title,
                "permission": share.permission.value,
                "view_count": share.view_count,
                "download_count": share.download_count,
                "is_valid": share.is_valid(),
                "created_at": share.created_at
            }
            for share in shares
        ]
    }

@router.get("/share/{share_id}/statistics")
async def get_share_statistics(
    share_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    获取分享统计信息

    - **share_id**: 分享ID
    """
    share_service = ShareService(db)
    stats = share_service.get_share_statistics(
        share_id=share_id,
        user_id=current_user.id
    )

    if not stats:
        raise HTTPException(status_code=404, detail="分享不存在")

    return stats

@router.put("/share/{share_id}")
async def update_share(
    share_id: str,
    title: Optional[str] = None,
    description: Optional[str] = None,
    permission: Optional[SharePermission] = None,
    password: Optional[str] = None,
    max_access_count: Optional[int] = None,
    max_download_count: Optional[int] = None,
    expires_days: Optional[int] = None,
    is_active: Optional[bool] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    更新分享设置

    - **share_id**: 分享ID
    - 其他参数同创建分享
    """
    share_service = ShareService(db)

    # 构建更新参数
    update_data = {}
    if title is not None:
        update_data['title'] = title
    if description is not None:
        update_data['description'] = description
    if permission is not None:
        update_data['permission'] = permission
    if password is not None:
        update_data['password'] = password
    if max_access_count is not None:
        update_data['max_access_count'] = max_access_count
    if max_download_count is not None:
        update_data['max_download_count'] = max_download_count
    if is_active is not None:
        update_data['is_active'] = is_active

    if expires_days is not None:
        from datetime import timedelta
        update_data['expires_at'] = datetime.utcnow() + timedelta(days=expires_days)

    share = share_service.update_share(
        share_id=share_id,
        user_id=current_user.id,
        **update_data
    )

    if not share:
        raise HTTPException(status_code=404, detail="分享不存在")

    return {
        "success": True,
        "share": {
            "id": share.id,
            "share_code": share.share_code,
            "title": share.title,
            "permission": share.permission.value,
            "is_active": share.is_active
        }
    }

@router.delete("/share/{share_id}")
async def delete_share(
    share_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    删除分享

    - **share_id**: 分享ID
    """
    share_service = ShareService(db)
    success = share_service.delete_share(
        share_id=share_id,
        user_id=current_user.id
    )

    if not success:
        raise HTTPException(status_code=404, detail="分享不存在")

    return {"success": True}
```

---

## TypeScript/React 前端实现

### 1. 分享服务

```typescript
// frontend/src/services/api/share.ts
import api from './client';

export enum SharePermission {
  PUBLIC = 'public',
  PASSWORD = 'password',
  PRIVATE = 'private',
  TOKEN = 'token',
}

export interface ShareInfo {
  id: string;
  share_code: string;
  title: string;
  description: string;
  permission: SharePermission;
  view_count: number;
  download_count: number;
  is_valid: boolean;
  created_at: string;
}

export interface CreateShareRequest {
  scenario_id: string;
  title?: string;
  description?: string;
  permission?: SharePermission;
  password?: string;
  max_access_count?: number;
  max_download_count?: number;
  expires_days?: number;
}

class ShareService {
  /**
   * 创建分享链接
   */
  async createShare(data: CreateShareRequest): Promise<{
    success: boolean;
    share_code: string;
    share_url: string;
    permission: string;
    expires_at: string | null;
  }> {
    try {
      const response = await api.post(
        `/api/v1/share/scenarios/${data.scenario_id}/share`,
        data
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '创建分享失败');
    }
  }

  /**
   * 获取分享信息
   */
  async getShareInfo(shareCode: string): Promise<{
    share_code: string;
    title: string;
    description: string;
    permission: string;
    requires_password: boolean;
    is_valid: boolean;
  }> {
    try {
      const response = await api.get(`/api/v1/share/share/${shareCode}`);
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取分享信息失败');
    }
  }

  /**
   * 访问分享
   */
  async accessShare(
    shareCode: string,
    password?: string
  ): Promise<{
    success: boolean;
    scenario_id: string;
    can_download: boolean;
    download_count: number;
    max_download_count: number | null;
  }> {
    try {
      const response = await api.post(`/api/v1/share/share/${shareCode}/access`, {
        password,
      });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '访问分享失败');
    }
  }

  /**
   * 获取我的分享列表
   */
  async getMyShares(params?: {
    skip?: number;
    limit?: number;
  }): Promise<{ shares: ShareInfo[] }> {
    try {
      const response = await api.get('/api/v1/share/users/me/shares', {
        params,
      });
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取分享列表失败');
    }
  }

  /**
   * 获取分享统计
   */
  async getShareStatistics(shareId: string): Promise<{
    total_views: number;
    total_downloads: number;
    access_count: number;
    recent_views: number;
    recent_downloads: number;
    created_at: string;
    last_accessed_at: string | null;
  }> {
    try {
      const response = await api.get(`/api/v1/share/share/${shareId}/statistics`);
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '获取统计信息失败');
    }
  }

  /**
   * 更新分享
   */
  async updateShare(
    shareId: string,
    data: Partial<CreateShareRequest> & { is_active?: boolean }
  ): Promise<{ success: boolean; share: Partial<ShareInfo> }> {
    try {
      const response = await api.put(`/api/v1/share/share/${shareId}`, data);
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '更新分享失败');
    }
  }

  /**
   * 删除分享
   */
  async deleteShare(shareId: string): Promise<{ success: boolean }> {
    try {
      const response = await api.delete(`/api/v1/share/share/${shareId}`);
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.detail || '删除分享失败');
    }
  }
}

export default new ShareService();
```

### 2. 分享创建组件

```typescript
// frontend/src/components/scenario/ShareCreator.tsx
import React, { useState } from 'react';
import {
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  Switch,
  Button,
  Space,
  message,
  Tabs,
  Card,
  Statistic,
  Row,
  Col,
  Tag,
} from 'antd';
import {
  ShareAltOutlined,
  LinkOutlined,
  LockOutlined,
  CopyOutlined,
} from '@ant-design/icons';
import shareService, { SharePermission } from '@/services/api/share';

interface ShareCreatorProps {
  scenarioId: string;
  scenarioName: string;
  visible: boolean;
  onClose: () => void;
}

const ShareCreator: React.FC<ShareCreatorProps> = ({
  scenarioId,
  scenarioName,
  visible,
  onClose,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [shareResult, setShareResult] = useState<{
    share_code: string;
    share_url: string;
  } | null>(null);
  const [permission, setPermission] = useState(SharePermission.PUBLIC);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const result = await shareService.createShare({
        scenario_id: scenarioId,
        ...values,
      });

      setShareResult({
        share_code: result.share_code,
        share_url: `${window.location.origin}${result.share_url}`,
      });

      message.success('分享链接创建成功');
    } catch (error: any) {
      message.error(error.message || '创建失败');
    } finally {
      setLoading(false);
    }
  };

  const copyLink = () => {
    if (shareResult?.share_url) {
      navigator.clipboard.writeText(shareResult.share_url);
      message.success('链接已复制');
    }
  };

  const copyCode = () => {
    if (shareResult?.share_code) {
      navigator.clipboard.writeText(shareResult.share_code);
      message.success('分享码已复制');
    }
  };

  return (
    <Modal
      title={<><ShareAltOutlined /> 创建分享</>}
      open={visible}
      onCancel={onClose}
      width={600}
      footer={null}
    >
      <Tabs
        items={[
          {
            key: 'create',
            label: '创建分享',
            children: (
              <Form
                form={form}
                layout="vertical"
                initialValues={{
                  permission: SharePermission.PUBLIC,
                  expires_days: null,
                }}
              >
                <Form.Item label="场景包">
                  <Input value={scenarioName} disabled />
                </Form.Item>

                <Form.Item
                  label="分享标题"
                  name="title"
                  rules={[{ required: true, message: '请输入分享标题' }]}
                >
                  <Input placeholder="为分享添加一个标题" />
                </Form.Item>

                <Form.Item label="分享描述" name="description">
                  <Input.TextArea
                    rows={3}
                    placeholder="描述这个场景包（可选）"
                  />
                </Form.Item>

                <Form.Item
                  label="权限设置"
                  name="permission"
                >
                  <Select
                    onChange={setPermission}
                    options={[
                      { label: '公开访问', value: SharePermission.PUBLIC },
                      { label: '密码保护', value: SharePermission.PASSWORD },
                      { label: '私有访问', value: SharePermission.PRIVATE },
                    ]}
                  />
                </Form.Item>

                {permission === SharePermission.PASSWORD && (
                  <Form.Item
                    label="访问密码"
                    name="password"
                    rules={[{ required: true, message: '请设置访问密码' }]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="设置访问密码"
                    />
                  </Form.Item>
                )}

                <Form.Item label="访问限制">
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Form.Item name="max_access_count" noStyle>
                      <InputNumber
                        placeholder="最大访问次数"
                        min={1}
                        style={{ width: '100%' }}
                        addonBefore="访问次数"
                      />
                    </Form.Item>
                    <Form.Item name="max_download_count" noStyle>
                      <InputNumber
                        placeholder="最大下载次数"
                        min={1}
                        style={{ width: '100%' }}
                        addonBefore="下载次数"
                      />
                    </Form.Item>
                  </Space>
                </Form.Item>

                <Form.Item
                  label="有效期"
                  name="expires_days"
                >
                  <InputNumber
                    placeholder="永久有效"
                    min={1}
                    max={365}
                    style={{ width: '100%' }}
                    addonAfter="天"
                  />
                </Form.Item>

                <Form.Item>
                  <Button
                    type="primary"
                    onClick={handleCreate}
                    loading={loading}
                    block
                    icon={<ShareAltOutlined />}
                  >
                    创建分享链接
                  </Button>
                </Form.Item>
              </Form>
            ),
          },
          {
            key: 'result',
            label: '分享结果',
            disabled: !shareResult,
            children: shareResult && (
              <Space direction="vertical" style={{ width: '100%' }} size="large">
                <Card>
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <div>
                      <div style={{ marginBottom: 8 }}>分享链接</div>
                      <Input
                        value={shareResult.share_url}
                        addonAfter={
                          <Button
                            type="link"
                            icon={<CopyOutlined />}
                            onClick={copyLink}
                          >
                            复制
                          </Button>
                        }
                      />
                    </div>
                    <div>
                      <div style={{ marginBottom: 8 }}>分享码</div>
                      <Input
                        value={shareResult.share_code}
                        addonAfter={
                          <Button
                            type="link"
                            icon={<CopyOutlined />}
                            onClick={copyCode}
                          >
                            复制
                          </Button>
                        }
                      />
                    </div>
                  </Space>
                </Card>

                <Card title="分享提示">
                  <ul style={{ margin: 0, paddingLeft: 20 }}>
                    <li>将链接发送给好友即可分享场景包</li>
                    <li>分享码可用于快速搜索场景包</li>
                    <li>您可以在"我的分享"中管理此分享</li>
                  </ul>
                </Card>
              </Space>
            ),
          },
        ]}
      />
    </Modal>
  );
};

export default ShareCreator;
```

---

## 涉及文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `/backend/app/models/share.py` | 分享数据模型 |
| `/backend/app/services/share_service.py` | 分享服务 |
| `/backend/app/api/v1/endpoints/share.py` | 分享API路由 |

| 文件路径 | 说明 |
|---------|------|
| `/frontend/src/services/api/share.ts` | 分享服务API |
| `/frontend/src/components/scenario/ShareCreator.tsx` | 分享创建组件 |

---

## 验收标准

### 功能验收

- [ ] 成功创建分享链接
- [ ] 支持公开/密码保护/私有三种模式
- [ ] 密码保护模式正确验证密码
- [ ] 支持访问次数和下载次数限制
- [ ] 支持有效期设置
- [ ] 正确记录访问日志
- [ ] 统计数据准确

### 安全性验收

- [ ] 密码正确哈希存储
- [ ] 分享码足够随机且唯一
- [ ] 过期/达到限制的分享无法访问
- [ ] 仅创建者可以修改/删除分享

### 性能验收

- [ ] 分享链接访问响应时间 < 500ms
- [ ] 统计查询响应时间 < 1s

---

## 参考文档

### 内部文档

- [M4-013: CDN 上传](/Users/guochangxi/git/monika/docs/tasks/tasks-detailed/M4-013-cdn-upload.md)

### 技术文档

- [bcrypt Documentation](https://bcrypt.github.io/)
- [secrets - Secure random module](https://docs.python.org/3/library/secrets.html)

---

**创建日期**: 2026-02-06
**最后更新**: 2026-02-06
