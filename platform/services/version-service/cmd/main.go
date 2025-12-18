package main

import (
    "log"
    "net/http"
    "platform/pkg/server"
    "platform/services/version-service/internal/router"
)

func main() {
    mux := http.NewServeMux()
    router.RegisterRoutes(mux)

    srv := server.New(mux)
    log.Println("version-service running on :3005")
    if err := srv.Start(":3005"); err != nil {
        log.Fatal(err)
    }
}
