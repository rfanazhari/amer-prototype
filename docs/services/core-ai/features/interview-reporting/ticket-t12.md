# [FEATURE] Isolasi pesan Interview Reporting yang malformed

Ticket: `CORE-IRP-012` / T12.

## Description

Consumer menolak JSON atau identitas pesan yang tidak valid dengan `Reject(false)`
sebelum worker dipanggil. Topology yang disetujui owner mengarahkan pesan ke karantina
agar pesan rusak tidak dianggap berhasil dan tidak menghambat pemrosesan job valid.

## Goal

Pisahkan pesan malformed untuk pemeriksaan operator tanpa mengubah hasil interview.

## In Scope

- Durable direct exchange `interview-report.dlx`, quorum queue `interview-report.invalid`,
  dan routing key `malformed`.
- Source quorum dengan at-least-once dead-lettering; kegagalan deklarasi mencegah intake.
- Reject error dipropagasikan; database error tetap memakai recovery T11.
- Mockery untuk interface, bukti RabbitMQ/PostgreSQL nyata, dan sinkronisasi keputusan Q1.

## Out of Scope

- Replay otomatis, UI operasional, dan topology jenis job lain.
- Writer/evidence adapter serta registrasi runtime T13–T15.
- Perubahan schema, SQL produksi, HTTP atau lifecycle interview.

## QA Impact

| No | Area | Yang dicek | Expected | Dikerjakan oleh |
|----|------|-----------|----------|-----------------|
| 1 | Consumer report | JSON rusak atau identitas invalid | Masuk karantina tanpa success ACK, requeue, atau panggilan worker | QA Engineer |
| 2 | Topology | Deklarasi exchange, queue atau binding gagal | Error dikembalikan sebelum intake dimulai | QA Engineer |
| 3 | Recovery | Database unavailable pada pesan valid | Tetap unacknowledged; bukan dipindahkan ke karantina malformed | QA Engineer |
| 4 | Interview | Bandingkan seluruh kolom assignment/execution setelah isolasi | Tidak berubah dan tidak ada attempt report baru | QA Engineer |
| 5 | Regresi T11 | Confirm, redelivery, duplicate dan outage | Seluruh skenario tetap lulus dengan topology Q1 | QA Engineer |

## Acceptance Criteria

- Pesan malformed dipindahkan melalui exchange/key/queue yang disetujui.
- Tidak ada retry otomatis dari karantina atau requeue tanpa akhir.
- Worker tidak dipanggil untuk pesan malformed; job valid berikutnya tetap diproses.
- Tidak ada perubahan row interview akibat isolasi.
- Unit test, suite T11/T12 nyata, lint, arsitektur, kontrak dan security lulus.

## Related

- [Task-plan T12](task-plan.md)
- [Technical design §5/§7/Q1](technical-design.md)
- [Keputusan D7](decision-log.md)
- [Tactical strategy](tactical-strategy.md)
- [Bukti eksekusi T12](t12-verification.md)
