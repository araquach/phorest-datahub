package phorest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
	"github.com/araquach/phorest-datahub/internal/repos"
)

// SyncProductsFromAPI pulls products/stock for all configured branches
// and writes to ph_products, ph_product_stock, and ph_product_stock_history.
func (r *Runner) SyncProductsFromAPI() error {
	lg := r.Logger

	lg.Println("🚿 Starting PRODUCTS sync from Phorest API…")

	pc := NewProductsClient(
		"",
		r.Cfg.PhorestBusiness,
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
	)

	productRepo := repos.NewPhProductRepo(r.DB)
	stockRepo := repos.NewPhProductStockRepo(r.DB)
	watermarks := repos.NewWatermarksRepo(r.DB, r.Logger)

	ctx := context.Background()

	productType := os.Getenv("PRODUCT_TYPE_FILTER") // "" = all
	if productType == "" {
		lg.Println("   PRODUCT_TYPE_FILTER not set → syncing ALL product types")
	} else {
		lg.Printf("   PRODUCT_TYPE_FILTER=%s → syncing only this type", productType)
	}

	for _, b := range r.Cfg.Branches {
		lg.Printf("➡️  Syncing PRODUCTS for branch %s (ID: %s)", b.Name, b.BranchID)

		wm, err := watermarks.GetLastUpdated("products_api", b.BranchID)
		if err != nil {
			return fmt.Errorf("get products watermark for %s: %w", b.BranchID, err)
		}

		var updatedAfter, updatedBefore *time.Time

		if wm != nil {
			after := wm.UTC().Add(1 * time.Second)
			now := time.Now().UTC()
			updatedAfter = &after
			updatedBefore = &now
			lg.Printf("   Using incremental window updatedAfter=%s, updatedBefore=%s",
				after.Format(time.RFC3339), now.Format(time.RFC3339))
		} else {
			lg.Println("   No products watermark → full sync (no date filters)")
		}

		maxUpdatedAt, err := r.syncProductsForBranch(
			ctx,
			pc,
			productRepo,
			stockRepo,
			b.BranchID,
			productType,
			updatedAfter,
			updatedBefore,
		)
		if err != nil {
			return fmt.Errorf("sync products for branch %s (%s): %w", b.Name, b.BranchID, err)
		}

		if maxUpdatedAt != nil {
			if err := watermarks.UpsertLastUpdated("products_api", b.BranchID, *maxUpdatedAt); err != nil {
				return fmt.Errorf("update products_api watermark for %s: %w", b.BranchID, err)
			}
		}
	}

	lg.Println("✅ PRODUCTS sync complete for all branches.")
	return nil
}

// syncProductsForBranch does the paging + upserts for a single branch.
// It returns the maximum UpdatedAt timestamp from Phorest for this run.
func (r *Runner) syncProductsForBranch(
	ctx context.Context,
	pc *ProductsClient,
	productRepo *repos.PhProductRepo,
	stockRepo *repos.PhProductStockRepo,
	branchID string,
	productType string,
	updatedAfter *time.Time,
	updatedBefore *time.Time,
) (*time.Time, error) {
	page := 0
	size := 100

	var maxUpdatedAt *time.Time

	for {
		resp, err := pc.ListProducts(ctx, ListProductsOptions{
			BranchID:      branchID,
			ProductType:   productType,
			UpdatedAfter:  updatedAfter,
			UpdatedBefore: updatedBefore,
			Page:          page,
			Size:          size,
		})
		if err != nil {
			return nil, err
		}

		if len(resp.Embedded.Products) == 0 {
			break
		}

		for _, pp := range resp.Embedded.Products {
			if err := r.processProductRecord(ctx, productRepo, stockRepo, branchID, pp); err != nil {
				return nil, err
			}

			// Track max UpdatedAt from Phorest
			if maxUpdatedAt == nil || pp.UpdatedAt.After(*maxUpdatedAt) {
				t := pp.UpdatedAt
				maxUpdatedAt = &t
			}
		}

		page++
		if resp.Page.TotalPages > 0 && page >= resp.Page.TotalPages {
			break
		}
	}

	return maxUpdatedAt, nil
}

// processProductRecord maps a single PhorestProduct into:
//   - ph_products (master)
//   - ph_product_stock (current state per branch)
//   - ph_product_stock_history (time series when quantity changes)
func (r *Runner) processProductRecord(
	ctx context.Context,
	productRepo *repos.PhProductRepo,
	stockRepo *repos.PhProductStockRepo,
	branchID string,
	pp PhorestProduct,
) error {
	// --- Upsert product master ---
	product := &models.PhProduct{
		ID:       pp.ProductID,
		Name:     pp.Name,
		Archived: pp.Archived,
	}

	if pp.ParentProductID != "" {
		product.ParentID = &pp.ParentProductID
	}
	if pp.BrandID != "" {
		product.BrandID = &pp.BrandID
	}
	if pp.BrandName != "" {
		product.BrandName = &pp.BrandName
	}
	if pp.CategoryID != "" {
		product.CategoryID = &pp.CategoryID
	}
	if pp.CategoryName != "" {
		product.CategoryName = &pp.CategoryName
	}
	if pp.Code != "" {
		product.Code = &pp.Code
	}
	if pp.Type != "" {
		product.TypeRaw = &pp.Type
	}
	if pp.MeasurementQuantity != 0 {
		v := pp.MeasurementQuantity
		product.MeasurementQty = &v
	}
	if pp.MeasurementUnit != "" {
		product.MeasurementUnit = &pp.MeasurementUnit
	}

	product.CreatedAtPh = &pp.CreatedAt
	product.UpdatedAtPh = &pp.UpdatedAt

	if err := productRepo.Upsert(ctx, product); err != nil {
		return err
	}

	// --- Upsert current stock row ---

	newStock := &models.PhProductStock{
		ProductID:   pp.ProductID,
		BranchID:    branchID,
		Archived:    pp.Archived,
		CreatedAtPh: &pp.CreatedAt,
		UpdatedAtPh: &pp.UpdatedAt,
	}

	if pp.Price != 0 {
		v := pp.Price
		newStock.Price = &v
	}
	if pp.MinQuantity != 0 {
		v := pp.MinQuantity
		newStock.MinQuantity = &v
	}
	if pp.MaxQuantity != 0 {
		v := pp.MaxQuantity
		newStock.MaxQuantity = &v
	}
	// quantity can be 0 and that’s meaningful, so always store it
	{
		v := pp.QuantityInStock
		newStock.QuantityInStock = &v
	}
	if pp.ReorderCount != 0 {
		v := pp.ReorderCount
		newStock.ReorderCount = &v
	}
	if pp.ReorderCost != 0 {
		v := pp.ReorderCost
		newStock.ReorderCost = &v
	}

	existing, err := stockRepo.GetByProductAndBranch(ctx, pp.ProductID, branchID)
	if err != nil {
		return err
	}

	// Upsert current state
	if err := stockRepo.Upsert(ctx, newStock); err != nil {
		return err
	}

	// --- History logging when quantity changes (or first time) ---

	shouldLogHistory := false
	var oldQty *float64

	if existing == nil {
		shouldLogHistory = true
	} else {
		oldQty = existing.QuantityInStock
		newQty := newStock.QuantityInStock
		if (oldQty == nil && newQty != nil) ||
			(oldQty != nil && newQty == nil) ||
			(oldQty != nil && newQty != nil && *oldQty != *newQty) {
			shouldLogHistory = true
		}
	}

	if shouldLogHistory {
		h := &models.PhProductStockHistory{
			ProductID:       pp.ProductID,
			BranchID:        branchID,
			QuantityInStock: newStock.QuantityInStock,
			Source:          "sync",
			SnapshotTime:    time.Now(),
		}
		if newStock.Price != nil {
			v := *newStock.Price
			h.Price = &v
		}

		if err := stockRepo.InsertHistory(ctx, h); err != nil {
			return err
		}
	}

	return nil
}
