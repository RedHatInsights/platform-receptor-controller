import requests
import sys
import json

for i in range(int(sys.argv[1]), int(sys.argv[2])):
    account = "%02d" % i
    data = {"account": account, "node_id": "1234"}
    payload = json.dumps(data)
    r = requests.post("http://localhost:9090/management/disconnect", data=payload)

