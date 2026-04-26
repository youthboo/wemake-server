package service

import (
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type AdminDashboardService struct {
	repo *repository.AdminDashboardRepository
}

func NewAdminDashboardService(repo *repository.AdminDashboardRepository) *AdminDashboardService {
	return &AdminDashboardService{repo: repo}
}

func (s *AdminDashboardService) GetSummary(from, to time.Time, period string) (*domain.AdminDashboardSummary, error) {
	return s.repo.GetSummary(from, to, period)
}

func (s *AdminDashboardService) GetRevenueChart(from, to time.Time, granularity string) ([]domain.RevenueChartPoint, error) {
	return s.repo.GetRevenueChart(from, to, granularity)
}

func (s *AdminDashboardService) GetTopFactories(from, to time.Time, limit int) ([]domain.TopFactoryRow, error) {
	return s.repo.GetTopFactories(from, to, limit)
}
