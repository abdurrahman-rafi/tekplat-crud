# TekPlat CRUD Go di aaPanel

Repository ini berisi aplikasi CRUD sederhana menggunakan Go dan MySQL untuk praktikum Teknologi Platform. Fokus utamanya adalah:

1. menjalankan aplikasi Go di Ubuntu VM,
2. menghubungkan aplikasi ke database yang dibuat di aaPanel,
3. mengakses aplikasi dari browser melalui aaPanel,
4. menyelesaikan misi CRUD sampai kondisi akhir tertentu,
5. mengumpulkan bukti berupa screenshot dan SQL dump.

## Isi repository

- [cmd/web/main.go](cmd/web/main.go): entry point aplikasi
- [internal/config/config.go](internal/config/config.go): konfigurasi environment
- [internal/store/user_store.go](internal/store/user_store.go): akses database MySQL
- [internal/web/app.go](internal/web/app.go): routing dan handler CRUD
- [db/01_schema.sql](db/01_schema.sql): skema tabel
- [db/02_seed.sql](db/02_seed.sql): data awal
- [docs/MISI-DAN-ENDGOAL.md](docs/MISI-DAN-ENDGOAL.md): target tugas
- [submissions/CONTOH-FORMAT-SUBMISSION.md](submissions/CONTOH-FORMAT-SUBMISSION.md): format pengumpulan

## Quick start lokal

1. Salin file environment:

```bash
cp .env.example .env
```

2. Buat database `crud_db` dan import file SQL:

```sql
SOURCE db/01_schema.sql;
SOURCE db/02_seed.sql;
```

3. Jalankan aplikasi:

```bash
go mod tidy
go run ./cmd/web
```

4. Buka `http://localhost:8080`

Catatan:
- Aplikasi akan otomatis membaca file `.env` jika file tersebut ada di root project.

## Dokumen penting

- [docs/PANDUAN-DEPLOYMENT-AAPANEL.md](docs/PANDUAN-DEPLOYMENT-AAPANEL.md)
- [docs/MISI-DAN-ENDGOAL.md](docs/MISI-DAN-ENDGOAL.md)
- [docs/REFERENSI-AAPANEL.md](docs/REFERENSI-AAPANEL.md)

## Catatan

Antarmuka frontend akan dipakai untuk operasi CRUD mahasiswa. Tugas utama mereka bukan membangun UI, tetapi memastikan aplikasi berjalan dan seluruh CRUD tersimpan ke database aaPanel.
