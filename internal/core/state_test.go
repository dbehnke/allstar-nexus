package core

import (
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/ami"
)

func TestStateApplyBasic(t *testing.T) {
	sm := NewStateManager()
	msg := ami.Message{Headers: map[string]string{"RPT_TXKEYED": "1", "RPT_RXKEYED": "0", "RPT_LINKS": "2000,3000"}}
	sm.apply(msg)
	snap := sm.Snapshot()
	if !snap.TxKeyed || snap.RxKeyed {
		t.Fatalf("unexpected keyed states: %+v", snap)
	}
	if len(snap.Links) != 2 {
		t.Fatalf("expected 2 links got %d", len(snap.Links))
	}
}

func TestFullyBootedUptime(t *testing.T) {
	sm := NewStateManager()
	bootEvt := ami.Message{Headers: map[string]string{"Event": "FullyBooted", "Uptime": "123", "LastReload": "123"}}
	sm.apply(bootEvt)
	snap := sm.Snapshot()
	if snap.UptimeSec != 123 || snap.LastReloadSec != 123 {
		t.Fatalf("uptime parse failed: %+v", snap)
	}
	if snap.BootedAt == nil {
		t.Fatalf("expected BootedAt to be set")
	}
	banner := ami.Message{Headers: map[string]string{"Asterisk Call Manager/Version": "11.0.0"}}
	sm.apply(banner)
	snap2 := sm.Snapshot()
	if snap2.UptimeSec != 0 || snap2.BootedAt != nil {
		t.Fatalf("expected reset after banner: %+v", snap2)
	}
}

func TestLinkAddRemoveDiffs(t *testing.T) {
	sm := NewStateManager()
	// consume channels non-blockingly
	// Add initial links
	sm.apply(ami.Message{Headers: map[string]string{"RPT_LINKS": "1000,2000"}})
	// Expect additions 1000,2000
	select {
	case added := <-sm.LinkUpdates():
		if len(added) != 2 {
			t.Fatalf("expected 2 added got %d", len(added))
		}
	default:
		t.Fatalf("expected additions emitted")
	}
	// Apply subset to cause removal of 2000 and add 3000
	sm.apply(ami.Message{Headers: map[string]string{"RPT_LINKS": "1000,3000"}})
	// Expect added 3000, removed 2000
	var sawAdd, sawRem bool
	timeout := time.After(500 * time.Millisecond)
	for !(sawAdd && sawRem) {
		select {
		case add := <-sm.LinkUpdates():
			if len(add) == 1 && add[0].Node == 3000 {
				sawAdd = true
			}
		case rem := <-sm.LinkRemovals():
			if len(rem) == 1 && rem[0] == 2000 {
				sawRem = true
			}
		case <-timeout:
			t.Fatalf("timed out waiting for add/rem events add=%v rem=%v", sawAdd, sawRem)
		}
	}
	snap := sm.Snapshot()
	if snap.Version == "" || snap.Heartbeat == 0 {
		t.Fatalf("expected version & heartbeat set: %+v", snap)
	}
}

func TestParseLinkIDs(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"", nil},
		{"2000,3000", []int{2000, 3000}},
		{"6,T588841,T590110,T586671,T58840,T550465,T586081", []int{588841, 590110, 586671, 58840, 550465, 586081}},
		{"6,588841TU,590110TU,586671TU,58840TU,550465TK,586081TU", []int{588841, 590110, 586671, 58840, 550465, 586081}},
		{"6,588841TU,588841TU", []int{588841}}, // dedupe
	}
	for i, c := range cases {
		got := parseLinkIDs(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("case %d length mismatch got=%v want=%v", i, got, c.want)
		}
		for j := range got {
			if got[j] != c.want[j] {
				t.Fatalf("case %d mismatch got=%v want=%v", i, got, c.want)
			}
		}
	}
}

func TestParseALinksAndTxAttribution(t *testing.T) {
	// simulate ALINKS event with one keyed (TK) and others TU
	sm := NewStateManager()
	alinks := "6,588841TU,590110TU,586671TU,58840TU,550465TK,586081TU"
	sm.apply(ami.Message{Headers: map[string]string{"RPT_ALINKS": alinks}})
	snap := sm.Snapshot()
	if len(snap.LinksDetailed) == 0 {
		t.Fatalf("expected links detailed populated")
	}
	var keyedCount int
	for _, li := range snap.LinksDetailed {
		if li.CurrentTx {
			keyedCount++
		}
	}
	if keyedCount != 1 {
		t.Fatalf("expected 1 keyed link got %d", keyedCount)
	}
}

func TestPerLinkTxStop(t *testing.T) {
	sm := NewStateManager()
	keyed := "3,123456TK,222333TU,444555TU"
	sm.apply(ami.Message{Headers: map[string]string{"RPT_ALINKS": keyed}})
	snap1 := sm.Snapshot()
	var found *LinkInfo
	for i := range snap1.LinksDetailed {
		if snap1.LinksDetailed[i].Node == 123456 {
			found = &snap1.LinksDetailed[i]
			break
		}
	}
	if found == nil || !found.CurrentTx {
		t.Fatalf("expected node 123456 keyed in first snapshot: %+v", snap1.LinksDetailed)
	}
	// Wait tiny duration to accumulate some tx time
	time.Sleep(20 * time.Millisecond)
	// Now send ALINKS without TK for 123456 so it should stop.
	unkeyed := "3,123456TU,222333TU,444555TU"
	sm.apply(ami.Message{Headers: map[string]string{"RPT_ALINKS": unkeyed}})
	snap2 := sm.Snapshot()
	var after *LinkInfo
	for i := range snap2.LinksDetailed {
		if snap2.LinksDetailed[i].Node == 123456 {
			after = &snap2.LinksDetailed[i]
			break
		}
	}
	if after == nil {
		t.Fatalf("missing node after unkey: %+v", snap2.LinksDetailed)
	}
	if after.CurrentTx {
		t.Fatalf("expected CurrentTx false after stop")
	}
	if after.LastTxStart == nil || after.LastTxEnd == nil {
		t.Fatalf("expected tx start/end timestamps recorded: %+v", after)
	}
}

func TestTextNodeCallsignAndDescription(t *testing.T) {
	sm := NewStateManager()
	// Test with a text-based node (callsign without numeric ID)
	alinks := "2,W1ABCTU,KF8STTK"
	sm.apply(ami.Message{Headers: map[string]string{"RPT_ALINKS": alinks}})
	
	snap := sm.Snapshot()
	if len(snap.LinksDetailed) != 2 {
		t.Fatalf("expected 2 links, got %d", len(snap.LinksDetailed))
	}
	
	// Both should have negative node IDs (hashed)
	for _, link := range snap.LinksDetailed {
		if link.Node >= 0 {
			t.Errorf("expected negative node ID for text node, got %d", link.Node)
		}
		if link.NodeCallsign == "" {
			t.Errorf("expected callsign to be set for node %d", link.Node)
		}
		if link.NodeDescription != "VOIP Client" {
			t.Errorf("expected 'VOIP Client' description, got '%s'", link.NodeDescription)
		}
	}
	
	// Check specific callsigns are preserved
	foundW1ABC := false
	foundKF8ST := false
	for _, link := range snap.LinksDetailed {
		if link.NodeCallsign == "W1ABC" {
			foundW1ABC = true
		}
		if link.NodeCallsign == "KF8ST" {
			foundKF8ST = true
		}
	}
	
	if !foundW1ABC {
		t.Error("expected to find W1ABC callsign")
	}
	if !foundKF8ST {
		t.Error("expected to find KF8ST callsign")
	}
}
