# M1-019: 实现 StatePanel 状态面板组件

**任务ID**: M1-019
**标题**: 实现 StatePanel 状态面板组件
**类型**: frontend (前端开发)
**预估工时**: 2.5h
**依赖**: M1-032

---

## 任务描述

实现游戏状态面板组件，用于显示和编辑角色属性、状态、HP、SAN 等核心数据。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-019-01 | 设计状态面板结构 | 布局设计 | 20min |
| M1-019-02 | 实现属性显示组件 | STR/DEX/POW 等 | 30min |
| M1-019-03 | 实现衍生属性 | HP/MP/SAN 等 | 25min |
| M1-019-04 | 实现状态编辑功能 | 可编辑字段 | 30min |
| M1-019-05 | 实现状态指示器 | 异常状态 | 20min |
| M1-019-06 | 实现快速操作 | 常用操作 | 25min |
| M1-019-07 | 实现数据同步 | 后端同步 | 20min |

---

## 状态面板组件

```tsx
// frontend/src/components/game/StatePanel.tsx
import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import type { Character } from '@/types/character'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Heart, Brain, Zap, AlertTriangle } from 'lucide-react'

interface StatePanelProps {
  character: Character
  onUpdate?: (updates: Partial<Character>) => void
  readonly?: boolean
}

export function StatePanel({ character, onUpdate, readonly }: StatePanelProps) {
  const { user } = useAuth()
  const [editing, setEditing] = useState(false)
  const [tempValues, setTempValues] = useState(character)

  const isKp = user?.role === 'kp'
  const isOwner = user?.id === character.user_id
  const canEdit = !readonly && (isKp || isOwner)

  // 计算衍生属性
  const hp = character.attributes?.pow || 0
  const mp = character.attributes?.pow || 0
  const san = character.attributes?.pow || 0
  const currentHp = character.status?.hp ?? hp
  const currentMp = character.status?.mp ?? mp
  const currentSan = character.status?.san ?? san

  // 获取状态颜色
  const getStatusColor = (current: number, max: number) => {
    const ratio = current / max
    if (ratio > 0.5) return 'bg-green-500'
    if (ratio > 0.25) return 'bg-yellow-500'
    return 'bg-red-500'
  }

  // 保存更改
  const handleSave = () => {
    onUpdate?.(tempValues)
    setEditing(false)
  }

  // 取消更改
  const handleCancel = () => {
    setTempValues(character)
    setEditing(false)
  }

  return (
    <div className="space-y-4">
      {/* 头部 */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">状态面板</h3>
        {canEdit && !editing && (
          <Button size="sm" onClick={() => setEditing(true)}>
            编辑
          </Button>
        )}
        {editing && (
          <div className="space-x-2">
            <Button size="sm" variant="outline" onClick={handleCancel}>
              取消
            </Button>
            <Button size="sm" onClick={handleSave}>
              保存
            </Button>
          </div>
        )}
      </div>

      {/* 核心属性 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">核心属性</CardTitle>
        </CardHeader>
        <CardContent>
          <AttributesGrid
            attributes={character.attributes || {}}
            editing={editing}
            values={tempValues.attributes || {}}
            onChange={(attrs) => setTempValues({ ...tempValues, attributes: attrs })}
          />
        </CardContent>
      </Card>

      {/* 生命值 */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center">
              <Heart className="h-4 w-4 mr-2 text-red-500" />
              生命值 (HP)
            </CardTitle>
            <span className="text-sm">
              {currentHp} / {hp}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <Progress
            value={(currentHp / hp) * 100}
            className={getStatusColor(currentHp, hp)}
          />
          {currentHp <= 0 && (
            <div className="flex items-center text-red-500 text-sm mt-2">
              <AlertTriangle className="h-4 w-4 mr-1" />
              角色已昏迷
            </div>
          )}
        </CardContent>
      </Card>

      /* 魔法值 */
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center">
              <Zap className="h-4 w-4 mr-2 text-blue-500" />
              魔法值 (MP)
            </CardTitle>
            <span className="text-sm">
              {currentMp} / {mp}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <Progress
            value={(currentMp / mp) * 100}
            className="bg-blue-500"
          />
        </CardContent>
      </Card>

      {/* 理智值 */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center">
              <Brain className="h-4 w-4 mr-2 text-purple-500" />
              理智值 (SAN)
            </CardTitle>
            <span className="text-sm">
              {currentSan} / {san}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <Progress
            value={(currentSan / san) * 100}
            className="bg-purple-500"
          />
          {currentSan < san * 0.2 && (
            <div className="flex items-center text-purple-500 text-sm mt-2">
              <AlertTriangle className="h-4 w-4 mr-1" />
              临时疯狂风险
            </div>
          )}
        </CardContent>
      </Card>

      /* 异常状态 */
      <Card>
        <CardHeader>
          <CardTitle className="text-base">异常状态</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {character.status?.conditions?.map(condition => (
              <Badge key={condition} variant="destructive">
                {condition}
              </Badge>
            )) || (
              <span className="text-sm text-muted-foreground">无异常状态</span>
            )}
          </div>
        </CardContent>
      </Card>

      {/* 快速操作 */}
      {canEdit && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">快速操作</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-2">
              <QuickActionButton
                label="伤害 -1"
                onClick={() => onUpdate?.({
                  status: {
                    ...character.status,
                    hp: Math.max(0, (character.status?.hp || hp) - 1)
                  }
                })}
              />
              <QuickActionButton
                label="治疗 +1"
                onClick={() => onUpdate?.({
                  status: {
                    ...character.status,
                    hp: Math.min(hp, (character.status?.hp || hp) + 1)
                  }
                })}
              />
              <QuickActionButton
                label="SAN -1"
                onClick={() => onUpdate?.({
                  status: {
                    ...character.status,
                    san: Math.max(0, (character.status?.san || san) - 1)
                  }
                })}
              />
              <QuickActionButton
                label="恢复 SAN"
                onClick={() => onUpdate?.({
                  status: {
                    ...character.status,
                    san: san
                  }
                })}
              />
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// 属性网格
interface AttributesGridProps {
  attributes: Record<string, number>
  editing: boolean
  values: Record<string, number>
  onChange: (attrs: Record<string, number>) => void
}

function AttributesGrid({ attributes, editing, values, onChange }: AttributesGridProps) {
  const attrNames = {
    str: '力量',
    con: '体质',
    siz: '体型',
    dex: '敏捷',
    app: '外貌',
    int: '智力',
    pow: '意志',
    edu: '教育',
  }

  return (
    <div className="grid grid-cols-4 gap-3">
      {Object.entries(attrNames).map(([key, label]) => (
        <div key={key} className="text-center">
          <div className="text-xs text-muted-foreground mb-1">{label}</div>
          {editing ? (
            <Input
              type="number"
              value={values[key] || 0}
              onChange={(e) => onChange({
                ...values,
                [key]: parseInt(e.target.value) || 0
              })}
              className="text-center"
              min={0}
              max={100}
            />
          ) : (
            <div className="text-lg font-semibold">
              {attributes[key] || 0}
            </div>
          )}
          <div className="text-xs text-muted-foreground">
            {key.toUpperCase()}
          </div>
        </div>
      ))}
    </div>
  )
}

// 快速操作按钮
function QuickActionButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <Button variant="outline" size="sm" onClick={onClick}>
      {label}
    </Button>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `frontend/src/components/game/StatePanel.tsx` | 创建 | 状态面板主组件 |
| `frontend/src/components/ui/progress.tsx` | 创建 | 进度条组件 |

---

## 验收标准

- [ ] 核心属性正确显示
- [ ] HP/MP/SAN 计算正确
- [ ] 编辑功能有效
- [ ] 状态指示器准确
- [ ] 快速操作可用
- [ ] KP 可编辑他人状态

---

## 参考文档

- M1-003: 角色卡数据模型
- M1-032: GameConsole 布局

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
