//go:build windows

package platform

func ExitSignal(err error) string {
	return ""
}
