#!/bin/sh
set -e

# Always stop and disable the service
systemctl stop haltonika.service
systemctl disable haltonika.service

case "$1" in
    remove)
        ;;

    purge)
        # Logical "Purge": Clean up everything
        echo "Purging all data and configurations..."
        #rm -rf /etc/haltonika/haltonika.met
        rm -rf /etc/haltonika
        ;;

    upgrade|failed-upgrade|abort-install|abort-upgrade|disappear)
        ;;

    *)
        exit 0
        ;;
esac
