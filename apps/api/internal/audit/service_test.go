package audit

import (
	"context"
	"errors"
	"testing"
)

func newSvc(repo Repository) *Service {
	return NewService(repo, fixedClock{t: d("2026-08-01")})
}

func TestRecord_Happy(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo)

	e, err := svc.Record(context.Background(), RecordCommand{
		VillaSlug:  "casadana",
		Type:       TypePricing,
		Message:    "Prix mis à jour",
		ActorEmail: "admin@casadana.fr",
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if e.Type != TypePricing || e.VillaSlug != "casadana" {
		t.Errorf("unexpected event: %+v", e)
	}
	if !e.CreatedAt.Equal(d("2026-08-01")) {
		t.Errorf("CreatedAt = %v, want the clock's now", e.CreatedAt)
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved count = %d, want 1", len(repo.saved))
	}
}

func TestRecord_InvalidPayload(t *testing.T) {
	tests := []struct {
		name string
		cmd  RecordCommand
	}{
		{"unknown villa", RecordCommand{VillaSlug: "ghost", Type: TypePricing, Message: "x"}},
		{"unknown type", RecordCommand{VillaSlug: "casadana", Type: "nope", Message: "x"}},
		{"empty message", RecordCommand{VillaSlug: "casadana", Type: TypePricing}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := newSvc(repo).Record(context.Background(), tt.cmd)
			if err != ErrInvalidPayload {
				t.Fatalf("err = %v, want ErrInvalidPayload", err)
			}
			if len(repo.saved) != 0 {
				t.Errorf("saved count = %d, want 0", len(repo.saved))
			}
		})
	}
}

func TestRecord_RepoErrorIsWrapped(t *testing.T) {
	boom := errors.New("boom")
	svc := newSvc(&fakeRepo{saveErr: boom})

	_, err := svc.Record(context.Background(), RecordCommand{
		VillaSlug: "casadana", Type: TypeSystem, Message: "x",
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap boom", err)
	}
}

func TestList_Pagination(t *testing.T) {
	tests := []struct {
		name        string
		page, limit int
		wantLimit   int
		wantOffset  int
	}{
		{"defaults", 0, 0, 20, 0},
		{"page below 1 floors to 1", -3, 5, 5, 0},
		{"second page", 2, 5, 5, 5},
		{"limit clamped to 100", 3, 250, 100, 200},
		{"limit below 1 defaults to 20", 2, 0, 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := newSvc(repo)

			if _, _, err := svc.List(context.Background(), ListQuery{
				VillaSlug: "casadana", Page: tt.page, Limit: tt.limit,
			}); err != nil {
				t.Fatalf("List: %v", err)
			}
			if repo.gotLimit != tt.wantLimit || repo.gotOffset != tt.wantOffset {
				t.Errorf("repo got (limit=%d, offset=%d), want (limit=%d, offset=%d)",
					repo.gotLimit, repo.gotOffset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestList_FiltersByVillaAndCounts(t *testing.T) {
	repo := &fakeRepo{saved: []Event{
		{ID: "1", VillaSlug: "casadana", Type: TypePricing},
		{ID: "2", VillaSlug: "casacasay", Type: TypeReview},
		{ID: "3", VillaSlug: "casadana", Type: TypeReview},
	}}
	svc := newSvc(repo)

	events, total, err := svc.List(context.Background(), ListQuery{VillaSlug: "casadana"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("len = %d, want 2", len(events))
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestList_UnknownVilla(t *testing.T) {
	repo := &fakeRepo{}
	_, _, err := newSvc(repo).List(context.Background(), ListQuery{VillaSlug: "ghost"})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestRecorderFor_UsesFixedTypeAndContextActor(t *testing.T) {
	repo := &fakeRepo{}
	rec := RecorderFor(newSvc(repo), TypePricing)

	ctx := WithActor(context.Background(), "admin@casadana.fr")
	if err := rec.Record(ctx, "casadana", "Prix mis à jour"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("saved count = %d, want 1", len(repo.saved))
	}
	got := repo.saved[0]
	if got.Type != TypePricing {
		t.Errorf("Type = %q, want pricing", got.Type)
	}
	if got.ActorEmail != "admin@casadana.fr" {
		t.Errorf("ActorEmail = %q, want admin@casadana.fr", got.ActorEmail)
	}
}

func TestRecorderFor_ActorFallsBackToEmpty(t *testing.T) {
	repo := &fakeRepo{}
	rec := RecorderFor(newSvc(repo), TypeReview)

	if err := rec.Record(context.Background(), "casadana", "Avis supprimé"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if repo.saved[0].ActorEmail != "" {
		t.Errorf("ActorEmail = %q, want empty", repo.saved[0].ActorEmail)
	}
}

func TestRecorderFor_ReturnsErrorForCallerToLog(t *testing.T) {
	rec := RecorderFor(newSvc(&fakeRepo{}), TypeReview)

	if err := rec.Record(context.Background(), "ghost", "x"); err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestRecorder_WithActorResolver(t *testing.T) {
	repo := &fakeRepo{}
	rec := RecorderFor(newSvc(repo), TypeOwner).
		WithActorResolver(func(context.Context) string { return "cron@casadana.fr" })

	if err := rec.Record(context.Background(), "casacasay", "Propriétaire mis à jour"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if repo.saved[0].ActorEmail != "cron@casadana.fr" {
		t.Errorf("ActorEmail = %q, want cron@casadana.fr", repo.saved[0].ActorEmail)
	}
}
