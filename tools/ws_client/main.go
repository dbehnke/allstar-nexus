package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "server address")
	path := flag.String("path", "/ws", "websocket path")
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *addr, Path: *path}
	q := u.Query()
	q.Set("token", "")
	u.RawQuery = q.Encode()

	log.Printf("connecting to %s", u.String())
	dialer := websocket.DefaultDialer
	c, resp, err := dialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil {
			log.Fatalf("dial error: %v (status=%s)", err, resp.Status)
		}
		log.Fatalf("dial error: %v", err)
	}
	defer c.Close()

	// Read up to 10 messages (or until read error / timeout).
	c.SetReadDeadline(time.Now().Add(15 * time.Second))
	for i := 0; i < 10; i++ {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			os.Exit(1)
		}
		fmt.Printf("msg[%d]=%s\n", i, string(msg))
	}
}
