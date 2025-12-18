package config

import "os"

type VersionConfig struct {
    Min     string
    Latest  string
    Force   bool
}

func Get() VersionConfig {
    return VersionConfig{
        Min:    os.Getenv("MIN_VERSION"),
        Latest: os.Getenv("LATEST_VERSION"),
        Force:  os.Getenv("FORCE_UPDATE") == "true",
    }
}
