package main

import (
	"log"
	"net/http"

	"knx-updater/internal/api"
	"knx-updater/internal/config"
	"knx-updater/internal/jobs"
	"knx-updater/internal/services"
)

func main() {
	cfg := config.Load()
	domainService := services.NewDomainService(cfg.CustomComponentsDir)
	haService := services.NewHAService(cfg)
	updaterService := services.NewUpdaterService(cfg, haService, domainService)
	jobManager := jobs.NewManager()

	handler := api.NewHandler(cfg, domainService, updaterService, jobManager, haService)

	log.Printf("starting knx manager on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, handler.Router()); err != nil {
		log.Fatal(err)
	}
}
