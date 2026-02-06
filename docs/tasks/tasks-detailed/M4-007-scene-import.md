# M4-007: 实现场景包导入功能

**任务ID**: M4-007
**标题**: 实现场景包导入功能
**类型**: fullstack (全栈开发)
**预估工时**: 2.5h
**依赖**: M4-001, M4-008

---

## 任务描述

实现场景包的导入功能，支持从外部 ZIP 文件或在线仓库导入预制的场景包。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-007-01 | 设计导入流程 | Import Flow | 20min |
| M4-007-02 | 实现文件解析 | File Parser | 30min |
| M4-007-03 | 实现数据验证 | Validation | 30min |
| M4-007-04 | 实现批量导入 | Batch Import | 25min |
| M4-007-05 | 实现冲突处理 | Conflict Handling | 25min |
| M4-007-06 | 实现导入历史 | Import History | 20min |
| M4-007-07 | 编写导入测试 | 测试覆盖 | 15min |

---

## 场景导入服务

```python
# app/services/scene_import.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session
import zipfile
import json
import tempfile
import os
from pathlib import Path

from app.services.scene_parser import SceneParser
from app.db.models.scene import ScenePackage
from app.core.security import generate_id

class SceneImportService:
    """场景导入服务"""

    def __init__(self, db: Session, upload_dir: str = "data/scenes"):
        self.db = db
        self.upload_dir = Path(upload_dir)
        self.upload_dir.mkdir(parents=True, exist_ok=True)
        self.parser = SceneParser(db)

    async def import_from_zip(
        self,
        zip_file_path: str,
        user_id: str,
        room_id: str = None,
        overwrite: bool = False,
    ) -> Dict[str, Any]:
        """从 ZIP 文件导入场景包"""
        try:
            with zipfile.ZipFile(zip_file_path, 'r') as zip_ref:
                # 验证 ZIP 结构
                if 'scene.json' not in zip_ref.namelist():
                    return {
                        "success": False,
                        "error": "缺少 scene.json 文件",
                    }

                # 读取并解析场景数据
                with zip_ref.open('scene.json') as f:
                    scene_data = json.load(f)

                # 验证场景数据
                validation = await self.parser.validate_scene_data(scene_data)
                if not validation["valid"]:
                    return {
                        "success": False,
                        "error": "场景数据验证失败",
                        "errors": validation.get("errors", []),
                    }

                # 检查是否已存在同名场景
                existing = self.db.query(ScenePackage)\
                    .filter(
                        ScenePackage.metadata['name'] == scene_data['metadata']['name']
                    )\
                    .first()

                if existing and not overwrite:
                    return {
                        "success": False,
                        "error": f"场景包 '{scene_data['metadata']['name']}' 已存在",
                        "existing_id": existing.id,
                        "suggestion": "使用 overwrite 参数覆盖",
                    }

                # 解压资源文件
                assets_dir = self.upload_dir / f"{generate_id('scene')}_assets"
                assets_dir.mkdir(parents=True, exist_ok=True)

                extracted_files = []
                for file in zip_ref.namelist():
                    if file.startswith('assets/'):
                        # 解压资源文件
                        zip_ref.extract(file, assets_dir)
                        extracted_files.append(file)
                    elif file == 'scene.json':
                        # scene.json 已处理
                        pass

                # 创建场景包记录
                scene_package = ScenePackage(
                    id=generate_id('scene_package'),
                    uploaded_by=user_id,
                    room_id=room_id,
                    file_path=zip_file_path,
                    metadata=scene_data['metadata'],
                    scene_count=len(scene_data.get('scenes', [])),
                    npc_count=len(scene_data.get('npcs', [])),
                    clue_count=len(scene_data.get('clues', [])),
                    handout_count=len(scene_data.get('handouts', [])),
                )

                self.db.add(scene_package)
                self.db.commit()
                self.db.refresh(scene_package)

                # 保存详细数据
                await self.parser.import_scene_data(
                    scene_package.id,
                    scene_data,
                    assets_dir=str(assets_dir),
                )

                return {
                    "success": True,
                    "package_id": scene_package.id,
                    "name": scene_package.metadata['name'],
                    "scenes_count": scene_package.scene_count,
                    "extracted_files": len(extracted_files),
                }

        except zipfile.BadZipFile:
            return {
                "success": False,
                "error": "无效的 ZIP 文件",
            }
        except Exception as e:
            return {
                "success": False,
                "error": f"导入失败: {str(e)}",
            }

    async def import_from_url(
        self,
        url: str,
        user_id: str,
        room_id: str = None,
    ) -> Dict[str, Any]:
        """从 URL 导入场景包"""
        import aiohttp
        import tempfile
        import shutil

        try:
            async with aiohttp.ClientSession() as session:
                async with session.get(url) as response:
                    if response.status != 200:
                        return {
                            "success": False,
                            "error": f"下载失败: {response.status}",
                        }

                    # 保存到临时文件
                    suffix = '.zip'
                    with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
                        tmp.write(await response.read())
                        tmp_path = tmp.name

            # 导入 ZIP
            result = await self.import_from_zip(tmp_path, user_id, room_id)

            # 清理临时文件
            try:
                os.unlink(tmp_path)
            except:
                pass

            return result

        except Exception as e:
            return {
                "success": False,
                "error": f"导入失败: {str(e)}",
            }

    def get_import_history(
        self,
        user_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """获取导入历史"""
        # 从事件日志获取导入记录
        from app.db.models.event import EventLog

        events = self.db.query(EventLog)\
            .filter(
                EventLog.event_type == 'scene_import',
                EventLog.user_id == user_id,
            )\
            .order_by(EventLog.created_at.desc())\
            .limit(limit)\
            .all()

        return [
            {
                "event_id": e.id,
                "package_id": e.data.get('package_id'),
                "package_name": e.data.get('package_name'),
                "status": e.data.get('status'),
                "created_at": e.created_at.isoformat(),
            }
            for e in events
        ]

    async def resolve_conflicts(
        self,
        existing_id: str,
        new_data: Dict,
        resolution: str,  # skip, overwrite, rename, merge
    ) -> Dict[str, Any]:
        """解决导入冲突"""
        existing = self.db.query(ScenePackage)\
            .filter(ScenePackage.id == existing_id)\
            .first()

        if not existing:
            return {"success": False, "error": "原场景包不存在"}

        if resolution == "skip":
            return {"success": True, "action": "skipped", "existing_id": existing_id}

        elif resolution == "overwrite":
            # 删除旧场景包，创建新的
            self.db.delete(existing)
            # ... 创建新场景包
            return {"success": True, "action": "overwritten", "existing_id": existing_id}

        elif resolution == "rename":
            # 重命名新场景包
            old_name = existing.metadata['name']
            new_data['metadata']['name'] = f"{old_name} (导入)"
            # ... 创建重命名后的场景包
            return {"success": True, "action": "renamed", "old_name": old_name}

        elif resolution == "merge":
            # 合并数据
            # TODO: 实现场景包合并逻辑
            return {"success": True, "action": "merged", "existing_id": existing_id}

        return {"success": False, "error": f"不支持的冲突解决方式: {resolution}"}
```

---

## 场景导入 API

```python
# app/api/import.py
from fastapi import APIRouter, Depends, UploadFile, File, Form
from fastapi.responses import JSONResponse
from sqlalchemy.orm import Session

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.scene_import import SceneImportService

router = APIRouter(prefix="/import", tags=["import"])

@router.post("/scene/zip")
async def import_scene_zip(
    file: UploadFile = File(..., description="场景包 ZIP 文件"),
    room_id: str = Form(None),
    overwrite: bool = Form(False),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """导入 ZIP 场景包"""
    # 保存上传的文件
    import tempfile
    import shutil

    suffix = '.zip'
    with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
        shutil.copyfileobj(file.file, tmp)

    try:
        service = SceneImportService(db)
        result = await service.import_from_zip(tmp.name, current_user.id, room_id, overwrite)

        if not result["success"]:
            return JSONResponse(content=result, status_code=400)

        # 记录导入事件
        # TODO: 添加事件日志

        return JSONResponse(content=result)

    finally:
        # 清理临时文件
        try:
            os.unlink(tmp.name)
        except:
            pass

@router.post("/scene/url")
async def import_scene_url(
    url: str,
    room_id: str = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """从 URL 导入场景包"""
    service = SceneImportService(db)
    result = await service.import_from_url(url, current_user.id, room_id)

    if not result["success"]:
        return JSONResponse(content=result, status_code=400)

    return JSONResponse(content=result)

@router.get("/history")
async def get_import_history(
    limit: int = 50,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取导入历史"""
    service = SceneImportService(db)
    history = service.get_import_history(current_user.id, limit)

    return {"history": history}
```

---

## 前端导入组件

```tsx
// frontend/src/components/scene/SceneImport.tsx
import { useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Upload, Link, Download, AlertCircle, Check, Loader2 } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

interface ImportResult {
  success: boolean
  package_id?: string
  name?: string
  error?: string
  existing_id?: string
  suggestion?: string
}

export function SceneImport() {
  const [uploading, setUploading] = useState(false)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [url, setUrl] = useState('')
  const [importingUrl, setImportingUrl] = useState(false)
  const [showConflictDialog, setShowConflictDialog] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)
  const { toast } = useToast()

  const handleFileUpload = async (file: File) => {
    setUploading(true)
    setResult(null)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await fetch('/api/import/scene/zip', {
        method: 'POST',
        body: formData,
      })

      const data = await response.json()

      if (!response.ok || !data.success) {
        setResult({
          success: false,
          error: data.error || '导入失败',
        })

        // 检查是否是冲突错误
        if (data.existing_id) {
          setShowConflictDialog(true)
        }

        return
      }

      setResult(data)

      toast({
        title: '导入成功',
        description: `已导入场景: ${data.name}`,
      })
    } catch (error) {
      setResult({
        success: false,
        error: '导入失败',
      })
    } finally {
      setUploading(false)
    }
  }

  const handleUrlImport = async () => {
    if (!url.trim()) return

    setImportingUrl(true)

    try {
      const response = await fetch('/api/scene/url', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
      })

      const data = await response.json()

      if (!response.ok || !data.success) {
        toast({
          title: '导入失败',
          description: data.error || '无法从 URL 导入',
          variant: 'destructive',
        })
        return
      }

      toast({
        title: '导入成功',
        description: `已导入场景: ${data.name}`,
      })

      setUrl('')
    } catch (error) {
      toast({
        title: '导入失败',
        variant: 'destructive',
      })
    } finally {
      setImportingUrl(false)
    }
  }

  const handleConflictResolve = async (resolution: string) => {
    if (!result) return

    // 重新导入，带上解决方式
    if (resolution === 'overwrite') {
      try {
        const formData = new FormData()
        if (fileInputRef.current?.files?.[0]) {
          formData.append('file', fileInputRef.current.files[0])
          formData.append('overwrite', 'true')
        }

        const response = await fetch('/api/import/scene/zip', {
          method: 'POST',
          body: formData,
        })

        const data = await response.json()
        setResult(data)
      } catch (error) {
        console.error('Failed to resolve conflict:', error)
      }
    }

    setShowConflictDialog(false)
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">场景包导入</CardTitle>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* 文件上传 */}
          <div>
            <input
              ref={fileInputRef}
              type="file"
              accept=".zip"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) handleFileUpload(file)
              }}
            />
            <Button
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
              className="w-full"
            >
              <Upload className="h-4 w-4 mr-2" />
              {uploading ? '导入中...' : '选择 ZIP 文件'}
            </Button>
          </div>

          {/* URL 导入 */}
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">或从 URL 导入:</p>
            <div className="flex space-x-2">
              <Input
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://example.com/scene.zip"
                disabled={importingUrl}
              />
              <Button
                onClick={handleUrlImport}
                disabled={importingUrl || !url.trim()}
              >
                <Link className="h-4 w-4 mr-2" />
                导入
              </Button>
            </div>
          </div>

          {/* 结果显示 */}
          {result && (
            <div className={`p-3 rounded ${
              result.success
                ? 'bg-green-50 dark:bg-green-900/20 text-green-800 dark:text-green-200'
                : 'bg-red-50 dark:bg-red-900/20 text-red-800 dark:text-red-200'
            }`}>
              {result.success ? (
                <div className="flex items-center">
                  <Check className="h-4 w-4 mr-2" />
                  <span>导入成功: {result.name}</span>
                  <Button
                    size="sm"
                    variant="outline"
                    className="ml-auto"
                    onClick={() => window.location.href = `/scenes/${result.package_id}`}
                  >
                    查看场景
                  </Button>
                </div>
              ) : (
                <div className="flex items-start">
                  <AlertCircle className="h-4 w-4 mr-2 flex-shrink-0" />
                  <div>
                    <p className="font-medium">导入失败</p>
                    <p className="text-sm opacity-80">{result.error}</p>
                    {result.suggestion && (
                      <p className="text-sm opacity-80 mt-1">
                        提示: {result.suggestion}
                      </p>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/scene_import.py` | 创建 | 场景导入服务 |
| `app/api/import.py` | 创建 | 导入 API |
| `frontend/src/components/scene/SceneImport.tsx` | 创建 | 导入组件 |
| `frontend/src/components/game/ConflictResolutionDialog.tsx` | 创建 | 冲突解决对话框 |

---

## 验收标准

- [ ] ZIP 导入成功
- [ ] URL 导入有效
- [ ] 数据验证严格
- [ ] 冲突处理安全
- [ ] 历史记录完整
- [ ] 错误提示友好

---

## 参考文档

- M4-001: 场景包上传功能
- M4-008: JSON 解析器

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
