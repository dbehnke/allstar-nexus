# Pure Go SQLite Migration

This document explains the migration from CGO-based SQLite to pure Go SQLite.

## Problem

The application was using `mattn/go-sqlite3` through GORM's default SQLite driver, which requires CGO. This made cross-compilation difficult and added complexity to the build process.

## Solution

We migrated to `modernc.org/sqlite`, a pure Go implementation of SQLite that doesn't require CGO.

### Changes Made

1. **Updated GORM Configuration**: All `gorm.Open()` calls now explicitly use `modernc.org/sqlite`:
   ```go
   gorm.Open(sqlite.New(sqlite.Config{
       DriverName: "sqlite",  // Uses modernc.org/sqlite instead of mattn/go-sqlite3
       DSN:        dbPath,
   }), &gorm.Config{})
   ```

2. **Converted database.go**: The `backend/database/database.go` package now uses GORM instead of direct SQL queries.

3. **Updated main.go**: Removed the old `database.Open()` call and use GORM directly with modernc.org/sqlite.

4. **Updated all tests**: All test files import `_ "modernc.org/sqlite"` and use the explicit driver configuration.

## Benefits

- ✅ **No CGO Required**: Application now builds and runs with `CGO_ENABLED=0`
- ✅ **Easier Cross-Compilation**: Can easily cross-compile for different platforms without CGO dependencies
- ✅ **Pure Go**: Entire stack is now pure Go
- ✅ **Same Functionality**: All tests pass, no behavioral changes

## Verification

Run tests without CGO to verify:
```bash
CGO_ENABLED=0 go test ./backend/...
```

Build without CGO:
```bash
CGO_ENABLED=0 go build .
```

Cross-compile for different platforms (examples):
```bash
# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build .

# macOS ARM64 (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build .

# Windows AMD64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build .
```

All of these now work without any CGO dependencies!

## Notes

- `mattn/go-sqlite3` is still listed as an indirect dependency in go.mod because `gorm.io/driver/sqlite` depends on it. However, it's not used at runtime since we explicitly specify `DriverName: "sqlite"`.
- SQLite version 3.46.0 is provided by modernc.org/sqlite.
- All database operations (including PRAGMA settings) continue to work as before.
