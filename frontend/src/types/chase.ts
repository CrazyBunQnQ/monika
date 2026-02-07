/**
 * Chase-related TypeScript types for Monika frontend
 *
 * These types correspond to the backend chase API schemas in
 * backend/src/api/chase.py and models in backend/src/models/chase.py
 */

/**
 * Chase session state
 */
export type ChaseState = 'active' | 'paused' | 'ended'

/**
 * Why a chase ended
 */
export type ChaseEndReason = 'escaped' | 'caught' | 'abandoned' | 'failed_forward'

/**
 * Role in the chase
 */
export type ChaseParticipantRole = 'fugitive' | 'pursuer'

/**
 * Types of obstacles in a chase
 */
export type ObstacleType = 'physical' | 'environmental' | 'skill_check' | 'combat'

/**
 * Obstacle difficulty levels
 */
export type ObstacleDifficulty = 'regular' | 'hard' | 'extreme'

/**
 * Success level for skill rolls in CoC 7e
 */
export type SuccessLevel = 'extreme' | 'hard' | 'regular' | 'failure'

/**
 * Action types available during a chase
 */
export type ActionType = 'accelerate' | 'decelerate' | 'overcome_obstacle' | 'attack'

/**
 * Chase session with all participants and obstacles
 */
export interface Chase {
  id: string
  session_id: string
  state: ChaseState
  round: number
  location: string
  setting: string
  started_at: string
  ended_at: string | null
  end_reason: ChaseEndReason | null
  failed_forward_scene: string | null
  chase_metadata: Record<string, unknown>
  participants: ChaseParticipant[]
  obstacles: ChaseObstacle[]
}

/**
 * Participant in a chase session
 */
export interface ChaseParticipant {
  id: string
  chase_id: string
  character_id: number | null
  name: string
  role: ChaseParticipantRole
  is_player: boolean
  move_rate: number
  current_speed: number
  position_index: number
  is_active: boolean
  is_exhausted: boolean
  failed_obstacle_count: number
  speed_penalty: number
  consecutive_failures: number
  participant_metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

/**
 * Obstacle encountered during a chase
 */
export interface ChaseObstacle {
  id: string
  chase_id: string
  name: string
  description: string
  obstacle_type: ObstacleType
  appears_at_round: number
  appears_at_distance: number
  difficulty: ObstacleDifficulty
  skill_required: string | null
  failure_penalty: number
  failure_damage: number | null
  failure_san_cost: number | null
  fail_forward_result: string | null
  details: Record<string, unknown>
  created_at: string
}

/**
 * Record of an action taken during a chase
 */
export interface ChaseAction {
  id: string
  chase_id: string
  round: number
  participant_id: string
  obstacle_id: string | null
  action_type: ActionType
  skill_used: string | null
  roll_value: number | null
  skill_value: number | null
  success_level: SuccessLevel | null
  speed_change: number
  position_change: number
  damage_taken: number | null
  san_lost: number | null
  details: Record<string, unknown>
  created_at: string
}

/**
 * Request to create a new chase session
 */
export interface ChaseCreateRequest {
  session_id: string
  location: string
  setting?: string
}

/**
 * Request to add a participant to a chase
 */
export interface ChaseParticipantCreateRequest {
  name: string
  role: ChaseParticipantRole
  move_rate?: number
  is_player?: boolean
  character_id?: number
}

/**
 * Request to resolve a chase round
 */
export interface ChaseRoundRequest {
  actions: ChaseActionRequestItem[]
}

/**
 * Request for a single chase action
 */
export interface ChaseActionRequestItem {
  participant_id: string
  action_type: ActionType
  obstacle_id?: string
  skill?: number
}

/**
 * Request to manually end a chase
 */
export interface ChaseEndRequest {
  reason: ChaseEndReason
  fail_forward_scene?: string
}

/**
 * Response with chase data
 */
export interface ChaseResponse {
  id: string
  state: ChaseState
  round: number
  location: string
  setting: string
  started_at: string | null
  ended_at: string | null
  end_reason: string | null
  failed_forward_scene: string | null
  participants: Record<string, unknown>[]
  obstacles: Record<string, unknown>[]
}

/**
 * Response for round resolution
 */
export interface ChaseRoundResponse {
  chase_id: string
  round: number
  actions: Record<string, unknown>[]
  positions: Record<string, unknown>[]
  chase_ended: boolean
  end_reason: string | null
}

/**
 * Response for obstacle data
 */
export interface ObstacleResponse {
  id: string
  name: string
  description: string
  type: ObstacleType
  difficulty: ObstacleDifficulty
  skill_required: string | null
  failure_penalty: number
  failure_damage: number | null
  fail_forward_result: string | null
}

/**
 * Legacy: Round result (for backward compatibility)
 * @deprecated Use ChaseRoundResponse instead
 */
export interface RoundResult {
  round: number
  actions: ActionResult[]
  new_distance_level: number
  new_pressure: number
  chase_ended: boolean
  winner?: 'fugitive' | 'pursuer'
}

/**
 * Legacy: Action result (for backward compatibility)
 * @deprecated Use ChaseAction instead
 */
export interface ActionResult {
  participant_id: string
  action: ActionType
  success: boolean
  new_speed: number
  damage_taken?: number
  obstacle_overcome?: boolean
}

/**
 * Legacy: Check result (for backward compatibility)
 * @deprecated Use SuccessLevel and related types instead
 */
export interface CheckResult {
  success: boolean
  roll_value: number
  success_level: 'regular' | 'hard' | 'extreme' | 'critical' | 'fumble'
  damage?: number
  speed_penalty?: number
  message: string
}

/**
 * Legacy: Obstacle check request (for backward compatibility)
 * @deprecated Use ChaseActionRequestItem instead
 */
export interface ObstacleCheckRequest {
  participant_id: string
  obstacle_id: string
  skill_name: string
  skill_value: number
  use_luck?: boolean
}

/**
 * Legacy: Chase action request (for backward compatibility)
 * @deprecated Use ChaseActionRequestItem instead
 */
export interface ChaseActionRequest {
  participant_id: string
  action: ActionType
  target_id?: string
  check_value?: number
}

/**
 * WebSocket event: Chase started
 */
export interface ChaseStartedEvent {
  chase_id: string
  chase: Chase
}

/**
 * WebSocket event: Chase updated
 */
export interface ChaseUpdatedEvent {
  chase_id: string
  chase: Chase
}

/**
 * WebSocket event: Chase ended
 */
export interface ChaseEndedEvent {
  chase_id: string
  winner?: 'fugitive' | 'pursuer'
}

/**
 * Single entry in chase log
 */
export interface ChaseLogEntry {
  id: string
  round: number
  type: 'action' | 'obstacle' | 'round_change' | 'chase_start' | 'chase_end'
  actor?: string
  target?: string
  description: string
  success_level?: SuccessLevel
  damage?: number
  san_lost?: number
  speed_change?: number
  position_change?: number
  obstacle_overcome?: boolean
  timestamp: Date
}
