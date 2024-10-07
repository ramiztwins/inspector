package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"inspector/mylogger"
	"github.com/robfig/cron/v3"
)

type ScheduleOptions struct {
	// Cron expression for scheduling (optional).
	CronExpr string
	// Timezone string ("Europe/Istanbul")
	TimeZone string
	// Schedule type: "daily", "weekly", "monthly", or "yearly".
	Schedule string
	// Time string in "HH:MM" format.
	Time string
}

// ScheduleTask schedules a task based on the provided options.
func ScheduleTask(options ScheduleOptions, task func()) error {
	// Validate and generate the cron expression if it's not provided.
	if options.CronExpr == "" {
		var err error
		options.CronExpr, err = TimeToCron(options.Schedule, options.Time)
		if err != nil {
			mylogger.MainLogger.Errorf("Failed to generate cron expression: %v", err)
			return fmt.Errorf("failed to generate cron expression: %w", err)
		}
	}

	location, err := time.LoadLocation(options.TimeZone)
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to load timezone '%s': %v", options.TimeZone, err)
		return fmt.Errorf("failed to load timezone: %w", err)
	}

	scheduler := cron.New(cron.WithLocation(location))

	_, err = scheduler.AddFunc(options.CronExpr, task)
	if err != nil {
		mylogger.MainLogger.Errorf("Failed to schedule task with cron expression '%s': %v", options.CronExpr, err)
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	mylogger.MainLogger.Infof("Task scheduled with cron expression '%s' in timezone '%s'", options.CronExpr, options.TimeZone)

	scheduler.Start()
	go func() {
		<-context.Background().Done()
		scheduler.Stop()
	}()

	return nil
}

// TimeToCron converts a time string and schedule type to a cron expression.
func TimeToCron(schedule, timeStr string) (string, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		err := fmt.Errorf("invalid time format, expected HH:MM, got: %s", timeStr)
		mylogger.MainLogger.Errorf("TimeToCron error: %v", err)
		return "", err
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		err = fmt.Errorf("invalid hour value: %s", parts[0])
		mylogger.MainLogger.Errorf("TimeToCron error: %v", err)
		return "", err
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		err = fmt.Errorf("invalid minute value: %s", parts[1])
		mylogger.MainLogger.Errorf("TimeToCron error: %v", err)
		return "", err
	}

	// Generate the cron expression based on the schedule type.
	var cronExpr string
	switch schedule {
	case "daily":
		// At the specified time every day.
		cronExpr = fmt.Sprintf("%d %d * * *", minute, hour)
	case "weekly":
		// At the specified time every Sunday.
		cronExpr = fmt.Sprintf("%d %d * * 0", minute, hour)
	case "monthly":
		// At the specified time on the 1st of every month.
		cronExpr = fmt.Sprintf("%d %d 1 * *", minute, hour)
	case "yearly":
		// At the specified time on January 1st every year.
		cronExpr = fmt.Sprintf("%d %d 1 1 *", minute, hour)
	default:
		err = fmt.Errorf("unsupported schedule type: %s", schedule)
		mylogger.MainLogger.Errorf("TimeToCron error: %v", err)
		return "", err
	}

	mylogger.MainLogger.Infof("Generated cron expression '%s' for schedule type '%s' and time '%s'", cronExpr, schedule, timeStr)
	return cronExpr, nil
}
