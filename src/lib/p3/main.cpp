#include <iostream>
#include "double_ref_counter.hpp"

int main(){
	lockfree::double_ref_counter<int> rc;
	
	std::cout << alignof(char) << "|" << alignof(short) << "|" << alignof(int) << "|" << alignof(long) << "|" << alignof(long long) << "\n";
	
	return 0;
}