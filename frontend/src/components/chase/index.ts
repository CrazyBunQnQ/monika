/**
 * Chase system components barrel export
 *
 * Provides structured exports for chase UI components:
 * - Info panels: DistanceTrack, PressureBar, ParticipantList, ChaseInfoPanel
 * - Action panels: ObstacleCard, ActionSelector, CheckResult, ChaseActionPanel
 * - Log panels: ChaseLogPanel
 * - Main overlay: (TODO)
 */

// Info Panels
export { DistanceTrack } from "./DistanceTrack"
export { PressureBar } from "./PressureBar"
export { ParticipantList } from "./ParticipantList"
export { ChaseInfoPanel } from "./ChaseInfoPanel"

// Action Panels
export { ObstacleCard } from "./ObstacleCard"
export { ActionSelector } from "./ActionSelector"
export { CheckResult } from "./CheckResult"
export { ChaseActionPanel } from "./ChaseActionPanel"

// Log Panels
export { ChaseLogPanel } from "./ChaseLogPanel"

// Re-export types for convenience
export type {
  Chase,
  ChaseParticipant,
  ChaseObstacle,
  ChaseAction,
  ChaseState,
  ChaseEndReason,
  ChaseParticipantRole,
  ObstacleType,
  ObstacleDifficulty,
  SuccessLevel,
  ActionType,
  ChaseCreateRequest,
  ChaseParticipantCreateRequest,
  ChaseRoundRequest,
  ChaseActionRequestItem,
  ChaseEndRequest,
  ChaseResponse,
  ChaseRoundResponse,
  ObstacleResponse,
  ChaseLogEntry,
} from "@/types/chase"
