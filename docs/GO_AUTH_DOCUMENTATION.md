# Otorisasi UMMU Go Auth pada Profile Service

Profile Service menggunakan `github.com/AgungDev/ummu-go-auth` v0.2.0 untuk
memvalidasi JWT dan membaca claim yang sudah diverifikasi. Role dan permission
dikelola Auth Service; Profile Service tidak menyimpan atau mengelola katalog
authorization.

## Principal dan audience

Semua endpoint `/api/v1/profiles` membutuhkan JWT principal `user` dengan
audience `profile-service`. Token harus memiliki issuer yang sama dengan
`AUTH_ISSUER`, `sub` berupa UUID user, serta claim standar `iat`, `exp`, dan
`jti`. Audience default service adalah `profile-service`; sesuaikan
`AUTH_AUDIENCE` hanya jika penerbit token memang mengeluarkan audience lain.

Profile Service menolak principal client untuk operasi profil karena identitas
pemilik profil harus berasal dari user claim `sub`. Untuk resource milik
sendiri, service mengambil user ID dari `sub` terverifikasi dan mengabaikan
user ID yang mungkin dikirim oleh client.

SDK v0.2.0 membedakan user dan client principal serta menolak claim principal
yang tidak sesuai. User token tidak boleh membawa OAuth `scope` atau
`client_id`; role dan permission user berada pada claim RBAC.

## Role dan permission

Role adalah kumpulan permission Auth Service. API tidak memberi hak akses hanya
karena nama role ada di token; endpoint admin memeriksa permission pada access
context bertipe `global`.

| Endpoint | Otorisasi |
|---|---|
| `GET /api/v1/profiles/me` | User token valid; profil diambil berdasarkan `sub` |
| `PATCH /api/v1/profiles/me` | User token valid; hanya profil berdasarkan `sub` |
| `POST /api/v1/profiles/me/avatar` | User token valid; hanya avatar milik `sub` |
| `DELETE /api/v1/profiles/me/avatar` | User token valid; hanya avatar milik `sub` |
| `GET /api/v1/profiles/{user_id}` | `profile.admin:read` dalam access context `global` |
| `PATCH /api/v1/profiles/{user_id}` | `profile.admin:update` dalam access context `global` |
| `GET /healthz` | Publik |

Auth Service menyediakan permission `profile.admin:read` dan
`profile.admin:update`, dan role `super_admin` mendapat keduanya dari seeder.
Role dan assignment harus tetap ditinjau sesuai prinsip least privilege.
Endpoint admin GET hanya mengembalikan data profil publik minimum, bukan nomor
telepon atau alamat.

## Mengatur role melalui Auth Service

Gunakan token admin Auth Service yang berwenang mengelola role dan akses.
Permission dapat diperiksa melalui `GET /api/v1/permissions`, role melalui
`GET /api/v1/roles`, dan assignment user melalui
`GET /api/v1/users/{user_id}/roles`.

Untuk memberi akses admin profil, hubungkan permission yang dibutuhkan ke role
melalui `POST /api/v1/roles/{role_id}/permissions`:

```json
{
  "permission_ids": [
    "<profile_admin_read_permission_id>",
    "<profile_admin_update_permission_id>"
  ]
}
```

Kemudian tetapkan role kepada user dengan access context `global` melalui
`POST /api/v1/users/{user_id}/roles`:

```json
{
  "assignments": [
    {
      "role_id": "<profile_admin_role_id>",
      "scope_type": "global"
    }
  ]
}
```

Gunakan role terpisah jika pengguna hanya memerlukan baca atau hanya perlu
memperbarui profil. Jangan memberikan permission admin kepada role umum
pengguna hanya agar operasi `/me` dapat digunakan; endpoint `/me` tidak
memerlukan permission admin.

Setelah assignment role berubah, terbitkan ulang atau refresh token agar
`access_contexts` membawa permission terbaru. Token lama tetap berisi snapshot
claim saat token dibuat.

Contoh claim yang relevan untuk akses admin:

```json
{
  "principal_type": "user",
  "sub": "user-uuid",
  "aud": ["auth-service", "profile-service"],
  "iat": 1790000000,
  "exp": 1790000900,
  "jti": "token-uuid",
  "roles": ["profile_admin"],
  "access_contexts": [
    {
      "role": "profile_admin",
      "scope": {"type": "global", "id": null},
      "permissions": ["profile.admin:read", "profile.admin:update"]
    }
  ]
}
```

Gunakan token yang diterbitkan Auth Service; contoh claim di atas bukan token
untuk ditandatangani atau dipakai langsung.

## Konfigurasi dan perilaku akses

- `AUTH_ISSUER` harus cocok persis dengan claim `iss`.
- `AUTH_AUDIENCE` harus cocok dengan audience yang diterbitkan untuk Profile
  Service, biasanya `profile-service`.
- `JWKS_URL` menunjuk JWKS Auth Service. HTTPS dianjurkan; untuk network privat
  gunakan `JWKS_ALLOWED_HTTP_HOSTS` hanya bagi host tepercaya.
- Token hilang/tidak valid menghasilkan HTTP `401`; permission admin kurang
  atau hanya tersedia pada access context non-global menghasilkan HTTP `403`.
- `GET /healthz` publik dan tidak memvalidasi token.

`HasGlobalPermission` mengharuskan permission ada di context bertipe `global`;
permission pada context fakultas, department, `self`, atau tipe lain tidak
memberi akses ke endpoint admin. Ini membatasi endpoint admin, tetapi tidak
mengubah endpoint `/me` yang tetap terikat ke `sub`.
