# M4-004: 实现场景编辑器

**任务ID**: M4-004
**标题**: 实现场景编辑器
**类型**: frontend (前端开发)
**预估工时**: 3h
**依赖**: M4-001, M4-008

---

## 任务描述

实现可视化的场景包编辑器，允许 KP 在线创建和编辑场景包，包括场景、NPC、线索、手递物等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-004-01 | 设计编辑器布局 | Layout Design | 25min |
| M4-004-02 | 实现场景列表组件 | Scene List | 30min |
| M4-004-03 | 实现 NPC 编辑器 | NPC Editor | 35min |
| M4-004-04 | 实现场景编辑器 | Scene Editor | 35min |
| M4-004-05 | 实现线索编辑器 | Clue Editor | 25min |
| M4-004-06 | 实现手递物编辑器 | Handout Editor | 25min |
| M4-004-07 | 实现导出功能 | Export | 20min |
| M4-004-08 | 编写编辑器测试 | 测试覆盖 | 15min |

---

## 编辑器布局结构

```tsx
// frontend/src/components/scene/SceneEditor.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Save, Download, Upload } from 'lucide-react'
import { SceneList } from './SceneList'
import { SceneEditor } from './SceneEditorPanel'
import { NPCEditor } from './NPCEditor'
import { ClueEditor } from './ClueEditor'
import { HandoutEditor } from './HandoutEditor'
import { MetadataEditor } from './MetadataEditor'

interface ScenePackage {
  metadata: any
  scenes: any[]
  npcs: any[]
  clues: any[]
  handouts: any[]
}

export function SceneEditor() {
  const [pkg, setPkg] = useState<ScenePackage>({
    metadata: {
      name: '',
      version: '1.0.0',
      author: '',
      description: '',
      tags: [],
    },
    scenes: [],
    npcs: [],
    clues: [],
    handouts: [],
  })

  const [activeTab, setActiveTab] = useState('metadata')
  const [selectedScene, setSelectedScene] = useState<any>(null)
  const [unsavedChanges, setUnsavedChanges] = useState(false)

  const handleSave = async () => {
    try {
      const response = await fetch('/api/scenes/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(pkg),
      })

      if (!response.ok) throw new Error('保存失败')

      setUnsavedChanges(false)
    } catch (error) {
      console.error('Failed to save:', error)
    }
  }

  const handleExport = () => {
    const blob = new Blob([JSON.stringify(pkg, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${pkg.metadata.name || 'scene'}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleImport = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      try {
        const data = JSON.parse(event.target?.result as string)
        setPkg(data)
        setUnsavedChanges(false)
      } catch (error) {
        console.error('Failed to parse JSON:', error)
      }
    }
    reader.readAsText(file)
  }

  return (
    <div className="h-screen flex flex-col">
      {/* 顶部工具栏 */}
      <header className="border-b p-4">
        <div className="flex items-center justify-between">
          <Input
            value={pkg.metadata.name}
            onChange={(e) => {
              setPkg({
                ...pkg,
                metadata: { ...pkg.metadata, name: e.target.value }
              })
              setUnsavedChanges(true)
            }}
            placeholder="场景包名称"
            className="text-xl font-semibold max-w-md"
          />

          <div className="flex space-x-2">
            <Button size="sm" variant="outline" onClick={handleSave}>
              <Save className="h-4 w-4 mr-1" />
              保存
            </Button>
            <Button size="sm" variant="outline" onClick={handleExport}>
              <Download className="h-4 w-4 mr-1" />
              导出
            </Button>
            <Button size="sm" variant="outline" onClick={() => document.getElementById('import-input')?.click()}>
              <Upload className="h-4 w-4 mr-1" />
              导入
            </Button>
            <input
              id="import-input"
              type="file"
              accept=".json"
              className="hidden"
              onChange={handleImport}
            />
          </div>
        </div>

        {unsavedChanges && (
          <p className="text-sm text-yellow-600 mt-2">
            ⚠️ 有未保存的更改
          </p>
        )}
      </header>

      {/* 主内容区 */}
      <div className="flex-1 overflow-hidden">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="h-full">
          <div className="border-b px-4">
            <TabsList>
              <TabsTrigger value="metadata">元信息</TabsTrigger>
              <TabsTrigger value="scenes">场景 ({pkg.scenes.length})</TabsTrigger>
              <TabsTrigger value="npcs">NPC ({pkg.npcs.length})</TabsTrigger>
              <TabsTrigger value="clues">线索 ({pkg.clues.length})</TabsTrigger>
              <TabsTrigger value="handouts">手递物 ({pkg.handouts.length})</TabsTrigger>
            </TabsList>
          </div>

          <div className="p-4 h-full overflow-y-auto">
            <TabsContent value="metadata" className="mt-0">
              <MetadataEditor
                metadata={pkg.metadata}
                onChange={(metadata) => {
                  setPkg({ ...pkg, metadata })
                  setUnsavedChanges(true)
                }}
              />
            </TabsContent>

            <TabsContent value="scenes" className="mt-0">
              <SceneEditor
                scenes={pkg.scenes}
                npcs={pkg.npcs}
                clues={pkg.clues}
                handouts={pkg.handouts}
                selectedScene={selectedScene}
                onSelectScene={setSelectedScene}
                onChange={(scenes) => {
                  setPkg({ ...pkg, scenes })
                  setUnsavedChanges(true)
                }}
              />
            </TabsContent>

            <TabsContent value="npcs" className="mt-0">
              <NPCEditor
                npcs={pkg.npcs}
                onChange={(npcs) => {
                  setPkg({ ...pkg, npcs })
                  setUnsavedChanges(true)
                }}
              />
            </TabsContent>

            <TabsContent value="clues" className="mt-0">
              <ClueEditor
                clues={pkg.clues}
                scenes={pkg.scenes}
                onChange={(clues) => {
                  setPkg({ ...pkg, clues })
                  setUnsavedChanges(true)
                }}
              />
            </TabsContent>

            <TabsContent value="handouts" className="mt-0">
              <HandoutEditor
                handouts={pkg.handouts}
                onChange={(handouts) => {
                  setPkg({ ...pkg, handouts })
                  setUnsavedChanges(true)
                }}
              />
            </TabsContent>
          </div>
        </Tabs>
      </div>
    </div>
  )
}
```

---

## 场景编辑面板

```tsx
// frontend/src/components/scene/SceneEditorPanel.tsx
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Plus, Trash2 } from 'lucide-react'
import { RichTextEditor } from '@/components/ui/rich-text-editor'

interface Scene {
  id: string
  name: string
  description: string
  location?: string
  time?: string
  clues?: string[]
  npcs?: string[]
  handouts?: string[]
}

interface SceneEditorProps {
  scenes: Scene[]
  npcs: any[]
  clues: any[]
  handouts: any[]
  selectedScene: Scene | null
  onSelectScene: (scene: Scene) => void
  onChange: (scenes: Scene[]) => void
}

export function SceneEditor({
  scenes,
  npcs,
  clues,
  handouts,
  selectedScene,
  onSelectScene,
  onChange,
}: SceneEditorProps) {
  const [editingScene, setEditingScene] = useState<Scene | null>(null)

  const handleAddScene = () => {
    const newScene: Scene = {
      id: `scene_${Date.now()}`,
      name: '新场景',
      description: '',
    }
    onChange([...scenes, newScene])
    setEditingScene(newScene)
  }

  const handleUpdateScene = (updates: Partial<Scene>) => {
    if (!editingScene) return

    const updated = { ...editingScene, ...updates }
    setEditingScene(updated)

    onChange(scenes.map(s => s.id === updated.id ? updated : s))
  }

  const handleDeleteScene = (sceneId: string) => {
    onChange(scenes.filter(s => s.id !== sceneId))
    if (editingScene?.id === sceneId) {
      setEditingScene(null)
    }
  }

  return (
    <div className="grid grid-cols-3 gap-4 h-full">
      {/* 场景列表 */}
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle className="text-base flex items-center justify-between">
            <span>场景列表</span>
            <Button size="sm" onClick={handleAddScene}>
              <Plus className="h-4 w-4 mr-1" />
              添加
            </Button>
          </CardTitle>
        </CardHeader>

        <CardContent>
          <div className="space-y-2">
            {scenes.map((scene) => (
              <div
                key={scene.id}
                className={`p-3 border rounded cursor-pointer ${
                  selectedScene?.id === scene.id ? 'bg-primary/10 border-primary' : ''
                }`}
                onClick={() => {
                  onSelectScene(scene)
                  setEditingScene(scene)
                }}
              >
                <div className="font-medium">{scene.name}</div>
                {scene.location && (
                  <div className="text-xs text-muted-foreground">
                    📍 {scene.location}
                  </div>
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 编辑区 */}
      {editingScene && (
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle className="text-base flex items-center justify-between">
              <Input
                value={editingScene.name}
                onChange={(e) => handleUpdateScene({ name: e.target.value })}
                className="text-lg font-semibold"
              />
              <Button
                size="sm"
                variant="destructive"
                onClick={() => handleDeleteScene(editingScene.id)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </CardTitle>
          </CardHeader>

          <CardContent className="space-y-4">
            {/* 基本信息 */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>地点</Label>
                <Input
                  value={editingScene.location || ''}
                  onChange={(e) => handleUpdateScene({ location: e.target.value })}
                  placeholder="如：波士顿图书馆"
                />
              </div>
              <div>
                <Label>时间</Label>
                <Input
                  value={editingScene.time || ''}
                  onChange={(e) => handleUpdateScene({ time: e.target.value })}
                  placeholder="如：1920年1月15日 14:00"
                />
              </div>
            </div>

            {/* 描述 */}
            <div>
              <Label>场景描述</Label>
              <RichTextEditor
                content={editingScene.description}
                onChange={(content) => handleUpdateScene({ description: content })}
              />
            </div>

            {/* 关联元素 */}
            <div className="grid grid-cols-3 gap-4">
              <div>
                <Label>线索</Label>
                <div className="border rounded p-2 min-h-[100px]">
                  {editingScene.clues?.map(clueId => {
                    const clue = clues.find(c => c.id === clueId)
                    return clue ? (
                      <div key={clueId} className="text-sm p-1 bg-muted rounded mb-1">
                        {clue.name}
                      </div>
                    ) : null
                  }) || <div className="text-sm text-muted-foreground">无</div>}
                </div>
              </div>

              <div>
                <Label>NPC</Label>
                <div className="border rounded p-2 min-h-[100px]">
                  {editingScene.npcs?.map(npcId => {
                    const npc = npcs.find(n => n.id === npcId)
                    return npc ? (
                      <div key={npcId} className="text-sm p-1 bg-muted rounded mb-1">
                        {npc.name}
                      </div>
                    ) : null
                  }) || <div className="text-sm text-muted-foreground">无</div>}
                </div>
              </div>

              <div>
                <Label>手递物</Label>
                <div className="border rounded p-2 min-h-[100px]">
                  {editingScene.handouts?.map(handoutId => {
                    const handout = handouts.find(h => h.id === handoutId)
                    return handout ? (
                      <div key={handoutId} className="text-sm p-1 bg-muted rounded mb-1">
                        {handout.title}
                      </div>
                    ) : null
                  }) || <div className="text-sm text-muted-foreground">无</div>}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
```

---

## NPC 编辑器

```tsx
// frontend/src/components/scene/NPCEditor.tsx
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Plus, Trash2 } from 'lucide-react'

interface NPC {
  id: string
  name: string
  age?: number
  occupation?: string
  description: string
  personality?: string
  appearance?: string
  notes?: string
}

interface NPCEditorProps {
  npcs: NPC[]
  onChange: (npcs: NPC[]) => void
}

export function NPCEditor({ npcs, onChange }: NPCEditorProps) {
  const [editingNPC, setEditingNPC] = useState<NPC | null>(null)

  const handleAddNPC = () => {
    const newNPC: NPC = {
      id: `npc_${Date.now()}`,
      name: '新角色',
      description: '',
    }
    onChange([...npcs, newNPC])
    setEditingNPC(newNPC)
  }

  const handleUpdateNPC = (updates: Partial<NPC>) => {
    if (!editingNPC) return

    const updated = { ...editingNPC, ...updates }
    setEditingNPC(updated)
    onChange(npcs.map(n => n.id === updated.id ? updated : n))
  }

  const handleDeleteNPC = (npcId: string) => {
    onChange(npcs.filter(n => n.id !== npcId))
    if (editingNPC?.id === npcId) {
      setEditingNPC(null)
    }
  }

  return (
    <div className="grid grid-cols-3 gap-4">
      {/* NPC 列表 */}
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle className="text-base flex items-center justify-between">
            <span>NPC 列表</span>
            <Button size="sm" onClick={handleAddNPC}>
              <Plus className="h-4 w-4 mr-1" />
              添加
            </Button>
          </CardTitle>
        </CardHeader>

        <CardContent>
          <div className="space-y-2">
            {npcs.map((npc) => (
              <div
                key={npc.id}
                className={`p-3 border rounded cursor-pointer ${
                  editingNPC?.id === npc.id ? 'bg-primary/10 border-primary' : ''
                }`}
                onClick={() => setEditingNPC(npc)}
              >
                <div className="font-medium">{npc.name}</div>
                {npc.occupation && (
                  <div className="text-xs text-muted-foreground">
                    {npc.occupation}
                  </div>
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 编辑区 */}
      {editingNPC && (
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle className="text-base flex items-center justify-between">
              <Input
                value={editingNPC.name}
                onChange={(e) => handleUpdateNPC({ name: e.target.value })}
                className="text-lg font-semibold"
              />
              <Button
                size="sm"
                variant="destructive"
                onClick={() => handleDeleteNPC(editingNPC.id)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </CardTitle>
          </CardHeader>

          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>年龄</Label>
                <Input
                  type="number"
                  value={editingNPC.age || ''}
                  onChange={(e) => handleUpdateNPC({ age: parseInt(e.target.value) || undefined })}
                />
              </div>
              <div>
                <Label>职业</Label>
                <Input
                  value={editingNPC.occupation || ''}
                  onChange={(e) => handleUpdateNPC({ occupation: e.target.value })}
                />
              </div>
            </div>

            <div>
              <Label>描述</Label>
              <Textarea
                value={editingNPC.description}
                onChange={(e) => handleUpdateNPC({ description: e.target.value })}
                rows={3}
              />
            </div>

            <div>
              <Label>性格</Label>
              <Textarea
                value={editingNPC.personality || ''}
                onChange={(e) => handleUpdateNPC({ personality: e.target.value })}
                rows={2}
              />
            </div>

            <div>
              <Label>外貌</Label>
              <Textarea
                value={editingNPC.appearance || ''}
                onChange={(e) => handleUpdateNPC({ appearance: e.target.value })}
                rows={2}
              />
            </div>

            <div>
              <Label>备注</Label>
              <Textarea
                value={editingNPC.notes || ''}
                onChange={(e) => handleUpdateNPC({ notes: e.target.value })}
                rows={3}
              />
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/scene/SceneEditor.tsx` | 创建 | 场景编辑器主组件 |
| `frontend/src/components/scene/SceneEditorPanel.tsx` | 创建 | 场景编辑面板 |
| `frontend/src/components/scene/NPCEditor.tsx` | 创建 | NPC 编辑器 |
| `frontend/src/components/scene/ClueEditor.tsx` | 创建 | 线索编辑器 |
| `frontend/src/components/scene/HandoutEditor.tsx` | 创建 | 手递物编辑器 |
| `frontend/src/components/scene/MetadataEditor.tsx` | 创建 | 元信息编辑器 |
| `frontend/src/components/ui/rich-text-editor.tsx` | 创建 | 富文本编辑器 |

---

## 验收标准

- [ ] 所有元素可编辑
- [ ] 实时保存提示
- [ ] 导出功能正常
- [ ] 导入功能正常
- [ ] 验证规则有效
- [ ] 用户体验流畅

---

## 参考文档

- M4-001: 场景包上传功能
- M4-008: JSON 解析器
- M0-022: 场景包 JSON Schema

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
