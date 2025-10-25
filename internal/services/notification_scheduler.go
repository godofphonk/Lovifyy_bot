package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/godofphonk/lovifyy-bot/internal/models"
)

type ScheduledNotification struct {
	ID         string                 `json:"id"`
	Type       models.NotificationType `json:"type"`
	SendAt     time.Time              `json:"send_at"`
	Recipients []int64                `json:"recipients"`
	CreatedAt  time.Time              `json:"created_at"`
	Message    string                 `json:"message,omitempty"`    // Предварительно сгенерированное сообщение
	CustomText string                 `json:"custom_text,omitempty"` // Кастомный текст для кастомных уведомлений
}

// scheduler state
type scheduleStore struct {
	Items []ScheduledNotification `json:"items"`
}

// internal helpers
func (ns *NotificationService) scheduleFile() string {
	return filepath.Join(ns.dataDir, "schedule.json")
}

var schedMu sync.Mutex

func (ns *NotificationService) LoadSchedule() ([]ScheduledNotification, error) {
	schedMu.Lock()
	defer schedMu.Unlock()

	file := ns.scheduleFile()
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduledNotification{}, nil
		}
		return nil, err
	}
	var store scheduleStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store.Items, nil
}

func (ns *NotificationService) saveSchedule(items []ScheduledNotification) error {
	schedMu.Lock()
	defer schedMu.Unlock()

	store := scheduleStore{Items: items}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ns.scheduleFile(), data, 0644)
}

func (ns *NotificationService) ListScheduled() ([]ScheduledNotification, error) {
	return ns.LoadSchedule()
}

func (ns *NotificationService) ScheduleNotification(sendAt time.Time, typ models.NotificationType, recipients []int64) (string, error) {
	items, err := ns.LoadSchedule()
	if err != nil { return "", err }
	
	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	
	// НЕ генерируем сообщение заранее - будем генерировать при отправке
	// Это обеспечит уникальность каждого уведомления
	
	items = append(items, ScheduledNotification{
		ID: id,
		Type: typ,
		SendAt: sendAt,
		Recipients: recipients,
		CreatedAt: time.Now(),
		Message: "", // Пустое сообщение - будет генерироваться при отправке
	})
	if err := ns.saveSchedule(items); err != nil { return "", err }
	return id, nil
}

// ScheduleCustomNotification планирует кастомное уведомление с заданным текстом
func (ns *NotificationService) ScheduleCustomNotification(sendAt time.Time, customText string, recipients []int64) (string, error) {
	items, err := ns.LoadSchedule()
	if err != nil { return "", err }
	
	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	
	items = append(items, ScheduledNotification{
		ID: id,
		Type: "custom", // Специальный тип для кастомных уведомлений
		SendAt: sendAt,
		Recipients: recipients,
		CreatedAt: time.Now(),
		CustomText: customText,
	})
	if err := ns.saveSchedule(items); err != nil { return "", err }
	return id, nil
}

func (ns *NotificationService) CancelScheduled(id string) error {
	items, err := ns.LoadSchedule()
	if err != nil { return err }
	var out []ScheduledNotification
	for _, it := range items {
		if it.ID != id { out = append(out, it) }
	}
	return ns.saveSchedule(out)
}

// StartScheduler runs background loop to deliver notifications
func (ns *NotificationService) StartScheduler(stop <-chan struct{}) {
	// tick every 30s
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			items, err := ns.LoadSchedule()
			if err != nil { continue }
			now := time.Now()
			var remaining []ScheduledNotification
			for _, it := range items {
				if it.SendAt.After(now) {
					remaining = append(remaining, it)
					continue
				}
				// due: send notification
				var message string
				var err error
				
				// Если есть кастомный текст, используем его
				if it.CustomText != "" {
					message = it.CustomText
				} else if it.Message != "" {
					// Если есть заранее сгенерированное сообщение, используем его
					message = it.Message
				} else {
					// Если нет заранее сгенерированного сообщения, генерируем новое
					message, err = ns.GenerateNotification(it.Type)
					if err != nil {
						// Если не удалось сгенерировать, пропускаем
						continue
					}
				}
				
				// Отправляем сообщение
				if len(it.Recipients) == 0 {
					_ = ns.SendNotificationToAll(message)
				} else {
					for _, userID := range it.Recipients {
						_ = ns.SendNotificationToUser(userID, message)
					}
				}
			}
			_ = ns.saveSchedule(remaining)
		}
	}
}
