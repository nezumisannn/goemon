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
	"github.com/spf13/viper"
)

//CheckFlag is struct
type CheckFlag struct {
	Config string
}

type Config struct {
	Notify []Notify `yaml:notify`
}

type Notify struct {
	RoomID    string   `yaml:roomid`
	Region    string   `yaml:region`
	Profile   string   `yaml:profile`
	To        []string `yaml:toport`
	Instances []string `yaml:instances`
}

var config Config

// Check is check AWS infomation.
func Check(flag *CheckFlag) {

	viper.SetConfigFile(flag.Config)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	for _, i := range config.Notify {
		roomid := i.RoomID
		region := i.Region
		profile := i.Profile
		to := i.To
		instances := i.Instances

		cred := credentials.NewSharedCredentials("", profile)
		svc := ec2.New(
			sess,
			aws.NewConfig().WithRegion(region).WithCredentials(cred),
		)

		for _, j := range instances {
			params := &ec2.DescribeInstanceStatusInput{
				InstanceIds: []*string{
					aws.String(j),
				},
			}

			resp, err := svc.DescribeInstanceStatus(params)
			if err != nil {
				panic(err)
			}

			for _, status := range resp.InstanceStatuses {
				for _, event := range status.Events {
					Code := event.Code
					Description := event.Description
					NotBefore := event.NotBefore

					Iscompleted := strings.Contains(*Description, "Completed")

					if Iscompleted != true {
						url := "https://api.chatwork.com/v2/rooms/" + roomid + "/messages"

						param := "body="

						for _, k := range to {
							param += "[To:" + k + "]"
						}

						param += "\n"
						param += "[info][title]Goemon AWS EC2 Schedule Event Notify[/title]"
						param += "Host : " + j + "\n"
						param += "Code : " + *Code + "\n"
						param += "Description : " + *Description + "\n"
						param += "NotBefore : " + NotBefore.Format(time.ANSIC) + " UTC [/info]"

						request, error := http.NewRequest("POST", url, bytes.NewBufferString(param))
						if error != nil {
							log.Fatal(error)
						}

						apiKey := "5d885ec366178e84854d76e4a19f5784"
						request.Header.Add("X-ChatWorkToken", apiKey)
						request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
						response, error := http.DefaultClient.Do(request)
						if error != nil {
							log.Fatal(error)
						}
						fmt.Println(response)
					}
				}
			}
		}
	}
}
