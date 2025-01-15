package tmdbankigenerator

import (
	"cmp"
	"encoding/json"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

var Cast = []string{"Tim Burton", "Nicolas Winding Refn", "Danny Elfman", "Nikolaj Lie Kaas", "Hans Zimmer", "David Lynch", "Christopher Nolan", "Mads Mikkelsen", "Pilou Asbæk", "Bill Murray", "Scarlett Johansson", "Warwick Davis", "Thomas Vinterberg", "Steven Spielberg", "Quentin Tarantino", "Ron Howard", "Laurence Fishburne"}

// These people are clozed but their movies arent added to the list
var ExtraCast = []string{"Samuel L. Jackson", "Willem Dafoe", "Johnny Depp", "Shia LaBeouf", "Tom Hanks", "Gary Oldman", "Tom Cruise", "Danny DeVito", "Morgan Freeman", "Matt Damon", "Brad Pitt", "George Clooney", "Anne Hathaway", "Bruce Willis", "Mark Ruffalo", "Ben Affleck", "Stellan Skarsgård", "Robert De Niro", "Keira Knightley", "Robin Williams", "Jim Carrey", "Orlando Bloom", "Natalie Portman", "Matthew McConaughey"}

/*var Cast = []string{"Emily Blunt", "Wes Anderson", "Tim Burton", "Danny Elfman", "Hans Zimmer", "Mel Gibson", "Sydney Sweeney", "Jason Statham", "Keanu Reeves", "Tom Cruise", "David Lynch", "Timothée Chalamet", "James Woods", "Jenna Ortega", "Leonardo DiCaprio", "Gary Oldman", "Jake Gyllenhaal", "Rebecca Ferguson", "Nicole Kidman", "이병헌/Lee Byung-hun", "Margaret Qualley", "Ana de Armas", "Alexandra Daddario", "Pedro Pascal", "Florence Pugh", "Dwayne Johnson", "Ben Whishaw", "Tom Hardy", "Nicolas Cage", "Scarlett Johansson", "Mads Mikkelsen", "Tom Hanks", "Jackie Chan", "Margot Robbie", "Sylvester Stallone", "Dolph Lundgren", "Elizabeth Banks", "Ryan Gosling", "Nicolas Winding Refn", "Chris Evans", "Cate Blanchett", "Charlize Theron", "Vin Diesel", "Jean-Claude Van Damme", "Jim Carrey", "Anne Hathaway", "Katheryn Winnick", "Ryan Reynolds", "Jason Momoa", "Brad Pitt", "Tom Hiddleston", "Ben Affleck", "Zendaya", "Cillian Murphy", "Kelly Reilly", "Kevin Bacon", "Robert Downey Jr.", "Robert De Niro", "Jean Reno", "Johnny Depp", "Angelina Jolie", "Henry Cavill", "Sandra Bullock", "Linda Cardellini", "John Goodman", "Zoe Saldaña", "Eddie Redmayne", "Emma Watson", "Andrew Garfield", "Jackie Sandler", "Harrison Ford", "Idris Elba", "Gerard Butler", "Halle Berry", "Bill Skarsgård", "Jon Hamm", "William Fichtner", "Mila Kunis", "Amanda Seyfried", "Dakota Fanning", "Rosamund Pike", "Kevin Costner", "J.K. Simmons", "Ralph Fiennes", "Al Pacino", "Demi Moore", "Brie Larson", "Kate Beckinsale", "Donal Logue", "Mckenna Grace", "Anya Taylor-Joy", "Matt Damon", "Kristen Wiig", "Tom Holland", "Brian Cox", "Mark Wahlberg", "Dakota Johnson", "גל גדות/Gal Gadot", "Frank Welker", "Chris Pratt", "Russell Crowe", "Christopher Nolan", "Joey King", "Kevin Hart", "Harris Dickinson", "Bruce Willis", "Matthew McConaughey", "Robert De Niro", "Tom Hanks", "Leonardo DiCaprio", "Morgan Freeman", "Johnny Depp", "Al Pacino", "Tom Cruise", "Mads Mikkelsen", "Ron Howard", "Heath Ledger", "Joaquin Phoenix", "George Clooney", "Russell Crowe", "Nicolas Cage", "Brad Pitt", "Robert Downey Jr.", "Jim Carrey", "Christoph Waltz", "Willem Dafoe", "Ben Stiller", "Liam Neeson", "Benedict Cumberbatch", "Christian Bale", "Chris Hemsworth", "Jake Gyllenhaal", "Keanu Reeves", "Hugh Jackman", "Daniel Radcliffe", "Daniel Craig", "Ben Affleck", "Chris Evans", "Will Ferrell", "Ryan Reynolds", "Paul Rudd", "Robert Pattinson", "Mark Ruffalo", "Kevin Spacey", "Bill Murray", "Jon Favreau", "Kevin Hart", "Natalie Portman", "Bruce Willis", "Harrison Ford", "Denzel Washington", "Samuel L. Jackson", "Robin Williams", "Sandra Bullock", "Danny DeVito", "Robert Downey Jr.", "Tom Hanks", "Anne Hathaway", "Julia Roberts", "Scarlett Johansson", "Laurence Fishburne", "Nikolaj Lie Kaas", "Pilou Asbæk", "George Lucas", "Ron Howard", "Thomas Vinterberg", "Susanne Bier", "Steven Spielberg", "Quentin Tarantino", "Martin Scorsese", "James Cameron", "David Lynch", "Woody Allen"}
 */
func GetCastIDs() ([]int, []int) {
	popularPeopleJson, err := os.ReadFile("person_ids_01_11_2025.json")
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to read person_ids file"))
	}
	popularMoviesJson, err := os.ReadFile("movie_ids_01_10_2025.json")
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to read movie_ids file"))
	}

	movies := strings.Replace(string(slices.Concat([]byte{'['}, popularMoviesJson, []byte{']'})), "}\n", "},\n", -1)
	movies = strings.Replace(movies, ",\n]", "\n]", 1)
	people := strings.Replace(string(slices.Concat([]byte{'['}, popularPeopleJson, []byte{']'})), "}\n", "},\n", -1)
	people = strings.Replace(people, ",\n]", "\n]", 1)

	if err := json.Unmarshal([]byte(movies), &PopularMovies); err != nil {
		log.Fatalf("Failed to unmarshal popular movies json: %s", err)
	}
	if err := json.Unmarshal([]byte(people), &PopularPeople); err != nil {
		log.Fatalf("Failed to unmarshal popular people json: %s", err)
	}

	slices.SortFunc(PopularPeople, func(a, b PopularPerson) int {
		return cmp.Compare(a.Popularity, b.Popularity)
	})

	slices.SortFunc(PopularMovies, func(a, b PopularMovie) int {
		return cmp.Compare(a.Popularity, b.Popularity)
	})

	// Create a map for quick lookups by name
	personMap := make(map[string]int)
	for _, person := range PopularPeople {
		personMap[strings.ToLower(person.Name)] = person.ID
	}

	var cast []int
	for _, name := range Cast {
		var (
			realName = name
			enName   = name
		)

		if result := strings.Split(name, "/"); len(result) > 1 {
			realName = result[0]
			enName = result[1]
		}

		id, found := personMap[strings.ToLower(realName)]
		if !found {
			log.Fatalf("could not find %s (%s)", enName, realName)
		}
		cast = append(cast, id)
	}

	var extraCast []int
	for _, name := range ExtraCast {
		var (
			realName = name
			enName   = name
		)

		if result := strings.Split(name, "/"); len(result) > 1 {
			realName = result[0]
			enName = result[1]
		}

		id, found := personMap[strings.ToLower(realName)]
		if !found {
			log.Fatalf("could not find %s (%s)", enName, realName)
		}
		extraCast = append(extraCast, id)
	}

	return cast, extraCast
}
