package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/JonasRothmann/ankiconnect"
	tmdbankigenerator "github.com/JonasRothmann/cine2nerdle-trainer"
	"github.com/JonasRothmann/cine2nerdle-trainer/anki"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

func main() {
	db, err := tmdbankigenerator.NewDatabase()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "unable to start database"))
	}
	ids, extraIds := tmdbankigenerator.GetCastIDs()

	movies, err := db.GetMoviesByPersonIDs(ids, extraIds, 28)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to get movies"))
	}

	fmt.Println(strings.Join(lo.Map(ids, func(id int, index int) string {
		return strconv.Itoa(id)
	}), ", "))

	result := []tmdbankigenerator.Movie{}
	addedMovieIDs := make(map[int]bool)

	personToMovies := make(map[int][]tmdbankigenerator.Movie)
	for _, movie := range movies {
		personMap := make(map[int]tmdbankigenerator.MoviePerson)
		for _, person := range movie.Persons {
			personMap[person.ID] = person
		}
		movie.Persons = lo.Values(personMap)

		for _, person := range movie.Persons {
			personToMovies[person.ID] = append(personToMovies[person.ID], movie)
		}
	}

	for _, id := range ids {
		if movies, ok := personToMovies[id]; ok {
			for _, movie := range movies {
				if !addedMovieIDs[movie.ID] {
					result = append(result, movie)
					addedMovieIDs[movie.ID] = true
				}
			}
		} else {
			log.Printf("Warning: Person ID %d not found in map\n", id)
		}
	}

	client, err := anki.NewAnkiClient("Cine2Nerdle")
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to connect to ankiconnect"))
	}

	g := errgroup.Group{}
	g.SetLimit(5)
	mu := sync.Mutex{}

	moviesToKeep := make([]int64, 0, len(result))

	for _, movie := range result {
		note := anki.MovieNote{
			MovieTitle:  movie.Title,
			ReleaseDate: movie.ReleaseDate,
			TMDbID:      movie.ID,
			Popularity:  movie.Popularity,
			Genres:      strings.Split(movie.Genres, ", "),
		}

		for _, image := range movie.Images {
			after, found := strings.CutPrefix(image.Path, "/")
			if !found {
				fmt.Printf("no image in %s\n", movie.Title)
				continue
			}
			note.Pictures = append(note.Pictures, ankiconnect.Picture{
				Filename: after,
				URL:      fmt.Sprintf("https://image.tmdb.org/t/p/w500/%s", after),
				Fields:   []string{},
			})
		}

		for _, person := range movie.Persons {
			cloze := anki.MaybeCloze{
				IsCloze: person.InList,
				Content: person.Name,
			}

			switch person.JobType {
			case tmdbankigenerator.JobTypeCast:
				note.Cast = append(note.Cast, cloze)
			case tmdbankigenerator.JobTypeWriter:
				note.Writer = append(note.Writer, cloze)
			case tmdbankigenerator.JobTypeComposer, tmdbankigenerator.JobTypeComposer2, tmdbankigenerator.JobTypeComposer3, tmdbankigenerator.JobTypeComposer4:
				note.Composer = append(note.Composer, cloze)
			case tmdbankigenerator.JobTypeCinematographer:
				note.Cinematograper = append(note.Cinematograper, cloze)
			case tmdbankigenerator.JobTypeDirector:
				note.Director = append(note.Director, cloze)
			}
		}

		g.Go(func() error {
			id, err := client.UpsertMovieNote(&note)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			moviesToKeep = append(moviesToKeep, id)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Fatalln(errors.Wrap(err, "reason"))
	}

	client.RemoveUnusedIDs(moviesToKeep)
}
