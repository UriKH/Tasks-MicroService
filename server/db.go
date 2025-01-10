package main

import (
	"context"
	"fmt"
	"time"

	ppb "github.com/TekClinic/Patients-MicroService/tasks_protobuf"
	sf "github.com/sa-/slicefunk"
	"github.com/uptrace/bun"
)

const birthDateFormat = "2006-01-02"

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

// Patient defines a schema of patients.
//type Patient struct {
//	ID                int32               `bun:",pk,autoincrement" `
//	Active            bool                ``
//	Name              string              `validate:"required,min=1,max=100"`
//	PersonalID        PersonalID          `bun:"embed:personal_id_" validate:"required"`
//	Gender            ppb.Patient_Gender  ``
//	PhoneNumber       string              `validate:"omitempty,e164"`
//	Languages         []string            `bun:",array" validate:"max=10,dive,max=100"`
//	BirthDate         time.Time           `validate:"required"`
//	ReferredBy        string              `validate:"max=100"`
//	EmergencyContacts []*EmergencyContact `bun:"rel:has-many,join:id=patient_id" validate:"max=10,dive"`
//	SpecialNote       string              `validate:"max=500"`
//	CreatedAt         time.Time           `bun:",nullzero,notnull,default:current_timestamp"`
//	DeletedAt         time.Time           `bun:",soft_delete,nullzero"`
//}

// Task defines a schema of patients.
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

// toGRPC returns a GRPC version of PersonalID.
func (personalId PersonalID) toGRPC() *ppb.Patient_PersonalID {
	return &ppb.Patient_PersonalID{
		Id:   personalId.ID,
		Type: personalId.Type,
	}
}

// personalIDFromGRPC returns a PersonalID from a GRPC version.
func personalIDFromGRPC(personalID *ppb.Patient_PersonalID) PersonalID {
	return PersonalID{
		ID:   personalID.GetId(),
		Type: personalID.GetType(),
	}
}

// toGRPC returns a GRPC version of EmergencyContact.
func (contact EmergencyContact) toGRPC() *ppb.Patient_EmergencyContact {
	return &ppb.Patient_EmergencyContact{
		Name:      contact.Name,
		Closeness: contact.Closeness,
		Phone:     contact.Phone,
	}
}

// emergencyContactFromGRPC returns an EmergencyContact from a GRPC version.
func emergencyContactFromGRPC(contact *ppb.Patient_EmergencyContact) *EmergencyContact {
	return &EmergencyContact{
		Name:      contact.GetName(),
		Closeness: contact.GetCloseness(),
		Phone:     contact.GetPhone(),
	}
}

// toGRPC returns a GRPC version of Patient.
func (task Task) toGRPC() *ppb.Task {
	return &ppb.Task{
		Id:                task.ID,
		Active:            task.Active,
		Name:              task.Name,
		PersonalId:        task.PersonalID.toGRPC(),
		Gender:            task.Gender,
		PhoneNumber:       patient.PhoneNumber,
		Languages:         patient.Languages,
		BirthDate:         patient.BirthDate.Format(birthDateFormat),
		Age:               int32(time.Now().Year() - patient.BirthDate.Year()),
		ReferredBy:        patient.ReferredBy,
		EmergencyContacts: emergencyContacts,
		SpecialNote:       patient.SpecialNote,
	}
}

// patientFromGRPC returns a Patient from a GRPC version.
func patientFromGRPC(patient *ppb.Patient) (Patient, error) {
	emergencyContacts := sf.Map(patient.GetEmergencyContacts(), emergencyContactFromGRPC)
	birthDate, err := time.Parse(birthDateFormat, patient.GetBirthDate())
	if err != nil {
		return Patient{}, fmt.Errorf("failed to parse birth date: %w", err)
	}
	return Patient{
		ID:                patient.GetId(),
		Active:            patient.GetActive(),
		Name:              patient.GetName(),
		PersonalID:        personalIDFromGRPC(patient.GetPersonalId()),
		Gender:            patient.GetGender(),
		PhoneNumber:       patient.GetPhoneNumber(),
		Languages:         patient.GetLanguages(),
		BirthDate:         birthDate,
		ReferredBy:        patient.GetReferredBy(),
		EmergencyContacts: emergencyContacts,
		SpecialNote:       patient.GetSpecialNote(),
	}, nil
}

// createSchemaIfNotExists creates all required schemas for patient microservice.
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
