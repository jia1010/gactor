all:
	protoc -I protos --go_out=gen/pt protos/*.proto
	go run gen_pbmsg.go
	go run gen_excel.go
