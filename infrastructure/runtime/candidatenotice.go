package runtime

// CandidateNotice is the notice version and statement a candidate is asked to accept before an interview starts.
type CandidateNotice struct {
	// Version identifies the statement and is stored with each acceptance.
	Version string
	// Statement is the full approved text shown to the candidate.
	Statement string
}

// noticeVersion is the approved notice version, recorded in technical design section 6.
const noticeVersion = "notice-261102"

// noticeStatement is the approved notice text, recorded verbatim in technical design section 6.
const noticeStatement = `Pemberitahuan dan Persetujuan Interview

Interview ini dilakukan secara lisan dengan agen AI dan berlangsung maksimal 15 menit. Dengan menekan "Saya setuju", Anda menyatakan memahami dan menyetujui hal berikut:

1. Suara Anda direkam dan diubah menjadi transkrip selama interview berlangsung.
2. Rekaman, transkrip, dan hasil penilaian dipakai hanya untuk proses seleksi pada lowongan yang Anda lamar.
3. Hasil interview hanya dapat dilihat oleh tim rekrutmen yang berwenang. Kandidat lain tidak dapat melihatnya.
4. Anda dapat menghentikan interview kapan saja dengan tombol berhenti. Interview yang dihentikan dicatat sebagai dibatalkan.
5. Persetujuan ini tercatat bersama versi pemberitahuan dan waktu Anda menyetujuinya.

Jika Anda tidak setuju, tutup halaman ini dan hubungi tim rekrutmen.`

// CurrentCandidateNotice returns the notice catalog value composed into the candidate interview.
func CurrentCandidateNotice() CandidateNotice {
	return CandidateNotice{Version: noticeVersion, Statement: noticeStatement}
}
