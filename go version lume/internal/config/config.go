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
	DryRun       bool   `json:"dry_run"`
	Rename       bool   `json:"rename"`
}

func getConfigPath() string {
	exe, err := os.Executable()
	var dir string
	if err != nil {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("USERPROFILE")
		}
		dir = filepath.Join(appData, "Lume")
	} else {
		dir = filepath.Dir(exe)
	}

	dir = filepath.Clean(dir)

	testFile := filepath.Join(dir, ".lume_write_test")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err == nil {
		os.Remove(testFile)
		return filepath.Join(dir, "lume_config.json")
	}

	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("USERPROFILE")
	}
	lumeDir := filepath.Clean(filepath.Join(appData, "Lume"))
	os.MkdirAll(lumeDir, 0755)
	return filepath.Join(lumeDir, "lume_config.json")
}

func LoadConfig() Config {
	path := getConfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		return Config{Language: "tr"}
	}

	var conf Config
	if err := json.Unmarshal(file, &conf); err != nil {

		return Config{Language: "tr"}
	}

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

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
