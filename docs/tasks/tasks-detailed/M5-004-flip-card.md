# M5-004: 实现卡牌翻转功能

**任务ID**: M5-004
**标题**: 实现卡牌翻转功能
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M0-020

---

## 任务描述

实现可翻转的卡牌组件，用于显示手递物、线索等内容时的翻转效果。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-004-01 | 设计翻转动画 | Animation | 30min |
| M5-004-02 | 实现卡牌组件 | Card Component | 35min |
| M5-004-03 | 实现正面/反面内容 | Front/Back | 25min |
| M5-004-04 | 实现点击翻转 | Click to Flip | 20min |
| M5-004-05 | 实现批量操作 | Batch Actions | 20min |
| M5-004-06 | 编写动画测试 | 测试覆盖 | 10min |

---

## 卡牌翻转组件

```tsx
// frontend/src/components/game/FlipCard.tsx
import { useState, useRef } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent } from '@/components/ui/card'

interface FlipCardProps {
  front: React.ReactNode
  back: React.ReactNode
  width?: string
  height?: string
  onFlip?: (isFlipped: boolean) => void
}

export function FlipCard({
  front,
  back,
  width = '300px',
  height = '420px',
  onFlip,
}: FlipCardProps) {
  const [isFlipped, setIsFlipped] = useState(false)

  const handleFlip = () => {
    const newState = !isFlipped
    setIsFlipped(newState)
    onFlip?.(newState)
  }

  return (
    <div
      style={{ width, height, perspective: '1000px' }}
      className="cursor-pointer"
      onClick={handleFlip}
    >
      <motion.div
        className="relative w-full h-full"
        style={{ transformStyle: 'preserve-3d' }}
        initial={false}
        animate={{ rotateY: isFlipped ? 180 : 0 }}
        transition={{ duration: 0.6, type: 'spring' }}
      >
        {/* 正面 */}
        <Card
          className="absolute w-full h-full backface-hidden"
          style={{ backfaceVisibility: 'hidden' }}
        >
          <CardContent className="p-0 h-full">
            {front}
          </CardContent>
        </Card>

        {/* 反面 */}
        <Card
          className="absolute w-full h-full backface-hidden"
          style={{
            backfaceVisibility: 'hidden',
            transform: 'rotateY(180deg)',
          }}
        >
          <CardContent className="p-0 h-full">
            {back}
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}
```

---

## 手递物卡片组件

```tsx
// frontend/src/components/game/HandoutCard.tsx
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Eye, EyeOff, Download } from 'lucide-react'
import { FlipCard } from './FlipCard'

interface Handout {
  id: string
  title: string
  front_content: string
  back_content: string
  revealed_to?: string[]
  is_secret?: boolean
  type: 'text' | 'image'
  image_url?: string
}

interface HandoutCardProps {
  handout: Handout
  onReveal?: (handoutId: string) => void
}

export function HandoutCard({ handout, onReveal }: HandoutCardProps) {
  const { user } = useAuth()
  const [isRevealed, setIsRevealed] = useState(false)
  const [showBack, setShowBack] = useState(false)

  const isKp = user?.role === 'kp'
  const canViewBack = handout.revealed_to?.includes(user?.id) || isKp
  const showRevealButton = !isRevealed && handout.revealed_to?.includes(user?.id)

  const handleReveal = () => {
    onReveal?.(handout.id)
    setIsRevealed(true)
  }

  // 正面内容
  const front = (
    <div className="h-full flex flex-col">
      {handout.type === 'image' && handout.image_url ? (
        <div className="flex-1 relative bg-muted">
          <img
            src={handout.image_url}
            alt={handout.title}
            className="w-full h-full object-contain"
          />
        </div>
      ) : (
        <div className="flex-1 p-4 flex items-center justify-center">
          <span className="text-sm text-muted-foreground">
            {handout.front_content}
          </span>
        </div>
      )}

      <div className="p-3 border-t">
        <div className="flex items-center justify-between">
          <span className="font-medium text-sm">{handout.title}</span>
          {handout.is_secret && (
            <Badge variant="secondary" className="text-xs">
              <EyeOff className="h-3 w-3" />
            </Badge>
          )}
        </div>
      </div>

      {showRevealButton && (
        <Button
          size="sm"
          variant="outline"
          className="w-full mt-2"
          onClick={handleReveal}
        >
          查看内容
        </Button>
      )}
    </div>
  )

  // 反面内容
  const back = (
    <div className="h-full flex flex-col">
      <div className="flex-1 p-4 overflow-y-auto">
        <p className="text-sm whitespace-pre-wrap">
          {handout.back_content}
        </p>
      </div>

      {handout.type === 'image' && handout.image_url && (
        <div className="p-3 border-t">
          <img
            src={handout.image_url}
            alt={handout.title}
            className="w-full h-32 object-cover rounded"
          />
        </div>
      )}

      <div className="p-3 border-t flex items-center justify-between">
        <span className="text-xs text-muted-foreground">
          {handout.revealed_to?.length || 0} 人已查看
        </span>
      </div>
    </div>
  )

  return (
    <div className="space-y-2">
      {/* 翻转按钮 */}
      {canViewBack && (
        <Button
          size="sm"
          variant="outline"
          onClick={() => setShowBack(!showBack)}
          className="w-full"
        >
          {showBack ? <Eye className="h-4 w-4 mr-1" /> : <EyeOff className="h-4 w-4 mr-1" />}
          {showBack ? '查看反面' : '查看正面'}
        </Button>
      )}

      {/* 卡片 */}
      <FlipCard
        front={front}
        back={canViewBack ? back : null}
        width="300px"
        height="420px"
        onFlip={(flipped) => setShowBack(flipped)}
      />
    </div>
  )
}
```

---

## 批量手递物管理

```tsx
// frontend/src/components/game/HandoutManager.tsx
import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import { Download } from 'lucide-react'
import { HandoutCard } from './HandoutCard'

interface Handout {
  id: string
  title: string
  is_distributed: boolean
  revealed_count: number
}

export function HandoutManager() {
  const { user } = useAuth()
  const [handouts, setHandouts] = useState<Handout[]>([])
  const [selectedHandouts, setSelectedHandouts] = useState<Set<string>>(new Set())

  const isKp = user?.role === 'kp'

  useEffect(() => {
    loadHandouts()
  }, [])

  const loadHandouts = async () => {
    try {
      const response = await fetch('/api/handouts')
      if (!response.ok) throw new Error('Failed to load handouts')

      const data = await response.json()
      setHandouts(data.handouts || [])
    } catch (error) {
      console.error('Failed to load handouts:', error)
    }
  }

  const handleDistribute = async () => {
    if (selectedHandouts.size === 0) return

    try {
      const response = await fetch('/api/handouts/distribute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          handout_ids: Array.from(selectedHandouts),
        }),
      })

      if (!response.ok) throw new Error('分发失败')

      // 刷新列表
      await loadHandouts()
      setSelectedHandouts(new Set())
    } catch (error) {
      console.error('Failed to distribute handouts:', error)
    }
  }

  const toggleSelect = (handoutId: string) => {
    const newSet = new Set(selectedHandouts)
    if (newSet.has(handoutId)) {
      newSet.delete(handoutId)
    } else {
      newSet.add(handoutId)
    }
    setSelectedHandouts(newSet)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span>手递物管理</span>
          {isKp && (
            <Button
              size="sm"
              onClick={handleDistribute}
              disabled={selectedHandouts.size === 0}
            >
              分发 ({selectedHandouts.size})
            </Button>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="space-y-3">
          {handouts.map((handout) => (
            <div
              key={handout.id}
              className="flex items-center justify-between p-2 border rounded"
            >
              <div className="flex items-center space-x-3">
                {isKp && (
                  <Checkbox
                    checked={selectedHandouts.has(handout.id)}
                    onCheckedChange={() => toggleSelect(handout.id)}
                  />
                )}

                <span className="text-sm">{handout.title}</span>

                {handout.is_distributed && (
                  <Badge variant="secondary" className="text-xs">
                    已分发
                  </Badge>
                )}

                <span className="text-xs text-muted-foreground">
                  {handout.revealed_count} 人查看
                </span>
              </div>

              <Button
                size="sm"
                variant="ghost"
                onClick={() => {/* 下载/查看 */}}
              >
                <Download className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 3D 翻转样式（可选）

```css
/* frontend/src/components/game/FlipCard.css */
.flip-card {
  perspective: 1000px;
  width: 300px;
  height: 420px;
}

.flip-card-inner {
  position: relative;
  width: 100%;
  height: 100%;
  transition: transform 0.6s;
  transform-style: preserve-3d;
}

.flip-card.flipped .flip-card-inner {
  transform: rotateY(180deg);
}

.flip-card-front,
.flip-card-back {
  position: absolute;
  width: 100%;
  height: 100%;
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
}

.flip-card-back {
  transform: rotateY(180deg);
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/FlipCard.tsx` | 创建 | 翻转卡片组件 |
| `frontend/src/components/game/HandoutCard.tsx` | 创建 | 手递物卡片 |
| `frontend/src/components/game/HandoutManager.tsx` | 创建 | 手递物管理 |
| `frontend/src/components/game/FlipCard.css` | 创建 | 3D 样式 |

---

## 验收标准

- [ ] 翻转动画流畅
- [ ] 正面/反面内容正确
- [ ] 点击翻转有效
- [ ] 权限控制正确
- [ ] 批量操作可用
- [ ] 3D 效果良好

---

## 参考文档

- M0-020: Handout 手递物格式
- Framer Motion 动画库

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
