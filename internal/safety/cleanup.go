package safety

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type CleanupStep struct {
	Description string
	Func        func(context.Context) error
}

type CleanupManager struct {
	mu    sync.Mutex
	steps []CleanupStep
}

func NewCleanupManager() *CleanupManager {
	return &CleanupManager{
		steps: make([]CleanupStep, 0, 4),
	}
}

func (m *CleanupManager) Register(description string, fn func(context.Context) error) {
	if fn == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps = append(m.steps, CleanupStep{
		Description: description,
		Func:        fn,
	})
}

func (m *CleanupManager) Run(ctx context.Context) error {
	m.mu.Lock()
	steps := make([]CleanupStep, len(m.steps))
	copy(steps, m.steps)
	m.mu.Unlock()

	var errs []string
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if err := step.Func(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", step.Description, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
}
