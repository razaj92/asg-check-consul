# asg-check-consul

Simple service to alert the AWS autoscaling API if the instance is unhealthy based on local consul agent services.

## Usage
 ```
 $ ./asg-check-consul --help
asg-check-consul helps healthcheck instances in an AWS autoscaling group via Consul Healthchecks

Usage:
  asg-check-consul [flags]

Flags:
  -h, --help                 help for asg-check-consul
      --recheck-delay int    Grace period to wait between checking agent health (default 30)
      --service-tag string   Service Tag to look up for critical services. This will only mark the service Unhealthy if a service with this tag fails (optional)
 ```

## Requirements
The instance IAM Profile needs the `autoscaling:SetInstanceHealth` permission
