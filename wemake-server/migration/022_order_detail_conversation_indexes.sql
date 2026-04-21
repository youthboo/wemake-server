CREATE INDEX IF NOT EXISTS idx_conversations_customer_id ON conversations(customer_id);
CREATE INDEX IF NOT EXISTS idx_conversations_factory_id ON conversations(factory_id);
CREATE INDEX IF NOT EXISTS idx_quotations_rfq_id ON quotations(rfq_id);
