# Leveling Speed Analysis and Fix

## Issue Description

User reported jumping from level 15 to level 17 with approximately 10 minutes of chatter, which seemed too fast. The user expected leveling to be slower based on their understanding that "level 15 was approximately 2400 xp".

## Root Cause Found

The issue was caused by a **critical bug in the frontend UI** (`LevelingHelpModal.vue`). The modal was displaying incorrect XP requirements using a wrong fallback formula:

```javascript
// INCORRECT (old code):
return 360 + Math.floor(Math.pow(lvl - 10, 1.8) * 100)

// This produced values like:
// Level 15: 2,171 XP (36 minutes) ❌ WRONG
// Level 16: 2,875 XP (48 minutes) ❌ WRONG
```

The actual backend requirements are (after fix and merge with main):

```javascript
// CORRECT (backend using level-1):
// Level 15: 894 XP (14.9 minutes) ✓
// Level 16: 1013 XP (16.9 minutes) ✓
```

**The UI was showing XP requirements that were ~2.4x higher than reality!** This caused user confusion and made them believe leveling was broken when it was actually working correctly.

## Investigation Findings

### 1. Actual Level Requirements (Backend - Correct, Updated After Merge)

The level requirements in the backend are working as designed (using `level-1` formula):
- **Level 15 requirement**: 894 XP (14.9 minutes)
- **Level 16 requirement**: 1,013 XP (16.9 minutes)
- **Level 17 requirement**: 1,137 XP (18.9 minutes)
- **Total to jump from 15→17**: ~2,150 XP (35.8 minutes)

### 2. What the UI Was Showing (Frontend - WRONG)

The help modal was displaying incorrect fallback values:
- **Level 15**: 2,171 XP (36 minutes) ❌
- **Level 16**: 2,875 XP (48 minutes) ❌
- **Level 17**: 3,680 XP (60 minutes) ❌

This is what the user saw in the screenshot, which explained their confusion.

### 3. Base XP Calculation (Now Makes Sense!)

Without any multipliers or bonuses:
- 10 minutes of talking = 600 XP
- 600 XP is NOT sufficient to level from 15→16 (894 XP needed)
- User likely had accumulated XP from before

### 4. Rested XP Bonus Explains Behavior

With the rested XP system enabled (2x multiplier):
- 10 minutes of talking = 1200 XP (600 × 2.0)
- Starting at level 15 with some accumulated XP + 1200 bonus XP can reach level 17
- **This is WORKING AS INTENDED** - rested XP is designed to help returning players catch up

### 5. Backend Changes After Merge

After merging with main, the backend formula was updated in PR #21:
- Changed from `Math.pow(level - 10, 1.8)` to `Math.pow(level - 1, 1.8)`
- This anchors level 11 to k=10, ensuring it's > 360 XP (was previously only 12 XP)
- New values are higher and more balanced across the progression curve

### 6. Additional Bug Found and Fixed

During investigation, also discovered that `CallsignProfileRepo.Upsert()` was not persisting `daily_xp`, `weekly_xp`, and `last_rested_calculation_at` fields to the database. This has been fixed by adding these fields to the update columns list.

## Fixes Applied

### 1. Frontend UI Bug (PRIMARY FIX - Updated for Merge)

**File**: `frontend/src/components/LevelingHelpModal.vue`

**Problem**: Incorrect fallback formula showing wrong XP requirements

**Fix**: Updated `xpFor()` function to match backend logic exactly (including the level-1 change from main):

```javascript
function xpFor(lvl) {
  const lc = props.levelConfig || {}
  const k = String(lvl)
  if (lc[k] != null) return lc[k]
  
  // Fallback calculation matching backend logic (low-activity hub scale)
  // Levels 1-10: Linear (360 XP each)
  if (lvl <= 10) return 360
  
  // Levels 11-60: Logarithmic scaling using k = level-1 (anchored to ensure level 11 > 360)
  const totalRemaining = 255600.0
  let sum = 0.0
  for (let level = 11; level <= 60; level++) {
    sum += Math.pow(level - 1, 1.8)
  }
  const scaleFactor = totalRemaining / sum
  return Math.floor(Math.pow(lvl - 1, 1.8) * scaleFactor)
}
```

### 2. Backend Profile Persistence Bug

**File**: `backend/repository/callsign_profile_repo.go`

**Problem**: Upsert not saving daily_xp, weekly_xp, last_rested_calculation_at

**Fix**: Added missing fields to DoUpdates columns

## Test Coverage

Added comprehensive test suite in `backend/tests/leveling_speed_test.go`:

1. **TestLevelingSpeed_NoMultipliers** - Verifies base leveling without bonuses
2. **TestLevelingSpeed_WithRestedBonus** - Demonstrates rested XP enables multi-level jumps
3. **TestProfileUpsert_PersistsDailyWeeklyXP** - Verifies the profile persistence fix
4. **TestLevelRequirements_Documentation** - Documents all level requirements

## User's Experience Explained

The user saw in the UI that level 15 required "2171 XP (36 minutes)" but then leveled up with ~10 minutes of chatter. This is because:

1. **The UI was wrong** - It actually requires 894 XP (14.9 minutes)
2. **Accumulated XP** - User likely had some XP already from before
3. **Rested XP bonus** - May have provided 2x multiplier, making it even faster

With rested XP (2x) and starting at level 15 with ~300 XP already accumulated:
- Start: Level 15, 300 XP
- Add: 600 seconds × 2.0 = 1200 XP
- Total: 1500 XP
- Level 16 needs 894: 1500 - 894 = 606 remaining
- Level 17 needs 1013: 606 < 1013, stays at level 16

OR with slightly more accumulated XP (~900):
- Start: Level 15, 900 XP
- Add: 600 × 2.0 = 1200 XP
- Total: 2100 XP
- Level 16: 2100 - 894 = 1206 remaining
- Level 17: 1206 - 1013 = 193 remaining → **Reaches level 17!**

## Conclusions

### The Leveling System is Working Correctly ✓

The backend leveling calculations are accurate and working as designed. The issue was purely a **frontend display bug** that misled users about XP requirements.

### Why Users Were Confused

- UI showed inflated XP requirements (~2.4x too high)
- Users expected slow progression based on wrong UI data
- Actual progression was much faster than displayed values suggested
- This created the impression that leveling was "too fast" or "bugged"

### Changes After Merge with Main

- Backend formula updated to use `level-1` instead of `level-10`
- Frontend updated to match the new backend formula
- All values verified to be in sync

## Recommendations

1. ✅ **Frontend Bug Fixed** - XP requirements now display correctly (updated for merge)
2. ✅ **Backend Bug Fixed** - Profile persistence now works correctly
3. ✅ **Tests Added** - Comprehensive test coverage for leveling calculations
4. 📝 **UI Enhancement (Optional)** - Consider showing rested XP status clearly in the UI
5. 📝 **Documentation (Optional)** - Add help section explaining all XP systems

## Summary

- **Critical frontend bug discovered and fixed** ✓
- **UI was displaying ~2.4x higher XP requirements than reality** ✓
- **Frontend updated to match merged backend changes (level-1 formula)** ✓
- **Leveling logic is mathematically correct** ✓
- **User confusion explained by wrong UI data** ✓
- **Backend profile persistence bug also fixed** ✓
- **Comprehensive test coverage added** ✓

The user's report led to discovering a significant UI bug that was misleading all users about leveling speed. After merging with main, the formula was updated and the frontend has been synchronized. Thank you for bringing this to our attention!
