package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type FavoriteService struct {
	repo *repository.FavoriteRepository
}

func NewFavoriteService(repo *repository.FavoriteRepository) *FavoriteService {
	return &FavoriteService{repo: repo}
}

func (s *FavoriteService) ListByUserID(userID int64) ([]domain.Favorite, error) {
	return s.repo.ListByUserID(userID)
}

func (s *FavoriteService) Add(fav *domain.Favorite) error {
	return s.repo.Add(fav)
}

func (s *FavoriteService) Remove(userID, showcaseID int64) error {
	return s.repo.Remove(userID, showcaseID)
}
