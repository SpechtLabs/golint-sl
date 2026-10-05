package pattern

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

type FakeClock struct{}

type Timestamp struct{}

func use() {
	_ = time.Now()
	_ = time.Now()
	<-time.After(time.Second)
	time.Sleep(time.Second)
	other()
}

func other() {}
