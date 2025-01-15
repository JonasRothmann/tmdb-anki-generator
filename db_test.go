package tmdbankigenerator

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestDatabase(t *testing.T) {
	conn, err := sqlx.Connect("sqlite3", "test.db")
	if err != nil {
		t.Fatalf("failed to connect to in-memory DB: %v", err)
	}

	_, _ = conn.Exec("PRAGMA foreign_keys = ON;")

	conn.MustExec(schema)

	db := &Database{conn: conn}

	t.Run("UpsertPerson", func(t *testing.T) {
		person := Person{
			ID:                 123,
			Birthday:           "1985-09-26",
			KnownForDepartment: "Acting",
			Name:               "John Test",
			AlsoKnownAs:        []string{"Johnny T.", "JTest"},
			Gender:             2,
			Popularity:         7.5,
			PlaceOfBirth:       "Test City",
			ProfilePath:        "/profile_test.jpg",
			Adult:              false,
			IMDbID:             "nm1234567",
		}

		if err := db.UpsertPerson(person, nil); err != nil {
			t.Errorf("UpsertPerson failed: %v", err)
		}

		var count int
		err := conn.Get(&count, "SELECT COUNT(*) FROM persons WHERE id = ?", person.ID)
		if err != nil {
			t.Fatalf("failed to query persons table: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 person row, got %d", count)
		}

		person.Name = "John Test Updated"
		if err := db.UpsertPerson(person, nil); err != nil {
			t.Errorf("UpsertPerson (update) failed: %v", err)
		}

		var updatedName string
		err = conn.Get(&updatedName, "SELECT name FROM persons WHERE id = ?", person.ID)
		if err != nil {
			t.Fatalf("failed to select updated name: %v", err)
		}
		if updatedName != "John Test Updated" {
			t.Errorf("expected updated name to be %q, got %q", person.Name, updatedName)
		}
	})

	t.Run("UpsertMovie", func(t *testing.T) {
		movie := Movie{
			ID:    456,
			Title: "Movie Title",
		}
		if err := db.UpsertMovie(movie, nil); err != nil {
			t.Errorf("UpsertMovie failed: %v", err)
		}

		var count int
		err := conn.Get(&count, "SELECT COUNT(*) FROM movies WHERE id = ?", movie.ID)
		if err != nil {
			t.Fatalf("failed to query movies table: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 movie row, got %d", count)
		}

		movie.Title = "Updated Movie Title"
		if err := db.UpsertMovie(movie, nil); err != nil {
			t.Errorf("UpsertMovie (update) failed: %v", err)
		}

		var updatedTitle string
		err = conn.Get(&updatedTitle, "SELECT title FROM movies WHERE id = ?", movie.ID)
		if err != nil {
			t.Fatalf("failed to select updated title: %v", err)
		}
		if updatedTitle != "Updated Movie Title" {
			t.Errorf("expected updated title to be %q, got %q", movie.Title, updatedTitle)
		}
	})

	t.Run("UpsertCredit", func(t *testing.T) {
		person := Person{ID: 999, Name: "Credit Tester"}
		movie := Movie{ID: 888, Title: "Credit Movie"}
		if err := db.UpsertPerson(person, nil); err != nil {
			t.Errorf("UpsertPerson (credit test) failed: %v", err)
		}
		if err := db.UpsertMovie(movie, nil); err != nil {
			t.Errorf("UpsertMovie (credit test) failed: %v", err)
		}

		credit := Credit{
			PersonID: person.ID,
			MovieID:  movie.ID,
			JobType:  JobTypeCast, // e.g. "Cast"
		}

		if err := db.UpsertCredit(credit); err != nil {
			t.Errorf("UpsertCredit failed: %v", err)
		}

		var count int
		err := conn.Get(&count, "SELECT COUNT(*) FROM credits WHERE person_id = ? AND movie_id = ? AND job_type = ?", person.ID, movie.ID, string(JobTypeCast))
		if err != nil {
			t.Fatalf("failed to query credits table: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 credit row, got %d", count)
		}

		if err := db.UpsertCredit(credit); err != nil {
			t.Errorf("UpsertCredit (again) failed: %v", err)
		}
		err = conn.Get(&count, "SELECT COUNT(*) FROM credits WHERE person_id = ? AND movie_id = ? AND job_type = ?", person.ID, movie.ID, string(JobTypeCast))
		if err != nil {
			t.Fatalf("failed to query credits table after re-upsert: %v", err)
		}
		if count != 1 {
			t.Errorf("expected count to remain 1, got %d", count)
		}
	})
}
