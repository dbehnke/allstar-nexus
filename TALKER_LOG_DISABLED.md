# Talker Log Feature - Temporarily Disabled

## Date: October 7, 2025

### Status: DISABLED

The talker log feature has been temporarily disabled in the UI to allow development to continue on other features. The backend functionality remains in place and can be re-enabled later.

### What Was Disabled

**Frontend (`frontend/src/views/Dashboard.vue`):**
- Commented out the entire Talker Log card/section in the dashboard grid
- Feature can be re-enabled by uncommenting lines 18-32

### Backend Remains Active

The backend talker log functionality is still running:
- Events are still being tracked in the ring buffer
- WebSocket still broadcasts `TALKER_EVENT` messages
- API endpoint `/api/talker-log` still returns snapshot
- Deduplication logic is active and working

### Issues to Resolve When Re-enabling

1. **Empty on Reload**: After server restart, talker log buffer is empty until first TX event occurs
   - Consider: Add persistence to save/restore events across restarts
   - Consider: Show a message like "Waiting for activity..." instead of blank

2. **Duplicate Detection Edge Cases**: While we fixed most duplicates, some edge cases may remain
   - Backend deduplication works but may need tuning for specific scenarios
   - Frontend filtering of node=0 events is working

3. **Duration Calculation**: Mostly working but may have edge cases
   - Uses LinkInfo timestamps when available
   - Falls back to state lookup for older events

### Quick Re-enable Steps

1. Uncomment lines 18-32 in `frontend/src/views/Dashboard.vue`
2. Rebuild frontend: `cd frontend && npm run build`
3. Restart server

### Backend Code Locations

If you need to modify backend behavior later:
- `internal/core/talker.go` - TalkerLog ring buffer implementation
- `internal/core/state.go` - Event emission (`emitTalker`, `emitTalkerFromLink`)
- `internal/web/ws.go` - WebSocket snapshot sending
- Debug logging available (commented out) in `state.go` lines ~377 and ~420

### What's Working

✅ Clickable node numbers (AllStarLink stats)
✅ Clickable callsigns (QRZ.com)
✅ Network Map view with bubble chart
✅ Active Links display
✅ Top Links stats
✅ Backend event tracking and deduplication

### To Revisit Later

- Complete talker log persistence across restarts
- Fine-tune duplicate detection
- Add better empty state messaging
- Consider showing "live" indicator when events are flowing
