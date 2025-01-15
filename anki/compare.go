package anki

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

func normalizeMaybeClozeSlice(slice []MaybeCloze) []MaybeCloze {
	if len(slice) == 0 {
		return nil
	}

	// Create a copy to avoid modifying the original slice
	normalized := make([]MaybeCloze, len(slice))
	copy(normalized, slice)

	// Sort using a custom comparator
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].IsCloze != normalized[j].IsCloze {
			return normalized[i].IsCloze
		}
		return normalized[i].Content < normalized[j].Content
	})

	return normalized
}

func normalizeStringSlice(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}
	// Create a copy to avoid modifying the original slice
	normalized := make([]string, len(slice))
	copy(normalized, slice)

	// Sort the slice
	sort.Strings(normalized)

	return normalized
}

func (n MovieNote) IsEqual(other MovieNote) bool {
	// Compare TMDbID
	if n.TMDbID != other.TMDbID {
		fmt.Printf("TMDbID not equal: %v != %v\n", n.TMDbID, other.TMDbID)
		return false
	}

	if strconv.FormatFloat(float64(n.Popularity), 'g', 2, 64) != strconv.FormatFloat(float64(other.Popularity), 'g', 2, 64) {
		fmt.Printf("Popularity not equal: %.2f != %.2f\n", n.Popularity, other.Popularity)
		return false
	}

	// Compare MovieTitle
	if n.MovieTitle != other.MovieTitle {
		fmt.Printf("MovieTitle not equal: %q != %q\n", n.MovieTitle, other.MovieTitle)
		return false
	}

	// Compare ReleaseDate
	if n.ReleaseDate.Format("2006") != other.ReleaseDate.Format("2006") {
		fmt.Printf("ReleaseDate not equal: %s != %s\n", n.ReleaseDate.Format("2006"), other.ReleaseDate.Format("2006"))
		return false
	}

	// Compare normalized Genres
	if !reflect.DeepEqual(normalizeStringSlice(n.Genres), normalizeStringSlice(other.Genres)) {
		fmt.Println("Genres not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n",
			normalizeStringSlice(n.Genres),
			normalizeStringSlice(other.Genres),
		)
		return false
	}

	// Compare normalized Cast
	if !reflect.DeepEqual(normalizeMaybeClozeSlice(n.Cast), normalizeMaybeClozeSlice(other.Cast)) {
		fmt.Println("Cast not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n",
			normalizeMaybeClozeSlice(n.Cast),
			normalizeMaybeClozeSlice(other.Cast),
		)
		return false
	}

	// Compare normalized Director
	if !reflect.DeepEqual(normalizeMaybeClozeSlice(n.Director), normalizeMaybeClozeSlice(other.Director)) {
		fmt.Println("Director not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n",
			normalizeMaybeClozeSlice(n.Director),
			normalizeMaybeClozeSlice(other.Director),
		)
		return false
	}

	// Compare normalized Composer
	if !reflect.DeepEqual(normalizeMaybeClozeSlice(n.Composer), normalizeMaybeClozeSlice(other.Composer)) {
		fmt.Println("Composer not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n",
			normalizeMaybeClozeSlice(n.Composer),
			normalizeMaybeClozeSlice(other.Composer),
		)
		return false
	}

	// Compare normalized Writer
	if !reflect.DeepEqual(normalizeMaybeClozeSlice(n.Writer), normalizeMaybeClozeSlice(other.Writer)) {
		fmt.Println("Writer not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n",
			normalizeMaybeClozeSlice(n.Writer),
			normalizeMaybeClozeSlice(other.Writer),
		)
		return false
	}

	// Compare normalized Cinematographer
	if !reflect.DeepEqual(normalizeMaybeClozeSlice(n.Cinematograper), normalizeMaybeClozeSlice(other.Cinematograper)) {
		fmt.Println("Cinematograper not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n",
			normalizeMaybeClozeSlice(n.Cinematograper),
			normalizeMaybeClozeSlice(other.Cinematograper),
		)
		return false
	}

	// Compare Pictures (assuming they need exact order comparison)
	if !reflect.DeepEqual(n.Pictures, other.Pictures) && len(n.Pictures) != 0 || len(other.Pictures) != 0 {
		fmt.Println("Pictures not equal")
		fmt.Printf("Got: %#v\nWant: %#v\n", n.Pictures, other.Pictures)
		return false
	}

	return true
}
