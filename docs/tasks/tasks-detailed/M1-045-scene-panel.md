# M1-045: 实现 ScenePanel 场景面板组件

**任务ID**: M1-045
**标题**: 实现 ScenePanel 场景面板组件
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-017

---

## 任务描述

实现场景显示面板组件，用于显示当前场景信息、描述、NPC、线索等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-045-01 | 设计场景面板布局 | UI 设计 | 20min |
| M1-045-02 | 实现场景描述显示 | Description | 25min |
| M1-045-03 | 实现 NPC 列表 | NPCs | 25min |
| M1-045-04 | 实现线索显示 | Clues | 25min |
| M1-045-05 | 实现场景切换 | Navigation | 20min |
| M1-045-06 | 实现图片显示 | Images | 15min |
| M1-045-07 | 编写面板测试 | 测试覆盖 | 10min |

---

## 场景面板组件

```tsx
// frontend/src/components/game/ScenePanel.tsx
import { useState, useEffect } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Image } from '@/components/ui/image'
import type { Scene } from '@/types/scene'

interface ScenePanelProps {
  scene: Scene
  onSceneChange?: (sceneId: string) => void
}

export function ScenePanel({ scene, onSceneChange }: ScenePanelProps) {
  const { user } = useAuth()
  const [activeTab, setActiveTab] = useState('info')
  const [currentImageIndex, setCurrentImageIndex] = useState(0)

  const isKp = user?.role === 'kp'

  const images = scene.images || []
  const currentImage = images[currentImageIndex]

  const handlePrevImage = () => {
    setCurrentImageIndex((i) => (i > 0 ? i - 1 : images.length - 1))
  }

  const handleNextImage = () => {
    setCurrentImageIndex((i) => (i < images.length - 1 ? i + 1 : 0))
  }

  return (
    <div className="space-y-4">
      {/* 场景标题 */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="space-y-1">
              <CardTitle>{scene.name}</CardTitle>
              <p className="text-sm text-muted-foreground">
                {scene.location}
              </p>
            </div>

            {/* 场景切换（仅 KP） */}
            {isKp && onSceneChange && (
              <div className="flex space-x-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {/* 切换到上一个场景 */}}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {/* 切换到下一个场景 */}}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            )}
          </div>
        </CardHeader>
      </Card>

      {/* 场景图片 */}
      {images.length > 0 && (
        <Card>
          <CardContent className="p-4">
            <div className="relative aspect-video rounded-lg overflow-hidden bg-black">
              <Image
                src={currentImage}
                alt={scene.name}
                fill
                className="object-cover"
              />
            </div>

            {images.length > 1 && (
              <div className="flex items-center justify-between mt-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handlePrevImage}
                >
                  上一张
                </Button>
                <span className="text-sm text-muted-foreground">
                  {currentImageIndex + 1} / {images.length}
                </span>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleNextImage}
                >
                  下一张
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* 场景信息标签页 */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="info">描述</TabsTrigger>
          <TabsTrigger value="npcs">NPC</TabsTrigger>
          <TabsTrigger value="clues">线索</TabsTrigger>
        </TabsList>

        <TabsContent value="info" className="mt-4">
          <Card>
            <CardContent className="p-4">
              <div className="prose prose-sm max-w-none">
                <p>{scene.description}</p>

                {scene.atmosphere && (
                  <div className="mt-4 p-3 bg-muted rounded-lg">
                    <div className="text-xs text-muted-foreground mb-1">
                      氛围
                    </div>
                    <p className="text-sm">{scene.atmosphere}</p>
                  </div>
                )}

                {scene.sounds && (
                  <div className="mt-2">
                    <span className="text-xs text-muted-foreground">
                      声音: {scene.sounds}
                    </span>
                  </div>
                )}

                {scene.smells && (
                  <div className="mt-2">
                    <span className="text-xs text-muted-foreground">
                      气味: {scene.smells}
                    </span>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="npcs" className="mt-4">
          <Card>
            <CardContent className="p-4">
              {scene.npcs && scene.npcs.length > 0 ? (
                <div className="space-y-3">
                  {scene.npcs.map((npc, index) => (
                    <div
                      key={index}
                      className="flex items-start justify-between p-2 rounded hover:bg-muted"
                    >
                      <div className="flex-1">
                        <div className="font-medium">{npc.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {npc.description}
                        </div>
                      </div>
                      <Badge variant="secondary">
                        {npc.status || '在场'}
                      </Badge>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center text-sm text-muted-foreground py-4">
                  本场景没有 NPC
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="clues" className="mt-4">
          <Card>
            <CardContent className="p-4">
              {scene.clues && scene.clues.length > 0 ? (
                <div className="space-y-3">
                  {scene.clues.map((clue, index) => (
                    <div
                      key={index}
                      className="p-3 border rounded-lg"
                    >
                      <div className="font-medium mb-1">{clue.title}</div>
                      <div className="text-sm text-muted-foreground">
                        {clue.content}
                      </div>
                      {clue.revealed_to && (
                        <div className="mt-2">
                          {clue.revealed_to.map((userId, i) => (
                            <Badge key={i} variant="outline" className="mr-1">
                              {userId}
                            </Badge>
                          ))}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center text-sm text-muted-foreground py-4">
                  本场景没有线索
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 场景选项（KP 可见） */}
      {isKp && scene.options && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">场景选项</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {scene.options.map((option, index) => (
                <Button
                  key={index}
                  variant="outline"
                  size="sm"
                  className="w-full justify-start"
                  onClick={() => option.action()}
                >
                  {option.label}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
```

---

## 场景类型定义

```tsx
// frontend/src/types/scene.ts
export interface Scene {
  id: string
  name: string
  location: string
  description: string
  atmosphere?: string
  sounds?: string
  smells?: string
  images?: string[]
  npcs?: SceneNPC[]
  clues?: SceneClue[]
  options?: SceneOption[]
}

export interface SceneNPC {
  id: string
  name: string
  description: string
  status?: string
  position?: string
}

export interface SceneClue {
  id: string
  title: string
  content: string
  revealed_to?: string[]
  difficulty?: string
}

export interface SceneOption {
  label: string
  action: () => void
}
```

---

## 小地图组件

```tsx
// frontend/src/components/game/Minimap.tsx
import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'

interface MinimapProps {
  locations: Array<{
    id: string
    name: string
    x: number
    y: number
    current?: boolean
  }>
  onLocationClick?: (locationId: string) => void
}

export function Minimap({ locations, onLocationClick }: MinimapProps) {
  const [hoveredLocation, setHoveredLocation] = useState<string | null>(null)

  return (
    <Card>
      <CardContent className="p-4">
        <div className="relative w-full aspect-square bg-muted rounded-lg">
          {/* 地图背景 */}
          <div className="absolute inset-0 opacity-20">
            <svg width="100%" height="100%">
              <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
                <path d="M 20 0 L 0 0 0 20" fill="none" stroke="currentColor" strokeWidth="0.5"/>
              </pattern>
              <rect width="100%" height="100%" fill="url(#grid)" />
            </svg>
          </div>

          {/* 位置标记 */}
          {locations.map((location) => (
            <div
              key={location.id}
              className={`absolute w-3 h-3 rounded-full cursor-pointer transition-transform hover:scale-150 ${
                location.current ? 'bg-primary' : 'bg-muted-foreground'
              }`}
              style={{
                left: `${location.x}%`,
                top: `${location.y}%`,
                transform: 'translate(-50%, -50%)',
              }}
              onClick={() => onLocationClick?.(location.id)}
              onMouseEnter={() => setHoveredLocation(location.id)}
              onMouseLeave={() => setHoveredLocation(null)}
            />
          ))}

          {/* 悬停提示 */}
          {hoveredLocation && (
            <div className="absolute bottom-0 left-0 right-0 bg-background border rounded p-2 text-xs">
              {locations.find(l => l.id === hoveredLocation)?.name}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/ScenePanel.tsx` | 创建 | 场景面板主组件 |
| `frontend/src/components/game/Minimap.tsx` | 创建 | 小地图组件 |
| `frontend/src/types/scene.ts` | 创建 | 场景类型定义 |

---

## 验收标准

- [ ] 场景信息显示完整
- [ ] 图片切换流畅
- [ ] NPC 列表准确
- [ ] 线索显示正确
- [ ] 场景切换有效
- [ ] 小地图功能正常

---

## 参考文档

- M0-016: 场景集合结构
- M0-018: Location 地点结构

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
