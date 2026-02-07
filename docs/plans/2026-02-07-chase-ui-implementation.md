# 追逐系统前端实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现CoC 7th Edition追逐系统的前端界面，包括距离可视化、速度管理、障碍物检定和追逐日志。

**Architecture:**
- 复用战斗系统的覆盖层模式（`CombatOverlay`）
- 三栏布局：信息面板 | 行动面板 | 日志面板
- WebSocket实时状态同步
- 自定义hooks管理状态和操作

**Tech Stack:**
- React 19 + TypeScript
- TailwindCSS + shadcn/ui
- WebSocket (复用 `useWebSocket`)
- 后端API已完成 (`/chase/*` endpoints)

---

## 前置条件

### 需要阅读的文档
- `docs/plans/2026-02-07-chase-ui-design.md` - 完整设计文档
- `docs/specs/api.md` - API接口规范
- `frontend/src/components/combat/CombatOverlay.tsx` - 参考实现

### 后端API Endpoints (已实现)
- `GET /chase/{chase_id}` - 获取追逐状态
- `POST /chase/{chase_id}/round` - 执行回合行动
- `POST /chase/{chase_id}/obstacles/check` - 障碍物检定
- `POST /chase/{chase_id}/end` - 结束追逐

---

## Task 1: 创建类型定义

**Files:**
- Create: `frontend/src/types/chase.ts`

**Step 1: 创建类型定义文件**

创建 `frontend/src/types/chase.ts`，添加以下内容：

```typescript
// 追逐状态
export type ChaseState = 'active' | 'paused' | 'ended'

// 参与者角色
export type ChaseParticipantRole = 'fugitive' | 'pursuer'

// 障碍物类型
export type ObstacleType = 'physical' | 'environmental' | 'skill_check' | 'combat'

// 行动类型
export type ActionType = 'accelerate' | 'decouple' | 'overcome_obstacle' | 'attack'

// 追逐主体
export interface Chase {
  id: string
  session_id: string
  state: ChaseState
  current_round: number
  distance_level: number
  pressure: number
  environment_type: string
  participants: ChaseParticipant[]
  obstacles: Obstacle[]
  created_at: string
  updated_at: string
}

// 追逐参与者
export interface ChaseParticipant {
  id: string
  chase_id: string
  character_id: string | null
  role: ChaseParticipantRole
  position_index: number
  move_rate: number
  current_speed: number
  is_active: boolean
  name?: string
  icon?: string
}

// 障碍物
export interface Obstacle {
  id: string
  chase_id: string
  type: ObstacleType
  difficulty: 'easy' | 'medium' | 'hard' | 'extreme'
  required_skill: string | null
  description: string
  penalty: number
  damage: number
}

// 行动请求
export interface ChaseActionRequest {
  participant_id: string
  action: ActionType
  target_id?: string
  check_value?: number
}

// 技能检定请求
export interface ObstacleCheckRequest {
  participant_id: string
  obstacle_id: string
  skill_name: string
  skill_value: number
  use_luck?: boolean
}

// 回合结果
export interface RoundResult {
  round: number
  actions: ActionResult[]
  new_distance_level: number
  new_pressure: number
  chase_ended: boolean
  winner?: 'fugitive' | 'pursuer'
}

// 行动结果
export interface ActionResult {
  participant_id: string
  action: ActionType
  success: boolean
  new_speed: number
  damage_taken?: number
  obstacle_overcome?: boolean
}

// 检定结果
export interface CheckResult {
  success: boolean
  roll_value: number
  success_level: 'regular' | 'hard' | 'extreme' | 'critical' | 'fumble'
  damage?: number
  speed_penalty?: number
  message: string
}

// WebSocket事件类型
export interface ChaseStartedEvent {
  chase_id: string
  chase: Chase
}

export interface ChaseUpdatedEvent {
  chase_id: string
  chase: Chase
}

export interface ChaseEndedEvent {
  chase_id: string
  winner?: 'fugitive' | 'pursuer'
}
```

**Step 2: 提交**

```bash
git add frontend/src/types/chase.ts
git commit -m "feat(M1-093): add chase type definitions"
```

---

## Task 2: 创建API服务层

**Files:**
- Create: `frontend/src/services/chase.ts`
- Modify: `frontend/src/services/api.ts` (export chaseService)

**Step 1: 创建chase服务**

创建 `frontend/src/services/chase.ts`：

```typescript
import { api } from './api'
import type {
  Chase,
  ChaseActionRequest,
  ObstacleCheckRequest,
  RoundResult,
  CheckResult,
  Obstacle
} from '@/types/chase'

class ChaseService {
  private baseURL = '/chase'

  async getChase(chaseId: string): Promise<Chase> {
    const { data } = await api.get(`${this.baseURL}/${chaseId}`)
    return data
  }

  async executeRoundAction(
    chaseId: string,
    request: ChaseActionRequest
  ): Promise<RoundResult> {
    const { data } = await api.post(`${this.baseURL}/${chaseId}/round`, request)
    return data
  }

  async performObstacleCheck(
    chaseId: string,
    request: ObstacleCheckRequest
  ): Promise<CheckResult> {
    const { data } = await api.post(
      `${this.baseURL}/${chaseId}/obstacles/check`,
      request
    )
    return data
  }

  async generateObstacles(chaseId: string): Promise<Obstacle[]> {
    const { data } = await api.post(`${this.baseURL}/${chaseId}/obstacles/generate`)
    return data
  }

  async endChase(chaseId: string): Promise<void> {
    await api.post(`${this.baseURL}/${chaseId}/end`)
  }
}

export const chaseService = new ChaseService()
```

**Step 2: 导出chaseService**

修改 `frontend/src/services/api.ts`，添加导出：

```typescript
// ... existing exports ...

export { chaseService } from './chase'
```

**Step 3: 提交**

```bash
git add frontend/src/services/chase.ts frontend/src/services/api.ts
git commit -m "feat(M1-093): add chase API service"
```

---

## Task 3: 创建useChaseState hook

**Files:**
- Create: `frontend/src/hooks/useChaseState.ts`

**Step 1: 创建useChaseState hook**

创建 `frontend/src/hooks/useChaseState.ts`：

```typescript
import { useState, useEffect } from 'react'
import { useWebSocket } from './useWebSocket'
import { chaseService } from '@/services/chase'
import type { Chase } from '@/types/chase'

interface UseChaseStateReturn {
  chase: Chase | null
  currentRound: number
  distanceLevel: number
  pressure: number
  participants: Chase['participants']
  obstacles: Chase['obstacles']
  isLoading: boolean
  error: string | null
}

export function useChaseState(chaseId: string | null): UseChaseStateReturn {
  const [chase, setChase] = useState<Chase | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { ws, isConnected } = useWebSocket()

  useEffect(() => {
    if (!chaseId || !isConnected) {
      setChase(null)
      return
    }

    // 监听追逐更新事件
    const handleChaseUpdate = (data: { chase: Chase }) => {
      setChase(data.chase)
    }

    ws.on('chase_updated', handleChaseUpdate)

    // 初始加载
    const loadChase = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const data = await chaseService.getChase(chaseId)
        setChase(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : '加载失败')
      } finally {
        setIsLoading(false)
      }
    }

    loadChase()

    return () => {
      ws.off('chase_updated', handleChaseUpdate)
    }
  }, [chaseId, isConnected, ws])

  return {
    chase,
    currentRound: chase?.current_round ?? 0,
    distanceLevel: chase?.distance_level ?? 0,
    pressure: chase?.pressure ?? 0,
    participants: chase?.participants ?? [],
    obstacles: chase?.obstacles ?? [],
    isLoading,
    error
  }
}
```

**Step 2: 提交**

```bash
git add frontend/src/hooks/useChaseState.ts
git commit -m "feat(M1-093): add useChaseState hook"
```

---

## Task 4: 创建useChaseActions hook

**Files:**
- Create: `frontend/src/hooks/useChaseActions.ts`

**Step 1: 创建useChaseActions hook**

创建 `frontend/src/hooks/useChaseActions.ts`：

```typescript
import { useCallback, useState } from 'react'
import { chaseService } from '@/services/chase'
import type { ChaseActionRequest, ObstacleCheckRequest } from '@/types/chase'
import { toast } from 'sonner'

interface UseChaseActionsReturn {
  executeAction: (request: ChaseActionRequest) => Promise<void>
  performCheck: (request: ObstacleCheckRequest) => Promise<void>
  skipTurn: (participantId: string) => Promise<void>
  endChase: () => Promise<void>
  isExecuting: boolean
}

export function useChaseActions(chaseId: string | null): UseChaseActionsReturn {
  const [isExecuting, setIsExecuting] = useState(false)

  const executeAction = useCallback(async (request: ChaseActionRequest) => {
    if (!chaseId) return
    setIsExecuting(true)
    try {
      const result = await chaseService.executeRoundAction(chaseId, request)
      if (result.chase_ended) {
        toast.success(result.winner === 'fugitive' ? '逃脱成功！' : '被追上了！')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败')
      throw err
    } finally {
      setIsExecuting(false)
    }
  }, [chaseId])

  const performCheck = useCallback(async (request: ObstacleCheckRequest) => {
    if (!chaseId) return
    setIsExecuting(true)
    try {
      await chaseService.performObstacleCheck(chaseId, request)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '检定失败')
      throw err
    } finally {
      setIsExecuting(false)
    }
  }, [chaseId])

  const skipTurn = useCallback(async (participantId: string) => {
    await executeAction({
      participant_id: participantId,
      action: 'attack' // placeholder, use skip action
    })
  }, [executeAction])

  const endChase = useCallback(async () => {
    if (!chaseId) return
    setIsExecuting(true)
    try {
      await chaseService.endChase(chaseId)
      toast.success('追逐已结束')
    } catch (err) {
      toast.error('结束追逐失败')
      throw err
    } finally {
      setIsExecuting(false)
    }
  }, [chaseId])

  return {
    executeAction,
    performCheck,
    skipTurn,
    endChase,
    isExecuting
  }
}
```

**Step 2: 提交**

```bash
git add frontend/src/hooks/useChaseActions.ts
git commit -m "feat(M1-093): add useChaseActions hook"
```

---

## Task 5: 创建ChaseInfoPanel组件

**Files:**
- Create: `frontend/src/components/chase/ChaseInfoPanel.tsx`
- Create: `frontend/src/components/chase/DistanceTrack.tsx`
- Create: `frontend/src/components/chase/PressureBar.tsx`
- Create: `frontend/src/components/chase/ParticipantList.tsx`

**Step 1: 创建DistanceTrack组件**

创建 `frontend/src/components/chase/DistanceTrack.tsx`：

```typescript
import React from 'react'

interface DistanceTrackProps {
  distanceLevel: number
}

export function DistanceTrack({ distanceLevel }: DistanceTrackProps) {
  const track = '━'.repeat(distanceLevel)

  return (
    <div className="text-center bg-slate-800 rounded-lg p-4">
      <div className="text-sm text-slate-400 mb-2">距离: {distanceLevel} 格</div>
      <div className="text-xl font-mono tracking-wider">
        <span className="text-green-400">逃跑者</span>
        {' '}{/*spacer*/}●{track}●{' '}{/*spacer*/}
        <span className="text-red-400">追逐者</span>
      </div>
    </div>
  )
}
```

**Step 2: 创建PressureBar组件**

创建 `frontend/src/components/chase/PressureBar.tsx`：

```typescript
import React from 'react'

interface PressureBarProps {
  pressure: number
}

export function PressureBar({ pressure }: PressureBarProps) {
  const getColor = () => {
    if (pressure >= 80) return 'bg-red-500'
    if (pressure >= 50) return 'bg-yellow-500'
    return 'bg-green-500'
  }

  return (
    <div className="bg-slate-800 rounded-lg p-4">
      <div className="flex justify-between text-sm mb-2">
        <span className="text-slate-400">压力</span>
        <span className={pressure >= 80 ? 'text-red-400 animate-pulse' : 'text-slate-300'}>
          {pressure}%
        </span>
      </div>
      <div className="w-full bg-slate-700 rounded-full h-3">
        <div
          className={`${getColor()} h-3 rounded-full transition-all duration-300`}
          style={{ width: `${pressure}%` }}
        />
      </div>
    </div>
  )
}
```

**Step 3: 创建ParticipantList组件**

创建 `frontend/src/components/chase/ParticipantList.tsx`：

```typescript
import React from 'react'
import type { ChaseParticipant } from '@/types/chase'

interface ParticipantListProps {
  participants: ChaseParticipant[]
  currentParticipantId?: string
}

export function ParticipantList({ participants, currentParticipantId }: ParticipantListProps) {
  const getSpeedLabel = (speed: number) => {
    if (speed > 0) return `+${speed}`
    return speed.toString()
  }

  const getIcon = (participant: ChaseParticipant) => {
    if (participant.character_id) return '🏃'
    return '🧟'
  }

  return (
    <div className="bg-slate-800 rounded-lg p-4">
      <h3 className="text-sm text-slate-400 mb-3">参与者</h3>
      <div className="space-y-2">
        {participants.map((p) => (
          <div
            key={p.id}
            className={`flex items-center justify-between p-2 rounded ${
              p.id === currentParticipantId ? 'bg-blue-600' : 'bg-slate-700'
            }`}
          >
            <div className="flex items-center gap-2">
              <span>{getIcon(p)}</span>
              <span className="text-sm">{p.name || '未知'}</span>
            </div>
            <div className="text-sm font-mono">
              速度: {getSpeedLabel(p.current_speed)}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
```

**Step 4: 创建ChaseInfoPanel组件**

创建 `frontend/src/components/chase/ChaseInfoPanel.tsx`：

```typescript
import React from 'react'
import type { ChaseParticipant } from '@/types/chase'
import { DistanceTrack } from './DistanceTrack'
import { PressureBar } from './PressureBar'
import { ParticipantList } from './ParticipantList'

interface ChaseInfoPanelProps {
  distanceLevel: number
  pressure: number
  participants: ChaseParticipant[]
  currentParticipantId?: string
}

export function ChaseInfoPanel({
  distanceLevel,
  pressure,
  participants,
  currentParticipantId
}: ChaseInfoPanelProps) {
  return (
    <div className="space-y-4">
      <DistanceTrack distanceLevel={distanceLevel} />
      <PressureBar pressure={pressure} />
      <ParticipantList
        participants={participants}
        currentParticipantId={currentParticipantId}
      />
    </div>
  )
}
```

**Step 5: 提交**

```bash
git add frontend/src/components/chase/
git commit -m "feat(M1-093~095): add ChaseInfoPanel with distance, pressure, participants"
```

---

## Task 6: 创建ChaseActionPanel组件

**Files:**
- Create: `frontend/src/components/chase/ChaseActionPanel.tsx`
- Create: `frontend/src/components/chase/ObstacleCard.tsx`
- Create: `frontend/src/components/chase/ActionSelector.tsx`
- Create: `frontend/src/components/chase/CheckResult.tsx`

**Step 1: 创建ObstacleCard组件**

创建 `frontend/src/components/chase/ObstacleCard.tsx`：

```typescript
import React from 'react'
import { AlertTriangle } from 'lucide-react'
import type { Obstacle } from '@/types/chase'
import { Button } from '@/components/ui/button'

interface ObstacleCardProps {
  obstacle: Obstacle
  onCheck: () => void
  onUseLuck: () => void
  isLoading?: boolean
}

export function ObstacleCard({ obstacle, onCheck, onUseLuck, isLoading }: ObstacleCardProps) {
  return (
    <div className="bg-slate-800 rounded-lg p-4 border-2 border-yellow-600">
      <div className="flex items-center gap-2 mb-4">
        <AlertTriangle className="text-yellow-500" />
        <h3 className="text-lg font-semibold">前方有障碍物!</h3>
      </div>

      <div className="space-y-2 mb-4">
        <div className="text-sm text-slate-400">障碍: {obstacle.description}</div>
        <div className="text-sm text-slate-400">类型: {obstacle.type}</div>
        <div className="text-sm text-slate-400">
          难度: <span className="text-yellow-400">{obstacle.difficulty}</span>
        </div>
        {obstacle.required_skill && (
          <div className="text-sm text-slate-400">
            检定: {obstacle.required_skill}
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <Button onClick={onCheck} disabled={isLoading} className="flex-1">
          立即检定
        </Button>
        <Button onClick={onUseLuck} variant="outline" disabled={isLoading}>
          消耗幸运
        </Button>
      </div>
    </div>
  )
}
```

**Step 2: 创建ActionSelector组件**

创建 `frontend/src/components/chase/ActionSelector.tsx`：

```typescript
import React from 'react'
import { Zap } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { ChaseParticipant, ActionType } from '@/types/chase'

interface ActionSelectorProps {
  participant: ChaseParticipant
  onAction: (action: ActionType) => void
  isLoading?: boolean
}

export function ActionSelector({ participant, onAction, isLoading }: ActionSelectorProps) {
  const actions = [
    { type: 'accelerate' as ActionType, label: '风险加速', icon: Zap, desc: '成功+1速/失败-1速' }
  ]

  return (
    <div className="space-y-4">
      <div className="bg-slate-800 rounded-lg p-4">
        <div className="text-sm text-slate-400 mb-2">当前行动: {participant.name}</div>

        <div className="space-y-2">
          {actions.map((action) => (
            <div key={action.type} className="bg-slate-700 rounded p-3">
              <div className="flex items-center gap-2 mb-1">
                <action.icon className="w-4 h-4" />
                <span className="font-semibold">{action.label}</span>
              </div>
              <div className="text-sm text-slate-400 mb-3">{action.desc}</div>
              <Button
                onClick={() => onAction(action.type)}
                disabled={isLoading}
                className="w-full"
              >
                执行
              </Button>
            </div>
          ))}
        </div>
      </div>

      <Button variant="outline" className="w-full" disabled={isLoading}>
        跳过回合
      </Button>
    </div>
  )
}
```

**Step 3: 创建CheckResult组件**

创建 `frontend/src/components/chase/CheckResult.tsx`：

```typescript
import React from 'react'
import { Dice } from 'lucide-react'
import type { CheckResult } from '@/types/chase'

interface CheckResultDisplayProps {
  result: CheckResult
}

export function CheckResult({ result }: CheckResultDisplayProps) {
  const getSuccessColor = () => {
    if (result.success) return 'text-green-400'
    return 'text-red-400'
  }

  const getSuccessLabel = () => {
    if (result.success_level === 'critical') return '大成功!'
    if (result.success_level === 'extreme') return '极难成功!'
    if (result.success_level === 'hard') return '困难成功'
    if (result.success_level === 'regular') return '普通成功'
    if (result.success_level === 'fumble') return '大失败!'
    return '失败'
  }

  return (
    <div className="bg-slate-800 rounded-lg p-4 border-2 border-blue-600">
      <div className="flex items-center gap-2 mb-3">
        <Dice className="text-blue-400" />
        <h3 className="font-semibold">检定结果</h3>
      </div>

      <div className="space-y-1 text-sm">
        <div>🎲 掷骰: {result.roll_value}</div>
        <div className={getSuccessColor()}>
          {result.success ? '✅' : '❌'} {getSuccessLabel()}
        </div>
        {result.damage && <div className="text-red-400">受到伤害: {result.damage}</div>}
        {result.speed_penalty && <div className="text-yellow-400">速度惩罚: {result.speed_penalty}</div>}
      </div>
    </div>
  )
}
```

**Step 4: 创建ChaseActionPanel组件**

创建 `frontend/src/components/chase/ChaseActionPanel.tsx`：

```typescript
import React, { useState } from 'react'
import type { Chase, Obstacle } from '@/types/chase'
import { ObstacleCard } from './ObstacleCard'
import { ActionSelector } from './ActionSelector'
import type { CheckResult } from './CheckResult'

interface ChaseActionPanelProps {
  chase: Chase
  onExecuteAction: (action) => Promise<void>
  onPerformCheck: (check) => Promise<void>
  isExecuting?: boolean
}

export function ChaseActionPanel({
  chase,
  onExecuteAction,
  onPerformCheck,
  isExecuting
}: ChaseActionPanelProps) {
  const [checkResult, setCheckResult] = useState<CheckResult | null>(null)

  // 获取当前需要行动的参与者
  const currentParticipant = chase.participants.find(p => p.is_active)
  if (!currentParticipant) {
    return <div className="text-center text-slate-400">等待其他参与者...</div>
  }

  // 检查是否有障碍物
  const currentObstacle = chase.obstacles.find(o => !o.resolved)

  const handleCheck = async () => {
    if (!currentObstacle || !currentParticipant) return

    const result = await onPerformCheck({
      participant_id: currentParticipant.id,
      obstacle_id: currentObstacle.id,
      skill_name: currentObstacle.required_skill || '体质',
      skill_value: 50 // placeholder
    })

    setCheckResult(result)
  }

  return (
    <div className="space-y-4">
      {checkResult ? (
        <CheckResult result={checkResult} />
      ) : currentObstacle ? (
        <ObstacleCard
          obstacle={currentObstacle}
          onCheck={handleCheck}
          onUseLuck={() => {}}
          isLoading={isExecuting}
        />
      ) : (
        <ActionSelector
          participant={currentParticipant}
          onAction={(action) => onExecuteAction({
            participant_id: currentParticipant.id,
            action
          })}
          isLoading={isExecuting}
        />
      )}
    </div>
  )
}
```

**Step 5: 提交**

```bash
git add frontend/src/components/chase/
git commit -m "feat(M1-096): add ChaseActionPanel with obstacle and action handling"
```

---

## Task 7: 创建ChaseLogPanel组件

**Files:**
- Create: `frontend/src/components/chase/ChaseLogPanel.tsx`

**Step 1: 创建ChaseLogPanel组件**

创建 `frontend/src/components/chase/ChaseLogPanel.tsx`：

```typescript
import React, { useState, useEffect } from 'react'
import { ScrollText } from 'lucide-react'
import { useWebSocket } from '@/hooks/useWebSocket'

interface LogEntry {
  id: string
  round: number
  message: string
  timestamp: Date
}

interface ChaseLogPanelProps {
  chaseId: string
}

export function ChaseLogPanel({ chaseId }: ChaseLogPanelProps) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const { ws, isConnected } = useWebSocket()

  useEffect(() => {
    if (!chaseId || !isConnected) return

    const handleChaseEvent = (data: { round: number; message: string }) => {
      const entry: LogEntry = {
        id: `${Date.now()}-${Math.random()}`,
        round: data.round,
        message: data.message,
        timestamp: new Date()
      }

      setLogs(prev => [entry, ...prev].slice(0, 50))
    }

    ws.on('chase_log', handleChaseEvent)

    return () => {
      ws.off('chase_log', handleChaseEvent)
    }
  }, [chaseId, isConnected, ws])

  return (
    <div className="bg-slate-800 rounded-lg p-4 h-full flex flex-col">
      <div className="flex items-center gap-2 mb-3">
        <ScrollText className="w-4 h-4 text-slate-400" />
        <h3 className="text-sm font-semibold">追逐日志</h3>
      </div>

      <div className="flex-1 overflow-y-auto space-y-2">
        {logs.length === 0 ? (
          <div className="text-sm text-slate-500 text-center">等待事件...</div>
        ) : (
          logs.map((log) => (
            <div key={log.id} className="text-sm bg-slate-700 rounded p-2 animate-in fade-in slide-in-from-top-1">
              <div className="text-xs text-slate-400">[回合{log.round}]</div>
              <div>{log.message}</div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
```

**Step 2: 提交**

```bash
git add frontend/src/components/chase/ChaseLogPanel.tsx
git commit -m "feat(M1-093): add ChaseLogPanel component"
```

---

## Task 8: 创建ChaseOverlay主容器

**Files:**
- Create: `frontend/src/components/chase/ChaseOverlay.tsx`
- Create: `frontend/src/components/chase/index.ts` (barrel export)

**Step 1: 创建ChaseOverlay组件**

创建 `frontend/src/components/chase/ChaseOverlay.tsx`：

```typescript
import React, { useEffect } from 'react'
import { X, Minus } from 'lucide-react'
import { useChaseState } from '@/hooks/useChaseState'
import { useChaseActions } from '@/hooks/useChaseActions'
import { ChaseInfoPanel } from './ChaseInfoPanel'
import { ChaseActionPanel } from './ChaseActionPanel'
import { ChaseLogPanel } from './ChaseLogPanel'
import type { ChaseActionRequest, ObstacleCheckRequest } from '@/types/chase'

interface ChaseOverlayProps {
  chaseId: string
  onClose: () => void
  onMinimize: () => void
}

export function ChaseOverlay({ chaseId, onClose, onMinimize }: ChaseOverlayProps) {
  const { chase, distanceLevel, pressure, participants, obstacles } = useChaseState(chaseId)
  const { executeAction, performCheck, isExecuting } = useChaseActions(chaseId)

  // ESC键关闭
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [onClose])

  if (!chase) {
    return (
      <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center">
        <div className="text-white">加载中...</div>
      </div>
    )
  }

  // 获取当前行动的参与者
  const currentParticipant = participants.find(p => p.is_active)

  return (
    <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
      <div
        className="bg-slate-900 rounded-lg shadow-2xl w-full max-w-6xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="bg-slate-800 border-b border-slate-700 px-4 py-3 flex justify-between items-center rounded-t-lg">
          <h2 className="text-lg font-semibold">
            追逐中 - 第 {chase.current_round} 回合
          </h2>
          <div className="flex gap-2">
            <button
              onClick={onMinimize}
              className="p-1 hover:bg-slate-700 rounded"
              title="最小化"
            >
              <Minus className="w-5 h-5" />
            </button>
            <button
              onClick={onClose}
              className="p-1 hover:bg-slate-700 rounded"
              title="关闭"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Main Content */}
        <div className="grid grid-cols-3 gap-4 p-4 h-[600px]">
          <ChaseInfoPanel
            distanceLevel={distanceLevel}
            pressure={pressure}
            participants={participants}
            currentParticipantId={currentParticipant?.id}
          />
          <ChaseActionPanel
            chase={chase}
            onExecuteAction={executeAction as (req: ChaseActionRequest) => Promise<void>}
            onPerformCheck={performCheck as (req: ObstacleCheckRequest) => Promise<void>}
            isExecuting={isExecuting}
          />
          <ChaseLogPanel chaseId={chaseId} />
        </div>
      </div>
    </div>
  )
}
```

**Step 2: 创建barrel export**

创建 `frontend/src/components/chase/index.ts`：

```typescript
export { ChaseOverlay } from './ChaseOverlay'
export { ChaseInfoPanel } from './ChaseInfoPanel'
export { ChaseActionPanel } from './ChaseActionPanel'
export { ChaseLogPanel } from './ChaseLogPanel'
export { DistanceTrack } from './DistanceTrack'
export { PressureBar } from './PressureBar'
export { ParticipantList } from './ParticipantList'
```

**Step 3: 提交**

```bash
git add frontend/src/components/chase/
git commit -m "feat(M1-093): add ChaseOverlay main container"
```

---

## Task 9: 集成到GameConsole

**Files:**
- Modify: `frontend/src/pages/GameConsole.tsx` (或相关主游戏界面文件)

**Step 1: 添加chase状态管理**

找到 `GameConsole.tsx` 文件，添加追逐状态：

```typescript
// 在现有状态声明附近添加
const [chaseId, setChaseId] = useState<string | null>(null)
const [isChaseMinimized, setIsChaseMinimized] = useState(false)
```

**Step 2: 添加WebSocket监听**

在WebSocket监听部分添加：

```typescript
// 在现有useEffect中添加
ws.on('chase_started', (data: { chase_id: string }) => {
  setChaseId(data.chase_id)
})

ws.on('chase_ended', () => {
  setChaseId(null)
  setIsChaseMinimized(false)
})
```

**Step 3: 添加ChaseOverlay渲染**

在JSX返回中添加：

```typescript
{/* 在现有JSX中，CombatOverlay附近添加 */}
{chaseId && !isChaseMinimized && (
  <ChaseOverlay
    chaseId={chaseId}
    onClose={() => setChaseId(null)}
    onMinimize={() => setIsChaseMinimized(true)}
  />
)}
```

**Step 4: 添加import**

在文件顶部添加：

```typescript
import { ChaseOverlay } from '@/components/chase'
```

**Step 5: 提交**

```bash
git add frontend/src/pages/GameConsole.tsx
git commit -m "feat(M1-093): integrate ChaseOverlay into GameConsole"
```

---

## Task 10: 编写测试

**Files:**
- Create: `frontend/src/components/chase/__tests__/DistanceTrack.test.tsx`
- Create: `frontend/src/components/chase/__tests__/PressureBar.test.tsx`
- Create: `frontend/src/hooks/__tests__/useChaseState.test.ts`

**Step 1: 编写DistanceTrack测试**

创建 `frontend/src/components/chase/__tests__/DistanceTrack.test.tsx`：

```typescript
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { DistanceTrack } from '../DistanceTrack'

describe('DistanceTrack', () => {
  it('显示正确的距离等级', () => {
    render(<DistanceTrack distanceLevel={3} />)
    expect(screen.getByText(/距离: 3 格/)).toBeTruthy()
  })

  it('渲染正确数量的轨道符号', () => {
    const { container } = render(<DistanceTrack distanceLevel={2} />)
    const track = container.textContent || ''
    expect(track.split('━').length - 1).toBe(2)
  })
})
```

**Step 2: 编写PressureBar测试**

创建 `frontend/src/components/chase/__tests__/PressureBar.test.tsx`：

```typescript
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { PressureBar } from '../PressureBar'

describe('PressureBar', () => {
  it('显示低压力为绿色', () => {
    render(<PressureBar pressure={30} />)
    const bar = document.querySelector('.bg-green-500')
    expect(bar).toBeTruthy()
  })

  it('显示高压力为红色', () => {
    render(<PressureBar pressure={85} />)
    const bar = document.querySelector('.bg-red-500')
    expect(bar).toBeTruthy()
  })
})
```

**Step 3: 编写useChaseState测试**

创建 `frontend/src/hooks/__tests__/useChaseState.test.ts`：

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { useChaseState } from '../useChaseState'

// Mock WebSocket
vi.mock('../useWebSocket', () => ({
  useWebSocket: () => ({
    ws: {
      on: vi.fn(),
      off: vi.fn()
    },
    isConnected: true
  })
}))

// Mock chaseService
vi.mock('@/services/chase', () => ({
  chaseService: {
    getChase: vi.fn(() => Promise.resolve({
      id: 'test-chase-id',
      current_round: 1,
      distance_level: 3,
      pressure: 40,
      participants: [],
      obstacles: []
    }))
  }
}))

describe('useChaseState', () => {
  it('加载追逐状态', async () => {
    const { result } = renderHook(() => useChaseState('test-chase-id'))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
      expect(result.current.distanceLevel).toBe(3)
    })
  })
})
```

**Step 4: 运行测试**

```bash
cd frontend
npm test -- ChaseOverlay
```

**Step 5: 提交**

```bash
git add frontend/src/components/chase/__tests__/ frontend/src/hooks/__tests__/
git commit -m "test(M1-093): add chase system tests"
```

---

## Task 11: 响应式适配

**Files:**
- Modify: `frontend/src/components/chase/ChaseOverlay.tsx`

**Step 1: 添加响应式类名**

修改 `ChaseOverlay.tsx` 中的网格布局：

```typescript
// 修改网格布局部分
<div className="grid grid-cols-1 lg:grid-cols-3 gap-4 p-4 h-[600px]">
  {/* 在小屏幕上堆叠，大屏幕上三栏 */}
  <div className="lg:col-span-1 space-y-4">
    <ChaseInfoPanel ... />
  </div>
  <div className="lg:col-span-1 space-y-4">
    <ChaseActionPanel ... />
  </div>
  <div className="lg:col-span-1">
    <ChaseLogPanel chaseId={chaseId} />
  </div>
</div>
```

**Step 2: 提交**

```bash
git add frontend/src/components/chase/ChaseOverlay.tsx
git commit -m "feat(M1-093): add responsive layout to ChaseOverlay"
```

---

## Task 12: 更新任务状态

**Files:**
- Modify: `docs/tasks/02-m1-single-player-web.md`

**Step 1: 更新任务状态**

在 `docs/tasks/02-m1-single-player-web.md` 中，将以下任务标记为完成：

```markdown
| M1-093 | [x] 实现 ChaseTracker 组件 | frontend | 4h | M1-085 | [x] |
| M1-094 | [x] 实现距离可视化 | frontend | 2h | M1-093 | [x] |
| M1-095 | [x] 实现压力指示器 | frontend | 1h | M1-093 | [x] |
| M1-096 | [x] 实现障碍展示 | frontend | 1h | M1-093 | [x] |
```

**Step 2: 提交**

```bash
git add docs/tasks/02-m1-single-player-web.md
git commit -m "docs(M1-093~096): mark chase UI tasks as completed"
```

---

## 验收检查清单

在完成所有任务后，运行以下检查：

```bash
# 1. 前端编译检查
cd frontend
npm run build

# 2. 类型检查
npx tsc --noEmit

# 3. 测试通过
npm test

# 4. 启动开发服务器测试
npm run dev
```

**手动测试场景：**
1. [ ] 追逐开始时Overlay自动弹出
2. [ ] 距离轨道正确显示
3. [ ] 压力进度条颜色随值变化
4. [ ] 参与者列表正确显示
5. [ ] 障碍物卡片正确显示
6. [ ] 行动按钮点击后执行
7. [ ] ESC键关闭Overlay
8. [ ] WebSocket更新实时反映

---

## 完成后

实施完成后，需要：

1. **更新验收标准** - 在 `docs/tasks/02-m1-single-player-web.md` 中更新相关验收标准
2. **创建PR** - 如果使用git worktree，创建Pull Request
3. **测试E2E** - 运行完整的追逐场景测试

---

**预估总工时**: 10小时
**风险**: 低 - 后端已完成，只需前端实现
