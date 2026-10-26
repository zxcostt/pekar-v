-- 1. Пакет CoreAndUsers: Сотрудники и роли
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(150) NOT NULL,
    login VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(30) NOT NULL CHECK (role IN ('CASHIER', 'BAKER', 'STOREKEEPER', 'MANAGER')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    -- Специфичные поля дочерних классов (Single Table Inheritance)
    register_number INT,
    workshop_station VARCHAR(100),
    warehouse_zone_id INT
);

-- 2. Пакет ProductionAndWarehouse: Сырьё, Продукция, Рецептуры и Выпуск
CREATE TABLE ingredients (
    ingredient_id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    current_stock NUMERIC(10, 3) NOT NULL DEFAULT 0 CHECK (current_stock >= 0),
    unit VARCHAR(20) NOT NULL,
    purchase_price NUMERIC(10, 2) NOT NULL CHECK (purchase_price >= 0)
);

CREATE TABLE recipes (
    recipe_id SERIAL PRIMARY KEY,
    recipe_title VARCHAR(150) NOT NULL,
    yield_quantity INT NOT NULL CHECK (yield_quantity > 0)
);

CREATE TABLE recipe_items (
    recipe_item_id SERIAL PRIMARY KEY,
    recipe_id INT NOT NULL REFERENCES recipes(recipe_id) ON DELETE CASCADE,
    ingredient_id INT NOT NULL REFERENCES ingredients(ingredient_id) ON DELETE RESTRICT,
    required_quantity NUMERIC(10, 3) NOT NULL CHECK (required_quantity > 0),
    UNIQUE (recipe_id, ingredient_id)
);

CREATE TABLE products (
    product_id SERIAL PRIMARY KEY,
    recipe_id INT REFERENCES recipes(recipe_id) ON DELETE SET NULL,
    name VARCHAR(150) NOT NULL,
    barcode VARCHAR(50) UNIQUE NOT NULL,
    retail_price NUMERIC(10, 2) NOT NULL CHECK (retail_price >= 0),
    current_quantity INT NOT NULL DEFAULT 0 CHECK (current_quantity >= 0)
);

CREATE TABLE production_orders (
    order_id SERIAL PRIMARY KEY,
    recipe_id INT NOT NULL REFERENCES recipes(recipe_id),
    baker_id INT NOT NULL REFERENCES users(id),
    order_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    produced_units INT NOT NULL CHECK (produced_units > 0),
    order_status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED' CHECK (order_status IN ('COMPLETED', 'CANCELLED')),
    cancel_reason TEXT
);

-- 3. Пакет SalesAndPOS: Чеки и Платежи
CREATE TABLE receipts (
    receipt_id SERIAL PRIMARY KEY,
    cashier_id INT NOT NULL REFERENCES users(id),
    receipt_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    discount_rate NUMERIC(5, 2) NOT NULL DEFAULT 0 CHECK (discount_rate >= 0 AND discount_rate <= 100),
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    is_refunded BOOLEAN NOT NULL DEFAULT FALSE,
    fiscal_doc_number VARCHAR(100) UNIQUE
);

CREATE TABLE receipt_items (
    receipt_item_id SERIAL PRIMARY KEY,
    receipt_id INT NOT NULL REFERENCES receipts(receipt_id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(product_id),
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(10, 2) NOT NULL CHECK (unit_price >= 0)
);

CREATE TABLE payments (
    payment_id SERIAL PRIMARY KEY,
    receipt_id INT UNIQUE NOT NULL REFERENCES receipts(receipt_id) ON DELETE CASCADE,
    payment_type VARCHAR(20) NOT NULL CHECK (payment_type IN ('CASH', 'CARD')),
    amount NUMERIC(10, 2) NOT NULL CHECK (amount >= 0),
    payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Поля для CashPayment
    tendered_amount NUMERIC(10, 2),
    change_amount NUMERIC(10, 2),
    -- Поля для CardPayment
    card_pan_masked VARCHAR(30),
    authorization_code VARCHAR(50)
);

-- 4. Начальные тестовые данные (Seed) для демонстрации
INSERT INTO users (full_name, login, password_hash, role, register_number, workshop_station, warehouse_zone_id) VALUES
('Иванова Анна (Кассир)', 'cashier', 'pass123', 'CASHIER', 1, NULL, NULL),
('Петров Сергей (Пекарь)', 'baker', 'pass123', 'BAKER', NULL, 'Печной цех №1', NULL),
('Сидоров Алексей (Кладовщик)', 'store', 'pass123', 'STOREKEEPER', NULL, NULL, 101),
('Дяченко Владислав (Управляющий)', 'admin', 'admin123', 'MANAGER', NULL, NULL, NULL);

INSERT INTO ingredients (name, current_stock, unit, purchase_price) VALUES
('Мука пшеничная в/с', 100.000, 'кг', 45.00),
('Масло сливочное 82.5%', 25.000, 'кг', 650.00),
('Сахар-песок', 50.000, 'кг', 60.00),
('Дрожжи прессованные', 5.000, 'кг', 180.00);

INSERT INTO recipes (recipe_title, yield_quantity) VALUES
('ТК №1: Круассан классический (партия 10 шт.)', 10);

INSERT INTO recipe_items (recipe_id, ingredient_id, required_quantity) VALUES
(1, 1, 0.500), -- 0.5 кг муки на 10 круассанов
(1, 2, 0.250), -- 0.25 кг масла
(1, 3, 0.100), -- 0.1 кг сахара
(1, 4, 0.020); -- 0.02 кг дрожжей

INSERT INTO products (recipe_id, name, barcode, retail_price, current_quantity) VALUES
(1, 'Круассан классический', '4601234567890', 85.00, 20);