package anki

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/JonasRothmann/ankiconnect"
	"github.com/pkg/errors"
	ankierrors "github.com/privatesquare/bkst-go-utils/utils/errors"
)

type MaybeCloze struct {
	IsCloze bool
	Content string
}

type MovieNote struct {
	NoteID *int64

	TMDbID         int
	MovieTitle     string
	ReleaseDate    time.Time
	Genres         []string
	Popularity     float32
	Cast           []MaybeCloze
	Director       []MaybeCloze
	Composer       []MaybeCloze
	Writer         []MaybeCloze
	Cinematograper []MaybeCloze

	Pictures []ankiconnect.Picture
}

const (
	movieTitle       = "Movie Title"
	movieReleaseDate = "Release Date"
	moviePeople      = "People"
	movieGenres      = "Genres"
	movieImage       = "Image"
	moviePopularity  = "Popularity"
)

const modelName = "Movie"

// plusOne returns i+1. Alternatively, you can define "add" to handle i+b.
func plusOne(i int) int {
	return i + 1
}

var peopleTemplate = template.Must(
	template.New("people-template").
		Funcs(template.FuncMap{
			"add": func(a, b int) int { return a + b },
			"inc": func(counter map[string]interface{}) int {
				// Perform type assertion and increment
				if val, ok := counter["value"].(int); ok {
					val++
					counter["value"] = val
					return val
				}
				// Fallback if value is not an int (shouldn't happen)
				counter["value"] = 1
				return 1
			},
			"dict": func(values ...interface{}) map[string]interface{} {
				dict := make(map[string]interface{})
				for i := 0; i < len(values); i += 2 {
					key := values[i].(string)
					dict[key] = values[i+1]
				}
				return dict
			},
		}).
		Parse(`
{{- define "RenderPeople" -}}
  {{- $counter := .Counter -}}
  {{- range $i, $person := .People -}}
    {{- if $i}}, {{end}}
    {{- if $person.IsCloze}}
      {{"{{c"}}{{inc $counter}}::{{$person.Content}}{{"}"}}}
    {{- else}}
      {{$person.Content}}
    {{- end}}
  {{- end}}
{{- end -}}

{{- $counter := dict "value" 0 -}} <!-- Initialize counter map -->

{{- if .Director -}}
Director(s):<br />
{{template "RenderPeople" dict "People" .Director "Counter" $counter}}
<br /><br />
{{end}}

{{- if .Composer -}}
Composer(s):<br />
{{template "RenderPeople" dict "People" .Composer "Counter" $counter}}
<br /><br />
{{end}}

{{- if .Writer -}}
Writer(s):<br />
{{template "RenderPeople" dict "People" .Writer "Counter" $counter}}
<br /><br />
{{end}}

{{- if .Cinematograper -}}
Cinematographer(s):<br />
{{template "RenderPeople" dict "People" .Cinematograper "Counter" $counter}}
<br /><br />
{{end}}

{{- if .Cast -}}
Cast:<br />
{{template "RenderPeople" dict "People" .Cast "Counter" $counter}}
<br /><br />
{{end}}
`),
)

func (c *AnkiClient) AddMovieNote(note MovieNote) (int64, error) {
	if !note.HasCloze() {
		return 0, ErrNoCloze
	}

	var sb strings.Builder
	if err := peopleTemplate.Execute(&sb, note); err != nil {
		panic(err)
	}

	ankiNote := ankiconnect.Note{
		DeckName:  c.deckName,
		ModelName: modelName,
		Fields: ankiconnect.Fields{
			movieTitle:       note.MovieTitle,
			movieReleaseDate: note.ReleaseDate.Format("2006"),
			moviePeople:      sb.String(),
			movieGenres:      strings.Join(note.Genres, ", "),
			movieImage:       picturesToField(note.Pictures),
			moviePopularity:  strconv.FormatFloat(float64(note.Popularity), 'f', 2, 64),
		},
		Picture: note.Pictures,
		Tags: Tags{}.
			Set(TagTMDbID, strconv.Itoa(note.TMDbID)).
			SetManyCloze(TagCast, note.Cast).
			SetManyCloze(TagComposer, note.Composer).
			SetManyCloze(TagWriter, note.Writer).
			SetManyCloze(TagDirector, note.Director).
			SetManyCloze(TagCinematographer, note.Cinematograper).
			SetMany(TagGenres, note.Genres),
	}

	var attempt int
	var id int64
	var restErr *ankierrors.RestErr

	for attempt = 0; attempt < 3; attempt++ {
		id, restErr = c.Connect.Notes.Add(ankiNote)
		if restErr != nil {
			if restErr.Error == "cannot create note because it is a duplicate" {
				ankiNote.Fields[movieTitle] += fmt.Sprintf(" %d", note.TMDbID)
			}
			fmt.Println("retrying")
		} else {
			break
		}
	}

	if restErr != nil {
		return 0, errors.Wrapf(RestErr(*restErr), "error when adding note via ankiconnect: note: %+v", ankiNote)
	}
	if id == 0 {
		return 0, errors.New("id zero value")
	}

	return id, nil
}

func (c *AnkiClient) UpsertMovieNote(note *MovieNote) (int64, error) {
	if !note.HasCloze() {
		return 0, ErrNoCloze
	}

	var attempt int
	var result *[]ankiconnect.ResultNotesInfo
	var restErr *ankierrors.RestErr

	for attempt = 0; attempt < 3; attempt++ {
		result, restErr = c.Connect.Notes.Get(c.ToQuery(*note))
		if restErr != nil {
			if restErr.Error == "cannot create note because it is a duplicate" {
				note.MovieTitle += fmt.Sprintf(" %d", note.TMDbID)
			}
			fmt.Printf("retrying %s\n", note.MovieTitle)
		} else {
			break
		}
	}

	if restErr != nil {
		return 0, errors.Wrapf(RestErr(*restErr), "error when getting note via ankiconnect 2: %s", note.MovieTitle)
	}

	if len(*result) > 0 {
		id, existingNote, err := resultNotesToMovieNote((*result)[0])
		if err != nil {
			return 0, err
		}
		if id == 0 {
			return 0, errors.New("id zero value")
		}
		note.NoteID = &id

		if existingNote.IsEqual(*note) {
			//fmt.Println("identical - skipping")
			return id, nil
		} else {
			fmt.Println("not identical - updating")
			var sb strings.Builder
			if err := peopleTemplate.Execute(&sb, *note); err != nil {
				panic(err)
			}

			id, err := c.Connect.Notes.Update(ankiconnect.UpdateNote{
				Id: id,
				Fields: ankiconnect.Fields{
					movieTitle:       note.MovieTitle,
					movieReleaseDate: note.ReleaseDate.Format("2006"),
					moviePeople:      sb.String(),
					movieGenres:      strings.Join(note.Genres, ", "),
					movieImage:       picturesToField(note.Pictures),
					moviePopularity:  strconv.FormatFloat(float64(note.Popularity), 'f', 2, 64),
				},
				Picture: note.Pictures,
				Tags: Tags{}.
					Set(TagTMDbID, strconv.Itoa(note.TMDbID)).
					SetManyCloze(TagCast, note.Cast).
					SetManyCloze(TagComposer, note.Composer).
					SetManyCloze(TagWriter, note.Writer).
					SetManyCloze(TagDirector, note.Director).
					SetManyCloze(TagCinematographer, note.Cinematograper).
					SetMany(TagGenres, note.Genres),
			})
			if err != nil {
				return 0, errors.Errorf("error when update note via ankiconnect: %s", err.Error)
			}

			return id, nil
		}
	} else {
		fmt.Println("doesn't exist - creating" + c.ToQuery(*note))

		noteID, err := c.AddMovieNote(*note)
		if err != nil {
			return 0, errors.Errorf("failed to add movie note: %s", err)
		}
		if noteID == 0 {
			return 0, errors.New("id zero value")
		}

		note.NoteID = &noteID

		return noteID, nil
	}
}

func (c *AnkiClient) ToQuery(note MovieNote) string {
	return fmt.Sprintf("note:%s deck:%s tag:tmdb:%d", modelName, c.deckName, note.TMDbID)
}

func (n MovieNote) HasCloze() bool {
	for _, val := range slices.Concat(n.Cast, n.Cinematograper, n.Composer, n.Director, n.Writer) {
		if val.IsCloze {
			return true
		}
	}

	return false
}

func picturesToField(pictures []ankiconnect.Picture) (content string) {
	for _, picture := range pictures {
		content += fmt.Sprintf("<img src='%s'>", picture.Filename)
	}

	return content
}
