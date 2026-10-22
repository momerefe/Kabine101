package management

import (
	"net/http"

	"github.com/momerefe/Kabine101/internal/benefitimage"
)

func (s *Service) benefitImage(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	v, err := s.q.ManagementBenefitImage(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	benefitimage.Serve(w, v.ContentType, v.Data)
}
