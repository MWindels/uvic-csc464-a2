#include <mutex>
#include <vector>
#include <thread>
#include <cstdlib>
#include <iostream>
#include <functional>
#include "src/lib/p3/double_ref_counter.hpp"

std::mutex out_mu;

void obtainer(int i, lockfree::double_ref_counter<int>& counter){
	auto value = *(counter.obtain());
	
	std::unique_lock lk(out_mu);
	std::cout << i << ":" << value << "\n";
}

void replacer(int i, lockfree::double_ref_counter<int>& counter){
	counter.replace(i);
}

int main(){
	lockfree::double_ref_counter<int> reference(0);
	std::vector<std::thread> obtainers;
	std::vector<std::thread> replacers;
	int obtns = 100, rplcrs = 10;
	
	for(int os = 0, rs = 0; os < obtns || rs < rplcrs;){
		if(os < obtns && rs < rplcrs){
			if(std::rand() % 2){
				obtainers.push_back(std::thread(obtainer, os, std::ref(reference)));
				++os;
			}else{
				replacers.push_back(std::thread(replacer, rs, std::ref(reference)));
				++rs;
			}
		}else if(os < obtns){
			obtainers.push_back(std::thread(obtainer, os, std::ref(reference)));
			++os;
		}else{
			replacers.push_back(std::thread(replacer, rs, std::ref(reference)));
			++rs;
		}
	}
	
	for(auto i = obtainers.begin(); i != obtainers.end(); ++i){
		if(i->joinable()){
			i->join();
		}
	}
	for(auto i = replacers.begin(); i != replacers.end(); ++i){
		if(i->joinable()){
			i->join();
		}
	}
	
	return 0;
}