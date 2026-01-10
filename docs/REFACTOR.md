#### 1. Konfiguracja: Hot Reload (Viper WatchConfig)

**Cel:** Umożliwienie aplikacji aktualizacji ustawień w locie (np. zmiana poziomu logowania, zmiana limitów body) bez konieczności restartowania kontenera/serwisu.

**Jak to wdrożyć w `pkg/viper/config.go`:**
Wystarczy dodać te linie po `viper.ReadInConfig()`:

```go
viper.WatchConfig()
viper.OnConfigChange(func(e fsnotify.Event) {
    fmt.Printf("Konfiguracja zmieniona: %s\n", e.Name)
    // Ważne: musisz ponownie wywołać Unmarshal, aby zaktualizować globalną strukturę
    if err := viper.Unmarshal(cfg, ...); err != nil {
        fmt.Printf("Błąd przeładowania configu: %v\n", err)
    }
})

```

**Zalety:**

* **Zero downtime:** Zmieniasz `.env` lub ConfigMapę w Kubernetes, a aplikacja od razu używa nowych wartości.
* **Dynamiczne debugowanie:** Możesz włączyć tryb DEBUG tylko na chwilę i wyłączyć go po zebraniu logów.

**Wymagania:**

* Biblioteka `github.com/fsnotify/fsnotify`.
* Trzeba uważać na wyścigi (race conditions), jeśli wiele wątków czyta config w momencie zapisu (warto wtedy użyć `sync.RWMutex` do ochrony struktury `AppConfig`).
