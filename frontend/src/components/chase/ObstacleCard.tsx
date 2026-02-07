import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { AlertTriangle, Zap, Mountain, Wrench, Shield, Sword } from "lucide-react"
import { cn } from "@/lib/utils"
import type { ChaseObstacle, ObstacleType, ObstacleDifficulty } from "@/types/chase"

interface ObstacleCardProps {
  obstacle: ChaseObstacle
  onCheckNow: () => void
  onSpendLuck?: () => void
  canSpendLuck?: boolean
  isLoading?: boolean
  className?: string
}

/**
 * Get icon for obstacle type
 */
function getObstacleIcon(type: ObstacleType) {
  switch (type) {
    case "physical":
      return <Mountain className="h-5 w-5" />
    case "environmental":
      return <Zap className="h-5 w-5" />
    case "skill_check":
      return <Wrench className="h-5 w-5" />
    case "combat":
      return <Sword className="h-5 w-5" />
    default:
      return <AlertTriangle className="h-5 w-5" />
  }
}

/**
 * Get badge variant for obstacle difficulty
 */
function getDifficultyBadgeVariant(difficulty: ObstacleDifficulty): "default" | "secondary" | "outline" | "destructive" {
  switch (difficulty) {
    case "extreme":
      return "destructive" // Red for extreme
    case "hard":
      return "outline" // Yellow/outline for hard
    case "regular":
      return "secondary" // Green for regular
  }
}

/**
 * Get difficulty label with color
 */
function getDifficultyLabel(difficulty: ObstacleDifficulty): string {
  switch (difficulty) {
    case "extreme":
      return "Extreme"
    case "hard":
      return "Hard"
    case "regular":
      return "Regular"
  }
}

/**
 * ObstacleCard - Display obstacle information with action buttons
 *
 * Features:
 * - Shows obstacle name, type, difficulty, and required skill
 * - Yellow border warning style
 * - "Check Now" button for immediate skill check
 * - "Spend Luck" button (optional, if player has luck points)
 * - Displays failure penalties (damage, speed penalty)
 */
export function ObstacleCard({
  obstacle,
  onCheckNow,
  onSpendLuck,
  canSpendLuck = false,
  isLoading = false,
  className,
}: ObstacleCardProps) {
  return (
    <Card
      className={cn(
        "border-2 border-yellow-500/50 dark:border-yellow-500/70 bg-yellow-50/50 dark:bg-yellow-950/20",
        className
      )}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2">
            <div className="text-yellow-600 dark:text-yellow-400">
              {getObstacleIcon(obstacle.obstacle_type)}
            </div>
            <CardTitle className="text-lg">{obstacle.name}</CardTitle>
          </div>
          <Badge variant={getDifficultyBadgeVariant(obstacle.difficulty)} className="shrink-0">
            {getDifficultyLabel(obstacle.difficulty)}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Description */}
        <div className="text-sm text-muted-foreground">
          {obstacle.description}
        </div>

        {/* Required Skill */}
        {obstacle.skill_required && (
          <div className="flex items-center justify-between p-2 bg-background rounded-lg border">
            <span className="text-sm text-muted-foreground">Required Skill</span>
            <Badge variant="outline" className="font-mono">
              {obstacle.skill_required}
            </Badge>
          </div>
        )}

        {/* Failure Penalties */}
        {(obstacle.failure_damage !== null || obstacle.failure_penalty > 0) && (
          <div className="space-y-2 p-3 bg-red-50/50 dark:bg-red-950/20 rounded-lg border border-red-200 dark:border-red-800">
            <div className="flex items-center gap-2 text-sm font-semibold text-red-700 dark:text-red-300">
              <AlertTriangle className="h-4 w-4" />
              <span>Failure Consequences</span>
            </div>
            <div className="space-y-1 text-sm text-red-600 dark:text-red-400">
              {obstacle.failure_damage !== null && (
                <div className="flex items-center justify-between">
                  <span>Damage</span>
                  <span className="font-mono font-semibold">
                    -{obstacle.failure_damage} HP
                  </span>
                </div>
              )}
              {obstacle.failure_penalty > 0 && (
                <div className="flex items-center justify-between">
                  <span>Speed Penalty</span>
                  <span className="font-mono font-semibold">
                    -{obstacle.failure_penalty}
                  </span>
                </div>
              )}
              {obstacle.failure_san_cost !== null && obstacle.failure_san_cost > 0 && (
                <div className="flex items-center justify-between">
                  <span>SAN Loss</span>
                  <span className="font-mono font-semibold">
                    -{obstacle.failure_san_cost}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Action Buttons */}
        <div className="grid grid-cols-2 gap-2 pt-2">
          {/* Check Now Button */}
          <Button
            onClick={onCheckNow}
            disabled={isLoading}
            className="col-span-2"
            size="lg"
          >
            {isLoading ? (
              <>
                <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                Checking...
              </>
            ) : (
              <>
                <Zap className="mr-2 h-5 w-5" />
                Check Now
              </>
            )}
          </Button>

          {/* Spend Luck Button */}
          {canSpendLuck && onSpendLuck && (
            <Button
              onClick={onSpendLuck}
              disabled={isLoading}
              variant="outline"
              size="lg"
              className="col-span-2"
            >
              <Shield className="mr-2 h-4 w-4" />
              Spend Luck Point
            </Button>
          )}
        </div>

        {/* Obstacle Type Badge */}
        <div className="flex items-center justify-center pt-2 border-t">
          <Badge variant="outline" className="text-xs">
            Type: {obstacle.obstacle_type.replace(/_/g, " ")}
          </Badge>
        </div>
      </CardContent>
    </Card>
  )
}
