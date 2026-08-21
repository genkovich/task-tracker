-- Bootstrap seed: the product always has exactly one board (CONTEXT.md invariant).
INSERT INTO boards (id)
VALUES ('019a0000-0000-7000-8000-000000000101')
ON CONFLICT (id) DO NOTHING;
