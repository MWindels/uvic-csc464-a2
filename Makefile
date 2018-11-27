p3_make: src/tst/p3/ref_tester.cpp
	g++ -Wall -std=c++17 -Isrc src/tst/p3/ref_tester.cpp -pthread -latomic -march=native -o ref_tester