package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"github.com/dbehnke/allstar-nexus/internal/astdb"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func main() {
	// Command-line flags
	callsignsFile := flag.String("f", "", "Path to callsigns file (one callsign per line)")
	outputFile := flag.String("o", "", "Path to output whitelist file")
	ipAddress := flag.String("i", "", "IP address for the whitelist entries")
	dbPath := flag.String("db", "data/cgnat-whitelist.db", "Path to SQLite database (default: data/cgnat-whitelist.db)")
	astdbURL := flag.String("astdb-url", "http://allmondb.allstarlink.org/", "URL to download astdb from")
	flag.Parse()

	// Validate IP address
	if net.ParseIP(*ipAddress) == nil {
		log.Fatalf("Error: Invalid IP address: %s", *ipAddress)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("CGNAT Whitelist Generator starting",
		zap.String("callsigns_file", *callsignsFile),
		zap.String("output_file", *outputFile),
		zap.String("ip_address", *ipAddress),
		zap.String("db_path", *dbPath),
	)

	// Initialize database
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        *dbPath,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Database open error: %v", err)
	}

	// Set PRAGMA settings for optimized performance
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		logger.Warn("Failed to set journal_mode=WAL", zap.Error(err))
	}
	if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		logger.Warn("Failed to set synchronous=NORMAL", zap.Error(err))
	}

	// Auto-migrate NodeInfo table
	if err := gormDB.AutoMigrate(&models.NodeInfo{}); err != nil {
		log.Fatalf("Database migration error: %v", err)
	}

	// Initialize node info repository
	nodeInfoRepo := repository.NewNodeInfoRepository(gormDB)

	// Check if database is empty or needs update
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	count, err := nodeInfoRepo.GetCount(ctx)
	cancel()

	if err != nil || count == 0 {
		logger.Info("Database is empty or needs initialization. Downloading astdb...")

		// Download and import astdb to a temp location
		astdbPath, err := os.CreateTemp("", "cgnat-astdb-*.txt")
		if err != nil {
			log.Fatalf("Failed to create temp file for astdb: %v", err)
		}
		_ = astdbPath.Close()
		defer func() { _ = os.Remove(astdbPath.Name()) }()

		downloader := astdb.NewDownloader(*astdbURL, astdbPath.Name(), 24, logger)
		downloader.SetNodeInfoRepository(nodeInfoRepo)

		if err := downloader.DownloadAndImport(); err != nil {
			log.Fatalf("Failed to download and import astdb: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		count, _ = nodeInfoRepo.GetCount(ctx)
		cancel()
		logger.Info("astdb imported successfully", zap.Int64("node_count", count))
	} else {
		logger.Info("Using existing database", zap.Int64("node_count", count))
	}

	// Read callsigns from file
	callsigns, err := readCallsigns(*callsignsFile)
	if err != nil {
		log.Fatalf("Failed to read callsigns file: %v", err)
	}

	if len(callsigns) == 0 {
		log.Fatal("No callsigns found in file")
	}

	logger.Info("Callsigns loaded", zap.Int("count", len(callsigns)), zap.Strings("callsigns", callsigns))

	// Open output file
	outFile, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer func() { _ = outFile.Close() }()

	writer := bufio.NewWriter(outFile)
	defer func() { _ = writer.Flush() }()

	// Process each callsign
	totalEntries := 0
	for _, callsign := range callsigns {
		// Lookup nodes for this callsign
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		nodes, err := nodeInfoRepo.GetByCallsign(ctx, strings.ToUpper(callsign))
		cancel()

		if err != nil {
			logger.Error("Failed to lookup nodes", zap.String("callsign", callsign), zap.Error(err))
			continue
		}

		if len(nodes) == 0 {
			logger.Warn("No nodes found", zap.String("callsign", callsign))
			// Still write the comment line to show the callsign was processed
			_, _ = fmt.Fprintf(writer, ";%s Nodes (none found)\n", strings.ToUpper(callsign))
			continue
		}

	// Write comment line for this callsign
	_, _ = fmt.Fprintf(writer, ";%s Nodes\n", strings.ToUpper(callsign))

		// Write each node entry
		for _, node := range nodes {
			nodeIDStr := fmt.Sprintf("%-6d", node.NodeID)
			_, _ = fmt.Fprintf(writer, "%s = radio@%s/%d,NONE\n", nodeIDStr, *ipAddress, node.NodeID)
			totalEntries++
		}

		logger.Info("Processed callsign",
			zap.String("callsign", callsign),
			zap.Int("nodes_found", len(nodes)),
		)
	}

	if err := writer.Flush(); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	logger.Info("Whitelist generation completed",
		zap.String("output_file", *outputFile),
		zap.Int("callsigns_processed", len(callsigns)),
		zap.Int("total_entries", totalEntries),
	)

	fmt.Printf("\nWhitelist generation completed successfully!\n")
	fmt.Printf("  Callsigns processed: %d\n", len(callsigns))
	fmt.Printf("  Total entries: %d\n", totalEntries)
	fmt.Printf("  Output file: %s\n", *outputFile)
}

// readCallsigns reads callsigns from a file, one per line
// Skips empty lines and lines starting with # or ;
func readCallsigns(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var callsigns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Extract just the callsign (in case there are extra spaces or comments)
		parts := strings.Fields(line)
		if len(parts) > 0 {
			callsigns = append(callsigns, strings.ToUpper(parts[0]))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return callsigns, nil
}
