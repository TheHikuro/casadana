package audit

import "context"

type actorCtxKey struct{}

// WithActor stamps the acting admin's email onto the request context so that
// recorders further down the call chain can attribute the events they append.
// adminauth keeps its identity under an unexported key, so whoever owns the
// request has to hand the email over explicitly.
func WithActor(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, actorCtxKey{}, email)
}

// ActorFromContext returns the acting admin's email, or "" for an
// unauthenticated or unstamped context.
func ActorFromContext(ctx context.Context) string {
	email, _ := ctx.Value(actorCtxKey{}).(string)
	return email
}

// ActorResolver extracts the acting admin's email from a request-scoped
// context, returning "" when the context carries no identity.
type ActorResolver func(ctx context.Context) string

// Recorder satisfies the one-method EventRecorder port that the pricing and
// review slices each declare locally, binding the log to a fixed event type so
// callers only pass the villa and the message.
type Recorder struct {
	svc   *Service
	typ   EventType
	actor ActorResolver
}

// RecorderFor adapts svc to the EventRecorder shape for a single event type,
// e.g. RecorderFor(auditSvc, TypePricing).
func RecorderFor(svc *Service, typ EventType) *Recorder {
	return &Recorder{svc: svc, typ: typ, actor: ActorFromContext}
}

// WithActorResolver overrides how the actor's email is read off the context,
// for callers that keep the identity somewhere other than WithActor.
func (r *Recorder) WithActorResolver(resolve ActorResolver) *Recorder {
	r.actor = resolve
	return r
}

func (r *Recorder) Record(ctx context.Context, villaSlug, message string) error {
	_, err := r.svc.Record(ctx, RecordCommand{
		VillaSlug:  villaSlug,
		Type:       r.typ,
		Message:    message,
		ActorEmail: r.actor(ctx),
	})
	return err
}
