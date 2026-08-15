package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `yaml:"env" env-default:"development"`
	Storage    `yaml:"storage"`
	Provider   `yaml:"provider"`
	HTTPServer `yaml:"http_server"`
}

type Storage struct {
	Host     string        `yaml:"host" env-default:"localhost"`
	Port     int           `yaml:"port" env-default:"5432"`
	User     string        `yaml:"user" env-default:"postgres"`
	Password string        `yaml:"password" env-default:"postgres"`
	Database string        `yaml:"database" env-default:"postgres"`
	Timeout  time.Duration `yaml:"timeout" env-default:"5s"`
}

func (s Storage) String() string {
	// postgres://jack:secret@foo.example.com:5432,bar.example.com:5432/mydb
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", s.User, s.Password, s.Host, s.Port, s.Database)
}

type Provider struct {
	Timeout time.Duration `yaml:"timeout" env-default:"5s"`
}

type HTTPServer struct {
	Address      string        `yaml:"address" env-default:":8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env-default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env-default:"60s"`
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
