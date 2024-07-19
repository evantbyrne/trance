package trance

import "context"

type WeaveStreamer[T any] struct {
	Error        error
	Value        *T
	WeaveConfigs []WeaveConfig
}

func (stream *WeaveStreamer[T]) Collect() (*T, error) {
	return stream.Value, stream.Error
}

func (stream *WeaveStreamer[T]) Guard(ctx context.Context) *MapStream {
	result := &MapStream{
		Error: stream.Error,
	}
	if result.Error == nil {
		result.Value, result.Error = Guard(ctx, stream.Value, stream.WeaveConfigs...)
	}
	return result
}

func (stream *WeaveStreamer[T]) JSON() *JSONStreamer {
	result := &JSONStreamer{
		Error: stream.Error,
	}
	if result.Error == nil {
		weave := Use[T](stream.WeaveConfigs...)
		result.Value = weave.ToJsonMap(stream.Value)
	}
	return result
}

func (stream *WeaveStreamer[T]) OnError(callback func(error) error) *WeaveStreamer[T] {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *WeaveStreamer[T]) Then(callback func(*T) error) *WeaveStreamer[T] {
	if stream.Error == nil {
		stream.Error = callback(stream.Value)
	}
	return stream
}
