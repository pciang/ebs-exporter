package ebs

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"
	"github.com/thunderbottom/ebs-exporter/pkg/config"
	"github.com/thunderbottom/ebs-exporter/pkg/exporter"
	"golang.org/x/sync/errgroup"
)

const (
	namespace = "ebs"
)

// EBSExporter is a struct that holds an instance
// of EC2 client and the job details required to
// scrape EBS metrics
type EBSExporter struct {
	client     *ec2.EC2
	cloudwatch *cloudwatch.CloudWatch
	filters    []*ec2.Filter
	job        *config.Job
	logger     *logrus.Logger
	metrics    *metrics.Set
}

// New returns a new instance of EBSExporter
func New(j *config.Job, log *logrus.Logger, m *metrics.Set, rc *aws.Config, sess *session.Session) *EBSExporter {
	// create instances of ec2 and cloudwatch clients
	var (
		client *ec2.EC2
		cw     *cloudwatch.CloudWatch
	)
	// RoleARN config overrides Access Key and Secret
	if rc != nil {
		client = ec2.New(sess, rc)
		cw = cloudwatch.New(sess, rc)
	} else {
		client = ec2.New(sess)
		cw = cloudwatch.New(sess)
	}

	describeVolumesFilters := make([]*ec2.Filter, 0, len(j.Filters))
	for _, tag := range j.Filters {
		if tag.Name != "" && len(tag.Values) > 0 {
			describeVolumesFilters = append(describeVolumesFilters, &ec2.Filter{
				Name:   aws.String(tag.Name),
				Values: tag.Values,
			})
		}
	}

	log.Debugf("%s: setting up a new EBS exporter", j.Name)
	return &EBSExporter{
		client:     client,
		cloudwatch: cw,
		filters:    describeVolumesFilters,
		job:        j,
		logger:     log,
		metrics:    m,
	}
}

// Collect calls methods to collect metrics from AWS
func (e *EBSExporter) Collect() error {
	var g errgroup.Group

	g.Go(e.ec2DescribeVolumes)

	// Return if any of the errgroup
	// goroutines returns an error
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// ec2DescribeVolumes scrapes EBS volume usage metrics from AWS
func (e *EBSExporter) ec2DescribeVolumes() error {
	input := &ec2.DescribeVolumesInput{}
	if len(e.filters) > 0 {
		input.Filters = e.filters
	}

	volumes, err := e.client.DescribeVolumes(input)
	if err != nil {
		e.logger.Errorf("An error occurred while retrieving volume usage data: %s", err)
		return err
	}

	e.logger.Debugf("%s: Got %d Volumes", e.job.Name, len(volumes.Volumes))
	for _, v := range volumes.Volumes {
		// Labels to attach in ec2_describe_volumes
		labels := fmt.Sprintf(`job="%s",region="%s",vol_id="%s",type="%s",status="%s",availability_zone="%s"`,
			e.job.Name, e.job.AWS.Region, *v.VolumeId, *v.VolumeType, *v.State, *v.AvailabilityZone)

		// Check whether the volume contains any tags
		// that we want to export
		for _, et := range e.job.Tags {
			for _, t := range v.Tags {
				if *t.Key == et.Tag {
					// Ensure that the tags are correct by replacing
					// unsupported characters with underscore
					labels = labels + fmt.Sprintf(`,%s="%s"`, exporter.FormatTag(et.ExportedTag), *t.Value)
				}
			}
		}

		ebsVolume := fmt.Sprintf(`ec2_describe_volumes{%s}`, labels)
		e.metrics.GetOrCreateGauge(ebsVolume, func() float64 {
			return 1
		})
	}

	return nil
}
