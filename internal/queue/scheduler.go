package queue

import "github.com/hibiken/asynq"

func RegisterSchedules(s *asynq.Scheduler, defaultQueue string) error {
	// Every minute: enqueue a ping task into default queue.
	task := asynq.NewTask(TypeExamplePing, []byte(`{"hello":"world"}`))
	_, err := s.Register("@every 1m", task, asynq.Queue(defaultQueue))
	return err
}
