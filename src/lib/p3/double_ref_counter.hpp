#ifndef DOUBLE_REF_COUNTER_H_INCLUDED
#define DOUBLE_REF_COUNTER_H_INCLUDED

#include <atomic>

namespace lockfree {

template <class T>
class double_ref_counter {
public:
	
	//Types
	using value_type = T;
	class counted_ptr;
	
	//Constructors/Destructor
	double_ref_counter();
	double_ref_counter(const double_ref_counter&) = delete;
	double_ref_counter(double_ref_counter&&) = delete;	//Might not be unreasonable to move it...
	~double_ref_counter();
	
	//Assignment Operators
	double_ref_counter& operator=(const double_ref_counter&) = delete;
	double_ref_counter& operator=(double_ref_counter&&) = delete;
	
	//Operations
	counted_ptr obtain();
	//swap
	//try_swap
	
private:
	
	//Types
	class internal_counter;
	struct external_counter {	//This may be too large for lock-free operations on some architectures.
		internal_counter* internal_ptr;
		int counter;		//Padding after this should assure that this struct is lock-free.
	};
	
	//Members.
	std::atomic<external_counter> front_end;
	
};

template <class T>
double_ref_counter<T>::double_ref_counter() {
	front_end.store(external_counter{nullptr, 0}, std::memory_order_relaxed);
}

template <class T>
double_ref_counter<T>::~double_ref_counter() {
	//get the internal counter, and destroy it if no one's looking at it
}

template <class T>
typename double_ref_counter<T>::counted_ptr double_ref_counter<T>::obtain() {
	external_counter prev = front_end.load(std::memory_order_relaxed), next;		//It's fine to use a relaxed ordering here, since there is no shared memory that is synchronized using this atomic (see https://gcc.gnu.org/wiki/Atomic/GCCMM/AtomicSync).
	do{
		next = prev;	//Note, if CAS fails below, prev will be set to the actual value of front_end.
		++next.count;
	}while(!front_end.compare_exchange_weak(prev, next), std::memory_order_relaxed);	//We only need weak here, since it'll just repeat on a spurious failure.
	return counted_ptr(next.internal_ptr);
}

template <class T>
class double_ref_counter<T>::counted_ptr {
public:
	
	//Constructors/Destructor
	counted_ptr(internal_counter* ptr) : internal_ptr(ptr) {}
	counted_ptr(const counted_ptr&) = delete;
	counted_ptr(counted_ptr&& other) : internal_ptr(other.internal_ptr) {other.internal_ptr = nullptr;}	//Could this cause trouble?
	~counted_ptr() {if(!internal_ptr){release();}}
	
	//Assignment Operators
	counted_ptr& operator=(const counted_ptr&) = delete;
	counted_ptr& operator=(counted_ptr&&);
	
	//Accessors
	
	
	//Mutators
	
	
private:
	
	//Private Functions
	void release();
	
	//Members
	internal_counter* internal_ptr;
	
};

template <class T>
typename double_ref_counter<T>::counted_ptr& double_ref_counter<T>::counted_ptr::operator=(counted_ptr&& other) {
	if(!internal_ptr){
		release();
	}
	internal_ptr = other.internal_ptr;
	other.internal_ptr = nullptr;
	return *this;
}

template <class T>
void double_ref_counter<T>::counted_ptr::release() {
	if(internal_ptr->counter.fetch_add(1) == -1){	//Implies the next value is 0.
		delete internal_ptr;
	}
}

template <class T>
class double_ref_counter<T>::internal_counter {
public:
	
	//Constructors/Destructor
	internal_counter();
	internal_counter(const internal_counter&) = delete;
	internal_counter(internal_counter&&) = delete;
	~internal_counter();
	
	//Assignment Operators
	internal_counter& operator=(const internal_counter&) = delete;
	internal_counter& operator=(internal_counter&&) = delete;
	
private:
	
	//Members
	value_type data;
	std::atomic<int> counter;
	
};

}

#endif