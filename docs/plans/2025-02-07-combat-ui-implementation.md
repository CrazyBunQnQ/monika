# Combat System Frontend UI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the combat system frontend UI for Monika's CoC 7e TRPG platform, enabling players to view and interact with turn-based combat sessions.

**Architecture:**
- Full-screen overlay layer with semi-transparent background over the game console
- Three-column layout: turn info (left), actions (center), combat log (right)
- React 19 + TypeScript + shadcn/ui components
- REST API calls to backend combat endpoints
- WebSocket integration for real-time combat updates

**Tech Stack:**
- React 19, TypeScript, Vite
- shadcn/ui (Card, Button, Dialog, Badge, ScrollArea, Progress)
- TailwindCSS for styling
- Axios for API calls
- WebSocket for real-time updates

---

## Prerequisites

**Worktree Location:** `d:\git\monika\.worktrees\combat-ui`
**Branch:** `feature/combat-ui`
**Backend Tests:** 147 passing (verified baseline)

**Key Backend Endpoints (Already Implemented):**
| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/combat/start` | Create combat session |
| GET | `/combat/{id}` | Get combat summary |
| POST | `/combat/{id}/turn` | Advance to next turn |
| POST | `/combat/{id}/attack` | Resolve attack |
| POST | `/combat/{id}/heal` | Heal combatant |
| POST | `/combat/{id}/end` | End combat |

**Related Documentation:**
- Design doc: `docs/plans/2025-02-07-combat-ui-design.md`
- Backend combat service: `backend/src/services/combat.py`
- Backend combat API: `backend/src/api/combat.py`

---

## Task 1: Create Combat Types and API Client

**Files:**
- Create: `frontend/src/types/combat.ts`
- Modify: `frontend/src/lib/api.ts`

**Step 1: Write combat types**

Create `frontend/src/types/combat.ts`:

```typescript
/**
 * Combat system types for CoC 7e turn-based combat
 */

export type CombatantRole = 'pc' | 'npc' | 'ally'
export type CombatState = 'active' | 'paused' | 'ended'
export type DamageType = 'lethal' | 'non_lethal'
export type SuccessLevel = 'extreme' | 'hard' | 'regular' | 'failure'

export interface Combatant {
  id: string
  name: string
  role: CombatantRole
  initiative: number
  dex: number
  hp: number
  hp_max: number
  is_active: boolean
  is_dying: boolean
  has_major_wound: boolean
  is_unconscious: boolean
  position?: string
  character_id?: number
}

export interface Combat {
  id: string
  state: CombatState
  round: number
  location?: string
  description?: string
  started_at?: string
  ended_at?: string
  combatants: Combatant[]
  current_turn?: Combatant
  total_actions?: number
}

export interface TurnResponse {
  combat_id: string
  current_round: number
  current_turn_index: number
  current_combatant: Combatant | null
  is_new_round: boolean
  turn_order: Combatant[]
}

export interface AttackRequest {
  attacker_id: string
  target_id: string
  attack_skill: number
  attack_roll?: number
  damage_roll?: number
  damage_bonus: number
}

export interface AttackResponse {
  attacker: string
  target: string
  attack_roll: number
  attack_skill: number
  success_level: SuccessLevel
  hit: boolean
  damage: number
  target_hp_before: number
  target_hp_after: number
  target_status: 'active' | 'dying' | 'dead'
  action_id: string
}

export interface HealRequest {
  target_id: string
  heal_amount: number
  first_aid_skill: number
  first_aid_roll?: number
}

export interface HealResponse {
  target: string
  first_aid_roll: number
  first_aid_skill: number
  success_level: SuccessLevel
  hp_before: number
  healing: number
  hp_after: number
  action_id: string
}

export interface CombatCreateRequest {
  session_id: string
  location?: string
  description?: string
}

export interface CombatantCreateRequest {
  name: string
  hp: number
  hp_max: number
  dex: number
  role: CombatantRole
  character_id?: number
}

export interface CombatLogEntry {
  id: string
  round: number
  type: 'attack' | 'heal' | 'turn_start' | 'combat_end'
  actor?: string
  target?: string
  description: string
  success_level?: SuccessLevel
  damage?: number
  healing?: number
  timestamp: Date
}
```

**Step 2: Add combat API client**

Add to `frontend/src/lib/api.ts` (after line 162):

```typescript
// Combat API types
export type {
  Combat,
  Combatant,
  TurnResponse,
  AttackRequest,
  AttackResponse,
  HealRequest,
  HealResponse,
  CombatCreateRequest,
  CombatantCreateRequest,
  CombatLogEntry,
  CombatState,
  CombatantRole,
  SuccessLevel,
} from '../types/combat'

// Combat API
export const combatApi = {
  // Start new combat session
  start: async (data: CombatCreateRequest): Promise<Combat> => {
    const response = await api.post<Combat>('/combat/start', data)
    return response.data
  },

  // Get combat summary
  getById: async (id: string): Promise<Combat> => {
    const response = await api.get<Combat>(`/combat/${id}`)
    return response.data
  },

  // Get turn order
  getTurnOrder: async (id: string): Promise<Combatant[]> => {
    const response = await api.get<Combatant[]>(`/combat/${id}/turn-order`)
    return response.data
  },

  // Advance to next turn
  nextTurn: async (id: string): Promise<TurnResponse> => {
    const response = await api.post<TurnResponse>(`/combat/${id}/turn`)
    return response.data
  },

  // Resolve attack
  attack: async (id: string, data: AttackRequest): Promise<AttackResponse> => {
    const response = await api.post<AttackResponse>(`/combat/${id}/attack`, data)
    return response.data
  },

  // Heal combatant
  heal: async (id: string, data: HealRequest): Promise<HealResponse> => {
    const response = await api.post<HealResponse>(`/combat/${id}/heal`, data)
    return response.data
  },

  // Add combatant
  addCombatant: async (id: string, data: CombatantCreateRequest): Promise<Combatant> => {
    const response = await api.post<Combatant>(`/combat/${id}/combatants`, data)
    return response.data
  },

  // End combat
  end: async (id: string): Promise<Combat> => {
    const response = await api.post<Combat>(`/combat/${id}/end`)
    return response.data
  },
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 4: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/types/combat.ts frontend/src/lib/api.ts
git commit -m "feat(M1-080): add combat types and API client"
```

---

## Task 2: Create Combat State Management Hook

**Files:**
- Create: `frontend/src/hooks/useCombatState.ts`
- Create: `frontend/src/hooks/useCombatActions.ts`

**Step 1: Write useCombatState hook**

Create `frontend/src/hooks/useCombatState.ts`:

```typescript
import { useState, useEffect, useCallback } from 'react'
import type { Combat, Combatant, TurnResponse } from '../types/combat'
import { combatApi } from '../lib/api'

export function useCombatState(combatId: string | null) {
  const [combat, setCombat] = useState<Combat | null>(null)
  const [combatants, setCombatants] = useState<Combatant[]>([])
  const [currentTurn, setCurrentTurn] = useState<Combatant | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Fetch combat data
  const fetchCombat = useCallback(async () => {
    if (!combatId) return

    setIsLoading(true)
    setError(null)
    try {
      const data = await combatApi.getById(combatId)
      setCombat(data)
      setCombatants(data.combatants || [])
      setCurrentTurn(data.current_turn || null)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch combat'
      setError(message)
      console.error('Error fetching combat:', err)
    } finally {
      setIsLoading(false)
    }
  }, [combatId])

  // Fetch on combatId change
  useEffect(() => {
    fetchCombat()
  }, [fetchCombat])

  // Update combatants from turn response
  const updateFromTurnResponse = useCallback((response: TurnResponse) => {
    setCombatants(response.turn_order)
    setCurrentTurn(response.current_combatant)
    if (combat) {
      setCombat({
        ...combat,
        current_round: response.current_round,
      })
    }
  }, [combat])

  // Update single combatant (e.g., after damage)
  const updateCombatant = useCallback((updatedCombatant: Combatant) => {
    setCombatants(prev =>
      prev.map(c => c.id === updatedCombatant.id ? updatedCombatant : c)
    )
    if (currentTurn?.id === updatedCombatant.id) {
      setCurrentTurn(updatedCombatant)
    }
  }, [currentTurn?.id])

  return {
    combat,
    combatants,
    currentTurn,
    isLoading,
    error,
    fetchCombat,
    updateFromTurnResponse,
    updateCombatant,
  }
}
```

**Step 2: Write useCombatActions hook**

Create `frontend/src/hooks/useCombatActions.ts`:

```typescript
import { useCallback } from 'react'
import { combatApi } from '../lib/api'
import type { AttackRequest, HealRequest, TurnResponse, AttackResponse, HealResponse } from '../types/combat'

export function useCombatActions(combatId: string | null) {
  const nextTurn = useCallback(async (): Promise<TurnResponse> => {
    if (!combatId) throw new Error('No combat ID provided')

    const response = await combatApi.nextTurn(combatId)
    return response
  }, [combatId])

  const attack = useCallback(async (request: AttackRequest): Promise<AttackResponse> => {
    if (!combatId) throw new Error('No combat ID provided')

    const response = await combatApi.attack(combatId, request)
    return response
  }, [combatId])

  const heal = useCallback(async (request: HealRequest): Promise<HealResponse> => {
    if (!combatId) throw new Error('No combat ID provided')

    const response = await combatApi.heal(combatId, request)
    return response
  }, [combatId])

  const endCombat = useCallback(async () => {
    if (!combatId) throw new Error('No combat ID provided')

    const response = await combatApi.end(combatId)
    return response
  }, [combatId])

  return {
    nextTurn,
    attack,
    heal,
    endCombat,
  }
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 4: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/hooks/useCombatState.ts frontend/src/hooks/useCombatActions.ts
git commit -m "feat(M1-080): add combat state and actions hooks"
```

---

## Task 3: Create Combatant Card Component

**Files:**
- Create: `frontend/src/components/combat/CombatantCard.tsx`

**Step 1: Create combat directory**

Run: `mkdir -p frontend/src/components/combat`

**Step 2: Write CombatantCard component**

Create `frontend/src/components/combat/CombatantCard.tsx`:

```typescript
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Skull, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Combatant, CombatantRole } from '@/types/combat'

interface CombatantCardProps {
  combatant: Combatant
  isCurrentTurn?: boolean
  isSelected?: boolean
  onSelect?: () => void
  showHpChange?: boolean
  previousHp?: number
  className?: string
}

function getRoleBadgeVariant(role: CombatantRole): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (role) {
    case 'pc':
      return 'default'
    case 'ally':
      return 'secondary'
    case 'npc':
      return 'destructive'
  }
}

function getRoleLabel(role: CombatantRole): string {
  switch (role) {
    case 'pc':
      return 'PC'
    case 'ally':
      return 'Ally'
    case 'npc':
      return 'NPC'
  }
}

function getHpColor(hp: number, hpMax: number): string {
  const percent = hp / hpMax
  if (percent <= 0.25) return 'bg-red-600'
  if (percent <= 0.5) return 'bg-orange-600'
  return 'bg-green-600'
}

export function CombatantCard({
  combatant,
  isCurrentTurn = false,
  isSelected = false,
  onSelect,
  showHpChange = false,
  previousHp,
  className,
}: CombatantCardProps) {
  const hpPercent = (combatant.hp / combatant.hp_max) * 100
  const hpColor = getHpColor(combatant.hp, combatant.hp_max)
  const hpChange = previousHp !== undefined ? combatant.hp - previousHp : 0

  return (
    <Card
      onClick={onSelect}
      className={cn(
        'transition-all duration-200 cursor-pointer',
        isCurrentTurn && 'ring-2 ring-green-500 ring-offset-2',
        isSelected && 'ring-2 ring-blue-500 ring-offset-2',
        !combatant.is_active && 'opacity-50',
        onSelect && 'hover:shadow-md',
        className
      )}
    >
      <CardContent className="p-3">
        {/* Header: Name + Role + Status */}
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm">{combatant.name}</span>
            <Badge variant={getRoleBadgeVariant(combatant.role)} className="text-xs">
              {getRoleLabel(combatant.role)}
            </Badge>
          </div>

          {/* Status Icons */}
          <div className="flex items-center gap-1">
            {combatant.is_dying && (
              <Skull className="h-4 w-4 text-red-600 dark:text-red-400" title="Dying" />
            )}
            {combatant.has_major_wound && (
              <Zap className="h-4 w-4 text-orange-600 dark:text-orange-400" title="Major Wound" />
            )}
            {combatant.is_unconscious && (
              <span className="text-xs">😵</span>
            )}
          </div>
        </div>

        {/* Initiative */}
        <div className="text-xs text-muted-foreground mb-2">
          Initiative: {combatant.initiative} (DEX: {combatant.dex})
        </div>

        {/* HP Bar */}
        <div className="space-y-1">
          <div className="flex items-center justify-between text-xs">
            <span>HP</span>
            <div className="flex items-center gap-1">
              {showHpChange && hpChange !== 0 && (
                <span
                  className={cn(
                    'font-bold text-xs',
                    hpChange < 0 ? 'text-red-600' : 'text-green-600'
                  )}
                >
                  {hpChange > 0 ? '+' : ''}{hpChange}
                </span>
              )}
              <span className="font-medium">{combatant.hp}/{combatant.hp_max}</span>
            </div>
          </div>
          <Progress
            value={hpPercent}
            className={cn(
              'h-2',
              hpPercent <= 25 && 'animate-pulse'
            )}
          />
        </div>

        {/* Position (if any) */}
        {combatant.position && (
          <div className="text-xs text-muted-foreground mt-1">
            Position: {combatant.position}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 4: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/combat/CombatantCard.tsx
git commit -m "feat(M1-080): add CombatantCard component"
```

---

## Task 4: Create Initiative List Component (Round Indicator)

**Files:**
- Create: `frontend/src/components/combat/InitiativeList.tsx`

**Step 1: Write InitiativeList component**

Create `frontend/src/components/combat/InitiativeList.tsx`:

```typescript
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { CombatantCard } from './CombatantCard'
import type { Combatant, TurnResponse } from '@/types/combat'

interface InitiativeListProps {
  combatId: string
  combatants: Combatant[]
  currentTurn: Combatant | null
  currentRound: number
  onNextTurn: () => Promise<TurnResponse>
  isLoading?: boolean
  className?: string
}

export function InitiativeList({
  combatId,
  combatants,
  currentTurn,
  currentRound,
  onNextTurn,
  isLoading = false,
  className,
}: InitiativeListProps) {
  const handleNextTurn = async () => {
    await onNextTurn()
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-semibold text-center">
          Round {currentRound}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {/* Current Turn */}
        {currentTurn && (
          <div className="space-y-2">
            <div className="text-xs font-medium text-muted-foreground">Current Turn</div>
            <CombatantCard
              combatant={currentTurn}
              isCurrentTurn={true}
            />
          </div>
        )}

        {/* Initiative Order */}
        <div className="space-y-2">
          <div className="text-xs font-medium text-muted-foreground">Initiative Order</div>
          <ScrollArea className="h-48">
            <div className="space-y-2 pr-4">
              {combatants.map((combatant, index) => (
                <div key={combatant.id} className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground w-6">
                    {index + 1}.
                  </span>
                  <div className="flex-1 min-w-0">
                    <div
                      className={cn(
                        'text-sm truncate',
                        combatant.id === currentTurn?.id && 'font-medium text-green-600 dark:text-green-400'
                      )}
                    >
                      {combatant.name}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      DEX: {combatant.dex} HP: {combatant.hp}/{combatant.hp_max}
                    </div>
                  </div>
                  {!combatant.is_active && (
                    <span className="text-xs">💀</span>
                  )}
                </div>
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Next Turn Button */}
        <Button
          onClick={handleNextTurn}
          disabled={isLoading}
          className="w-full"
          variant="default"
        >
          {isLoading ? 'Processing...' : 'Next Turn'}
        </Button>
      </CardContent>
    </Card>
  )
}

// Import cn utility
import { cn } from '@/lib/utils'
```

**Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 3: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/combat/InitiativeList.tsx
git commit -m "feat(M1-081): add InitiativeList component with round indicator"
```

---

## Task 5: Create Combat Action Panel

**Files:**
- Create: `frontend/src/components/combat/CombatActionPanel.tsx`
- Create: `frontend/src/components/combat/AttackDialog.tsx`

**Step 1: Write AttackDialog component**

Create `frontend/src/components/combat/AttackDialog.tsx`:

```typescript
import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Combatant, AttackRequest, SuccessLevel } from '@/types/combat'

interface AttackDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: (request: AttackRequest) => Promise<void>
  attacker: Combatant
  combatants: Combatant[]
  isLoading?: boolean
}

export function AttackDialog({
  open,
  onClose,
  onConfirm,
  attacker,
  combatants,
  isLoading = false,
}: AttackDialogProps) {
  const [targetId, setTargetId] = useState<string>('')
  const [attackSkill, setAttackSkill] = useState<string>('50')
  const [damageBonus, setDamageBonus] = useState<string>('0')

  // Filter: can't attack self, only active targets
  const validTargets = combatants.filter(
    c => c.id !== attacker.id && c.is_active
  )

  const handleSubmit = async () => {
    if (!targetId) return

    const request: AttackRequest = {
      attacker_id: attacker.id,
      target_id: targetId,
      attack_skill: parseInt(attackSkill),
      damage_bonus: parseInt(damageBonus),
    }

    await onConfirm(request)
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Attack Roll</DialogTitle>
          <DialogDescription>
            {attacker.name} is attacking. Choose target and confirm skill.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Target Selection */}
          <div className="space-y-2">
            <Label htmlFor="target">Target</Label>
            <Select value={targetId} onValueChange={setTargetId}>
              <SelectTrigger id="target">
                <SelectValue placeholder="Select target" />
              </SelectTrigger>
              <SelectContent>
                {validTargets.map(target => (
                  <SelectItem key={target.id} value={target.id}>
                    {target.name} (HP: {target.hp}/{target.hp_max})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Attack Skill */}
          <div className="space-y-2">
            <Label htmlFor="skill">Attack Skill</Label>
            <Input
              id="skill"
              type="number"
              min="0"
              max="100"
              value={attackSkill}
              onChange={(e) => setAttackSkill(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Enter your attack skill value (e.g., Fighting 60)
            </p>
          </div>

          {/* Damage Bonus */}
          <div className="space-y-2">
            <Label htmlFor="db">Damage Bonus (DB)</Label>
            <Input
              id="db"
              type="number"
              min="0"
              value={damageBonus}
              onChange={(e) => setDamageBonus(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Damage Bonus from Strength (usually 0, +1, +1d4, or +1d6)
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={!targetId || isLoading}
          >
            {isLoading ? 'Rolling...' : 'Roll Attack'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

**Step 2: Write CombatActionPanel component**

Create `frontend/src/components/combat/CombatActionPanel.tsx`:

```typescript
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { AttackDialog } from './AttackDialog'
import { CombatantCard } from './CombatantCard'
import type { Combatant, AttackRequest, AttackResponse } from '@/types/combat'

interface CombatActionPanelProps {
  currentTurn: Combatant | null
  combatants: Combatant[]
  onAttack: (request: AttackRequest) => Promise<AttackResponse>
  onDodge?: (combatantId: string) => Promise<void>
  onHeal?: (targetId: string, amount: number) => Promise<void>
  onEndTurn?: () => Promise<void>
  isLoading?: boolean
  className?: string
}

export function CombatActionPanel({
  currentTurn,
  combatants,
  onAttack,
  onDodge,
  onHeal,
  onEndTurn,
  isLoading = false,
  className,
}: CombatActionPanelProps) {
  const [attackDialogOpen, setAttackDialogOpen] = useState(false)
  const [selectedTargetId, setSelectedTargetId] = useState<string | null>(null)

  const handleAttack = async (request: AttackRequest) => {
    await onAttack(request)
    setSelectedTargetId(null)
  }

  if (!currentTurn) {
    return (
      <Card className={className}>
        <CardContent className="p-6 text-center text-muted-foreground">
          No active turn
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-semibold">
          Current: {currentTurn.name}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {/* Action Buttons */}
        <div className="grid grid-cols-2 gap-2">
          <Button
            onClick={() => setAttackDialogOpen(true)}
            disabled={isLoading}
            className="w-full"
            size="lg"
          >
            ⚔️ Attack
          </Button>
          <Button
            onClick={() => onDodge?.(currentTurn.id)}
            disabled={isLoading || !onDodge}
            variant="secondary"
            className="w-full"
          >
            🛡️ Dodge
          </Button>
          <Button
            onClick={() => onHeal?.(currentTurn.id, 1)}
            disabled={isLoading || !onHeal}
            variant="secondary"
            className="w-full"
          >
            💊 Heal
          </Button>
          <Button
            onClick={onEndTurn}
            disabled={isLoading || !onEndTurn}
            variant="outline"
            className="w-full"
          >
            🔄 End Turn
          </Button>
        </div>

        {/* Target Selection */}
        <div className="space-y-2">
          <div className="text-xs font-medium text-muted-foreground">
            Select Target
          </div>
          <div className="space-y-1">
            {combatants
              .filter(c => c.id !== currentTurn.id && c.is_active)
              .map(combatant => (
                <div
                  key={combatant.id}
                  onClick={() => setSelectedTargetId(
                    selectedTargetId === combatant.id ? null : combatant.id
                  )}
                  className={`
                    cursor-pointer rounded border p-2 text-sm transition-colors
                    ${selectedTargetId === combatant.id
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/30'
                      : 'border-border hover:bg-muted/50'
                    }
                  `}
                >
                  <div className="font-medium">{combatant.name}</div>
                  <div className="text-xs text-muted-foreground">
                    HP: {combatant.hp}/{combatant.hp_max}
                  </div>
                </div>
              ))}
          </div>
        </div>

        {/* Attack Dialog */}
        <AttackDialog
          open={attackDialogOpen}
          onClose={() => setAttackDialogOpen(false)}
          onConfirm={handleAttack}
          attacker={currentTurn}
          combatants={combatants}
          isLoading={isLoading}
        />
      </CardContent>
    </Card>
  )
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 4: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/combat/AttackDialog.tsx frontend/src/components/combat/CombatActionPanel.tsx
git commit -m "feat(M1-083): add CombatActionPanel and AttackDialog components"
```

---

## Task 6: Create Combat Log Panel

**Files:**
- Create: `frontend/src/components/combat/CombatLogPanel.tsx`

**Step 1: Write CombatLogPanel component**

Create `frontend/src/components/combat/CombatLogPanel.tsx`:

```typescript
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { CombatLogEntry, SuccessLevel } from '@/types/combat'

interface CombatLogPanelProps {
  logs: CombatLogEntry[]
  onClear?: () => void
  className?: string
}

function getSuccessColor(level: SuccessLevel): string {
  switch (level) {
    case 'extreme':
      return 'text-yellow-600 dark:text-yellow-400'
    case 'hard':
      return 'text-blue-600 dark:text-blue-400'
    case 'regular':
      return 'text-green-600 dark:text-green-400'
    case 'failure':
      return 'text-gray-600 dark:text-gray-400'
  }
}

function getSuccessVariant(level: SuccessLevel): 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive' {
  switch (level) {
    case 'extreme':
      return 'warning'
    case 'hard':
      return 'default'
    case 'regular':
      return 'secondary'
    case 'failure':
      return 'outline'
  }
}

export function CombatLogPanel({
  logs,
  onClear,
  className,
}: CombatLogPanelProps) {
  return (
    <Card className={className}>
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-semibold">Combat Log</CardTitle>
        {onClear && logs.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onClear}
            className="h-6 text-xs"
          >
            Clear
          </Button>
        )}
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-64">
          {logs.length === 0 ? (
            <div className="text-center text-sm text-muted-foreground py-8">
              No combat actions yet
            </div>
          ) : (
            <div className="space-y-3 pr-4">
              {logs.map((log) => (
                <div key={log.id} className="space-y-1">
                  {/* Round Header */}
                  {log.type === 'turn_start' && (
                    <div className="text-xs font-semibold text-muted-foreground border-b pb-1">
                      Round {log.round}
                    </div>
                  )}

                  {/* Log Entry */}
                  <div className="text-sm pl-2 border-l-2 border-border">
                    {/* Action Description */}
                    <div
                      className={cn(
                        'font-medium',
                        log.success_level && getSuccessColor(log.success_level)
                      )}
                    >
                      {log.actor && `${log.actor} `}
                      {log.description}
                    </div>

                    {/* Success Badge */}
                    {log.success_level && (
                      <Badge variant={getSuccessVariant(log.success_level)} className="text-xs mt-1">
                        {log.success_level}
                      </Badge>
                    )}

                    {/* Damage/Healing */}
                    {log.damage !== undefined && (
                      <div className="text-xs text-red-600 dark:text-red-400 mt-1">
                        -{log.damage} damage → {log.target} HP: {log.damage}
                      </div>
                    )}
                    {log.healing !== undefined && (
                      <div className="text-xs text-green-600 dark:text-green-400 mt-1">
                        +{log.healing} healing → {log.target}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  )
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 3: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/combat/CombatLogPanel.tsx
git commit -m "feat(M1-084): add CombatLogPanel component"
```

---

## Task 7: Create Main Combat Overlay Component

**Files:**
- Create: `frontend/src/components/combat/CombatOverlay.tsx`

**Step 1: Write CombatOverlay component**

Create `frontend/src/components/combat/CombatOverlay.tsx`:

```typescript
import { useState, useCallback, useEffect } from 'react'
import { X, Minus2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useCombatState } from '@/hooks/useCombatState'
import { useCombatActions } from '@/hooks/useCombatActions'
import { InitiativeList } from './InitiativeList'
import { CombatActionPanel } from './CombatActionPanel'
import { CombatLogPanel } from './CombatLogPanel'
import type { CombatLogEntry } from '@/types/combat'

interface CombatOverlayProps {
  combatId: string
  onClose: () => void
  onMinimize?: () => void
}

export function CombatOverlay({ combatId, onClose, onMinimize }: CombatOverlayProps) {
  const [logs, setLogs] = useState<CombatLogEntry[]>([])
  const [previousHpMap, setPreviousHpMap] = useState<Record<string, number>>({})

  const {
    combat,
    combatants,
    currentTurn,
    isLoading,
    error,
    updateFromTurnResponse,
    updateCombatant,
  } = useCombatState(combatId)

  const { nextTurn, attack, heal } = useCombatActions(combatId)

  // Add log entry
  const addLog = useCallback((entry: CombatLogEntry) => {
    setLogs(prev => [entry, ...prev])
  }, [])

  // Handle next turn
  const handleNextTurn = useCallback(async () => {
    const response = await nextTurn()
    updateFromTurnResponse(response)

    // Add round start log
    if (response.is_new_round) {
      addLog({
        id: `turn-${response.current_round}`,
        round: response.current_round,
        type: 'turn_start',
        description: `Round ${response.current_round} begins`,
        timestamp: new Date(),
      })
    }
  }, [nextTurn, updateFromTurnResponse, addLog])

  // Handle attack
  const handleAttack = useCallback(async (request: import('@/types/combat').AttackRequest) => {
    // Store previous HP for animation
    const target = combatants.find(c => c.id === request.target_id)
    if (target) {
      setPreviousHpMap(prev => ({ ...prev, [target.id]: target.hp }))
    }

    const response = await attack(request)

    // Update combatant with new HP
    const updatedCombatant = combatants.find(c => c.id === request.target_id)
    if (updatedCombatant) {
      updateCombatant({
        ...updatedCombatant,
        hp: response.target_hp_after,
        is_dying: response.target_status === 'dying',
        is_active: response.target_status === 'active',
      })
    }

    // Add log entry
    addLog({
      id: response.action_id,
      round: combat?.current_round || 1,
      type: 'attack',
      actor: response.attacker,
      target: response.target,
      description: `attacks ${response.target}`,
      success_level: response.success_level,
      damage: response.damage,
      timestamp: new Date(),
    })

    return response
  }, [attack, combatants, combat?.current_round, updateCombatant, addLog])

  // Close on ESC key
  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose])

  if (error) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
        <div className="bg-background p-6 rounded-lg shadow-lg max-w-md">
          <h2 className="text-lg font-semibold mb-2">Combat Error</h2>
          <p className="text-muted-foreground">{error}</p>
          <Button onClick={onClose} className="mt-4">
            Close
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-4">
      <div className="bg-background rounded-lg shadow-xl w-full max-w-5xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div>
            <h2 className="text-lg font-semibold">
              {combat?.location || 'Combat'}
            </h2>
            {combat?.description && (
              <p className="text-sm text-muted-foreground">{combat.description}</p>
            )}
          </div>
          <div className="flex items-center gap-2">
            {onMinimize && (
              <Button variant="ghost" size="icon" onClick={onMinimize}>
                <Minus2 className="h-4 w-4" />
              </Button>
            )}
            <Button variant="ghost" size="icon" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Three-Column Layout */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 p-4">
          {/* Left: Turn Info */}
          <div>
            {combat && currentTurn && (
              <InitiativeList
                combatId={combatId}
                combatants={combatants}
                currentTurn={currentTurn}
                currentRound={combat.round}
                onNextTurn={handleNextTurn}
                isLoading={isLoading}
              />
            )}
          </div>

          {/* Center: Actions */}
          <div>
            {currentTurn && (
              <CombatActionPanel
                currentTurn={currentTurn}
                combatants={combatants}
                onAttack={handleAttack}
                onEndTurn={handleNextTurn}
                isLoading={isLoading}
              />
            )}
          </div>

          {/* Right: Combat Log */}
          <div>
            <CombatLogPanel
              logs={logs}
              onClear={() => setLogs([])}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 3: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/combat/CombatOverlay.tsx
git commit -m "feat(M1-080): add CombatOverlay main component"
```

---

## Task 8: Add Damage Animation

**Files:**
- Create: `frontend/src/components/combat/CombatantCard.tsx` (modify)

**Step 1: Add animation styles to globals.css**

Add to `frontend/src/index.css` (at end of file):

```css
/* Combat damage animations */
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-4px); }
  75% { transform: translateX(4px); }
}

@keyframes float-up-fade-out {
  0% { opacity: 1; transform: translateY(0); }
  100% { opacity: 0; transform: translateY(-20px); }
}

.damage-shake {
  animation: shake 0.3s ease-in-out;
}

.damage-float {
  animation: float-up-fade-out 1s ease-out forwards;
  position: absolute;
  font-weight: bold;
  pointer-events: none;
}
```

**Step 2: Update CombatantCard with damage animation**

Modify `frontend/src/components/combat/CombatantCard.tsx` to add damage animation:

Replace the HP bar section (around line 130-145) with:

```typescript
        {/* HP Bar */}
        <div className="space-y-1 relative">
          <div className="flex items-center justify-between text-xs">
            <span>HP</span>
            <div className="flex items-center gap-1">
              {/* Damage Float Animation */}
              {showHpChange && hpChange < 0 && (
                <span className="damage-float text-red-600 dark:text-red-400 text-sm absolute -top-4 right-0">
                  {hpChange}💔
                </span>
              )}
              {showHpChange && hpChange > 0 && (
                <span className="damage-float text-green-600 dark:text-green-400 text-sm absolute -top-4 right-0">
                  +{hpChange}💚
                </span>
              )}
              <span className="font-medium">{combatant.hp}/{combatant.hp_max}</span>
            </div>
          </div>
          <Progress
            value={hpPercent}
            className={cn(
              'h-2 transition-all duration-500',
              hpPercent <= 25 && 'animate-pulse',
              showHpChange && hpChange < 0 && 'damage-shake'
            )}
          />
        </div>
```

**Step 3: Verify CSS and TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 4: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/index.css frontend/src/components/combat/CombatantCard.tsx
git commit -m "feat(M1-084): add damage shake and float animations"
```

---

## Task 9: Integrate CombatOverlay into GameConsole

**Files:**
- Modify: `frontend/src/components/GameConsole.tsx`

**Step 1: Add combat state to GameConsole**

Modify `frontend/src/components/GameConsole.tsx`:

Add combat state after existing state (around line 71):

```typescript
  const [world, setWorld] = useState<WorldState>({
    currentScene: "陌生的房间",
    location: "未知地点",
    timer: undefined,
    leads: [
      { id: "1", text: "调查房间环境", verified: false },
      { id: "2", text: "检查门锁", verified: false },
      { id: "3", text: "搜寻物品", verified: false },
    ],
  })

  // Combat state
  const [activeCombatId, setActiveCombatId] = useState<string | null>(null)
  const [isCombatMinimized, setIsCombatMinimized] = useState(false)

  const handleCombatClose = () => {
    setActiveCombatId(null)
    setIsCombatMinimized(false)
  }

  const handleCombatMinimize = () => {
    setIsCombatMinimized(true)
  }

  const handleCombatExpand = () => {
    setIsCombatMinimized(false)
  }
```

**Step 2: Add CombatOverlay and minimized card to return JSX**

Replace the return statement (around line 216-239) with:

```typescript
  return (
    <div className="flex flex-col h-screen">
      <Header characterName="调查员" />
      <div className="flex-1 flex overflow-hidden">
        <div className="flex-1 flex flex-col min-w-0">
          <MessageList messages={messages} />
          <Footer onSendMessage={handleSendMessage} />
        </div>
        <StatePanel
          character={{
            id: 1,
            name: user?.username || "调查员",
            hp: character.hp,
            maxHp: character.hpMax,
            mp: character.mp,
            maxMp: character.mpMax,
            san: character.san,
            maxSan: character.sanMax,
            luck: character.luck,
          }}
        />
      </div>

      {/* Combat Overlay */}
      {activeCombatId && !isCombatMinimized && (
        <CombatOverlay
          combatId={activeCombatId}
          onClose={handleCombatClose}
          onMinimize={handleCombatMinimize}
        />
      )}

      {/* Minimized Combat Card */}
      {activeCombatId && isCombatMinimized && combat && (
        <div className="fixed bottom-4 right-4 bg-background border rounded-lg shadow-lg p-3 z-40 w-48">
          <div className="text-xs text-muted-foreground mb-1">Combat Active</div>
          <div className="font-semibold text-sm mb-1">Round {combat.round}</div>
          <div className="text-xs mb-2">{currentTurn?.name}'s turn</div>
          <Button
            size="sm"
            variant="secondary"
            className="w-full text-xs"
            onClick={handleCombatExpand}
          >
            ▶ Expand
          </Button>
        </div>
      )}
    </div>
  )
```

**Step 3: Add import for CombatOverlay**

Add to imports at top of file:

```typescript
import { CombatOverlay } from "@/components/combat/CombatOverlay"
```

**Step 4: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 5: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/GameConsole.tsx
git commit -m "feat(M1-080): integrate CombatOverlay into GameConsole"
```

---

## Task 10: Export Components from Index

**Files:**
- Create: `frontend/src/components/combat/index.ts`

**Step 1: Create barrel export file**

Create `frontend/src/components/combat/index.ts`:

```typescript
// Combat system components
export { CombatOverlay } from './CombatOverlay'
export { CombatantCard } from './CombatantCard'
export { InitiativeList } from './InitiativeList'
export { CombatActionPanel } from './CombatActionPanel'
export { AttackDialog } from './AttackDialog'
export { CombatLogPanel } from './CombatLogPanel'
```

**Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 3: Commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git add frontend/src/components/combat/index.ts
git commit -m "feat(M1-080): add combat component barrel exports"
```

---

## Task 11: Frontend Build Verification

**Files:** None (verification only)

**Step 1: Run full TypeScript check**

Run: `cd frontend && npx tsc --noEmit`

Expected: No type errors

**Step 2: Run frontend build**

Run: `cd frontend && npm run build`

Expected: Build completes successfully with no errors

**Step 3: Run backend tests (ensure no regression)**

Run: `cd backend && uv run pytest --tb=short`

Expected: All 147 tests pass

**Step 4: Final commit**

```bash
cd d:/git/monika/.worktrees/combat-ui
git commit --allow-empty -m "feat(M1-080~084): combat UI implementation complete"
```

---

## Acceptance Criteria Verification

After implementation, verify:

- [ ] Combat overlay appears when `activeCombatId` is set
- [ ] Round indicator shows current round number
- [ ] Initiative list displays all combatants sorted by DEX
- [ ] Current turn combatant is highlighted with green border
- [ ] "Next Turn" button advances turn correctly
- [ ] Attack dialog opens and captures target/skill input
- [ ] Attack results display in combat log with success level
- [ ] Damage animation plays (shake + float-up number)
- [ ] HP updates reflect in UI immediately after damage
- [ ] Overlay minimizes to floating card in bottom-right
- [ ] ESC key closes the overlay
- [ ] All TypeScript compiles without errors
- [ ] Frontend builds successfully

---

## Summary

**Total Tasks:** 11
**Estimated Time:** 12 hours
**Files Created:** 10
**Files Modified:** 3

**New Components:**
- `CombatOverlay` - Main overlay container
- `CombatantCard` - Individual combatant display with HP bar
- `InitiativeList` - Turn order and round indicator
- `CombatActionPanel` - Action buttons (attack/dodge/heal)
- `AttackDialog` - Attack roll modal dialog
- `CombatLogPanel` - Combat action log

**New Hooks:**
- `useCombatState` - Combat state management
- `useCombatActions` - Combat API actions

**Animations Added:**
- Damage shake (CSS keyframes)
- HP float-up/fade-out
- HP progress bar smooth transition

---

**End of Implementation Plan**

Use `superpowers:executing-plans` skill to execute this plan task-by-task.
