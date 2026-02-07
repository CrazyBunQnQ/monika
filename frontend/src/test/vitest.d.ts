/**
 * Vitest Type Declarations
 *
 * Extends the Vitest global namespace with custom matchers.
 */

import { expect } from 'vitest'

// Extend Vitest's expect with jest-dom matchers
// Note: In modern vitest + @testing-library/jest-dom, this is typically automatic
// If you encounter issues with jest-dom matchers not being available,
// you may need to add additional setup here
