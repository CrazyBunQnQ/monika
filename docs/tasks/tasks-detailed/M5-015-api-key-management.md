# M5-015: 实现 API 密钥管理

**任务ID**: M5-015
**标题**: 实现 API 密钥管理系统
**类型**: fullstack (全栈开发)
**预估工时**: 8h
**依赖**: M0 完成

---

## 任务描述

实现一个完整的 API 密钥管理系统，允许用户生成和管理 API 密钥用于第三方集成。密钥需要安全存储、支持权限控制、使用限制、过期管理等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-015-01 | 设计 API 密钥数据模型 | 密钥、权限、使用限制 | 1h |
| M5-015-02 | 实现密钥生成与存储 | 安全生成、加密存储 | 2h |
| M5-015-03 | 实现密钥认证中间件 | JWT + API Key 双重认证 | 1.5h |
| M5-015-04 | 实现密钥管理 API | 创建、撤销、轮换 | 1.5h |
| M5-015-05 | 实现前端密钥管理界面 | 密钥列表、创建、查看 | 2h |

---

## 完整后端代码示例 (Python + Agno)

### 数据模型

```python
# backend/app/models/api_keys.py
from datetime import datetime, timedelta
from typing import List, Optional
from enum import Enum
from sqlalchemy import Column, String, JSON, DateTime, Boolean, Integer, ForeignKey, Text
from sqlalchemy.dialects.postgresql import UUID
import uuid
import secrets

from app.db.base_class import Base


class APIKeyScope(str, Enum):
    """API 密钥权限范围"""
    READ = "read"  # 只读权限
    WRITE = "write"  # 读写权限
    ADMIN = "admin"  # 管理员权限
    GAME = "game"  # 游戏操作权限
    WEBHOOK = "webhook"  # Webhook 权限


class APIKey(Base):
    """API 密钥表"""
    __tablename__ = "api_keys"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)

    # 密钥信息（存储哈希值）
    key_hash = Column(String(255), nullable=False, unique=True)
    key_prefix = Column(String(10), nullable=False)  # 用于显示前8位

    # 密钥名称
    name = Column(String(100), nullable=False)

    # 所属用户
    account_id = Column(UUID(as_uuid=True), ForeignKey("accounts.id"), nullable=False)

    # 权限范围
    scopes = Column(JSON, default=list)  # List[APIKeyScope]

    # 使用限制
    rate_limit = Column(Integer, default=1000)  # 每小时请求限制
    ip_whitelist = Column(JSON, default=list)  # IP 白名单

    # 过期时间
    expires_at = Column(DateTime, nullable=True)

    # 最后使用时间
    last_used_at = Column(DateTime, nullable=True)

    # 使用统计
    total_requests = Column(Integer, default=0)

    # 是否启用
    is_active = Column(Boolean, default=True)

    # 撤销原因
    revoked_at = Column(DateTime, nullable=True)
    revoke_reason = Column(String(255), nullable=True)

    # 创建信息
    created_at = Column(DateTime, default=datetime.utcnow)

    @property
    def is_expired(self) -> bool:
        """是否已过期"""
        if not self.expires_at:
            return False
        return datetime.utcnow() > self.expires_at

    @property
    def is_valid(self) -> bool:
        """是否有效"""
        return self.is_active and not self.is_expired and self.revoked_at is None


class APIKeyUsageLog(Base):
    """API 密钥使用日志"""
    __tablename__ = "api_key_usage_logs"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    api_key_id = Column(UUID(as_uuid=True), ForeignKey("api_keys.id"), nullable=False)

    # 请求信息
    endpoint = Column(String(255), nullable=False)
    method = Column(String(10), nullable=False)
    status_code = Column(Integer, nullable=False)

    # 客户端信息
    ip_address = Column(String(50), nullable=True)
    user_agent = Column(String(500), nullable=True)

    # 时间
    requested_at = Column(DateTime, default=datetime.utcnow)


class APIKeyService:
    """API 密钥服务"""

    @staticmethod
    def generate_api_key() -> str:
        """
        生成 API 密钥
        格式: monika_<32位随机字符>
        """
        random_part = secrets.token_urlsafe(32)
        return f"monika_{random_part}"

    @staticmethod
    def hash_api_key(api_key: str) -> str:
        """哈希 API 密钥"""
        import hashlib
        return hashlib.sha256(api_key.encode()).hexdigest()

    @staticmethod
    def create_api_key(
        db,
        account_id: str,
        name: str,
        scopes: List[APIKeyScope],
        expires_in_days: Optional[int] = None,
        rate_limit: int = 1000,
        ip_whitelist: Optional[List[str]] = None
    ) -> tuple[APIKey, str]:
        """
        创建 API 密钥

        Returns:
            (APIKey对象, 原始密钥字符串) - 原始密钥只在创建时返回一次
        """
        # 生成密钥
        api_key = APIKeyService.generate_api_key()
        key_hash = APIKeyService.hash_api_key(api_key)
        key_prefix = api_key[:10]

        # 计算过期时间
        expires_at = None
        if expires_in_days:
            expires_at = datetime.utcnow() + timedelta(days=expires_in_days)

        # 创建记录
        db_key = APIKey(
            key_hash=key_hash,
            key_prefix=key_prefix,
            name=name,
            account_id=account_id,
            scopes=[s.value for s in scopes],
            rate_limit=rate_limit,
            ip_whitelist=ip_whitelist or [],
            expires_at=expires_at
        )

        db.add(db_key)
        db.commit()
        db.refresh(db_key)

        return db_key, api_key

    @staticmethod
    def verify_api_key(db, api_key: str) -> Optional[APIKey]:
        """
        验证 API 密钥

        Args:
            db: 数据库会话
            api_key: 原始密钥字符串

        Returns:
            APIKey 对象，如果密钥无效则返回 None
        """
        key_hash = APIKeyService.hash_api_key(api_key)

        db_key = db.query(APIKey).filter(
            APIKey.key_hash == key_hash
        ).first()

        if not db_key:
            return None

        if not db_key.is_valid:
            return None

        return db_key

    @staticmethod
    def check_rate_limit(db, api_key_id: str) -> bool:
        """检查速率限制"""
        from sqlalchemy import func

        # 获取密钥
        api_key = db.query(APIKey).filter(APIKey.id == api_key_id).first()
        if not api_key:
            return False

        # 检查过去一小时的请求数
        one_hour_ago = datetime.utcnow() - timedelta(hours=1)
        request_count = db.query(APIKeyUsageLog).filter(
            APIKeyUsageLog.api_key_id == api_key_id,
            APIKeyUsageLog.requested_at >= one_hour_ago
        ).count()

        return request_count < api_key.rate_limit

    @staticmethod
    def log_usage(
        db,
        api_key_id: str,
        endpoint: str,
        method: str,
        status_code: int,
        ip_address: Optional[str] = None,
        user_agent: Optional[str] = None
    ):
        """记录使用"""
        log = APIKeyUsageLog(
            api_key_id=api_key_id,
            endpoint=endpoint,
            method=method,
            status_code=status_code,
            ip_address=ip_address,
            user_agent=user_agent
        )
        db.add(log)

        # 更新最后使用时间和请求总数
        api_key = db.query(APIKey).filter(APIKey.id == api_key_id).first()
        if api_key:
            api_key.last_used_at = datetime.utcnow()
            api_key.total_requests += 1

        db.commit()

    @staticmethod
    def revoke_key(
        db,
        api_key_id: str,
        reason: Optional[str] = None
    ) -> bool:
        """撤销密钥"""
        api_key = db.query(APIKey).filter(APIKey.id == api_key_id).first()

        if not api_key:
            return False

        api_key.is_active = False
        api_key.revoked_at = datetime.utcnow()
        api_key.revoke_reason = reason

        db.commit()
        return True

    @staticmethod
    def rotate_key(
        db,
        api_key_id: str
    ) -> tuple[APIKey, str]:
        """
        轮换密钥（生成新密钥，保留旧配置）

        Returns:
            (新APIKey对象, 新原始密钥)
        """
        old_key = db.query(APIKey).filter(APIKey.id == api_key_id).first()

        if not old_key:
            raise ValueError("API key not found")

        # 创建新密钥
        new_key, raw_key = APIKeyService.create_api_key(
            db,
            old_key.account_id,
            old_key.name,
            [APIKeyScope(s) for s in old_key.scopes],
            rate_limit=old_key.rate_limit,
            ip_whitelist=old_key.ip_whitelist
        )

        # 撤销旧密钥
        APIKeyService.revoke_key(db, api_key_id, "Key rotated")

        return new_key, raw_key

    @staticmethod
    def get_user_keys(db, account_id: str) -> List[APIKey]:
        """获取用户的所有密钥"""
        return db.query(APIKey).filter(
            APIKey.account_id == account_id
        ).order_by(
            APIKey.created_at.desc()
        ).all()
```

### 认证中间件

```python
# backend/app/api/deps/api_key.py
from fastapi import Header, HTTPException, Depends, status
from sqlalchemy.orm import Session

from app.api.deps import get_db
from app.services.api_key_service import APIKeyService


async def get_api_key(
    x_api_key: str = Header(..., description="API Key"),
    db: Session = Depends(get_db)
):
    """
    API 密钥认证依赖

    Usage:
        @router.get("/protected")
        async def protected_route(api_key = Depends(get_api_key)):
            ...
    """
    # 验证密钥
    db_key = APIKeyService.verify_api_key(db, x_api_key)

    if not db_key:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid API key"
        )

    # 检查速率限制
    if not APIKeyService.check_rate_limit(db, str(db_key.id)):
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Rate limit exceeded"
        )

    return db_key


async def get_api_key_with_scope(required_scope: str):
    """
    带权限检查的 API 密钥认证

    Usage:
        @router.get("/admin")
        async def admin_route(api_key = Depends(get_api_key_with_scope("admin"))):
            ...
    """
    async def verify_scope(
        api_key = Depends(get_api_key)
    ):
        if required_scope not in api_key.scopes:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail=f"Missing required scope: {required_scope}"
            )
        return api_key

    return verify_scope
```

### API 路由

```python
# backend/app/api/api_keys.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.api.deps import get_db, get_current_active_user
from app.schemas.api_keys import (
    APIKeyCreate,
    APIKeyResponse,
    APIKeyCreateResponse
)
from app.services.api_key_service import APIKeyService
from app.models.api_keys import APIKeyScope

router = APIRouter()


@router.post("/", response_model=APIKeyCreateResponse)
def create_api_key(
    key_in: APIKeyCreate,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """创建 API 密钥"""
    scopes = [APIKeyScope(s) for s in key_in.scopes]

    db_key, raw_key = APIKeyService.create_api_key(
        db,
        current_user.id,
        key_in.name,
        scopes,
        key_in.expires_in_days,
        key_in.rate_limit,
        key_in.ip_whitelist
    )

    return {
        "key": db_key,
        "raw_key": raw_key  # 只在创建时返回一次
    }


@router.get("/", response_model=List[APIKeyResponse])
def list_api_keys(
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """列出所有 API 密钥"""
    return APIKeyService.get_user_keys(db, current_user.id)


@router.get("/{key_id}", response_model=APIKeyResponse)
def get_api_key(
    key_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """获取 API 密钥详情"""
    api_key = db.query(APIKey).filter(
        APIKey.id == key_id,
        APIKey.account_id == current_user.id
    ).first()

    if not api_key:
        raise HTTPException(status_code=404, detail="API key not found")

    return api_key


@router.post("/{key_id}/revoke")
def revoke_api_key(
    key_id: str,
    reason: str = None,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """撤销 API 密钥"""
    api_key = db.query(APIKey).filter(
        APIKey.id == key_id,
        APIKey.account_id == current_user.id
    ).first()

    if not api_key:
        raise HTTPException(status_code=404, detail="API key not found")

    success = APIKeyService.revoke_key(db, key_id, reason)

    if not success:
        raise HTTPException(status_code=400, detail="Failed to revoke key")

    return {"message": "API key revoked successfully"}


@router.post("/{key_id}/rotate", response_model=APIKeyCreateResponse)
def rotate_api_key(
    key_id: str,
    db: Session = Depends(get_db),
    current_user = Depends(get_current_active_user)
):
    """轮换 API 密钥"""
    api_key = db.query(APIKey).filter(
        APIKey.id == key_id,
        APIKey.account_id == current_user.id
    ).first()

    if not api_key:
        raise HTTPException(status_code=404, detail="API key not found")

    new_key, raw_key = APIKeyService.rotate_key(db, key_id)

    return {
        "key": new_key,
        "raw_key": raw_key
    }
```

---

## 完整前端代码示例 (TypeScript + React + shadcn/ui)

### 类型定义

```typescript
// frontend/src/types/api-keys.ts
export enum APIKeyScope {
  READ = "read",
  WRITE = "write",
  ADMIN = "admin",
  GAME = "game",
  WEBHOOK = "webhook"
}

export interface APIKey {
  id: string;
  key_prefix: string;
  name: string;
  scopes: APIKeyScope[];
  rate_limit: number;
  ip_whitelist: string[];
  expires_at: string | null;
  last_used_at: string | null;
  total_requests: number;
  is_active: boolean;
  revoked_at: string | null;
  created_at: string;
}

export interface APIKeyCreateRequest {
  name: string;
  scopes: APIKeyScope[];
  expires_in_days: number | null;
  rate_limit: number;
  ip_whitelist: string[];
}
```

### API 密钥管理组件

```tsx
// frontend/src/components/settings/APIKeyManagement.tsx
import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Plus, Copy, Trash2, RotateCcw, AlertTriangle } from "lucide-react";

import { APIKey, APIKeyScope, APIKeyCreateRequest } from "@/types/api-keys";

export function APIKeyManagement() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    loadKeys();
  }, []);

  const loadKeys = async () => {
    const res = await fetch("/api/api-keys");
    const data = await res.json();
    setKeys(data);
  };

  const handleCopyKey = (key: string) => {
    navigator.clipboard.writeText(key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleRevoke = async (keyId: string) => {
    if (!confirm("确定要撤销此 API 密钥吗？此操作不可撤销。")) return;

    await fetch(`/api/api-keys/${keyId}/revoke`, { method: "POST" });
    loadKeys();
  };

  const handleRotate = async (keyId: string) => {
    if (!confirm("轮换密钥将使旧密钥失效，确定继续吗？")) return;

    const res = await fetch(`/api/api-keys/${keyId}/rotate`, { method: "POST" });
    const data = await res.json();
    setNewKey(data.raw_key);
    loadKeys();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">API 密钥</h2>
          <p className="text-muted-foreground">管理用于第三方集成的 API 密钥</p>
        </div>
        <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="w-4 h-4 mr-2" />
              创建密钥
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>创建 API 密钥</DialogTitle>
            </DialogHeader>
            <CreateKeyForm
              onSuccess={(rawKey) => {
                setNewKey(rawKey);
                setShowCreateDialog(false);
                loadKeys();
              }}
              onCancel={() => setShowCreateDialog(false)}
            />
          </DialogContent>
        </Dialog>
      </div>

      {/* 新密钥显示 */}
      {newKey && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <div className="space-y-2">
              <p className="font-semibold">请立即复制您的 API 密钥</p>
              <p className="text-sm text-muted-foreground">
                此密钥只会显示一次，请妥善保管。
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-muted px-3 py-2 rounded text-sm">
                  {newKey}
                </code>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleCopyKey(newKey)}
                >
                  <Copy className="w-4 h-4 mr-2" />
                  {copied ? "已复制" : "复制"}
                </Button>
              </div>
            </div>
          </AlertDescription>
        </Alert>
      )}

      {/* 密钥列表 */}
      <div className="space-y-4">
        {keys.map((key) => (
          <Card key={key.id}>
            <CardContent className="p-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold">{key.name}</h3>
                    {key.is_active ? (
                      <Badge variant="default">活跃</Badge>
                    ) : (
                      <Badge variant="destructive">已撤销</Badge>
                    )}
                    {key.revoked_at && (
                      <Badge variant="outline">已过期</Badge>
                    )}
                  </div>

                  <code className="text-sm bg-muted px-2 py-1 rounded mt-2 inline-block">
                    {key.key_prefix}...
                  </code>

                  <div className="flex flex-wrap gap-2 mt-3">
                    {key.scopes.map((scope) => (
                      <Badge key={scope} variant="secondary" className="text-xs">
                        {scope}
                      </Badge>
                    ))}
                  </div>

                  <div className="grid grid-cols-3 gap-4 mt-3 text-sm text-muted-foreground">
                    <div>
                      <span className="font-medium">请求限制:</span> {key.rate_limit}/小时
                    </div>
                    <div>
                      <span className="font-medium">总请求:</span> {key.total_requests}
                    </div>
                    <div>
                      <span className="font-medium">创建时间:</span>{" "}
                      {new Date(key.created_at).toLocaleDateString()}
                    </div>
                  </div>
                </div>

                <div className="flex gap-2">
                  {key.is_active && (
                    <>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRotate(key.id)}
                      >
                        <RotateCcw className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRevoke(key.id)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

// 创建密钥表单组件
function CreateKeyForm({
  onSuccess,
  onCancel
}: {
  onSuccess: (key: string) => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<APIKeyScope[]>([APIKeyScope.READ]);
  const [expiresInDays, setExpiresInDays] = useState<number | null>(null);
  const [rateLimit, setRateLimit] = useState(1000);
  const [creating, setCreating] = useState(false);

  const handleScopeToggle = (scope: APIKeyScope) => {
    setScopes((prev) =>
      prev.includes(scope)
        ? prev.filter((s) => s !== scope)
        : [...prev, scope]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);

    try {
      const res = await fetch("/api/api-keys", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name,
          scopes,
          expires_in_days: expiresInDays,
          rate_limit: rateLimit,
          ip_whitelist: []
        })
      });

      const data = await res.json();
      onSuccess(data.raw_key);
    } finally {
      setCreating(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="name">密钥名称</Label>
        <Input
          id="name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="例如: 生产环境集成"
          required
        />
      </div>

      <div className="space-y-2">
        <Label>权限范围</Label>
        <div className="flex flex-wrap gap-2">
          {Object.values(APIKeyScope).map((scope) => (
            <Badge
              key={scope}
              variant={scopes.includes(scope) ? "default" : "outline"}
              className="cursor-pointer"
              onClick={() => handleScopeToggle(scope)}
            >
              {scope}
            </Badge>
          ))}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="rateLimit">速率限制（每小时请求数）</Label>
        <Input
          id="rateLimit"
          type="number"
          value={rateLimit}
          onChange={(e) => setRateLimit(parseInt(e.target.value))}
          min={1}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="expires">过期时间（天）</Label>
        <Input
          id="expires"
          type="number"
          value={expiresInDays || ""}
          onChange={(e) => setExpiresInDays(e.target.value ? parseInt(e.target.value) : null)}
          placeholder="留空表示永不过期"
          min={1}
        />
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button type="submit" disabled={creating || !name}>
          {creating ? "创建中..." : "创建密钥"}
        </Button>
      </div>
    </form>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `backend/app/models/api_keys.py` | 创建 | API 密钥数据模型 |
| `backend/app/services/api_key_service.py` | 创建 | API 密钥服务 |
| `backend/app/api/deps/api_key.py` | 创建 | API 密钥认证依赖 |
| `backend/app/api/api_keys.py` | 创建 | API 密钥管理路由 |
| `backend/app/schemas/api_keys.py` | 创建 | Pydantic 模型 |
| `backend/app/db/migrations/versions/xxx_create_api_keys.py` | 创建 | 数据库迁移 |
| `frontend/src/types/api-keys.ts` | 创建 | 类型定义 |
| `frontend/src/components/settings/APIKeyManagement.tsx` | 创建 | API 密钥管理组件 |
| `frontend/src/pages/Settings.tsx` | 修改 | 添加 API 密钥管理选项卡 |

---

## 验收标准

- [ ] 用户可以创建 API 密钥
- [ ] 密钥安全存储（哈希）
- [ ] 密钥创建后只显示一次完整值
- [ ] 支持权限范围控制
- [ ] 支持速率限制
- [ ] 支持 IP 白名单
- [ ] 支持密钥撤销和轮换
- [ ] 记录密钥使用历史

---

## 参考文档

- API 密钥安全最佳实践
- OWASP API 安全指南
- FastAPI 认证与授权

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
