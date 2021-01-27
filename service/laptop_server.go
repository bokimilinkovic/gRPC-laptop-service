package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"

	"github.com/google/uuid"
	"gitlab.techschool.pcbook/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxImageSize = 1 << 20

type LaptopServer struct {
	laptopStore LaptopStore
	imageStore  ImageStore
	ratingScore RatingStore
}

func NewLaptopServer(laptopStore LaptopStore, imageStore ImageStore, ratingStore RatingStore) *LaptopServer {
	return &LaptopServer{laptopStore, imageStore, ratingStore}
}

func (server *LaptopServer) CreateLaptop(
	ctx context.Context,
	req *pb.CreateLaptopRequest,
) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop()
	log.Printf("receive a craete-laptop request with id: %s", laptop.Id)

	if len(laptop.Id) > 0 {
		//check if valid uuid
		_, err := uuid.Parse(laptop.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "laptop ID is not a valid UUID:%v", err)
		}
	} else {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot generate a new laptop ID %v", err)
		}

		laptop.Id = id.String()
	}

	// Save the laptop to in-memory store
	err := server.laptopStore.Save(laptop)
	if err != nil {
		code := codes.Internal
		if errors.Is(err, ErrAlreadyExists) {
			code = codes.AlreadyExists
		}

		return nil, status.Errorf(code, "cannot save laptop to store %v", err)
	}

	log.Printf("Saved laptop with id: %s", laptop.Id)
	res := &pb.CreateLaptopResponse{
		Id: laptop.Id,
	}

	return res, nil
}

func (server *LaptopServer) SearchLaptop(
	req *pb.SearchLaptopRequest,
	stream pb.LaptopService_SearchLaptopServer,
) error {
	filter := req.GetFilter()
	log.Printf("Receive a search laptop request with filter: %v", filter)
	err := server.laptopStore.Search(
		stream.Context(),
		filter,
		func(laptop *pb.Laptop) error {
			res := &pb.SearchLaptopResponse{Laptop: laptop}
			err := stream.Send(res)
			if err != nil {
				return err
			}

			log.Printf("sent laptop with id:%s", laptop.GetId())
			return nil
		})

	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}

	return nil
}

func (server *LaptopServer) UploadImage(stream pb.LaptopService_UploadImageServer) error {
	req, err := stream.Recv()
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image infp"))
	}

	laptopID := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()
	log.Printf("receive an upload image request for laptop %s with image type %s", laptopID, imageType)

	laptop, err := server.laptopStore.Find(laptopID)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot find laptop %v", err))
	}
	if laptop == nil {
		return logError(status.Errorf(codes.InvalidArgument, "laptop %s doesnt exist", laptopID))
	}

	imageData := bytes.Buffer{}
	imageSize := 0

	for {
		log.Print("waitnig for receive more data")
		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive ckunk data %v", err))
		}

		chunk := req.GetChunkData()
		size := len(chunk)

		log.Printf("received a chunk with size %d", size)

		imageSize += size
		if imageSize > maxImageSize {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data %v", err))
		}

		_, err = imageData.Write(chunk)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data %v:", err))
		}
	}

	imageId, err := server.imageStore.Save(laptopID, imageType, imageData)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannotsave iamge to the store %v", err))
	}

	res := &pb.UploadImageResponse{
		Id:   imageId,
		Size: uint32(imageSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response %v", err))
	}

	log.Printf("Saved images with id: %s , size %d", imageId, imageSize)

	return nil
}

func (server *LaptopServer) RateLaptop(stream pb.LaptopService_RateLaptopServer) error {
	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		req, err := stream.Recv()
		if err != io.EOF {
			log.Print("no more data")
			break
		}

		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive stream request %v", err))
		}

		laptopID := req.GetLaptopId()
		score := req.GetScore()

		log.Printf("received a rate lapop request id = %s score %.2f", laptopID, score)
		found, err := server.laptopStore.Find(laptopID)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot find laptop %v", err))
		}

		if found == nil {
			return logError(status.Errorf(codes.NotFound, "Laptop id is not found  %s", laptopID))
		}

		rating, err := server.ratingScore.Add(laptopID, score)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot ad rating ot the store %v", err))
		}

		res := &pb.RateLaptopResponse{
			LaptopId:     laptopID,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		err = stream.Send(res)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "Cannot send response %v", err))
		}
	}

	return nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.Canceled, "deadline is exceeded"))
	default:
		return nil
	}
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}

	return err
}
