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
    - roomid specifies the group ID of the chatwork to notify
    - region describes the region to get maintenance events
    - profile describes the name of the AWS account profile for acquiring maintenance events
    - to specify the ID of the member you want to add a mention when making a notification
    - instances specifies the ID of the instance for which you want to get maintenance events

    ```
    notify:
      - 
        roomid: "356482"
        region: "ap-northeast-1"
        profile: "test_profile"
        to:
          - "1786285"
        instances:
          - "i-XXXXXXXXXXXXXXXXX"
          - "i-XXXXXXXXXXXXXXXXX"
    ```