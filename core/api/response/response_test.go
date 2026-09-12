package response

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseResponseRejectsNullAndMissingCode(t *testing.T) {
	_, err := ParseResponse("null")
	require.Error(t, err)
	_, err = ParseResponse(`{"data":{}}`)
	require.Error(t, err)
}

func TestParseResponseReturnsDecodedResponse(t *testing.T) {
	resp, err := ParseResponse(`{"code":"1","msg":"ok","data":{"id":1}}`)
	require.NoError(t, err)
	require.Equal(t, Success, resp.Code)
	require.Equal(t, "ok", resp.Msg)
}
