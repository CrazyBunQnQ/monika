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
