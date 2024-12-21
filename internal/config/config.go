package config

import (
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Proxy     ProxyConfig     `mapstructure:"proxy"`
	Log       LogConfig       `mapstructure:"log"`
	DB        DBConfig        `mapstructure:"db"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type ProxyConfig struct {
	Target                string          `mapstructure:"target"`
	Timeout               time.Duration   `mapstructure:"timeout"`
	MaxIdleConns          int             `mapstructure:"max_idle_conns"`
	IdleConnTimeout       time.Duration   `mapstructure:"idle_conn_timeout"`
	TLSTimeout            time.Duration   `mapstructure:"tls_timeout"`
	ResponseHeaderTimeout time.Duration   `mapstructure:"response_header_timeout"`
	ExpectContinueTimeout time.Duration   `mapstructure:"expect_continue_timeout"`
	MaxConnsPerHost       int             `mapstructure:"max_conns_per_host"`
	RetryCount            int             `mapstructure:"retry_count"`
	RetryWaitTime         time.Duration   `mapstructure:"retry_wait_time"`
	Transform             TransformConfig `mapstructure:"transform"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type DBConfig struct {
	Type     string `mapstructure:"type"` // postgres, mongodb, etc.
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Pool     struct {
		MaxConns  int `mapstructure:"max_conns"`
		MinConns  int `mapstructure:"min_conns"`
		BatchSize int `mapstructure:"batch_size"`
	} `mapstructure:"pool"`
}

type RateLimitConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Global rate limits
	Global struct {
		Requests int           `mapstructure:"requests"` // Number of requests
		Window   time.Duration `mapstructure:"window"`   // Time window
		Burst    int           `mapstructure:"burst"`    // Burst size
	} `mapstructure:"global"`

	// Per IP rate limits
	PerIP struct {
		Enabled   bool          `mapstructure:"enabled"`
		Requests  int           `mapstructure:"requests"`
		Window    time.Duration `mapstructure:"window"`
		Burst     int           `mapstructure:"burst"`
		WhiteList []string      `mapstructure:"whitelist"` // IP whitelist
	} `mapstructure:"per_ip"`

	// Per Route rate limits
	Routes []RouteLimit `mapstructure:"routes"`

	// Token bucket configuration
	TokenBucket struct {
		Enabled      bool          `mapstructure:"enabled"`
		Capacity     int           `mapstructure:"capacity"`
		FillRate     float64       `mapstructure:"fill_rate"`
		FillInterval time.Duration `mapstructure:"fill_interval"`
	} `mapstructure:"token_bucket"`

	// Sliding window configuration
	SlidingWindow struct {
		Enabled  bool          `mapstructure:"enabled"`
		Size     time.Duration `mapstructure:"size"`
		Segments int           `mapstructure:"segments"`
	} `mapstructure:"sliding_window"`

	// Response configuration
	Response struct {
		StatusCode int    `mapstructure:"status_code"` // HTTP status code for rate limit exceeded
		Message    string `mapstructure:"message"`     // Error message
		Headers    bool   `mapstructure:"headers"`     // Include rate limit headers
	} `mapstructure:"response"`

	// Storage configuration for distributed rate limiting
	Storage struct {
		Type  string `mapstructure:"type"` // memory, redis, etc.
		Redis struct {
			Host     string        `mapstructure:"host"`
			Port     int           `mapstructure:"port"`
			Password string        `mapstructure:"password"`
			DB       int           `mapstructure:"db"`
			Timeout  time.Duration `mapstructure:"timeout"`
		} `mapstructure:"redis"`
	} `mapstructure:"storage"`
}

type RouteLimit struct {
	Path     string        `mapstructure:"path"`     // Route path (supports wildcards)
	Method   string        `mapstructure:"method"`   // HTTP method
	Requests int           `mapstructure:"requests"` // Number of requests
	Window   time.Duration `mapstructure:"window"`   // Time window
	Burst    int           `mapstructure:"burst"`    // Burst size
	Group    string        `mapstructure:"group"`    // Route group for shared limits
	Priority int           `mapstructure:"priority"` // Priority for overlapping rules
}

// TransformConfig represents the configuration for request/response transformations
type TransformConfig struct {
	// Directory containing transformation scripts
	ScriptsDir string `mapstructure:"scripts_dir"`
	// Service mappings
	Services map[string]ServiceTransform `mapstructure:"services"`
}

// ServiceTransform represents transformation rules for a specific service
type ServiceTransform struct {
	// Exact URL to match
	URL string `mapstructure:"url"`
	// Service name for script directory
	ServiceName string `mapstructure:"service_name"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Dir(configPath))
	viper.SetConfigFile(configPath)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
