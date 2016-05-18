package wray

import "time"

type schedular interface {
	wait(time.Duration, func())
	delay() time.Duration
}

type channelSchedular struct {
}

func (cs channelSchedular) wait(delay time.Duration, callback func()) {
	go func() {
		time.Sleep(delay)
		callback()
	}()
}

func (cs channelSchedular) delay() time.Duration {
	return (1 * time.Minute)
}
