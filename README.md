# Cloud Receptor Controller

The Cloud Receptor Controller is designed to receive work requests from internal
clients and route the work requests to the target receptor node which runs in
the customer's environment.

### Submitting A Work Request

A work request can be submitted by sending a work request message to the _/job_ endpoint.

```
  $ curl -v -X POST -d '{"account": "01", "recipient": "node-b", "payload": "fix_an_issue", "directive": "workername:action"}' http://localhost:9090/job
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
