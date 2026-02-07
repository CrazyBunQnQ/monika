/**
 * useChaseState Hook Tests
 *
 * Tests the useChaseState custom hook which manages chase state.
 *
 * Test Coverage:
 * - Initial state correctness
 * - Chase data fetching
 * - chaseId change triggers
 * - Error handling
 * - State updates (participant, obstacles, round)
 * - Loading states
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useChaseState } from '../useChaseState'
import { chaseApi } from '@/lib/api'
import type { Chase, ChaseParticipant, ChaseObstacle } from '@/types/chase'

// Mock the chaseApi
vi.mock('@/lib/api', () => ({
  chaseApi: {
    getById: vi.fn(),
  },
}))

/**
 * Mock Chase data factory
 */
const createMockChase = (overrides?: Partial<Chase>): Chase => ({
  id: 'chase-123',
  session_id: 'session-456',
  state: 'active',
  round: 3,
  location: '2',
  setting: 'Dark Alley',
  started_at: '2025-01-15T10:00:00Z',
  ended_at: null,
  end_reason: null,
  failed_forward_scene: null,
  chase_metadata: {
    pressure: 65,
  },
  participants: [
    {
      id: 'part-1',
      chase_id: 'chase-123',
      character_id: 1,
      name: 'Alice',
      role: 'fugitive',
      is_player: true,
      move_rate: 8,
      current_speed: 8,
      position_index: 0,
      is_active: true,
      is_exhausted: false,
      failed_obstacle_count: 0,
      speed_penalty: 0,
      consecutive_failures: 0,
      participant_metadata: {},
      created_at: '2025-01-15T10:00:00Z',
      updated_at: '2025-01-15T10:00:00Z',
    },
  ],
  obstacles: [
    {
      id: 'obs-1',
      chase_id: 'chase-123',
      name: 'Broken Fence',
      description: 'A damaged fence blocking the path',
      obstacle_type: 'physical',
      appears_at_round: 2,
      appears_at_distance: 1,
      difficulty: 'regular',
      skill_required: 'str',
      failure_penalty: 2,
      failure_damage: null,
      failure_san_cost: null,
      fail_forward_result: null,
      details: {},
      created_at: '2025-01-15T10:00:00Z',
    },
  ],
  ...overrides,
})

describe('useChaseState', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('Initial State', () => {
    it('should return initial state with null values', () => {
      const { result } = renderHook(() => useChaseState(null))

      expect(result.current.chase).toBeNull()
      expect(result.current.currentRound).toBe(0)
      expect(result.current.location).toBe('')
      expect(result.current.pressure).toBe(0)
      expect(result.current.participants).toEqual([])
      expect(result.current.obstacles).toEqual([])
      expect(result.current.isLoading).toBe(false)
      expect(result.current.error).toBeNull()
    })

    it('should not fetch data when chaseId is null', () => {
      renderHook(() => useChaseState(null))

      expect(chaseApi.getById).not.toHaveBeenCalled()
    })

    it('should not fetch data when chaseId is empty string', () => {
      renderHook(() => useChaseState(''))

      expect(chaseApi.getById).not.toHaveBeenCalled()
    })
  })

  describe('Data Fetching', () => {
    it('should fetch chase data when chaseId is provided', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      // Initially loading
      expect(result.current.isLoading).toBe(true)

      // Wait for fetch to complete
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Verify data was fetched
      expect(chaseApi.getById).toHaveBeenCalledWith('chase-123')
      expect(result.current.chase).toEqual(mockChase)
      expect(result.current.currentRound).toBe(3)
      expect(result.current.location).toBe('2')
      expect(result.current.pressure).toBe(65)
      expect(result.current.participants).toEqual(mockChase.participants)
      expect(result.current.obstacles).toEqual(mockChase.obstacles)
    })

    it('should set loading state during fetch', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve(mockChase), 100))
      )

      const { result } = renderHook(() => useChaseState('chase-123'))

      // Should be loading immediately
      expect(result.current.isLoading).toBe(true)

      // Wait for completion
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })
    })

    it('should reset error state on successful fetch', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.error).toBeNull()
    })
  })

  describe('chaseId Change Triggers', () => {
    it('should refetch data when chaseId changes', async () => {
      const mockChase1 = createMockChase({ id: 'chase-1', round: 1 })
      const mockChase2 = createMockChase({ id: 'chase-2', round: 2 })

      vi.mocked(chaseApi.getById)
        .mockResolvedValueOnce(mockChase1)
        .mockResolvedValueOnce(mockChase2)

      const { result, rerender } = renderHook(
        ({ chaseId }) => useChaseState(chaseId),
        { initialProps: { chaseId: 'chase-1' } }
      )

      // Wait for first fetch
      await waitFor(() => {
        expect(result.current.chase?.id).toBe('chase-1')
      })

      // Change chaseId
      rerender({ chaseId: 'chase-2' })

      // Should load new data
      await waitFor(() => {
        expect(result.current.chase?.id).toBe('chase-2')
      })

      expect(chaseApi.getById).toHaveBeenCalledWith('chase-1')
      expect(chaseApi.getById).toHaveBeenCalledWith('chase-2')
    })

    it('should reset state when chaseId changes to null', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result, rerender } = renderHook(
        ({ chaseId }) => useChaseState(chaseId),
        { initialProps: { chaseId: 'chase-123' } }
      )

      // Wait for initial fetch
      await waitFor(() => {
        expect(result.current.chase).not.toBeNull()
      })

      // Change to null
      rerender({ chaseId: null })

      // State should remain as is (no refetch triggered)
      expect(chaseApi.getById).toHaveBeenCalledTimes(1)
    })
  })

  describe('Error Handling', () => {
    it('should handle API errors', async () => {
      const error = new Error('Network error')
      vi.mocked(chaseApi.getById).mockRejectedValue(error)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.error).toBe('Network error')
      expect(result.current.chase).toBeNull()
    })

    it('should handle non-Error errors', async () => {
      vi.mocked(chaseApi.getById).mockRejectedValue('String error')

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.error).toBe('Failed to fetch chase')
    })

    it('should clear previous error on successful fetch', async () => {
      // First call fails
      vi.mocked(chaseApi.getById).mockRejectedValueOnce(new Error('First error'))
      const mockChase = createMockChase()

      const { result, rerender } = renderHook(
        ({ chaseId }) => useChaseState(chaseId),
        { initialProps: { chaseId: 'chase-1' } }
      )

      // Wait for error
      await waitFor(() => {
        expect(result.current.error).toBe('First error')
      })

      // Second call succeeds
      vi.mocked(chaseApi.getById).mockResolvedValueOnce(mockChase)
      rerender({ chaseId: 'chase-2' })

      // Wait for success
      await waitFor(() => {
        expect(result.current.error).toBeNull()
      })
    })
  })

  describe('State Updates', () => {
    it('should update participant in list', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updatedParticipant: ChaseParticipant = {
        ...mockChase.participants[0],
        current_speed: 10,
      }

      act(() => {
        result.current.updateParticipant(updatedParticipant)
      })

      expect(result.current.participants[0].current_speed).toBe(10)
      expect(result.current.participants[0].id).toBe('part-1')
    })

    it('should update obstacles list', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const newObstacles: ChaseObstacle[] = [
        ...mockChase.obstacles,
        {
          id: 'obs-2',
          chase_id: 'chase-123',
          name: 'Locked Door',
          description: 'A door blocking the way',
          obstacle_type: 'physical',
          appears_at_round: 3,
          appears_at_distance: 2,
          difficulty: 'hard',
          skill_required: 'str',
          failure_penalty: 3,
          failure_damage: 2,
          failure_san_cost: null,
          fail_forward_result: null,
          details: {},
          created_at: '2025-01-15T11:00:00Z',
        },
      ]

      act(() => {
        result.current.updateObstacles(newObstacles)
      })

      expect(result.current.obstacles.length).toBe(2)
      expect(result.current.obstacles[1].name).toBe('Locked Door')
    })

    it('should update round and pressure', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      act(() => {
        result.current.updateRound(5, 80)
      })

      expect(result.current.currentRound).toBe(5)
      expect(result.current.pressure).toBe(80)
    })

    it('should update round without pressure', async () => {
      const mockChase = createMockChase({ chase_metadata: { pressure: 65 } })
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const initialPressure = result.current.pressure

      act(() => {
        result.current.updateRound(4)
      })

      expect(result.current.currentRound).toBe(4)
      expect(result.current.pressure).toBe(initialPressure) // Should not change
    })

    it('should update chase object when round changes', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      act(() => {
        result.current.updateRound(6)
      })

      expect(result.current.chase?.round).toBe(6)
    })
  })

  describe('Manual Refetch', () => {
    it('should allow manual refetch via fetchChase function', async () => {
      const mockChase1 = createMockChase({ round: 1 })
      const mockChase2 = createMockChase({ round: 2 })

      vi.mocked(chaseApi.getById)
        .mockResolvedValueOnce(mockChase1)
        .mockResolvedValueOnce(mockChase2)

      const { result } = renderHook(() => useChaseState('chase-123'))

      // Wait for initial fetch
      await waitFor(() => {
        expect(result.current.chase?.round).toBe(1)
      })

      // Manual refetch
      await act(async () => {
        await result.current.fetchChase()
      })

      expect(result.current.chase?.round).toBe(2)
      expect(chaseApi.getById).toHaveBeenCalledTimes(2)
    })
  })

  describe('Pressure Extraction', () => {
    it('should extract pressure from chase_metadata', async () => {
      const mockChase = createMockChase({
        chase_metadata: { pressure: 75 },
      })
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.pressure).toBe(75)
    })

    it('should default to 0 when pressure is not in metadata', async () => {
      const mockChase = createMockChase({
        chase_metadata: {},
      })
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.pressure).toBe(0)
    })

    it('should default to 0 when chase_metadata is null', async () => {
      const mockChase = createMockChase({
        chase_metadata: null as unknown as Record<string, unknown>,
      })
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.pressure).toBe(0)
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty participants array', async () => {
      const mockChase = createMockChase({
        participants: [],
      })
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.participants).toEqual([])
    })

    it('should handle empty obstacles array', async () => {
      const mockChase = createMockChase({
        obstacles: [],
      })
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.obstacles).toEqual([])
    })

    it('should handle participant not found in update', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const nonExistentParticipant: ChaseParticipant = {
        id: 'non-existent',
        chase_id: 'chase-123',
        character_id: null,
        name: 'Ghost',
        role: 'fugitive',
        is_player: false,
        move_rate: 8,
        current_speed: 8,
        position_index: 0,
        is_active: true,
        is_exhausted: false,
        failed_obstacle_count: 0,
        speed_penalty: 0,
        consecutive_failures: 0,
        participant_metadata: {},
        created_at: '2025-01-15T10:00:00Z',
        updated_at: '2025-01-15T10:00:00Z',
      }

      act(() => {
        result.current.updateParticipant(nonExistentParticipant)
      })

      // Original participant should remain unchanged
      expect(result.current.participants.length).toBe(1)
      expect(result.current.participants[0].id).toBe('part-1')
    })

    it('should handle chase object being null when updating round', async () => {
      const { result } = renderHook(() => useChaseState(null))

      act(() => {
        result.current.updateRound(5, 50)
      })

      // Should not throw error
      expect(result.current.currentRound).toBe(5)
      expect(result.current.pressure).toBe(50)
      expect(result.current.chase).toBeNull()
    })
  })

  describe('Return Values', () => {
    it('should return all expected state values', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Check all return values exist
      expect(result.current).toHaveProperty('chase')
      expect(result.current).toHaveProperty('currentRound')
      expect(result.current).toHaveProperty('location')
      expect(result.current).toHaveProperty('pressure')
      expect(result.current).toHaveProperty('participants')
      expect(result.current).toHaveProperty('obstacles')
      expect(result.current).toHaveProperty('isLoading')
      expect(result.current).toHaveProperty('error')
      expect(result.current).toHaveProperty('fetchChase')
      expect(result.current).toHaveProperty('updateParticipant')
      expect(result.current).toHaveProperty('updateObstacles')
      expect(result.current).toHaveProperty('updateRound')
    })

    it('should return functions that maintain referential equality', async () => {
      const mockChase = createMockChase()
      vi.mocked(chaseApi.getById).mockResolvedValue(mockChase)

      const { result } = renderHook(() => useChaseState('chase-123'))

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const fetchChase1 = result.current.fetchChase
      const fetchChase2 = result.current.fetchChase

      // Functions should maintain reference (useCallback)
      expect(fetchChase1).toBe(fetchChase2)
    })
  })
})
