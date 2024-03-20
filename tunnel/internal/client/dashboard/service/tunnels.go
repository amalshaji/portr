package service

import "github.com/amalshaji/portr/internal/client/db"

func (s *Service) GetTunnels() ([]*db.Request, error) {
	var result []*db.Request
	s.db.Conn.Distinct("subdomain", "localport").Find(&result)
	return result, nil
}

func (s *Service) GetRequests(subdomain string, port string) (*[]db.Request, error) {
	var result []db.Request
	s.db.Conn.Where("subdomain = ? AND localport = ?", subdomain, port).Order("logged_at desc").Find(&result)
	return &result, nil
}

func (s *Service) GetRequestById(id string) (*db.Request, error) {
	var result db.Request
	err := s.db.Conn.Where("id = ?", id).Find(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}
