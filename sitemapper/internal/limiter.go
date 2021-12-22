package sitemap

import (
	"errors"
)

// A Limiter provides a way of governing the number of concurrent goroutines using a buffered channel.
type Limiter struct {
	ch    chan struct{}
	limit int
}

// NewLimiter returns an instance of Limiter and initialises the limiter channel, filling it with empty structs.
func NewLimiter(limit int) *Limiter {
	ch := make(chan struct{}, limit)

	for i := 0; i < limit; i++ {
		ch <- struct{}{}
	}

	return &Limiter{
		ch:    ch,
		limit: limit,
	}
}

// RunFunc checks to see if there is room to run an additional concurrent activity by reading from the Limiter's
// buffered channel. If a struct is returned after reading from the channel the function is run.
// When the function is complete a new empty struct is placed in the channel.
// If there are no structs available to read from the channel it means we have no room to run any additional
// activities, and an error is returned.
func (l *Limiter) RunFunc(f func()) error {

	select {
	case <-l.ch:
		f()
		l.ch <- struct{}{}
		return nil
	default:
		return errors.New("limit reached, try again later")
	}
}
