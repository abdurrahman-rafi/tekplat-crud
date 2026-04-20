# Panduan Setup dan Deployment di aaPanel

Dokumen ini adalah panduan utama untuk mahasiswa. Semua langkah ditulis dalam Bahasa Indonesia dan mengasumsikan aplikasi Go akan dijalankan di Ubuntu VM yang dikelola melalui aaPanel.

## Tujuan praktikum

Setelah mengikuti panduan ini, mahasiswa harus bisa:

1. meng-clone repository aplikasi CRUD,
2. membuat database di aaPanel,
3. menghubungkan aplikasi Go ke database tersebut,
4. menjalankan aplikasi Go di Ubuntu VM,
5. mendaftarkan aplikasi ke aaPanel agar bisa diakses lewat browser dari luar mesin,
6. menyelesaikan misi CRUD,
7. mengumpulkan screenshot dan SQL dump hasil akhir.

## Arsitektur yang diasumsikan

Arsitektur praktikum ini menggunakan komponen berikut:

1. Ubuntu VM sebagai server.
2. aaPanel sebagai panel manajemen server.
3. MySQL atau MariaDB yang dibuat dari aaPanel.
4. Aplikasi Go dari repo ini sebagai web app CRUD.
5. Domain, subdomain, atau IP publik yang diarahkan ke VM.

Alur sederhananya:

`Browser` -> `aaPanel` -> `Go Project / reverse proxy` -> `Aplikasi Go` -> `MySQL/MariaDB`

## Prasyarat

Sebelum mulai, pastikan mahasiswa punya:

1. akses ke Ubuntu VM,
2. akses login ke aaPanel,
3. Git terpasang di VM,
4. Go terpasang di VM,
5. database engine MySQL/MariaDB aktif di aaPanel,
6. domain atau IP publik yang bisa dipakai untuk akses dari luar VM.

## Struktur file yang penting

- [README.md](../README.md)
- [db/01_schema.sql](../db/01_schema.sql)
- [db/02_seed.sql](../db/02_seed.sql)
- [docs/MISI-DAN-ENDGOAL.md](./MISI-DAN-ENDGOAL.md)
- [submissions/CONTOH-FORMAT-SUBMISSION.md](../submissions/CONTOH-FORMAT-SUBMISSION.md)

## Langkah 1 - Clone repository

Masuk ke Ubuntu VM, lalu clone repository:

```bash
git clone <url-repository> tekplat-crud
cd tekplat-crud
```

Jika repository belum dipush ke GitHub, salin dulu isi folder repo ini ke VM dengan metode yang tersedia, misalnya `scp`, `rsync`, atau upload manual.

## Langkah 2 - Siapkan database di aaPanel

Bagian ini mengikuti alur yang dijelaskan pada PDF tugas.

1. Login ke aaPanel.
2. Buka menu `Database`.
3. Klik `Add Database`.
4. Buat database dengan contoh nilai berikut:
   - Database name: `crud_db`
   - Username: `crud_user`
   - Password: bebas, simpan baik-baik
5. Setelah database terbentuk, buka `phpMyAdmin`.
6. Pilih database `crud_db`.
7. Jalankan file SQL schema:

```sql
CREATE TABLE IF NOT EXISTS users (
    id INT NOT NULL AUTO_INCREMENT,
    nama VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email)
);
```

Atau langsung copy isi file [db/01_schema.sql](../db/01_schema.sql).

8. Setelah tabel terbentuk, jalankan data awal dari file [db/02_seed.sql](../db/02_seed.sql).

Jika lebih nyaman memakai terminal MySQL, gunakan pola berikut:

```bash
mysql -u crud_user -p crud_db < db/01_schema.sql
mysql -u crud_user -p crud_db < db/02_seed.sql
```

## Langkah 3 - Siapkan environment aplikasi

Di Ubuntu VM, dari folder project:

```bash
cp .env.example .env
```

Edit file `.env` sesuai database yang dibuat di aaPanel:

```env
APP_ADDR=:8080
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=crud_db
DB_USER=crud_user
DB_PASSWORD=password_yang_dibuat_di_aapanel
```

Catatan:
- Jika database berada pada host berbeda, ganti `DB_HOST`.
- Jika port database bukan `3306`, sesuaikan `DB_PORT`.
- Jika port aplikasi `8080` sedang dipakai service lain pada mesin lokal, ganti `APP_ADDR`, misalnya menjadi `:8090`.

## Langkah 4 - Install dependency dan jalankan aplikasi

Masih dari folder project:

```bash
go mod tidy
go run ./cmd/web
```

Jika berhasil, server akan berjalan di `:8080`.

Untuk pengujian cepat di VM:

```bash
curl http://127.0.0.1:8080/healthz
```

Hasil yang diharapkan:

```text
ok
```

## Langkah 5 - Build binary produksi

Supaya lebih rapi saat didaftarkan ke aaPanel, build binary terlebih dahulu:

```bash
mkdir -p bin
go build -o bin/tekplat-crud ./cmd/web
```

Setelah itu jalankan:

```bash
./bin/tekplat-crud
```

Catatan:
- Aplikasi akan otomatis membaca file `.env` dari root project.
- Jika aaPanel menjalankan binary dari working directory lain, pastikan environment variable tetap disediakan dari konfigurasi process aaPanel.

## Langkah 6 - Daftarkan aplikasi di aaPanel

### Opsi utama: gunakan `Go Project`

Berdasarkan dokumentasi aaPanel, ini adalah opsi yang paling cocok untuk aplikasi Go.

Langkah umum yang diasumsikan:

1. Buka menu `Website` atau menu plugin `Go Project` di aaPanel.
2. Pilih `Add Go Project`.
3. Masukkan domain atau subdomain yang akan dipakai.
4. Pilih path project atau path binary.
5. Atur port internal aplikasi, misalnya `8080`.
6. Pastikan execution command mengarah ke binary aplikasi Go.
7. Pastikan environment variable database ikut dimasukkan jika aaPanel meminta konfigurasi environment terpisah.
8. Simpan konfigurasi dan jalankan project.

Hal yang harus dipastikan:

1. aplikasi Go benar-benar listen di port yang sama dengan konfigurasi aaPanel,
2. domain diarahkan ke project yang benar,
3. proses aplikasi berstatus running,
4. firewall VM membuka port web yang dipakai oleh Nginx/aaPanel, biasanya `80` dan `443`.

### Opsi fallback: Website + reverse proxy

Jika plugin `Go Project` tidak tersedia, gunakan site biasa di aaPanel lalu reverse proxy ke aplikasi Go yang berjalan pada `127.0.0.1:8080`.

Alur fallback:

1. buat site/domain di aaPanel,
2. jalankan binary Go secara manual di background atau melalui process manager,
3. aktifkan reverse proxy dari site tersebut ke `http://127.0.0.1:8080`.

## Langkah 7 - Uji akses dari browser luar

Setelah konfigurasi aaPanel selesai:

1. buka domain atau IP publik dari browser di laptop,
2. pastikan halaman daftar user muncul,
3. pastikan data seed awal tampil,
4. jika gagal, cek:
   - status aplikasi Go,
   - port internal,
   - mapping domain di aaPanel,
   - firewall VM,
   - DNS domain jika memakai domain.

## Langkah 8 - Kerjakan misi CRUD

Ikuti misi pada file [docs/MISI-DAN-ENDGOAL.md](./MISI-DAN-ENDGOAL.md).

Ringkasannya:

1. Tambah `Dina Pramesti`
2. Pastikan data bisa dibaca di halaman daftar
3. Ubah data `id = 1` menjadi `Andi Pratama`
4. Hapus data `id = 3`
5. Tambah `Raka Nugraha`

Seluruh operasi harus dilakukan lewat web interface, bukan hanya dengan query manual.

## Langkah 9 - Export SQL dump hasil akhir

### Opsi 1: dari phpMyAdmin

1. Buka database `crud_db`
2. Klik `Export`
3. Pilih mode `Quick`
4. Pilih format `SQL`
5. Download file hasil export

### Opsi 2: dari terminal

```bash
mysqldump -u crud_user -p crud_db > nim_nama_crud_db.sql
```

File inilah yang nantinya dikumpulkan bersama screenshot.

Jika client `mysqldump` menampilkan error terkait privilege `PROCESS` atau tablespaces, gunakan:

```bash
mysqldump --no-tablespaces -u crud_user -p crud_db > nim_nama_crud_db.sql
```

## Langkah 10 - Siapkan submission

Ikuti format pada file [submissions/CONTOH-FORMAT-SUBMISSION.md](../submissions/CONTOH-FORMAT-SUBMISSION.md).

Checklist minimum:

1. link repository,
2. screenshot aplikasi yang dapat diakses dari browser,
3. screenshot bukti Create,
4. screenshot bukti Read,
5. screenshot bukti Update,
6. screenshot bukti Delete,
7. SQL dump final.

## Troubleshooting

### Aplikasi tidak bisa konek ke database

Periksa:

1. isi `.env`,
2. username dan password database,
3. host dan port database,
4. apakah database `crud_db` sudah dibuat,
5. apakah tabel `users` sudah di-import.

### Halaman kosong atau error 500

Periksa:

1. log terminal aplikasi Go,
2. apakah folder `web/templates` dan `web/static` ikut ter-copy ke VM,
3. apakah binary dijalankan dari folder project yang benar.

### Domain tidak bisa diakses dari luar VM

Periksa:

1. status site atau Go Project di aaPanel,
2. DNS domain,
3. firewall Ubuntu,
4. security group cloud jika VM berjalan di provider cloud,
5. reverse proxy atau mapping domain.
