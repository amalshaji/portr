package service

import (
	"github.com/amalshaji/portr/internal/client/db"
)

func (s *Service) GetTunnels() ([]*db.Request, error) {
	var result []*db.Request
	s.db.Conn.Raw(`
		SELECT r.subdomain AS subdomain, r.localport AS localport
		FROM requests r
		INNER JOIN (
			SELECT subdomain, localport, MAX(logged_at) as max_logged_at
			FROM requests
			GROUP BY subdomain, localport
		) latest
		ON r.subdomain = latest.subdomain
		AND r.localport = latest.localport
		AND r.logged_at = latest.max_logged_at
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
