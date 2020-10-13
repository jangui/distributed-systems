package main

import (
    "fmt"
)

/*
struct used to hold number and its collatz length
*/
type Collat struct {
    number int
    length int // length of collatz sequence for number
}

/*
recursively calculate the length of the collatz sequence
n: input number we want to calculate length of sequence
count: var used to recursively keep track of length of collatz sequence
return: length of collatz sequnce for n
to call this func pass in your desired number as n and count as 0.
ex: collatz(13, 0) -> returns 10 (the length of the collatz sequence starting from 13)
*/
func collatz(n int, count int) int {
    if n == 1 {
        // base case
        return count + 1
    } else if (n % 2 == 0) {
        // even
        count = collatz(n / 2, count + 1)
    } else {
        // odd
        count = collatz(3*n+1, count + 1)
    }
    return count
}

/*
sequentially puts ints into channel
max: highest number that will get put in channel
numbers: channel of ints that numbers will get put into
*/
func generate(max int, numbers chan int) {
    for i:=1; i<max+1; i++ {
        numbers <- i
    }
}

/*
function used by worker thread
calculates the collat length of number taken from generators channel
puts the results into a 'Collat' struct and puts it into channel for collector thread
numbers: channel of ints that gets filled by generator thread
finished: channel of 'Collat' structs filled by worker, emptied by generator
*/
func worker(numbers chan int, finished chan Collat) {
    for {
        num := <-numbers
        length := collatz(num, 0)
        finished <- Collat{num, length}
    }
}

/*
collecting functions that stores and returns the 'Collat' struct with longest collatz sequence
total: total amount numbers we are expecting to collect
finished: channel of 'Collat' structs
return: 'Collat' struct w/ longest collatz sequnece (largest length attribute)
*/
func collect(total int, finished chan Collat) Collat {
    var maxCollat Collat
    for i:=0; i<total; i++ {
        currCollat := <-finished
        if currCollat.length > maxCollat.length {
            maxCollat = currCollat
        }
    }
    return maxCollat
}

/*
finds the number with the longest collatz in range 1 to n
n: upper bound on range 
return: 'Collat' struct that connectains the number with the longest collatz sequence
*/
func CollatzLength(n int) Collat {
    workers := 5
    // channel buffer sizes were chosen through repeated testing
    // current sizes gave the fastest preformance
    numbers := make(chan int, workers*100)
    finished := make(chan Collat, workers*500)

    // start genator thread
    go generate(n, numbers)

    // start worker threads
    for i:=0; i<workers; i++ {
        go worker(numbers, finished)
    }

    // use current thread as collector thread and return value from collect func
    return collect(n, finished)

}

func main() {
    maxCollat := CollatzLength(10000000)
    fmt.Println("Longest sequence starts at", maxCollat.number, "length", maxCollat.length)
}
