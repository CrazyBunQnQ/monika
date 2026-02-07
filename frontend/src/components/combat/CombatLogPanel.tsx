import { useEffect, useRef } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Trash2, Swords, Heart, RotateCcw } from "lucide-react"
import { cn } from "@/lib/utils"
import type { CombatLogEntry, SuccessLevel } from "@/types/combat"

interface CombatLogPanelProps {
  logs: CombatLogEntry[]
  onClear?: () => void
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
      return "Extreme"
    case "hard":
      return "Hard"
    case "regular":
      return "Regular"
    case "failure":
      return "Failure"
  }
}

/**
 * Get icon for log entry type
 */
function getLogIcon(type: CombatLogEntry["type"]) {
  switch (type) {
    case "attack":
      return <Swords className="h-4 w-4" />
    case "heal":
      return <Heart className="h-4 w-4" />
    case "turn_change":
      return <RotateCcw className="h-4 w-4" />
    default:
      return null
  }
}

/**
 * Format timestamp for display
 */
function formatTimestamp(timestamp: Date): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
}

export function CombatLogPanel({ logs, onClear, className }: CombatLogPanelProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  /**
   * Auto-scroll to bottom when new logs are added
   */
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs])

  /**
   * Group logs by round for better organization
   */
  const groupedLogs: Record<number, CombatLogEntry[]> = {}
  logs.forEach((log) => {
    if (!groupedLogs[log.round]) {
      groupedLogs[log.round] = []
    }
    groupedLogs[log.round].push(log)
  })

  /**
   * Get sorted rounds (newest first)
   */
  const rounds = Object.keys(groupedLogs)
    .map(Number)
    .sort((a, b) => b - a)

  return (
    <Card className={cn("flex flex-col h-full", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Combat Log</CardTitle>
          {onClear && logs.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onClear}
              className="h-8 px-2"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 p-3 pt-0 overflow-hidden">
        <ScrollArea className="h-full pr-4" ref={scrollRef}>
          {logs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
              No combat actions yet
            </div>
          ) : (
            <div className="space-y-4">
              {rounds.map((round) => (
                <div key={round} className="space-y-2">
                  {/* Round Header */}
                  <div className="flex items-center gap-2 sticky top-0 bg-background/95 backdrop-blur py-1 border-b">
                    <Badge variant="outline" className="text-xs">
                      Round {round}
                    </Badge>
                  </div>

                  {/* Log Entries for this round */}
                  <div className="space-y-2 pl-2">
                    {groupedLogs[round]
                      .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
                      .map((log) => (
                        <div
                          key={log.id}
                          className={cn(
                            "text-sm p-2 rounded-lg border bg-card",
                            log.success_level === "failure" && "border-red-500/30",
                            log.success_level === "extreme" && "border-yellow-500/30"
                          )}
                        >
                          {/* Icon + Actor + Target */}
                          <div className="flex items-start gap-2">
                            <div className="mt-0.5 text-muted-foreground">
                              {getLogIcon(log.type)}
                            </div>
                            <div className="flex-1 space-y-1">
                              {/* Actor and Target */}
                              <div className="flex items-center gap-2 flex-wrap">
                                {log.actor && (
                                  <span className="font-semibold">{log.actor}</span>
                                )}
                                {log.actor && log.target && (
                                  <span className="text-muted-foreground">→</span>
                                )}
                                {log.target && (
                                  <span className="font-medium">{log.target}</span>
                                )}
                              </div>

                              {/* Description */}
                              <div className="text-muted-foreground">
                                {log.description}
                              </div>

                              {/* Success Level Badge */}
                              {log.success_level && (
                                <Badge
                                  variant={getSuccessBadgeVariant(log.success_level)}
                                  className="text-xs"
                                >
                                  {getSuccessLevelLabel(log.success_level)}
                                </Badge>
                              )}

                              {/* Damage/Healing */}
                              {(log.damage !== undefined || log.healing !== undefined) && (
                                <div className="flex items-center gap-3 text-xs">
                                  {log.damage !== undefined && (
                                    <span className="text-red-600 dark:text-red-400 font-semibold">
                                      -{log.damage} HP
                                    </span>
                                  )}
                                  {log.healing !== undefined && (
                                    <span className="text-green-600 dark:text-green-400 font-semibold">
                                      +{log.healing} HP
                                    </span>
                                  )}
                                </div>
                              )}

                              {/* Timestamp */}
                              <div className="text-xs text-muted-foreground">
                                {formatTimestamp(log.timestamp)}
                              </div>
                            </div>
                          </div>
                        </div>
                      ))}
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
