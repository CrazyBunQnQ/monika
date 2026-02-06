# M5-002: 实现 SAN 检定 UI 组件

**任务ID**: M5-002
**标题**: 实现 SAN 检定 UI 组件
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-040, M5-001

---

## 任务描述

实现 SAN（理智值）检定的前端 UI 组件，包括检定触发、损耗显示、疯狂症状展示等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-002-01 | 设计 SAN 检定界面 | UI 设计 | 20min |
| M5-002-02 | 实现 SAN 检定组件 | 检定 UI | 30min |
| M5-002-03 | 实现损耗选择器 | 损耗输入 | 25min |
| M5-002-04 | 实现结果显示 | 结果展示 | 30min |
| M5-002-05 | 实现疯狂症状显示 | 疯狂展示 | 25min |
| M5-002-06 | 实现 SAN 历史记录 | 历史追踪 | 15min |
| M5-002-07 | 编写组件测试 | 测试覆盖 | 15min |

---

## SAN 检定组件

```tsx
// frontend/src/components/game/SanityCheck.tsx
import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Brain, AlertTriangle } from 'lucide-react'

interface SanityCheckProps {
  currentSan: number
  maxSan: number
  onSanChange?: (newSan: number) => void
}

interface SanCheckResult {
  passed: boolean
  loss: number
  newSan: number
  madness?: string
}

const SAN_LOSS_PRESETS = [
  { label: '轻微', loss: '1d4' },
  { label: '中等', loss: '1d6' },
  { label: '严重', loss: '1d10' },
  { label: '致命', loss: '2d10' },
]

export function SanityCheck({ currentSan, maxSan, onSanChange }: SanityCheckProps) {
  const { user } = useAuth()
  const [loss, setLoss] = useState('1d6')
  const [customLoss, setCustomLoss] = useState('')
  const [checking, setChecking] = useState(false)
  const [result, setResult] = useState<SanCheckResult | null>(null)
  const [history, setHistory] = useState<SanCheckResult[]>([])

  const isKp = user?.role === 'kp'

  // 执行 SAN 检定
  const handleCheck = async () => {
    setChecking(true)

    try {
      const response = await fetch('/api/san/check', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          loss: loss || customLoss,
          current_san: currentSan,
        }),
      })

      if (!response.ok) throw new Error('SAN 检定失败')

      const data: SanCheckResult = await response.json()

      setResult(data)
      setHistory([data, ...history].slice(0, 10))
      onSanChange?.(data.newSan)
    } catch (error) {
      console.error('SAN check error:', error)
    } finally {
      setChecking(false)
    }
  }

  // SAN 值状态
  const getSanStatus = () => {
    const ratio = currentSan / maxSan
    if (ratio > 0.5) return { label: '正常', color: 'text-green-500' }
    if (ratio > 0.2) return { label: '紧张', color: 'text-yellow-500' }
    return { label: '危险', color: 'text-red-500' }
  }

  const sanStatus = getSanStatus()

  return (
    <div className="space-y-4">
      {/* 当前 SAN 状态 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center">
            <Brain className="h-4 w-4 mr-2 text-purple-500" />
            理智值 (SAN)
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center space-y-2">
            <div className="text-sm text-muted-foreground">当前值</div>
            <div className="text-3xl font-bold">
              {currentSan}
              <span className="text-lg text-muted-foreground"> / {maxSan}</span>
            </div>
            <Badge className={sanStatus.color}>{sanStatus.label}</Badge>
          </div>
        </CardContent>
      </Card>

      {/* SAN 检定 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">SAN 检定</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* 预设损耗 */}
          <div className="space-y-2">
            <Label>SAN 损耗</Label>
            <div className="grid grid-cols-4 gap-2">
              {SAN_LOSS_PRESETS.map(preset => (
                <Badge
                  key={preset.label}
                  variant={loss === preset.loss ? 'default' : 'outline'}
                  className="cursor-pointer justify-center py-2"
                  onClick={() => {
                    setLoss(preset.loss)
                    setCustomLoss('')
                  }}
                >
                  {preset.label}
                  <span className="ml-1 text-xs opacity-70">{preset.loss}</span>
                </Badge>
              ))}
            </div>
          </div>

          {/* 自定义损耗 */}
          <div className="space-y-2">
            <Label>自定义损耗</Label>
            <Input
              placeholder="如: 1d6/1d20, 5, 2d6"
              value={customLoss}
              onChange={(e) => {
                setCustomLoss(e.target.value)
                if (e.target.value) setLoss('')
              }}
            />
          </div>

          {/* 检定描述（KP 可见） */}
          {isKp && (
            <div className="space-y-2">
              <Label>触发原因（内部记录）</Label>
              <Input
                placeholder="如: 见到尸体, 恐怖仪式"
                type="text"
              />
            </div>
          )}

          {/* 执行检定 */}
          <Button
            className="w-full"
            onClick={handleCheck}
            disabled={checking}
          >
            {checking ? '检定中...' : '执行 SAN 检定'}
          </Button>
        </CardContent>
      </Card>

      {/* 检定结果 */}
      {result && (
        <Card className={`border-l-4 ${
          result.passed ? 'border-l-green-500' : 'border-l-red-500'
        }`}>
          <CardContent className="p-4">
            <div className="text-center space-y-2">
              <div className="flex items-center justify-center space-x-2">
                {result.passed ? (
                  <Badge className="bg-green-500">通过</Badge>
                ) : (
                  <Badge className="bg-red-500">失败</Badge>
                )}
              </div>

              <div className="text-2xl font-bold">
                -{result.loss} SAN
              </div>

              <div className="text-sm">
                {currentSan} → {result.newSan}
              </div>

              {result.madness && (
                <Alert variant="destructive">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    <div className="font-bold">陷入疯狂！</div>
                    <div className="text-sm">{result.madness}</div>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 历史记录 */}
      {history.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">检定历史</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {history.map((item, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between text-sm p-2 rounded bg-muted"
                >
                  <span>
                    {item.passed ? '✅' : '❌'} SAN -{item.loss}
                  </span>
                  <span>{item.newSan}/{maxSan}</span>
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

## 疯狂症状显示组件

```tsx
// frontend/src/components/game/MadnessDisplay.tsx
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { AlertTriangle, Ghost } from 'lucide-react'

interface Madness {
  type: 'temporary' | 'indefinite' | 'total'
  name: string
  description: string
  duration?: string
}

interface MadnessDisplayProps {
  madness: Madness[]
}

export function MadnessDisplay({ madness }: MadnessDisplayProps) {
  if (madness.length === 0) {
    return (
      <Alert>
        <Ghost className="h-4 w-4" />
        <AlertDescription>
          目前没有疯狂症状
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-3">
      {madness.map((m, index) => (
        <Alert key={index} variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <div className="space-y-1">
              <div className="font-bold">{m.name}</div>
              <div className="text-sm">{m.description}</div>
              {m.duration && (
                <div className="text-xs text-muted-foreground">
                  持续时间: {m.duration}
                </div>
              )}
            </div>
          </AlertDescription>
        </Alert>
      ))}
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/SanityCheck.tsx` | 创建 | SAN 检定组件 |
| `frontend/src/components/game/MadnessDisplay.tsx` | 创建 | 疯狂显示组件 |
| `frontend/src/types/sanity.ts` | 创建 | SAN 类型定义 |

---

## 验收标准

- [ ] SAN 检定功能正常
- [ ] 损耗输入有效
- [ ] 结果显示准确
- [ ] 疯狂症状清晰
- [ ] 历史记录完整
- [ ] KP 功能可用

---

## 参考文档

- M1-040: SAN 值系统
- M5-001: SAN 检定数据结构
- M1-050: 疯狂机制

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
