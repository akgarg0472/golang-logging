#!/bin/bash

go build -o golang-logging cmd/golang-logging/main.go

if [ $? -ne 0 ]; then
    echo "Build failed. Exiting."
    exit 1
fi

if [ -f ".env" ]; then
    set -a
    source .env
    set +a
else
    echo ".env file not found. Continuing with existing environment variables."
fi

clear
clear
./golang-logging
