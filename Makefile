# Makefile for simple-blockchain-client

# Go 실행 명령
run: 
	go run .

# Go 빌드 명령
simple-blockchain-client:
	@echo "Building simple-blockchain-client..."
	go build -o simple-blockchain-client .

# Clean 명령 (삭제)
clean:
	@echo "Cleaning up..."
	
	rm -f simple-blockchain-client

# Install 명령 (글로벌 명령어로 설치)
install:
	@echo "Installing simple-blockchain-client to /usr/local/bin..."
	sudo mv simple-blockchain-client /usr/local/bin

# Uninstall 명령 (글로벌 명령어 삭제)
uninstall:
	@echo "Uninstalling simple-blockchain-client from /usr/local/bin..."
	sudo rm -f /usr/local/bin/simple-blockchain-client

# Reset 명령 (DB 및 모든 임시 파일 삭제)
reset:
	@echo "Resetting project state..."
	rm -f simple-blockchain-client

# Default target (빌드 후 설치)
all: simple-blockchain-client install

