package main

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/araquach/phorest-datahub/internal/config"
	"github.com/araquach/phorest-datahub/internal/db"
	"github.com/araquach/phorest-datahub/internal/phorest"
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

	// ---------- BOOTSTRAP PHASE ----------

	// Clients + transactions from local CSVs (only on fresh DB)
	if err := runner.BootstrapFromCSVsIfNeeded(); err != nil {
		logger.Fatalf("CSV bootstrap failed: %v", err)
	}

	// Reviews from local CSV backups (only on fresh DB)
	if err := runner.BootstrapReviewsFromCSVsIfNeeded(); err != nil {
		logger.Fatalf("Reviews CSV bootstrap failed: %v", err)
	}

	// ---------- ONGOING “EVERY RUN” API SYNC ----------

	if err := runner.SyncStaffFromAPI(); err != nil {
		logger.Fatalf("staff sync failed: %v", err)
	}

	if err := runner.SyncBranchesFromAPI(); err != nil {
		logger.Printf("branch sync ended with errors: %v", err)
	}

	// ---------- INCREMENTAL CSV + REVIEWS (ENV-GUARDED) ----------

	// Clients incremental (CLIENT_CSV)
	if os.Getenv("RUN_CLIENTS_INCREMENTAL") == "1" {
		logger.Println("🚀 Running incremental CLIENT_CSV sync…")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := runner.RunIncrementalClientsSync(ctx); err != nil {
			logger.Fatalf("CLIENT_CSV incremental sync failed: %v", err)
		}

		logger.Println("✅ Incremental CLIENT_CSV sync complete.")
	}

	// Transactions incremental (TRANSACTIONS_CSV)
	if os.Getenv("RUN_TRANSACTIONS_INCREMENTAL") == "1" {
		logger.Println("🚀 Running incremental TRANSACTIONS_CSV sync…")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := runner.RunIncrementalTransactionsSync(ctx); err != nil {
			logger.Fatalf("TRANSACTIONS_CSV incremental sync failed: %v", err)
		}

		logger.Println("✅ Incremental TRANSACTIONS_CSV sync complete.")
	}

	// Reviews incremental
	if os.Getenv("RUN_REVIEWS_INCREMENTAL") == "1" {
		logger.Println("🚀 Running incremental REVIEWS sync…")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := runner.RunIncrementalReviewsSync(ctx); err != nil {
			logger.Fatalf("REVIEWS incremental sync failed: %v", err)
		}

		logger.Println("✅ Incremental REVIEWS sync complete.")
	}

	if os.Getenv("RUN_PRODUCTS_SYNC") == "1" {
		logger.Println("🚀 Running PRODUCTS sync…")
		if err := runner.SyncProductsFromAPI(); err != nil {
			logger.Fatalf("products sync failed: %v", err)
		}
		logger.Println("✅ PRODUCTS sync complete.")
	}
}
