package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/TheHikuro/casadana/internal/adminauth"
	"github.com/TheHikuro/casadana/internal/audit"
	"github.com/TheHikuro/casadana/internal/booking"
	"github.com/TheHikuro/casadana/internal/db"
	"github.com/TheHikuro/casadana/internal/openapi"
	"github.com/TheHikuro/casadana/internal/platform/config"
	"github.com/TheHikuro/casadana/internal/platform/email"
	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/logger"
	"github.com/TheHikuro/casadana/internal/platform/postgres"
	"github.com/TheHikuro/casadana/internal/pricing"
	"github.com/TheHikuro/casadana/internal/review"
	"github.com/TheHikuro/casadana/internal/villaslug"
)

// Budget for the villa-page review form, per caller IP.
//
// A visitor writes one review per stay, so anything above a handful in an hour
// is a script or a mistake. The ceiling is deliberately generous: the cost of
// being too tight is a real guest silently unable to post, while the cost of
// being too loose is only a longer moderation queue — nothing a visitor submits
// reaches the site before an admin approves it.
const (
	publicReviewLimit  = 5
	publicReviewWindow = time.Hour
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type slugAllowlist struct{}

func (slugAllowlist) IsKnown(slug string) bool { return villaslug.IsKnown(slug) }

// bookingReaderAdapter lets the review module look up a booking's villa_slug
// via the booking service without a direct module-to-module import.
type bookingReaderAdapter struct{ svc *booking.Service }

func (a bookingReaderAdapter) GetVillaSlug(ctx context.Context, id string) (string, error) {
	b, err := a.svc.Get(ctx, id)
	if err != nil {
		if errors.Is(err, booking.ErrNotFound) {
			return "", review.ErrBookingNotFound
		}
		return "", err
	}
	return b.VillaSlug, nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err.Error())
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx := context.Background()
	pool, err := postgres.Open(ctx, cfg.DB.DSN())
	if err != nil {
		log.Error("postgres open failed", "err", err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.MigrateOnBoot {
		if err := postgres.MigrateUp(pool, db.Migrations, "migrations"); err != nil {
			log.Error("migrate failed", "err", err.Error())
			os.Exit(1)
		}
		log.Info("migrations applied")
	}

	mailer := email.NewMailer(cfg.ResendKey, cfg.MailFrom, cfg.AdminNotifyEmail)
	adminAuthSvc := adminauth.NewService(adminauth.NewPgRepo(pool), realClock{}, cfg.JWTSecret)
	bookingSvc := booking.NewService(
		booking.NewPgRepo(pool),
		booking.NewResendMailer(mailer),
		slugAllowlist{},
		realClock{},
	)
	auditSvc := audit.NewService(audit.NewPgRepo(pool), realClock{})
	// The audit log attributes changes to whoever is behind the admin session;
	// RequireAdminSession puts them on the request context.
	auditActor := audit.ActorResolver(adminauth.AdminEmailFromContext)

	pricingSvc := pricing.NewService(
		pricing.NewPgRepo(pool),
		slugAllowlist{},
		audit.RecorderFor(auditSvc, audit.TypePricing).WithActorResolver(auditActor),
	)
	reviewSvc := review.NewService(
		review.NewPgRepo(pool),
		bookingReaderAdapter{svc: bookingSvc},
		slugAllowlist{},
		realClock{},
		audit.RecorderFor(auditSvc, audit.TypeReview).WithActorResolver(auditActor),
	)

	requireAdmin := adminauth.RequireAdminSession(adminAuthSvc)

	r := httpserver.NewRouter(log, cfg.WebOrigin)
	openapi.Mount(r)
	adminauth.Mount(r, adminAuthSvc, cfg.CookieSecure)
	booking.Mount(r, bookingSvc, requireAdmin)
	pricing.Mount(r, pricingSvc, requireAdmin)
	review.Mount(r, reviewSvc, requireAdmin, httpserver.RateLimit(publicReviewLimit, publicReviewWindow))
	audit.Mount(r, auditSvc, requireAdmin)

	if err := httpserver.Run(r, cfg.Port, log); err != nil {
		log.Error("server crashed", "err", err.Error())
		os.Exit(1)
	}
}
