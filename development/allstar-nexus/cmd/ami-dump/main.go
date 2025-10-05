package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// simple heuristic: filter TCP packets on default AMI port (5038 unless overridden) and dump ASCII-ish payload lines.
func main() {
	file := flag.String("file", "ami-session.pcap", "pcap file path")
	port := flag.Int("port", 5038, "AMI TCP port to filter")
	outPath := flag.String("out", "ami-session.txt", "output sanitized text file")
	includeRaw := flag.Bool("raw", false, "include raw hex dump blocks as comments")
	flag.Parse()

	handle, err := pcap.OpenOffline(*file)
	if err != nil {
		fatalf("open pcap: %v", err)
	}
	defer handle.Close()

	fOut, err := os.Create(*outPath)
	if err != nil {
		fatalf("create output: %v", err)
	}
	defer fOut.Close()
	w := bufio.NewWriter(fOut)
	defer w.Flush()

	fmt.Fprintf(w, "# AMI session dump generated %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "# Source: %s  Filter port: %d\n\n", *file, *port)

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	secretRe := regexp.MustCompile(`(?i)^(Secret:\s*)(.*)$`)
	asciiLine := func(b []byte) bool {
		for _, c := range b {
			if c == '\r' || c == '\n' || (c >= 32 && c < 127) {
				continue
			}
			return false
		}
		return true
	}

	pktCount := 0
	frameCount := 0
	for pkt := range src.Packets() {
		pktCount++
		tcpLayer := pkt.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			continue
		}
		tcp, _ := tcpLayer.(*layers.TCP)
		if int(tcp.SrcPort) != *port && int(tcp.DstPort) != *port {
			continue
		}
		app := tcp.Payload
		if len(app) == 0 {
			continue
		}
		if !asciiLine(app) {
			continue
		}
		// Split into frames by blank lines; accumulate until double CRLF.
		// Since we only have payload segment boundaries, just output the text with markers.
		rdr := bufio.NewReader(bytes.NewReader(app))
		var block []string
		flush := func() {
			if len(block) == 0 {
				return
			}
			frameCount++
			fmt.Fprintf(w, "--- FRAME %d (packet=%d) ---\n", frameCount, pktCount)
			for _, ln := range block {
				ln = strings.TrimRight(ln, "\r\n")
				if secretRe.MatchString(ln) {
					ln = secretRe.ReplaceAllString(ln, "$1REDACTED")
				}
				fmt.Fprintln(w, ln)
			}
			fmt.Fprintln(w)
			block = block[:0]
		}
		for {
			line, err := rdr.ReadString('\n')
			if len(line) > 0 {
				trimmed := strings.TrimRight(line, "\r\n")
				if trimmed == "" { // boundary
					flush()
				} else {
					block = append(block, line)
				}
			}
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(w, "# read err: %v\n", err)
				}
				flush()
				break
			}
		}
		if *includeRaw {
			fmt.Fprintf(w, "# RAWHEX %d %d bytes: %x\n\n", pktCount, len(app), app)
		}
	}
	fmt.Fprintf(w, "# Done. packets=%d frames=%d\n", pktCount, frameCount)
}

func fatalf(f string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(1)
}
