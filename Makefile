NAME=deployer
HARDWARE=$(shell uname -m)
VERSION=0.0.1

build:
	mkdir -p stage
	go build -o stage/deployer
	docker build -t deployer .

release:
	rm -rf release
	mkdir release
	GOOS=linux go build -o release/$(NAME)
	cd release && tar -zcf $(NAME)_$(VERSION)_linux_$(HARDWARE).tgz $(NAME)
	
	rm release/$(NAME)
	echo "$(VERSION)" > release/version
	echo "progrium/$(NAME)" > release/repo
	gh-release

.PHONY: release