package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students/app/model"
)

var (
	ErrPortfolioNotFound = errors.New("portofolio tidak ditemukan")
	ErrProjectLimit      = errors.New("jumlah proyek sudah mencapai batas maksimal 20")
	ErrInvalidProjectID  = errors.New("format id proyek tidak valid")
)

const MaxProjectsPerPortfolio = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindByStudentID(ctx context.Context, studentID int) (model.Portfolio, error)
	UpsertByStudentID(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.Portfolio, error)
	DeleteProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error)
	GetTopTechnologies(ctx context.Context) ([]model.TechnologyStat, error)
}

type portfolioMongoRepository struct {
	coll *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{
		coll: db.Collection("student_portfolios"),
	}
}

// Struct internal BSON dokumen di MongoDB (memisahkan layer repo dari model)
type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	Name   string `bson:"name"`
	Issuer string `bson:"issuer"`
	Date   string `bson:"date"`
}

func now() time.Time {
	return time.Now().Truncate(time.Millisecond)
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		projects = append(projects, model.Project{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: p.Technologies,
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{
			Name:   c.Name,
			Issuer: c.Issuer,
			Date:   c.Date,
		})
	}

	return model.Portfolio{
		ID:           d.ID.Hex(),
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certs,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

// EnsureIndexes membuat Unique Index pada student_id sesuai syarat C.1
func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_student_id"),
		},
	}
	_, err := r.coll.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) FindByStudentID(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, err
	}
	return doc.toModel(), nil
}

func (r *portfolioMongoRepository) UpsertByStudentID(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.Portfolio, error) {
	t := now()

	projectsDoc := make([]projectDocument, 0, len(req.Projects))
	for _, p := range req.Projects {
		pID := bson.NewObjectID()
		if p.ID != "" {
			if parsedID, err := bson.ObjectIDFromHex(p.ID); err == nil {
				pID = parsedID
			}
		}
		projectsDoc = append(projectsDoc, projectDocument{
			ID:           pID,
			Title:        p.Title,
			Technologies: p.Technologies,
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certsDoc := make([]certificateDocument, 0, len(req.Certificates))
	for _, c := range req.Certificates {
		certsDoc = append(certsDoc, certificateDocument{
			Name:   c.Name,
			Issuer: c.Issuer,
			Date:   c.Date,
		})
	}

	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$set": bson.M{
			"headline":     req.Headline,
			"bio":          req.Bio,
			"projects":     projectsDoc,
			"certificates": certsDoc,
			"updated_at":   t,
		},
		"$setOnInsert": bson.M{
			"student_id": studentID,
			"created_at": t,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		return model.Portfolio{}, err
	}
	return doc.toModel(), nil
}

// AddProject menambah satu proyek dengan $push dan pembatasan maksimal 20 secara atomik
func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.Portfolio, error) {
	t := now()
	newProj := projectDocument{
		ID:           bson.NewObjectID(),
		Title:        req.Title,
		Technologies: req.Technologies,
		Link:         req.Link,
		Year:         req.Year,
	}

	// Filter atomik: pastikan proyek ke-20 (indeks 19) BELUM ada
	filter := bson.M{
		"student_id":  studentID,
		"projects.19": bson.M{"$exists": false},
	}
	update := bson.M{
		"$push": bson.M{"projects": newProj},
		"$set":  bson.M{"updated_at": t},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var doc portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Cek apakah mahasiswa punya portofolio tapi proyeknya sudah mencapai limit 20
			count, _ := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
			if count > 0 {
				return model.Portfolio{}, ErrProjectLimit
			}
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, err
	}
	return doc.toModel(), nil
}

// DeleteProject menghapus proyek dari array menggunakan operator $pull
func (r *portfolioMongoRepository) DeleteProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error) {
	objID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return model.Portfolio{}, ErrInvalidProjectID
	}

	t := now()
	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$pull": bson.M{"projects": bson.M{"_id": objID}},
		"$set":  bson.M{"updated_at": t},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var doc portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, err
	}
	return doc.toModel(), nil
}

// GetTopTechnologies menggunakan aggregation pipeline 2x $unwind
func (r *portfolioMongoRepository) GetTopTechnologies(ctx context.Context) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$unwind", Value: "$projects"}},
		bson.D{{Key: "$unwind", Value: "$projects.technologies"}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$projects.technologies"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		bson.D{{Key: "$limit", Value: 10}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "technology", Value: "$_id"},
			{Key: "count", Value: 1},
		}}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("agregasi teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var results []model.TechnologyStat
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("membaca hasil agregasi: %w", err)
	}

	if results == nil {
		results = []model.TechnologyStat{}
	}

	return results, nil
}
