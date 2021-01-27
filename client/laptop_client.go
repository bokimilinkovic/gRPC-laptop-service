package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gitlab.techschool.pcbook/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LaptopClient struct {
	service pb.LaptopServiceClient
}

func NewLaptopClient(cc *grpc.ClientConn) *LaptopClient {
	service := pb.NewLaptopServiceClient(cc)
	return &LaptopClient{service}
}

func (laptopClient *LaptopClient) CreateLaptop(laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	res, err := laptopClient.service.CreateLaptop(context.Background(), req)
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

func (laptopClient *LaptopClient) SearchLaptop(filter *pb.Filter) {
	log.Print("search filter: ", filter)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.service.SearchLaptop(ctx, req)
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

func (laptopClient *LaptopClient) RateLaptop(laptopIds []string, scores []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.service.RateLaptop(ctx)
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

func (laptopClient *LaptopClient) UploadImage(laptopID, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open image ifle", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.service.UploadImage(ctx)
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
