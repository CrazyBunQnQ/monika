/**
 * React hook for managing combat state
 * Handles fetching combat data and updating combatants/turn info
 */

import { useState, useEffect, useCallback } from 'react';
import type { Combat, Combatant, TurnResponse } from '../types/combat';
import { combatApi } from '../lib/api';

export function useCombatState(combatId: string | null) {
  const [combat, setCombat] = useState<Combat | null>(null);
  const [combatants, setCombatants] = useState<Combatant[]>([]);
  const [currentTurn, setCurrentTurn] = useState<Combatant | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetch combat data by ID
   * Updates combat, combatants, and currentTurn state
   */
  const fetchCombat = useCallback(async () => {
    if (!combatId) return;

    setIsLoading(true);
    setError(null);
    try {
      const data = await combatApi.getById(combatId);
      setCombat(data);
      setCombatants(data.combatants || []);
      setCurrentTurn(data.current_turn || null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch combat';
      setError(message);
      console.error('Error fetching combat:', err);
    } finally {
      setIsLoading(false);
    }
  }, [combatId]);

  // Auto-fetch when combatId changes
  useEffect(() => {
    fetchCombat();
  }, [fetchCombat]);

  /**
   * Update combatants and currentTurn from a turn response
   * Called after advancing to next turn
   * @param response - TurnResponse from nextTurn API call
   */
  const updateFromTurnResponse = useCallback((response: TurnResponse) => {
    setCombatants(response.turn_order);
    setCurrentTurn(response.current_combatant);
    setCombat(prev => prev ? { ...prev, current_round: response.current_round } : null);
  }, []);

  /**
   * Update a single combatant in the list
   * Used for damage/healing updates without full refetch
   * @param updatedCombatant - Combatant with updated HP or status
   */
  const updateCombatant = useCallback((updatedCombatant: Combatant) => {
    setCombatants(prev =>
      prev.map(c => c.id === updatedCombatant.id ? updatedCombatant : c)
    );
    setCurrentTurn(prev => prev?.id === updatedCombatant.id ? updatedCombatant : prev);
  }, []);

  return {
    combat,
    combatants,
    currentTurn,
    isLoading,
    error,
    fetchCombat,
    updateFromTurnResponse,
    updateCombatant,
  };
}
