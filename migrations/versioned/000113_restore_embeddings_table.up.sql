-- Repair databases where migration 000002 was recorded while PostgreSQL
-- retrieval was disabled. That migration intentionally skipped the table in
-- that case, but changing RETRIEVE_DRIVER later does not replay old versions.
DO $$
BEGIN
    IF current_setting('app.skip_embedding', true) = 'true' THEN
        RAISE NOTICE '[Migration 000113] PostgreSQL retrieval is disabled; skipping embeddings repair';
        RETURN;
    END IF;

    CREATE EXTENSION IF NOT EXISTS vector;
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
    CREATE EXTENSION IF NOT EXISTS pg_search;

    CREATE TABLE IF NOT EXISTS embeddings (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        source_id VARCHAR(64) NOT NULL,
        source_type INTEGER NOT NULL,
        chunk_id VARCHAR(64),
        knowledge_id VARCHAR(64),
        knowledge_base_id VARCHAR(64),
        tag_id VARCHAR(36),
        content TEXT,
        dimension INTEGER NOT NULL,
        embedding halfvec,
        is_enabled BOOLEAN DEFAULT TRUE
    );

    -- Complete partially-created tables as well as entirely missing tables.
    ALTER TABLE embeddings ADD COLUMN IF NOT EXISTS tag_id VARCHAR(36);
    ALTER TABLE embeddings ADD COLUMN IF NOT EXISTS is_enabled BOOLEAN DEFAULT TRUE;

    CREATE UNIQUE INDEX IF NOT EXISTS embeddings_unique_source
        ON embeddings(source_id, source_type);
    CREATE INDEX IF NOT EXISTS idx_embeddings_tag_id ON embeddings(tag_id);
    CREATE INDEX IF NOT EXISTS idx_embeddings_is_enabled ON embeddings(is_enabled);
    CREATE INDEX IF NOT EXISTS idx_embeddings_knowledge_base_id
        ON embeddings(knowledge_base_id);

    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'embeddings_search_idx') THEN
        CREATE INDEX embeddings_search_idx ON embeddings
        USING bm25 (id, knowledge_base_id, content, knowledge_id, chunk_id)
        WITH (
            key_field = 'id',
            text_fields = '{
                "content": {
                  "tokenizer": {"type": "chinese_lindera"}
                }
            }'
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'embeddings_embedding_idx_3584'
           OR indexname LIKE 'embeddings_embedding%3584%'
    ) THEN
        CREATE INDEX embeddings_embedding_idx_3584 ON embeddings
        USING hnsw ((embedding::halfvec(3584)) halfvec_cosine_ops)
        WITH (m = 16, ef_construction = 64)
        WHERE (dimension = 3584);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'embeddings_embedding_idx_798'
           OR indexname LIKE 'embeddings_embedding%798%'
    ) THEN
        CREATE INDEX embeddings_embedding_idx_798 ON embeddings
        USING hnsw ((embedding::halfvec(798)) halfvec_cosine_ops)
        WITH (m = 16, ef_construction = 64)
        WHERE (dimension = 798);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'embeddings_embedding_idx_1024'
           OR indexname LIKE 'embeddings_embedding%1024%'
    ) THEN
        CREATE INDEX embeddings_embedding_idx_1024 ON embeddings
        USING hnsw ((embedding::halfvec(1024)) halfvec_cosine_ops)
        WITH (m = 16, ef_construction = 64)
        WHERE (dimension = 1024);
    END IF;

    RAISE NOTICE '[Migration 000113] embeddings table and indexes are ready';
END $$;
