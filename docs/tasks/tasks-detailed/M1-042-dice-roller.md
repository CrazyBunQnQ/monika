# M1-042: 实现 DiceRoller 掷骰组件

**任务ID**: M1-040
**标题**: 实现 DiceRoller 掷骰组件
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-032

---

## 任务描述

实现可视化的掷骰组件，支持各种骰子类型、骰子表达式解析、结果显示等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-042-01 | 设计掷骰界面 | UI 设计 | 20min |
| M1-042-02 | 实现骰子选择器 | 骰子类型选择 | 20min |
| M1-042-03 | 实现表达式输入 | 表达式解析 | 25min |
| M1-042-04 | 实现掷骰动画 | 动画效果 | 30min |
| M1-042-05 | 实现结果显示 | 结果展示 | 20min |
| M1-042-06 | 实现历史记录 | 历史管理 | 15min |
| M1-042-07 | 实现快捷掷骰 | 常用表达式 | 10min |

---

## 掷骰组件

```tsx
// frontend/src/components/game/DiceRoller.tsx
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Dice1, Dice2, Dice3, Dice4, Dice5, Dice6 } from 'lucide-react'

interface DiceRollerProps {
  onRoll?: (expression: string, result: RollResult) => void
  secret?: boolean
}

interface RollResult {
  expression: string
  rolls: number[]
  total: number
  modifier: number
}

const DICE_TYPES = [
  { value: 'd4', faces: 4, icon: Dice4 },
  { value: 'd6', faces: 6, icon: Dice6 },
  { value: 'd8', faces: 8, icon: null },
  { value: 'd10', faces: 10, icon: null },
  { value: 'd12', faces: 12, icon: null },
  { value: 'd20', faces: 20, icon: null },
  { value: 'd100', faces: 100, icon: null },
]

const QUICK_ROLLS = [
  { label: 'd100', expression: '1d100' },
  { label: '2d6', expression: '2d6' },
  { label: 'd20', expression: '1d20' },
]

export function DiceRoller({ onRoll, secret }: DiceRollerProps) {
  const [expression, setExpression] = useState('1d100')
  const [rolling, setRolling] = useState(false)
  const [history, setHistory] = useState<RollResult[]>([])
  const [result, setResult] = useState<RollResult | null>(null)

  // 解析并执行掷骰
  const handleRoll = async () => {
    setRolling(true)

    try {
      // 调用后端 API
      const response = await fetch('/api/roll', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ expression, secret }),
      })

      if (!response.ok) throw new Error('掷骰失败')

      const data: RollResult = await response.json()

      setResult(data)
      setHistory([data, ...history].slice(0, 10))
      onRoll?.(expression, data)
    } catch (error) {
      console.error('Roll error:', error)
    } finally {
      setTimeout(() => setRolling(false), 500)
    }
  }

  // 添加快捷骰子
  const addDice = (dice: string) => {
    setExpression(prev => {
      if (prev === '1d100') return dice
      return `${prev}+${dice}`
    })
  }

  return (
    <div className="space-y-4">
      {/* 骰子选择器 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">骰子选择</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-7 gap-2">
            {DICE_TYPES.map(dice => (
              <Button
                key={dice.value}
                variant="outline"
                size="sm"
                onClick={() => addDice(dice.value)}
                className="flex flex-col items-center"
              >
                {dice.icon && <dice.icon className="h-4 w-4 mb-1" />}
                <span className="text-xs">{dice.value}</span>
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 表达式输入 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">掷骰表达式</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex space-x-2">
            <Input
              value={expression}
              onChange={(e) => setExpression(e.target.value)}
              placeholder="1d100"
              className="flex-1"
            />
            <Button
              onClick={handleRoll}
              disabled={rolling}
              className="min-w-[80px]"
            >
              {rolling ? '掷骰中...' : '掷骰'}
            </Button>
          </div>

          {/* 快捷掷骰 */}
          <div className="flex flex-wrap gap-2">
            <span className="text-sm text-muted-foreground">快捷:</span>
            {QUICK_ROLLS.map(quick => (
              <Badge
                key={quick.expression}
                variant="secondary"
                className="cursor-pointer hover:bg-primary hover:text-primary-foreground"
                onClick={() => {
                  setExpression(quick.expression)
                  handleRoll()
                }}
              >
                {quick.label}
              </Badge>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 结果显示 */}
      {result && (
        <Card className="border-primary">
          <CardHeader>
            <CardTitle className="text-base">结果</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-center space-y-2">
              <div className="text-3xl font-bold text-primary">
                {result.total}
              </div>
              {result.rolls.length > 1 && (
                <div className="text-sm text-muted-foreground">
                  掷出: [{result.rolls.join(', ')}]
                  {result.modifier !== 0 && (
                    <span>
                      {result.modifier > 0 ? '+' : ''}
                      {result.modifier}
                    </span>
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 历史记录 */}
      {history.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">历史记录</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {history.map((item, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between text-sm p-2 rounded hover:bg-muted cursor-pointer"
                  onClick={() => setExpression(item.expression)}
                >
                  <span className="font-mono">{item.expression}</span>
                  <Badge variant="outline">{item.total}</Badge>
                </div>
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

## 掷骰动画组件

```tsx
// frontend/src/components/game/DiceAnimation.tsx
import { useEffect, useState } from 'react'

interface DiceAnimationProps {
  faces: number
  duration?: number
  onComplete?: (result: number) => void
}

export function DiceAnimation({ faces, duration = 1000, onComplete }: DiceAnimationProps) {
  const [current, setCurrent] = useState(1)
  const [rolling, setRolling] = useState(true)

  useEffect(() => {
    const interval = setInterval(() => {
      setCurrent(Math.floor(Math.random() * faces) + 1)
    }, 50)

    const timeout = setTimeout(() => {
      clearInterval(interval)
      setRolling(false)
      const final = Math.floor(Math.random() * faces) + 1
      setCurrent(final)
      onComplete?.(final)
    }, duration)

    return () => {
      clearInterval(interval)
      clearTimeout(timeout)
    }
  }, [faces, duration, onComplete])

  return (
    <div className={`
      relative w-20 h-20 flex items-center justify-center
      rounded-lg text-2xl font-bold transition-all
      ${rolling ? 'animate-pulse' : ''}
    `}>
      {rolling ? (
        <div className="absolute inset-0 bg-primary/20 rounded-lg animate-spin" />
      ) : null}
      <span className="relative z-10">{current}</span>
    </div>
  )
}
```

---

## 骰子结果卡片

```tsx
// frontend/src/components/game/DiceResultCard.tsx
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

interface DiceResultCardProps {
  expression: string
  rolls: number[]
  total: number
  modifier?: number
  timestamp: Date
}

export function DiceResultCard({
  expression,
  rolls,
  total,
  modifier = 0,
  timestamp,
}: DiceResultCardProps) {
  return (
    <Card className="border-l-4 border-l-primary">
      <CardContent className="p-3">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center space-x-2 mb-1">
              <span className="font-mono text-sm">{expression}</span>
              <Badge variant="secondary">{timestamp.toLocaleTimeString()}</Badge>
            </div>

            {rolls.length > 0 && (
              <div className="text-sm text-muted-foreground mb-2">
                掷出: [{rolls.map((r, i) => (
                  <span key={i} className="font-mono">{r}</span>
                )).join(', ')}]
                {modifier !== 0 && (
                  <span className="font-mono ml-1">
                    {modifier > 0 ? '+' : ''}{modifier}
                  </span>
                )}
              </div>
            )}

            <div className="text-2xl font-bold text-primary">
              {total}
            </div>
          </div>
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
| `frontend/src/components/game/DiceRoller.tsx` | 创建 | 掷骰组件主文件 |
| `frontend/src/components/game/DiceAnimation.tsx` | 创建 | 掷骰动画 |
| `frontend/src/components/game/DiceResultCard.tsx` | 创建 | 结果卡片 |
| `frontend/src/types/dice.ts` | 创建 | 骰子类型定义 |

---

## 验收标准

- [ ] 骰子选择功能正常
- [ ] 表达式解析正确
- [ ] 掷骰动画流畅
- [ ] 结果显示准确
- [ ] 历史记录完整
- [ ] 快捷掷骰可用

---

## 参考文档

- M1-057: d100 随机数生成
- M1-058: 成功判定
- M1-032: GameConsole 布局

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
