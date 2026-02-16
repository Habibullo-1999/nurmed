package config

import (
	"path"
	_ "path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var Module = fx.Provide(NewConfig)

type Config interface {
	Get(key string) interface{}
	GetBool(key string) bool
	GetFloat64(key string) float64
	GetInt(key string) int
	GetInt64(key string) int64
	GetIntSlice(key string) []int
	GetString(key string) string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	UnmarshalKey(key string, val interface{}) error
	GetStringSlice(key string) []string
	GetDuration(key string) time.Duration
}

type config struct {
	cfg *viper.Viper
}

func NewConfig() Config {

	cfg := viper.New()
	cfg.SetConfigName("appConfig")
	cfg.SetConfigType("json")
	cfg.AddConfigPath("./configs")

	cfg.AddConfigPath(getConfigPath())

	if err := cfg.ReadInConfig(); err != nil {
		panic(err)
	}

	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()

	cfg.WatchConfig()

	return &config{cfg: cfg}
}

func (c *config) Get(key string) interface{} {
	return c.cfg.Get(key)
}

func (c *config) GetBool(key string) bool {
	return c.cfg.GetBool(key)
}

func (c *config) GetFloat64(key string) float64 {
	return c.cfg.GetFloat64(key)
}

func (c *config) GetInt(key string) int {
	return c.cfg.GetInt(key)
}

func (c *config) GetInt64(key string) int64 {
	return c.cfg.GetInt64(key)
}

func (c *config) GetIntSlice(key string) []int {
	return c.cfg.GetIntSlice(key)
}

func (c *config) GetString(key string) string {
	return c.cfg.GetString(key)
}

func (c *config) GetStringSlice(key string) []string {
	return c.cfg.GetStringSlice(key)
}

func (c *config) GetStringMap(key string) map[string]interface{} {
	return c.cfg.GetStringMap(key)
}
func (c *config) GetStringMapString(key string) map[string]string {
	return c.cfg.GetStringMapString(key)
}

func (c *config) UnmarshalKey(key string, val interface{}) error {
	return c.cfg.UnmarshalKey(key, &val)
}

func (c *config) GetDuration(key string) time.Duration {
	return c.cfg.GetDuration(key)
}

func getConfigPath() string {
	_, currFilePath, _, _ := runtime.Caller(0)       // get current file full path
	d := path.Join(path.Dir(path.Dir(currFilePath))) // this equals: `cd ../../config.go` command
	return filepath.Dir(d) + "/configs"
}
