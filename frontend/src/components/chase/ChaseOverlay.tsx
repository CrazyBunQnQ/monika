import { useEffect, useState } from "react"
import { X, Minimize2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { Chase, ChaseParticipant, ChaseObstacle, ChaseLogEntry, ActionType, SuccessLevel } from "@/types/chase"
import { ChaseInfoPanel } from "./ChaseInfoPanel"
import { ChaseActionPanel } from "./ChaseActionPanel"
import { ChaseLogPanel } from "./ChaseLogPanel"

interface CheckResultData {
  success: boolean
  rollValue: number
  successLevel: SuccessLevel
  damage?: number
  speedPenalty?: number
  message?: string
}

interface ChaseOverlayProps {
  chaseId: string
  onClose: () => void
  onMinimize: () => void
}

/**
 * ChaseOverlay - Full-screen chase interface
 *
 * Features:
 * - Semi-transparent overlay (bg-black/60)
 * - ESC key to close
 * - Three-column layout: Chase Info | Actions | Chase Log
 * - Minimize to floating card
 * - Real-time chase state updates
 */
export function ChaseOverlay({
  chaseId,
  onClose,
  onMinimize,
}: ChaseOverlayProps) {
  // Local state for chase data
  const [chase, setChase] = useState<Chase | null>(null)
  const [pressure, setPressure] = useState<number>(0)
  const [currentParticipantId, setCurrentParticipantId] = useState<string | null>(null)
  const [currentParticipant, setCurrentParticipant] = useState<ChaseParticipant | null>(null)
  const [currentObstacle, setCurrentObstacle] = useState<ChaseObstacle | null>(null)
  const [logs, setLogs] = useState<ChaseLogEntry[]>([])
  const [lastCheckResult, setLastCheckResult] = useState<CheckResultData | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  /**
   * Handle ESC key to close overlay
   */
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose()
      }
    }

    window.addEventListener("keydown", handleEscape)
    return () => window.removeEventListener("keydown", handleEscape)
  }, [onClose])

  /**
   * Load initial chase data
   */
  useEffect(() => {
    const loadChase = async () => {
      setIsLoading(true)
      try {
        // TODO: Replace with actual API call using useChaseState
        // For now, this is a placeholder that will be replaced
        // when integrating with the actual hooks
        const response = await fetch(`/api/chase/${chaseId}`)
        if (!response.ok) throw new Error("Failed to load chase")
        const data: Chase = await response.json()

        setChase(data)
        setPressure((data.chase_metadata?.pressure as number) || 0)

        // Set current participant (first active participant)
        const activeParticipant = data.participants.find(p => p.is_active && p.is_player)
        if (activeParticipant) {
          setCurrentParticipantId(activeParticipant.id)
          setCurrentParticipant(activeParticipant)
        }

        // Set current obstacle (first obstacle for current round)
        const roundObstacle = data.obstacles.find(o => o.appears_at_round === data.round)
        setCurrentObstacle(roundObstacle || null)

      } catch (error) {
        console.error("Error loading chase:", error)
      } finally {
        setIsLoading(false)
      }
    }

    loadChase()
  }, [chaseId])

  /**
   * Handle action execution
   */
  const handleExecuteAction = async (actionType: ActionType, obstacleId?: string, skillValue?: number) => {
    if (!chase) return

    setIsLoading(true)
    try {
      // TODO: Replace with actual API call using useChaseActions
      const response = await fetch(`/api/chase/${chaseId}/round`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          actions: [{
            participant_id: currentParticipantId,
            action_type: actionType,
            obstacle_id: obstacleId,
            skill: skillValue,
          }]
        })
      })

      if (!response.ok) throw new Error("Failed to execute action")

      const result = await response.json()

      // Update check result if available
      if (result.check_result) {
        setLastCheckResult(result.check_result)
      }

      // Add log entry
      const newLog: ChaseLogEntry = {
        id: `log-${Date.now()}`,
        round: chase.round,
        type: "action",
        actor: currentParticipant?.name,
        description: `Performed ${actionType}`,
        timestamp: new Date(),
      }
      setLogs(prev => [newLog, ...prev])

      // Reload chase data to get updated state
      const updatedChase: Chase = await (await fetch(`/api/chase/${chaseId}`)).json()
      setChase(updatedChase)
      setPressure((updatedChase.chase_metadata?.pressure as number) || 0)

    } catch (error) {
      console.error("Error executing action:", error)
    } finally {
      setIsLoading(false)
    }
  }

  /**
   * Handle spend luck
   */
  const handleSpendLuck = async () => {
    // TODO: Implement luck spending
    console.log("Spend luck")
  }

  /**
   * Handle dismiss result
   */
  const handleDismissResult = () => {
    setLastCheckResult(null)
  }

  /**
   * Handle clear logs
   */
  const handleClearLogs = () => {
    setLogs([])
  }

  // Show loading state
  if (!chase) {
    return (
      <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center">
        <div className="bg-background rounded-lg shadow-xl p-8">
          <p className="text-lg">Loading chase...</p>
        </div>
      </div>
    )
  }

  return (
    <div
      className={cn(
        "fixed inset-0 z-50 bg-black/60 backdrop-blur-sm",
        "flex items-center justify-center p-4"
      )}
    >
      {/* Main Container */}
      <div
        className={cn(
          "bg-background rounded-lg shadow-xl w-full max-w-6xl",
          "flex flex-col max-h-[90vh]"
        )}
      >
        {/* Header Bar */}
        <div className="flex items-center justify-between p-4 border-b bg-slate-800 text-white">
          <div className="flex-1">
            {/* Chase Location/Title */}
            <h2 className="text-xl font-bold">
              {chase.location || "Chase"}
            </h2>
            {chase.setting && (
              <p className="text-sm text-slate-300">
                {chase.setting}
              </p>
            )}
          </div>

          {/* Round Indicator */}
          <div className="text-center">
            <div className="text-2xl font-bold">
              Round {chase.round}
            </div>
            {currentParticipant && (
              <div className="text-sm text-slate-300">
                {currentParticipant.name}&apos;s turn
              </div>
            )}
          </div>

          {/* Control Buttons */}
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={onMinimize}
              className="h-9 w-9 text-white hover:bg-slate-700"
            >
              <Minimize2 className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={onClose}
              className="h-9 w-9 text-white hover:bg-slate-700"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Three-Column Layout */}
        <div className="grid grid-cols-3 gap-4 p-4 h-[600px] overflow-hidden">
          {/* Left: Chase Info Panel */}
          <div className="flex-shrink-0 overflow-hidden">
            <ChaseInfoPanel
              chase={chase}
              pressure={pressure}
              currentParticipantId={currentParticipantId}
              className="h-full"
            />
          </div>

          {/* Center: Action Panel */}
          <div className="flex-shrink-0 overflow-hidden">
            <ChaseActionPanel
              chase={chase}
              currentParticipant={currentParticipant}
              currentObstacle={currentObstacle}
              lastCheckResult={lastCheckResult}
              onExecuteAction={handleExecuteAction}
              onSpendLuck={handleSpendLuck}
              onDismissResult={handleDismissResult}
              isLoading={isLoading}
              canSpendLuck={true}
              className="h-full"
            />
          </div>

          {/* Right: Chase Log Panel */}
          <div className="flex-shrink-0 overflow-hidden">
            <ChaseLogPanel
              logs={logs}
              onClear={handleClearLogs}
              className="h-full"
            />
          </div>
        </div>

        {/* Footer Status */}
        <div className="px-4 py-2 border-t text-xs text-muted-foreground">
          Press <kbd className="px-1 py-0.5 rounded bg-muted">ESC</kbd> to end chase
        </div>
      </div>
    </div>
  )
}
