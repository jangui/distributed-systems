package main

import "os"
import "fmt"
import "mapreduce"
//import "strings"
import "strconv"
import "unicode"

// Map function takes a chunk of data from the
// input file and breaks it into a sequence
// of key/value pairs
func Map(value string) []mapreduce.KeyValue {
    output := []mapreduce.KeyValue{}
    var wordStart int
    // iterate over characters
    // keep track of start and end of word
    // ignore characters that arent letters
	for i, char := range value {
        if unicode.IsLetter(char) {
            continue
        } else if wordStart == i {
            wordStart += 1
        } else {
            keyVal := mapreduce.KeyValue{Key: value[wordStart:i], Value: "1"}
            output = append(output, keyVal)
            wordStart = i+1
        }
	}
    return output
}

// called once for each key generated by Map, with a list
// of that key's associate values. should return a single
// output value for that key
func Reduce(key string, values []string) string {
    return strconv.Itoa(len(values))
}

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("%s: Invalid invocation\n", os.Args[0])
	} else if os.Args[1] == "master" {
		if os.Args[3] == "sequential" {
			mapreduce.RunSingle(5, 3, os.Args[2], Map, Reduce)
		} else {
			mr := mapreduce.MakeMapReduce(5, 3, os.Args[2], os.Args[3])
			// Wait until MR is done
			<-mr.DoneChannel
		}
	} else if os.Args[1] == "worker" {
		mapreduce.RunWorker(os.Args[2], os.Args[3], Map, Reduce, 100)
	} else {
		fmt.Printf("Unexpected input")
	}
}
