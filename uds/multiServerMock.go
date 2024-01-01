package uds

type MultiServerMock struct {
}

func (ms *MultiServerMock) Stop() error {
	return nil
}

func (ms *MultiServerMock) StartServer(deviceID string, toDevice, fromDevice chan string) (*Server, error) {
	return nil, nil
}

func (ms *MultiServerMock) StopServer(deviceID string) error {
	return nil
}

func (ms *MultiServerMock) StopAllServers() error {
	return nil
}

func (ms *MultiServerMock) KeepAlive(deviceID string) {
}

func (ms *MultiServerMock) GetServer(deviceID string) (*Server, error) {
	return &Server{}, nil
}
