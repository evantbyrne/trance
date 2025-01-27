package trance

import (
	"context"
	"database/sql"
)

type QueryResultStreamer[T any] struct {
	Error       error
	Result      sql.Result
	Value       *T
	WeaveConfig WeaveConfig
}

func (stream *QueryResultStreamer[T]) Collect() (sql.Result, *T, error) {
	return stream.Result, stream.Value, stream.Error
}

func (stream *QueryResultStreamer[T]) Guard(ctx context.Context) *MapStream {
	result := &MapStream{
		Error: stream.Error,
	}
	if result.Error == nil {
		result.Value, result.Error = Guard(ctx, stream.Value, stream.WeaveConfig)
	}
	return result
}

func (stream *QueryResultStreamer[T]) JSON() *JSONStreamer {
	result := &JSONStreamer{
		Error: stream.Error,
	}
	if result.Error == nil {
		weave := UseWith[T](stream.WeaveConfig)
		result.Value = weave.ToJsonMap(stream.Value)
	}
	return result
}

func (stream *QueryResultStreamer[T]) OnError(callback func(error) error) *QueryResultStreamer[T] {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *QueryResultStreamer[T]) Then(callback func(sql.Result, *T) error) *QueryResultStreamer[T] {
	if stream.Error == nil {
		stream.Error = callback(stream.Result, stream.Value)
	}
	return stream
}
