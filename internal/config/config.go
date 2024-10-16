package config

import (
	"encoding/json"
	"os"
	"sync"
)

type GlobalAppData struct {
	AppPathPrefix     string
	RedirectorBaseURL string
}

var AppData = &GlobalAppData{
	AppPathPrefix:     os.Getenv("APP_PATH_PREFIX"),
	RedirectorBaseURL: os.Getenv("REDIRECTOR_BASE_URL"),
}

type CustomPathConfig struct {
	ReservedWords []string `json:"reserved_words"`
	MinLength     int      `json:"min_length"`
	MaxLength     int      `json:"max_length"`
}

var (
	customPathConfig *CustomPathConfig
	once             sync.Once
)

func LoadCustomPathConfig() (*CustomPathConfig, error) {
	var loadErr error
	once.Do(func() {
		file, err := os.ReadFile("./internal/config/custom-paths.json")
		if err != nil {
			loadErr = err
		}
		customPathConfig = &CustomPathConfig{}
		err = json.Unmarshal(file, &customPathConfig)
		if err != nil {
			loadErr = err
		}
	})
	return customPathConfig, loadErr
}
