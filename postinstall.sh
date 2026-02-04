#!/bin/sh
set -e
systemctl daemon-reload
systemctl enable haltonika.service
systemctl restart haltonika.service
