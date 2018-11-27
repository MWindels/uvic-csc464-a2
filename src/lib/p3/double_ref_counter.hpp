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
	
	//Public Types
	using value_type = T;
	class counted_ptr;
	
	//Constructors/Destructor
	template<class... Args> double_ref_counter(Args&&... args) : front_end(external_counter{new internal_counter(std::forward<Args>(args)...), 0}) {}
	double_ref_counter(const double_ref_counter&) = delete;
	double_ref_counter(double_ref_counter&&) = delete;				//May implement in the future, but not exactly paramount.
	~double_ref_counter();
	
	//Assignment Operators
	double_ref_counter& operator=(const double_ref_counter&) = delete;
	double_ref_counter& operator=(double_ref_counter&&) = delete;	//Likewise.
	
	//Member Functions
	counted_ptr obtain();
	template<class... Args> void replace(Args&&... args);
	template<class... Args> bool try_replace(const counted_ptr& expected, Args&&... args);
	
private:
	
	//Private Types
	class internal_counter;
	struct external_counter{
		internal_counter* internals;
		int ex_count;
	};
	
	//Data Members
	std::atomic<external_counter> front_end;
	
};

template <class T>
double_ref_counter<T>::~double_ref_counter(){
	external_counter prev_front_end = front_end.load();		//memory order?
	while(!front_end.compare_exchange_weak(prev_front_end, external_counter{nullptr, 0})){}	//memory order?
	if(prev_front_end.internals != nullptr){
		prev_front_end.internals->close(prev_front_end.ex_count);
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
	if(old_front_end.internals != nullptr){
		old_front_end.internals->close(old_front_end.ex_count);
	}
}

template <class T>
template <class... Args>
bool double_ref_counter<T>::try_replace(const counted_ptr& expected, Args&&... args){
	external_counter old_front_end = front_end.load();	//memory order?
	if(old_front_end.internals != expected.counted_internals){
		return false;
	}
	
	external_counter new_front_end{new internal_counter(std::forward<Args>(args)...), 0};
	while(!front_end.compare_exchange_weak(old_front_end, new_front_end)){	//memory order?
		if(old_front_end.internals != expected.counted_internals){
			delete new_front_end.internals;
			return false;
		}
	}
	if(old_front_end.internals != nullptr){
		old_front_end.internals->close(old_front_end.ex_count);
	}
	return true;
}

/*
 * The internal counter for the double-counting reference counter.
 */
template <class T>
class double_ref_counter<T>::internal_counter{
public:
	
	//Constructors/Destructor
	template <class... Args> internal_counter(Args&&... args) : data(std::forward<Args>(args)...), in_count(0) {}
	internal_counter(const internal_counter&) = delete;
	internal_counter(internal_counter&&) = delete;
	~internal_counter() = default;	//Destructor doesn't need to do anything.
	
	//Assignment Operators
	internal_counter& operator=(const internal_counter&) = delete;
	internal_counter& operator=(internal_counter&&) = delete;
	
	//Member Functions
	void release();
	void close(int referrers);
	
private:
	
	friend double_ref_counter<T>::counted_ptr;
	
	//Data Members
	value_type data;
	std::atomic<int> in_count;
	
};

template <class T>
void double_ref_counter<T>::internal_counter::release(){
	if(in_count.fetch_add(1) == -1){	//memory order?
		delete this;
	}
}

template <class T>
void double_ref_counter<T>::internal_counter::close(int referrers){
	if(in_count.fetch_sub(referrers) == referrers){	//memory order?
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
	
	//Constructors/Destructor
	counted_ptr(internal_counter* ptr = nullptr) : counted_internals(ptr) {}
	counted_ptr(const counted_ptr&) = delete;
	counted_ptr(counted_ptr&& other) : counted_internals(other.counted_internals) {other.counted_internals = nullptr;}
	~counted_ptr() {if(counted_internals){counted_internals->release();}}
	
	//Assignment Operators
	counted_ptr& operator=(const counted_ptr&) = delete;
	counted_ptr& operator=(counted_ptr&& other);
	
	//Const Accessors.  Can throw nullptr exceptions (so can the Mutable Accessors).
	std::add_const_t<value_type>& operator*() const {return counted_internals->data;}
	std::add_const_t<value_type>* operator->() const {return &(counted_internals->data);}
	
	//Mutable Accessors.  Disabled if value_type is const (with SFINAE).  Mutating data with these functions is not thread-safe unless the underlying data structure is thread-safe.
	template <class = std::enable_if_t<std::is_same_v<value_type, std::remove_const_t<value_type>>>> value_type& operator*() {return counted_internals->data;}
	template <class = std::enable_if_t<std::is_same_v<value_type, std::remove_const_t<value_type>>>> value_type* operator->() {return &(counted_internals->data);}
	
private:
	
	friend double_ref_counter<T>;
	
	//Data Members
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