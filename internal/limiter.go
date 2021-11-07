package internal

import (
	"errors"
)

type Limiter struct {
	ch chan struct{}
}

func NewLimiter(limit int) *Limiter {
	ch := make(chan struct{}, limit)

	for i := 0; i < limit; i++ {
		ch <- struct{}{}
	}

	return &Limiter{
		ch: ch,
	}
}

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
