package auth

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/database"
)

func TestEligibilityRestrictsEmailExceptionToApprovedActiveOperators(t *testing.T) {
	s := &Service{cfg: config.Config{SchoolDomain: "school.example"}}
	operator := database.Account{Username: pgtype.Text{String: "DeskOperator", Valid: true}, Role: "admin", Status: "active", EligibilityStatus: "approved"}
	if !s.eligible(operator) {
		t.Fatal("approved operator requires an email")
	}
	for _, mutate := range []func(*database.Account){
		func(a *database.Account) { a.Role = "student" },
		func(a *database.Account) { a.Status = "disabled" },
		func(a *database.Account) { a.Status = "pending" },
		func(a *database.Account) { a.EligibilityStatus = "pending" },
		func(a *database.Account) { a.Email = pgtype.Text{String: "person@school.example", Valid: true} },
		func(a *database.Account) { a.EmailVerifiedAt = stamp(time.Now()) },
		func(a *database.Account) { a.Username.String = "invalid@identity" },
		func(a *database.Account) { a.Username.Valid = false },
	} {
		a := operator
		mutate(&a)
		if s.eligible(a) {
			t.Fatalf("invalid operator eligible: %+v", a)
		}
	}
	student := database.Account{Email: pgtype.Text{String: "person@school.example", Valid: true}, EmailVerifiedAt: stamp(time.Now()), Role: "student", Status: "active", EligibilityStatus: "approved"}
	if !s.eligible(student) {
		t.Fatal("verified and approved student rejected")
	}
	for _, mutate := range []func(*database.Account){
		func(a *database.Account) { a.EmailVerifiedAt.Valid = false },
		func(a *database.Account) { a.Email.String = "person@other.example" },
		func(a *database.Account) { a.EligibilityStatus = "pending" },
		func(a *database.Account) { a.Email.Valid = false },
	} {
		a := student
		mutate(&a)
		if s.eligible(a) {
			t.Fatal("normal account bypassed email or approval requirements")
		}
	}
	if accountIdentity(operator) != "username:deskoperator" || accountIdentity(student) != "email:person@school.example" {
		t.Fatal("rate limit identities do not identify each account")
	}
}

func TestUsernamePolicyIsBoundedASCII(t *testing.T) {
	for _, username := range []string{"ab", "1account", "name@example.com", "na me", "İdentity", "abc\n", "_operator", "abcdefghijklmnopqrstuvwxyzabcdefg"} {
		if validUsername(username) {
			t.Fatalf("invalid username accepted: %q", username)
		}
	}
	for _, username := range []string{"abc", "Desk_Operator3", "abcdefghijklmnopqrstuvwxyzabcdef"} {
		if !validUsername(username) {
			t.Fatalf("valid username rejected: %q", username)
		}
	}
}
