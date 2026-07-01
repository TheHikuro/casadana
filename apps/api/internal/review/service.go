package review

import (
	"context"
)

type Service struct {
	repo     Repository
	bookings BookingReader
	clock    Clock
}

func NewService(repo Repository, bookings BookingReader, clock Clock) *Service {
	return &Service{repo: repo, bookings: bookings, clock: clock}
}

type SubmitCommand struct {
	BookingID  string
	AuthorName string
	Rating     int
	Body       string
}

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
	return r, nil
}

func (s *Service) ListByVilla(ctx context.Context, slug string) ([]Review, error) {
	return s.repo.ListByVillaSlug(ctx, slug)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
