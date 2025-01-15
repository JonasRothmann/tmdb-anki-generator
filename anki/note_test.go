package anki_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	tmdbankigenerator "github.com/JonasRothmann/cine2nerdle-trainer"
	"github.com/JonasRothmann/cine2nerdle-trainer/anki"
	"github.com/fatih/set"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	client, err := anki.NewAnkiClient("test")
	require.NoError(t, err)
	defer client.Connect.Decks.Delete("test")

	releaseDate, err := time.Parse("2006", time.Now().Format("2006"))
	require.NoError(t, err)

	movie := anki.MovieNote{
		TMDbID:      118,
		MovieTitle:  "Shawshank Redemption",
		ReleaseDate: releaseDate,
		Genres:      []string{"Drama", "Thriller"},
		Cast:        []anki.MaybeCloze{{IsCloze: false, Content: "Jon Robinsson"}, {IsCloze: true, Content: "Morgan Freeman"}},
		Director:    []anki.MaybeCloze{{IsCloze: true, Content: "Bobby Nobody"}},
	}
	id, err := client.AddMovieNote(movie)
	movie.NoteID = tmdbankigenerator.Ptr(id)
	require.NoError(t, err)

	results, restErr := client.Connect.Cards.Get("tag:tmdb:118")
	require.Nil(t, restErr)
	require.Len(t, *results, 1)

	movies, err := client.GetAllMovies()
	require.NoError(t, err)
	require.Len(t, movies, 1)

	expectedMovie := movie
	returnedMovie := movies[0]

	require.NotEmpty(t, returnedMovie.TMDbID)

	sortStringSlice(expectedMovie.Genres)
	sortStringSlice(returnedMovie.Genres)

	sortMaybeClozeSlice(expectedMovie.Cast)
	sortMaybeClozeSlice(returnedMovie.Cast)

	sortMaybeClozeSlice(expectedMovie.Director)
	sortMaybeClozeSlice(returnedMovie.Director)

	require.Equal(t, expectedMovie, returnedMovie, "Movies should match")
}

func TestNewCast(t *testing.T) {
	tests := []struct {
		name    string
		initial []anki.MovieNote
		updated []anki.MovieNote
		wants   []anki.MovieNote
		err     bool
	}{
		{
			name:    "Add a single new movie",
			initial: []anki.MovieNote{},
			updated: []anki.MovieNote{
				{
					TMDbID:      500,
					MovieTitle:  "Pulp Fiction - TEST",
					ReleaseDate: mustParseYear("1994"),
					Genres:      []string{"Crime", "Drama"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "John Travolta"},
						{IsCloze: true, Content: "Samuel L. Jackson"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: true, Content: "Quentin Tarantino"},
					},
				},
			},
			wants: []anki.MovieNote{
				{
					TMDbID:      500,
					MovieTitle:  "Pulp Fiction - TEST",
					ReleaseDate: mustParseYear("1994"),
					Genres:      []string{"Crime", "Drama"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "John Travolta"},
						{IsCloze: true, Content: "Samuel L. Jackson"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: true, Content: "Quentin Tarantino"},
					},
				},
			},
			err: false,
		},
		{
			name: "Add multiple and update one",
			initial: []anki.MovieNote{
				{
					TMDbID:      42,
					MovieTitle:  "The Hitchhiker's Guide to the Galaxy - TEST",
					ReleaseDate: mustParseYear("2005"),
					Genres:      []string{"Adventure", "Comedy", "Sci-Fi"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Martin Freeman"},
						{IsCloze: true, Content: "Zooey Deschanel"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Garth Jennings"},
					},
				},
				{
					TMDbID:      7,
					MovieTitle:  "Se7en - TEST",
					ReleaseDate: mustParseYear("1995"),
					Genres:      []string{"Crime", "Mystery", "Thriller"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Brad Pitt"},
						{IsCloze: true, Content: "Morgan Freeman"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "David Fincher"},
					},
				},
			},
			updated: []anki.MovieNote{
				// Updating Hitchhiker with a new cast member
				{
					TMDbID:      42,
					MovieTitle:  "The Hitchhiker's Guide to the Galaxy - TEST",
					ReleaseDate: mustParseYear("2005"),
					Genres:      []string{"Adventure", "Comedy", "Sci-Fi"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Martin Freeman"},
						{IsCloze: true, Content: "Zooey Deschanel"},
						{IsCloze: true, Content: "Alan Rickman"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Garth Jennings"},
					},
				},
				// Adding a completely new movie
				{
					TMDbID:      2023,
					MovieTitle:  "Dune Part Two - TEST",
					ReleaseDate: mustParseYear("2023"),
					Genres:      []string{"Adventure", "Drama", "Sci-Fi"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Timothée Chalamet"},
						{IsCloze: true, Content: "Zendaya"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Denis Villeneuve"},
					},
				},
			},
			wants: []anki.MovieNote{
				{
					TMDbID:      42,
					MovieTitle:  "The Hitchhiker's Guide to the Galaxy - TEST",
					ReleaseDate: mustParseYear("2005"),
					Genres:      []string{"Adventure", "Comedy", "Sci-Fi"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Martin Freeman"},
						{IsCloze: false, Content: "Zooey Deschanel"},
						{IsCloze: true, Content: "Alan Rickman"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Garth Jennings"},
					},
				},
				{
					TMDbID:      7,
					MovieTitle:  "Se7en - TEST",
					ReleaseDate: mustParseYear("1995"),
					Genres:      []string{"Crime", "Mystery", "Thriller"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Brad Pitt"},
						{IsCloze: true, Content: "Morgan Freeman"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "David Fincher"},
					},
				},
				{
					TMDbID:      2023,
					MovieTitle:  "Dune Part Two - TEST",
					ReleaseDate: mustParseYear("2023"),
					Genres:      []string{"Adventure", "Drama", "Sci-Fi"},
					Cast: []anki.MaybeCloze{
						{IsCloze: false, Content: "Timothée Chalamet"},
						{IsCloze: true, Content: "Zendaya"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Denis Villeneuve"},
					},
				},
			},
			err: false,
		},
		{
			name: "No changes from initial to updated",
			initial: []anki.MovieNote{
				{
					TMDbID:     1000,
					MovieTitle: "Some Original Movie - TEST",
					// Optional fields to illustrate a more “complete” note
					ReleaseDate: mustParseYear("2020"),
					Genres:      []string{"Mystery"},
					Cast: []anki.MaybeCloze{
						{IsCloze: true, Content: "Actor One"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Director One"},
					},
				},
			},
			updated: []anki.MovieNote{
				// Exactly the same data
				{
					TMDbID:      1000,
					MovieTitle:  "Some Original Movie - TEST",
					ReleaseDate: mustParseYear("2020"),
					Genres:      []string{"Mystery"},
					Cast: []anki.MaybeCloze{
						{IsCloze: true, Content: "Actor One"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Director One"},
					},
				},
			},
			wants: []anki.MovieNote{
				{
					TMDbID:      1000,
					MovieTitle:  "Some Original Movie - TEST",
					ReleaseDate: mustParseYear("2020"),
					Genres:      []string{"Mystery"},
					Cast: []anki.MaybeCloze{
						{IsCloze: true, Content: "Actor One"},
					},
					Director: []anki.MaybeCloze{
						{IsCloze: false, Content: "Director One"},
					},
				},
			},
			err: false,
		},
		{
			name:    "Attempt to upsert invalid data",
			initial: []anki.MovieNote{},
			updated: []anki.MovieNote{
				{
					TMDbID:     -1, // Possibly triggers an error in your client
					MovieTitle: "Invalid ID Movie",
				},
			},
			wants: []anki.MovieNote{},
			err:   true, // If we expect an error for invalid TMDbID
		},
	}

	allTMDbIds := set.New(set.NonThreadSafe)

	for i, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			client, err := anki.NewAnkiClient(fmt.Sprintf("test-%d", i))
			require.NoError(t, err)
			defer client.Connect.Decks.Delete(fmt.Sprintf("test-%d", i))

			for _, movie := range tst.initial {
				id, err := client.UpsertMovieNote(&movie)
				movie.NoteID = tmdbankigenerator.Ptr(id)
				require.NoError(t, err)

				allTMDbIds.Add(movie.TMDbID)
			}

			keepIds := []int64{}
			for _, movie := range tst.updated {
				noteID, err := client.UpsertMovieNote(&movie)
				if tst.err {
					require.Error(t, err)
					return
				} else {
					require.NoError(t, err)
				}
				require.NotNil(t, noteID)
				keepIds = append(keepIds, noteID)

				allTMDbIds.Add(movie.TMDbID)
			}

			movies, err := client.GetAllMovies()
			require.NoError(t, err)

			expectedTMDbIds := set.New(set.NonThreadSafe)
			for _, movie := range movies {
				expectedTMDbIds.Add(movie.TMDbID)
			}
			require.ElementsMatch(t, expectedTMDbIds.List(), allTMDbIds.List())

			if len(keepIds) > 0 {
				err = client.RemoveUnusedIDs(keepIds)
				require.NoError(t, err)
			}
		})
	}
}

func sortMaybeClozeSlice(slice []anki.MaybeCloze) {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Content < slice[j].Content
	})
}

func sortStringSlice(slice []string) {
	sort.Strings(slice)
}

func mustParseYear(y string) time.Time {
	t, _ := time.Parse("2006", y)
	return t
}
