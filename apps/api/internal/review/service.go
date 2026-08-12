package review

import (
	"context"
	"fmt"
	"log/slog"
)

type Service struct {
	repo     Repository
	bookings BookingReader
	clock    Clock
	events   EventRecorder
}

func NewService(repo Repository, bookings BookingReader, clock Clock, events EventRecorder) *Service {
	return &Service{repo: repo, bookings: bookings, clock: clock, events: events}
}

type SubmitCommand struct {
	BookingID  string
	AuthorName string
	Rating     int
	Body       string
}

// Submit is the guest path: a review must be backed by a real booking of ours,
// and lands pending until an admin moderates it.
func (s *Service) Submit(ctx context.Context, cmd SubmitCommand) (*Review, error) {
	villaSlug, err := s.bookings.GetVillaSlug(ctx, cmd.BookingID)
	if err != nil {
		return nil, err
	}

	r, err := NewReview(NewReviewInput{
		BookingID:  cmd.BookingID,
		VillaSlug:  villaSlug,
		AuthorName: cmd.AuthorName,
		Rating:     cmd.Rating,
		Body:       cmd.Body,
		Now:        s.clock.Now(),
	})
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, r); err != nil {
		return nil, err
	}
	s.record(ctx, r.VillaSlug, fmt.Sprintf("Review by %s submitted", r.AuthorName))
	return r, nil
}

type CreateByAdminCommand struct {
	VillaSlug  string
	AuthorName string
	Rating     int
	Body       string
	Status     Status
	Meta       string
	Source     string
	Featured   bool
	Categories CategoryRatings
}

// CreateByAdmin adds a review with no booking behind it — typically one
// transcribed from an external platform. It defaults to approved.
func (s *Service) CreateByAdmin(ctx context.Context, cmd CreateByAdminCommand) (*Review, error) {
	r, err := NewAdminReview(NewAdminReviewInput{
		VillaSlug:  cmd.VillaSlug,
		AuthorName: cmd.AuthorName,
		Rating:     cmd.Rating,
		Body:       cmd.Body,
		Status:     cmd.Status,
		Meta:       cmd.Meta,
		Source:     cmd.Source,
		Featured:   cmd.Featured,
		Categories: cmd.Categories,
		Now:        s.clock.Now(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, r); err != nil {
		return nil, err
	}
	s.record(ctx, r.VillaSlug, fmt.Sprintf("Review by %s added", r.AuthorName))
	return r, nil
}

// ListByVilla is the public listing: approved reviews only. Pending and
// rejected reviews must never reach a guest.
func (s *Service) ListByVilla(ctx context.Context, slug string) ([]Review, error) {
	approved := StatusApproved
	return s.repo.ListByVillaAndStatus(ctx, slug, &approved)
}

// ListForAdmin lists every status by default; status narrows it.
func (s *Service) ListForAdmin(ctx context.Context, villaSlug string, status *Status) ([]Review, error) {
	if status != nil && !status.valid() {
		return nil, ErrInvalidPayload
	}
	return s.repo.ListByVillaAndStatus(ctx, villaSlug, status)
}

func (s *Service) Get(ctx context.Context, id string) (*Review, error) {
	return s.repo.Get(ctx, id)
}

// Update applies a partial moderation edit. Unknown id yields ErrNotFound.
func (s *Service) Update(ctx context.Context, id string, patch UpdatePatch) (*Review, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return nil, err
	}
	s.record(ctx, updated.VillaSlug, patchMessage(updated, patch))
	return updated, nil
}

// UpdateStatus is the common case of Update: moderate a single review.
func (s *Service) UpdateStatus(ctx context.Context, id string, status Status) (*Review, error) {
	return s.Update(ctx, id, UpdatePatch{Status: &status})
}

// Delete hard-deletes a review. Returns ErrNotFound if no row matched.
func (s *Service) Delete(ctx context.Context, id string) error {
	// Read first so the audit line can name the review being removed; a
	// missing row surfaces as ErrNotFound either way.
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.record(ctx, existing.VillaSlug, fmt.Sprintf("Review by %s deleted", existing.AuthorName))
	return nil
}

// Meta returns the villa's published rating, computed from its approved
// reviews. There is no setter: the figures move only when a review is added,
// moderated or removed. A villa with no approved reviews reads as a zero count
// rather than an error.
func (s *Service) Meta(ctx context.Context, villaSlug string) (ReviewMeta, error) {
	return s.repo.GetAggregate(ctx, villaSlug)
}

// patchMessage renders the audit line for a moderation edit, leading with the
// change an admin most likely came for.
func patchMessage(r *Review, patch UpdatePatch) string {
	switch {
	case patch.Status != nil:
		return fmt.Sprintf("Review by %s set to %s", r.AuthorName, *patch.Status)
	case patch.Featured != nil && *patch.Featured:
		return fmt.Sprintf("Review by %s featured", r.AuthorName)
	case patch.Featured != nil:
		return fmt.Sprintf("Review by %s unfeatured", r.AuthorName)
	default:
		return fmt.Sprintf("Review by %s edited", r.AuthorName)
	}
}

// record is best-effort: the activity log must never cost us a mutation the
// database already accepted.
func (s *Service) record(ctx context.Context, villaSlug, message string) {
	if s.events == nil {
		return
	}
	if err := s.events.Record(ctx, villaSlug, message); err != nil {
		slog.WarnContext(ctx, "review event recording failed", "villa_slug", villaSlug, "err", err.Error())
	}
}
