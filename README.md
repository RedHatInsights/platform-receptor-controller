# Cloud Receptor Controller

The Cloud Receptor Controller is designed to receive work requests from internal
clients and route the work requests to the target receptor node which runs in
the customer's environment.

### Submitting A Work Request

A work request can be submitted by sending a work request message to the _/job_ endpoint.

```
  $ curl -v -X POST -d '{"account": "01", "recipient": "node-b", "payload": "fix_an_issue", "directive": "workername:action"}' -H "x-rh-identity:eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0=" http://localhost:9090/job
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
  $ curl -v -X POST -d '{"account": "02", "node_id": "1234"}' -H "x-rh-identity:eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0=" http://localhost:9090/connection/status
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

### Kafka Topics

The receptor controller will utilize two kafka topics:

  - Consume jobs from: `platform.receptor-controller.jobs`
  - Produce job responses to: `platform.receptor-controller.responses`

The response will contain the following information:

  - response key: MessageID
  - response value: json containing Account, Sender, MessageID, MessageType, Payload, Code, and InResponseTo
  - example response from a ping: `426f674d-42d5-11ea-bea3-54e1ad81c0b2: {"account":"0000001","sender":"node-a","message_id":"b959e674-4a2d-48be-88e3-8bb44000f040","code": 0,"message_type": "response","payload":"{\"initial_time\": \"2020-01-29T20:23:49,811218829+00:00\", \"response_time\": \"2020-01-29 20:23:49.830491\", \"active_work\": []}", "in_response_to": "426f674d-42d5-11ea-bea3-54e1ad81c0b2"}`

  The key for the message on the kafka topic will be the message id that was returned by the receptor-controller when the original message was submitted.  The _message\_id_ will be the message id as it is passed along from the receptor mesh network.  The _in\_response\_to_ will be the _in\_response\_to_ value as it is passed along from the receptor mesh network.  The key for the response message and the _in\_response\_to_ value can be used to match the response to the original message.

  The _code_ and _message\_type_ field as passed as is from the receptor mesh network.  The _code_ can be used to determine if the message was able to be handed over to a plugin and processed successfully (code=0) or if the plugin failed to process the message (code=1).  The _message\_type_ field can be either "response" or "eof".  If the value is "response", then the plugin has not completed processing and more responses are expected.  If the value is "eof", then the plugin has completed processing and no more responses are expected.

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
