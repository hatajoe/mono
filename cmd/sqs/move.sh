#!/bin/bash

set -e

aws sqs list-queues --output json | jq -r '.QueueUrls[]' | fzf \
	| xargs -I{} aws sqs get-queue-attributes --queue-url {} --attribute-names QueueArn --query 'Attributes.QueueArn' --output text \
	| xargs -I{} aws sqs start-message-move-task --source-arn {} --max-number-of-messages-per-second 1
