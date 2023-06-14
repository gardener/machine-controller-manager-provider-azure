package utils

import "time"

func ElapsedDuration(name string, invocationTime time.Time) {
	elapsed := time.Since(invocationTime)
}
