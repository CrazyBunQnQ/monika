# M2-005: 实现文件共享功能

**任务ID**: M2-005
**标题**: 实现文件共享功能
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M2-001

---

## 任务描述

实现房间内的文件共享功能，允许 KP 和玩家上传、下载、预览图片、PDF 等文件。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-005-01 | 设计文件存储方案 | Storage Design | 20min |
| M2-005-02 | 实现文件上传 API | Upload API | 25min |
| M2-005-03 | 实现文件下载 API | Download API | 20min |
| M2-005-04 | 实现文件列表 API | List API | 15min |
| M2-005-05 | 实现文件删除 API | Delete API | 15min |
| M2-005-06 | 实现 WebSocket 同步 | WS Sync | 15min |
| M2-005-07 | 编写文件测试 | 测试覆盖 | 15min |

---

## 文件数据模型

```python
# app/db/models/file.py
from sqlalchemy import Column, String, Integer, DateTime, ForeignKey, BigInteger
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class SharedFile(Base):
    """共享文件"""
    __tablename__ = 'shared_files'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 文件信息
    filename = Column(String, nullable=False)
    original_filename = Column(String, nullable=False)
    file_path = Column(String, nullable=False)
    file_size = Column(BigInteger, nullable=False)
    mime_type = Column(String, nullable=False)
    file_type = Column(String, nullable=False)  # image, pdf, audio, video, other

    # 上传者
    uploaded_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 缩略图（用于图片）
    thumbnail_path = Column(String)

    # 元数据
    width = Column(Integer)  # 图片宽度
    height = Column(Integer)  # 图片高度

    created_at = Column(DateTime, default=func.now(), nullable=False)

    # 关系
    uploader = relationship("User", back_populates="uploaded_files")
    room = relationship("Room", back_populates="shared_files")

    def __repr__(self):
        return f"<SharedFile {self.original_filename}>"
```

---

## 文件存储服务

```python
# app/services/file_storage.py
import os
import shutil
import uuid
from pathlib import Path
from typing import Optional
from fastapi import UploadFile, HTTPException
from PIL import Image

class FileStorageService:
    """文件存储服务"""

    def __init__(
        self,
        upload_dir: str = "data/uploads",
        max_file_size: int = 50 * 1024 * 1024,  # 50MB
        allowed_types: list = None,
    ):
        self.upload_dir = Path(upload_dir)
        self.max_file_size = max_file_size
        self.allowed_types = allowed_types or [
            'image/jpeg', 'image/png', 'image/gif', 'image/webp',
            'application/pdf',
            'audio/mpeg', 'audio/wav',
            'video/mp4',
        ]

        # 创建目录
        self.upload_dir.mkdir(parents=True, exist_ok=True)
        (self.upload_dir / "thumbnails").mkdir(parents=True, exist_ok=True)

    async def save_file(
        self,
        file: UploadFile,
        room_id: str,
        user_id: str,
    ) -> dict:
        """保存上传的文件"""
        # 验证文件类型
        if file.content_type not in self.allowed_types:
            raise HTTPException(
                status_code=400,
                detail=f"不支持的文件类型: {file.content_type}"
            )

        # 生成唯一文件名
        file_id = str(uuid.uuid4())
        ext = Path(file.filename).suffix
        filename = f"{file_id}{ext}"
        file_path = self.upload_dir / filename

        # 保存文件
        try:
            with open(file_path, "wb") as buffer:
                shutil.copyfileobj(file.file, buffer)
        except Exception as e:
            raise HTTPException(
                status_code=500,
                detail=f"文件保存失败: {str(e)}"
            )

        # 获取文件大小
        file_size = file_path.stat().st_size
        if file_size > self.max_file_size:
            file_path.unlink()
            raise HTTPException(
                status_code=400,
                detail=f"文件过大 (最大 {self.max_file_size / 1024 / 1024}MB)"
            )

        # 生成缩略图（图片）
        thumbnail_path = None
        width, height = None, None

        if file.content_type.startswith('image/'):
            thumbnail_path, width, height = await self._generate_thumbnail(
                file_path, file_id
            )

        return {
            'id': file_id,
            'filename': filename,
            'file_path': str(file_path),
            'file_size': file_size,
            'mime_type': file.content_type,
            'file_type': self._get_file_type(file.content_type),
            'thumbnail_path': thumbnail_path,
            'width': width,
            'height': height,
        }

    async def _generate_thumbnail(
        self,
        file_path: Path,
        file_id: str,
        max_size: tuple = (300, 300),
    ) -> tuple:
        """生成缩略图"""
        try:
            with Image.open(file_path) as img:
                img.thumbnail(max_size, Image.Resampling.LANCZOS)

                thumbnail_path = self.upload_dir / "thumbnails" / f"{file_id}_thumb.jpg"
                img.convert('RGB').save(thumbnail_path, 'JPEG', quality=85)

                return str(thumbnail_path), img.width, img.height
        except Exception as e:
            print(f"生成缩略图失败: {e}")
            return None, None, None

    def _get_file_type(self, mime_type: str) -> str:
        """获取文件类型分类"""
        if mime_type.startswith('image/'):
            return 'image'
        elif mime_type == 'application/pdf':
            return 'pdf'
        elif mime_type.startswith('audio/'):
            return 'audio'
        elif mime_type.startswith('video/'):
            return 'video'
        else:
            return 'other'

    def delete_file(self, file_path: str, thumbnail_path: Optional[str] = None):
        """删除文件"""
        path = Path(file_path)
        if path.exists():
            path.unlink()

        if thumbnail_path:
            thumb_path = Path(thumbnail_path)
            if thumb_path.exists():
                thumb_path.unlink()
```

---

## 文件管理 API

```python
# app/api/files.py
from fastapi import APIRouter, Depends, UploadFile, File, HTTPException
from sqlalchemy.orm import Session
from typing import List

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.db.models.file import SharedFile
from app.services.file_storage import FileStorageService
from app.core.security import generate_id

router = APIRouter(prefix="/files", tags=["files"])
storage = FileStorageService()

@router.post("/upload/{room_id}")
async def upload_file(
    room_id: str,
    file: UploadFile = File(...),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """上传文件到房间"""
    # 验证用户在房间中
    # TODO: 检查房间成员

    # 保存文件
    file_info = await storage.save_file(file, room_id, current_user.id)

    # 创建数据库记录
    db_file = SharedFile(
        id=generate_id('file'),
        room_id=room_id,
        filename=file_info['filename'],
        original_filename=file.filename,
        file_path=file_info['file_path'],
        file_size=file_info['file_size'],
        mime_type=file_info['mime_type'],
        file_type=file_info['file_type'],
        uploaded_by=current_user.id,
        thumbnail_path=file_info.get('thumbnail_path'),
        width=file_info.get('width'),
        height=file_info.get('height'),
    )

    db.add(db_file)
    db.commit()
    db.refresh(db_file)

    # 通知房间成员
    # TODO: WebSocket 事件

    return {
        'id': db_file.id,
        'filename': db_file.original_filename,
        'file_type': db_file.file_type,
        'size': db_file.file_size,
        'uploaded_by': current_user.username,
        'created_at': db_file.created_at,
    }

@router.get("/room/{room_id}")
async def list_room_files(
    room_id: str,
    file_type: str = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间文件列表"""
    query = db.query(SharedFile).filter(SharedFile.room_id == room_id)

    if file_type:
        query = query.filter(SharedFile.file_type == file_type)

    files = query.order_by(SharedFile.created_at.desc()).all()

    return {
        'files': [
            {
                'id': f.id,
                'filename': f.original_filename,
                'file_type': f.file_type,
                'size': f.file_size,
                'thumbnail': f.thumbnail_path,
                'uploaded_by': f.uploader.username,
                'created_at': f.created_at,
            }
            for f in files
        ]
    }

@router.get("/download/{file_id}")
async def download_file(
    file_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """下载文件"""
    file = db.query(SharedFile).filter(SharedFile.id == file_id).first()

    if not file:
        raise HTTPException(status_code=404, detail="文件不存在")

    from fastapi.responses import FileResponse
    return FileResponse(
        path=file.file_path,
        filename=file.original_filename,
        media_type=file.mime_type,
    )

@router.delete("/{file_id}")
async def delete_file(
    file_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除文件"""
    file = db.query(SharedFile).filter(SharedFile.id == file_id).first()

    if not file:
        raise HTTPException(status_code=404, detail="文件不存在")

    # 只有上传者和 KP 可以删除
    # TODO: 权限检查

    # 删除文件
    storage.delete_file(file.file_path, file.thumbnail_path)

    # 删除记录
    db.delete(file)
    db.commit()

    return {'message': '文件已删除'}
```

---

## WebSocket 文件事件

```python
# app/api/websocket/handlers/files.py
from typing import Dict, Any

async def handle_file_upload(
    ws_manager,
    data: Dict[str, Any],
):
    """处理文件上传事件"""
    room_id = data.get('room_id')
    file_info = data.get('file')

    # 广播文件上传通知
    await ws_manager.broadcast_to_room(
        room_id,
        {
            'type': 'file_uploaded',
            'data': file_info,
        }
    )

async def handle_file_delete(
    ws_manager,
    data: Dict[str, Any],
):
    """处理文件删除事件"""
    room_id = data.get('room_id')
    file_id = data.get('file_id')

    # 广播文件删除通知
    await ws_manager.broadcast_to_room(
        room_id,
        {
            'type': 'file_deleted',
            'data': {'file_id': file_id},
        }
    )
```

---

## 前端文件上传组件

```tsx
// frontend/src/components/game/FileUpload.tsx
import { useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Upload, X, FileImage, FileText, Download } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

interface SharedFile {
  id: string
  filename: string
  file_type: string
  size: number
  thumbnail?: string
  uploaded_by: string
  created_at: string
}

interface FileUploadProps {
  roomId: string
}

export function FileUpload({ roomId }: FileUploadProps) {
  const [files, setFiles] = useState<SharedFile[]>([])
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const { toast } = useToast()

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setUploading(true)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await fetch(`/api/files/upload/${roomId}`, {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) throw new Error('上传失败')

      const data = await response.json()
      setFiles(prev => [data, ...prev])

      toast({
        title: '上传成功',
        description: `${file.name} 已上传`,
      })
    } catch (error) {
      toast({
        title: '上传失败',
        description: error.message,
        variant: 'destructive',
      })
    } finally {
      setUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const handleDelete = async (fileId: string) => {
    try {
      const response = await fetch(`/api/files/${fileId}`, {
        method: 'DELETE',
      })

      if (!response.ok) throw new Error('删除失败')

      setFiles(prev => prev.filter(f => f.id !== fileId))
    } catch (error) {
      toast({
        title: '删除失败',
        description: error.message,
        variant: 'destructive',
      })
    }
  }

  const getFileIcon = (fileType: string) => {
    switch (fileType) {
      case 'image':
        return <FileImage className="h-5 w-5" />
      case 'pdf':
        return <FileText className="h-5 w-5" />
      default:
        return <FileText className="h-5 w-5" />
    }
  }

  return (
    <Card className="p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium">共享文件</h3>
        <Button
          size="sm"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
        >
          <Upload className="h-4 w-4 mr-1" />
          上传
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleUpload}
          accept="image/*,.pdf,audio/*,video/*"
        />
      </div>

      <div className="space-y-2">
        {files.map((file) => (
          <div
            key={file.id}
            className="flex items-center space-x-3 p-2 border rounded"
          >
            <div className="text-muted-foreground">
              {getFileIcon(file.file_type)}
            </div>

            {file.thumbnail && (
              <img
                src={file.thumbnail}
                alt={file.filename}
                className="w-12 h-12 object-cover rounded"
              />
            )}

            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">
                {file.filename}
              </p>
              <p className="text-xs text-muted-foreground">
                {(file.size / 1024).toFixed(1)} KB • {file.uploaded_by}
              </p>
            </div>

            <Button
              size="sm"
              variant="ghost"
              onClick={() => window.open(`/api/files/download/${file.id}`)}
            >
              <Download className="h-4 w-4" />
            </Button>

            <Button
              size="sm"
              variant="ghost"
              onClick={() => handleDelete(file.id)}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        ))}
      </div>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/file.py` | 创建 | 文件数据模型 |
| `app/services/file_storage.py` | 创建 | 文件存储服务 |
| `app/api/files.py` | 创建 | 文件 API |
| `app/api/websocket/handlers/files.py` | 创建 | WebSocket 处理 |
| `frontend/src/components/game/FileUpload.tsx` | 创建 | 文件上传组件 |

---

## 验收标准

- [ ] 文件上传成功
- [ ] 文件下载正常
- [ ] 缩略图生成正确
- [ ] 文件类型验证有效
- [ ] WebSocket 同步及时
- [ ] 权限控制正确

---

## 参考文档

- M2-001: 房间管理系统
- FastAPI UploadFile 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
