package goconcurrentcounter

type Int interface {
	initialize(value int)
	executeTriggerFunctions(value int)
	GetValue() int
	Update(upd int)
	GetTriggerOnValue(value int, name string) concurrentIntFunc
	SetTriggerOnValue(value int, name string, fn concurrentIntFunc)
	UnsetTriggerOnValue(value int, name string)
	UnsetTriggersOnValue(value int)
	EnqueueToRunAfterCurrentTriggerFunctions(fn concurrentIntFunc)
}
