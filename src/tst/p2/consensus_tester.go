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
		original := append([]byzantine.General{}, (*permutations)[index]...)
		
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
			
			for _, gen := range permutation {
				if gen.ID() == 0 {
					go func(g byzantine.General) {
						defer func() {done <- true}()
						g.OMLeader(tree, initial)
					}(gen)
				}else{
					go func(g byzantine.General, resultCorrect bool) {
						defer func() {done <- resultCorrect}()
						result := g.OM(tree)
						
						mutex.Lock()
						defer mutex.Unlock()
						
						if !g.Traitor() {
							if validResult {
								if result != consensus {
									fmt.Printf("\tFAILURE: Lieutenant %d (loyal) decided to %s while consensus was %s.\n", g.ID(), orderToString(result), orderToString(consensus))
									return
								}
							}else{
								validResult = true
								consensus = result
							}
						}
						resultCorrect = true
					}(gen, false)
				}
			}
			
			success := true
			for range permutation {
				success = success && (<- done)
			}
			totalSuccess = totalSuccess && success
			if success {
				if validResult {
					fmt.Printf("Test (n = %d, m = %d, permute = ", totalGenerals, m)
					for _, gen := range permutation {
						if gen.Traitor() {
							fmt.Print("T")
						}else{
							fmt.Print(".")
						}
					}
					fmt.Printf(") passed.  Consensus: %s.\n", orderToString(consensus))
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
	
	if gens, err := strconv.Atoi(os.Args[1]); err == nil && gens >= 2 {
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
		if gens < 2 {
			fmt.Println("Error: Read in a value for number_of_generals that was less than 2.")
		}else{
			fmt.Println("Error: Read in a value for number_of_generals that wasn't an integer.")
		}
	}
}