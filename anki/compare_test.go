package anki

import (
	"testing"
	"time"

	tmdbankigenerator "github.com/JonasRothmann/cine2nerdle-trainer"
)

func TestIsEqual(t *testing.T) {
	// Helper to create MaybeCloze slices
	makeMaybeCloze := func(contents ...string) []MaybeCloze {
		var result []MaybeCloze
		for _, content := range contents {
			result = append(result, MaybeCloze{Content: content, IsCloze: false})
		}
		return result
	}

	releaseDate := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	note1 := MovieNote{
		NoteID:         tmdbankigenerator.Ptr(int64(1)),
		TMDbID:         1001,
		MovieTitle:     "Movie A",
		ReleaseDate:    releaseDate,
		Genres:         []string{"Action", "Adventure"},
		Cast:           makeMaybeCloze("Actor A", "Actor B"),
		Director:       makeMaybeCloze("Director A"),
		Composer:       makeMaybeCloze("Composer A"),
		Writer:         makeMaybeCloze("Writer A"),
		Cinematograper: makeMaybeCloze("Cinematographer A"),
	}

	note2 := MovieNote{
		NoteID:         tmdbankigenerator.Ptr(int64(1)),
		TMDbID:         1001,
		MovieTitle:     "Movie A",
		ReleaseDate:    releaseDate,
		Genres:         []string{"Adventure", "Action"},      // Different order
		Cast:           makeMaybeCloze("Actor B", "Actor A"), // Different order
		Director:       makeMaybeCloze("Director A"),
		Composer:       makeMaybeCloze("Composer A"),
		Writer:         makeMaybeCloze("Writer A"),
		Cinematograper: makeMaybeCloze("Cinematographer A"),
	}

	note3 := MovieNote{
		NoteID:         tmdbankigenerator.Ptr(int64(1)),
		TMDbID:         1001,
		MovieTitle:     "Movie A",
		ReleaseDate:    releaseDate,
		Genres:         []string{"Action"}, // Different genres
		Cast:           makeMaybeCloze("Actor A", "Actor B"),
		Director:       makeMaybeCloze("Director A"),
		Composer:       makeMaybeCloze("Composer A"),
		Writer:         makeMaybeCloze("Writer A"),
		Cinematograper: makeMaybeCloze("Cinematographer A"),
	}

	tests := []struct {
		name     string
		first    MovieNote
		second   MovieNote
		expected bool
	}{
		{"Identical notes", note1, note1, true},
		{"Order doesn't matter", note1, note2, true},
		{"Different genres", note1, note3, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := test.first.IsEqual(test.second); result != test.expected {
				t.Errorf("IsEqual(%v, %v) = %v; want %v", test.first, test.second, result, test.expected)
			}
		})
	}
}
