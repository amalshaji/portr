package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/amalshaji/portr/internal/client/db"
)

type latestRequestRow struct {
	ID                 string
	Subdomain          string
	Localport          int
	Method             string
	URL                string
	ResponseStatusCode int
	LoggedAt           time.Time
}

type requestCountRow struct {
	Subdomain string
	Localport int
	Count     int64
}

type websocketCountRow struct {
	Subdomain      string
	Localport      int
	SessionCount   int64
	ActiveCount    int64
	LastActivityAt *string
}

func tunnelKey(subdomain string, localport int) string {
	return fmt.Sprintf("%s:%d", subdomain, localport)
}

func parseDBTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}

func (s *Service) GetTunnels() ([]TunnelSummary, error) {
	var latestRequests []latestRequestRow
	if err := s.db.Conn.Raw(`
		WITH ranked_requests AS (
			SELECT
				id,
				subdomain,
				localport,
				method,
				url,
				response_status_code,
				logged_at,
				ROW_NUMBER() OVER (
					PARTITION BY subdomain, localport
					ORDER BY logged_at DESC, id DESC
				) AS row_number
			FROM requests
		)
		SELECT
			id,
			subdomain,
			localport,
			method,
			url,
			response_status_code,
			logged_at
		FROM ranked_requests
		WHERE row_number = 1
	`).Scan(&latestRequests).Error; err != nil {
		return nil, err
	}

	var requestCounts []requestCountRow
	if err := s.db.Conn.Raw(`
		SELECT subdomain, localport, COUNT(*) AS count
		FROM requests
		GROUP BY subdomain, localport
	`).Scan(&requestCounts).Error; err != nil {
		return nil, err
	}

	var websocketCounts []websocketCountRow
	if err := s.db.Conn.Raw(`
		SELECT
			subdomain,
			localport,
			COUNT(*) AS session_count,
			SUM(CASE WHEN closed_at IS NULL THEN 1 ELSE 0 END) AS active_count,
			MAX(COALESCE(last_event_at, started_at)) AS last_activity_at
		FROM web_socket_sessions
		GROUP BY subdomain, localport
	`).Scan(&websocketCounts).Error; err != nil {
		return nil, err
	}

	summariesByKey := make(map[string]*TunnelSummary)
	for _, row := range requestCounts {
		key := tunnelKey(row.Subdomain, row.Localport)
		summariesByKey[key] = &TunnelSummary{
			Subdomain:        row.Subdomain,
			Localport:        row.Localport,
			HTTPRequestCount: row.Count,
		}
	}

	for _, row := range latestRequests {
		key := tunnelKey(row.Subdomain, row.Localport)
		summary := summariesByKey[key]
		if summary == nil {
			summary = &TunnelSummary{
				Subdomain: row.Subdomain,
				Localport: row.Localport,
			}
			summariesByKey[key] = summary
		}

		summary.LastRequestID = row.ID
		summary.LastMethod = row.Method
		summary.LastURL = row.URL
		summary.LastStatus = row.ResponseStatusCode
		if row.LoggedAt.After(summary.LastActivityAt) {
			summary.LastActivityAt = row.LoggedAt
			summary.LastActivityKind = "http"
		}
	}

	for _, row := range websocketCounts {
		key := tunnelKey(row.Subdomain, row.Localport)
		summary := summariesByKey[key]
		if summary == nil {
			summary = &TunnelSummary{
				Subdomain: row.Subdomain,
				Localport: row.Localport,
			}
			summariesByKey[key] = summary
		}

		summary.WebSocketSessionCount = row.SessionCount
		summary.ActiveWebSocketCount = row.ActiveCount
		if row.LastActivityAt == nil {
			continue
		}

		lastActivityAt, ok := parseDBTime(*row.LastActivityAt)
		if ok && lastActivityAt.After(summary.LastActivityAt) {
			summary.LastActivityAt = lastActivityAt
			summary.LastActivityKind = "websocket"
		}
	}

	summaries := make([]TunnelSummary, 0, len(summariesByKey))
	for _, summary := range summariesByKey {
		summaries = append(summaries, *summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].LastActivityAt.Equal(summaries[j].LastActivityAt) {
			if summaries[i].Subdomain == summaries[j].Subdomain {
				return summaries[i].Localport < summaries[j].Localport
			}
			return summaries[i].Subdomain < summaries[j].Subdomain
		}
		return summaries[i].LastActivityAt.After(summaries[j].LastActivityAt)
	})

	return summaries, nil
}

func (s *Service) GetRequests(subdomain string, port string) (*[]db.Request, error) {
	var result []db.Request
	if err := s.db.Conn.
		Where("subdomain = ? AND localport = ?", subdomain, port).
		Order("logged_at DESC").
		Order("id DESC").
		Find(&result).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) GetRequestById(id string) (*db.Request, error) {
	var request db.Request
	if err := s.db.Conn.Where("id = ?", id).First(&request).Error; err != nil {
		return nil, err
	}

	return &request, nil
}

func (s *Service) GetWebSocketSessions(subdomain string, port string) (*[]db.WebSocketSession, error) {
	var sessions []db.WebSocketSession
	if err := s.db.Conn.
		Where("subdomain = ? AND localport = ?", subdomain, port).
		Order("COALESCE(last_event_at, started_at) DESC").
		Order("started_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	return &sessions, nil
}

func (s *Service) GetWebSocketSessionByID(id string) (*WebSocketSessionWithEvents, error) {
	var session db.WebSocketSession
	if err := s.db.Conn.Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}

	var events []db.WebSocketEvent
	if err := s.db.Conn.
		Where("session_id = ?", id).
		Order("logged_at ASC").
		Order("id ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}

	return &WebSocketSessionWithEvents{
		Session: &session,
		Events:  events,
	}, nil
}

func (s *Service) DeleteTunnelLogs(subdomain string, port int) (int64, error) {
	tx := s.db.Conn.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var deletedTotal int64
	var sessionIDs []string
	if err := tx.Model(&db.WebSocketSession{}).
		Where("subdomain = ? AND localport = ?", subdomain, port).
		Pluck("id", &sessionIDs).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if len(sessionIDs) > 0 {
		result := tx.Where("session_id IN ?", sessionIDs).Delete(&db.WebSocketEvent{})
		if result.Error != nil {
			tx.Rollback()
			return 0, result.Error
		}
		deletedTotal += result.RowsAffected
	}

	sessionResult := tx.Where("subdomain = ? AND localport = ?", subdomain, port).Delete(&db.WebSocketSession{})
	if sessionResult.Error != nil {
		tx.Rollback()
		return 0, sessionResult.Error
	}
	deletedTotal += sessionResult.RowsAffected

	requestResult := tx.Where("subdomain = ? AND localport = ?", subdomain, port).Delete(&db.Request{})
	if requestResult.Error != nil {
		tx.Rollback()
		return 0, requestResult.Error
	}
	deletedTotal += requestResult.RowsAffected

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return deletedTotal, nil
}
