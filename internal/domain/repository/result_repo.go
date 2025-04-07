package repository

import (
	"github.com/yourusername/trivia-api/internal/domain/entity"
)

// ResultRepository определяет методы для работы с результатами
type ResultRepository interface {
	SaveUserAnswer(answer *entity.UserAnswer) error
	GetUserAnswers(userID uint, quizID uint) ([]entity.UserAnswer, error)
	GetQuizUserAnswers(quizID uint) ([]entity.UserAnswer, error)
	SaveResult(result *entity.Result) error
	GetQuizResults(quizID uint) ([]entity.Result, error)
	GetUserResult(userID uint, quizID uint) (*entity.Result, error)
	GetUserResults(userID uint, limit, offset int) ([]entity.Result, error)
	CalculateRanks(quizID uint) error
	GetQuizWinners(quizID uint) ([]entity.Result, error)
}
