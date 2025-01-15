package main

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync"

	tmdbankigenerator "github.com/JonasRothmann/cine2nerdle-trainer"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"gitlab.com/metakeule/fmtdate"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	WhitelistPersonIDs []int
	Limit              int
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database, err := tmdbankigenerator.NewDatabase()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to start database"))
	}
	defer database.Close()

	database.SetReferentialIntegrity(false)
	defer database.SetReferentialIntegrity(true)

	tmdbApiKey, ok := os.LookupEnv("TMDB_API_KEY")
	if !ok {
		log.Fatalln("No tmdb api key set")
	}

	tmdb, err := tmdbankigenerator.NewTMDbClient(tmdbApiKey)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to connect to tmdb"))
	}

	/*
		config := Config{}
		pflag.IntSliceVar(&config.WhitelistPersonIDs, "whitelist", nil, "Only pull the data of these people")
		pflag.Parse()

		config.WhitelistPersonIDs = tmdbankigenerator.GetCastIDs()

		if len(config.WhitelistPersonIDs) == 0 {
			log.Fatalln("full indexing not implemented")
		}*/

	/*topPeople := tmdbankigenerator.GetTopPeople(50)

	for _, person := range topPeople {
		fmt.Println(person.Name)
		config.WhitelistPersonIDs = append(config.WhitelistPersonIDs, person.ID)
	}*/

	g := errgroup.Group{}

	var (
		movies    map[int64]tmdbankigenerator.Movie = make(map[int64]tmdbankigenerator.Movie)
		movieLock sync.Mutex

		people     map[int64]tmdbankigenerator.Person = make(map[int64]tmdbankigenerator.Person)
		peopleLock sync.Mutex

		credits    []tmdbankigenerator.Credit
		creditLock sync.Mutex
	)

	popularMovies, err := tmdbankigenerator.GetTopMovies()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to get top movies"))
	}

	slices.SortFunc(popularMovies, func(a, b tmdbankigenerator.PopularMovie) int {
		return cmp.Compare(b.Popularity, a.Popularity)
	})

	max := 10000
	total := max
	count := 0
	minPopularity := float32(1)

	log.Print("starting")
	for i, movie := range popularMovies[:max] {
		if movie.Popularity < minPopularity {
			total--
			continue
		}
		g.Go(func() error {
			tmdbMovie, err := tmdb.GetMovieDetails(movie.ID, map[string]string{
				"append_to_response": "keywords,credits",
			})
			if err != nil {
				return err
			}

			if tmdbMovie.VoteCount < 10 ||
				tmdbankigenerator.IsDisallowedGenre(tmdbMovie.Genres) ||
				tmdbankigenerator.IsShortFilm(*tmdbMovie.Keywords.MovieKeywords, tmdbMovie.Runtime) ||
				!slices.Contains(tmdbankigenerator.ValidLanguages, tmdbankigenerator.Language(tmdbMovie.OriginalLanguage)) {
				return nil
			}

			movieLock.Lock()
			if _, ok := movies[tmdbMovie.ID]; !ok {
				releaseDate, err := fmtdate.Parse("YYYY-MM-DD", tmdbMovie.ReleaseDate)
				if err != nil {
					movieLock.Unlock()
					fmt.Println(err)
					return err
				}
				movies[tmdbMovie.ID] = tmdbankigenerator.Movie{
					ID:          int(tmdbMovie.ID),
					Title:       tmdbMovie.Title,
					Popularity:  tmdbMovie.Popularity,
					ReleaseDate: releaseDate,
					Adult:       tmdbMovie.Adult,
					Language:    tmdbankigenerator.Language(tmdbMovie.OriginalLanguage),
					Runtime:     tmdbMovie.Runtime,
					Images: []tmdbankigenerator.MovieImage{
						{
							MovieID: int(tmdbMovie.ID),
							Path:    tmdbMovie.PosterPath,
						},
					},
				}
			}
			movieLock.Unlock()

			for _, person := range tmdbMovie.Credits.MovieCredits.Cast {
				if person.Popularity < minPopularity {
					continue
				}

				peopleLock.Lock()
				if _, ok := people[person.ID]; !ok {
					people[person.ID] = tmdbankigenerator.Person{
						ID:                 int(person.ID),
						KnownForDepartment: person.KnownForDepartment,
						Name:               person.Name,
						Popularity:         person.Popularity,
						Gender:             person.Gender,
						AlsoKnownAs:        []string{},
						ProfilePath:        person.ProfilePath,
						Adult:              person.Adult,
						Images: []tmdbankigenerator.PersonImage{
							{
								PersonID: int(person.ID),
								Path:     person.ProfilePath,
							},
						},
					}
				}
				peopleLock.Unlock()

				creditLock.Lock()
				credits = append(credits, tmdbankigenerator.Credit{
					PersonID: int(person.ID),
					MovieID:  int(tmdbMovie.ID),
					JobType:  tmdbankigenerator.JobTypeCast,
				})
				creditLock.Unlock()
			}

			for _, person := range tmdbMovie.Credits.MovieCredits.Crew {

				if person.Popularity < minPopularity {
					continue
				}

				jobType := tmdbankigenerator.JobType(strings.Trim(person.Job, " "))
				if !jobType.IsValid() {
					fmt.Printf("Invalid job type on movie %s: %s\n", tmdbMovie.Title, jobType)
					continue
				}

				peopleLock.Lock()
				if _, ok := people[person.ID]; !ok {
					people[person.ID] = tmdbankigenerator.Person{
						ID:                 int(person.ID),
						KnownForDepartment: person.KnownForDepartment,
						Name:               person.Name,
						Popularity:         person.Popularity,
						Gender:             person.Gender,
						ProfilePath:        person.ProfilePath,
						Adult:              person.Adult,
						AlsoKnownAs:        []string{},
						Images: []tmdbankigenerator.PersonImage{
							{
								PersonID: int(person.ID),
								Path:     person.ProfilePath,
							},
						},
					}
				}
				peopleLock.Unlock()

				creditLock.Lock()
				credits = append(credits, tmdbankigenerator.Credit{
					PersonID: int(person.ID),
					MovieID:  int(tmdbMovie.ID),
					JobType:  jobType,
				})
				creditLock.Unlock()
			}

			count++
			if i%100 == 0 {
				fmt.Printf("%d%% done\n", (count*100)/max)
			}

			return nil
		})
	}

	/*for _, id := range config.WhitelistPersonIDs {
		g.Go(func() error {
			person, credits, err := tmdb.GetAllCreditsByPersonID(id)
			if err != nil {
				return err
			}

			database.UpsertPerson(person, person.Images)

			mu.Lock()
			defer mu.Unlock()

			for _, credit := range credits {
				if err := database.UpsertMovie(*credit.Movie, credit.Movie.Images); err != nil {
					return err
				}
				if err := database.UpsertCredit(credit); err != nil {
					return err
				}
			}

			return nil
		})
	}*/

	if err := g.Wait(); err != nil {
		log.Fatalln(errors.Wrap(err, "reason2"))
	}

	fmt.Printf("Got %d people, %d movies and %d credits\n", len(people), len(movies), len(credits))

	var peopleArray = []tmdbankigenerator.Person{}
	for _, person := range people {
		peopleArray = append(peopleArray, person)
	}
	var movieArray = []tmdbankigenerator.Movie{}
	for _, movie := range movies {
		movieArray = append(movieArray, movie)
	}

	if err := database.UpsertMovies(movieArray); err != nil {
		log.Fatalln(errors.Wrap(err, "failed to insert movies"))
	}
	if err := database.UpsertPeople(peopleArray); err != nil {
		log.Fatalln(errors.Wrap(err, "faild to insert people"))
	}
	if err := database.UpsertCredits(credits); err != nil {
		log.Fatalln(errors.Wrap(err, "failed to insert credits"))
	}

	fmt.Println("done")
}
