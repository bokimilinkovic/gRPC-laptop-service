package serializer

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
)

// WriteProtobufToBinaryFile writes a protocol buffer message to binary file
func WriteProtobufToBinaryFile(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("Can not marshal proto message to binary %w", err)
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// ReadProtobufFromBinaryFile reads a protocol buffer message from binary file
func ReadProtobufFromBinaryFile(filename string, message proto.Message) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = proto.Unmarshal(data, message)
	if err != nil {
		return err
	}

	return nil
}

func WriteProtobufToJsonFile(message proto.Message, filename string) error {
	data, err := ProtobufToJSON(message)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, []byte(data), 0644)
	if err != nil {
		return err
	}
	return nil
}
