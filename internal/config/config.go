package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/araquach/phorest-datahub/internal/util"
)

// BranchConfig holds the name + ID of each branch.
type BranchConfig struct {
	Name     string
	BranchID string
}

// Config centralises all environment and runtime configuration.
type Config struct {
	Logger             *log.Logger
	DatabaseURL        string
	SandboxDatabaseURL string
	SandboxMode        bool
	PhorestUsername    string
	PhorestPassword    string
	PhorestBusiness    string

	Branches  []BranchConfig
	ExportDir string

	AutoMigrate bool
}

// Load builds the Config struct, validating critical env vars.
func Load() *Config {
	logger := util.NewLogger()
	logger.Println("Loading environment configuration...")

	cfg := &Config{
		Logger:             logger,
		DatabaseURL:        getEnvOrFail(logger, "DATABASE_URL"),
		SandboxDatabaseURL: os.Getenv("SANDBOX_DATABASE_URL"),
		SandboxMode:        parseBoolEnv(os.Getenv("SANDBOX_MODE")),
		PhorestUsername:    getEnvOrFail(logger, "PHOREST_USERNAME"),
		PhorestPassword:    getEnvOrFail(logger, "PHOREST_PASSWORD"),
		PhorestBusiness:    getEnvOrFail(logger, "PHOREST_BUSINESS"),
		AutoMigrate:        os.Getenv("AUTO_MIGRATE") == "1",
		ExportDir:          getEnvOrDefault("EXPORT_DIR", "data/exports"),
		Branches: []BranchConfig{
			{
				Name:     getEnvOrDefault("SITE_1_NAME", "Jakata"),
				BranchID: getEnvOrFail(logger, "SITE_1_BRANCH_ID"),
			},
			{
				Name:     getEnvOrDefault("SITE_2_NAME", "PK"),
				BranchID: getEnvOrFail(logger, "SITE_2_BRANCH_ID"),
			},
			{
				Name:     getEnvOrDefault("SITE_3_NAME", "Base"),
				BranchID: getEnvOrFail(logger, "SITE_3_BRANCH_ID"),
			},
		},
	}

	logger.Printf("✅ Loaded config for %d branches\n", len(cfg.Branches))
	logger.Printf("📁 ExportDir: %s", cfg.ExportDir)
	return cfg
}

func (c *Config) ActiveDatabaseURL() (string, error) {
	if c.SandboxMode {
		if strings.TrimSpace(c.SandboxDatabaseURL) == "" {
			return "", fmt.Errorf("SANDBOX_MODE is enabled but SANDBOX_DATABASE_URL is empty")
		}
		return c.SandboxDatabaseURL, nil
	}

	if strings.TrimSpace(c.DatabaseURL) == "" {
		return "", fmt.Errorf("DATABASE_URL is empty")
	}
	return c.DatabaseURL, nil
}

func getEnvOrFail(logger *log.Logger, key string) string {
	val := os.Getenv(key)
	if val == "" {
		logger.Fatalf("❌ Environment variable %s is required but not set", key)
	}
	return val
}

func getEnvOrDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func parseBoolEnv(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}
