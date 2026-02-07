/**
 * React hook for managing chase state
 * Handles fetching chase data and tracking participants, obstacles, and rounds
 */

import { useState, useEffect, useCallback } from 'react';
import type { Chase, ChaseParticipant, ChaseObstacle } from '../types/chase';
import { chaseApi } from '../lib/api';

export function useChaseState(chaseId: string | null) {
  const [chase, setChase] = useState<Chase | null>(null);
  const [currentRound, setCurrentRound] = useState<number>(0);
  const [location, setLocation] = useState<string>('');
  const [pressure, setPressure] = useState<number>(0);
  const [participants, setParticipants] = useState<ChaseParticipant[]>([]);
  const [obstacles, setObstacles] = useState<ChaseObstacle[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetch chase data by ID
   * Updates chase, currentRound, location, participants, and obstacles state
   */
  const fetchChase = useCallback(async () => {
    if (!chaseId) return;

    setIsLoading(true);
    setError(null);
    try {
      const data = await chaseApi.getById(chaseId);
      setChase(data);
      setCurrentRound(data.round);
      setLocation(data.location);
      setPressure(data.chase_metadata?.pressure as number || 0);
      setParticipants(data.participants || []);
      setObstacles(data.obstacles || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch chase';
      setError(message);
      console.error('Error fetching chase:', err);
    } finally {
      setIsLoading(false);
    }
  }, [chaseId]);

  // Auto-fetch when chaseId changes
  useEffect(() => {
    fetchChase();
  }, [fetchChase]);

  /**
   * Update a single participant in the list
   * Used for position/speed updates without full refetch
   * @param updatedParticipant - Participant with updated position or speed
   */
  const updateParticipant = useCallback((updatedParticipant: ChaseParticipant) => {
    setParticipants(prev =>
      prev.map(p => p.id === updatedParticipant.id ? updatedParticipant : p)
    );
  }, []);

  /**
   * Update obstacles list
   * Used when new obstacles are generated during chase
   * @param updatedObstacles - Updated list of obstacles
   */
  const updateObstacles = useCallback((updatedObstacles: ChaseObstacle[]) => {
    setObstacles(updatedObstacles);
  }, []);

  /**
   * Update chase round and pressure
   * Used after round resolution
   * @param round - New round number
   * @param newPressure - New pressure value
   */
  const updateRound = useCallback((round: number, newPressure?: number) => {
    setCurrentRound(round);
    if (newPressure !== undefined) {
      setPressure(newPressure);
    }
    setChase(prev => prev ? { ...prev, round } : null);
  }, []);

  return {
    chase,
    currentRound,
    location,
    pressure,
    participants,
    obstacles,
    isLoading,
    error,
    fetchChase,
    updateParticipant,
    updateObstacles,
    updateRound,
  };
}
