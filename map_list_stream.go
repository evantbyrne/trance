package trance

type MapListStream struct {
	Error  error
	Values []map[string]any
}

func (stream *MapListStream) Collect() ([]map[string]any, error) {
	return stream.Values, stream.Error
}

func (stream *MapListStream) First() *MapStream {
	result := &MapStream{
		Error: stream.Error,
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

func (stream *MapListStream) ForEach(callback func(i int, value map[string]any) error) *MapListStream {
	i := 0
	for stream.Error == nil && i < len(stream.Values) {
		stream.Error = callback(i, stream.Values[i])
		i++
	}
	return stream
}

func (stream *MapListStream) JSON() *JSONStreamer {
	return &JSONStreamer{
		Error: stream.Error,
		Value: stream.Values,
	}
}

func (stream *MapListStream) Map(callback func(i int, value map[string]any) (map[string]any, error)) *MapListStream {
	i := 0
	for stream.Error == nil && i < len(stream.Values) {
		stream.Values[i], stream.Error = callback(i, stream.Values[i])
		i++
	}
	return stream
}

func (stream *MapListStream) OnError(callback func(error) error) *MapListStream {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *MapListStream) Reduce(callback func(i int, value map[string]any, acc map[string]any) (map[string]any, error)) *MapStream {
	result := &MapStream{
		Error: stream.Error,
	}
	i := 0
	for result.Error == nil && i < len(stream.Values) {
		result.Value, result.Error = callback(i, stream.Values[i], result.Value)
		i++
	}
	return result
}

func (stream *MapListStream) Then(callback func(stream *MapListStream) error) *MapListStream {
	if stream.Error == nil {
		stream.Error = callback(stream)
	}
	return stream
}
