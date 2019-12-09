all:
	protoc -I protos --go_out=protos protos/*.proto
	go run gen_pbmsg.go
	go run gen_excel.go
