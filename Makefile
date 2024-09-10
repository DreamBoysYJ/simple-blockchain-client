# Makefile for tmax-p2p

# Go 실행 명령
run: 
	go run .

# Go 빌드 명령
tmax-p2p:
	@echo "Building tmax-p2p..."
	go build -o tmax-p2p .

# Clean 명령 (삭제)
clean:
	@echo "Cleaning up..."
	rm -f tmax-p2p

# Install 명령 (글로벌 명령어로 설치)
install:
	@echo "Installing tmax-p2p to /usr/local/bin..."
	sudo mv tmax-p2p /usr/local/bin

# Uninstall 명령 (글로벌 명령어 삭제)
uninstall:
	@echo "Uninstalling tmax-p2p from /usr/local/bin..."
	sudo rm -f /usr/local/bin/tmax-p2p

# Default target (빌드 후 설치)
all: tmax-p2p install

# 장난
jangnan:
	@echo "HELLO"

