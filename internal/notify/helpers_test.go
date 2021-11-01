package notify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertDollar(t *testing.T) {
	require.Equal(t, "0.00", convertDollar(0))
	require.Equal(t, "0.01", convertDollar(1))
	require.Equal(t, "0.67", convertDollar(67))
	require.Equal(t, "5.67", convertDollar(567))
	require.Equal(t, "345.67", convertDollar(34567))
	require.Equal(t, "2,345.67", convertDollar(234567))
	require.Equal(t, "12,345.67", convertDollar(1234567))
	require.Equal(t, "1,234,567.89", convertDollar(123456789))
}
