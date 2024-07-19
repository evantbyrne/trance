package trance

import (
	"encoding/json"
	"io"
)

type JSONStreamer struct {
	Error error
	Value any
}

func (stream *JSONStreamer) Collect() ([]byte, error) {
	if stream.Error != nil {
		return []byte{}, stream.Error
	}

	return json.Marshal(stream.Value)
}

func (stream *JSONStreamer) OnError(callback func(error) error) *JSONStreamer {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *JSONStreamer) Then(callback func(any) error) *JSONStreamer {
	if stream.Error == nil {
		stream.Error = callback(stream.Value)
	}
	return stream
}

func (stream *JSONStreamer) Write(writer io.Writer) *JSONStreamer {
	if stream.Error == nil {
		var encoded []byte
		encoded, stream.Error = json.Marshal(stream.Value)
		if stream.Error != nil {
			return stream
		}
		_, stream.Error = writer.Write(encoded)
	}
	return stream
}
