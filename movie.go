package tmdbankigenerator

import (
	"slices"
	"time"
)

type Language string

const (
	LanguageKorean  = "ko"
	LanguageEnglish = "en"
	LanguageDanish  = "da"
)

var ValidLanguages = []Language{LanguageDanish, LanguageEnglish}

type Movie struct {
	ID          int       `db:"id"`
	Title       string    `db:"title"`
	Popularity  float32   `db:"popularity"`
	ReleaseDate time.Time `db:"release_date"`
	Adult       bool      `db:"adult"`
	Runtime     int       `db:"runtime"`
	Language    Language  `db:"language"`
	Genres      string    `db:"genres"`

	Images  []MovieImage
	Persons []MoviePerson
}

type MoviePerson struct {
	JobType JobType
	InList  bool
	Person
}

type MovieImage struct {
	MovieID int    `db:"movie_id"`
	Path    string `db:"path"`
}

type JobType string

const (
	JobTypeCast            JobType = "Cast"
	JobTypeDirector                = "Director"
	JobTypeComposer                = "Composer"
	JobTypeComposer2               = "Original Music Composer"
	JobTypeComposer3               = "Music"
	JobTypeComposer4               = "Songs"
	JobTypeWriter                  = "Writer"
	JobTypeCinematographer         = "Cinematography"
)

var allJobTypes = []JobType{JobTypeCast, JobTypeComposer, JobTypeComposer2, JobTypeComposer3, JobTypeComposer4, JobTypeDirector, JobTypeWriter, JobTypeCinematographer}

type Credit struct {
	PersonID int     `db:"person_id"`
	MovieID  int     `db:"movie_id"`
	JobType  JobType `db:"job_type"`

	Movie *Movie
}

type Person struct {
	ID                 int      `json:"id" db:"id"`
	Birthday           string   `json:"birthday" db:"birthday"`
	KnownForDepartment string   `json:"known_for_department" db:"known_for_department"`
	Name               string   `json:"name" db:"name"`
	AlsoKnownAs        []string `json:"also_known_as" db:"-"` // We'll handle as JSON in the DB
	Gender             int      `json:"gender" db:"gender"`
	Popularity         float32  `json:"popularity" db:"popularity"`
	PlaceOfBirth       string   `json:"place_of_birth" db:"place_of_birth"`
	ProfilePath        string   `json:"profile_path" db:"profile_path"`
	Adult              bool     `json:"adult" db:"adult"`
	IMDbID             string   `json:"imdb_id" db:"imdb_id"`

	Images []PersonImage
}

type PersonImage struct {
	PersonID int    `db:"person_id"`
	Path     string `db:"path"`
}

func (t JobType) IsValid() bool {
	if slices.Contains(allJobTypes, t) {
		return true
	} else {
		return false
	}
}
