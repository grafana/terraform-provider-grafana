package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
)

// errBoom is a sentinel non-retryable error used across the retry tests.
var errBoom = errors.New("boom")

// noSleepBackoff exercises the retry control flow without real waits: Duration 0 (with
// Steps n) means wait.ExponentialBackoffWithContext never sleeps, so tests stay fast and
// deterministic regardless of the jitter the production policies use.
func noSleepBackoff(steps int) wait.Backoff {
	return wait.Backoff{Steps: steps}
}

func TestRetryOnConflict(t *testing.T) {
	t.Run("retries conflict until success", func(t *testing.T) {
		attempts := 0

		err := RetryOnConflict(context.Background(), noSleepBackoff(3), func() error {
			attempts++
			if attempts < 3 {
				return apierrors.NewConflict(k8sschema.GroupResource{Group: "grafana.app", Resource: "tests"}, "test", nil)
			}

			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 3, attempts)
	})

	t.Run("stops on non-conflict", func(t *testing.T) {
		attempts := 0

		err := RetryOnConflict(context.Background(), noSleepBackoff(3), func() error {
			attempts++
			return errBoom
		})

		require.ErrorIs(t, err, errBoom)
		require.Equal(t, 1, attempts)
	})

	t.Run("respects context cancellation while waiting", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		errCh := make(chan error, 1)
		go func() {
			// A real Duration forces an actual sleep between attempts, which the
			// cancellation must interrupt.
			errCh <- RetryOnConflict(ctx, wait.Backoff{Duration: time.Second, Steps: 3}, func() error {
				attempts++
				return apierrors.NewConflict(k8sschema.GroupResource{Group: "grafana.app", Resource: "tests"}, "test", nil)
			})
		}()

		time.Sleep(10 * time.Millisecond)
		cancel()

		err := <-errCh
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, attempts)
	})
}

func TestRetryWhile(t *testing.T) {
	t.Run("retries while predicate reports retryable", func(t *testing.T) {
		attempts := 0
		err := RetryWhile(context.Background(), noSleepBackoff(4), func(error) bool { return true }, func() error {
			attempts++
			if attempts < 3 {
				return errBoom
			}
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 3, attempts)
	})

	t.Run("stops immediately when predicate reports non-retryable", func(t *testing.T) {
		attempts := 0
		err := RetryWhile(context.Background(), noSleepBackoff(4), func(error) bool { return false }, func() error {
			attempts++
			return errBoom
		})

		require.ErrorIs(t, err, errBoom)
		require.Equal(t, 1, attempts)
	})

	t.Run("returns last error after exhausting attempts", func(t *testing.T) {
		attempts := 0
		err := RetryWhile(context.Background(), noSleepBackoff(3), func(error) bool { return true }, func() error {
			attempts++
			return errBoom
		})

		require.ErrorIs(t, err, errBoom)
		require.Equal(t, 3, attempts)
	})

	// A non-positive Steps budget must still run fn once and surface its error, never
	// return nil (or wait's ErrWaitTimeout) without calling fn at all.
	t.Run("treats non-positive steps as a single attempt", func(t *testing.T) {
		for _, n := range []int{0, -3} {
			attempts := 0
			err := RetryWhile(context.Background(), noSleepBackoff(n), func(error) bool { return true }, func() error {
				attempts++
				return errBoom
			})

			require.ErrorIs(t, err, errBoom, "steps=%d should surface fn's error", n)
			require.Equal(t, 1, attempts, "steps=%d should run exactly once", n)
		}
	})
}

// backoffWorstCase returns the maximum total time a policy can spend sleeping across all
// of its retries: every sleep capped at Cap and inflated by the upper jitter bound. There
// are Steps-1 sleeps (the final attempt is not followed by a wait).
func backoffWorstCase(b wait.Backoff) time.Duration {
	var total time.Duration
	d := b.Duration
	for i := 0; i < b.Steps-1; i++ {
		step := d
		if b.Cap > 0 && step > b.Cap {
			step = b.Cap
		}
		total += step + time.Duration(float64(step)*b.Jitter)
		d = time.Duration(float64(d) * b.Factor)
	}
	return total
}

// TestRetryBackoffPolicies guards the production retry policies against typos: each must
// run at least once, grow an exponential/jittered/capped delay when it retries, and keep
// its worst-case total wait bounded so a transient failure never stalls an operation.
func TestRetryBackoffPolicies(t *testing.T) {
	// A transient failure should never make an operation wait longer than this in total.
	const maxBudget = 10 * time.Second

	for name, b := range map[string]wait.Backoff{
		"conflict": ConflictBackoff,
		"delete":   DeleteBackoff,
		"read":     ReadBackoff,
	} {
		t.Run(name, func(t *testing.T) {
			require.GreaterOrEqual(t, b.Steps, 1, "must run at least once")
			require.Greater(t, b.Duration, time.Duration(0), "needs a positive base delay")
			require.Greater(t, b.Factor, 1.0, "factor must grow the delay")
			require.Greater(t, b.Jitter, 0.0, "jitter desynchronizes parallel retries")
			require.GreaterOrEqual(t, b.Cap, b.Duration, "cap must not be below the base delay")
			require.LessOrEqual(t, backoffWorstCase(b), maxBudget, "worst-case retry budget must stay within %s", maxBudget)
		})
	}
}

func referencedByConnectionError() error {
	return apierrors.NewInvalid(
		k8sschema.GroupKind{Group: "provisioning.grafana.app", Kind: "Connection"},
		"my-connection",
		field.ErrorList{field.Forbidden(
			field.NewPath("metadata", "name"),
			"cannot delete connection: referenced by 1 repository(s): [my-repo]",
		)},
	)
}

func TestIsReferencedDependencyError(t *testing.T) {
	gk := k8sschema.GroupKind{Group: "provisioning.grafana.app", Kind: "Connection"}
	gr := k8sschema.GroupResource{Group: gk.Group, Resource: "connections"}

	require.True(t, IsReferencedDependencyError(referencedByConnectionError()))

	// A different Invalid error (genuine validation failure) must NOT be retried.
	otherInvalid := apierrors.NewInvalid(gk, "my-connection", field.ErrorList{
		field.Required(field.NewPath("spec", "title"), "title is required"),
	})
	require.False(t, IsReferencedDependencyError(otherInvalid))

	require.False(t, IsReferencedDependencyError(apierrors.NewConflict(gr, "my-connection", nil)))
	require.False(t, IsReferencedDependencyError(apierrors.NewNotFound(gr, "my-connection")))
	require.False(t, IsReferencedDependencyError(nil))
	require.False(t, IsReferencedDependencyError(context.Canceled))
}

func TestIsRetryableDeleteError(t *testing.T) {
	gr := k8sschema.GroupResource{Group: "provisioning.grafana.app", Resource: "connections"}

	require.True(t, IsRetryableDeleteError(apierrors.NewConflict(gr, "my-connection", nil)))
	require.True(t, IsRetryableDeleteError(referencedByConnectionError()))
	require.False(t, IsRetryableDeleteError(apierrors.NewNotFound(gr, "my-connection")))
	require.False(t, IsRetryableDeleteError(apierrors.NewBadRequest("nope")))
	require.False(t, IsRetryableDeleteError(nil))
}

func TestIsRetryableServerError(t *testing.T) {
	gr := k8sschema.GroupResource{Group: "provisioning.grafana.app", Resource: "repositories"}

	require.True(t, IsRetryableServerError(apierrors.NewInternalError(errors.New("boom"))))
	require.True(t, IsRetryableServerError(apierrors.NewServiceUnavailable("down")))
	require.True(t, IsRetryableServerError(apierrors.NewServerTimeout(gr, "get", 1)))
	require.True(t, IsRetryableServerError(apierrors.NewTooManyRequestsError("slow down")))

	require.False(t, IsRetryableServerError(apierrors.NewNotFound(gr, "my-repo")))
	require.False(t, IsRetryableServerError(apierrors.NewBadRequest("bad")))
	require.False(t, IsRetryableServerError(referencedByConnectionError()))
	require.False(t, IsRetryableServerError(nil))
}
