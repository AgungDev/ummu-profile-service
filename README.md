# UMMU Profile Service

Profile Service menyimpan atribut profil umum pengguna: nama lengkap, nomor
telepon, alamat, dan referensi avatar. Service ini **tidak** menyimpan email
login, password, identitas OIDC, NIM/status mahasiswa, NIDN/data dosen, data
kepegawaian, role, atau permission.

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

## Kepemilikan dan provisioning

| Data | Pemilik |
|---|---|
| User ID, email login, password, OIDC | Auth Service |
| Nama, nomor telepon, alamat, avatar key | Profile Service |
| NIM dan status mahasiswa | Student Service |
| NIDN dan data dosen | Lecturer Service |
| Data kepegawaian | HR Service |
| Role dan permission | Authorization Service |

Profile Service hanya memakai `sub` untuk referensi identitas dan tidak
menduplikasi profil akademik. `GET /me` adalah provisioning awal idempotent.
Auth Service tidak lagi menyimpan atau mengembalikan `full_name`. Nama dari
database Auth lama dipindahkan menggunakan importer satu kali yang dijelaskan
di [`auth-service/docs/PROFILE_SERVICE_MIGRATION.md`](../auth-service/docs/PROFILE_SERVICE_MIGRATION.md).
Untuk akun baru, profil dibuat otomatis oleh `GET /me` dan nama lengkap dapat
dilengkapi melalui `PATCH /me`. Sinkronisasi event-driven dengan transactional
outbox tetap dapat ditambahkan kemudian.

## Database dan migration

Migration SQL tertanam ke binary dan dijalankan saat startup secara berurutan.
Tabel `schema_migrations` melacak migration yang berhasil. Schema awal membuat
`user_profiles` dan `user_addresses`; keduanya tidak memiliki foreign key ke
database Auth Service karena service memiliki database sendiri.

## Pengembangan lokal

```powershell
go test ./...
go run ./cmd/server
```

Jalankan service dengan environment yang sama seperti di `.env.example`.
Repository ini memakai UMMU Go Auth v0.2.0, termasuk allowlist hostname JWKS
internal. Checkout SDK lokal `../ummu-go-outh` tidak diperlukan untuk build.
