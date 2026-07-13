"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  Alert,
  AlertIcon,
  Box,
  Button,
  Container,
  Heading,
  Stack,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  Text,
  Card,
  CardBody,
  Badge,
} from "@chakra-ui/react";
import EmployeeForm from "../../components/EmployeeForm";
import { Employee, getEmployees, createEmployee, updateEmployee, deleteEmployee } from "../../lib/api";

export default function EmployeesPage() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const editingEmployee = useMemo(() => employees?.find((e) => e.id === editingId) || null, [employees, editingId]);

  async function refresh() {
    try {
      setError(null);
      const data = await getEmployees();
      setEmployees(data);
    } catch (e: any) {
      setError(e?.message || "Failed to load");
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  async function handleCreate(name: string) {
    const emp = await createEmployee(name);
    setEmployees((prev) => [...(prev || []), emp]);
  }

  async function handleUpdate(name: string) {
    if (editingEmployee) {
      const updated = await updateEmployee(editingEmployee.id, name);
      setEmployees((prev) => prev.map((e) => (e.id === updated.id ? updated : e)));
      setEditingId(null);
    }
  }

  async function handleDelete(id: number) {
    const target = employees.find((e) => e.id === id);
    const label = target ? target.name : `ID ${id}`;
    if (typeof window !== "undefined" && !window.confirm(`¿Eliminar ${label}? Esta acción no se puede deshacer.`)) {
      return;
    }
    await deleteEmployee(id);
    setEmployees((prev) => prev.filter((e) => e.id !== id));
    if (editingId === id) {
      setEditingId(null);
    }
  }

  return (
    <Box bg="gray.50" minH="100vh" py={10}>
      <Container maxW="5xl">
        <Stack spacing={6}>
          <Stack direction={{ base: "column", md: "row" }} align={{ base: "flex-start", md: "center" }} justify="space-between">
            <div>
              <Heading size="lg" color="brand.700">
                Gestión de empleados
              </Heading>
              <Text color="gray.600">Creá o actualizá personas dentro del equipo. Asegura que se actualice.</Text>
            </div>
            <Button as={Link} href="/" variant="ghost" colorScheme="blue">
              ← Volver al inicio
            </Button>
          </Stack>

          {error && (
            <Alert status="error" borderRadius="md">
              <AlertIcon />
              {error}
            </Alert>
          )}

          <Card>
            <CardBody>
              <Heading size="md" mb={4}>
                Crear empleado
              </Heading>
              <EmployeeForm onSubmit={handleCreate} submitLabel="Crear" />
            </CardBody>
          </Card>

          {editingEmployee && (
            <Card borderColor="brand.200" borderWidth="1px">
              <CardBody>
                <Heading size="md" mb={2}>
                  Editando: {editingEmployee.name}
                </Heading>
                <EmployeeForm
                  initialName={editingEmployee.name}
                  onSubmit={handleUpdate}
                  submitLabel="Actualizar"
                />
              </CardBody>
            </Card>
          )}

          <Card>
            <CardBody>
              <Stack direction="row" justify="space-between" align="center" mb={4}>
                <Heading size="md">Listado</Heading>
                <Badge colorScheme="blue">{employees.length} personas</Badge>
              </Stack>
              <Box overflowX="auto">
                <Table variant="simple">
                  <Thead bg="gray.100">
                    <Tr>
                      <Th>ID</Th>
                      <Th>Nombre</Th>
                      <Th>Acciones</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {employees?.map((e) => (
                      <Tr key={e.id}>
                        <Td>{e.id}</Td>
                        <Td>{e.name}</Td>
                        <Td>
                          <Stack direction="row" spacing={3}>
                            <Button size="sm" variant="outline" onClick={() => setEditingId(e.id)}>
                              Editar
                            </Button>
                            <Button size="sm" colorScheme="red" variant="outline" onClick={() => handleDelete(e.id)}>
                              Eliminar
                            </Button>
                          </Stack>
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </Box>
            </CardBody>
          </Card>
        </Stack>
      </Container>
    </Box>
  );
}


