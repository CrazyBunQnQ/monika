import { DistanceTrack } from "./DistanceTrack"
import { PressureBar } from "./PressureBar"
import { ParticipantList } from "./ParticipantList"
import type { Chase } from "@/types/chase"

interface ChaseInfoPanelProps {
  chase: Chase
  pressure: number
  currentParticipantId?: string | null
  className?: string
}

/**
 * ChaseInfoPanel - Main chase information panel
 *
 * Combines three sub-components:
 * - DistanceTrack: Visual distance between fugitive and pursuer
 * - PressureBar: Current pressure level indicator
 * - ParticipantList: All participants with their status
 *
 * Layout: Vertical stack with space-y-4 spacing
 */
export function ChaseInfoPanel({
  chase,
  pressure,
  currentParticipantId,
  className,
}: ChaseInfoPanelProps) {
  return (
    <div className={className}>
      {/* Three-column layout matching CombatOverlay */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Left: Distance Track */}
        <div className="w-full">
          <DistanceTrack chase={chase} />
        </div>

        {/* Center: Pressure Bar */}
        <div className="w-full">
          <PressureBar pressure={pressure} />
        </div>

        {/* Right: Participant List */}
        <div className="w-full">
          <ParticipantList
            chase={chase}
            currentParticipantId={currentParticipantId}
          />
        </div>
      </div>
    </div>
  )
}
