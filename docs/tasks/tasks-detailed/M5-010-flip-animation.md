# M5-010: 实现卡牌翻转动画效果

**任务ID**: M5-010
**标题**: 实现卡牌翻转动画效果
**类型**: frontend (前端开发)
**预估工时**: 1.5h
**依赖**: M5-004

---

## 任务描述

实现流畅的卡牌翻转动画效果，用于线索卡、手递物等内容的展示，提供更好的视觉体验。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-010-01 | 设计翻转动画 | Flip Animation | 25min |
| M5-010-02 | 实现 3D 翻转效果 | 3D Flip | 30min |
| M5-010-03 | 实现正反面内容 | Front & Back | 20min |
| M5-010-04 | 添加音效反馈 | Sound Effects | 15min |
| M5-010-05 | 优化性能 | Performance | 15min |

---

## 卡牌翻转核心组件

```typescript
// frontend/src/components/card/FlipCard.tsx
import { ReactNode, useRef, useState } from 'react'
import { motion, useMotionValue, useTransform, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'

export interface FlipCardProps {
  front: ReactNode
  back: ReactNode
  width?: number
  height?: number
  className?: string
  flipDuration?: number
  onFlip?: (isFlipped: boolean) => void
  disabled?: boolean
}

export function FlipCard({
  front,
  back,
  width = 200,
  height = 280,
  className,
  flipDuration = 0.6,
  onFlip,
  disabled = false,
}: FlipCardProps) {
  const [isFlipped, setIsFlipped] = useState(false)
  const rotateX = useMotionValue(0)
  const rotateY = useMotionValue(0)

  // 计算透视效果的旋转
  const transform = useTransform(
    () => `perspective(1000px) rotateX(${rotateX.get()}deg) rotateY(${rotateY.get()}deg)`
  )

  const handleFlip = () => {
    if (disabled) return

    const newFlipped = !isFlipped
    setIsFlipped(newFlipped)
    onFlip?.(newFlipped)
  }

  return (
    <div
      className={cn('flip-card-container', className)}
      style={{ width, height, perspective: '1000px' }}
    >
      <motion.div
        className="flip-card-inner relative w-full h-full"
        style={{
          transformStyle: 'preserve-3d',
          rotateY: isFlipped ? 180 : 0,
          transition: `transform ${flipDuration}s cubic-bezier(0.4, 0, 0.2, 1)`,
        }}
        onClick={handleFlip}
      >
        {/* 正面 */}
        <div
          className="flip-card-front absolute w-full h-full backface-hidden"
          style={{
            backfaceVisibility: 'hidden',
            WebkitBackfaceVisibility: 'hidden',
          }}
        >
          {front}
        </div>

        {/* 背面 */}
        <div
          className="flip-card-back absolute w-full h-full backface-hidden"
          style={{
            backfaceVisibility: 'hidden',
            WebkitBackfaceVisibility: 'hidden',
            transform: 'rotateY(180deg)',
          }}
        >
          {back}
        </div>
      </motion.div>
    </div>
  )
}
```

---

## 线索卡组件

```tsx
// frontend/src/components/card/ClueCard.tsx
import { useState } from 'react'
import { FlipCard } from './FlipCard'
import { Card, CardContent } from '@/components/ui/card'
import { Lock, Unlock, Eye } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ClueCardProps {
  id: string
  title: string
  description: string
  secret?: string
  isRevealed?: boolean
  onReveal?: () => void
  className?: string
}

export function ClueCard({
  id,
  title,
  description,
  secret,
  isRevealed = false,
  onReveal,
  className,
}: ClueCardProps) {
  const [isFlipped, setIsFlipped] = useState(false)

  const front = (
    <Card className="w-full h-full bg-gradient-to-br from-amber-100 to-amber-200 dark:from-amber-900 dark:to-amber-800 border-amber-300 dark:border-amber-700">
      <CardContent className="flex flex-col items-center justify-center h-full p-6 text-center">
        <div className="text-4xl mb-4">🔍</div>
        <h3 className="text-xl font-bold text-amber-900 dark:text-amber-100 mb-2">
          {title}
        </h3>
        <p className="text-sm text-amber-700 dark:text-amber-300">
          点击查看线索
        </p>
        {!isRevealed && (
          <div className="mt-4 flex items-center text-amber-600 dark:text-amber-400">
            <Lock className="h-4 w-4 mr-1" />
            <span className="text-xs">未揭示</span>
          </div>
        )}
      </CardContent>
    </Card>
  )

  const backContent = (
    <Card className="w-full h-full bg-gradient-to-br from-slate-100 to-slate-200 dark:from-slate-800 dark:to-slate-900 border-slate-300 dark:border-slate-700">
      <CardContent className="h-full p-6 overflow-auto">
        <h3 className="text-lg font-bold text-slate-900 dark:text-slate-100 mb-3">
          {title}
        </h3>
        <p className="text-sm text-slate-700 dark:text-slate-300 mb-4">
          {description}
        </p>

        {secret && isRevealed && (
          <div className="mt-4 pt-4 border-t border-slate-300 dark:border-slate-700">
            <p className="text-xs font-semibold text-slate-600 dark:text-slate-400 mb-2">
              隐藏信息：
            </p>
            <p className="text-sm text-slate-800 dark:text-slate-200 italic">
              {secret}
            </p>
          </div>
        )}

        {!isRevealed && onReveal && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onReveal()
            }}
            className="mt-4 w-full py-2 px-4 bg-amber-500 hover:bg-amber-600 text-white rounded-lg transition-colors flex items-center justify-center"
          >
            <Unlock className="h-4 w-4 mr-2" />
            揭示线索
          </button>
        )}
      </CardContent>
    </Card>
  )

  return (
    <div className={cn('clue-card-wrapper', className)}>
      <FlipCard
        front={front}
        back={backContent}
        width={220}
        height={300}
        onFlip={setIsFlipped}
      />
    </div>
  )
}
```

---

## 手递物组件

```tsx
// frontend/src/components/card/HandoutCard.tsx
import { useState } from 'react'
import { FlipCard } from './FlipCard'
import { Card, CardContent } from '@/components/ui/card'
import { FileText, Eye, Download, Share } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface HandoutCardProps {
  id: string
  title: string
  description: string
  content?: string
  imageUrl?: string
  isPrivate?: boolean
  onReveal?: () => void
  onDownload?: () => void
  onShare?: () => void
  className?: string
}

export function HandoutCard({
  id,
  title,
  description,
  content,
  imageUrl,
  isPrivate = false,
  onReveal,
  onDownload,
  onShare,
  className,
}: HandoutCardProps) {
  const [isFlipped, setIsFlipped] = useState(false)
  const [isRevealed, setIsRevealed] = useState(!isPrivate)

  const handleReveal = () => {
    setIsRevealed(true)
    onReveal?.()
  }

  const front = (
    <Card className="w-full h-full bg-gradient-to-br from-blue-100 to-indigo-100 dark:from-blue-900 dark:to-indigo-900 border-blue-300 dark:border-blue-700">
      <CardContent className="flex flex-col items-center justify-center h-full p-6 text-center">
        <div className="text-5xl mb-4">📄</div>
        <h3 className="text-xl font-bold text-blue-900 dark:text-blue-100 mb-2">
          {title}
        </h3>
        <p className="text-sm text-blue-700 dark:text-blue-300 line-clamp-2">
          {description}
        </p>
        {isPrivate && !isRevealed && (
          <div className="mt-4 flex items-center text-blue-600 dark:text-blue-400">
            <Eye className="h-4 w-4 mr-1" />
            <span className="text-xs">私密内容</span>
          </div>
        )}
      </CardContent>
    </Card>
  )

  const back = (
    <Card className="w-full h-full bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-700">
      <CardContent className="h-full p-4 overflow-auto">
        <h3 className="text-lg font-bold text-slate-900 dark:text-slate-100 mb-3">
          {title}
        </h3>

        {imageUrl && isRevealed && (
          <div className="mb-4 rounded-lg overflow-hidden bg-slate-100 dark:bg-slate-800">
            <img
              src={imageUrl}
              alt={title}
              className="w-full h-auto object-contain"
            />
          </div>
        )}

        {content && isRevealed && (
          <div className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap mb-4">
            {content}
          </div>
        )}

        {!isRevealed && (
          <div className="flex items-center justify-center h-32 text-slate-400 dark:text-slate-600">
            <div className="text-center">
              <Eye className="h-8 w-8 mx-auto mb-2" />
              <p className="text-sm">内容已隐藏</p>
              <Button
                size="sm"
                variant="outline"
                onClick={handleReveal}
                className="mt-3"
              >
                显示内容
              </Button>
            </div>
          </div>
        )}

        {isRevealed && (
          <div className="flex gap-2 mt-4">
            {onDownload && (
              <Button size="sm" variant="outline" onClick={onDownload}>
                <Download className="h-4 w-4 mr-1" />
                下载
              </Button>
            )}
            {onShare && (
              <Button size="sm" variant="outline" onClick={onShare}>
                <Share className="h-4 w-4 mr-1" />
                分享
              </Button>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )

  return (
    <div className={cn('handout-card-wrapper', className)}>
      <FlipCard
        front={front}
        back={back}
        width={240}
        height={320}
        onFlip={setIsFlipped}
      />
    </div>
  )
}
```

---

## 卡牌样式

```css
/* frontend/src/components/card/flip-card.css */
.flip-card-container {
  position: relative;
  cursor: pointer;
}

.flip-card-inner {
  position: relative;
  width: 100%;
  height: 100%;
  transition: transform 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  transform-style: preserve-3d;
}

.flip-card-front,
.flip-card-back {
  position: absolute;
  width: 100%;
  height: 100%;
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
  border-radius: 0.75rem;
  overflow: hidden;
}

.flip-card-front {
  z-index: 2;
}

.flip-card-back {
  transform: rotateY(180deg);
}

/* 悬停效果 */
.flip-card-container:hover .flip-card-inner {
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

/* 翻转动画 */
@keyframes flip {
  0% {
    transform: perspective(1000px) rotateY(0deg);
  }
  100% {
    transform: perspective(1000px) rotateY(180deg);
  }
}

.flip-card-container.flipping .flip-card-inner {
  animation: flip 0.6s cubic-bezier(0.4, 0, 0.2, 1) forwards;
}

/* 响应式 */
@media (max-width: 640px) {
  .flip-card-container {
    transform: scale(0.9);
  }
}

/* 暗色主题优化 */
@media (prefers-color-scheme: dark) {
  .flip-card-front,
  .flip-card-back {
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  }
}
```

---

## 卡牌堆叠效果

```tsx
// frontend/src/components/card/CardStack.tsx
import { ReactNode, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface CardStackProps {
  cards: ReactNode[]
  className?: string
  onCardChange?: (index: number) => void
}

export function CardStack({ cards, className, onCardChange }: CardStackProps) {
  const [currentIndex, setCurrentIndex] = useState(0)
  const [direction, setDirection] = useState(0)

  const handleNext = () => {
    if (currentIndex < cards.length - 1) {
      setDirection(1)
      setCurrentIndex(currentIndex + 1)
      onCardChange?.(currentIndex + 1)
    }
  }

  const handlePrev = () => {
    if (currentIndex > 0) {
      setDirection(-1)
      setCurrentIndex(currentIndex - 1)
      onCardChange?.(currentIndex - 1)
    }
  }

  const variants = {
    enter: (direction: number) => ({
      x: direction > 0 ? 300 : -300,
      opacity: 0,
      scale: 0.8,
      rotateY: direction > 0 ? -30 : 30,
    }),
    center: {
      zIndex: 1,
      x: 0,
      opacity: 1,
      scale: 1,
      rotateY: 0,
    },
    exit: (direction: number) => ({
      zIndex: 0,
      x: direction < 0 ? 300 : -300,
      opacity: 0,
      scale: 0.8,
      rotateY: direction < 0 ? -30 : 30,
    }),
  }

  return (
    <div className={cn('card-stack-container', className)}>
      <div className="relative w-full h-[400px] perspective-1000">
        <AnimatePresence initial={false} custom={direction}>
          <motion.div
            key={currentIndex}
            custom={direction}
            variants={variants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={{
              x: { type: 'spring', stiffness: 300, damping: 30 },
              opacity: { duration: 0.2 },
              scale: { duration: 0.2 },
              rotateY: { duration: 0.4 },
            }}
            className="absolute w-full h-full flex items-center justify-center"
            style={{ transformStyle: 'preserve-3d' }}
          >
            {cards[currentIndex]}
          </motion.div>
        </AnimatePresence>

        {/* 背景卡片（堆叠效果） */}
        {currentIndex > 0 && (
          <div className="absolute inset-0 flex items-center justify-center">
            <motion.div
              key={`bg-${currentIndex - 1}`}
              initial={{ scale: 0.95, opacity: 0.5 }}
              animate={{ scale: 0.95, opacity: 0.3 }}
              className="w-[90%] h-[90%] bg-slate-200 dark:bg-slate-800 rounded-xl"
            />
          </div>
        )}
      </div>

      {/* 导航按钮 */}
      <div className="flex items-center justify-center gap-4 mt-6">
        <Button
          variant="outline"
          size="icon"
          onClick={handlePrev}
          disabled={currentIndex === 0}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>

        <div className="text-sm text-muted-foreground">
          {currentIndex + 1} / {cards.length}
        </div>

        <Button
          variant="outline"
          size="icon"
          onClick={handleNext}
          disabled={currentIndex === cards.length - 1}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
```

---

## 音效集成

```typescript
// frontend/src/lib/card/card-sounds.ts
import { playSound } from '@/lib/sounds/sound-manager'

export function playFlipSound() {
  playSound('card-flip', {
    volume: 0.3,
    speed: 1.0,
  })
}

export function playRevealSound() {
  playSound('card-reveal', {
    volume: 0.4,
    speed: 1.0,
  })
}

export function playLockSound() {
  playSound('card-lock', {
    volume: 0.2,
    speed: 1.2,
  })
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/card/FlipCard.tsx` | 创建 | 翻转卡牌核心组件 |
| `frontend/src/components/card/ClueCard.tsx` | 创建 | 线索卡组件 |
| `frontend/src/components/card/HandoutCard.tsx` | 创建 | 手递物组件 |
| `frontend/src/components/card/CardStack.tsx` | 创建 | 卡牌堆叠组件 |
| `frontend/src/components/card/flip-card.css` | 创建 | 翻转动画样式 |
| `frontend/src/lib/card/card-sounds.ts` | 创建 | 卡牌音效 |

---

## 验收标准

- [ ] 翻转动画流畅
- [ ] 3D 效果真实
- [ ] 正反面切换正确
- [ ] 响应式适配良好
- [ ] 音效反馈及时
- [ ] 性能优化到位

---

## 参考文档

- M5-004: 卡牌翻转功能
- Framer Motion 文档
- CSS 3D Transforms

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
