/**
 * React hook for combat action operations
 * Provides methods to interact with combat API endpoints
 */

import { useCallback } from 'react';
import { combatApi } from '../lib/api';
import type {
  AttackRequest,
  HealRequest,
  TurnResponse,
  AttackResponse,
  HealResponse,
  Combat,
} from '../types/combat';

export function useCombatActions(combatId: string | null) {
  /**
   * Advance to next turn in combat
   * @returns TurnResponse with updated turn order and current combatant
   * @throws Error if no combat ID is provided
   */
  const nextTurn = useCallback(async (): Promise<TurnResponse> => {
    if (!combatId) {
      throw new Error('No combat ID provided');
    }

    const response = await combatApi.nextTurn(combatId);
    return response;
  }, [combatId]);

  /**
   * Make an attack against a target combatant
   * @param request - AttackRequest with attacker_id, target_id, and roll data
   * @returns AttackResponse with attack result and damage dealt
   * @throws Error if no combat ID is provided
   */
  const attack = useCallback(async (request: AttackRequest): Promise<AttackResponse> => {
    if (!combatId) {
      throw new Error('No combat ID provided');
    }

    const response = await combatApi.attack(combatId, request);
    return response;
  }, [combatId]);

  /**
   * Heal a combatant using first aid
   * @param request - HealRequest with target_id, heal_amount, and skill data
   * @returns HealResponse with healing result and updated HP
   * @throws Error if no combat ID is provided
   */
  const heal = useCallback(async (request: HealRequest): Promise<HealResponse> => {
    if (!combatId) {
      throw new Error('No combat ID provided');
    }

    const response = await combatApi.heal(combatId, request);
    return response;
  }, [combatId]);

  /**
   * End the current combat session
   * @returns Combat object with ended state
   * @throws Error if no combat ID is provided
   */
  const endCombat = useCallback(async (): Promise<Combat> => {
    if (!combatId) {
      throw new Error('No combat ID provided');
    }

    const response = await combatApi.end(combatId);
    return response;
  }, [combatId]);

  return {
    nextTurn,
    attack,
    heal,
    endCombat,
  };
}
