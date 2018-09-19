GO = go
ENV = CGO_ENABLED=0
LDFLAGS = -ldflags '-s -w'

macos:
	GOOS=darwin $(ENV) $(GO) build $(LDFLAGS) -o big-rainbow .

linux:
	GOOS=linux $(ENV) $(GO) build $(LDFLAGS) -o big-rainbow .

windows:
	GOOS=windows $(ENV) $(GO) build $(LDFLAGS) -o big-rainbow.exe .
