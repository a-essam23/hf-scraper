// Path: internal/config/config.go
package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Scraper  ScraperConfig
	Watcher  WatcherConfig
}

// ServerConfig holds the API server settings.
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// DatabaseConfig holds the database connection settings.
type DatabaseConfig struct {
	URI              string `mapstructure:"uri"`
	Name             string `mapstructure:"name"`
	Collection       string `mapstructure:"collection"`
	StatusCollection string `mapstructure:"status_collection"`
}

// ScraperConfig holds settings for the Hugging Face API scraper.
type ScraperConfig struct {
	BaseURL           string `mapstructure:"base_url"`
	RequestsPerSecond int    `mapstructure:"requests_per_second"`
	BurstLimit        int    `mapstructure:"burst_limit"`
}

// WatcherConfig holds settings for the "Watch Mode" logic.
type WatcherConfig struct {
	IntervalMinutes int `mapstructure:"interval_minutes"`
}

// Load loads the configuration from file and environment variables.
func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("SERVER.PORT", "8080")
	viper.SetDefault("DATABASE.NAME", "hf-scraper")
	viper.SetDefault("DATABASE.COLLECTION", "models")
	viper.SetDefault("DATABASE.STATUS_COLLECTION", "_status")
	viper.SetDefault("SCRAPER.BASE_URL", "https://huggingface.co")
	viper.SetDefault("SCRAPER.REQUESTS_PER_SECOND", 5)
	viper.SetDefault("SCRAPER.BURST_LIMIT", 10)
	viper.SetDefault("WATCHER.INTERVAL_MINUTES", 5)

	// Load from config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err // Only return error if it's not a "file not found" error
		}
	}

	// Load from environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
