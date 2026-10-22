package student

import (
	"net/http"

	"github.com/momerefe/Kabine101/internal/benefitimage"
)

func (s *Service) benefitImage(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAccount(w, r); !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	v, err := s.q.StudentBenefitImage(r.Context(), id)
	if err != nil {
		dbfail(w, err)
		return
	}
	benefitimage.Serve(w, v.ContentType, v.Data)
}
