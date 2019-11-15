Setup python virtualenv
=======================
python3 -m venv aiohttp_env
source aiohttp_env/bin/activate
pip install -r requirements.txt 


Start 10 websocket client connections
=====================================
 python web_socket_client.py 1 10


Stop the 10 websocket client connections
=====================================
python shutdown_connections.py 1 10
