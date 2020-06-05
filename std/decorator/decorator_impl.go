package decorator

func NewThread(ID, PID string) *Thread {
	realPID := ""
	if ID != PID {
		realPID = PID
	}
	return &Thread{ID: ID, PID: realPID}
}

func CheckThread(thread *Thread, ID string) *Thread {
	if thread == nil {
		return &Thread{ID: ID}
	}
	if thread.ID == "" {
		thread.ID = ID
	}
	return thread
}
