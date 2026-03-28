package service

import (
	"github.com/amalshaji/portr/internal/client/db"
)

func (s *Service) GetTunnels() ([]*db.Request, error) {
	var result []*db.Request
	s.db.Conn.Raw(`
		WITH latest_requests AS (
			SELECT subdomain, localport, MAX(logged_at) as max_logged_at
			FROM requests
			GROUP BY subdomain, localport
		)
		SELECT r.*
		FROM requests r
		JOIN latest_requests lr
			ON r.subdomain = lr.subdomain
			AND r.localport = lr.localport
			AND r.logged_at = lr.max_logged_at
		ORDER BY r.logged_at DESC
	`).Find(&result)
	return result, nil
}

func (s *Service) GetRequests(subdomain string, port string) (*[]db.Request, error) {
	var result []db.Request
	s.db.Conn.Where("subdomain = ? AND localport = ?", subdomain, port).Order("logged_at desc").Find(&result)
	return &result, nil
}

func (s *Service) GetRequestById(id string) (*db.Request, error) {
	var request db.Request
	err := s.db.Conn.Where("id = ?", id).Find(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (s *Service) ReplayRequestById(id string) error {
	var request db.Request
	err := s.db.Conn.Where("id = ?", id).Find(&request).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteTunnelLogs(subdomain string, port int) (int64, error) {
	result := s.db.Conn.Where(
		"subdomain = ? AND localport = ?",
		subdomain,
		port,
	).Delete(&db.Request{})

	return result.RowsAffected, result.Error
}
