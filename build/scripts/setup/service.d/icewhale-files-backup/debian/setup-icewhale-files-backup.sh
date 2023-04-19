#!/bin/bash
###
# @Author: LinkLeong link@icewhale.org
# @Date: 2022-08-25 11:41:22
 # @LastEditors: LinkLeong
 # @LastEditTime: 2022-08-31 17:54:17
 # @FilePath: /CasaOS/build/scripts/setup/service.d/casaos/debian/setup-casaos.sh
# @Description:

# @Website: https://www.casaos.io
# Copyright (c) 2022 by icewhale, All Rights Reserved.
###

set -e

APP_NAME="icewhale-files-backup"
APP_SHORT_NAME="files-backup"

# copy config files
CONF_PATH=/etc/icewhale
CONF_FILE=${CONF_PATH}/${APP_SHORT_NAME}.conf
CONF_FILE_SAMPLE=${CONF_PATH}/${APP_SHORT_NAME}.conf.sample

if [ ! -f "${CONF_FILE}" ]; then
    echo "Initializing config file..."
    cp -v "${CONF_FILE_SAMPLE}" "${CONF_FILE}"
fi

# enable service (without starting)
echo "Enabling service..."
systemctl enable --force --no-ask-password "${APP_NAME}.service"
