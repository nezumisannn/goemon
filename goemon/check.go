package goemon

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
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

// GetEC2InstanceEvent is get event information from EC2 instance status
func GetEC2InstanceEvent(notifier Notifier, ec2service *ec2.EC2) (results [][]string) {
	var result [][]string
	var event []string
	for _, ec2 := range notifier.EC2 {
		for _, instance := range ec2.Instances {
			statuses, err := GetEC2InstanceStatus(ec2service, instance)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			for _, status := range statuses.InstanceStatuses {
				for _, events := range status.Events {
					event = append(event, instance)
					event = append(event, *events.Code)
					event = append(event, *events.Description)
					event = append(event, events.NotBefore.Format(time.ANSIC))
					result = append(result, event)
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
	fmt.Println(response)
}

// NotifyChatwork is notify Chatwork message
func NotifyChatwork(chatwork []ChatworkNotifer, ec2events [][]string) {
	for _, ec2event := range ec2events {
		completed := strings.Contains(ec2event[2], "Completed")
		if completed != true {
			for _, notify := range chatwork {
				roomid := notify.Roomid
				apikey := notify.Apikey
				body := "body="

				for to := range notify.To {
					body += "[To:" + strconv.Itoa(to) + "]"
				}

				body += "\n"
				body += "[info][title]Goemon AWS EC2 Schedule Event Notify[/title]"
				body += "Host : " + ec2event[0] + "\n"
				body += "Code : " + ec2event[1] + "\n"
				body += "Description : " + ec2event[2] + "\n"
				body += "NotBefore : " + ec2event[3] + " UTC [/info]"

				PostChatwork(roomid, apikey, body)
			}
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
		ec2events := GetEC2InstanceEvent(notifier, ec2service)

		switch notification {
		case "chatwork":
			NotifyChatwork(chatwork, ec2events)
		}
	}
}
