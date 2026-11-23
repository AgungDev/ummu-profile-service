# UMMU Profile Service

Profile Service menyimpan identitas personal utama pengguna (nama lengkap,
nomor telepon, alamat, dan referensi avatar) serta data profesional mahasiswa,
dosen, dan pegawai. Email login, password, identitas OIDC, role, dan permission
tetap dimiliki Auth/Authorization Service.

Stack: Go, Gin, GORM, PostgreSQL, dan `github.com/AgungDev/ummu-go-auth`.

## Jalankan dengan Docker Compose

Salin `.env.example` menjadi `.env`, lalu atur `DB_PASSWORD` dan konfigurasi
Auth Service yang sesuai. Avatar secara default disimpan di `data/avatars`
pada host tanpa object storage eksternal. Auth Service harus berjalan terlebih
dahulu agar endpoint JWKS dapat dijangkau di network Docker bersama.

```powershell
Copy-Item .env.example .env
docker compose -f docker-compose.yml up -d --build
```

Port API tidak dipublikasikan ke host; Kong meneruskannya ke
`profile_service:9002`. Hanya PostgreSQL yang dipublikasikan secara lokal pada
`127.0.0.1:54322`. File avatar disimpan pada bind mount lokal
`ummu-profile-service/data/avatars`, dan dibaca melalui route publik
`/api/v1/profiles/avatars/...` di Kong. Network default harus sama dengan
network Auth Service (`auth-service_auth_net`); ubah `AUTH_DOCKER_NETWORK` jika
nama network berbeda.

Daftarkan route Kong dengan menjalankan atau memuat ulang konfigurasi gateway
di `api-getway/kong/kong.yml`. Route `/api/v1/profiles` diteruskan ke Profile
Service tanpa mengubah path. Setelah mengubah file route, muat ulang Kong:

```powershell
docker compose -f api-getway/docker-compose.yml up -d --force-recreate kong
```

Jika endpoint mengembalikan `resource not found`, pastikan route `profile-routes`
sudah dimuat pada Kong dan container `profile_service` berstatus healthy.
Jika token lama menghasilkan `invalid_token`, login ulang melalui Auth Service
untuk memperoleh token baru dengan issuer dan audience `profile-service`.

## Konfigurasi

| Variable | Fungsi |
|---|---|
| `APP_NAME`, `APP_VERSION`, `APP_ENV`, `APP_PORT` | Identitas service, mode, dan port internal |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Koneksi PostgreSQL |
| `AUTH_ISSUER` | Issuer JWT, harus sama persis dengan token Auth Service |
| `AUTH_AUDIENCE` | Audience yang divalidasi; default `profile-service` |
| `JWKS_URL` | URL JWKS Auth Service; default contoh ada di `.env.example` |
| `JWKS_ALLOWED_HTTP_HOSTS` | Allowlist hostname internal untuk URL JWKS HTTP; gunakan hanya pada network service tepercaya |
| `AVATAR_STORAGE_DIR` | Direktori penyimpanan lokal; default compose `/data/avatars` |
| `AVATAR_PUBLIC_BASE_URL` | URL publik browser untuk file avatar |
| `AVATAR_ENDPOINT`, `AVATAR_BUCKET`, `AVATAR_ACCESS_KEY`, `AVATAR_SECRET_KEY` | Opsional; atur bersama untuk memakai storage S3-compatible alih-alih filesystem |
| `AVATAR_USE_SSL` | Menggunakan TLS ke object storage |
| `AVATAR_MAX_BYTES` | Batas ukuran avatar; default 5 MiB, maksimum 20 MiB |

File avatar tidak disimpan di PostgreSQL. Jika menggunakan S3-compatible
storage, bucket harus sudah dibuat dan URL publiknya dapat dijangkau browser.
Tanpa konfigurasi filesystem lokal atau object storage, upload avatar
mengembalikan HTTP `503`; penghapusan avatar tanpa object tersimpan tetap
idempotent.

## API

Base path: `/api/v1/profiles`. Endpoint API profil membutuhkan
`Authorization: Bearer <access_token>`, kecuali pembacaan file avatar publik
pada mode filesystem lokal. Token harus memiliki issuer dan audience yang
dikonfigurasi serta `sub` UUID dari Auth Service.

Collection Postman siap impor tersedia di
[`docs/UMMU Profile Service.postman_collection.json`](docs/UMMU%20Profile%20Service.postman_collection.json).
Detail kontrak JWT, role, permission, dan pengaturan access context ada di
[GO_AUTH_DOCUMENTATION.md](./docs/GO_AUTH_DOCUMENTATION.md).
Gunakan Kong (`http://localhost:7000`) sebagai `base_url`, login melalui folder
Auth Service untuk mengisi token, lalu panggil request Profile. Request admin
memerlukan token dengan permission yang sesuai.

| Method | Endpoint | Izin |
|---|---|---|
| `GET` | `/api/v1/profiles/me` | Login; memastikan profil tersedia secara idempotent |
| `PATCH` | `/api/v1/profiles/me` | Login; hanya mengubah profil milik token `sub` |
| `GET` | `/api/v1/profiles/{user_id}` | Permission `profile.admin:read` pada context `global`; hanya data profil publik minimum |
| `PATCH` | `/api/v1/profiles/{user_id}` | Permission `profile.admin:update` pada context `global` |
| `GET` | `/api/v1/profiles/avatars/{key}` | Publik, hanya file avatar JPEG/PNG pada mode filesystem lokal |
| `POST` | `/api/v1/profiles/me/avatar` | Login; multipart field `avatar` |
| `DELETE` | `/api/v1/profiles/me/avatar` | Login |
| `GET` | `/api/v1/profiles/me/students` | Login; data mahasiswa sendiri |
| `PUT` | `/api/v1/profiles/me/students` | Login; membuat/mengganti data mahasiswa sendiri |
| `DELETE` | `/api/v1/profiles/me/students` | Login; menghapus data mahasiswa sendiri |
| `GET` | `/api/v1/profiles/me/lecturers` | Login; data dosen sendiri |
| `PUT` | `/api/v1/profiles/me/lecturers` | Login; membuat/mengganti data dosen sendiri |
| `DELETE` | `/api/v1/profiles/me/lecturers` | Login; menghapus data dosen sendiri |
| `GET` | `/api/v1/profiles/me/employees` | Login; data pegawai sendiri |
| `PUT` | `/api/v1/profiles/me/employees` | Login; membuat/mengganti data pegawai sendiri |
| `DELETE` | `/api/v1/profiles/me/employees` | Login; menghapus data pegawai sendiri |
| `GET` | `/api/v1/profiles/{user_id}/students` | Permission `profile.admin:read` |
| `PUT`, `DELETE` | `/api/v1/profiles/{user_id}/students` | Permission `profile.admin:update` |
| `GET` | `/api/v1/profiles/{user_id}/lecturers` | Permission `profile.admin:read` |
| `PUT`, `DELETE` | `/api/v1/profiles/{user_id}/lecturers` | Permission `profile.admin:update` |
| `GET` | `/api/v1/profiles/{user_id}/employees` | Permission `profile.admin:read` |
| `PUT`, `DELETE` | `/api/v1/profiles/{user_id}/employees` | Permission `profile.admin:update` |
| `GET` | `/healthz` | Publik, liveness sederhana |

### Melihat profil sendiri

```http
GET /api/v1/profiles/me
Authorization: Bearer <access_token>
```

Jika profil belum ada, service membuat baris profil kosong dengan `user_id`
dari claim `sub`, bukan dari body request. Request berulang tidak membuat
duplikasi.

### Memperbarui profil sendiri

`PATCH` menerima field yang ingin diubah saja. `addresses`, jika dikirim,
mengganti daftar alamat secara keseluruhan; `addresses: []` menghapus semua
alamat pengguna. Field yang tidak dikirim tidak berubah.

```json
{
  "full_name": "Alya Mahasiswa",
  "phone_number": "+628123456789",
  "addresses": [
    {
      "label": "home",
      "address_line": "Jl. Pendidikan No. 10",
      "city": "Ternate",
      "province": "Maluku Utara",
      "postal_code": "97711",
      "country": "ID",
      "is_primary": true
    }
  ]
}
```

Response profil sendiri berisi `user_id`, `full_name`, `phone_number`,
`avatar_url`, `addresses`, `created_at`, dan `updated_at`.

### Akses admin

Endpoint admin memeriksa permission RBAC pada context `global` di claim JWT.
Tambahkan permission `profile.admin:read` dan `profile.admin:update` pada katalog Authorization dan
berikan hanya kepada role admin yang disetujui. Token harus diterbitkan ulang
dengan permission tersebut pada access context `global`. Tidak cukup hanya
mempunyai role tanpa permission yang sesuai. Seeder Auth Service memberikan
keduanya kepada `super_admin`. Endpoint GET admin tidak mengembalikan nomor
telepon atau alamat.

### Avatar

Gunakan `multipart/form-data` dengan file field `avatar`. Hanya JPEG dan PNG
yang diterima; ukuran dibatasi konfigurasi. Storage object memakai key
acak ber-prefiks user UUID. Penghapusan avatar idempotent dan file lama dibuang
setelah perubahan database berhasil.

### Data profesional

Data personal tetap berada pada satu baris `user_profiles` untuk setiap
`user_id`. Tabel `students`, `lecturers`, dan `employees` masing-masing
mereferensikan `user_profiles.user_id`, sehingga satu profil dapat sekaligus
memiliki ketiga jenis data profesional. Setiap jenis maksimal satu baris per
profil; ID baris dan ID relasi menggunakan UUID. `study_program_id` dan
`organizational_unit_id` adalah UUID referensi lintas-service dan sengaja tidak
memiliki foreign key.

Endpoint data profesional memakai bearer JWT yang sama. Endpoint `/me`
mengambil pemilik dari claim `sub`; endpoint untuk pengguna lain memakai
permission global admin yang sama dengan endpoint profil. `PUT` bersifat
upsert per profil, sedangkan `DELETE` menghapus hanya jenis data yang diminta.
Nilai `nidn` dan `nidk` dapat dikirim salah satu atau keduanya, tetapi minimal
satu wajib terisi. Field status dan jenis admisi menerima string non-kosong
(maksimal 32 karakter), tanpa membatasi katalog nilai yang dikelola service
lain.

Contoh `PUT /api/v1/profiles/me/students`:

```json
{
  "nim": "20240001",
  "study_program_id": "d51fb41b-7f86-4a1f-928f-13dca492601f",
  "cohort_year": 2024,
  "admission_type": "regular",
  "academic_status": "active"
}
```

`PUT /api/v1/profiles/me/lecturers` menerima `nidn`, `nidk`, `expertise`, dan
`academic_position`:

```json
{
  "nidn": "0123456789",
  "nidk": null,
  "expertise": "Computer Science",
  "academic_position": "assistant"
}
```

`PUT /api/v1/profiles/me/employees` menerima `employee_number`,
`employment_status`, `position`, `organizational_unit_id`, dan
`employment_start_date` dengan format `YYYY-MM-DD`:

```json
{
  "employee_number": "EMP-001",
  "employment_status": "active",
  "position": "staff",
  "organizational_unit_id": "bd2f91f3-987d-4c64-bc92-8f0b9855da0b",
  "employment_start_date": "2024-01-15"
}
```

NIM, NIDN, NIDK, dan nomor pegawai unik di seluruh tabel sejenis; duplikasi
menghasilkan HTTP `409`.

## Kepemilikan dan provisioning

| Data | Pemilik |
|---|---|
| User ID, email login, password, OIDC | Auth Service |
| Nama, nomor telepon, alamat, avatar key, dan profil profesional | Profile Service |
| Detail akademik selain atribut identitas profesional | Student/Lecturer Service |
| Data organisasi selain atribut identitas pegawai | HR Service |
| Role dan permission | Authorization Service |

Profile Service menggunakan `sub` sebagai `user_id` relasi, tidak menyimpan
email/password, dan memiliki satu profil personal utama untuk setiap pengguna.
`GET /me` adalah provisioning awal idempotent.
Auth Service tidak lagi menyimpan atau mengembalikan `full_name`. Nama dari
database Auth lama dipindahkan menggunakan importer satu kali yang dijelaskan
di [`auth-service/docs/PROFILE_SERVICE_MIGRATION.md`](../auth-service/docs/PROFILE_SERVICE_MIGRATION.md).
Untuk akun baru, profil dibuat otomatis oleh `GET /me` dan nama lengkap dapat
dilengkapi melalui `PATCH /me`. Sinkronisasi event-driven dengan transactional
outbox tetap dapat ditambahkan kemudian.

## Database dan migration

Migration SQL tertanam ke binary dan dijalankan saat startup secara berurutan.
Tabel `schema_migrations` melacak migration yang berhasil. Migration awal
membuat `user_profiles` dan `user_addresses`; migration berikutnya membuat
tabel profesional dengan FK hanya ke `user_profiles` dalam database ini.

## Pengembangan lokal

```powershell
go test ./...
go run ./cmd/server
```

Jalankan service dengan environment yang sama seperti di `.env.example`.
Repository ini memakai UMMU Go Auth v0.2.0, termasuk allowlist hostname JWKS
internal. Checkout SDK lokal `../ummu-go-outh` tidak diperlukan untuk build.
