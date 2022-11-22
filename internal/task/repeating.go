package task

import "time"

type RepeatingTask struct {
	Interval   time.Duration
	RunAtStart bool
	Action     func()

	running bool
	stop    chan struct{}
}

func (task *RepeatingTask) Start() {
	if task.running {
		return
	}
	if task.RunAtStart {
		task.Action()
	}
	task.running = true
	task.stop = make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-task.stop:
				return
			case <-time.After(task.Interval):
				task.Action()
			}
		}
	}()
}

func (task *RepeatingTask) Stop() {
	if !task.running {
		return
	}
	close(task.stop)
	task.running = false
}
