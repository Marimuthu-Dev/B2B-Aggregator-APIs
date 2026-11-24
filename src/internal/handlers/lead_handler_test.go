package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"

	"github.com/gin-gonic/gin"
)

type stubLeadListService struct {
	service.LeadService
	capturedFilter repository.LeadListFilter
	called         bool
}

func (s *stubLeadListService) ListLeads(filter repository.LeadListFilter) ([]domain.Lead, int64, error) {
	s.called = true
	s.capturedFilter = filter
	return []domain.Lead{}, 0, nil
}

func TestLeadHandlerGetAllCandidateStateCityFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("binds camelCase stateId and cityId independently from storeStateId and storeCityId", func(t *testing.T) {
		stub := &stubLeadListService{}
		h := NewLeadHandler(stub, nil)
		r := gin.New()
		r.Use(middleware.ValidationErrorMiddleware())
		r.GET("/leads", h.GetAll)

		req := httptest.NewRequest(http.MethodGet, "/leads?stateId=12&cityId=345&storeStateId=2&storeCityId=44", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
		}
		if !stub.called {
			t.Fatal("expected ListLeads to be called")
		}
		if stub.capturedFilter.StateID == nil || *stub.capturedFilter.StateID != 12 {
			t.Fatalf("StateID = %v, want 12", stub.capturedFilter.StateID)
		}
		if stub.capturedFilter.CityID == nil || *stub.capturedFilter.CityID != 345 {
			t.Fatalf("CityID = %v, want 345", stub.capturedFilter.CityID)
		}
		if stub.capturedFilter.StoreStateID == nil || *stub.capturedFilter.StoreStateID != 2 {
			t.Fatalf("StoreStateID = %v, want 2", stub.capturedFilter.StoreStateID)
		}
		if stub.capturedFilter.StoreCityID == nil || *stub.capturedFilter.StoreCityID != 44 {
			t.Fatalf("StoreCityID = %v, want 44", stub.capturedFilter.StoreCityID)
		}
	})

	t.Run("binds PascalCase and candidate-prefixed query keys", func(t *testing.T) {
		cases := []struct {
			name      string
			query     string
			wantState int32
			wantCity  int32
		}{
			{name: "PascalCase", query: "StateID=7&CityID=101", wantState: 7, wantCity: 101},
			{name: "candidate camelCase", query: "candidateStateId=9&candidateCityId=202", wantState: 9, wantCity: 202},
			{name: "candidate PascalCase", query: "CandidateStateID=15&CandidateCityID=303", wantState: 15, wantCity: 303},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				stub := &stubLeadListService{}
				h := NewLeadHandler(stub, nil)
				r := gin.New()
				r.Use(middleware.ValidationErrorMiddleware())
				r.GET("/leads", h.GetAll)

				req := httptest.NewRequest(http.MethodGet, "/leads?"+tc.query, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
				}
				if stub.capturedFilter.StateID == nil || *stub.capturedFilter.StateID != tc.wantState {
					t.Fatalf("StateID = %v, want %d", stub.capturedFilter.StateID, tc.wantState)
				}
				if stub.capturedFilter.CityID == nil || *stub.capturedFilter.CityID != tc.wantCity {
					t.Fatalf("CityID = %v, want %d", stub.capturedFilter.CityID, tc.wantCity)
				}
			})
		}
	})

	t.Run("rejects invalid non-positive or non-numeric stateId and cityId", func(t *testing.T) {
		invalidQueries := []string{
			"stateId=0",
			"stateId=-1",
			"stateId=abc",
			"cityId=0",
			"cityId=-3",
			"cityId=xyz",
			"StateID=0",
			"CityID=invalid",
			"candidateStateId=0",
			"candidateCityId=-5",
		}

		for _, qs := range invalidQueries {
			t.Run(qs, func(t *testing.T) {
				stub := &stubLeadListService{}
				h := NewLeadHandler(stub, nil)
				r := gin.New()
				r.Use(middleware.ValidationErrorMiddleware())
				r.GET("/leads", h.GetAll)

				req := httptest.NewRequest(http.MethodGet, "/leads?"+qs, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Fatalf("query %q: status = %d, want 400; body = %s", qs, w.Code, w.Body.String())
				}
				if stub.called {
					t.Fatalf("query %q: ListLeads should not be called on invalid filter", qs)
				}
			})
		}
	})
}
