# Misi CRUD dan End Goal

Dokumen ini menjelaskan target akhir yang harus dicapai mahasiswa.

## Kondisi awal

Sebelum mulai, database harus berisi 3 data awal dari file [db/02_seed.sql](../db/02_seed.sql):

1. `Andi Saputra` - `andi.saputra@student.test`
2. `Siti Rahma` - `siti.rahma@student.test`
3. `Budi Santoso` - `budi.santoso@student.test`

## Misi yang harus dikerjakan

Mahasiswa wajib menyelesaikan empat operasi berikut melalui web interface yang ada pada aplikasi Go:

1. `Create`
Tambahkan user baru dengan data:
`Dina Pramesti` - `dina.pramesti@student.test`

2. `Read`
Tampilkan seluruh data user pada halaman daftar user.

3. `Update`
Ubah data dengan `id = 1` menjadi:
`Andi Pratama` - `andi.pratama@student.test`

4. `Delete`
Hapus data dengan `id = 3`

5. `Create` lagi
Tambahkan user baru dengan data:
`Raka Nugraha` - `raka.nugraha@student.test`

## End goal

Tugas dianggap selesai jika seluruh kondisi berikut terpenuhi:

1. Aplikasi Go berjalan di Ubuntu VM dan didaftarkan melalui aaPanel.
2. Aplikasi dapat diakses melalui browser dari luar mesin VM menggunakan domain atau IP publik yang sudah dikonfigurasi.
3. Seluruh operasi CRUD berjalan melalui antarmuka web, bukan hanya lewat phpMyAdmin.
4. Setelah semua misi selesai, isi tabel `users` harus sama secara logis dengan target akhir berikut:

| id | nama | email |
| --- | --- | --- |
| 1 | Andi Pratama | andi.pratama@student.test |
| 2 | Siti Rahma | siti.rahma@student.test |
| 4 | Dina Pramesti | dina.pramesti@student.test |
| 5 | Raka Nugraha | raka.nugraha@student.test |

Catatan:
- `id = 3` harus sudah hilang karena data tersebut dihapus.
- Nilai `id` 4 dan 5 diasumsikan muncul setelah proses create dilakukan berurutan dari data seed awal.

## Bukti yang harus dikumpulkan

1. Screenshot aplikasi yang terbuka dari browser melalui domain/IP VM.
2. Screenshot halaman daftar user yang menampilkan hasil akhir.
3. Screenshot form tambah data.
4. Screenshot form edit data.
5. Screenshot atau bukti proses hapus data.
6. SQL dump hasil akhir database.

## File referensi

- Skema tabel: [db/01_schema.sql](../db/01_schema.sql)
- Data awal: [db/02_seed.sql](../db/02_seed.sql)
- Contoh target akhir: [db/03_expected_final_state.sql](../db/03_expected_final_state.sql)
