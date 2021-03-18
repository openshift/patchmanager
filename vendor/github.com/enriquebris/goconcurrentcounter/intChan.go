package goconcurrentcounter

import (
	"github.com/enriquebris/goconcurrentqueue"
)

const (
	concurrentIntValueChanCapacity   = 10
	concurrentIntTriggerChanCapacity = 10

	triggerActionSet           = "trigger.action.set"
	triggerActionGet           = "trigger.action.get"
	triggerActionUnset         = "trigger.action.unset"
	triggerActionUnsetTriggers = "trigger.action.unset.triggers"
	triggerActionExecute       = "trigger.action.execute"
)

// concurrent safe Int
type IntChan struct {
	value                    int
	valueChan                chan valueData
	triggerChan              chan concurrentIntTrigger
	triggerMap               map[int][]concurrentIntFuncEntry
	runAfterTriggerFunctions goconcurrentqueue.Queue
	keepWorking              bool
}

// valueData holds data to travel over the value channel
type valueData struct {
	update   int
	feedback chan valueData
}

// concurrentIntTrigger holds data to travel over trigger channel
type concurrentIntTrigger struct {
	Action   string
	Value    int
	Name     string
	Func     concurrentIntFunc
	Callback chan interface{}
}

// information about concurrentIntFunc
type concurrentIntFuncEntry struct {
	Name string
	Func concurrentIntFunc
}

// function to be executed by concurrentInt
type concurrentIntFunc func()

func NewIntChan(value int) *IntChan {
	ret := &IntChan{}
	ret.initialize(value)

	return ret
}

func (st *IntChan) initialize(value int) {
	st.keepWorking = true
	st.value = value
	st.valueChan = make(chan valueData, concurrentIntValueChanCapacity)
	st.triggerChan = make(chan concurrentIntTrigger, concurrentIntTriggerChanCapacity)
	st.triggerMap = make(map[int][]concurrentIntFuncEntry)
	st.runAfterTriggerFunctions = goconcurrentqueue.NewFIFO()

	go st.valueListener()
	go st.triggerListener()
}

func (st *IntChan) GetValue() int {
	//return st.value

	getValueChan := make(chan valueData, 2)

	st.valueChan <- valueData{
		update:   0,
		feedback: getValueChan,
	}

	vData := <-getValueChan
	return vData.update
}

func (st *IntChan) Update(value int) {
	st.valueChan <- valueData{
		update: value,
	}
}

func (st *IntChan) SetTriggerOnValue(value int, name string, fn concurrentIntFunc) {
	st.triggerChan <- concurrentIntTrigger{
		Action: triggerActionSet,
		Value:  value,
		Name:   name,
		Func:   fn,
	}
}

func (st *IntChan) GetTriggerOnValue(value int, name string) concurrentIntFunc {
	waitingForGetTriggerOnValue := make(chan interface{}, 2)
	st.triggerChan <- concurrentIntTrigger{
		Action:   triggerActionGet,
		Value:    value,
		Name:     name,
		Func:     nil,
		Callback: waitingForGetTriggerOnValue,
	}

	// wait for the callback
	rawTriggerOnValue := <-waitingForGetTriggerOnValue
	triggerOnValue, _ := rawTriggerOnValue.(concurrentIntFunc)
	return triggerOnValue
}

func (st *IntChan) UnsetTriggerOnValue(value int, name string) {
	st.triggerChan <- concurrentIntTrigger{
		Action:   triggerActionUnset,
		Value:    value,
		Name:     name,
		Func:     nil,
		Callback: nil,
	}
}

func (st *IntChan) UnsetTriggersOnValue(value int) {
	waitingForUnsetTriggersOnValue := make(chan interface{}, 2)
	st.triggerChan <- concurrentIntTrigger{
		Action:   triggerActionUnsetTriggers,
		Value:    value,
		Name:     "",
		Func:     nil,
		Callback: waitingForUnsetTriggersOnValue,
	}

	<-waitingForUnsetTriggersOnValue
}

// executeTriggerFunctions executes all trigger functions associated to the given value
func (st *IntChan) executeTriggerFunctions(value int) {
	waitingForExecuteTriggerFunctions := make(chan interface{}, 2)
	st.triggerChan <- concurrentIntTrigger{
		Action:   triggerActionExecute,
		Value:    value,
		Name:     "",
		Func:     nil,
		Callback: waitingForExecuteTriggerFunctions,
	}

	// wait for []concurrentIntFuncEntry
	rawEntries := <-waitingForExecuteTriggerFunctions

	if rawEntries != nil {
		defer func() {
			var (
				rawFunc interface{}
				err     error
			)
			// dequeue && run functions to be executed after trigger functions
			for err == nil {
				rawFunc, err = st.runAfterTriggerFunctions.Dequeue()
				if rawFunc != nil {
					fn, _ := rawFunc.(concurrentIntFunc)
					fn()
				}
			}
		}()

		entries, _ := rawEntries.([]concurrentIntFuncEntry)

		// execute functions
		for i := 0; i < len(entries); i++ {
			entries[i].Func()
		}
	}
}

// *************************************************************************************************************
// Listeners  **************************************************************************************************
// *************************************************************************************************************

func (st *IntChan) valueListener() {
	for st.keepWorking {
		update := <-st.valueChan
		if update.update != 0 {
			st.value += update.update

			// execute trigger functions
			st.executeTriggerFunctions(st.value)
		} else {
			if update.feedback != nil {
				select {
				case update.feedback <- valueData{
					update: st.value,
				}:
				default:
					// couldn't send the feedback
				}
			}
		}
	}
}

func (st *IntChan) triggerListener() {
	for st.keepWorking {
		triggerSignal := <-st.triggerChan

		switch triggerSignal.Action {
		case triggerActionSet:
			st.setTriggerOnValue(triggerSignal.Value, triggerSignal.Name, triggerSignal.Func)
		case triggerActionGet:
			triggerSignal.Callback <- st.getTriggerOnValue(triggerSignal.Value, triggerSignal.Name)
		case triggerActionUnset:
			st.unsetTriggerOnValue(triggerSignal.Value, triggerSignal.Name)
		case triggerActionUnsetTriggers:
			st.unsetTriggersOnValue(triggerSignal.Value)
		case triggerActionExecute:
			triggerSignal.Callback <- st.getTriggersOnValue(triggerSignal.Value)
		default:
		}
	}
}

// setTriggerOnValue adds a named function to a value.
// This function is only intended to be called by triggerListener
func (st *IntChan) setTriggerOnValue(value int, name string, fn concurrentIntFunc) {
	slice, ok := st.triggerMap[value]
	if !ok {
		slice = make([]concurrentIntFuncEntry, 0)
	}

	slice = append(slice, concurrentIntFuncEntry{
		Name: name,
		Func: fn,
	})

	st.triggerMap[value] = slice
}

// getTriggerOnValue returns the function for the given value and name
func (st *IntChan) getTriggerOnValue(value int, name string) concurrentIntFunc {
	sl, ok := st.triggerMap[value]
	if !ok {
		return nil
	}

	for i := 0; i < len(sl); i++ {
		if sl[i].Name == name {
			return sl[i].Func
		}
	}

	return nil
}

// getTriggersOnValue returns all the functions on a value
func (st *IntChan) getTriggersOnValue(value int) []concurrentIntFuncEntry {
	sl, ok := st.triggerMap[value]
	if !ok {
		return nil
	}

	return sl
}

func (st *IntChan) unsetTriggerOnValue(value int, name string) bool {
	sl, ok := st.triggerMap[value]
	if !ok {
		return false
	}

	newSlice := make([]concurrentIntFuncEntry, 0)
	for i := 0; i < len(sl); i++ {
		if sl[i].Name == name {
			continue
		}

		newSlice = append(newSlice, sl[i])
	}
	// assign the new slice (after remove one element)
	st.triggerMap[value] = newSlice

	return true
}

func (st *IntChan) unsetTriggersOnValue(value int) {
	delete(st.triggerMap, value)
}

// EnqueueToRunAfterCurrentTriggerFunctions enqueues a function to be executed just after the trigger functions
func (st *IntChan) EnqueueToRunAfterCurrentTriggerFunctions(fn concurrentIntFunc) {
	st.runAfterTriggerFunctions.Enqueue(fn)
}
