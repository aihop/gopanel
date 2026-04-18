package config

type ServerConfig struct {
	Debug     bool      `mapstructure:"debug"`
	System    System    `mapstructure:"system"`
	LogConfig LogConfig `mapstructure:"log"`
}
