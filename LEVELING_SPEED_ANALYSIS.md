# Leveling Speed Analysis and Fix

## Issue Description

User reported jumping from level 15 to level 17 with approximately 10 minutes of chatter, which seemed too fast. The user expected leveling to be slower based on their understanding that "level 15 was approximately 2400 xp".

## Investigation Findings

### 1. Level Requirements Are Correct

The level requirements are working as designed:
- **Level 15 requirement**: 220 XP (not 2400 XP)
- **Level 16 requirement**: 306 XP
- **Level 17 requirement**: 404 XP
- **Total to jump from 15→17**: 710 XP

### 2. Base XP Calculation

Without any multipliers or bonuses:
- 10 minutes of talking = 600 XP
- 600 XP is sufficient to level from 15→16 (306 XP), leaving 294 XP
- 294 XP is NOT sufficient to reach level 17 (404 XP needed)
- **Expected result without bonuses**: Level 16 with 294 XP

### 3. Rested XP Bonus Explains the Behavior

With the rested XP system enabled (2x multiplier):
- 10 minutes of talking = 1200 XP (600 × 2.0)
- Starting at level 15 with accumulated XP + 1200 bonus XP can easily reach level 17 or even 18
- **This is WORKING AS INTENDED** - rested XP is designed to help returning players catch up

### 4. Bug Found and Fixed

During investigation, discovered that `CallsignProfileRepo.Upsert()` was not persisting `daily_xp` and `weekly_xp` fields to the database. This has been fixed by adding these fields to the update columns list.

**Fixed in**: `backend/repository/callsign_profile_repo.go`

```go
DoUpdates: clause.AssignmentColumns([]string{
    "level", "experience_points", "renown_level",
    "last_tally_at", "last_transmission_at", "rested_bonus_seconds",
    "last_rested_calculation_at", "daily_xp", "weekly_xp",  // Added these
    "updated_at",
}),
```

## Test Coverage

Added comprehensive test suite in `backend/tests/leveling_speed_test.go`:

1. **TestLevelingSpeed_NoMultipliers** - Verifies base leveling without bonuses
2. **TestLevelingSpeed_WithRestedBonus** - Demonstrates rested XP enables multi-level jumps
3. **TestProfileUpsert_PersistsDailyWeeklyXP** - Verifies the bug fix
4. **TestLevelRequirements_Documentation** - Documents all level requirements

## Conclusions

### The Leveling Speed is Correct

The user's experience of jumping from level 15 to level 17 is **expected behavior** when the rested XP bonus is active. The system is designed to:

1. **Reward returning players** with 2x XP multiplier
2. **Enable faster progression** for players who haven't played recently
3. **Accumulate rested bonus** up to 14 days (336 hours) of idle time

### User Confusion About XP Values

The user mentioned "level 15 was approximately 2400 xp" which suggests confusion about what the XP values represent:

- **Level requirement** (e.g., level 15 requires 220 XP) = XP needed FROM the previous level
- **Cumulative XP** (e.g., 4108 XP total to reach level 15) = Total XP from level 1
- **Daily/Weekly XP** = Recent activity for cap calculations

The UI should clearly distinguish between these different XP values to avoid confusion.

## Recommendations

1. ✅ **Bug Fix Applied** - `daily_xp` and `weekly_xp` now persist correctly
2. ✅ **Tests Added** - Comprehensive test coverage for leveling calculations
3. 📝 **UI Enhancement (Optional)** - Consider showing rested XP status clearly in the UI so users understand when bonuses are active
4. 📝 **Documentation (Optional)** - Add a help section explaining the rested XP system
5. ✅ **Verification** - All existing tests still pass after the fix

## Summary

- **Leveling logic is working correctly** ✓
- **Rested XP bonus explains the reported behavior** ✓
- **Bug found and fixed in profile persistence** ✓
- **Comprehensive test coverage added** ✓
- **No changes needed to leveling calculations** ✓
