// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	csvFixture = `name,country,age
John,US,19
Mike,US,20
`
	badCSVFixture = `name,country,age
John,US,19
Mike,US
`
	commentedCSVFixture = `# heading line
name,country,age
#John's record
John,US,19
#Mike's record
Mike,US,20
`
)

var testCSVRecords = [][]string{
	{"name", "country", "age"},
	{"John", "US", "19"},
	{"Mike", "US", "20"},
}

func TestCSVConsumer(t *testing.T) {
	consumer := CSVConsumer()

	t.Run("can consume as a *csv.Writer", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var buf bytes.Buffer
		dest := csv.NewWriter(&buf)

		err := consumer.Consume(reader, dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, buf.String())
	})

	t.Run("can consume as a CSVReader", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var dest csvRecordsWriter

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assertCSVRecords(t, dest.records)
	})

	t.Run("can consume as a Writer", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var dest closingWriter

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, dest.b.String())
	})

	t.Run("can consume as a ReaderFrom", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var dest readerFromDummy

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, dest.b.String())
	})

	t.Run("can consume as a BinaryUnmarshaler", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var dest binaryUnmarshalDummy

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, dest.str)
	})

	t.Run("can consume as a *[][]string", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		dest := [][]string{}

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assertCSVRecords(t, dest)
	})

	t.Run("can consume as an alias to *[][]string", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		type records [][]string
		var dest records

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assertCSVRecords(t, dest)
	})

	t.Run("can consume as a *[]byte", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var dest []byte

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, string(dest))
	})

	t.Run("can consume as an alias to *[]byte", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		type buffer []byte
		var dest buffer

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, string(dest))
	})

	t.Run("can consume as a *string", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		var dest string

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, dest)
	})

	t.Run("can consume as an alias to *string", func(t *testing.T) {
		reader := bytes.NewBufferString(csvFixture)
		type buffer string
		var dest buffer

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, string(dest))
	})

	t.Run("can consume from an empty reader", func(t *testing.T) {
		reader := &csvEmptyReader{}
		var dest bytes.Buffer

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)
		assert.Empty(t, dest.String())
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("nil data is never accepted", func(t *testing.T) {
			var rdr bytes.Buffer

			require.Error(t, consumer.Consume(&rdr, nil))
		})

		t.Run("nil readers should also never be acccepted", func(t *testing.T) {
			var buf bytes.Buffer

			err := consumer.Consume(nil, &buf)
			require.Error(t, err)
		})

		t.Run("data must be a pointer", func(t *testing.T) {
			var rdr bytes.Buffer
			var dest []byte

			err := consumer.Consume(&rdr, dest)
			require.Error(t, err)
		})

		t.Run("unsupported type", func(t *testing.T) {
			var rdr bytes.Buffer
			var dest struct{}

			err := consumer.Consume(&rdr, &dest)
			require.Error(t, err)
		})

		t.Run("should propagate CSV error (buffered)", func(t *testing.T) {
			reader := bytes.NewBufferString(badCSVFixture)
			var dest []byte

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "record on line 3: wrong number of fields")
		})

		t.Run("should propagate CSV error (buffered, string)", func(t *testing.T) {
			reader := bytes.NewBufferString(badCSVFixture)
			var dest string

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "record on line 3: wrong number of fields")
		})

		t.Run("should propagate CSV error (buffered, ReaderFrom)", func(t *testing.T) {
			reader := bytes.NewBufferString(badCSVFixture)
			var dest readerFromDummy

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "record on line 3: wrong number of fields")
		})

		t.Run("should propagate CSV error (buffered, BinaryUnmarshaler)", func(t *testing.T) {
			reader := bytes.NewBufferString(badCSVFixture)
			var dest binaryUnmarshalDummy

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "record on line 3: wrong number of fields")
		})

		t.Run("should propagate CSV error (streaming)", func(t *testing.T) {
			reader := bytes.NewBufferString(badCSVFixture)
			var dest bytes.Buffer

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "record on line 3: wrong number of fields")
		})

		t.Run("should propagate CSV error (streaming, write error)", func(t *testing.T) {
			reader := bytes.NewBufferString(csvFixture)
			var buf bytes.Buffer
			dest := csvWriterDummy{err: errors.New("test error"), Writer: csv.NewWriter(&buf)}

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "test error")
		})

		t.Run("should propagate ReaderFrom error", func(t *testing.T) {
			reader := bytes.NewBufferString(csvFixture)
			dest := readerFromDummy{err: errors.New("test error")}

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "test error")
		})

		t.Run("should propagate BinaryUnmarshaler error", func(t *testing.T) {
			reader := bytes.NewBufferString(csvFixture)
			dest := binaryUnmarshalDummy{err: errors.New("test error")}

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "test error")
		})
	})
}

func TestCSVConsumerWithOptions(t *testing.T) {
	semiColonFixture := strings.ReplaceAll(csvFixture, ",", ";")

	t.Run("with CSV reader Comma", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVReaderOpts(csv.Reader{Comma: ';', FieldsPerRecord: 3}))

		t.Run("should not read comma-separated input", func(t *testing.T) {
			reader := bytes.NewBufferString(csvFixture)
			var dest bytes.Buffer

			err := consumer.Consume(reader, &dest)
			require.Error(t, err)
			require.EqualError(t, err, "record on line 1: wrong number of fields")
		})

		t.Run("should read semicolon-separated input and convert it to colon-separated", func(t *testing.T) {
			reader := bytes.NewBufferString(semiColonFixture)
			var dest bytes.Buffer

			err := consumer.Consume(reader, &dest)
			require.NoError(t, err)
			assert.Equal(t, csvFixture, dest.String())
		})
	})

	t.Run("with CSV reader Comment", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVReaderOpts(csv.Reader{Comment: '#'}))

		t.Run("should read input and skip commented lines", func(t *testing.T) {
			reader := bytes.NewBufferString(commentedCSVFixture)
			var dest [][]string

			err := consumer.Consume(reader, &dest)
			require.NoError(t, err)
			assertCSVRecords(t, dest)
		})
	})

	t.Run("with CSV writer Comma", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVWriterOpts(csv.Writer{Comma: ';'}))

		t.Run("should read comma-separated input and convert it to semicolon-separated", func(t *testing.T) {
			reader := bytes.NewBufferString(csvFixture)
			var dest bytes.Buffer

			err := consumer.Consume(reader, &dest)
			require.NoError(t, err)
			assert.Equal(t, semiColonFixture, dest.String())
		})
	})

	t.Run("with SkipLines (streaming)", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVSkipLines(1))
		reader := bytes.NewBufferString(csvFixture)
		var dest [][]string

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)

		expected := testCSVRecords[1:]
		assert.Equalf(t, expected, dest, "expected output to skip header")
	})

	t.Run("with SkipLines (buffered)", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVSkipLines(1))
		reader := bytes.NewBufferString(csvFixture)
		var dest []byte

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)

		r := csv.NewReader(bytes.NewReader(dest))
		consumed, err := r.ReadAll()
		require.NoError(t, err)
		expected := testCSVRecords[1:]
		assert.Equalf(t, expected, consumed, "expected output to skip header")
	})

	t.Run("should detect errors on skipped lines (streaming)", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVSkipLines(1))
		reader := bytes.NewBufferString(strings.ReplaceAll(csvFixture, ",age", `,"age`))
		var dest [][]string

		err := consumer.Consume(reader, &dest)
		require.Error(t, err)
		require.ErrorContains(t, err, "record on line 1; parse error")
	})

	t.Run("should detect errors on skipped lines (buffered)", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVSkipLines(1))
		reader := bytes.NewBufferString(strings.ReplaceAll(csvFixture, ",age", `,"age`))
		var dest []byte

		err := consumer.Consume(reader, &dest)
		require.Error(t, err)
		require.ErrorContains(t, err, "record on line 1; parse error")
	})

	t.Run("with SkipLines greater than the total number of lines (streaming)", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVSkipLines(4))
		reader := bytes.NewBufferString(csvFixture)
		var dest [][]string

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)

		assert.Empty(t, dest)
	})

	t.Run("with SkipLines greater than the total number of lines (buffered)", func(t *testing.T) {
		consumer := CSVConsumer(WithCSVSkipLines(4))
		reader := bytes.NewBufferString(csvFixture)
		var dest []byte

		err := consumer.Consume(reader, &dest)
		require.NoError(t, err)

		assert.Empty(t, dest)
	})

	t.Run("with CloseStream", func(t *testing.T) {
		t.Run("wants to close stream", func(t *testing.T) {
			closingConsumer := CSVConsumer(WithCSVClosesStream())
			var dest bytes.Buffer
			r := &closingReader{b: bytes.NewBufferString(csvFixture)}

			require.NoError(t, closingConsumer.Consume(r, &dest))
			assert.Equal(t, csvFixture, dest.String())
			assert.EqualValues(t, 1, r.calledClose)
		})

		t.Run("don't want to close stream", func(t *testing.T) {
			nonClosingConsumer := CSVConsumer()
			var dest bytes.Buffer
			r := &closingReader{b: bytes.NewBufferString(csvFixture)}

			require.NoError(t, nonClosingConsumer.Consume(r, &dest))
			assert.Equal(t, csvFixture, dest.String())
			assert.EqualValues(t, 0, r.calledClose)
		})
	})
}

func TestCSVProducer(t *testing.T) {
	producer := CSVProducer()

	t.Run("can produce CSV from *csv.Reader", func(t *testing.T) {
		writer := new(bytes.Buffer)
		buf := bytes.NewBufferString(csvFixture)
		data := csv.NewReader(buf)

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from CSVReader", func(t *testing.T) {
		writer := new(bytes.Buffer)
		data := &csvRecordsWriter{
			records: testCSVRecords,
		}

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from Reader", func(t *testing.T) {
		writer := new(bytes.Buffer)
		data := bytes.NewReader([]byte(csvFixture))

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from WriterTo", func(t *testing.T) {
		writer := new(bytes.Buffer)
		buf := bytes.NewBufferString(csvFixture)
		data := &writerToDummy{
			b: *buf,
		}

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from BinaryMarshaler", func(t *testing.T) {
		writer := new(bytes.Buffer)
		data := &binaryMarshalDummy{str: csvFixture}

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from [][]string", func(t *testing.T) {
		writer := new(bytes.Buffer)
		data := testCSVRecords

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from alias to [][]string", func(t *testing.T) {
		writer := new(bytes.Buffer)
		type records [][]string
		data := records(testCSVRecords)

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.String())
	})

	t.Run("can produce CSV from []byte", func(t *testing.T) {
		writer := httptest.NewRecorder()
		data := []byte(csvFixture)

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.Body.String())
	})

	t.Run("can produce CSV from alias to []byte", func(t *testing.T) {
		writer := httptest.NewRecorder()
		type buffer []byte
		data := buffer(csvFixture)

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.Body.String())
	})

	t.Run("can produce CSV from string", func(t *testing.T) {
		writer := httptest.NewRecorder()
		data := csvFixture

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.Body.String())
	})

	t.Run("can produce CSV from alias to string", func(t *testing.T) {
		writer := httptest.NewRecorder()
		type buffer string
		data := buffer(csvFixture)

		err := producer.Produce(writer, data)
		require.NoError(t, err)
		assert.Equal(t, csvFixture, writer.Body.String())
	})

	t.Run("always close data reader whenever possible", func(t *testing.T) {
		nonClosingProducer := CSVProducer()
		r := &closingWriter{}
		data := &closingReader{b: bytes.NewBufferString(csvFixture)}

		require.NoError(t, nonClosingProducer.Produce(r, data))
		assert.Equal(t, csvFixture, r.String())
		assert.EqualValuesf(t, 0, r.calledClose, "expected the input reader NOT to be closed")
		assert.EqualValuesf(t, 1, data.calledClose, "expected the data reader to be closed")
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("unsupported type", func(t *testing.T) {
			writer := httptest.NewRecorder()
			var data struct{}

			err := producer.Produce(writer, data)
			require.Error(t, err)
		})

		t.Run("data cannot be nil", func(t *testing.T) {
			writer := httptest.NewRecorder()

			err := producer.Produce(writer, nil)
			require.Error(t, err)
		})

		t.Run("writer cannot be nil", func(t *testing.T) {
			data := []byte(csvFixture)

			err := producer.Produce(nil, data)
			require.Error(t, err)
		})

		t.Run("should propagate error from BinaryMarshaler", func(t *testing.T) {
			var rdr bytes.Buffer
			data := new(binaryMarshalDummy)

			err := producer.Produce(&rdr, data)
			require.Error(t, err)
			require.ErrorContains(t, err, "no text set")
		})
	})
}

func TestCSVProducerWithOptions(t *testing.T) {
	t.Run("with CloseStream", func(t *testing.T) {
		t.Run("wants to close stream", func(t *testing.T) {
			closingProducer := CSVProducer(WithCSVClosesStream())
			r := &closingWriter{}
			data := bytes.NewBufferString(csvFixture)

			require.NoError(t, closingProducer.Produce(r, data))
			assert.Equal(t, csvFixture, r.String())
			assert.EqualValues(t, 1, r.calledClose)
		})

		t.Run("don't want to close stream", func(t *testing.T) {
			nonClosingProducer := CSVProducer()
			r := &closingWriter{}
			data := bytes.NewBufferString(csvFixture)

			require.NoError(t, nonClosingProducer.Produce(r, data))
			assert.Equal(t, csvFixture, r.String())
			assert.EqualValues(t, 0, r.calledClose)
		})
	})
}

func assertCSVRecords(t testing.TB, dest [][]string) {
	assert.Len(t, dest, 3)
	for i, record := range dest {
		assert.Equal(t, testCSVRecords[i], record)
	}
}

type csvEmptyReader struct{}

func (r *csvEmptyReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

type readerFromDummy struct {
	err error
	b   bytes.Buffer
}

func (r *readerFromDummy) ReadFrom(rdr io.Reader) (int64, error) {
	if r.err != nil {
		return 0, r.err
	}

	return r.b.ReadFrom(rdr)
}

type writerToDummy struct {
	b bytes.Buffer
}

func (w *writerToDummy) WriteTo(writer io.Writer) (int64, error) {
	return w.b.WriteTo(writer)
}

type csvWriterDummy struct {
	err error
	*csv.Writer
}

func (w *csvWriterDummy) Write(record []string) error {
	if w.err != nil {
		return w.err
	}

	return w.Writer.Write(record)
}

func (w *csvWriterDummy) Error() error {
	if w.err != nil {
		return w.err
	}

	return w.Writer.Error()
}
