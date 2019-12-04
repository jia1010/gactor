gen:
	protoc -I rpc_proto --go_out=plugins=grpc:rpc_proto rpc_proto/*.proto
