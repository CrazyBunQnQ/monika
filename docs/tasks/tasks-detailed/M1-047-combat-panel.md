# M1-047: 实现 CombatPanel 战斗面板组件

**任务ID**: M1-047
**标题**: 实现 CombatPanel 战斗面板组件
**类型**: frontend (前端开发)
**预估工时**: 2.5h
**依赖**: M1-020

---

## 任务描述

实现战斗面板组件，用于管理战斗流程、轮次顺序、伤害计算等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-047-01 | 设计战斗面板布局 | UI 设计 | 25min |
| M1-047-02 | 实现轮次管理 | Turn Order | 35min |
| M1-047-03 | 实现战斗状态显示 | Combat State | 30min |
| M1-047-04 | 实现伤害输入 | Damage Input | 30min |
| M1-047-05 | 实现战斗日志 | Combat Log | 25min |
| M1-047-06 | 实现战斗控制 | Control | 20min |
| M1-047-07 | 编写面板测试 | 测试覆盖 | 15min |

---

## 战斗面板组件

```tsx
// frontend/src/components/game/CombatPanel.tsx
import { useState, useEffect } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Sword, Heart, Play, Pause, SkipForward } from 'lucide-react'

interface Combatant {
  id: string
  name: string
  type: 'player' | 'npc' | 'enemy'
  hp: number
  max_hp: number
  mp: number
  max_mp: number
  initiative: number
  is_current: boolean
}

interface CombatPanelProps {
  combatants: Combatant[]
  currentTurn: number
  onAttack?: (attacker: string, defender: string) => void
  onDamage?: (target: string, damage: number) => void
  onNextTurn?: () => void
  onEndCombat?: () => void
}

export function CombatPanel({
  combatants,
  currentTurn,
  onAttack,
  onDamage,
  onNextTurn,
  onEndCombat,
}: CombatPanelProps) {
  const { user } = useAuth()
  const [selectedTarget, setSelectedTarget] = useState<string | null>(null)
  const [damageInput, setDamageInput] = useState('')
  const [combatLog, setCombatLog] = useState<string[]>([])

  const isKp = user?.role === 'kp'
  const currentCombatant = combatants[currentTurn]

  // 按先攻排序
  const sortedCombatants = [...combatants].sort((a, b) => b.initiative - a.initiative)

  const handleNextTurn = () => {
    setCombatLog([...combatLog, `${currentCombatant.name} 的回合结束`])
    onNextTurn?.()
    setSelectedTarget(null)
  }

  const handleDamage = () => {
    if (!selectedTarget || !damageInput) return

    const damage = parseInt(damageInput)
    if (isNaN(damage)) return

    onDamage?.(selectedTarget, damage)
    setCombatLog([...combatLog, `对 ${selectedTarget} 造成 ${damage} 点伤害`])

    setDamageInput('')
  }

  const getHpColor = (current: number, max: number) => {
    const ratio = current / max
    if (ratio > 0.5) return 'bg-green-500'
    if (ratio > 0.25) return 'bg-yellow-500'
    return 'bg-red-500'
  }

  return (
    <div className="space-y-4">
      {/* 当前回合 */}
      <Card className="border-primary">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">
              战斗中 - 第 {currentTurn + 1} 回合
            </CardTitle>
            <Badge variant="outline">{currentCombatant?.name} 的回合</Badge>
          </div>
        </CardHeader>
      </Card>

      {/* 轮次顺序 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">轮次顺序</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {sortedCombatants.map((combatant, index) => (
              <div
                key={combatant.id}
                className={`flex items-center justify-between p-2 rounded ${
                  combatant.is_current ? 'bg-primary/10 border border-primary' : ''
                }`}
              >
                <div className="flex items-center space-x-3">
                  <span className="text-sm font-mono w-6">
                    {index + 1}
                  </span>
                  <span className="font-medium">{combatant.name}</span>
                  <Badge variant={
                    combatant.type === 'player' ? 'default' :
                    combatant.type === 'npc' ? 'secondary' : 'destructive'
                  }>
                    {combatant.type}
                  </Badge>
                  <Badge variant="outline">
                    DEX {combatant.initiative}
                  </Badge>
                </div>

                <div className="flex items-center space-x-2">
                  {/* HP */}
                  <div className="flex items-center space-x-1">
                    <Heart className="h-3 w-3" />
                    <Progress
                      value={(combatant.hp / combatant.max_hp) * 100}
                      className={getHpColor(combatant.hp, combatant.max_hp)}
                    />
                    <span className="text-xs w-12 text-center">
                      {combatant.hp}/{combatant.max_hp}
                    </span>
                  </div>

                  {/* MP */}
                  <div className="flex items-center space-x-1">
                    <span className="text-xs text-blue-500">MP</span>
                    <Progress
                      value={(combatant.mp / combatant.max_mp) * 100}
                      className="bg-blue-500 w-16"
                    />
                    <span className="text-xs w-12 text-center">
                      {combatant.mp}/{combatant.max_mp}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 操作面板（仅 KP） */}
      {isKp && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">战斗操作</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* 选择目标 */}
            <div className="space-y-2">
              <label className="text-sm font-medium">选择目标</label>
              <div className="grid grid-cols-2 gap-2">
                {combatants.map((combatant) => (
                  <Button
                    key={combatant.id}
                    variant={selectedTarget === combatant.id ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setSelectedTarget(combatant.id)}
                  >
                    {combatant.name}
                  </Button>
                ))}
              </div>
            </div>

            {/* 造成伤害 */}
            {selectedTarget && (
              <div className="space-y-2">
                <label className="text-sm font-medium">伤害点数</label>
                <div className="flex space-x-2">
                  <Input
                    type="number"
                    value={damageInput}
                    onChange={(e) => setDamageInput(e.target.value)}
                    placeholder="输入伤害值"
                    className="flex-1"
                  />
                  <Button onClick={handleDamage}>
                    确认
                  </Button>
                </div>
              </div>
            )}

            {/* 快捷操作 */}
            <div className="grid grid-cols-2 gap-2">
              <Button
                variant="outline"
                onClick={() => {/* 下一回合 */}}
              >
                <SkipForward className="h-4 w-4 mr-1" />
                跳过回合
              </Button>
              <Button
                variant="destructive"
                onClick={() => {
                  if (confirm('确定要结束战斗吗？')) {
                    onEndCombat?.()
                  }
                }}
              >
                结束战斗
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 战斗日志 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">战斗日志</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-1 max-h-48 overflow-y-auto">
            {combatLog.length === 0 ? (
              <div className="text-sm text-muted-foreground text-center py-4">
                战斗日志为空
              </div>
            ) : (
              combatLog.map((log, index) => (
                <div key={index} className="text-sm">
                  {log}
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      {/* 控制按钮 */}
      <div className="flex justify-between">
        <Button
          variant="outline"
          onClick={handleNextTurn}
          className="flex-1 mr-2"
        >
          下一回合
        </Button>
      </div>
    </div>
  )
}
```

---

## 战斗状态管理

```tsx
// frontend/src/hooks/useCombat.ts
import { useState, useEffect } from 'react'
import { socket } from '@/lib/socket'

interface UseCombatProps {
  roomId: string
}

export function useCombat({ roomId }: UseCombatProps) {
  const [combatants, setCombatants] = useState<any[]>([])
  const [currentTurn, setCurrentTurn] = useState(0)
  const [isActive, setIsActive] = useState(false)

  useEffect(() => {
    // 监听战斗事件
    socket.on('combat_start', handleCombatStart)
    socket.on('combat_update', handleCombatUpdate)
    socket.on('combat_next_turn', handleNextTurn)
    socket.on('combat_end', handleCombatEnd)

    return () => {
      socket.off('combat_start')
      socket.off('combat_update')
      socket.off('combat_next_turn')
      socket.off('combat_end')
    }
  }, [roomId])

  const handleCombatStart = (data: any) => {
    setCombatants(data.combatants)
    setCurrentTurn(data.current_turn)
    setIsActive(true)
  }

  const handleCombatUpdate = (data: any) => {
    setCombatants(data.combatants)
  }

  const handleNextTurn = (data: any) => {
    setCurrentTurn(data.current_turn)
    setCombatants(data.combatants)
  }

  const handleCombatEnd = () => {
    setIsActive(false)
    setCombatants([])
    setCurrentTurn(0)
  }

  const startCombat = async () => {
    socket.emit('combat_start', { room_id: roomId })
  }

  const nextTurn = async () => {
    socket.emit('combat_next_turn', { room_id: roomId })
  }

  const endCombat = async () => {
    socket.emit('combat_end', { room_id: roomId })
  }

  return {
    combatants,
    currentTurn,
    isActive,
    startCombat,
    nextTurn,
    endCombat,
  }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/CombatPanel.tsx` | 创建 | 战斗面板组件 |
| `frontend/src/hooks/useCombat.ts` | 创建 | 战斗 Hook |
| `frontend/src/types/combat.ts` | 创建 | 战斗类型定义 |

---

## 验收标准

- [ ] 轮次顺序显示正确
- [ ] 战斗状态同步准确
- [ ] 伤害功能有效
- [ ] 战斗日志完整
- [ ] KP 控制功能正常
- [ ] WebSocket 通信稳定

---

## 参考文档

- M1-020: 战斗系统
- M2-002: WebSocket 事件系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
