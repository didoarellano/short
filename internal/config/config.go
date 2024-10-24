package config

import (
	_ "embed"
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

//go:embed custom-paths.json
var customPathsJSON []byte

func LoadCustomSlugConfig() (*CustomSlugConfig, error) {
	var loadErr error
	once.Do(func() {
		customSlugConfig = &CustomSlugConfig{}
		err := json.Unmarshal(customPathsJSON, &customSlugConfig)
		if err != nil {
			loadErr = err
		}
	})
	return customSlugConfig, loadErr
}
