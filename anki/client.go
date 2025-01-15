package anki

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JonasRothmann/ankiconnect"
	"github.com/fatih/set"
	"github.com/pkg/errors"
	ankierrors "github.com/privatesquare/bkst-go-utils/utils/errors"
)

var (
	ErrNoteInvalid = errors.New("The format of the anki note is invalid")
	ErrNotFound    = errors.New("Not found")
	ErrNoCloze     = errors.New("No cloze in payload")
)

type AnkiClient struct {
	deckName string
	Connect  *ankiconnect.Client
}

func NewAnkiClient(deckName string) (*AnkiClient, error) {
	client := ankiconnect.NewClient()
	restErr := client.Ping()
	if restErr != nil {
		return nil, RestErr(*restErr)
	}

	/*
		decks, restErr := client.Decks.GetAll()
		if restErr != nil {
			return nil, errors.New(restErr.Error)
		}

		deckExists := false
		for _, deck := range *decks {
			if deck == deckName
			deckExists = true
		} */

	if restErr = client.Decks.Create(deckName); restErr != nil {
		return nil, RestErr(*restErr)
	}

	return &AnkiClient{
		deckName: deckName,
		Connect:  client,
	}, nil
}

func (c *AnkiClient) RemoveUnusedIDs(keepIds []int64) error {
	removeIds := []int64{}
	keepIdsSet := set.New(set.NonThreadSafe)
	for _, id := range keepIds {
		keepIdsSet.Add(id)
	}

	results, restErr := c.Connect.Notes.Get(fmt.Sprintf("note:Movie deck:%s", c.deckName))
	if restErr != nil {
		return RestErr(*restErr)
	}

	for _, result := range *results {
		if !keepIdsSet.Has(result.NoteId) {
			removeIds = append(removeIds, result.NoteId)
		}
	}

	if len(removeIds) == 0 {
		fmt.Println("Empty removeIds")
		return nil
	}

	restErr = c.Connect.Notes.Delete(removeIds)
	if restErr != nil {
		return RestErr(*restErr)
	}

	return nil
}

func (c *AnkiClient) GetAllMovies() ([]MovieNote, error) {
	results, restErr := c.Connect.Notes.Get(fmt.Sprintf("note:%s deck:%s", modelName, c.deckName))
	if restErr != nil {
		return nil, RestErr(*restErr)
	}

	notes := make([]MovieNote, 0, len(*results))
	for _, result := range *results {
		_, note, err := resultNotesToMovieNote(result)
		if err != nil {
			return nil, err
		}

		notes = append(notes, note)
	}

	return notes, nil
}

func resultNotesToMovieNote(result ankiconnect.ResultNotesInfo) (int64, MovieNote, error) {
	tags := Tags(result.Tags)

	tmdbIDTag, ok := tags.GetOne(TagTMDbID)
	if !ok {
		return 0, MovieNote{}, errors.Wrap(ErrNoteInvalid, "tmdb tag missing")
	}
	tmdbID, err := strconv.Atoi(tmdbIDTag)
	if err != nil {
		return 0, MovieNote{}, err
	}

	note := MovieNote{
		NoteID:         &result.NoteId,
		TMDbID:         tmdbID,
		Cast:           tags.GetAllMaybeCloze(TagCast),
		Director:       tags.GetAllMaybeCloze(TagDirector),
		Composer:       tags.GetAllMaybeCloze(TagComposer),
		Writer:         tags.GetAllMaybeCloze(TagWriter),
		Cinematograper: tags.GetAllMaybeCloze(TagCinematographer),
		Genres:         tags.GetAll(TagGenres),
	}

	if value, ok := result.Fields[movieTitle]; !ok || value.Value == "" {
		return 0, MovieNote{}, errors.Wrap(ErrNoteInvalid, "movie title missing")
	} else {
		note.MovieTitle = value.Value
	}

	if value, ok := result.Fields[moviePopularity]; !ok || value.Value == "" {
		return 0, MovieNote{}, errors.Wrap(ErrNoteInvalid, "popularity missing")
	} else {
		popularity, err := strconv.ParseFloat(value.Value, 64)
		if err != nil {
			return 0, MovieNote{}, err
		}

		note.Popularity = float32(popularity)
	}

	if value, ok := result.Fields[movieReleaseDate]; !ok {
		return 0, MovieNote{}, errors.Wrap(ErrNoteInvalid, "release date missing")
	} else if value.Value != "" {
		var err error
		note.ReleaseDate, err = time.Parse("2006", value.Value)
		if err != nil {
			return 0, MovieNote{}, err
		}
	}
	return result.NoteId, note, nil
}

type Tags []string

var (
	TagTMDbID          string = "tmdb"
	TagCast                   = "cast"
	TagDirector               = "director"
	TagComposer               = "composer"
	TagCinematographer        = "cinematographer"
	TagWriter                 = "writer"
	TagGenres                 = "genres"
)

func (t Tags) GetOne(key string) (string, bool) {
	for _, tag := range t {
		if after, found := strings.CutPrefix(tag, fmt.Sprintf("%s:", key)); found {
			after = strings.ReplaceAll(after, "_", " ")
			return after, true
		}
	}

	return "", false
}

func (t Tags) GetAll(key string) []string {
	results := []string{}

	for _, tag := range t {
		if after, found := strings.CutPrefix(tag, fmt.Sprintf("%s:", key)); found {
			after = strings.ReplaceAll(after, "_", " ")
			results = append(results, after)
		}
	}

	return results
}

func (t Tags) GetAllMaybeCloze(key string) []MaybeCloze {
	var results []MaybeCloze

	for _, tag := range t {
		if after, found := strings.CutPrefix(tag, fmt.Sprintf("%s:", key)); found {
			isCloze := strings.HasSuffix(after, ":cloze")
			if isCloze {
				after = strings.TrimSuffix(after, ":cloze")
			}
			after = strings.ReplaceAll(after, "_", " ")
			results = append(results, MaybeCloze{
				Content: after,
				IsCloze: isCloze,
			})
		}
	}

	return results
}

func (t Tags) Set(key string, value string) Tags {
	value = strings.ReplaceAll(value, " ", "_")
	t = append(t, fmt.Sprintf("%s:%s", key, value))

	return t
}

func (t Tags) SetCloze(key string, value MaybeCloze) Tags {
	suffix := ""
	if value.IsCloze {
		suffix = ":cloze"
	}

	value.Content = strings.ReplaceAll(value.Content, " ", "_")
	t = append(t, fmt.Sprintf("%s:%s%s", key, value.Content, suffix))

	return t
}

func (t Tags) SetManyCloze(key string, values []MaybeCloze) Tags {
	for _, value := range values {
		suffix := ""
		if value.IsCloze {
			suffix = ":cloze"
		}

		value.Content = strings.ReplaceAll(value.Content, " ", "_")
		t = append(t, fmt.Sprintf("%s:%s%s", key, value.Content, suffix))
	}

	return t
}

func (t Tags) SetMany(key string, values []string) Tags {
	for _, value := range values {
		value = strings.ReplaceAll(value, " ", "_")
		t = append(t, fmt.Sprintf("%s:%s", key, value))
	}

	return t
}

func RestErr(err ankierrors.RestErr) error {
	return errors.New(fmt.Sprintf("%s (code: %d)", err.Error, err.StatusCode))
}
