package api

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestAPI_WritePaginated(t *testing.T) {
	t.Parallel()

	t.Run("writes the data with the page meta", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		data := []string{
			f.Word(),
			f.Word(),
		}
		totalCount := f.IntRange(2, 100)
		limit := f.IntRange(2, 50)
		offset := f.IntRange(0, 50)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		newBareAPI().writePaginated(recorder, request, data, totalCount, limit, offset)

		assert.Equal(t, http.StatusOK, recorder.Code)
		var body PaginatedResponse[string]
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Equal(t, PaginatedResponse[string]{
			Data: data,
			Meta: PaginatedMeta{
				TotalCount: totalCount,
				Limit:      limit,
				Offset:     offset,
			},
		}, body)
	})
}

func TestQueryPageParams(t *testing.T) {
	t.Parallel()

	t.Run("returns the limit and the offset", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		limit := f.IntRange(1, 100)
		offset := f.IntRange(0, 1000)
		values := url.Values{
			"page[limit]":  {strconv.Itoa(limit)},
			"page[offset]": {strconv.Itoa(offset)},
		}

		gotLimit, gotOffset, err := queryPageParams(values, 10, 100)

		require.NoError(t, err)
		assert.Equal(t, limit, gotLimit)
		assert.Equal(t, offset, gotOffset)
	})

	t.Run("falls back to the default limit and the first item", func(t *testing.T) {
		t.Parallel()
		defaultLimit := apitest.Faker(t).IntRange(1, 100)

		limit, offset, err := queryPageParams(url.Values{}, defaultLimit, 100)

		require.NoError(t, err)
		assert.Equal(t, defaultLimit, limit)
		assert.Equal(t, 0, offset)
	})

	t.Run("caps the limit", func(t *testing.T) {
		t.Parallel()
		maxLimit := apitest.Faker(t).IntRange(1, 100)
		values := url.Values{
			"page[limit]": {strconv.Itoa(maxLimit + 1)},
		}

		limit, _, err := queryPageParams(values, 1, maxLimit)

		require.NoError(t, err)
		assert.Equal(t, maxLimit, limit)
	})

	f := apitest.Faker(t)
	tests := []struct {
		name string
		key  string
		raw  string
	}{
		{
			name: "rejects a malformed page limit",
			key:  "page[limit]",
			raw:  f.LetterN(8),
		},
		{
			name: "rejects a page limit of zero",
			key:  "page[limit]",
			raw:  "0",
		},
		{
			name: "rejects a malformed page offset",
			key:  "page[offset]",
			raw:  f.LetterN(8),
		},
		{
			name: "rejects a negative page offset",
			key:  "page[offset]",
			raw:  strconv.Itoa(-f.IntRange(1, 1000)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := queryPageParams(url.Values{tt.key: {tt.raw}}, 10, 100)

			assert.EqualError(t, err, "invalid query parameter "+tt.key)
		})
	}
}
