package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(t *testing.T) (*Store, *http.ServeMux) {
	t.Helper()
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.Init(); err != nil {
		store.Close()
		t.Fatalf("init store: %v", err)
	}
	mux := http.NewServeMux()
	NewAPI(store).RegisterRoutes(mux)
	return store, mux
}

func doJSON(t *testing.T, mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func TestCreateEmployee_ok(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodPost, "/employees", map[string]string{"name": "Alice"})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	var got Employee
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.ID == 0 || got.Name != "Alice" {
		t.Fatalf("unexpected body: %+v", got)
	}
}

func TestUpdateEmployee_ok(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	created, err := store.CreateEmployee("Bob")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	resp := doJSON(t, mux, http.MethodPut, "/employees/1", map[string]string{"name": "Robert"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var got Employee
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.ID != created.ID || got.Name != "Robert" {
		t.Fatalf("unexpected body: %+v", got)
	}
}

func TestCreateEmployee_422(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodPost, "/employees", map[string]string{"name": "  "})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.Code)
	}
}

func TestUpdateEmployee_404(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodPut, "/employees/999", map[string]string{"name": "X"})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

func TestListEmployees_empty(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	rr := doJSON(t, mux, http.MethodGet, "/employees", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got []Employee
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %v", got)
	}
}

func TestListEmployees_withData(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	if _, err := store.CreateEmployee("A"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := store.CreateEmployee("B"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rr := doJSON(t, mux, http.MethodGet, "/employees", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got []Employee
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(got) != 2 || got[0].Name != "A" || got[1].Name != "B" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestUpdateEmployee_invalidID_422(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	rr := doJSON(t, mux, http.MethodPut, "/employees/abc", map[string]string{"name": "X"})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestEmployees_MethodNotAllowed(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	rr := doJSON(t, mux, http.MethodDelete, "/employees", nil)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestUpdateEmployee_422(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	if _, err := store.CreateEmployee("A"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	rr := doJSON(t, mux, http.MethodPut, "/employees/1", map[string]string{"name": "  "})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestCreateEmployee_invalidJSON_422(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	req := httptest.NewRequest(http.MethodPost, "/employees", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestUpdateEmployee_invalidJSON_422(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	if _, err := store.CreateEmployee("A"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/employees/1", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestListEmployees_500(t *testing.T) {
	store, mux := setupTestServer(t)
	// Simulate DB failure
	_ = store.Close()
	rr := doJSON(t, mux, http.MethodGet, "/employees", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestCreateEmployee_500(t *testing.T) {
	store, mux := setupTestServer(t)
	_ = store.Close()
	rr := doJSON(t, mux, http.MethodPost, "/employees", map[string]string{"name": "X"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestUpdateEmployee_500(t *testing.T) {
	store, mux := setupTestServer(t)
	// Close DB to force error path
	_ = store.Close()
	rr := doJSON(t, mux, http.MethodPut, "/employees/1", map[string]string{"name": "Y"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestCORS_OPTIONS_OK(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()
	handler := withCORS(mux)

	req := httptest.NewRequest(http.MethodOptions, "/employees", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatalf("missing CORS header")
	}
}

func TestDeleteEmployee_ok(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	emp, err := store.CreateEmployee("ToDelete")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	rr := doJSON(t, mux, http.MethodDelete, "/employees/1", nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body %s", rr.Code, rr.Body.String())
	}

	_, err = store.UpdateEmployee(emp.ID, "new")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestDeleteEmployee_notFound(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	rr := doJSON(t, mux, http.MethodDelete, "/employees/99", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteEmployee_invalidID(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	rr := doJSON(t, mux, http.MethodDelete, "/employees/abc", nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

type reviewList struct {
	Items      []PerformanceReview       `json:"items"`
	Aggregates []ReviewEmployeeAggregate `json:"aggregates"`
}

func TestReviews_CreateListTransition(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	emp, err := store.CreateEmployee("Alice")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	createResp := doJSON(t, mux, http.MethodPost, "/reviews", map[string]any{
		"employeeId": emp.ID,
		"period":     "2024-Q4",
		"reviewer":   "Manager",
		"rating":     4,
		"strengths":  "Teamwork",
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}
	var created PerformanceReview
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("json: %v", err)
	}
	if created.State != ReviewStateDraft || created.EmployeeID != emp.ID {
		t.Fatalf("unexpected review %+v", created)
	}

	listResp := doJSON(t, mux, http.MethodGet, "/reviews?employeeId=1", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
	var list reviewList
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(list.Items) != 1 || len(list.Aggregates) != 1 {
		t.Fatalf("expected 1 item and aggregate, got %+v", list)
	}

	updateResp := doJSON(t, mux, http.MethodPut, "/reviews/1", map[string]any{
		"rating": 5,
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 update, got %d", updateResp.Code)
	}
	var updated PerformanceReview
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("json: %v", err)
	}
	if updated.Rating != 5 {
		t.Fatalf("expected rating 5, got %d", updated.Rating)
	}

	transitionResp := doJSON(t, mux, http.MethodPut, "/reviews/1/status", map[string]string{"state": ReviewStateSubmitted})
	if transitionResp.Code != http.StatusOK {
		t.Fatalf("expected 200 transition, got %d", transitionResp.Code)
	}
	var transitioned PerformanceReview
	if err := json.Unmarshal(transitionResp.Body.Bytes(), &transitioned); err != nil {
		t.Fatalf("json: %v", err)
	}
	if transitioned.State != ReviewStateSubmitted {
		t.Fatalf("unexpected state %+v", transitioned)
	}

	transitionResp = doJSON(t, mux, http.MethodPut, "/reviews/1/status", map[string]string{"state": ReviewStateApproved})
	if transitionResp.Code != http.StatusOK {
		t.Fatalf("expected 200 transition approved, got %d", transitionResp.Code)
	}

	invalidTransition := doJSON(t, mux, http.MethodPut, "/reviews/1/status", map[string]string{"state": ReviewStateSubmitted})
	if invalidTransition.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid transition, got %d", invalidTransition.Code)
	}
}

func TestReviews_InvalidPayload(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodPost, "/reviews", map[string]any{
		"employeeId": 0,
		"period":     "",
		"reviewer":   "",
		"rating":     10,
	})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.Code)
	}

	resp = doJSON(t, mux, http.MethodPut, "/reviews/abc", map[string]any{"rating": 3})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid id, got %d", resp.Code)
	}
}

func TestReviews_UpdateInvalidRating(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	emp, err := store.CreateEmployee("Alice")
	if err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	if _, err := store.CreatePerformanceReview(PerformanceReviewInput{
		EmployeeID: emp.ID,
		Period:     "2024-Q4",
		Reviewer:   "Boss",
		Rating:     4,
	}); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	resp := doJSON(t, mux, http.MethodPut, "/reviews/1", map[string]any{"rating": 6})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid rating, got %d", resp.Code)
	}
}

func TestReviews_UpdateNotFound(t *testing.T) {
	_, mux := setupTestServer(t)

	resp := doJSON(t, mux, http.MethodPut, "/reviews/999", map[string]any{"rating": 4})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body %s", resp.Code, resp.Body.String())
	}
}

func TestReviews_UpdateInternalError(t *testing.T) {
	store, mux := setupTestServer(t)
	_ = store.Close()

	resp := doJSON(t, mux, http.MethodPut, "/reviews/1", map[string]any{"rating": 4})
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestReviews_ListInvalidEmployeeID(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodGet, "/reviews?employeeId=abc", nil)
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid employeeId, got %d", resp.Code)
	}
}

func TestReviews_ListInternalError(t *testing.T) {
	store, mux := setupTestServer(t)
	_ = store.Close()

	resp := doJSON(t, mux, http.MethodGet, "/reviews", nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestReviews_TransitionInvalidJSON(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	emp, err := store.CreateEmployee("Bob")
	if err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	if _, err := store.CreatePerformanceReview(PerformanceReviewInput{
		EmployeeID: emp.ID,
		Period:     "2024-Q4",
		Reviewer:   "Boss",
		Rating:     4,
	}); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/reviews/1/status", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid json, got %d", rr.Code)
	}
}

func TestReviews_TransitionBlankState(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	emp, err := store.CreateEmployee("Carol")
	if err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	if _, err := store.CreatePerformanceReview(PerformanceReviewInput{
		EmployeeID: emp.ID,
		Period:     "2024-Q4",
		Reviewer:   "Boss",
		Rating:     4,
	}); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	resp := doJSON(t, mux, http.MethodPut, "/reviews/1/status", map[string]string{"state": "  "})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 blank state, got %d", resp.Code)
	}
}

func TestReviews_TransitionNotFound(t *testing.T) {
	_, mux := setupTestServer(t)

	resp := doJSON(t, mux, http.MethodPut, "/reviews/999/status", map[string]string{"state": ReviewStateSubmitted})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body %s", resp.Code, resp.Body.String())
	}
}

type payrollList struct {
	Items      []PayrollRecord           `json:"items"`
	Aggregates payrollAggregatesResponse `json:"aggregates"`
}

func TestPayroll_CreateAndList(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	emp, err := store.CreateEmployee("Bob")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	resp := doJSON(t, mux, http.MethodPost, "/payroll", map[string]any{
		"employeeId":    emp.ID,
		"period":        "2024-11",
		"baseSalary":    1000.0,
		"overtimeHours": 10.0,
		"overtimeRate":  50.0,
		"bonuses":       200.0,
		"deductions":    100.0,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body %s", resp.Code, resp.Body.String())
	}
	var created PayrollRecord
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("json: %v", err)
	}
	expectedNet := 1000.0 + (10.0 * 50.0) + 200.0 - 100.0
	if created.NetPay != expectedNet {
		t.Fatalf("expected net %.2f, got %.2f", expectedNet, created.NetPay)
	}

	listResp := doJSON(t, mux, http.MethodGet, "/payroll?employeeId=1", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
	var list payrollList
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected payroll list 1, got %+v", list)
	}
	if list.Aggregates.GrandTotalNet != expectedNet {
		t.Fatalf("expected grand total %.2f, got %.2f", expectedNet, list.Aggregates.GrandTotalNet)
	}
}

func TestPayroll_InvalidPayload(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodPost, "/payroll", map[string]any{
		"employeeId": 0,
		"period":     "",
		"baseSalary": -1,
	})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.Code)
	}
}

func TestPayroll_CreateInvalidJSON(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	req := httptest.NewRequest(http.MethodPost, "/payroll", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestPayroll_ListInvalidEmployeeID(t *testing.T) {
	store, mux := setupTestServer(t)
	defer store.Close()

	resp := doJSON(t, mux, http.MethodGet, "/payroll?employeeId=abc", nil)
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid employeeId, got %d", resp.Code)
	}
}

func TestPayroll_ListInternalError(t *testing.T) {
	store, mux := setupTestServer(t)
	_ = store.Close()

	resp := doJSON(t, mux, http.MethodGet, "/payroll", nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestPayroll_CreateInternalError(t *testing.T) {
	store, mux := setupTestServer(t)
	_ = store.Close()

	resp := doJSON(t, mux, http.MethodPost, "/payroll", map[string]any{
		"employeeId": 1,
		"period":     "2024-01",
		"baseSalary": 1,
	})
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}
