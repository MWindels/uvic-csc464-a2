#ifndef DOUBLE_REF_COUNTER_H_INCLUDED
#define DOUBLE_REF_COUNTER_H_INCLUDED

#include <atomic>
#include <utility>
#include <type_traits>

namespace lockfree{

/*
 * Double-counting reference counter class.
 */
template <class T>
class double_ref_counter{
public:
	
	using value_type = T;
	class counted_ptr;
	
	double_ref_counter() : front_end(external_counter{nullptr, 0}) {}
	template<class... Args> double_ref_counter(Args&&... args) : front_end(external_counter{new internal_counter(std::forward<Args>(args)...), 0}) {}
	double_ref_counter(const double_ref_counter&) = delete;
	double_ref_counter(double_ref_counter&&) = delete;				//May implement in the future, but not exactly paramount.
	~double_ref_counter();
	
	double_ref_counter& operator=(const double_ref_counter&) = delete;
	double_ref_counter& operator=(double_ref_counter&&) = delete;	//Likewise.
	
	counted_ptr obtain();
	template<class... Args> void replace(Args&&... args);
	template<class... Args> bool try_replace(Args&&... args);
	
private:
	
	struct internal_counter;
	struct external_counter{
		internal_counter* internals;
		int ex_count;
	};
	
	std::atomic<external_counter> front_end;
	
};

template <class T>
double_ref_counter<T>::~double_ref_counter(){
	external_counter prev_front_end = front_end.load();		//memory order?
	while(!front_end.compare_exchange_weak(prev_front_end, external_counter{nullptr, 0})){}	//memory order?
	if(prev_front_end.internals->in_count.fetch_sub(prev_front_end.ex_count) == prev_front_end.ex_count){	//memory order?
		delete prev_front_end.internals;
	}
}

template <class T>
typename double_ref_counter<T>::counted_ptr double_ref_counter<T>::obtain(){
	external_counter prev_front_end = front_end.load(), next_front_end;		//memory order?
	do{
		next_front_end = prev_front_end;	//Note that if CAS fails below, prev_front_end will change to the actual value.
		++(next_front_end.ex_count);
	}while(!front_end.compare_exchange_weak(prev_front_end, next_front_end));	//memory order? We only need weak CAS here, since we just repeat.
	return counted_ptr(next_front_end.internals);
}

template <class T>
template <class... Args>
void double_ref_counter<T>::replace(Args&&... args){
	external_counter old_front_end = front_end.load(), new_front_end{new internal_counter(std::forward<Args>(args)...), 0};	//memory order?
	while(!front_end.compare_exchange_weak(old_front_end, new_front_end)){}	//need to ensure that the new_front_end was actually initialized before this CAS, memory order?
	if(old_front_end.internals->in_count.fetch_sub(old_front_end.ex_count) == old_front_end.ex_count){	//memory order? don't want the delete happening before this...
		delete old_front_end.internals;
	}
}

template <class T>
template <class... Args>
bool double_ref_counter<T>::try_replace(Args&&... args){
	external_counter old_front_end = front_end.load(), new_front_end{new internal_counter(std::forward<Args>(args)...), 0};	//memory order?
	if(front_end.compare_exchange_strong(old_front_end, new_front_end)){	//memory order? should probably have different orders for success and failure
		if(old_front_end.internals->in_count.fetch_sub(old_front_end.ex_count) == old_front_end.ex_count){	//memory order? also don't want delete happening before this...
			delete old_front_end.internals;
		}
		return true;
	}else{
		delete new_front_end.internals;
		return false;
	}
}

/*
 * The internal counter for the double-counting reference counter.
 */
template <class T>
struct double_ref_counter<T>::internal_counter{
	
	internal_counter() = delete;
	template <class... Args> internal_counter(Args&&... args) : data(std::forward<Args>(args)...), in_count(0) {}
	internal_counter(const internal_counter&) = delete;
	internal_counter(internal_counter&&) = delete;
	~internal_counter() = default;	//Destructor doesn't need to do anything.
	
	internal_counter& operator=(const internal_counter&) = delete;
	internal_counter& operator=(internal_counter&&) = delete;
	
	void release();
	
	value_type data;
	std::atomic<int> in_count;
	
};

template <class T>
void double_ref_counter<T>::internal_counter::release(){
	if(in_count.fetch_add(1) == -1){	//memory order?
		delete this;
	}
}

/*
 * The RAII class protecting access to an internal counter object.
 * This object is not thread-safe, and should not be shared between threads.
 */
template <class T>
class double_ref_counter<T>::counted_ptr{
public:
	
	counted_ptr(internal_counter* ptr = nullptr) : counted_internals(ptr) {}
	counted_ptr(const counted_ptr&) = delete;
	counted_ptr(counted_ptr&& other) : counted_internals(other.counted_internals) {other.counted_internals = nullptr;}
	~counted_ptr() {if(counted_internals){counted_internals->release();}}
	
	counted_ptr& operator=(const counted_ptr&) = delete;
	counted_ptr& operator=(counted_ptr&& other);
	
	const value_type& operator*() const {return counted_internals->data;}
	const value_type* operator->() const {return &(counted_internals->data);}
	
	value_type& operator*() {return counted_internals->data;}		//Some use of SFINAE would be good here to hide these when value_type is const.
	value_type* operator->() {return &(counted_internals->data);}	//Likewise.
	
private:
	
	internal_counter* counted_internals;
	
};

template <class T>
typename double_ref_counter<T>::counted_ptr& double_ref_counter<T>::counted_ptr::operator=(counted_ptr&& other){
	if(counted_internals){
		counted_internals->release();
	}
	counted_internals = other.counted_internals;
	other.counted_internals = nullptr;
	return *this;
}

}

#endif