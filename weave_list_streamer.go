package trance

import (
	"context"
)

type WeaveListStreamer[T any] struct {
	Error        error
	Values       []*T
	WeaveConfigs []WeaveConfig
}

func (stream *WeaveListStreamer[T]) Collect() ([]*T, error) {
	return stream.Values, stream.Error
}

func (stream *WeaveListStreamer[T]) First() *WeaveStreamer[T] {
	result := &WeaveStreamer[T]{
		Error:        stream.Error,
		WeaveConfigs: stream.WeaveConfigs,
	}
	if result.Error == nil {
		if len(stream.Values) > 0 {
			result.Value = stream.Values[0]
		} else {
			result.Error = ErrorNotFound{}
		}
	}
	return result
}

func (stream *WeaveListStreamer[T]) ForEach(callback func(i int, value *T) error) *WeaveListStreamer[T] {
	i := 0
	for stream.Error == nil && i < len(stream.Values) {
		stream.Error = callback(i, stream.Values[i])
		i++
	}
	return stream
}

func (stream *WeaveListStreamer[T]) Guard(ctx context.Context) *MapListStream {
	result := &MapListStream{
		Error: stream.Error,
	}
	if result.Error == nil {
		result.Values, result.Error = GuardList(ctx, stream.Values, stream.WeaveConfigs...)
	}
	return result
}

func (stream *WeaveListStreamer[T]) JSON() *JSONStreamer {
	result := &JSONStreamer{
		Error: stream.Error,
	}
	if result.Error == nil {
		weave := Use[T](stream.WeaveConfigs...)
		values := make([]map[string]any, 0)
		for _, row := range stream.Values {
			values = append(values, weave.ToJsonMap(row))
		}
		result.Value = values
	}
	return result
}

func (stream *WeaveListStreamer[T]) Map(callback func(i int, value *T) (*T, error)) *WeaveListStreamer[T] {
	i := 0
	for stream.Error == nil && i < len(stream.Values) {
		stream.Values[i], stream.Error = callback(i, stream.Values[i])
		i++
	}
	return stream
}

func (stream *WeaveListStreamer[T]) OnError(callback func(error) error) *WeaveListStreamer[T] {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *WeaveListStreamer[T]) Reduce(callback func(i int, value *T, acc *T) (*T, error)) *WeaveStreamer[T] {
	result := &WeaveStreamer[T]{
		Error:        stream.Error,
		WeaveConfigs: stream.WeaveConfigs,
	}
	i := 0
	for result.Error == nil && i < len(stream.Values) {
		result.Value, result.Error = callback(i, stream.Values[i], result.Value)
		i++
	}
	return result
}

func (stream *WeaveListStreamer[T]) Then(callback func([]*T) error) *WeaveListStreamer[T] {
	if stream.Error == nil {
		stream.Error = callback(stream.Values)
	}
	return stream
}
