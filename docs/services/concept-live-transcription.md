# AI Interview Agent — Live Transcript Technical Design

**Status:** Conceptual technical design  
**Scope:** Live transcript collection, timeline, persistence, dan integrasi interview orchestration  
**Baseline:** OpenAI GPT-Live, backend service, WebRTC di client  
**Terakhir diverifikasi:** 3 November 2026

## 1. Tujuan

Menyediakan transkrip wawancara yang dapat diamati dan disimpan **selama sesi voice berlangsung**, tanpa mewajibkan download rekaman dan transkripsi ulang sebagai jalur utama.

Hasil yang ingin dicapai:

- Memiliki timeline percakapan kandidat dan AI interviewer dengan estimasi waktu.
- Memberikan input yang cukup untuk memantau progres mandatory questions, follow-up, dan interview stage.
- Memungkinkan backend mengambil keputusan penutupan sesi berdasarkan state dan time budget.
- Menghasilkan evidence yang dapat dirujuk dalam ringkasan HR.
- Tetap mempertahankan recording provider sebagai artefak audit dan opsi pemulihan.

**Bukan tujuan:** presisi timestamp per kata, speaker diarization dari audio campuran, atau keputusan otomatis hire/reject.

## 2. Kondisi Existing dan Perubahan yang Diusulkan

### Existing

1. Backend membuat sesi melalui `POST /v1/live/sessions`.
2. Client menjalin komunikasi audio dua arah dengan GPT-Live melalui WebRTC.
3. Provider menyediakan recording yang dapat diunduh setelah sesi sesuai konfigurasi penyimpanan.
4. Rekaman diproses untuk memperoleh teks wawancara, misalnya melalui pemisahan kanal, chunking, dan transkripsi.

Pendekatan ini valid untuk arsip, tetapi hasil transcript baru tersedia setelah sesi selesai. Karena itu, hasilnya tidak ideal sebagai sumber observasi untuk orchestration selama wawancara berlangsung.

### Proposed

Pertahankan koneksi WebRTC dan tambahkan **server-side sideband WebSocket** ke sesi Live yang sama. Backend mengumpulkan transcript events ketika sesi berlangsung, menyimpannya, dan membentuk timeline yang dapat digunakan oleh orchestrator.

```text
                Go Backend / Interview Service
                     |                  ^
                     | create session   | transcript/events
                     v                  |
              OpenAI GPT-Live <---- Sideband WebSocket
                     ^                  |
                     | WebRTC           v
                     |          Transcript Collector
              Candidate Browser         |
                                        v
                               Transcript Persistence
                                        |
                                        v
                              Interview Orchestrator
                                        |
                                        v
                                HR Evidence Report

              Provider Recording (parallel audit/fallback)
```

**Boundary penting:** WebRTC tetap membawa audio secara langsung antara client dan provider. Sideband menjadi channel observasi dan kontrol pada backend; bukan proxy audio yang wajib dilalui seluruh traffic.

## 3. Kemampuan Provider yang Menjadi Dasar Desain

Untuk GPT-Live, dokumentasi OpenAI menyebut dua event transkrip:

| Event | Konten | Pemanfaatan |
|---|---|---|
| `session.input_transcript.delta` | Fragmen ucapan pengguna/kandidat | Menyusun transkrip kandidat |
| `session.output_transcript.delta` | Fragmen ucapan AI interviewer | Menyusun transkrip interviewer |

Masing-masing event mencakup `event_id`, `delta`, `start_ms`, dan `end_ms`.

- `start_ms` dan `end_ms` adalah posisi relatif terhadap **awal Live session**, dalam milidetik.
- Waktu tersebut merupakan **perkiraan interval fragmen teks**, bukan timestamp kata yang presisi.
- Event dapat datang tidak merata dan fragmen tidak selalu merupakan kalimat atau giliran bicara yang selesai.
- Tidak tersedia penanda `transcript.done` untuk masing-masing turn pada dua event ini. Aplikasi harus menetapkan aturan grouping sendiri.
- GPT-Live bersifat *full duplex*: ucapan kandidat dan AI bisa overlap.
- Event timestamp berbeda dari waktu event diterima oleh backend ataupun waktu audio benar-benar terdengar pada browser.

**Implikasi:** gunakan waktu sesi dari provider sebagai acuan penyusunan transcript timeline, sambil menyimpan arrival time terpisah untuk observability. Jangan menafsirkan setiap delta sebagai satu jawaban lengkap.

## 4. Integrasi Session dan Sideband

### 4.1 Session identity

Setelah create session, backend menyimpan hubungan antara:

- Internal interview ID;
- Provider Live session ID;
- Interview configuration/version;
- Session status, waktu mulai, target end, dan hard deadline.

Provider session ID harus diteruskan **tanpa diubah** untuk melakukan sideband attach dan lifecycle control.

### 4.2 Sideband connection

Backend melakukan attach ke sesi yang sudah ada melalui:

`wss://api.openai.com/v1/live/sessions/{session_id}/attach`

Sideband menggunakan kredensial server yang berwenang mengakses sesi tersebut. Koneksi ini tidak membuat percakapan baru dan **tidak memerlukan `session.start` ulang**. Koneksi client WebRTC tetap berjalan seperti sebelumnya.

### 4.3 Startup sequencing

Urutan konseptual yang disarankan:

1. Backend membuat Live session dan mencatat provider session ID.
2. Backend memulai sideband observer menggunakan ID yang sama.
3. Backend memastikan observer siap menerima event; readiness ini adalah status internal aplikasi.
4. Client menjalankan interaksi interview melalui WebRTC.
5. Transcript Collector menerima dan menyimpan event sampai sesi berakhir.

Attach sedini mungkin untuk mengurangi kemungkinan fragmen pembuka tidak terobservasi. Jika sideband terputus, **jangan menganggap otomatis ada replay seluruh event yang terlewat**.

### 4.4 Pembagian ownership

| Tanggung jawab | Owner utama |
|---|---|
| Capture microphone dan playback audio | Client WebRTC |
| Live conversation dan transcript emission | GPT-Live |
| Provider authentication, session mapping, sideband | Backend |
| Pengumpulan/persistensi transkrip | Backend Transcript Collector |
| Mandatory question coverage, stage, deadline | Backend Interview Orchestrator |
| Keputusan session completion dan finalisasi report | Backend |

Jika event juga diterima client melalui data channel, tetapkan satu owner untuk persistence dan business action agar tidak terjadi eksekusi ganda. Client boleh menggunakannya untuk caption/UI, sedangkan backend menjadi source of truth untuk audit dan state.

## 5. Konsep Transcript Data Model

Pisahkan **raw event stream** dari **derived conversation timeline**.

### 5.1 Raw transcript events

Data minimal untuk setiap event:

| Field konseptual | Fungsi |
|---|---|
| `interview_id` | Hubungan dengan wawancara internal |
| `provider_session_id` | Hubungan dengan sesi Live |
| `provider_event_id` | Identitas unik event untuk deduplikasi |
| `speaker` | `candidate` atau `interviewer` berdasarkan jenis event |
| `delta` | Teks fragmen asli, termasuk spasi dan pengulangan |
| `start_ms`, `end_ms` | Perkiraan interval pada timeline sesi |
| `received_at` | Waktu event diterima sistem |
| `ingest_status` | Status pemrosesan internal bila diperlukan |

Simpan `delta` apa adanya. Jangan menghapus spasi, pengulangan, atau merapikan teks pada raw layer. Deduplikasi berdasarkan identitas event dalam sesi.

### 5.2 Derived transcript segments

Derived segment mengelompokkan beberapa delta untuk konsumsi UI dan report:

- Speaker;
- Gabungan teks;
- Waktu mulai dan akhir estimatif;
- Referensi event asal;
- Status confidence/quality aplikasi jika diperlukan.

Proses grouping harus deterministik dan dapat dijalankan ulang tanpa mengubah raw events. Segment merupakan **hasil interpretasi aplikasi**, bukan bentuk event asli provider.

### 5.3 Session timeline

Satu timeline dapat berisi input dan output yang overlap. Pengurutan untuk display mempertimbangkan `start_ms`, sementara event asal dan urutan penerimaan masing-masing speaker tetap disimpan. Jangan memaksa seluruh percakapan menjadi urutan non-overlap jika kedua pihak berbicara bersamaan.

## 6. Transcript Processing dan Turn Grouping

Delta tidak sama dengan satu conversational turn. Untuk MVP, turn grouping dapat memakai kombinasi:

- Urutan fragmen per speaker;
- Kedekatan `start_ms`/`end_ms`;
- Ambang jeda berbicara yang diuji dengan data nyata;
- Pergantian aktivitas speaker;
- Penanganan overlap atau interupsi.

Jangan menjadikan jeda sebagai satu-satunya bukti turn selesai: jeda dapat terjadi karena berpikir, jaringan, ataupun perubahan ritme berbicara. Simpan batasan yang dibuat sebagai **heuristik**, sehingga evaluator bisa menerima revisi saat fragmen tambahan datang.

Untuk live captions, perbarui dua speaker secara independen. Untuk laporan HR, gunakan versi transcript segment yang sudah distabilkan setelah sesi ditutup.

## 7. Integrasi dengan Interview Orchestration

Transcript Collector **mengamati percakapan**; Orchestrator **mengambil keputusan state**. Keduanya sebaiknya tidak digabung menjadi satu tanggung jawab.

```text
Transcript Delta
      |
      v
Raw Event Persistence
      |
      v
Timeline / Turn Projection
      |
      v
Evidence & Coverage Evaluation
      |
      v
Interview FSM / Time Budget Guard
      |
      +--> Continue / follow-up
      +--> Propose authorized stage transition
      +--> Request closing
      +--> Request provider session close
```

### Mandatory question coverage

Bedakan paling tidak empat kondisi:

1. **Not asked:** pertanyaan belum diajukan;
2. **Asked:** AI sudah mengajukan pertanyaan;
3. **Answer in progress:** ada respons kandidat yang masih berkembang;
4. **Answered / incomplete:** evaluator telah memiliki evidence cukup, atau jawaban memang tidak lengkap.

Satu keyword atau satu transcript delta tidak cukup untuk menyatakan pertanyaan sudah terjawab. Evaluasi dilakukan terhadap konteks yang relevan serta evidence yang bisa ditelusuri ke transcript segment.

### Session closing

Backend menggunakan gabungan mandatory question coverage, stage state, dan deadline untuk menentukan kapan closing diizinkan atau diwajibkan. Saat closing sudah dilakukan, backend meminta provider menutup sesi dan menunggu `session.closed` untuk mengonfirmasi finalisasi. Keberhasilan penutupan koneksi jaringan saja bukan bukti session finalized.

### Report generation

HR report membaca persisted evidence, bukan sekadar ringkasan bebas model. Klaim substantif harus mengarah ke transcript segment atau rentang waktu sesi. Status pertanyaan yang tidak terjawab perlu dinyatakan eksplisit. Report tidak mengambil keputusan otomatis hire/reject.

## 8. Peran Recording dan Post-Processing

Recording provider tetap relevan untuk:

- Replay audit apabila consent dan retention policy mengizinkan;
- Verifikasi klaim dan transkrip yang meragukan;
- Pemulihan jika event stream hilang atau transkrip tidak lengkap.

Namun, download WAV → split/channel → chunk → transcribe ulang **tidak lagi menjadi dependency utama** agar Orchestrator mengetahui isi interview.

Pembedaan penting:

| Aspek | Live transcript event | Recording + alignment |
|---|---|---|
| Ketersediaan | Selama sesi berlangsung | Setelah audio tersedia dan diproses |
| Nilai utama | Observability, orchestration, early evidence | Audit dan presisi alignment tambahan |
| Timestamp | Estimasi interval fragmen pada session timeline | Bergantung metode alignment |
| Recovery saat ada event hilang | Terbatas jika tidak ada buffer/replay aplikasi | Bisa direkonstruksi dari audio yang tersimpan |

**Recording dan transcript bukan substitusi sempurna satu sama lain.** Mengaitkan teks Live dengan titik putar WAV perlu verifikasi apakah timeline file hasil recording punya offset, silence, trimming, atau perbedaan track yang harus dikompensasi.

## 9. Failure Modes dan Mitigasi

| Situasi | Dampak | Respons desain |
|---|---|---|
| Sideband attach setelah percakapan mulai | Event awal mungkin terlewat | Attach sebelum interview dimulai; tandai gap |
| Observer disconnect/restart | Kehilangan kontinuitas event | Reconnect dengan kontrol lifecycle; tandai interval tidak terobservasi; siapkan recovery recording |
| Event diterima ulang | Duplikasi transcript | Deduplikasi menggunakan provider event ID |
| Event tertunda | Timeline dan coverage sementara belum lengkap | Buffer/projection dan evaluasi ulang |
| Speech overlap/interruption | Segment salah jika asumsi turn bergantian | Pertahankan dua speaker timeline dan event asal |
| Transkripsi salah | Evidence bisa menyesatkan | Tandai ketidakpastian, gunakan verifikasi manual/audio saat perlu |
| Browser disconnect tapi provider belum close | Ketidakjelasan status akhir | Backend memegang lifecycle authority; rekonsiliasi status |
| Sideband ditutup tanpa `session.closed` | Finalisasi belum terkonfirmasi | Catat status pending/unknown, jangan laporkan sukses |
| Duplikasi pemilik action antara client/backend | Tool/action dieksekusi dua kali | Definisikan satu owner setiap business action |

## 10. Security, Privacy, dan Operational Considerations

- API key provider dan authorization rules tetap di backend.
- Validasi ownership internal interview sebelum attach, membaca transkrip, atau menutup sesi.
- Terapkan consent, akses terbatas, encryption, retention, dan deletion policy untuk transcript serta audio.
- Hindari log mentah berisi seluruh data kandidat di log aplikasi standar; gunakan penyimpanan yang sesuai akses dan retention.
- Monitor sideband connection status, event ingestion lag, interval tanpa event, duplikasi, dan penyelesaian `session.closed`.
- Pisahkan kualitas transkripsi dan kualitas penilaian evidence; teks yang tersedia belum tentu bukti yang cukup.

## 11. Rencana Integrasi Bertahap (Konseptual)

**Fase A — Observability:** observasi kedua transcript event pada sesi WebRTC yang sama; simpan raw events dan timestamp tanpa mengubah flow interview.

**Fase B — Transcript Projection:** bentuk timeline dua speaker, buat grouping, lalu bandingkan sampel dengan audio playback untuk memahami ketepatan dan offset.

**Fase C — Orchestration Integration:** hubungkan stabilized transcript dengan question coverage, authorized stage changes, dan time-budget enforcement.

**Fase D — Evidence Report:** jadikan projected transcript sebagai input utama report, dengan recording untuk verifikasi dan recovery.

Kriteria desain dianggap terpenuhi jika backend dapat mengobservasi transcript kedua speaker sepanjang sesi, menyimpan event tanpa duplikasi, merekonstruksi timeline yang dapat dirujuk, mengenali data yang hilang, dan menggunakan evidence tersebut tanpa bergantung pada transkripsi ulang setelah sesi.

## 12. Keputusan yang Masih Perlu Diverifikasi pada Implementasi Existing

- Apakah client WebRTC telah menerima dan menggunakan Live data-channel events?
- Apakah backend sudah mempunyai koneksi sideband, atau baru menggunakan REST session lifecycle?
- Apakah provider session disimpan dengan mapping yang konsisten ke internal interview?
- Apakah mode penyimpanan recording sudah aktif dan memenuhi consent/retention?
- Apakah ada kebutuhan presisi timestamp untuk playback, atau cukup interval segment untuk evidence HR?

Keputusan ini mempengaruhi titik integrasi, tetapi **tidak mengubah pilihan arsitektur utama**: WebRTC untuk media, sideband untuk observasi/control, backend untuk state dan evidence.

## 13. Referensi Resmi

Dokumen ini mengacu pada dokumentasi OpenAI berikut (diakses 3 November 2026):

1. [Managing GPT-Live sessions](https://developers.openai.com/api/docs/guides/live-conversations) — transcript deltas, interval waktu, overlap, dan turn grouping.
2. [Server-side controls](https://developers.openai.com/api/docs/guides/voice-server-controls) — sideband attach pada sesi WebRTC, ownership, dan lifecycle.
3. [GPT-Live API event reference](https://developers.openai.com/api/reference/resources/live/fork-websocket) — struktur event transkrip.
4. [Live session creation](https://developers.openai.com/api/reference/resources/live/methods/create) — session identity dan transport.

---

**Prinsip arsitektur:** Live transcript adalah sumber observasi percakapan untuk orchestration dan evidence; backend tetap authoritative untuk business state dan session lifecycle; audio recording tetap digunakan sebagai audit/fallback, bukan jalur utama transcript collection.
