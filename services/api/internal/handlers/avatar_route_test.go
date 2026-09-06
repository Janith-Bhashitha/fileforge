package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

// Avatars are served by storage key, and storage keys are namespaced per
// owner - so they contain slashes. A single chi path parameter ("{key}")
// matches one segment only, which made every newly uploaded avatar 404
// while older flat-key avatars kept working: the worst kind of regression,
// silent and partial.
//
// This pins the routing shape rather than the handler, because the handler
// was never the problem. It builds the key with storage.ObjectKey so that
// changing the key layout again fails here instead of in production.
func TestAvatarRouteCapturesMultiSegmentKey(t *testing.T) {
	key := storage.ObjectKey(uuid.New(), ".png")
	if !strings.Contains(key, "/") {
		t.Fatalf("expected a namespaced key containing a slash, got %q", key)
	}

	var captured string
	r := chi.NewRouter()
	r.Get("/avatars/*", func(w http.ResponseWriter, req *http.Request) {
		captured = chi.URLParam(req, "*")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/avatars/"+key, nil))

	if rec.Code == http.StatusNotFound {
		t.Fatalf("route did not match /avatars/%s", key)
	}
	if captured != key {
		t.Errorf("captured key = %q, want %q", captured, key)
	}
}
