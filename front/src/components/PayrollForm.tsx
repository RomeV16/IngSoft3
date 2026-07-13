"use client";

import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  AlertIcon,
  Button,
  FormControl,
  FormLabel,
  Input,
  Select,
  SimpleGrid,
  Stack,
  Text,
} from "@chakra-ui/react";
import { Employee } from "../lib/api";

export type PayrollFormValues = {
  employeeId: number;
  period: string;
  baseSalary: number;
  overtimeHours: number;
  overtimeRate: number;
  bonuses: number;
  deductions: number;
};

type Props = {
  employees: Employee[];
  onSubmit: (values: PayrollFormValues) => Promise<void> | void;
  submitLabel?: string;
};

export default function PayrollForm({ employees, onSubmit, submitLabel }: Props) {
  const initialState: PayrollFormValues = {
    employeeId: 0,
    period: "",
    baseSalary: 0,
    overtimeHours: 0,
    overtimeRate: 0,
    bonuses: 0,
    deductions: 0,
  };
  const [form, setForm] = useState<PayrollFormValues>(initialState);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setForm(initialState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function handleNumberChange(key: keyof PayrollFormValues, value: string) {
    const parsed = Number(value);
    setForm((prev) => ({ ...prev, [key]: Number.isNaN(parsed) ? 0 : parsed }));
  }

  function handleTextChange(key: keyof PayrollFormValues, value: string) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const previewNet = useMemo(() => {
    return form.baseSalary + form.overtimeHours * form.overtimeRate + form.bonuses - form.deductions;
  }, [form]);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    if (form.employeeId <= 0) {
      setError("Seleccione un empleado");
      return;
    }
    if (!form.period.trim()) {
      setError("El período es obligatorio");
      return;
    }
    if (form.baseSalary < 0) {
      setError("El salario base debe ser mayor o igual a cero");
      return;
    }

    try {
      setSubmitting(true);
      await onSubmit({
        ...form,
        period: form.period.trim(),
      });
      setForm(initialState);
    } catch (err: any) {
      setError(err?.message || "No se pudo registrar la nómina");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Stack as="form" spacing={4} aria-label="payroll-form" onSubmit={handleSubmit}>
      {error && (
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          {error}
        </Alert>
      )}

      <FormControl>
        <FormLabel>Empleado</FormLabel>
        <Select
          value={form.employeeId || ""}
          onChange={(e) => handleNumberChange("employeeId", e.target.value)}
          placeholder="Seleccionar..."
          aria-label="payroll-employee"
        >
          {employees.map((emp) => (
            <option key={emp.id} value={emp.id}>
              {emp.name}
            </option>
          ))}
        </Select>
      </FormControl>

      <FormControl>
        <FormLabel>Período (YYYY-MM)</FormLabel>
        <Input
          value={form.period}
          onChange={(e) => handleTextChange("period", e.target.value)}
          aria-label="payroll-period"
        />
      </FormControl>

      <SimpleGrid columns={{ base: 1, md: 3 }} spacing={4}>
        <FormControl>
          <FormLabel>Salario base</FormLabel>
          <Input
            type="number"
            value={form.baseSalary}
            onChange={(e) => handleNumberChange("baseSalary", e.target.value)}
            min={0}
            aria-label="payroll-base-salary"
          />
        </FormControl>
        <FormControl>
          <FormLabel>Horas extra</FormLabel>
          <Input
            type="number"
            value={form.overtimeHours}
            onChange={(e) => handleNumberChange("overtimeHours", e.target.value)}
            min={0}
            aria-label="payroll-overtime-hours"
          />
        </FormControl>
        <FormControl>
          <FormLabel>Tarifa hora extra</FormLabel>
          <Input
            type="number"
            value={form.overtimeRate}
            onChange={(e) => handleNumberChange("overtimeRate", e.target.value)}
            min={0}
            aria-label="payroll-overtime-rate"
          />
        </FormControl>
        <FormControl>
          <FormLabel>Bonos</FormLabel>
          <Input
            type="number"
            value={form.bonuses}
            onChange={(e) => handleNumberChange("bonuses", e.target.value)}
            min={0}
            aria-label="payroll-bonuses"
          />
        </FormControl>
        <FormControl>
          <FormLabel>Deducciones</FormLabel>
          <Input
            type="number"
            value={form.deductions}
            onChange={(e) => handleNumberChange("deductions", e.target.value)}
            min={0}
            aria-label="payroll-deductions"
          />
        </FormControl>
      </SimpleGrid>

      <Text fontWeight="semibold">
        Neto estimado: <Text as="span">${previewNet.toFixed(2)}</Text>
      </Text>

      <Button type="submit" colorScheme="blue" isLoading={submitting} alignSelf="flex-start">
        {submitLabel ?? "Registrar nómina"}
      </Button>
    </Stack>
  );
}

