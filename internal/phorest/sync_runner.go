package phorest

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/config"
	"github.com/araquach/phorest-datahub/internal/repos"
	"gorm.io/gorm"
)

type Runner struct {
	DB     *gorm.DB
	Cfg    *config.Config
	Logger *log.Logger
	Export *ExportClient
}

// Accept cfg and store it so r.Cfg is valid everywhere
func NewRunner(db *gorm.DB, cfg *config.Config, lg *log.Logger) *Runner {
	export := NewExportClient(
		cfg.PhorestUsername,
		cfg.PhorestPassword,
		cfg.PhorestBusiness,
	)

	return &Runner{
		DB:     db,
		Cfg:    cfg,
		Logger: lg,
		Export: export,
	}
}

// ImportAllTransactionsCSVs loops through all .csv files in a directory and imports them.
func (r *Runner) ImportAllTransactionsCSVs(dir string) error {
	lg := r.Logger
	lg.Printf("🔍 Scanning directory: %s", dir)

	paths, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}
	if len(paths) == 0 {
		lg.Printf("⚠️  No CSV files found in %s", dir)
		return nil
	}

	lg.Printf("📂 Found %d CSV files", len(paths))
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		lg.Printf("──────────────────────────────────────────────")
		lg.Printf("🏁 Starting import for file: %s", name)

		if err := r.importSingleTransactionsCSV(path); err != nil {
			lg.Printf("❌ Failed import for %s: %v", name, err)
			continue
		}
		lg.Printf("✅ Completed import for %s", name)
	}

	lg.Printf("🎉 All CSV imports complete.")
	return nil
}

func (r *Runner) importSingleTransactionsCSV(csvPath string) error {
	lg := r.Logger

	batch, err := ParseTransactionsCSV(csvPath, lg)
	if err != nil {
		return err
	}
	lg.Printf("Importing CSV %s: %d transactions, %d items", csvPath, len(batch.Transactions), len(batch.Items))

	tx := r.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	tr := repos.NewTransactionsRepo(tx, lg)
	ir := repos.NewItemsRepo(tx, lg)

	if err := tr.UpsertBatch(batch.Transactions, 500); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ir.UpsertBatch(batch.Items, 500); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	lg.Printf("✅ CSV %s committed.", filepath.Base(csvPath))
	return nil
}

// ImportAllClientCSVs scans a dir and imports every .csv as clients
func (r *Runner) ImportAllClientCSVs(dir string) error {
	r.Logger.Printf("🔍 Scanning clients dir: %s", dir)
	paths, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}
	if len(paths) == 0 {
		r.Logger.Printf("⚠️  No client CSV files found in %s", dir)
		return nil
	}
	for _, p := range paths {
		if err := r.importSingleClientsCSV(p); err != nil {
			r.Logger.Printf("❌ Client import failed: %s: %v", p, err)
			continue
		}
	}
	r.Logger.Printf("✅ All client CSV imports complete.")
	return nil
}

func (r *Runner) importSingleClientsCSV(csvPath string) error {
	lg := r.Logger
	batch, err := ParseClientsCSV(csvPath, lg)
	if err != nil {
		return err
	}
	lg.Printf("Importing Clients CSV %s: %d clients", csvPath, len(batch.Clients))

	var maxTS *time.Time
	for i := range batch.Clients {
		if ts := batch.Clients[i].UpdatedAtPhorest; ts != nil {
			if maxTS == nil || ts.After(*maxTS) {
				maxTS = ts
			}
		}
	}
	if maxTS == nil {
		lg.Printf("⚠️  No UpdatedAtPhorest values in %s; skipping watermark update", csvPath)
	}

	tx := r.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	cr := repos.NewClientsRepo(tx, lg)
	if err := cr.UpsertBatch(batch.Clients, 1000); err != nil {
		_ = tx.Rollback()
		return err
	}

	if maxTS != nil {
		wr := repos.NewWatermarksRepo(tx, lg)
		// NOTE: branch = "ALL" for global clients CSV
		if err := wr.UpsertLastUpdated("clients_csv", "ALL", *maxTS); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("update clients_csv watermark: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	lg.Printf("✅ Clients CSV %s committed.", csvPath)
	return nil
}

// archiveCSVToSeed copies a CSV from srcPath into destDir
// so it becomes part of the “bootstrap” dataset.
func (r *Runner) archiveCSVToSeed(srcPath, destDir string) {
	lg := r.Logger

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		lg.Printf("⚠️  archiveCSVToSeed: unable to create dir %s: %v", destDir, err)
		return
	}

	dstPath := filepath.Join(destDir, filepath.Base(srcPath))

	// Don’t hard fail the sync if this fails – just log it.
	if err := copyFile(srcPath, dstPath); err != nil {
		lg.Printf("⚠️  archiveCSVToSeed: copy %s → %s failed: %v", srcPath, dstPath, err)
		return
	}

	lg.Printf("📦 Archived %s → %s (for future bootstrap)", srcPath, dstPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Make sure it’s flushed
	return out.Sync()
}

func (r *Runner) BootstrapFromCSVsIfNeeded() error {
	lg := r.Logger

	entities := []string{"clients_csv", "transactions_csv"}

	var watermarkCount int64
	if err := r.DB.Table("sync_watermarks").Count(&watermarkCount).Error; err != nil {
		return fmt.Errorf("check sync_watermarks: %w", err)
	}
	if watermarkCount > 0 {
		r.Logger.Printf("⏭  CSV bootstrap skipped: %d sync_watermarks already present", watermarkCount)
		return nil
	}

	var count int64
	if err := r.DB.Table("sync_watermarks").
		Where("entity IN ?", entities).
		Distinct("entity").
		Count(&count).Error; err != nil {
		return fmt.Errorf("check CSV watermarks: %w", err)
	}

	if count == int64(len(entities)) {
		lg.Printf("⏭  CSV bootstrap already done (%d/%d watermarks present); skipping initial CSV imports.", count, len(entities))
		return nil
	}

	lg.Println("📥 Running one-off CSV bootstrap (transactions + clients)...")

	if err := r.ImportAllTransactionsCSVs("data/transactions"); err != nil {
		return fmt.Errorf("bootstrap transactions CSVs: %w", err)
	}

	if err := r.ImportAllClientCSVs("data/clients"); err != nil {
		return fmt.Errorf("bootstrap clients CSVs: %w", err)
	}

	// Seeds: transactions_csv, clients_csv
	if err := r.BootstrapWatermarks(); err != nil {
		return fmt.Errorf("bootstrap watermarks: %w", err)
	}

	lg.Println("✅ CSV bootstrap completed and watermarks seeded.")
	return nil
}
