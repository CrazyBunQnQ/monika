import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { AttackDialog } from "./AttackDialog"
import { CombatantCard } from "./CombatantCard"
import { Sword, Shield, Heart, SkipForward, Target } from "lucide-react"
import { cn } from "@/lib/utils"
import type { Combatant, AttackRequest, HealRequest } from "@/types/combat"

interface CombatActionPanelProps {
  currentTurn: Combatant | null
  combatants: Combatant[]
  onAttack: (request: AttackRequest) => Promise<void>
  onHeal: (request: HealRequest) => Promise<void>
  onDodge: () => Promise<void>
  onEndTurn: () => Promise<void>
  isLoading?: boolean
  className?: string
}

export function CombatActionPanel({
  currentTurn,
  combatants,
  onAttack,
  onHeal,
  onDodge,
  onEndTurn,
  isLoading = false,
  className,
}: CombatActionPanelProps) {
  const [selectedTargetId, setSelectedTargetId] = useState<string | null>(null)
  const [isAttackDialogOpen, setIsAttackDialogOpen] = useState(false)

  /**
   * Get selected target combatant
   */
  const selectedTarget = combatants.find((c) => c.id === selectedTargetId)

  /**
   * Handle attack button click
   * Opens dialog if target selected, otherwise prompts to select
   */
  const handleAttackClick = () => {
    if (!currentTurn) return

    if (!selectedTarget) {
      // TODO: Show toast or prompt to select target
      return
    }
    setIsAttackDialogOpen(true)
  }

  /**
   * Handle attack from dialog
   */
  const handleAttack = async (request: AttackRequest) => {
    await onAttack(request)
  }

  /**
   * Handle dodge action
   * Dodging adds a temporary bonus to next defensive roll
   */
  const handleDodge = async () => {
    await onDodge()
  }

  /**
   * Handle heal action
   * Uses first aid skill to restore HP to selected target
   */
  const handleHeal = async () => {
    if (!currentTurn || !selectedTarget) return

    const request: HealRequest = {
      target_id: selectedTarget.id,
      heal_amount: 0, // Will be calculated by backend based on first aid roll
      first_aid_skill: 30, // TODO: Get from character stats
    }

    await onHeal(request)
  }

  /**
   * Handle end turn
   * Advances to next combatant in initiative order
   */
  const handleEndTurn = async () => {
    await onEndTurn()
    setSelectedTargetId(null)
  }

  /**
   * Filter potential targets (active combatants, not self)
   */
  const potentialTargets = combatants.filter(
    (c) => c.is_active && c.id !== currentTurn?.id
  )

  return (
    <Card className={cn("flex flex-col h-full", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Actions</CardTitle>
          {currentTurn && (
            <Badge variant="outline" className="text-sm">
              <Target className="h-3 w-3 mr-1" />
              {currentTurn.name}
            </Badge>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 flex flex-col gap-3 p-3 pt-0 overflow-hidden">
        {/* Action Buttons */}
        <div className="grid grid-cols-2 gap-2">
          {/* Attack - Primary action */}
          <Button
            onClick={handleAttackClick}
            disabled={!currentTurn || isLoading}
            size="lg"
            className="col-span-2"
          >
            <Sword className="h-5 w-5 mr-2" />
            Attack
          </Button>

          {/* Dodge - Defensive action */}
          <Button
            onClick={handleDodge}
            disabled={!currentTurn || isLoading}
            variant="outline"
            size="lg"
          >
            <Shield className="h-4 w-4 mr-2" />
            Dodge
          </Button>

          {/* Heal - Support action */}
          <Button
            onClick={handleHeal}
            disabled={!currentTurn || !selectedTarget || isLoading}
            variant="outline"
            size="lg"
          >
            <Heart className="h-4 w-4 mr-2" />
            Heal
          </Button>

          {/* End Turn */}
          <Button
            onClick={handleEndTurn}
            disabled={!currentTurn || isLoading}
            variant="secondary"
            size="lg"
            className="col-span-2"
          >
            <SkipForward className="h-4 w-4 mr-2" />
            End Turn
          </Button>
        </div>

        {/* Target Selection */}
        <div className="flex-1 flex flex-col min-h-0">
          <div className="text-sm text-muted-foreground mb-2">
            Select Target
          </div>
          <ScrollArea className="flex-1 pr-4">
            <div className="space-y-2">
              {potentialTargets.length === 0 ? (
                <div className="text-sm text-muted-foreground text-center py-4">
                  No valid targets
                </div>
              ) : (
                potentialTargets.map((combatant) => (
                  <div
                    key={combatant.id}
                    onClick={() => setSelectedTargetId(
                      selectedTargetId === combatant.id ? null : combatant.id
                    )}
                    className="cursor-pointer"
                  >
                    <CombatantCard
                      combatant={combatant}
                      isSelected={selectedTargetId === combatant.id}
                      showHpChange={false}
                    />
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </div>
      </CardContent>

      {/* Attack Dialog */}
      {currentTurn && (
        <AttackDialog
          isOpen={isAttackDialogOpen}
          onClose={() => setIsAttackDialogOpen(false)}
          onAttack={handleAttack}
          attacker={currentTurn}
          targets={combatants}
          isLoading={isLoading}
        />
      )}
    </Card>
  )
}
