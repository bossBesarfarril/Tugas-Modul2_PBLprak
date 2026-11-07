CREATE TABLE IF NOT EXISTS prestasi (
    id            SERIAL                PRIMARY KEY,
    id_mahasiswa  INT                   NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    nama_prestasi VARCHAR(100)          NOT NULL,
    juara         VARCHAR(50)           NOT NULL
);

INSERT INTO prestasi (id_mahasiswa, nama_prestasi, juara)
SELECT s.id, p.nama_prestasi, p.juara
FROM (VALUES
    (1, 'Juara Lomba 1', 'Juara 1'),
    (2, 'Juara Lomba 2', 'Juara 2'),
    (3, 'Juara Lomba 3', 'Juara 3')
) AS p(id_mahasiswa, nama_prestasi, juara)
JOIN students s ON s.id = p.id_mahasiswa
ON CONFLICT DO NOTHING;
