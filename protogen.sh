protoc --proto_path=proto --go-grpc_out=./ --experimental_allow_proto3_optional proto/users.proto
protoc --proto_path=proto --go_out=./ --experimental_allow_proto3_optional proto/users.proto
protoc-go-inject-tag -input=internal/api/users.pb.go