package config

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/jessevdk/go-flags"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
)

type awsCredentials struct {
	Profile     string `koanf:"profile"`
	AccessKey   string `koanf:"access_key"`
	SecretKey   string `koanf:"secret_key"`
	SecretToken string `koanf:"secret_token"`
	Region      string `koanf:"region"`
	RoleARN     string `koanf:"role_arn"`
}

type Config struct {
	Debug  bool   `koanf:"debug"`
	Jobs   []Job  `koanf:"jobs"`
	Server server `koanf:"server"`
}

type Filter struct {
	Name   string    `koanf:"name"`
	Values []*string `koanf:"values"`
}

type Job struct {
	Name         string         `koanf:"name"`
	AWS          awsCredentials `koanf:"aws"`
	Filters      []Filter       `koanf:"filters"`
	Tags         []Tag          `koanf:"tags"`
	AwsAccountId string
}

type server struct {
	Address      string        `koanf:"address"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
}

type Tag struct {
	Tag         string `koanf:"tag"`
	ExportedTag string `koanf:"exported_tag"`
}

type options struct {
	ConfigFile string `short:"c" long:"config" description:"Path to configuration file" default:"config.toml"`
	Debug      bool   `short:"d" long:"debug" description:"Enable debug level logging"`
}

var (
	opt    options
	parser = flags.NewParser(&opt, flags.Default)
)

func NewAwsSessionFromJob(job *Job) *session.Session {
	var awsSession *session.Session

	commonAwsConfig := aws.Config{
		Region: aws.String(job.AWS.Region),
	}
	if job.AWS.RoleARN != "" {
		awsSession = session.Must(session.NewSession(&commonAwsConfig))
		awsSession.Config.Credentials = stscreds.NewCredentials(
			awsSession,
			job.AWS.RoleARN,
		)
	} else if job.AWS.Profile != "" {
		awsSession = session.Must(session.NewSessionWithOptions(session.Options{
			Profile: job.AWS.Profile,
			Config:  commonAwsConfig,
		}))
	} else if job.AWS.AccessKey != "" && job.AWS.SecretKey != "" {
		awsSession = session.Must(session.NewSession(&commonAwsConfig))
		awsSession.Config.Credentials = credentials.NewStaticCredentials(
			job.AWS.AccessKey,
			job.AWS.SecretKey,
			job.AWS.SecretToken,
		)
	} else {
		awsSession = session.Must(session.NewSession(&commonAwsConfig))
	}

	return awsSession
}

// ReadConfig reads and returns the configuration file
func ReadConfig() (*Config, error) {
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		default:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			return nil, fmt.Errorf("error loading configuration file: %s", err)
		}
	}

	var cfg = Config{}
	var koanf = koanf.New(".")
	if err := koanf.Load(file.Provider(opt.ConfigFile), toml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading configuration file: %s", err)
	}
	if err := koanf.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error loading configuration file: %s", err)
	}

	if opt.Debug {
		cfg.Debug = true
	}

	for jobIdx, _ := range cfg.Jobs {
		awsSession := NewAwsSessionFromJob(&cfg.Jobs[jobIdx])
		awsCallerIdentity, err := sts.New(awsSession).GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			return nil, err
		}
		cfg.Jobs[jobIdx].AwsAccountId = *awsCallerIdentity.Account
	}

	return &cfg, nil
}
