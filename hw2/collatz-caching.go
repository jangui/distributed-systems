package main

import (
    "fmt"
    "sync"
)

type Cache struct {
    data map[int]int  // map of numbers to collatz sequence length
    lock sync.RWMutex // RW Lock for accessing data
}

/*
global cache var used by workers
data: map from numbers to their collatz length
    key: number
    value: length of collatz sequqnce for key
lock: read write lock used to access data
*/
var cache = Cache{}

/*
struct used to hold number and its collatz length
*/
type Collat struct {
    number int
    length int // length of collatz sequence for number
}


/*
caculates collatz len for number n
relies on helper function for recursively solving sequence
uses global cache to aviod uneccesary recursion
n: number we want to know collatz length of
return: collatz length of n
*/
func collatz(n int) int {
    //  check if val in cache
    cache.lock.RLock()
    if collatzLen, ok := cache.data[n]; ok {
        cache.lock.RUnlock()
        return collatzLen
    }
    cache.lock.RUnlock()

    // knowning val not in cache, add it to our cache
    length := collatzHelper(n, 0)
    cache.lock.Lock()
    cache.data[n] = length
    cache.lock.Unlock()
    return length
}

/*
recursively calculate the length of the collatz sequence
checks global cache on each recursion
this function should only be called recursively or from the collatz func
n: input number we want to calculate length of sequence
count: var used to recursively keep track of length of collatz sequence
return: length of collatz sequnce for n
*/
func collatzHelper(n int, count int) int {
    // check if number in global cache var
    // dont check if we're on first iteration
    // (already checked in collatz func)
    if count != 0 {
        cache.lock.RLock()
        if collatzLen, ok := cache.data[n]; ok {
            cache.lock.RUnlock()
            return collatzLen + count
        }
        cache.lock.RUnlock()
    }
    // base case
    if n == 1 {
        return count + 1
    } else if (n % 2 == 0) {
        // even
        count = collatzHelper(n / 2, count + 1)
    } else {
        // odd
        count = collatzHelper(3*n+1, count + 1)
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
        length := collatz(num)
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
    maxCollat.number = 0
    maxCollat.length = 0
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
    numbers := make(chan int, workers*100)
    finished := make(chan Collat, workers*500)

    // init map in global cache
    cache.data = make(map[int]int)

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
