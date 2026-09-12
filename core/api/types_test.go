package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAttachmentsScanAcceptsDatabaseRepresentations(t *testing.T) {
	var attachments Attachments
	require.NoError(t, attachments.Scan(`[{"fileId":"1","fileName":"a.txt"}]`))
	require.Len(t, attachments, 1)
	require.Equal(t, "a.txt", attachments[0].FileName)

	require.NoError(t, attachments.Scan(nil))
	require.Nil(t, attachments)
	require.Error(t, attachments.Scan(123))
}

func TestMapConfigScanAcceptsDatabaseRepresentations(t *testing.T) {
	var config MapConfig
	require.NoError(t, config.Scan([]byte(`{"enabled":true}`)))
	require.Equal(t, true, config["enabled"])
	require.NoError(t, config.Scan(nil))
	require.Nil(t, config)
	require.Error(t, config.Scan(123))
}

func TestLocalTimeScanAndUnmarshalValidateInput(t *testing.T) {
	var local LocalTime
	require.NoError(t, local.Scan(time.Date(2026, 9, 12, 10, 11, 12, 0, time.Local)))
	require.Equal(t, "2026-09-12 10:11:12", local.String())
	require.NoError(t, local.Scan([]byte("2026-09-12 10:11:12")))
	require.Error(t, local.Scan(123))
	require.Error(t, local.UnmarshalJSON([]byte(`"invalid"`)))
	require.NoError(t, local.UnmarshalJSON([]byte("null")))
}
