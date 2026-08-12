package audit

import (
	"context"
	"time"
)

type fakeRepo struct {
	saved    []Event
	saveErr  error
	listErr  error
	countErr error

	// last call recorded by List, so pagination tests can assert the maths
	// the service handed down to the repository.
	gotLimit  int
	gotOffset int
}

func (f *fakeRepo) Save(_ context.Context, e *Event) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, *e)
	return nil
}

func (f *fakeRepo) List(_ context.Context, villaSlug string, limit, offset int) ([]Event, error) {
	f.gotLimit, f.gotOffset = limit, offset
	if f.listErr != nil {
		return nil, f.listErr
	}
	matching := []Event{}
	for _, e := range f.saved {
		if e.VillaSlug == villaSlug {
			matching = append(matching, e)
		}
	}
	if offset >= len(matching) {
		return []Event{}, nil
	}
	end := offset + limit
	if end > len(matching) {
		end = len(matching)
	}
	return matching[offset:end], nil
}

func (f *fakeRepo) Count(_ context.Context, villaSlug string) (int, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}
	n := 0
	for _, e := range f.saved {
		if e.VillaSlug == villaSlug {
			n++
		}
	}
	return n, nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
