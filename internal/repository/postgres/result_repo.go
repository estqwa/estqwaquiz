package postgres

import (
	"errors"

	"gorm.io/gorm"

	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// ResultRepo реализует repository.ResultRepository
type ResultRepo struct {
	db *gorm.DB
}

// NewResultRepo создает новый репозиторий результатов
func NewResultRepo(db *gorm.DB) *ResultRepo {
	return &ResultRepo{db: db}
}

// SaveUserAnswer сохраняет ответ пользователя
func (r *ResultRepo) SaveUserAnswer(answer *entity.UserAnswer) error {
	return r.db.Create(answer).Error
}

// GetUserAnswers возвращает все ответы пользователя для конкретной викторины
func (r *ResultRepo) GetUserAnswers(userID uint, quizID uint) ([]entity.UserAnswer, error) {
	var answers []entity.UserAnswer
	err := r.db.Where("user_id = ? AND quiz_id = ?", userID, quizID).
		Order("created_at").
		Find(&answers).Error
	return answers, err
}

// SaveResult сохраняет итоговый результат пользователя
func (r *ResultRepo) SaveResult(result *entity.Result) error {
	return r.db.Create(result).Error
}

// GetQuizResults возвращает все результаты для викторины
func (r *ResultRepo) GetQuizResults(quizID uint) ([]entity.Result, error) {
	var results []entity.Result
	err := r.db.Where("quiz_id = ?", quizID).
		Order("score DESC").
		Find(&results).Error
	return results, err
}

// GetUserResult возвращает результат пользователя для конкретной викторины
func (r *ResultRepo) GetUserResult(userID uint, quizID uint) (*entity.Result, error) {
	var result entity.Result
	err := r.db.Where("user_id = ? AND quiz_id = ?", userID, quizID).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("result not found")
		}
		return nil, err
	}
	return &result, nil
}

// GetUserResults возвращает все результаты пользователя с пагинацией
func (r *ResultRepo) GetUserResults(userID uint, limit, offset int) ([]entity.Result, error) {
	var results []entity.Result
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&results).Error
	return results, err
}

// CalculateRanks вычисляет ранги всех участников викторины
func (r *ResultRepo) CalculateRanks(quizID uint) error {
	// Получаем все результаты викторины, отсортированные по счету
	var results []entity.Result
	if err := r.db.Where("quiz_id = ?", quizID).
		Order("score DESC, correct_answers DESC").
		Find(&results).Error; err != nil {
		return err
	}

	// Получаем викторину для определения общего количества вопросов
	var quiz entity.Quiz
	if err := r.db.Preload("Questions").First(&quiz, quizID).Error; err != nil {
		return err
	}

	totalQuestions := len(quiz.Questions)
	prizeFundPerUser := 0

	// Определяем победителей (ответившие на все вопросы правильно И не выбывшие)
	var winners []entity.Result
	if totalQuestions > 0 {
		for _, result := range results {
			if result.CorrectAnswers == totalQuestions && !result.IsEliminated {
				winners = append(winners, result)
			}
		}
	}

	// Установка базового призового фонда
	const totalPrizeFund = 1000000 // 1,000,000 (можно настроить)

	// Распределяем призовой фонд между победителями
	if len(winners) > 0 {
		prizeFundPerUser = totalPrizeFund / len(winners)
	} else {
		prizeFundPerUser = 0 // Явно устанавливаем 0, если победителей нет
	}

	// Вычисляем ранги с учетом одинаковых очков
	if len(results) > 0 {
		currentRank := 1
		currentScore := results[0].Score
		results[0].Rank = currentRank

		// Отмечаем победителей и устанавливаем призовой фонд
		if totalQuestions > 0 && results[0].CorrectAnswers == totalQuestions && !results[0].IsEliminated {
			results[0].IsWinner = true
			results[0].PrizeFund = prizeFundPerUser
		} else {
			results[0].IsWinner = false
			results[0].PrizeFund = 0
		}

		// Обновляем первый результат (только rank, is_winner, prize_fund)
		if err := r.db.Model(&entity.Result{}).
			Where("id = ?", results[0].ID).
			Updates(map[string]interface{}{
				"rank":       results[0].Rank,
				"is_winner":  results[0].IsWinner,
				"prize_fund": results[0].PrizeFund,
			}).Error; err != nil {
			return err
		}

		// Обрабатываем остальные результаты
		skippedPositions := 0
		for i := 1; i < len(results); i++ {
			if results[i].Score < currentScore {
				// Если очки отличаются, увеличиваем ранг с учетом пропущенных мест
				currentRank = i + 1
				currentScore = results[i].Score
				skippedPositions = 0
			} else {
				// Если очки равны, сохраняем тот же ранг
				skippedPositions++
			}

			results[i].Rank = currentRank

			// Определяем, является ли игрок победителем
			if totalQuestions > 0 && results[i].CorrectAnswers == totalQuestions && !results[i].IsEliminated {
				results[i].IsWinner = true
				results[i].PrizeFund = prizeFundPerUser
			} else {
				results[i].IsWinner = false
				results[i].PrizeFund = 0
			}

			// Обновляем результат в БД (только rank, is_winner, prize_fund)
			if err := r.db.Model(&entity.Result{}).
				Where("id = ?", results[i].ID).
				Updates(map[string]interface{}{
					"rank":       results[i].Rank,
					"is_winner":  results[i].IsWinner,
					"prize_fund": results[i].PrizeFund,
				}).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// GetQuizUserAnswers возвращает все ответы пользователей для конкретной викторины
func (r *ResultRepo) GetQuizUserAnswers(quizID uint) ([]entity.UserAnswer, error) {
	var answers []entity.UserAnswer
	err := r.db.Where("quiz_id = ?", quizID).Find(&answers).Error
	return answers, err
}

// GetQuizWinners возвращает список победителей викторины
func (r *ResultRepo) GetQuizWinners(quizID uint) ([]entity.Result, error) {
	var winners []entity.Result
	err := r.db.Where("quiz_id = ? AND is_winner = true", quizID).
		Order("score DESC"). // Сортируем победителей по очкам
		Find(&winners).Error
	return winners, err
}
