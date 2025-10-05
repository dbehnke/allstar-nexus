# Allstar Nexus - Project Status & Roadmap

## ✅ **Completed Features**

### Frontend (Vue.js 3)
- ✅ **Modern SPA Dashboard** with Vue Router and Pinia
- ✅ **Card-Based UI** with dark theme and responsive design
- ✅ **4 Main Views:**
  - Dashboard - Real-time node monitoring
  - Node Lookup - Search 43K+ AllStar nodes
  - RPT Stats - Asterisk statistics (basic)
  - Voter Display - RTCM visualization (basic)
- ✅ **Real-time WebSocket** integration for live updates
- ✅ **Authentication** system with JWT tokens
- ✅ **Responsive Design** - Works on mobile, tablet, desktop

### Backend (Go)
- ✅ **RESTful API** with rate limiting
- ✅ **Auto-Download astdb.txt** - 43,096 nodes from AllStarLink
- ✅ **Node Lookup API** - Search by number/callsign
- ✅ **RPT Stats API** - Via AMI (basic implementation)
- ✅ **Voter Stats API** - Via AMI (basic implementation)
- ✅ **Basic AMI Connector** - Connect, login, send commands
- ✅ **WebSocket Hub** - Real-time event broadcasting
- ✅ **SQLite Database** - User management, link stats

### Infrastructure
- ✅ **Single Binary** - Vue embedded in Go executable
- ✅ **Auto-Updates** - astdb refreshes daily
- ✅ **Structured Logging** - Using zap logger
- ✅ **Build System** - Vite for Vue, Go build pipeline

### Documentation
- ✅ **FEATURES.md** - Complete feature documentation
- ✅ **QUICKSTART.md** - Quick start guide
- ✅ **SUMMARY.md** - Implementation summary
- ✅ **ASTDB_AUTO_DOWNLOAD.md** - Auto-download docs
- ✅ **ASTDB_SOLUTION.md** - Problem resolution
- ✅ **AMI_COMMANDS_REFERENCE.md** - AMI command catalog

---

## 🔄 **In Progress / Next Steps**

### ✅ Phase 1: Core AMI Enhancement - COMPLETED

**Status:** ✅ Complete
**Duration:** Completed

Added support for AllStar-specific `RptStatus` action:
- ✅ `XStat` - Extended node status, connections, RX/TX state
- ✅ `SawStat` - Keying history, last heard times

**Files created:**
- ✅ `internal/ami/types.go` - Complete data structures (Connection, XStatResult, SawStatResult, etc.)
- ✅ `internal/ami/parsers.go` - Full XStat/SawStat parsers with EchoLink support
- ✅ `internal/ami/parsers_test.go` - Comprehensive unit tests
- ✅ `internal/ami/testdata/` - Test fixtures (xstat_basic, xstat_echolink, sawstat_basic)
- ✅ `internal/core/enhanced_poller.go` - New poller using combined status

**Files modified:**
- ✅ `internal/ami/connector.go` - Added RptStatus(), GetXStat(), GetSawStat(), GetCombinedStatus()
- ✅ `internal/core/state.go` - Added ApplyCombinedStatus() method
- ✅ `internal/core/links.go` - Enhanced LinkInfo with AMI fields (IP, Mode, LastHeard, etc.)

**Impact:** ✅ Full XStat/SawStat integration complete with tests passing

---

#### 2. **Connection State Tracking** - ✅ COMPLETED

**Status:** ✅ Complete

Parse connection details from XStat:
- ✅ Node number, IP address, direction (IN/OUT)
- ✅ Link mode (Transceive, Receive-only, Connecting)
- ✅ Connection duration
- ✅ Currently keyed status

**Files modified:**
- ✅ `internal/core/state.go` - Added ApplyCombinedStatus() with full connection tracking
- ✅ `internal/core/links.go` - Enhanced LinkInfo with all XStat/SawStat fields

**Impact:** ✅ Complete connection state tracking in StateManager

---

#### 3. **RX/TX Keying Detection** - ✅ COMPLETED

**Status:** ✅ Complete (backend)

Extract and broadcast:
- ✅ `RPT_RXKEYED` - Local receiver COS detection
- ✅ `RPT_TXKEYED` - Local transmitter PTT status

**Files modified:**
- ✅ `internal/ami/parsers.go` - Extracts Var: RPT_RXKEYED and RPT_TXKEYED
- ✅ `internal/core/state.go` - Tracks RxKeyed and TxKeyed state
- ⏳ `vue-dashboard/src/components/StatusCard.vue` - Show COS/PTT (pending)

**Impact:** ✅ Backend complete, frontend display pending

---

#### 4. **Last Heard Tracking** - ✅ COMPLETED

**Status:** ✅ Complete (backend)

Use SawStat to track:
- ✅ Seconds since last keyed
- ✅ Seconds since last unkeyed
- ✅ "Never" heard detection (large values)
- ✅ Human-readable formatting (HH:MM:SS or "Keying"/"Never")

**Files modified:**
- ✅ `internal/ami/parsers.go` - ParseSawStat() with full keying history
- ✅ `internal/ami/types.go` - FormatLastHeard() helper function
- ✅ `internal/core/state.go` - Tracks LastHeardAt, LastKeyedTime, SecsSinceKeyed
- ✅ `internal/core/links.go` - Enhanced LinkInfo with keying timestamps
- ⏳ `vue-dashboard/src/components/LinksCard.vue` - Sort by last heard (pending)

**Impact:** ✅ Backend complete with formatted last-heard times, frontend sorting pending

---

#### 5. **Node Type Detection**
**Priority:** Low
**Complexity:** Low

Detect and display node types:
- AllStar (< 80000)
- IRLP (80000-89999)
- EchoLink (> 3000000)
- Web/Phone portals

**Files to create:**
- `backend/api/node_types.go` - Detection logic
- `vue-dashboard/src/utils/nodeTypes.js` - Frontend helpers

**Impact:** Better node identification

---

#### 6. **Voter Command Enhancement**
**Priority:** Low
**Complexity:** Medium

Try multiple voter command formats:
- `voter show <node>`
- `rpt voter <node>`
- `rpt localplay <node> ...`

Parse various output formats from different app_voter versions.

**Files to modify:**
- `backend/api/voter_stats.go` - Try multiple commands
- Enhanced parsing for different formats

**Impact:** More reliable voter display

---

## 📋 **Feature Comparison**

| Feature | Supermon | Allstar Nexus | Status |
|---------|----------|---------------|--------|
| **Node Monitoring** | ✅ | ✅ | Complete |
| **WebSocket/SSE** | SSE | WebSocket | Better |
| **Node Lookup** | ✅ | ✅ | Complete |
| **Auto-Download DB** | Manual | ✅ Auto | Better |
| **RPT Stats** | ✅ Full | ⚠️ Basic | Needs enhancement |
| **Voter Display** | ✅ Full | ⚠️ Basic | Needs enhancement |
| **COS/PTT Indicators** | ✅ | ❌ | Not yet |
| **Link Modes (T/R/C)** | ✅ | ❌ | Not yet |
| **Last Heard Sort** | ✅ | ❌ | Not yet |
| **EchoLink/IRLP** | ✅ | ⚠️ Partial | Needs detection |
| **Modern UI** | ❌ | ✅ | Complete |
| **Responsive** | ⚠️ | ✅ | Better |
| **Auth System** | Basic | ✅ JWT | Better |
| **Single Binary** | ❌ | ✅ | Better |

---

## 🎯 **Recommended Implementation Order**

### ✅ Phase 1: Core AMI Enhancement - COMPLETED
1. ✅ Add `RptStatus` action support
2. ✅ Implement XStat parser with EchoLink support
3. ✅ Implement SawStat parser
4. ✅ Add connection state tracking
5. ✅ Extract RX/TX variables
6. ✅ Create comprehensive unit tests
7. ✅ Add test fixtures

### ✅ Phase 2: State Management - COMPLETED
1. ✅ Update StateManager with new AMI data
2. ✅ Track per-link keying state
3. ✅ Calculate elapsed times
4. ✅ Store last-heard timestamps
5. ✅ Create EnhancedPoller for XStat/SawStat polling

### Phase 3: WebSocket Events (1 day)
1. Add new message types for RX/TX
2. Include link mode in events
3. Send connection details
4. Update event batching

### Phase 4: Frontend Updates (1-2 days)
1. Add COS/PTT indicators to StatusCard
2. Show link modes in LinksCard
3. Implement last-heard sorting
4. Add node type badges
5. Enhanced voter visualization

### Phase 5: Testing & Refinement (1-2 days)
1. Test with real AllStar node
2. Validate voter commands
3. Performance testing
4. Bug fixes

**Total Estimated Time:** 6-10 days

---

## 🧪 **Testing Strategy**

### Without AllStar Hardware

**Mock AMI Responses:**
- Create test fixtures with sample XStat/SawStat data
- Unit tests for parsers
- Integration tests with mock AMI server

**Files to create:**
- `internal/ami/parsers_test.go`
- `internal/ami/testdata/xstat_*.txt`
- `internal/ami/testdata/sawstat_*.txt`

### With AllStar Hardware

**Real Node Testing:**
- Connect to live Asterisk AMI
- Verify XStat/SawStat parsing
- Test voter commands
- Validate event stream accuracy

---

## 📚 **Reference Materials**

### Supermon Source Code Analysis
- ✅ **server.php** - Analyzed (SSE loop, XStat/SawStat)
- ✅ **amifunctions.inc** - Analyzed (AMI helpers)
- ✅ **nodeinfo.inc** - Analyzed (Node type detection)
- ✅ **link.php** - Analyzed (Link display logic)
- ✅ **voter.php** - Analyzed (Voter visualization)

### AllStar Documentation
- AllStar Wiki: https://wiki.allstarlink.org/
- app_rpt Commands: AMI reference
- RTCM/Voter: app_voter documentation

---

## 🚀 **Current State**

**What Works Today:**
```bash
# Build and run
go build -o allstar-nexus .
./allstar-nexus

# Features working:
✅ Modern Vue dashboard
✅ Real-time WebSocket updates
✅ Node lookup (43K+ nodes)
✅ Basic RPT stats
✅ Basic voter display
✅ Authentication
✅ Auto-download astdb
```

**What Needs Enhancement:**
```
❌ COS/PTT indicators
❌ Link mode display (T/R/C)
❌ Last-heard sorting
❌ Enhanced voter parsing
❌ Node type detection
```

---

## 💡 **Quick Wins**

Easy improvements that add value:

1. **Add COS/PTT badges** (30 min)
   - Parse RPT_RXKEYED/RPT_TXKEYED
   - Add pulsing indicators to StatusCard

2. **Node type badges** (1 hour)
   - Detect AllStar/IRLP/EchoLink
   - Add colored badges in LinksCard

3. **Last heard sorting** (2 hours)
   - Parse SawStat response
   - Sort links by last keyed time

4. **Connection details** (1 hour)
   - Show IN/OUT direction
   - Display IP addresses

---

## 🎓 **Learning Resources**

For contributors wanting to enhance AMI support:

- **AMI Protocol:** `/var/log/asterisk/messages` on AllStar node
- **app_rpt Source:** https://github.com/AllStarLink/app_rpt
- **Supermon Source:** Included in `external/` directory
- **This Project's Docs:** All `*.md` files in root

---

## 📞 **Support & Contribution**

**Current State:** Production-ready for basic monitoring
**Enhancement State:** Well-documented, ready for community contributions

**To Contribute:**
1. Read [AMI_COMMANDS_REFERENCE.md](AMI_COMMANDS_REFERENCE.md)
2. Pick a feature from "In Progress" section above
3. Follow existing code patterns
4. Submit PR with tests

---

## 🎉 **Achievements So Far**

✅ **Modern Stack** - Vue 3 + Go (vs jQuery + PHP)
✅ **Better Performance** - WebSocket (vs SSE polling)
✅ **Auto-Updates** - astdb downloads automatically
✅ **Single Binary** - Easy deployment
✅ **Responsive UI** - Works on all devices
✅ **Full Documentation** - 6 comprehensive guides
✅ **Production Ready** - Built, tested, documented

**Ready for production use with current features!**
**Enhancement roadmap clearly defined for future development.**

---

**Last Updated:** October 5, 2025
**Status:** ✅ Core features complete, AMI enhancements documented
