## NONAME

This is POC(proof of concept). For an application that I'm building, I needed a docker container that has a WebSocket
client
that can communicate directly with another WebSocket client(Browser) without a middle man.

### Features

- [ ] Create a configuration file.
- [x] Connect two WebSocket clients.
- [x] 1 is a simple client( web-browser) the other 1.
- [x] The second one is a docker container that is automatically created by this application. The application inside the
  docker container should have a WebSocket client that can communicate with
  the WebSocket server running on port 8080 of this application.

```bash
go run *.go
```