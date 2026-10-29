.PHONY: proto up

proto:
	protoc -I=proto \
		--go_out=bot/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=bot/proto \
		--go-grpc_opt=paths=source_relative \
		proto/subscriptions.proto

up:
	sudo docker compose up --build --abort-on-container-failure