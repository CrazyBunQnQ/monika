# 追逐系统前端设计文档

**日期**: 2026-02-07
**任务**: M1-093 ~ M1-096 (追逐系统前端)
**状态**: 设计完成，待实施

---

## 1. 概述

### 1.1 目标
实现CoC 7th Edition追逐系统的前端界面，包括距离可视化、速度管理、障碍物检定和追逐日志。

### 1.2 设计原则
- 复用战斗系统的覆盖层模式
- 上下文动态行动面板，减少玩家决策负担
- 快节奏体验，减少弹窗切换
- 与现有UI风格保持一致

---

## 2. 组件架构

### 2.1 组件结构树

```
ChaseOverlay (主容器)
├── ChaseHeader (头部：回合信息、最小化/关闭按钮)
├── ChaseMain (主内容区 - 三栏布局)
│   ├── ChaseInfoPanel (左栏 - 距离/速度/参与者信息)
│   │   ├── DistanceTrack (横向轨道图)
│   │   ├── PressureBar (压力进度条)
│   │   └── ParticipantList (参与者列表)
│   ├── ChaseActionPanel (中栏 - 上下文动态行动面板)
│   │   ├── ActionSelector (行动选择器)
│   │   ├── ObstacleCard (障碍物卡片)
│   │   └── CheckResult (检定结果)
│   └── ChaseLogPanel (右栏 - 追逐日志)
└── ChaseFooter (底部 - 快捷操作提示)
```

### 2.2 数据流

```
WebSocket → chase状态更新 → ChaseOverlay重新渲染
    ↓
用户点击行动 → API调用 → 后端处理 → WebSocket广播更新
```

### 2.3 与GameConsole集成

```typescript
// GameConsole.tsx
const [chaseId, setChaseId] = useState<string | null>(null)
const [isChaseMinimized, setIsChaseMinimized] = useState(false)

// WebSocket监听chase事件
useEffect(() => {
  ws.on('chase_started', (data) => setChaseId(data.chase_id))
  ws.on('chase_ended', () => setChaseId(null))
}, [])

// 条件渲染
{chaseId && !isChaseMinimized && (
  <ChaseOverlay
    chaseId={chaseId}
    onClose={() => setChaseId(null)}
    onMinimize={() => setIsChaseMinimized(true)}
  />
)}
```

---

## 3. UI组件详细设计

### 3.1 ChaseOverlay 主容器

**样式规格：**
- 全屏覆盖层：`fixed inset-0 bg-black/60 z-50`
- 主面板居中：`flex items-center justify-center`
- 面板尺寸：`max-w-6xl w-full mx-4`
- 面板样式：`bg-slate-900 rounded-lg shadow-2xl`

**交互：**
- ESC键关闭
- 点击遮罩关闭（排除面板区域）
- 最小化按钮收起到右下角小浮窗

### 3.2 ChaseHeader 头部

```
┌─────────────────────────────────────────────┐
│ 追逐中 - 第 3 回合        [─] [✕]           │
└─────────────────────────────────────────────┘
```

- 左侧：`追逐中 - 第 {round} 回合`
- 右侧：最小化按钮 [─] / 关闭按钮 [✕]
- 样式：`bg-slate-800 border-b border-slate-700 px-4 py-3 flex justify-between items-center`

### 3.3 ChaseInfoPanel 左栏 - 信息面板

**距离可视化（横向轨道图）：**

```
┌───────────────────────┐
│ 距离: 3 格             │
│ 逃跑者 ●━━● 追逐者     │
└───────────────────────┘
```

**实现逻辑：**
```typescript
function DistanceTrack({ distanceLevel }: { distanceLevel: number }) {
  const track = '━'.repeat(distanceLevel)
  return (
    <div className="text-center">
      <div className="text-sm text-slate-400 mb-1">距离: {distanceLevel} 格</div>
      <div className="text-2xl font-mono">
        逃跑者 ●{track}● 追逐者
      </div>
    </div>
  )
}
```

**压力进度条：**
```
压力: ████████░░ 40%
```
- 样式：类似HP/SAN进度条
- 颜色：0-50%绿色，50-80%黄色，80-100%红色闪烁

**参与者速度徽章：**
```
┌───────────────────────┐
│ 参与者                 │
│ ┌─────────────────┐   │
│ │ 🏃 侦探(你) +1  │   │
│ │ 🧟 深潜者 0     │   │
│ │ 🧟 深潜者 -1    │   │
│ └─────────────────┘   │
└───────────────────────┘
```

- 每个参与者一行
- 角色图标 + 名称 + 相对速度
- 当前行动的参与者高亮（`bg-blue-600`）

### 3.4 ChaseActionPanel 中栏 - 行动面板

**无障碍时（正常状态）：**
```
┌───────────────────────────┐
│ 当前行动: 侦探(你)         │
│                           │
│ ┌─────────────────────┐   │
│ │ ⚡ 风险加速          │   │
│ │ 成功速度+1，失败-1   │   │
│ └─────────────────────┘   │
│                           │
│ [执行加速]  [跳过回合]     │
└───────────────────────────┘
```

**有障碍物时：**
```
┌───────────────────────────┐
│ 当前行动: 侦探(你)         │
│                           │
│ 🚧 前方有障碍物!           │
│ ┌─────────────────────┐   │
│ │ 障碍: 倒塌的围栏     │   │
│ │ 类型: 环境障碍       │   │
│ │ 难度: 困难          │   │
│ │ 检定: 攀爬 60%      │   │
│ └─────────────────────┘   │
│                           │
│ [立即检定]  [消耗幸运]     │
│                           │
│ 其他行动: [风险加速]       │
└───────────────────────────┘
```

**检定结果显示（内嵌）：**
```
│ ┌─────────────────────┐   │
│ │ 🎲 掷骰: 35        │   │
│ │ ✅ 极难成功!        │   │
│ │ 速度保持不变        │   │
│ └─────────────────────┘   │
```

**上下文切换逻辑：**
```typescript
function ChaseActionPanel({ chase, currentParticipant }: Props) {
  const currentObstacle = getCurrentObstacle(chase)

  if (currentObstacle) {
    return <ObstacleCard obstacle={currentObstacle} />
  }

  return <ActionSelector participant={currentParticipant} />
}
```

### 3.5 ChaseLogPanel 右栏 - 追逐日志

```
┌───────────────────────────┐
│ 📜 追逐日志                │
├───────────────────────────┤
│ [回合3] 侦探加速成功      │
│ [回合3] 深潜者A前进1格    │
│ [回合2] 侦探克服障碍成功  │
│ [回合1] 追逐开始!         │
└───────────────────────────┘
```

- 最新消息在顶部
- 带回合标签 `[回合N]`
- 自动滚动到顶部
- 最大显示50条，超出滚动

---

## 4. 技术实现

### 4.1 类型定义

**文件：`frontend/src/types/chase.ts`**

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
  name?: string  // 前端显示用
  icon?: string  // 前端显示用
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
  target_id?: string  // 用于攻击行动
  check_value?: number  // 技能检定值
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
```

### 4.2 API服务层

**文件：`frontend/src/services/chase.ts`**

```typescript
import { api } from './api'
import type {
  Chase,
  ChaseActionRequest,
  ObstacleCheckRequest,
  RoundResult,
  CheckResult
} from '@/types/chase'

export class ChaseService {
  private baseURL = '/chase'

  // 获取追逐状态
  async getChase(chaseId: string): Promise<Chase> {
    const { data } = await api.get(`${this.baseURL}/${chaseId}`)
    return data
  }

  // 执行回合行动
  async executeRoundAction(
    chaseId: string,
    request: ChaseActionRequest
  ): Promise<RoundResult> {
    const { data } = await api.post(`${this.baseURL}/${chaseId}/round`, request)
    return data
  }

  // 执行障碍物检定
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

  // 生成障碍物
  async generateObstacles(chaseId: string): Promise<Obstacle[]> {
    const { data } = await api.post(`${this.baseURL}/${chaseId}/obstacles/generate`)
    return data
  }

  // 结束追逐
  async endChase(chaseId: string): Promise<void> {
    await api.post(`${this.baseURL}/${chaseId}/end`)
  }
}

export const chaseService = new ChaseService()
```

### 4.3 自定义Hooks

**文件：`frontend/src/hooks/useChaseState.ts`**

```typescript
import { useState, useEffect } from 'react'
import { useWebSocket } from './useWebSocket'
import type { Chase } from '@/types/chase'

export function useChaseState(chaseId: string | null) {
  const [chase, setChase] = useState<Chase | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { ws, isConnected } = useWebSocket()

  useEffect(() => {
    if (!chaseId || !isConnected) return

    // 监听追逐更新事件
    const handleChaseUpdate = (data: { chase: Chase }) => {
      setChase(data.chase)
    }

    ws.on('chase_updated', handleChaseUpdate)

    // 初始加载
    const loadChase = async () => {
      setIsLoading(true)
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
  }, [chaseId, isConnected])

  // 派生状态
  const currentRound = chase?.current_round ?? 0
  const distanceLevel = chase?.distance_level ?? 0
  const pressure = chase?.pressure ?? 0
  const participants = chase?.participants ?? []
  const obstacles = chase?.obstacles ?? []

  return {
    chase,
    currentRound,
    distanceLevel,
    pressure,
    participants,
    obstacles,
    isLoading,
    error
  }
}
```

**文件：`frontend/src/hooks/useChaseActions.ts`**

```typescript
import { useCallback, useState } from 'react'
import { chaseService } from '@/services/chase'
import type { ChaseActionRequest, ObstacleCheckRequest } from '@/types/chase'
import { toast } from 'sonner'

export function useChaseActions(chaseId: string | null) {
  const [isExecuting, setIsExecuting] = useState(false)

  const executeAction = useCallback(async (request: ChaseActionRequest) => {
    if (!chaseId) return
    setIsExecuting(true)
    try {
      const result = await chaseService.executeRoundAction(chaseId, request)
      if (result.chase_ended) {
        toast.success(result.winner === 'fugitive' ? '逃脱成功！' : '被追上了！')
      }
      return result
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
      const result = await chaseService.performObstacleCheck(chaseId, request)
      return result
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '检定失败')
      throw err
    } finally {
      setIsExecuting(false)
    }
  }, [chaseId])

  const skipTurn = useCallback(async (participantId: string) => {
    return executeAction({
      participant_id: participantId,
      action: 'skip'
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

### 4.4 组件实现骨架

**文件：`frontend/src/components/chase/ChaseOverlay.tsx`**

```typescript
import { useEffect } from 'react'
import { X, Minus } from 'lucide-react'
import { useChaseState } from '@/hooks/useChaseState'
import { useChaseActions } from '@/hooks/useChaseActions'
import { ChaseInfoPanel } from './ChaseInfoPanel'
import { ChaseActionPanel } from './ChaseActionPanel'
import { ChaseLogPanel } from './ChaseLogPanel'

interface ChaseOverlayProps {
  chaseId: string
  onClose: () => void
  onMinimize: () => void
}

export function ChaseOverlay({ chaseId, onClose, onMinimize }: ChaseOverlayProps) {
  const { chase, currentRound, distanceLevel, pressure, participants, obstacles } =
    useChaseState(chaseId)
  const { executeAction, performCheck, endChase, isExecuting } =
    useChaseActions(chaseId)

  // ESC键关闭
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [onClose])

  if (!chase) return null

  return (
    <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
      <div
        className="bg-slate-900 rounded-lg shadow-2xl w-full max-w-6xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="bg-slate-800 border-b border-slate-700 px-4 py-3 flex justify-between items-center rounded-t-lg">
          <h2 className="text-lg font-semibold">追逐中 - 第 {currentRound} 回合</h2>
          <div className="flex gap-2">
            <button onClick={onMinimize} className="p-1 hover:bg-slate-700 rounded">
              <Minus className="w-5 h-5" />
            </button>
            <button onClick={onClose} className="p-1 hover:bg-slate-700 rounded">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Main Content */}
        <div className="grid grid-cols-3 gap-4 p-4">
          <ChaseInfoPanel
            distanceLevel={distanceLevel}
            pressure={pressure}
            participants={participants}
          />
          <ChaseActionPanel
            chase={chase}
            onExecuteAction={executeAction}
            onPerformCheck={performCheck}
            isExecuting={isExecuting}
          />
          <ChaseLogPanel chaseId={chaseId} />
        </div>
      </div>
    </div>
  )
}
```

---

## 5. 错误处理与边界情况

### 5.1 错误处理

| 场景 | 处理方式 |
|------|----------|
| API调用失败 | toast错误提示 + 重试按钮 |
| WebSocket断线 | 显示"连接中..." + 自动重连 |
| 状态不一致 | 重新获取状态 + 版本号检测 |
| 检定失败 | 显示结果 + 速度惩罚 |

### 5.2 边界情况

| 场景 | 处理方式 |
|------|----------|
| 追逐结束 | 显示结算弹窗，3秒后自动关闭 |
| 所有参与者被追上 | 显示失败信息 |
| 逃跑者成功逃脱 | 显示胜利信息 |
| 压力值达到100% | 显示警告，自动失败 |
| 当前参与者不是玩家 | 禁用操作按钮，显示等待 |

### 5.3 加载状态

```typescript
{isExecutingAction && (
  <Button disabled>
    <Loader className="animate-spin mr-2" />
    执行中...
  </Button>
)}
```

---

## 6. 动画效果

| 场景 | 动画效果 |
|------|----------|
| 距离变化 | 横向轨道的间隔符数量过渡 |
| 速度变化 | 徽章数字滚动动画 |
| 新日志 | 淡入效果 (fade-in) |
| 障碍物出现 | 卡片滑入效果 (slide-in) |
| 检定成功 | 绿色闪烁 |
| 检定失败 | 红色闪烁 + 震动 |

---

## 7. 验收标准

### M1-093 ChaseTracker 组件
- [ ] 全屏覆盖层正常显示/关闭
- [ ] 三栏布局响应式适配
- [ ] ESC键关闭功能
- [ ] 最小化功能

### M1-094 距离可视化
- [ ] 横向轨道图正确显示距离等级
- [ ] 距离变化时有动画效果
- [ ] 逃跑者/追逐者位置正确标识

### M1-095 压力指示器
- [ ] 压力值用进度条显示
- [ ] 压力达到100%时显示警告
- [ ] 压力颜色随值变化（绿/黄/红）

### M1-096 障碍物展示
- [ ] 障碍物卡片正确显示信息
- [ ] 检定结果内嵌显示在面板中
- [ ] 障碍物克服后自动移除
- [ ] 上下文动态切换正常工作

---

## 8. 测试计划

### 单元测试
- `useChaseState` hook - WebSocket监听逻辑
- `useChaseActions` hook - API调用和错误处理
- `ChaseInfoPanel` - 距离和速度显示逻辑
- `ChaseActionPanel` - 上下文动态切换逻辑
- `DistanceTrack` - 轨道图渲染逻辑
- `ParticipantList` - 参与者列表渲染

### 集成测试
- 完整追逐流程（开始→多回合→结束）
- 障碍物检定流程
- WebSocket状态同步
- 错误处理和重试机制

### E2E测试
- 用户发起加速 → 后端处理 → 前端更新
- 障碍物出现 → 用户检定 → 结果显示
- 追逐结束 → 覆盖层关闭
- 压力达到100% → 自动失败

---

## 9. 实施计划

### 阶段1：基础架构 (2h)
- [ ] 创建 `frontend/src/types/chase.ts` 类型定义
- [ ] 创建 `frontend/src/services/chase.ts` API服务
- [ ] 创建 `frontend/src/hooks/useChaseState.ts`
- [ ] 创建 `frontend/src/hooks/useChaseActions.ts`

### 阶段2：信息面板 (2h)
- [ ] 实现 `ChaseInfoPanel` 组件
- [ ] 实现 `DistanceTrack` 横向轨道图
- [ ] 实现 `PressureBar` 压力进度条
- [ ] 实现 `ParticipantList` 参与者列表

### 阶段3：行动面板 (3h)
- [ ] 实现 `ChaseActionPanel` 组件
- [ ] 实现 `ActionSelector` 行动选择器
- [ ] 实现 `ObstacleCard` 障碍物卡片
- [ ] 实现 `CheckResult` 检定结果显示

### 阶段4：日志与主组件 (2h)
- [ ] 实现 `ChaseLogPanel` 日志面板
- [ ] 实现 `ChaseOverlay` 主容器
- [ ] 集成到 `GameConsole`

### 阶段5：测试与优化 (1h)
- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 添加动画效果
- [ ] 响应式适配

---

**预估总工时**: 10小时
**风险**: 低 - 后端已完成，只需前端实现
**依赖**: WebSocket连接稳定，后端API正常工作
