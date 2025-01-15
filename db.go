package tmdbankigenerator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var schema = `
CREATE TABLE IF NOT EXISTS persons (
    id                  INTEGER PRIMARY KEY,
    birthday            TEXT,
    known_for_department TEXT,
    name                TEXT,
    also_known_as       TEXT,    -- JSON array for multiple names
    gender              INTEGER,
    popularity          REAL,
    place_of_birth      TEXT,
    profile_path        TEXT,
    adult               BOOLEAN,
    imdb_id             TEXT
);

CREATE TABLE IF NOT EXISTS movies (
    id       		INTEGER PRIMARY KEY,
    title    		TEXT,
    language 		TEXT,
    popularity 		FLOAT,
    runtime  		FLOAT,
    genres			TEXT,
    release_date    TEXT,
    adult 			BOOLEAN
);

CREATE TABLE IF NOT EXISTS credits (
    person_id   INTEGER NOT NULL,
    movie_id    INTEGER NOT NULL,
    job_type    TEXT NOT NULL,

    FOREIGN KEY(person_id) REFERENCES persons(id),
    FOREIGN KEY(movie_id) REFERENCES movies(id),
    PRIMARY KEY(person_id, movie_id, job_type)
);

CREATE TABLE IF NOT EXISTS person_images (
    person_id   INTEGER NOT NULL,
    path        TEXT NOT NULL,

    FOREIGN KEY(person_id) REFERENCES persons(id),
    PRIMARY KEY(person_id, path)
);

CREATE TABLE IF NOT EXISTS movie_images (
    movie_id   INTEGER NOT NULL,
    path       TEXT NOT NULL,

    FOREIGN KEY(movie_id) REFERENCES movies(id),
    PRIMARY KEY(movie_id, path)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_credits ON credits(person_id, movie_id, job_type);
`

type Database struct {
	conn *sqlx.DB
}

func NewDatabase() (*Database, error) {
	conn, err := sqlx.Connect("sqlite3", "data.db?_cache=shared&_mode=rwc")
	if err != nil {
		return nil, err
	}

	conn.MustExec(schema)

	return &Database{
		conn: conn,
	}, nil
}

func (d *Database) Close() {
	d.conn.Close()
}

func (d *Database) GetMoviesByPersonIDs(personIds []int, extraIds []int, popularity int) ([]Movie, error) {
	query := `
	SELECT DISTINCT c.job_type, m.id, m.title, m.language, m.popularity, m.runtime, m.release_date, m.adult,
                    mi.path AS movie_image_path,
                    p.id AS person_id, p.name AS person_name, pi.path AS person_image_path,
    CASE
        WHEN c.person_id IN (?, ?) THEN 1 ELSE 0
    END AS is_in_list
FROM
    movies m
    INNER JOIN credits c ON m.id = c.movie_id
    LEFT JOIN movie_images mi ON m.id = mi.movie_id
    INNER JOIN persons p ON p.id = c.person_id
    LEFT JOIN person_images pi ON p.id = pi.person_id
    WHERE p.id IN (?) OR p.popularity > ?
ORDER BY
    m.popularity DESC;
    `

	// Create a query with the person IDs expanded
	query, args, err := sqlx.In(query, personIds, extraIds, personIds, popularity)
	if err != nil {
		return nil, fmt.Errorf("failed to construct query: %w", err)
	}

	// Rebind the query to match the SQLite driver
	query = d.conn.Rebind(query)

	// Execute the query
	rows, err := d.conn.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Prepare the result slice
	var movies = map[int]*Movie{}
	var movieIdsOrder = []int{}

	for rows.Next() {
		var movie Movie
		var movieImagePath sql.NullString
		var person Person
		var personImagePath sql.NullString
		var personInList bool
		var jobType JobType
		var releaseDate string

		// Scan the row into the structs
		err := rows.Scan(
			&jobType,
			&movie.ID, &movie.Title, &movie.Language, &movie.Popularity, &movie.Runtime, &releaseDate, &movie.Adult,
			&movieImagePath, &person.ID, &person.Name, &personImagePath,
			&personInList,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if _, exists := movies[movie.ID]; !exists {
			movieIdsOrder = append(movieIdsOrder, movie.ID)
			movies[movie.ID] = &movie
		}

		layout := "2006-01-02 15:04:05-07:00"
		movie.ReleaseDate, err = time.Parse(layout, releaseDate)
		if err != nil {
			return nil, err
		}

		// Set the image paths if they are not null
		if movieImagePath.Valid {
			movie.Images = append(movie.Images, MovieImage{Path: movieImagePath.String})
		}
		if personImagePath.Valid {
			person.Images = append(person.Images, PersonImage{Path: personImagePath.String})
		}

		existingMovie := movies[movie.ID]
		existingMovie.Persons = append(existingMovie.Persons, MoviePerson{
			Person:  person,
			JobType: jobType,
			InList:  personInList,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	result := make([]Movie, len(movieIdsOrder))
	for i, id := range movieIdsOrder {
		result[i] = *movies[id]
	}

	return result, nil
}

func (d *Database) SetReferentialIntegrity(value bool) error {
	onValue := "ON"
	if value == false {
		onValue = "OFF"
	}

	_, err := d.conn.Exec(fmt.Sprintf("PRAGMA foreign_keys = %s;", onValue))
	return err
}

func (d *Database) UpsertPerson(person Person, images []PersonImage) error {
	akaBytes, err := json.Marshal(person.AlsoKnownAs)
	if err != nil {
		return fmt.Errorf("failed to marshal AlsoKnownAs: %w", err)
	}

	query := `
    INSERT INTO persons (
        id, birthday, known_for_department, name, also_known_as,
        gender, popularity, place_of_birth, profile_path, adult, imdb_id
    )
    VALUES (
        :id, :birthday, :known_for_department, :name, :also_known_as,
        :gender, :popularity, :place_of_birth, :profile_path, :adult, :imdb_id
    )
    ON CONFLICT(id) DO UPDATE SET
        birthday = excluded.birthday,
        known_for_department = excluded.known_for_department,
        name = excluded.name,
        also_known_as = excluded.also_known_as,
        gender = excluded.gender,
        popularity = excluded.popularity,
        place_of_birth = excluded.place_of_birth,
        profile_path = excluded.profile_path,
        adult = excluded.adult,
        imdb_id = excluded.imdb_id
    `

	params := map[string]interface{}{
		"id":                   person.ID,
		"birthday":             person.Birthday,
		"known_for_department": person.KnownForDepartment,
		"name":                 person.Name,
		"also_known_as":        string(akaBytes),
		"gender":               person.Gender,
		"popularity":           person.Popularity,
		"place_of_birth":       person.PlaceOfBirth,
		"profile_path":         person.ProfilePath,
		"adult":                person.Adult,
		"imdb_id":              person.IMDbID,
	}

	_, err = d.conn.NamedExec(query, params)
	if err != nil {
		return fmt.Errorf("failed to upsert person: %w", err)
	}

	for _, image := range images {
		imgQuery := `
        INSERT INTO person_images (person_id, path)
        VALUES (:person_id, :path)
        ON CONFLICT(person_id, path) DO NOTHING
        `
		_, err = d.conn.NamedExec(imgQuery, map[string]interface{}{
			"person_id": person.ID,
			"path":      image.Path,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert person image: %w", err)
		}
	}

	return nil
}

func (d *Database) UpsertMovie(movie Movie, images []MovieImage) error {
	query := `
    INSERT INTO movies (id, title, release_date, adult, popularity, runtime)
    VALUES (:id, :title, :release_date, :adult, :popularity, :runtime)
    ON CONFLICT(id) DO UPDATE SET
        title = excluded.title,
        popularity = excluded.popularity,
        release_date = excluded.release_date,
        adult = excluded.adult,
        runtime = excluded.runtime
    `

	_, err := d.conn.NamedExec(query, movie)
	if err != nil {
		return fmt.Errorf("failed to upsert movie: %w", err)
	}

	for _, image := range images {
		imgQuery := `
        INSERT INTO movie_images (movie_id, path)
        VALUES (:movie_id, :path)
        ON CONFLICT(movie_id, path) DO NOTHING
        `
		_, err = d.conn.NamedExec(imgQuery, map[string]interface{}{
			"movie_id": movie.ID,
			"path":     image.Path,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert movie image: %w", err)
		}
	}

	return nil
}

func (d *Database) UpsertCredit(credit Credit) error {
	if !credit.JobType.IsValid() {
		return fmt.Errorf("invalid job type in upsert credit: %s", credit.JobType)
	}

	query := `
    INSERT INTO credits (person_id, movie_id, job_type)
    VALUES (:person_id, :movie_id, :job_type)
    ON CONFLICT(person_id, movie_id, job_type) DO NOTHING
    `

	_, err := d.conn.NamedExec(query, credit)
	if err != nil {
		return fmt.Errorf("failed to upsert credit: %w", err)
	}

	return nil
}

func (d *Database) UpsertMovies(movies []Movie) error {
	tx, err := d.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	query := `
    INSERT INTO movies (id, title, language, genres, release_date, adult, popularity, runtime)
    VALUES (:id, :title, :language, :genres, :release_date, :adult, :popularity, :runtime)
    ON CONFLICT(id) DO UPDATE SET
        title = excluded.title,
        genres = excluded.genres,
        language = excluded.language,
        adult = excluded.adult,
        release_date = excluded.release_date,
        popularity = excluded.popularity,
        runtime = excluded.runtime
    `

	stmt, err := tx.PrepareNamed(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}

	images := []MovieImage{}
	for _, movie := range movies {
		images = append(images, movie.Images...)
		_, err := stmt.Exec(movie)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert movie: %w", err)
		}
	}

	for _, image := range images {
		imgQuery := `
        INSERT INTO movie_images (movie_id, path)
        VALUES (:movie_id, :path)
        ON CONFLICT(movie_id, path) DO NOTHING
        `
		_, err = tx.NamedExec(imgQuery, map[string]interface{}{
			"movie_id": image.MovieID,
			"path":     image.Path,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert movie image: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (d *Database) UpsertPeople(people []Person) error {
	tx, err := d.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	query := `
    INSERT INTO persons (id, birthday, known_for_department, name, gender, popularity, place_of_birth, profile_path, adult, imdb_id)
    VALUES (:id, :birthday, :known_for_department, :name, :gender, :popularity, :place_of_birth, :profile_path, :adult, :imdb_id)
    ON CONFLICT(id) DO UPDATE SET
        birthday = excluded.birthday,
        known_for_department = excluded.known_for_department,
        name = excluded.name,
        gender = excluded.gender,
        popularity = excluded.popularity,
        place_of_birth = excluded.place_of_birth,
        profile_path = excluded.profile_path,
        adult = excluded.adult,
        imdb_id = excluded.imdb_id
    `

	stmt, err := tx.PrepareNamed(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}

	images := []PersonImage{}
	for _, person := range people {
		images = append(images, person.Images...)
		_, err := stmt.Exec(person)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert person: %w", err)
		}
	}

	for _, image := range images {
		imgQuery := `
        INSERT INTO person_images (person_id, path)
        VALUES (:person_id, :path)
        ON CONFLICT(person_id, path) DO NOTHING
        `
		_, err = tx.NamedExec(imgQuery, map[string]interface{}{
			"person_id": image.PersonID,
			"path":      image.Path,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert movie image: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (d *Database) UpsertCredits(credits []Credit) error {
	tx, err := d.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	query := `
    INSERT INTO credits (person_id, movie_id, job_type)
    VALUES (:person_id, :movie_id, :job_type)
    ON CONFLICT(person_id, movie_id, job_type) DO NOTHING
    `

	stmt, err := tx.PrepareNamed(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}

	for _, credit := range credits {
		_, err := stmt.Exec(credit)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert credit: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
