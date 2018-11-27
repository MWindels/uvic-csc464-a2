#include <mutex>
#include <vector>
#include <thread>
#include <cstdlib>
#include <iostream>
#include <functional>
#include "lib/p3/double_ref_counter.hpp"

std::mutex out_mu;

class loud_object{
public:
	
	loud_object() : num(-1) {std::unique_lock lk(out_mu); std::cout << "(-1) Init.\n";}
	loud_object(int i) : num(i) {std::unique_lock lk(out_mu); std::cout << "(" << num << ") Init.\n";}
	loud_object(int i, int j) : num(i + j) {std::unique_lock lk(out_mu); std::cout << "(" << i << "+" << j << "=" << (i + j) << ") Init.\n";}
	loud_object(const loud_object& other) : num(other.num) {std::unique_lock lk(out_mu); std::cout << "(" << num << ") Copy Init.\n";}
	loud_object(loud_object&& other) : num(other.num) {std::unique_lock lk(out_mu); std::cout << "(" << num << ") Move Init.\n";}
	~loud_object() {std::unique_lock lk(out_mu); std::cout << "(" << num << ") Destroyed.\n";}
	
	loud_object& operator=(const loud_object& other) {num = other.num; std::unique_lock lk(out_mu); std::cout << "(" << num << ") Copy Assign.\n"; return *this;}
	loud_object& operator=(loud_object&& other) {num = other.num; std::unique_lock lk(out_mu); std::cout << "(" << num << ") Move Assign.\n"; return *this;}
	
	int get_num() const {return num;}
	void set_num(int i) {num = i;}
	
private:
	
	int num;
	
};

std::ostream& operator<<(std::ostream& out, const loud_object& lo){
	return out << lo.get_num();
}

template <class T>
void obtainer(lockfree::double_ref_counter<T>& counter){
	typename lockfree::double_ref_counter<T>::counted_ptr value = counter.obtain();
	
	//value->set_num(-2);	//This was here just to verify that my use of SFINAE in counted_ptr worked.
	//(*value).set_num(-3);	//Likewise.
	
	std::unique_lock lk(out_mu);
	std::cout << "\t[Obtain] " << *value << ".\n";
}

template <class T>
void replacer(int i, lockfree::double_ref_counter<T>& counter){
	{
		std::unique_lock lk(out_mu);
		std::cout << "\t[Replace " << i << "] Replacing...\n";
	}
	counter.replace(i);
	{
		std::unique_lock lk(out_mu);
		std::cout << "\t[Replace " << i << "] Done.\n";
	}
}

template <class T>
void try_replacer(int i, lockfree::double_ref_counter<T>& counter){
	{
		std::unique_lock lk(out_mu);
		std::cout << "\t[Try-Replace " << i << "] Trying...\n";
	}
	typename lockfree::double_ref_counter<T>::counted_ptr expected = counter.obtain();
	if(counter.try_replace(expected, i)){
		std::unique_lock lk(out_mu);
		std::cout << "\t[Try-Replace " << i << "] Success.\n";
	}else{
		std::unique_lock lk(out_mu);
		std::cout << "\t[Try-Replace " << i << "] FAILURE!\n";
	}
}

template <class T>
void test_scenario(int obtns, int rplcrs, int t_rplcrs){
	lockfree::double_ref_counter<T> reference;
	std::vector<std::thread> obtainers;
	std::vector<std::thread> replacers;
	std::vector<std::thread> try_replacers;
	
	for(int os = 0, rs = 0, t_rs = 0; os < obtns || rs < rplcrs || t_rs < t_rplcrs;){
		if(os < obtns && rs < rplcrs && t_rs < t_rplcrs){
			switch(std::rand() % 3){
			case 0:
				obtainers.push_back(std::thread(obtainer<T>, std::ref(reference)));
				os++;
				break;
			case 1:
				replacers.push_back(std::thread(replacer<T>, rs++ + t_rs, std::ref(reference)));
				break;
			case 2:
				try_replacers.push_back(std::thread(try_replacer<T>, rs + t_rs++, std::ref(reference)));
				break;
			}
		}else if(os < obtns && rs < rplcrs){
			if(std::rand() % 2){
				obtainers.push_back(std::thread(obtainer<T>, std::ref(reference)));
				os++;
			}else{
				replacers.push_back(std::thread(replacer<T>, rs++ + t_rs, std::ref(reference)));
			}
		}else if(os < obtns && t_rs < t_rplcrs){
			if(std::rand() % 2){
				obtainers.push_back(std::thread(obtainer<T>, std::ref(reference)));
				os++;
			}else{
				try_replacers.push_back(std::thread(try_replacer<T>, rs + t_rs++, std::ref(reference)));
			}
		}else if(rs < rplcrs && t_rs < t_rplcrs){
			if(std::rand() % 2){
				replacers.push_back(std::thread(replacer<T>, rs++ + t_rs, std::ref(reference)));
			}else{
				try_replacers.push_back(std::thread(try_replacer<T>, rs + t_rs++, std::ref(reference)));
			}
		}else if(os < obtns){
			obtainers.push_back(std::thread(obtainer<T>, std::ref(reference)));
			os++;
		}else if(rs < rplcrs){
			replacers.push_back(std::thread(replacer<T>, rs++ + t_rs, std::ref(reference)));
		}else if(t_rs < t_rplcrs){
			try_replacers.push_back(std::thread(try_replacer<T>, rs + t_rs++, std::ref(reference)));
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
	for(auto i = try_replacers.begin(); i != try_replacers.end(); ++i){
		if(i->joinable()){
			i->join();
		}
	}
}

int main(int argc, char *argv[]){
	std::srand(std::time(0));
	
	if(argc < 4){
		std::cerr << "Insufficient arguments:\n\tTry: " << argv[0] << " obtainers replacers try_replacers\n";
		return -1;
	}
	
	test_scenario<const loud_object>(std::atoi(argv[1]), std::atoi(argv[2]), std::atoi(argv[3]));
	return 0;
}