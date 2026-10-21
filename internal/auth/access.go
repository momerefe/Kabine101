package auth

import (
	"errors"
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

// RequireAccount authenticates an eligible student or administrator, including operators.
func (s *Service) RequireAccount(w http.ResponseWriter, r *http.Request) (pgtype.UUID, []byte, bool) {
	a, hash, err := s.authenticated(r)
	if err != nil {
		if errors.Is(err, errDenied) {
			fail(w, 401, "unauthorized", "Oturum açmanız gerekiyor.")
		} else {
			unavailable(w)
		}
		return a.ID, nil, false
	}
	return a.ID, hash, true
}

// RecheckAccount derives the current identity from the session and holds its lock.
func (s *Service) RecheckAccount(r *http.Request, tx pgx.Tx, id pgtype.UUID, hash []byte) bool {
	q := database.New(tx)
	a, e := q.LockAccount(r.Context(), id)
	if e != nil || !s.eligible(a) {
		return false
	}
	current, e := q.SessionAccount(r.Context(), hash)
	return e == nil && current.ID == a.ID
}
