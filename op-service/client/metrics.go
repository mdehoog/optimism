package client

import "time"

const (
	BatchMethod = "<batch>"
)

type DurationObserver interface {
	ObserveDuration() time.Duration
}

type Metrics interface {
	RecordRPCClientRequest(method string) func(err error)
	RecordRPCClientResponse(method string, err error)
	RecordBatchDuration(method string) DurationObserver
	RecordBatchMethod(method string)
}
