package exercises

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WeekExercise представляет упражнения для одной недели
type WeekExercise struct {
	Week                int    `json:"week"`
	Title               string `json:"title"`
	WelcomeMessage      string `json:"welcome_message"`      // Приветственное сообщение
	Questions           string `json:"questions"`            // Кнопка вопросов
	Tips                string `json:"tips"`                 // Кнопка подсказки (статичная)
	Insights            string `json:"insights"`             // Кнопка инсайт
	JointQuestions      string `json:"joint_questions"`      // Совместные вопросы в конце недели
	DiaryInstructions   string `json:"diary_instructions"`   // Что делать в дневнике
	IsActive            bool   `json:"is_active"`            // Доступна ли неделя для пользователей
}

// Manager управляет упражнениями
type Manager struct {
	exercisesDir string
}

// NewManager создает новый менеджер упражнений
func NewManager() *Manager {
	exercisesDir := "exercises"
	os.MkdirAll(exercisesDir, 0755)
	return &Manager{exercisesDir: exercisesDir}
}

// SaveWeekExercise сохраняет упражнения для недели
func (m *Manager) SaveWeekExercise(week int, title, welcomeMessage, questions, tips, insights, jointQuestions, diaryInstructions string) error {
	exercise := WeekExercise{
		Week:                week,
		Title:               title,
		WelcomeMessage:      welcomeMessage,
		Questions:           questions,
		Tips:                tips,
		Insights:            insights,
		JointQuestions:      jointQuestions,
		DiaryInstructions:   diaryInstructions,
	}

	filename := filepath.Join(m.exercisesDir, fmt.Sprintf("week_%d.json", week))
	
	data, err := json.MarshalIndent(exercise, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// SaveWeekField сохраняет отдельное поле недели
func (m *Manager) SaveWeekField(week int, field, value string) error {
	// Получаем существующие упражнения
	exercise, err := m.GetWeekExercise(week)
	if err != nil {
		return err
	}
	
	// Если упражнений нет, создаем новые
	if exercise == nil {
		exercise = &WeekExercise{Week: week}
	}
	
	// Обновляем нужное поле
	switch field {
	case "title":
		exercise.Title = value
	case "welcome":
		exercise.WelcomeMessage = value
	case "questions":
		exercise.Questions = value
	case "tips":
		exercise.Tips = value
	case "insights":
		exercise.Insights = value
	case "joint":
		exercise.JointQuestions = value
	case "diary":
		exercise.DiaryInstructions = value
	case "active":
		// Для активации принимаем "true"/"false" или "1"/"0"
		exercise.IsActive = (value == "true" || value == "1")
	default:
		return fmt.Errorf("неизвестное поле: %s", field)
	}
	
	// Сохраняем обновленные упражнения
	filename := filepath.Join(m.exercisesDir, fmt.Sprintf("week_%d.json", week))
	data, err := json.MarshalIndent(exercise, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// GetWeekExercise получает упражнения для недели
func (m *Manager) GetWeekExercise(week int) (*WeekExercise, error) {
	filename := filepath.Join(m.exercisesDir, fmt.Sprintf("week_%d.json", week))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Упражнения для этой недели не настроены
		}
		return nil, err
	}

	var exercise WeekExercise
	err = json.Unmarshal(data, &exercise)
	if err != nil {
		return nil, err
	}

	return &exercise, nil
}

// GetAllExercises получает все настроенные упражнения
func (m *Manager) GetAllExercises() ([]WeekExercise, error) {
	var exercises []WeekExercise
	
	for week := 1; week <= 4; week++ {
		exercise, err := m.GetWeekExercise(week)
		if err != nil {
			return nil, err
		}
		if exercise != nil {
			exercises = append(exercises, *exercise)
		}
	}
	
	return exercises, nil
}

// DeleteWeekExercise удаляет упражнения для недели
func (m *Manager) DeleteWeekExercise(week int) error {
	filename := filepath.Join(m.exercisesDir, fmt.Sprintf("week_%d.json", week))
	return os.Remove(filename)
}

// GetActiveWeeks возвращает список номеров активных недель
func (m *Manager) GetActiveWeeks() []int {
	var activeWeeks []int
	
	for week := 1; week <= 4; week++ {
		exercise, err := m.GetWeekExercise(week)
		if err == nil && exercise != nil && exercise.IsActive {
			activeWeeks = append(activeWeeks, week)
		}
	}
	
	return activeWeeks
}

// IsWeekActive проверяет, активна ли неделя
func (m *Manager) IsWeekActive(week int) bool {
	exercise, err := m.GetWeekExercise(week)
	if err != nil || exercise == nil {
		return false
	}
	return exercise.IsActive
}
