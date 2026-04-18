//go:build windows
// +build windows

package supervisord

// ReapZombie windows 平台下为 no-op，避免编译 unix-only 依赖
func ReapZombie() {
	// no-op on windows
}

func Init() {

}

func (s *Supervisor) checkRequiredResources() error {

	return nil

}
