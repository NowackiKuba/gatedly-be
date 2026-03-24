package evaluation

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/analytics"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/flagrule"
)

// cacheKey uniquely identifies a rule in the cache.
// We need both envID and flagKey because the same flag key
// can exist in multiple environments with different rules.
type cacheKey struct {
	envID   uuid.UUID
	flagKey string
}

// Service holds the cache and owns the background goroutine.
type Service struct {
	repo     flagrule.Repository
	cache    map[cacheKey]domain.FlagRule
	mu       sync.RWMutex  // protects cache
	stopCh   chan struct{} // closed on shutdown → goroutine exits
	reloadCh chan struct{} // send a value → goroutine reloads immediately

	analyticsRollups *analytics.Service
}

// New creates a Service, loads the cache for the first time,
// and starts the background refresh goroutine.
func New(repo flagrule.Repository, analyticsRollups *analytics.Service) *Service {
	s := &Service{
		repo:   repo,
		cache:  make(map[cacheKey]domain.FlagRule),
		stopCh: make(chan struct{}),
		// Buffer of 1 so the sender never blocks if a reload is already pending.
		reloadCh:         make(chan struct{}, 1),
		analyticsRollups: analyticsRollups,
	}

	// Load rules once before we return so the first request is never a cache miss.
	if err := s.reload(context.Background()); err != nil {
		slog.Error("evaluation: initial cache load failed", "error", err)
	}

	s.startRefresh()
	return s
}

// Stop signals the background goroutine to exit.
// Call this during application shutdown.
func (s *Service) Stop() {
	close(s.stopCh)
}

// InvalidateCache triggers an immediate cache reload from outside the package.
// Call this from the FlagRule CRUD service whenever a rule is created/updated/deleted.
// The send is non-blocking — if a reload is already queued (buffer is full) we skip.
func (s *Service) InvalidateCache() {
	select {
	case s.reloadCh <- struct{}{}:
	default:
	}
}

// startRefresh runs a goroutine that refreshes the cache every 30 seconds
// or immediately when InvalidateCache() is called.
func (s *Service) startRefresh() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			// select blocks until one of the cases is ready.
			// Go picks a random ready case — there is no priority.
			select {
			case <-ticker.C:
				// Regular interval reload.
				if err := s.reload(context.Background()); err != nil {
					slog.Error("evaluation: cache refresh failed", "error", err)
				}

			case <-s.reloadCh:
				// Someone changed a rule in the dashboard — reload right away
				// so the next evaluation request sees the new value.
				if err := s.reload(context.Background()); err != nil {
					slog.Error("evaluation: cache reload failed", "error", err)
				}

			case <-s.stopCh:
				// Application is shutting down — exit the goroutine cleanly.
				return
			}
		}
	}()
}

// reload fetches all rules from the database and replaces the cache atomically.
func (s *Service) reload(ctx context.Context) error {
	rules, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("evaluation reload: %w", err)
	}

	// Build a brand new map — safer than updating in place.
	next := make(map[cacheKey]domain.FlagRule, len(rules))
	for _, r := range rules {
		if r.Flag == nil {
			continue // skip orphaned rules or missing preload
		}
		next[cacheKey{envID: r.EnvironmentID, flagKey: r.Flag.Key}] = r
	}

	// Lock for writing — this blocks all readers until the swap is done.
	// The critical section is tiny (just a pointer swap) so contention is minimal.
	s.mu.Lock()
	s.cache = next
	s.mu.Unlock()

	return nil
}

// get reads a single rule from the cache.
// RLock allows multiple goroutines to read concurrently —
// only reload() uses a write lock, everything else can read in parallel.
func (s *Service) get(envID uuid.UUID, flagKey string) (domain.FlagRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rule, ok := s.cache[cacheKey{envID: envID, flagKey: flagKey}]
	return rule, ok
}

// ── public evaluation methods ─────────────────────────────────────────────────

// Evaluate returns the result for a single flag.
func (s *Service) Evaluate(ctx context.Context, envID uuid.UUID, flagKey, userID string, attributes map[string]any) (EvaluationResult, error) {
	rule, ok := s.get(envID, flagKey)
	if !ok {
		return EvaluationResult{}, fmt.Errorf("flag %q not found in env %s", flagKey, envID)
	}

	result := Evaluate(rule, userID, attributes)
	result.FlagKey = flagKey

	// Record analytics rollups. Never block evaluation if analytics fails.
	if s.analyticsRollups != nil && rule.Flag != nil {
		// Use the same UTC date for all increments within this request.
		now := time.Now().UTC()
		date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		projectID := rule.Flag.ProjectID
		if err := s.analyticsRollups.IncrementEnvEvaluationsDaily(ctx, projectID, envID, date, 1); err != nil {
			slog.Error("analytics env rollup increment failed", "env_id", envID, "project_id", projectID, "err", err)
		}
		if err := s.analyticsRollups.IncrementFlagEvaluationsDaily(ctx, projectID, rule.FlagID, date, 1); err != nil {
			slog.Error("analytics flag rollup increment failed", "flag_id", rule.FlagID, "project_id", projectID, "err", err)
		}
	}

	return result, nil
}

// EvaluateBatch evaluates multiple flags in parallel using one goroutine per flag.
// Results are written at a fixed index so no mutex is needed on the slice —
// each goroutine owns its own slot.
func (s *Service) EvaluateBatch(ctx context.Context, envID uuid.UUID, flagKeys []string, userID string, attributes map[string]any) []EvaluationResult {
	results := make([]EvaluationResult, len(flagKeys))

	var wg sync.WaitGroup

	var date time.Time
	if s.analyticsRollups != nil {
		now := time.Now().UTC()
		date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	for i, key := range flagKeys {
		wg.Add(1)

		// Pass i and key as arguments — if we closed over the loop variables
		// directly, every goroutine would see the last value of i and key
		// by the time they run (classic Go loop-closure gotcha).
		go func(i int, key string) {
			defer wg.Done()

			rule, ok := s.get(envID, key)
			if !ok {
				results[i] = EvaluationResult{FlagKey: key, Enabled: false, Reason: string(ReasonDisabled)}
				return
			}

			result := Evaluate(rule, userID, attributes)
			result.FlagKey = key
			results[i] = result

			// Record analytics rollups. Never block evaluation if analytics fails.
			if s.analyticsRollups != nil && rule.Flag != nil {
				projectID := rule.Flag.ProjectID
				if err := s.analyticsRollups.IncrementEnvEvaluationsDaily(ctx, projectID, envID, date, 1); err != nil {
					slog.Error("analytics env rollup increment failed", "env_id", envID, "project_id", projectID, "err", err)
				}
				if err := s.analyticsRollups.IncrementFlagEvaluationsDaily(ctx, projectID, rule.FlagID, date, 1); err != nil {
					slog.Error("analytics flag rollup increment failed", "flag_id", rule.FlagID, "project_id", projectID, "err", err)
				}
			}
		}(i, key)
	}

	// Wait for every goroutine to finish before returning.
	wg.Wait()
	return results
}
