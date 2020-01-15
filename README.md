# Cloud Receptor Controller

The Cloud Receptor Controller is designed to receive work requests from internal
clients and route the work requests to the target receptor node which runs in
the customer's environment.

### Submitting A Work Request

A work request can be submitted by sending a work request message to the _/job_ endpoint.

```
  $ curl -v -X POST -d '{"account": "01", "recipient": "node-b", "payload": "fix_an_issue", "directive": "workername:action"}' -H "x-rh-identity:eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0=" http://localhost:8081/job
```

#### Work Request Message Format

```
  {
    "account": <account number>,
    "recipient": <node id of the receptor node>,
    "payload": <work reqeust payload>,
    "directive": <work request directive (for example: "workername:action">
  }
```

#### Work Request Response Message Format

```
  {
    "id": <uuid for the work request>
  }
```

### Checking the status of a connection

The status of a connection can be checked by sending a POST to the _/connection/status_ endpoint.


```
  $ curl -v -X POST -d '{"account": "02", "node_id": "1234"}' -H "x-rh-identity:eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0=" http://localhost:8081/connection/status
```

#### Connection Status Message Format

```
  {
    "account": <account number>,
    "node_id": <node id of the receptor node>,
  }
```

#### Connection Status Response Message Format

```
  {
    "status":"connected" or "disconnected"
  }
```

### Development

Install the project dependencies:

```
  $ make deps
```

#### Building

```
  $ make
```

#### Local testing with receptor

Start the server

```
  $ ./gateway
```

##### Receptor node configuration

Use the receptor's _--peer_ option to configure a receptor node to connect to the platform receptor controller.  The url used with the _--peer_ option should look like _ws://localhost:8080/wss/receptor-controller/gateway_.

The following command can be used to connect a local receptor node to a local receptor controller:

```
  $ python -m receptor  --debug -d  /tmp/node-b --node-id=node-b node --peer=ws://localhost:8080/wss/receptor-controller/gateway --peer=receptor://localhost:8889/
```

#### Testing

Run the unit tests:

```
  $ make test
```

Verbose output from the tests:

```
  $ TEST_ARGS="-v" make test
```

Run specific tests:

```
  $ TEST_ARGS="-run TestReadMessage -v" make test
```

Test coverage:

```
  $ make coverage
  ...
  ...
  file:///home/dehort/dev/go/src/github.com/RedHatInsights/platform-receptor-controller/coverage.html
```

Load the coverage.html file in your browser
