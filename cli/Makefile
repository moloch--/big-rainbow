all: macos linux windows

macos:
	GOOS=darwin go build -o bigrainbow .

linux:
	GOOS=linux go build -o bigrainbow.elf .

windows:
	GOOS=windows go build -o bigrainbow.exe .

clean:
	rm -f bigrainbow
	rm -f bigrainbow.elf
	rm -f bigrainbow.exe