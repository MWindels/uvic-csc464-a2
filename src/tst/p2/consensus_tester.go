package main

import (
	"os"
	"fmt"
	"sync"
	"strconv"
	"strings"
	"lib/p2/byzantine"
)

func recursivelyPermute(index int, initial, total, traitors uint, permutations *[][]byzantine.General) {
	if traitors == 0 || total - initial <= traitors {
		for i := initial; i < total; i++ {
			(*permutations)[index] = append((*permutations)[index], byzantine.InitGeneral(i, traitors != 0))
		}
	}else{
		original := (*permutations)[index]
		
		(*permutations)[index] = append(original, byzantine.InitGeneral(initial, false))
		recursivelyPermute(index, initial + 1, total, traitors, permutations)
		
		(*permutations) = append((*permutations), append(original, byzantine.InitGeneral(initial, true)))
		recursivelyPermute(len((*permutations)) - 1, initial + 1, total, traitors - 1, permutations)
	}
}

func permuteGenerals(totalGenerals, traitorGenerals uint) [][]byzantine.General {
	permutations := make([][]byzantine.General, 1, 1)
	permutations[0] = []byzantine.General{}
	recursivelyPermute(0, 0, totalGenerals, traitorGenerals, &permutations)
	return permutations
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

func testScenario(totalGenerals uint, initial byzantine.Order) bool {
	totalSuccess := true
	for m := uint(0); m <= (totalGenerals - 1) / 3; m++ {
		tree := byzantine.InitConsensusTree(m, totalGenerals)
		done := make(chan bool)
		
		for _, permutation := range permuteGenerals(totalGenerals, m) {
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
			totalSuccess = totalSuccess && success
			if success {
				if validResult {
					fmt.Printf("Test (n = %d, m = %d, ct = %t) passed.  Consensus: %s.\n", totalGenerals, m, permutation[0].Traitor(), orderToString(consensus))
				}
			}
		}
	}
	return totalSuccess
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Insufficient arguments.\n\tTry: %s number_of_generals attack|retreat\n", os.Args[0])
		return
	}
	
	if gens, err := strconv.Atoi(os.Args[1]); err == nil {
		if strings.ToLower(os.Args[2]) == "attack" {
			if testScenario(uint(gens), byzantine.Attack) {
				fmt.Println("\nAll tests passed!")
			}else{
				fmt.Println("\nA TEST FAILED!")
			}
		}else if strings.ToLower(os.Args[2]) == "retreat" {
			if testScenario(uint(gens), byzantine.Retreat) {
				fmt.Println("\nAll tests passed!")
			}else{
				fmt.Println("\nA TEST FAILED!")
			}
		}else{
			fmt.Println("Error: Read in an order value that was not attack or retreat.")
			return
		}
	}else{
		fmt.Println("Error: Read in a value for number_of_generals that wasn't an integer.")
	}
}