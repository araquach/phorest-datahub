package main

import (
	"context"
	"os"
	"time"

	"github.com/araquach/phorest-datahub/internal/repos"
	"github.com/araquach/phorest-datahub/internal/services"

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

	// -----------------------------
	// Sandbox-aware DB selection
	// -----------------------------
	dsn, err := cfg.ActiveDatabaseURL()
	if err != nil {
		logger.Fatalf("database URL resolution failed: %v", err)
	}

	if cfg.SandboxMode {
		logger.Println("🧪 SANDBOX MODE ENABLED — using SANDBOX_DATABASE_URL")
	} else {
		logger.Println("⚠️  NORMAL MODE — using DATABASE_URL")
	}

	// Optional safety guard (recommended on this feature branch)
	// if !cfg.SandboxMode && os.Getenv("ALLOW_NON_SANDBOX") != "1" {
	// 	logger.Fatalf("refusing to run without sandbox: set SANDBOX_MODE=1 or ALLOW_NON_SANDBOX=1")
	// }

	gdb, err := db.Open(dsn)
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
		if err := db.RunMigrations(dsn, "migrations", logger); err != nil {
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

	// Clients via API (live state)
	if os.Getenv("RUN_CLIENTS_API_INCREMENTAL") == "1" {
		logger.Println("🚀 Running incremental CLIENTS_API sync…")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := runner.RunIncrementalClientsAPISync(ctx); err != nil {
			logger.Fatalf("CLIENTS_API incremental sync failed: %v", err)
		}

		logger.Println("✅ Incremental CLIENTS_API sync complete.")
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

	if os.Getenv("RUN_STOCK_RECONCILE_DRY_RUN") == "1" {
		logger.Println("🧪 Running STOCK reconcile (dry-run)…")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		sqlDB, err := gdb.DB()
		if err != nil {
			logger.Fatalf("failed to get raw sql DB: %v", err)
		}

		repo := repos.StockReconcileRepo{DB: sqlDB}

		svc := services.StockReconcileService{
			Repo:       repo,
			PKBranchID: os.Getenv("SITE_2_BRANCH_ID"),
			DryRun:     true,

			// FIXED start date — no historical processing before this
			FromTS: time.Date(2026, time.January, 01, 0, 0, 0, 0, time.UTC),
			ToTS:   time.Now().UTC(),

			Limit: 500,

			TestBarcode: os.Getenv("STOCK_RECONCILE_TEST_BARCODE"),

			MaxPreview: 25,
			PrintJSON:  os.Getenv("STOCK_RECONCILE_PRINT_JSON") == "1",
		}

		if err := svc.Run(ctx); err != nil {
			logger.Fatalf("stock reconcile dry-run failed: %v", err)
		}

		logger.Println("✅ STOCK reconcile (dry-run) complete.")
	}

	if os.Getenv("RUN_STOCK_RECONCILE_LIVE") == "1" {
		logger.Println("🚨 Running STOCK reconcile (LIVE)…")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		sqlDB, err := gdb.DB()
		if err != nil {
			logger.Fatalf("failed to get raw sql DB: %v", err)
		}

		repo := repos.StockReconcileRepo{DB: sqlDB}

		adjuster := phorest.NewStockAdjuster(
			"https://api-gateway-eu.phorest.com/third-party-api-server",
			cfg.PhorestBusiness,
			cfg.PhorestUsername,
			cfg.PhorestPassword,
		)

		svc := services.StockReconcileService{
			Repo:       repo,
			Adjuster:   adjuster,
			PKBranchID: os.Getenv("SITE_2_BRANCH_ID"),
			DryRun:     false,

			FromTS: time.Date(2026, time.January, 16, 0, 0, 0, 0, time.UTC),
			ToTS:   time.Now().UTC(),

			Limit: 500,

			// Leave unset for full live:
			TestBarcode: os.Getenv("STOCK_RECONCILE_TEST_BARCODE"),

			MaxPreview: 25,
			PrintJSON:  os.Getenv("STOCK_RECONCILE_PRINT_JSON") == "1",
		}

		if err := svc.Run(ctx); err != nil {
			logger.Fatalf("stock reconcile LIVE failed: %v", err)
		}

		logger.Println("✅ STOCK reconcile (LIVE) complete.")
	}

	if os.Getenv("RUN_APPOINTMENTS_API_INCREMENTAL") == "1" {
		logger.Println("🚀 Running incremental APPOINTMENTS_API sync…")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		if err := runner.RunIncrementalAppointmentsAPISync(ctx); err != nil {
			logger.Fatalf("APPOINTMENTS_API incremental sync failed: %v", err)
		}

		logger.Println("✅ Incremental APPOINTMENTS_API sync complete.")
	}
}
