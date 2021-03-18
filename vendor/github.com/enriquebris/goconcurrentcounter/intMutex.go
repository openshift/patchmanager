package goconcurrentcounter

import (
	"sync"

	"github.com/enriquebris/goconcurrentqueue"
)

// IntMutex defines a concurrent-safe int
type IntMutex struct {
	value int
	// mutex for main value
	//mutex deadlock.Mutex
	mutex sync.Mutex
	// mutex for trigger's function
	//triggerFuncMutex deadlock.Mutex
	triggerFuncMutex sync.Mutex
	// map with functions per value
	mpFuncs map[int][]concurrentIntFuncEntry
	// queue of functions to be executed just after trigger functions
	runAfterTriggerFunctions goconcurrentqueue.Queue
}

// NewIntMutex creates and returns a *concurrentInt
func NewIntMutex(value int) *IntMutex {
	ret := &IntMutex{}
	ret.initialize(value)

	return ret
}

func (st *IntMutex) initialize(value int) {
	st.value = value
	st.mpFuncs = make(map[int][]concurrentIntFuncEntry)
	st.runAfterTriggerFunctions = goconcurrentqueue.NewFIFO()
}

// GetValue returns the value
func (st *IntMutex) GetValue() int {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	return st.value
}

// Update updates the value. It adds the upd parameter to the value.
// If the updated value has an associated function, it will be executed.
func (st *IntMutex) Update(upd int) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.value = st.value + upd

	// executes associated function, if any
	st.executeTriggerFunctions(st.value)
}

// executeTriggerFunctions executes all trigger functions associated to the given value
func (st *IntMutex) executeTriggerFunctions(value int) {
	st.triggerFuncMutex.Lock()
	defer func() {
		// unlock
		st.triggerFuncMutex.Unlock()

		// execute the functions enqueued in runAfterTriggerFunctions
		var (
			rawFunc interface{}
			err     error
		)
		for err == nil {
			rawFunc, err = st.runAfterTriggerFunctions.Dequeue()
			if rawFunc != nil {
				fn, _ := rawFunc.(concurrentIntFunc)
				fn()
			}
		}
	}()

	if fns, ok := st.mpFuncs[value]; ok {
		for i := 0; i < len(fns); i++ {
			fns[i].Func()
		}
	}
}

// GetTriggerOnValue returns the trigger function associated to the given value-name. It returns nil if no function
// is associated to the given values.
func (st *IntMutex) GetTriggerOnValue(value int, name string) concurrentIntFunc {
	st.triggerFuncMutex.Lock()
	defer st.triggerFuncMutex.Unlock()

	if triggerFunctions, ok := st.mpFuncs[value]; ok {
		for _, triggerFn := range triggerFunctions {
			if triggerFn.Name == name {
				return triggerFn.Func
			}
		}
	}

	return nil
}

// SetTriggerOnValue sets a trigger that would be executed once the given value were reached.
// Note: do not call UnsetTriggerOnValue() as part of the given function (it would arrive to a deadlock scenario),
// instead enqueue a function (using EnqueueToRunAfterCurrentTriggerFunctions()) that call it.
func (st *IntMutex) SetTriggerOnValue(value int, name string, fn concurrentIntFunc) {
	st.triggerFuncMutex.Lock()
	defer st.triggerFuncMutex.Unlock()

	slice, ok := st.mpFuncs[value]
	if !ok {
		slice = make([]concurrentIntFuncEntry, 0)
	}

	entry := concurrentIntFuncEntry{
		Name: name,
		Func: fn,
	}
	slice = append(slice, entry)
	st.mpFuncs[value] = slice
}

// UnsetTriggerOnValue removes the function to be executed on a value
func (st *IntMutex) UnsetTriggerOnValue(value int, name string) {
	st.triggerFuncMutex.Lock()
	defer st.triggerFuncMutex.Unlock()

	if slice, ok := st.mpFuncs[value]; ok {
		if len(slice) == 1 {
			delete(st.mpFuncs, value)
			return
		}

		newSlice := make([]concurrentIntFuncEntry, len(slice)-1)
		index := 0
		for i := 0; i < len(slice); i++ {
			if slice[i].Name == name {
				continue
			}

			newSlice[index] = slice[i]
			index++
		}

		st.mpFuncs[value] = newSlice
	}
}

// UnsetTriggersOnValue removes all function to be executed on a given value
func (st *IntMutex) UnsetTriggersOnValue(value int) {
	delete(st.mpFuncs, value)
}

// EnqueueToRunAfterCurrentTriggerFunctions enqueues a function to be executed just after the trigger functions
func (st *IntMutex) EnqueueToRunAfterCurrentTriggerFunctions(fn concurrentIntFunc) {
	st.runAfterTriggerFunctions.Enqueue(fn)
}
