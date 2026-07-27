package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetHandlerRoutesAPIRequestsToFileHandler(t *testing.T) {
	apiCalled := false
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("frontend fallback must not receive /api requests")
	})

	request := httptest.NewRequest(http.MethodGet, "/api/file/tab-1", nil)
	response := httptest.NewRecorder()
	NewAssetHandler(api, fallback).ServeHTTP(response, request)

	if !apiCalled || response.Code != http.StatusNoContent {
		t.Fatalf("apiCalled=%v status=%d", apiCalled, response.Code)
	}
}

func TestAssetHandlerDelegatesFrontendAssets(t *testing.T) {
	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusAccepted)
	})

	request := httptest.NewRequest(http.MethodGet, "/assets/index.js", nil)
	response := httptest.NewRecorder()
	NewAssetHandler(http.NotFoundHandler(), fallback).ServeHTTP(response, request)

	if !fallbackCalled || response.Code != http.StatusAccepted {
		t.Fatalf("fallbackCalled=%v status=%d", fallbackCalled, response.Code)
	}
}

func TestAssetHandlerDoesNotTreatAPIPrefixLookalikeAsAPI(t *testing.T) {
	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/apiary", nil)
	response := httptest.NewRecorder()
	NewAssetHandler(http.NotFoundHandler(), fallback).ServeHTTP(response, request)

	if !fallbackCalled {
		t.Fatal("/apiary must go to the frontend fallback")
	}
}
