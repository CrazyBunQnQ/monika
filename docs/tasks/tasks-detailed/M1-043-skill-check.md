# M1-043: 实现 SkillCheck 技能检定组件

**任务ID**: M1-041
**标题**: 实现 SkillCheck 技能检定组件
**类型**: frontend (前端开发)
**预估工时**: 2h
**依赖**: M1-042

---

## 任务描述

实现技能检定组件，支持技能选择、修正值输入、奖励/惩罚骰、暗骰等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-043-01 | 设计检定界面 | UI 设计 | 20min |
| M1-043-02 | 实现技能选择器 | 技能列表 | 25min |
| M1-043-03 | 实现修正值输入 | 修正控制 | 15min |
| M1-043-04 | 实现奖励/惩罚骰 | 骰子控制 | 20min |
| M1-043-05 | 实现暗骰功能 | 秘密检定 | 15min |
| M1-043-06 | 实现结果显示 | 成功等级 | 25min |
| M1-043-07 | 实现快速检定 | 快捷按钮 | 10min |

---

## 技能检定组件

```tsx
// frontend/src/components/game/SkillCheck.tsx
import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { useCharacter } from '@/hooks/useCharacter'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Eye, EyeOff } from 'lucide-react'
import type { Character } from '@/types/character'

interface SkillCheckProps {
  character: Character
  onCheck?: (result: CheckResult) => void
}

interface CheckResult {
  skill: string
  target: number
  rolled: number
  successLevel: string
  modifier: number
  bonusDice?: number
  penaltyDice?: number
  secret: boolean
}

const SUCCESS_LEVELS = ['大成功', '极难成功', '困难成功', '成功', '失败', '大失败']

const COMMON_SKILLS = [
  '侦查', '聆听', '图书馆使用', '心理学',
  '说服', '恐吓', '躲藏', '潜行',
]

export function SkillCheck({ character, onCheck }: SkillCheckProps) {
  const { user } = useAuth()
  const { skills } = useCharacter(character.id)
  const [selectedSkill, setSelectedSkill] = useState('')
  const [modifier, setModifier] = useState(0)
  const [bonusDice, setBonusDice] = useState(0)
  const [penaltyDice, setPenaltyDice] = useState(0)
  const [secret, setSecret] = useState(false)
  const [checking, setChecking] = useState(false)
  const [result, setResult] = useState<CheckResult | null>(null)

  // 获取技能值
  const getSkillValue = (skillName: string): number => {
    const skill = skills.find(s => s.name === skillName)
    return skill?.value || 0
  }

  // 执行检定
  const handleCheck = async () => {
    if (!selectedSkill) return

    setChecking(true)

    try {
      const target = getSkillValue(selectedSkill) + modifier

      const response = await fetch('/api/check', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          skill: selectedSkill,
          target,
          bonus_dice: bonusDice,
          penalty_dice: penaltyDice,
          secret,
        }),
      })

      if (!response.ok) throw new Error('检定失败')

      const data: CheckResult = await response.json()

      setResult(data)
      onCheck?.(data)
    } catch (error) {
      console.error('Check error:', error)
    } finally {
      setChecking(false)
    }
  }

  // 快速检定
  const quickCheck = (skillName: string) => {
    setSelectedSkill(skillName)
    setModifier(0)
    setBonusDice(0)
    setPenaltyDice(0)
    // 自动执行检定
    setTimeout(() => handleCheck(), 100)
  }

  // 是否是 KP
  const isKp = user?.role === 'kp'

  return (
    <div className="space-y-4">
      {/* 技能选择 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">技能检定</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* 技能选择器 */}
          <div className="space-y-2">
            <Label>选择技能</Label>
            <Select value={selectedSkill} onValueChange={setSelectedSkill}>
              <SelectTrigger>
                <SelectValue placeholder="选择要检定的技能" />
              </SelectTrigger>
              <SelectContent>
                {skills.map(skill => (
                  <SelectItem key={skill.id} value={skill.name}>
                    {skill.name} ({skill.value})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* 快捷技能 */}
          <div className="space-y-2">
            <Label>常用技能</Label>
            <div className="flex flex-wrap gap-2">
              {COMMON_SKILLS.map(skill => {
                const value = getSkillValue(skill)
                return (
                  <Badge
                    key={skill}
                    variant={selectedSkill === skill ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setSelectedSkill(skill)}
                  >
                    {skill} ({value})
                  </Badge>
                )
              })}
            </div>
          </div>

          {/* 目标值显示 */}
          {selectedSkill && (
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-sm text-muted-foreground">目标值</div>
              <div className="text-2xl font-bold">
                {getSkillValue(selectedSkill)}
                {modifier !== 0 && (
                  <span className={modifier > 0 ? 'text-green-500' : 'text-red-500'}>
                    {' '}{modifier > 0 ? '+' : ''}{modifier}
                  </span>
                )}
                {' = '}
                {getSkillValue(selectedSkill) + modifier}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 检定选项 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">检定选项</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* 修正值 */}
          <div className="space-y-2">
            <Label>修正值</Label>
            <div className="flex items-center space-x-2">
              <Button
                size="sm"
                variant="outline"
                onClick={() => setModifier(m => m - 5)}
              >
                -5
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setModifier(m => m - 1)}
              >
                -1
              </Button>
              <Input
                type="number"
                value={modifier}
                onChange={(e) => setModifier(parseInt(e.target.value) || 0)}
                className="w-20 text-center"
              />
              <Button
                size="sm"
                variant="outline"
                onClick={() => setModifier(m => m + 1)}
              >
                +1
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setModifier(m => m + 5)}
              >
                +5
              </Button>
            </div>
          </div>

          {/* 奖励骰 */}
          <div className="space-y-2">
            <Label>奖励骰</Label>
            <div className="flex space-x-2">
              {[0, 1, 2].map(n => (
                <Badge
                  key={n}
                  variant={bonusDice === n ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setBonusDice(n)}
                >
                  +b{n}
                </Badge>
              ))}
            </div>
          </div>

          {/* 惩罚骰 */}
          <div className="space-y-2">
            <Label>惩罚骰</Label>
            <div className="flex space-x-2">
              {[0, 1, 2].map(n => (
                <Badge
                  key={n}
                  variant={penaltyDice === n ? 'destructive' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setPenaltyDice(n)}
                >
                  -p{n}
                </Badge>
              ))}
            </div>
          </div>

          {/* 暗骰选项 */}
          {isKp && (
            <div className="flex items-center space-x-2">
              <Checkbox
                id="secret"
                checked={secret}
                onCheckedChange={(checked) => setSecret(checked as boolean)}
              />
              <Label htmlFor="secret" className="flex items-center cursor-pointer">
                {secret ? <EyeOff className="h-4 w-4 mr-1" /> : <Eye className="h-4 w-4 mr-1" />}
                暗骰（仅 KP 可见）
              </Label>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 执行检定 */}
      <Button
        className="w-full"
        size="lg"
        onClick={handleCheck}
        disabled={!selectedSkill || checking}
      >
        {checking ? '检定中...' : '执行检定'}
      </Button>

      {/* 结果显示 */}
      {result && (
        <Card className={`border-l-4 ${
          result.successLevel === '大成功' ? 'border-l-emerald-500' :
          result.successLevel === '大失败' ? 'border-l-red-500' :
          ['极难成功', '困难成功'].includes(result.successLevel) ? 'border-l-green-500' :
          result.successLevel === '成功' ? 'border-l-yellow-500' :
          'border-l-orange-500'
        }`}>
          <CardContent className="p-4">
            <div className="text-center space-y-2">
              <div className="text-sm text-muted-foreground">
                {result.skill} 检定
              </div>
              <div className="text-3xl font-bold">
                {result.rolled}
              </div>
              <div className="text-sm">
                目标: {result.target}
              </div>
              <Badge className={
                result.successLevel === '大成功' ? 'bg-emerald-500' :
                result.successLevel === '大失败' ? 'bg-red-500' :
                ['极难成功', '困难成功'].includes(result.successLevel) ? 'bg-green-500' :
                result.successLevel === '成功' ? 'bg-yellow-500' :
                'bg-orange-500'
              }>
                {result.successLevel}
              </Badge>
              {secret && (
                <div className="text-xs text-muted-foreground mt-2">
                  <EyeOff className="h-3 w-3 inline mr-1" />
                  暗骰结果
                </div>
              )}
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
| `frontend/src/components/game/SkillCheck.tsx` | 创建 | 技能检定组件 |
| `frontend/src/hooks/useCharacter.ts` | 创建 | 角色数据 Hook |
| `frontend/src/types/check.ts` | 创建 | 检定类型定义 |

---

## 验收标准

- [ ] 技能选择功能正常
- [ ] 修正值输入有效
- [ ] 奖励/惩罚骰正确
- [ ] 暗骰功能可用
- [ ] 结果显示准确
- [ ] 成功等级判定正确

---

## 参考文档

- M1-058: 大成功/大失败判定
- M1-059: 奖励骰逻辑
- M1-060: 惩罚骰逻辑
- M1-003: 角色卡数据模型

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
