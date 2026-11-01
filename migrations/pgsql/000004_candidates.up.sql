-- Candidate identity. Email is contact only and is not unique.
-- Apply after versions 1–2. Do not AutoMigrate this table.

CREATE TABLE candidates (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT candidates_created_by_fkey FOREIGN KEY (created_by) REFERENCES administrative_accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT candidates_updated_at_check CHECK (updated_at >= created_at)
);

CREATE INDEX "candidates_created_by_INDEX" ON candidates (created_by);
