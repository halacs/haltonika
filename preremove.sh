#!/bin/sh
set -e

# Always stop and disable the service
systemctl stop haltonika.service
systemctl disable haltonika.service

