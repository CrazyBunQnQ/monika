# M4-005: 实现场景资源管理器

**任务ID**: M4-005
**标题**: 实现场景资源管理器
**类型**: fullstack (全栈开发)
**预估工时**: 2.5h
**依赖**: M4-001

---

## 任务描述

实现场景包资源管理功能，允许用户上传、管理、预览场景包中使用的图片、音频等资源文件。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-005-01 | 设计资源数据结构 | Asset Model | 20min |
| M4-005-02 | 实现资源上传 API | Upload API | 25min |
| M4-005-03 | 实现资源列表 API | List API | 20min |
| M4-005-04 | 实现资源管理 UI | Asset Manager UI | 35min |
| M4-005-05 | 实现拖拽上传 | Drag & Drop | 25min |
| M4-005-06 | 实现预览功能 | Preview | 20min |
| M4-005-07 | 编写资源测试 | 测试覆盖 | 15min |

---

## 资源数据模型

```python
# app/db/models/asset.py
from sqlalchemy import Column, String, Integer, BigInteger, ForeignKey, DateTime
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class SceneAsset(Base):
    """场景资源"""
    __tablename__ = 'scene_assets'

    id = Column(String, primary_key=True, index=True)
    scene_package_id = Column(String, ForeignKey('scene_packages.id'), nullable=False, index=True)

    # 文件信息
    filename = Column(String, nullable=False)
    original_filename = Column(String, nullable=False)
    file_path = Column(String, nullable=False)
    file_size = Column(BigInteger, nullable=False)
    mime_type = Column(String, nullable=False)

    # 资源类型
    asset_type = Column(String, nullable=False, index=True)  # image, audio, video, document

    # 元数据
    width = Column(Integer)  # 图片宽度
    height = Column(Integer)  # 图片高度
    duration = Column(Integer)  # 音频/视频时长（秒）

    # 缩略图
    thumbnail_path = Column(String)

    # 上传者
    uploaded_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False)

    # 关系
    scene_package = relationship("ScenePackage", back_populates="assets")
    uploader = relationship("User", back_populates="uploaded_assets")

    def __repr__(self):
        return f"<SceneAsset {self.original_filename}>"
```

---

## 资源管理服务

```python
# app/services/asset.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from pathlib import Path
import uuid
import os

from app.db.models.asset import SceneAsset
from app.core.security import generate_id

class AssetService:
    """资源管理服务"""

    def __init__(self, upload_dir: str = "data/assets/scenes"):
        self.upload_dir = Path(upload_dir)
        self.upload_dir.mkdir(parents=True, exist_ok=True)

        # 创建子目录
        (self.upload_dir / "images").mkdir(parents=True, exist_ok=True)
        (self.upload_dir / "audio").mkdir(parents=True, exist_ok=True)
        (self.upload_dir / "videos").mkdir(parents=True, exist_ok=True)
        (self.upload_dir / "documents").mkdir(parents=True, exist_ok=True)
        (self.upload_dir / "thumbnails").mkdir(parents=True, exist_ok=True)

    def get_asset_type_dir(self, asset_type: str) -> Path:
        """获取资源类型目录"""
        type_dirs = {
            'image': 'images',
            'audio': 'audio',
            'video': 'videos',
            'document': 'documents',
        }
        return self.upload_dir / type_dirs.get(asset_type, 'documents')

    async def save_asset(
        self,
        file,
        scene_package_id: str,
        uploaded_by: str,
        db: Session,
    ) -> SceneAsset:
        """保存资源文件"""
        # 读取文件
        contents = await file.read()
        filename = file.filename

        # 确定资源类型
        asset_type = self._get_asset_type(file.content_type)

        # 生成文件名
        file_id = str(uuid.uuid4())
        ext = Path(filename).suffix
        new_filename = f"{file_id}{ext}"

        # 保存文件
        asset_dir = self.get_asset_type_dir(asset_type)
        file_path = asset_dir / new_filename

        with open(file_path, 'wb') as f:
            f.write(contents)

        # 提取元数据
        metadata = await self._extract_metadata(file_path, file.content_type)

        # 生成缩略图（图片）
        thumbnail_path = None
        if asset_type == 'image':
            thumbnail_path = await self._generate_thumbnail(file_path, file_id)

        # 创建数据库记录
        asset = SceneAsset(
            id=generate_id('asset'),
            scene_package_id=scene_package_id,
            filename=new_filename,
            original_filename=filename,
            file_path=str(file_path),
            file_size=len(contents),
            mime_type=file.content_type,
            asset_type=asset_type,
            width=metadata.get('width'),
            height=metadata.get('height'),
            duration=metadata.get('duration'),
            thumbnail_path=thumbnail_path,
            uploaded_by=uploaded_by,
        )

        db.add(asset)
        db.commit()
        db.refresh(asset)

        return asset

    def _get_asset_type(self, mime_type: str) -> str:
        """获取资源类型"""
        if mime_type.startswith('image/'):
            return 'image'
        elif mime_type.startswith('audio/'):
            return 'audio'
        elif mime_type.startswith('video/'):
            return 'video'
        elif mime_type == 'application/pdf':
            return 'document'
        else:
            return 'document'

    async def _extract_metadata(self, file_path: Path, mime_type: str) -> Dict[str, Any]:
        """提取文件元数据"""
        metadata = {}

        if mime_type.startswith('image/'):
            from PIL import Image
            try:
                with Image.open(file_path) as img:
                    metadata['width'] = img.width
                    metadata['height'] = img.height
            except:
                pass

        elif mime_type.startswith('audio/'):
            # 提取音频时长
            try:
                import mutagen
                from mutagen.mp3 import MP3
                audio = MP3(file_path)
                metadata['duration'] = int(audio.info.length)
            except:
                pass

        return metadata

    async def _generate_thumbnail(self, file_path: Path, file_id: str) -> Optional[str]:
        """生成缩略图"""
        from PIL import Image

        try:
            with Image.open(file_path) as img:
                img.thumbnail((200, 200), Image.Resampling.LANCZOS)

                thumbnail_path = self.upload_dir / "thumbnails" / f"{file_id}_thumb.jpg"
                img.convert('RGB').save(thumbnail_path, 'JPEG', quality=80)

                return str(thumbnail_path)
        except:
            return None

    def get_assets(
        self,
        scene_package_id: str,
        asset_type: str = None,
        db: Session = None,
    ) -> List[SceneAsset]:
        """获取资源列表"""
        query = db.query(SceneAsset)\
            .filter(SceneAsset.scene_package_id == scene_package_id)

        if asset_type:
            query = query.filter(SceneAsset.asset_type == asset_type)

        return query\
            .order_by(SceneAsset.created_at.desc())\
            .all()

    def delete_asset(self, asset_id: str, db: Session) -> bool:
        """删除资源"""
        asset = db.query(SceneAsset)\
            .filter(SceneAsset.id == asset_id)\
            .first()

        if not asset:
            return False

        # 删除文件
        file_path = Path(asset.file_path)
        if file_path.exists():
            file_path.unlink()

        # 删除缩略图
        if asset.thumbnail_path:
            thumb_path = Path(asset.thumbnail_path)
            if thumb_path.exists():
                thumb_path.unlink()

        # 删除记录
        db.delete(asset)
        db.commit()

        return True
```

---

## 资源管理 API

```python
# app/api/assets.py
from fastapi import APIRouter, Depends, UploadFile, File, Query
from sqlalchemy.orm import Session
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.asset import AssetService
from app.schemas.asset import AssetResponse

router = APIRouter(prefix="/assets", tags=["assets"])
asset_service = AssetService()

@router.post("/upload/{scene_package_id}")
async def upload_asset(
    scene_package_id: str,
    file: UploadFile = File(...),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """上传资源"""
    # 验证场景包归属
    # TODO: 检查权限

    asset = await asset_service.save_asset(file, scene_package_id, current_user.id, db)

    return AssetResponse.from_orm(asset)

@router.get("/{scene_package_id}")
async def list_assets(
    scene_package_id: str,
    asset_type: Optional[str] = Query(None),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取资源列表"""
    assets = asset_service.get_assets(scene_package_id, asset_type, db)

    return [AssetResponse.from_orm(a) for a in assets]

@router.get("/download/{asset_id}")
async def download_asset(
    asset_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """下载资源"""
    from app.db.models.asset import SceneAsset

    asset = db.query(SceneAsset)\
        .filter(SceneAsset.id == asset_id)\
        .first()

    if not asset:
        raise HTTPException(status_code=404, detail="资源不存在")

    from fastapi.responses import FileResponse
    return FileResponse(
        path=asset.file_path,
        filename=asset.original_filename,
        media_type=asset.mime_type,
    )

@router.delete("/{asset_id}")
async def delete_asset(
    asset_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除资源"""
    success = asset_service.delete_asset(asset_id, db)

    if not success:
        raise HTTPException(status_code=404, detail="资源不存在")

    return {"message": "资源已删除"}
```

---

## 前端资源管理组件

```tsx
// frontend/src/components/scene/AssetManager.tsx
import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Upload, Image, Music, Video, File, Trash2, Download, X } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { useDropzone } from 'react-dropzone'

interface Asset {
  id: string
  filename: string
  original_filename: string
  file_size: number
  asset_type: string
  mime_type: string
  thumbnail_path?: string
  created_at: string
}

interface AssetManagerProps {
  scenePackageId: string
}

export function AssetManager({ scenePackageId }: AssetManagerProps) {
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [activeTab, setActiveTab] = useState('all')

  const { toast } = useToast()

  useEffect(() => {
    loadAssets()
  }, [scenePackageId])

  const loadAssets = async () => {
    setLoading(true)
    try {
      const response = await fetch(`/api/assets/${scenePackageId}`)
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setAssets(data)
    } catch (error) {
      console.error('Failed to load assets:', error)
    } finally {
      setLoading(false)
    }
  }

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    setUploading(true)

    for (const file of acceptedFiles) {
      try {
        const formData = new FormData()
        formData.append('file', file)

        const response = await fetch(`/api/assets/upload/${scenePackageId}`, {
          method: 'POST',
          body: formData,
        })

        if (!response.ok) throw new Error('上传失败')

        const data = await response.json()
        setAssets(prev => [data, ...prev])
      } catch (error) {
        toast({
          title: '上传失败',
          description: `${file.name} 上传失败`,
          variant: 'destructive',
        })
      }
    }

    setUploading(false)
    toast({
      title: '上传完成',
      description: `成功上传 ${acceptedFiles.length} 个文件`,
    })
  }, [scenePackageId])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    noClick: true,
  })

  const handleDelete = async (assetId: string) => {
    try {
      await fetch(`/api/assets/${assetId}`, {
        method: 'DELETE',
      })

      setAssets(assets.filter(a => a.id !== assetId))
    } catch (error) {
      toast({
        title: '删除失败',
        variant: 'destructive',
      })
    }
  }

  const handleDownload = (asset: Asset) => {
    window.open(`/api/assets/download/${asset.id}`)
  }

  const getAssetIcon = (assetType: string) => {
    switch (assetType) {
      case 'image':
        return <Image className="h-5 w-5" />
      case 'audio':
        return <Music className="h-5 w-5" />
      case 'video':
        return <Video className="h-5 w-5" />
      default:
        return <File className="h-5 w-5" />
    }
  }

  const filteredAssets = activeTab === 'all'
    ? assets
    : assets.filter(a => a.asset_type === activeTab)

  return (
    <div {...getRootProps()} className="space-y-4">
      <input {...getInputProps()} />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">场景资源管理</CardTitle>
        </CardHeader>

        <CardContent>
          {/* 拖拽上传区域 */}
          <div
            className={`border-2 border-dashed rounded-lg p-8 text-center mb-4 transition-colors ${
              isDragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
            }`}
          >
            <Upload className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-sm text-muted-foreground mb-2">
              拖拽文件到此处，或
            </p>
            <Button size="sm" disabled={uploading}>
              {uploading ? '上传中...' : '选择文件'}
            </Button>
          </div>

          {/* 资源列表 */}
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="mb-4">
              <TabsTrigger value="all">
                全部 ({assets.length})
              </TabsTrigger>
              <TabsTrigger value="image">
                图片 ({assets.filter(a => a.asset_type === 'image').length})
              </TabsTrigger>
              <TabsTrigger value="audio">
                音频 ({assets.filter(a => a.asset_type === 'audio').length})
              </TabsTrigger>
              <TabsTrigger value="video">
                视频 ({assets.filter(a => a.asset_type === 'video').length})
              </TabsTrigger>
            </TabsList>

            <div className="grid grid-cols-4 gap-4">
              {filteredAssets.map((asset) => (
                <div
                  key={asset.id}
                  className="border rounded-lg overflow-hidden group"
                >
                  {/* 预览 */}
                  <div className="aspect-square bg-muted flex items-center justify-center">
                    {asset.asset_type === 'image' && asset.thumbnail_path ? (
                      <img
                        src={asset.thumbnail_path}
                        alt={asset.original_filename}
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="text-muted-foreground">
                        {getAssetIcon(asset.asset_type)}
                      </div>
                    )}
                  </div>

                  {/* 信息 */}
                  <div className="p-2">
                    <p className="text-sm font-medium truncate">
                      {asset.original_filename}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {(asset.file_size / 1024).toFixed(1)} KB
                    </p>
                  </div>

                  {/* 操作 */}
                  <div className="flex border-t">
                    <Button
                      size="sm"
                      variant="ghost"
                      className="flex-1 rounded-none"
                      onClick={() => handleDownload(asset)}
                    >
                      <Download className="h-3 w-3" />
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="flex-1 rounded-none text-destructive"
                      onClick={() => handleDelete(asset.id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>

            {filteredAssets.length === 0 && (
              <div className="text-center text-muted-foreground py-8">
                暂无资源
              </div>
            )}
          </Tabs>
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
| `app/db/models/asset.py` | 创建 | 资源数据模型 |
| `app/services/asset.py` | 创建 | 资源管理服务 |
| `app/api/assets.py` | 创建 | 资源 API |
| `frontend/src/components/scene/AssetManager.tsx` | 创建 | 资源管理组件 |

---

## 验收标准

- [ ] 文件上传成功
- [ ] 拖拽功能流畅
- [ ] 预览显示正确
- [ ] 分类过滤有效
- [ ] 文件下载正常
- [ ] 删除操作安全

---

## 参考文档

- M4-001: 场景包上传功能
- react-dropzone 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
