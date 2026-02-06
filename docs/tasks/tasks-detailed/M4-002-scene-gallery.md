# M4-002: 实现场景库管理界面

**任务ID**: M4-002
**标题**: 实现场景库管理界面
**类型**: frontend (前端开发)
**预估工时**: 2.5h
**依赖**: M4-001

---

## 任务描述

实现场景库管理前端界面，支持场景浏览、上传、编辑、删除等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-002-01 | 设计场景库布局 | UI 设计 | 25min |
| M4-002-02 | 实现场景列表 | Scene List | 30min |
| M4-002-03 | 实现场景卡片 | Scene Card | 30min |
| M4-002-04 | 实现上传界面 | Upload UI | 35min |
| M4-002-05 | 实现场景编辑 | Edit | 30min |
| M4-002-06 | 实现场景删除 | Delete | 20min |
| M4-002-07 | 编写界面测试 | 测试覆盖 | 10min |

---

## 场景库主组件

```tsx
// frontend/src/pages/SceneGalleryPage.tsx
import { useState, useEffect } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Search, Upload, Edit, Trash2, Eye } from 'lucide-react'
import { SceneCard } from '@/components/scene/SceneCard'
import { UploadSceneDialog } from '@/components/scene/UploadSceneDialog'
import { SceneDetailDialog } from '@/components/scene/SceneDetailDialog'

interface Scene {
  id: string
  name: string
  description: string
  thumbnail?: string
  author: string
  version: string
  tags: string[]
  created_at: string
}

export default function SceneGalleryPage() {
  const { user } = useAuth()
  const [scenes, setScenes] = useState<Scene[]>([])
  const [filtered, setFiltered] = useState<Scene[]>([])
  const [search, setSearch] = useState('')
  const [activeTab, setActiveTab] = useState('all')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [detailScene, setDetailScene] = useState<Scene | null>(null)

  const isKp = user?.role === 'kp'

  useEffect(() => {
    loadScenes()
  }, [])

  useEffect(() => {
    filterScenes()
  }, [scenes, search, activeTab])

  const loadScenes = async () => {
    try {
      const response = await fetch('/api/scenes')
      if (!response.ok) throw new Error('Failed to load scenes')

      const data = await response.json()
      setScenes(data.scenes || [])
    } catch (error) {
      console.error('Failed to load scenes:', error)
    }
  }

  const filterScenes = () => {
    let filtered = scenes

    // 搜索过滤
    if (search) {
      filtered = filtered.filter(scene =>
        scene.name.toLowerCase().includes(search.toLowerCase()) ||
        scene.description.toLowerCase().includes(search.toLowerCase()) ||
        scene.tags.some(tag => tag.toLowerCase().includes(search.toLowerCase()))
      )
    }

    // 标签过滤
    if (activeTab !== 'all') {
      filtered = filtered.filter(scene =>
        scene.tags.includes(activeTab)
      )
    }

    setFiltered(filtered)
  }

  const handleDelete = async (sceneId: string) => {
    if (!confirm('确定要删除这个场景吗？')) return

    try {
      const response = await fetch(`/api/scenes/${sceneId}`, {
        method: 'DELETE',
      })

      if (!response.ok) throw new Error('Failed to delete scene')

      setScenes(scenes.filter(s => s.id !== sceneId))
    } catch (error) {
      console.error('Failed to delete scene:', error)
    }
  }

  const tags = Array.from(new Set(scenes.flatMap(s => s.tags)))

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* 头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">场景库</h1>
          <p className="text-muted-foreground">
            浏览和管理 CoC 战役场景
          </p>
        </div>

        {isKp && (
          <Button onClick={() => setUploadOpen(true)}>
            <Upload className="h-4 w-4 mr-2" />
            上传场景
          </Button>
        )}
      </div>

      {/* 搜索和过滤 */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center space-x-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索场景..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>

            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList>
                <TabsTrigger value="all">全部</TabsTrigger>
                {tags.slice(0, 5).map(tag => (
                  <TabsTrigger key={tag} value={tag}>
                    {tag}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>
        </CardContent>
      </Card>

      {/* 场景列表 */}
      {filtered.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center">
            <div className="text-muted-foreground">
              {search ? '没有找到匹配的场景' : '场景库为空'}
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filtered.map((scene) => (
            <SceneCard
              key={scene.id}
              scene={scene}
              onView={() => setDetailScene(scene)}
              onEdit={isKp ? () => {/* 编辑功能 */} : undefined}
              onDelete={isKp ? () => handleDelete(scene.id) : undefined}
            />
          ))}
        </div>
      )}

      {/* 上传对话框 */}
      {uploadOpen && (
        <UploadSceneDialog
          open={uploadOpen}
          onClose={() => setUploadOpen(false)}
          onUploaded={loadScenes}
        />
      )}

      {/* 详情对话框 */}
      {detailScene && (
        <SceneDetailDialog
          scene={detailScene}
          open={!!detailScene}
          onClose={() => setDetailScene(null)}
        />
      )}
    </div>
  )
}
```

---

## 场景卡片组件

```tsx
// frontend/src/components/scene/SceneCard.tsx
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Eye, Edit, Trash2 } from 'lucide-react'
import { Image } from '@/components/ui/image'
import type { Scene } from '@/types/scene'

interface SceneCardProps {
  scene: Scene
  onView: () => void
  onEdit?: () => void
  onDelete?: () => void
}

export function SceneCard({ scene, onView, onEdit, onDelete }: SceneCardProps) {
  return (
    <Card className="overflow-hidden hover:shadow-lg transition-shadow">
      {/* 缩略图 */}
      <div className="aspect-video relative bg-muted">
        {scene.thumbnail ? (
          <Image
            src={scene.thumbnail}
            alt={scene.name}
            fill
            className="object-cover"
          />
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            暂无封面
          </div>
        )}
      </div>

      {/* 内容 */}
      <CardHeader className="pb-3">
        <CardTitle className="text-lg line-clamp-1">{scene.name}</CardTitle>
        <p className="text-sm text-muted-foreground line-clamp-2">
          {scene.description}
        </p>
      </CardHeader>

      <CardContent className="pb-3">
        <div className="flex flex-wrap gap-1">
          {scene.tags.slice(0, 3).map(tag => (
            <Badge key={tag} variant="secondary" className="text-xs">
              {tag}
            </Badge>
          ))}
          {scene.tags.length > 3 && (
            <Badge variant="outline" className="text-xs">
              +{scene.tags.length - 3}
            </Badge>
          )}
        </div>

        <div className="mt-2 text-xs text-muted-foreground">
          {scene.author} • v{scene.version}
        </div>
      </CardContent>

      {/* 操作按钮 */}
      <CardFooter className="flex justify-between">
        <Button
          size="sm"
          variant="outline"
          onClick={onView}
        >
          <Eye className="h-4 w-4 mr-1" />
          查看
        </Button>

        {(onEdit || onDelete) && (
          <div className="flex space-x-2">
            {onEdit && (
              <Button
                size="sm"
                variant="outline"
                onClick={onEdit}
              >
                <Edit className="h-4 w-4" />
              </Button>
            )}
            {onDelete && (
              <Button
                size="sm"
                variant="outline"
                onClick={onDelete}
              >
                <Trash2 className="h-4 w-4 text-red-500" />
              </Button>
            )}
          </div>
        )}
      </CardFooter>
    </Card>
  )
}
```

---

## 上传场景对话框

```tsx
// frontend/src/components/scene/UploadSceneDialog.tsx
import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { Upload, FileText, X } from 'lucide-react'

interface UploadSceneDialogProps {
  open: boolean
  onClose: () => void
  onUploaded: () => void
}

export function UploadSceneDialog({
  open,
  onClose,
  onUploaded,
}: UploadSceneDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)

  const handleUpload = async () => {
    if (!file) return

    setUploading(true)
    setProgress(0)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const xhr = new XMLHttpRequest()

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          setProgress((e.loaded / e.total) * 100)
        }
      }

      xhr.onload = () => {
        if (xhr.status === 200) {
          onUploaded()
          onClose()
          setFile(null)
          setProgress(0)
        }
      }

      xhr.onerror = () => {
        console.error('Upload failed')
      }

      xhr.open('POST', '/api/scenes/upload')
      xhr.send(formData)
    } catch (error) {
      console.error('Upload error:', error)
    } finally {
      setUploading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>上传场景包</DialogTitle>
          <DialogDescription>
            上传 ZIP 格式的场景包文件
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* 文件选择 */}
          <div className="space-y-2">
            <Label>场景包文件</Label>
            <div className="border-2 border-dashed rounded-lg p-8 text-center">
              {file ? (
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <FileText className="h-8 w-8 text-muted-foreground" />
                    <span className="text-sm">{file.name}</span>
                    <span className="text-xs text-muted-foreground">
                      ({(file.size / 1024 / 1024).toFixed(2)} MB)
                    </span>
                  </div>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setFile(null)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ) : (
                <label htmlFor="file-upload" className="cursor-pointer">
                  <Upload className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                  <div className="text-sm text-muted-foreground">
                    点击选择或拖拽文件到此处
                  </div>
                  <input
                    id="file-upload"
                    type="file"
                    accept=".zip"
                    className="hidden"
                    onChange={(e) => setFile(e.target.files?.[0] || null)}
                  />
                </label>
              )}
            </div>
          </div>

          {/* 上传进度 */}
          {uploading && (
            <div className="space-y-2">
              <div className="text-sm text-muted-foreground">
                上传中... {Math.round(progress)}%
              </div>
              <Progress value={progress} />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={onClose}
            disabled={uploading}
          >
            取消
          </Button>
          <Button
            onClick={handleUpload}
            disabled={!file || uploading}
          >
            {uploading ? '上传中...' : '上传'}
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
| `frontend/src/pages/SceneGalleryPage.tsx` | 创建 | 场景库主页面 |
| `frontend/src/components/scene/SceneCard.tsx` | 创建 | 场景卡片组件 |
| `frontend/src/components/scene/UploadSceneDialog.tsx` | 创建 | 上传对话框 |
| `frontend/src/components/scene/SceneDetailDialog.tsx` | 创建 | 详情对话框 |
| `frontend/src/types/scene.ts` | 创建 | 场景类型定义 |

---

## 验收标准

- [ ] 场景列表正确显示
- [ ] 搜索功能有效
- [ ] 过滤功能正常
- [ ] 上传功能可用
- [ ] 编辑功能正常
- [ ] 删除功能安全

---

## 参考文档

- M4-001: 场景包上传功能
- M0-014: 场景包根结构

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
