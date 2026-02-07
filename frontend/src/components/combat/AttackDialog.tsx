import { useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { Combatant, AttackRequest } from "@/types/combat"

interface AttackDialogProps {
  isOpen: boolean
  onClose: () => void
  onAttack: (request: AttackRequest) => Promise<void>
  attacker: Combatant
  targets: Combatant[]
  isLoading?: boolean
}

/**
 * Bonus/Penalty dice options for CoC 7e
 * Positive values = bonus dice (roll multiple, take lowest)
 * Negative values = penalty dice (roll multiple, take highest)
 */
const BONUS_PENALTY_OPTIONS = [
  { label: "None", value: 0 },
  { label: "+1 Bonus", value: 1 },
  { label: "+2 Bonus", value: 2 },
  { label: "-1 Penalty", value: -1 },
  { label: "-2 Penalty", value: -2 },
]

export function AttackDialog({
  isOpen,
  onClose,
  onAttack,
  attacker,
  targets,
  isLoading = false,
}: AttackDialogProps) {
  const [selectedTargetId, setSelectedTargetId] = useState<string>("")
  const [attackSkill, setAttackSkill] = useState<number>(50) // Default Fight skill
  const [bonusPenalty, setBonusPenalty] = useState<number>(0)

  /**
   * Handle attack roll
   * Validates selection and calls onAttack callback
   */
  const handleAttack = async () => {
    if (!selectedTargetId) {
      return
    }

    const request: AttackRequest = {
      attacker_id: attacker.id,
      target_id: selectedTargetId,
      attack_skill: attackSkill,
      damage_bonus: 0, // TODO: Calculate from character stats
    }

    await onAttack(request)
    onClose()

    // Reset form
    setSelectedTargetId("")
    setBonusPenalty(0)
  }

  /**
   * Filter targets to only include active combatants
   */
  const activeTargets = targets.filter((t) => t.is_active && t.id !== attacker.id)

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Attack Roll</DialogTitle>
          <DialogDescription>
            Roll an attack against a target in combat
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* Attacker Info */}
          <div className="text-sm text-muted-foreground">
            Attacking as: <span className="font-semibold text-foreground">{attacker.name}</span>
          </div>

          {/* Target Selection */}
          <div className="grid gap-2">
            <Label htmlFor="target">Target</Label>
            <Select value={selectedTargetId} onValueChange={setSelectedTargetId}>
              <SelectTrigger id="target">
                <SelectValue placeholder="Select a target" />
              </SelectTrigger>
              <SelectContent>
                {activeTargets.map((target) => (
                  <SelectItem key={target.id} value={target.id}>
                    {target.name} (HP: {target.hp}/{target.hp_max})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Attack Skill */}
          <div className="grid gap-2">
            <Label htmlFor="skill">Attack Skill</Label>
            <Input
              id="skill"
              type="number"
              min="0"
              max="100"
              value={attackSkill}
              onChange={(e) => setAttackSkill(parseInt(e.target.value) || 0)}
              placeholder="50"
            />
            <p className="text-xs text-muted-foreground">
              Enter your Fight or weapon skill value (0-100)
            </p>
          </div>

          {/* Bonus/Penalty Dice */}
          <div className="grid gap-2">
            <Label htmlFor="bonus-penalty">Bonus/Penalty Dice</Label>
            <Select value={String(bonusPenalty)} onValueChange={(v) => setBonusPenalty(parseInt(v))}>
              <SelectTrigger id="bonus-penalty">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {BONUS_PENALTY_OPTIONS.map((option) => (
                  <SelectItem key={option.label} value={String(option.value)}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button onClick={handleAttack} disabled={!selectedTargetId || isLoading}>
            {isLoading ? "Rolling..." : "Roll"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
