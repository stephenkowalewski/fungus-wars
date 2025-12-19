# Fungus Wars

Online multiplayer board game that is heavily inspired by [https://github.com/tristanstcyr/MacFungus-1.0](https://github.com/tristanstcyr/MacFungus-1.0).
See `how_to_play.md` for the game rules.

This repo contains both the HTTP server and the client-side files for use in a web browser.

# Server setup

Ensure that `git` and `go` (>= 1.24) are installed.

Checkout the repo, and run the following:
```
go generate
go run .
```

&#9432; This will start a server listening on port 8080. Run with the `--help` flag to see other options.

On Unix-like systems, the included Makefile can be used to build artifacts such as containers or RPMs.

# Testing notes

Players are identified by cookies. To play as multiple players on a single device, each player needs their own set of cookies. You can use a unique domain name (e.g. with a hostfile) or web browser profile for each local player.

The `--debug` flag and `/debug` endpoint are helpful to avoid navigating through the lobby every time the server restarts.
