#!/bin/bash

set -e

aws sqs list-queues --output json | jq -r '.QueueUrls[]' | fzf | xargs -I{} aws sqs receive-message --queue-url {} --max-number-of-messages 10
