package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func call(h http.HandlerFunc, target string) int {
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec.Code
}

func TestWorkSucceedsByDefault(t *testing.T) {
	errorRate.Store(0)
	if code := call(work, "/checkout"); code != http.StatusOK {
		t.Fatalf("got %d", code)
	}
}

func TestChaosInjectsErrors(t *testing.T) {
	if code := call(chaos, "/admin/chaos?error_rate=1"); code != http.StatusOK {
		t.Fatalf("chaos: got %d", code)
	}
	if code := call(work, "/checkout"); code != http.StatusInternalServerError {
		t.Fatalf("work with error_rate=1: got %d", code)
	}
	call(chaos, "/admin/chaos?error_rate=0")
}

func TestChaosRejectsBadInput(t *testing.T) {
	for _, q := range []string{"error_rate=2", "error_rate=x", "latency_ms=-1"} {
		if code := call(chaos, "/admin/chaos?"+q); code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", q, code)
		}
	}
}
