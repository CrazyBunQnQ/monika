/**
 * Combat-related TypeScript types for Monika frontend
 *
 * These types correspond to the backend combat API schemas in
 * backend/src/api/combat.py
 */

/**
 * Combatant role determines UI display and targeting options
 */
export type CombatantRole = 'pc' | 'npc' | 'ally'

/**
 * Combat session state
 */
export type CombatState = 'active' | 'paused' | 'ended'

/**
 * Damage type (lethal vs non-lethal affects dying mechanics)
 */
export type DamageType = 'lethal' | 'non_lethal'

/**
 * Success level for skill rolls in CoC 7e
 */
export type SuccessLevel = 'extreme' | 'hard' | 'regular' | 'failure'

/**
 * Individual combatant in a combat session
 */
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
  position?: string  // Optional: 'front', 'flank', 'rear'
  character_id?: number  // Optional: links to Character if PC
}

/**
 * Combat session with all combatants and turn info
 */
export interface Combat {
  id: string
  state: CombatState
  round: number
  location: string | null
  description: string | null
  started_at: string | null
  ended_at: string | null
  combatants: Combatant[]
  current_turn: Combatant | null
  total_actions?: number  // Optional: total actions in current turn
}

/**
 * Response when advancing to next turn
 */
export interface TurnResponse {
  combat_id: string
  current_round: number
  current_turn_index: number
  current_combatant: Combatant | null
  is_new_round: boolean
  turn_order: Combatant[]
}

/**
 * Request to make an attack
 */
export interface AttackRequest {
  attacker_id: string
  target_id: string
  attack_skill: number
  attack_roll?: number
  damage_roll?: number
  damage_bonus: number
}

/**
 * Response after resolving an attack
 */
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
  target_status: string
  action_id: string
}

/**
 * Request to heal a combatant
 */
export interface HealRequest {
  target_id: string
  heal_amount: number
  first_aid_skill: number
  first_aid_roll?: number
}

/**
 * Response after healing
 */
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

/**
 * Request to create a new combat session
 */
export interface CombatCreateRequest {
  session_id: string
  location?: string
  description?: string
}

/**
 * Request to add a combatant to existing combat
 */
export interface CombatantCreateRequest {
  name: string
  hp: number
  hp_max: number
  dex: number
  role: CombatantRole
  character_id?: number
}

/**
 * Single entry in combat log
 */
export interface CombatLogEntry {
  id: string
  round: number
  type: 'attack' | 'heal' | 'turn_change' | 'combat_start' | 'combat_end'
  actor?: string
  target?: string
  description: string
  success_level?: SuccessLevel
  damage?: number
  healing?: number
  timestamp: string
}
