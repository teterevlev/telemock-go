Telemock - Testing telego Bots

This provides a convenient local environment for testing bot logic with multiple accounts.
The bot handles commands and any messages similarly to [telego](https://github.com/mymmrac/telego).

## How to use

1. Import `telemock` instead of `telego`:
```
import (
    telego "github.com/teterevlev/telemock-go"
)
```
2. Init your bot as you usually do with telego.
3. Open `index.html` in your browser and run as many chats as you need.
4. Define message handlers as you usually do with telego.
5. This item is necessary because I have OCD and 5 is a nice number.

Check demo: [examples/simple/main.go](examples/simple/main.go)

## Dependencies

Setup dependenicies automatically
```
go mod tidy
```

## Testing

Run `go test ./...`


