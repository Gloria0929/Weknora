-- This repair migration is intentionally non-destructive. Removing it must
-- not drop an embeddings table that may contain production vector data.
DO $$
BEGIN
    RAISE NOTICE '[Migration 000113] embeddings repair rollback is a no-op';
END $$;
