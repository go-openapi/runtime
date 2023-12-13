package runtime

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByteStreamConsumer(t *testing.T) {
	const expected = "the data for the stream to be sent over the wire"
	consumer := ByteStreamConsumer()

	t.Run("can consume as a ReaderFrom", func(t *testing.T) {
		var dest = &readerFromDummy{}
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), dest))
		assert.Equal(t, expected, dest.b.String())
	})

	t.Run("can consume as a Writer", func(t *testing.T) {
		dest := &closingWriter{}
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), dest))
		assert.Equal(t, expected, dest.String())
	})

	t.Run("can consume as a string", func(t *testing.T) {
		var dest string
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		assert.Equal(t, expected, dest)
	})

	t.Run("can consume as a binary unmarshaler", func(t *testing.T) {
		var dest binaryUnmarshalDummy
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		assert.Equal(t, expected, dest.str)
	})

	t.Run("can consume as a binary slice", func(t *testing.T) {
		var dest []byte
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		assert.Equal(t, expected, string(dest))
	})

	t.Run("can consume as a type, with underlying as a binary slice", func(t *testing.T) {
		type binarySlice []byte
		var dest binarySlice
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		assert.Equal(t, expected, string(dest))
	})

	t.Run("can consume as a type, with underlying as a string", func(t *testing.T) {
		type aliasedString string
		var dest aliasedString
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		assert.Equal(t, expected, string(dest))
	})

	t.Run("can consume as an interface with underlying type []byte", func(t *testing.T) {
		var dest interface{} = []byte{}
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		asBytes, ok := dest.([]byte)
		require.True(t, ok)
		assert.Equal(t, expected, string(asBytes))
	})

	t.Run("can consume as an interface with underlying type string", func(t *testing.T) {
		var dest interface{} = "x"
		require.NoError(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		asString, ok := dest.(string)
		require.True(t, ok)
		assert.Equal(t, expected, asString)
	})

	t.Run("with CloseStream option", func(t *testing.T) {
		t.Run("wants to close stream", func(t *testing.T) {
			closingConsumer := ByteStreamConsumer(ClosesStream)
			var dest bytes.Buffer
			r := &closingReader{b: bytes.NewBufferString(expected)}

			require.NoError(t, closingConsumer.Consume(r, &dest))
			assert.Equal(t, expected, dest.String())
			assert.EqualValues(t, 1, r.calledClose)
		})

		t.Run("don't want to close stream", func(t *testing.T) {
			nonClosingConsumer := ByteStreamConsumer()
			var dest bytes.Buffer
			r := &closingReader{b: bytes.NewBufferString(expected)}

			require.NoError(t, nonClosingConsumer.Consume(r, &dest))
			assert.Equal(t, expected, dest.String())
			assert.EqualValues(t, 0, r.calledClose)
		})
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("passing in a nil slice will result in an error", func(t *testing.T) {
			var dest *[]byte
			require.Error(t, consumer.Consume(bytes.NewBufferString(expected), &dest))
		})

		t.Run("passing a non-pointer will result in an error", func(t *testing.T) {
			var dest []byte
			require.Error(t, consumer.Consume(bytes.NewBufferString(expected), dest))
		})

		t.Run("passing in nil destination result in an error", func(t *testing.T) {
			require.Error(t, consumer.Consume(bytes.NewBufferString(expected), nil))
		})

		t.Run("a reader who results in an error, will make it fail", func(t *testing.T) {
			t.Run("binaryUnmarshal case", func(t *testing.T) {
				var dest binaryUnmarshalDummy
				require.Error(t, consumer.Consume(new(nopReader), &dest))
			})

			t.Run("[]byte case", func(t *testing.T) {
				var dest []byte
				require.Error(t, consumer.Consume(new(nopReader), &dest))
			})
		})

		t.Run("the reader cannot be nil", func(t *testing.T) {
			var dest []byte
			require.Error(t, consumer.Consume(nil, &dest))
		})
	})
}

func BenchmarkByteStreamConsumer(b *testing.B) {
	const bufferSize = 1000
	expected := make([]byte, bufferSize)
	_, err := rand.Read(expected)
	require.NoError(b, err)
	consumer := ByteStreamConsumer()
	input := bytes.NewReader(expected)

	b.Run("with writer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var dest bytes.Buffer
		for i := 0; i < b.N; i++ {
			err = consumer.Consume(input, &dest)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = input.Seek(0, io.SeekStart)
			dest.Reset()
		}
	})
	b.Run("with BinaryUnmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var dest binaryUnmarshalDummyZeroAlloc
		for i := 0; i < b.N; i++ {
			err = consumer.Consume(input, &dest)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = input.Seek(0, io.SeekStart)
		}
	})
	b.Run("with string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var dest string
		for i := 0; i < b.N; i++ {
			err = consumer.Consume(input, &dest)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = input.Seek(0, io.SeekStart)
		}
	})
	b.Run("with []byte", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var dest []byte
		for i := 0; i < b.N; i++ {
			err = consumer.Consume(input, &dest)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = input.Seek(0, io.SeekStart)
		}
	})
	b.Run("with aliased string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		type aliasedString string
		var dest aliasedString
		for i := 0; i < b.N; i++ {
			err = consumer.Consume(input, &dest)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = input.Seek(0, io.SeekStart)
		}
	})
	b.Run("with aliased []byte", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		type binarySlice []byte
		var dest binarySlice
		for i := 0; i < b.N; i++ {
			err = consumer.Consume(input, &dest)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = input.Seek(0, io.SeekStart)
		}
	})
}

func TestByteStreamProducer(t *testing.T) {
	const expected = "the data for the stream to be sent over the wire"
	producer := ByteStreamProducer()

	t.Run("can produce from a WriterTo", func(t *testing.T) {
		var w bytes.Buffer
		var data io.WriterTo = bytes.NewBufferString(expected)
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from a Reader", func(t *testing.T) {
		var w bytes.Buffer
		var data io.Reader = bytes.NewBufferString(expected)
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from a binary marshaler", func(t *testing.T) {
		var w bytes.Buffer
		data := &binaryMarshalDummy{str: expected}
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from a string", func(t *testing.T) {
		var w bytes.Buffer
		data := expected
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from a []byte", func(t *testing.T) {
		var w bytes.Buffer
		data := []byte(expected)
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from an error", func(t *testing.T) {
		var w bytes.Buffer
		data := errors.New(expected)
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from an aliased string", func(t *testing.T) {
		var w bytes.Buffer
		type aliasedString string
		var data aliasedString = expected
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from an interface with underlying type string", func(t *testing.T) {
		var w bytes.Buffer
		var data interface{} = expected
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from an aliased []byte", func(t *testing.T) {
		var w bytes.Buffer
		type binarySlice []byte
		var data binarySlice = []byte(expected)
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce from an interface with underling type []byte", func(t *testing.T) {
		var w bytes.Buffer
		var data interface{} = []byte(expected)
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, expected, w.String())
	})

	t.Run("can produce JSON from an arbitrary struct", func(t *testing.T) {
		var w bytes.Buffer
		type dummy struct {
			Message string `json:"message,omitempty"`
		}
		data := dummy{Message: expected}
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, fmt.Sprintf(`{"message":%q}`, expected), w.String())
	})

	t.Run("can produce JSON from a pointer to an arbitrary struct", func(t *testing.T) {
		var w bytes.Buffer
		type dummy struct {
			Message string `json:"message,omitempty"`
		}
		data := dummy{Message: expected}
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, fmt.Sprintf(`{"message":%q}`, expected), w.String())
	})

	t.Run("can produce JSON from an arbitrary slice", func(t *testing.T) {
		var w bytes.Buffer
		data := []string{expected}
		require.NoError(t, producer.Produce(&w, data))
		assert.Equal(t, fmt.Sprintf(`[%q]`, expected), w.String())
	})

	t.Run("with CloseStream option", func(t *testing.T) {
		t.Run("wants to close stream", func(t *testing.T) {
			closingProducer := ByteStreamProducer(ClosesStream)
			w := &closingWriter{}
			data := bytes.NewBufferString(expected)

			require.NoError(t, closingProducer.Produce(w, data))
			assert.Equal(t, expected, w.String())
			assert.EqualValues(t, 1, w.calledClose)
		})

		t.Run("don't want to close stream", func(t *testing.T) {
			nonClosingProducer := ByteStreamProducer()
			w := &closingWriter{}
			data := bytes.NewBufferString(expected)

			require.NoError(t, nonClosingProducer.Produce(w, data))
			assert.Equal(t, expected, w.String())
			assert.EqualValues(t, 0, w.calledClose)
		})

		t.Run("always close data reader whenever possible", func(t *testing.T) {
			nonClosingProducer := ByteStreamProducer()
			w := &closingWriter{}
			data := &closingReader{b: bytes.NewBufferString(expected)}

			require.NoError(t, nonClosingProducer.Produce(w, data))
			assert.Equal(t, expected, w.String())
			assert.EqualValuesf(t, 0, w.calledClose, "expected the input reader NOT to be closed")
			assert.EqualValuesf(t, 1, data.calledClose, "expected the data reader to be closed")
		})
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("MarshalBinary error gets propagated", func(t *testing.T) {
			var writer bytes.Buffer
			data := new(binaryMarshalDummy)
			require.Error(t, producer.Produce(&writer, data))
		})

		t.Run("nil data is never accepted", func(t *testing.T) {
			var writer bytes.Buffer
			require.Error(t, producer.Produce(&writer, nil))
		})

		t.Run("nil writer should also never be acccepted", func(t *testing.T) {
			data := expected
			require.Error(t, producer.Produce(nil, data))
		})

		t.Run("bool is an unsupported type", func(t *testing.T) {
			var writer bytes.Buffer
			data := true
			require.Error(t, producer.Produce(&writer, data))
		})

		t.Run("WriteJSON error gets propagated", func(t *testing.T) {
			var writer bytes.Buffer
			type cannotMarshal struct {
				X func() `json:"x"`
			}
			data := cannotMarshal{}
			require.Error(t, producer.Produce(&writer, data))
		})

	})
}

type binaryUnmarshalDummy struct {
	err error
	str string
}

type binaryUnmarshalDummyZeroAlloc struct {
	b []byte
}

func (b *binaryUnmarshalDummy) UnmarshalBinary(data []byte) error {
	if b.err != nil {
		return b.err
	}

	if len(data) == 0 {
		return errors.New("no text given")
	}

	b.str = string(data)
	return nil
}

func (b *binaryUnmarshalDummyZeroAlloc) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return errors.New("no text given")
	}

	b.b = data
	return nil
}

type binaryMarshalDummy struct {
	str string
}

func (b *binaryMarshalDummy) MarshalBinary() ([]byte, error) {
	if len(b.str) == 0 {
		return nil, errors.New("no text set")
	}

	return []byte(b.str), nil
}

type closingWriter struct {
	calledClose int64
	calledWrite int64
	b           bytes.Buffer
}

func (c *closingWriter) Close() error {
	atomic.AddInt64(&c.calledClose, 1)
	return nil
}

func (c *closingWriter) Write(p []byte) (n int, err error) {
	atomic.AddInt64(&c.calledWrite, 1)
	return c.b.Write(p)
}

func (c *closingWriter) String() string {
	return c.b.String()
}

type closingReader struct {
	calledClose int64
	calledRead  int64
	b           *bytes.Buffer
}

func (c *closingReader) Close() error {
	atomic.AddInt64(&c.calledClose, 1)
	return nil
}

func (c *closingReader) Read(p []byte) (n int, err error) {
	atomic.AddInt64(&c.calledRead, 1)
	return c.b.Read(p)
}
