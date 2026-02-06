package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Stats struct {
	TotalFiles     int   `json:"total_files"`
	TotalSize      int64 `json:"total_size"`
	TotalOrganized int   `json:"total_organized"`
}

type Config struct {
	DarkMode     bool   `json:"dark_mode"`
	Language     string `json:"language"`
	TargetFolder string `json:"target_folder"`
	Stats        Stats  `json:"stats"`
}

func getConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "lume_config.json"
	}
	return filepath.Join(filepath.Dir(exe), "lume_config.json")
}

func LoadConfig() Config {
	path := getConfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		return Config{Language: "tr"}
	}
	
	var conf Config
	json.Unmarshal(file, &conf)
	
	if conf.Language != "tr" && conf.Language != "en" {
		conf.Language = "tr"
	}
	
	return conf
}

func SaveConfig(conf Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}
