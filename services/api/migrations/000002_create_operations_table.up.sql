CREATE TABLE operations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    category TEXT NOT NULL,
    configuration JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name, version)
);

INSERT INTO operations (id, name, version, category, configuration) VALUES
    ('a1111111-0000-4000-8000-000000000001', 'image-to-pdf', 'v1', 'image', '{}'),
    ('a1111111-0000-4000-8000-000000000002', 'pdf-to-image', 'v1', 'pdf', '{}'),
    ('a1111111-0000-4000-8000-000000000003', 'docx-to-pdf', 'v1', 'office', '{}'),
    ('a1111111-0000-4000-8000-000000000004', 'pdf-merge', 'v1', 'pdf', '{}'),
    ('a1111111-0000-4000-8000-000000000005', 'pdf-split', 'v1', 'pdf', '{}'),
    ('a1111111-0000-4000-8000-000000000006', 'pdf-compress', 'v1', 'pdf', '{}');
