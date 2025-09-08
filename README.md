Telemock - Testing telego Bots

This provides a convenient local environment for testing bot logic with multiple accounts.
The bot handles commands and any messages similarly to [telego](https://github.com/mymmrac/telego).

![client](https://github.com/user-attachments/assets/3f645594-63ec-4ea9-9a8b-ef9de17d65db)

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

Run [demo:](examples/simple/main.go)
```
go run ./examples/simple
```
## Dependencies

Setup dependenicies automatically
```
go mod tidy
```

## Testing

Run `go test ./...`


