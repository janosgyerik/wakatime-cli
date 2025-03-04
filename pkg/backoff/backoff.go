package backoff

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/wakatime/wakatime-cli/pkg/api"
	"github.com/wakatime/wakatime-cli/pkg/config"
	"github.com/wakatime/wakatime-cli/pkg/heartbeat"
	"github.com/wakatime/wakatime-cli/pkg/ini"
	"github.com/wakatime/wakatime-cli/pkg/log"

	"github.com/spf13/viper"
)

const (
	// resetAfter sets the total seconds a backoff will last.
	resetAfter = 3600
	// factor is the total seconds to be multiplied by.
	factor = 15
)

// Config defines backoff data.
type Config struct {
	// At is the time when the first failure happened.
	At time.Time
	// Retries is the number of attempts to connect.
	Retries int
	// V is an instance of Viper.
	V *viper.Viper
}

// WithBackoff initializes and returns a heartbeat handle option, which
// can be used in a heartbeat processing pipeline to prevent trying to send
// a heartbeat when the api is unresponsive.
func WithBackoff(c Config) heartbeat.HandleOption {
	return func(next heartbeat.Handle) heartbeat.Handle {
		return func(hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			log.Debugln("execute heartbeat backoff algorithm")

			if shouldBackoff(c.Retries, c.At) {
				return nil, api.Err("won't send heartbeat due to backoff")
			}

			results, err := next(hh)
			if err != nil {
				log.Debugf("incrementing backoff due to error")

				// error response, increment backoff
				if updateErr := updateBackoffSettings(c.V, c.Retries+1, time.Now()); updateErr != nil {
					log.Warnf("failed to update backoff settings: %s", updateErr)
				}

				return nil, err
			}

			if !c.At.IsZero() {
				// success response, reset backoff
				if resetErr := updateBackoffSettings(c.V, 0, time.Time{}); resetErr != nil {
					log.Warnf("failed to reset backoff settings: %s", resetErr)
				}
			}

			return results, nil
		}
	}
}

func shouldBackoff(retries int, at time.Time) bool {
	if retries < 1 || at.IsZero() {
		return false
	}

	now := time.Now()
	duration := time.Duration(float64(factor)*math.Pow(2, float64(retries))) * time.Second

	log.Debugf(
		"exponential backoff tried %d times since %s, will retry at %s",
		retries,
		at.Format(time.Stamp),
		at.Add(duration).Format(time.Stamp),
	)

	return now.Before(at.Add(duration)) && now.Before(at.Add(resetAfter*time.Second))
}

func updateBackoffSettings(v *viper.Viper, retries int, at time.Time) error {
	keys := map[ini.Key]string{
		{
			Section: "internal",
			Name:    "backoff_retries",
		}: strconv.Itoa(retries),
		{
			Section: "internal",
			Name:    "backoff_at",
		}: "",
	}

	if !at.IsZero() {
		keys[ini.Key{
			Section: "internal",
			Name:    "backoff_at",
		}] = at.Format(config.DateFormat)
	}

	iniFile, err := config.InternalFilePath(v)
	if err != nil {
		return fmt.Errorf("error getting filepath: %s", err)
	}

	return ini.SetKeys(iniFile, keys)
}
