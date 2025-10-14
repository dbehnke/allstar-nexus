# Rested XP Accumulation Bug Fix

## Issue
The rested XP accumulation system had two critical bugs:
1. **Exponential accumulation bug** - Recalculated from LastTransmissionAt on every tally cycle
2. **Idle users not processed** - Only users with recent transmissions were processed for rested XP

## Root Causes

### Bug 1: Exponential Accumulation
The `updateRestedBonus()` function calculated idle time from `LastTransmissionAt` on every tally cycle (every 30 minutes) and added it to the existing rested bonus **without tracking what had already been accumulated**.

Example of the bug:
- User idle for 12 hours
- Tally runs at 30 min: adds (0.5h × 1.5) = 0.75 hours
- Tally runs at 1 hour: adds (1.0h × 1.5) = 1.5 hours (doesn't subtract the 0.75!)
- Tally runs at 1.5 hours: adds (1.5h × 1.5) = 2.25 hours
- Result: Massively inflated rested time

### Bug 2: Idle Users Skipped
The tally service only processed callsigns that had transmissions in the current window. Users who were completely idle (no recent activity) were never processed, so their rested XP never accumulated at all.

## Fix

### 1. Added Tracking Field
Added `LastRestedCalculationAt` to `CallsignProfile` model to track when rested was last calculated.

### 2. Fixed Calculation Logic
Modified `updateRestedBonus()` to:
- Initialize `LastRestedCalculationAt` on first run
- Calculate idle time from `LastRestedCalculationAt` instead of `LastTransmissionAt`
- Update `LastRestedCalculationAt` after each calculation
- This ensures only **new** idle time is accumulated

### 3. Process All Profiles for Rested XP
Added logic to process **all** profiles after processing transmissions:
- Added `GetAllProfiles()` method to `CallsignProfileRepo`
- After processing transmission logs, iterate through all profiles
- Skip profiles already processed (have recent transmissions)
- Update rested bonus for idle profiles and save them
- Logs idle profiles processed for visibility

### 4. Adjusted Default Values
Changed defaults to be less generous:
- **Accumulation rate:** 1.5 → 0.006 (~1 hour per week maximum)
- **Max hours:** 336 (14 days) → 2 hours total cap
- **Rationale:** Prevents excessive rested accumulation for low-activity hubs

## Migration Notes

For existing databases, the `LastRestedCalculationAt` field will be zero. The code handles this by:
1. Detecting zero value: `if profile.LastRestedCalculationAt.IsZero()`
2. Initializing to `LastTransmissionAt` on first calculation
3. Future calculations work correctly from that point forward

Existing `RestedBonusSeconds` values will be preserved but won't grow as quickly with the new rate.

## Configuration

New recommended defaults:
```yaml
gamification:
  rested_bonus:
    enabled: true
    accumulation_rate: 0.006  # ~1 hour per week (1/168)
    idle_threshold_seconds: 300
    max_hours: 2
    multiplier: 2.0
```

Operators can adjust these values based on their hub's activity level:
- **High-activity hubs:** Keep rate low (0.006 or less)
- **Low-activity hubs:** Can increase rate (up to 0.05 for ~1 hour per day)
- **Max hours:** Should reflect realistic idle patterns (2-24 hours recommended)
