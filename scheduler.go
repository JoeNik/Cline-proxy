package main

import (
	"log"
	"strconv"
	"strings"
	"time"
)

type CronScheduler struct {
	expression string
	stopChan   chan bool
	running    bool
}

func NewCronScheduler(expression string) *CronScheduler {
	return &CronScheduler{
		expression: expression,
		stopChan:   make(chan bool),
		running:    false,
	}
}

func (c *CronScheduler) Start(task func()) {
	if c.running {
		return
	}
	c.running = true

	go func() {
		for {
			next := c.nextExecution()
			if next.IsZero() {
				log.Printf("Invalid cron expression: %s", c.expression)
				return
			}

			waitDuration := time.Until(next)
			log.Printf("Next auto-refresh scheduled at: %s (in %v)", next.Format("2006-01-02 15:04:05"), waitDuration)

			select {
			case <-time.After(waitDuration):
				log.Println("Starting scheduled auto-refresh...")
				task()
			case <-c.stopChan:
				log.Println("Auto-refresh scheduler stopped")
				return
			}
		}
	}()
}

func (c *CronScheduler) Stop() {
	if c.running {
		c.stopChan <- true
		c.running = false
	}
}

func (c *CronScheduler) nextExecution() time.Time {
	now := time.Now()
	fields := strings.Fields(c.expression)
	if len(fields) != 5 {
		return time.Time{}
	}

	minute, err1 := parseCronField(fields[0], 0, 59)
	hour, err2 := parseCronField(fields[1], 0, 23)
	// day := fields[2]   // day of month (not fully implemented)
	// month := fields[3] // month (not fully implemented)
	// weekday := fields[4] // day of week (not fully implemented)

	if err1 != nil || err2 != nil {
		return time.Time{}
	}

	// Simple implementation: only supports */N and N syntax
	next := now.Add(time.Minute)
	next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), 0, 0, next.Location())

	for i := 0; i < 24*60; i++ {
		if matchesCronField(next.Minute(), minute) && matchesCronField(next.Hour(), hour) {
			return next
		}
		next = next.Add(time.Minute)
	}

	return time.Time{}
}

func parseCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		result := make([]int, max-min+1)
		for i := range result {
			result[i] = min + i
		}
		return result, nil
	}

	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil {
			return nil, err
		}
		result := []int{}
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	val, err := strconv.Atoi(field)
	if err != nil {
		return nil, err
	}
	return []int{val}, nil
}

func matchesCronField(value int, allowed []int) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

var scheduler *CronScheduler

func startAutoRefresh() {
	cfg := loadConfig()
	if !cfg.AutoRefreshEnabled {
		log.Println("Auto-refresh is disabled")
		return
	}

	if scheduler != nil {
		scheduler.Stop()
	}

	scheduler = NewCronScheduler(cfg.AutoRefreshCron)
	scheduler.Start(func() {
		refreshExpiredAccounts()
	})
}

func stopAutoRefresh() {
	if scheduler != nil {
		scheduler.Stop()
		scheduler = nil
	}
}

func refreshExpiredAccounts() {
	p := loadPool()
	poolMu.Lock()
	defer poolMu.Unlock()

	refreshed := 0
	failed := 0

	for _, acc := range p.Accounts {
		// Refresh accounts that are expired or in cooldown
		if acc.Status == "expired" || acc.Status == "cooldown" {
			log.Printf("Refreshing account: %s (%s)", acc.Email, acc.AccountID)
			if err := refreshAccountToken(acc); err != nil {
				log.Printf("Failed to refresh %s: %v", acc.Email, err)
				failed++
			} else {
				log.Printf("Successfully refreshed %s", acc.Email)
				refreshed++
			}
		}
	}

	log.Printf("Auto-refresh completed: %d refreshed, %d failed", refreshed, failed)
}
