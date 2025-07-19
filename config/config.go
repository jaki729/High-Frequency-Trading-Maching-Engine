package config

import (
    "gopkg.in/yaml.v3"
    "os"
)

type ExchangeConfig struct {
    Name      string `yaml:"name"`
    WSUrl     string `yaml:"ws_url"`
    Symbols   []string `yaml:"symbols"`
}

type Config struct {
    Server struct {
        Port int `yaml:"port"`
    } `yaml:"server"`
    
    Exchanges []ExchangeConfig `yaml:"exchanges"`
    
    Logging struct {
        Level string `yaml:"level"`
        File  string `yaml:"file"`
    } `yaml:"logging"`
    
    Metrics struct {
        Enabled bool `yaml:"enabled"`
        Port    int  `yaml:"port"`
    } `yaml:"metrics"`
}

func LoadConfig(filename string) (*Config, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    var config Config
    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return nil, err
    }
    
    return &config, nil
}