# T12 — isolasi pesan broker malformed

Ticket: `CORE-IRP-012`. Eksekusi: 2026-11-03.
Semua perintah dijalankan dari `/Users/rfanazhari/Projects/go-workspace/prototype`.

## Pemetaan desain

`reportbroker.Consumer` memiliki validasi identitas transport dan settlement per pesan.
`Topology` mendeklarasikan resource Q1 sebelum intake. Aggregate, repository dan worker
aplikasi tetap memiliki lifecycle, transaksi, deduplikasi, fencing dan anggaran attempt.
Tidak ada perubahan schema, SQL produksi, HTTP, provider, atau registrasi runtime T15.

Q1 disetujui owner: durable direct `interview-report.dlx`, durable quorum
`interview-report.invalid`, binding key `malformed`. Source quorum menggunakan
`x-dead-letter-strategy=at-least-once`, `x-overflow=reject-publish`, dan
`x-delivery-limit=-1`. Penonaktifan delivery limit mempertahankan recovery database
untuk pesan valid; attempt provider tetap dibatasi oleh PostgreSQL/domain.

Malformed JSON atau identitas tidak valid memanggil `Reject(false)` sebelum worker.
Nil dari `Handle` berarti settlement berhasil, termasuk reject; bukan selalu ACK sukses.
Error reject dipropagasikan agar caller menghentikan konsumsi dan menutup channel.
Karantina tidak memiliki replay otomatis, TTL, atau binding kembali. Operator meninjau
dan memperbaiki producer sebelum replay manual. Body tidak dicatat ke log.

## RED

```sh
GOCACHE=/tmp/irp-t12-go-cache go test ./infrastructure/service/reportbroker -run 'TestConsumerHandle|TestTopologyDeclare' -count=1
```

**Failed sesuai tujuan:** consumer belum memanggil `Reject(false)` dan stub topology
belum mendeklarasikan exchange, queue serta binding. Ekspektasi Mockery tidak terpenuhi.
Stub `Topology`, interface channel, konfigurasi dan output Mockery diperlukan agar
skenario dapat dikompilasi dan dieksekusi; tidak ada implementasi topology pada RED.

SHA: `f44020b6b507e6a34303223320e0e2ce1e433d0b`.
Subject: `test: [CORE-IRP-012][rfanazhari] RED cover malformed report isolation and approved topology`.

Generasi mock awal gagal karena cache Go tidak lengkap; cache sementara memperbaikinya.
Kegagalan tooling tersebut bukan bukti RED.

## GREEN dan pemeriksaan

```sh
GOCACHE=/tmp/irp-t12-go-cache go test ./infrastructure/service/reportbroker -run 'TestConsumerHandle|TestTopologyDeclare' -count=1
GOCACHE=/tmp/irp-t12-go-cache make test
GOCACHE=/tmp/irp-t12-go-cache make test.report-broker
GOCACHE=/tmp/irp-t12-go-cache make check
```

- **Passed:** unit consumer (malformed, mismatch, reject error, database error,
  cancellation dan ACK normal), topology sukses dan kegagalan setiap tahap deklarasi.
- **Passed:** domain/application race tests melalui `make test`.
- **Passed:** sembilan skenario real RabbitMQ/PostgreSQL melalui runner terisolasi:
  tiga malformed (JSON rusak, identitas kosong, identitas tidak cocok) serta enam
  regresi T11 (confirm, redelivery, active/terminal duplicate, database error, outage).
- **Passed:** pesan malformed diterima pada exchange/key karantina dengan metadata
  `x-first-death-reason=rejected`; source kosong setelah consumer ditutup; pesan valid
  berikutnya tetap diproses. Tidak ada attempt sebelum pemrosesan pesan valid.
  Row assignment dan execution dibandingkan seluruh kolom sebelum/sesudah isolasi.
- **Passed:** `make check` mencakup all-package race tests, kontrak OpenAPI, arsitektur,
  dokumentasi deklarasi, lint (0 issues), dan security (0 reachable vulnerabilities).
  Scanner juga melaporkan masing-masing satu kerentanan pada imported package dan
  required module yang simbol rentannya tidak dipanggil.
- **Skipped:** suite database/broker pada run umum tanpa environment resource;
  bukti T12/T11 berasal dari runner Docker yang menyediakan resource terisolasi.
- **Tidak berlaku:** sqlmock untuk T12 karena tidak ada SQL baru/berubah atau SQL mock.
  Pemeriksaan SQL T12 memakai PostgreSQL nyata. Interface mock dibuat dengan `make mocks`.

Run `make check` awal terhenti karena sandbox melarang bind port localhost pada httptest.
Run berikutnya menemukan pemakaian `QueueInspect` deprecated pada tes baru; diganti
`QueueDeclarePassive`. Pemeriksaan dan runner broker pada snapshot final lulus.
Runner membersihkan hanya container, network dan volume pengujiannya.

## Assessment dan batas bukti

Dipertahankan: tanggung jawab transport tetap di infrastructure; interface executor
kecil tidak memindahkan aturan attempt ke consumer. Deklarasi topology dipisahkan dari
handler delivery, sehingga error deklarasi mencegah intake dan resource tetap dimiliki
caller. Tidak ada refactor tambahan yang bermanfaat untuk scope T12.

Kontrak pesan valid dan recovery T11 dipertahankan, dengan bukti regresi real broker.
Dokumen platform §6, feature §5/§7/Q1, D7 dan tactical strategy telah disinkronkan.
Single-node Compose tidak membuktikan failover cluster, gangguan disk, atau recovery
karantina yang unavailable; at-least-once dikonfigurasi, bukan janji exactly-once.
Source classic yang sudah ada perlu cutover operator karena tipe queue tidak dapat
diubah lewat redeclaration. Registrasi produksi dan loop recovery tetap milik T15.
