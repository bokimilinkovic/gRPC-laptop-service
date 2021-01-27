package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"gitlab.techschool.pcbook/pb"
	"gitlab.techschool.pcbook/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const (
	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
	serverCert    = "cert/server-cert.pem"
	serverKey     = "cert/server-key.pem"
)

func loadTLDCredentials() (credentials.TransportCredentials, error) {
	pecClientCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pecClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	serverCert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(config), nil
}

func seedUsers(userStore service.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return err
	}

	return createUser(userStore, "user1", "secret", "user")
}

func accessibleRoles() map[string][]string {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"
	return map[string][]string{
		laptopServicePath + "CreateLaptop": {"admin"},
		laptopServicePath + "UploadImage":  {"admin"},
		laptopServicePath + "RateLaptop":   {"admin", "user"},
	}
}

func createUser(userStore service.UserStore, username, password, role string) error {
	user, err := service.NewUser(username, password, role)
	if err != nil {
		return err
	}

	return userStore.Save(user)
}

func runGRPCServer(
	authServer pb.AuthServiceServer,
	laptopServer pb.LaptopServiceServer,
	jwtManager *service.JWTManager,
	enableTLS bool,
	listener net.Listener,
) error {
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles())

	serverOptios := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	}

	if enableTLS {
		tlsCredentials, err := loadTLDCredentials()
		if err != nil {
			return err
		}

		serverOptios = append(serverOptios, grpc.Creds(tlsCredentials))
	}

	grpcServer := grpc.NewServer(serverOptios...)

	pb.RegisterAuthServiceServer(grpcServer, authServer)
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	reflection.Register(grpcServer)

	return grpcServer.Serve(listener)

}

func runRESETServer(
	authServer pb.AuthServiceServer,
	laptopServer pb.LaptopServiceServer,
	jwtManager *service.JWTManager,
	enableTLS bool,
	listener net.Listener,
	grpcEndpoint string,
) error {
	mux := runtime.NewServeMux()
	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//err := pb.RegisterAuthServiceHandlerServer(ctx, mux, authServer)
	err := pb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, dialOptions)
	if err != nil {
		return err
	}

	err = pb.RegisterLaptopServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, dialOptions)
	if err != nil {
		return err
	}

	log.Printf("Starting REST server at %s TLS = %t", listener.Addr().String(), enableTLS)
	if enableTLS {
		return http.ServeTLS(listener, mux, serverCert, serverKey)
	}
	return http.Serve(listener, mux)
}
func main() {
	port := flag.Int("port", 0, "the server port")
	enableTLS := flag.Bool("tls", false, "enable ssl/tls")
	serverType := flag.String("type", "grpc", "type of server grpc/rest")
	endpoint := flag.String("endpoint", "", "grpc endpoint")
	flag.Parse()
	log.Printf("start server on port %d TLS = %t", *port, *enableTLS)

	userStore := service.NewInMemoryUserStore()
	err := seedUsers(userStore)
	if err != nil {
		log.Fatal("cannot seed users")
	}
	jwtManager := service.NewJWTManager(secretKey, tokenDuration)
	authServer := service.NewAuthServer(userStore, jwtManager)

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	ratingStore := service.NewInMemoryRatingStore()
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)

	address := fmt.Sprintf("localhost:%d", *port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Cannot start server: ", err)
	}

	if *serverType == "grpc" {
		err = runGRPCServer(authServer, laptopServer, jwtManager, *enableTLS, listener)
	} else {
		err = runRESETServer(authServer, laptopServer, jwtManager, *enableTLS, listener, *endpoint)
	}
	if err != nil {
		log.Fatal("cannot start server ", err)
	}

}
