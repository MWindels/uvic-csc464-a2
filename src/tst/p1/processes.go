package main

import (
	"os"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
	"lib/p1/vector_clock"
)

type encodedEvent struct {
	Wait int	`json:"wait"`
	Prev []int	`json:"prev"`
	Next []int	`json:"next"`
}

type event struct {
	wait time.Duration
	prev [](chan vector_clock.VectorClock)
	next [](chan vector_clock.VectorClock)
}

func unmarshalProcs(path string) [][]event {
	file, openErr := os.Open(path)
	if openErr != nil {
		panic(openErr)
	}
	defer file.Close()
	
	bytes, readErr := ioutil.ReadAll(file)
	if readErr != nil {
		panic(readErr)
	}
	
	var encodedProcs [][]encodedEvent
	if unmarshalErr := json.Unmarshal(bytes, &encodedProcs); unmarshalErr != nil {
		panic(unmarshalErr)
	}
	
	var procs [][]event
	chans := make(map[int](chan vector_clock.VectorClock))
	for _, p := range encodedProcs {
		procs = append(procs, []event{})
		for _, e := range p {
			prevs := make([](chan vector_clock.VectorClock), 0, 1)
			for _, prev := range e.Prev {
				if _, valid := chans[prev]; !valid {
					chans[prev] = make(chan vector_clock.VectorClock)
				}
				prevs = append(prevs, chans[prev])
			}
			nexts := make([](chan vector_clock.VectorClock), 0, 1)
			for _, next := range e.Next {
				if _, valid := chans[next]; !valid {
					chans[next] = make(chan vector_clock.VectorClock)
				}
				nexts = append(nexts, chans[next])
			}
			procs[len(procs) - 1] = append(procs[len(procs) - 1], event{time.Duration(e.Wait) * time.Millisecond, prevs, nexts})
		}
	}
	
	return procs
}

func process(id int, events []event, done chan<- bool) {
	clock := vector_clock.InitVectorClock(id)
	
	fmt.Printf("(P%d : INIT) %s\n", id, clock.String())
	for i, e := range events {
		for _, prev := range e.prev {
			clock.Merge(<- prev)
		}
		
		time.Sleep(e.wait)
		clock.Increment()
		fmt.Printf("(P%d : E%d) %s\n", id, i + 1, clock.String())
		
		for _, next := range e.next {
			next <- clock
		}
	}
	
	done <- true
}

func main() {
	done := make(chan bool)
	
	if len(os.Args) < 2 {
		fmt.Printf("Insufficient arguments.\n\tTry: %s process_file\n", os.Args[0])
		return
	}
	
	procs := unmarshalProcs(os.Args[1])
	
	for i := 0; i < len(procs); i++ {
		go process(i + 1, procs[i], done)
	}
	
	for i := 0; i < len(procs); i++ {
		<- done
	}
}