package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
)

// Global Vars
var nodeHealth bool
var instanceID string
var region string

// Arguements
var recheckDelay int
var serviceTag string

// Main Function
func main() {

	// Configure Logging
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// Configure CLI options
	var rootCmd = &cobra.Command{
		Use:   "asg-check-consul",
		Short: "asg-check-consul helps healthcheck instances in an AWS autoscaling group via Consul Healthchecks",
		Run: func(cmd *cobra.Command, args []string) {

			log.Infoln("Starting asg-check-consul")

			// Fetch EC2 metadata - region and instanceID
			log.Infoln("Contacting ec2 metadata service")

			ec2 := ec2metadata.New(session.New(), aws.NewConfig())

			ec2Region, err := ec2.Region()
			if err != nil {
				panic(err)
			}
			ec2Identity, err := ec2.GetInstanceIdentityDocument()
			if err != nil {
				panic(err)
			}

			region = ec2Region
			log.Infof("Instance Region: %s", region)
			instanceID = ec2Identity.InstanceID
			log.Infof("Instance ID: %s", instanceID)

			// Main loop
			log.Infof("Waiting grace period of %v seconds before checking", recheckDelay)
			for true {
				// Wait for the delay period before checking
				time.Sleep(time.Duration(recheckDelay) * time.Second)

				// Get service health from Consul agent
				oldHealth := nodeHealth
				nodeHealth = getConsulHealth()

				// If health has changed, notify AWS ASG
				if nodeHealth != oldHealth {
					setInstanceHealth(nodeHealth)
				}

			}
		},
	}

	// Command line arguments
	rootCmd.PersistentFlags().IntVarP(&recheckDelay, "recheck-delay", "", 30, "Grace period to wait between checking agent health")
	rootCmd.PersistentFlags().StringVarP(&serviceTag, "service-tag", "", "", "Service Tag to look up for critical services. This will only mark the service Unhealthy if a service with this tag fails (optional)")

	// RUN CLI
	rootCmd.Execute()
}

// Function to query consul services health
func getConsulHealth() bool {

	// Connect to Consul Client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Errorln("Cannot connect to consul")
		return false
	}

	// Retrieve checks through agent endpoint
	checks, err := client.Agent().Checks()
	if err != nil {
		log.Errorln("Cannot retrieve healthchecks")
		return false
	}

	// Look for failing checks
	failures := false
	for k := range checks {
		if checks[k].Status == "critical" {
			// Get tags for failing service
			service, _, _ := client.Agent().Service(checks[k].ServiceID, &api.QueryOptions{})

			// If service is tagged with the serviceTag parameter, mark instance as unhealthy
			if contains(service.Tags, serviceTag) || (serviceTag == "") {
				log.WithFields(log.Fields{"service": checks[k].ServiceName}).Errorln("failing service detected with specified tag, will mark as unhealthy")
				failures = true
			} else {
				// Service was not tagged with serviceTag parameter, so just log
				log.WithFields(log.Fields{"service": checks[k].ServiceName}).Warnln("failing service detected")
			}

		}
	}

	if failures {
		return false
	}
	log.Infoln("All Consul checks healthy")
	return true
}

// Function to set instance health to unhealthy
func setInstanceHealth(h bool) {

	// Convert h to String AWS api expects
	var Status string
	if h {
		Status = "Healthy"
	} else {
		Status = "Unhealthy"
	}

	// Construct instance health input for failing check
	svc := autoscaling.New(session.New(), aws.NewConfig().WithRegion(region))
	input := &autoscaling.SetInstanceHealthInput{
		HealthStatus: aws.String(Status),
		InstanceId:   aws.String(instanceID),
	}

	// Set instance health to failed
	log.Infof("Attempting to set instance health to %s", Status)
	_, err := svc.SetInstanceHealth(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeResourceContentionFault:
				log.Errorln(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				log.Errorln(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Errorln(aerr.Error())
		}
		return
	}

	log.Infof("instance health set to %s", Status)
}

// Function to looking value in array
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
