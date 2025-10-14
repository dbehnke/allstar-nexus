# Gamification System Implementation Plan

## Overview
This document outlines the complete implementation plan for adding a gamification system to Allstar Nexus, including a comprehensive migration of all database operations to GORM.

## Goals
1. Migrate existing database operations from raw SQL to GORM
2. Implement a callsign-based experience and leveling system
3. Add a renown/prestige system for unlimited progression
4. **NEW:** Implement rested XP bonus to reward returning players
5. **NEW:** Add diminishing returns to prevent marathon abuse
6. **NEW:** Detect and penalize kerchunking/spam
7. **NEW:** Optional daily/weekly XP caps (scaled for low-activity hubs)
8. Replace the Node Status tab with a new Talker Log view
9. Create an engaging scoreboard and transmission history UI

---

## ⚠️ Low-Activity Hub Scaling

This hub has relatively low talk volume. All XP values and thresholds are **scaled down** to create meaningful progression with limited activity:

- **Weekly XP Cap:** 2 hours maximum (7,200 seconds)
- **Level Requirements:** Scaled to match low-volume environment
- **Diminishing Returns:** Adjusted thresholds for shorter sessions
- **Rested Bonus:** More generous to reward infrequent check-ins

---

## Phase 1: GORM Migration of Existing Models

### 1.1 Convert Existing Models to GORM

#### Users Model ([backend/models/user.go](backend/models/user.go))
- Add GORM tags to existing `User` struct
- Tags needed: `gorm:"primaryKey;autoIncrement"`, `gorm:"unique;not null"`, etc.
- Add `TableName()` method returning `"users"`

**Example:**
```go
type User struct {
    ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
    Email        string    `gorm:"unique;not null;size:255" json:"email"`
    PasswordHash string    `gorm:"not null" json:"-"`
    Role         string    `gorm:"not null;default:user;size:50" json:"role"`
    CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (User) TableName() string {
    return "users"
}
```

#### LinkStats Model (NEW: [backend/models/link_stats.go](backend/models/link_stats.go))
- Create new GORM model for existing `link_stats` table
- Fields: `Node` (PK), `TotalTxSeconds`, `LastTxStart`, `LastTxEnd`, `ConnectedSince`, `UpdatedAt`
- Add GORM tags and `TableName()` method

**Example:**
```go
type LinkStat struct {
    Node           int        `gorm:"primaryKey" json:"node"`
    TotalTxSeconds int        `gorm:"not null;default:0" json:"total_tx_seconds"`
    LastTxStart    *time.Time `gorm:"type:timestamp" json:"last_tx_start"`
    LastTxEnd      *time.Time `gorm:"type:timestamp" json:"last_tx_end"`
    ConnectedSince *time.Time `gorm:"type:timestamp" json:"connected_since"`
    UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (LinkStat) TableName() string {
    return "link_stats"
}
```

### 1.2 Convert Repositories to GORM

#### UserRepo ([backend/repository/user_repo.go](backend/repository/user_repo.go))
Replace `*sql.DB` with `*gorm.DB` and convert methods:

- `Create()` → `db.Create(&user)`
- `GetByEmail()` → `db.Where("email = ?", email).First(&user)`
- `Count()` → `db.Model(&models.User{}).Count(&count)`
- `RoleCounts()` → Use `db.Model(&models.User{}).Select("role, count(*) as count").Group("role").Scan(&results)`
- `NewUsersSince()` → `db.Model(&models.User{}).Where("created_at >= ?", since).Count(&count)`

#### LinkStatsRepo ([backend/repository/link_stats_repo.go](backend/repository/link_stats_repo.go))
Replace `*sql.DB` with `*gorm.DB` and convert methods:

- `Upsert()` → Use `db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&stat)`
- `GetAll()` → `db.Find(&stats)`
- `DeleteNotIn()` → `db.Where("node NOT IN ?", activeNodes).Delete(&models.LinkStat{})`

### 1.3 Update main.go Database Initialization
- Keep both `database.Open()` (for sql.DB) and `gorm.Open()` (for GORM) temporarily
- Pass GORM DB to repositories that use it
- Phase out `database.DB` wrapper once migration complete
- Update `database.Migrate()` to only handle legacy migrations; rely on GORM `AutoMigrate()` going forward

---

## Phase 2: New Gamification Models (GORM-native)

### 2.1 CallsignProfile Model ([backend/models/callsign_profile.go](backend/models/callsign_profile.go))

**Purpose:** Track experience, level, and renown for each callsign

```go
type CallsignProfile struct {
    ID               uint      `gorm:"primaryKey" json:"id"`
    Callsign         string    `gorm:"uniqueIndex;size:20;not null" json:"callsign"`
    Level            int       `gorm:"default:1;index" json:"level"`
    ExperiencePoints int       `gorm:"default:0" json:"experience_points"`
    RenownLevel      int       `gorm:"default:0;index" json:"renown_level"`
    LastTallyAt      time.Time `gorm:"index" json:"last_tally_at"`
    CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CallsignProfile) TableName() string {
    return "callsign_profiles"
}
```

### 2.2 LevelConfig Model ([backend/models/level_config.go](backend/models/level_config.go))

**Purpose:** Store XP requirements for each level (configurable scaling)

```go
type LevelConfig struct {
    Level              int    `gorm:"primaryKey" json:"level"`
    RequiredExperience int    `gorm:"not null" json:"required_experience"`
    Name               string `gorm:"size:100" json:"name"` // Future: "Newbie", "Veteran", etc.
}

func (LevelConfig) TableName() string {
    return "level_configs"
}
```

### 2.3 Repositories

#### CallsignProfileRepo ([backend/repository/callsign_profile_repo.go](backend/repository/callsign_profile_repo.go))

**Methods:**
- `GetByCallsign(ctx context.Context, callsign string) (*models.CallsignProfile, error)`
  - Returns profile or creates new one if not exists
- `Upsert(ctx context.Context, profile *models.CallsignProfile) error`
  - Create or update profile
- `AddExperience(ctx context.Context, callsign string, xpToAdd int) error`
  - Increment XP atomically using: `db.Model(&profile).Update("experience_points", gorm.Expr("experience_points + ?", xpToAdd))`
- `GetLeaderboard(ctx context.Context, limit int) ([]models.CallsignProfile, error)`
  - Order by: `renown_level DESC, level DESC, experience_points DESC`
- `GetProfilesNeedingLevelUp(ctx context.Context, levelConfigs map[int]int) ([]models.CallsignProfile, error)`
  - Find profiles where XP >= required for next level
- `BulkUpdate(ctx context.Context, profiles []models.CallsignProfile) error`
  - Update multiple profiles in transaction (for level-ups)

#### LevelConfigRepo ([backend/repository/level_config_repo.go](backend/repository/level_config_repo.go))

**Methods:**
- `GetAll(ctx context.Context) ([]models.LevelConfig, error)`
  - Load all level configs (cache in memory)
- `SeedDefaults(ctx context.Context, configs []models.LevelConfig) error`
  - Insert default level scaling on first run
  - Use `db.Clauses(clause.OnConflict{DoNothing: true}).Create(&configs)` to avoid duplicates

---

## Phase 3: Gamification Service

### 3.1 Level Scaling Calculator ([backend/gamification/levels.go](backend/gamification/levels.go))

**Function:** `CalculateLevelRequirements(config map[string]interface{}) map[int]int`

**Default Scaling (Low-Activity Hub Version):**
- **Levels 1-10:** 360 XP each (6 minutes per level = easy progression for newcomers)
  - With 2 hours/week cap: ~3.3 levels per week possible for active users
- **Levels 11-60:** Logarithmic curve
  - Formula: `base + int((level-10)^1.8 * scale_factor)`
  - Tuned so total XP from 1→60 = 259,200 seconds (72 hours of talking = ~36 weeks at max participation)

**Rationale:**
- 2 hours/week cap = 7,200 XP max per week
- 10x reduction from high-activity hub scaling
- Reaching level 60 requires ~36 weeks of consistent weekly participation (sustainable for low-volume)

**Configurable via config.yaml** (see Phase 5)

**Example Implementation (Low-Activity Hub Version):**
```go
func CalculateLevelRequirements(cfg map[string]interface{}) map[int]int {
    requirements := make(map[int]int)

    // Levels 1-10: Linear (360 XP = 6 minutes each)
    // Low-activity scaling: 10x reduction from standard
    for level := 1; level <= 10; level++ {
        requirements[level] = 360  // 6 minutes per level
    }

    // Levels 11-60: Logarithmic scaling
    // Target: 259,200 total XP (72 hours = ~36 weeks at 2hr/week cap)
    // Already used: 10 * 360 = 3,600 XP
    // Remaining: 255,600 XP across 50 levels

    totalRemaining := 255600.0
    sum := 0.0
    for level := 11; level <= 60; level++ {
        sum += math.Pow(float64(level-10), 1.8)
    }

    scaleFactor := totalRemaining / sum
    for level := 11; level <= 60; level++ {
        xp := int(math.Pow(float64(level-10), 1.8) * scaleFactor)
        requirements[level] = xp
    }

    return requirements
}
```

**Example Level Progression:**
| Level | XP Required | Cumulative XP | Weeks at Max Cap* |
|-------|-------------|---------------|-------------------|
| 1     | 360         | 360           | 0.05              |
| 5     | 360         | 1,800         | 0.25              |
| 10    | 360         | 3,600         | 0.5               |
| 20    | ~4,200      | ~45,000       | 6.25              |
| 30    | ~9,800      | ~115,000      | 16                |
| 40    | ~16,500     | ~205,000      | 28.5              |
| 50    | ~24,000     | ~295,000      | 41                |
| 60    | ~32,000     | ~390,000      | 54                |

\* Assuming 7,200 XP/week (2 hours at 100% rate)

### 3.2 Tally Service ([backend/gamification/tally_service.go](backend/gamification/tally_service.go))

**Purpose:** Periodically process XP from transmission logs and handle level-ups

```go
type TallyService struct {
    db              *gorm.DB
    txLogRepo       *repository.TransmissionLogRepository
    profileRepo     *repository.CallsignProfileRepo
    levelConfigRepo *repository.LevelConfigRepo
    levelRequirements map[int]int // level → xp_required
    tallyInterval   time.Duration
    ticker          *time.Ticker
    stopChan        chan struct{}
    logger          *zap.Logger
}

func NewTallyService(db *gorm.DB, txRepo, profileRepo, levelRepo, interval, logger) *TallyService
func (s *TallyService) Start() error
func (s *TallyService) Stop()
func (s *TallyService) ProcessTally() error
```

**ProcessTally() Logic:**
1. Query transmission logs: `SELECT callsign, SUM(duration_seconds) as total_xp FROM transmission_logs GROUP BY callsign`
2. For each callsign with XP:
   - Load or create `CallsignProfile`
   - Add XP to profile: `AddExperience(callsign, xp)`
   - Check for level-up in a loop (handle multiple level-ups at once):
     ```go
     for profile.ExperiencePoints >= levelRequirements[profile.Level+1] {
         profile.ExperiencePoints -= levelRequirements[profile.Level+1]
         profile.Level++

         // Check for renown (prestige) at level 60
         if profile.Level >= 60 {
             profile.RenownLevel++
             profile.Level = 1
             profile.ExperiencePoints = 0
             // Emit level-up event (future: notifications)
         }
     }
     ```
   - Save updated profile
3. Update `LastTallyAt` timestamp
4. (Optional) Emit level-up events for notifications

**Tally Interval:** Default 30 minutes (configurable in config.yaml)

### 3.3 API Handler ([backend/api/gamification.go](backend/api/gamification.go))

**New API Layer:**
```go
type GamificationAPI struct {
    profileRepo *repository.CallsignProfileRepo
    txLogRepo   *repository.TransmissionLogRepository
    levelConfig map[int]int
}
```

**Endpoints:**

1. **GET /api/gamification/scoreboard?limit=50**
   - Returns top N callsigns ranked by: `renown_level DESC, level DESC, experience_points DESC`
   - Response includes: callsign, level, XP, renown, XP needed for next level

2. **GET /api/gamification/profile/:callsign**
   - Returns detailed profile for specific callsign
   - Includes: current level, XP, progress to next level, renown, rank, total talk time

3. **GET /api/gamification/recent-transmissions?limit=50&offset=0**
   - Paginated list of recent transmissions from `transmission_logs`
   - Returns: timestamp, callsign, node, duration
   - Used by frontend TransmissionHistoryCard

---

## Phase 4: Frontend Implementation

### 4.1 Router Changes ([frontend/src/router/index.js](frontend/src/router/index.js))

**Remove:**
```js
{
  path: '/status',
  name: 'NodeStatus',
  component: NodeStatus
}
```

**Add:**
```js
import TalkerLog from '../views/TalkerLog.vue'

{
  path: '/talker',
  name: 'TalkerLog',
  component: TalkerLog
}
```

**Update Navigation:** In [App.vue](frontend/src/App.vue), replace "Node Status" link with "Talker Log"

### 4.2 New View: TalkerLog.vue ([frontend/src/views/TalkerLog.vue](frontend/src/views/TalkerLog.vue))

**Purpose:** Main view for gamification features

**Layout:**
```vue
<template>
  <div class="talker-log-page">
    <div class="grid-layout">
      <!-- Recent Transmissions Card (left/top on mobile) -->
      <TransmissionHistoryCard
        :transmissions="recentTransmissions"
        :currentPage="currentPage"
        :totalPages="totalPages"
        @page-change="handlePageChange"
      />

      <!-- Scoreboard Card (right/bottom on mobile) -->
      <ScoreboardCard
        :scoreboard="scoreboard"
        :levelConfig="levelConfig"
        @refresh="refreshScoreboard"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import TransmissionHistoryCard from '../components/TransmissionHistoryCard.vue'
import ScoreboardCard from '../components/ScoreboardCard.vue'
import { useAuthStore } from '../stores/auth'

const scoreboard = ref([])
const recentTransmissions = ref([])
const currentPage = ref(1)
const totalPages = ref(5)
const levelConfig = ref({})

async function fetchScoreboard() {
  const auth = useAuthStore()
  const res = await fetch('/api/gamification/scoreboard?limit=50', {
    headers: auth.getAuthHeaders()
  })
  const data = await res.json()
  scoreboard.value = data.scoreboard || []
}

async function fetchRecentTransmissions(page = 1) {
  const limit = 10
  const offset = (page - 1) * limit
  const auth = useAuthStore()
  const res = await fetch(`/api/gamification/recent-transmissions?limit=50&offset=${offset}`, {
    headers: auth.getAuthHeaders()
  })
  const data = await res.json()
  recentTransmissions.value = data.transmissions || []
  totalPages.value = Math.ceil(50 / limit)
}

function handlePageChange(page) {
  currentPage.value = page
  fetchRecentTransmissions(page)
}

onMounted(() => {
  fetchScoreboard()
  fetchRecentTransmissions(1)
})
</script>

<style scoped>
.talker-log-page {
  padding: 1rem;
  max-width: 1600px;
  margin: 0 auto;
}

.grid-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

@media (max-width: 1024px) {
  .grid-layout {
    grid-template-columns: 1fr;
  }
}
</style>
```

### 4.3 New Components

#### TransmissionHistoryCard.vue ([frontend/src/components/TransmissionHistoryCard.vue](frontend/src/components/TransmissionHistoryCard.vue))

**Purpose:** Display last 50 transmissions with pagination

**Features:**
- Shows 10 transmissions per page (5 pages total)
- Columns: Timestamp, Callsign (with QRZ link), Node, Duration
- Scrollable container with custom scrollbar styling
- Pagination controls at bottom

**Props:**
- `transmissions: Array` - Array of transmission objects
- `currentPage: Number` - Current page number
- `totalPages: Number` - Total number of pages

**Emits:**
- `page-change` - When user clicks pagination button

**UI Elements:**
- Table with headers: Time | Callsign | Node | Duration
- Clickable callsign links to QRZ.com
- Relative time display (e.g., "2m ago", "1h 15m ago")
- Pagination: « Prev | 1 2 3 4 5 | Next »

#### ScoreboardCard.vue ([frontend/src/components/ScoreboardCard.vue](frontend/src/components/ScoreboardCard.vue))

**Purpose:** Display ranked leaderboard of callsigns

**Features:**
- Ranked list (top 50 by default)
- Display format: Rank badge, Callsign, Level badge, Renown badge (if >0), XP progress bar
- Top 3 get special styling (gold/silver/bronze rank badges)
- Hover tooltips showing detailed XP/level info
- Refresh button in card header

**Props:**
- `scoreboard: Array` - Array of profile objects
- `levelConfig: Object` - Map of level → XP required

**Emits:**
- `refresh` - When user clicks refresh button

**UI Elements:**
- Rank badge (circular, gradient for top 3)
- Callsign (bold, with QRZ link)
- Level indicator: "Level 15" (styled badge)
- Renown indicator: "⭐ Renown 3" (only if renown > 0)
- XP progress bar with text: "8,432 / 10,000 XP"

#### LevelProgressBar.vue ([frontend/src/components/LevelProgressBar.vue](frontend/src/components/LevelProgressBar.vue))

**Purpose:** Visual progress bar for XP → next level

**Props:**
- `currentXP: Number` - Current experience points
- `requiredXP: Number` - XP needed for next level
- `level: Number` - Current level

**Features:**
- Horizontal progress bar with gradient fill
- Color changes based on progress: blue (0-33%), green (34-66%), gold (67-100%)
- Text overlay: "Level 15: 8,432 / 10,000 XP (84%)"
- Animated fill on mount/update

### 4.4 Store Updates ([frontend/src/stores/node.js](frontend/src/stores/node.js))

**Add new refs:**
```js
const gamificationEnabled = ref(false)
const scoreboard = ref([])
const recentTransmissions = ref([])
const levelConfig = ref({})
```

**Add new methods:**
```js
async function fetchScoreboard(limit = 50) {
  const auth = useAuthStore()
  const res = await fetch(`/api/gamification/scoreboard?limit=${limit}`, {
    headers: auth.getAuthHeaders()
  })
  const data = await res.json()
  scoreboard.value = data.scoreboard || []
  gamificationEnabled.value = data.enabled || false
}

async function fetchRecentTransmissions(limit = 50, offset = 0) {
  const auth = useAuthStore()
  const res = await fetch(`/api/gamification/recent-transmissions?limit=${limit}&offset=${offset}`, {
    headers: auth.getAuthHeaders()
  })
  const data = await res.json()
  recentTransmissions.value = data.transmissions || []
}

async function fetchLevelConfig() {
  const res = await fetch('/api/gamification/level-config')
  const data = await res.json()
  levelConfig.value = data.config || {}
}
```

**Export new state/methods:**
```js
return {
  // ... existing exports
  gamificationEnabled,
  scoreboard,
  recentTransmissions,
  levelConfig,
  fetchScoreboard,
  fetchRecentTransmissions,
  fetchLevelConfig
}
```

### 4.5 Dashboard.vue Cleanup ([frontend/src/views/Dashboard.vue](frontend/src/views/Dashboard.vue))

**Remove:**
- Commented-out "Talker Log" section (lines 16-32)
- Any "Top Talkers" UI (if present)

**Keep:**
- `TopLinksCard` - Shows adjacent link statistics (node-based, not callsign)
- `SourceNodeCard` - Shows per-node live keying state
- All existing WebSocket integration and link tracking

**Rationale:**
- Dashboard focuses on real-time node/link status
- Talker Log view handles historical transmission stats and gamification

---

## Phase 5: Configuration

### 5.1 config.yaml additions

Add new `gamification` section:

```yaml
# Gamification System Configuration
gamification:
  enabled: true

  # How often to process XP and handle level-ups (in minutes)
  tally_interval_minutes: 30

  # Level scaling configuration
  level_scale:
    # Levels 1-10: Linear scaling (easy for newcomers)
    - levels: "1-10"
      xp_per_level: 3600  # 1 hour of talking per level

    # Levels 11-60: Logarithmic scaling to 30 days total
    - levels: "11-60"
      scaling: "logarithmic"
      target_total_seconds: 2592000  # 30 days of total talking time

  # Future: Level names/titles
  # level_names:
  #   1-10: "Newcomer"
  #   11-20: "Regular"
  #   21-40: "Veteran"
  #   41-59: "Legend"
  #   60: "Master"
```

### 5.2 Backend Config Struct ([backend/config/config.go](backend/config/config.go))

Add to existing config struct:

```go
type GamificationConfig struct {
    Enabled              bool                   `yaml:"enabled"`
    TallyIntervalMinutes int                    `yaml:"tally_interval_minutes"`
    LevelScale           []LevelScaleConfig     `yaml:"level_scale"`
}

type LevelScaleConfig struct {
    Levels             string `yaml:"levels"`              // e.g., "1-10", "11-60"
    XPPerLevel         int    `yaml:"xp_per_level"`         // For linear scaling
    Scaling            string `yaml:"scaling"`              // "linear" or "logarithmic"
    TargetTotalSeconds int    `yaml:"target_total_seconds"` // For logarithmic target
}

// Add to main Config struct:
type Config struct {
    // ... existing fields
    Gamification GamificationConfig `yaml:"gamification"`
}
```

---

## Phase 6: Integration & Wiring

### 6.1 main.go updates ([main.go](main.go))

**Steps:**

1. **Initialize GORM for all models:**
```go
// Auto-migrate all models (GORM handles schema)
if err := gormDB.AutoMigrate(
    &models.User{},
    &models.TransmissionLog{},
    &models.NodeInfo{},
    &models.LinkStat{},
    &models.CallsignProfile{},
    &models.LevelConfig{},
); err != nil {
    log.Fatalf("GORM auto-migrate error: %v", err)
}
logger.Info("GORM database migrated successfully")
```

2. **Convert existing repositories to use GORM:**
```go
// Update UserRepo to use GORM
userRepo := repository.NewUserRepo(gormDB)  // Changed from db.DB

// Update LinkStatsRepo to use GORM
lsRepo := repository.NewLinkStatsRepo(gormDB)  // Changed from db.DB
```

3. **Initialize gamification repositories:**
```go
// Initialize gamification repositories
profileRepo := repository.NewCallsignProfileRepo(gormDB)
levelConfigRepo := repository.NewLevelConfigRepo(gormDB)

// Seed level config defaults
if cfg.Gamification.Enabled {
    levelRequirements := gamification.CalculateLevelRequirements(cfg.Gamification.LevelScale)
    if err := levelConfigRepo.SeedDefaults(context.Background(), levelRequirements); err != nil {
        logger.Warn("failed to seed level config", zap.Error(err))
    }
}
```

4. **Initialize and start TallyService:**
```go
if cfg.Gamification.Enabled {
    tallyInterval := time.Duration(cfg.Gamification.TallyIntervalMinutes) * time.Minute
    tallyService := gamification.NewTallyService(
        gormDB,
        txLogRepo,
        profileRepo,
        levelConfigRepo,
        tallyInterval,
        logger,
    )

    if err := tallyService.Start(); err != nil {
        logger.Error("failed to start tally service", zap.Error(err))
    } else {
        logger.Info("gamification tally service started", zap.Duration("interval", tallyInterval))
    }

    // Cleanup on shutdown
    defer tallyService.Stop()
}
```

5. **Register gamification API routes:**
```go
// Gamification API endpoints
if cfg.Gamification.Enabled {
    gamificationAPI := api.NewGamificationAPI(profileRepo, txLogRepo, levelConfigRepo)

    if cfg.AllowAnonDashboard {
        publicLimiter := middleware.RateLimiter(cfg.PublicStatsRateLimitRPM)
        mux.Handle("/api/gamification/scoreboard", publicLimiter(http.HandlerFunc(gamificationAPI.Scoreboard)))
        mux.Handle("/api/gamification/profile/", publicLimiter(http.HandlerFunc(gamificationAPI.Profile)))
        mux.Handle("/api/gamification/recent-transmissions", publicLimiter(http.HandlerFunc(gamificationAPI.RecentTransmissions)))
    } else {
        mux.Handle("/api/gamification/scoreboard", authMW(http.HandlerFunc(gamificationAPI.Scoreboard)))
        mux.Handle("/api/gamification/profile/", authMW(http.HandlerFunc(gamificationAPI.Profile)))
        mux.Handle("/api/gamification/recent-transmissions", authMW(http.HandlerFunc(gamificationAPI.RecentTransmissions)))
    }
}
```

6. **Phase out old database wrapper (optional, later):**
```go
// Eventually remove db.DB and rely solely on gormDB
// For now, keep both during transition period
```

### 6.2 Transmission Log Integration

**Current State:**
- `transmission_logs` table already has `callsign` field
- Populated by `StateManager` during TX events
- Used by TallyService to aggregate XP by callsign

**Key Points:**
- Tally service queries: `SELECT callsign, SUM(duration_seconds) FROM transmission_logs GROUP BY callsign`
- XP is callsign-based (not node-based), so users get credit regardless of which node they transmit through
- Multiple nodes transmitting with same callsign = combined XP

---

## Phase 7: Testing & Validation

### 7.1 Database Migration Testing

**Test Cases:**
1. Start with existing database (users + link_stats)
2. Run new code with GORM auto-migrate
3. Verify:
   - Existing users still load correctly
   - Login/authentication still works
   - Link stats persist and update correctly
   - No data loss or corruption
   - New tables created: `callsign_profiles`, `level_configs`

**Rollback Plan:**
- Keep backup of `allstar.db` before migration
- If issues occur, restore backup and fix migration code

### 7.2 Gamification Logic Testing

**Test Scenarios:**

1. **XP Accumulation:**
   - Seed transmission logs with various callsigns
   - Run tally service
   - Verify XP correctly aggregated by callsign

2. **Level-Up Logic:**
   - Create profile with XP just below level-up threshold
   - Add XP to trigger level-up
   - Verify level incremented, XP reduced appropriately

3. **Multiple Level-Ups:**
   - Add enough XP to jump multiple levels at once
   - Verify all levels processed correctly

4. **Renown System:**
   - Create level 59 profile near level-up
   - Add XP to reach level 60
   - Verify: renown +1, level reset to 1, XP reset to 0

5. **Scoreboard Ranking:**
   - Create profiles with various level/renown combinations
   - Fetch scoreboard
   - Verify sort order: renown DESC, level DESC, XP DESC

### 7.3 Frontend Testing

**Test Cases:**

1. **Talker Log View:**
   - Navigate to `/talker` route
   - Verify both cards render correctly
   - Check data loads from API

2. **Pagination:**
   - Click through all 5 pages of recent transmissions
   - Verify correct data displayed for each page
   - Test prev/next buttons

3. **Scoreboard:**
   - Verify top 3 have special styling (gold/silver/bronze)
   - Hover over entries to see tooltips
   - Click refresh button to reload data

4. **Progress Bars:**
   - Verify XP progress bars show correct percentage
   - Check color changes at 33% and 67% thresholds
   - Verify text overlay shows correct XP/level info

5. **Responsive Design:**
   - Test on mobile (grid should stack vertically)
   - Test on tablet and desktop
   - Verify scrollbars work correctly

### 7.4 Performance Testing

**Metrics to Monitor:**

1. **Tally Service:**
   - Time to process tally (should be < 1 second for 1000 callsigns)
   - Database query performance
   - CPU/memory usage during tally

2. **API Endpoints:**
   - Response time for scoreboard (< 100ms)
   - Response time for recent transmissions (< 100ms)
   - Database connection pool utilization

3. **Frontend:**
   - Page load time for Talker Log view
   - Render performance with 50 scoreboard entries
   - Smooth pagination transitions

---

## Implementation Status

### ✅ Backend GORM Migration (COMPLETED)
- ✅ Convert `User` model to GORM with tags
- ✅ Convert `LinkStat` model to GORM with tags
- ✅ Convert `UserRepo` to use GORM API
- ✅ Convert `LinkStatsRepo` to use GORM API
- ✅ Update main.go to use GORM for existing repos
- ✅ Update handlers.go to accept gorm.DB

### ✅ New Gamification Models (COMPLETED)
- ✅ `CallsignProfile` model for XP/level/renown tracking
- ✅ `LevelConfig` model for XP requirements
- ✅ `XPActivityLog` model for anti-cheating transparency
- ✅ `CallsignProfileRepo` with GORM methods
- ✅ `LevelConfigRepo` with GORM methods
- ✅ `XPActivityRepo` with comprehensive tracking methods

### ✅ Gamification Service (COMPLETED)
- ✅ Level scaling calculator (linear + logarithmic) - low-activity hub version
- ✅ Tally service for periodic XP processing (30-minute intervals)
- ✅ Level-up logic with renown prestige system
- ✅ Complete anti-cheating implementation:
  - ✅ Rested XP bonus (2.0x multiplier, 14-day max)
  - ✅ Diminishing returns (4 tiers: 100% → 75% → 50% → 25%)
  - ✅ Kerchunk detection (penalizes <3sec spam)
  - ✅ Daily/weekly XP caps (20 min/day, 2 hrs/week)
- ✅ API endpoints: scoreboard, profile, recent transmissions, level config
- ✅ Integrated into main.go with graceful shutdown

### ✅ Frontend Changes (COMPLETED) — updated 2025-10-11
- ✅ Remove NodeStatus route, add TalkerLog route
  - Route added: `/talker` in `frontend/src/router/index.js`
  - Nav updated in `frontend/src/App.vue` ("Node Status" → "Talker Log")
- ✅ Create TalkerLog.vue view with grid layout
  - File: `frontend/src/views/TalkerLog.vue`
  - Loads scoreboard, recent transmissions, and level config
- ✅ Create TransmissionHistoryCard component (paginated)
  - File: `frontend/src/components/TransmissionHistoryCard.vue`
  - 10 items/page, 5 pages (50 recent)
- ✅ Create ScoreboardCard component (ranked leaderboard)
  - File: `frontend/src/components/ScoreboardCard.vue`
  - Rank badges (gold/silver/bronze), callsign links, level/renown
- ✅ Create LevelProgressBar component
  - File: `frontend/src/components/LevelProgressBar.vue`
  - Animated fill, color thresholds, overlay text
- ✅ Update Dashboard.vue: remove talker log, keep TopLinksCard
  - File: `frontend/src/views/Dashboard.vue` (removed deprecated commented block)
- ✅ Update store with gamification state/methods
  - File: `frontend/src/stores/node.js`
  - Added: `gamificationEnabled`, `scoreboard`, `recentTransmissions`, `levelConfig`
  - Added methods: `fetchScoreboard`, `fetchRecentTransmissions`, `fetchLevelConfig`

Build status: Frontend production build PASS (vite)

### ✅ Configuration (COMPLETED) — updated 2025-10-11
- ✅ Add `gamification` section to config.yaml with full low-activity hub settings
- ✅ Support for customizable level scaling
- ✅ Configurable tally interval (default 30 min)
- ✅ All anti-cheating mechanics configurable
- ✅ Default: disabled (set `enabled: true` to activate)

Build status: Backend build PASS (go)

Files changed in Phase 5:
- `backend/config/config.go` — add `LevelScale` config and defaults
- `backend/gamification/levels.go` — implement configurable level scaling
- `main.go` — seed level config using configured scale
- `config.yaml.example` — include `gamification` section and examples

---

### ✅ Testing & Validation (COMPLETED) — updated 2025-10-11

Scope covered by automated tests and a light micro-benchmark:

- API tests: scoreboard, level-config, recent-transmissions pagination
  - File: `backend/tests/gamification_test.go`
- Level scaling unit tests: default, linear override, logarithmic target sum
  - File: `backend/tests/levels_calc_test.go`
- Tally Service E2E: diminishing returns, caps, kerchunk penalty, rested bonus, idempotency
  - File: `backend/tests/tally_service_test.go`
- Profile API integration: aggregates, daily/weekly totals, daily_breakdown, next_level_xp
  - File: `backend/tests/profile_api_test.go`
- Leveling behavior: carryover XP across level-ups, renown reset at 60→1
  - File: `backend/tests/levelup_renown_test.go`
- Performance baseline: ProcessTally micro-benchmark (no assertions; for local tuning)
  - File: `backend/tests/benchmark_tally_test.go`

Status: Full test suite PASS.

How to run locally:

- Run all tests
  - From repo root: `go test ./...`
- Run only backend test package (verbose)
  - `go test ./backend/tests -v`
- Run the micro-benchmark (optional)
  - `go test -bench=ProcessTally -run '^$' ./backend/tests`

Notes:

- Tests initialize GORM SQLite and call TallyService.Start() where required.
- Time-based tests seed transmission logs before service start to ensure expected cutoffs.

---

## Benefits

1. **Modern ORM:** GORM provides cleaner, safer, more maintainable database code
2. **Type Safety:** GORM models catch errors at compile time
3. **Auto Migrations:** Schema changes handled automatically
4. **Fun Engagement:** Gamification encourages community participation
5. **Unlimited Progression:** Renown system keeps veterans engaged
6. **Fair XP System:** Callsign-based XP works across multiple nodes
7. **Scalable Design:** Logarithmic scaling prevents early saturation
8. **Transparent Tracking:** Scoreboard and history provide visibility

---

## Implementation Order

**Recommended sequence:**

1. **Phase 1:** GORM migration (stabilize existing functionality)
2. **Phase 2:** Add gamification models
3. **Phase 3:** Implement gamification service
4. **Phase 5:** Add configuration
5. **Phase 6:** Wire up in main.go
6. **Phase 4:** Build frontend (can work in parallel with backend)
7. **Phase 7:** Test everything

**Estimated Timeline:**
- Phase 1: 4-6 hours (GORM migration)
- Phase 2: 2-3 hours (models + repos)
- Phase 3: 4-5 hours (service + API)
- Phase 4: 6-8 hours (frontend)
- Phase 5: 1 hour (config)
- Phase 6: 2-3 hours (integration)
- Phase 7: 3-4 hours (testing)

**Total: ~22-30 hours**

---

## Low-Activity Hub Scaling Summary

### Key Differences from Standard Configuration

| Metric | Standard Hub | Low-Activity Hub | Scale Factor |
|--------|--------------|------------------|--------------|
| **Weekly XP Cap** | 40 hours (144,000) | 2 hours (7,200) | 20x reduction |
| **Daily XP Cap** | 8 hours (28,800) | 20 min (1,200) | 24x reduction |
| **Level 1-10 XP** | 3,600 (1 hour) | 360 (6 min) | 10x reduction |
| **Total XP to 60** | 2,592,000 (30 days) | 259,200 (72 hours) | 10x reduction |
| **Time to Max Level** | ~9 weeks (40hr/wk) | ~36 weeks (2hr/wk) | Slower but achievable |
| **Rested Multiplier** | 1.5x (50% bonus) | 2.0x (100% bonus) | More generous |
| **Rested Max** | 168 hours (7 days) | 336 hours (14 days) | More forgiving |
| **DR Threshold 1** | 4 hours | 20 minutes | Proportional |

### Why This Works for Low-Activity Hubs

1. **Achievable Goals:** Level progression feels meaningful even with limited weekly activity
2. **Generous Rested Bonus:** Casual participants get doubled XP to stay competitive
3. **Anti-Grinding:** Daily caps prevent someone from monopolizing weekly allowance in one day
4. **Long-Term Engagement:** 36-week journey to max level creates sustained motivation
5. **Fair Competition:** Everyone has same 2hr/week limit, levels the playing field
6. **Immediate Gratification:** First 10 levels are quick (6 min each) to hook new users

### Example Player Journey (Low-Activity Hub)

**Week 1:**
- Talks for 30 minutes on Monday (all at 100% rate = 1,800 XP)
- Talks for 30 minutes on Friday (all at 100% rate = 1,800 XP)
- Total: 3,600 XP = **Level 10** (completed newbie tier!)

**Week 2-4:** (Taking a 2-week break)
- Rested bonus accumulates: 14 days idle = 21 hours of 2.0x XP bonus available
- Returns Week 4 with massive bonus ready

**Week 4:**
- Talks for 2 hours (weekly cap) with 2.0x rested multiplier
- Earns: 7,200 base XP × 2.0 = **14,400 XP** (would have been 7,200)
- Bonus XP helps catch up from missed weeks

**Long-Term:**
- Consistent 2hr/week player reaches Level 60 in ~36 weeks
- Players who take breaks get rested bonus to stay competitive
- No single player can dominate through excessive grinding

---

## Future Enhancements

1. **Level Names:** Add titles to levels (Newcomer, Regular, Veteran, Legend, Master)
2. **Achievements:** Special badges for milestones (first transmission, 100 hours, etc.)
3. **Notifications:** Real-time level-up notifications via WebSocket
4. **Profile Pages:** Dedicated page per callsign with detailed stats
5. **Leaderboard Filters:** Filter by time period (daily, weekly, monthly, all-time)
6. **XP Multipliers:** Bonus XP for late-night ops, special events, etc.
7. **Team/Club Leaderboards:** Aggregate XP by club membership
8. **Statistics Dashboard:** Charts showing XP trends, most active times, etc.

---

## Notes

- Ensure callsign normalization (uppercase, trim whitespace) for consistency
- Consider rate limiting on scoreboard API to prevent abuse
- Monitor database size; may need log retention policy for old transmissions
- Level 60 renown reset is intentional to maintain engagement
- Logarithmic scaling ensures level 60 requires significant dedication (~30 days of talking)
