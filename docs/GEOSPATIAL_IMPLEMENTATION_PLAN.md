# Rencana Implementasi Data Geospasial Location Service

## 1. Tujuan

Menambah kemampuan geospasial ke `location-service` tanpa mengganti fondasi Go, PostgreSQL, Redis, dan frontend statis yang sudah berjalan.

Urutan implementasi:

1. Boundary polygon dan centroid wilayah.
2. Detail wilayah dan peta interaktif.
3. Data pulau.
4. Data luas, penduduk, dan logo setelah sumber datanya tervalidasi.

## 2. Sumber Data

Data kandidat berasal dari clone lokal `wilayah-indonesia-api`:

| Data | Lokasi sumber | Jumlah | Keputusan |
|---|---|---:|---|
| Wilayah administratif | `init-db/02-data.sql` | 91.599 | Tidak diimpor ulang; sudah ada di `data/wilayah.sql` |
| Boundary dan centroid | `init-db/03-boundaries-part1.sql.gz` sampai `06-boundaries-part4.sql.gz` | 91.241 | Implementasi fase 1 |
| Pulau | `init-db/02-data.sql` | 17.374 | Implementasi fase 3 |
| Penduduk | `init-db/02-data.sql` | 553 | Tunda sampai metadata sumber tersedia |
| Luas | `init-db/02-data.sql` | 552 | Tunda sampai data dibersihkan |
| Logo | `cahyadsn/wilayah_logo` | Provinsi dan kabupaten/kota | Opsional |

Hasil audit boundary terhadap `data/wilayah.sql`:

- 91.216 kode cocok.
- 383 wilayah tidak memiliki boundary.
- 25 boundary memakai kode yang tidak ada dalam data wilayah aktif.
- Seluruh 38 provinsi, 514 kabupaten/kota, dan 7.285 kecamatan memiliki boundary.
- 83.379 dari 83.762 desa/kelurahan memiliki boundary yang cocok.

Data polygon sumber memakai pasangan koordinat `[latitude, longitude]` agar langsung dibaca Leaflet. Format ini bukan GeoJSON standar, yang memakai `[longitude, latitude]`.

## 3. Prinsip Implementasi

- Pertahankan Go standard library HTTP server.
- Pertahankan PostgreSQL sebagai sumber data utama dan Redis sebagai cache opsional.
- Pertahankan frontend HTML, CSS, dan JavaScript tanpa migrasi ke React.
- Impor hanya boundary dengan kode yang ada di `raw_locations`.
- Simpan polygon sebagai `jsonb`, bukan `text`.
- Jangan menambah PostGIS sebelum ada kebutuhan pencarian spasial, point-in-polygon, atau perhitungan jarak.
- Jangan mengirim seluruh polygon turunan provinsi dalam satu response.
- Boundary boleh tidak tersedia; detail wilayah tetap harus berhasil.

## 4. Fase 1: Boundary dan Centroid

### 4.1 Skema database

Tambahkan migration baru, tidak mengubah migration awal yang sudah dipakai deployment aktif.

```sql
CREATE TABLE IF NOT EXISTS location_boundaries (
    code varchar(13) PRIMARY KEY
        REFERENCES raw_locations(code) ON DELETE CASCADE,
    centroid_lat double precision NOT NULL,
    centroid_lng double precision NOT NULL,
    leaflet_path jsonb NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now(),
    CHECK (centroid_lat BETWEEN -90 AND 90),
    CHECK (centroid_lng BETWEEN -180 AND 180)
);
```

Nama boundary diambil dari `raw_locations`, sehingga tidak diduplikasi. `leaflet_path` dipilih sebagai nama eksplisit karena urutan koordinat sumber mengikuti Leaflet dan belum dikonversi menjadi GeoJSON. Index koordinat belum diperlukan karena fase ini belum menjalankan spatial query.

### 4.2 Importer boundary

Tambahkan command:

```text
go run . import-boundaries -dir <path-ke-init-db>
```

Alur importer:

1. Baca seluruh file `*-boundaries-*.sql.gz` secara streaming dengan `compress/gzip`.
2. Parse `code`, `name`, `latitude`, `longitude`, dan `path`.
3. Validasi kode terhadap `raw_locations`.
4. Validasi `latitude` pada rentang `-90..90` dan `longitude` pada rentang `-180..180`.
5. Validasi `path` sebagai JSON array.
6. Abaikan dan catat boundary yang kodenya tidak dikenal.
7. Impor data valid dalam satu transaksi menggunakan PostgreSQL `COPY` atau prepared statement batch.
8. Hapus cache boundary setelah transaksi berhasil.

Importer harus menghasilkan ringkasan:

```text
rows_read=91241
rows_imported=91216
unknown_codes=25
locations_without_boundary=383
invalid_coordinates=0
invalid_paths=0
```

Proses gagal tanpa mengubah data aktif jika parsing, validasi, atau transaksi database gagal.

### 4.3 Domain dan repository

Tambahkan model minimal:

```go
type Boundary struct {
    Code        string          `json:"code"`
    Name        string          `json:"name,omitempty"`
    Latitude    *float64        `json:"latitude,omitempty"`
    Longitude   *float64        `json:"longitude,omitempty"`
    LeafletPath json.RawMessage `json:"leaflet_path,omitempty"`
}
```

Repository hanya membutuhkan operasi awal:

```go
FindBoundaryByCode(ctx context.Context, code string) (*Boundary, error)
```

Tidak perlu membuat repository geospasial generik atau abstraksi PostGIS.

### 4.4 API

Endpoint:

```text
GET /api/locations/{code}/boundary
```

Response sukses:

```json
{
  "log_id": "019ab0f0-c8ec-7d25-9f62-2f23d92fcda3",
  "code": 200,
  "status": true,
  "message": "Success",
  "data": {
    "code": "32.73",
    "name": "Kota Bandung",
    "latitude": -6.9147,
    "longitude": 107.6098,
    "leaflet_path": []
  }
}
```

Jika wilayah ada tetapi boundary tidak tersedia, endpoint mengembalikan `404`. Kode harus divalidasi sebelum query database.

Response boundary diberi header:

```text
Cache-Control: public, max-age=86400, stale-while-revalidate=86400
```

Redis key:

```text
location:boundary:{code}
```

Gunakan negative cache singkat untuk boundary yang tidak tersedia agar kode yang sama tidak terus membebani PostgreSQL.

### 4.5 Batas payload

Fase pertama tidak menyediakan endpoint seluruh boundary anak. Jika fitur tersebut diperlukan kemudian:

- Hanya ambil anak langsung, bukan seluruh keturunan.
- Metadata dan geometry harus dapat diminta terpisah.
- Batasi jumlah kode per request.
- Aktifkan compression pada response polygon.

## 5. Fase 2: Detail Wilayah dan Peta

### 5.1 Detail wilayah

Tambahkan endpoint:

```text
GET /api/locations/{code}
```

Data minimum:

```json
{
  "code": "32.73",
  "full_code": "32.73",
  "name": "Kota Bandung",
  "level": "regency",
  "parent_code": "32",
  "coordinates": {
    "latitude": -6.9147,
    "longitude": 107.6098
  },
  "has_boundary": true
}
```

Polygon tetap diambil dari endpoint boundary terpisah agar detail wilayah tetap ringan.

### 5.2 Frontend

Tambahkan panel peta pada alur browse yang sudah ada:

1. Pengguna memilih wilayah.
2. Frontend meminta detail wilayah.
3. Jika `has_boundary=true`, frontend meminta endpoint boundary.
4. Leaflet menggambar polygon dan marker centroid.
5. Jika boundary tidak tersedia, peta tetap menampilkan centroid jika tersedia.

Ketentuan:

- Gunakan instance Leaflet tunggal.
- Hapus layer lama sebelum menggambar wilayah baru.
- Escape teks sebelum dimasukkan ke popup.
- Tampilkan loading dan error state.
- Jangan memuat 91 ribu polygon saat halaman dibuka.
- Jangan mengandalkan URL API hardcoded; gunakan konfigurasi frontend yang sudah tersedia.

### 5.3 Acceptance criteria

- Provinsi, kabupaten/kota, kecamatan, dan desa dapat ditampilkan di peta.
- Pergantian pilihan tidak meninggalkan polygon lama.
- Wilayah tanpa boundary tidak merusak browse flow.
- Peta dapat digunakan pada desktop dan mobile.
- Payload boundary dikompresi pada deployment produksi.

## 6. Fase 3: Data Pulau

### 6.1 Skema

```sql
CREATE TABLE IF NOT EXISTS islands (
    code varchar(11) PRIMARY KEY,
    province_code varchar(2),
    name varchar(255) NOT NULL,
    latitude double precision,
    longitude double precision,
    status varchar(10),
    area double precision,
    notes text,
    imported_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_islands_province_name
    ON islands (province_code, name);
```

### 6.2 API

```text
GET /api/islands?province_code=11&page=1&limit=50
GET /api/islands/{code}
```

Aturan:

- `page` minimal `1`.
- `limit` minimal `1`, maksimal `500`.
- Response daftar selalu memiliki metadata pagination.
- Arti status seperti `BP` dan `TBP` harus didokumentasikan sebelum rilis.
- Koordinat dan duplikasi nama harus diaudit saat import.

Audit parser terhadap 17.374 tuple sumber menemukan sejumlah catatan yang bergeser ke kolom `status`/`area` dan dapat dipulihkan secara deterministik, satu baris `91.19.40014` yang rusak dan dilewati, serta 80 kemunculan kode duplikat. Dengan aturan first-write-wins, hasil bersihnya 17.293 pulau unik. Importer melaporkan jumlah `read`, `imported`, `skipped`, dan `duplicate_codes`; data sumber tidak disalin ke repository ini.

## 7. Fase 4: Luas, Penduduk, dan Logo

Fase ini belum boleh dimulai sebelum provenance dan tanggal referensi data tersedia.

Masalah yang sudah ditemukan:

- Data penduduk hanya mencakup nasional, provinsi, dan kabupaten/kota.
- Data luas hanya mencakup provinsi dan kabupaten/kota.
- Terdapat kode luas `11.1` yang tidak cocok dengan kode resmi `11.10`.
- Terdapat nama `Papua Barat Oaya` yang perlu dikoreksi menjadi sumber resmi yang valid.
- File boundary memuat pemberitahuan lisensi MIT dari `cahya dsn`; provenance dan tanggal referensi data statistik tetap belum cukup jelas untuk dirilis.

Jika dilanjutkan, setiap record statistik harus menyimpan:

```text
source
reference_date
imported_at
```

Logo tidak membutuhkan service MinIO baru. Pilihan awal:

1. Simpan sebagai static assets jika ukuran repository masih wajar.
2. Gunakan object storage/CDN yang sudah tersedia.
3. Pin commit sumber dan simpan checksum manifest.

Jangan mengunduh logo otomatis dari branch `main` pada setiap startup.

## 8. Cache dan Pembaruan Data

TTL Redis saat ini panjang. Import data baru harus mencegah response lama bertahan selama 180 hari.

Pilih salah satu strategi sederhana:

1. Hapus seluruh key dengan prefix terkait setelah import berhasil.
2. Tambahkan versi dataset ke cache key.

Strategi awal yang disarankan: invalidasi prefix setelah transaksi import berhasil. Versi dataset baru diperlukan jika deployment memiliki banyak instance dan invalidasi tidak dapat dijamin.

Prefix baru:

```text
location:boundary:
location:islands:
```

## 9. Pengujian

Minimum check per fase:

### Importer boundary

- Bisa membaca file `.sql.gz`.
- Menolak JSON path rusak.
- Melewati kode yang tidak ada di `raw_locations`.
- Rollback ketika import gagal.
- Menghasilkan jumlah import yang sesuai audit.

### API boundary

- Kode valid dengan boundary menghasilkan `200`.
- Kode valid tanpa boundary menghasilkan `404`.
- Kode tidak valid menghasilkan `400`.
- Database error menghasilkan `500` tanpa membocorkan detail internal.
- Cache hit dan fallback tanpa Redis tetap bekerja.

### Frontend map

- Polygon dan centroid tampil untuk kode contoh tiap level.
- Layer lama dibersihkan.
- Missing boundary ditangani.
- Popup tidak menerima HTML mentah dari response API.

### Regression

```bash
go test ./...
go build ./...
```

Endpoint provinces, regencies, districts, villages, search, stats, dan health harus tetap kompatibel.

## 10. Deployment

Urutan deployment:

1. Backup PostgreSQL.
2. Jalankan migration boundary.
3. Jalankan importer boundary pada staging.
4. Cocokkan jumlah hasil import dengan laporan audit.
5. Deploy endpoint boundary.
6. Uji response dan compression.
7. Deploy frontend map.
8. Pantau latency, ukuran response, error rate, dan penggunaan Redis.
9. Ulangi proses di production.

Rollback:

- Frontend dapat dikembalikan tanpa menghapus data boundary.
- Route boundary dapat dinonaktifkan tanpa memengaruhi endpoint lama.
- Tabel baru dapat dipertahankan karena tidak mengubah tabel lokasi aktif.

## 11. Definition of Done

Fase boundary dan peta dianggap selesai ketika:

- Migration dan importer dapat dijalankan ulang secara aman.
- 91.216 boundary yang cocok berhasil diimpor.
- 25 kode usang dilaporkan dan tidak masuk database.
- API boundary tervalidasi, tercache, dan terdokumentasi.
- Frontend menampilkan polygon dan centroid berdasarkan pilihan pengguna.
- Wilayah tanpa boundary tetap dapat digunakan.
- Test dan build berhasil.
- Sumber data, keterbatasan format koordinat, dan tanggal import didokumentasikan.

## 12. Non-goals

- Migrasi backend ke NestJS.
- Migrasi frontend ke React.
- Menambah MinIO hanya untuk logo.
- Menambah PostGIS sebelum ada spatial query.
- Routing, geocoding alamat, atau navigasi.
- Mengklaim polygon sebagai GeoJSON tanpa konversi urutan koordinat.
- Menyediakan semua boundary Indonesia dalam satu response.
