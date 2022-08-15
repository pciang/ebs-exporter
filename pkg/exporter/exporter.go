package exporter

import (
	"regexp"

	"github.com/VictoriaMetrics/metrics"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/deliveryhero/misc-exporter/pkg/config"
	"github.com/sirupsen/logrus"
)

// Exporter is a struct that contains an instance
// of AWS clients and job configuration
type Exporter struct {
	clients []MetricsCollector
	job     *config.Job
	logger  *logrus.Logger
	session *session.Session
}

// MetricsCollector is an interface for
// a set of methods to interact with AWS
type MetricsCollector interface {
	Collect() error
}

// FormatTag replaces special characters with
// underscores for prometheus metric naming convention:
// https://prometheus.io/docs/instrumenting/writing_exporters/#naming
func FormatTag(text string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9:_]+")
	return re.ReplaceAllString(text, "_")
}

// New returns a new instance of Exporter for a job config
func New(log *logrus.Logger, j *config.Job, m *metrics.Set) *Exporter {
	log.Debugf("setting up exporter for job: %s", j.Name)
	exporter := &Exporter{
		job:     j,
		logger:  log,
		session: config.NewAwsSessionFromJob(j),
	}

	return exporter
}

// AddClient adds a MetricsCollector client to the Exporter.
func (ex *Exporter) AddClient(client MetricsCollector) {
	ex.clients = append(ex.clients, client)
}

// Clients returns the Exporter's clients.
func (ex *Exporter) Clients() []MetricsCollector {
	return ex.clients
}

// Job returns the Exporter's job.
func (ex *Exporter) Job() *config.Job {
	return ex.job
}

// Logger returns the exporter's logging instance.
func (ex *Exporter) Logger() *logrus.Logger {
	return ex.logger
}

// Session returns the Exporter's AWS Session.
func (ex *Exporter) Session() *session.Session {
	return ex.session
}
