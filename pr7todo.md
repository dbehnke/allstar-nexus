# Implementation Plan: Review Suggestions for PR #7

## Overview
Implementing 4 improvements suggested in the PR review, prioritized by impact and complexity.

---

## Task 1: Add Thread Safety to AMI Registry ⚠️ HIGH PRIORITY

**File**: `internal/ami/parsers.go`

**Current Code (lines 417-430)**:
```go
// AMI-layer text node registry
var amiTextNodeRegistry = make(map[int]string)

func registerTextNodeInAMI(nodeID int, text string) {
	amiTextNodeRegistry[nodeID] = strings.ToUpper(text)
}

func GetTextNodeFromAMI(nodeID int) (string, bool) {
	text, ok := amiTextNodeRegistry[nodeID]
	return text, ok
}
```

**Changes Needed**:
1. Add `sync` import to imports section (line 3-8)
2. Replace registry declaration with mutex-protected version:
```go
var (
	amiTextNodeRegistry = make(map[int]string)
	amiRegistryMu       sync.RWMutex
)
```

3. Update `registerTextNodeInAMI()`:
```go
func registerTextNodeInAMI(nodeID int, text string) {
	amiRegistryMu.Lock()
	defer amiRegistryMu.Unlock()
	amiTextNodeRegistry[nodeID] = strings.ToUpper(text)
}
```

4. Update `GetTextNodeFromAMI()`:
```go
func GetTextNodeFromAMI(nodeID int) (string, bool) {
	amiRegistryMu.RLock()
	defer amiRegistryMu.RUnlock()
	text, ok := amiTextNodeRegistry[nodeID]
	return text, ok
}
```

**Why**: Prevents race conditions when multiple goroutines access the global registry concurrently.

---

## Task 2: Add Hash Collision Detection ⚠️ HIGH PRIORITY

**File**: `internal/ami/parsers.go`

**Current Code (lines 422-424)**:
```go
func registerTextNodeInAMI(nodeID int, text string) {
	amiTextNodeRegistry[nodeID] = strings.ToUpper(text)
}
```

**Updated Code** (after adding mutex from Task 1):
```go
func registerTextNodeInAMI(nodeID int, text string) {
	amiRegistryMu.Lock()
	defer amiRegistryMu.Unlock()

	upperText := strings.ToUpper(text)

	// Check for hash collisions
	if existing, exists := amiTextNodeRegistry[nodeID]; exists && existing != upperText {
		log.Printf("WARNING: Hash collision detected! Callsigns %s and %s both hash to %d",
			existing, upperText, nodeID)
	}

	amiTextNodeRegistry[nodeID] = upperText
}
```

**Additional Changes**:
- Add `log` import to imports section (line 3-8)

**Why**: Provides visibility into hash collision issues that could cause callsign misidentification.

---

## Task 3: Replace Debug Print with Proper Logging 📝 MEDIUM PRIORITY

**File**: `internal/ami/parsers.go` (line 151)

**Current Code**:
```go
fmt.Printf("[AMI DEBUG] Registered text node: %s -> %d\n", callsign, nodeNum)
```

**Replace With**:
```go
log.Printf("[AMI] Registered text node: %s -> %d", callsign, nodeNum)
```

**Additional Cleanup**:
- Check if `fmt` package is still needed elsewhere in the file
- If line 151 is the only `fmt.Printf` usage, the import can stay (used for `fmt.Errorf`)

**Why**: Proper logging that respects Go conventions and can be controlled via logging configuration.

---

## Task 4: Add Documentation Comment 📚 LOW PRIORITY

**File**: `frontend/src/components/SourceNodeCard.vue` (lines 766-770)

**Current Code**:
```javascript
    // Normalize fractional seconds longer than milliseconds to 3 digits
    // Examples:
    // 2025-10-12T22:12:14.822791-04:00 -> 2025-10-12T22:12:14.822-04:00
    // 2025-10-12T22:12:14.822791Z -> 2025-10-12T22:12:14.822Z
    iso = iso.replace(/(\.\d{3})\d+([Zz]|[+\-]\d{2}:?\d{2})$/, '$1$2')
```

**Enhanced Comment**:
```javascript
    // Normalize fractional seconds longer than milliseconds to 3 digits
    // Go backend returns timestamps with microsecond precision (6 digits after decimal)
    // but JavaScript Date.parse() can have issues with non-standard precision in some browsers.
    // This normalizes to standard millisecond precision (3 digits).
    // Examples:
    //   2025-10-12T22:12:14.822791-04:00 -> 2025-10-12T22:12:14.822-04:00
    //   2025-10-12T22:12:14.822791Z -> 2025-10-12T22:12:14.822Z
    iso = iso.replace(/(\.\d{3})\d+([Zz]|[+\-]\d{2}:?\d{2})$/, '$1$2')
```

**Why**: Helps future maintainers understand the purpose of this normalization.

---

## Task 5: Testing & Verification ✅

**Test Commands**:
```bash
# Backend tests
go test ./...

# Specific AMI tests
go test ./internal/ami -v

# Frontend tests
cd frontend && npm test

# E2E tests (optional)
cd frontend && npm run test:e2e

# Optional: Check for race conditions
go test -race ./...
```

**What to Verify**:
- No race conditions detected (use `go test -race ./...` if desired)
- All existing tests still pass
- Hash collision detection logs appropriately
- No regressions in text node handling

---

## Task 6: Commit & Documentation 📝

**Commit Message**:
```
refactor: add thread safety and improve logging for text node registry

- Add mutex protection to AMI text node registry to prevent race conditions
- Add hash collision detection with warning logs
- Replace debug print statement with proper logging
- Improve documentation for microsecond timestamp normalization

Addresses review feedback from PR #7.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## Summary of Changes

| Task | File | Lines Changed | Priority | Risk |
|------|------|---------------|----------|------|
| Thread Safety | `internal/ami/parsers.go` | ~15 | HIGH | Low |
| Collision Detection | `internal/ami/parsers.go` | ~8 | HIGH | Low |
| Replace Print | `internal/ami/parsers.go` | 1 | MEDIUM | Very Low |
| Documentation | `SourceNodeCard.vue` | ~3 | LOW | None |

**Estimated Time**: 15-20 minutes
**Testing Time**: 5-10 minutes
**Total**: ~30 minutes

---

## Implementation Order

1. ✅ Add `sync` and `log` imports
2. ✅ Add mutex to registry declaration
3. ✅ Update `registerTextNodeInAMI()` with mutex + collision detection
4. ✅ Update `GetTextNodeFromAMI()` with read mutex
5. ✅ Replace debug print statement
6. ✅ Add frontend documentation
7. ✅ Run tests
8. ✅ Commit changes

---

## Files Affected

### Backend
- `internal/ami/parsers.go` - Main changes for thread safety and logging

### Frontend
- `frontend/src/components/SourceNodeCard.vue` - Documentation improvement

### Tests to Run
- `internal/ami/text_node_test.go` - Verify text node functionality
- `internal/core/state_test.go` - Verify core layer integration
- `backend/api/status_test.go` - Verify API layer
- All frontend tests

---

## Related Files (for reference, no changes needed)

- `internal/core/state.go` - Calls `GetTextNodeFromAMI()`
- `internal/core/nodelookup.go` - Calls `GetTextNodeFromAMI()`
- `internal/ami/text_node_test.go` - Tests the registry functions

---

## Notes

- The mutex changes ensure thread safety without changing the API of the registry functions
- Hash collision detection is purely observational (logs warnings but doesn't fail)
- All changes are backward compatible
- No breaking changes to existing functionality

---
---

# Part 2: Fix golangci-lint Issues

## Overview

Fix all 40 golangci-lint issues found in the codebase:
- **errcheck**: 36 issues (unchecked error returns)
- **staticcheck**: 2 issues (code simplification)
- **ineffassign**: 1 issue (ineffectual assignment)
- **unused**: 1 issue (unused type)

**Total Estimated Time**: 45-60 minutes

---

## Priority Levels

### 🔴 Critical (Database/Connection Operations)
Files with database or connection resource leaks

### 🟠 High (HTTP/File Operations)
Files with HTTP response or file handle leaks

### 🟡 Medium (Test Files)
Test files with unchecked errors

### 🟢 Low (Code Quality)
Simplification and cleanup

---

## Task 1: Fix API Layer Issues 🟠 HIGH PRIORITY

**File**: `backend/api/gamification.go`

**Issues**: 3 unchecked `json.Encoder.Encode()` calls

**Lines**: 101, 162, 223

**Fix Pattern**:
```go
// BEFORE:
json.NewEncoder(w).Encode(map[string]interface{}{
    "ok": true,
    "data": data,
})

// AFTER:
if err := json.NewEncoder(w).Encode(map[string]interface{}{
    "ok": true,
    "data": data,
}); err != nil {
    log.Printf("Failed to encode response: %v", err)
}
```

**Why**: Encoding errors are rare but should be logged for debugging.

---

## Task 2: Fix Test File Issues 🟡 MEDIUM PRIORITY

### File: `backend/api/status_test.go`

**Issue**: Line 33 - unchecked `res.Body.Close()`

**Fix**:
```go
// BEFORE:
defer res.Body.Close()

// AFTER:
defer func() {
    if err := res.Body.Close(); err != nil {
        t.Logf("Failed to close response body: %v", err)
    }
}()
```

---

### File: `backend/database/database_test.go`

**Issue**: Line 53 - unchecked `db.CloseSafe()`

**Fix**:
```go
// BEFORE:
defer db.CloseSafe()

// AFTER:
defer func() {
    if err := db.CloseSafe(); err != nil {
        t.Logf("Failed to close database: %v", err)
    }
}()
```

---

### File: `backend/tests/expiry_test.go`

**Issues**: 7 unchecked errors (lines 55, 61, 79, 80, 89, 90)

**Fixes**:

Line 55:
```go
// BEFORE:
client.Post(srv.URL+"/api/auth/register", "application/json", bytesReader(`{"email":"e@x","password":"Password!1"}`))

// AFTER:
_, _ = client.Post(srv.URL+"/api/auth/register", "application/json", bytesReader(`{"email":"e@x","password":"Password!1"}`))
```

Line 61:
```go
// BEFORE:
json.Unmarshal(e.Data, &payload)

// AFTER:
_ = json.Unmarshal(e.Data, &payload)
```

Lines 79-80, 89-90:
```go
// BEFORE:
defer resp.Body.Close()
json.NewDecoder(resp.Body).Decode(v)

// AFTER:
defer func() { _ = resp.Body.Close() }()
_ = json.NewDecoder(resp.Body).Decode(v)
```

---

### File: `backend/tests/handlers_test.go`

**Issues**: 4 unchecked errors (lines 65, 117, 131, 217)

**Fixes**:

Line 65:
```go
// BEFORE:
os.RemoveAll(dir)

// AFTER:
_ = os.RemoveAll(dir)
```

Lines 117, 131:
```go
// BEFORE:
json.Unmarshal(env1.Data, &u1)

// AFTER:
_ = json.Unmarshal(env1.Data, &u1)
```

Line 217:
```go
// BEFORE:
respDash.Body.Close()

// AFTER:
_ = respDash.Body.Close()
```

---

### File: `backend/tests/ws_integration_test.go`

**Issues**: 5 unchecked errors (lines 47, 54, 57, 62, 128)

**Fixes**:

Lines 47, 54:
```go
// BEFORE:
defer conn.Close()
defer adminConn.Close()

// AFTER:
defer func() { _ = conn.Close() }()
defer func() { _ = adminConn.Close() }()
```

Lines 57, 62, 128:
```go
// BEFORE:
conn.SetReadDeadline(time.Now().Add(2 * time.Second))

// AFTER:
_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
```

**Ineffassign Issue** (Line 148):
```go
// BEFORE:
found = true

// AFTER (remove if not used after):
// Just remove the line or use it in condition
```

---

### File: `backend/tests/link_stats_test.go`

**Issue**: Line 31 - unused type `linkStatsResp`

**Fix**: Remove the unused type definition
```go
// REMOVE:
type linkStatsResp struct {
    // ...
}
```

---

## Task 3: Fix Gamification Service 🟡 MEDIUM PRIORITY

**File**: `backend/gamification/tally_service.go`

**Issue**: Line 328 - unchecked `recover()`

**Fix**:
```go
// BEFORE:
defer func() { recover() }()

// AFTER:
defer func() {
    if r := recover(); r != nil {
        log.Printf("Recovered from panic in goroutine: %v", r)
    }
}()
```

---

## Task 4: Fix Internal AMI Issues 🔴 CRITICAL

**File**: `internal/ami/connector.go`

**Issue**: Line 196 - unchecked `conn.Close()`

**Fix**:
```go
// BEFORE:
conn.Close()

// AFTER:
if err := conn.Close(); err != nil {
    log.Printf("Failed to close AMI connection: %v", err)
}
```

---

## Task 5: Fix AstDB Downloader Issues 🟠 HIGH PRIORITY

**File**: `internal/astdb/downloader.go`

**Issues**: 8 unchecked errors (lines 67, 68, 78, 94, 142, 347, 367)

**Fixes**:

Lines 67-68 (defer cleanup):
```go
// BEFORE:
defer tmpFile.Close()
defer os.Remove(tmpPath)

// AFTER:
defer func() { _ = tmpFile.Close() }()
defer func() { _ = os.Remove(tmpPath) }()
```

Line 78:
```go
// BEFORE:
defer resp.Body.Close()

// AFTER:
defer func() { _ = resp.Body.Close() }()
```

Line 94 (explicit close before rename):
```go
// BEFORE:
tmpFile.Close()

// AFTER:
if err := tmpFile.Close(); err != nil {
    return fmt.Errorf("failed to close temp file: %w", err)
}
```

Lines 142, 347, 367:
```go
// BEFORE:
defer file.Close()

// AFTER:
defer func() { _ = file.Close() }()
```

---

## Task 6: Fix WebSocket Issues 🟠 HIGH PRIORITY

**File**: `internal/web/ws.go`

**Issues**: 5 unchecked errors (lines 82, 107, 169, 171, 185)

**Fixes**:

Line 82:
```go
// BEFORE:
w.Write([]byte(`{"ok":false,"error":"websocket_upgrade_required"}`))

// AFTER:
_, _ = w.Write([]byte(`{"ok":false,"error":"websocket_upgrade_required"}`))
```

Line 107:
```go
// BEFORE:
defer func() { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock(); c.Close(websocket.StatusNormalClosure, "") }()

// AFTER:
defer func() {
    h.mu.Lock()
    delete(h.clients, c)
    h.mu.Unlock()
    _ = c.Close(websocket.StatusNormalClosure, "")
}()
```

Lines 169, 171, 185:
```go
// BEFORE:
go func(conn *websocket.Conn, p []byte) { conn.Write(context.Background(), websocket.MessageText, p) }(c, payload)

// AFTER:
go func(conn *websocket.Conn, p []byte) {
    _ = conn.Write(context.Background(), websocket.MessageText, p)
}(c, payload)
```

---

## Task 7: Fix Main.go Issues 🟠 HIGH PRIORITY

**File**: `main.go`

**Issues**: 2 unchecked `logger.Sync()` calls (lines 49, 583)

**Fixes**:

```go
// BEFORE:
defer logger.Sync()
defer zapLogger.Sync()

// AFTER:
defer func() { _ = logger.Sync() }()
defer func() { _ = zapLogger.Sync() }()
```

**Why**: `Sync()` can fail but it's called during shutdown, so we ignore errors.

---

## Task 8: Fix Staticcheck Issues 🟢 LOW PRIORITY

### File: `internal/ami/parsers.go`

**Issue**: Line 53 - QF1003: could use tagged switch

**Fix**:
```go
// BEFORE:
if key == "RPT_RXKEYED" {
    // ...
} else if key == "RPT_TXKEYED" {
    // ...
}

// AFTER:
switch key {
case "RPT_RXKEYED":
    // ...
case "RPT_TXKEYED":
    // ...
}
```

---

### File: `internal/core/state_test.go`

**Issue**: Line 61 - QF1001: could apply De Morgan's law

**Fix**:
```go
// BEFORE:
for !(sawAdd && sawRem) {

// AFTER:
for !sawAdd || !sawRem {
```

---

## Implementation Order

1. ✅ **Critical** - AMI connector (connection leaks)
2. ✅ **High** - Main.go, API layer, WebSocket, AstDB (resource leaks)
3. ✅ **Medium** - Test files, gamification service
4. ✅ **Low** - Staticcheck suggestions, unused code

---

## Testing & Verification

```bash
# Run golangci-lint to verify all issues fixed
golangci-lint run

# Run all tests to ensure no regressions
go test ./...

# Run with race detector
go test -race ./...

# Run specific linters
golangci-lint run --disable-all --enable=errcheck
golangci-lint run --disable-all --enable=staticcheck
```

---

## Summary Table

| Priority | Files | Issues | Est. Time |
|----------|-------|--------|-----------|
| 🔴 Critical | 1 | 1 | 5 min |
| 🟠 High | 5 | 19 | 20 min |
| 🟡 Medium | 5 | 18 | 15 min |
| 🟢 Low | 2 | 2 | 5 min |
| **Total** | **13** | **40** | **45 min** |

---

## Commit Strategy

**Option 1: Single Commit**
```
fix: resolve all golangci-lint issues (40 total)

- Fix 36 errcheck issues (unchecked error returns)
- Fix 2 staticcheck issues (code simplifications)
- Fix 1 ineffassign issue (ineffectual assignment)
- Remove 1 unused type

All issues resolved with proper error handling or explicit ignoring.
Tests passing, no regressions introduced.
```

**Option 2: Multiple Commits by Priority**
```
1. fix(critical): resolve resource leak in AMI connector
2. fix(high): add error handling for HTTP and file operations
3. fix(medium): improve error checking in test files
4. refactor: apply staticcheck suggestions and remove unused code
```

---

## Notes

- **Test files**: Use `_` to explicitly ignore errors that are not critical for test execution
- **Deferred operations**: Wrap in anonymous functions to avoid issues with error returns
- **Production code**: Log errors when they might indicate problems but shouldn't halt execution
- **Unused code**: Remove completely rather than comment out
