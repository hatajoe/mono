#!/bin/bash

set -e

parse_arguments() {
	while getopts ":u:f:t:l:" opt; do
		case $opt in
			u) gh_user=$OPTARG ;;
			f) from=$OPTARG ;;
			t) to=$OPTARG ;;
			l) limit=$OPTARG ;;
			\?) echo "Invalid option: $OPTARG" ;;
		esac
	done
}

parse_arguments "$@"

if [ -z "$gh_user" ]; then
	gh_user=@me
fi

if [ -z "$from" ]; then
	echo "Please provide a start date with -f option"
	exit 1
fi

if [ -z "$to" ]; then
	echo "Please provide an end date with -t option"
	exit 1
fi

if [ -z "$limit" ]; then
	limit=1000
fi

echo "created pull-request count by repository"
gh search prs --author=${gh_user} --created=${from}..${to} --limit ${limit} | awk '{print $1}' | sort | uniq -c | sort -r

echo "reviewed pull-request count by repository"
gh search prs --reviewed-by=${gh_user} --created=${from}..${to} --limit ${limit} | awk '{print $1}' | sort | uniq -c | sort -r

echo "created issue count by repository"
gh search issues --author=${gh_user} --created=${from}..${to} --limit ${limit} | awk '{print $1}' | sort | uniq -c | sort -r

echo "commented issue count by repository"
gh search issues --commenter=${gh_user} --created=${from}..${to} --limit ${limit} | awk '{print $1}' | sort | uniq -c | sort -r
