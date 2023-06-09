package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/nadavbm/etzba/roles/apiclient"
	"github.com/nadavbm/etzba/roles/worker"
	"go.uber.org/zap"
)

var wg sync.WaitGroup
var mutex = &sync.Mutex{}

// ExecuteJobByDuration when "--duration=Xx" is given via command line, shceduler will fill work channel with assignments until the job duration is over
// after execution is completed, it will return the result of the load test
func (s *Scheduler) ExecuteJobByDuration() (*Result, error) {
	assignments, err := s.setAssignmentsToWorkers()
	if err != nil {
		s.Logger.Fatal("could not create assignments")
		panic(err)
	}

	allAssignmentsExecutionsDurations, allAssignmentsExecutionsResponses, err := s.prepareAssignmentsForResultCollection(assignments)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	wg.Add(s.numberOfWorkers + 3)
	for i := 0; i < s.numberOfWorkers; i++ {
		go func(num int) {
			defer wg.Done()
			for a := range s.tasksChan {
				if s.Verbose {
					s.Logger.Info(fmt.Sprintf("worker %d execute task %v", num, &a))
				}
				duration, resp, err := s.executeTaskFromAssignment(&a)
				if err != nil {
					s.Logger.Error(fmt.Sprintf("worker could not execute task %v", &a), zap.Error(err))
				}
				title := getAssignmentAsString(a, s.ExecutionType)
				mutex.Lock()
				allAssignmentsExecutionsDurations = appendDurationToAssignmentResults(title, allAssignmentsExecutionsDurations, duration)
				allAssignmentsExecutionsResponses = appendResponseToAssignmentResults(title, allAssignmentsExecutionsResponses, resp)
				mutex.Unlock()
			}
		}(i)
	}

	go s.addToWorkChannel(s.setRps(), s.jobDuration, s.tasksChan, assignments)

	go func() {
		wg.Wait()
	}()

	for {
		val, ok := <-s.tasksChan
		if ok == false {
			wg.Done()
			break
		} else {
			s.tasksChan <- val
		}
	}

	res := &Result{
		JobDuration: time.Since(now) - time.Second,
		RequestRate: s.calculateRequestRate(time.Since(now)-time.Second, len(concatAllDurations(allAssignmentsExecutionsDurations))),
		Assignments: allAssignmentsExecutionsDurations,
		Durations:   concatAllDurations(allAssignmentsExecutionsDurations),
		Responses:   allAssignmentsExecutionsResponses,
	}

	return res, nil

}

// addToWorkChannel will add assignments to work channel and close the channel when the duration time is over
func (s *Scheduler) addToWorkChannel(sleepTime, duration time.Duration, c chan worker.Assignment, assigments []worker.Assignment) {
	defer wg.Done()
	timer := time.NewTimer(duration)

	for {
		select {
		case <-timer.C:
			timer.Stop()
			fmt.Println(fmt.Sprintf("job completed after %v", duration))
			wg.Done()
			// TODO: set max query time and sleep before closing the channl to allow all workers finish their assignment executions.
			time.Sleep(1 * time.Second)
			close(c)
			return
		default:
			for _, a := range assigments {
				time.Sleep(sleepTime)
				c <- a
			}
		}
	}
}

// appendDurationToAssignmentResults collect all durations per assignment during job execution for any worker and return a map with all assignments and their durations
func appendDurationToAssignmentResults(title string, assignmentResults map[string][]time.Duration, duration time.Duration) map[string][]time.Duration {
	for key, val := range assignmentResults {
		if title == key {
			val = append(val, duration)
			assignmentResults[key] = val
		}
	}

	return assignmentResults
}

// appendResponseToAssignmentResults collect all responses from server per assignment
func appendResponseToAssignmentResults(title string, assignmentResponses map[string][]*apiclient.Response, response *apiclient.Response) map[string][]*apiclient.Response {
	for key, val := range assignmentResponses {
		if title == key {
			val = append(val, response)
			assignmentResponses[key] = val
		}
	}

	return assignmentResponses
}

// concatAllDurations from assignment results to return durations from all assignments
func concatAllDurations(assignmentResults map[string][]time.Duration) []time.Duration {
	var allDurations []time.Duration
	for _, val := range assignmentResults {
		for _, dur := range val {
			allDurations = append(allDurations, dur)
		}
	}
	return allDurations
}
