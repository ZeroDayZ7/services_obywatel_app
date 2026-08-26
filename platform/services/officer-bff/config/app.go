package config

type App struct {
	Config *Config
}

//#region InitApp
func InitApp() *App {
	if err := LoadConfigGlobal(); err != nil {
		panic(err)
	}

	return &App{
		Config: &AppConfig,
	}
}
