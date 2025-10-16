# Leveling Thresholds API

## Overview
This document describes the leveling thresholds API that provides authoritative XP requirements for each level. The API ensures a single source of truth for leveling calculations, preventing divergence between frontend and backend implementations.

## Why This API Exists
Previously, the frontend computed XP thresholds locally using a formula synchronized with the backend. While the formula was aligned, client-side computation risked future drift as the system evolved. The leveling thresholds API centralizes this logic on the server, ensuring consistency.

## API Endpoint

### GET /api/leveling/thresholds

Returns authoritative leveling XP thresholds derived from the backend's level configuration.

**Base URL**: `/api/leveling/thresholds`

**Method**: `GET`

**Authentication**: 
- Public endpoint if `ALLOW_ANON_DASHBOARD=true` (with rate limiting)
- Requires authentication otherwise

**Rate Limiting**: Subject to public stats rate limit (default: configurable via `PUBLIC_STATS_RPM`)

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_level` | integer | 60 | Maximum level to return (1-60) |
| `levels` | string | - | Comma-separated list of specific levels to return (e.g., "10,15,20") |

### Response Format

**Success Response (200 OK)**

```json
{
  "levels": [
    { "level": 1, "xp": 360 },
    { "level": 2, "xp": 360 },
    { "level": 3, "xp": 360 },
    ...
    { "level": 60, "xp": 9894 }
  ],
  "calculation": "level-based pow 1.8 scaled"
}
```

**Response Headers**
- `Cache-Control: public, max-age=300` - Response can be cached for 5 minutes
- `Content-Type: application/json`

**Fields**
- `levels`: Array of level threshold objects
  - `level`: Integer level number (1-60)
  - `xp`: Integer XP required to advance from this level to the next
- `calculation`: String describing the calculation method used

## Usage Examples

### Example 1: Get All Thresholds (Default)
```bash
curl http://localhost:3000/api/leveling/thresholds
```

Returns all 60 level thresholds.

### Example 2: Get First 15 Levels Only
```bash
curl http://localhost:3000/api/leveling/thresholds?max_level=15
```

Returns levels 1-15 only.

### Example 3: Get Specific Levels
```bash
curl http://localhost:3000/api/leveling/thresholds?levels=5,10,15,20
```

Returns only the specified levels (5, 10, 15, 20).

### Example 4: JavaScript/TypeScript Frontend Usage
```javascript
async function fetchLevelingThresholds() {
  try {
    const response = await fetch('/api/leveling/thresholds?max_level=60')
    if (!response.ok) {
      throw new Error('Failed to fetch thresholds')
    }
    const data = await response.json()
    
    // Convert to map for easy lookup
    const thresholdMap = {}
    data.levels.forEach(item => {
      thresholdMap[item.level] = item.xp
    })
    
    return thresholdMap
  } catch (error) {
    console.error('Error fetching thresholds:', error)
    // Fall back to local calculation
    return null
  }
}
```

## Frontend Integration

The `LevelingHelpModal.vue` component automatically fetches thresholds from this API when opened:

1. **Server Data (Preferred)**: When API call succeeds, server values are used and a green indicator shows "✓ Values provided by server"

2. **Fallback to Props**: If API fails but `levelConfig` prop is provided, those values are used

3. **Local Calculation**: If both server and prop data unavailable, component falls back to synchronized local formula

4. **Caching**: The component fetches data only once per lifecycle, not on every modal open

## Backend Implementation

The endpoint is implemented in `backend/api/gamification.go` as `LevelingThresholds()` method of `GamificationAPI`.

**Data Source**: 
- Reads from `level_config` database table (seeded at startup from `gamification.CalculateLevelRequirementsWithScale()`)
- Uses the same authoritative calculation that the gamification tally service uses
- Ensures frontend and backend always agree on XP requirements

**Performance**:
- Database query with in-memory caching (GORM)
- HTTP cache headers allow client/proxy caching for 5 minutes
- Minimal computational overhead (simple map lookup)

## Leveling Calculation Method

The default calculation uses a two-phase approach:

**Levels 1-10**: Linear scaling
- Fixed 360 XP per level (6 minutes of talk time each)

**Levels 11-60**: Logarithmic scaling
- Formula: `XP = (level-1)^1.8 × scale_factor`
- Total XP target: 259,200 seconds (72 hours)
- Ensures progression difficulty increases gradually

**Configurable Scaling**: Server can override defaults via `config.yaml` `gamification.level_scale` section.

## Testing

### Backend Tests
Located in `backend/tests/leveling_thresholds_test.go`:
- Default behavior (60 levels)
- `max_level` parameter handling
- `levels` parameter (specific levels)
- Cache header verification
- Method validation (GET only)

### Frontend Tests
Located in `frontend/tests/leveling.modal.spec.js`:
- API fetch on modal open
- Server data indicator display
- Fallback when API fails
- levelConfig prop fallback
- Single fetch per lifecycle

Run tests:
```bash
# Backend
go test ./backend/tests/leveling_thresholds_test.go -v

# Frontend
cd frontend && npm test
```

## Migration Notes

**Before**: Frontend computed XP using local formula in `xpFor()` function.

**After**: Frontend fetches from API first, falls back to local formula only if needed.

**Benefits**:
- Single source of truth
- No formula synchronization needed
- Server-side config changes immediately reflected in UI
- Graceful degradation if API unavailable

## Security Considerations

- Endpoint is read-only (GET only)
- Rate limited to prevent abuse
- No sensitive data exposed
- Can be made public or require authentication via config

## Future Enhancements

Potential improvements:
- Add cumulative XP totals per level
- Include time estimates at various XP rates
- Support for renown XP requirements
- Historical level curve changes
