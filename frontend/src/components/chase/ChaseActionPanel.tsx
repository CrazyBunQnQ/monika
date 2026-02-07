import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Zap, Target } from "lucide-react"
import { cn } from "@/lib/utils"
import { ObstacleCard } from "./ObstacleCard"
import { ActionSelector } from "./ActionSelector"
import { CheckResult } from "./CheckResult"
import type { Chase, ChaseParticipant, ChaseObstacle, ActionType } from "@/types/chase"
import type { SuccessLevel } from "@/types/chase"

interface CheckResultData {
  success: boolean
  rollValue: number
  successLevel: SuccessLevel
  damage?: number
  speedPenalty?: number
  message?: string
}

interface ChaseActionPanelProps {
  chase: Chase
  currentParticipant: ChaseParticipant | null
  currentObstacle: ChaseObstacle | null
  lastCheckResult: CheckResultData | null
  onExecuteAction: (actionType: ActionType, obstacleId?: string, skillValue?: number) => Promise<void>
  onSpendLuck?: () => Promise<void>
  onDismissResult?: () => void
  isLoading?: boolean
  canSpendLuck?: boolean
  className?: string
}

/**
 * ChaseActionPanel - Main action panel for chase system
 *
 * Dynamic display based on context:
 * 1. If there's a check result → show CheckResult
 * 2. If there's an obstacle → show ObstacleCard
 * 3. Otherwise → show ActionSelector
 *
 * Features:
 * - Context-aware UI that changes based on game state
 * - Action execution with loading states
 * - Luck point spending support
 * - Current participant display
 */
export function ChaseActionPanel({
  chase,
  currentParticipant,
  currentObstacle,
  lastCheckResult,
  onExecuteAction,
  onSpendLuck,
  onDismissResult,
  isLoading = false,
  canSpendLuck = false,
  className,
}: ChaseActionPanelProps) {
  const [internalResult, setInternalResult] = useState<CheckResultData | null>(lastCheckResult)

  /**
   * Update internal result state when prop changes
   */
  useEffect(() => {
    setInternalResult(lastCheckResult)
  }, [lastCheckResult])

  /**
   * Handle action execution from ActionSelector
   */
  const handleExecuteAction = async (actionType: ActionType) => {
    // For obstacle action, we need to pass the obstacle ID
    if (actionType === "overcome_obstacle" && currentObstacle) {
      // The skill value will be prompted from the user in a real implementation
      // For now, we'll pass undefined and let the backend handle it
      await onExecuteAction(actionType, currentObstacle.id, undefined)
    } else {
      await onExecuteAction(actionType)
    }
  }

  /**
   * Handle check now from ObstacleCard
   */
  const handleCheckNow = async () => {
    if (!currentObstacle) return
    await handleExecuteAction("overcome_obstacle")
  }

  /**
   * Handle spend luck from ObstacleCard
   */
  const handleSpendLuck = async () => {
    if (onSpendLuck) {
      await onSpendLuck()
    }
  }

  /**
   * Handle dismiss result
   */
  const handleDismissResult = () => {
    setInternalResult(null)
    if (onDismissResult) {
      onDismissResult()
    }
  }

  /**
   * Determine what to display based on context
   * Priority: CheckResult > ObstacleCard > ActionSelector
   */
  const getDisplayContent = () => {
    // 1. Show check result if available
    if (internalResult) {
      return (
        <CheckResult
          {...internalResult}
          onDismiss={handleDismissResult}
        />
      )
    }

    // 2. Show obstacle card if there's an active obstacle
    if (currentObstacle) {
      return (
        <ObstacleCard
          obstacle={currentObstacle}
          onCheckNow={handleCheckNow}
          onSpendLuck={handleSpendLuck}
          canSpendLuck={canSpendLuck}
          isLoading={isLoading}
        />
      )
    }

    // 3. Otherwise show action selector
    return (
      <ActionSelector
        availableActions={["accelerate", "decelerate", "attack"]}
        onExecuteAction={handleExecuteAction}
        currentSpeed={currentParticipant?.current_speed ?? 0}
        isLoading={isLoading}
      />
    )
  }

  return (
    <Card className={cn("flex flex-col h-full", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Actions</CardTitle>
          <div className="flex items-center gap-2">
            {/* Chase State Badge */}
            <Badge variant="outline" className="text-sm">
              Round {chase.round}
            </Badge>
            {/* Current Participant Badge */}
            {currentParticipant && (
              <Badge variant="secondary" className="text-sm">
                <Target className="h-3 w-3 mr-1" />
                {currentParticipant.name}
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="flex-1 p-3 pt-0 overflow-hidden">
        <ScrollArea className="h-full pr-4">
          <div className="space-y-4">
            {/* Context Indicator */}
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Zap className="h-4 w-4" />
              <span>
                {internalResult && "Check Result"}
                {currentObstacle && !internalResult && "Obstacle Ahead!"}
                {!currentObstacle && !internalResult && "Choose Your Action"}
              </span>
            </div>

            {/* Dynamic Content */}
            <div className="min-h-[400px]">
              {getDisplayContent()}
            </div>
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  )
}
