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

// Storage keys are namespaced per owner, so they contain slashes. A single
// chi path parameter matches one segment only, which 404'd every new avatar
// while older flat-key ones kept working. This pins the routing shape, and
// builds the key with storage.ObjectKey so a layout change fails here.
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
