# Profile Service API

Dokumentasi API, payload, authorization, dan konfigurasi object storage ada di
[`README.md`](../README.md#api).
Contoh request siap impor ke Postman ada di
[`UMMU Profile Service.postman_collection.json`](./UMMU%20Profile%20Service.postman_collection.json).

## Response umum

Sukses:

```json
{
  "success": true,
  "data": {}
}
```

Error:

```json
{
  "success": false,
  "message": "deskripsi error"
}
```

## Endpoint

| Method | Path | Sukses | Error utama |
|---|---|---|---|
| `GET` | `/healthz` | `200` | — |
| `GET` | `/api/v1/profiles/me` | `200`, membuat profil jika belum ada | `401`, `500` |
| `PATCH` | `/api/v1/profiles/me` | `200` | `400`, `401`, `500` |
| `GET` | `/api/v1/profiles/{user_id}` | `200` | `400`, `401`, `403`, `404` |
| `PATCH` | `/api/v1/profiles/{user_id}` | `200` | `400`, `401`, `403`, `500` |
| `GET` | `/api/v1/profiles/avatars/{key}` | `200`, file avatar publik pada mode filesystem lokal | `404` |
| `POST` | `/api/v1/profiles/me/avatar` | `200` | `400`, `401`, `413`, `503` |
| `DELETE` | `/api/v1/profiles/me/avatar` | `200` | `401`, `503` jika objek tersimpan tetapi storage tidak tersedia |
| `GET` | `/api/v1/profiles/me/{students,lecturers,employees}` | `200` | `401`, `404` |
| `PUT` | `/api/v1/profiles/me/{students,lecturers,employees}` | `200` | `400`, `401`, `409`, `413` |
| `DELETE` | `/api/v1/profiles/me/{students,lecturers,employees}` | `200` | `401`, `404` |
| `GET` | `/api/v1/profiles/{user_id}/{students,lecturers,employees}` | `200` | `400`, `401`, `403`, `404` |
| `PUT` | `/api/v1/profiles/{user_id}/{students,lecturers,employees}` | `200` | `400`, `401`, `403`, `404`, `409` |
| `DELETE` | `/api/v1/profiles/{user_id}/{students,lecturers,employees}` | `200` | `401`, `403`, `404` |

Semua endpoint API profil selain pembacaan file avatar lokal memerlukan bearer
JWT dengan issuer/audience yang valid. `user_id` pemilik endpoint `/me` selalu
berasal dari claim `sub`.
Endpoint pengguna lain memerlukan permission `profile.admin:read` atau
`profile.admin:update` pada access context `global`. Data alamat dan telepon
tidak disertakan pada response admin GET.

## `PATCH /api/v1/profiles/me`

Field bersifat opsional, namun sedikitnya satu field harus dikirim.

```json
{
  "full_name": "Alya Mahasiswa",
  "phone_number": "+628123456789",
  "addresses": []
}
```

`addresses` mengganti seluruh daftar alamat; array kosong berarti menghapus
daftar. `full_name` dibatasi 128 karakter, `phone_number` 32 karakter. Alamat
memerlukan `address_line`, `city`, dan `province`; `country` adalah kode dua
huruf ISO, default `ID`.

## Upload avatar

```http
POST /api/v1/profiles/me/avatar
Content-Type: multipart/form-data

avatar=<image file>
```

Menerima JPEG atau PNG sampai batas `AVATAR_MAX_BYTES`. Default lokal
menyimpan file di direktori `AVATAR_STORAGE_DIR` dan menyajikannya tanpa
autentikasi melalui `/api/v1/profiles/avatars/{key}`. Mode S3-compatible juga
tersedia. PostgreSQL hanya menyimpan object key.

## Data profesional

Endpoint `/me` menggunakan `sub` JWT dan endpoint `/{user_id}` memerlukan
permission global `profile.admin:read` untuk GET atau `profile.admin:update`
untuk PUT/DELETE. PUT melakukan upsert tunggal per tipe profesional.
`study_program_id` serta `organizational_unit_id` harus UUID, tetapi tidak
memiliki foreign key lintas-service. NIDN/NIDK masing-masing opsional, dengan
setidaknya salah satunya harus diisi. `employment_start_date` menerima
`YYYY-MM-DD`. Duplikasi NIM, NIDN, NIDK, atau employee number menghasilkan
`409 Conflict`.

Request mahasiswa berisi `nim`, `study_program_id`, `cohort_year`,
`admission_type`, dan `academic_status`. Request dosen berisi `nidn` atau
`nidk` (atau keduanya), `expertise`, dan `academic_position`. Request pegawai
berisi `employee_number`, `employment_status`, `position`,
`organizational_unit_id`, dan `employment_start_date`.
