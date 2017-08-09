# ping
Ping project has been built to keep checking if the endpoints (some JSON REST API feeds) are responding 
with the HTTP Status Code 200. If not, it sends an email notification with some description about the error.

The main package also provides a few REST APIs endpoints so you can build a front end on top of it.

-----
TODO:
- Tests
- Refactor
- Error handling (stderr)
- Crash notifications