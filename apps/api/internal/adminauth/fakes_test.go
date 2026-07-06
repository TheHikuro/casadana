package adminauth

import (
	"context"
	"time"
)

type fakeRepo struct {
	users   []AdminUser
	saveErr error
}

func (f *fakeRepo) Save(_ context.Context, u *AdminUser) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	for _, existing := range f.users {
		if existing.Email == u.Email {
			return ErrEmailTaken
		}
	}
	f.users = append(f.users, *u)
	return nil
}

func (f *fakeRepo) FindByEmail(_ context.Context, email string) (*AdminUser, error) {
	for i := range f.users {
		if f.users[i].Email == email {
			u := f.users[i]
			return &u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) FindByID(_ context.Context, id string) (*AdminUser, error) {
	for i := range f.users {
		if f.users[i].ID == id {
			u := f.users[i]
			return &u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) List(_ context.Context) ([]AdminUser, error) {
	return f.users, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	for i, u := range f.users {
		if u.ID == id {
			f.users = append(f.users[:i], f.users[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }
