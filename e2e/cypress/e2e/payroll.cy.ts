// payroll.cy.ts — TEST DE INTEGRACIÓN (E2E) del flujo de nómina.
//
// Diferencia clave con un test unitario: acá NO hay mocks. Cypress abre un
// navegador real contra el frontend DEPLOYADO en Railway DEV, que le pega
// al backend real, que escribe en la PostgreSQL real. Se prueba la
// FUNCIONALIDAD COMPLETA de punta a punta.
//
// API_URL la inyecta el pipeline (--env API_URL=...) y se usa solo para
// preparar datos de prueba vía cy.request (crear el empleado antes de
// cargar su nómina). La navegación (cy.visit) va contra FRONTEND_URL,
// definida en cypress.config.
//
// Este spec es el que usamos para el demo del caso 3: si alguien cambia el
// aria-label "payroll-form" en PayrollForm.tsx, Jest no lo nota (no testea
// el DOM deployado) pero este cy.get() no encuentra el formulario y falla.
const API_URL = Cypress.env('API_URL') || 'http://localhost:8080';

describe('Payroll flow', () => {
  it('registers a payroll record and shows totals', () => {
    const employeeName = `Payroll User ${Date.now()}`;
    cy.request('POST', `${API_URL}/employees`, { name: employeeName });

    cy.visit('/payroll');

    cy.get('form[aria-label="payroll-form"]').within(() => {
      cy.get('[aria-label="payroll-employee"]').select(employeeName);
      cy.get('[aria-label="payroll-period"]').type('2024-12');
      cy.get('[aria-label="payroll-base-salary"]').clear().type('1000');
      cy.get('[aria-label="payroll-overtime-hours"]').clear().type('5');
      cy.get('[aria-label="payroll-overtime-rate"]').clear().type('60');
      cy.get('[aria-label="payroll-bonuses"]').clear().type('150');
      cy.get('[aria-label="payroll-deductions"]').clear().type('50');
      cy.contains('button', 'Registrar nómina').click();
    });

    cy.contains('td', employeeName)
      .parent('tr')
      .within(() => {
        cy.contains('td', '$');
        cy.contains('td', '2024-12');
      });

    cy.contains('.chakra-card', 'Total acumulado').should('contain.text', '$');
  });
});


