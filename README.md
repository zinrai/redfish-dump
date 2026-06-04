# redfish-dump

Crawl a Redfish service from its entry point and dump every reachable resource as JSON.

## What it does

redfish-dump starts at a Redfish entry point and recursively follows every
`@odata.id` reference it finds, fetching each resource over HTTP. It prints the
collected resources as a JSON array, where each element pairs a resource URI
with the raw JSON body the BMC returned. Requests run serially with a random
sleep between them to avoid overloading low-powered BMCs. Errors are recorded
per resource and do not stop the crawl.

## Usage

```
redfish-dump -base https://192.168.2.144 > dump.json
```

Specify credentials:

```
redfish-dump -base https://192.168.2.144 -user ADMIN -pass ADMIN > dump.json
```

Start from a specific path instead of the service root:

```
redfish-dump -base https://192.168.2.144 -entry /redfish/v1/UpdateService/FirmwareInventory/ > fw.json
```

Limit how deep the crawl follows links. With `-max-depth 1`, only the entry
resource and the resources it links to directly are fetched:

```
redfish-dump -base https://192.168.2.144 -entry /redfish/v1/UpdateService/FirmwareInventory/ -max-depth 1 > fw.json
```

Adjust the per-request sleep range for a fragile BMC:

```
redfish-dump -base https://192.168.2.144 -min-sleep 2s -max-sleep 8s > dump.json
```

Progress is written to stderr, so it stays out of the JSON on stdout:

```
[depth 0] GET /redfish/v1/
[depth 1] GET /redfish/v1/Systems
[depth 1] GET /redfish/v1/Chassis
...
done: 142 resources fetched (0 errors)
```

The output is a JSON array of `{uri, body}` objects. Extract fields with jq:

```
jq '.[].uri' dump.json
jq '[.[] | {uri, Name: .body.Name, Version: .body.Version}]' dump.json
```

### Options

```
-base       Redfish base URL, for example https://192.168.2.144 (required)
-user       BMC username (default ADMIN)
-pass       BMC password (default ADMIN)
-entry      Entry point path to start crawling (default /redfish/v1/)
-min-sleep  Minimum sleep between requests (default 1s)
-max-sleep  Maximum sleep between requests (default 5s)
-timeout    Per-request timeout (default 30s)
-max-depth  Maximum crawl depth, 0 means unlimited (default 0)
```

TLS certificate verification is disabled, since BMCs commonly use self-signed
certificates. Authentication uses HTTP Basic. Sending SIGINT (Ctrl-C) stops the
crawl and writes the resources collected so far.

## License

This project is licensed under the [MIT License](./LICENSE).
