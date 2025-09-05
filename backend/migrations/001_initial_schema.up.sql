-- Create products table
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    price INTEGER NOT NULL DEFAULT 0, -- in cents
    version INTEGER NOT NULL DEFAULT 1, -- for optimistic locking
    created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products (id),
    buyer_id VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    total_cents INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    merchant_id VARCHAR(255) NOT NULL,
    amount_cents INTEGER NOT NULL,
    fee_cents INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    paid_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL,
        created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create settlements table
CREATE TABLE IF NOT EXISTS settlements (
    id SERIAL PRIMARY KEY,
    merchant_id VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    gross_cents INTEGER NOT NULL DEFAULT 0,
    fee_cents INTEGER NOT NULL DEFAULT 0,
    net_cents INTEGER NOT NULL DEFAULT 0,
    txn_count INTEGER NOT NULL DEFAULT 0,
    generated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL,
        unique_run_id UUID NOT NULL,
        created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        UNIQUE (merchant_id, date)
);

-- Create jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'QUEUED',
    progress DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
    processed INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    parameters JSONB NOT NULL,
    result_path TEXT,
    download_url TEXT,
    error TEXT,
    started_at TIMESTAMP
    WITH
        TIME ZONE,
        completed_at TIMESTAMP
    WITH
        TIME ZONE,
        created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_orders_buyer_id ON orders (buyer_id);

CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at);

CREATE INDEX IF NOT EXISTS idx_transactions_merchant_id ON transactions (merchant_id);

CREATE INDEX IF NOT EXISTS idx_transactions_paid_at ON transactions (paid_at);

CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions (status);

CREATE INDEX IF NOT EXISTS idx_settlements_merchant_id ON settlements (merchant_id);

CREATE INDEX IF NOT EXISTS idx_settlements_date ON settlements (date);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status);

CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at);

-- Insert sample product
INSERT INTO
    products (name, stock, price)
VALUES (
        'Limited Edition Product',
        100,
        9999
    ) ON CONFLICT DO NOTHING;