package main

import (
	"context"
	"fmt"
	"time"

	ppb "github.com/TekClinic/Tasks-MicroService/tasks_protobuf"
	"github.com/uptrace/bun"
)

const yyyy_mm_dd = "2006-01-02"

// PersonalID defines a schema of personal ids.
type PersonalID struct {
	ID   string
	Type string
}

// EmergencyContact defines a schema of emergency contacts.
type EmergencyContact struct {
	ID        int32  `bun:",pk,autoincrement"`
	Name      string `validate:"required,min=1,max=100"`
	Closeness string `validate:"required,min=1,max=100"`
	Phone     string `validate:"required,e164"`
	PatientID int32
}

// Task defines a schema of task.
type Task struct {
	ID          int32     `bun:",pk,autoincrement"`
	Active      bool      ``
	Title       string    `validate:"required,min=1,max=100"`
	Description string    `validate:"max=500"`
	Expertise   string    `validate:"required,min=1,max=100"`
	PatientID   int32     `validate:"min=1,max=100"` // TODO: change to ID in the patients db
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	DeletedAt   time.Time `bun:",soft_delete,nullzero"`
}

// toGRPC returns a GRPC version of Task.
func (task Task) toGRPC() *ppb.Task {
	return &ppb.Task{
		Id:          task.ID,
		Active:      task.Active,
		Title:       task.Title,
		Description: task.Description,
		Expertise:   task.Expertise,
		PatientId:   task.PatientID,
		CreatedAt:   task.CreatedAt.Format(yyyy_mm_dd),
		DeletedAt:   task.DeletedAt.Format(yyyy_mm_dd),
	}
}

// taskFromGRPC returns a Task from a GRPC version.
func taskFromGRPC(task *ppb.Task) (Task, error) {
	created, err1 := time.Parse(yyyy_mm_dd, task.GetCreatedAt())
	deleted, err2 := time.Parse(yyyy_mm_dd, task.GetDeletedAt())
	if err1 != nil {
		return Task{}, fmt.Errorf("failed to parse birth date: %w", err1)
	}
	if err2 != nil {
		return Task{}, fmt.Errorf("failed to parse birth date: %w", err2)
	}
	return Task{
		ID:          task.GetId(),
		Active:      task.GetActive(),
		Title:       task.GetTitle(),
		Description: task.GetDescription(),
		Expertise:   task.GetExpertise(),
		PatientID:   task.GetPatientId(),
		CreatedAt:   created,
		DeletedAt:   deleted,
	}, nil
}

// createSchemaIfNotExists creates all required schemas for task microservice.
func createSchemaIfNotExists(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*Task)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return err
		}
	}

	// Migration code. Add created_at and deleted_at columns to the patient table for soft delete.
	if _, err := db.NewRaw(
		"ALTER TABLE tasks " +
			"ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now(), " +
			"ADD COLUMN IF NOT EXISTS deleted_at timestamptz;").Exec(ctx); err != nil {
		return err
	}

	// TODO: fill the column names of the table
	// Postgres specific code. Add a text_searchable column for full-text search.
	if _, err := db.NewRaw(
		"ALTER TABLE tasks " +
			"ADD COLUMN IF NOT EXISTS text_searchable tsvector " +
			"GENERATED ALWAYS AS " +
			"(" +
			"setweight(to_tsvector('simple', coalesce(personal_id_id, '')), 'A') || " +
			"setweight(to_tsvector('simple', coalesce(, '')), 'A')   || " +
			"setweight(to_tsvector('simple', coalesce(name, '')), 'B')           || " +
			"setweight(to_tsvector('simple', coalesce(special_note, '')), 'C')   || " +
			"setweight(to_tsvector('simple', coalesce(, '')), 'D')" +
			") STORED").Exec(ctx); err != nil {
		return err
	}

	return nil
}
