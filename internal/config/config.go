package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func Load(path string) (*Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("no path specified")
	}

	fullpath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("path %s expansion failed: %v", path, err)
	}

	var cfg Config

	fd, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}

	reader := viper.New()
	reader.SetConfigType("yaml")
	if err := reader.ReadConfig(fd); err != nil {
		return nil, err
	}
	if err := reader.UnmarshalExact(&cfg); err != nil {
		return nil, err
	}

	// Read environment variables for config
	if pd := ReadPagerDutyFromEnv(); pd != nil {
		cfg.Alert.PagerDuty = pd
	}
	if sk := ReadSlackFromEnv(); sk != nil {
		cfg.Alert.Slack = sk
	}

	return &cfg, nil
}

type Config struct {
	Checks []Check `yaml:"checks"`

	Alert  Alert        `yaml:"alert"`
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	BindAddress string `yaml:"bindAddress"`
}

type Check struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	Schedule ScheduleConfig `yaml:"schedule"`

	Alert Alert `yaml:"alert"`
}

type ScheduleConfig struct {
	Every       *EveryConfig `yaml:"every"`
	Weekdays    *PartialDay  `yaml:"weekdays"`
	BankingDays *PartialDay  `yaml:"bankingDays"`
}

type EveryConfig struct {
	Interval time.Duration `yaml:"interval"`

	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

type PartialDay struct {
	Timezone  string   `yaml:"timezone"`
	Times     []string `yaml:"times"`
	Tolerance string   `yaml:"tolerance"`
}

func (t PartialDay) GetTimes() ([]time.Time, error) {
	times := make([]string, len(t.Times))
	copy(times, t.Times)
	slices.Sort(times)

	var out []time.Time
	for _, tt := range times {
		when, err := time.Parse("15:04", tt)
		if err != nil {
			return nil, fmt.Errorf("parsing %s failed: %w", tt, err)
		}
		out = append(out, when)
	}
	return out, nil
}

type Alert struct {
	PagerDuty *PagerDuty `yaml:"pagerduty"`
	Slack     *Slack
}

type PagerDuty struct {
	ApiKey           string `yaml:"apiKey"`
	EscalationPolicy string `yaml:"escalationPolicy"`

	// From is an email address of a valid user associated with the account making the request
	From string `yaml:"from"`

	RoutingKey string `yaml:"routingKey"`

	Urgency string `yaml:"urgency"`
}

func ReadPagerDutyFromEnv() *PagerDuty {
	apiKey := strings.TrimSpace(os.Getenv("DEADCHECK_PAGERDUTY_API_KEY"))
	escPolicy := os.Getenv("DEADCHECK_PAGERDUTY_ESCALATION_POLICY")
	from := os.Getenv("DEADCHECK_PAGERDUTY_FROM")

	if apiKey != "" && escPolicy != "" && from != "" {
		return &PagerDuty{
			ApiKey:           apiKey,
			EscalationPolicy: escPolicy,
			From:             from,
			RoutingKey:       os.Getenv("DEADCHECK_PAGERDUTY_ROUTING_KEY"),
		}
	}
	return nil
}

type Slack struct {
	ApiToken  string
	ChannelID string

	Username string
	ImageURI string
}

func ReadSlackFromEnv() *Slack {
	apiToken := os.Getenv("DEADCHECK_SLACK_API_TOKEN")
	channelID := os.Getenv("DEADCHECK_SLACK_CHANNEL_ID")

	username := os.Getenv("DEADCHECK_SLACK_USERNAME")
	imageURI := os.Getenv("DEADCHECK_SLACK_IMAGE_URI")

	if apiToken != "" && channelID != "" {
		return &Slack{
			ApiToken:  apiToken,
			ChannelID: channelID,
			Username:  username,
			ImageURI:  imageURI,
		}
	}

	return nil
}
