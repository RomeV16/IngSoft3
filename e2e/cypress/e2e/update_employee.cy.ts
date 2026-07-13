describe("Update employee", () => {
  it("edits the last employee and verifies the change", () => {
    const apiUrl = Cypress.env("API_URL") || "http://localhost:8080";
    // limpiar residuos de corridas anteriores para que el test sea determinista
    cy.request(`${apiUrl}/employees`).then((resp) => {
      const leftovers = resp.body.filter(
        (e: { id: number; name: string }) =>
          e.name === "User To Edit" || e.name === "User Edited"
      );
      leftovers.forEach((e: { id: number }) => {
        cy.request("DELETE", `${apiUrl}/employees/${e.id}`);
      });
    });

    cy.visit("/employees");
    // ensure at least one employee exists
    cy.get('form[aria-label="employee-form"] input[aria-label="name"]').type(
      "User To Edit"
    );
    cy.contains("button", "Crear").click();
    cy.wait(200);
    // click edit on last row
    cy.contains("tr", "User To Edit").within(() => {
      cy.contains("button", "Editar").click();
    });
    // target the last (edit) form explicitly
    cy.get('form[aria-label="employee-form"]')
      .last()
      .find('input[aria-label="name"]')
      .clear()
      .type("User Edited");
    cy.contains("button", "Actualizar").click();
    cy.wait(200);
    cy.get("table tbody tr:last-child td:nth-child(2)").should(
      "have.text",
      "User Edited"
    );
  });
});
