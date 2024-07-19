package trance

type MapStream struct {
	Error error
	Value map[string]any
}

func (stream *MapStream) Collect() (map[string]any, error) {
	return stream.Value, stream.Error
}

func (stream *MapStream) JSON() *JSONStreamer {
	return &JSONStreamer{
		Error: stream.Error,
		Value: stream.Value,
	}
}

func (stream *MapStream) OnError(callback func(error) error) *MapStream {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *MapStream) Then(callback func(map[string]any) error) *MapStream {
	if stream.Error == nil {
		stream.Error = callback(stream.Value)
	}
	return stream
}
