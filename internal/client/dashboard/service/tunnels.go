package service

import "github.com/amalshaji/portr/internal/client/db"

func (s *Service) GetTunnels() ([]*db.Request, error) {
	var result []*db.Request
	s.db.Conn.Distinct("subdomain", "localport").Find(&result)
	return result, nil
}

func (s *Service) GetRequests(subdomain string, port string) (*[]db.Request, error) {
	var result []db.Request
	s.db.Conn.Where("subdomain = ? AND localport = ?", subdomain, port).Find(&result)
	return &result, nil
}
