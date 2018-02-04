package core

type DeferredAction func()

//Utility interface used for communication between functions in context of single logical operation
type OperationContext interface {
	//similar to defer keyword in Go but in context of single logical operation
	Defer(action DeferredAction)
}
