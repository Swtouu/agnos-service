package mockhis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/patient/search/:id", Search)
	return r
}

func TestSearch_ByNationalID_Found(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/patient/search/1100000000001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestSearch_ByPassportID_Found(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/patient/search/P1234567", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestSearch_UnknownID_NotFound(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/patient/search/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusNotFound)
	}
}
