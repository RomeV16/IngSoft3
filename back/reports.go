package main

// Módulo de reportes de nómina — pendiente de tests (demo cobertura Sonar).

import (
	"fmt"
	"sort"
	"strings"
)

type PayrollReportRow struct {
	EmployeeName string
	Period       string
	NetPay       float64
	Category     string
}

func categorizeNetPay(net float64) string {
	switch {
	case net <= 0:
		return "sin-remuneracion"
	case net < 300000:
		return "banda-inicial"
	case net < 700000:
		return "banda-media"
	case net < 1200000:
		return "banda-alta"
	default:
		return "banda-ejecutiva"
	}
}

func buildPayrollReport(records []PayrollRecord) []PayrollReportRow {
	rows := make([]PayrollReportRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, PayrollReportRow{
			EmployeeName: r.EmployeeName,
			Period:       r.Period,
			NetPay:       r.NetPay,
			Category:     categorizeNetPay(r.NetPay),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Period != rows[j].Period {
			return rows[i].Period < rows[j].Period
		}
		return rows[i].NetPay > rows[j].NetPay
	})
	return rows
}

func summarizeByCategory(rows []PayrollReportRow) map[string]int {
	out := map[string]int{}
	for _, r := range rows {
		out[r.Category]++
	}
	return out
}

func formatReportCSV(rows []PayrollReportRow) string {
	var b strings.Builder
	b.WriteString("empleado,periodo,neto,categoria\n")
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("%s,%s,%.2f,%s\n", r.EmployeeName, r.Period, r.NetPay, r.Category))
	}
	return b.String()
}

func averageNetByPeriod(rows []PayrollReportRow) map[string]float64 {
	sums := map[string]float64{}
	counts := map[string]int{}
	for _, r := range rows {
		sums[r.Period] += r.NetPay
		counts[r.Period]++
	}
	out := map[string]float64{}
	for p, s := range sums {
		out[p] = s / float64(counts[p])
	}
	return out
}

func topEarners(rows []PayrollReportRow, n int) []PayrollReportRow {
	sorted := make([]PayrollReportRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].NetPay > sorted[j].NetPay })
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}
