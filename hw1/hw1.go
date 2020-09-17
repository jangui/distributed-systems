package main

import (
  "fmt"
  "os"
  "bufio"
  "container/list"
  "errors"
)

type Actor struct {
  name string
  score int
  movies []*Movie
  linkedBy *Movie
  inQueue bool
}

type Movie struct {
  name string
  score int
  cast []*Actor
  lowestScorer *Actor
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
      curMovie = &Movie{name: line, score: -1}
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
        curActor = &Actor{name: line, score: -1, movies: []*Movie{curMovie}}
        actors[line] = curActor
      }
      // add actor to cast in current movie
      curMovie.cast = append(curMovie.cast, curActor)
    }
  }
  f.Close()
  return nil
}

func score(actorQ *list.List) {
  for actorQ.Len() != 0 {
    // deque actor
    elem := actorQ.Front()
    actorQ.Remove(elem)
    actor := elem.Value.(*Actor)
    actor.inQueue = false

    // score all of actor's movies
    // scoreing movies adds move actors to queue
    for _, movie := range actor.movies {
      scoreMovie(movie, actor, actorQ)
    }
  }
}

// update score of movies for a given actor (scorer)
func scoreMovie(movie *Movie, scorer *Actor, actorQ *list.List) {
  // if no score, score
  if movie.score == -1 {
    movie.score = 1 + scorer.score
    movie.lowestScorer = scorer
    // update score of all actors and add to queue
    for _, actor := range movie.cast {
      scoreActor(movie, actor, actorQ)
    }
  } else if movie.score - 1 > scorer.score {
    // update score if better
    movie.score = scorer.score + 1
    movie.lowestScorer = scorer
    // update score of all actors and add to queue
    for _, actor := range movie.cast {
      scoreActor(movie, actor, actorQ)
    }

  }
}

// update actors score based on movie (scorer)
func scoreActor(scorer *Movie, actor *Actor, actorQ *list.List) {
  // if no score, update
  if actor.score == -1 {
    actor.score = scorer.score
    actor.linkedBy = scorer
    // add to queue
    if actor.inQueue == false {
      actorQ.PushBack(actor)
      actor.inQueue = true
    }
  } else if scorer.score < actor.score {
    // if movie score less than ours, update
    actor.score = scorer.score
    actor.linkedBy = scorer
    // add to queue
    if actor.inQueue == false {
      actorQ.PushBack(actor)
      actor.inQueue = true
    }

  }
}

func lookup(name string, actors map[string]*Actor) (*Actor, error) {
  if _, ok := actors[name]; ok {
    return actors[name], nil
  } else {
    return nil, errors.New("Unknown actor name")
  }
}

func displayScore(actor *Actor) {
  if actor.score == -1 {
    fmt.Println("Infinite KBN\n")
    return
  }

  curr := actor
  for curr.linkedBy != nil {
    movie := curr.linkedBy
    fmt.Println(curr.name, "was in", movie.name, "with", movie.lowestScorer.name)
    curr = movie.lowestScorer
  }

  fmt.Println("Found with KBN of", actor.score, "\n")
}

func main() {
  movies := map[string]*Movie{}
  actors := map[string]*Actor{}
  processCastFile("cast.txt", movies, actors)

  // set all scores
  kevin := actors["Kevin Bacon"]
  kevin.score = 0
  queue := list.New()
  queue.PushBack(kevin)
  score(queue)

  for {
    // get name from stdin
    if name, err := getName(); err != nil {
      fmt.Println(err)
    } else {
      // lookup actor
      if name == "" { return }
      if actor, err := lookup(name, actors); err != nil {
        fmt.Println(err)
      } else {
        // display score
        displayScore(actor)
      }
    }
  }
}
