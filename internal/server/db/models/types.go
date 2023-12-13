package db

type UserWithTeams struct {
	GetUserBySessionRow
	Teams []Team
}
