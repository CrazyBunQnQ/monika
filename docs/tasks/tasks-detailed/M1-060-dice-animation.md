# M1-060: 实现掷骰动画效果

**任务ID**: M1-060
**标题**: 实现掷骰动画效果
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-042

---

## 任务描述

实现掷骰子的 3D 动画效果，包括骰子滚动、碰撞、停止等视觉效果，增强游戏体验。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-060-01 | 选择动画库 | Animation Library | 15min |
| M1-060-02 | 实现 3D 骰子模型 | Dice Model | 35min |
| M1-060-03 | 实现滚动动画 | Roll Animation | 30min |
| M1-060-04 | 实现碰撞效果 | Collision Effect | 25min |
| M1-060-05 | 实现结果展示 | Result Display | 20min |
| M1-060-06 | 优化性能 | Performance | 20min |
| M1-060-07 | 编写动画测试 | 测试覆盖 | 10min |

---

## 技术选型

推荐使用 **Three.js** + **Cannon.js** 实现物理效果：

```bash
npm install three @react-three/fiber @react-three/drei cannon
```

---

## 3D 骰子组件

```tsx
// frontend/src/components/game/Dice3D.tsx
import { useRef, useState } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, PerspectiveCamera, Environment } from '@react-three/drei'
import { Physics, useBox } from '@react-three/cannon'
import * as THREE from 'three'

interface DiceProps {
  value: number
  position: [number, number, number]
  onRollComplete: (value: number) => void
}

function DiceModel({ value, position, onRollComplete }: DiceProps) {
  const meshRef = useRef<THREE.Mesh>(null)
  const [isRolling, setIsRolling] = useState(false)
  const [rotation, setRotation] = useState([0, 0, 0])

  // 骰子纹理（点数）
  const diceTexture = useRef(createDiceTexture())

  useFrame((state, delta) => {
    if (isRolling && meshRef.current) {
      // 旋转动画
      meshRef.current.rotation.x += delta * 5
      meshRef.current.rotation.y += delta * 3
      meshRef.current.rotation.z += delta * 4

      // 停止条件
      if (state.clock.elapsedTime > 2) {
        setIsRolling(false)
        onRollComplete(value)

        // 设置最终朝向
        const finalRotation = getValueRotation(value)
        setRotation(finalRotation)
      }
    }
  })

  return (
    <mesh
      ref={meshRef}
      position={position}
      rotation={rotation}
    >
      <boxGeometry args={[1, 1, 1]} />
      <meshStandardMaterial
        map={diceTexture.current}
        color="#ffffff"
        roughness={0.3}
        metalness={0.1}
      />
    </mesh>
  )
}

// 创建骰子纹理
function createDiceTexture() {
  const canvas = document.createElement('canvas')
  canvas.width = 512
  canvas.height = 512
  const ctx = canvas.getContext('2d')!

  // 白色背景
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, 512, 512)

  // 绘制边框
  ctx.strokeStyle = '#000000'
  ctx.lineWidth = 4
  ctx.strokeRect(0, 0, 512, 512)

  return new THREE.CanvasTexture(canvas)
}

// 根据点数获取最终旋转角度
function getValueRotation(value: number): [number, number, number] {
  const rotations: Record<number, [number, number, number]> = {
    1: [0, 0, 0],
    2: [0, Math.PI / 2, 0],
    3: [0, Math.PI, 0],
    4: [0, -Math.PI / 2, 0],
    5: [Math.PI / 2, 0, 0],
    6: [-Math.PI / 2, 0, 0],
  }
  return rotations[value] || [0, 0, 0]
}

// 骰子场景
export function DiceScene({ diceCount, onRollComplete }: { diceCount: number; onRollComplete: (results: number[]) => void }) {
  const [results, setResults] = useState<number[]>([])
  const [rolling, setRolling] = useState(false)

  const handleRoll = () => {
    setRolling(true)
    const newResults = Array.from({ length: diceCount }, () => Math.floor(Math.random() * 6) + 1)
    setResults(newResults)
  }

  const handleDiceComplete = (value: number, index: number) => {
    if (results.every((r, i) => i === index || r !== 0)) {
      setRolling(false)
      onRollComplete(results)
    }
  }

  return (
    <div className="w-full h-64 bg-gradient-to-b from-gray-900 to-gray-800 rounded-lg overflow-hidden">
      <Canvas shadows>
        <PerspectiveCamera makeDefault position={[0, 5, 5]} />
        <OrbitControls enablePan={false} enableZoom={false} />
        <Environment preset="city" />

        {/* 灯光 */}
        <ambientLight intensity={0.5} />
        <directionalLight position={[5, 5, 5]} castShadow intensity={1} />
        <pointLight position={[-5, 5, -5]} intensity={0.5} color="#ff6b6b" />

        {/* 地面 */}
        <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -1, 0]} receiveShadow>
          <planeGeometry args={[10, 10]} />
          <meshStandardMaterial color="#2d3748" roughness={0.8} />
        </mesh>

        {/* 骰子 */}
        <Physics gravity={[0, -30, 0]}>
          {results.map((value, index) => (
            <DiceModel
              key={index}
              value={value}
              position={[index * 1.5 - (diceCount - 1) * 0.75, 3, 0]}
              onRollComplete={(v) => handleDiceComplete(v, index)}
            />
          ))}
        </Physics>
      </Canvas>

      {/* 掷骰按钮 */}
      {!rolling && (
        <button
          onClick={handleRoll}
          className="absolute bottom-4 left-1/2 -translate-x-1/2 px-6 py-2 bg-primary text-primary-foreground rounded-lg font-medium hover:bg-primary/90 transition-colors"
        >
          掷骰
        </button>
      )}
    </div>
  )
}
```

---

## 简化版动画组件

```tsx
// frontend/src/components/game/DiceAnimation.tsx
import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Dice1, Dice2, Dice3, Dice4, Dice5, Dice6 } from 'lucide-react'

interface DiceAnimationProps {
  count: number
  results?: number[]
  duration?: number
  onComplete?: (results: number[]) => void
}

export function DiceAnimation({
  count,
  results: externalResults,
  duration = 2000,
  onComplete,
}: DiceAnimationProps) {
  const [rolling, setRolling] = useState(true)
  const [currentValues, setCurrentValues] = useState<number[]>([])
  const [finalValues, setFinalValues] = useState<number[]>([])

  useEffect(() => {
    if (externalResults) {
      setFinalValues(externalResults)
      setCurrentValues(externalResults)
      setRolling(false)
    } else {
      // 生成随机结果
      const results = Array.from({ length: count }, () => Math.floor(Math.random() * 6) + 1)
      setFinalValues(results)

      // 滚动动画
      const interval = setInterval(() => {
        setCurrentValues(Array.from({ length: count }, () => Math.floor(Math.random() * 6) + 1))
      }, 100)

      setTimeout(() => {
        clearInterval(interval)
        setCurrentValues(results)
        setRolling(false)
        onComplete?.(results)
      }, duration)

      return () => clearInterval(interval)
    }
  }, [externalResults, count, duration, onComplete])

  const DiceIcon = ({ value }: { value: number }) => {
    const icons = {
      1: Dice1,
      2: Dice2,
      3: Dice3,
      4: Dice4,
      5: Dice5,
      6: Dice6,
    }
    const Icon = icons[value as keyof typeof icons] || Dice1
    return <Icon className="w-full h-full" />
  }

  return (
    <div className="flex items-center justify-center gap-4 py-8">
      <AnimatePresence>
        {currentValues.map((value, index) => (
          <motion.div
            key={index}
            initial={{ scale: 0, rotate: -180 }}
            animate={{
              scale: rolling ? [1, 1.2, 1] : 1,
              rotate: rolling ? [0, 360, 0] : 0,
              y: rolling ? [0, -20, 0] : 0,
            }}
            transition={{
              duration: rolling ? 0.5 : 0.3,
              repeat: rolling ? Infinity : 0,
              ease: "easeInOut"
            }}
            className="w-16 h-16"
          >
            <div className="w-full h-full bg-gradient-to-br from-amber-100 to-amber-200 dark:from-amber-900 dark:to-amber-800 rounded-xl shadow-lg border-2 border-amber-300 dark:border-amber-700 flex items-center justify-center p-2">
              <DiceIcon value={value} />
            </div>
          </motion.div>
        ))}
      </AnimatePresence>

      {/* 结果显示 */}
      {!rolling && finalValues.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="ml-8"
        >
          <div className="text-2xl font-bold">
            总计: {finalValues.reduce((a, b) => a + b, 0)}
          </div>
          <div className="text-sm text-muted-foreground">
            {finalValues.join(' + ')}
          </div>
        </motion.div>
      )}
    </div>
  )
}
```

---

## 骰子结果弹窗

```tsx
// frontend/src/components/game/DiceResultDialog.tsx
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { DiceAnimation } from './DiceAnimation'
import { Button } from '@/components/ui/button'
import { Copy, Check } from 'lucide-react'
import { useState } from 'react'

interface DiceResultDialogProps {
  open: boolean
  onClose: () => void
  count: number
  results: number[]
  description?: string
}

export function DiceResultDialog({
  open,
  onClose,
  count,
  results,
  description,
}: DiceResultDialogProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    const text = `🎲 掷骰结果：${results.join(', ')} (总计: ${results.reduce((a, b) => a + b, 0)})`
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>掷骰结果</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {description && (
            <p className="text-sm text-muted-foreground text-center">
              {description}
            </p>
          )}

          {/* 骰子动画 */}
          <DiceAnimation count={count} results={results} />

          {/* 结果详情 */}
          <div className="border rounded-lg p-4 bg-muted/30">
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div>
                <span className="text-muted-foreground">骰子数:</span>
                <span className="ml-2 font-medium">{count}</span>
              </div>
              <div>
                <span className="text-muted-foreground">总计:</span>
                <span className="ml-2 font-medium text-lg">
                  {results.reduce((a, b) => a + b, 0)}
                </span>
              </div>
            </div>
            <div className="mt-2">
              <span className="text-muted-foreground text-sm">详情:</span>
              <div className="flex gap-1 mt-1">
                {results.map((r, i) => (
                  <span key={i} className="px-2 py-1 bg-background rounded text-sm">
                    {r}
                  </span>
                ))}
              </div>
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="flex-1"
              onClick={handleCopy}
            >
              {copied ? (
                <>
                  <Check className="h-4 w-4 mr-1" />
                  已复制
                </>
              ) : (
                <>
                  <Copy className="h-4 w-4 mr-1" />
                  复制结果
                </>
              )}
            </Button>
            <Button className="flex-1" onClick={onClose}>
              关闭
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
```

---

## CSS 动画版本（备选）

```css
/* frontend/src/styles/animations/dice.css */
@keyframes diceRoll {
  0% {
    transform: rotateX(0deg) rotateY(0deg) scale(1);
  }
  25% {
    transform: rotateX(90deg) rotateY(90deg) scale(1.2);
  }
  50% {
    transform: rotateX(180deg) rotateY(180deg) scale(1);
  }
  75% {
    transform: rotateX(270deg) rotateY(270deg) scale(1.1);
  }
  100% {
    transform: rotateX(360deg) rotateY(360deg) scale(1);
  }
}

@keyframes diceBounce {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-20px);
  }
}

.dice-roll {
  animation: diceRoll 0.6s ease-in-out infinite,
             diceBounce 0.3s ease-in-out infinite;
}

.dice-value-1 { --rotate-x: 0deg; --rotate-y: 0deg; }
.dice-value-2 { --rotate-x: 0deg; --rotate-y: 90deg; }
.dice-value-3 { --rotate-x: 0deg; --rotate-y: 180deg; }
.dice-value-4 { --rotate-x: 0deg; --rotate-y: -90deg; }
.dice-value-5 { --rotate-x: 90deg; --rotate-y: 0deg; }
.dice-value-6 { --rotate-x: -90deg; --rotate-y: 0deg; }
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/Dice3D.tsx` | 创建 | 3D 骰子组件 |
| `frontend/src/components/game/DiceAnimation.tsx` | 创建 | 简化动画组件 |
| `frontend/src/components/game/DiceResultDialog.tsx` | 创建 | 结果弹窗组件 |
| `frontend/src/styles/animations/dice.css` | 创建 | CSS 动画 |

---

## 验收标准

- [ ] 3D 动画流畅
- [ ] 滚动效果自然
- [ ] 结果显示准确
- [ ] 性能表现良好
- [ ] 响应式适配
- [ ] 键盘操作支持

---

## 参考文档

- M1-042: DiceRoller 掷骰组件
- Three.js 文档
- Framer Motion 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
