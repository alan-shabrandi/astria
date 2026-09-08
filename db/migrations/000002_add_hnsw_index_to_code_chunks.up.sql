SET maintenance_work_mem = '1GB';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_code_chunks_embedding_hnsw 
ON code_chunks 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);