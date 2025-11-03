package admin

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"gorm.io/gorm"
)

// SystemMonitor collects and provides system health metrics
type SystemMonitor struct {
	db        *gorm.DB
	startTime time.Time
}

// NewSystemMonitor creates a new system monitor instance
func NewSystemMonitor(db *gorm.DB) *SystemMonitor {
	return &SystemMonitor{
		db:        db,
		startTime: time.Now(),
	}
}

// SystemMetrics represents current system health metrics
type SystemMetrics struct {
	Timestamp       time.Time              `json:"timestamp"`
	Uptime          string                 `json:"uptime"`
	UptimeSeconds   int64                  `json:"uptime_seconds"`
	CPU             CPUMetrics             `json:"cpu"`
	Memory          MemoryMetrics          `json:"memory"`
	Database        DatabaseMetrics        `json:"database"`
	Goroutines      int                    `json:"goroutines"`
	ConnectionCount int                    `json:"connection_count,omitempty"`
}

// CPUMetrics represents CPU usage information
type CPUMetrics struct {
	NumCPU      int     `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	UsagePercent float64 `json:"usage_percent,omitempty"` // Optional, requires external monitoring
}

// MemoryMetrics represents memory usage information
type MemoryMetrics struct {
	Alloc        uint64  `json:"alloc"`         // Currently allocated memory in bytes
	TotalAlloc   uint64  `json:"total_alloc"`   // Total allocated memory in bytes
	Sys          uint64  `json:"sys"`           // Total memory from system in bytes
	HeapAlloc    uint64  `json:"heap_alloc"`    // Allocated heap objects in bytes
	HeapSys      uint64  `json:"heap_sys"`      // Total heap memory from system
	HeapInuse    uint64  `json:"heap_inuse"`    // Heap memory in use
	AllocHuman   string  `json:"alloc_human"`   // Human readable allocated memory
	SysHuman     string  `json:"sys_human"`     // Human readable system memory
	UsagePercent float64 `json:"usage_percent"` // Percentage of heap in use
}

// DatabaseMetrics represents database statistics
type DatabaseMetrics struct {
	Size         int64  `json:"size"`          // Database file size in bytes
	SizeHuman    string `json:"size_human"`    // Human readable size
	TableCount   int    `json:"table_count"`   // Number of tables
	UserCount    int64  `json:"user_count"`    // Total users
	AuditLogCount int64  `json:"audit_log_count"` // Total audit logs
}

// GetMetrics collects and returns current system metrics
func (sm *SystemMonitor) GetMetrics(ctx context.Context) (*SystemMetrics, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(sm.startTime)
	metrics := &SystemMetrics{
		Timestamp:     time.Now(),
		Uptime:        formatDuration(uptime),
		UptimeSeconds: int64(uptime.Seconds()),
		CPU: CPUMetrics{
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
		Memory: MemoryMetrics{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			HeapAlloc:  m.HeapAlloc,
			HeapSys:    m.HeapSys,
			HeapInuse:  m.HeapInuse,
			AllocHuman: FormatBytes(int64(m.Alloc)),
			SysHuman:   FormatBytes(int64(m.Sys)),
		},
		Goroutines: runtime.NumGoroutine(),
	}

	// Calculate memory usage percentage
	if m.HeapSys > 0 {
		metrics.Memory.UsagePercent = float64(m.HeapInuse) / float64(m.HeapSys) * 100
	}

	// Get database metrics
	dbMetrics, err := sm.getDatabaseMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database metrics: %w", err)
	}
	metrics.Database = *dbMetrics

	return metrics, nil
}

// getDatabaseMetrics collects database-specific metrics
func (sm *SystemMonitor) getDatabaseMetrics(ctx context.Context) (*DatabaseMetrics, error) {
	metrics := &DatabaseMetrics{}

	// Get user count
	if err := sm.db.WithContext(ctx).Table("users").Count(&metrics.UserCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Get audit log count
	if err := sm.db.WithContext(ctx).Table("audit_logs").Count(&metrics.AuditLogCount).Error; err != nil {
		// Audit logs table might not exist yet, that's okay
		metrics.AuditLogCount = 0
	}

	// Get database size (SQLite specific)
	sqlDB, err := sm.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	var pageCount, pageSize int64
	row := sqlDB.QueryRowContext(ctx, "PRAGMA page_count")
	if err := row.Scan(&pageCount); err == nil {
		row = sqlDB.QueryRowContext(ctx, "PRAGMA page_size")
		if err := row.Scan(&pageSize); err == nil {
			metrics.Size = pageCount * pageSize
			metrics.SizeHuman = FormatBytes(metrics.Size)
		}
	}

	// Get table count
	rows, err := sqlDB.QueryContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table'")
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&metrics.TableCount)
		}
	}

	return metrics, nil
}

// HealthStatus represents overall system health
type HealthStatus struct {
	Status   string            `json:"status"`   // "healthy", "degraded", "unhealthy"
	Checks   map[string]Check  `json:"checks"`   // Individual health checks
	Metrics  *SystemMetrics    `json:"metrics"`  // Current metrics
	Timestamp time.Time        `json:"timestamp"`
}

// Check represents an individual health check result
type Check struct {
	Status  string `json:"status"`  // "pass", "warn", "fail"
	Message string `json:"message,omitempty"`
}

// GetHealth performs health checks and returns overall status
func (sm *SystemMonitor) GetHealth(ctx context.Context) (*HealthStatus, error) {
	metrics, err := sm.GetMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	health := &HealthStatus{
		Status:    "healthy",
		Checks:    make(map[string]Check),
		Metrics:   metrics,
		Timestamp: time.Now(),
	}

	// Check database connectivity
	sqlDB, err := sm.db.DB()
	if err != nil {
		health.Checks["database"] = Check{Status: "fail", Message: "Failed to get database connection"}
		health.Status = "unhealthy"
	} else if err := sqlDB.PingContext(ctx); err != nil {
		health.Checks["database"] = Check{Status: "fail", Message: "Database ping failed"}
		health.Status = "unhealthy"
	} else {
		health.Checks["database"] = Check{Status: "pass"}
	}

	// Check memory usage
	if metrics.Memory.UsagePercent > 90 {
		health.Checks["memory"] = Check{Status: "fail", Message: "Memory usage critical (>90%)"}
		health.Status = "unhealthy"
	} else if metrics.Memory.UsagePercent > 75 {
		health.Checks["memory"] = Check{Status: "warn", Message: "Memory usage high (>75%)"}
		if health.Status == "healthy" {
			health.Status = "degraded"
		}
	} else {
		health.Checks["memory"] = Check{Status: "pass"}
	}

	// Check goroutine count
	if metrics.Goroutines > 10000 {
		health.Checks["goroutines"] = Check{Status: "warn", Message: "High goroutine count (>10000)"}
		if health.Status == "healthy" {
			health.Status = "degraded"
		}
	} else {
		health.Checks["goroutines"] = Check{Status: "pass"}
	}

	return health, nil
}

// MetricsHistory represents historical metrics data points
type MetricsHistory struct {
	Points []MetricsDataPoint `json:"points"`
	Start  time.Time          `json:"start"`
	End    time.Time          `json:"end"`
}

// MetricsDataPoint represents a single metrics snapshot
type MetricsDataPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	MemoryAlloc    uint64    `json:"memory_alloc"`
	MemoryUsage    float64   `json:"memory_usage_percent"`
	Goroutines     int       `json:"goroutines"`
	DatabaseSize   int64     `json:"database_size"`
}

// formatDuration formats a duration into human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
