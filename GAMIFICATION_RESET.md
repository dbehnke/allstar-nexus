# Gamification Reset Procedure

## When to Use This

Use this procedure if you need to recalculate all XP and levels from scratch due to:
- Fixing a bug in the tally service
- Changing XP calculation rules
- Data corruption or inconsistencies

## Full Reset (Accurate Reprocessing)

This will delete all gamification state and reprocess all transmission logs to recalculate accurate XP totals, levels, and rested bonuses.

### Steps

1. **Stop the server** (if running)

2. **Backup your database** (recommended)
   ```bash
   cp data/allstar.db data/allstar.db.backup-$(date +%Y%m%d-%H%M%S)
   ```

3. **Clear gamification tables**
   ```bash
   sqlite3 data/allstar.db << 'EOF'
   -- Clear all callsign profiles (XP, levels, rested bonuses)
   DELETE FROM callsign_profiles;
   
   -- Clear XP activity logs
   DELETE FROM xp_activity_logs;
   
   -- Clear tally state (checkpoint for what's been processed)
   DELETE FROM tally_state;
   EOF
   ```

4. **Restart the server**
   ```bash
   ./allstar-nexus
   ```

### What Happens

On startup, the tally service will:
1. Seed `last_tally_at` from the oldest transmission log timestamp
2. Process all transmission logs in 30-minute windows (or your configured `tally_interval_minutes`)
3. Create fresh `callsign_profiles` with accurate XP and levels
4. Generate new `xp_activity_logs` entries
5. Persist the `tally_state` after each window

### Monitoring Progress

Watch the logs for tally processing:
```bash
tail -f logs/allstar-nexus.log | grep -E "Tally window|tally.window.results|Processing XP tally"
```

You should see:
- `"Processing XP tally (windowed)"` - tally cycle starting
- `"Tally window" from=<time> to=<time>` - processing each window
- `"tally.window.results" callsigns=X tx_count=Y` - transmissions processed per window

### Verification

After the tally catches up to current time, verify profiles:

```bash
# Check top profiles by XP
sqlite3 data/allstar.db "SELECT callsign, level, experience_points, renown_level, rested_bonus_seconds, last_tally_at FROM callsign_profiles ORDER BY experience_points DESC LIMIT 10;"

# Check activity log counts
sqlite3 data/allstar.db "SELECT COUNT(*) as total_activity_logs FROM xp_activity_logs;"

# Check tally state
sqlite3 data/allstar.db "SELECT * FROM tally_state;"
```

### Notes

- **Users will lose their progress** - all levels and XP reset to zero, then recalculated from transmission logs
- **Rested bonuses are recalculated** - based on `last_transmission_at` timestamps
- **Processing time depends on log volume** - expect ~1-2 seconds per 1000 transmissions
- **Safe to interrupt** - if you stop the server mid-catch-up, it will resume from `last_tally_at` on next start

#### Tip: Epoch backfill after timestamp normalization

If you recently normalized legacy timestamp strings (e.g., by running the `db backfill-timestamps` command), it’s recommended to also populate the epoch columns for efficient and correct time-window queries:

```bash
go run . db backfill-epochs
```

This fills `start_unix`/`end_unix` from the normalized `timestamp_start`/`timestamp_end` values. It’s safe to run multiple times and can be done before or after the full reset steps above.

---

## Partial Reset (Backdate Only)

If you only want to **add missing XP** without resetting existing progress:

```bash
# Backdate tally to specific date
sqlite3 data/allstar.db "UPDATE tally_state SET last_tally_at = '2025-10-01 00:00:00' WHERE id = 1;"
```

⚠️ **Warning**: This may double-count transmissions if they were already processed. Only use this if you're certain the time period wasn't previously tallied.

---

## Individual Callsign Cap Reset

If you need to reset the daily/weekly XP caps for a specific callsign without affecting their total XP or level:

### Understanding Cap Calculation

The dashboard cap display is calculated from `xp_activity_logs`, not from the `daily_xp`/`weekly_xp` columns in `callsign_profiles`:

- **`callsign_profiles.daily_xp`** and **`weekly_xp`** - Cached values, not used for cap calculations
- **`xp_activity_logs.raw_xp`** - Summed for current day/week to calculate actual cap usage
- **`callsign_profiles.experience_points`** - Your actual XP total (unaffected by cap resets)

### Reset Caps for One Callsign

To reset caps for a specific callsign (e.g., N8DBF) while preserving their total XP and level:

```bash
sqlite3 data/allstar.db << 'EOF'
-- Clear activity logs for current week (keeps total XP intact)
DELETE FROM xp_activity_logs WHERE callsign = 'N8DBF';

-- Optional: Also reset cached cap values
UPDATE callsign_profiles
SET daily_xp = 0, weekly_xp = 0
WHERE callsign = 'N8DBF';
EOF
```

**What this does:**
- ✅ Resets cap display to 0/10000 (daily) and 0/40000 (weekly)
- ✅ Clears activity history for that callsign
- ✅ Preserves total XP and level in `callsign_profiles`
- ❌ Does NOT affect other callsigns

**Important:** This removes the hourly activity breakdown for the callsign. They will start fresh from the next transmission.

### Reset Only Current Week/Day

To preserve older activity history:

```bash
# Reset only today's caps
sqlite3 data/allstar.db "DELETE FROM xp_activity_logs WHERE callsign = 'N8DBF' AND hour_bucket >= datetime('now', 'start of day');"

# Reset only this week's caps (week starts Sunday at 00:00 UTC)
sqlite3 data/allstar.db "DELETE FROM xp_activity_logs WHERE callsign = 'N8DBF' AND hour_bucket >= datetime('now', 'weekday 0', '-7 days', 'start of day');"
```

---

## Rollback

If you need to restore the backup:

```bash
# Stop server first
./allstar-nexus stop  # or kill process

# Restore backup
cp data/allstar.db.backup-YYYYMMDD-HHMMSS data/allstar.db

# Restart
./allstar-nexus
```
