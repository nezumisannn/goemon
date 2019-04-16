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
	Notifer []Notifer `mapstructure:"notifier"`
}

// Notifer is a notification notifier
type Notifer struct {
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

// GetInstanceStatus is get list of instance status
func GetInstanceStatus(service *ec2.EC2, instance string) (result *ec2.DescribeInstanceStatusOutput, err error) {
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

// NotifyChatwork is Notify Chatwork
func NotifyChatwork(chatwork []ChatworkNotifer, host string, code string, description string, notbefore string) {
	api := "https://api.chatwork.com/v2/rooms/"
	path := "/messages"
	completed := strings.Contains(description, "Completed")

	if completed != true {
		for _, notify := range chatwork {
			roomid := notify.Roomid
			apikey := notify.Apikey
			url := api + roomid + path

			body := "body="

			for to := range notify.To {
				body += "[To:" + strconv.Itoa(to) + "]"
			}

			body += "\n"
			body += "[info][title]Goemon AWS EC2 Schedule Event Notify[/title]"
			body += "Host : " + host + "\n"
			body += "Code : " + code + "\n"
			body += "Description : " + description + "\n"
			body += "NotBefore : " + notbefore + " UTC [/info]"

			request, err := http.NewRequest("POST", url, bytes.NewBufferString(body))

			if err != nil {
				log.Fatal(err)
			}

			request.Header.Add("X-ChatWorkToken", apikey)
			request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			response, err := http.DefaultClient.Do(request)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(response)
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

	for _, notifier := range config.Notifer {
		region := notifier.Region
		profile := notifier.Profile
		notification := notifier.Notification
		chatwork := notifier.Chatwork

		ec2service := ConnectEC2(session, region, profile)
		for _, ec2 := range notifier.EC2 {
			for _, instance := range ec2.Instances {
				statuses, err := GetInstanceStatus(ec2service, instance)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				for _, status := range statuses.InstanceStatuses {
					for _, events := range status.Events {
						code := *events.Code
						description := *events.Description
						notbefore := events.NotBefore.Format(time.ANSIC)

						switch notification {
						case "chatwork":
							NotifyChatwork(chatwork, instance, code, description, notbefore)
						}
					}
				}
			}
		}
	}
}
