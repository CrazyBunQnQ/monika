# M4-006: 实现场景包版本控制

**任务ID**: M4-006
**标题**: 实现场景包版本控制
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M4-001

---

## 任务描述

实现场景包版本控制功能，支持版本管理、历史记录、回滚等操作。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-006-01 | 设计版本数据模型 | Version Model | 20min |
| M4-006-02 | 实现版本服务 | Version Service | 30min |
| M4-006-03 | 实现版本创建 | Create Version | 25min |
| M4-006-04 | 实现版本回滚 | Rollback | 25min |
| M4-006-05 | 实现版本对比 | Diff | 30min |
| M4-006-06 | 编写版本测试 | 测试覆盖 | 15min |

---

## 版本数据模型

```python
# app/db/models/version.py
from sqlalchemy import Column, String, Text, ForeignKey, Integer, DateTime
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class ScenePackageVersion(Base):
    """场景包版本"""
    __tablename__ = 'scene_package_versions'

    id = Column(String, primary_key=True, index=True)
    package_id = Column(String, ForeignKey('scene_packages.id'), nullable=False, index=True)

    # 版本信息
    version = Column(String, nullable=False)  # 如 "1.0.0", "1.1.0"
    version_number = Column(Integer, nullable=False)  # 递增版本号

    # 变更内容
    changelog = Column(Text)
    changes = Column(JSON)  # 详细变更记录

    # 完整数据快照
    data_snapshot = Column(JSON, nullable=False)

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False)

    # 关系
    package = relationship("ScenePackage", back_populates="versions")
    creator = relationship("User", back_populates="created_versions")

    def __repr__(self):
        return f"<ScenePackageVersion {self.package_id}@{self.version}>"
```

---

## 版本控制服务

```python
# app/services/version_control.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime
import json

from app.db.models.version import ScenePackageVersion
from app.db.models.scene import ScenePackage
from app.core.security import generate_id

class VersionControlService:
    """版本控制服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_version(
        self,
        package_id: str,
        version: str,
        changelog: str,
        created_by: str,
        changes: List[Dict[str, Any]] = None,
    ) -> ScenePackageVersion:
        """创建新版本"""
        # 获取当前包数据
        package = self.db.query(ScenePackage)\
            .filter(ScenePackage.id == package_id)\
            .first()

        if not package:
            raise ValueError("场景包不存在")

        # 获取下一个版本号
        version_number = self._get_next_version_number(package_id)

        # 创建快照
        snapshot = {
            "metadata": package.metadata,
            "scenes": package.scenes,
            "npcs": package.npcs,
            "clues": package.clues,
            "handouts": package.handouts,
        }

        version_obj = ScenePackageVersion(
            id=generate_id('version'),
            package_id=package_id,
            version=version,
            version_number=version_number,
            changelog=changelog,
            changes=changes or [],
            data_snapshot=snapshot,
            created_by=created_by,
        )

        self.db.add(version_obj)

        # 更新包的当前版本
        package.current_version = version

        self.db.commit()
        self.db.refresh(version_obj)

        return version_obj

    def _get_next_version_number(self, package_id: str) -> int:
        """获取下一个版本号"""
        latest = self.db.query(ScenePackageVersion)\
            .filter(ScenePackageVersion.package_id == package_id)\
            .order_by(ScenePackageVersion.version_number.desc())\
            .first()

        return (latest.version_number + 1) if latest else 1

    def get_versions(self, package_id: str) -> List[ScenePackageVersion]:
        """获取版本列表"""
        return self.db.query(ScenePackageVersion)\
            .filter(ScenePackageVersion.package_id == package_id)\
            .order_by(ScenePackageVersion.version_number.desc())\
            .all()

    def get_version(self, version_id: str) -> Optional[ScenePackageVersion]:
        """获取单个版本"""
        return self.db.query(ScenePackageVersion)\
            .filter(ScenePackageVersion.id == version_id)\
            .first()

    def rollback_to_version(
        self,
        package_id: str,
        version_id: str,
        created_by: str,
    ) -> ScenePackage:
        """回滚到指定版本"""
        version = self.get_version(version_id)
        if not version or version.package_id != package_id:
            raise ValueError("版本不存在")

        package = self.db.query(ScenePackage)\
            .filter(ScenePackage.id == package_id)\
            .first()

        if not package:
            raise ValueError("场景包不存在")

        # 恢复快照数据
        snapshot = version.data_snapshot
        package.metadata = snapshot.get("metadata", {})
        package.scenes = snapshot.get("scenes", [])
        package.npcs = snapshot.get("npcs", [])
        package.clues = snapshot.get("clues", [])
        package.handouts = snapshot.get("handouts", [])

        # 创建回滚版本
        self.create_version(
            package_id=package_id,
            version=f"{package.current_version}-rollback",
            changelog=f"回滚到 {version.version}",
            created_by=created_by,
            changes=[{"type": "rollback", "to_version": version.version}],
        )

        self.db.commit()
        self.db.refresh(package)

        return package

    def compare_versions(
        self,
        version_id_1: str,
        version_id_2: str,
    ) -> Dict[str, Any]:
        """比较两个版本的差异"""
        v1 = self.get_version(version_id_1)
        v2 = self.get_version(version_id_2)

        if not v1 or not v2:
            raise ValueError("版本不存在")

        snapshot1 = v1.data_snapshot
        snapshot2 = v2.data_snapshot

        # 比较各个部分
        diff = {
            "metadata": self._compare_dict(snapshot1.get("metadata", {}), snapshot2.get("metadata", {})),
            "scenes": self._compare_list(snapshot1.get("scenes", []), snapshot2.get("scenes", []), "id"),
            "npcs": self._compare_list(snapshot1.get("npcs", []), snapshot2.get("npcs", []), "id"),
            "clues": self._compare_list(snapshot1.get("clues", []), snapshot2.get("clues", []), "id"),
            "handouts": self._compare_list(snapshot1.get("handouts", []), snapshot2.get("handouts", []), "id"),
        }

        return diff

    def _compare_dict(self, dict1: Dict, dict2: Dict) -> List[Dict[str, Any]]:
        """比较两个字典的差异"""
        diff = []

        all_keys = set(dict1.keys()) | set(dict2.keys())

        for key in all_keys:
            val1 = dict1.get(key)
            val2 = dict2.get(key)

            if val1 != val2:
                diff.append({
                    "key": key,
                    "old": val1,
                    "new": val2,
                    "type": "changed" if key in dict1 else "added",
                })

        return diff

    def _compare_list(self, list1: List, list2: List, key_field: str) -> List[Dict[str, Any]]:
        """比较两个列表的差异"""
        diff = []

        map1 = {item.get(key_field): item for item in list1 if key_field in item}
        map2 = {item.get(key_field): item for item in list2 if key_field in item}

        all_keys = set(map1.keys()) | set(map2.keys())

        for key in all_keys:
            item1 = map1.get(key)
            item2 = map2.get(key)

            if item1 and item2:
                # 比较内容
                if item1 != item2:
                    diff.append({
                        "key": key,
                        "old": item1,
                        "new": item2,
                        "type": "changed",
                    })
            elif item2:
                diff.append({
                    "key": key,
                    "new": item2,
                    "type": "added",
                })
            elif item1:
                diff.append({
                    "key": key,
                    "old": item1,
                    "type": "removed",
                })

        return diff

    def get_version_history(self, package_id: str) -> List[Dict[str, Any]]:
        """获取版本历史摘要"""
        versions = self.get_versions(package_id)

        return [
            {
                "id": v.id,
                "version": v.version,
                "version_number": v.version_number,
                "changelog": v.changelog,
                "created_at": v.created_at.isoformat(),
                "created_by": v.creator.username,
                "changes_count": len(v.changes) if v.changes else 0,
            }
            for v in versions
        ]
```

---

## 版本控制 API

```python
# app/api/versions.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.version_control import VersionControlService

router = APIRouter(prefix="/versions", tags=["versions"])

class CreateVersionRequest(BaseModel):
    package_id: str
    version: str
    changelog: str
    changes: Optional[List[dict]] = None

@router.post("")
async def create_version(
    request: CreateVersionRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建新版本"""
    service = VersionControlService(db)

    try:
        version = service.create_version(
            package_id=request.package_id,
            version=request.version,
            changelog=request.changelog,
            created_by=current_user.id,
            changes=request.changes,
        )
        return {"version_id": version.id, "version": version.version}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.get("/package/{package_id}")
async def list_versions(
    package_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取版本列表"""
    service = VersionControlService(db)
    return service.get_version_history(package_id)

@router.get("/{version_id}")
async def get_version(
    version_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取版本详情"""
    service = VersionControlService(db)
    version = service.get_version(version_id)

    if not version:
        raise HTTPException(status_code=404, detail="版本不存在")

    return {
        "id": version.id,
        "version": version.version,
        "changelog": version.changelog,
        "changes": version.changes,
        "created_at": version.created_at,
        "snapshot": version.data_snapshot,
    }

@router.post("/rollback/{version_id}")
async def rollback_to_version(
    version_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """回滚到指定版本"""
    service = VersionControlService(db)
    version = service.get_version(version_id)

    if not version:
        raise HTTPException(status_code=404, detail="版本不存在")

    package = service.rollback_to_version(version.package_id, version_id, current_user.id)

    return {
        "message": f"已回滚到 {version.version}",
        "package_id": package.id,
    }

@router.get("/compare/{version_id_1}/{version_id_2}")
async def compare_versions(
    version_id_1: str,
    version_id_2: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """比较两个版本"""
    service = VersionControlService(db)
    return service.compare_versions(version_id_1, version_id_2)
```

---

## 前端版本控制组件

```tsx
// frontend/src/components/scene/VersionControl.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { History, RotateCcw, GitCompareArrows } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

interface Version {
  id: string
  version: string
  version_number: number
  changelog: string
  created_at: string
  created_by: string
  changes_count: number
}

interface VersionControlProps {
  packageId: string
}

export function VersionControl({ packageId }: VersionControlProps) {
  const [versions, setVersions] = useState<Version[]>([])
  const [selectedVersions, setSelectedVersions] = useState<string[]>([])
  const [showCompareDialog, setShowCompareDialog] = useState(false)
  const [diff, setDiff] = useState<any>(null)

  const { toast } = useToast()

  useEffect(() => {
    loadVersions()
  }, [packageId])

  const loadVersions = async () => {
    try {
      const response = await fetch(`/api/versions/package/${packageId}`)
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setVersions(data)
    } catch (error) {
      console.error('Failed to load versions:', error)
    }
  }

  const handleRollback = async (versionId: string) => {
    if (!confirm('确定要回滚到此版本吗？当前版本将被保存为新版本。')) {
      return
    }

    try {
      const response = await fetch(`/api/versions/rollback/${versionId}`, {
        method: 'POST',
      })

      if (!response.ok) throw new Error('回滚失败')

      toast({
        title: '回滚成功',
        description: '场景包已回滚到指定版本',
      })

      await loadVersions()
    } catch (error) {
      toast({
        title: '回滚失败',
        variant: 'destructive',
      })
    }
  }

  const handleCompare = async () => {
    if (selectedVersions.length !== 2) {
      toast({
        title: '请选择两个版本进行比较',
        variant: 'destructive',
      })
      return
    }

    try {
      const response = await fetch(`/api/versions/compare/${selectedVersions[0]}/${selectedVersions[1]}`)
      if (!response.ok) throw new Error('比较失败')

      const data = await response.json()
      setDiff(data)
      setShowCompareDialog(true)
    } catch (error) {
      toast({
        title: '比较失败',
        variant: 'destructive',
      })
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center">
          <History className="h-4 w-4 mr-2" />
          版本历史 ({versions.length})
        </CardTitle>
      </CardHeader>

      <CardContent>
        {/* 版本列表 */}
        <div className="space-y-2 max-h-64 overflow-y-auto">
          {versions.map((version, index) => (
            <div
              key={version.id}
              className="flex items-center justify-between p-3 border rounded hover:bg-muted"
            >
              <div className="flex-1">
                <div className="flex items-center space-x-2">
                  <span className="font-medium text-sm">v{version.version}</span>
                  {index === 0 && (
                    <Badge variant="secondary" className="text-xs">当前</Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-1 line-clamp-1">
                  {version.changelog || '无变更说明'}
                </p>
                <p className="text-xs text-muted-foreground">
                  {new Date(version.created_at).toLocaleString('zh-CN')} • {version.created_by}
                </p>
              </div>

              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={selectedVersions.includes(version.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedVersions(prev =>
                        prev.length < 2 ? [...prev, version.id] : prev
                      )
                    } else {
                      setSelectedVersions(prev => prev.filter(id => id !== version.id))
                    }
                  }}
                  className="h-4 w-4"
                />
                {index > 0 && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleRollback(version.id)}
                  >
                    <RotateCcw className="h-3 w-3 mr-1" />
                    回滚
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* 比较按钮 */}
        {selectedVersions.length === 2 && (
          <Button
            className="w-full mt-3"
            onClick={handleCompare}
          >
            <GitCompareArrows className="h-4 w-4 mr-2" />
            比较版本
          </Button>
        )}
      </CardContent>

      {/* 比较对话框 */}
      <Dialog open={showCompareDialog} onOpenChange={setShowCompareDialog}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>版本比较</DialogTitle>
          </DialogHeader>

          {diff && (
            <div className="space-y-4">
              {Object.entries(diff).map(([section, changes]) => (
                <div key={section}>
                  <h3 className="font-medium mb-2 capitalize">{section}</h3>
                  {Array.isArray(changes) && changes.length > 0 ? (
                    <div className="space-y-1">
                      {changes.map((change: any, index: number) => (
                        <div
                          key={index}
                          className={`p-2 rounded text-sm ${
                            change.type === 'added' ? 'bg-green-50 dark:bg-green-900/20' :
                            change.type === 'removed' ? 'bg-red-50 dark:bg-red-900/20' :
                            'bg-yellow-50 dark:bg-yellow-900/20'
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="font-medium">{change.key}</span>
                            <Badge variant="outline" className="text-xs">
                              {change.type === 'added' ? '新增' :
                               change.type === 'removed' ? '删除' : '修改'}
                            </Badge>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">无变更</p>
                  )}
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/version.py` | 创建 | 版本数据模型 |
| `app/services/version_control.py` | 创建 | 版本控制服务 |
| `app/api/versions.py` | 创建 | 版本 API |
| `frontend/src/components/scene/VersionControl.tsx` | 创建 | 版本控制组件 |

---

## 验收标准

- [ ] 版本创建成功
- [ ] 版本列表正确
- [ ] 回滚功能有效
- [ ] 版本对比准确
- [ ] 变更记录完整
- [ ] 快照数据完整

---

## 参考文档

- M4-001: 场景包上传功能
- Git 版本控制概念

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
