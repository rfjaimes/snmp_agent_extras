#!/bin/sh

set -e

APP_ROOT="/opt/iway/sentinel/snmp_subagent"

if [ ! -z "$SNMP_SUBAGENT_ROOT_PATH" ];then
    echo "Replacing SNMP Subagent root path with $SNMP_SUBAGENT_ROOT_PATH"
    APP_ROOT=$SNMP_SUBAGENT_ROOT_PATH
fi

SNMPD_CONF_FILE=$APP_ROOT/conf/snmpd.conf

CONF_FILE=$APP_ROOT/conf/config.yaml

if [ ! -z "$AGENTX_PROTOCOL" ];then
    echo "Replacing agentx_protocol with $AGENTX_PROTOCOL"
    sed -i "s/agentx_protocol: \".*\"/agentx_protocol: \"$AGENTX_PROTOCOL\"/" $CONF_FILE
fi

if [ ! -z "$AGENTX_ADDRESS" ];then
    echo "Replacing agentx_address with $AGENTX_ADDRESS"
    sed -i "s/agentx_address: \".*\"/agentx_address: \"$AGENTX_ADDRESS\"/" $CONF_FILE
fi

if [ ! -z "$DISCOVER_TIMEOUT" ];then
    echo "Replacing discover_timeout with $DISCOVER_TIMEOUT"
    sed -i "s/discover_timeout: \".*\"/discover_timeout: \"$DISCOVER_TIMEOUT\"/" $CONF_FILE
fi

if [ ! -z "$DB_FILE" ];then
    echo "Replacing db_file with $DB_FILE"
    sed -i "s/db_file: \".*\"/db_file: \"$DB_FILE\"/" $CONF_FILE
fi

if [ ! -z "$GET_TIMEOUT" ];then
    echo "Replacing get_timeout with $GET_TIMEOUT"
    sed -i "s/get_timeout: \".*\"/get_timeout: \"$GET_TIMEOUT\"/" $CONF_FILE
fi

if [ ! -z "$ENDPOINT_ADDRESS" ];then
    echo "Replacing endpoint_address with $ENDPOINT_ADDRESS"
    sed -i "s/endpoint_address: \".*\"/endpoint_address: \"$ENDPOINT_ADDRESS\"/" $CONF_FILE
fi

if [ ! -z "$DISABLE_SNMPD_IPV6" ];then
    if [ $DISABLE_SNMPD_IPV6 == "yes" ]; then
        echo "Disabling IPv6 for daemon snmp"
        sed -i "s/\(agentAddress [^,]*\),.*/\1/" $SNMPD_CONF_FILE
    else
        echo "DISABLE_SNMPD_IPV6 was set ('$DISABLE_SNMPD_IPV6') but it was not 'yes'. Skipping"
    fi
fi


echo "Launching snmpd"
/etc/init.d/iwsnmpd573 start

echo "Launching supervisor"
supervisord -c /etc/supervisord.conf &


sleep 5
tail -F /var/log/supervisor/supervisord.log /var/log/iway/snmp-subagent/snmp-subagent.log
