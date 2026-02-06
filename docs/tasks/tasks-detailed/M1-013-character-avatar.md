# M1-013: 实现角色头像上传功能

**任务ID**: M1-013
**标题**: 实现角色头像上传功能
**类型**: fullstack (全栈开发)
**预估工时**: 1.5h
**依赖**: M1-004

---

## 任务描述

实现角色头像上传功能，支持裁剪、缩放、格式转换等图像处理操作。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-013-01 | 设计头像存储方案 | Storage Design | 15min |
| M1-013-02 | 实现图片上传 API | Upload API | 20min |
| M1-013-03 | 实现图片处理服务 | Image Processing | 25min |
| M1-013-04 | 实现前端上传组件 | Upload Component | 25min |
| M1-013-05 | 实现头像裁剪器 | Cropper | 20min |
| M1-013-06 | 编写图片测试 | 测试覆盖 | 10min |

---

## 图片存储服务

```python
# app/services/image.py
from pathlib import Path
from typing import Optional
import uuid
from PIL import Image, ImageOps
from fastapi import UploadFile, HTTPException

class ImageService:
    """图片处理服务"""

    def __init__(
        self,
        upload_dir: str = "data/avatars",
        max_size: int = 2 * 1024 * 1024,  # 2MB
        avatar_sizes: list = None,
    ):
        self.upload_dir = Path(upload_dir)
        self.max_size = max_size
        self.avatar_sizes = avatar_sizes or [
            (32, 32),   # small
            (64, 64),   # medium
            (128, 128), # large
            (256, 256), # xlarge
        ]

        self.upload_dir.mkdir(parents=True, exist_ok=True)
        for size in self.avatar_sizes:
            (self.upload_dir / f"{size[0]}x{size[1]}").mkdir(parents=True, exist_ok=True)

    async def save_avatar(
        self,
        file: UploadFile,
        character_id: str,
    ) -> dict:
        """保存并处理角色头像"""
        # 验证文件类型
        if not file.content_type.startswith('image/'):
            raise HTTPException(status_code=400, detail="只支持图片文件")

        # 读取图片
        contents = await file.read()

        if len(contents) > self.max_size:
            raise HTTPException(status_code=400, detail="图片过大")

        try:
            img = Image.open(io.BytesIO(contents))

            # 转换为 RGB
            if img.mode != 'RGB':
                img = img.convert('RGB')

            # 生成文件名
            file_id = str(uuid.uuid4())
            base_filename = f"{character_id}_{file_id}"

            # 生成不同尺寸
            sizes = {}
            for size in self.avatar_sizes:
                resized = ImageOps.fit(img, size, Image.Resampling.LANCZOS)
                size_dir = self.upload_dir / f"{size[0]}x{size[1]}"
                resized.save(size_dir / f"{base_filename}.jpg", "JPEG", quality=85)
                sizes[f"{size[0]}x{size[1]}"] = f"{size[0]}x{size[1]}/{base_filename}.jpg"

            return {
                'file_id': file_id,
                'base_filename': base_filename,
                'sizes': sizes,
                'original_format': img.format,
            }

        except Exception as e:
            raise HTTPException(status_code=500, detail=f"图片处理失败: {str(e)}")

    def delete_avatar(self, character_id: str, file_id: str):
        """删除头像"""
        for size in self.avatar_sizes:
            size_dir = self.upload_dir / f"{size[0]}x{size[1]}"
            for ext in ['jpg', 'png']:
                file_path = size_dir / f"{character_id}_{file_id}.{ext}"
                if file_path.exists():
                    file_path.unlink()
```

---

## 头像上传 API

```python
# app/api/avatars.py
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException
from sqlalchemy.orm import Session

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.image import ImageService
from app.db.models.character import Character

router = APIRouter(prefix="/avatars", tags=["avatars"])
image_service = ImageService()

@router.post("/character/{character_id}")
async def upload_character_avatar(
    character_id: str,
    file: UploadFile = File(...),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """上传角色头像"""
    # 验证角色归属
    character = db.query(Character)\
        .filter(Character.id == character_id)\
        .first()

    if not character:
        raise HTTPException(status_code=404, detail="角色不存在")

    if character.user_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权修改此角色")

    # 保存头像
    avatar_data = await image_service.save_avatar(file, character_id)

    # 更新角色记录
    character.avatar_file = avatar_data['file_id']
    db.commit()

    return {
        'character_id': character_id,
        'avatar_file': avatar_data['file_id'],
        'sizes': avatar_data['sizes'],
    }

@router.get("/character/{character_id}/{size}")
async def get_character_avatar(
    character_id: str,
    size: str,
    db: Session = Depends(get_db),
):
    """获取角色头像"""
    character = db.query(Character)\
        .filter(Character.id == character_id)\
        .first()

    if not character or not character.avatar_file:
        raise HTTPException(status_code=404, detail="头像不存在")

    # 返回图片文件
    from fastapi.responses import FileResponse
    avatar_path = image_service.upload_dir / size / f"{character_id}_{character.avatar_file}.jpg"

    if not avatar_path.exists():
        raise HTTPException(status_code=404, detail="头像不存在")

    return FileResponse(avatar_path, media_type="image/jpeg")
```

---

## 前端头像上传组件

```tsx
// frontend/src/components/game/AvatarUpload.tsx
import { useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Upload, Camera, X } from 'lucide-react'
import { AvatarCropper } from './AvatarCropper'

interface AvatarUploadProps {
  characterId: string
  currentAvatar?: string
  onAvatarChange: (avatarUrl: string) => void
}

export function AvatarUpload({
  characterId,
  currentAvatar,
  onAvatarChange,
}: AvatarUploadProps) {
  const [showCropper, setShowCropper] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    // 验证文件类型
    if (!file.type.startsWith('image/')) {
      alert('请选择图片文件')
      return
    }

    // 验证文件大小 (2MB)
    if (file.size > 2 * 1024 * 1024) {
      alert('图片大小不能超过 2MB')
      return
    }

    setSelectedFile(file)

    // 显示预览
    const reader = new FileReader()
    reader.onload = (e) => {
      setPreview(e.target?.result as string)
      setShowCropper(true)
    }
    reader.readAsDataURL(file)
  }

  const handleCropComplete = async (croppedBlob: Blob) => {
    setShowCropper(false)
    setUploading(true)

    try {
      const formData = new FormData()
      formData.append('file', croppedBlob, 'avatar.jpg')

      const response = await fetch(`/api/avatars/character/${characterId}`, {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) throw new Error('上传失败')

      const data = await response.json()
      onAvatarChange(data.sizes['128x128'])
    } catch (error) {
      console.error('Upload failed:', error)
      alert('上传失败，请重试')
    } finally {
      setUploading(false)
      setSelectedFile(null)
      setPreview(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  return (
    <>
      <Card className="p-4">
        <div className="space-y-4">
          {/* 当前头像 */}
          <div className="flex items-center space-x-4">
            <div className="relative">
              <img
                src={currentAvatar || '/default-avatar.png'}
                alt="Avatar"
                className="w-20 h-20 rounded-full object-cover border"
              />
              <button
                onClick={() => fileInputRef.current?.click()}
                className="absolute bottom-0 right-0 p-1 bg-primary rounded-full text-primary-foreground hover:bg-primary/90"
                disabled={uploading}
              >
                {uploading ? (
                  <div className="animate-spin h-4 w-4 border-2 border-current border-t-transparent rounded-full" />
                ) : (
                  <Camera className="h-3 w-3" />
                )}
              </button>
            </div>

            <div>
              <Button
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
              >
                <Upload className="h-4 w-4 mr-1" />
                更换头像
              </Button>
              <p className="text-xs text-muted-foreground mt-1">
                支持 JPG、PNG 格式，最大 2MB
              </p>
            </div>

            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              accept="image/jpeg,image/png"
              onChange={handleFileSelect}
            />
          </div>
        </div>
      </Card>

      {/* 裁剪对话框 */}
      {showCropper && preview && (
        <AvatarCropper
          image={preview}
          onCancel={() => {
            setShowCropper(false)
            setSelectedFile(null)
            setPreview(null)
          }}
          onCrop={handleCropComplete}
        />
      )}
    </>
  )
}
```

---

## 头像裁剪组件

```tsx
// frontend/src/components/game/AvatarCropper.tsx
import { useState, useRef, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { X, ZoomIn, ZoomOut, RotateCw } from 'lucide-react'

interface AvatarCropperProps {
  image: string
  onCancel: () => void
  onCrop: (blob: Blob) => void
}

export function AvatarCropper({ image, onCancel, onCrop }: AvatarCropperProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [scale, setScale] = useState(1)
  const [position, setPosition] = useState({ x: 0, y: 0 })
  const [rotation, setRotation] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 })

  const canvasSize = 300
  const cropSize = 200

  useEffect(() => {
    drawImage()
  }, [image, scale, position, rotation])

  const drawImage = () => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // 清空画布
    ctx.clearRect(0, 0, canvasSize, canvasSize)

    // 绘制图片
    const img = new Image()
    img.onload = () => {
      ctx.save()
      ctx.translate(canvasSize / 2, canvasSize / 2)
      ctx.rotate((rotation * Math.PI) / 180)
      ctx.scale(scale, scale)
      ctx.translate(-canvasSize / 2 + position.x, -canvasSize / 2 + position.y)
      ctx.drawImage(img, 0, 0, canvasSize, canvasSize)
      ctx.restore()

      // 绘制裁剪框
      ctx.strokeStyle = '#fff'
      ctx.lineWidth = 2
      ctx.setLineDash([5, 5])
      const cropX = (canvasSize - cropSize) / 2
      const cropY = (canvasSize - cropSize) / 2
      ctx.strokeRect(cropX, cropY, cropSize, cropSize)
    }
    img.src = image
  }

  const handleCrop = () => {
    const canvas = canvasRef.current
    if (!canvas) return

    // 创建裁剪后的画布
    const croppedCanvas = document.createElement('canvas')
    croppedCanvas.width = cropSize
    croppedCanvas.height = cropSize
    const ctx = croppedCanvas.getContext('2d')
    if (!ctx) return

    // 裁剪图片
    const cropX = (canvasSize - cropSize) / 2
    const cropY = (canvasSize - cropSize) / 2
    ctx.drawImage(canvas, cropX, cropY, cropSize, cropSize, 0, 0, cropSize, cropSize)

    // 转换为 Blob
    croppedCanvas.toBlob((blob) => {
      if (blob) onCrop(blob)
    }, 'image/jpeg', 0.85)
  }

  return (
    <Dialog open={true} onOpenChange={onCancel}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>裁剪头像</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* 画布 */}
          <div className="flex justify-center">
            <canvas
              ref={canvasRef}
              width={canvasSize}
              height={canvasSize}
              className="border rounded cursor-move"
              onMouseDown={(e) => {
                setDragging(true)
                setDragStart({ x: e.clientX - position.x, y: e.clientY - position.y })
              }}
              onMouseMove={(e) => {
                if (dragging) {
                  setPosition({
                    x: e.clientX - dragStart.x,
                    y: e.clientY - dragStart.y,
                  })
                }
              }}
              onMouseUp={() => setDragging(false)}
              onMouseLeave={() => setDragging(false)}
            />
          </div>

          {/* 控制按钮 */}
          <div className="flex justify-center space-x-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setScale(Math.max(0.5, scale - 0.1))}
            >
              <ZoomOut className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setScale(Math.min(3, scale + 0.1))}
            >
              <ZoomIn className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setRotation((rotation + 90) % 360)}
            >
              <RotateCw className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            取消
          </Button>
          <Button onClick={handleCrop}>
            确认
          </Button>
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
| `app/services/image.py` | 创建 | 图片处理服务 |
| `app/api/avatars.py` | 创建 | 头像 API |
| `frontend/src/components/game/AvatarUpload.tsx` | 创建 | 头像上传组件 |
| `frontend/src/components/game/AvatarCropper.tsx` | 创建 | 头像裁剪组件 |

---

## 验收标准

- [ ] 图片上传成功
- [ ] 多尺寸生成正确
- [ ] 裁剪功能流畅
- [ ] 格式转换有效
- [ ] 文件大小限制生效
- [ ] 错误处理友好

---

## 参考文档

- M1-004: 角色卡 CRUD API
- PIL/Pillow 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
