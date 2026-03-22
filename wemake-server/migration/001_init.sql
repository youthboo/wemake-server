-- Create factories table
CREATE TABLE IF NOT EXISTS factories (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),
    address TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create entrepreneurs table
CREATE TABLE IF NOT EXISTS entrepreneurs (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),
    company VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create connections table
CREATE TABLE IF NOT EXISTS connections (
    id UUID PRIMARY KEY,
    factory_id UUID NOT NULL REFERENCES factories(id) ON DELETE CASCADE,
    entrepreneur_id UUID NOT NULL REFERENCES entrepreneurs(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'pending',
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_factories_email ON factories(email);
CREATE INDEX idx_entrepreneurs_email ON entrepreneurs(email);
CREATE INDEX idx_connections_factory_id ON connections(factory_id);
CREATE INDEX idx_connections_entrepreneur_id ON connections(entrepreneur_id);
