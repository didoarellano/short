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

type CustomSlugConfig struct {
	ReservedWords []string `json:"reserved_words"`
	MinLength     int      `json:"min_length"`
	MaxLength     int      `json:"max_length"`
}

var (
	customSlugConfig *CustomSlugConfig
	once             sync.Once
)

func LoadCustomSlugConfig() (*CustomSlugConfig, error) {
	var loadErr error
	once.Do(func() {
		file, err := os.ReadFile("./internal/config/custom-paths.json")
		if err != nil {
			loadErr = err
		}
		customSlugConfig = &CustomSlugConfig{}
		err = json.Unmarshal(file, &customSlugConfig)
		if err != nil {
			loadErr = err
		}
	})
	return customSlugConfig, loadErr
}
