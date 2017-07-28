# SNMP Subagent
Simple subagent that translates snmp queries into API requests and responds.

## Registration
![flow](registration1.png)

1. An application can be registered with a PUT with the following content:
```
{
    "name": "App1",
    "discover_url": "http://localhost:8080/discover",
    "base_oid": "1.3.6.1.4.1.8072.100.1"
}
```

* `name`: just a name identifying the application
* `discover_url`: url of the application where all oids are specified
* `base_oid`: base oid of the application.

2. SNMP Subagent will try to get all the oids the application manages with a GET to `discover_url`.
The application must answer with something like:
```
{
    "status": "success",
    "errcode": "",
    "errmessage": "",
    "_embedded": { "oids": [
        {"oid": "1.3.6.1.4.1.8072.100.1.1.1.0", "type": "OctetString", "url": "http://localhost/aaa", "jsonpath": "$.first"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.2.0", "type": "OctetString", "url": "http://localhost/aaa", "jsonpath": "$.second"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.3.0", "type": "Integer", "url": "http://localhost/aaa", "jsonpath": "$.third"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.4.0", "type": "TimeTicks", "url": "http://localhost/aaa", "jsonpath": "$.uptime"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.5.0", "type": "IpAddress", "url": "http://localhost/bbb", "jsonpath": "$.first"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.6.0", "type": "Counter32", "url": "http://localhost/bbb", "jsonpath": "$.second"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.7.0", "type": "Gauge32", "url": "http://localhost/bbb", "jsonpath": "$.third"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.8.0", "type": "Counter64", "url": "http://localhost/bbb", "jsonpath": "$[\\"aaa.bbb.ccc\\"]"}
        ]
    }
}
```
The sequence of oids must have `oid`, `type`, `url` and `jsonpath`.

3. SNMP Subagent will try to register the `base_oid` via AgentX to the snmp daemon.


Another posibility is to register the oids directly with in the PUT, without specifying a `discover_url`:
![flow](registration2.png)

PUT content:
```
{
    "name": "App1",
    "base_oid": "1.3.6.1.4.1.8072.100.1",
    "oids": [
        {"oid": "1.3.6.1.4.1.8072.100.1.1.1.0", "type": "OctetString", "url": "http://localhost/aaa", "jsonpath": "$.first"},
        {"oid": "1.3.6.1.4.1.8072.100.1.1.2.0", "type": "OctetString", "url": "http://localhost/aaa", "jsonpath": "$.second"}
    ]
}
```

* `oids`: every oid is specified as a list.
* If `discover_url` is specified, the contents of the discover will override the contents of the PUT command.


## Flow
![flow](flow.png)

1. Somebody queries the server via snmp. For example: `snmpget -v2c -c sentinel 127.0.0.1 1.3.6.1.4.1.8072.100.1.1.1.0`
2. snmp daemon asks via AgentX to SNMP Subagent.
3. SNMP Subagent queries the proper application with a GET to the proper url: `http://localhost/aaa`
4. The application must answer with something like:
```
{
    "first": "App 1",
    "second": "1.5.12",
    "third": 1,
    "uptime": 123456
}
```
5. SNMP Subagent applies the proper jsonpath (`$.first`)
6. SNMP Subagent answers the snmp daemon via AgentX.
7. snmp daemon answers the request.

## Unregistration
![flow](unregistration.png)

1. An application can unregister by sending a DELETE command with the following content:
```
{
    "base_oid": "1.3.6.1.4.1.8072.100.1"
}
```


# Testing
## Quick test
`docker-compose up` the environment and test it (you should use `image: registry.intraway.com/sentinel/snmp_subagent-6_64:development`)
```
$ ./gradlew buildAllImages -Pstaging=true
$ cd environment
$ docker-compose up -d

# Get IP address of snmp subagent:
$ docker inspect -f '{{.NetworkSettings.Networks.environment_default.IPAddress}}' environment_snmp_subagent_1
172.18.0.3

# Register app1
$  curl -X PUT 'http://172.18.0.3:11111/applications/' -H "Content-Type: application/json" -d '{"name": "App 1", "base_oid": "1.3.6.1.4.1.8072.100.1", "discover_url": "http://app1-mock/discover"}'

# Register app2
$  curl -X PUT 'http://172.18.0.3:11111/applications/' -H "Content-Type: application/json" -d '{"name": "App 2", "base_oid": "1.3.6.1.4.1.8072.100.2", "discover_url": "http://app2-mock/discover"}'

$ snmpwalk -v2c -c sentinel 172.18.0.3:9161 1.3.6.1.4.1.8072.100
NET-SNMP-MIB::netSnmp.100.1.1.1.0 = STRING: "App 1"
NET-SNMP-MIB::netSnmp.100.1.1.2.0 = STRING: "1.5.12"
NET-SNMP-MIB::netSnmp.100.1.1.3.0 = INTEGER: 1
NET-SNMP-MIB::netSnmp.100.1.1.4.0 = Timeticks: (12345600) 1 day, 10:17:36.00
NET-SNMP-MIB::netSnmp.100.1.1.5.0 = IpAddress: 10.20.30.40
NET-SNMP-MIB::netSnmp.100.1.1.6.0 = Counter32: 4222
NET-SNMP-MIB::netSnmp.100.1.1.7.0 = Gauge32: 11333
NET-SNMP-MIB::netSnmp.100.1.1.8.0 = Counter64: 771
NET-SNMP-MIB::netSnmp.100.2.1.1.0 = STRING: "App 2"
NET-SNMP-MIB::netSnmp.100.2.1.2.0 = STRING: "3.14.15"
NET-SNMP-MIB::netSnmp.100.2.1.3.0 = INTEGER: 1
NET-SNMP-MIB::netSnmp.100.2.1.4.0 = Timeticks: (211600) 0:35:16.00

$ snmpget -v2c -c sentinel 172.18.0.3:9161 1.3.6.1.4.1.8072.100.1.1.1.0
NET-SNMP-MIB::netSnmp.100.1.1.1.0 = STRING: "App 1"

$ curl -X GET 'http://172.18.0.5/aaa'
{
    "first": "App 1",
    "second": "1.5.12",
    "third": 1,
    "uptime": 123456
}
```

## Test your application
* Build your application with the endpoints mentioned before. Example (assuming the IP is 10.20.30.40 and the port 12345):

```
# OIDs reposnses
$ curl -X GET 'http://10.20.30.40:12345/sentinel'
{
    "name": "My App",
    "version": "1.2.3",
}
```
* Tune-up your docker-compose.yml:

```
version: '2'

services:
  snmp_subagent:
    image: registry.intraway.com/sentinel/snmp_subagent-6_64:development
  my_app:
    image: ...
```
* Register your application:

```
$  curl -X PUT 'http://<IP snmp_subagent>:11111/applications/' -H "Content-Type: application/json" -d '{"name": "App 1", "base_oid": "1.3.6.1.4.1.8072.100.1","oids": [{"oid": "1.3.6.1.4.1.8072.100.1.1.1.0", "type": "OctetString", "url": "http://my_app:12345/sentinel", "jsonpath": "$.name"}, {"oid": "1.3.6.1.4.1.8072.100.1.1.2.0", "type": "OctetString", "url": "http://my_app:12345/sentinel", "jsonpath": "$.version"} ] }'
```
* Walk your app:

```
$ snmpwalk -v2c -c sentinel <IP snmp subagent>:9161 1.3.6.1.4.1.8072.100
```
* Mix it with coke


# Docker configuration
Though the docker image has a default config, it can be changed with environment variables:
```
version: '2'

services:
  snmp_subagent:
    image: registry.intraway.com/sentinel/snmp_subagent-6_64:master
    environment:
        AGENTX_PROTOCOL=tcp
        AGENTX_ADDRESSlocalhost:9705
        DISCOVER_TIMEOUT=100
        GET_TIMEOUT=100
        ENDPOINT_ADDRESS=0.0.0.0:11111
```
