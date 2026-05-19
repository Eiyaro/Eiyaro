//go:generate protoc --go_out=. --go_opt=paths=source_relative  --go-vtproto_out=. --go-vtproto_opt=paths=source_relative,pool=truesize=true eiyarowalletd.proto

package pb

//og:generate protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative
