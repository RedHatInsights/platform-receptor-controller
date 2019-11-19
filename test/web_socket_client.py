import asyncio
import aiohttp
import json
import random
import sys

from datetime import datetime, timezone

#TIMESTAMP = "2006-01-02T15:04:05Z07:00"
TIMESTAMP = datetime.now(timezone.utc).astimezone().isoformat()

def generate_route_message(id_, edges, seen):
    return {"cmd": "ROUTE",
            "id": id_,
            "edges": edges,
            "seen": seen}

def generate_cmd(cmd, id_, timestamp):
    return {"cmd": cmd,
            "id": id_,
            "expire_time": timestamp}


async def test_client(loop, account_number, node_id):
    basic_auth = aiohttp.BasicAuth(account_number, "imapassord")
    session = aiohttp.ClientSession(auth=basic_auth)
    ws = await session.ws_connect('http://localhost:8080/receptor-controller')

    edges = [["node-a", "node-b", 1]]
    seen = []

    async def periodic_writer():
        await asyncio.sleep(2)
        while True:
            print("writing")
            await ws.send_str(json.dumps(generate_route_message(node_id, edges, seen)))
            delay_msecs = random.randrange(100, 1000) / 1000
            await asyncio.sleep(delay_msecs)

    loop.create_task(periodic_writer())

    while True:
        print("here")
        msg = await ws.receive()
        print("there")
        #print("type(msg):", type(msg))
        #print("dir(msg):", dir(msg))

        if msg.type == aiohttp.WSMsgType.text:
            if msg.data[:2] == "HI":
                print("Gotta HI...")
                print("Sending HI...")
                await ws.send_str(json.dumps(generate_cmd("HI", node_id, TIMESTAMP)))
                #await ws.send_str("ROUTE:node-x:timestamp")
            if msg.data == 'close':
               print("CLOSE!")
               await ws.close()
               break
            else:
               print("recv:", msg.data)
            #   await ws.send_str(msg.data + '/answer')
        elif msg.type == aiohttp.WSMsgType.closed:
            print("WSMsgType.closed")
            break
        elif msg.type == aiohttp.WSMsgType.error:
            print("WSMsgType.error")
            break


if __name__ == "__main__":
    loop = asyncio.new_event_loop()

    coros = [test_client(loop, "%02d"%i, "node_%02d"%i) for i in range(int(sys.argv[1]), int(sys.argv[2]))]

    loop.run_until_complete(asyncio.wait(coros))

    #task = loop.create_task(test_client(loop, sys.argv[1], sys.argv[2]))
    #loop.run_until_complete(coros)
