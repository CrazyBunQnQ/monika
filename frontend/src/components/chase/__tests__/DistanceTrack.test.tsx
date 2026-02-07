/**
 * DistanceTrack Component Tests
 *
 * Tests the DistanceTrack component which visualizes the distance between
 * fugitives and pursuers in the chase system.
 *
 * Test Coverage:
 * - Rendering correctness
 * - Distance level display
 * - Track symbol calculations
 * - Edge cases (caught, escaped, extreme values)
 * - Participant count display
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DistanceTrack } from '../DistanceTrack'
import type { Chase } from '@/types/chase'

/**
 * Mock Chase data factory for testing
 */
const createMockChase = (overrides?: Partial<Chase>): Chase => ({
  id: 'chase-123',
  session_id: 'session-456',
  state: 'active',
  round: 1,
  location: '2',
  setting: 'Dark Alley',
  started_at: '2025-01-15T10:00:00Z',
  ended_at: null,
  end_reason: null,
  failed_forward_scene: null,
  chase_metadata: {},
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
    {
      id: 'part-2',
      chase_id: 'chase-123',
      character_id: null,
      name: 'Cultist',
      role: 'pursuer',
      is_player: false,
      move_rate: 9,
      current_speed: 9,
      position_index: 1,
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
  obstacles: [],
  ...overrides,
})

describe('DistanceTrack', () => {
  describe('Rendering', () => {
    it('should render without crashing', () => {
      const chase = createMockChase()
      const { container } = render(<DistanceTrack chase={chase} />)

      expect(container).toBeInTheDocument()
    })

    it('should render the distance title', () => {
      const chase = createMockChase()
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Distance')).toBeInTheDocument()
    })

    it('should render the distance level badge', () => {
      const chase = createMockChase({ location: '2' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 2')).toBeInTheDocument()
    })

    it('should render current distance label', () => {
      const chase = createMockChase({ location: '2' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Current Distance')).toBeInTheDocument()
    })

    it('should render fugitive and pursuer labels', () => {
      const chase = createMockChase()
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('逃跑者')).toBeInTheDocument()
      expect(screen.getByText('追逐者')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      const chase = createMockChase()
      const { container } = render(
        <DistanceTrack chase={chase} className="custom-class" />
      )

      const card = container.querySelector('.custom-class')
      expect(card).toBeInTheDocument()
    })
  })

  describe('Distance Level Display', () => {
    it('should display Level 0 (Caught)', () => {
      const chase = createMockChase({ location: '0' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 0')).toBeInTheDocument()
      expect(screen.getByText('Caught!')).toBeInTheDocument()
    })

    it('should display Level 1 (Very Close)', () => {
      const chase = createMockChase({ location: '1' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 1')).toBeInTheDocument()
      expect(screen.getByText('Very Close')).toBeInTheDocument()
    })

    it('should display Level 2 (Moderate)', () => {
      const chase = createMockChase({ location: '2' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 2')).toBeInTheDocument()
      expect(screen.getByText('Moderate')).toBeInTheDocument()
    })

    it('should display Level 3 (Far Ahead)', () => {
      const chase = createMockChase({ location: '3' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 3')).toBeInTheDocument()
      expect(screen.getByText('Far Ahead')).toBeInTheDocument()
    })

    it('should display Level 4 (Escaped)', () => {
      const chase = createMockChase({ location: '4' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 4')).toBeInTheDocument()
      expect(screen.getByText('Escaped!')).toBeInTheDocument()
    })

    it('should clamp invalid values to valid range', () => {
      const chase = createMockChase({ location: '10' })
      render(<DistanceTrack chase={chase} />)

      // Should clamp to max level 4
      expect(screen.getByText('Level 4')).toBeInTheDocument()
    })

    it('should handle negative values', () => {
      const chase = createMockChase({ location: '-5' })
      render(<DistanceTrack chase={chase} />)

      // Should clamp to min level 0
      expect(screen.getByText('Level 0')).toBeInTheDocument()
    })

    it('should default to level 2 for non-numeric values', () => {
      const chase = createMockChase({ location: 'invalid' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 2')).toBeInTheDocument()
    })
  })

  describe('Track Symbol Calculations', () => {
    it('should render correct track symbols for level 0', () => {
      const chase = createMockChase({ location: '0' })
      const { container } = render(<DistanceTrack chase={chase} />)

      // Level 0: caught (no gap)
      const track = container.textContent || ''
      expect(track).toContain('●')
      expect(track).toContain('━')
    })

    it('should render correct track symbols for level 2 (moderate)', () => {
      const chase = createMockChase({ location: '2' })
      const { container } = render(<DistanceTrack chase={chase} />)

      // Level 2: moderate gap
      const track = container.textContent || ''
      expect(track).toContain('●')
      expect(track).toContain('━')
    })

    it('should render correct track symbols for level 4 (escaped)', () => {
      const chase = createMockChase({ location: '4' })
      const { container } = render(<DistanceTrack chase={chase} />)

      // Level 4: maximum gap
      const track = container.textContent || ''
      expect(track).toContain('●')
      expect(track).toContain('━')
    })

    it('should render colored dots for fugitive (green) and pursuer (red)', () => {
      const chase = createMockChase()
      const { container } = render(<DistanceTrack chase={chase} />)

      const dots = container.querySelectorAll('span')
      const greenDots = Array.from(dots).filter(
        (el) => el.textContent === '●' && el.className.includes('green')
      )
      const redDots = Array.from(dots).filter(
        (el) => el.textContent === '●' && el.className.includes('red')
      )

      expect(greenDots.length).toBeGreaterThan(0)
      expect(redDots.length).toBeGreaterThan(0)
    })
  })

  describe('Badge Variants', () => {
    it('should use destructive variant for level 0 (caught)', () => {
      const chase = createMockChase({ location: '0' })
      const { container } = render(<DistanceTrack chase={chase} />)

      const badge = container.querySelector('.badge-destructive, [class*="destructive"]')
      expect(badge).toBeInTheDocument()
    })

    it('should use default variant for level 4 (escaped)', () => {
      const chase = createMockChase({ location: '4' })
      const { container } = render(<DistanceTrack chase={chase} />)

      // Default variant badges have bg-primary text-primary-foreground classes
      const badge = screen.getByText('Level 4')
      expect(badge).toBeInTheDocument()
      expect(badge.className).toContain('bg-primary')
    })

    it('should use secondary variant for level 3 (far ahead)', () => {
      const chase = createMockChase({ location: '3' })
      const { container } = render(<DistanceTrack chase={chase} />)

      const badge = container.querySelector('.badge-secondary, [class*="secondary"]')
      expect(badge).toBeInTheDocument()
    })

    it('should use outline variant for levels 1-2 (close)', () => {
      const chase = createMockChase({ location: '1' })
      const { container } = render(<DistanceTrack chase={chase} />)

      const badge = container.querySelector('.badge-outline, [class*="outline"]')
      expect(badge).toBeInTheDocument()
    })
  })

  describe('Color Coding', () => {
    it('should apply red color for level 0 (caught)', () => {
      const chase = createMockChase({ location: '0' })
      const { container } = render(<DistanceTrack chase={chase} />)

      const label = screen.getByText('Caught!')
      expect(label.className).toContain('text-red')
    })

    it('should apply yellow color for levels 1-3 (in progress)', () => {
      const chase = createMockChase({ location: '2' })
      const { container } = render(<DistanceTrack chase={chase} />)

      const label = screen.getByText('Moderate')
      expect(label.className).toContain('text-yellow')
    })

    it('should apply green color for level 4 (escaped)', () => {
      const chase = createMockChase({ location: '4' })
      const { container } = render(<DistanceTrack chase={chase} />)

      const label = screen.getByText('Escaped!')
      expect(label.className).toContain('text-green')
    })
  })

  describe('Participant Display', () => {
    it('should display correct count for single fugitive', () => {
      const chase = createMockChase({
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
      })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('1 Fugitive')).toBeInTheDocument()
    })

    it('should display correct count for multiple fugitives', () => {
      const chase = createMockChase({
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
          {
            id: 'part-2',
            chase_id: 'chase-123',
            character_id: 2,
            name: 'Bob',
            role: 'fugitive',
            is_player: true,
            move_rate: 7,
            current_speed: 7,
            position_index: 1,
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
      })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('2 Fugitives')).toBeInTheDocument()
    })

    it('should display correct count for single pursuer', () => {
      const chase = createMockChase({
        participants: [
          {
            id: 'part-1',
            chase_id: 'chase-123',
            character_id: null,
            name: 'Cultist',
            role: 'pursuer',
            is_player: false,
            move_rate: 9,
            current_speed: 9,
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
      })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('1 Pursuer')).toBeInTheDocument()
    })

    it('should display correct count for multiple pursuers', () => {
      const chase = createMockChase({
        participants: [
          {
            id: 'part-1',
            chase_id: 'chase-123',
            character_id: null,
            name: 'Cultist 1',
            role: 'pursuer',
            is_player: false,
            move_rate: 9,
            current_speed: 9,
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
          {
            id: 'part-2',
            chase_id: 'chase-123',
            character_id: null,
            name: 'Cultist 2',
            role: 'pursuer',
            is_player: false,
            move_rate: 9,
            current_speed: 9,
            position_index: 1,
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
      })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('2 Pursuers')).toBeInTheDocument()
    })

    it('should only count active participants', () => {
      const chase = createMockChase({
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
          {
            id: 'part-2',
            chase_id: 'chase-123',
            character_id: 2,
            name: 'Bob',
            role: 'fugitive',
            is_player: true,
            move_rate: 7,
            current_speed: 7,
            position_index: 1,
            is_active: false, // Inactive
            is_exhausted: false,
            failed_obstacle_count: 0,
            speed_penalty: 0,
            consecutive_failures: 0,
            participant_metadata: {},
            created_at: '2025-01-15T10:00:00Z',
            updated_at: '2025-01-15T10:00:00Z',
          },
        ],
      })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('1 Fugitive')).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty participants array', () => {
      const chase = createMockChase({
        participants: [],
      })
      const { container } = render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('0 Fugitives')).toBeInTheDocument()
      expect(screen.getByText('0 Pursuers')).toBeInTheDocument()
    })

    it('should handle string location that parses to number', () => {
      const chase = createMockChase({ location: '3' })
      render(<DistanceTrack chase={chase} />)

      expect(screen.getByText('Level 3')).toBeInTheDocument()
    })

    it('should handle decimal location values', () => {
      const chase = createMockChase({ location: '2.5' })
      render(<DistanceTrack chase={chase} />)

      // Should parse to 2
      expect(screen.getByText('Level 2')).toBeInTheDocument()
    })
  })
})
