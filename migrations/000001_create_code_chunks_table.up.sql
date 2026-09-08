CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS code_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    chunk_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    signature TEXT,
    doc_comment TEXT,
    content TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    
    content_hash CHAR(64) NOT NULL,
    
    embedding vector(1536),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_code_chunks_repo_hash 
ON code_chunks (repo_name, content_hash);

CREATE INDEX IF NOT EXISTS idx_code_chunks_repo_path 
ON code_chunks (repo_name, file_path);