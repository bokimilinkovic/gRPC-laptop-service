package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.techschool.pcbook/client"
	"gitlab.techschool.pcbook/pb"
	"gitlab.techschool.pcbook/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func createLaptop(laptopClient pb.LaptopServiceClient, laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			// not a big deal
			log.Print("Laptop already exists")
		} else {
			log.Fatal("can not create laptop", err)
		}

		return
	}

	log.Printf("Created laptop with id: %s", res.Id)
}

func searchLaptop(laptopClient pb.LaptopServiceClient, filter *pb.Filter) {
	log.Print("search filter: ", filter)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.SearchLaptop(ctx, req)
	if err != nil {
		log.Fatal("cannot search lkaptop", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal("Cannot receive response", err)
		}

		laptop := res.GetLaptop()
		log.Print(" - found", laptop.GetId())
		log.Print(" + brand", laptop.GetBrand())
		log.Print(" + name", laptop.GetName())
		log.Print(" + cpu cores:", laptop.GetCpu().GetNumberCores())
		log.Print(" + ram", laptop.GetRam().GetValue(), laptop.GetRam().GetUnit())
		log.Print(" + price: ", laptop.GetPriceUsd(), "usd")
	}
}

func testSearchLaptop(laptopClient *client.LaptopClient) {
	for i := 0; i < 10; i++ {
		laptopClient.CreateLaptop(sample.NewLaptop())
	}

	filter := &pb.Filter{
		MaxPriceUsd: 3000,
		MinCpuGhz:   2.5,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	laptopClient.SearchLaptop(filter)
}

func uploadImage(laptopClient pb.LaptopServiceClient, laptopID, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open image ifle", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.UploadImage(ctx)
	if err != nil {
		log.Fatal("cannot upload image", err)
	}

	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptopID,
				ImageType: filepath.Ext(imagePath),
			},
		},
	}

	err = stream.Send(req)
	if err != nil {
		log.Fatal("cannot send image info", err)
	}

	render := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	for {
		n, err := render.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer", err)
		}

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{ChunkData: buffer[:n]},
		}

		err = stream.Send(req)
		if err != nil {
			err2 := stream.RecvMsg(nil)
			log.Fatal("cannot send to server", err, err2)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response", err)
	}

	log.Printf("image upload with id %s size %d", res.GetId(), res.GetSize())
}

func rateLaptop(laptopClient pb.LaptopServiceClient, laptopIds []string, scores []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.RateLaptop(ctx)
	if err != nil {
		return fmt.Errorf("cannot rate laptop %v", err)
	}

	waitResponse := make(chan error)
	//go rutine to receive response
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				log.Print("no more responses")
				waitResponse <- nil
				return
			}
			if err != nil {
				waitResponse <- fmt.Errorf("ccannot receive stream respionses %v", err)
				return
			}

			log.Print("received response", res)
		}
	}()

	// send request

	for i, laptopId := range laptopIds {
		req := &pb.RateLaptopRequest{
			LaptopId: laptopId,
			Score:    scores[i],
		}

		err := stream.Send(req)
		if err != nil {
			return fmt.Errorf("cannot send stream request %v - %v", err, stream.RecvMsg(nil))
		}

		log.Print("send request", req)
	}

	err = stream.CloseSend()
	if err != nil {
		return fmt.Errorf("cannot close send %v", err)
	}

	err = <-waitResponse
	return err
}

func testCreateLaptop(laptopClient *client.LaptopClient) {
	laptopClient.CreateLaptop(sample.NewLaptop())
}

func testRateLaptop(laptopClient *client.LaptopClient) {
	n := 3
	laptopIds := make([]string, n)
	for i := 0; i < n; i++ {
		laptop := sample.NewLaptop()
		laptopIds[i] = laptop.GetId()
		laptopClient.CreateLaptop(laptop)
	}

	scores := make([]float64, n)
	for {
		fmt.Print("rate lapotp (y/n)")
		var answer string
		fmt.Scan(&answer)

		if strings.ToLower(answer) != "y" {
			break
		}

		for i := 0; i < n; i++ {
			scores[i] = sample.RandomLaptopScore()
		}

		err := laptopClient.RateLaptop(laptopIds, scores)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func testUploadImage(laptopClient *client.LaptopClient) {
	laptop := sample.NewLaptop()
	laptopClient.CreateLaptop(laptop)
	laptopClient.UploadImage(laptop.GetId(), "tmp/laptop.png")
}

const (
	username        = "admin1"
	password        = "secret"
	refreshDuration = 30 * time.Second
)

func authMethods() map[string]bool {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"
	return map[string]bool{
		laptopServicePath + "CreateLaptop": true,
		laptopServicePath + "UploadImage":  true,
		laptopServicePath + "RateLaptop":   true,
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	clinetCert, err := tls.LoadX509KeyPair("cert/client-cert.pem", "cert/client-key.pem")
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{clinetCert},
		RootCAs:      certPool,
	}

	return credentials.NewTLS(config), nil
}

func main() {
	serverAddress := flag.String("address", "", "the server address")
	enableTLS := flag.Bool("tls", false, "enable ssl/tls")
	flag.Parse()
	log.Printf("dial server %s, TLS = %t", *serverAddress, *enableTLS)

	transportOption := grpc.WithInsecure()
	if *enableTLS {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			log.Fatal("cannot load TLD credentials", err)
		}

		transportOption = grpc.WithTransportCredentials(tlsCredentials)
	}

	cc1, err := grpc.Dial(*serverAddress,transportOption)
	if err != nil {
		log.Fatal("cannot dial server")
	}

	authClient := client.NewAuthClient(cc1, username, password)
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), refreshDuration)
	if err != nil {
		log.Fatal("cannot create interceptor %v", err)
	}

	cc2, err := grpc.Dial(
		*serverAddress,
		transportOption,
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	)
	laptopClient := client.NewLaptopClient(cc2)
	testRateLaptop(laptopClient)
}
