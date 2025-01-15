package tmdbankigenerator

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

var PopularMovies []PopularMovie

type PopularMovie struct {
	ID         int     `json:"id"`
	Title      string  `json:"original_title"`
	Popularity float32 `json:"popularity"`
	Adult      bool    `json:"adult"`
	Video      bool    `json:"video"`
}

var PopularPeople []PopularPerson

type PopularPerson struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Popularity float32 `json:"popularity"`
	Adult      bool    `json:"adult"`
}

func GetTopMovies() ([]PopularMovie, error) {
	if len(PopularMovies) > 0 {
		return PopularMovies, nil
	}

	popularMoviesJson, err := os.ReadFile("movie_ids_01_10_2025.json")
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to read movie_ids"))
	}

	movies := strings.Replace(string(slices.Concat([]byte{'['}, popularMoviesJson, []byte{']'})), "}\n", "},\n", -1)
	movies = strings.Replace(movies, ",\n]", "\n]", 1)

	if err := json.Unmarshal([]byte(movies), &PopularMovies); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal popular movies json: %s")
	}

	return PopularMovies, nil
}

/*
func init() {
	popularPeopleJson, err := os.ReadFile("person_ids_01_11_2025.json")
	if err != nil {
		log.Fatalln(err)
	}
	popularMoviesJson, err := os.ReadFile("movie_ids_01_10_2025.json")
	if err != nil {
		log.Fatalln(err)
	}

	movies := strings.Replace(string(slices.Concat([]byte{'['}, popularMoviesJson, []byte{']'})), "}\n", "},\n", -1)
	movies = strings.Replace(movies, ",\n]", "\n]", 1)
	people := strings.Replace(string(slices.Concat([]byte{'['}, popularPeopleJson, []byte{']'})), "}\n", "},\n", -1)
	people = strings.Replace(people, ",\n]", "\n]", 1)

	fmt.Println(movies)

	if err := json.Unmarshal([]byte(movies), &PopularMovies); err != nil {
		log.Fatalf("Failed to unmarshal popular movies json: %s", err)
	}
	if err := json.Unmarshal([]byte(people), &PopularPeople); err != nil {
		log.Fatalf("Failed to unmarshal popular people json: %s", err)
	}
}

func GetTopPeople(amount int) []PopularPerson {
	people := PopularPeople

	slices.SortFunc(people, func(a, b PopularPerson) int {
		return cmp.Compare(b.Popularity, a.Popularity)
	})

	return people[:amount]
}
*/
