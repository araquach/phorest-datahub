package main

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"

	"staff-appraisals/internal/config"
	"staff-appraisals/internal/db"
	"staff-appraisals/internal/phorest"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger := cfg.Logger

	if err := os.MkdirAll(cfg.ExportDir, 0o755); err != nil {
		logger.Fatalf("create export dir %q: %v", cfg.ExportDir, err)
	}
	logger.Printf("📂 Using export dir: %s", cfg.ExportDir)

	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("DB connection failed: %v", err)
	}
	defer db.Close(gdb)

	if err := db.HealthCheck(gdb, 3*time.Second); err != nil {
		logger.Fatalf("DB health check failed: %v", err)
	}
	logger.Println("✅ Database connection healthy.")

	// Real Migrator
	if cfg.AutoMigrate {
		logger.Println("Running SQL migrations...")
		if err := db.RunMigrations(cfg.DatabaseURL, "migrations", logger); err != nil {
			logger.Fatalf("Database migration failed: %v", err)
		}
		logger.Println("✅ Database migrated successfully.")
	}

	for _, b := range cfg.Branches {
		logger.Printf("Branch: %s (ID: %s)\n", b.Name, b.BranchID)
	}

	logger.Println("✅ Startup complete. Ready to sync Phorest data.")

	runner := phorest.NewRunner(gdb, cfg, logger)

	// 🔹 ONE-OFF bootstrap from local CSVs, then seed watermarks.
	//     - First run: imports data/transactions + data/clients and calls BootstrapWatermarks()
	//     - Later runs: detects existing CSV watermarks and skips entirely.
	if err := runner.BootstrapFromCSVsIfNeeded(); err != nil {
		logger.Fatalf("CSV bootstrap failed: %v", err)
	}

	// 🔹 Ongoing API-based syncs (safe to run every time)
	if err := runner.SyncStaffFromAPI(); err != nil {
		logger.Fatalf("staff sync failed: %v", err)
	}

	if err := runner.SyncBranchesFromAPI(); err != nil {
		logger.Printf("branch sync ended with errors: %v", err)
	}

	if err := runner.SyncLatestReviewsFromAPI(10); err != nil {
		logger.Printf("reviews latest-N sync ended with errors: %v", err)
	}

	// 🔹 Incremental CSV export → download → import, guarded by env flag
	if os.Getenv("RUN_CLIENTS_INCREMENTAL") == "1" {
		logger.Println("🚀 Running incremental CLIENT_CSV & TRANSACTIONS_CSV sync…")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := runner.RunIncrementalClientsSync(ctx); err != nil {
			logger.Fatalf("CLIENT_CSV incremental sync failed: %v", err)
		}

		if err := runner.RunIncrementalTransactionsSync(ctx); err != nil {
			logger.Fatalf("TRANSACTIONS_CSV incremental sync failed: %v", err)
		}

		logger.Println("✅ Incremental CSV syncs complete.")
	}
}
