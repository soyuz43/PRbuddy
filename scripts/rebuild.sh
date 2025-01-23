#!/bin/bash
# Rebuild and reinstall
sudo rm /usr/bin/prbuddy-go
go build -o prbuddy-go
sudo mv prbuddy-go /usr/bin/