package jobs

import (
	"fmt"
	"sync"
	"time"
)

type Status string

const (
	StatusQueued  Status = "queued"
	StatusRunning Status = "running"
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Target    string    `json:"target"`
	Status    Status    `json:"status"`
	Logs      []string  `json:"logs"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewManager() *Manager {
	return &Manager{jobs: map[string]*Job{}}
}

func (m *Manager) NewJob(jobType string, target string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	job := &Job{
		ID:        fmt.Sprintf("%d", now.UnixNano()),
		Type:      jobType,
		Target:    target,
		Status:    StatusQueued,
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.jobs[job.ID] = job
	return cloneJob(job)
}

func (m *Manager) GetJob(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneJob(job), true
}

func (m *Manager) HasRunningJob() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, job := range m.jobs {
		if job.Status == StatusQueued || job.Status == StatusRunning {
			return true
		}
	}

	return false
}

func (m *Manager) Run(id string, run func(logf func(string)) error) {
	go func() {
		m.setStatus(id, StatusRunning, "")
		err := run(func(line string) {
			m.appendLog(id, line)
		})
		if err != nil {
			m.setStatus(id, StatusFailed, err.Error())
			return
		}
		m.setStatus(id, StatusSuccess, "")
	}()
}

func (m *Manager) setStatus(id string, status Status, errMessage string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return
	}
	job.Status = status
	job.Error = errMessage
	job.UpdatedAt = time.Now().UTC()
}

func (m *Manager) appendLog(id string, line string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return
	}
	job.Logs = append(job.Logs, line)
	job.UpdatedAt = time.Now().UTC()
}

func cloneJob(in *Job) *Job {
	logs := make([]string, len(in.Logs))
	copy(logs, in.Logs)
	return &Job{
		ID:        in.ID,
		Type:      in.Type,
		Target:    in.Target,
		Status:    in.Status,
		Logs:      logs,
		Error:     in.Error,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}
