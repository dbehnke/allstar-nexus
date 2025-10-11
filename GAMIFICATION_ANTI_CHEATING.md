# Anti-Cheating & Balance Mechanics for Low-Activity Hub

## Overview
This document extends the base GAMIFICATION.md with comprehensive anti-cheating mechanics scaled for a **low-activity hub** with a 2-hour weekly XP cap.

---

## Key Scaling Parameters

| Parameter | Standard Hub | Low-Activity Hub (2hr/week) | Ratio |
|-----------|--------------|----------------------------|-------|
| Weekly XP Cap | 144,000 sec (40 hours) | 7,200 sec (2 hours) | 20x |
| Daily XP Cap | 28,800 sec (8 hours) | 1,200 sec (20 minutes) | 24x |
| Level 1-10 XP | 3,600 (1 hour each) | 360 (6 min each) | 10x |
| Total to Level 60 | 2,592,000 (720 hours) | 259,200 (72 hours) | 10x |
| Weeks to Level 60 | ~18 weeks (40hr/wk) | ~36 weeks (2hr/wk) | 2x |

---

## 1. Rested XP Bonus System

### Purpose
Reward players who take breaks and return to the network. Helps casual participants stay competitive.

### Mechanics (Low-Activity Hub)

**Accumulation:**
- **Rate:** 1.5 hours of rested bonus per 1 hour offline (50% faster than standard)
- **Maximum:** 336 hours (14 days worth) - accommodates infrequent users
- **Starts Accumulating:** After 24 hours of no transmissions

**Usage:**
- When player returns, next N hours of talking get **2.0x XP** multiplier (double XP!)
- Rested bonus depletes as player talks: 1 hour talking = 1 hour bonus consumed
- If bonus runs out mid-transmission, applies proportionally

**Example:**
```
Player offline for 7 days:
→ Accumulates: 7 days × 24 hours × 1.5 = 252 hours of rested bonus

Returns and talks for 2 hours (weekly cap):
→ Base XP: 7,200 seconds
→ With 2.0x rested multiplier: 14,400 XP earned
→ Remaining rested bonus: 250 hours

Result: Player earned double XP for returning!
```

### Database Schema

```go
type CallsignProfile struct {
    // ... existing fields
    LastTransmissionAt time.Time `gorm:"index" json:"last_transmission_at"`
    RestedBonusSeconds int       `gorm:"default:0" json:"rested_bonus_seconds"`
}
```

### Configuration

```yaml
gamification:
  rested_bonus:
    enabled: true
    accumulation_rate: 1.5    # 1 hour offline = 1.5 hours bonus
    max_hours: 336            # 14 days max (2 weeks)
    multiplier: 2.0           # Double XP when rested!
```

---

## 2. Diminishing Returns System

### Purpose
Prevent grinding entire weekly allowance in one day. Encourages healthy distribution of activity throughout the week.

### Mechanics (Low-Activity Hub)

**Rolling 24-Hour Window:**
- Tracks XP earned in last 24 hours (not calendar day)
- Applies multipliers based on current daily talk time
- Window slides continuously (oldest hour drops off)

**Tier Thresholds:**

| Daily Talk Time | XP Multiplier | Notes |
|-----------------|---------------|-------|
| 0-20 minutes (0-1,200 sec) | 100% (1.0x) | Normal, healthy daily participation |
| 20-40 minutes (1,200-2,400 sec) | 75% (0.75x) | Mild reduction, approaching daily target |
| 40-60 minutes (2,400-3,600 sec) | 50% (0.5x) | Significant reduction, over daily target |
| 60+ minutes (3,600+ sec) | 25% (0.25x) | Severe penalty, trying to grind entire week in one day |

**Rationale:**
- Weekly cap = 2 hours (7,200 seconds)
- Ideal distribution = ~17 minutes per day (1,020 seconds)
- 20-minute threshold = reasonable daily activity
- Beyond 60 minutes in a day = clear grinding attempt

**Example:**
```
Monday: Player talks for 90 minutes straight
→ First 20 minutes: 1,200 XP at 100% = 1,200 XP
→ Next 20 minutes: 1,200 XP at 75% = 900 XP
→ Next 20 minutes: 1,200 XP at 50% = 600 XP
→ Final 30 minutes: 1,800 XP at 25% = 450 XP
→ Total: 3,150 XP (vs. 5,400 at full rate)

Tuesday-Sunday: Player has 4,050 XP remaining in weekly cap
```

### Database Schema

```go
type XPActivityLog struct {
    ID               uint      `gorm:"primaryKey" json:"id"`
    Callsign         string    `gorm:"index;size:20;not null" json:"callsign"`
    HourBucket       time.Time `gorm:"index;not null" json:"hour_bucket"`     // Truncated to hour
    RawXP            int       `gorm:"not null" json:"raw_xp"`                // Before multipliers
    AwardedXP        int       `gorm:"not null" json:"awarded_xp"`            // After multipliers
    RestedMultiplier float64   `gorm:"default:1.0" json:"rested_multiplier"`
    DRMultiplier     float64   `gorm:"default:1.0" json:"dr_multiplier"`      // Diminishing returns
    KerchunkPenalty  float64   `gorm:"default:1.0" json:"kerchunk_penalty"`
    CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}
```

### Configuration

```yaml
gamification:
  diminishing_returns:
    enabled: true
    tiers:
      - max_seconds: 1200     # 0-20 minutes: 100% XP
        multiplier: 1.0
      - max_seconds: 2400     # 20-40 minutes: 75% XP
        multiplier: 0.75
      - max_seconds: 3600     # 40-60 minutes: 50% XP
        multiplier: 0.5
      - max_seconds: 999999   # 60+ minutes: 25% XP
        multiplier: 0.25
```

---

## 3. Kerchunking Detection & Penalties

### Purpose
Prevent spam abuse where players rapidly key up for 1-2 seconds repeatedly to farm XP.

### Mechanics (Universal - Not Scaled)

**Detection:**
- Transmission < 3 seconds = flagged as "kerchunk"
- Track consecutive kerchunks within 30-second windows
- Apply escalating penalties for repeated offenses

**Penalty Tiers:**

| Pattern | Penalty | XP Multiplier | Example |
|---------|---------|---------------|---------|
| Single < 3s TX | Light | 0.5 (50% XP) | Could be legitimate quick reply |
| 2-3 consecutive | Moderate | 0.25 (25% XP) | Suspicious pattern |
| 4-5 consecutive | Heavy | 0.1 (10% XP) | Clear abuse |
| 6+ consecutive | Severe | 0.0 (0% XP) | Spam, no reward |

**Normal transmissions (≥3 seconds):** Always 100% XP, no penalty

**Reset:** Penalty counter resets after any transmission ≥3 seconds

**Example:**
```
Player transmissions:
1. 2 seconds → 50% XP (single kerchunk)
2. 2 seconds → 25% XP (2 consecutive)
3. 10 seconds → 100% XP (normal TX, resets counter)
4. 2 seconds → 50% XP (single kerchunk, counter reset)
```

### Implementation

```go
func CalculateKerchunkPenalty(logs []TransmissionLog, currentTX TransmissionLog) float64 {
    if currentTX.DurationSeconds >= 3 {
        return 1.0  // Normal transmission, no penalty
    }

    // Count consecutive kerchunks in last 30 seconds
    consecutiveCount := 0
    cutoff := currentTX.TimestampStart.Add(-30 * time.Second)

    for i := len(logs) - 1; i >= 0; i-- {
        log := logs[i]
        if log.TimestampStart.Before(cutoff) {
            break  // Outside 30-second window
        }
        if log.DurationSeconds < 3 {
            consecutiveCount++
        } else {
            break  // Normal TX breaks the chain
        }
    }

    // Apply penalty based on count
    switch consecutiveCount {
    case 0:
        return 0.5   // First kerchunk
    case 1, 2:
        return 0.25  // 2-3 consecutive
    case 3, 4:
        return 0.1   // 4-5 consecutive
    default:
        return 0.0   // 6+ = no XP
    }
}
```

### Configuration

```yaml
gamification:
  kerchunk_detection:
    enabled: true
    threshold_seconds: 3        # TX < 3s = kerchunk
    consecutive_window: 30      # Check last 30 seconds
    penalties:
      single: 0.5
      two_to_three: 0.25
      four_to_five: 0.1
      six_plus: 0.0
```

---

## 4. Weekly/Daily XP Caps (Hard Limits)

### Purpose
Absolute upper limit to ensure no player can gain unfair advantage through extreme grinding.

### Mechanics (Low-Activity Hub)

**Weekly Cap:** 7,200 seconds (2 hours) - PRIMARY CAP
- Hard limit per week (Sunday 00:00 UTC → Saturday 23:59 UTC)
- Once reached, all further transmissions earn 0 XP until reset
- Resets every Sunday at midnight UTC

**Daily Cap:** 1,200 seconds (20 minutes) - DISTRIBUTION CAP
- Helps distribute weekly allowance across 7 days
- Prevents single-day grinding of entire weekly allowance
- Resets every day at midnight UTC

**Cap Tracking:**
- Query `XPActivityLog` table for awarded XP in time window
- Check before awarding XP during tally
- If at cap, skip XP award but still log transmission

**Example:**
```
Player tries to talk 90 minutes on Monday:
→ First 20 minutes: Awarded XP (daily cap reached)
→ Next 70 minutes: 0 XP (daily cap exceeded)
→ Monday total: 1,200 XP earned

Tuesday-Sunday: Player can earn 1,200 XP each day
→ Maximum possible: 7 days × 1,200 = 8,400 XP
→ But weekly cap is 7,200, so actually capped at 6 days worth

Result: Player forced to spread activity across week
```

### Database Queries

```go
// Check weekly cap
func (r *ActivityRepo) GetWeeklyXP(callsign string) (int, error) {
    startOfWeek := getStartOfWeek() // Last Sunday 00:00 UTC
    var totalXP int64
    err := r.db.Model(&models.XPActivityLog{}).
        Where("callsign = ? AND hour_bucket >= ?", callsign, startOfWeek).
        Select("COALESCE(SUM(awarded_xp), 0)").
        Scan(&totalXP).Error
    return int(totalXP), err
}

// Check daily cap
func (r *ActivityRepo) GetDailyXP(callsign string) (int, error) {
    startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
    var totalXP int64
    err := r.db.Model(&models.XPActivityLog{}).
        Where("callsign = ? AND hour_bucket >= ?", callsign, startOfDay).
        Select("COALESCE(SUM(awarded_xp), 0)").
        Scan(&totalXP).Error
    return int(totalXP), err
}
```

### Configuration

```yaml
gamification:
  xp_caps:
    enabled: true               # ENABLED for low-activity hub
    daily_cap_seconds: 1200     # 20 minutes max per day
    weekly_cap_seconds: 7200    # 2 HOURS MAX PER WEEK
    reset_hour: 0               # Midnight UTC
    week_starts: "sunday"       # Week starts Sunday
```

---

## 5. Enhanced Tally Service Logic

### ProcessTally() with All Anti-Cheating

```go
func (s *TallyService) ProcessTally() error {
    ctx := context.Background()

    // 1. Get recent transmissions grouped by callsign
    transmissions, err := s.txLogRepo.GetRecentTransmissions(ctx, s.lastTallyTime)
    if err != nil {
        return err
    }

    for callsign, txLogs := range transmissions {
        // 2. Load or create profile
        profile, err := s.profileRepo.GetByCallsign(ctx, callsign)
        if err != nil {
            continue  // Skip on error
        }

        // 3. Update rested bonus accumulation
        hoursOffline := time.Since(profile.LastTransmissionAt).Hours()
        if hoursOffline >= 24 {
            // Accumulate rested bonus
            bonusHours := hoursOffline * s.config.RestedAccumulationRate
            profile.RestedBonusSeconds += int(bonusHours * 3600)

            // Cap at maximum
            if profile.RestedBonusSeconds > s.config.RestedMaxSeconds {
                profile.RestedBonusSeconds = s.config.RestedMaxSeconds
            }
        }

        // 4. Get current daily/weekly XP for caps
        weeklyXP, _ := s.activityRepo.GetWeeklyXP(ctx, callsign)
        dailyXP, _ := s.activityRepo.GetDailyXP(ctx, callsign)

        // 5. Get 24-hour activity for diminishing returns
        recentActivity, _ := s.activityRepo.GetLast24Hours(ctx, callsign)
        currentDailySeconds := sumSeconds(recentActivity)

        // 6. Process each transmission
        for i, tx := range txLogs {
            rawXP := tx.DurationSeconds

            // --- Apply all multipliers and caps ---

            // 6a. Check weekly cap first (hard limit)
            if weeklyXP >= s.config.WeeklyCapSeconds {
                // At cap, award 0 XP but still log
                s.activityRepo.LogActivity(ctx, callsign, rawXP, 0, 0, 0, 0)
                continue
            }

            // 6b. Check daily cap
            if dailyXP >= s.config.DailyCapSeconds {
                s.activityRepo.LogActivity(ctx, callsign, rawXP, 0, 0, 0, 0)
                continue
            }

            // 6c. Apply rested bonus (if available)
            restedMultiplier := 1.0
            if profile.RestedBonusSeconds > 0 {
                restedMultiplier = s.config.RestedMultiplier
                profile.RestedBonusSeconds -= tx.DurationSeconds

                // Partial bonus if running out
                if profile.RestedBonusSeconds < 0 {
                    // Calculate partial multiplier
                    secondsWithBonus := tx.DurationSeconds + profile.RestedBonusSeconds
                    secondsWithoutBonus := -profile.RestedBonusSeconds
                    restedMultiplier = (float64(secondsWithBonus) * s.config.RestedMultiplier + float64(secondsWithoutBonus)) / float64(tx.DurationSeconds)
                    profile.RestedBonusSeconds = 0
                }
            }

            // 6d. Apply diminishing returns
            drMultiplier := s.calculateDRMultiplier(currentDailySeconds)
            currentDailySeconds += tx.DurationSeconds

            // 6e. Apply kerchunk penalty
            kerchunkPenalty := s.calculateKerchunkPenalty(txLogs[:i], tx)

            // 6f. Calculate final XP
            finalXP := float64(rawXP) * restedMultiplier * drMultiplier * kerchunkPenalty
            awardedXP := int(finalXP)

            // 6g. Apply daily cap to awarded amount
            remainingDaily := s.config.DailyCapSeconds - dailyXP
            if awardedXP > remainingDaily {
                awardedXP = remainingDaily
            }

            // 6h. Apply weekly cap to awarded amount
            remainingWeekly := s.config.WeeklyCapSeconds - weeklyXP
            if awardedXP > remainingWeekly {
                awardedXP = remainingWeekly
            }

            // 6i. Log activity (transparency)
            s.activityRepo.LogActivity(ctx, callsign, rawXP, awardedXP, restedMultiplier, drMultiplier, kerchunkPenalty)

            // 6j. Award XP to profile
            profile.ExperiencePoints += awardedXP

            // Update running totals
            weeklyXP += awardedXP
            dailyXP += awardedXP
        }

        // 7. Update last transmission timestamp
        profile.LastTransmissionAt = time.Now()

        // 8. Process level-ups (existing logic)
        leveledUp := s.processLevelUps(ctx, profile)
        if leveledUp {
            // Emit level-up event (future: notifications)
            s.emitLevelUpEvent(profile)
        }

        // 9. Save updated profile
        s.profileRepo.Upsert(ctx, profile)
    }

    s.lastTallyTime = time.Now()
    return nil
}

func (s *TallyService) calculateDRMultiplier(secondsInLast24Hours int) float64 {
    for _, tier := range s.config.DRTiers {
        if secondsInLast24Hours <= tier.MaxSeconds {
            return tier.Multiplier
        }
    }
    // Fallback to last tier
    return s.config.DRTiers[len(s.config.DRTiers)-1].Multiplier
}
```

---

## 6. Frontend Transparency Features

### Profile Display with Anti-Cheating Info

```vue
<template>
  <div class="profile-card">
    <h2>{{ callsign }} - Level {{ level }}</h2>
    <div class="xp-progress">
      <LevelProgressBar :current-xp="xp" :required-xp="nextLevelXP" />
    </div>

    <!-- Rested Bonus Status -->
    <div v-if="restedBonusHours > 0" class="rested-bonus">
      🎁 <strong>Rested Bonus:</strong> {{ restedBonusHours }}h available
      <span class="bonus-multiplier">(2.0x XP!)</span>
    </div>

    <!-- Weekly Progress -->
    <div class="weekly-activity">
      📅 <strong>This Week:</strong> {{ weeklyMinutes }} / 120 minutes
      <div class="progress-bar">
        <div class="fill" :style="{ width: weeklyPercent + '%' }"></div>
      </div>
    </div>

    <!-- Current XP Rate -->
    <div class="xp-rate" :class="rateClass">
      <span v-if="currentMultiplier >= 1.0">
        ✅ Normal XP Rate (100%)
      </span>
      <span v-else-if="currentMultiplier >= 0.75">
        ⚠️ Reduced XP Rate ({{ Math.round(currentMultiplier * 100) }}%)
      </span>
      <span v-else>
        🛑 Heavily Reduced ({{ Math.round(currentMultiplier * 100) }}%)
      </span>
    </div>

    <!-- Daily Activity Warning -->
    <div v-if="dailyMinutes >= 15" class="warning">
      ⏰ Daily cap approaching ({{ dailyMinutes }}/20 minutes today)
    </div>
  </div>
</template>
```

### Activity Breakdown API

**Endpoint:** `GET /api/gamification/activity/:callsign?days=7`

**Response:**
```json
{
  "callsign": "K8FBI",
  "total_xp": 45000,
  "level": 25,
  "rested_bonus_hours": 12,
  "daily_breakdown": [
    {
      "date": "2025-01-15",
      "raw_xp": 1500,
      "awarded_xp": 1200,
      "multipliers": {
        "rested": 1.0,
        "diminishing_returns": 0.8,
        "kerchunk_penalty": 1.0
      },
      "talk_time_minutes": 25
    }
  ],
  "weekly_total": {
    "raw_xp": 8400,
    "awarded_xp": 6800,
    "cap_remaining": 400
  }
}
```

---

## 7. Complete Configuration Example

```yaml
gamification:
  enabled: true
  tally_interval_minutes: 30

  # Level scaling (low-activity hub)
  level_scale:
    - levels: "1-10"
      xp_per_level: 360         # 6 minutes per level
    - levels: "11-60"
      scaling: "logarithmic"
      target_total_seconds: 259200  # 72 hours total

  # Rested Bonus (generous for low activity)
  rested_bonus:
    enabled: true
    accumulation_rate: 1.5      # 1hr offline = 1.5hr bonus
    max_hours: 336              # 14 days max
    multiplier: 2.0             # Double XP!

  # Diminishing Returns (scaled down)
  diminishing_returns:
    enabled: true
    tiers:
      - max_seconds: 1200       # 0-20 min: 100% XP
        multiplier: 1.0
      - max_seconds: 2400       # 20-40 min: 75% XP
        multiplier: 0.75
      - max_seconds: 3600       # 40-60 min: 50% XP
        multiplier: 0.5
      - max_seconds: 999999     # 60+ min: 25% XP
        multiplier: 0.25

  # Kerchunk Detection (universal)
  kerchunk_detection:
    enabled: true
    threshold_seconds: 3
    consecutive_window: 30
    penalties:
      single: 0.5
      two_to_three: 0.25
      four_to_five: 0.1
      six_plus: 0.0

  # XP Caps (enabled for low-activity hub)
  xp_caps:
    enabled: true
    daily_cap_seconds: 1200     # 20 minutes/day
    weekly_cap_seconds: 7200    # 2 hours/week (PRIMARY)
    reset_hour: 0               # Midnight UTC
    week_starts: "sunday"
```

---

## Benefits Summary

1. **Fair Competition:** Everyone has same 2hr/week limit
2. **Anti-Grinding:** Daily caps + diminishing returns prevent weekend warriors
3. **Rewards Breaks:** Rested bonus (2.0x!) helps casual players catch up
4. **Prevents Abuse:** Kerchunk detection stops spam exploitation
5. **Transparency:** Activity logs show exact XP calculations
6. **Long-Term Engagement:** 36-week journey to max level is sustainable
7. **Immediate Gratification:** First 10 levels (6 min each) hook new users
8. **Flexible:** All systems can be individually enabled/disabled

---

## Example Player Scenarios

### Scenario 1: Consistent Weekly Player
```
Player talks 30 minutes every Monday and Friday:
- Monday: 30 min at 100% = 1,800 XP
- Friday: 30 min at 100% = 1,800 XP
- Weekly total: 3,600 XP
- Progress: Completes Level 1-10 in first week!
```

### Scenario 2: Weekend Warrior (Grinding Attempt)
```
Player tries to talk 2 hours on Saturday:
- First 20 min: 1,200 XP at 100% = 1,200 XP (daily cap hit)
- Next 100 min: 0 XP (daily cap exceeded)
- Total: Only 1,200 XP earned despite 2 hours
- Lesson: Forced to spread activity across week
```

### Scenario 3: Casual with Long Break (Rested Bonus)
```
Player talks 1 hour Week 1, then takes 2-week break:
- Week 1: 3,600 XP
- Weeks 2-3: Offline, accumulates 21 hours rested bonus
- Week 4: Returns, talks 2 hours with 2.0x multiplier
- Earns: 7,200 base × 2.0 = 14,400 XP!
- Total: 18,000 XP vs. 10,800 without bonus
- Outcome: Caught up despite missing 2 weeks
```

### Scenario 4: Kerchunker (Spam Attempt)
```
Player rapidly keys up 10 times for 2 seconds each:
- TX 1: 2 sec × 0.5 = 1 XP
- TX 2: 2 sec × 0.25 = 0.5 XP
- TX 3: 2 sec × 0.25 = 0.5 XP
- TX 4-10: 2 sec × 0.0 = 0 XP
- Total: 2 XP from 20 seconds (vs. 20 XP normally)
- Lesson: Spam doesn't pay
```

---

## Implementation Status

### ✅ BACKEND COMPLETE (All phases implemented!)

1. ✅ **XP Caps** - Daily (20 min) and weekly (2 hrs) limits fully implemented
2. ✅ **Rested Bonus** - 1.5x accumulation, 336hr max, 2.0x multiplier active
3. ✅ **Diminishing Returns** - 4-tier system with rolling 24-hour window
4. ✅ **Kerchunk Detection** - Spam detection with escalating penalties
5. ✅ **Transparency System** - XPActivityLog tracks all multipliers

### 🚧 Frontend Transparency Dashboard (TODO - Phase 4)
- Profile UI showing rested bonus status
- Profile UI showing weekly/daily cap progress
- Profile UI showing current XP rate multiplier
- Activity breakdown visualization

---

## Testing Checklist

### Backend Logic (Ready for Testing)
- [ ] Weekly cap enforces 2-hour limit
- [ ] Daily cap enforces 20-minute limit
- [ ] Rested bonus accumulates correctly (1.5x rate)
- [ ] Rested bonus applies 2.0x multiplier
- [ ] Rested bonus caps at 336 hours (14 days)
- [ ] Diminishing returns triggers at correct thresholds
- [ ] Diminishing returns uses rolling 24-hour window
- [ ] Kerchunk detection identifies consecutive < 3s transmissions
- [ ] Kerchunk penalties escalate correctly
- [ ] Activity logs record all multipliers
- [ ] Multiple simultaneous multipliers stack correctly
- [ ] Caps prevent XP award but still log transmission
- [ ] API endpoints return correct data

### Frontend UI (TODO - Not Yet Implemented)
- [ ] Profile UI shows rested bonus status
- [ ] Profile UI shows weekly cap progress
- [ ] Profile UI warns when approaching daily cap
- [ ] Scoreboard displays correctly
- [ ] Transmission history paginates properly
