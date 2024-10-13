package config

import "os"

type GlobalAppData struct {
	AppPathPrefix     string
	RedirectorBaseURL string
}

var AppData = &GlobalAppData{
	AppPathPrefix:     os.Getenv("APP_PATH_PREFIX"),
	RedirectorBaseURL: os.Getenv("REDIRECTOR_BASE_URL"),
}
