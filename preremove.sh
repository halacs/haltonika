#!/bin/sh
set -e
systemctl stop haltonika.service
systemctl disable haltonika.service
