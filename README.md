# GOEMON

## Overview
- GOEMON is a CLI tool to get AWS maintenance events and notify any chat.
- You need a configuration file in yaml format when you run GOEMON.

## Commands
- help

    ```
    # goemon help
    ```

- check

    ```
    # goemon check
    ```

## Options
- config

    ```
    # goemon check --config config_file_path
    ```

## Usage
- Notify chat on AWS maintenance events

    ```
    # goemon check --config goemon.config.yaml
    ```

- The following configuration file is required to check
    - notification describes the chat tool that sends the notification
    - region describes the region to get maintenance events
    - profile describes the name of the AWS account profile for acquiring maintenance events
    - asumerole specifies whether to obtain temporary credentials from sts and connect
    - rolearn specifies the ARN of the IAM Role that is specified when performing the assumeerole
    - chatwork describes the information needed to notify chatwork
        - roomid specifies the group ID of the chatwork to notify
        - apikey describes the API key for sending notifications to chatwork. Must be issued in advance
        - to specify the ID of the member you want to add a mention when making a notification
    - ec2 describes the information of EC2 instance
        - instances specifies the ID of the ec2 instance for which you want to get maintenance events
    - rds describes the information of RDS instance
        - instance describes the ARN of the RDS instance that gets pending actions

    ```
    notify:
      - 
        notification: "chatwork" 
        region: "ap-northeast-1"
        profile: "test_profile"
        assumerole: true
        rolearn: "XXXXXXXX"
        chatwork:
          roomid: "1234567"
          apikey: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
          to:
            - "1234567"
            - "7654321"
        ec2:
          -
            instances:
              - "i-XXXXXXXXXXXXXXXXX"
              - "i-XXXXXXXXXXXXXXXXX"
        rds:
          -
            instances:
              - arn:aws:rds:ap-northeast-1:123456789123:cluster:XXXXXXXX
              - arn:aws:rds:ap-northeast-1:123456789123:db:XXXXXXXX
    ```