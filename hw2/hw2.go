package main

import (
  "fmt"
)

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
returns the 
n: 
*/
func CollatzLength(n int) int {

}

func main() {
  fmt.Println("count: ", collatz(13, 0))
}`
