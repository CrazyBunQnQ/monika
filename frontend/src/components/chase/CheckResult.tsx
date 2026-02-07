import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { CheckCircle2, XCircle, TrendingDown, Heart, AlertTriangle } from "lucide-react"
import { cn } from "@/lib/utils"
import type { SuccessLevel } from "@/types/chase"

interface CheckResultProps {
  success: boolean
  rollValue: number
  successLevel: SuccessLevel
  damage?: number
  speedPenalty?: number
  message?: string
  onDismiss?: () => void
  className?: string
}

/**
 * Get badge variant based on success level
 */
function getSuccessBadgeVariant(successLevel: SuccessLevel): "default" | "secondary" | "outline" | "destructive" {
  switch (successLevel) {
    case "extreme":
      return "default" // Gold/yellow for extreme success
    case "hard":
      return "secondary" // Green for hard success
    case "regular":
      return "outline" // Gray for regular success
    case "failure":
      return "destructive" // Red for failure
  }
}

/**
 * Get success level label
 */
function getSuccessLevelLabel(successLevel: SuccessLevel): string {
  switch (successLevel) {
    case "extreme":
      return "Extreme Success"
    case "hard":
      return "Hard Success"
    case "regular":
      return "Regular Success"
    case "failure":
      return "Failure"
  }
}

/**
 * Get success level description
 */
function getSuccessLevelDescription(successLevel: SuccessLevel): string {
  switch (successLevel) {
    case "extreme":
      return "Outstanding! You overcome the obstacle with ease."
    case "hard":
      return "Great effort! You manage to pass."
    case "regular":
      return "You made it through."
    case "failure":
      return "You failed to overcome the obstacle."
  }
}

/**
 * CheckResult - Display skill check result
 *
 * Features:
 * - Shows roll value and success level
 * - Color-coded success/failure indication
 * - Displays damage and speed penalty if applicable
 * - Success/failure icons and badges
 * - Optional dismiss button
 */
export function CheckResult({
  success,
  rollValue,
  successLevel,
  damage = 0,
  speedPenalty = 0,
  message,
  onDismiss,
  className,
}: CheckResultProps) {
  return (
    <Card
      className={cn(
        "border-2",
        success
          ? "border-green-500/50 dark:border-green-500/70 bg-green-50/50 dark:bg-green-950/20"
          : "border-red-500/50 dark:border-red-500/70 bg-red-50/50 dark:bg-red-950/20",
        className
      )}
    >
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {success ? (
              <CheckCircle2 className="h-6 w-6 text-green-600 dark:text-green-400" />
            ) : (
              <XCircle className="h-6 w-6 text-red-600 dark:text-red-400" />
            )}
            <CardTitle className={cn(
              "text-lg",
              success ? "text-green-700 dark:text-green-300" : "text-red-700 dark:text-red-300"
            )}>
              {success ? "Success!" : "Failed!"}
            </CardTitle>
          </div>
          <Badge variant={getSuccessBadgeVariant(successLevel)} className="text-sm">
            {getSuccessLevelLabel(successLevel)}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Roll Value Display */}
        <div className="flex items-center justify-between p-3 bg-background rounded-lg border">
          <span className="text-sm text-muted-foreground">Roll Value</span>
          <span
            className={cn(
              "text-2xl font-mono font-bold",
              success ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"
            )}
          >
            {rollValue}
          </span>
        </div>

        {/* Success Level Description */}
        <div
          className={cn(
            "text-sm p-3 rounded-lg",
            success
              ? "bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-200"
              : "bg-red-100 dark:bg-red-900/20 text-red-800 dark:text-red-200"
          )}
        >
          {message || getSuccessLevelDescription(successLevel)}
        </div>

        {/* Penalties (only show if applicable) */}
        {(damage > 0 || speedPenalty > 0) && (
          <div className="space-y-2 p-3 bg-red-50/50 dark:bg-red-950/20 rounded-lg border border-red-200 dark:border-red-800">
            <div className="flex items-center gap-2 text-sm font-semibold text-red-700 dark:text-red-300">
              <AlertTriangle className="h-4 w-4" />
              <span>Penalties Applied</span>
            </div>
            <div className="space-y-1 text-sm">
              {damage > 0 && (
                <div className="flex items-center justify-between text-red-600 dark:text-red-400">
                  <div className="flex items-center gap-1">
                    <Heart className="h-3 w-3" />
                    <span>Damage</span>
                  </div>
                  <span className="font-mono font-semibold">
                    -{damage} HP
                  </span>
                </div>
              )}
              {speedPenalty > 0 && (
                <div className="flex items-center justify-between text-red-600 dark:text-red-400">
                  <div className="flex items-center gap-1">
                    <TrendingDown className="h-3 w-3" />
                    <span>Speed Penalty</span>
                  </div>
                  <span className="font-mono font-semibold">
                    -{speedPenalty}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Dismiss Button */}
        {onDismiss && (
          <Button
            onClick={onDismiss}
            variant={success ? "default" : "destructive"}
            className="w-full"
            size="lg"
          >
            Continue
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
