import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { CombatantCard } from "./CombatantCard"
import { Play, ArrowDown } from "lucide-react"
import { cn } from "@/lib/utils"
import type { Combatant } from "@/types/combat"

interface InitiativeListProps {
  round: number
  currentTurn: Combatant | null
  combatants: Combatant[]
  onNextTurn: () => void
  isLoading?: boolean
  className?: string
}

export function InitiativeList({
  round,
  currentTurn,
  combatants,
  onNextTurn,
  isLoading = false,
  className,
}: InitiativeListProps) {
  /**
   * Sort combatants by initiative (highest first)
   * In CoC 7e, higher DEX roll goes first
   */
  const sortedCombatants = [...combatants].sort((a, b) => b.initiative - a.initiative)

  return (
    <Card className={cn("flex flex-col h-full", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Turn Order</CardTitle>
          <Badge variant="outline" className="text-sm">
            Round {round}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="flex-1 flex flex-col gap-3 p-3 pt-0 overflow-hidden">
        {/* Current Turn */}
        {currentTurn && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Play className="h-4 w-4 text-green-500" />
              <span>Current Turn</span>
            </div>
            <CombatantCard
              combatant={currentTurn}
              isCurrentTurn
              showHpChange={false}
            />
          </div>
        )}

        {/* Initiative Order */}
        <div className="flex-1 flex flex-col min-h-0">
          <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
            <ArrowDown className="h-4 w-4" />
            <span>Initiative Order</span>
          </div>
          <ScrollArea className="flex-1 pr-4">
            <div className="space-y-2">
              {sortedCombatants.map((combatant, index) => (
                <div
                  key={combatant.id}
                  className={cn(
                    "flex items-center gap-2 p-2 rounded-lg border bg-card",
                    !combatant.is_active && "opacity-50",
                    currentTurn?.id === combatant.id && "ring-2 ring-green-500"
                  )}
                >
                  <span className="text-xs text-muted-foreground w-6">
                    {index + 1}.
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium text-sm truncate">
                        {combatant.name}
                      </span>
                      <Badge variant="outline" className="text-xs shrink-0">
                        Init: {combatant.initiative}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span>DEX: {combatant.dex}</span>
                      <span className={cn(
                        "font-medium",
                        combatant.hp <= combatant.hp_max * 0.25
                          ? "text-red-600 dark:text-red-400"
                          : combatant.hp <= combatant.hp_max * 0.5
                          ? "text-orange-600 dark:text-orange-400"
                          : "text-green-600 dark:text-green-400"
                      )}>
                        HP: {combatant.hp}/{combatant.hp_max}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Next Turn Button */}
        <Button
          onClick={onNextTurn}
          disabled={isLoading}
          className="w-full"
          size="lg"
        >
          {isLoading ? "Advancing..." : "Next Turn"}
        </Button>
      </CardContent>
    </Card>
  )
}
