package main

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type Employee struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PerformanceReview struct {
	ID            int64  `json:"id"`
	EmployeeID    int64  `json:"employeeId"`
	EmployeeName  string `json:"employeeName"`
	Period        string `json:"period"`
	Reviewer      string `json:"reviewer"`
	Rating        int    `json:"rating"`
	Strengths     string `json:"strengths"`
	Opportunities string `json:"opportunities"`
	State         string `json:"state"`
}

type ReviewEmployeeAggregate struct {
	EmployeeID   int64   `json:"employeeId"`
	EmployeeName string  `json:"employeeName"`
	Average      float64 `json:"averageRating"`
	LatestState  string  `json:"latestState"`
	Count        int64   `json:"count"`
}

type PerformanceReviewFilter struct {
	EmployeeID int64
	Period     string
	State      string
}

type PerformanceReviewInput struct {
	EmployeeID    int64
	Period        string
	Reviewer      string
	Rating        int
	Strengths     string
	Opportunities string
}

type PerformanceReviewUpdate struct {
	Reviewer      *string
	Rating        *int
	Strengths     *string
	Opportunities *string
}

type PayrollRecord struct {
	ID            int64   `json:"id"`
	EmployeeID    int64   `json:"employeeId"`
	EmployeeName  string  `json:"employeeName"`
	Period        string  `json:"period"`
	BaseSalary    float64 `json:"baseSalary"`
	OvertimeHours float64 `json:"overtimeHours"`
	OvertimeRate  float64 `json:"overtimeRate"`
	Bonuses       float64 `json:"bonuses"`
	Deductions    float64 `json:"deductions"`
	NetPay        float64 `json:"netPay"`
}

type PayrollFilter struct {
	EmployeeID int64
	Period     string
}

type PayrollPeriodTotal struct {
	Period string  `json:"period"`
	Total  float64 `json:"totalNet"`
}

var ErrNotFound = errors.New("not found")
var ErrInvalidTransition = errors.New("invalid transition")

type Store struct {
	db       *sql.DB
	postgres bool
}

const (
	ReviewStateDraft     = "draft"
	ReviewStateSubmitted = "submitted"
	ReviewStateApproved  = "approved"
)

func NewStore(dsn string) (*Store, error) {
	driver := "sqlite"
	postgres := false
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		driver = "postgres"
		postgres = true
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, postgres: postgres}, nil
}

// pq replaces ? placeholders with $1, $2, ... for PostgreSQL.
var rePlaceholder = regexp.MustCompile(`\?`)

func (s *Store) pq(query string) string {
	if !s.postgres {
		return query
	}
	n := 0
	return rePlaceholder.ReplaceAllStringFunc(query, func(_ string) string {
		n++
		return fmt.Sprintf("$%d", n)
	})
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Init() error {
	var schema string
	if s.postgres {
		schema = `
			CREATE TABLE IF NOT EXISTS employees (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS performance_reviews (
				id SERIAL PRIMARY KEY,
				employee_id INTEGER NOT NULL,
				period TEXT NOT NULL,
				reviewer TEXT NOT NULL,
				rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
				strengths TEXT,
				opportunities TEXT,
				state TEXT NOT NULL,
				FOREIGN KEY(employee_id) REFERENCES employees(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_reviews_employee_period ON performance_reviews(employee_id, period);

			CREATE TABLE IF NOT EXISTS payroll_records (
				id SERIAL PRIMARY KEY,
				employee_id INTEGER NOT NULL,
				period TEXT NOT NULL,
				base_salary DOUBLE PRECISION NOT NULL,
				overtime_hours DOUBLE PRECISION NOT NULL DEFAULT 0,
				overtime_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
				bonuses DOUBLE PRECISION NOT NULL DEFAULT 0,
				deductions DOUBLE PRECISION NOT NULL DEFAULT 0,
				net_pay DOUBLE PRECISION NOT NULL,
				FOREIGN KEY(employee_id) REFERENCES employees(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_payroll_employee_period ON payroll_records(employee_id, period);
		`
	} else {
		schema = `
			CREATE TABLE IF NOT EXISTS employees (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS performance_reviews (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				employee_id INTEGER NOT NULL,
				period TEXT NOT NULL,
				reviewer TEXT NOT NULL,
				rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
				strengths TEXT,
				opportunities TEXT,
				state TEXT NOT NULL,
				FOREIGN KEY(employee_id) REFERENCES employees(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_reviews_employee_period ON performance_reviews(employee_id, period);

			CREATE TABLE IF NOT EXISTS payroll_records (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				employee_id INTEGER NOT NULL,
				period TEXT NOT NULL,
				base_salary REAL NOT NULL,
				overtime_hours REAL NOT NULL DEFAULT 0,
				overtime_rate REAL NOT NULL DEFAULT 0,
				bonuses REAL NOT NULL DEFAULT 0,
				deductions REAL NOT NULL DEFAULT 0,
				net_pay REAL NOT NULL,
				FOREIGN KEY(employee_id) REFERENCES employees(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_payroll_employee_period ON payroll_records(employee_id, period);
		`
	}
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) ListEmployees() ([]Employee, error) {
	rows, err := s.db.Query("SELECT id, name FROM employees ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Employee, 0)
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.Name); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Store) CreateEmployee(name string) (Employee, error) {
	var id int64
	err := s.db.QueryRow(s.pq("INSERT INTO employees(name) VALUES(?) RETURNING id"), name).Scan(&id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return Employee{}, err
	}
	return Employee{ID: id, Name: name}, nil
}

func (s *Store) UpdateEmployee(id int64, name string) (Employee, error) {
	res, err := s.db.Exec(s.pq("UPDATE employees SET name=? WHERE id=?"), name, id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return Employee{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Employee{}, err
	}
	if affected == 0 {
		return Employee{}, ErrNotFound
	}
	return Employee{ID: id, Name: name}, nil
}

func (s *Store) DeleteEmployee(id int64) error {
	res, err := s.db.Exec(s.pq("DELETE FROM employees WHERE id=?"), id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// Performance Reviews

func (s *Store) ListPerformanceReviews(filter PerformanceReviewFilter) ([]PerformanceReview, error) {
	builder := strings.Builder{}
	builder.WriteString(`SELECT r.id, r.employee_id, e.name, r.period, r.reviewer, r.rating, r.strengths, r.opportunities, r.state
		FROM performance_reviews r
		JOIN employees e ON e.id = r.employee_id`)
	args := make([]any, 0)
	whereClauses, wArgs := buildReviewFilter(filter)
	if len(whereClauses) > 0 {
		where, err := joinAllowedClauses(whereClauses, allowedReviewFilterClauses, " AND ")
		if err != nil {
			return nil, err
		}
		builder.WriteString(" WHERE ")
		builder.WriteString(where)
		args = append(args, wArgs...)
	}
	builder.WriteString(" ORDER BY r.id DESC")

	rows, err := s.db.Query(s.pq(builder.String()), args...) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]PerformanceReview, 0)
	for rows.Next() {
		var pr PerformanceReview
		if err := rows.Scan(&pr.ID, &pr.EmployeeID, &pr.EmployeeName, &pr.Period, &pr.Reviewer, &pr.Rating, &pr.Strengths, &pr.Opportunities, &pr.State); err != nil {
			return nil, err
		}
		result = append(result, pr)
	}
	return result, rows.Err()
}

var (
	allowedReviewFilterClauses = map[string]struct{}{
		"r.employee_id = ?": {},
		"r.period = ?":      {},
		"r.state = ?":       {},
	}
	allowedReviewUpdateClauses = map[string]struct{}{
		"reviewer = ?":      {},
		"rating = ?":        {},
		"strengths = ?":     {},
		"opportunities = ?": {},
	}
	allowedPayrollFilterClauses = map[string]struct{}{
		"p.employee_id = ?": {},
		"p.period = ?":      {},
	}
)

func buildReviewFilter(filter PerformanceReviewFilter) ([]string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if filter.EmployeeID > 0 {
		clauses = append(clauses, "r.employee_id = ?")
		args = append(args, filter.EmployeeID)
	}
	if filter.Period != "" {
		clauses = append(clauses, "r.period = ?")
		args = append(args, filter.Period)
	}
	if filter.State != "" {
		clauses = append(clauses, "r.state = ?")
		args = append(args, filter.State)
	}
	return clauses, args
}

func (s *Store) CreatePerformanceReview(input PerformanceReviewInput) (PerformanceReview, error) {
	if err := validateReviewInput(input); err != nil {
		return PerformanceReview{}, err
	}
	var id int64
	insertReviewSQL := `INSERT INTO performance_reviews
		(employee_id, period, reviewer, rating, strengths, opportunities, state)
		VALUES(?, ?, ?, ?, ?, ?, ?) RETURNING id`
	err := s.db.QueryRow(s.pq(insertReviewSQL), input.EmployeeID, input.Period, input.Reviewer, input.Rating, input.Strengths, input.Opportunities, ReviewStateDraft).Scan(&id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return PerformanceReview{}, err
	}
	return s.getPerformanceReviewByID(id)
}

func (s *Store) UpdatePerformanceReview(id int64, update PerformanceReviewUpdate) (PerformanceReview, error) {
	setClauses := make([]string, 0)
	args := make([]any, 0)
	if update.Reviewer != nil {
		setClauses = append(setClauses, "reviewer = ?")
		args = append(args, strings.TrimSpace(*update.Reviewer))
	}
	if update.Rating != nil {
		if err := validateRating(*update.Rating); err != nil {
			return PerformanceReview{}, err
		}
		setClauses = append(setClauses, "rating = ?")
		args = append(args, *update.Rating)
	}
	if update.Strengths != nil {
		setClauses = append(setClauses, "strengths = ?")
		args = append(args, strings.TrimSpace(*update.Strengths))
	}
	if update.Opportunities != nil {
		setClauses = append(setClauses, "opportunities = ?")
		args = append(args, strings.TrimSpace(*update.Opportunities))
	}
	if len(setClauses) == 0 {
		return s.getPerformanceReviewByID(id)
	}
	setClauseString, err := joinAllowedClauses(setClauses, allowedReviewUpdateClauses, ", ")
	if err != nil {
		return PerformanceReview{}, err
	}
	args = append(args, id)
	res, err := s.db.Exec(s.pq(`UPDATE performance_reviews SET `+setClauseString+` WHERE id = ?`), args...) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return PerformanceReview{}, err
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		return PerformanceReview{}, err
	}
	if cnt == 0 {
		return PerformanceReview{}, ErrNotFound
	}
	return s.getPerformanceReviewByID(id)
}

func (s *Store) TransitionPerformanceReview(id int64, nextState string) (PerformanceReview, error) {
	review, err := s.getPerformanceReviewByID(id)
	if err != nil {
		return PerformanceReview{}, err
	}
	if !isValidTransition(review.State, nextState) {
		return PerformanceReview{}, ErrInvalidTransition
	}
	_, err = s.db.Exec(s.pq(`UPDATE performance_reviews SET state = ? WHERE id = ?`), nextState, id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return PerformanceReview{}, err
	}
	return s.getPerformanceReviewByID(id)
}

func (s *Store) getPerformanceReviewByID(id int64) (PerformanceReview, error) {
	getReviewSQL := `SELECT r.id, r.employee_id, e.name, r.period, r.reviewer, r.rating, r.strengths, r.opportunities, r.state
		FROM performance_reviews r
		JOIN employees e ON e.id = r.employee_id
		WHERE r.id = ?`
	row := s.db.QueryRow(s.pq(getReviewSQL), id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	var pr PerformanceReview
	if err := row.Scan(&pr.ID, &pr.EmployeeID, &pr.EmployeeName, &pr.Period, &pr.Reviewer, &pr.Rating, &pr.Strengths, &pr.Opportunities, &pr.State); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PerformanceReview{}, ErrNotFound
		}
		return PerformanceReview{}, err
	}
	return pr, nil
}

func validateReviewInput(input PerformanceReviewInput) error {
	if input.EmployeeID == 0 {
		return fmt.Errorf("employee is required")
	}
	if strings.TrimSpace(input.Period) == "" {
		return fmt.Errorf("period is required")
	}
	if strings.TrimSpace(input.Reviewer) == "" {
		return fmt.Errorf("reviewer is required")
	}
	return validateRating(input.Rating)
}

func validateRating(rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	return nil
}

func isValidTransition(current, next string) bool {
	switch current {
	case ReviewStateDraft:
		return next == ReviewStateSubmitted
	case ReviewStateSubmitted:
		return next == ReviewStateApproved
	default:
		return false
	}
}

func (s *Store) ListReviewAggregates(filter PerformanceReviewFilter) ([]ReviewEmployeeAggregate, error) {
	builder := strings.Builder{}
	builder.WriteString(`
		SELECT e.id, e.name, AVG(r.rating) as avg_rating, COUNT(*) as total_reviews,
			(SELECT state FROM performance_reviews r2 WHERE r2.employee_id = e.id ORDER BY r2.id DESC LIMIT 1) as latest_state
		FROM performance_reviews r
		JOIN employees e ON e.id = r.employee_id`)
	args := make([]any, 0)
	whereClauses, wArgs := buildReviewFilter(filter)
	if len(whereClauses) > 0 {
		where, err := joinAllowedClauses(whereClauses, allowedReviewFilterClauses, " AND ")
		if err != nil {
			return nil, err
		}
		builder.WriteString(" WHERE ")
		builder.WriteString(where)
		args = append(args, wArgs...)
	}
	builder.WriteString(" GROUP BY e.id, e.name")

	rows, err := s.db.Query(s.pq(builder.String()), args...) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ReviewEmployeeAggregate, 0)
	for rows.Next() {
		var agg ReviewEmployeeAggregate
		if err := rows.Scan(&agg.EmployeeID, &agg.EmployeeName, &agg.Average, &agg.Count, &agg.LatestState); err != nil {
			return nil, err
		}
		result = append(result, agg)
	}
	return result, rows.Err()
}

// Payroll

func (s *Store) ListPayrollRecords(filter PayrollFilter) ([]PayrollRecord, error) {
	builder := strings.Builder{}
	builder.WriteString(`SELECT p.id, p.employee_id, e.name, p.period, p.base_salary, p.overtime_hours, p.overtime_rate, p.bonuses, p.deductions, p.net_pay
		FROM payroll_records p
		JOIN employees e ON e.id = p.employee_id`)
	args := make([]any, 0)
	whereClauses, wArgs := buildPayrollFilter(filter)
	if len(whereClauses) > 0 {
		where, err := joinAllowedClauses(whereClauses, allowedPayrollFilterClauses, " AND ")
		if err != nil {
			return nil, err
		}
		builder.WriteString(" WHERE ")
		builder.WriteString(where)
		args = append(args, wArgs...)
	}
	builder.WriteString(" ORDER BY p.period DESC, p.id DESC")

	rows, err := s.db.Query(s.pq(builder.String()), args...) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]PayrollRecord, 0)
	for rows.Next() {
		var pr PayrollRecord
		if err := rows.Scan(&pr.ID, &pr.EmployeeID, &pr.EmployeeName, &pr.Period, &pr.BaseSalary, &pr.OvertimeHours, &pr.OvertimeRate, &pr.Bonuses, &pr.Deductions, &pr.NetPay); err != nil {
			return nil, err
		}
		result = append(result, pr)
	}
	return result, rows.Err()
}

func buildPayrollFilter(filter PayrollFilter) ([]string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if filter.EmployeeID > 0 {
		clauses = append(clauses, "p.employee_id = ?")
		args = append(args, filter.EmployeeID)
	}
	if filter.Period != "" {
		clauses = append(clauses, "p.period = ?")
		args = append(args, filter.Period)
	}
	return clauses, args
}

type PayrollRecordInput struct {
	EmployeeID    int64
	Period        string
	BaseSalary    float64
	OvertimeHours float64
	OvertimeRate  float64
	Bonuses       float64
	Deductions    float64
}

func (s *Store) CreatePayrollRecord(input PayrollRecordInput) (PayrollRecord, error) {
	if err := validatePayrollInput(input); err != nil {
		return PayrollRecord{}, err
	}
	net := calculateNetPay(input.BaseSalary, input.OvertimeHours, input.OvertimeRate, input.Bonuses, input.Deductions)
	var id int64
	insertPayrollSQL := `INSERT INTO payroll_records
		(employee_id, period, base_salary, overtime_hours, overtime_rate, bonuses, deductions, net_pay)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	err := s.db.QueryRow(s.pq(insertPayrollSQL), input.EmployeeID, input.Period, input.BaseSalary, input.OvertimeHours, input.OvertimeRate, input.Bonuses, input.Deductions, net).Scan(&id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return PayrollRecord{}, err
	}
	return s.getPayrollByID(id)
}

func calculateNetPay(base, overtimeHours, overtimeRate, bonuses, deductions float64) float64 {
	return base + (overtimeHours * overtimeRate) + bonuses - deductions
}

func validatePayrollInput(input PayrollRecordInput) error {
	if input.EmployeeID == 0 {
		return fmt.Errorf("employee is required")
	}
	if strings.TrimSpace(input.Period) == "" {
		return fmt.Errorf("period is required")
	}
	if input.BaseSalary < 0 {
		return fmt.Errorf("base salary must be >= 0")
	}
	if input.OvertimeHours < 0 || input.OvertimeRate < 0 {
		return fmt.Errorf("overtime values must be >= 0")
	}
	return nil
}

func (s *Store) getPayrollByID(id int64) (PayrollRecord, error) {
	getPayrollSQL := `SELECT p.id, p.employee_id, e.name, p.period, p.base_salary, p.overtime_hours, p.overtime_rate, p.bonuses, p.deductions, p.net_pay
		FROM payroll_records p
		JOIN employees e ON e.id = p.employee_id
		WHERE p.id = ?`
	row := s.db.QueryRow(s.pq(getPayrollSQL), id) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	var pr PayrollRecord
	if err := row.Scan(&pr.ID, &pr.EmployeeID, &pr.EmployeeName, &pr.Period, &pr.BaseSalary, &pr.OvertimeHours, &pr.OvertimeRate, &pr.Bonuses, &pr.Deductions, &pr.NetPay); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PayrollRecord{}, ErrNotFound
		}
		return PayrollRecord{}, err
	}
	return pr, nil
}

func (s *Store) PayrollTotals(filter PayrollFilter) ([]PayrollPeriodTotal, float64, error) {
	builder := strings.Builder{}
	builder.WriteString(`SELECT p.period, SUM(p.net_pay) as total
		FROM payroll_records p`)
	args := make([]any, 0)
	whereClauses, wArgs := buildPayrollFilter(filter)
	if len(whereClauses) > 0 {
		where, err := joinAllowedClauses(whereClauses, allowedPayrollFilterClauses, " AND ")
		if err != nil {
			return nil, 0, err
		}
		builder.WriteString(" WHERE ")
		builder.WriteString(where)
		args = append(args, wArgs...)
	}
	builder.WriteString(" GROUP BY p.period ORDER BY p.period DESC")

	rows, err := s.db.Query(s.pq(builder.String()), args...) // NOSONAR: consulta parametrizada; pq() solo convierte placeholders ? a $N
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	totalList := make([]PayrollPeriodTotal, 0)
	var grandTotal float64
	for rows.Next() {
		var rec PayrollPeriodTotal
		if err := rows.Scan(&rec.Period, &rec.Total); err != nil {
			return nil, 0, err
		}
		totalList = append(totalList, rec)
		grandTotal += rec.Total
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return totalList, grandTotal, nil
}

func joinAllowedClauses(clauses []string, allowed map[string]struct{}, sep string) (string, error) {
	if len(clauses) == 0 {
		return "", nil
	}
	for _, clause := range clauses {
		if _, ok := allowed[clause]; !ok {
			return "", fmt.Errorf("unsupported clause %q", clause)
		}
	}
	return strings.Join(clauses, sep), nil
}
