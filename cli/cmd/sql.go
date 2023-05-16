package cmd

import (
	"github.com/nadavbm/etzba/pkg/printer"
	"github.com/nadavbm/etzba/roles/scheduler"
	"github.com/nadavbm/zlog"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	sqlCmd = &cobra.Command{
		Use:       "sql",
		Short:     "Start benchmarking your sql instance",
		Long:      `Start benchmarking db, defining the number of workers and csv file input`,
		ValidArgs: validArgs,
		Run:       benchmarkSql,
	}
)

func benchmarkSql(cmd *cobra.Command, args []string) {
	logger := zlog.New()

	jobDuration, err := setDurationFromString(duration)
	if err != nil {
		logger.Fatal("could set job duration")
	}

	s, err := scheduler.NewScheduler(logger, jobDuration, "sql", configFile, helpersFile, workersCount, Verbose)
	if err != nil {
		logger.Fatal("could not create a scheduler instance")
	}

	var result *scheduler.Result
	if duration != "" {
		result, err = s.ExecuteJobByDuration()
		if err != nil {
			s.Logger.Fatal("could not start execution", zap.Error(err))
		}
	} else {
		if result, err = s.ExecuteJobUntilCompletion(); err != nil {
			s.Logger.Fatal("could not start execution")
		}
	}

	printer.PrintToTerminal(result, false)
}
