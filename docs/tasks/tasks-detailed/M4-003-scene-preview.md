# M4-003: 实现场景预览功能

**任务ID**: M4-003
**标题**: 实现场景预览功能
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M4-001, M0-022

---

## 任务描述

实现场景包的在线预览功能，允许用户在上传前或上传后查看场景内容。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-003-01 | 设计预览 API 结构 | API Design | 15min |
| M4-003-02 | 实现场景提取 | Extract | 25min |
| M4-003-03 | 实现内容解析 | Parse | 25min |
| M4-003-04 | 实现预览数据生成 | Preview Data | 30min |
| M4-003-05 | 实现缩略图生成 | Thumbnail | 25min |
| M4-003-06 | 编写预览测试 | 测试覆盖 | 20min |

---

## 场景预览服务

```python
# app/services/scene_preview.py
from typing import Dict, Any, List, Optional
import zipfile
import json
from pathlib import Path

class ScenePreviewService:
    """场景预览服务"""

    def __init__(self, upload_dir: str = "data/scenes"):
        self.upload_dir = Path(upload_dir)

    async def preview_from_zip(
        self,
        zip_path: str,
    ) -> Dict[str, Any]:
        """从 ZIP 文件生成预览"""
        try:
            with zipfile.ZipFile(zip_path, 'r') as zip_ref:
                # 读取 scene.json
                try:
                    with zip_ref.open('scene.json') as f:
                        scene_data = json.load(f)
                except KeyError:
                    return {
                        "valid": False,
                        "error": "缺少 scene.json 文件"
                    }

                # 验证场景
                validation = await self._validate_scene(scene_data)
                if not validation["valid"]:
                    return validation

                # 提取预览信息
                preview = {
                    "valid": True,
                    "metadata": scene_data.get("metadata", {}),
                    "scenes_count": len(scene_data.get("scenes", [])),
                    "npcs_count": len(scene_data.get("npcs", [])),
                    "clues_count": len(scene_data.get("clues", [])),
                    "handouts_count": len(scene_data.get("handouts", [])),
                    "first_scene": self._get_first_scene(scene_data),
                    "assets": self._list_assets(zip_ref),
                }

                return preview

        except Exception as e:
            return {
                "valid": False,
                "error": f"预览失败: {str(e)}"
            }

    def _validate_scene(self, scene_data: Dict) -> Dict[str, Any]:
        """验证场景数据"""
        required_fields = ["metadata", "scenes"]
        errors = []

        for field in required_fields:
            if field not in scene_data:
                errors.append(f"缺少 {field}")

        return {
            "valid": len(errors) == 0,
            "errors": errors,
        }

    def _get_first_scene(self, scene_data: Dict) -> Optional[Dict]:
        """获取第一个场景信息"""
        scenes = scene_data.get("scenes", [])
        if not scenes:
            return None

        return {
            "id": scenes[0].get("id"),
            "name": scenes[0].get("name"),
            "description": scenes[0].get("description", ""),
            "location": scenes[0].get("location", ""),
        }

    def _list_assets(self, zip_ref: zipfile.ZipFile) -> List[str]:
        """列出资源文件"""
        assets = []

        for file in zip_ref.namelist():
            if file.startswith('assets/'):
                assets.append(file)

        return assets

    async def generate_preview_image(
        self,
        scene_data: Dict,
        output_path: str,
    ):
        """生成场景预览图"""
        # 使用第一个场景的封面图，或生成默认预览
        # 这里可以集成 Chart.js 或其他图表库
        pass
```

---

## 预览 API

```python
# app/api/preview.py
from fastapi import APIRouter, UploadFile, File
from fastapi.responses import JSONResponse
import tempfile
import shutil

from app.services.scene_preview import ScenePreviewService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/preview", tags=["preview"])
preview_service = ScenePreviewService()

@router.post("/scene")
async def preview_scene_upload(
    file: UploadFile = File(..., description="场景包 ZIP 文件"),
    current_user: User = Depends(get_current_user),
):
    """预览上传的场景包"""
    # 保存到临时文件
    suffix = '.zip'
    with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
        shutil.copyfileobj(file.file, tmp)

    try:
        # 生成预览
        preview = await preview_service.preview_from_zip(tmp.name)

        # 删除临时文件
        import os
        os.unlink(tmp.name)

        if not preview["valid"]:
            return JSONResponse(
                content=preview,
                status_code=400
            )

        return JSONResponse(content=preview)

    except Exception as e:
        # 清理临时文件
        if os.path.exists(tmp.name):
            os.unlink(tmp.name)

        return JSONResponse(
            content={"valid": False, "error": str(e)},
            status_code=500
        )

@router.get("/scene/{scene_id}")
async def preview_existing_scene(
    scene_id: str,
    current_user: User = Depends(get_current_user),
):
    """预览已存在的场景"""
    # 从数据库加载场景数据
    # 生成预览信息
    return JSONResponse(content={})
```

---

## 前端预览组件

```tsx
// frontend/src/components/scene/ScenePreview.tsx
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { FileText, Users, AlertTriangle } from 'lucide-react'
import { ScenePreviewDialog } from './ScenePreviewDialog'

interface ScenePreviewProps {
  zipFile?: File
  sceneId?: string
}

export function ScenePreview({ zipFile, sceneId }: ScenePreviewProps) {
  const [preview, setPreview] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showPreview, setShowPreview] = useState(false)

  const handlePreview = async () => {
    setLoading(true)
    setError(null)

    try {
      let response

      if (zipFile) {
        // 预览上传的文件
        const formData = new FormData()
        formData.append('file', zipFile)

        response = await fetch('/api/preview/scene', {
          method: 'POST',
          body: formData,
        })
      } else if (sceneId) {
        // 预览已存在的场景
        response = await fetch(`/api/preview/scene/${sceneId}`)
      } else {
        return
      }

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || '预览失败')
      }

      setPreview(data)
      setShowPreview(true)
    } catch (err: any) {
      setError(err.message || '预览失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      {zipFile && (
        <Button
          variant="outline"
          size="sm"
          onClick={handlePreview}
          disabled={loading}
        >
          <FileText className="h-4 w-4 mr-2" />
          {loading ? '预览中...' : '预览'}
        </Button>
      )}

      {error && (
        <div className="text-red-500 text-sm mt-2">
          {error}
        </div>
      )}

      {showPreview && preview && (
        <ScenePreviewDialog
          open={showPreview}
          onClose={() => setShowPreview(false)}
          preview={preview}
        />
      )}
    </div>
  )
}
```

---

## 预览对话框组件

```tsx
// frontend/src/components/scene/ScenePreviewDialog.tsx
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

interface ScenePreviewDialogProps {
  open: boolean
  onClose: () => void
  preview: any
}

export function ScenePreviewDialog({
  open,
  onClose,
  preview,
}: ScenePreviewDialogProps) {
  if (!preview || !preview.valid) {
    return (
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>预览失败</DialogTitle>
          </DialogHeader>
          <DialogFooter>
            <DialogDescription>
              {preview?.error || '无法预览此场景包'}
            </DialogDescription>
            <Button onClick={onClose}>关闭</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    )
  }

  const { metadata, first_scene, scenes_count, npcs_count, clues_count, handouts_count } = preview

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{metadata.name}</DialogTitle>
          <DialogDescription>
            v{metadata.version} • {metadata.author}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* 基本统计 */}
          <div className="grid grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold">{scenes_count}</div>
              <div className="text-xs text-muted-foreground">场景</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{npcs_count}</div>
              <div className="text-xs text-muted-foreground">NPC</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{clues_count}</div>
              <div className="text-xs text-muted-foreground">线索</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{handouts_count}</div>
              <div className="text-xs text-muted-foreground">手递物</div>
            </div>
          </div>

          {/* 标签 */}
          {metadata.tags && metadata.tags.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {metadata.tags.map((tag: string) => (
                <Badge key={tag} variant="secondary">
                  {tag}
                </Badge>
              ))}
            </div>
          )}

          {/* 第一个场景预览 */}
          {first_scene && (
            <div>
              <h3 className="text-sm font-medium mb-2">起始场景</h3>
              <div className="p-3 bg-muted rounded-lg">
                <div className="font-medium">{first_scene.name}</div>
                <div className="text-sm text-muted-foreground">
                  {first_scene.location}
                </div>
                <p className="text-sm mt-2 line-clamp-3">
                  {first_scene.description}
                </p>
              </div>
            </div>
          )}

          {/* 描述 */}
          {metadata.description && (
            <div>
              <h3 className="text-sm font-medium mb-2">简介</h3>
              <p className="text-sm line-clamp-3">
                {metadata.description}
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button onClick={onClose}>关闭</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/scene_preview.py` | 创建 | 预览服务 |
| `app/api/preview.py` | 创建 | 预览 API |
| `frontend/src/components/scene/ScenePreview.tsx` | 创建 | 预览组件 |
| `frontend/src/components/scene/ScenePreviewDialog.tsx` | 创建 | 预览对话框 |

---

## 验收标准

- [ ] ZIP 解析正确
- [ ] 场景验证有效
- [ ] 预览信息完整
- [ ] 错误提示友好
- [ ] 预览生成快速

---

## 参考文档

- M4-001: 场景包上传功能
- M0-022: 场景包 JSON Schema

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
