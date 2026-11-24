package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore implements Store for testing.
type mockStore struct {
	scriptResults map[string]interface{}
}

func (m *mockStore) EvalScript(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return m.scriptResults["eval"], nil
}

func (m *mockStore) ScriptLoad(ctx context.Context, script string) (string, error) {
	return "test_sha", nil
}

func (m *mockStore) ScriptEvalSHA(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	if result, ok := m.scriptResults["evalsha"]; ok {
		return result, nil
	}
	return m.scriptResults["eval"], nil
}

func TestRedisLimiter_Allow_Allowed(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{
		scriptResults: map[string]interface{}{
			"eval": []interface{}{int64(1), int64(1), int64(10), int64(9), int64(60000), int64(0)},
		},
	}

	limiter := NewRedisLimiter(store)
	limiter.sha = "test_sha" // bypass LoadScript

	decision, err := limiter.Allow(ctx, "test:key", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(1), decision.Current)
	assert.Equal(t, 10, decision.Limit)
	assert.Equal(t, int64(9), decision.Remaining)
	assert.Equal(t, time.Minute, decision.ResetAfter)
	assert.Equal(t, time.Duration(0), decision.RetryAfter)
}

func TestRedisLimiter_Allow_Denied(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{
		scriptResults: map[string]interface{}{
			"eval": []interface{}{int64(0), int64(10), int64(10), int64(0), int64(60000), int64(60000)},
		},
	}

	limiter := NewRedisLimiter(store)
	limiter.sha = "test_sha"

	decision, err := limiter.Allow(ctx, "test:key", 10, time.Minute)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, int64(10), decision.Current)
	assert.Equal(t, 10, decision.Limit)
	assert.Equal(t, int64(0), decision.Remaining)
	assert.Equal(t, time.Minute, decision.ResetAfter)
	assert.Equal(t, time.Minute, decision.RetryAfter)
}

func TestRedisLimiter_LoadScript(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}

	limiter := NewRedisLimiter(store)
	err := limiter.LoadScript(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test_sha", limiter.sha)
}

func TestParseDecision(t *testing.T) {
	tests := []struct {
		name     string
		result   interface{}
		expected Decision
		wantErr  bool
	}{
		{
			name: "allowed",
			result: []interface{}{int64(1), int64(5), int64(10), int64(5), int64(30000), int64(0)},
			expected: Decision{
				Allowed:    true,
				Current:    5,
				Limit:      10,
				Remaining:  5,
				ResetAfter: 30 * time.Second,
				RetryAfter: 0,
			},
			wantErr: false,
		},
		{
			name: "denied",
			result: []interface{}{int64(0), int64(10), int64(10), int64(0), int64(60000), int64(60000)},
			expected: Decision{
				Allowed:    false,
				Current:    10,
				Limit:      10,
				Remaining:  0,
				ResetAfter: time.Minute,
				RetryAfter: time.Minute,
			},
			wantErr: false,
		},
		{
			name:    "invalid result type",
			result:  "not an array",
			wantErr: true,
		},
		{
			name:    "wrong array length",
			result:  []interface{}{int64(1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := parseDecision(tt.result)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, decision)
			}
		})
	}
}

func TestDecisionToHeaders(t *testing.T) {
	d := Decision{
		Allowed:    true,
		Current:    5,
		Limit:      100,
		Remaining:  95,
		ResetAfter: 45 * time.Second,
		RetryAfter: 0,
	}

	headers := DecisionToHeaders(d)
	assert.Equal(t, "100", headers["RateLimit-Limit"])
	assert.Equal(t, "95", headers["RateLimit-Remaining"])
	assert.Equal(t, "45", headers["RateLimit-Reset"])
	assert.Equal(t, "100", headers["X-RateLimit-Limit"])
	assert.Equal(t, "95", headers["X-RateLimit-Remaining"])
	// X-RateLimit-Reset is unix timestamp, just check it's present
	assert.NotEmpty(t, headers["X-RateLimit-Reset"])
}

func TestDecisionToRetryAfter(t *testing.T) {
	d := Decision{RetryAfter: 30 * time.Second}
	assert.Equal(t, "30", DecisionToRetryAfter(d))

	d = Decision{RetryAfter: 0}
	assert.Equal(t, "0", DecisionToRetryAfter(d))
}