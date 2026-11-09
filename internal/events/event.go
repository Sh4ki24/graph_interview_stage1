package events

// Event is fired on every task state change and consumed asynchronously
// by a worker goroutine.
type Event struct {
	TaskID int64
	Action string // created | updated | closed
	Status string
}
