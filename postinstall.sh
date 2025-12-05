#!/bin/sh
set -e
systemctl daemon-reload
systemctl enable haltonika.service
systemctl start haltonika.service