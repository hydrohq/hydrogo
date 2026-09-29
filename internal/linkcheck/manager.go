package linkcheck

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var ErrScanInProgress = errors.New("link scan already in progress")

const maxStoredScans = 32

type Job struct {
	ID         string
	TenantKey  string
	Running    bool
	StartedAt  time.Time
	FinishedAt time.Time
	Findings   []Finding
	Failed     bool
}

type Manager struct {
	mu      sync.Mutex
	options Options
	jobs    map[string]Job
}

func NewManager(options Options) *Manager {
	return &Manager{options: options, jobs: make(map[string]Job)}
}

func (m *Manager) Start(tenantKey string, startURL string) (Job, error) {
	if m == nil {
		return Job{}, errors.New("link scan manager unavailable")
	}
	m.mu.Lock()
	m.pruneLocked(time.Now())
	for _, job := range m.jobs {
		if job.TenantKey == tenantKey && job.Running {
			m.mu.Unlock()
			return Job{}, ErrScanInProgress
		}
	}
	if len(m.jobs) >= maxStoredScans {
		m.mu.Unlock()
		return Job{}, errors.New("link scan capacity reached")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		m.mu.Unlock()
		return Job{}, err
	}
	id := hex.EncodeToString(random[:])
	job := Job{ID: id, TenantKey: tenantKey, Running: true, StartedAt: time.Now().UTC()}
	m.jobs[id] = job
	m.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		options := m.options
		options.StartURL = startURL
		findings, err := Crawl(ctx, options)
		m.mu.Lock()
		current, ok := m.jobs[id]
		if ok {
			current.Running = false
			current.FinishedAt = time.Now().UTC()
			current.Failed = err != nil
			if err == nil {
				current.Findings = findings
			}
			m.jobs[id] = current
		}
		m.mu.Unlock()
	}()
	return job, nil
}

func (m *Manager) Get(tenantKey, id string) (Job, bool) {
	if m == nil {
		return Job{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())
	job, ok := m.jobs[id]
	if !ok || job.TenantKey != tenantKey {
		return Job{}, false
	}
	job.Findings = append([]Finding(nil), job.Findings...)
	return job, true
}

func (m *Manager) Latest(tenantKey string) (Job, bool) {
	if m == nil {
		return Job{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now())
	var latest Job
	for _, job := range m.jobs {
		if job.TenantKey == tenantKey && !job.Running && !job.Failed && job.StartedAt.After(latest.StartedAt) {
			latest = job
		}
	}
	if latest.ID == "" {
		return Job{}, false
	}
	latest.Findings = append([]Finding(nil), latest.Findings...)
	return latest, true
}

func (m *Manager) pruneLocked(now time.Time) {
	for id, job := range m.jobs {
		if !job.Running && now.Sub(job.FinishedAt) > time.Hour {
			delete(m.jobs, id)
		}
	}
}
