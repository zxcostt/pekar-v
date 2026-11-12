const API_BASE = 'http://localhost:8080/api'

async function request(endpoint, options = {}) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const data = await response.json()
  if (!response.ok) {
    throw new Error(data.error || 'Ошибка запроса к серверу')
  }
  return data
}

export const api = {
  // UC 1: Авторизация и пользователи
  login: (login, password) =>
    request('/auth/login', { method: 'POST', body: JSON.stringify({ login, password }) }),
  getUsers: () => request('/users'),
  createUser: (payload) =>
    request('/users', { method: 'POST', body: JSON.stringify(payload) }),
  deleteUser: (id, managerId) =>
    request(`/users/${id}${managerId ? `?managerId=${managerId}` : ''}`, { method: 'DELETE' }),

  // UC 2, 4, 5, 6, 7: Склад сырья, рецептуры и производство
  getIngredients: () => request('/ingredients'),
  supplyIngredient: (ingredientId, amount) =>
    request('/ingredients/supply', { method: 'POST', body: JSON.stringify({ ingredientId, amount }) }),
  writeOffIngredient: (ingredientId, amount, reason) =>
    request('/ingredients/writeoff', { method: 'POST', body: JSON.stringify({ ingredientId, amount, reason }) }),
  getRecipes: () => request('/recipes'),
  createRecipe: (payload) =>
    request('/recipes', { method: 'POST', body: JSON.stringify(payload) }),
  getProducts: () => request('/products'),
  getProductionOrders: () => request('/production/orders'),
  produceProduct: (productId, bakerId, producedUnits) =>
    request('/production/orders', { method: 'POST', body: JSON.stringify({ productId, bakerId, producedUnits }) }),
  getProductWriteOffs: () => request('/products/writeoffs'),
  writeOffProduct: (productId, userId, quantity, reason) =>
    request('/products/writeoff', { method: 'POST', body: JSON.stringify({ productId, userId, quantity, reason }) }),

  // UC 3, 8..14: Касса, продажи, возвраты и сменный отчёт
  getReceipts: () => request('/sales/receipts'),
  checkout: (payload) =>
    request('/sales/checkout', { method: 'POST', body: JSON.stringify(payload) }),
  refund: (fiscalDocNumber, reason) =>
    request('/sales/refund', { method: 'POST', body: JSON.stringify({ fiscalDocNumber, reason }) }),
  getShiftReport: () => request('/sales/shift-report'),

  // Кассовые смены (X-отчёт, Z-отчёт, инкассация)
  getCurrentShift: () => request('/sales/shifts/current'),
  openShift: (cashierId, initialCash, registerNumber = 1) =>
    request('/sales/shifts/open', {
      method: 'POST',
      body: JSON.stringify({ cashierId, initialCash, registerNumber })
    }),
  getXReport: () => request('/sales/shifts/current/x-report'),
  closeShift: (cashierId, actualCash, encashmentAmount) =>
    request('/sales/shifts/close', {
      method: 'POST',
      body: JSON.stringify({ cashierId, actualCash, encashmentAmount })
    }),
  getShifts: () => request('/sales/shifts'),
}