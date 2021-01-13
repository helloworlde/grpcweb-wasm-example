package main

import (
	"context"
	"crypto/x509"
	"io"
	"io/ioutil"
	"syscall/js"

	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/grpc_channelz_v1"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"

	web "github.com/johanbrandhorst/grpcweb-wasm-example/proto"
)

// Build with Go WASM fork
//go:generate rm -f ./html/*
//go:generate bash -c "GOOS=js GOARCH=wasm go build -o ./html/test.wasm frontend.go"

//go:generate bash -c "cp $DOLLAR(go env GOROOT)/misc/wasm/wasm_exec.html ./html/index.html"
//go:generate bash -c "cp $DOLLAR(go env GOROOT)/misc/wasm/wasm_exec.js ./html/wasm_exec.js"
//go:generate bash -c "sed -i -e 's;</button>;</button>\\n\\t<div id=\"target\"></div>;' ./html/index.html"

// Integrate generated JS into a Go file for static loading.
//go:generate bash -c "go run assets_generate.go"

var document js.Value

type DivWriter js.Value

func (d DivWriter) Write(p []byte) (n int, err error) {
	node := document.Call("createElement", "div")
	node.Set("innerHTML", string(p))
	js.Value(d).Call("appendChild", node)
	return len(p), nil
}

func init() {
	document = js.Global().Get("document")
	div := document.Call("getElementById", "target")
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(DivWriter(div), ioutil.Discard, ioutil.Discard))
}

var serverPem = "-----BEGIN CERTIFICATE-----\nMIIFBTCCAu2gAwIBAgIRALMJszk83WyAmur9Jod2ur4wDQYJKoZIhvcNAQELBQAw\nEjEQMA4GA1UEChMHQWNtZSBDbzAgFw0xOTAyMjYyMjA1MzNaGA8yMTE5MDIwMjIy\nMDUzM1owEjEQMA4GA1UEChMHQWNtZSBDbzCCAiIwDQYJKoZIhvcNAQEBBQADggIP\nADCCAgoCggIBAKaKQfMZzT1/5rbyN1OkBD7HEu6Yo2zsyjzpRMge/FXPkWkcYRKh\n8FhXu7sntEtXraxpTmToFTYRfN87PEQ51X3mhKKhh0pnn0IZC7u0x5Uw8dwSCksA\nwua2asUi4kaq4b9mLy4t1F1eAmsd9Q/g22PSvBOQfFwde0hIoAnF9InzEsf+Nziw\nf4gS8LY1YU0K6X7uRORaLz2AkaOKR1GS43Th/YccLoRYf99duF3ZH+0HKOYwllBY\nbKRnvOJh8XkzazvHZ71TzDS0fL3iG6u3tvvtvdSa+ptobi7tKVd5Qa7ulwFASihM\nSMnrSjx/ZygKNcuRo1MLeoLBwCcjy/CAAExqfDJEdV8xT2enm5d4aylG9mH2IFsE\nP5rFzbIQaoDwZlKGUA8l42L1JxKW0my4t1NKat8qKSfSATA8/UwFZo5/0QwXB2Mp\nwh+nK8J3MtAuJxZMglGLovfAldpU6dEYvvb93AoBsbUGAxIW6QgHiiU8pNBk2QcM\ngmoOMnEFqrf0CBWP5tSoTKenrCIcG14nNrE3w0xS8uaqJxndLozTNAfdnP2vKsjP\n+YDQfDazoTthThivQ01RASw82O+bQ6B6bPa+QOwGEQjsym7x4MeYBUM835j6hNlB\nOK12fHmA45ezKEnYd4uhW6jteJOVNNh5pIk9zUR0kHq+JaGo8VODtSMnAgMBAAGj\nVDBSMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMB\nAf8EBTADAQH/MBoGA1UdEQQTMBGCCWxvY2FsaG9zdIcEfwAAATANBgkqhkiG9w0B\nAQsFAAOCAgEAIhBDmL4I13gPMEj8GVkjMn/Pn+kPk8jAwp5a0ESi3Ad0yrAnLM7h\n3AuWzLDfus4EpDnJ2oT+ohcx69yOwRlE3zfZVhSe+NKJRfndiBjgDKnnJ+Tyo9HH\nvLh4RQpa8npIqb75VkcdBdr++4VtfjH+bz/9VdAX8wrVRCGEZaHkkR8gmZKD/D9q\ndzAuVHd6ZQ6xH1y8Wg+TYRLfk8ENJgjvIFD0K2+k3EUN01P2TLWzdPL+POur5INO\nOzrE2QKekrs3JmJPoDSPNW8T2bGn0Hsms19JdlCf2XhU28XGq5CwVjlATyMeKUJ+\nFtMpuUHjLwykEAlxFrxpn9+Xq03t09G8Uz/joBH49OCRySwEuw4lPmEtvm1t7FdR\nuQWB30rmoS6NjwQzUdRO9n3zHZT+DsNc9cyaMxH3sycrOYU0d6pLGW/ZN5YInWK1\n/Cypgiu54PvP1+z3vounHtA9uvLHwi1u2BdhkdEM0QK45nlEYN/s4CmQooLhVaFK\nUAElGcmQ6vJwars2hnQOQNaJJQK2zFKhN5osNqMb/1FkVwMgorbHg/OJCWiagvKV\nTK2z6bwgrkcLBYDgey4xieQnhBXRJZ2+FwhWQk5ysr/wv1/2UvB3ldPz/b0r5JJj\nyus7qjHIVUHaHzlDwHPvVFtcoF4C+xaiJXuDBUWW5yYopn0toy8hEv8=\n-----END CERTIFICATE-----\n"

func main() {
	bytes := []byte(serverPem)
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(bytes)

	cert := credentials.NewClientTLSFromCert(roots, "")

	cc, err := grpc.Dial("localhost:10000", grpc.WithTransportCredentials(cert))

	// cc, err := grpc.Dial("")
	if err != nil {
		grpclog.Println(err)
		return
	}

	channelzClient := grpc_channelz_v1.NewChannelzClient(cc)
	servers, err := channelzClient.GetServers(context.Background(), &grpc_channelz_v1.GetServersRequest{})
	if err != nil {
		st := status.Convert(err)
		grpclog.Println(st.Code(), st.Message(), st.Details())
	} else {
		grpclog.Println(servers)
	}

	client := web.NewBackendClient(cc)
	resp, err := client.GetUser(context.Background(), &web.GetUserRequest{
		UserId: "1234",
	})
	if err != nil {
		st := status.Convert(err)
		grpclog.Println(st.Code(), st.Message(), st.Details())
	} else {
		grpclog.Println(resp)
	}
	resp, err = client.GetUser(context.Background(), &web.GetUserRequest{
		UserId: "123",
	})
	if err != nil {
		st := status.Convert(err)
		grpclog.Println(st.Code(), st.Message(), st.Details())
	} else {
		grpclog.Println(resp)
	}

	srv, err := client.GetUsers(context.Background(), &web.GetUsersRequest{
		NumUsers: 3,
	})
	if err != nil {
		st := status.Convert(err)
		grpclog.Println(st.Code(), st.Message(), st.Details())
	} else {
		for {
			user, err := srv.Recv()
			if err != nil {
				if err != io.EOF {
					st := status.Convert(err)
					grpclog.Println(st.Code(), st.Message(), st.Details())
				}
				break
			}

			grpclog.Println(user)
		}
	}

	grpclog.Println("finished")
}
