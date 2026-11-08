package client

import (
	"context"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Retry policies for the transient failure classes the App Platform API exhibits.
// Each is a k8s wait.Backoff (exponential: Duration×Factor per step, jittered up to
// +Jitter×duration to desynchronize parallel retries, capped at Cap, bounded by Steps).
// Approximate worst-case waits assume the upper jitter bound.
var (
	// ConflictBackoff smooths optimistic-lock (409) races on update. Grafana reconciles
	// provisioning resources asynchronously, so the ResourceVersion Terraform holds can go
	// stale between plan/apply and the write.
	// Sources: pkg/registry/apis/provisioning/controller/connection.go,
	// pkg/storage/unified/{apistore/store.go,sql/backend.go}.
	ConflictBackoff = wait.Backoff{Duration: 200 * time.Millisecond, Factor: 2, Jitter: 0.5, Cap: 2 * time.Second, Steps: 5} // ~200ms→1.6s, ≲4.5s total

	// DeleteBackoff covers the cross-resource referential-integrity race when a connection is
	// deleted right after the repository that references it (spec.connection.name). Grafana's
	// connection delete validator now ignores repositories that are themselves terminating
	// (grafana/grafana#126822), so this 422 is no longer the seconds-long wait for the repo's
	// finalizers to complete — what remains is a storage read-after-write window: the validator
	// decides at admission via a live ListByConnection, and that list can briefly not yet observe
	// the repository's freshly-written deletionTimestamp, still count it as a live reference, and
	// return 422. With exponential backoff the first retry fires quickly (~500ms) to catch the
	// common case where the window has already cleared, then grows to cover a laggier index path,
	// with the whole budget kept under ~10s so a genuinely permanent reference (a connection still
	// referenced by a LIVE repository) surfaces its real error promptly rather than hanging.
	DeleteBackoff = wait.Backoff{Duration: 500 * time.Millisecond, Factor: 2, Jitter: 0.5, Cap: 2 * time.Second, Steps: 4} // ~500ms→2s, ≲5.5s total

	// ReadBackoff covers transient server-side failures (5xx/429) so a single backend blip does not
	// fail an entire plan/refresh. Reads only sleep when a request actually errors, so the first
	// retry fires quickly (~500ms) to recover fast from a momentary blip and then backs off to give
	// a slower-clearing 5xx room mid-reconcile.
	ReadBackoff = wait.Backoff{Duration: 500 * time.Millisecond, Factor: 2, Jitter: 0.5, Cap: 2 * time.Second, Steps: 4} // ~500ms→2s, ≲5.5s total

	// DeletionWaitBackoff paces the poll that waits for an object to actually leave storage after
	// a DELETE is accepted (see ResourceConfig.WaitForDeletion). Unlike the retry backoffs above it
	// is a wait-for-state loop, not a retry-a-failing-call budget: each step re-reads the object and
	// keeps waiting while it still exists (finalizers running) or a read transiently 5xxs. The first
	// poll fires quickly to catch the common case where finalizers have already completed, then backs
	// off; the budget is bounded (~4-5 min worst case) so a stuck finalizer surfaces a clear timeout
	// rather than hanging Terraform indefinitely when the delete context carries no deadline.
	DeletionWaitBackoff = wait.Backoff{Duration: 1 * time.Second, Factor: 1.5, Jitter: 0.2, Cap: 10 * time.Second, Steps: 30}
)

// RetryWhile runs fn with exponential backoff (per backoff) until it succeeds, returns
// a non-retryable error (per retryable), the context is canceled, or the backoff's
// Steps are exhausted. It wraps k8s wait.ExponentialBackoffWithContext, which owns the
// context-aware sleep between attempts. On Steps exhaustion it surfaces fn's last
// (retryable) error rather than wait's sentinel ErrWaitTimeout, and on cancellation it
// surfaces the context error. A non-positive Steps is clamped to a single attempt so a
// misconfigured budget still runs fn once and surfaces its error (never a silent nil).
func RetryWhile(
	ctx context.Context,
	backoff wait.Backoff,
	retryable func(error) bool,
	fn func() error,
) error {
	if backoff.Steps < 1 {
		backoff.Steps = 1
	}

	var lastErr error
	err := wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		lastErr = fn()
		switch {
		case lastErr == nil:
			return true, nil
		case retryable(lastErr):
			return false, nil // transient — back off and retry
		default:
			return false, lastErr // permanent — stop and surface it
		}
	})
	if wait.Interrupted(err) {
		// Steps exhausted or context cancelled. Prefer the context error if the context
		// is actually done; otherwise surface fn's last error instead of ErrWaitTimeout.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return lastErr
	}

	return err
}

// RetryOnConflict smooths over optimistic-lock races with Grafana's unified storage.
// Provisioning resources can be reconciled asynchronously by Grafana controllers, so
// the ResourceVersion held by Terraform can go stale between plan/apply and the API
// write. Grafana updates enforce ResourceVersion matching, and deletes also resolve
// the current object and pass its ResourceVersion through the delete command, so both
// operations can legitimately return 409 conflicts during normal reconciliation.
// Sources:
// - pkg/registry/apis/provisioning/controller/connection.go
// - pkg/storage/unified/apistore/store.go
// - pkg/storage/unified/sql/backend.go
func RetryOnConflict(ctx context.Context, backoff wait.Backoff, fn func() error) error {
	return RetryWhile(ctx, backoff, apierrors.IsConflict, fn)
}

// referencedDependencyMarker is the substring Grafana's provisioning admission controller
// uses when a resource cannot be deleted because another resource still references it
// (e.g. deleting a connection while a repository still points at it via spec.connection.name).
// Source: apps/provisioning/pkg/connection/delete_validator.go
const referencedDependencyMarker = "referenced by"

// IsReferencedDependencyError reports whether err is a transient 422 Invalid error caused by
// a not-yet-observed cross-resource reference during teardown. On destroy, Terraform deletes
// the referencing resource (e.g. a repository) first; its DELETE returns 200 and stamps a
// deletionTimestamp, but the object lingers in storage until its finalizers complete. The
// connection delete validator ignores terminating repositories (grafana/grafana#126822), so
// this 422 only occurs when the validator's live ListByConnection has not yet observed that
// freshly-written deletionTimestamp and still counts the repository as a live reference.
// Retrying lets the read catch up; a connection still referenced by a genuinely LIVE
// repository keeps failing until the attempt budget is exhausted, surfacing the real error.
func IsReferencedDependencyError(err error) bool {
	return apierrors.IsInvalid(err) &&
		strings.Contains(strings.ToLower(err.Error()), referencedDependencyMarker)
}

// IsRetryableDeleteError reports whether a failed delete is worth retrying: either an
// optimistic-lock conflict (409) or a transient referential-integrity race (422).
func IsRetryableDeleteError(err error) bool {
	return apierrors.IsConflict(err) || IsReferencedDependencyError(err)
}

// IsRetryableServerError reports whether err is a transient server-side failure
// (HTTP 5xx, timeouts, or throttling) that is worth retrying on read. The provisioning
// backend intermittently returns 500s on GET while reconciling, which would otherwise
// fail an entire plan/refresh.
func IsRetryableServerError(err error) bool {
	return apierrors.IsInternalError(err) || // 500 internal error
		apierrors.IsServiceUnavailable(err) || // 503 service unavailable
		apierrors.IsServerTimeout(err) || // reason=ServerTimeout (sent as HTTP 500)
		apierrors.IsTimeout(err) || // reason=Timeout / HTTP 504 gateway timeout
		apierrors.IsTooManyRequests(err) // 429 too many requests
}

func AnyOfErr(errFuncs ...func(err error) bool) func(err error) bool {
	return func(err error) bool {
		for _, f := range errFuncs {
			if f(err) {
				return true
			}
		}

		return false
	}
}
