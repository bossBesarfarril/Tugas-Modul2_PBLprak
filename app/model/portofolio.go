package model

import "time"

// Portfolio adalah bentuk data portofolio mahasiswa (Clean Architecture: murni struct Go tanpa tipe driver Mongo)
type Portfolio struct {
	ID           string        `json:"id"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

type Certificate struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
}

// UpsertPortfolioRequest untuk endpoint PUT /students/:id/portfolio
type UpsertPortfolioRequest struct {
	Headline     string        `json:"headline" validate:"required,min=3,max=100"`
	Bio          string        `json:"bio" validate:"required,min=5,max=1000"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
}

// AddProjectRequest untuk endpoint POST /students/:id/portfolio/projects
type AddProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=2,max=100"`
	Technologies []string `json:"technologies" validate:"required,min=1,dive,min=1,max=50"`
	Link         string   `json:"link" validate:"required"`
	Year         int      `json:"year" validate:"required,min=2000,max=2100"`
}

// TechnologyStat untuk endpoint agregasi GET /portfolios/stats/technologies
type TechnologyStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
