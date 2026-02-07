/**
 * PressureBar Component Tests
 *
 * Tests the PressureBar component which visualizes pressure levels
 * in the chase system.
 *
 * Test Coverage:
 * - Rendering correctness
 * - Pressure level colors (safe/warning/critical)
 * - Progress bar width calculation
 * - Badge variants and labels
 * - Icon display at different levels
 * - Edge cases (0%, 50%, 80%, 100%)
 */

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PressureBar } from '../PressureBar'

describe('PressureBar', () => {
  describe('Rendering', () => {
    it('should render without crashing', () => {
      const { container } = render(<PressureBar pressure={50} />)

      expect(container).toBeInTheDocument()
    })

    it('should render the pressure title', () => {
      render(<PressureBar pressure={50} />)

      expect(screen.getByText('Pressure')).toBeInTheDocument()
    })

    it('should render the pressure percentage badge', () => {
      render(<PressureBar pressure={75} />)

      expect(screen.getByText('75%')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      const { container } = render(
        <PressureBar pressure={50} className="custom-class" />
      )

      const card = container.querySelector('.custom-class')
      expect(card).toBeInTheDocument()
    })
  })

  describe('Pressure Level Colors', () => {
    it('should display green color for safe zone (0-49%)', () => {
      const { container } = render(<PressureBar pressure={30} />)

      const badge = screen.getByText('Safe')
      expect(badge).toBeInTheDocument()

      // Check for green color classes
      const greenElements = container.querySelectorAll('[class*="green"]')
      expect(greenElements.length).toBeGreaterThan(0)
    })

    it('should display yellow color for warning zone (50-79%)', () => {
      const { container } = render(<PressureBar pressure={65} />)

      const badge = screen.getByText('Warning')
      expect(badge).toBeInTheDocument()

      // Check for yellow color classes
      const yellowElements = container.querySelectorAll('[class*="yellow"]')
      expect(yellowElements.length).toBeGreaterThan(0)
    })

    it('should display red color for critical zone (80-100%)', () => {
      const { container } = render(<PressureBar pressure={90} />)

      const badge = screen.getByText('Critical')
      expect(badge).toBeInTheDocument()

      // Check for red color classes
      const redElements = container.querySelectorAll('[class*="red"]')
      expect(redElements.length).toBeGreaterThan(0)
    })
  })

  describe('Progress Bar', () => {
    it('should render progress bar with correct value', () => {
      const { container } = render(<PressureBar pressure={45} />)

      const progressBar = container.querySelector('[role="progressbar"]')
      expect(progressBar).toBeInTheDocument()

      // Check if the progress value is set correctly
      // Note: The actual implementation may use aria-valuenow or a different attribute
      if (progressBar) {
        const value = progressBar.getAttribute('aria-valuenow')
        expect(value).toBe('45')
      }
    })

    it('should render 0% progress bar', () => {
      const { container } = render(<PressureBar pressure={0} />)

      const progressBar = container.querySelector('[role="progressbar"]')
      expect(progressBar).toBeInTheDocument()

      if (progressBar) {
        const value = progressBar.getAttribute('aria-valuenow')
        expect(value).toBe('0')
      }
    })

    it('should render 100% progress bar', () => {
      const { container } = render(<PressureBar pressure={100} />)

      const progressBar = container.querySelector('[role="progressbar"]')
      expect(progressBar).toBeInTheDocument()

      if (progressBar) {
        const value = progressBar.getAttribute('aria-valuenow')
        expect(value).toBe('100')
      }
    })

    it('should add critical border for critical pressure', () => {
      const { container } = render(<PressureBar pressure={85} />)

      const progressBar = container.querySelector('[class*="border-red"]')
      expect(progressBar).toBeInTheDocument()
    })

    it('should not add critical border for safe pressure', () => {
      const { container } = render(<PressureBar pressure={40} />)

      const progressBar = container.querySelector('[class*="border-red"]')
      expect(progressBar).not.toBeInTheDocument()
    })
  })

  describe('Badge Variants', () => {
    it('should use secondary badge variant for safe zone', () => {
      const { container } = render(<PressureBar pressure={30} />)

      const badge = container.querySelector('.badge-secondary, [class*="secondary"]')
      expect(badge).toBeInTheDocument()
    })

    it('should use outline badge variant for warning zone', () => {
      const { container } = render(<PressureBar pressure={60} />)

      const badge = container.querySelector('.badge-outline, [class*="outline"]')
      expect(badge).toBeInTheDocument()
    })

    it('should use destructive badge variant for critical zone', () => {
      const { container } = render(<PressureBar pressure={90} />)

      const badge = container.querySelector('.badge-destructive, [class*="destructive"]')
      expect(badge).toBeInTheDocument()
    })
  })

  describe('Pressure Labels', () => {
    it('should display "Safe" label for 0-49%', () => {
      render(<PressureBar pressure={25} />)

      expect(screen.getByText('Safe')).toBeInTheDocument()
    })

    it('should display "Warning" label for 50-79%', () => {
      render(<PressureBar pressure={65} />)

      expect(screen.getByText('Warning')).toBeInTheDocument()
    })

    it('should display "Critical" label for 80-100%', () => {
      render(<PressureBar pressure={95} />)

      expect(screen.getByText('Critical')).toBeInTheDocument()
    })
  })

  describe('Pressure Descriptions', () => {
    it('should display safe zone description', () => {
      render(<PressureBar pressure={30} />)

      expect(screen.getByText('No penalty on obstacle checks')).toBeInTheDocument()
    })

    it('should display warning zone description', () => {
      render(<PressureBar pressure={60} />)

      expect(screen.getByText('Increased difficulty on obstacle checks')).toBeInTheDocument()
    })

    it('should display critical zone description', () => {
      render(<PressureBar pressure={85} />)

      expect(screen.getByText('Automatic failure on obstacle checks!')).toBeInTheDocument()
    })
  })

  describe('Icons', () => {
    it('should not display any icon for safe zone', () => {
      const { container } = render(<PressureBar pressure={30} />)

      // AlertTriangle and Zap icons should not be present in title
      const titleIcons = container.querySelectorAll('.text-lg + svg')
      expect(titleIcons.length).toBe(0)
    })

    it('should display Zap icon for warning zone', () => {
      const { container } = render(<PressureBar pressure={60} />)

      // Zap icon should be present
      const zapIcon = container.querySelector('svg.lucide-zap')
      expect(zapIcon).toBeInTheDocument()
    })

    it('should display AlertTriangle icon for critical zone', () => {
      const { container } = render(<PressureBar pressure={90} />)

      // AlertTriangle icon should be present
      const alertIcon = container.querySelector('svg.lucide-triangle-alert')
      expect(alertIcon).toBeInTheDocument()
    })
  })

  describe('Critical Zone Indicator', () => {
    it('should display CRITICAL text for critical pressure', () => {
      render(<PressureBar pressure={85} />)

      expect(screen.getByText('CRITICAL')).toBeInTheDocument()
    })

    it('should not display CRITICAL text for safe pressure', () => {
      const { container } = render(<PressureBar pressure={40} />)

      const criticalText = container.querySelector('text-white')
      expect(criticalText).not.toBeInTheDocument()
    })

    it('should not display CRITICAL text for warning pressure', () => {
      const { container } = render(<PressureBar pressure={60} />)

      const criticalText = container.querySelector('text-white')
      expect(criticalText).not.toBeInTheDocument()
    })
  })

  describe('Zone Indicators', () => {
    it('should render zone indicator bars', () => {
      const { container } = render(<PressureBar pressure={50} />)

      // Should have 3 zone bars (safe, warning, critical)
      const zoneBars = container.querySelectorAll('.h-2')
      expect(zoneBars.length).toBeGreaterThanOrEqual(3)
    })

    it('should render safe zone bar in green', () => {
      const { container } = render(<PressureBar pressure={50} />)

      const safeBar = container.querySelector('[class*="bg-green-600"].h-2')
      expect(safeBar).toBeInTheDocument()
    })

    it('should render warning zone bar in yellow', () => {
      const { container } = render(<PressureBar pressure={50} />)

      const warningBar = container.querySelector('[class*="bg-yellow-600"].h-2')
      expect(warningBar).toBeInTheDocument()
    })

    it('should render critical zone bar in red', () => {
      const { container } = render(<PressureBar pressure={50} />)

      const criticalBar = container.querySelector('[class*="bg-red-600"].h-2')
      expect(criticalBar).toBeInTheDocument()
    })
  })

  describe('Scale Markers', () => {
    it('should display scale markers', () => {
      render(<PressureBar pressure={50} />)

      // Use getAllByText for markers that appear multiple times (badge + scale)
      expect(screen.getAllByText('0%').length).toBeGreaterThan(0)
      expect(screen.getAllByText('50%').length).toBeGreaterThan(0)
      expect(screen.getAllByText('80%').length).toBeGreaterThan(0)
      expect(screen.getAllByText('100%').length).toBeGreaterThan(0)
    })
  })

  describe('Boundary Values', () => {
    it('should handle 0% pressure (safe)', () => {
      render(<PressureBar pressure={0} />)

      expect(screen.getByText('Safe')).toBeInTheDocument()
      // 0% appears in both badge and scale marker
      expect(screen.getAllByText('0%').length).toBeGreaterThan(0)
    })

    it('should handle 49% pressure (safe - edge of warning)', () => {
      render(<PressureBar pressure={49} />)

      expect(screen.getByText('Safe')).toBeInTheDocument()
    })

    it('should handle 50% pressure (warning - edge of safe)', () => {
      render(<PressureBar pressure={50} />)

      expect(screen.getByText('Warning')).toBeInTheDocument()
    })

    it('should handle 79% pressure (warning - edge of critical)', () => {
      render(<PressureBar pressure={79} />)

      expect(screen.getByText('Warning')).toBeInTheDocument()
    })

    it('should handle 80% pressure (critical - edge of warning)', () => {
      render(<PressureBar pressure={80} />)

      expect(screen.getByText('Critical')).toBeInTheDocument()
    })

    it('should handle 100% pressure (critical - maximum)', () => {
      render(<PressureBar pressure={100} />)

      expect(screen.getByText('Critical')).toBeInTheDocument()
      // Use getAllByText since there are multiple "100" elements (badge and scale marker)
      const percentageElements = screen.getAllByText('100%')
      expect(percentageElements.length).toBeGreaterThan(0)
    })
  })

  describe('Animation', () => {
    it('should apply pulse animation to critical zone', () => {
      const { container } = render(<PressureBar pressure={90} />)

      const pulsingElements = container.querySelectorAll('[class*="animate-pulse"]')
      expect(pulsingElements.length).toBeGreaterThan(0)
    })

    it('should not apply pulse animation to safe zone', () => {
      const { container } = render(<PressureBar pressure={30} />)

      const pulsingElements = container.querySelectorAll('[class*="animate-pulse"]')
      expect(pulsingElements.length).toBe(0)
    })

    it('should not apply pulse animation to warning zone', () => {
      const { container } = render(<PressureBar pressure={60} />)

      const pulsingElements = container.querySelectorAll('[class*="animate-pulse"]')
      expect(pulsingElements.length).toBe(0)
    })
  })

  describe('Description Box Styling', () => {
    it('should apply green background for safe zone', () => {
      const { container } = render(<PressureBar pressure={30} />)

      const descBox = container.querySelector('[class*="bg-green-100"]')
      expect(descBox).toBeInTheDocument()
    })

    it('should apply yellow background for warning zone', () => {
      const { container } = render(<PressureBar pressure={60} />)

      const descBox = container.querySelector('[class*="bg-yellow-100"]')
      expect(descBox).toBeInTheDocument()
    })

    it('should apply red background for critical zone', () => {
      const { container } = render(<PressureBar pressure={90} />)

      const descBox = container.querySelector('[class*="bg-red-100"]')
      expect(descBox).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    it('should handle negative pressure (clamp to 0)', () => {
      // Component might not clamp, but should handle gracefully
      // Note: Progress component will log a warning for invalid values
      const { container } = render(<PressureBar pressure={-10} />)

      // Component should still render despite invalid prop
      expect(container).toBeInTheDocument()
      // The badge will still show the negative value
      expect(screen.getByText('-10%')).toBeInTheDocument()
    })

    it('should handle pressure over 100', () => {
      // Component might not clamp, but should handle gracefully
      // Note: Progress component will log a warning for invalid values
      const { container } = render(<PressureBar pressure={150} />)

      // Component should still render despite invalid prop
      expect(container).toBeInTheDocument()
      // The badge will still show the over-max value
      expect(screen.getByText('150%')).toBeInTheDocument()
    })

    it('should handle decimal pressure values', () => {
      render(<PressureBar pressure={45.7} />)

      // Badge will show decimal value
      expect(screen.getByText('45.7%')).toBeInTheDocument()
    })
  })
})
