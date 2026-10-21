package auth

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

func (s *Service) CSRF(next http.Handler) http.Handler { return s.csrf(next) }
func (s *Service) RequireAdmin(w http.ResponseWriter, r *http.Request) (pgtype.UUID, []byte, bool) {
	a, h, ok := s.requireAdmin(w, r)
	return a.ID, h, ok
}

// RecheckAdmin holds the account lock until commit, serializing revocation with writes.
func (s *Service) RecheckAdmin(r *http.Request, tx pgx.Tx, id pgtype.UUID, hash []byte) bool {
	q := database.New(tx)
	a, e := q.LockAccount(r.Context(), id)
	if e != nil || a.Role != "admin" || !s.eligible(a) {
		return false
	}
	current, e := q.SessionAccount(r.Context(), hash)
	return e == nil && current.ID == id
}
