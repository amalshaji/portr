package db

import "time"

// RequestLog is a lightweight representation of a request/response pair.
// It is intentionally defined without any SQL/GORM types so that "nosql" builds
// can compile without pulling in database dependencies.
type RequestLog struct {
	ID                  string
	Subdomain           string
	LocalPort           int
	Host                string
	URL                 string
	Method              string
	HeadersJSON         []byte
	Body                []byte
	ResponseHeadersJSON []byte
	ResponseBody        []byte
	ResponseStatusCode  int
	LoggedAt            time.Time
	IsReplayed          bool
	ParentID            string
}
