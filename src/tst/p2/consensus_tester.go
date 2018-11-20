package main

import (
	"os"
	"fmt"
	"sync"
	"time"
	"strconv"
	"math/rand"
	"lib/p2/byzantine"
)

//Causes a stack overflow for any number of traitors over 0.  Guess I can't test this way...
/*func recursivelyPermute(index int, initial, total, traitors uint, permutations *[][]byzantine.General) {
	if traitors == 0 {
		for i := initial; i < total; i++ {
			(*permutations)[index] = append((*permutations)[index], byzantine.InitGeneral(i, traitors != 0))
		}
	}else{
		(*permutations)[index] = append((*permutations)[index], byzantine.InitGeneral(initial, false))
		recursivelyPermute(index, initial + 1, total, traitors, permutations)
		
		(*permutations) = append((*permutations), append((*permutations)[index][:len(*permutations) - 1], byzantine.InitGeneral(initial, true)))
		recursivelyPermute(len((*permutations)) - 1, initial + 1, total, traitors - 1, permutations)
	}
}*/

/*func permuteGenerals(totalGenerals, traitorGenerals uint) [][]byzantine.General {
	permutations := make([][]byzantine.General, 1, 1)
	permutations[0] = []byzantine.General{}
	recursivelyPermute(0, 0, totalGenerals, traitorGenerals, &permutations)
	return permutations
}*/

//Generates a random permutation of a number of generals and traitors (precondition: generals >= traitors).
func permuteGenerals(totalGenerals, traitorGenerals uint) []byzantine.General {
	permutation := make([]byzantine.General, 0, 1)
	id, generals, traitors := uint(0), totalGenerals, traitorGenerals
	for traitors < generals {
		if rand.Intn(2) == 0 && traitors > 0 {
			permutation = append(permutation, byzantine.InitGeneral(id, true))
			traitors--
		}else{
			permutation = append(permutation, byzantine.InitGeneral(id, false))
		}
		generals--
		id++
	}
	for i := uint(0); i < traitors; i++ {
		permutation = append(permutation, byzantine.InitGeneral(id, true))
		id++
	}
	return permutation
}

func orderToString(o byzantine.Order) string {
	consensusString := "?"
	if o == byzantine.Retreat {
		consensusString = "Retreat"
	}else if o == byzantine.Attack {
		consensusString = "Attack"
	}
	return consensusString
}

func testScenario(totalGenerals uint, initial byzantine.Order) {
	for m := uint(0); m <= (totalGenerals - 1) / 3; m++ {
		tree := byzantine.InitConsensusTree(m, totalGenerals)
		done := make(chan bool)
		
		//for _, permutation := range permuteGenerals(totalGenerals, m) {
		permutation := permuteGenerals(totalGenerals, m)
		mutex := sync.Mutex{}
		validResult := !permutation[0].Traitor()
		consensus := initial
		
		for i, gen := range permutation {
			if i == 0 {
				go func(g byzantine.General) {
					defer func() {done <- true}()
					g.OMLeader(tree, initial)
				}(gen)
			}else{
				go func(g byzantine.General) {
					result, valid := g.OM(tree)
					
					mutex.Lock()
					defer mutex.Unlock()
					
					if valid {
						if !g.Traitor() {
							if validResult {
								if result != consensus {
									fmt.Printf("TEST (n = %d, m = %d, ct = %t) FAILED\n\tLieutenant %d (loyal) not reach consensus!\n", totalGenerals, m, permutation[0].Traitor(), g.ID())
									fmt.Printf("\tDecided to %s while consensus was %s.\n", orderToString(result), orderToString(consensus))
									done <- false
									return
								}
							}else{
								validResult = true
								consensus = result
							}
						}
					}else{
						fmt.Printf("TEST (n = %d, m = %d, ct = %t) FAILED\n\tLieutenant %d returned invalid result!\n", totalGenerals, m, permutation[0].Traitor(), g.ID())
						done <- false
						return
					}
					done <- true
				}(gen)
			}
		}
		
		success := true
		for range permutation {
			success = success && (<- done)
		}
		if success {
			if validResult {
				fmt.Printf("Test (n = %d, m = %d, ct = %t) passed.  Consensus: %s.\n", totalGenerals, m, permutation[0].Traitor(), orderToString(consensus))
			}
		}
		//}
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Args) < 2 {
		fmt.Printf("Insufficient arguments.\n\tTry: %s number_of_generals\n", os.Args[0])
		return
	}
	
	if gens, err := strconv.Atoi(os.Args[1]); err == nil {
		fmt.Println("----------{!!Attack!!}----------")
		testScenario(uint(gens), byzantine.Attack)
		fmt.Println()
		
		fmt.Println("----------{!!Retreat!!}----------")
		testScenario(uint(gens), byzantine.Retreat)
		fmt.Println()
	}else{
		fmt.Println("Error: Read in a value that wasn't an integer.")
	}
}