package service

import (
	"errors"
	"fmt"

	"api-students/app/model"
)

var errUnknownEvent = errors.New("jenis event tidak dikenal")

// BuildMessage menyusun pesan notifikasi berdasarkan jenis event.
func BuildMessage(eventType string, event model.EnrollmentEvent, user model.User) (model.Message, error) {
	course := event.CourseCode + " " + event.CourseName
	var subject, body string

	switch eventType {
	case model.EventEnrollmentCreated:
		subject = "KRS: berhasil mendaftar " + course
		body = fmt.Sprintf("Halo %s, Anda berhasil terdaftar pada kelas %s.", user.Username, course)

	case model.EventEnrollmentDropped:
		subject = "KRS: keluar dari " + course
		body = fmt.Sprintf("Halo %s, Anda telah membatalkan pendaftaran kelas %s.", user.Username, course)

	case model.EventEnrollmentSwapped:
		from := event.FromCourseCode + " " + event.FromCourseName
		subject = "KRS: pindah dari " + from + " ke " + course
		body = fmt.Sprintf("Halo %s, Anda telah pindah dari kelas %s ke kelas %s.", user.Username, from, course)

	default:
		return model.Message{}, fmt.Errorf("%w: %q", errUnknownEvent, eventType)
	}

	return model.Message{To: user.Email, Subject: subject, Body: body}, nil
}

// BuildQuotaMessage menyusun pesan notifikasi perubahan kuota (Tugas Mandiri D.2).
func BuildQuotaMessage(qc model.QuotaChangedEvent, user model.User) model.Message {
	subject := fmt.Sprintf("KRS: kuota %s %s berubah menjadi %d", qc.CourseCode, qc.CourseName, qc.NewQuota)
	body := fmt.Sprintf("Halo %s, kuota mata kuliah %s (%s) telah diubah menjadi %d peserta.", user.Username, qc.CourseName, qc.CourseCode, qc.NewQuota)
	return model.Message{To: user.Email, Subject: subject, Body: body}
}
