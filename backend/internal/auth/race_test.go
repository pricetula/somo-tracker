package auth

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestHandler_Register_ConcurrentSameSessionRef fires two concurrent
// registration requests using the *same* session_ref (i.e. the same
// intermediate session token). Exactly one must succeed (204); the other
// must fail, because the IST is meant to be single-use. This guards against
// a check-then-act race in the IST consumption logic (e.g. GET followed by
// a separate DEL, rather than an atomic GETDEL/Lua script).
func TestHandler_Register_ConcurrentSameSessionRef(t *testing.T) {
	h := newHandlerTestHarness(t)

	sessionRef := "550e8400-e29b-41d4-a716-446655440099"
	seedIST(t, h, sessionRef, "ist_race", "race@example.com")

	const attempts = 10
	var successCount int32
	var wg sync.WaitGroup
	statusCodes := make([]int, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
				SchoolName: "Race School",
				SessionRef: sessionRef,
				FullName:   "Race User",
			})
			statusCodes[idx] = resp.StatusCode
			if resp.StatusCode == fiber.StatusNoContent {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful registration out of %d concurrent attempts, got %d; status codes: %v",
			attempts, successCount, statusCodes)
	}
}
