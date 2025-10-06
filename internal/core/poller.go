package core

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/ami"
)

type LinkPoller struct {
	Conn     *ami.Connector
	State    *StateManager
	Interval time.Duration
	stop     chan struct{}
}

func NewLinkPoller(conn *ami.Connector, sm *StateManager, interval time.Duration) *LinkPoller {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &LinkPoller{Conn: conn, State: sm, Interval: interval, stop: make(chan struct{})}
}

func (lp *LinkPoller) Start(ctx context.Context) {
	ticker := time.NewTicker(lp.Interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				lp.poll(ctx)
			case <-ctx.Done():
				return
			case <-lp.stop:
				return
			}
		}
	}()
}

func (lp *LinkPoller) Stop() { close(lp.stop) }

func (lp *LinkPoller) poll(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	resp, err := lp.Conn.SendCommand(cctx, "rpt stats")
	if err != nil {
		return
	}
	re := regexp.MustCompile(`\b(\d{3,7})\b`)
	found := map[int]struct{}{}
	for _, line := range resp.Raw {
		matches := re.FindAllString(line, -1)
		for _, m := range matches {
			if n, err := strconv.Atoi(m); err == nil {
				found[n] = struct{}{}
			}
		}
	}
	if len(found) == 0 {
		return
	}
	ids := make([]int, 0, len(found))
	for id := range found {
		ids = append(ids, id)
	}
	headers := map[string]string{"RPT_LINKS": intsToCSV(ids)}
	lp.State.apply(ami.Message{Headers: headers})
}

func intsToCSV(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	out := make([]byte, 0, len(ids)*8)
	first := true
	for _, id := range ids {
		if !first {
			out = append(out, ',')
		} else {
			first = false
		}
		out = append(out, []byte(strconv.Itoa(id))...)
	}
	return string(out)
}
