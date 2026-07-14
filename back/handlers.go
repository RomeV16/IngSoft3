// handlers.go — CAPA HTTP (controladores de la API REST).
//
// Responsabilidad: recibir requests HTTP, validar el payload, delegar la
// lógica de datos al Store (store.go) y devolver JSON con el código de
// estado correcto. Acá NO hay SQL: la separación en capas es
//   handlers.go (HTTP) → store.go (acceso a datos) → base de datos
//
// Convención de códigos de estado:
//   200 OK               → lectura/actualización exitosa
//   201 Created          → recurso creado
//   204 No Content       → borrado exitoso (sin body)
//   404 Not Found        → el id no existe
//   422 Unprocessable    → payload inválido (validación de negocio)
//   405 Method Not Allowed → verbo HTTP no soportado en esa ruta
//   500 Internal Error   → falla de la base de datos u otra falla interna
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// API agrupa los handlers y guarda la referencia al Store (inyección de
// dependencia: en producción recibe la BD real, en los tests unitarios
// recibe un Store con SQLite en memoria — por eso los tests no necesitan
// levantar PostgreSQL).
type API struct {
	store *Store
}

func NewAPI(store *Store) *API {
	return &API{store: store}
}

// Mensaje genérico para errores 500: nunca exponemos el error interno real
// al cliente (podría filtrar información de la BD).
const internalErrorMsg = "internal error"

// RegisterRoutes registra todas las rutas de la API en el mux estándar de Go.
// Rutas: /employees (GET lista, POST crea), /employees/{id} (PUT, DELETE),
// /reviews (GET, POST), /reviews/{id} (PUT), /reviews/{id}/status (PUT
// transición de estado), /payroll (GET, POST).
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/employees", func(w http.ResponseWriter, r *http.Request) {
		setJSON(w)
		switch r.Method {
		case http.MethodGet:
			a.handleListEmployees(w, r)
		case http.MethodPost:
			a.handleCreateEmployee(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/employees/", func(w http.ResponseWriter, r *http.Request) {
		setJSON(w)
		idStr := strings.TrimPrefix(r.URL.Path, "/employees/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid id")
			return
		}
		switch r.Method {
		case http.MethodPut:
			a.handleUpdateEmployee(w, r, id)
		case http.MethodDelete:
			a.handleDeleteEmployee(w, id)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/reviews", func(w http.ResponseWriter, r *http.Request) {
		setJSON(w)
		switch r.Method {
		case http.MethodGet:
			a.handleListReviews(w, r)
		case http.MethodPost:
			a.handleCreateReview(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/reviews/", func(w http.ResponseWriter, r *http.Request) {
		setJSON(w)
		path := strings.TrimPrefix(r.URL.Path, "/reviews/")
		if strings.HasSuffix(path, "/status") {
			if r.Method != http.MethodPut {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			idStr := strings.TrimSuffix(path, "/status")
			idStr = strings.TrimSuffix(idStr, "/")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				writeError(w, http.StatusUnprocessableEntity, "invalid id")
				return
			}
			a.handleTransitionReview(w, r, id)
			return
		}
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id, err := strconv.ParseInt(path, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid id")
			return
		}
		a.handleUpdateReview(w, r, id)
	})

	mux.HandleFunc("/payroll", func(w http.ResponseWriter, r *http.Request) {
		setJSON(w)
		switch r.Method {
		case http.MethodGet:
			a.handleListPayroll(w, r)
		case http.MethodPost:
			a.handleCreatePayroll(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// setJSON marca la respuesta como JSON (todas las respuestas de la API lo son).
func setJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

// writeError devuelve un error en formato uniforme: {"error": "mensaje"}.
// El frontend lee este campo para mostrar el mensaje al usuario.
func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// employeePayload es el body esperado en POST/PUT de empleados: {"name": "..."}
type employeePayload struct {
	Name string `json:"name"`
}

func (a *API) handleListEmployees(w http.ResponseWriter, _ *http.Request) {
	list, err := a.store.ListEmployees()
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	_ = json.NewEncoder(w).Encode(list)
}

// handleCreateEmployee: POST /employees
// Flujo típico de todos los handlers de creación:
//  1. decodificar el JSON del body   → si es inválido: 422
//  2. validar reglas de negocio      → si falla: 422
//  3. delegar al Store               → si la BD falla: 500
//  4. responder 201 con el recurso creado (incluye el id asignado)
func (a *API) handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	var p employeePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid payload")
		return
	}
	// TrimSpace: "  " (solo espacios) cuenta como nombre vacío
	name := strings.TrimSpace(p.Name)
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	created, err := a.store.CreateEmployee(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (a *API) handleUpdateEmployee(w http.ResponseWriter, r *http.Request, id int64) {
	var p employeePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid payload")
		return
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	updated, err := a.store.UpdateEmployee(id, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	_ = json.NewEncoder(w).Encode(updated)
}

func (a *API) handleDeleteEmployee(w http.ResponseWriter, id int64) {
	if err := a.store.DeleteEmployee(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Handlers de evaluaciones de desempeño (performance reviews) ───────────
// Las reviews tienen una máquina de estados: draft → submitted → approved.
// La transición se hace vía PUT /reviews/{id}/status y el Store valida que
// el cambio de estado sea legal (no se puede pasar de draft a approved
// directamente) — si no lo es devuelve ErrInvalidTransition → 422.

type reviewPayload struct {
	EmployeeID    int64  `json:"employeeId"`
	Period        string `json:"period"`
	Reviewer      string `json:"reviewer"`
	Rating        int    `json:"rating"`
	Strengths     string `json:"strengths"`
	Opportunities string `json:"opportunities"`
}

type reviewListResponse struct {
	Items      []PerformanceReview       `json:"items"`
	Aggregates []ReviewEmployeeAggregate `json:"aggregates"`
}

func (a *API) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	var payload reviewPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid payload")
		return
	}
	if err := validateReviewPayload(payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	created, err := a.store.CreatePerformanceReview(PerformanceReviewInput{
		EmployeeID:    payload.EmployeeID,
		Period:        strings.TrimSpace(payload.Period),
		Reviewer:      strings.TrimSpace(payload.Reviewer),
		Rating:        payload.Rating,
		Strengths:     strings.TrimSpace(payload.Strengths),
		Opportunities: strings.TrimSpace(payload.Opportunities),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func validateReviewPayload(p reviewPayload) error {
	if p.EmployeeID == 0 {
		return fmt.Errorf("employeeId is required")
	}
	if strings.TrimSpace(p.Period) == "" {
		return fmt.Errorf("period is required")
	}
	if strings.TrimSpace(p.Reviewer) == "" {
		return fmt.Errorf("reviewer is required")
	}
	if p.Rating < 1 || p.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	return nil
}

func (a *API) handleListReviews(w http.ResponseWriter, r *http.Request) {
	filter := PerformanceReviewFilter{}
	if v := r.URL.Query().Get("employeeId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid employeeId")
			return
		}
		filter.EmployeeID = id
	}
	if v := r.URL.Query().Get("period"); v != "" {
		filter.Period = v
	}
	if v := r.URL.Query().Get("state"); v != "" {
		filter.State = v
	}

	items, err := a.store.ListPerformanceReviews(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	aggregates, err := a.store.ListReviewAggregates(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	_ = json.NewEncoder(w).Encode(reviewListResponse{
		Items:      items,
		Aggregates: aggregates,
	})
}

// reviewUpdatePayload usa punteros para distinguir "campo no enviado" (nil,
// no se toca) de "campo enviado vacío" — permite actualizaciones parciales
// tipo PATCH: solo se modifican los campos presentes en el JSON.
type reviewUpdatePayload struct {
	Reviewer      *string `json:"reviewer"`
	Rating        *int    `json:"rating"`
	Strengths     *string `json:"strengths"`
	Opportunities *string `json:"opportunities"`
}

func (a *API) handleUpdateReview(w http.ResponseWriter, r *http.Request, id int64) {
	var payload reviewUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid payload")
		return
	}
	if payload.Rating != nil && (*payload.Rating < 1 || *payload.Rating > 5) {
		writeError(w, http.StatusUnprocessableEntity, "rating must be between 1 and 5")
		return
	}
	updated, err := a.store.UpdatePerformanceReview(id, PerformanceReviewUpdate{
		Reviewer:      payload.Reviewer,
		Rating:        payload.Rating,
		Strengths:     payload.Strengths,
		Opportunities: payload.Opportunities,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, ErrInvalidTransition) {
			writeError(w, http.StatusUnprocessableEntity, "invalid state transition")
			return
		}
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	_ = json.NewEncoder(w).Encode(updated)
}

type reviewTransitionPayload struct {
	State string `json:"state"`
}

func (a *API) handleTransitionReview(w http.ResponseWriter, r *http.Request, id int64) {
	var payload reviewTransitionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid payload")
		return
	}
	state := strings.TrimSpace(payload.State)
	if state == "" {
		writeError(w, http.StatusUnprocessableEntity, "state is required")
		return
	}
	updated, err := a.store.TransitionPerformanceReview(id, state)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, ErrInvalidTransition) {
			writeError(w, http.StatusUnprocessableEntity, "invalid transition")
			return
		}
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	_ = json.NewEncoder(w).Encode(updated)
}

// ─── Handlers de nómina (payroll) ───────────────────────────────────────────
// El neto NO lo manda el cliente: lo calcula el backend en store.go con
// calculateNetPay(base + horasExtra*tarifa + bonos - deducciones).
// GET /payroll admite filtros por employeeId y period via query string, y
// devuelve además agregados (totales por período y total general).

type payrollPayload struct {
	EmployeeID    int64   `json:"employeeId"`
	Period        string  `json:"period"`
	BaseSalary    float64 `json:"baseSalary"`
	OvertimeHours float64 `json:"overtimeHours"`
	OvertimeRate  float64 `json:"overtimeRate"`
	Bonuses       float64 `json:"bonuses"`
	Deductions    float64 `json:"deductions"`
}

type payrollListResponse struct {
	Items      []PayrollRecord           `json:"items"`
	Aggregates payrollAggregatesResponse `json:"aggregates"`
}

type payrollAggregatesResponse struct {
	TotalsByPeriod []PayrollPeriodTotal `json:"totalsByPeriod"`
	GrandTotalNet  float64              `json:"grandTotalNet"`
}

func (a *API) handleCreatePayroll(w http.ResponseWriter, r *http.Request) {
	var payload payrollPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid payload")
		return
	}
	if err := validatePayrollPayload(payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	created, err := a.store.CreatePayrollRecord(PayrollRecordInput{
		EmployeeID:    payload.EmployeeID,
		Period:        strings.TrimSpace(payload.Period),
		BaseSalary:    payload.BaseSalary,
		OvertimeHours: payload.OvertimeHours,
		OvertimeRate:  payload.OvertimeRate,
		Bonuses:       payload.Bonuses,
		Deductions:    payload.Deductions,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func validatePayrollPayload(p payrollPayload) error {
	if p.EmployeeID == 0 {
		return fmt.Errorf("employeeId is required")
	}
	if strings.TrimSpace(p.Period) == "" {
		return fmt.Errorf("period is required")
	}
	if p.BaseSalary < 0 {
		return fmt.Errorf("baseSalary must be >= 0")
	}
	if p.OvertimeHours < 0 || p.OvertimeRate < 0 {
		return fmt.Errorf("overtime values must be >= 0")
	}
	return nil
}

func (a *API) handleListPayroll(w http.ResponseWriter, r *http.Request) {
	filter := PayrollFilter{}
	if v := r.URL.Query().Get("employeeId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid employeeId")
			return
		}
		filter.EmployeeID = id
	}
	if v := r.URL.Query().Get("period"); v != "" {
		filter.Period = v
	}

	items, err := a.store.ListPayrollRecords(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	totals, grand, err := a.store.PayrollTotals(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, internalErrorMsg)
		return
	}
	_ = json.NewEncoder(w).Encode(payrollListResponse{
		Items: items,
		Aggregates: payrollAggregatesResponse{
			TotalsByPeriod: totals,
			GrandTotalNet:  grand,
		},
	})
}

