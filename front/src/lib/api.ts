// api.ts — CLIENTE HTTP del frontend.
// Único punto del frontend que habla con el backend: los componentes React
// importan estas funciones (getEmployees, createPayroll, etc.) y nunca hacen
// fetch directo. Los tipos de acá espejan los structs JSON del backend Go.

export type Employee = { id: number; name: string };

export type PerformanceReview = {
  id: number;
  employeeId: number;
  employeeName: string;
  period: string;
  reviewer: string;
  rating: number;
  strengths: string;
  opportunities: string;
  state: string;
};

export type ReviewAggregate = {
  employeeId: number;
  employeeName: string;
  averageRating: number;
  latestState: string;
  count: number;
};

export type ReviewListResponse = {
  items: PerformanceReview[];
  aggregates: ReviewAggregate[];
};

export type PayrollRecord = {
  id: number;
  employeeId: number;
  employeeName: string;
  period: string;
  baseSalary: number;
  overtimeHours: number;
  overtimeRate: number;
  bonuses: number;
  deductions: number;
  netPay: number;
};

export type PayrollPeriodTotal = {
  period: string;
  totalNet: number;
};

export type PayrollListResponse = {
  items: PayrollRecord[];
  aggregates: {
    totalsByPeriod: PayrollPeriodTotal[];
    grandTotalNet: number;
  };
};

// resolveApiBase decide a qué backend pegarle. Este es el truco clave del
// deploy: en Next.js las variables NEXT_PUBLIC_* se "hornean" EN BUILD TIME,
// pero nosotros construimos UNA sola imagen Docker y la usamos en DEV y PROD
// con backends distintos. La solución es inyección en RUNTIME:
//   1. Al arrancar el container, runtime-entrypoint.sh lee NEXT_PUBLIC_API_URL
//      del entorno (configurada en Railway) y escribe public/runtime-env.js
//   2. Ese script define window.__ENV__ en el navegador
//   3. Acá leemos primero window.__ENV__ (runtime) y solo si no existe caemos
//      a process.env (build time) o a localhost:8080 (desarrollo local)
// El chequeo de "__NEXT_PUBLIC_API_URL__" descarta el placeholder sin
// reemplazar (cuando el entrypoint no corrió).
function resolveApiBase(): string {
  if (typeof window !== "undefined") {
    const runtime = (window as any).__ENV__?.NEXT_PUBLIC_API_URL;
    if (runtime && !runtime.includes("__NEXT_PUBLIC_API_URL__")) {
      return runtime;
    }
  }

  const fromProcess =
    process.env.NEXT_PUBLIC_API_URL ||
    process.env["NEXT_PUBLIC_API_URL"] ||
    process.env.API_URL ||
    "http://localhost:8080";

  return fromProcess;
}

function apiUrl(path: string): string {
  const base = resolveApiBase();
  return `${base}${path}`;
}

// handleJson centraliza el manejo de respuestas: si el status no es 2xx,
// lanza un Error con el mensaje del campo {"error": ...} que manda el
// backend — los componentes lo capturan y lo muestran en un <Alert>.
async function handleJson<T>(res: Response): Promise<T> {
  const text = await res.text();
  const data = text ? JSON.parse(text) : undefined;
  if (!res.ok) {
    const message = data?.error || res.statusText || "Request failed";
    throw new Error(message);
  }
  return data as T;
}

export async function getEmployees(): Promise<Employee[]> {
  const res = await fetch(apiUrl("/employees"), { cache: "no-store" });
  return handleJson<Employee[]>(res);
}

export async function createEmployee(name: string): Promise<Employee> {
  const res = await fetch(apiUrl("/employees"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: name.trim() }),
  });
  return handleJson<Employee>(res);
}

export async function updateEmployee(
  id: number,
  name: string
): Promise<Employee> {
  const res = await fetch(apiUrl(`/employees/${id}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: name.trim() }),
  });
  return handleJson<Employee>(res);
}

export async function deleteEmployee(id: number): Promise<void> {
  const res = await fetch(apiUrl(`/employees/${id}`), { method: "DELETE" });
  if (!res.ok) {
    const text = await res.text();
    const data = text ? JSON.parse(text) : undefined;
    throw new Error(data?.error || res.statusText || "Request failed");
  }
}

export async function getReviews(
  params: { employeeId?: number; period?: string; state?: string } = {}
): Promise<ReviewListResponse> {
  const query = new URLSearchParams();
  if (params.employeeId) query.set("employeeId", String(params.employeeId));
  if (params.period) query.set("period", params.period);
  if (params.state) query.set("state", params.state);
  const qs = query.toString();
  const res = await fetch(apiUrl(`/reviews${qs ? `?${qs}` : ""}`), {
    cache: "no-store",
  });
  return handleJson<ReviewListResponse>(res);
}

export async function createReview(payload: {
  employeeId: number;
  period: string;
  reviewer: string;
  rating: number;
  strengths?: string;
  opportunities?: string;
}): Promise<PerformanceReview> {
  const res = await fetch(apiUrl("/reviews"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      ...payload,
      period: payload.period.trim(),
      reviewer: payload.reviewer.trim(),
      strengths: payload.strengths?.trim() ?? "",
      opportunities: payload.opportunities?.trim() ?? "",
    }),
  });
  return handleJson<PerformanceReview>(res);
}

export async function updateReview(
  id: number,
  payload: {
    reviewer?: string;
    rating?: number;
    strengths?: string;
    opportunities?: string;
  }
): Promise<PerformanceReview> {
  const res = await fetch(apiUrl(`/reviews/${id}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      ...payload,
      reviewer: payload.reviewer?.trim(),
      strengths: payload.strengths?.trim(),
      opportunities: payload.opportunities?.trim(),
    }),
  });
  return handleJson<PerformanceReview>(res);
}

export async function transitionReview(
  id: number,
  state: string
): Promise<PerformanceReview> {
  const res = await fetch(apiUrl(`/reviews/${id}/status`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ state: state.trim() }),
  });
  return handleJson<PerformanceReview>(res);
}

export async function getPayroll(
  params: { employeeId?: number; period?: string } = {}
): Promise<PayrollListResponse> {
  const query = new URLSearchParams();
  if (params.employeeId) query.set("employeeId", String(params.employeeId));
  if (params.period) query.set("period", params.period);
  const qs = query.toString();
  const res = await fetch(apiUrl(`/payroll${qs ? `?${qs}` : ""}`), {
    cache: "no-store",
  });
  return handleJson<PayrollListResponse>(res);
}

export async function createPayroll(payload: {
  employeeId: number;
  period: string;
  baseSalary: number;
  overtimeHours?: number;
  overtimeRate?: number;
  bonuses?: number;
  deductions?: number;
}): Promise<PayrollRecord> {
  const res = await fetch(apiUrl("/payroll"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      ...payload,
      period: payload.period.trim(),
      overtimeHours: payload.overtimeHours ?? 0,
      overtimeRate: payload.overtimeRate ?? 0,
      bonuses: payload.bonuses ?? 0,
      deductions: payload.deductions ?? 0,
    }),
  });
  return handleJson<PayrollRecord>(res);
}
