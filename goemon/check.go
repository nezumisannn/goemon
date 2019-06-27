package goemon

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/spf13/viper"
)

// A variable that stores the value after Unmarshal
var config Config

// CheckFlag Option parsed by cobra
type CheckFlag struct {
	Config string
}

// Config Unmarshal yaml file
type Config struct {
	Notifier []Notifier `mapstructure:"notifier"`
}

// Notifier is a notification notifier
type Notifier struct {
	Notification string            `mapstructure:"notification"`
	Region       string            `mapstructure:"region"`
	Profile      string            `mapstructure:"profile"`
	Chatwork     []ChatworkNotifer `mapstructure:"chatwork"`
	EC2          []EC2Infomation   `mapstructure:"ec2"`
	RDS          []RDSInfomation   `mapstructure:"rds"`
}

// ChatworkNotifer is Notify to Chatwork
type ChatworkNotifer struct {
	Roomid string   `mapstructure:"roomid"`
	Apikey string   `mapstructure:"apikey"`
	To     []string `mapstructure:"to"`
}

// EC2Infomation is EC2 instance infomation
type EC2Infomation struct {
	Instances []string `mapstructure:"instances"`
}

// RDSInfomation is RDS instance infomation
type RDSInfomation struct {
	Instances []string `mapstructure:"instances"`
}

// Unmarshal is store yaml in a structure
func Unmarshal(file string) (err error) {
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return err
	}
	return nil
}

// ConnectEC2 is Connect EC2 Service
func ConnectEC2(session *session.Session, region string, profile string) (result *ec2.EC2) {
	credential := credentials.NewSharedCredentials("", profile)
	service := ec2.New(
		session,
		aws.NewConfig().WithRegion(region).WithCredentials(credential),
	)
	return service
}

// ConnectRDS is Connect RDS Service
func ConnectRDS(session *session.Session, region string, profile string) (result *rds.RDS) {
	credential := credentials.NewSharedCredentials("", profile)
	service := rds.New(
		session,
		aws.NewConfig().WithRegion(region).WithCredentials(credential),
	)
	return service
}

// GetEC2InstanceStatus is get list of instance status
func GetEC2InstanceStatus(service *ec2.EC2, instance string) (result *ec2.DescribeInstanceStatusOutput, err error) {
	params := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{
			aws.String(instance),
		},
	}
	response, err := service.DescribeInstanceStatus(params)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetEC2Instances is get list of instance infomation
func GetEC2Instances(service *ec2.EC2, instance string) (result *ec2.DescribeInstancesOutput, err error) {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instance),
		},
	}
	response, err := service.DescribeInstances(params)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetRDSPendingMaintenanceActions is get list of instance pending maintenance actions
func GetRDSPendingMaintenanceActions(service *rds.RDS, instance string) (result *rds.DescribePendingMaintenanceActionsOutput, err error) {
	params := &rds.DescribePendingMaintenanceActionsInput{
		ResourceIdentifier: aws.String(instance),
	}
	response, err := service.DescribePendingMaintenanceActions(params)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetEC2InstanceEvents is get event information from EC2 instance status
func GetEC2InstanceEvents(notifier Notifier, ec2service *ec2.EC2) (results [][]string) {
	var result [][]string
	for _, ec2 := range notifier.EC2 {
		for _, instance := range ec2.Instances {
			statuses, err := GetEC2InstanceStatus(ec2service, instance)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			instanceinfo, err := GetEC2Instances(ec2service, instance)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(instanceinfo.Reservations)
			for _, status := range statuses.InstanceStatuses {
				for _, events := range status.Events {
					var event []string
					for _, reservations := range instanceinfo.Reservations {
						for _, info := range reservations.Instances {
							for _, tag := range info.Tags {
								if *tag.Key == "Name" {
									hostinfo := *tag.Value + "(" + *info.InstanceId + ")"
									event = append(event, hostinfo)
								}
							}
							event = append(event, *info.PublicIpAddress)
							event = append(event, *info.PrivateIpAddress)
							event = append(event, *events.Code)
							event = append(event, *events.Description)
							event = append(event, events.NotBefore.Format(time.ANSIC))
							result = append(result, event)
						}
					}
				}
			}
		}
	}
	return result
}

// GetRDSPendingMaintenanceActionDetails is get maintenance action infomation from RDS pending maintenance actions
func GetRDSPendingMaintenanceActionDetails(notifier Notifier, rdsservice *rds.RDS) (results [][]string) {
	var result [][]string
	for _, rds := range notifier.RDS {
		for _, instance := range rds.Instances {
			fmt.Println(instance)
			actions, err := GetRDSPendingMaintenanceActions(rdsservice, instance)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			for _, action := range actions.PendingMaintenanceActions {
				var detail []string
				detail = append(detail, *action.ResourceIdentifier)
				for _, details := range action.PendingMaintenanceActionDetails {
					detail = append(detail, *details.Action)
					detail = append(detail, *details.Description)
					result = append(result, detail)
				}
			}
		}
	}
	return result
}

// PostChatwork is post chatwork api
func PostChatwork(roomid string, apikey string, body string) {
	api := "https://api.chatwork.com/v2/rooms/"
	path := "/messages"
	url := api + roomid + path

	request, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	request.Header.Add("X-ChatWorkToken", apikey)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		log.Fatal(err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
}

// NotifyEC2Chatwork is notify EC2 schedule events to chatwork
func NotifyEC2Chatwork(chatwork []ChatworkNotifer, ec2events [][]string) {
	for _, ec2event := range ec2events {
		completed := strings.Contains(ec2event[4], "Completed")
		if completed != true {
			for _, notify := range chatwork {
				roomid := notify.Roomid
				apikey := notify.Apikey
				body := "body="

				for _, to := range notify.To {
					body += "[To:" + to + "]"
				}

				body += "\n"
				body += "[info][title]Goemon AWS EC2 Schedule Event Notify[/title]"
				body += "Host : " + ec2event[0] + "\n"
				body += "PublicIpAddress : " + ec2event[1] + "\n"
				body += "PrivateIpAddress : " + ec2event[2] + "\n"
				body += "Code : " + ec2event[3] + "\n"
				body += "Description : " + ec2event[4] + "\n"
				body += "NotBefore : " + ec2event[5] + " UTC (JSTの場合は9時間加算) [/info]"

				PostChatwork(roomid, apikey, body)
			}
		}
	}
}

// NotifyRDSChatwork is notify RDS maintenance actions to chatwork
func NotifyRDSChatwork(chatwork []ChatworkNotifer, rdsactions [][]string) {
	for _, rdsaction := range rdsactions {
		for _, notify := range chatwork {
			roomid := notify.Roomid
			apikey := notify.Apikey
			body := "body="

			for _, to := range notify.To {
				body += "[To:" + to + "]"
			}

			body += "\n"
			body += "[info][title]Goemon AWS RDS Maintenance Action Notify[/title]"
			body += "ResourceIdentifier : " + rdsaction[0] + "\n"
			body += "Action : " + rdsaction[1] + "\n"
			body += "Description : " + rdsaction[2] + "[/info]"

			PostChatwork(roomid, apikey, body)
		}
	}
}

// Check AWS infomation
func Check(flag *CheckFlag) {

	if err := Unmarshal(flag.Config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	session := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	for _, notifier := range config.Notifier {
		region := notifier.Region
		profile := notifier.Profile
		notification := notifier.Notification
		chatwork := notifier.Chatwork

		ec2service := ConnectEC2(session, region, profile)
		rdsservice := ConnectRDS(session, region, profile)

		ec2events := GetEC2InstanceEvents(notifier, ec2service)
		rdsactions := GetRDSPendingMaintenanceActionDetails(notifier, rdsservice)

		switch notification {
		case "chatwork":
			NotifyEC2Chatwork(chatwork, ec2events)
			NotifyRDSChatwork(chatwork, rdsactions)
		}
	}
}
