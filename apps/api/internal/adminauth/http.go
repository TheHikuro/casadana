package adminauth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

const sessionCookieName = "casa_admin_session"

func init() {
	httpserver.Register(ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS")
	httpserver.Register(ErrEmailTaken, http.StatusConflict, "EMAIL_TAKEN")
	httpserver.Register(ErrNotFound, http.StatusNotFound, "ADMIN_NOT_FOUND")
	httpserver.Register(ErrCannotDeleteSelf, http.StatusConflict, "CANNOT_DELETE_SELF")
	httpserver.Register(ErrInvalidToken, http.StatusUnauthorized, "UNAUTHORIZED")
}

func Mount(r chi.Router, svc *Service, cookieSecure bool) {
	r.Post("/api/admin/login", loginHandler(svc, cookieSecure))
	r.Post("/api/admin/logout", logoutHandler(cookieSecure))
	r.Group(func(r chi.Router) {
		r.Use(RequireAdminSession(svc))
		r.Get("/api/admin/me", meHandler())
		r.Get("/api/admin/users", listUsersHandler(svc))
		r.Post("/api/admin/users", createUserHandler(svc))
		r.Delete("/api/admin/users/{id}", deleteUserHandler(svc))
	})
}

type ctxKey struct{}

func adminFromContext(ctx context.Context) (*AdminUser, bool) {
	u, ok := ctx.Value(ctxKey{}).(*AdminUser)
	return u, ok
}

// AdminEmailFromContext returns the authenticated admin's email, or "" when the
// request did not pass through RequireAdminSession. Deliberately narrow: the
// audit log only needs to attribute an action, so it gets the email rather than
// the whole AdminUser.
func AdminEmailFromContext(ctx context.Context) string {
	u, ok := adminFromContext(ctx)
	if !ok {
		return ""
	}
	return u.Email
}

// RequireAdminSession validates the session cookie and injects the admin
// identity into the request context. Exported so other modules (e.g.
// booking) can gate their own routes without importing adminauth's
// internals — they only depend on this function's stdlib-shaped signature.
func RequireAdminSession(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				httpserver.WriteError(w, r, ErrInvalidToken)
				return
			}
			admin, err := svc.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				httpserver.WriteError(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

type adminUserDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func toDTO(u *AdminUser) adminUserDTO {
	return adminUserDTO{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt.Format(time.RFC3339)}
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func loginHandler(svc *Service, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		token, admin, err := svc.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		setSessionCookie(w, token, cookieSecure)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toDTO(admin))
	}
}

func logoutHandler(cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clearSessionCookie(w, cookieSecure)
		w.WriteHeader(http.StatusNoContent)
	}
}

func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := adminFromContext(r.Context())
		if !ok {
			httpserver.WriteError(w, r, ErrInvalidToken)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toDTO(admin))
	}
}

type createUserRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func createUserHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		admin, err := svc.CreateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toDTO(admin))
	}
}

type listUsersResponse struct {
	Users []adminUserDTO `json:"users"`
}

func listUsersHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := svc.ListUsers(r.Context())
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		resp := listUsersResponse{Users: make([]adminUserDTO, 0, len(users))}
		for i := range users {
			resp.Users = append(resp.Users, toDTO(&users[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func deleteUserHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, ok := adminFromContext(r.Context())
		if !ok {
			httpserver.WriteError(w, r, ErrInvalidToken)
			return
		}
		id := chi.URLParam(r, "id")
		if err := svc.DeleteUser(r.Context(), caller.ID, id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
