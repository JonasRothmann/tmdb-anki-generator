package tmdbankigenerator

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"golang.org/x/time/rate"
)

type TMDbClient struct {
	client *tmdb.Client

	timeLimit *rate.Limiter
}

func (c *TMDbClient) Wait() {
	c.WaitN(1)
}

func (c *TMDbClient) WaitN(n int) {
	c.timeLimit.WaitN(context.Background(), n)
}

func NewTMDbClient(apiKey string) (*TMDbClient, error) {
	client, err := tmdb.Init(apiKey)
	if err != nil {
		return nil, err
	}

	return &TMDbClient{
		client:    client,
		timeLimit: rate.NewLimiter(rate.Every(time.Second/30), 30),
	}, nil
}

const minVoteCount = 10

var disallowedGenres = []int64{99 /* Documentaries */, 10402 /* Music */}
var disallowedGenresMap = map[int64]struct{}{
	99:    struct{}{},
	10402: struct{}{},
}

func (c *TMDbClient) GetAllCreditsByPersonID(personId int) (Person, []Credit, error) {
	c.Wait()
	tmdbPerson, err := c.client.GetPersonDetails(personId, map[string]string{
		"append_to_response": "movie_credits,images",
	})
	if err != nil {
		return Person{}, nil, err
	}

	if tmdbPerson.MovieCredits == nil {
		return Person{}, nil, errors.New("No movie credits returned")
	}

	var credits []Credit
	for _, credit := range *&tmdbPerson.MovieCredits.Cast {
		if credit.VoteCount < minVoteCount || IsDisallowedGenreIDs(credit.GenreIDs) {
			continue
		}

		movie, keywords, err := c.GetMovie(int(credit.ID))
		if err != nil {
			return Person{}, nil, err
		}

		if IsShortFilm(keywords, movie.Runtime) {
			fmt.Printf("%s is a short film\n", movie.Title)
			continue
		}

		credits = append(credits, Credit{
			PersonID: personId,
			MovieID:  int(credit.ID),
			JobType:  JobTypeCast,
			Movie:    &movie,
		})
	}

	for _, credit := range *&tmdbPerson.MovieCredits.Crew {
		if credit.VoteCount < minVoteCount || IsDisallowedGenreIDs(credit.GenreIDs) {
			continue
		}

		jobType := JobType(strings.Trim(credit.Job, " "))
		if !jobType.IsValid() {
			fmt.Printf("Invalid job type on movie %s: %s\n", credit.Title, jobType)
			continue
		}

		movie, keywords, err := c.GetMovie(int(credit.ID))
		if err != nil {
			return Person{}, nil, err
		}

		if IsShortFilm(keywords, movie.Runtime) {
			fmt.Printf("%s is a short film\n", movie.Title)
			continue
		}

		credits = append(credits, Credit{
			PersonID: personId,
			MovieID:  int(credit.ID),
			JobType:  jobType,
			Movie:    &movie,
		})
	}

	return Person{
		ID:                 int(tmdbPerson.ID),
		Name:               tmdbPerson.Name,
		Birthday:           tmdbPerson.Birthday,
		KnownForDepartment: tmdbPerson.KnownForDepartment,
		AlsoKnownAs:        tmdbPerson.AlsoKnownAs,
		Gender:             tmdbPerson.Gender,
		Popularity:         tmdbPerson.Popularity,
		PlaceOfBirth:       tmdbPerson.PlaceOfBirth,
		ProfilePath:        tmdbPerson.ProfilePath,
		Adult:              tmdbPerson.Adult,
		IMDbID:             tmdbPerson.IMDbID,
		Images: []PersonImage{
			{
				PersonID: int(tmdbPerson.ID),
				Path:     tmdbPerson.ProfilePath,
			},
		},
	}, credits, nil
}

func (c *TMDbClient) GetMovieDetails(id int, urlOptions map[string]string) (*tmdb.MovieDetails, error) {
	c.Wait()
	return c.client.GetMovieDetails(id, urlOptions)
}

func (c *TMDbClient) GetMovie(movieId int) (Movie, tmdb.MovieKeywords, error) {
	c.Wait()

	tmdbMovie, err := c.client.GetMovieDetails(movieId, map[string]string{
		"append_to_response": "keywords",
	})
	if err != nil {
		return Movie{}, tmdb.MovieKeywords{}, err
	}

	return Movie{
		ID:         int(tmdbMovie.ID),
		Title:      tmdbMovie.Title,
		Popularity: tmdbMovie.Popularity,
		Runtime:    tmdbMovie.Runtime,
		Images: []MovieImage{
			{
				MovieID: int(tmdbMovie.ID),
				Path:    tmdbMovie.PosterPath,
			},
		},
	}, *tmdbMovie.Keywords.MovieKeywords, nil
}

func IsDisallowedGenreIDs(genres []int64) bool {
	for _, genre := range genres {
		if slices.Contains(disallowedGenres, genre) {
			fmt.Printf("Has dissallowed genre: %d\n", genre)
			return true
		}
	}

	return false
}

func IsDisallowedGenre(genres []struct {
	ID   int64  "json:\"id\""
	Name string "json:\"name\""
}) bool {
	for _, genre := range genres {
		if _, ok := disallowedGenresMap[genre.ID]; ok {
			fmt.Printf("Has dissallowed genre: %d\n", genre)
			return true
		}
	}

	return false
}

var disallowedKeywords = []string{"short film"}

func IsShortFilm(keywords tmdb.MovieKeywords, runtime int) bool {
	if runtime < 35 {
		return true
	}

	for _, keyword := range keywords.Keywords {
		if slices.Contains(disallowedKeywords, strings.ToLower(keyword.Name)) {
			return true
		}
	}

	return false
}

/*
func (c *TMDbClient) GetAllCreditsByPersonID(personId int) ([]Credit, error) {
	var credits []Credit

	c.Wait()
	tmdbCredits, err := c.client.GetPersonMovieCredits(personId, map[string]string{})
	if err != nil {
		return nil, err
	}

	for _, credit := range tmdbCredits.Crew {
		jobType := JobType(credit.Job)
		if jobType.IsValid() == false {
			fmt.Printf("Skipping invalid cine2nerdle job: %s\n", jobType)
			continue
		}

		credits = append(credits, Credit{
			PersonID: personId,
			MovieID:  credit.ID,
			JobType:  jobType,
		})
	}

	for _, credit := range tmdbCredits.Cast {
		credits = append(credits, Credit{
			PersonID: personId,
			MovieID:  credit.ID,
			JobType:  JobTypeCast,
		})
	}

	return credits, nil
} */

/*

func (c *TMDbClient) GetCreditsFromPersonID(ids []int) ([]any, error) {
	g := errgroup.Group{}
	mu := sync.Mutex{}

	for _, id := range ids {
		g.Go(func() error {
			credits, err := c.GetPersonMovieCredits(id, map[string]string{})
			if err != nil {
				return nil
			}

			mu.Lock()
			defer mu.Unlock()

		})

	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return nil, nil
}*/
