package model

import "time"

type Course struct {
	ID            int       `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Credits       int       `json:"credits"`
	Quota         int       `json:"quota"`
	EnrolledCount int       `json:"enrolled_count"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

type Enrollment struct {
	ID        int64     `json:"id"`
	CourseID  int       `json:"course_id"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type SwapCourseRequest struct {
	ToCourseID int `json:"to_course_id"`
}

type UpdateQuotaRequest struct {
	Quota   int `json:"quota"`
	Version int `json:"version"`
}

type MyCourses struct {
	Courses      []Course `json:"courses"`
	TotalCredits int      `json:"total_credits"`
	MaxCredits   int      `json:"max_credits"`
}
