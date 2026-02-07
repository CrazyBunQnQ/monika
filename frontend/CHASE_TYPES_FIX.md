# Chase Types Fix Summary

## Date
2026-02-07

## Problem
The frontend `chase.ts` type definitions did not match the backend API models and schemas, causing type mismatches and potential runtime errors.

## Changes Made

### 1. Fixed Chase Interface
**Before:**
```typescript
export interface Chase {
  current_round: number
  distance_level: number
  pressure: number
  environment_type: string
  // ...
}
```

**After:**
```typescript
export interface Chase {
  round: number                    // ✅ was current_round
  location: string                 // ✅ was distance_level
  setting: string                  // ✅ was environment_type
  // removed: pressure (not in backend)
  end_reason: ChaseEndReason | null
  failed_forward_scene: string | null
  chase_metadata: Record<string, unknown>
  // ...
}
```

### 2. Fixed ChaseParticipant Interface
**Added Missing Fields:**
- `is_player: boolean`
- `is_exhausted: boolean`
- `failed_obstacle_count: number`
- `speed_penalty: number`
- `consecutive_failures: number`
- `participant_metadata: Record<string, unknown>`
- `created_at: string`
- `updated_at: string`

**Fixed Field Types:**
- `character_id: number | null` (was `string | null`)

### 3. Fixed ChaseObstacle Interface
**Renamed Fields:**
- `penalty` → `failure_penalty`
- `damage` → `failure_damage`
- `type` → `obstacle_type`
- `required_skill` → `skill_required`

**Added Missing Fields:**
- `name: string`
- `appears_at_round: number`
- `appears_at_distance: number`
- `failure_san_cost: number | null`
- `fail_forward_result: string | null`
- `details: Record<string, unknown>`

### 4. Added New Type Definitions

**Request Types:**
- `ChaseCreateRequest`
- `ChaseParticipantCreateRequest`
- `ChaseRoundRequest`
- `ChaseActionRequestItem`
- `ChaseEndRequest`

**Response Types:**
- `ChaseResponse`
- `ChaseRoundResponse`
- `ObstacleResponse`

**Additional Types:**
- `ChaseAction` - Complete action record from backend
- `ChaseEndReason` - Enum for chase end reasons
- `ObstacleDifficulty` - Regular/hard/extreme levels

### 5. Added JSDoc Comments
All interfaces now have comprehensive JSDoc comments following the style from `combat.ts`:
```typescript
/**
 * Chase session with all participants and obstacles
 */
export interface Chase { ... }
```

### 6. Backward Compatibility
Marked legacy types as `@deprecated` to allow gradual migration:
- `RoundResult` → Use `ChaseRoundResponse`
- `ActionResult` → Use `ChaseAction`
- `CheckResult` → Use `SuccessLevel` and related types
- `ChaseActionRequest` → Use `ChaseActionRequestItem`
- `ObstacleCheckRequest` → Use `ChaseActionRequestItem`

## Backend References

### Models (backend/src/models/chase.py)
- `Chase` - Main chase session model
- `ChaseParticipant` - Participant model
- `ChaseObstacle` - Obstacle model
- `ChaseAction` - Action record model

### API Schemas (backend/src/api/chase.py)
- `ChaseCreateRequest` - POST /chase/start
- `ChaseParticipantCreateRequest` - POST /chase/{id}/participants
- `ChaseRoundRequest` - POST /chase/{id}/round
- `ChaseActionRequest` - Single action in round
- `ChaseEndRequest` - POST /chase/{id}/end
- `ChaseResponse` - GET /chase/{id}
- `ChaseRoundResponse` - Round resolution response
- `ObstacleResponse` - Obstacle generation response

## Testing Recommendations

1. **Type Safety:** Verify all chase-related components use correct types
2. **API Integration:** Test API calls with new request/response types
3. **Backward Compatibility:** Ensure legacy code still functions
4. **Migration:** Gradually migrate from deprecated types to new types

## Files Modified
- `frontend/src/types/chase.ts` - Complete rewrite with proper types

## Next Steps
1. Update chase service layer to use new request/response types
2. Update React components to use corrected interfaces
3. Run full TypeScript type check
4. Test API integration with corrected types
