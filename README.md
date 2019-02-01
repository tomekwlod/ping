# ping
Ping project was created to help monitoring the endpoints by sending the ping signal. If an endpoint returns a status
code which is different to 200 then the response is stored in mongodb. Otherwise a simple information is also stored
in the db.

Within this project you can find two different apps:
- ping - is mainly described above, pinging the endopint and storing the info
- server - APIs for retrieving the data from the db

test1
