package scheduler

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// ExecuteJobUntilCompletion when omitting '--duration' from the command, this function will execute
// all assignments from the helpers file until all assignments completed
func (s *Scheduler) ExecuteJobUntilCompletion() (*Result, error) {
	assignments, err := s.setAssignmentsToWorkers()
	if err != nil {
		s.Logger.Fatal("could not create assignments")
		panic(err)
	}

	allAssignmentsExecutionsDurations, allAssignmentsExecutionsResponses, err := s.prepareAssignmentsForResultCollection(assignments)
	if err != nil {
		panic(err)
	}

	results := make(chan time.Duration)
	workCh := make(workerChannel)

	// Start workers
	var wg sync.WaitGroup
	wg.Add(s.numberOfWorkers)
	for i := 0; i < s.numberOfWorkers; i++ {
		go func(num int) {
			defer wg.Done()
			for a := range workCh {
				duration, resp, err := s.executeTaskFromAssignment(&a)
				if err != nil {
					s.Logger.Fatal("could not execute task from assignment", zap.Error(err))
				}

				title := getAssignmentAsString(a, s.ExecutionType)
				mutex.Lock()
				allAssignmentsExecutionsDurations = appendDurationToAssignmentResults(title, allAssignmentsExecutionsDurations, duration)
				allAssignmentsExecutionsResponses = appendResponseToAssignmentResults(title, allAssignmentsExecutionsResponses, resp)
				mutex.Unlock()

				results <- duration
			}
		}(i)
	}

	// Close result channel when workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Send work to be done
	go func() {
		for _, a := range assignments {
			workCh <- a
		}
		close(workCh)
	}()

	var allDurations []time.Duration
	// Process results
	for r := range results {
		allDurations = append(allDurations, r)
	}

	res := &Result{
		Assignments: allAssignmentsExecutionsDurations,
		Durations:   allDurations,
		Responses:   allAssignmentsExecutionsResponses,
	}

	return res, nil
}
