package config

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type config interface {
	Validate() error
}

var (
	_ = config(Storage{})
	_ = config(Provider{})
	_ = config(HTTPServer{})
	_ = config(Worker{})
)

type Config struct {
	Env        string `yaml:"env" env-default:"development"`
	Storage    `yaml:"storage"`
	Provider   `yaml:"provider"`
	HTTPServer `yaml:"http_server"`
	Worker     `yaml:"worker"`
}

func (cfg *Config) Validate() error {
	if cfg.Env != "development" && cfg.Env != "production" {
		return fmt.Errorf("env must be development or production")
	}
	if err := cfg.Storage.Validate(); err != nil {
		return fmt.Errorf("validate storage: %w", err)
	}
	if err := cfg.Provider.Validate(); err != nil {
		return fmt.Errorf("validate provider: %w", err)
	}
	if err := cfg.HTTPServer.Validate(); err != nil {
		return fmt.Errorf("validate http_server: %w", err)
	}
	if err := cfg.Worker.Validate(); err != nil {
		return fmt.Errorf("validate worker: %w", err)
	}
	return nil
}

type Storage struct {
	Host     string        `yaml:"host" env-default:"localhost"`
	Port     int           `yaml:"port" env-default:"5432"`
	User     string        `yaml:"user" env-default:"postgres"`
	Password string        `yaml:"password" env-default:"postgres"`
	Database string        `yaml:"database" env-default:"postgres"`
	Timeout  time.Duration `yaml:"timeout" env-default:"5s"`
}

func (s Storage) Validate() error {
	if s.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, have: %s", s.Timeout)
	}
	return nil
}

func (s Storage) String() string {
	return (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(s.User, s.Password),
		Host:   net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:   "/" + s.Database,
	}).String()
}

type Provider struct {
	Timeout        time.Duration `yaml:"timeout" env-default:"5s"`
	MaxAttempts    int           `yaml:"max_attempts" env-default:"3"`
	InitialBackoff time.Duration `yaml:"initial_backoff" env-default:"200ms"`
}

func (cfg Provider) Validate() error {
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, have: %s", cfg.Timeout)
	}
	if cfg.MaxAttempts <= 0 {
		return fmt.Errorf("max_attempts must be positive, have: %d", cfg.MaxAttempts)
	}
	if cfg.InitialBackoff < 0 {
		return fmt.Errorf("initial_backoff must not be negative, have: %s", cfg.InitialBackoff)
	}
	return nil
}

type HTTPServer struct {
	Address      string        `yaml:"address" env-default:":8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env-default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

func (cfg HTTPServer) Validate() error {
	if cfg.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive, have: %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive, have: %s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive, have: %s", cfg.IdleTimeout)
	}
	return nil
}

type Worker struct {
	BufferSize int `yaml:"buffer_size" env-default:"100"`
}

func (cfg Worker) Validate() error {
	if cfg.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be greater than 0")
	}
	return nil
}

var cfgPathEnv = "CONFIG_PATH"

func MustLoad() *Config {
	configPath := os.Getenv(cfgPathEnv)
	if configPath == "" {
		var cfg Config
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("error reading environment config: %v", err)
		}
		return &cfg
	}

	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config: %v", err)
	}
	return &cfg
}
