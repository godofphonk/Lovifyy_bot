package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/logger"
)

// Manager управляет graceful shutdown
type Manager struct {
	logger    *logger.Logger
	timeout   time.Duration
	callbacks []func() error
	mutex     sync.RWMutex
	done      chan struct{}
}

// NewManager создает новый менеджер shutdown
func NewManager(logger *logger.Logger, timeout time.Duration) *Manager {
	return &Manager{
		logger:    logger,
		timeout:   timeout,
		callbacks: make([]func() error, 0),
		done:      make(chan struct{}),
	}
}

// AddCallback добавляет callback для выполнения при shutdown
func (m *Manager) AddCallback(callback func() error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// Wait ожидает сигнал завершения и выполняет graceful shutdown
func (m *Manager) Wait() {
	// Создаем канал для получения сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Ожидаем сигнал
	sig := <-sigChan
	m.logger.WithFields(map[string]interface{}{
		"signal": sig.String(),
	}).Info("Received shutdown signal")

	// Выполняем graceful shutdown
	m.performShutdown()
}

// Shutdown выполняет немедленный graceful shutdown
func (m *Manager) Shutdown() {
	m.logger.Info("Performing graceful shutdown")
	m.performShutdown()
}

// performShutdown выполняет процедуру graceful shutdown
func (m *Manager) performShutdown() {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	// Создаем канал для завершения
	shutdownComplete := make(chan struct{})

	go func() {
		defer close(shutdownComplete)
		m.executeCallbacks()
	}()

	// Ожидаем завершения или таймаута
	select {
	case <-shutdownComplete:
		m.logger.Info("Graceful shutdown completed successfully")
	case <-ctx.Done():
		m.logger.WithError(ctx.Err()).Error("Shutdown timeout exceeded")
	}

	close(m.done)
}

// executeCallbacks выполняет все зарегистрированные callbacks
func (m *Manager) executeCallbacks() {
	m.mutex.RLock()
	callbacks := make([]func() error, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mutex.RUnlock()

	// Выполняем callbacks в обратном порядке (LIFO)
	for i := len(callbacks) - 1; i >= 0; i-- {
		callback := callbacks[i]
		if err := callback(); err != nil {
			m.logger.WithError(err).WithFields(map[string]interface{}{
				"callback_index": i,
			}).Error("Error during shutdown callback execution")
		}
	}
}

// Done возвращает канал, который закрывается при завершении shutdown
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// IsShuttingDown проверяет, выполняется ли shutdown
func (m *Manager) IsShuttingDown() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

// ShutdownHook представляет хук для graceful shutdown
type ShutdownHook struct {
	Name     string
	Priority int
	Callback func() error
}

// PriorityManager управляет shutdown с приоритетами
type PriorityManager struct {
	logger  *logger.Logger
	timeout time.Duration
	hooks   []ShutdownHook
	mutex   sync.RWMutex
	done    chan struct{}
}

// NewPriorityManager создает новый менеджер с приоритетами
func NewPriorityManager(logger *logger.Logger, timeout time.Duration) *PriorityManager {
	return &PriorityManager{
		logger:  logger,
		timeout: timeout,
		hooks:   make([]ShutdownHook, 0),
		done:    make(chan struct{}),
	}
}

// AddHook добавляет хук с приоритетом
func (pm *PriorityManager) AddHook(name string, priority int, callback func() error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	hook := ShutdownHook{
		Name:     name,
		Priority: priority,
		Callback: callback,
	}
	
	pm.hooks = append(pm.hooks, hook)
	pm.sortHooks()
}

// sortHooks сортирует хуки по приоритету (больший приоритет = выполняется первым)
func (pm *PriorityManager) sortHooks() {
	for i := 0; i < len(pm.hooks)-1; i++ {
		for j := i + 1; j < len(pm.hooks); j++ {
			if pm.hooks[i].Priority < pm.hooks[j].Priority {
				pm.hooks[i], pm.hooks[j] = pm.hooks[j], pm.hooks[i]
			}
		}
	}
}

// Wait ожидает сигнал завершения
func (pm *PriorityManager) Wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	pm.logger.WithFields(map[string]interface{}{
		"signal": sig.String(),
	}).Info("Received shutdown signal")

	pm.performShutdown()
}

// performShutdown выполняет shutdown с приоритетами
func (pm *PriorityManager) performShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), pm.timeout)
	defer cancel()

	shutdownComplete := make(chan struct{})

	go func() {
		defer close(shutdownComplete)
		pm.executeHooks()
	}()

	select {
	case <-shutdownComplete:
		pm.logger.Info("Priority shutdown completed successfully")
	case <-ctx.Done():
		pm.logger.WithError(ctx.Err()).Error("Priority shutdown timeout exceeded")
	}

	close(pm.done)
}

// executeHooks выполняет хуки в порядке приоритета
func (pm *PriorityManager) executeHooks() {
	pm.mutex.RLock()
	hooks := make([]ShutdownHook, len(pm.hooks))
	copy(hooks, pm.hooks)
	pm.mutex.RUnlock()

	for _, hook := range hooks {
		pm.logger.WithFields(map[string]interface{}{
			"hook_name": hook.Name,
			"priority":  hook.Priority,
		}).Info("Executing shutdown hook")

		if err := hook.Callback(); err != nil {
			pm.logger.WithError(err).WithFields(map[string]interface{}{
				"hook_name": hook.Name,
				"priority":  hook.Priority,
			}).Error("Error during shutdown hook execution")
		} else {
			pm.logger.WithFields(map[string]interface{}{
				"hook_name": hook.Name,
				"priority":  hook.Priority,
			}).Info("Shutdown hook completed successfully")
		}
	}
}

// Done возвращает канал завершения
func (pm *PriorityManager) Done() <-chan struct{} {
	return pm.done
}
