package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"knx-updater/internal/config"
	"knx-updater/internal/jobs"
	"knx-updater/internal/models"
	"knx-updater/internal/services"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	cfg     config.Config
	domains *services.DomainService
	updater *services.UpdaterService
	jobs    *jobs.Manager
	ha      *services.HAService
}

type createDomainRequest struct {
	Domain  string `json:"domain"`
	Version string `json:"version,omitempty"`
}

type jobResponse struct {
	JobID string `json:"jobId"`
}

func NewHandler(cfg config.Config, domains *services.DomainService, updater *services.UpdaterService, jobs *jobs.Manager, ha *services.HAService) *Handler {
	return &Handler{cfg: cfg, domains: domains, updater: updater, jobs: jobs, ha: ha}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/api/domains", h.listDomains)
	r.Post("/api/domains", h.createDomain)
	r.Delete("/api/domains/{domain}", h.deleteDomain)
	r.Post("/api/domains/{domain}/update", h.updateDomain)
	r.Post("/api/domains/update-all", h.updateAll)
	r.Get("/api/jobs/{jobID}", h.getJob)
	r.Get("/api/system/info", h.systemInfo)

	r.Get("/", h.index)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Clean(h.cfg.StaticDir)))))

	return r
}

func (h *Handler) listDomains(w http.ResponseWriter, r *http.Request) {
	items, err := h.domains.ListDomains()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) createDomain(w http.ResponseWriter, r *http.Request) {
	if !h.ensureNoRunningJob(w) {
		return
	}

	var req createDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.domains.ValidateDomain(req.Domain); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job := h.jobs.NewJob("create", req.Domain)
	h.jobs.Run(job.ID, func(logf func(string)) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		return h.updater.UpdateDomain(ctx, req.Domain, req.Version, logf)
	})

	writeJSON(w, http.StatusAccepted, jobResponse{JobID: job.ID})
}

func (h *Handler) deleteDomain(w http.ResponseWriter, r *http.Request) {
	if !h.ensureNoRunningJob(w) {
		return
	}

	domain := chi.URLParam(r, "domain")
	if err := h.domains.DeleteDomain(domain); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateDomain(w http.ResponseWriter, r *http.Request) {
	if !h.ensureNoRunningJob(w) {
		return
	}

	domain := chi.URLParam(r, "domain")
	if err := h.domains.ValidateDomain(domain); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job := h.jobs.NewJob("update", domain)
	h.jobs.Run(job.ID, func(logf func(string)) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		return h.updater.UpdateDomain(ctx, domain, "", logf)
	})

	writeJSON(w, http.StatusAccepted, jobResponse{JobID: job.ID})
}

func (h *Handler) updateAll(w http.ResponseWriter, r *http.Request) {
	if !h.ensureNoRunningJob(w) {
		return
	}

	job := h.jobs.NewJob("update-all", "all")
	h.jobs.Run(job.ID, func(logf func(string)) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		return h.updater.UpdateAll(ctx, logf)
	})

	writeJSON(w, http.StatusAccepted, jobResponse{JobID: job.ID})
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")
	job, ok := h.jobs.GetJob(id)
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) systemInfo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	version, err := h.ha.GetVersion(ctx)
	if err != nil {
		version = "unavailable: " + err.Error()
	}

	payload := models.SystemInfo{
		HAVersion:      version,
		SupervisorURL:  h.cfg.SupervisorURL,
		ComponentsPath: h.cfg.CustomComponentsDir,
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(h.cfg.StaticDir, "index.html"))
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func (h *Handler) ensureNoRunningJob(w http.ResponseWriter) bool {
	if h.jobs.HasRunningJob() {
		writeError(w, http.StatusConflict, "another job is already running")
		return false
	}

	return true
}
