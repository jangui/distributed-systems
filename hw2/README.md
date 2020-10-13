# Submission for hw2

## collatz.go
This program calculates the number with longest collatz sequence in the range of 1 to 10000000
Its output is the number with the highlest collatz sequence in the range
This program is multithreaded using a producer thread, 5 workers and 1 collector.

The producer inserts values in the range into a channel
The workers get numbers from that channel and calculate their collatz length
The workers then send that info in a struct into a channel that the collect thread will use
The collect thread reads structs from the channel filled by the worker and keeps track of the number with the longest collatz sequence

run using `go run collatz.go`
or compile with `go build collatz.go`

## collatz-caching.go
This program works the same as collatz.go however utilizes a cache.
The cache is a global map variable that the workers use to check if part of the sequence for their given number is already solved, and can use the cached value instead of recursing over again
To accesss the cache safetly, a read write lock is used

run using `go run collatz-caching.go`
or compile with `go build collatz-caching.go`

