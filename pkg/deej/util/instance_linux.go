package util

func AcquireInstance(directory string) (func(), error) { return func() {}, nil }

func InstanceAlreadyRunning(err error) bool { return false }
