package main

import "fmt"
import "os"
import "bufio"

type Actor struct {
  name string
  score int
  movies []*Movie
}

type Movie struct {
  name string
  score int
  cast []*Actor
}

// ask user for actor name
func getName() (string, error) {
  fmt.Print("Enter actor name: ")
  line, err := bufio.NewReader(os.Stdin).ReadString('\n')
  if err != nil {
    return "", err
  }
  line = line[:len(line)-1]
  return line, nil
}

// process input cast file
// populates map of actors and movies
func processCastFile(
    filename string,
    movies map[string]*Movie,
    actors map[string]*Actor) error {
  // open file
  f, err := os.Open(filename)
  if err != nil {
    return err
  }
  // iterate over each line
  newLine := true
  var curMovie *Movie
  scanner := bufio.NewScanner(f)
  for scanner.Scan() {
    line := scanner.Text()
    if line == "" {
      newLine = true
      continue
    }
    var curActor *Actor
    if newLine {
      // add line as movie
      curMovie = &Movie{name: line, cast: []*Actor{}}
      movies[line] = curMovie
      newLine = false
    } else {
      // line is actor's name
      // check if actor already in map
      if _, ok := actors[line]; ok {
        // append current movie to actor
        curActor = actors[line]
        curActor.movies = append(curActor.movies, curMovie)
      } else {
        // add actor to actors map
        curActor = &Actor{name: line, movies: []*Movie{curMovie}}
        actors[line] = curActor
      }
      // add actor to cast in current movie
      curMovie.cast = append(curMovie.cast, curActor)
    }
  }
  f.Close()
  return nil
}


func main() {
  movies := map[string]*Movie{}
  actors := map[string]*Actor{}
  processCastFile("cast.txt", movies, actors)

  /*
  for _, movie := range movies {
    fmt.Println(movie.name)
    fmt.Println("\tcast:")
    for _, actor := range movie.cast {
      fmt.Println("\t\t", actor.name)
    }
  }

  for _, actor := range actors {
    fmt.Println(actor.name)
    fmt.Println("\tfeatured in:")
    for _, movie := range actor.movies {
      fmt.Println("\t\t", movie.name)
    }
  }
  */
}

