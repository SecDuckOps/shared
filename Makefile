.PHONY: proto clean

PROTO_DIR := proto
GEN_DIR := proto/gen

proto:
	@echo "Generating Go code from proto files..."
	protoc \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/scan/v1/scan.proto \
		$(PROTO_DIR)/auth/v1/auth.proto
	@echo "Done."

clean:
	rm -rf $(GEN_DIR)/**/*.pb.go
