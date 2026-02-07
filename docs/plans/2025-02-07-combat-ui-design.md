# Combat System Frontend UI Design

**Date**: 2025-02-07
**Milestone**: M1 - Single-Player Web Version
**Related Tasks**: M1-080 ~ M1-084 (Combat System Frontend UI)

---

## Overview

Design and implement the frontend user interface for the CoC 7e combat system. The backend combat service is already complete; this document covers the frontend components needed to display combat sessions to players.

**Estimated Effort**: 12 hours (5 tasks)

---

## UI Presentation Mode

### Overlay Layer

Combat UI is presented as a **full-screen semi-transparent overlay** (`bg-black/60`) that appears over the existing game console. The overlay uses `z-50` to appear above all content.

**Rationale**: Keeps players in the game context while providing a focused combat interface. The game console remains visible but dimmed, allowing players to reference narrative context.

**Controls**:
- **Minimize**: Reduces to a floating card (200x100px) in bottom-right corner showing current round and turn
- **Close**: Exits overlay (combat automatically pauses via API)

---

## Layout Structure

### Top Control Bar

```
┌─────────────────────────────────────────────────────┐
│  [Combat Location]      Round 3     [−] [×]         │
├─────────────────────────────────────────────────────┤
│                                                       │
│  ┌──────────┐  ┌─────────────┐  ┌──────────────┐   │
│  │          │  │             │  │              │   │
│  │  Turn    │  │   Actions   │  │   Combat     │   │
│  │  Info    │  │             │  │     Log      │   │
│  │          │  │             │  │              │   │
│  └──────────┘  └─────────────┘  └──────────────┘   │
│                                                       │
└─────────────────────────────────────────────────────┘
```

- **Left**: Combat title (location/description from backend)
- **Center**: **Round Indicator** - Large "Round X" display with current turn highlighted
- **Right**: Minimize (`-`) and Close (`×`) buttons

### Three-Column Layout

| Column | Width | Purpose |
|--------|-------|---------|
| Left | 200px | Turn info & initiative list |
| Center | 300px | Action buttons & target selection |
| Right | 300px | Combat action log |

---

## Component: Turn Info Panel (Left)

**Purpose**: Display current round, current turn, and initiative order

```
┌──────────────────┐
│  Round 3 / ???   │
├──────────────────┤
│  Current Turn    │
│  ┌────────────┐  │
│  │ 🎭 Investigator│  │ ← Highlighted border
│  │ HP: 10/12  │  │
│  │ DEX: 65    │  │
│  └────────────┘  │
├──────────────────┤
│  Initiative Order│
│  1. Investigator(65)│ ← Current
│  2. Enemy A (58)   │
│  3. Ally (50)      │
│  4. Enemy B (42)   │
│                  │
│  [Next Turn]     │
└──────────────────┘
```

**Features**:
- Current turn character: Green border highlight
- HP color coding: >50% green, 26-50% orange, ≤25% red (pulsing)
- Status icons: 💀 dying, 😵 unconscious, 🩹 major wound
- Clicking "[Next Turn]" calls `POST /combat/{id}/turn`
- Initiative sorted by DEX roll (highest first, per CoC 7e rules)

---

## Component: Action Panel (Center)

**Purpose**: Execute combat actions (attack, dodge, heal, end turn)

```
┌──────────────────────┐
│   Current: 🎭 Player  │
├──────────────────────┤
│  ┌────────────────┐  │
│  │   ⚔️ Attack    │  │ ← Primary action, large button
│  └────────────────┘  │
│  ┌────────────────┐  │
│  │   🛡️ Dodge     │  │
│  └────────────────┘  │
│  ┌────────────────┐  │
│  │   💊 Heal      │  │
│  └────────────────┘  │
│  ┌────────────────┐  │
│  │   🔄 End Turn  │  │
│  └────────────────┘  │
├──────────────────────┤
│  Select Target        │
│  ┌────────────────┐  │
│  │ ○ Enemy A HP:8 │  │
│  │ ○ Enemy B HP:15│  │ ← Radio selection
│  │ ○ Ally   HP:10 │  │
│  └────────────────┘  │
└──────────────────────┘
```

### Attack Dialog

When clicking "Attack", a modal dialog appears:

```
┌────────────────────────┐
│  Attack Roll           │
├────────────────────────┤
│  Target: Enemy A       │
│  Attack Skill: [Fight(60)] │
│  Bonus/Penalty: [None ▼]   │
│                        │
│  ┌──────┐  ┌─────────┐ │
│  │ Roll │  │ Cancel  │ │
│  └──────┘  └─────────┘ │
└────────────────────────┘
```

**Flow**:
1. Click "Attack" button
2. Dialog opens with target pre-selected (or prompts to select)
3. User confirms attack skill and modifiers
4. Click "Roll" → calls `POST /combat/{id}/attack`
5. Backend returns `AttackResponse` with damage result
6. UI updates: damage animation + log entry + turn advances

---

## Component: Combat Log (Right)

**Purpose**: Display chronological record of combat actions

```
┌──────────────────────┐
│   Combat Log   [Clear]│
├──────────────────────┤
│                      │
│  Round 3             │
│  ──────────────────  │
│  ► Investigator atks EnemyA │
│    Rolled 42/60 Success     │
│    Dealt 5 damage           │
│    EnemyA HP: 8→3          │
│                      │
│  ► EnemyA atks Investigator │
│    Rolled 78/50 Failure     │
│    Missed                   │
│                      │
│  Round 2             │
│  ──────────────────  │
│  ► Ally heals Investigator  │
│    Rolled 23/45 Success      │
│    Restored 3 HP            │
│                      │
└──────────────────────┘
```

**Log Styling**:
- Successful actions: Green text
- Failed actions: Gray text
- Critical success: Gold highlight
- Critical failure: Red flashing
- Scrollable, newest entries at top

---

## Damage Animation

**Purpose**: Visual feedback when a combatant takes damage

```
Animation on injured combatant card:
┌──────────────────┐
│  🎭 Investigator  │
│  HP: 12 → 7       │ ← 12 turns red, shakes
│     ↑-5💔         │ ← Damage number floats up, fades out
│  DEX: 65          │
└──────────────────┘
```

**Animation Effects**:
- `shake` animation: 300ms duration
- Damage number `float-up-fade-out`: 1s duration
- HP progress bar smooth transition: 500ms

**CSS Classes**:
```css
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

@keyframes float-up-fade-out {
  0% { opacity: 1; transform: translateY(0); }
  100% { opacity: 0; transform: translateY(-20px); }
}
```

---

## Component Architecture

```
CombatOverlay (overlay container)
├── CombatHeader (top control bar)
│   ├── RoundIndicator (round display)
│   └── ControlButtons (minimize/close)
├── CombatInfoPanel (left - turn info)
│   ├── CurrentTurnCard (current turn combatant)
│   └── InitiativeList (initiative order)
├── CombatActionPanel (center - actions)
│   ├── ActionButtons (attack/dodge/heal)
│   ├── TargetSelector (target selection)
│   └── AttackDialog (attack modal)
└── CombatLogPanel (right - combat log)
    └── CombatLogEntry (log entry)
```

### Minimized State

Floating card in bottom-right corner (200x100px):
```
┌─────────────────┐
│  Round 3        │
│  Enemy A's turn │
│  [▶ Expand]     │
└─────────────────┘
```

---

## Data Flow

### User Action Flow

```
User clicks "Attack"
  → Opens attack dialog
  → User selects target + skill value
  → Calls POST /combat/{id}/attack
  → Backend returns AttackResponse
  → Update combatants state
  → Trigger damage animation
  → Add combat log entry
  → Update turn indicator
```

### State Management

```typescript
// Combat state hook
useCombatState(combatId: string)
  - combat: Combat
  - combatants: Combatant[]
  - currentTurn: Combatant
  - isLoading: boolean

// Combat actions hook
const {
  nextTurn,
  attack,
  heal,
  endCombat
} = useCombatActions(combatId)
```

---

## WebSocket Integration

Combat state changes are pushed via WebSocket for real-time updates:

| Event Type | Payload | Action |
|------------|---------|--------|
| `combat_turn_changed` | `{ round, turnIndex, currentCombatant }` | Update turn indicator, scroll initiative list |
| `combat_action` | `{ actionType, actor, target, result }` | Add log entry, trigger animations |
| `combat_ended` | `{ combatId, winner }` | Show end combat modal, close overlay |

**Rationale**: In multiplayer sessions (future), all players see combat updates in real-time. For single-player, provides immediate feedback without polling.

---

## API Integration

### Backend Endpoints (Already Implemented)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/combat/start` | Create new combat session |
| POST | `/combat/{id}/combatants` | Add combatant |
| GET | `/combat/{id}` | Get combat summary |
| POST | `/combat/{id}/turn` | Advance to next turn |
| POST | `/combat/{id}/attack` | Resolve attack |
| POST | `/combat/{id}/heal` | Heal combatant |
| POST | `/combat/{id}/end` | End combat session |

### Request/Response Schemas

**AttackRequest**:
```typescript
{
  attacker_id: string    // UUID
  target_id: string      // UUID
  attack_skill: number   // 0-100
  attack_roll?: number   // Optional fixed roll
  damage_roll?: number   // Optional fixed damage
  damage_bonus: number   // DB from strength
}
```

**AttackResponse**:
```typescript
{
  attacker: string
  target: string
  attack_roll: number
  attack_skill: number
  success_level: string  // "extreme" | "hard" | "regular" | "failure"
  hit: boolean
  damage: number
  target_hp_before: number
  target_hp_after: number
  target_status: string  // "active" | "dying" | "dead"
  action_id: string
}
```

---

## File Structure

```
frontend/src/
├── components/
│   ├── combat/
│   │   ├── CombatOverlay.tsx          # Main overlay container
│   │   ├── CombatHeader.tsx           # Top control bar
│   │   ├── CombatInfoPanel.tsx        # Left panel (turn info)
│   │   ├── CombatActionPanel.tsx      # Center panel (actions)
│   │   ├── CombatLogPanel.tsx         # Right panel (log)
│   │   ├── AttackDialog.tsx           # Attack modal
│   │   ├── CombatantCard.tsx          # Combatant display card
│   │   └── InitiativeList.tsx         # Initiative order list
│   └── ui/
│       └── ... (existing shadcn/ui components)
├── hooks/
│   ├── useCombatState.ts              # Combat state management
│   ├── useCombatActions.ts            # Combat API calls
│   └── useGameWebSocket.ts            # (existing, extend for combat)
├── services/
│   └── api.ts                         # (extend with combat endpoints)
└── types/
    └── combat.ts                       # Combat TypeScript types
```

---

## Implementation Tasks (M1-080 ~ M1-084)

| Task ID | Description | Hours |
|---------|-------------|-------|
| M1-080 | Implement CombatTracker component | 4h |
| M1-081 | Implement round indicator | 2h |
| M1-082 | Implement initiative list | 2h |
| M1-083 | Implement combat action buttons | 2h |
| M1-084 | Implement damage animation | 2h |

**Total**: 12 hours

---

## Acceptance Criteria

- [ ] Combat overlay appears when combat starts (`POST /combat/start`)
- [ ] Round indicator shows current round number
- [ ] Initiative list displays all combatants sorted by DEX
- [ ] Current turn combatant is highlighted
- [ ] "Next Turn" button advances turn correctly
- [ ] Attack dialog opens and captures target/skill input
- [ ] Attack results display in combat log
- [ ] Damage animation plays on injured combatants
- [ ] HP updates reflect in UI immediately
- [ ] Overlay minimizes to floating card
- [ ] Overlay closes when combat ends
- [ ] WebSocket events update UI in real-time

---

## Future Enhancements (Out of Scope for M1)

- Manual initiative roll entry
- Combatant positioning (front/flank/rear)
- Multiple attack types (full auto, maneuver, fight back)
- Combat statistics summary after battle
- Combat replay/join mid-combat
- Sound effects for attacks/impacts

---

**Design Approved**: 2025-02-07
**Status**: Ready for implementation
**Next Step**: Create implementation plan with `superpowers:writing-plans`
