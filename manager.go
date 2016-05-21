package fsync

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/alexflint/go-cloudfile"
)

const copyBlockSize = 1024000

// The Manager handles fetching files as needed
type Manager struct {
	plan          Plan
	lastFetchedAt time.Time
	nextFetch     <-chan time.Time
	mu            sync.RWMutex
	abortable     Abortable
	quit          chan struct{}
}

// NewManager builds a manager with the specified fetch plan
func NewManager(plan Plan) *Manager {
	manager := &Manager{
		plan: plan,
		quit: make(chan struct{}),
	}

	if info, err := os.Stat(plan.LocalPath()); err == nil {
		manager.lastFetchedAt = info.ModTime()
	}

	manager.nextFetch = manager.computeNextFetchTime()

	return manager
}

// Start starts the manager, updating the remote file as needed
func (m *Manager) Start() {
	if m.quit == nil {
		m.quit = make(chan struct{})
	}

	go func() {
		for {
			select {
			case <-m.quit:
				m.abortable.Abort() // abort any currently running fetches
				m.quit = nil
				return
			case <-m.nextFetch:
				m.Fetch()
				m.nextFetch = m.computeNextFetchTime()
			}
		}
	}()
}

// Open opens the local file for reading. If the file needs to be fetched, it will be
func (m *Manager) Open() (file *os.File, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.needsFetch() {
		return m.lockedFetch()
	}
	return os.Open(m.plan.LocalPath())
}

// Fetch copies a file from the remote path to the local path (unless they are the same)
// and returns the local file
func (m *Manager) Fetch() (file *os.File, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lockedFetch()
}

// Abort can be called during a fetch operation to cancel it
func (m *Manager) Abort() error {
	return m.abortable.Abort()
}

func (m *Manager) lockedFetch() (file *os.File, err error) {
	var (
		localPath  = m.plan.LocalPath()
		remotePath = m.plan.RemotePath()
	)

	if file, err = os.OpenFile(localPath, os.O_RDWR|os.O_CREATE, 0644); err != nil {
		return nil, err
	}

	if localPath != remotePath {
		remoteFile, err := cloudfile.Open(remotePath)
		if err != nil {
			return nil, err
		}
		defer remoteFile.Close()

		defer file.Seek(0, 0)

		done, errs := m.abortable.Run(func() (interface{}, error) {
			if _, err := io.CopyN(file, remoteFile, copyBlockSize); err != nil {
				if err == io.EOF {
					return file, nil
				}
				return nil, err
			}
			return nil, nil
		})

		select {
		case <-done:
			m.lastFetchedAt = time.Now()
			return file, nil
		case err := <-errs:
			return nil, err
		}
	}

	return file, nil
}

func (m *Manager) needsFetch() bool {
	targetFetchTime := m.lastFetchedAt.Add(m.plan.UpdateInterval())
	if time.Now().After(targetFetchTime) {
		return true
	}

	if _, err := os.Stat(m.plan.LocalPath()); os.IsNotExist(err) {
		return true
	}

	return false
}

func (m *Manager) computeNextFetchTime() <-chan time.Time {
	targetFetchTime := m.lastFetchedAt.Add(m.plan.UpdateInterval())
	timeDiff := targetFetchTime.Sub(time.Now())

	return time.After(timeDiff)
}
